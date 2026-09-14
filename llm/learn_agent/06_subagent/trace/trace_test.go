package trace

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	agentloop "github.com/liuliqiang/llmagent/06_subagent"
)

func newClockedRecorder() (*Recorder, *time.Time) {
	now := time.Date(2026, 9, 13, 10, 0, 0, 0, time.UTC)
	r := NewRecorder()
	r.now = func() time.Time { now = now.Add(time.Second); return now }
	return r, &now
}

func TestRecorder_CollectsRun(t *testing.T) {
	r, _ := newClockedRecorder()

	r.OnStart(agentloop.ModelDeepseekFlash, "be helpful", []agentloop.Message{
		{Role: agentloop.MessageRoleUser, Content: "list files"},
	})
	toolUse := agentloop.MessagesBlock{
		ID: "call_1", Type: agentloop.MessagesBlockTypeToolUse, Name: "run_bash",
		Input: map[string]interface{}{"command": "ls"},
	}
	r.OnResponse(1, agentloop.SendMessagesResponse{Content: []agentloop.MessagesBlock{
		{Type: agentloop.MessagesBlockTypeReasoning, Text: "need ls"},
		{Type: agentloop.MessagesBlockTypeText, Text: "checking"},
		toolUse,
	}})
	r.OnToolResult(1, toolUse, "a.txt\n", 1500*time.Millisecond)
	r.OnResponse(2, agentloop.SendMessagesResponse{Content: []agentloop.MessagesBlock{
		{Type: agentloop.MessagesBlockTypeText, Text: "done"},
	}})
	r.OnEnd(nil)

	tr := r.Trace()
	if tr.Model != "deepseek-flash" || tr.SystemPrompt != "be helpful" {
		t.Errorf("header = %+v", tr)
	}
	if tr.StartedAt.IsZero() || !tr.EndedAt.After(tr.StartedAt) {
		t.Errorf("timestamps = %v .. %v", tr.StartedAt, tr.EndedAt)
	}

	type want struct {
		turn int
		kind string
		text string
	}
	wants := []want{
		{0, "user", "list files"},
		{1, "reasoning", "need ls"},
		{1, "text", "checking"},
		{1, "tool_use", ""},
		{1, "tool_result", "a.txt\n"},
		{2, "text", "done"},
	}
	if len(tr.Events) != len(wants) {
		t.Fatalf("got %d events: %+v", len(tr.Events), tr.Events)
	}
	for i, w := range wants {
		e := tr.Events[i]
		if e.Seq != i+1 || e.Turn != w.turn || e.Kind != w.kind || e.Text != w.text {
			t.Errorf("event %d = %+v, want %+v", i, e, w)
		}
		if e.Time.IsZero() {
			t.Errorf("event %d has no time", i)
		}
	}

	tu := tr.Events[3]
	if tu.ToolName != "run_bash" || tu.ToolID != "call_1" || tu.ToolInput["command"] != "ls" {
		t.Errorf("tool_use = %+v", tu)
	}
	res := tr.Events[4]
	if res.ToolID != "call_1" || res.ToolName != "run_bash" || res.DurationMS != 1500 {
		t.Errorf("tool_result = %+v", res)
	}
}

func TestRecorder_OnEndWithError(t *testing.T) {
	r, _ := newClockedRecorder()
	r.OnStart("m", "s", []agentloop.Message{{Content: "hi"}})
	r.OnResponse(3, agentloop.SendMessagesResponse{Content: []agentloop.MessagesBlock{
		{Type: agentloop.MessagesBlockTypeText, Text: "x"},
	}})
	r.OnEnd(errors.New("llm down"))

	tr := r.Trace()
	last := tr.Events[len(tr.Events)-1]
	if last.Kind != "error" || last.Text != "llm down" || last.Turn != 3 {
		t.Errorf("last event = %+v", last)
	}
}

func TestRecorder_TraceIsACopy(t *testing.T) {
	r, _ := newClockedRecorder()
	r.OnStart("m", "s", []agentloop.Message{{Content: "hi"}})
	snap := r.Trace()
	r.OnEnd(nil)
	if len(snap.Events) != 1 || !snap.EndedAt.IsZero() {
		t.Errorf("snapshot changed after later calls: %+v", snap)
	}
}

func TestRenderHTML_EmbedsJSONAndEscapesScriptClose(t *testing.T) {
	tr := Trace{
		Model: "deepseek-flash",
		Events: []Event{
			{Seq: 1, Kind: "text", Text: "evil </script><script>alert(1)</script>"},
		},
	}
	html, err := RenderHTML(tr)
	if err != nil {
		t.Fatal(err)
	}
	s := string(html)

	if strings.Contains(s, jsonPlaceholder) {
		t.Error("placeholder not replaced")
	}
	if !strings.Contains(s, `"model":"deepseek-flash"`) {
		t.Error("trace JSON missing")
	}
	// the data block must not contain a raw closing tag; only the real one at the end may
	dataStart := strings.Index(s, `id="trace">`)
	dataEnd := strings.Index(s[dataStart:], "</script>") + dataStart
	block := s[dataStart:dataEnd]
	if !strings.Contains(block, `\u003c/script\u003e`) || strings.Contains(block, "</script>") {
		t.Errorf("script close not escaped in data block: %q", block)
	}
}

func TestRenderHTML_EmptyTrace(t *testing.T) {
	html, err := RenderHTML(Trace{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(html), `"events":null`) {
		t.Errorf("unexpected output: %s", html)
	}
}

func TestWriteHTML(t *testing.T) {
	r, _ := newClockedRecorder()
	r.OnStart("m", "s", []agentloop.Message{{Content: "hi"}})
	r.OnEnd(nil)

	path := filepath.Join(t.TempDir(), "trace.html")
	if err := r.WriteHTML(path); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(data), "<!doctype html>") || !strings.Contains(string(data), `"kind":"user"`) {
		t.Errorf("unexpected file content: %.200s", data)
	}

	if err := r.WriteHTML(filepath.Join(t.TempDir(), "missing", "trace.html")); err == nil {
		t.Error("expected error writing into a missing directory")
	}
}
