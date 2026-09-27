package agentloop

import (
	"context"
	"strings"
	"testing"
)

func verdict(json string) SendMessagesResponse { return text(json) }

func goalAgent(t *testing.T, responses ...SendMessagesResponse) (*agent, *scriptedLLM) {
	t.Helper()
	llm := &scriptedLLM{responses: responses}
	return NewAgent(llm, new(Hooks).OnStop(GoalHook())).(*agent), llm
}

func runPrompt(t *testing.T, a *agent, prompt string) {
	t.Helper()
	if err := a.RunLoop(context.Background(), []Message{{Role: MessageRoleUser, Content: prompt}}); err != nil {
		t.Fatal(err)
	}
}

func TestParseGoalEvaluation(t *testing.T) {
	cases := []struct {
		name    string
		reply   string
		want    GoalEvaluation
		wantErr string
	}{
		{"not met", `{"ok": false, "reason": "no test output yet", "impossible": false}`, GoalEvaluation{Reason: "no test output yet"}, ""},
		{"met in a fence", "```json\n{\"ok\": true, \"reason\": \"exit 0\"}\n```", GoalEvaluation{OK: true, Reason: "exit 0"}, ""},
		{"impossible", `{"ok": false, "reason": "file is gone", "impossible": true}`, GoalEvaluation{Reason: "file is gone", Impossible: true}, ""},
		{"prose", "looks done to me", GoalEvaluation{}, "invalid JSON"},
		{"missing ok", `{"reason": "x"}`, GoalEvaluation{}, "boolean 'ok'"},
		{"empty reason", `{"ok": true, "reason": "  "}`, GoalEvaluation{}, "non-empty 'reason'"},
		{"ok and impossible", `{"ok": true, "reason": "x", "impossible": true}`, GoalEvaluation{}, "both ok and impossible"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := parseGoalEvaluation(c.reply)
			if c.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), c.wantErr) {
					t.Fatalf("err = %v, want %q", err, c.wantErr)
				}
				return
			}
			if err != nil || got != c.want {
				t.Fatalf("got %+v, %v; want %+v", got, err, c.want)
			}
		})
	}
}

func TestGoalTranscript(t *testing.T) {
	messages := []Message{
		{Role: MessageRoleUser, Content: "old message that will not fit"},
		{Role: MessageRoleAssistant, Content: []MessagesBlock{{Type: MessagesBlockTypeToolUse, Name: "run_bash", Input: map[string]interface{}{"command": "go test"}}}},
		{Role: MessageRoleUser, Content: []MessagesBlock{{Type: MessagesBlockTypeToolResult, Content: "ok  pkg 0.1s"}}},
	}
	got := goalTranscript(messages, 100)
	if strings.Contains(got, "old message") {
		t.Errorf("oldest message should be dropped: %q", got)
	}
	if !strings.Contains(got, `[tool_use run_bash {"command":"go test"}]`) || !strings.HasSuffix(got, "[tool_result] ok  pkg 0.1s\n") {
		t.Errorf("recent messages missing or out of order: %q", got)
	}

	// a newest message larger than the budget keeps its head and tail
	huge := []Message{{Role: MessageRoleUser, Content: "HEAD" + strings.Repeat("x", 500) + "exit status 0"}}
	got = goalTranscript(huge, 100)
	if len(got) != 100 || !strings.Contains(got, "HEAD") || !strings.Contains(got, "middle omitted") || !strings.HasSuffix(got, "exit status 0\n") {
		t.Errorf("oversized message not trimmed to head and tail: %d %q", len(got), got)
	}
}

func TestGoalCommandFrom(t *testing.T) {
	for prompt, want := range map[string]bool{
		"/goal":            true,
		"  /goal go test ": true,
		"/goals":           false,
		"set a /goal":      false,
	} {
		if _, ok := goalCommandFrom([]Message{{Role: MessageRoleUser, Content: prompt}}); ok != want {
			t.Errorf("%q: command = %v, want %v", prompt, ok, want)
		}
	}
	if _, ok := goalCommandFrom([]Message{{Role: MessageRoleUser, Content: "/goal a"}, {Role: MessageRoleUser, Content: "b"}}); ok {
		t.Error("a multi-message run is not a command")
	}
}

func TestGoalController_Commands(t *testing.T) {
	g := newGoalController(nil)
	if _, reply, _ := g.handleCommand("/goal"); reply != "No goal set" {
		t.Errorf("status without goal = %q", reply)
	}
	work, _, err := g.handleCommand("/goal   go test ./... exits 0 ")
	if err != nil || work != "go test ./... exits 0" {
		t.Fatalf("set: work=%q err=%v", work, err)
	}
	if _, reply, _ := g.handleCommand("/goal"); !strings.Contains(reply, "Goal active: go test ./... exits 0") || !strings.Contains(reply, "Evaluations: 0") {
		t.Errorf("status = %q", reply)
	}
	// replacing is just setting again
	if work, _, _ := g.handleCommand("/goal lint is clean"); work != "lint is clean" || g.active.condition != "lint is clean" {
		t.Errorf("replace: work=%q active=%+v", work, g.active)
	}
	if _, reply, _ := g.handleCommand("/goal CANCEL"); reply != "Goal cleared: lint is clean" || g.active != nil {
		t.Errorf("clear alias: reply=%q active=%+v", reply, g.active)
	}
	if _, _, err := g.handleCommand("/goal " + strings.Repeat("x", maxGoalLength+1)); err == nil {
		t.Error("an over-long condition should be rejected")
	}
}

func TestGoal_BlocksUntilEvaluatorIsSatisfied(t *testing.T) {
	out := captureHookOut(t)
	a, llm := goalAgent(t,
		text("I think it's done"),
		verdict(`{"ok": false, "reason": "no go test result in the conversation", "impossible": false}`),
		bashCall("c1", "echo ok"),
		text("go test printed ok"),
		verdict(`{"ok": true, "reason": "the test output shows ok"}`),
	)
	runPrompt(t, a, "/goal go test passes")

	if len(llm.calls) != 5 {
		t.Fatalf("expected 5 LLM calls, got %d", len(llm.calls))
	}
	if first := llm.calls[0]; len(first) != 1 || first[0].Content != "go test passes" {
		t.Errorf("the worker should start on the condition itself, got %+v", first)
	}
	// the evaluator is a separate, tool-free call that reads the transcript
	eval := llm.calls[1]
	if llm.tools[1] != nil || llm.systems[1].Content != goalEvaluatorSystemPrompt || len(eval) != 1 {
		t.Errorf("evaluator call: tools=%v system=%v messages=%d", llm.tools[1], llm.systems[1].Content, len(eval))
	}
	if prompt := eval[0].Content.(string); !strings.Contains(prompt, `"completion_condition":"go test passes"`) || !strings.Contains(prompt, "I think it's done") {
		t.Errorf("evaluator input = %q", prompt)
	}
	// the block reason goes back to the worker in the same loop
	resumed := llm.calls[2]
	if hint, _ := resumed[len(resumed)-1].Content.(string); !strings.Contains(hint, "[Goal still active]") || !strings.Contains(hint, "no go test result") {
		t.Errorf("continuation = %q", hint)
	}
	if a.goal.active != nil || !strings.Contains(a.goal.status(), "Goal achieved: go test passes") {
		t.Errorf("status after success = %q", a.goal.status())
	}
	if !strings.Contains(out.String(), "[GOAL] achieved") {
		t.Errorf("printed %q", out.String())
	}
}

func TestGoal_ImpossibleEndsTheGoal(t *testing.T) {
	captureHookOut(t)
	a, llm := goalAgent(t,
		text("the service does not exist"),
		verdict(`{"ok": false, "reason": "the service was removed", "impossible": true}`),
	)
	runPrompt(t, a, "/goal the service answers on :8080")
	if len(llm.calls) != 2 || a.goal.active != nil || !strings.Contains(a.goal.status(), "Goal failed") {
		t.Errorf("calls=%d status=%q", len(llm.calls), a.goal.status())
	}
}

func TestGoal_EvaluatorErrorKeepsGoalActive(t *testing.T) {
	out := captureHookOut(t)
	a, llm := goalAgent(t, text("done"), text("sure, looks good"))
	runPrompt(t, a, "/goal go test passes")
	if len(llm.calls) != 2 {
		t.Fatalf("an unreadable verdict must not continue the loop, calls=%d", len(llm.calls))
	}
	if a.goal.active == nil || !strings.Contains(a.goal.status(), "invalid JSON") {
		t.Errorf("goal should stay active with the error as its last reason: %q", a.goal.status())
	}
	if !strings.Contains(out.String(), "evaluation failed") {
		t.Errorf("printed %q", out.String())
	}
}

func TestGoal_BlockCapReturnsControlAndResetsPerRun(t *testing.T) {
	orig := Goal
	Goal.BlockCap = 1
	t.Cleanup(func() { Goal = orig })
	out := captureHookOut(t)
	notYet := verdict(`{"ok": false, "reason": "still failing"}`)
	a, llm := goalAgent(t,
		text("try 1"), notYet, // blocked once: within the cap
		text("try 2"), notYet, // blocked again: over the cap, control returns
		text("try 3"), notYet, // a new run gets a fresh cap
		text("try 4"), notYet,
	)
	runPrompt(t, a, "/goal go test passes")
	if len(llm.calls) != 4 || a.goal.active == nil || a.goal.active.evaluations != 2 {
		t.Fatalf("calls=%d active=%+v", len(llm.calls), a.goal.active)
	}
	if !strings.Contains(out.String(), "blocked 1 consecutive stops") {
		t.Errorf("printed %q", out.String())
	}

	runPrompt(t, a, "keep going")
	if len(llm.calls) != 8 || a.goal.active == nil || a.goal.active.evaluations != 4 {
		t.Errorf("second run: calls=%d active=%+v", len(llm.calls), a.goal.active)
	}
}

func TestGoal_NoGoalStopsWithoutEvaluating(t *testing.T) {
	a, llm := goalAgent(t, text("hi there"))
	runPrompt(t, a, "hi")
	if len(llm.calls) != 1 {
		t.Errorf("no goal should mean no evaluator call, calls=%d", len(llm.calls))
	}
}

func TestGoal_StatusAndClearRunNoTurn(t *testing.T) {
	out := captureHookOut(t)
	a, llm := goalAgent(t)
	if err := a.goal.set("go test passes"); err != nil {
		t.Fatal(err)
	}
	runPrompt(t, a, "/goal")
	runPrompt(t, a, "/goal clear")
	if len(llm.calls) != 0 {
		t.Errorf("status and clear must not call the model, calls=%d", len(llm.calls))
	}
	if !strings.Contains(out.String(), "[GOAL] Goal active: go test passes") || !strings.Contains(out.String(), "[GOAL] Goal cleared: go test passes") {
		t.Errorf("printed %q", out.String())
	}

	if err := a.RunLoop(context.Background(), []Message{{Role: MessageRoleUser, Content: "/goal  "}}); err != nil {
		t.Errorf("a bare /goal with spaces is a status query, got %v", err)
	}
}

func TestGoalHook_IgnoresAgentsWithoutGoal(t *testing.T) {
	sub := &agent{name: "sub"}
	msg, err := GoalHook()(withAgent(context.Background(), sub), nil)
	if msg != nil || err != nil {
		t.Errorf("msg=%v err=%v", msg, err)
	}
}
