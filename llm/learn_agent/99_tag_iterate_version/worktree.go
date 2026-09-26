package agentloop

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/liuliqiang/log4go"
)

// worktreesDir holds the checkouts bound to tasks, relative to the working directory; tests swap it.
var worktreesDir = ".worktrees"

var worktreeNameRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,63}$`)

// worktreePath is where a named worktree lives. A name that is not a plain segment is rejected.
func worktreePath(name string) (string, error) {
	if !worktreeNameRe.MatchString(name) || name == "." || name == ".." {
		return "", fmt.Errorf("invalid worktree name %q", name)
	}
	return filepath.Join(worktreesDir, name), nil
}

// CreateWorktree checks out a new git worktree on branch wt/<name> and binds it to a task. The task must still be
// pending, unowned and unbound, so a running teammate never has the ground moved under it.
func (s *TaskStore) CreateWorktree(ctx context.Context, name, taskID string) (Task, error) {
	path, err := worktreePath(name)
	if err != nil {
		return Task{}, err
	}
	var out Task
	err = s.withLock(func() error {
		task, err := s.Load(taskID)
		if err != nil {
			return err
		}
		if task.Status != TaskPending || task.Owner != "" {
			return fmt.Errorf("task %s must be pending and unowned to bind a worktree", taskID)
		}
		if task.Worktree != "" {
			return fmt.Errorf("task %s is already bound to worktree %s", taskID, task.Worktree)
		}
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("worktree %s already exists at %s", name, path)
		}
		cmd := exec.CommandContext(ctx, "git", "worktree", "add", path, "-b", "wt/"+name)
		if output, err := cmd.CombinedOutput(); err != nil {
			log4go.DefaultLogger().Error(ctx, "git worktree add failed: %v, name: %s, output: %s", err, name, output)
			return fmt.Errorf("git worktree add: %w\n%s", err, output)
		}
		task.Worktree = name
		if err := s.Save(task); err != nil {
			// git already made the checkout and the branch: report the partial state instead of pretending either
			// side succeeded, and leave both for manual recovery
			log4go.DefaultLogger().Error(ctx, "binding worktree %s to %s failed after the checkout was created: %v", name, taskID, err)
			return fmt.Errorf("worktree %s was created at %s but task %s could not be bound to it: %w", name, path, taskID, err)
		}
		out = task
		return nil
	})
	return out, err
}

// taskWorktreeCwd is the directory a task's tools work in: its worktree when it is bound to one, otherwise the
// working directory. A binding that no longer resolves fails closed rather than quietly using the repository.
func taskWorktreeCwd(task Task) (string, error) {
	if task.Worktree == "" {
		return os.Getwd()
	}
	path, err := worktreePath(task.Worktree)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		log4go.DefaultLogger().Error(context.Background(), "worktree %s for task %s is missing: %v", task.Worktree, task.ID, err)
		return "", fmt.Errorf("worktree %s for task %s is missing at %s", task.Worktree, task.ID, path)
	}
	return filepath.Abs(path)
}

// RemoveWorktree deletes a task-bound checkout. It is deliberately not a tool: the host asks the user first, because
// the decision needs a look at task ownership and git status. The wt/<name> branch is always kept.
func RemoveWorktree(ctx context.Context, store *TaskStore, name string, discardChanges bool) error {
	path, err := worktreePath(name)
	if err != nil {
		return err
	}
	tasks, err := store.List()
	if err != nil {
		return err
	}
	var bound *Task
	for i := range tasks {
		if tasks[i].Worktree == name {
			bound = &tasks[i]
			break
		}
	}
	if bound != nil && bound.Status != TaskCompleted {
		return fmt.Errorf("task %s is %s in %s; finish or release it first", bound.ID, bound.Status, name)
	}
	if !discardChanges {
		status, err := exec.CommandContext(ctx, "git", "-C", path, "status", "--porcelain", "--ignored").CombinedOutput()
		if err != nil {
			log4go.DefaultLogger().Error(ctx, "git status in worktree %s failed: %v, output: %s", name, err, status)
			return fmt.Errorf("git status: %w\n%s", err, status)
		}
		if strings.TrimSpace(string(status)) != "" {
			return fmt.Errorf("worktree %s has local changes; inspect it, or pass discardChanges once the user agreed:\n%s", name, status)
		}
	}
	args := []string{"worktree", "remove", path}
	if discardChanges {
		args = append(args, "--force")
	}
	if output, err := exec.CommandContext(ctx, "git", args...).CombinedOutput(); err != nil {
		log4go.DefaultLogger().Error(ctx, "git worktree remove failed: %v, name: %s, output: %s", err, name, output)
		return fmt.Errorf("git worktree remove: %w\n%s", err, output)
	}
	if bound != nil {
		return store.withLock(func() error {
			task, err := store.Load(bound.ID)
			if err != nil {
				return err
			}
			task.Worktree = ""
			return store.Save(task)
		})
	}
	return nil
}

/* vvvvvvvvvvvvvvvvvvvvv workspace tools vvvvvvvvvvvvvvvvvvvvv */

// workspaceTools read or change files; for a teammate they run in its current task's directory.
var workspaceTools = map[string]bool{
	"run_bash":       true,
	"list_directory": true,
	"read_file":      true,
	"write_file":     true,
	"edit_file":      true,
	"grep_file":      true,
	"find_file":      true,
}

// assignmentCwd is the directory the agent's workspace tools operate in. A teammate must hold a task: without one
// there is no directory to fall back to, and silently using the repository would be the wrong answer when the work
// belongs in a worktree.
func (a *agent) assignmentCwd() (string, error) {
	id, err := a.tasks.inProgressFor(a.name)
	if err != nil {
		return "", err
	}
	if id == "" {
		return "", fmt.Errorf("claim a task with claim_task before using workspace tools")
	}
	task, err := a.tasks.Load(id)
	if err != nil {
		return "", err
	}
	return taskWorktreeCwd(task)
}

// inAssignment wraps a workspace tool so its paths resolve inside the agent's assigned directory and can never point
// outside it.
func (a *agent) inAssignment(tool Tool) Tool {
	handler := tool.Handler
	name := tool.Name
	tool.Handler = func(ctx context.Context, input map[string]interface{}) (string, error) {
		dir, err := a.assignmentCwd()
		if err != nil {
			log4go.DefaultLogger().Error(ctx, "[%s] %s has no assigned directory: %v", agentNameFrom(ctx), name, err)
			return "", err
		}
		rebased, err := rebaseToDir(name, input, dir)
		if err != nil {
			log4go.DefaultLogger().Error(ctx, "[%s] %s outside its assignment: %v, input: %+v", agentNameFrom(ctx), name, err, input)
			return "", err
		}
		return handler(ctx, rebased)
	}
	return tool
}

// rebaseToDir returns a copy of input whose path (or bash command) is anchored at dir.
func rebaseToDir(name string, input map[string]interface{}, dir string) (map[string]interface{}, error) {
	out := make(map[string]interface{}, len(input)+1)
	for k, v := range input {
		out[k] = v
	}
	if name == "run_bash" {
		command, err := stringArg(input, "command")
		if err != nil {
			return nil, err
		}
		out["command"] = "cd " + shellQuote(dir) + " && " + command
		return out, nil
	}

	raw, ok := input["path"].(string)
	if !ok || raw == "" {
		if name == "grep_file" || name == "find_file" {
			raw = "." // these default to the working directory, which for a teammate is its assignment
		} else {
			return nil, fmt.Errorf("missing required argument %q", "path")
		}
	}
	resolved := raw
	if !filepath.IsAbs(resolved) {
		resolved = filepath.Join(dir, resolved)
	}
	resolved = filepath.Clean(resolved)
	if rel, err := filepath.Rel(dir, resolved); err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return nil, fmt.Errorf("path %q is outside the assigned directory %s", raw, dir)
	}
	out["path"] = resolved
	return out, nil
}

func shellQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }

func (a *agent) runCreateWorktree(ctx context.Context, input map[string]interface{}) (string, error) {
	name, err := stringArg(input, "name")
	if err != nil {
		return "", err
	}
	taskID, err := stringArg(input, "task_id")
	if err != nil {
		return "", err
	}
	task, err := a.tasks.CreateWorktree(ctx, name, taskID)
	if err != nil {
		log4go.DefaultLogger().Error(ctx, "[%s] create_worktree failed: %v, input: %+v", agentNameFrom(ctx), err, input)
		return "", err
	}
	path, _ := worktreePath(name)
	return fmt.Sprintf("Created worktree %s at %s on branch wt/%s and bound it to %s; whoever claims that task works there.", name, path, name, task.ID), nil
}
