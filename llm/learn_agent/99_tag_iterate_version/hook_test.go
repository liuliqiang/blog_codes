package agentloop

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

// scriptedLLM replays canned responses in order; it records every message
// list it was called with so tests can inspect what hooks changed.
type scriptedLLM struct {
	responses []SendMessagesResponse
	calls     [][]Message
	systems   []Message // system prompt of each call
	tools     [][]Tool  // tool set of each call
	models    []Model   // model of each call
}

func (s *scriptedLLM) GetModel() Model { return "fake" }

func (s *scriptedLLM) SendMessages(_ context.Context, model Model, system Message, messages []Message, tools []Tool, _ SendMessagesOpts) (SendMessagesResponse, error) {
	s.calls = append(s.calls, append([]Message(nil), messages...))
	s.models = append(s.models, model)
	s.systems = append(s.systems, system)
	s.tools = append(s.tools, tools)
	if len(s.calls) > len(s.responses) {
		return SendMessagesResponse{}, errors.New("no more scripted responses")
	}
	return s.responses[len(s.calls)-1], nil
}

func text(t string) SendMessagesResponse {
	return SendMessagesResponse{Content: []MessagesBlock{{Type: MessagesBlockTypeText, Text: t}}}
}

func bashCall(id, command string) SendMessagesResponse {
	return SendMessagesResponse{Content: []MessagesBlock{
		{ID: id, Type: MessagesBlockTypeToolUse, Name: "run_bash", Input: map[string]interface{}{"command": command}},
	}}
}

func lastToolResult(t *testing.T, msgs []Message) MessagesBlock {
	t.Helper()
	for i := len(msgs) - 1; i >= 0; i-- {
		if blocks, ok := msgs[i].Content.([]MessagesBlock); ok && len(blocks) > 0 && blocks[0].Type == MessagesBlockTypeToolResult {
			return blocks[0]
		}
	}
	t.Fatalf("no tool result in %+v", msgs)
	return MessagesBlock{}
}

var userHi = []Message{{Role: MessageRoleUser, Content: "hi"}}

func TestHooks_AppendOnlyAndSnapshot(t *testing.T) {
	var calls []string
	h := new(Hooks).
		OnPreToolUse(func(_ context.Context, tu MessagesBlock) (MessagesBlock, error) {
			calls = append(calls, "first")
			return tu, nil
		}).
		OnPreToolUse(func(_ context.Context, tu MessagesBlock) (MessagesBlock, error) {
			calls = append(calls, "second")
			return tu, nil
		})

	a := NewAgent(nil, h).(*agent)
	// registering after construction must not reach the agent
	h.OnPreToolUse(func(_ context.Context, tu MessagesBlock) (MessagesBlock, error) {
		calls = append(calls, "late")
		return tu, nil
	})

	if _, err := a.runPreToolUseHooks(context.Background(), MessagesBlock{}); err != nil {
		t.Fatal(err)
	}
	if strings.Join(calls, ",") != "first,second" {
		t.Errorf("calls = %v, want first,second in order and no late hook", calls)
	}

	// nil hooks are fine
	if n := NewAgent(nil, nil).(*agent); len(n.hooks.preToolUse) != 0 {
		t.Errorf("nil hooks should give empty set")
	}
}

func TestHook_UserPromptSubmit_Rewrite(t *testing.T) {
	llm := &scriptedLLM{responses: []SendMessagesResponse{text("ok")}}
	h := new(Hooks).OnUserPromptSubmit(func(_ context.Context, msgs []Message) ([]Message, error) {
		return append(msgs, Message{Role: MessageRoleUser, Content: "and be brief"}), nil
	})

	if err := NewAgent(llm, h).RunLoop(context.Background(), userHi); err != nil {
		t.Fatal(err)
	}
	if got := llm.calls[0]; len(got) != 2 || got[1].Content != "and be brief" {
		t.Errorf("LLM saw %+v, want rewritten prompt", got)
	}
}

func TestHook_UserPromptSubmit_Reject(t *testing.T) {
	llm := &scriptedLLM{responses: []SendMessagesResponse{text("ok")}}
	rec := &fakeRecorder{}
	h := new(Hooks).OnUserPromptSubmit(func(context.Context, []Message) ([]Message, error) {
		return nil, errors.New("prompt contains secrets")
	})

	err := NewAgent(llm, h, rec).RunLoop(context.Background(), userHi)
	if err == nil || !strings.Contains(err.Error(), "prompt contains secrets") {
		t.Fatalf("err = %v", err)
	}
	if len(llm.calls) != 0 {
		t.Error("LLM must not be called when the prompt is rejected")
	}
	if rec.calls[len(rec.calls)-1] != "end" || rec.err == nil {
		t.Errorf("OnEnd(err) not delivered: %v", rec.calls)
	}
}

func TestHook_PreToolUse_Deny(t *testing.T) {
	llm := &scriptedLLM{responses: []SendMessagesResponse{bashCall("c1", "echo hi"), text("ok")}}
	ran := false
	h := new(Hooks).OnPreToolUse(func(_ context.Context, tu MessagesBlock) (MessagesBlock, error) {
		return tu, errors.New("no shell today")
	}).OnPostToolUse(func(_ context.Context, _ MessagesBlock, out string) (string, error) {
		ran = true
		return out, nil
	})

	if err := NewAgent(llm, h).RunLoop(context.Background(), userHi); err != nil {
		t.Fatal(err)
	}
	res := lastToolResult(t, llm.calls[1])
	if res.ToolUseID != "c1" || !strings.Contains(res.Content, "denied by hook") || !strings.Contains(res.Content, "no shell today") {
		t.Errorf("tool result = %+v", res)
	}
	if ran {
		t.Error("PostToolUse must not run for a denied call")
	}
}

func TestHook_PreToolUse_ModifyInput(t *testing.T) {
	llm := &scriptedLLM{responses: []SendMessagesResponse{bashCall("c1", "echo original"), text("ok")}}
	h := new(Hooks).OnPreToolUse(func(_ context.Context, tu MessagesBlock) (MessagesBlock, error) {
		tu.Input = map[string]interface{}{"command": "echo modified"}
		return tu, nil
	})

	if err := NewAgent(llm, h).RunLoop(context.Background(), userHi); err != nil {
		t.Fatal(err)
	}
	if res := lastToolResult(t, llm.calls[1]); res.Content != "modified\n" {
		t.Errorf("tool ran with unmodified input: %q", res.Content)
	}
}

func TestHook_PermissionHookSeesModifiedInput(t *testing.T) {
	llm := &scriptedLLM{responses: []SendMessagesResponse{bashCall("c1", "echo safe"), text("ok")}}
	autoAllow(t)
	h := new(Hooks).OnPreToolUse(func(_ context.Context, tu MessagesBlock) (MessagesBlock, error) {
		tu.Input = map[string]interface{}{"command": "rm -rf ."}
		return tu, nil
	}).OnPreToolUse(PermissionHook())

	if err := NewAgent(llm, h).RunLoop(context.Background(), userHi); err != nil {
		t.Fatal(err)
	}
	if res := lastToolResult(t, llm.calls[1]); !strings.Contains(res.Content, "not allowed") {
		t.Errorf("permission rules must see the hook-modified input, got %q", res.Content)
	}
}

func TestHook_PostToolUse_RewriteAndError(t *testing.T) {
	rec := &fakeRecorder{}
	llm := &scriptedLLM{responses: []SendMessagesResponse{bashCall("c1", "echo hi"), text("ok")}}
	h := new(Hooks).OnPostToolUse(func(_ context.Context, _ MessagesBlock, out string) (string, error) {
		return "[redacted] " + strings.TrimSpace(out), nil
	})

	if err := NewAgent(llm, h, rec).RunLoop(context.Background(), userHi); err != nil {
		t.Fatal(err)
	}
	if res := lastToolResult(t, llm.calls[1]); res.Content != "[redacted] hi" {
		t.Errorf("model saw %q", res.Content)
	}
	if !strings.Contains(strings.Join(rec.calls, "|"), `"[redacted] hi"`) {
		t.Errorf("recorder should see the rewritten output: %v", rec.calls)
	}

	llm = &scriptedLLM{responses: []SendMessagesResponse{bashCall("c1", "echo hi"), text("ok")}}
	h = new(Hooks).OnPostToolUse(func(context.Context, MessagesBlock, string) (string, error) {
		return "", errors.New("output too large")
	})
	if err := NewAgent(llm, h).RunLoop(context.Background(), userHi); err != nil {
		t.Fatal(err)
	}
	if res := lastToolResult(t, llm.calls[1]); res.Content != "error: output too large" {
		t.Errorf("model saw %q", res.Content)
	}
}

func TestHook_Stop_ContinueOnce(t *testing.T) {
	llm := &scriptedLLM{responses: []SendMessagesResponse{text("draft"), text("final")}}
	continued := false
	h := new(Hooks).OnStop(func(_ context.Context, msgs []Message) (*Message, error) {
		if continued {
			return nil, nil
		}
		continued = true
		return &Message{Role: MessageRoleUser, Content: "please double-check"}, nil
	})

	if err := NewAgent(llm, h).RunLoop(context.Background(), userHi); err != nil {
		t.Fatal(err)
	}
	if len(llm.calls) != 2 {
		t.Fatalf("expected 2 LLM calls, got %d", len(llm.calls))
	}
	second := llm.calls[1]
	if last := second[len(second)-1]; last.Role != MessageRoleUser || last.Content != "please double-check" {
		t.Errorf("follow-up not injected: %+v", second)
	}
}

func TestHook_Stop_Error(t *testing.T) {
	llm := &scriptedLLM{responses: []SendMessagesResponse{text("done")}}
	h := new(Hooks).OnStop(func(context.Context, []Message) (*Message, error) {
		return nil, errors.New("stop rejected")
	})
	err := NewAgent(llm, h).RunLoop(context.Background(), userHi)
	if err == nil || err.Error() != "stop rejected" {
		t.Errorf("err = %v", err)
	}
}

func TestHook_ChainAndShortCircuit(t *testing.T) {
	var order []string
	h := new(Hooks).
		OnPostToolUse(func(_ context.Context, _ MessagesBlock, out string) (string, error) {
			order = append(order, "a")
			return out + "a", nil
		}).
		OnPostToolUse(func(_ context.Context, _ MessagesBlock, out string) (string, error) {
			order = append(order, "b")
			return out + "b", errors.New("stop here")
		}).
		OnPostToolUse(func(_ context.Context, _ MessagesBlock, out string) (string, error) {
			order = append(order, "c")
			return out + "c", nil
		})
	a := NewAgent(nil, h).(*agent)

	out, err := a.runPostToolUseHooks(context.Background(), MessagesBlock{}, "x")
	if err == nil || strings.Join(order, "") != "ab" {
		t.Errorf("order = %v err = %v", order, err)
	}
	if out != "xa" {
		t.Errorf("on error the last good output should be kept, got %q", out)
	}
}

func TestHook_PanicBecomesError(t *testing.T) {
	h := new(Hooks).OnPreToolUse(func(context.Context, MessagesBlock) (MessagesBlock, error) {
		panic("boom")
	})
	a := NewAgent(nil, h).(*agent)

	_, err := a.runPreToolUseHooks(context.Background(), MessagesBlock{Name: "run_bash"})
	if err == nil || !strings.Contains(err.Error(), "panicked") || !strings.Contains(err.Error(), "boom") {
		t.Errorf("err = %v", err)
	}
}

func captureHookOut(t *testing.T) *bytes.Buffer {
	t.Helper()
	orig := hookOut
	buf := &bytes.Buffer{}
	hookOut = buf
	t.Cleanup(func() { hookOut = orig })
	return buf
}

func TestPermissionHook(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	autoAllow(t)
	hook := PermissionHook()

	tu := MessagesBlock{Name: "run_bash", Input: map[string]interface{}{"command": "ls"}}
	if out, err := hook(context.Background(), tu); err != nil || out.Name != "run_bash" {
		t.Errorf("allowed call: out=%+v err=%v", out, err)
	}

	tu = MessagesBlock{Name: "write_file", Input: map[string]interface{}{"path": "/etc/passwd", "content": "x"}}
	if _, err := hook(context.Background(), tu); err == nil || !strings.Contains(err.Error(), "WORKSPACE") {
		t.Errorf("rule denial should surface the rule message, got %v", err)
	}

	// end to end: denial reaches the model as a tool result, loop continues
	llm := &scriptedLLM{responses: []SendMessagesResponse{bashCall("c1", "rm -rf ."), text("ok")}}
	if err := NewAgent(llm, new(Hooks).OnPreToolUse(PermissionHook())).RunLoop(context.Background(), userHi); err != nil {
		t.Fatal(err)
	}
	if res := lastToolResult(t, llm.calls[1]); !strings.Contains(res.Content, "denied by hook") || !strings.Contains(res.Content, "not allowed") {
		t.Errorf("tool result = %q", res.Content)
	}
}

func TestLogToolUseHook(t *testing.T) {
	out := captureHookOut(t)
	tu := MessagesBlock{Name: "read_file", Input: map[string]interface{}{"path": "a.txt"}}

	got, err := LogToolUseHook()(context.Background(), tu)
	if err != nil || !reflect.DeepEqual(got, tu) {
		t.Errorf("hook must pass the call through unchanged: %+v %v", got, err)
	}
	if out.String() != "[HOOK] read_file(...)\n" {
		t.Errorf("printed %q", out.String())
	}
}

func TestLargeOutputHook(t *testing.T) {
	out := captureHookOut(t)
	tu := MessagesBlock{Name: "run_bash"}
	hook := LargeOutputHook()

	small := strings.Repeat("x", largeOutputThreshold)
	if got, err := hook(context.Background(), tu, small); err != nil || got != small {
		t.Errorf("small output changed: len=%d err=%v", len(got), err)
	}
	if out.Len() != 0 {
		t.Errorf("no warning expected at the threshold, got %q", out.String())
	}

	large := small + "x"
	if got, err := hook(context.Background(), tu, large); err != nil || got != large {
		t.Errorf("large output must not be modified: len=%d err=%v", len(got), err)
	}
	if !strings.Contains(out.String(), "[HOOK] \u26a0 Large output from run_bash") {
		t.Errorf("printed %q", out.String())
	}
}

func TestSummaryHook(t *testing.T) {
	out := captureHookOut(t)
	toolResult := func(id string) MessagesBlock {
		return MessagesBlock{Type: MessagesBlockTypeToolResult, ToolUseID: id, Content: "ok"}
	}
	messages := []Message{
		{Role: MessageRoleUser, Content: "hi"},
		{Role: MessageRoleAssistant, Content: []MessagesBlock{
			{Type: MessagesBlockTypeText, Text: "running"},
			{ID: "c1", Type: MessagesBlockTypeToolUse, Name: "run_bash"},
			{ID: "c2", Type: MessagesBlockTypeToolUse, Name: "read_file"},
		}},
		{Role: MessageRoleUser, Content: []MessagesBlock{toolResult("c1"), toolResult("c2")}},
		{Role: MessageRoleAssistant, Content: []MessagesBlock{{ID: "c3", Type: MessagesBlockTypeToolUse, Name: "run_bash"}}},
		{Role: MessageRoleUser, Content: []MessagesBlock{toolResult("c3")}},
		{Role: MessageRoleAssistant, Content: []MessagesBlock{{Type: MessagesBlockTypeText, Text: "done"}}},
	}

	followUp, err := SummaryHook()(context.Background(), messages)
	if err != nil || followUp != nil {
		t.Errorf("summary hook must always allow the stop: followUp=%v err=%v", followUp, err)
	}
	if out.String() != "\033[90m[HOOK] Stop: session used 3 tool calls\033[0m\n" {
		t.Errorf("printed %q", out.String())
	}

	// end to end: runs once at the end of a run and doesn't keep the loop going
	out.Reset()
	llm := &scriptedLLM{responses: []SendMessagesResponse{bashCall("c1", "echo hi"), text("ok")}}
	if err := NewAgent(llm, new(Hooks).OnStop(SummaryHook())).RunLoop(context.Background(), userHi); err != nil {
		t.Fatal(err)
	}
	if len(llm.calls) != 2 || !strings.Contains(out.String(), "session used 1 tool calls") {
		t.Errorf("calls=%d printed %q", len(llm.calls), out.String())
	}
}
