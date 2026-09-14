package agentloop

import (
	"bytes"
	"context"
	"reflect"
	"strings"
	"testing"
)

func todo(content string, status TodoStatus) map[string]interface{} {
	return map[string]interface{}{"content": content, "status": string(status)}
}

func TestTodoManager_UpdateAndRender(t *testing.T) {
	m := NewTodoManager()
	if got := m.Render(); got != "(no todos)" {
		t.Errorf("empty render = %q", got)
	}

	out, err := m.Update([]interface{}{
		todo("write tests", TodoCompleted),
		todo("implement feature", TodoInProgress),
		todo("update docs", TodoPending),
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "[x] write tests\n[>] implement feature\n[ ] update docs"
	if out != want {
		t.Errorf("render =\n%s\nwant\n%s", out, want)
	}

	// JSON string form is accepted too
	out, err = m.Update(`[{"content":"only one","status":"pending"}]`)
	if err != nil || out != "[ ] only one" {
		t.Errorf("string form: %q, %v", out, err)
	}

	// empty list clears
	out, err = m.Update([]interface{}{})
	if err != nil || out != "(no todos)" {
		t.Errorf("clear: %q, %v", out, err)
	}
}

func TestTodoManager_RejectsInvalidWithoutChangingList(t *testing.T) {
	m := NewTodoManager()
	if _, err := m.Update([]interface{}{todo("keep me", TodoPending)}); err != nil {
		t.Fatal(err)
	}
	before := m.Items()

	bad := []struct {
		name  string
		todos interface{}
		want  string
	}{
		{"empty content", []interface{}{todo("  ", TodoPending)}, "content must not be empty"},
		{"unknown status", []interface{}{todo("x", "done")}, "status must be one of"},
		{"second item bad", []interface{}{todo("ok", TodoPending), todo("x", "nope")}, "todo #2"},
		{"not an array", map[string]interface{}{"content": "x"}, "must be an array or a JSON string"},
		{"malformed json string", "[{", "JSON array"},
		{"wrong element type", []interface{}{"just a string"}, "JSON array of {content, status}"},
	}
	for _, c := range bad {
		_, err := m.Update(c.todos)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: err = %v, want containing %q", c.name, err, c.want)
		}
		if !reflect.DeepEqual(m.Items(), before) {
			t.Errorf("%s: list changed after rejected update: %+v", c.name, m.Items())
		}
	}
}

func TestRunTodoWrite_EchoesToTerminal(t *testing.T) {
	orig, origMgr := todoOut, todoManager
	var buf bytes.Buffer
	todoOut, todoManager = &buf, NewTodoManager()
	t.Cleanup(func() { todoOut, todoManager = orig, origMgr })

	out, err := runTodoWrite(context.Background(), map[string]interface{}{
		"todos": []interface{}{todo("a", TodoInProgress), todo("b", TodoPending)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != "[>] a\n[ ] b" || buf.String() != out+"\n" {
		t.Errorf("returned %q, printed %q", out, buf.String())
	}
	if len(todoManager.Items()) != 2 {
		t.Errorf("list not updated: %+v", todoManager.Items())
	}

	if _, err := runTodoWrite(context.Background(), map[string]interface{}{}); err == nil {
		t.Error("expected error for missing todos")
	}
}

func TestTodoWriteTool_Registered(t *testing.T) {
	a := NewAgent(nil, nil).(*agent)
	tool, ok := a.toolIndex["todo_write"]
	if !ok || tool.Handler == nil {
		t.Fatal("todo_write not registered")
	}
	props := tool.InputSchema["properties"].(map[string]interface{})
	if _, ok := props["todos"]; !ok {
		t.Errorf("schema missing todos: %+v", tool.InputSchema)
	}
}

func swapTodoManager(t *testing.T) *TodoManager {
	t.Helper()
	orig, origOut := todoManager, todoOut
	todoManager, todoOut = NewTodoManager(), &bytes.Buffer{}
	t.Cleanup(func() { todoManager, todoOut = orig, origOut })
	return todoManager
}

// reminderIn reports whether the last user message carries the todo reminder.
func reminderIn(msgs []Message) bool {
	blocks, _ := msgs[len(msgs)-1].Content.([]MessagesBlock)
	for _, b := range blocks {
		if b.Type == MessagesBlockTypeText && b.Text == todoReminderText {
			return true
		}
	}
	return false
}

func TestTodoReminder_NagsWhenListIsStale(t *testing.T) {
	m := swapTodoManager(t)
	if _, err := m.Update([]interface{}{todo("step 1", TodoInProgress)}); err != nil {
		t.Fatal(err)
	}

	// 4 tool rounds without todo_write, then a text reply
	llm := &scriptedLLM{responses: []SendMessagesResponse{
		bashCall("c1", "echo 1"), bashCall("c2", "echo 2"), bashCall("c3", "echo 3"), bashCall("c4", "echo 4"), text("done"),
	}}
	if err := NewAgent(llm, nil).RunLoop(context.Background(), userHi); err != nil {
		t.Fatal(err)
	}

	// calls[i] is what the LLM saw on round i+1; its last message holds round i's results
	for i, want := range []bool{false, false, true, false} {
		if got := reminderIn(llm.calls[i+1]); got != want {
			t.Errorf("after tool round %d: reminder=%v, want %v", i+1, got, want)
		}
	}
}

func TestTodoReminder_NeverNagsEmptyList(t *testing.T) {
	swapTodoManager(t)
	llm := &scriptedLLM{responses: []SendMessagesResponse{
		bashCall("c1", "echo 1"), bashCall("c2", "echo 2"), bashCall("c3", "echo 3"), bashCall("c4", "echo 4"), text("done"),
	}}
	if err := NewAgent(llm, nil).RunLoop(context.Background(), userHi); err != nil {
		t.Fatal(err)
	}
	for i := 1; i < len(llm.calls); i++ {
		if reminderIn(llm.calls[i]) {
			t.Errorf("round %d: reminder sent although the todo list is empty", i)
		}
	}
}

func TestTodoReminder_ResetByTodoWrite(t *testing.T) {
	swapTodoManager(t)
	todoCall := SendMessagesResponse{Content: []MessagesBlock{{
		ID: "t1", Type: MessagesBlockTypeToolUse, Name: "todo_write",
		Input: map[string]interface{}{"todos": []interface{}{todo("plan", TodoInProgress)}},
	}}}
	// todo_write, 2 rounds without it, todo_write again, 2 more rounds: never reaches 3
	llm := &scriptedLLM{responses: []SendMessagesResponse{
		todoCall, bashCall("c1", "echo 1"), bashCall("c2", "echo 2"),
		todoCall, bashCall("c3", "echo 3"), bashCall("c4", "echo 4"), text("done"),
	}}
	if err := NewAgent(llm, nil).RunLoop(context.Background(), userHi); err != nil {
		t.Fatal(err)
	}
	for i := 1; i < len(llm.calls); i++ {
		if reminderIn(llm.calls[i]) {
			t.Errorf("round %d: reminder sent although todo_write keeps resetting the counter", i)
		}
	}
}

func TestSystemPrompt_MentionsTodoWrite(t *testing.T) {
	if !strings.Contains(defaultSystemPrompt, "todo_write") || !strings.Contains(defaultSystemPrompt, "in_progress") {
		t.Errorf("system prompt should tell the model when and how to use todo_write:\n%s", defaultSystemPrompt)
	}
}
