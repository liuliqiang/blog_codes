package agentloop

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// useBackground swaps in a config and captures background output for the test.
func useBackground(t *testing.T, cfg BackgroundConfig) *bytes.Buffer {
	t.Helper()
	origCfg, origOut, origPoll := Background, backgroundOut, backgroundPollInterval
	out := &bytes.Buffer{}
	Background, backgroundOut, backgroundPollInterval = cfg, out, 50*time.Millisecond
	t.Cleanup(func() { Background, backgroundOut, backgroundPollInterval = origCfg, origOut, origPoll })
	return out
}

func bgCall(id, command string) SendMessagesResponse {
	return SendMessagesResponse{Content: []MessagesBlock{
		{ID: id, Type: MessagesBlockTypeToolUse, Name: "run_bash", Input: map[string]interface{}{"command": command, "run_in_background": true}},
	}}
}

func waitAll(t *testing.T, m *BackgroundManager) {
	t.Helper()
	if !m.Wait(5 * time.Second) {
		t.Fatal("background tasks did not finish")
	}
}

func TestBackgroundManager_StartCollect(t *testing.T) {
	out := useBackground(t, BackgroundConfig{MaxConcurrent: 4, Timeout: time.Minute})
	m := NewBackgroundManager()
	ctx := context.Background()

	if _, err := m.Start(ctx, "  "); err == nil {
		t.Error("blank command must be rejected")
	}
	id1, err := m.Start(ctx, "echo hello")
	if err != nil || id1 != "bg_0001" {
		t.Fatalf("Start = %q, %v", id1, err)
	}
	id2, _ := m.Start(ctx, "echo oops >&2; exit 3")
	if m.Pending() != 2 {
		t.Errorf("Pending = %d, want 2", m.Pending())
	}
	waitAll(t, m)

	notes := m.Collect()
	if len(notes) != 2 {
		t.Fatalf("Collect = %d notes: %v", len(notes), notes)
	}
	joined := strings.Join(notes, "\n")
	for _, want := range []string{
		"<task_id>" + id1 + "</task_id>", "<status>completed</status>", "<command>echo hello</command>", "<summary>hello</summary>",
		"<task_id>" + id2 + "</task_id>", "<status>failed</status>", "Error: exit status 3\noops",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("notes missing %q:\n%s", want, joined)
		}
	}
	if m.Pending() != 0 || len(m.Collect()) != 0 {
		t.Error("Collect must drain the queue")
	}
	if !strings.Contains(out.String(), "[background] started bg_0001") || !strings.Contains(out.String(), "[background] collected bg_0002: failed") {
		t.Errorf("out = %q", out.String())
	}
}

func TestBackgroundManager_ConcurrencyLimit(t *testing.T) {
	useBackground(t, BackgroundConfig{MaxConcurrent: 1, Timeout: time.Minute})
	m := NewBackgroundManager()
	lock := filepath.Join(t.TempDir(), "lock")
	// each command holds a lock file while it runs; overlapping runs would see it already present
	cmd := "if [ -e " + lock + " ]; then echo OVERLAP; fi; touch " + lock + "; sleep 0.15; rm " + lock + "; echo ok"

	for i := 0; i < 3; i++ {
		if _, err := m.Start(context.Background(), cmd); err != nil {
			t.Fatal(err)
		}
	}
	time.Sleep(20 * time.Millisecond)
	m.mu.Lock()
	running, queued := 0, 0
	for _, task := range m.tasks {
		switch task.status {
		case backgroundRunning:
			running++
		case backgroundQueued:
			queued++
		}
	}
	m.mu.Unlock()
	if running != 1 || queued != 2 {
		t.Errorf("running=%d queued=%d, want 1/2", running, queued)
	}

	waitAll(t, m)
	joined := strings.Join(m.Collect(), "\n")
	if strings.Contains(joined, "<summary>OVERLAP") {
		t.Errorf("commands overlapped despite MaxConcurrent=1:\n%s", joined)
	}
	if strings.Count(joined, "<summary>ok</summary>") != 3 {
		t.Errorf("want 3 completed with ok:\n%s", joined)
	}
}

func TestBackgroundManager_Unlimited(t *testing.T) {
	useBackground(t, BackgroundConfig{MaxConcurrent: 0, Timeout: time.Minute})
	m := NewBackgroundManager()
	if m.sem != nil {
		t.Fatal("MaxConcurrent <= 0 should mean no semaphore")
	}
	start := time.Now()
	for i := 0; i < 3; i++ {
		if _, err := m.Start(context.Background(), "sleep 0.2"); err != nil {
			t.Fatal(err)
		}
	}
	waitAll(t, m)
	if took := time.Since(start); took > 500*time.Millisecond {
		t.Errorf("three 0.2s commands took %s, expected them to run in parallel", took)
	}
}

func TestBackgroundManager_TimeoutAndShutdown(t *testing.T) {
	useBackground(t, BackgroundConfig{MaxConcurrent: 1, Timeout: 100 * time.Millisecond})
	m := NewBackgroundManager()
	if _, err := m.Start(context.Background(), "sleep 5"); err != nil {
		t.Fatal(err)
	}
	waitAll(t, m)
	notes := m.Collect()
	if len(notes) != 1 || !strings.Contains(notes[0], "<status>failed</status>") || !strings.Contains(notes[0], "timeout after 100ms") {
		t.Errorf("timeout note = %v", notes)
	}

	// shutdown kills a running command and fails the queued one; the manager is reusable afterwards
	useBackground(t, BackgroundConfig{MaxConcurrent: 1, Timeout: time.Minute})
	m = NewBackgroundManager()
	marker := filepath.Join(t.TempDir(), "done")
	if _, err := m.Start(context.Background(), "sleep 5; touch "+marker); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Start(context.Background(), "echo queued"); err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)
	shutdownStart := time.Now()
	m.Shutdown()
	if time.Since(shutdownStart) > 2*time.Second {
		t.Error("Shutdown should not wait for the full sleep")
	}
	if _, err := os.Stat(marker); err == nil {
		t.Error("killed command still ran to completion")
	}
	if m.Pending() != 0 || len(m.Collect()) != 0 {
		t.Error("Shutdown must drop all state")
	}
	if _, err := m.Start(context.Background(), "echo again"); err != nil {
		t.Fatal(err)
	}
	waitAll(t, m)
	if notes := m.Collect(); len(notes) != 1 || !strings.Contains(notes[0], "<summary>again</summary>") {
		t.Errorf("manager unusable after Shutdown: %v", notes)
	}
}

func TestShouldRunBackground(t *testing.T) {
	cases := []struct {
		name string
		tu   MessagesBlock
		want bool
	}{
		{"flag true", MessagesBlock{Name: "run_bash", Input: map[string]interface{}{"command": "x", "run_in_background": true}}, true},
		{"flag false", MessagesBlock{Name: "run_bash", Input: map[string]interface{}{"command": "x", "run_in_background": false}}, false},
		{"flag missing", MessagesBlock{Name: "run_bash", Input: map[string]interface{}{"command": "x"}}, false},
		{"flag not bool", MessagesBlock{Name: "run_bash", Input: map[string]interface{}{"command": "x", "run_in_background": "true"}}, false},
		{"other tool", MessagesBlock{Name: "read_file", Input: map[string]interface{}{"run_in_background": true}}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := shouldRunBackground(c.tu); got != c.want {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}

func TestInjectBackgroundResults(t *testing.T) {
	useBackground(t, BackgroundConfig{MaxConcurrent: 1, Timeout: time.Minute})
	note := func(a *agent) {
		if _, err := a.background.Start(context.Background(), "echo x"); err != nil {
			t.Fatal(err)
		}
		waitAll(t, a.background)
	}

	// nothing pending: messages untouched
	a := NewAgent(nil, nil).(*agent)
	a.messages = []Message{userMsg("hi")}
	a.injectBackgroundResults()
	if len(a.messages) != 1 || a.messages[0].Content != "hi" {
		t.Errorf("no-op changed messages: %+v", a.messages)
	}

	// trailing user message with blocks: appended to it
	a.messages = []Message{toolResultMsg("t1", "out")}
	note(a)
	a.injectBackgroundResults()
	blocks, _ := a.messages[0].Content.([]MessagesBlock)
	if len(a.messages) != 1 || len(blocks) != 2 || blocks[1].Type != MessagesBlockTypeText || !strings.Contains(blocks[1].Text, "<task_notification>") {
		t.Errorf("not appended to trailing user message: %+v", a.messages)
	}

	// trailing user message as string: converted to blocks
	a.messages = []Message{userMsg("hi")}
	note(a)
	a.injectBackgroundResults()
	blocks, _ = a.messages[0].Content.([]MessagesBlock)
	if len(a.messages) != 1 || len(blocks) != 2 || blocks[0].Text != "hi" {
		t.Errorf("string content not converted: %+v", a.messages)
	}

	// trailing assistant message: new user message
	a.messages = []Message{userMsg("hi"), assistantMsg("ok")}
	note(a)
	a.injectBackgroundResults()
	if len(a.messages) != 3 || a.messages[2].Role != MessageRoleUser {
		t.Errorf("no new user message: %+v", a.messages)
	}
}

func TestRunLoop_BackgroundRoundTrip(t *testing.T) {
	useBackground(t, BackgroundConfig{MaxConcurrent: 2, Timeout: time.Minute})
	autoAllow(t)
	responses := []SendMessagesResponse{
		bgCall("b1", "sleep 0.2; echo slow done"), // turn 1: background
		bashCall("f1", "echo fast"),               // turn 2: sync, while slow runs
	}
	for i := 0; i < 10; i++ { // the model keeps trying to stop; the hook nudges it until bg_0001 reports
		responses = append(responses, text("all done"))
	}
	llm := &scriptedLLM{responses: responses}
	h := new(Hooks).OnStop(BackgroundTasksHook())

	if err := NewAgent(llm, h).RunLoop(context.Background(), userHi); err != nil {
		t.Fatal(err)
	}

	// turn 1's tool result is the placeholder and came back immediately
	if res := lastToolResult(t, llm.calls[1]); res.ToolUseID != "b1" || !strings.HasPrefix(res.Content, "[Background task bg_0001 started]") {
		t.Errorf("placeholder = %+v", res)
	}
	// turn 2 ran the fast command normally while the slow one was still going
	if res := lastToolResult(t, llm.calls[2]); res.ToolUseID != "f1" || strings.TrimSpace(res.Content) != "fast" {
		t.Errorf("fast result = %+v", res)
	}
	if len(llm.calls) < 4 || len(llm.calls) > 12 {
		t.Fatalf("calls = %d, want the loop to continue past the first stop and end once the task reported", len(llm.calls))
	}
	// the notification reached the model before it was allowed to stop
	final := renderConversation(llm.calls[len(llm.calls)-1])
	if !strings.Contains(final, "<task_notification>") || !strings.Contains(final, "<summary>slow done</summary>") {
		t.Errorf("notification not delivered before the final call:\n%s", final)
	}
	// one tool_use, one tool_result: the notification never reuses b1
	for _, msgs := range llm.calls {
		for _, m := range msgs {
			blocks, _ := m.Content.([]MessagesBlock)
			for _, b := range blocks {
				if b.Type == MessagesBlockTypeToolResult && b.ToolUseID == "b1" && strings.Contains(b.Content, "slow done") {
					t.Error("background result must not be delivered as a second tool_result for b1")
				}
			}
		}
	}
}

func TestRunLoop_BackgroundRoundTrip_Scripted(t *testing.T) {
	// deterministic variant: the model stops immediately, the hook must wait then deliver
	useBackground(t, BackgroundConfig{MaxConcurrent: 2, Timeout: time.Minute})
	autoAllow(t)
	llm := &scriptedLLM{responses: []SendMessagesResponse{
		bgCall("b1", "sleep 0.1; echo late"),
		text("done?"),   // stop attempt 1: pending → hook waits 50ms, nudges
		text("done??"),  // stop attempt 2: maybe still pending → nudge or notification
		text("done???"), // stop attempt 3
		text("done!"),   // in case the timing needs one more
	}}
	h := new(Hooks).OnStop(BackgroundTasksHook())
	if err := NewAgent(llm, h).RunLoop(context.Background(), userHi); err != nil {
		t.Fatal(err)
	}
	all := renderConversation(llm.calls[len(llm.calls)-1])
	if !strings.Contains(all, "background task(s) are still pending") {
		t.Errorf("model was never told to wait:\n%s", all)
	}
	if !strings.Contains(all, "<summary>late</summary>") {
		t.Errorf("notification never delivered:\n%s", all)
	}
}

func TestBackgroundTasksHook_NoAgentOrNothingPending(t *testing.T) {
	useBackground(t, BackgroundConfig{MaxConcurrent: 1, Timeout: time.Minute})
	h := BackgroundTasksHook()
	if msg, err := h(context.Background(), nil); msg != nil || err != nil {
		t.Errorf("no agent in ctx: %v, %v", msg, err)
	}
	a := NewAgent(nil, nil).(*agent)
	if msg, err := h(withAgent(context.Background(), a), nil); msg != nil || err != nil {
		t.Errorf("nothing pending: %v, %v", msg, err)
	}
}

func TestRunLoop_ShutsDownBackgroundOnExit(t *testing.T) {
	useBackground(t, BackgroundConfig{MaxConcurrent: 1, Timeout: time.Minute})
	autoAllow(t)
	marker := filepath.Join(t.TempDir(), "leaked")
	llm := &scriptedLLM{responses: []SendMessagesResponse{bgCall("b1", "sleep 3; touch "+marker), text("bye")}}
	// no BackgroundTasksHook: the loop ends with the task still running
	a := NewAgent(llm, nil).(*agent)
	if err := a.RunLoop(context.Background(), userHi); err != nil {
		t.Fatal(err)
	}
	if a.background.Pending() != 0 {
		t.Error("resetLoop should have shut the manager down")
	}
	time.Sleep(100 * time.Millisecond)
	if _, err := os.Stat(marker); err == nil {
		t.Error("background command outlived the agent")
	}
}

func TestSubagent_OwnBackgroundManager(t *testing.T) {
	useBackground(t, BackgroundConfig{MaxConcurrent: 1, Timeout: time.Minute})
	a := NewAgent(nil, nil).(*agent)
	if sub := a.newSubagent(); sub.background == nil || sub.background == a.background {
		t.Error("subagent must have its own background manager")
	}
}
