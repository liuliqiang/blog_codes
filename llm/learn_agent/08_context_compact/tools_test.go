package agentloop

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func in(kv ...interface{}) map[string]interface{} {
	m := map[string]interface{}{}
	for i := 0; i < len(kv); i += 2 {
		m[kv[i].(string)] = kv[i+1]
	}
	return m
}

func TestRunBash(t *testing.T) {
	ctx := context.Background()

	out, err := runBash(ctx, in("command", "echo hello"))
	if err != nil || out != "hello\n" {
		t.Errorf("got %q, %v", out, err)
	}

	_, err = runBash(ctx, in("command", "echo oops >&2; exit 3"))
	if err == nil || !strings.Contains(err.Error(), "oops") {
		t.Errorf("expected failure with stderr, got %v", err)
	}

	if _, err := runBash(ctx, in()); err == nil {
		t.Error("expected error for missing command")
	}
	if _, err := runBash(ctx, in("command", 1)); err == nil {
		t.Error("expected error for non-string command")
	}
}

func TestListDirectory(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "b.txt"), nil, 0o644)
	os.Mkdir(filepath.Join(dir, "a"), 0o755)

	out, err := listDirectory(context.Background(), in("path", dir))
	if err != nil || out != "a/\nb.txt" {
		t.Errorf("got %q, %v", out, err)
	}

	if _, err := listDirectory(context.Background(), in("path", filepath.Join(dir, "missing"))); err == nil {
		t.Error("expected error for missing dir")
	}
}

func TestReadWriteFile(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "f.txt")

	out, err := writeFile(ctx, in("path", path, "content", "hi there"))
	if err != nil || !strings.Contains(out, "8 bytes") {
		t.Errorf("write: got %q, %v", out, err)
	}

	out, err = readFile(ctx, in("path", path))
	if err != nil || out != "hi there" {
		t.Errorf("read: got %q, %v", out, err)
	}

	if _, err := readFile(ctx, in("path", path+".missing")); err == nil {
		t.Error("expected error for missing file")
	}
	if _, err := writeFile(ctx, in("path", path)); err == nil {
		t.Error("expected error for missing content")
	}
}

func TestEditFile(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "f.go")
	os.WriteFile(path, []byte("a\nfoo\nb\nfoo\n"), 0o644)

	// ambiguous: old_string appears twice
	_, err := editFile(ctx, in("path", path, "old_string", "foo", "new_string", "bar"))
	if err == nil || !strings.Contains(err.Error(), "found 2") {
		t.Errorf("expected ambiguity error, got %v", err)
	}

	// not found
	_, err = editFile(ctx, in("path", path, "old_string", "zzz", "new_string", "bar"))
	if err == nil || !strings.Contains(err.Error(), "found 0") {
		t.Errorf("expected not-found error, got %v", err)
	}

	// exact match
	if _, err := editFile(ctx, in("path", path, "old_string", "a\nfoo", "new_string", "a\nbar")); err != nil {
		t.Fatalf("edit: %v", err)
	}
	got, _ := os.ReadFile(path)
	if string(got) != "a\nbar\nb\nfoo\n" {
		t.Errorf("file = %q", got)
	}
}

func TestGrepFile(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "sub"), 0o755)
	os.MkdirAll(filepath.Join(dir, ".git"), 0o755)
	os.WriteFile(filepath.Join(dir, "f.txt"), []byte("alpha\nbeta\ngamma\nbetamax\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "sub", "g.txt"), []byte("beta too\n"), 0o644)
	os.WriteFile(filepath.Join(dir, ".git", "h.txt"), []byte("beta in git\n"), 0o644)

	// single file
	out, err := grepFile(ctx, in("path", filepath.Join(dir, "f.txt"), "pattern", "^beta"))
	want := filepath.Join(dir, "f.txt") + ":2: beta\n" + filepath.Join(dir, "f.txt") + ":4: betamax"
	if err != nil || out != want {
		t.Errorf("got %q, %v; want %q", out, err, want)
	}

	// directory: recursive, .git skipped
	out, err = grepFile(ctx, in("path", dir, "pattern", "^beta"))
	want = filepath.Join(dir, "f.txt") + ":2: beta\n" +
		filepath.Join(dir, "f.txt") + ":4: betamax\n" +
		filepath.Join(dir, "sub", "g.txt") + ":1: beta too"
	if err != nil || out != want {
		t.Errorf("got %q, %v; want %q", out, err, want)
	}

	// path omitted: defaults to the working directory
	t.Chdir(dir)
	out, err = grepFile(ctx, in("pattern", "gamma"))
	if err != nil || out != "f.txt:3: gamma" {
		t.Errorf("got %q, %v", out, err)
	}

	out, err = grepFile(ctx, in("path", dir, "pattern", "nothing"))
	if err != nil || out != "no matches" {
		t.Errorf("got %q, %v", out, err)
	}

	if _, err := grepFile(ctx, in("path", dir, "pattern", "(")); err == nil {
		t.Error("expected error for bad regexp")
	}
	if _, err := grepFile(ctx, in("path", filepath.Join(dir, "missing"), "pattern", "x")); err == nil {
		t.Error("expected error for missing path")
	}
}

func TestFindFile(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "sub"), 0o755)
	os.WriteFile(filepath.Join(dir, "x.go"), nil, 0o644)
	os.WriteFile(filepath.Join(dir, "sub", "y.go"), nil, 0o644)
	os.WriteFile(filepath.Join(dir, "sub", "z.txt"), nil, 0o644)

	out, err := findFile(ctx, in("path", dir, "pattern", "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, "sub", "y.go") + "\n" + filepath.Join(dir, "x.go")
	if out != want {
		t.Errorf("got %q, want %q", out, want)
	}

	out, err = findFile(ctx, in("path", dir, "pattern", "*.rs"))
	if err != nil || out != "no matches" {
		t.Errorf("got %q, %v", out, err)
	}

	if _, err := findFile(ctx, in("path", dir, "pattern", "[")); err == nil {
		t.Error("expected error for bad glob")
	}
}

func TestRunTools_DispatchAndRecover(t *testing.T) {
	a := &agent{toolIndex: map[string]Tool{
		"ok": {Name: "ok", Handler: func(_ context.Context, input map[string]interface{}) (string, error) {
			return "got " + input["x"].(string), nil
		}},
		"boom": {Name: "boom", Handler: func(context.Context, map[string]interface{}) (string, error) {
			return "", errors.New("kaboom")
		}},
	}}

	a.runTools(context.Background(), []MessagesBlock{
		{ID: "c1", Type: MessagesBlockTypeToolUse, Name: "ok", Input: in("x", "1")},
		{ID: "c2", Type: MessagesBlockTypeToolUse, Name: "boom"},
		{ID: "c3", Type: MessagesBlockTypeToolUse, Name: "nope"},
	})

	if len(a.messages) != 1 || a.messages[0].Role != MessageRoleUser {
		t.Fatalf("messages = %+v", a.messages)
	}
	want := []MessagesBlock{
		{Type: MessagesBlockTypeToolResult, ToolUseID: "c1", Content: "got 1"},
		{Type: MessagesBlockTypeToolResult, ToolUseID: "c2", Content: "error: kaboom"},
		{Type: MessagesBlockTypeToolResult, ToolUseID: "c3", Content: `error: unknown tool "nope"`},
	}
	if got := a.messages[0].Content; !reflect.DeepEqual(got, want) {
		t.Errorf("results = %+v, want %+v", got, want)
	}
}

func TestNewAgent_RegistersAllTools(t *testing.T) {
	a := NewAgent(nil, nil).(*agent)
	if len(a.tools) != len(a.toolIndex) {
		t.Errorf("tools=%d index=%d", len(a.tools), len(a.toolIndex))
	}
	for _, tool := range a.tools {
		if tool.Handler == nil {
			t.Errorf("tool %s has no handler", tool.Name)
		}
		if _, ok := a.toolIndex[tool.Name]; !ok {
			t.Errorf("tool %s not indexed", tool.Name)
		}
	}
}

// fakeLLM replies with a tool call on the first turn and plain text afterwards.
type fakeLLM struct {
	calls [][]Message
}

func (f *fakeLLM) GetModel() Model { return "fake" }

func (f *fakeLLM) SendMessages(_ context.Context, _ Model, _ Message, messages []Message, tools []Tool, _ SendMessagesOpts) (SendMessagesResponse, error) {
	f.calls = append(f.calls, append([]Message(nil), messages...))
	if len(f.calls) == 1 {
		return SendMessagesResponse{Content: []MessagesBlock{
			{ID: "call_1", Type: MessagesBlockTypeToolUse, Name: "run_bash", Input: in("command", "echo from tool")},
		}}, nil
	}
	return SendMessagesResponse{Content: []MessagesBlock{{Type: MessagesBlockTypeText, Text: "done"}}}, nil
}

func TestRunLoop_ToolRoundTrip(t *testing.T) {
	llm := &fakeLLM{}
	a := NewAgent(llm, nil)

	err := a.RunLoop(context.Background(), []Message{{Role: MessageRoleUser, Content: "hi"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(llm.calls) != 2 {
		t.Fatalf("expected 2 LLM calls, got %d", len(llm.calls))
	}

	// second call must carry: user, assistant(tool_use), user(tool_result)
	second := llm.calls[1]
	if len(second) != 3 {
		t.Fatalf("second call messages = %+v", second)
	}
	results, ok := second[2].Content.([]MessagesBlock)
	if !ok || len(results) != 1 || results[0].ToolUseID != "call_1" || results[0].Content != "from tool\n" {
		t.Errorf("tool result message = %+v", second[2])
	}
}

type fakeRecorder struct {
	calls []string
	err   error
}

func (r *fakeRecorder) OnStart(model Model, systemPrompt string, msgs []Message) {
	r.calls = append(r.calls, fmt.Sprintf("start:%s:%d", model, len(msgs)))
}
func (r *fakeRecorder) OnResponse(turn int, resp SendMessagesResponse) {
	r.calls = append(r.calls, fmt.Sprintf("response:%d:%d", turn, len(resp.Content)))
}
func (r *fakeRecorder) OnToolResult(turn int, toolUse MessagesBlock, output string, took time.Duration) {
	r.calls = append(r.calls, fmt.Sprintf("tool:%d:%s:%q", turn, toolUse.Name, output))
}
func (r *fakeRecorder) OnEnd(err error) {
	r.err = err
	r.calls = append(r.calls, "end")
}

func TestRunLoop_RecorderHooks(t *testing.T) {
	rec := &fakeRecorder{}
	a := NewAgent(&fakeLLM{}, nil, rec)

	if err := a.RunLoop(context.Background(), []Message{{Role: MessageRoleUser, Content: "hi"}}); err != nil {
		t.Fatal(err)
	}

	want := []string{
		"start:fake:1",
		"response:1:1",
		`tool:1:run_bash:"from tool\n"`,
		"response:2:1",
		"end",
	}
	if !reflect.DeepEqual(rec.calls, want) {
		t.Errorf("calls = %q, want %q", rec.calls, want)
	}
	if rec.err != nil {
		t.Errorf("OnEnd err = %v", rec.err)
	}
}

type failingLLM struct{}

func (failingLLM) GetModel() Model { return "fake" }
func (failingLLM) SendMessages(context.Context, Model, Message, []Message, []Tool, SendMessagesOpts) (SendMessagesResponse, error) {
	return SendMessagesResponse{}, errors.New("llm down")
}

func TestRunLoop_OnEndCalledOnError(t *testing.T) {
	rec := &fakeRecorder{}
	a := NewAgent(failingLLM{}, nil, rec)

	err := a.RunLoop(context.Background(), []Message{{Role: MessageRoleUser, Content: "hi"}})
	if err == nil || err.Error() != "llm down" {
		t.Fatalf("err = %v", err)
	}
	if rec.err == nil || rec.calls[len(rec.calls)-1] != "end" {
		t.Errorf("OnEnd not called with error: calls=%q err=%v", rec.calls, rec.err)
	}
}
