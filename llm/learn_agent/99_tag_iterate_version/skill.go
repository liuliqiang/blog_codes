package agentloop

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/liuliqiang/log4go"
)

// skillsDir is scanned for <name>/SKILL.md when an agent is built. It is
// relative to the working directory, like the tools' default paths.
var skillsDir = "skills"

// Skill is one SKILL.md: its frontmatter name and description go into the
// system prompt catalog, the full content is returned by load_skill.
type Skill struct {
	Name        string
	Description string
	Content     string
}

// SkillLoader holds the skills found under one directory, keyed by name.
type SkillLoader struct {
	skills map[string]Skill
	names  []string // sorted, for stable catalog output
}

// NewSkillLoader scans dir/*/SKILL.md once. A missing dir simply yields no
// skills; unreadable or escaped (symlinked outside dir) manifests are logged
// and skipped.
func NewSkillLoader(dir string) *SkillLoader {
	ctx := context.Background()
	l := &SkillLoader{skills: map[string]Skill{}}

	manifests, err := filepath.Glob(filepath.Join(dir, "*", "SKILL.md"))
	if err != nil {
		log4go.DefaultLogger().Error(ctx, "glob skills failed: %v, dir: %s", err, dir)
		return l
	}
	root, err := filepath.EvalSymlinks(dir)
	if err != nil {
		if !os.IsNotExist(err) {
			log4go.DefaultLogger().Error(ctx, "resolve skills dir failed: %v, dir: %s", err, dir)
		}
		return l
	}

	for _, manifest := range manifests {
		resolved, err := filepath.EvalSymlinks(manifest)
		if err != nil {
			log4go.DefaultLogger().Error(ctx, "resolve skill manifest failed: %v, path: %s", err, manifest)
			continue
		}
		if rel, err := filepath.Rel(root, resolved); err != nil || strings.HasPrefix(rel, "..") {
			log4go.DefaultLogger().Error(ctx, "skill manifest escapes skills dir, skipped: %s -> %s", manifest, resolved)
			continue
		}
		content, err := os.ReadFile(manifest)
		if err != nil {
			log4go.DefaultLogger().Error(ctx, "read skill manifest failed: %v, path: %s", err, manifest)
			continue
		}

		meta, body := parseFrontmatter(string(content))
		name := strings.TrimSpace(meta["name"])
		if name == "" {
			name = filepath.Base(filepath.Dir(manifest))
		}
		description := strings.TrimSpace(meta["description"])
		if description == "" {
			description, _, _ = strings.Cut(body, "\n")
			description = strings.TrimLeft(description, "# ")
		}
		description = strings.Join(strings.Fields(description), " ")

		l.skills[name] = Skill{Name: name, Description: description, Content: string(content)}
	}

	for name := range l.skills {
		l.names = append(l.names, name)
	}
	sort.Strings(l.names)
	return l
}

// parseFrontmatter splits a leading "---" block of "key: value" lines from
// the rest of the document. Without such a block the whole text is the body.
func parseFrontmatter(text string) (map[string]string, string) {
	meta := map[string]string{}
	lines := strings.Split(text, "\n")
	if len(lines) == 0 || strings.TrimRight(lines[0], "\r") != "---" {
		return meta, text
	}
	for i := 1; i < len(lines); i++ {
		line := strings.TrimRight(lines[i], "\r")
		if line == "---" {
			return meta, strings.TrimSpace(strings.Join(lines[i+1:], "\n"))
		}
		if key, value, ok := strings.Cut(line, ":"); ok {
			meta[strings.TrimSpace(key)] = strings.Trim(strings.TrimSpace(value), `"'`)
		}
	}
	// no closing marker: not frontmatter after all
	return map[string]string{}, text
}

// Catalog renders the "- name: description" list that goes into the system
// prompt.
func (l *SkillLoader) Catalog() string {
	if len(l.names) == 0 {
		return "(no skills found)"
	}
	var lines []string
	for _, name := range l.names {
		lines = append(lines, fmt.Sprintf("- %s: %s", name, l.skills[name].Description))
	}
	return strings.Join(lines, "\n")
}

// Load returns the full SKILL.md for name.
func (l *SkillLoader) Load(name string) (string, error) {
	skill, ok := l.skills[name]
	if !ok {
		available := "none"
		if len(l.names) > 0 {
			available = strings.Join(l.names, ", ")
		}
		return "", fmt.Errorf("unknown skill %q, available: %s", name, available)
	}
	return skill.Content, nil
}

// withSkillCatalog appends the skill catalog and the instruction to use
// load_skill to a base system prompt.
func withSkillCatalog(base, catalog string) string {
	return base + "\n\nSkills available:\n" + catalog + "\n\nUse load_skill to read the full instructions when a skill applies."
}

// runLoadSkill is the handler of the load_skill tool.
func (a *agent) runLoadSkill(ctx context.Context, input map[string]interface{}) (string, error) {
	name, err := stringArg(input, "name")
	if err != nil {
		return "", err
	}
	log4go.DefaultLogger().Info(ctx, "[%s] loading skill %q", agentNameFrom(ctx), name)
	content, err := a.skills.Load(name)
	if err != nil {
		log4go.DefaultLogger().Error(ctx, "load skill failed: %v", err)
		return "", err
	}
	return content, nil
}
