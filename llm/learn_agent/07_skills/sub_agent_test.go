package agentloop

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func taskCall(id, prompt string) SendMessagesResponse {
	return SendMessagesResponse{Content: []MessagesBlock{
		{ID: id, Type: MessagesBlockTypeToolUse, Name: "task", Input: map[string]interface{}{"prompt": prompt}},
	}}
}

func toolNames(tools []Tool) []string {
	var names []string
	for _, t := range tools {
		names = append(names, t.Name)
	}
	return names
}

func TestTaskTool_Registered(t *testing.T) {
	a := NewAgent(nil, nil).(*agent)
	tool, ok := a.toolIndex["task"]
	if !ok {
		t.Fatalf("task tool not registered, have %v", toolNames(a.tools))
	}
	if !reflect.DeepEqual(tool.InputSchema["required"], []string{"prompt"}) {
		t.Errorf("required = %v, want [prompt]", tool.InputSchema["required"])
	}
	if !isAllowListed(MessagesBlock{Name: "task"}) {
		t.Error("task should be allow-listed; its inner tool calls are checked on their own")
	}
}

func TestNewSubagent(t *testing.T) {
	h := new(Hooks).OnPreToolUse(func(_ context.Context, tu MessagesBlock) (MessagesBlock, error) { return tu, nil })
	parent := NewAgent(&scriptedLLM{}, h, &fakeRecorder{}).(*agent)
	parent.messages = userHi

	sub := parent.newSubagent()

	if sub.llmClient != parent.llmClient {
		t.Error("subagent must share the parent's LLM client")
	}
	if len(sub.hooks.preToolUse) != 1 {
		t.Errorf("subagent hooks = %d, want parent's 1", len(sub.hooks.preToolUse))
	}
	if len(sub.recorders) != 0 || len(sub.messages) != 0 {
		t.Errorf("subagent must start fresh: recorders=%d messages=%d", len(sub.recorders), len(sub.messages))
	}
	if sub.maxLoop != subagentMaxLoop || !strings.HasPrefix(sub.systemPrompt, subagentSystemPrompt) {
		t.Errorf("maxLoop=%d systemPrompt=%q", sub.maxLoop, sub.systemPrompt)
	}
	names := toolNames(sub.tools)
	for _, excluded := range []string{"task", "todo_write"} {
		if _, ok := sub.toolIndex[excluded]; ok {
			t.Errorf("subagent must not get %s, have %v", excluded, names)
		}
	}
	if len(sub.tools) != len(parent.tools)-2 {
		t.Errorf("subagent tools = %v, want parent's minus task/todo_write", names)
	}
}

func TestTaskTool_ReturnsSubagentFinalText(t *testing.T) {
	llm := &scriptedLLM{responses: []SendMessagesResponse{
		taskCall("t1", "count the files"), // parent
		text("there are 3 files"),         // subagent
		text("done"),                      // parent
	}}

	if err := NewAgent(llm, nil).RunLoop(context.Background(), userHi); err != nil {
		t.Fatal(err)
	}

	if len(llm.calls) != 3 {
		t.Fatalf("LLM calls = %d, want 3", len(llm.calls))
	}
	// the subagent's call sees only the prompt, with its own system prompt and reduced tools
	if sub := llm.calls[1]; len(sub) != 1 || sub[0].Content != "count the files" {
		t.Errorf("subagent messages = %+v, want just the prompt", sub)
	}
	if !strings.HasPrefix(llm.systems[1].Content.(string), subagentSystemPrompt) || !strings.HasPrefix(llm.systems[0].Content.(string), defaultSystemPrompt) {
		t.Errorf("system prompts = %v / %v", llm.systems[0].Content, llm.systems[1].Content)
	}
	if names := toolNames(llm.tools[1]); strings.Contains(strings.Join(names, ","), "task") {
		t.Errorf("subagent was offered task: %v", names)
	}
	// the parent gets the subagent's answer as the tool result
	if res := lastToolResult(t, llm.calls[2]); res.ToolUseID != "t1" || res.Content != "there are 3 files" {
		t.Errorf("tool result = %+v", res)
	}
}

func TestTaskTool_SubagentUsesTools(t *testing.T) {
	autoAllow(t)
	llm := &scriptedLLM{responses: []SendMessagesResponse{
		taskCall("t1", "say hi"),     // parent
		bashCall("b1", "echo hello"), // subagent
		text("it said hello"),        // subagent
		text("done"),                 // parent
	}}

	if err := NewAgent(llm, nil).RunLoop(context.Background(), userHi); err != nil {
		t.Fatal(err)
	}
	if res := lastToolResult(t, llm.calls[2]); res.ToolUseID != "b1" || strings.TrimSpace(res.Content) != "hello" {
		t.Errorf("subagent bash result = %+v", res)
	}
	if res := lastToolResult(t, llm.calls[3]); res.Content != "it said hello" {
		t.Errorf("parent got %+v", res)
	}
}

func TestTaskTool_PermissionPromptNamesSubagent(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	origIn, origOut := promptIn, promptOut
	t.Cleanup(func() { promptIn, promptOut = origIn, origOut })
	var out bytes.Buffer
	promptIn, promptOut = yesReader{}, &out

	llm := &scriptedLLM{responses: []SendMessagesResponse{
		bashCall("b0", "cat a.txt"), // main agent, asks the user
		taskCall("t1", "do it"),     // main agent
		bashCall("b1", "cat a.txt"), // subagent, asks the user
		text("did it"),              // subagent
		text("done"),                // main agent
	}}
	h := new(Hooks).OnPreToolUse(PermissionHook())
	if err := NewAgent(llm, h).RunLoop(context.Background(), userHi); err != nil {
		t.Fatal(err)
	}

	prompts := out.String()
	if i, j := strings.Index(prompts, "[main] wants"), strings.Index(prompts, "[subagent] wants"); i < 0 || j < 0 || i > j {
		t.Errorf("want a [main] prompt followed by a [subagent] prompt, got: %q", prompts)
	}
}

func TestTaskTool_SubagentHooksInherited(t *testing.T) {
	llm := &scriptedLLM{responses: []SendMessagesResponse{
		taskCall("t1", "do it"),  // parent
		bashCall("b1", "echo x"), // subagent, denied by hook
		text("blocked"),          // subagent
		text("done"),             // parent
	}}
	h := new(Hooks).OnPreToolUse(func(_ context.Context, tu MessagesBlock) (MessagesBlock, error) {
		if tu.Name == "run_bash" {
			return tu, errors.New("no bash for you")
		}
		return tu, nil
	})

	if err := NewAgent(llm, h).RunLoop(context.Background(), userHi); err != nil {
		t.Fatal(err)
	}
	if res := lastToolResult(t, llm.calls[2]); !strings.Contains(res.Content, "no bash for you") {
		t.Errorf("parent hook did not reach subagent: %+v", res)
	}
}

func TestRunTask_Errors(t *testing.T) {
	t.Run("missing prompt", func(t *testing.T) {
		a := NewAgent(&scriptedLLM{}, nil).(*agent)
		if _, err := a.runTask(context.Background(), map[string]interface{}{}); err == nil {
			t.Error("want error for missing prompt")
		}
	})

	t.Run("llm failure propagates", func(t *testing.T) {
		a := NewAgent(&scriptedLLM{}, nil).(*agent) // no responses → error on first call
		_, err := a.runTask(context.Background(), map[string]interface{}{"prompt": "x"})
		if err == nil || !strings.Contains(err.Error(), "no more scripted responses") {
			t.Errorf("err = %v", err)
		}
	})

	t.Run("round limit hit", func(t *testing.T) {
		autoAllow(t)
		var responses []SendMessagesResponse
		for i := 0; i < subagentMaxLoop; i++ {
			responses = append(responses, bashCall("b", "echo loop"))
		}
		a := NewAgent(&scriptedLLM{responses: responses}, nil).(*agent)
		_, err := a.runTask(context.Background(), map[string]interface{}{"prompt": "x"})
		if err == nil || !strings.Contains(err.Error(), "without a final answer") {
			t.Errorf("err = %v", err)
		}
	})
}

func TestFinalText(t *testing.T) {
	assistant := func(blocks ...MessagesBlock) Message {
		return Message{Role: MessageRoleAssistant, Content: blocks}
	}
	cases := []struct {
		name string
		msgs []Message
		want string
		ok   bool
	}{
		{"no messages", nil, "", false},
		{"only user", userHi, "", false},
		{"text", append(userHi, assistant(MessagesBlock{Type: MessagesBlockTypeText, Text: "a"})), "a", true},
		{"reasoning skipped, texts joined", append(userHi, assistant(
			MessagesBlock{Type: MessagesBlockTypeReasoning, Text: "hmm"},
			MessagesBlock{Type: MessagesBlockTypeText, Text: "a"},
			MessagesBlock{Type: MessagesBlockTypeText, Text: "b"},
		)), "a\nb", true},
		{"cut off mid tool use", append(userHi, assistant(
			MessagesBlock{Type: MessagesBlockTypeText, Text: "let me check"},
			MessagesBlock{Type: MessagesBlockTypeToolUse, Name: "run_bash"},
		)), "", false},
		{"last assistant wins", []Message{
			assistant(MessagesBlock{Type: MessagesBlockTypeText, Text: "old"}),
			{Role: MessageRoleUser, Content: []MessagesBlock{{Type: MessagesBlockTypeToolResult}}},
			assistant(MessagesBlock{Type: MessagesBlockTypeText, Text: "new"}),
		}, "new", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := (&agent{messages: tc.msgs}).finalText()
			if got != tc.want || ok != tc.ok {
				t.Errorf("finalText() = %q, %v; want %q, %v", got, ok, tc.want, tc.ok)
			}
		})
	}
}
