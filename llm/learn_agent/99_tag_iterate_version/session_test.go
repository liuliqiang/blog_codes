package agentloop

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func newTestManager(t *testing.T, llm LLMClient) (*SessionManager, string) {
	t.Helper()
	t.Chdir(t.TempDir()) // memory, tasks and friends are relative to the working directory
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	root := filepath.Join(t.TempDir(), "sessions")
	m, err := NewSessionManager(ctx, root, llm, new(Hooks))
	if err != nil {
		t.Fatal(err)
	}
	return m, root
}

// waitIdle waits until the session has no turn queued or running.
func waitIdle(t *testing.T, s *Session) SessionView {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if view := s.View(true); !view.Busy {
			return view
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("session still busy")
	return SessionView{}
}

func nextEvent(t *testing.T, events <-chan SessionEvent, eventType string) SessionEvent {
	t.Helper()
	timeout := time.After(5 * time.Second)
	for {
		select {
		case ev, ok := <-events:
			if !ok {
				t.Fatalf("event stream closed while waiting for %s", eventType)
			}
			if ev.Type == eventType {
				return ev
			}
		case <-timeout:
			t.Fatalf("no %s event", eventType)
		}
	}
}

func TestSession_TurnFoldsStepsAndKeepsReply(t *testing.T) {
	llm := &scriptedLLM{responses: []SendMessagesResponse{
		{Content: []MessagesBlock{
			{Type: MessagesBlockTypeText, Text: "let me look"},
			{ID: "c1", Type: MessagesBlockTypeToolUse, Name: "run_bash", Input: map[string]interface{}{"command": "echo hi"}},
		}},
		text("it printed hi"),
	}}
	m, _ := newTestManager(t, llm)
	s, err := m.Create()
	if err != nil {
		t.Fatal(err)
	}
	snapshot, events, cancel := s.Subscribe()
	defer cancel()
	if snapshot.Title != defaultSessionTitle || len(snapshot.Turns) != 0 {
		t.Fatalf("snapshot = %+v", snapshot)
	}

	if err := s.Submit("run echo hi"); err != nil {
		t.Fatal(err)
	}
	running := nextEvent(t, events, "turn")
	if running.Turn.Status != "running" || running.Turn.Source != "user" || running.Turn.Prompt != "run echo hi" {
		t.Errorf("first turn event = %+v", running.Turn)
	}

	view := waitIdle(t, s)
	if view.Title != "run echo hi" {
		t.Errorf("title should come from the first prompt, got %q", view.Title)
	}
	if len(view.Turns) != 1 {
		t.Fatalf("turns = %+v", view.Turns)
	}
	turn := view.Turns[0]
	if turn.Status != "done" || turn.Reply != "it printed hi" {
		t.Errorf("turn = %+v", turn)
	}
	// the final reply is not repeated among the folded steps
	if len(turn.Steps) != 2 || turn.Steps[0].Text != "let me look" || turn.Steps[1].Tool != "run_bash" ||
		turn.Steps[1].Status != "done" || !strings.Contains(turn.Steps[1].Output, "hi") {
		t.Errorf("steps = %+v", turn.Steps)
	}
}

func TestSession_PersistsAndResumesHistory(t *testing.T) {
	llm := &scriptedLLM{responses: []SendMessagesResponse{text("first answer"), text("second answer")}}
	m, root := newTestManager(t, llm)
	s, _ := m.Create()
	if err := s.Submit("first question"); err != nil {
		t.Fatal(err)
	}
	waitIdle(t, s)

	// a new manager over the same root is a restart
	m2, err := NewSessionManager(context.Background(), root, llm, new(Hooks))
	if err != nil {
		t.Fatal(err)
	}
	list := m2.List()
	if len(list) != 1 || list[0].ID != s.ID() || list[0].Title != "first question" {
		t.Fatalf("restored list = %+v", list)
	}
	restored, _ := m2.Get(s.ID())
	if turns := restored.View(true).Turns; len(turns) != 1 || turns[0].Reply != "first answer" {
		t.Fatalf("restored turns = %+v", turns)
	}
	if err := restored.Submit("second question"); err != nil {
		t.Fatal(err)
	}
	waitIdle(t, restored)
	// the restored agent carries the earlier conversation into the next call
	second := llm.calls[1]
	if len(second) != 3 || second[0].Content != "first question" || second[2].Content != "second question" {
		t.Errorf("second call messages = %+v", second)
	}
	blocks, ok := second[1].Content.([]MessagesBlock)
	if !ok || blocks[0].Text != "first answer" {
		t.Errorf("assistant reply did not survive the round trip: %+v", second[1])
	}
}

func TestSession_BusyRejectsSecondPrompt(t *testing.T) {
	llm := &gateLLM{
		scriptedLLM: scriptedLLM{responses: []SendMessagesResponse{text("done")}},
		started:     make(chan struct{}, 1),
		release:     make(chan struct{}),
	}
	m, _ := newTestManager(t, llm)
	s, _ := m.Create()
	if err := s.Submit("slow"); err != nil {
		t.Fatal(err)
	}
	<-llm.started
	if err := s.Submit("another"); !errors.Is(err, ErrSessionBusy) {
		t.Errorf("err = %v, want ErrSessionBusy", err)
	}
	if !s.View(false).Busy {
		t.Error("session should report busy")
	}
	close(llm.release)
	waitIdle(t, s)
	if err := s.Submit(""); err == nil {
		t.Error("an empty prompt should be rejected")
	}
}

func TestSession_ScheduledTurnIsMarkedCron(t *testing.T) {
	llm := &scriptedLLM{responses: []SendMessagesResponse{text("report sent")}}
	m, _ := newTestManager(t, llm)
	s, _ := m.Create()
	// what the scheduler does when a job fires: a turn the user did not submit
	if err := s.sched.RunTurn(withNonInteractive(s.ctx), []Message{{Role: MessageRoleUser, Content: "[Scheduled] send the report"}}); err != nil {
		t.Fatal(err)
	}
	turns := s.View(true).Turns
	if len(turns) != 1 || turns[0].Source != "cron" || turns[0].Prompt != "[Scheduled] send the report" || turns[0].Reply != "report sent" {
		t.Errorf("turns = %+v", turns)
	}
}

func TestSession_GoalCommandsShowTheirReply(t *testing.T) {
	m, _ := newTestManager(t, &scriptedLLM{})
	s, _ := m.Create()
	captureHookOut(t)

	if err := s.Submit("/goal"); err != nil {
		t.Fatal(err)
	}
	view := waitIdle(t, s)
	if len(view.Turns) != 1 || view.Turns[0].Reply != "No goal set" || view.Turns[0].Status != "done" {
		t.Errorf("status turn = %+v", view.Turns)
	}

	// a goal refused before the loop starts still shows up, as a failed turn
	if err := s.Submit("/goal " + strings.Repeat("x", maxGoalLength+1)); err != nil {
		t.Fatal(err)
	}
	view = waitIdle(t, s)
	if len(view.Turns) != 2 || view.Turns[1].Status != "failed" || !strings.Contains(view.Turns[1].Error, "cannot exceed") {
		t.Errorf("rejected goal turn = %+v", view.Turns)
	}
}

func TestSession_PermissionRoundTrip(t *testing.T) {
	m, _ := newTestManager(t, &scriptedLLM{})
	s, _ := m.Create()
	_, events, cancel := s.Subscribe()
	defer cancel()
	tool := MessagesBlock{Name: "write_file", Input: map[string]interface{}{"path": "a.txt"}}

	result := make(chan bool, 1)
	go func() { result <- checkUserAllow(WithApprover(context.Background(), s.approve), tool) }()
	req := nextEvent(t, events, "permission").Permission
	if req.Tool != "write_file" || !strings.Contains(req.Input, `"path": "a.txt"`) {
		t.Errorf("request = %+v", req)
	}
	if view := s.View(true); len(view.Permissions) != 1 {
		t.Errorf("pending permissions = %+v", view.Permissions)
	}
	if err := s.ResolvePermission(req.ID, true); err != nil {
		t.Fatal(err)
	}
	if !<-result {
		t.Error("an allowed request should let the tool run")
	}
	if ev := nextEvent(t, events, "permission_resolved"); ev.PermissionID != req.ID {
		t.Errorf("resolved = %+v", ev)
	}
	if err := s.ResolvePermission(req.ID, true); err == nil {
		t.Error("answering twice should fail")
	}

	// a cancelled turn denies what it was waiting for
	ctx, stop := context.WithCancel(context.Background())
	go func() { result <- s.approve(ctx, tool) }()
	nextEvent(t, events, "permission")
	stop()
	if <-result {
		t.Error("a cancelled request must deny")
	}
}

func TestSessionManager_Delete(t *testing.T) {
	m, root := newTestManager(t, &scriptedLLM{})
	s, _ := m.Create()
	if _, err := os.Stat(filepath.Join(root, s.ID(), sessionFileName)); err != nil {
		t.Fatalf("session file not written: %v", err)
	}
	if err := m.Delete(s.ID()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, s.ID())); !os.IsNotExist(err) {
		t.Errorf("session dir should be gone, stat err = %v", err)
	}
	if _, err := m.Get(s.ID()); !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("get after delete err = %v", err)
	}
	if err := m.Delete(s.ID()); !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("second delete err = %v", err)
	}
}

func TestSessionManager_LoadMarksRunningTurnInterruptedAndSkipsBrokenSessions(t *testing.T) {
	m, root := newTestManager(t, &scriptedLLM{})
	s, _ := m.Create()
	s.mu.Lock()
	s.turns = append(s.turns, &SessionTurn{ID: 1, Source: "user", Prompt: "long job", Status: "running"})
	s.mu.Unlock()
	if err := s.save(); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "broken"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "broken", sessionFileName), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	m2, err := NewSessionManager(context.Background(), root, &scriptedLLM{}, new(Hooks))
	if err != nil {
		t.Fatal(err)
	}
	if list := m2.List(); len(list) != 1 {
		t.Fatalf("the broken session should be skipped, list = %+v", list)
	}
	restored, _ := m2.Get(s.ID())
	if turns := restored.View(true).Turns; turns[0].Status != "interrupted" {
		t.Errorf("turn status = %q", turns[0].Status)
	}
}

func TestSessionMessages_RoundTrip(t *testing.T) {
	in := []Message{
		{Role: MessageRoleUser, Content: "hi"},
		{Role: MessageRoleAssistant, Content: []MessagesBlock{{ID: "c1", Type: MessagesBlockTypeToolUse, Name: "run_bash", Input: map[string]interface{}{"command": "ls"}}}},
		{Role: MessageRoleUser, Content: []MessagesBlock{{Type: MessagesBlockTypeToolResult, ToolUseID: "c1", Content: "a.txt"}}},
	}
	out := decodeMessages(encodeMessages(in))
	if len(out) != 3 || out[0].Content != "hi" {
		t.Fatalf("out = %+v", out)
	}
	if b := out[1].Content.([]MessagesBlock)[0]; b.Name != "run_bash" || b.Input["command"] != "ls" {
		t.Errorf("tool use = %+v", b)
	}
	if b := out[2].Content.([]MessagesBlock)[0]; b.ToolUseID != "c1" || b.Content != "a.txt" {
		t.Errorf("tool result = %+v", b)
	}
}

func TestTruncateRunes(t *testing.T) {
	if got := truncateRunes("  帮我\n修复   测试  ", 40); got != "帮我 修复 测试" {
		t.Errorf("got %q", got)
	}
	if got := truncateRunes("一二三四五", 3); got != "一二三…" {
		t.Errorf("got %q", got)
	}
}
