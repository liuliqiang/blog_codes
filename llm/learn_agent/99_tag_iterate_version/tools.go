package agentloop

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/liuliqiang/log4go"
)

func (a *agent) generateTools() []Tool {
	return []Tool{
		{
			Name:        "run_bash",
			Handler:     runBash,
			Description: "Run a bash command and return the output.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"command": map[string]interface{}{
						"type": "string",
					},
				},
				"required": []string{"command"},
			},
		},
		{
			Name:        "list_directory",
			Handler:     listDirectory,
			Description: "List the contents of a directory.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type": "string",
					},
				},
				"required": []string{"path"},
			},
		},
		{
			Name:        "read_file",
			Handler:     readFile,
			Description: "Read the contents of a file and return it.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type": "string",
					},
				},
				"required": []string{"path"},
			},
		},
		{
			Name:        "write_file",
			Handler:     writeFile,
			Description: "Write content to a file.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type": "string",
					},
					"content": map[string]interface{}{
						"type": "string",
					},
				},
				"required": []string{"path", "content"},
			},
		},
		{
			Name:        "edit_file",
			Handler:     editFile,
			Description: "Edit a file by replacing old_string with new_string. old_string must appear exactly once.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type": "string",
					},
					"old_string": map[string]interface{}{
						"type": "string",
					},
					"new_string": map[string]interface{}{
						"type": "string",
					},
				},
				"required": []string{"path", "old_string", "new_string"},
			},
		},
		{
			Name:        "grep_file",
			Handler:     grepFile,
			Description: "Search for a regexp pattern in a file, or recursively in a directory (defaults to the working directory). Returns matching lines as path:line: text.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type": "string",
					},
					"pattern": map[string]interface{}{
						"type": "string",
					},
				},
				"required": []string{"pattern"},
			},
		},
		{
			Name:        "find_file",
			Handler:     findFile,
			Description: "Find files matching a pattern.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type": "string",
					},
					"pattern": map[string]interface{}{
						"type": "string",
					},
				},
				"required": []string{"pattern"},
			},
		},
		{
			Name:        "todo_write",
			Handler:     runTodoWrite,
			Description: "Create and manage a task list for the current session. Pass the complete list every time: mark the step you are working on in_progress and mark steps completed as soon as they are done.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"todos": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"content": map[string]interface{}{
									"type": "string",
								},
								"status": map[string]interface{}{
									"type": "string",
									"enum": []string{"pending", "in_progress", "completed"},
								},
							},
						},
					},
				},
			},
		},
		{
			Name:        "task",
			Handler:     a.runTask,
			Description: "Run a subagent with fresh conversation context and return its final text.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"prompt": map[string]interface{}{
						"type": "string",
					},
				},
				"required": []string{"prompt"},
			},
		},
		{
			Name:        "load_skill",
			Handler:     a.runLoadSkill,
			Description: "Load the full SKILL.md content by skill name.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type": "string",
					},
				},
				"required": []string{"name"},
			},
		},
		{
			Name:        "create_task",
			Handler:     a.runCreateTask,
			Description: "Create a task and return its runtime-generated ID.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"subject":     map[string]interface{}{"type": "string"},
					"description": map[string]interface{}{"type": "string"},
				},
				"required": []string{"subject"},
			},
		},
		{
			Name:        "update_task",
			Handler:     a.runUpdateTask,
			Description: "Add dependencies to a pending task using IDs returned by create_task.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"task_id": map[string]interface{}{"type": "string", "pattern": "^task_[0-9a-f]{8}$"},
					"addBlockedBy": map[string]interface{}{
						"type":     "array",
						"items":    map[string]interface{}{"type": "string", "pattern": "^task_[0-9a-f]{8}$"},
						"minItems": 1,
					},
				},
				"required": []string{"task_id", "addBlockedBy"},
			},
		},
		{
			Name:        "list_tasks",
			Handler:     a.runListTasks,
			Description: "List tasks with status, owner, and dependencies.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "get_task",
			Handler:     a.runGetTask,
			Description: "Get the full record of a task by ID.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"task_id": map[string]interface{}{"type": "string"},
				},
				"required": []string{"task_id"},
			},
		},
		{
			Name:        "claim_task",
			Handler:     a.runClaimTask,
			Description: "Claim a pending task whose dependencies are all completed; do this before working on it.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"task_id": map[string]interface{}{"type": "string"},
				},
				"required": []string{"task_id"},
			},
		},
		{
			Name:        "complete_task",
			Handler:     a.runCompleteTask,
			Description: "Complete a task you claimed; reports which tasks became unblocked.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"task_id": map[string]interface{}{"type": "string"},
				},
				"required": []string{"task_id"},
			},
		},
	}
}

// runTools dispatches each tool use to its registered handler and appends the
// results as a user message. Tool failures are returned to the LLM as results
// instead of aborting the loop, so the model can recover on its own.
func (a *agent) runTools(ctx context.Context, toolUses []MessagesBlock) {
	var results []MessagesBlock
	usedTodo := false
	for _, toolUse := range toolUses {
		toolUse, err := a.runPreToolUseHooks(ctx, toolUse)
		if err != nil {
			results = append(results, MessagesBlock{
				Type:      MessagesBlockTypeToolResult,
				ToolUseID: toolUse.ID,
				Content:   "Tool call denied by hook: " + err.Error(),
			})
			continue
		}

		start := time.Now()
		output := a.runTool(ctx, toolUse)
		if toolUse.Name == "todo_write" {
			usedTodo = true
		}
		if rewritten, err := a.runPostToolUseHooks(ctx, toolUse, output); err != nil {
			output = "error: " + err.Error()
		} else {
			output = rewritten
		}
		for _, r := range a.recorders {
			r.OnToolResult(a.turn(), toolUse, output, time.Since(start))
		}
		results = append(results, MessagesBlock{
			Type:      MessagesBlockTypeToolResult,
			ToolUseID: toolUse.ID,
			Content:   output,
		})
	}

	if reminder := a.todoReminder(usedTodo); reminder != "" {
		results = append(results, MessagesBlock{Type: MessagesBlockTypeText, Text: reminder})
	}

	a.messages = append(a.messages, Message{
		Role:    MessageRoleUser,
		Content: results,
	})
}

const (
	todoReminderRounds = 3
	todoReminderText   = "<reminder>Update your todos.</reminder>"
)

// todoReminder tracks how long the todo list has gone untouched and returns
// the reminder to attach to this round's results when it is stale. An empty
// list is never nagged: the system prompt already told the model when to
// plan, so no list means it decided this task doesn't need one.
func (a *agent) todoReminder(usedTodo bool) string {
	if usedTodo {
		a.roundsSinceTodo = 0
		return ""
	}
	a.roundsSinceTodo++
	if len(todoManager.Items()) == 0 || a.roundsSinceTodo < todoReminderRounds {
		return ""
	}
	a.roundsSinceTodo = 0
	return todoReminderText
}

func (a *agent) runTool(ctx context.Context, toolUse MessagesBlock) string {
	tool, ok := a.toolIndex[toolUse.Name]
	if !ok {
		log4go.DefaultLogger().Error(ctx, "unknown tool %q, tool use: %+v", toolUse.Name, toolUse)
		return fmt.Sprintf("error: unknown tool %q", toolUse.Name)
	}
	output, err := tool.Handler(ctx, toolUse.Input)
	if err != nil {
		log4go.DefaultLogger().Error(ctx, "tool %s failed: %v, input: %+v", toolUse.Name, err, toolUse.Input)
		return "error: " + err.Error()
	}
	return output
}

/* vvvvvvvvvvvvvvvvvvvvv tool handlers vvvvvvvvvvvvvvvvvvvvv */

func stringArg(input map[string]interface{}, key string) (string, error) {
	v, ok := input[key]
	if !ok {
		return "", fmt.Errorf("missing required argument %q", key)
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("argument %q must be a string, got %T", key, v)
	}
	return s, nil
}

func optionalStringArg(input map[string]interface{}, key, defaultValue string) (string, error) {
	if _, ok := input[key]; !ok {
		return defaultValue, nil
	}
	return stringArg(input, key)
}

func runBash(ctx context.Context, input map[string]interface{}) (string, error) {
	command, err := stringArg(input, "command")
	if err != nil {
		return "", err
	}
	output, err := exec.CommandContext(ctx, "bash", "-c", command).CombinedOutput()
	if err != nil {
		log4go.DefaultLogger().Error(ctx, "bash command failed: %v, command: %s, output: %s", err, command, output)
		return "", fmt.Errorf("%w\n%s", err, output)
	}
	return string(output), nil
}

func listDirectory(ctx context.Context, input map[string]interface{}) (string, error) {
	path, err := stringArg(input, "path")
	if err != nil {
		return "", err
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		log4go.DefaultLogger().Error(ctx, "read dir failed: %v, path: %s", err, path)
		return "", err
	}
	var lines []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			name += "/"
		}
		lines = append(lines, name)
	}
	return strings.Join(lines, "\n"), nil
}

func readFile(ctx context.Context, input map[string]interface{}) (string, error) {
	path, err := stringArg(input, "path")
	if err != nil {
		return "", err
	}
	content, err := os.ReadFile(path)
	if err != nil {
		log4go.DefaultLogger().Error(ctx, "read file failed: %v, path: %s", err, path)
		return "", err
	}
	return string(content), nil
}

func writeFile(ctx context.Context, input map[string]interface{}) (string, error) {
	path, err := stringArg(input, "path")
	if err != nil {
		return "", err
	}
	content, err := stringArg(input, "content")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		log4go.DefaultLogger().Error(ctx, "write file failed: %v, path: %s", err, path)
		return "", err
	}
	return fmt.Sprintf("wrote %d bytes to %s", len(content), path), nil
}

func editFile(ctx context.Context, input map[string]interface{}) (string, error) {
	path, err := stringArg(input, "path")
	if err != nil {
		return "", err
	}
	oldString, err := stringArg(input, "old_string")
	if err != nil {
		return "", err
	}
	newString, err := stringArg(input, "new_string")
	if err != nil {
		return "", err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		log4go.DefaultLogger().Error(ctx, "read file failed: %v, path: %s", err, path)
		return "", err
	}
	if n := strings.Count(string(content), oldString); n != 1 {
		log4go.DefaultLogger().Error(ctx, "old_string must appear exactly once, found %d, path: %s", n, path)
		return "", fmt.Errorf("old_string must appear exactly once in %s, found %d", path, n)
	}
	updated := strings.Replace(string(content), oldString, newString, 1)
	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		log4go.DefaultLogger().Error(ctx, "write file failed: %v, path: %s", err, path)
		return "", err
	}
	return fmt.Sprintf("edited %s", path), nil
}

func grepFile(ctx context.Context, input map[string]interface{}) (string, error) {
	pattern, err := stringArg(input, "pattern")
	if err != nil {
		return "", err
	}
	root, err := optionalStringArg(input, "path", ".")
	if err != nil {
		return "", err
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		log4go.DefaultLogger().Error(ctx, "compile pattern failed: %v, pattern: %s", err, pattern)
		return "", err
	}

	var matches []string
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for i, line := range strings.Split(string(content), "\n") {
			if re.MatchString(line) {
				matches = append(matches, fmt.Sprintf("%s:%d: %s", path, i+1, line))
			}
		}
		return nil
	})
	if err != nil {
		log4go.DefaultLogger().Error(ctx, "grep failed: %v, path: %s, pattern: %s", err, root, pattern)
		return "", err
	}
	if len(matches) == 0 {
		return "no matches", nil
	}
	return strings.Join(matches, "\n"), nil
}

func findFile(ctx context.Context, input map[string]interface{}) (string, error) {
	pattern, err := stringArg(input, "pattern")
	if err != nil {
		return "", err
	}
	root, err := optionalStringArg(input, "path", ".")
	if err != nil {
		return "", err
	}
	// validate the glob up front so a bad pattern fails loudly instead of matching nothing
	if _, err := filepath.Match(pattern, ""); err != nil {
		log4go.DefaultLogger().Error(ctx, "bad glob pattern: %v, pattern: %s", err, pattern)
		return "", err
	}

	var matches []string
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if matched, _ := filepath.Match(pattern, d.Name()); matched {
			matches = append(matches, path)
		}
		return nil
	})
	if err != nil {
		log4go.DefaultLogger().Error(ctx, "walk dir failed: %v, root: %s", err, root)
		return "", err
	}
	if len(matches) == 0 {
		return "no matches", nil
	}
	return strings.Join(matches, "\n"), nil
}

/* ^^^^^^^^^^^^^^^^^^^^^ tool handlers ^^^^^^^^^^^^^^^^^^^^^ */
