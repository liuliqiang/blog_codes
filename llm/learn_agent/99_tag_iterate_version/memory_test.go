package agentloop

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// useMemoryDir points the store at a temp dir and captures memory output.
func useMemoryDir(t *testing.T) (string, *bytes.Buffer) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), ".memory")
	origDir, origOut := memoryDir, memoryOut
	out := &bytes.Buffer{}
	memoryDir, memoryOut = dir, out
	t.Cleanup(func() { memoryDir, memoryOut = origDir, origOut })
	return dir, out
}

var tabsRecord = MemoryRecord{Name: "user-preference-tabs", Type: "user", Description: "User prefers tabs for indentation", Body: "User prefers using tabs, not spaces, for indentation."}

func TestMemorySlug(t *testing.T) {
	for in, want := range map[string]string{
		"User Preference: Tabs!": "user-preference-tabs",
		"  ---  ":                "memory",
		"缩进 偏好":                  "缩进-偏好",
		"a_b":                    "a-b",
	} {
		if got := memorySlug(in); got != want {
			t.Errorf("memorySlug(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestMemoryStore_WriteListReadIndex(t *testing.T) {
	dir, _ := useMemoryDir(t)
	store := NewMemoryStore(dir)

	if got := store.List(); len(got) != 0 || store.Index() != "" {
		t.Fatalf("fresh store: list=%v index=%q", got, store.Index())
	}
	if err := store.Write(tabsRecord); err != nil {
		t.Fatal(err)
	}
	if err := store.Write(MemoryRecord{Name: "Issue tracker", Type: "reference", Description: "Bugs live in Linear INGEST", Body: "See Linear project INGEST."}); err != nil {
		t.Fatal(err)
	}

	content, err := store.Read("user-preference-tabs.md")
	if err != nil {
		t.Fatal(err)
	}
	want := "---\nname: user-preference-tabs\ndescription: User prefers tabs for indentation\ntype: user\n---\n\nUser prefers using tabs, not spaces, for indentation.\n"
	if content != want {
		t.Errorf("document =\n%q\nwant\n%q", content, want)
	}

	records := store.List()
	if len(records) != 2 || records[0].Name != "Issue tracker" || !reflect.DeepEqual(records[1], tabsRecord) {
		t.Errorf("List() = %+v", records)
	}
	wantIndex := "- [Issue tracker](issue-tracker.md) - Bugs live in Linear INGEST\n- [user-preference-tabs](user-preference-tabs.md) - User prefers tabs for indentation"
	if got := store.Index(); got != wantIndex {
		t.Errorf("Index() =\n%s\nwant\n%s", got, wantIndex)
	}

	// index is never a record; path escapes are rejected; validation runs on write
	for _, name := range []string{memoryIndexFile, "../x.md", "sub/x.md", ""} {
		if _, err := store.Read(name); err == nil {
			t.Errorf("Read(%q) should fail", name)
		}
	}
	for _, bad := range []MemoryRecord{
		{Name: "", Type: "user", Description: "d", Body: "b"},
		{Name: "n", Type: "weird", Description: "d", Body: "b"},
		{Name: "n", Type: "user", Description: "", Body: "b"},
	} {
		if err := store.Write(bad); err == nil {
			t.Errorf("Write(%+v) should fail", bad)
		}
	}
}

func TestMemoryStore_ListFallbacks(t *testing.T) {
	dir, _ := useMemoryDir(t)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "plain-note.md"), []byte("first line is the description\nmore"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := NewMemoryStore(dir).List()
	want := []MemoryRecord{{Name: "plain-note", Type: "project", Description: "first line is the description", Body: "first line is the description\nmore"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("List() = %+v, want %+v", got, want)
	}
}

func TestMemoryStore_Replace(t *testing.T) {
	dir, _ := useMemoryDir(t)
	store := NewMemoryStore(dir)
	for _, r := range []MemoryRecord{tabsRecord, {Name: "old", Type: "project", Description: "stale", Body: "gone"}} {
		if err := store.Write(r); err != nil {
			t.Fatal(err)
		}
	}
	merged := MemoryRecord{Name: "merged", Type: "user", Description: "merged record", Body: "tabs + more"}
	if err := store.Replace([]MemoryRecord{merged}); err != nil {
		t.Fatal(err)
	}
	if got := store.List(); len(got) != 1 || got[0].Name != "merged" {
		t.Errorf("after Replace: %+v", got)
	}
	if !strings.Contains(store.Index(), "merged.md") || strings.Contains(store.Index(), "old.md") {
		t.Errorf("index not rebuilt: %q", store.Index())
	}
}

func TestMemoryStore_ReplaceRestoresOnFailure(t *testing.T) {
	dir, _ := useMemoryDir(t)
	store := NewMemoryStore(dir)
	if err := store.Write(tabsRecord); err != nil {
		t.Fatal(err)
	}
	// a record whose file name collides with a directory can't be written
	if err := os.MkdirAll(filepath.Join(dir, "blocked.md"), 0o755); err != nil {
		t.Fatal(err)
	}

	err := store.Replace([]MemoryRecord{
		{Name: "first", Type: "user", Description: "ok", Body: "ok"},
		{Name: "blocked", Type: "user", Description: "x", Body: "x"},
	})
	if err == nil {
		t.Fatal("want Replace to fail")
	}
	got := store.List()
	if len(got) != 1 || !reflect.DeepEqual(got[0], tabsRecord) {
		t.Errorf("store not restored: %+v", got)
	}
	if !strings.Contains(store.Index(), "user-preference-tabs.md") || strings.Contains(store.Index(), "first.md") {
		t.Errorf("index not restored: %q", store.Index())
	}
}

func TestShouldStoreMemory(t *testing.T) {
	existing := []MemoryRecord{tabsRecord}
	ok := MemoryRecord{Name: "no-db-mocks", Type: "feedback", Scope: "persistent", Description: "Do not mock the database", Body: "Tests hit a real database."}
	cases := []struct {
		name string
		rec  MemoryRecord
		want bool
	}{
		{"persistent new record", ok, true},
		{"current_task scope", with(ok, func(r *MemoryRecord) { r.Scope = "current_task" }), false},
		{"missing scope", with(ok, func(r *MemoryRecord) { r.Scope = "" }), false},
		{"bad type", with(ok, func(r *MemoryRecord) { r.Type = "note" }), false},
		{"empty body", with(ok, func(r *MemoryRecord) { r.Body = " " }), false},
		{"temporary marker en", with(ok, func(r *MemoryRecord) { r.Body = "Do not create files in this session." }), false},
		{"temporary marker zh", with(ok, func(r *MemoryRecord) { r.Description = "本次任务不要新建文件" }), false},
		{"duplicate slug", with(ok, func(r *MemoryRecord) { r.Name = "User Preference Tabs" }), false},
		{"duplicate description", with(ok, func(r *MemoryRecord) { r.Description = "  user PREFERS tabs for indentation " }), false},
		{"duplicate body", with(ok, func(r *MemoryRecord) { r.Body = tabsRecord.Body }), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := shouldStoreMemory(c.rec, existing); got != c.want {
				t.Errorf("shouldStoreMemory = %v, want %v", got, c.want)
			}
		})
	}
}

func with(r MemoryRecord, f func(*MemoryRecord)) MemoryRecord { f(&r); return r }

func TestUnmarshalJSONArray(t *testing.T) {
	cases := map[string]string{
		`[0, 2]`:                           `[0,2]`,
		"Sure: [1] and more":               `[1]`,
		"broken [1, then real [2,3]":       `[2,3]`,
		"```json\n[{\"name\":\"x\"}]\n```": `[{"name":"x"}]`,
		"indices [a] then [1]":             `[1]`,
	}
	for in, want := range cases {
		var v interface{}
		if err := unmarshalJSONArray(in, &v); err != nil {
			t.Errorf("unmarshalJSONArray(%q): %v", in, err)
			continue
		}
		got, _ := json.Marshal(v)
		if string(got) != want {
			t.Errorf("unmarshalJSONArray(%q) = %s, want %s", in, got, want)
		}
	}
	for _, in := range []string{"nothing here", "[unterminated", ""} {
		var v []int
		if err := unmarshalJSONArray(in, &v); err == nil {
			t.Errorf("unmarshalJSONArray(%q) should fail", in)
		}
	}
	var ints []int
	if err := unmarshalJSONArray(`[{"a":1}]`, &ints); err == nil {
		t.Error("type mismatch should fail")
	}
}

func TestRecentUserText(t *testing.T) {
	msgs := []Message{
		userMsg("first"), assistantMsg("a1"),
		userMsg("second"), assistantMsg("a2"),
		toolResultMsg("t", "ignored tool output"),
		{Role: MessageRoleUser, Content: []MessagesBlock{{Type: MessagesBlockTypeText, Text: "third"}}},
		userMsg("fourth"),
	}
	if got := recentUserText(msgs, 3); got != "second\nthird\nfourth" {
		t.Errorf("recentUserText = %q", got)
	}
	if got := recentUserText(nil, 3); got != "" {
		t.Errorf("empty = %q", got)
	}
}

func TestKeywordMemorySelection(t *testing.T) {
	records := []MemoryRecord{
		{Name: "tabs", Description: "User prefers tabs for indentation"},
		{Name: "linear", Description: "Bugs tracked in Linear"},
		{Name: "indent-width", Description: "indentation width is 4 and tabs render as 4"},
	}
	got := keywordMemorySelection(records, "what indentation and tabs do I use?", 5)
	if !reflect.DeepEqual(got, []string{"indent-width.md", "tabs.md"}) {
		t.Errorf("selection = %v", got)
	}
	if got := keywordMemorySelection(records, "zzz", 5); len(got) != 0 {
		t.Errorf("no match = %v", got)
	}
	if got := keywordMemorySelection(records, "tabs indentation", 1); len(got) != 1 {
		t.Errorf("max not applied: %v", got)
	}
}

func TestSelectMemories_ModelThenFallback(t *testing.T) {
	records := []MemoryRecord{
		{Name: "tabs", Description: "User prefers tabs for indentation"},
		{Name: "linear", Description: "Bugs tracked in Linear"},
	}
	ctx := context.Background()

	llm := &scriptedLLM{responses: []SendMessagesResponse{text("[1, 1, 7, -1, 0]")}}
	got := selectMemories(ctx, llm, "fake", records, "where are bugs?")
	if !reflect.DeepEqual(got, []string{"linear.md", "tabs.md"}) {
		t.Errorf("model selection = %v", got)
	}
	if len(llm.tools[0]) != 0 {
		t.Error("selection call must not offer tools")
	}
	if prompt := llm.calls[0][0].Content.(string); !strings.Contains(prompt, "0: tabs - User prefers tabs") || !strings.Contains(prompt, "Current request:\nwhere are bugs?") {
		t.Errorf("prompt = %q", prompt)
	}

	// no usable array → keyword fallback
	got = selectMemories(ctx, &scriptedLLM{responses: []SendMessagesResponse{text("I think the tabs one")}}, "fake", records, "indentation tabs")
	if !reflect.DeepEqual(got, []string{"tabs.md"}) {
		t.Errorf("fallback after bad output = %v", got)
	}
	// LLM error → keyword fallback
	got = selectMemories(ctx, failingLLM{}, "fake", records, "linear bugs")
	if !reflect.DeepEqual(got, []string{"linear.md"}) {
		t.Errorf("fallback after error = %v", got)
	}
}

func TestRecallMemories(t *testing.T) {
	dir, _ := useMemoryDir(t)
	store := NewMemoryStore(dir)
	ctx := context.Background()

	if got := recallMemories(ctx, &scriptedLLM{}, "fake", store, userHi); got != "" {
		t.Errorf("empty store should recall nothing without an LLM call, got %q", got)
	}
	if err := store.Write(tabsRecord); err != nil {
		t.Fatal(err)
	}
	if got := recallMemories(ctx, &scriptedLLM{}, "fake", store, nil); got != "" {
		t.Errorf("no user text should recall nothing, got %q", got)
	}

	got := recallMemories(ctx, &scriptedLLM{responses: []SendMessagesResponse{text("[0]")}}, "fake", store, []Message{userMsg("what indentation do I prefer?")})
	if !strings.HasPrefix(got, `<memory source="user-preference-tabs.md">`) || !strings.Contains(got, "name: user-preference-tabs") || !strings.HasSuffix(got, "</memory>") {
		t.Errorf("recalled = %q", got)
	}
}

func TestWithMemory(t *testing.T) {
	if got := withMemory("base", "", "x"); got != "base" {
		t.Errorf("empty index must leave the prompt alone, got %q", got)
	}
	got := withMemory("base", "- [a](a.md) - A", "")
	if !strings.HasPrefix(got, "base\n\n"+memoryPromptNote) || !strings.Contains(got, "Memory catalog:\n- [a](a.md) - A") || strings.Contains(got, "Relevant memory records") {
		t.Errorf("catalog only = %q", got)
	}
	got = withMemory("base", "- [a](a.md) - A", "<memory>...</memory>")
	if !strings.HasSuffix(got, "Relevant memory records:\n<memory>...</memory>") {
		t.Errorf("with records = %q", got)
	}
}

func TestRunLoop_RecallsMemoryIntoSystemPrompt(t *testing.T) {
	dir, _ := useMemoryDir(t)
	if err := NewMemoryStore(dir).Write(tabsRecord); err != nil {
		t.Fatal(err)
	}
	llm := &scriptedLLM{responses: []SendMessagesResponse{text("[0]"), text("tabs")}}

	if err := NewAgent(llm, nil).RunLoop(context.Background(), []Message{userMsg("what indentation do I prefer?")}); err != nil {
		t.Fatal(err)
	}
	if len(llm.calls) != 2 {
		t.Fatalf("calls = %d, want selection + answer", len(llm.calls))
	}
	sys := llm.systems[1].Content.(string)
	if !strings.HasPrefix(sys, defaultSystemPrompt) || !strings.Contains(sys, "Memory catalog:\n- [user-preference-tabs]") || !strings.Contains(sys, "User prefers using tabs, not spaces") {
		t.Errorf("system prompt lacks memory:\n%s", sys)
	}
	if a := NewAgent(llm, nil).(*agent); a.newSubagent().memory != nil {
		t.Error("subagents must not recall memory")
	}
}

func TestExtractMemories(t *testing.T) {
	dir, out := useMemoryDir(t)
	store := NewMemoryStore(dir)
	ctx := context.Background()
	msgs := []Message{userMsg("I prefer tabs. Remember that. Also don't create files in this session."), assistantMsg("Noted.")}

	llm := &scriptedLLM{responses: []SendMessagesResponse{text(`Here you go:
[
 {"name": "user-preference-tabs", "type": "user", "scope": "persistent", "description": "User prefers tabs for indentation", "body": "Use tabs, not spaces."},
 {"name": "no-files", "type": "feedback", "scope": "current_task", "description": "No new files", "body": "Do not create files in this session."},
 {"name": "bad", "type": "nope", "scope": "persistent", "description": "x", "body": "y"}
]`)}}
	if n := extractMemories(ctx, llm, "fake", store, msgs); n != 1 {
		t.Errorf("stored = %d, want 1", n)
	}
	prompt := llm.calls[0][0].Content.(string)
	if !strings.Contains(prompt, "Existing memory catalog:\n(none)") || !strings.Contains(prompt, "user: I prefer tabs") {
		t.Errorf("prompt = %q", prompt)
	}
	if got := store.List(); len(got) != 1 || got[0].Name != "user-preference-tabs" || got[0].Scope != "" {
		t.Errorf("store = %+v", got)
	}
	if !strings.Contains(out.String(), "[Memory: stored 1 records]") {
		t.Errorf("out = %q", out.String())
	}

	// second run: duplicate is filtered, catalog lists the existing record
	llm = &scriptedLLM{responses: []SendMessagesResponse{text(`[{"name": "tabs again", "type": "user", "scope": "persistent", "description": "User prefers tabs for indentation", "body": "different body"}]`)}}
	if n := extractMemories(ctx, llm, "fake", store, msgs); n != 0 {
		t.Errorf("duplicate stored: %d", n)
	}
	if !strings.Contains(llm.calls[0][0].Content.(string), "- user-preference-tabs: User prefers tabs") {
		t.Error("existing catalog missing from prompt")
	}

	// failures are reported and store nothing
	out.Reset()
	if n := extractMemories(ctx, failingLLM{}, "fake", store, msgs); n != 0 || !strings.Contains(out.String(), "[Memory extraction skipped") {
		t.Errorf("n=%d out=%q", n, out.String())
	}
	if n := extractMemories(ctx, llm, "fake", store, nil); n != 0 {
		t.Errorf("empty dialogue extracted %d", n)
	}
}

func TestConsolidateMemories(t *testing.T) {
	dir, out := useMemoryDir(t)
	store := NewMemoryStore(dir)
	ctx := context.Background()

	if n := consolidateMemories(ctx, &scriptedLLM{}, "fake", store); n != 0 {
		t.Errorf("below threshold consolidated %d", n)
	}
	for i := 0; i < consolidateThreshold; i++ {
		if err := store.Write(MemoryRecord{Name: "rec" + string(rune('a'+i)), Type: "project", Description: "fact " + string(rune('a'+i)), Body: "body " + string(rune('a'+i))}); err != nil {
			t.Fatal(err)
		}
	}

	llm := &scriptedLLM{responses: []SendMessagesResponse{text(`[
 {"name": "merged", "type": "project", "description": "all facts", "body": "a..j"},
 {"name": "keep", "type": "user", "description": "kept", "body": "kept"}
]`)}}
	if n := consolidateMemories(ctx, llm, "fake", store); n != 2 {
		t.Errorf("consolidated = %d, want 2", n)
	}
	if !strings.Contains(llm.calls[0][0].Content.(string), "## reca.md\nname: reca\ntype: project\ndescription: fact a\n\nbody a") {
		t.Error("catalog missing from prompt")
	}
	if got := store.List(); len(got) != 2 || got[0].Name != "keep" || got[1].Name != "merged" {
		t.Errorf("store = %+v", got)
	}
	if !strings.Contains(out.String(), "[Memory: consolidated 10 to 2 records]") {
		t.Errorf("out = %q", out.String())
	}
}

func TestConsolidateMemories_RejectsBadOutput(t *testing.T) {
	dir, out := useMemoryDir(t)
	store := NewMemoryStore(dir)
	ctx := context.Background()
	for i := 0; i < consolidateThreshold; i++ {
		if err := store.Write(MemoryRecord{Name: "rec" + string(rune('a'+i)), Type: "project", Description: "fact", Body: "body " + string(rune('a'+i))}); err != nil {
			t.Fatal(err)
		}
	}
	before := store.List()

	for name, llm := range map[string]LLMClient{
		"llm error":  failingLLM{},
		"no array":   &scriptedLLM{responses: []SendMessagesResponse{text("can't")}},
		"empty":      &scriptedLLM{responses: []SendMessagesResponse{text("[]")}},
		"duplicates": &scriptedLLM{responses: []SendMessagesResponse{text(`[{"name":"X","type":"user","description":"d","body":"b"},{"name":"x","type":"user","description":"d","body":"b"}]`)}},
		"invalid":    &scriptedLLM{responses: []SendMessagesResponse{text(`[{"name":"x","type":"bogus","description":"d","body":"b"}]`)}},
	} {
		t.Run(name, func(t *testing.T) {
			out.Reset()
			if n := consolidateMemories(ctx, llm, "fake", store); n != 0 {
				t.Errorf("consolidated %d", n)
			}
			if !reflect.DeepEqual(store.List(), before) {
				t.Error("store changed")
			}
			if !strings.Contains(out.String(), "[Memory consolidation skipped") {
				t.Errorf("out = %q", out.String())
			}
		})
	}
}

func TestMemoryHook(t *testing.T) {
	dir, _ := useMemoryDir(t)
	llm := &scriptedLLM{responses: []SendMessagesResponse{
		text("Got it, tabs."), // agent answer
		text(`[{"name": "tabs", "type": "user", "scope": "persistent", "description": "tabs", "body": "Use tabs."}]`), // extraction
	}}
	h := new(Hooks).OnStop(MemoryHook(llm))

	if err := NewAgent(llm, h).RunLoop(context.Background(), []Message{userMsg("I prefer tabs, remember that")}); err != nil {
		t.Fatal(err)
	}
	if len(llm.calls) != 2 {
		t.Fatalf("calls = %d, want answer + extraction", len(llm.calls))
	}
	if got := NewMemoryStore(dir).List(); len(got) != 1 || got[0].Name != "tabs" {
		t.Errorf("store = %+v", got)
	}

	// an earlier Stop hook that continues the loop short-circuits extraction
	dir2, _ := useMemoryDir(t)
	continued := false
	llm = &scriptedLLM{responses: []SendMessagesResponse{text("a"), text("b"), text(`[{"name":"x","type":"user","scope":"persistent","description":"d","body":"b"}]`)}}
	h = new(Hooks).OnStop(func(context.Context, []Message) (*Message, error) {
		if continued {
			return nil, nil
		}
		continued = true
		return &Message{Role: MessageRoleUser, Content: "go on"}, nil
	}).OnStop(MemoryHook(llm))
	if err := NewAgent(llm, h).RunLoop(context.Background(), userHi); err != nil {
		t.Fatal(err)
	}
	if len(llm.calls) != 3 {
		t.Errorf("calls = %d, want a, b, extraction", len(llm.calls))
	}
	if got := NewMemoryStore(dir2).List(); len(got) != 1 {
		t.Errorf("extraction after the final stop missing: %+v", got)
	}
}
