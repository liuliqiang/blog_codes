package agentloop

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// useCompaction swaps in a config for the test.
func useCompaction(t *testing.T, cfg CompactionConfig) {
	t.Helper()
	orig := compaction
	compaction = cfg
	t.Cleanup(func() { compaction = orig })
}

func userMsg(s string) Message { return Message{Role: MessageRoleUser, Content: s} }
func assistantMsg(s string) Message {
	return Message{Role: MessageRoleAssistant, Content: []MessagesBlock{{Type: MessagesBlockTypeText, Text: s}}}
}
func toolUseMsg(id, name, path string) Message {
	return Message{Role: MessageRoleAssistant, Content: []MessagesBlock{
		{ID: id, Type: MessagesBlockTypeToolUse, Name: name, Input: map[string]interface{}{"path": path}},
	}}
}
func toolResultMsg(id, out string) Message {
	return Message{Role: MessageRoleUser, Content: []MessagesBlock{{Type: MessagesBlockTypeToolResult, ToolUseID: id, Content: out}}}
}

// withUsage attaches token usage to a scripted response.
func withUsage(resp SendMessagesResponse, in, out int) SendMessagesResponse {
	resp.Usage = Usage{InputTokens: in, OutputTokens: out}
	return resp
}

func TestEstimateTokens(t *testing.T) {
	forty := strings.Repeat("x", 40)
	cases := []struct {
		name string
		msg  Message
		want int
	}{
		{"string", userMsg(forty), 10},
		{"text block", assistantMsg(forty), 10},
		{"tool use counts input json", toolUseMsg("1", "read_file", forty), (len(`{"path":"`+forty+`"}`) + 3) / 4},
		{"tool result counts content", toolResultMsg("1", forty), 10},
		{"empty", userMsg(""), 0},
		{"rounds up", userMsg("abcde"), 2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := estimateTokens(c.msg); got != c.want {
				t.Errorf("estimateTokens = %d, want %d", got, c.want)
			}
		})
	}
	if got := estimateTokens(userMsg(forty), userMsg(forty)); got != 20 {
		t.Errorf("variadic sum = %d, want 20", got)
	}
}

func TestFindCutPoint(t *testing.T) {
	big := strings.Repeat("x", 400) // 100 tokens
	msgs := []Message{
		userMsg(big),                      // 0
		toolUseMsg("a", "read_file", big), // 1
		toolResultMsg("a", big),           // 2
		assistantMsg(big),                 // 3
		toolUseMsg("b", "read_file", big), // 4
		toolResultMsg("b", big),           // 5
		assistantMsg(big),                 // 6
	}

	cases := []struct {
		name string
		keep int
		want int
	}{
		{"keep last two", 150, 5 - 1},      // lands on tool_result 5 → back to 4
		{"keep exactly last", 100, 6},      // 6 is plain text, no move
		{"lands on tool_result 2", 450, 1}, // 6,5,4,3 = 400 < 450 → 2 is tool_result → 1
		{"everything fits", 100_000, 0},
		{"zero keep still avoids orphan", 0, 6},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := findCutPoint(msgs, c.keep); got != c.want {
				t.Errorf("findCutPoint(keep=%d) = %d, want %d", c.keep, got, c.want)
			}
		})
	}

	if got := findCutPoint(nil, 10); got != 0 {
		t.Errorf("empty = %d", got)
	}
	// a run of tool results with no assistant message before them walks back to 0
	orphan := []Message{toolResultMsg("x", big), toolResultMsg("y", big)}
	if got := findCutPoint(orphan, 100); got != 0 {
		t.Errorf("all tool results before cut = %d, want 0", got)
	}
}

func TestShouldCompact(t *testing.T) {
	cases := []struct {
		name  string
		cfg   CompactionConfig
		usage Usage
		want  bool
	}{
		{"disabled", CompactionConfig{ContextWindow: 0, ReserveTokens: 1, KeepRecentTokens: 1}, Usage{1000, 1000}, false},
		{"no usage yet", CompactionConfig{ContextWindow: 100, ReserveTokens: 10, KeepRecentTokens: 1}, Usage{}, false},
		{"under line", CompactionConfig{ContextWindow: 100, ReserveTokens: 10, KeepRecentTokens: 1}, Usage{80, 10}, false},
		{"over line", CompactionConfig{ContextWindow: 100, ReserveTokens: 10, KeepRecentTokens: 1}, Usage{80, 11}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			useCompaction(t, c.cfg)
			a := &agent{lastUsage: c.usage}
			if got := a.shouldCompact(); got != c.want {
				t.Errorf("shouldCompact = %v, want %v", got, c.want)
			}
		})
	}
}

func TestRenderConversation(t *testing.T) {
	long := strings.Repeat("o", toolResultExcerpt+10)
	got := renderConversation([]Message{
		userMsg("fix it"),
		{Role: MessageRoleAssistant, Content: []MessagesBlock{
			{Type: MessagesBlockTypeReasoning, Text: "secret thoughts"},
			{Type: MessagesBlockTypeText, Text: "on it"},
			{ID: "c1", Type: MessagesBlockTypeToolUse, Name: "read_file", Input: map[string]interface{}{"path": "a.go"}},
		}},
		toolResultMsg("c1", long),
	})
	for _, want := range []string{
		"user: fix it\n",
		"assistant: on it\n",
		`assistant: [tool_use c1 read_file {"path":"a.go"}]`,
		"user: [tool_result c1] " + strings.Repeat("o", toolResultExcerpt) + "... [10 more bytes]",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "secret thoughts") {
		t.Error("reasoning must not be rendered")
	}
}

func TestSummarize(t *testing.T) {
	llm := &scriptedLLM{responses: []SendMessagesResponse{text("## Goal\nfix it")}}
	a := &agent{name: "main", llmClient: llm}
	span := []Message{userMsg("fix it"), assistantMsg("ok")}

	got, err := a.summarize(context.Background(), span)
	if err != nil || got != "## Goal\nfix it" {
		t.Fatalf("summarize = %q, %v", got, err)
	}
	if len(llm.tools[0]) != 0 {
		t.Error("summarizer must not be offered tools")
	}
	if sys := llm.systems[0].Content.(string); !strings.Contains(sys, "## Critical Context") || !strings.Contains(sys, "### Blocked") {
		t.Errorf("system prompt lacks the template:\n%s", sys)
	}
	input := llm.calls[0][0].Content.(string)
	if !strings.Contains(input, "<conversation>\nuser: fix it\nassistant: ok\n</conversation>") {
		t.Errorf("input lacks transcript:\n%s", input)
	}
	if strings.Contains(input, "previous-summary") {
		t.Error("first summary must not mention a previous one")
	}

	// second time round: previous summary is passed for an incremental update
	llm.responses = append(llm.responses, text("## Goal\nfix it more"))
	a.summary = "## Goal\nfix it"
	if _, err := a.summarize(context.Background(), span); err != nil {
		t.Fatal(err)
	}
	input = llm.calls[1][0].Content.(string)
	if !strings.Contains(input, summarizeUpdateInstruction) || !strings.Contains(input, "<previous-summary>\n## Goal\nfix it\n</previous-summary>") {
		t.Errorf("update input lacks previous summary:\n%s", input)
	}
}

func TestSummarize_Errors(t *testing.T) {
	a := &agent{name: "main", llmClient: &scriptedLLM{}}
	if _, err := a.summarize(context.Background(), []Message{userMsg("x")}); err == nil {
		t.Error("want LLM error to propagate")
	}
	a.llmClient = &scriptedLLM{responses: []SendMessagesResponse{{Content: []MessagesBlock{{Type: MessagesBlockTypeReasoning, Text: "hmm"}}}}}
	if _, err := a.summarize(context.Background(), []Message{userMsg("x")}); err == nil || !strings.Contains(err.Error(), "no text") {
		t.Errorf("err = %v, want no-text error", err)
	}
}

func TestCollectFiles_AccumulateAcrossCompactions(t *testing.T) {
	a := NewAgent(nil, nil).(*agent)
	a.collectFiles([]Message{
		toolUseMsg("1", "read_file", "a.go"),
		toolUseMsg("2", "edit_file", "b.go"),
		toolUseMsg("3", "run_bash", "ignored.go"),
		{Role: MessageRoleAssistant, Content: []MessagesBlock{{Type: MessagesBlockTypeToolUse, Name: "read_file", Input: map[string]interface{}{}}}}, // no path
	})
	a.collectFiles([]Message{
		toolUseMsg("4", "write_file", "c.go"),
		toolUseMsg("5", "read_file", "b.go"),
		toolUseMsg("6", "read_file", "a.go"), // duplicate
	})

	want := "\n<read-files>\na.go\nb.go\n</read-files>\n<modified-files>\nb.go\nc.go\n</modified-files>"
	if got := a.renderFiles(); got != want {
		t.Errorf("renderFiles() =\n%q\nwant\n%q", got, want)
	}
	if got := (&agent{readFiles: map[string]bool{}, modifiedFiles: map[string]bool{}}).renderFiles(); got != "" {
		t.Errorf("empty lists should render nothing, got %q", got)
	}
}

func TestCompact_EndToEnd(t *testing.T) {
	// 4 tool rounds of 100-token results; window line at 500, keep the newest ~250
	useCompaction(t, CompactionConfig{ContextWindow: 600, ReserveTokens: 100, KeepRecentTokens: 250})
	out := captureHookOut(t)
	autoAllow(t)
	big := strings.Repeat("x", 400)
	call := func(id string) SendMessagesResponse {
		return SendMessagesResponse{Content: []MessagesBlock{
			{ID: id, Type: MessagesBlockTypeToolUse, Name: "run_bash", Input: map[string]interface{}{"command": "echo " + big}},
		}}
	}
	llm := &scriptedLLM{responses: []SendMessagesResponse{
		withUsage(call("c1"), 100, 10),
		withUsage(call("c2"), 200, 10),
		withUsage(call("c3"), 300, 10),
		withUsage(call("c4"), 495, 10), // 505 > 500 → compaction before the next call
		text("## Goal\nsummary"),       // summarizer
		text("done"),
	}}

	h := new(Hooks).OnCompact(CompactLogHook())
	if err := NewAgent(llm, h).RunLoop(context.Background(), []Message{userMsg("start")}); err != nil {
		t.Fatal(err)
	}
	if len(llm.calls) != 6 {
		t.Fatalf("calls = %d, want 6", len(llm.calls))
	}

	// the summarizer saw the older part of the transcript
	summarizerInput := llm.calls[4][0].Content.(string)
	if !strings.Contains(summarizerInput, "user: start") || !strings.Contains(summarizerInput, "[tool_use c1 run_bash") {
		t.Errorf("summarizer input lacks the compacted span:\n%s", summarizerInput)
	}

	// the last agent call opens with the summary and keeps the recent rounds intact
	final := llm.calls[5]
	first, _ := final[0].Content.(string)
	if final[0].Role != MessageRoleUser || !strings.HasPrefix(first, summaryOpenTag+"## Goal\nsummary") {
		t.Errorf("messages[0] = %+v, want the summary", final[0])
	}
	if strings.Count(renderConversation(final), "user: start") != 0 {
		t.Error("original prompt should have been summarized away")
	}
	if !isToolResultMessage(final[len(final)-1]) || len(final) < 3 {
		t.Errorf("recent rounds not kept: %+v", final)
	}
	if final[1].Role != MessageRoleAssistant {
		t.Errorf("first kept message must be the assistant tool_use, got %+v", final[1])
	}
	if !strings.Contains(out.String(), "[HOOK] Compact: main summarized 5 messages") || !strings.Contains(out.String(), "kept 4") {
		t.Errorf("no progress line: %q", out.String())
	}
}

func TestHook_Compact_RewriteAndCancel(t *testing.T) {
	useCompaction(t, CompactionConfig{ContextWindow: 100, ReserveTokens: 10, KeepRecentTokens: 1})
	base := []Message{userMsg("start"), assistantMsg("mid"), assistantMsg("tail")}

	t.Run("rewrite", func(t *testing.T) {
		var gotCompacted, gotKept int
		h := new(Hooks).OnCompact(func(_ context.Context, compacted, kept []Message, summary string) (string, error) {
			gotCompacted, gotKept = len(compacted), len(kept)
			return summary + " [checked]", nil
		})
		a := NewAgent(&scriptedLLM{responses: []SendMessagesResponse{text("S")}}, h).(*agent)
		a.messages = append([]Message(nil), base...)

		a.compact(context.Background())
		if a.summary != "S [checked]" || gotCompacted != 2 || gotKept != 1 {
			t.Errorf("summary=%q compacted=%d kept=%d", a.summary, gotCompacted, gotKept)
		}
	})

	t.Run("cancel", func(t *testing.T) {
		h := new(Hooks).OnCompact(func(context.Context, []Message, []Message, string) (string, error) {
			return "", errors.New("summary lost the goal")
		})
		a := NewAgent(&scriptedLLM{responses: []SendMessagesResponse{text("S")}}, h).(*agent)
		a.messages = append([]Message(nil), base...)

		a.compact(context.Background())
		if a.summary != "" || len(a.messages) != 3 || a.messages[0].Content != "start" {
			t.Errorf("cancelled compaction must leave messages untouched: %+v", a.messages)
		}
	})
}

func TestCompact_SecondRoundReplacesSummary(t *testing.T) {
	useCompaction(t, CompactionConfig{ContextWindow: 100, ReserveTokens: 10, KeepRecentTokens: 1})
	a := NewAgent(&scriptedLLM{responses: []SendMessagesResponse{text("S1"), text("S2")}}, nil).(*agent)
	a.messages = []Message{userMsg("start"), toolUseMsg("1", "read_file", "a.go"), toolResultMsg("1", "aaa"), assistantMsg("mid"), assistantMsg("tail")}

	a.compact(context.Background())
	if a.summary != "S1\n<read-files>\na.go\n</read-files>" {
		t.Fatalf("summary = %q", a.summary)
	}
	a.messages = append(a.messages, toolUseMsg("2", "edit_file", "a.go"), toolResultMsg("2", "ok"), assistantMsg("end"))

	a.compact(context.Background())
	if !strings.HasPrefix(a.summary, "S2\n<read-files>\na.go\n</read-files>\n<modified-files>\na.go\n</modified-files>") {
		t.Errorf("summary = %q", a.summary)
	}
	if n := strings.Count(renderConversation(a.messages), summaryOpenTag); n != 1 {
		t.Errorf("%d summaries in messages, want exactly 1: %+v", n, a.messages)
	}
	if got, _ := a.messages[0].Content.(string); !strings.HasPrefix(got, summaryOpenTag+"S2") {
		t.Errorf("messages[0] = %q, want the new summary", got)
	}
}

func TestCompact_NothingToCut(t *testing.T) {
	useCompaction(t, CompactionConfig{ContextWindow: 100, ReserveTokens: 10, KeepRecentTokens: 100_000})
	llm := &scriptedLLM{}
	a := NewAgent(llm, nil).(*agent)
	a.messages = []Message{userMsg("start"), assistantMsg("tail")}

	a.compact(context.Background())
	if len(llm.calls) != 0 || a.summary != "" || len(a.messages) != 2 {
		t.Errorf("compact should be a no-op when everything is recent: calls=%d summary=%q msgs=%d", len(llm.calls), a.summary, len(a.messages))
	}
}

func TestCompact_SummarizeFailureKeepsMessages(t *testing.T) {
	useCompaction(t, CompactionConfig{ContextWindow: 100, ReserveTokens: 10, KeepRecentTokens: 1})
	a := NewAgent(failingLLM{}, nil).(*agent)
	before := []Message{userMsg("start"), assistantMsg("mid"), assistantMsg("tail")}
	a.messages = append([]Message(nil), before...)

	a.compact(context.Background())
	if a.summary != "" || len(a.messages) != len(before) || a.messages[0].Content != "start" {
		t.Errorf("messages changed after failed summarize: %+v", a.messages)
	}
}

func TestRunLoop_ResetsCompactionState(t *testing.T) {
	llm := &scriptedLLM{responses: []SendMessagesResponse{withUsage(text("a"), 5, 5), text("b")}}
	a := NewAgent(llm, nil).(*agent)
	a.summary, a.readFiles["x"] = "old", true
	if err := a.RunLoop(context.Background(), userHi); err != nil {
		t.Fatal(err)
	}
	if a.lastUsage != (Usage{}) || a.summary != "" || len(a.readFiles) != 0 {
		t.Errorf("state not reset: usage=%+v summary=%q read=%v", a.lastUsage, a.summary, a.readFiles)
	}
}
