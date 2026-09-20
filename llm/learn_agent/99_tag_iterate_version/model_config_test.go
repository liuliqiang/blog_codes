package agentloop

import (
	"context"
	"reflect"
	"testing"
)

// useModels sets the per-purpose model config for the test and restores it afterwards.
func useModels(t *testing.T, compaction, memory, subagent Model) {
	t.Helper()
	origC, origM, origS := Compaction.Model, Memory.Model, Subagent.Model
	Compaction.Model, Memory.Model, Subagent.Model = compaction, memory, subagent
	t.Cleanup(func() { Compaction.Model, Memory.Model, Subagent.Model = origC, origM, origS })
}

func TestModel_OrModel(t *testing.T) {
	if got := Model("").orModel("fallback"); got != "fallback" {
		t.Errorf("empty = %q", got)
	}
	if got := Model("set").orModel("fallback"); got != "set" {
		t.Errorf("set = %q", got)
	}
}

func TestSubagent_ModelInheritsOrOverrides(t *testing.T) {
	useModels(t, "", "", "")
	parent := NewAgent(&scriptedLLM{}, nil).(*agent)
	if parent.model != "fake" {
		t.Fatalf("main model = %q, want the client's", parent.model)
	}
	if sub := parent.newSubagent(); sub.model != "fake" {
		t.Errorf("unset config: subagent model = %q, want parent's", sub.model)
	}

	Subagent.Model = "small"
	if sub := parent.newSubagent(); sub.model != "small" {
		t.Errorf("configured: subagent model = %q, want small", sub.model)
	}

	// end to end: the subagent's calls go to its model, the parent's to its own
	llm := &scriptedLLM{responses: []SendMessagesResponse{taskCall("t1", "x"), text("sub done"), text("done")}}
	if err := NewAgent(llm, nil).RunLoop(context.Background(), userHi); err != nil {
		t.Fatal(err)
	}
	if want := []Model{"fake", "small", "fake"}; !reflect.DeepEqual(llm.models, want) {
		t.Errorf("models = %v, want %v", llm.models, want)
	}
}

func TestCompaction_ModelConfig(t *testing.T) {
	useCompaction(t, CompactionConfig{ContextWindow: 100, ReserveTokens: 10, KeepRecentTokens: 1})
	msgs := []Message{userMsg("start"), assistantMsg("mid"), assistantMsg("tail")}

	for _, c := range []struct{ configured, want Model }{{"", "fake"}, {"summarizer", "summarizer"}} {
		Compaction.Model = c.configured
		llm := &scriptedLLM{responses: []SendMessagesResponse{text("S")}}
		a := NewAgent(llm, nil).(*agent)
		a.messages = append([]Message(nil), msgs...)
		a.compact(context.Background())
		if len(llm.models) != 1 || llm.models[0] != c.want {
			t.Errorf("Compaction.Model=%q: summarizer used %v, want %q", c.configured, llm.models, c.want)
		}
	}
}

func TestMemory_ModelConfig(t *testing.T) {
	dir, _ := useMemoryDir(t)
	if err := NewMemoryStore(dir).Write(tabsRecord); err != nil {
		t.Fatal(err)
	}

	for _, c := range []struct{ configured, want Model }{{"", "fake"}, {"tiny", "tiny"}} {
		useModels(t, "", c.configured, "")
		llm := &scriptedLLM{responses: []SendMessagesResponse{
			text("[0]"),  // recall selection
			text("tabs"), // answer
			text("[]"),   // extraction
		}}
		h := new(Hooks).OnStop(MemoryHook(llm))
		if err := NewAgent(llm, h).RunLoop(context.Background(), []Message{userMsg("what indentation do I prefer?")}); err != nil {
			t.Fatal(err)
		}
		if want := []Model{c.want, "fake", c.want}; !reflect.DeepEqual(llm.models, want) {
			t.Errorf("Memory.Model=%q: models = %v, want %v", c.configured, llm.models, want)
		}
	}
}
