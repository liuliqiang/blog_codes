package agentloop

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// useTeam points the mailboxes at a temp dir, shortens the idle scan and captures team output.
func useTeam(t *testing.T, cfg TeamConfig) *bytes.Buffer {
	t.Helper()
	dir := filepath.Join(t.TempDir(), ".mailboxes")
	origDir, origOut, origCfg := mailboxDir, teamOut, Team
	out := &bytes.Buffer{}
	mailboxDir, teamOut, Team = dir, out, cfg
	t.Cleanup(func() { mailboxDir, teamOut, Team = origDir, origOut, origCfg })
	return out
}

func testTeamConfig() TeamConfig {
	return TeamConfig{MaxMates: 4, IdleScan: 30 * time.Millisecond, StopTimeout: 2 * time.Second}
}

func TestMessageBus_SendReadWait(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".mailboxes")
	b := NewMessageBus(dir)

	if b.Peek("alice") || len(b.ReadInbox("alice")) != 0 {
		t.Error("fresh mailbox should be empty")
	}
	for _, bad := range []string{"", "../x", "a/b"} {
		if err := b.Send(TeamMessage{To: bad, Content: "x"}); err == nil {
			t.Errorf("Send to %q should fail", bad)
		}
	}

	if err := b.Send(TeamMessage{From: "lead", To: "alice", Type: teamMessage, Content: "first"}); err != nil {
		t.Fatal(err)
	}
	if err := b.Send(TeamMessage{From: "lead", To: "alice", Type: teamMessage, Content: "second"}); err != nil {
		t.Fatal(err)
	}
	if !b.Peek("alice") {
		t.Error("Peek should see the messages")
	}
	msgs := b.ReadInbox("alice")
	if len(msgs) != 2 || msgs[0].Content != "first" || msgs[1].Content != "second" {
		t.Fatalf("ReadInbox = %+v", msgs)
	}
	// reading consumes: each message has exactly one reader
	if len(b.ReadInbox("alice")) != 0 || b.Peek("alice") {
		t.Error("mailbox should be empty after reading")
	}
	if _, err := os.Stat(filepath.Join(dir, "alice.jsonl")); !os.IsNotExist(err) {
		t.Error("mailbox file should be removed once consumed")
	}

	// an unreadable line is skipped, not fatal
	if err := os.WriteFile(filepath.Join(dir, "bob.jsonl"), []byte("{bad\n{\"from\":\"lead\",\"to\":\"bob\",\"content\":\"ok\"}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if msgs := b.ReadInbox("bob"); len(msgs) != 1 || msgs[0].Content != "ok" {
		t.Errorf("ReadInbox with a bad line = %+v", msgs)
	}
}

func TestMessageBus_WaitWakesAndTimesOut(t *testing.T) {
	b := NewMessageBus(filepath.Join(t.TempDir(), ".mailboxes"))
	ctx := context.Background()

	start := time.Now()
	if msgs := b.Wait(ctx, "alice", 50*time.Millisecond); msgs != nil {
		t.Errorf("Wait should time out empty: %+v", msgs)
	}
	if took := time.Since(start); took < 40*time.Millisecond {
		t.Errorf("Wait returned after %s, expected it to wait", took)
	}

	// a message sent while waiting wakes it promptly
	go func() {
		time.Sleep(20 * time.Millisecond)
		_ = b.Send(TeamMessage{From: "lead", To: "alice", Content: "wake up"})
	}()
	start = time.Now()
	msgs := b.Wait(ctx, "alice", 3*time.Second)
	if len(msgs) != 1 || msgs[0].Content != "wake up" {
		t.Fatalf("Wait = %+v", msgs)
	}
	if took := time.Since(start); took > 2*time.Second {
		t.Errorf("Wait took %s, expected the send to wake it", took)
	}

	// a cancelled ctx returns immediately
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if msgs := b.Wait(cancelled, "alice", time.Minute); msgs != nil {
		t.Errorf("cancelled Wait = %+v", msgs)
	}
}

// teamLLM answers each agent by name, so a lead and its teammates can be scripted independently.
type teamLLM struct {
	mu        sync.Mutex
	byAgent   map[string][]SendMessagesResponse
	calls     map[string][]([]Message)
	fallback  SendMessagesResponse
	sawSystem map[string]string
}

func newTeamLLM() *teamLLM {
	return &teamLLM{
		byAgent:   map[string][]SendMessagesResponse{},
		calls:     map[string][]([]Message){},
		fallback:  text("ok"),
		sawSystem: map[string]string{},
	}
}

func (l *teamLLM) GetModel() Model { return "fake" }

func (l *teamLLM) SendMessages(ctx context.Context, _ Model, system Message, messages []Message, _ []Tool, _ SendMessagesOpts) (SendMessagesResponse, error) {
	name := agentNameFrom(ctx)
	l.mu.Lock()
	defer l.mu.Unlock()
	l.calls[name] = append(l.calls[name], append([]Message(nil), messages...))
	l.sawSystem[name], _ = system.Content.(string)
	queue := l.byAgent[name]
	if len(queue) == 0 {
		return l.fallback, nil
	}
	l.byAgent[name] = queue[1:]
	return queue[0], nil
}

func (l *teamLLM) callsFor(name string) [][]Message {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([][]Message(nil), l.calls[name]...)
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.After(3 * time.Second)
	for !cond() {
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for %s", what)
		case <-time.After(5 * time.Millisecond):
		}
	}
}

func TestTeam_SpawnWorksAndReports(t *testing.T) {
	useTasksDir(t)
	out := useTeam(t, testTeamConfig())
	llm := newTeamLLM()
	llm.byAgent["alice"] = []SendMessagesResponse{text("config cleaned up")}
	lead := NewAgent(llm, nil).(*agent)
	task, err := lead.tasks.Create("clean up config", "make it tidy")
	if err != nil {
		t.Fatal(err)
	}

	mate, err := lead.team.Spawn("alice", "config", task.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	defer lead.team.Shutdown()

	// the first task was claimed before the teammate started, and it saw the assignment
	claimed, err := lead.tasks.Load(task.ID)
	if err != nil || claimed.Owner != "alice" || claimed.Status != TaskInProgress {
		t.Fatalf("task = %+v, %v", claimed, err)
	}
	waitFor(t, "alice's turn", func() bool { return len(llm.callsFor("alice")) > 0 })
	first := llm.callsFor("alice")[0]
	if got, _ := first[0].Content.(string); !strings.Contains(got, "[Task "+task.ID+"] clean up config") || !strings.Contains(got, "make it tidy") {
		t.Errorf("assignment = %q", got)
	}
	if sys := llm.sawSystem["alice"]; !strings.HasPrefix(sys, teammateSystemPrompt) || !strings.Contains(sys, "Your area of responsibility: config") {
		t.Errorf("teammate system prompt = %q", sys)
	}
	if _, ok := mate.agent.toolIndex["update_task"]; ok {
		t.Error("teammates must not rewrite the task graph")
	}
	for _, want := range []string{"claim_task", "complete_task", "send_message", "run_bash"} {
		if _, ok := mate.agent.toolIndex[want]; !ok {
			t.Errorf("teammate missing %s", want)
		}
	}

	// the result and the idle notification reach the lead as two separate events
	waitFor(t, "lead's events", func() bool { return lead.team.bus.Peek(leadName) })
	var notes []string
	waitFor(t, "both events", func() bool {
		notes = append(notes, lead.team.consumeTeamEvents()...)
		return len(notes) >= 2
	})
	joined := strings.Join(notes, "\n")
	if !strings.Contains(joined, `type="result"`) || !strings.Contains(joined, "config cleaned up") {
		t.Errorf("result event missing:\n%s", joined)
	}
	if !strings.Contains(joined, `type="idle_notification"`) {
		t.Errorf("idle event missing:\n%s", joined)
	}
	waitFor(t, "alice to go idle", func() bool { state, _ := mate.snapshot(); return state == teammateIdle })
	if !strings.Contains(out.String(), "[team] spawned alice (config)") {
		t.Errorf("out = %q", out.String())
	}
}

func TestTeam_SpawnValidation(t *testing.T) {
	useTasksDir(t)
	useTeam(t, TeamConfig{MaxMates: 1, IdleScan: 30 * time.Millisecond, StopTimeout: time.Second})
	llm := newTeamLLM()
	lead := NewAgent(llm, nil).(*agent)
	defer lead.team.Shutdown()

	for _, bad := range []string{"", "lead", "main", "subagent", "a/b"} {
		if _, err := lead.team.Spawn(bad, "", "", false); err == nil {
			t.Errorf("Spawn(%q) should fail", bad)
		}
	}
	if _, err := lead.team.Spawn("alice", "", "task_00000000", false); err == nil {
		t.Error("spawning on a missing task should fail")
	}
	if lead.team.Active() != 0 {
		t.Error("a failed spawn must not leave a teammate behind")
	}
	if _, err := lead.team.Spawn("alice", "", "", false); err != nil {
		t.Fatal(err)
	}
	if _, err := lead.team.Spawn("alice", "", "", false); err == nil {
		t.Error("duplicate name should fail")
	}
	if _, err := lead.team.Spawn("bob", "", "", false); err == nil || !strings.Contains(err.Error(), "already has 1 teammates") {
		t.Errorf("MaxMates not enforced: %v", err)
	}
}

func TestTeam_IdleClaimsFromBoard(t *testing.T) {
	useTasksDir(t)
	useTeam(t, testTeamConfig())
	llm := newTeamLLM()
	lead := NewAgent(llm, nil).(*agent)
	first, _ := lead.tasks.Create("first", "")
	second, _ := lead.tasks.Create("second", "")
	blocked, _ := lead.tasks.Create("blocked", "")
	if _, err := lead.tasks.AddDependencies(blocked.ID, []string{second.ID}); err != nil {
		t.Fatal(err)
	}

	if _, err := lead.team.Spawn("alice", "", first.ID, false); err != nil {
		t.Fatal(err)
	}
	defer lead.team.Shutdown()

	// the first task was never completed, so it goes back on the board when the turn ends
	waitFor(t, "the first task to be released", func() bool {
		task, err := lead.tasks.Load(first.ID)
		return err == nil && task.Status == TaskPending && task.Owner == ""
	})
	// then the idle teammate picks ready work off the board itself
	waitFor(t, "alice to claim a task off the board", func() bool {
		for _, id := range []string{first.ID, second.ID} {
			task, err := lead.tasks.Load(id)
			if err == nil && task.Owner == "alice" && task.Status == TaskInProgress {
				return true
			}
		}
		return false
	})
	// the blocked task stays untouched while its prerequisite is unfinished
	if task, err := lead.tasks.Load(blocked.ID); err != nil || task.Owner != "" {
		t.Errorf("blocked task was claimed: %+v, %v", task, err)
	}
}

func TestTaskStore_ClaimNextIsExclusive(t *testing.T) {
	useTasksDir(t)
	store := NewTaskStore(tasksDir)
	const tasks = 6
	for i := 0; i < tasks; i++ {
		if _, err := store.Create("task", ""); err != nil {
			t.Fatal(err)
		}
	}

	// several owners race for the same board; each may hold only one task, and no task may go to two owners
	var wg sync.WaitGroup
	var mu sync.Mutex
	claimed := map[string]string{}
	for i := 0; i < tasks; i++ {
		owner := string(rune('a' + i))
		wg.Add(1)
		go func() {
			defer wg.Done()
			task, ok, err := store.ClaimNext(owner)
			if err != nil || !ok {
				return
			}
			mu.Lock()
			defer mu.Unlock()
			if prev, dup := claimed[task.ID]; dup {
				t.Errorf("task %s claimed by both %s and %s", task.ID, prev, owner)
			}
			claimed[task.ID] = owner
		}()
	}
	wg.Wait()
	if len(claimed) == 0 {
		t.Fatal("nobody claimed anything")
	}
	all, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	owners := map[string]int{}
	for _, task := range all {
		if task.Owner != "" {
			owners[task.Owner]++
		}
	}
	for owner, n := range owners {
		if n != 1 {
			t.Errorf("%s holds %d tasks at once", owner, n)
		}
	}

	// an owner with work in progress gets nothing more
	for owner := range owners {
		if _, ok, err := store.ClaimNext(owner); err != nil || ok {
			t.Errorf("%s claimed a second task: ok=%v err=%v", owner, ok, err)
		}
		if err := store.Release(claimedIDFor(all, owner), owner); err != nil {
			t.Fatal(err)
		}
		if _, ok, err := store.ClaimNext(owner); err != nil || !ok {
			t.Errorf("%s could not claim after releasing: ok=%v err=%v", owner, ok, err)
		}
		break
	}
}

func claimedIDFor(tasks []Task, owner string) string {
	for _, t := range tasks {
		if t.Owner == owner {
			return t.ID
		}
	}
	return ""
}

func TestTeam_MessagesAndShutdownProtocol(t *testing.T) {
	useTasksDir(t)
	out := useTeam(t, testTeamConfig())
	llm := newTeamLLM()
	llm.byAgent["alice"] = []SendMessagesResponse{text("first done"), text("answered the lead")}
	lead := NewAgent(llm, nil).(*agent)
	ctx := withAgent(withAgentName(context.Background(), "main"), lead)

	if _, err := lead.toolIndex["spawn_teammate"].Handler(ctx, map[string]interface{}{"name": "alice", "role": "config"}); err != nil {
		t.Fatal(err)
	}
	list, err := lead.toolIndex["list_teammates"].Handler(ctx, nil)
	if err != nil || !strings.Contains(list, "alice [") || !strings.Contains(list, "config") {
		t.Errorf("list_teammates = %q, %v", list, err)
	}

	// a message from the lead becomes the teammate's next assignment
	if _, err := lead.toolIndex["send_message"].Handler(ctx, map[string]interface{}{"to": "alice", "content": "please check the config"}); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "alice to work on the message", func() bool {
		for _, call := range llm.callsFor("alice") {
			for _, m := range call {
				if s, _ := m.Content.(string); strings.Contains(s, "[Message from main] please check the config") {
					return true
				}
			}
		}
		return false
	})
	if _, err := lead.toolIndex["send_message"].Handler(ctx, map[string]interface{}{"to": "nobody", "content": "x"}); err == nil {
		t.Error("messaging an unknown teammate should fail")
	}

	// shutdown is a typed request answered by a typed response carrying the same id
	res, err := lead.toolIndex["shutdown_teammate"].Handler(ctx, map[string]interface{}{"name": "alice"})
	if err != nil || !strings.Contains(res, "Asked alice to stop") {
		t.Fatalf("shutdown_teammate = %q, %v", res, err)
	}
	_, rest, _ := strings.Cut(res, "(request ")
	reqID, _, _ := strings.Cut(rest, ")")
	waitFor(t, "alice to stop", func() bool { return lead.team.Active() == 0 })
	waitFor(t, "the shutdown response", func() bool {
		lead.team.consumeTeamEvents()
		lead.team.mu.Lock()
		defer lead.team.mu.Unlock()
		return lead.team.requests[reqID].approved
	})

	// a duplicate or mismatched reply changes nothing
	lead.team.matchShutdownResponse(TeamMessage{From: "alice", Type: teamShutdownResponse, RequestID: reqID})
	lead.team.matchShutdownResponse(TeamMessage{From: "mallory", Type: teamShutdownResponse, RequestID: reqID})
	lead.team.matchShutdownResponse(TeamMessage{From: "alice", Type: teamMessage, RequestID: reqID})
	if _, err := lead.toolIndex["shutdown_teammate"].Handler(ctx, map[string]interface{}{"name": "alice"}); err == nil {
		t.Error("shutting down a stopped teammate should fail")
	}
	if !strings.Contains(out.String(), "[team] alice stopping") {
		t.Errorf("out = %q", out.String())
	}
	lead.team.Shutdown()
}

func TestTeam_TeammateSendsToLead(t *testing.T) {
	useTasksDir(t)
	useTeam(t, testTeamConfig())
	llm := newTeamLLM()
	lead := NewAgent(llm, nil).(*agent)
	mate, err := lead.team.Spawn("alice", "", "", false)
	if err != nil {
		t.Fatal(err)
	}
	defer lead.team.Shutdown()

	// the teammate's send_message takes no recipient: it can only write to the lead
	tool := mate.agent.toolIndex["send_message"]
	if _, ok := tool.InputSchema["properties"].(map[string]interface{})["to"]; ok {
		t.Error("teammate send_message should not take a recipient")
	}
	ctx := withAgentName(context.Background(), "alice")
	if _, err := tool.Handler(ctx, map[string]interface{}{"content": "need a decision"}); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "the lead's mailbox", func() bool { return lead.team.bus.Peek(leadName) })
	notes := lead.team.consumeTeamEvents()
	if len(notes) != 1 || !strings.Contains(notes[0], `from="alice"`) || !strings.Contains(notes[0], "need a decision") {
		t.Errorf("events = %v", notes)
	}
}

func TestRunLoop_InjectsTeamEvents(t *testing.T) {
	useTasksDir(t)
	useTeam(t, testTeamConfig())
	llm := newTeamLLM()
	lead := NewAgent(llm, nil).(*agent)
	if err := lead.team.bus.Send(TeamMessage{From: "alice", To: leadName, Type: teamResult, Content: "did the thing", TaskID: "task_12345678"}); err != nil {
		t.Fatal(err)
	}

	if err := lead.RunLoop(context.Background(), userHi); err != nil {
		t.Fatal(err)
	}
	seen := renderConversation(llm.callsFor("main")[0])
	if !strings.Contains(seen, `<team_event from="alice" type="result" task="task_12345678">`) || !strings.Contains(seen, "did the thing") {
		t.Errorf("event not injected:\n%s", seen)
	}
	// subagents and teammates must not consume the lead's mailbox
	if sub := lead.newSubagent(); sub.team != lead.team {
		t.Error("subagent should share the runtime")
	} else {
		sub.messages = nil
		sub.injectTeamEvents()
		if len(sub.messages) != 0 {
			t.Error("a subagent must not inject the lead's team events")
		}
	}
}

func TestTeamEventsHook(t *testing.T) {
	useTasksDir(t)
	useTeam(t, testTeamConfig())
	llm := newTeamLLM()
	lead := NewAgent(llm, nil).(*agent)
	h := TeamEventsHook()

	// no team in ctx, or nothing happening: the lead may stop
	if msg, err := h(context.Background(), nil); msg != nil || err != nil {
		t.Errorf("no agent: %v, %v", msg, err)
	}
	ctx := withAgent(withAgentName(context.Background(), "main"), lead)
	if msg, err := h(ctx, nil); msg != nil || err != nil {
		t.Errorf("idle team: %v, %v", msg, err)
	}

	// a waiting event is delivered as the follow-up
	if err := lead.team.bus.Send(TeamMessage{From: "alice", To: leadName, Type: teamResult, Content: "done"}); err != nil {
		t.Fatal(err)
	}
	msg, err := h(ctx, nil)
	if err != nil || msg == nil {
		t.Fatalf("hook = %+v, %v", msg, err)
	}
	if blocks, _ := msg.Content.([]MessagesBlock); len(blocks) != 1 || !strings.Contains(blocks[0].Text, "done") {
		t.Errorf("follow-up = %+v", msg.Content)
	}
}
