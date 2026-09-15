package agentloop

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const reviewSkill = `---
name: code-review
description: "Review code for bugs and style."
---

# Code Review

Check every changed line.`

// writeSkill creates dir/<folder>/SKILL.md with content.
func writeSkill(t *testing.T, dir, folder, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, folder), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, folder, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestParseFrontmatter(t *testing.T) {
	cases := []struct {
		name string
		text string
		meta map[string]string
		body string
	}{
		{"with frontmatter", reviewSkill, map[string]string{"name": "code-review", "description": "Review code for bugs and style."}, "# Code Review\n\nCheck every changed line."},
		{"no frontmatter", "# Plain\n\nbody", map[string]string{}, "# Plain\n\nbody"},
		{"unclosed frontmatter", "---\nname: x\nbody", map[string]string{}, "---\nname: x\nbody"},
		{"crlf and colon in value", "---\r\nname: a\r\ndescription: use when: needed\r\n---\r\nbody\r\n", map[string]string{"name": "a", "description": "use when: needed"}, "body"},
		{"empty", "", map[string]string{}, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			meta, body := parseFrontmatter(c.text)
			if !reflect.DeepEqual(meta, c.meta) || body != c.body {
				t.Errorf("got %v %q, want %v %q", meta, body, c.meta, c.body)
			}
		})
	}
}

func TestNewSkillLoader(t *testing.T) {
	dir := t.TempDir()
	writeSkill(t, dir, "code-review", reviewSkill)
	// no frontmatter: name from folder, description from first heading
	writeSkill(t, dir, "pdf", "#  Work with   PDF files\n\ndetails")
	// not a SKILL.md, must be ignored
	if err := os.WriteFile(filepath.Join(dir, "code-review", "notes.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	l := NewSkillLoader(dir)

	want := "- code-review: Review code for bugs and style.\n- pdf: Work with PDF files"
	if got := l.Catalog(); got != want {
		t.Errorf("Catalog() =\n%s\nwant\n%s", got, want)
	}
	if content, err := l.Load("code-review"); err != nil || content != reviewSkill {
		t.Errorf("Load(code-review) = %q, %v", content, err)
	}
	_, err := l.Load("nope")
	if err == nil || !strings.Contains(err.Error(), `unknown skill "nope"`) || !strings.Contains(err.Error(), "code-review, pdf") {
		t.Errorf("Load(nope) err = %v, want unknown skill listing the available ones", err)
	}
}

func TestNewSkillLoader_MissingDir(t *testing.T) {
	l := NewSkillLoader(filepath.Join(t.TempDir(), "absent"))
	if got := l.Catalog(); got != "(no skills found)" {
		t.Errorf("Catalog() = %q", got)
	}
	if _, err := l.Load("x"); err == nil || !strings.Contains(err.Error(), "available: none") {
		t.Errorf("err = %v", err)
	}
}

func TestNewSkillLoader_SkipsSymlinkOutsideDir(t *testing.T) {
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "SKILL.md"), []byte("---\nname: leaked\n---\nsecret"), 0o644); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	writeSkill(t, dir, "ok", "---\nname: ok\ndescription: fine\n---\nbody")
	if err := os.Symlink(outside, filepath.Join(dir, "leak")); err != nil {
		t.Skip("symlinks not supported:", err)
	}

	l := NewSkillLoader(dir)
	if _, err := l.Load("leaked"); err == nil {
		t.Error("skill reached through a symlink outside the skills dir must be skipped")
	}
	if _, err := l.Load("ok"); err != nil {
		t.Errorf("regular skill missing: %v", err)
	}
}

// useSkillsDir points the agent at a temp skills dir for the test.
func useSkillsDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	orig := skillsDir
	skillsDir = dir
	t.Cleanup(func() { skillsDir = orig })
	return dir
}

func TestAgent_SystemPromptCarriesCatalog(t *testing.T) {
	dir := useSkillsDir(t)
	writeSkill(t, dir, "code-review", reviewSkill)

	a := NewAgent(nil, nil).(*agent)
	for _, want := range []string{
		defaultSystemPrompt,
		"Skills available:\n- code-review: Review code for bugs and style.",
		"Use load_skill",
	} {
		if !strings.Contains(a.systemPrompt, want) {
			t.Errorf("system prompt missing %q:\n%s", want, a.systemPrompt)
		}
	}

	// the subagent gets the same catalog on top of its own prompt
	sub := a.newSubagent()
	if sub.skills != a.skills || !strings.Contains(sub.systemPrompt, "- code-review:") {
		t.Errorf("subagent prompt lacks catalog:\n%s", sub.systemPrompt)
	}
	if _, ok := sub.toolIndex["load_skill"]; !ok {
		t.Error("subagent should be able to load skills")
	}
}

func TestLoadSkillTool(t *testing.T) {
	dir := useSkillsDir(t)
	writeSkill(t, dir, "code-review", reviewSkill)
	a := NewAgent(nil, nil).(*agent)

	tool, ok := a.toolIndex["load_skill"]
	if !ok {
		t.Fatalf("load_skill not registered, have %v", toolNames(a.tools))
	}
	if !reflect.DeepEqual(tool.InputSchema["required"], []string{"name"}) {
		t.Errorf("required = %v", tool.InputSchema["required"])
	}
	if !isAllowListed(MessagesBlock{Name: "load_skill"}) {
		t.Error("load_skill is read-only and should not prompt the user")
	}

	ctx := context.Background()
	if got, err := tool.Handler(ctx, map[string]interface{}{"name": "code-review"}); err != nil || got != reviewSkill {
		t.Errorf("handler = %q, %v", got, err)
	}
	if _, err := tool.Handler(ctx, map[string]interface{}{"name": "nope"}); err == nil {
		t.Error("want error for unknown skill")
	}
	if _, err := tool.Handler(ctx, map[string]interface{}{}); err == nil {
		t.Error("want error for missing name")
	}
}

func TestLoadSkill_EndToEnd(t *testing.T) {
	dir := useSkillsDir(t)
	writeSkill(t, dir, "code-review", reviewSkill)
	llm := &scriptedLLM{responses: []SendMessagesResponse{
		{Content: []MessagesBlock{{ID: "s1", Type: MessagesBlockTypeToolUse, Name: "load_skill", Input: map[string]interface{}{"name": "code-review"}}}},
		text("reviewed"),
	}}

	if err := NewAgent(llm, nil).RunLoop(context.Background(), userHi); err != nil {
		t.Fatal(err)
	}
	if res := lastToolResult(t, llm.calls[1]); res.ToolUseID != "s1" || res.Content != reviewSkill {
		t.Errorf("tool result = %+v", res)
	}
}
