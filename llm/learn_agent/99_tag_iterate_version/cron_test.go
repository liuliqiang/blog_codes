package agentloop

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// useCronPath points the cron store at a temp file, captures cron output and shortens the scheduler's poll.
func useCronPath(t *testing.T) (string, *bytes.Buffer) {
	t.Helper()
	path := filepath.Join(t.TempDir(), ".scheduled_tasks.json")
	origPath, origOut, origPoll := cronPath, cronOut, schedulerPollInterval
	out := &bytes.Buffer{}
	cronPath, cronOut, schedulerPollInterval = path, out, 20*time.Millisecond
	t.Cleanup(func() { cronPath, cronOut, schedulerPollInterval = origPath, origOut, origPoll })
	return path, out
}

func TestValidateCron(t *testing.T) {
	valid := []string{"* * * * *", "0 9 * * *", "*/5 * * * *", "0 9 * * 1-5", "0,30 8-18 * * 1,3,5", "  0   0  1  1  0 ", "*/1 * * * *"}
	for _, expr := range valid {
		if err := validateCron(expr); err != nil {
			t.Errorf("validateCron(%q) = %v, want nil", expr, err)
		}
	}
	invalid := map[string]string{
		"* * * *":       "expected 5 fields",
		"* * * * * *":   "expected 5 fields",
		"60 * * * *":    "minute: value 60 is outside",
		"* 24 * * *":    "hour: value 24",
		"* * 0 * *":     "day-of-month: value 0",
		"* * * 13 *":    "month: value 13",
		"* * * * 7":     "day-of-week: value 7",
		"*/0 * * * *":   "invalid step",
		"*/x * * * *":   "invalid step",
		"5-3 * * * *":   "start is greater than end",
		"50-70 * * * *": "outside",
		"a-b * * * *":   "invalid range",
		"abc * * * *":   "invalid field",
		"1,abc * * * *": "invalid field",
		"1,60 * * * *":  "outside",
	}
	for expr, want := range invalid {
		err := validateCron(expr)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("validateCron(%q) = %v, want error containing %q", expr, err, want)
		}
	}
}

func TestCronMatches(t *testing.T) {
	// Wednesday 2026-09-23 09:05, and Sunday 2026-09-27 00:00
	wed := time.Date(2026, 9, 23, 9, 5, 0, 0, time.Local)
	sun := time.Date(2026, 9, 27, 0, 0, 0, 0, time.Local)
	cases := []struct {
		expr string
		at   time.Time
		want bool
	}{
		{"* * * * *", wed, true},
		{"5 9 * * *", wed, true},
		{"0 9 * * *", wed, false},
		{"*/5 * * * *", wed, true},
		{"*/2 * * * *", wed, false},
		{"0-10 9 * * *", wed, true},
		{"1,3,5 9 * * *", wed, true},
		{"1,3 9 * * *", wed, false},
		{"5 9 * * 1-5", wed, true},  // weekday
		{"5 9 * * 0,6", wed, false}, // weekend only
		{"0 0 * * 0", sun, true},    // Sunday = 0
		{"5 9 23 * *", wed, true},   // day of month
		{"5 9 24 * *", wed, false},
		{"5 9 * 9 *", wed, true},
		{"5 9 * 10 *", wed, false},
		{"5 9 24 * 3", wed, true},  // dom fails, dow matches → either is enough
		{"5 9 23 * 4", wed, true},  // dom matches, dow fails
		{"5 9 24 * 4", wed, false}, // neither
		{"5 9 24 * *", wed, false}, // only dom restricted and it fails
		{"bad", wed, false},
	}
	for _, c := range cases {
		if got := cronMatches(c.expr, c.at); got != c.want {
			t.Errorf("cronMatches(%q, %s) = %v, want %v", c.expr, c.at.Format(time.RFC3339), got, c.want)
		}
	}
}

func TestCronStore_ScheduleCancelList(t *testing.T) {
	path, out := useCronPath(t)
	s := NewCronStore(path)

	if _, err := s.Schedule("bad", "x", true, true); err == nil {
		t.Error("invalid expression must be rejected")
	}
	if _, err := s.Schedule("* * * * *", "  ", true, true); err == nil {
		t.Error("blank prompt must be rejected")
	}
	if _, err := os.Stat(path); err == nil {
		t.Error("rejected jobs must not create the file")
	}

	durable, err := s.Schedule(" 0  9 * * * ", "run tests", true, true)
	if err != nil || !strings.HasPrefix(durable.ID, "cron_") || durable.Cron != "0 9 * * *" || !durable.Recurring || !durable.Durable {
		t.Fatalf("Schedule = %+v, %v", durable, err)
	}
	mem, err := s.Schedule("*/5 * * * *", "check ci", false, false)
	if err != nil {
		t.Fatal(err)
	}

	// only the durable job is on disk
	var saved []CronJob
	raw, err := os.ReadFile(path)
	if err != nil || json.Unmarshal(raw, &saved) != nil || len(saved) != 1 || saved[0].ID != durable.ID {
		t.Errorf("file = %s, %v", raw, err)
	}
	if got := s.List(); len(got) != 2 {
		t.Errorf("List = %+v", got)
	}

	if err := s.Cancel("cron_nope"); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("cancel unknown = %v", err)
	}
	if err := s.Cancel(mem.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.Cancel(durable.ID); err != nil {
		t.Fatal(err)
	}
	if len(s.List()) != 0 {
		t.Error("jobs remain after cancel")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("file should be removed once no durable job remains")
	}
	for _, want := range []string{"[cron] scheduled " + durable.ID, "[cron] cancelled " + mem.ID} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("out missing %q: %q", want, out.String())
		}
	}
}

func TestCronStore_LoadDurable(t *testing.T) {
	path, out := useCronPath(t)
	first := NewCronStore(path)
	job, err := first.Schedule("* * * * *", "durable one", true, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := first.Schedule("* * * * *", "memory only", true, false); err != nil {
		t.Fatal(err)
	}

	second := NewCronStore(path)
	got := second.List()
	if len(got) != 1 || got[0].ID != job.ID || got[0].Prompt != "durable one" {
		t.Errorf("reloaded = %+v", got)
	}
	if !strings.Contains(out.String(), "[cron] loaded 1 durable job(s)") {
		t.Errorf("out = %q", out.String())
	}

	// a job that was due but never acknowledged is queued again on load
	if err := os.WriteFile(path, []byte(`[{"id":"cron_deadbeef","cron":"0 9 * * *","prompt":"redo","recurring":true,"durable":true,"pending_delivery":true},
		{"id":"bad","cron":"0 9 * * *","prompt":"x","recurring":true,"durable":true},
		{"id":"cron_00000001","cron":"nope","prompt":"x","recurring":true,"durable":true}]`), 0o644); err != nil {
		t.Fatal(err)
	}
	third := NewCronStore(path)
	if third.loadErr != nil {
		t.Fatal(third.loadErr)
	}
	if queued := third.Consume(); len(queued) != 1 || queued[0].ID != "cron_deadbeef" {
		t.Errorf("pending job not re-queued: %+v", queued)
	}
	if len(third.List()) != 1 {
		t.Errorf("invalid saved jobs should be skipped: %+v", third.List())
	}

	// a corrupt file is remembered, not ignored
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if corrupt := NewCronStore(path); corrupt.loadErr == nil || !strings.Contains(corrupt.loadErr.Error(), "corrupt") {
		t.Errorf("loadErr = %v", corrupt.loadErr)
	}
}

func TestCronStore_PollAckRestore(t *testing.T) {
	path, out := useCronPath(t)
	s := NewCronStore(path)
	nine := time.Date(2026, 9, 23, 9, 0, 0, 0, time.Local)
	recurring, _ := s.Schedule("0 9 * * *", "recurring", true, true)
	once, _ := s.Schedule("0 9 * * *", "once", false, true)
	if _, err := s.Schedule("0 10 * * *", "later", true, false); err != nil {
		t.Fatal(err)
	}

	if due := s.PollDue(nine.Add(-time.Minute)); len(due) != 0 {
		t.Errorf("nothing should be due at 08:59: %+v", due)
	}
	due := s.PollDue(nine)
	if ids := jobIDs(due); !reflect.DeepEqual(sorted(ids), sorted([]string{recurring.ID, once.ID})) {
		t.Fatalf("due = %v", ids)
	}
	// same minute again: nothing new, and pending jobs are never queued twice
	if again := s.PollDue(nine.Add(30 * time.Second)); len(again) != 0 {
		t.Errorf("re-fired within the minute: %+v", again)
	}
	select {
	case <-s.wake:
	default:
		t.Error("PollDue should wake the scheduler")
	}
	// pending state reached disk before the queue
	if reloaded := NewCronStore(path).List(); !reloaded[0].PendingDelivery || reloaded[0].LastFired != "2026-09-23 09:00" {
		t.Errorf("pending not persisted: %+v", reloaded)
	}

	delivered := s.Consume()
	if len(delivered) != 2 || len(s.Consume()) != 0 {
		t.Fatalf("Consume = %+v", delivered)
	}

	// model unreachable: both go back to the queue, still pending
	s.Restore(delivered)
	if restored := s.Consume(); len(restored) != 2 {
		t.Errorf("Restore = %+v", restored)
	}

	// model accepted: one-shot removed, recurring cleared and eligible next minute-match
	if err := s.Acknowledge(delivered); err != nil {
		t.Fatal(err)
	}
	list := s.List()
	if len(list) != 2 {
		t.Fatalf("after ack = %+v", list)
	}
	for _, job := range list {
		if job.ID == once.ID {
			t.Error("one-shot job should be removed after acknowledgement")
		}
		if job.ID == recurring.ID && job.PendingDelivery {
			t.Error("recurring job should no longer be pending")
		}
	}
	if next := s.PollDue(nine.Add(24 * time.Hour)); len(next) != 1 || next[0].ID != recurring.ID {
		t.Errorf("next day due = %+v", next)
	}
	if !strings.Contains(out.String(), "[cron] due "+recurring.ID) {
		t.Errorf("out = %q", out.String())
	}
}

func jobIDs(jobs []CronJob) []string {
	var ids []string
	for _, j := range jobs {
		ids = append(ids, j.ID)
	}
	return ids
}

func TestCronTools(t *testing.T) {
	useCronPath(t)
	a := NewAgent(nil, nil).(*agent)
	ctx := withAgentName(context.Background(), "main")
	call := func(name string, input map[string]interface{}) (string, error) {
		return a.toolIndex[name].Handler(ctx, input)
	}
	for _, name := range []string{"schedule_cron", "list_crons", "cancel_cron"} {
		if _, ok := a.toolIndex[name]; !ok {
			t.Fatalf("%s not registered", name)
		}
		if !isAllowListed(MessagesBlock{Name: name}) {
			t.Errorf("%s should be allow-listed", name)
		}
	}
	if !strings.Contains(a.systemPrompt, "schedule_cron") {
		t.Error("system prompt lacks cron guidance")
	}
	if sub := a.newSubagent(); sub.cron != nil {
		t.Error("subagents must not schedule")
	} else if _, ok := sub.toolIndex["schedule_cron"]; ok {
		t.Error("subagent should not get the cron tools")
	}

	if out, err := call("list_crons", nil); err != nil || !strings.HasPrefix(out, "No cron jobs") {
		t.Errorf("empty list = %q, %v", out, err)
	}
	out, err := call("schedule_cron", map[string]interface{}{"cron": "*/2 * * * *", "prompt": "run date", "recurring": true, "durable": true})
	if err != nil || !strings.HasPrefix(out, "Scheduled cron_") || !strings.HasSuffix(out, ": */2 * * * * -> run date") {
		t.Fatalf("schedule = %q, %v", out, err)
	}
	id := strings.Fields(out)[1][:len("cron_deadbeef")]
	if _, err := call("schedule_cron", map[string]interface{}{"cron": "* * * *", "prompt": "x"}); err == nil {
		t.Error("bad cron should fail")
	}
	if _, err := call("schedule_cron", map[string]interface{}{"cron": "* * * * *", "prompt": "x", "recurring": "yes"}); err == nil {
		t.Error("non-bool recurring should fail")
	}
	if _, err := call("schedule_cron", map[string]interface{}{"prompt": "x"}); err == nil {
		t.Error("missing cron should fail")
	}
	// defaults: recurring + durable
	if _, err := call("schedule_cron", map[string]interface{}{"cron": "0 9 * * *", "prompt": "defaults"}); err != nil {
		t.Fatal(err)
	}
	out, _ = call("list_crons", nil)
	if !strings.Contains(out, id+"  */2 * * * *  [recurring, durable]  -> run date") || !strings.Contains(out, "[recurring, durable]  -> defaults") {
		t.Errorf("list = %q", out)
	}
	if out, err := call("cancel_cron", map[string]interface{}{"job_id": id}); err != nil || out != "Cancelled "+id {
		t.Errorf("cancel = %q, %v", out, err)
	}
	if _, err := call("cancel_cron", map[string]interface{}{"job_id": id}); err == nil {
		t.Error("cancelling twice should fail")
	}
}

func TestRunLoop_ScheduledDeliveryAckAndRestore(t *testing.T) {
	useCronPath(t)
	a := NewAgent(&scriptedLLM{responses: []SendMessagesResponse{text("did it")}}, nil).(*agent)
	job, _ := a.cron.Schedule("* * * * *", "run date", true, true)
	once, _ := a.cron.Schedule("* * * * *", "just once", false, true)
	delivered := a.cron.PollDue(time.Now())
	if len(delivered) != 2 {
		t.Fatalf("due = %+v", delivered)
	}
	a.cron.Consume()

	// model accepts: acknowledged (recurring cleared, one-shot gone), nothing back in the queue
	ctx := withCronDelivery(context.Background(), delivered)
	if err := a.RunLoop(ctx, []Message{userMsg("[Scheduled] run date")}); err != nil {
		t.Fatal(err)
	}
	if left := a.cron.List(); len(left) != 1 || left[0].ID != job.ID || left[0].PendingDelivery {
		t.Errorf("after accepted turn = %+v (once=%s)", left, once.ID)
	}
	if q := a.cron.Consume(); len(q) != 0 {
		t.Errorf("queue should be empty after ack: %+v", q)
	}

	// model unreachable: restored to the queue, still pending
	a.llmClient = failingLLM{}
	a.cron.PollDue(time.Now().Add(time.Minute))
	delivered = a.cron.Consume()
	if err := a.RunLoop(withCronDelivery(context.Background(), delivered), []Message{userMsg("[Scheduled] run date")}); err == nil {
		t.Fatal("want LLM error")
	}
	if q := a.cron.Consume(); len(q) != 1 || q[0].ID != job.ID || !q[0].PendingDelivery {
		t.Errorf("job not restored after failed turn: %+v", q)
	}

	// prompt rejected by a hook: also restored
	h := new(Hooks).OnUserPromptSubmit(func(context.Context, []Message) ([]Message, error) { return nil, errors.New("no") })
	b := NewAgent(&scriptedLLM{}, h).(*agent)
	b.cron = a.cron
	a.cron.Restore(delivered)
	delivered = a.cron.Consume()
	_ = b.RunLoop(withCronDelivery(context.Background(), delivered), userHi)
	if q := a.cron.Consume(); len(q) != 1 {
		t.Errorf("job not restored after rejected prompt: %+v", q)
	}

	// an ordinary turn never touches the store
	c := NewAgent(&scriptedLLM{responses: []SendMessagesResponse{text("hi")}}, nil).(*agent)
	c.cron = a.cron
	a.cron.Restore(delivered)
	if err := c.RunLoop(context.Background(), userHi); err != nil {
		t.Fatal(err)
	}
	if q := a.cron.Consume(); len(q) != 1 {
		t.Errorf("ordinary turn consumed the queue: %+v", q)
	}
}

func TestScheduledTurn_DeniesInteractivePermission(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	origIn, origOut := promptIn, promptOut
	t.Cleanup(func() { promptIn, promptOut = origIn, origOut })
	var out bytes.Buffer
	promptIn, promptOut = yesReader{}, &out // would say yes if asked

	tool := MessagesBlock{Name: "run_bash", Input: map[string]interface{}{"command": "cat a.txt"}}
	if allowed, _ := checkToolPermission(withNonInteractive(context.Background()), tool); allowed {
		t.Error("scheduled turn must deny a tool that needs approval")
	}
	if !strings.Contains(out.String(), "scheduled turn: denied") || strings.Contains(out.String(), "[y/N]") {
		t.Errorf("prompt = %q", out.String())
	}
	if allowed, _ := checkToolPermission(context.Background(), tool); !allowed {
		t.Error("interactive turn should still ask and accept y")
	}
	// allow-listed tools still run without asking
	if allowed, _ := checkToolPermission(withNonInteractive(context.Background()), MessagesBlock{Name: "run_bash", Input: map[string]interface{}{"command": "ls"}}); !allowed {
		t.Error("allow-listed command should not need approval in a scheduled turn")
	}
}

func TestScheduler_Run(t *testing.T) {
	useCronPath(t)
	llm := &scriptedLLM{responses: []SendMessagesResponse{text("scheduled done")}}
	ag := NewAgent(llm, nil)
	a := ag.(*agent)

	if _, err := NewScheduler(&fakeAgent{}); err == nil {
		t.Error("NewScheduler should reject agents without a store")
	}
	sched, err := NewScheduler(ag)
	if err != nil {
		t.Fatal(err)
	}
	if sched.HasJobs() {
		t.Error("no jobs yet")
	}
	job, err := a.cron.Schedule("* * * * *", "say hi", false, false) // due at the first poll
	if err != nil {
		t.Fatal(err)
	}
	if !sched.HasJobs() {
		t.Error("HasJobs should see the new job")
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- sched.Run(ctx) }()

	deadline := time.After(3 * time.Second)
	for len(llm.calls) == 0 {
		select {
		case <-deadline:
			t.Fatal("scheduled turn never ran")
		case <-time.After(10 * time.Millisecond):
		}
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatal(err)
	}

	if got := llm.calls[0]; len(got) != 1 || got[0].Content != "[Scheduled] say hi" {
		t.Errorf("scheduled turn messages = %+v", got)
	}
	if left := a.cron.List(); len(left) != 0 {
		t.Errorf("one-shot job should be gone after delivery: %+v (was %s)", left, job.ID)
	}
	if !isNonInteractive(withNonInteractive(context.Background())) {
		t.Error("withNonInteractive should mark the ctx")
	}
}

func TestScheduler_RunRefusesCorruptStore(t *testing.T) {
	path, _ := useCronPath(t)
	if err := os.WriteFile(path, []byte("nope"), 0o644); err != nil {
		t.Fatal(err)
	}
	sched, err := NewScheduler(NewAgent(nil, nil))
	if err != nil {
		t.Fatal(err)
	}
	if err := sched.Run(context.Background()); err == nil || !strings.Contains(err.Error(), "corrupt") {
		t.Errorf("Run = %v, want corrupt-store error", err)
	}
}

type fakeAgent struct{}

func (fakeAgent) RunLoop(context.Context, []Message) error { return nil }

// gateLLM answers like scriptedLLM but blocks inside SendMessages until released, so a test can hold a turn open.
type gateLLM struct {
	scriptedLLM
	started chan struct{}
	release chan struct{}
}

func (g *gateLLM) SendMessages(ctx context.Context, model Model, system Message, messages []Message, tools []Tool, opts SendMessagesOpts) (SendMessagesResponse, error) {
	select {
	case g.started <- struct{}{}:
	default:
	}
	<-g.release
	return g.scriptedLLM.SendMessages(ctx, model, system, messages, tools, opts)
}

func TestScheduler_UserTurnAndScheduledTurnDoNotOverlap(t *testing.T) {
	useCronPath(t)
	llm := &gateLLM{
		scriptedLLM: scriptedLLM{responses: []SendMessagesResponse{text("user turn done"), text("scheduled turn done")}},
		started:     make(chan struct{}, 1),
		release:     make(chan struct{}),
	}
	ag := NewAgent(llm, nil)
	sched, err := NewScheduler(ag)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ag.(*agent).cron.Schedule("* * * * *", "tick", false, false); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	schedDone := make(chan error, 1)
	go func() { schedDone <- sched.Run(ctx) }()

	// the user's turn starts and blocks inside the model call while the scheduler keeps polling
	turnDone := make(chan error, 1)
	go func() { turnDone <- sched.RunTurn(ctx, userHi) }()
	<-llm.started
	time.Sleep(5 * schedulerPollInterval) // long enough for the job to come due and be queued
	if q := sched.store.List(); len(q) != 1 || !q[0].PendingDelivery {
		t.Fatalf("job should be due and pending while the user turn runs: %+v", q)
	}
	if n := len(llm.calls); n != 0 {
		t.Fatalf("scheduled turn must not reach the model while the user turn holds the lock, calls=%d", n)
	}

	// release the user turn: it finishes, then the scheduled turn runs
	close(llm.release)
	if err := <-turnDone; err != nil {
		t.Fatal(err)
	}
	deadline := time.After(3 * time.Second)
	for len(llm.calls) < 2 {
		select {
		case <-deadline:
			t.Fatalf("scheduled turn never ran after the user turn, calls=%d", len(llm.calls))
		case <-time.After(10 * time.Millisecond):
		}
	}
	cancel()
	<-schedDone

	if llm.calls[0][0].Content != "hi" || llm.calls[1][0].Content != "[Scheduled] tick" || len(llm.calls[1]) != 1 {
		t.Errorf("turn order / isolation wrong: %+v", llm.calls)
	}
}
