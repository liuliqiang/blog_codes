package agentloop

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"syscall"

	"github.com/liuliqiang/log4go"
)

// tasksDir holds one JSON file per task. It is relative to the working directory like skillsDir; tests swap it.
var tasksDir = ".tasks"

type TaskStatus string

const (
	TaskPending    TaskStatus = "pending"
	TaskInProgress TaskStatus = "in_progress"
	TaskCompleted  TaskStatus = "completed"
)

var taskMarkers = map[TaskStatus]string{TaskPending: "[ ]", TaskInProgress: "[>]", TaskCompleted: "[x]"}

// Task is one persisted unit of work. Unlike a todo item it has an identity, an owner and prerequisites, so the
// harness can decide whether it may start and who is working on it.
type Task struct {
	ID          string     `json:"id"`
	Subject     string     `json:"subject"`
	Description string     `json:"description"`
	Status      TaskStatus `json:"status"`
	Owner       string     `json:"owner"`              // agent that claimed it; "" while pending
	BlockedBy   []string   `json:"blockedBy"`          // IDs that must be completed before this task can be claimed
	Worktree    string     `json:"worktree,omitempty"` // checkout its owner's tools work in; empty means the repository
}

var taskIDRe = regexp.MustCompile(`^task_[0-9a-f]{8}$`)

// TaskStore reads and writes the task files under one directory. Mutations run under both an in-process mutex and an
// advisory lock on <dir>/.lock, so several teammates — or another harness process sharing the directory — cannot
// claim the same task.
type TaskStore struct {
	dir string
	mu  sync.Mutex
}

func NewTaskStore(dir string) *TaskStore { return &TaskStore{dir: dir} }

// withLock runs fn while holding the store lock. The file lock is advisory: it only keeps out processes that take it
// too, which is every process using this package.
func (s *TaskStore) withLock(fn func() error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	ctx := context.Background()
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		log4go.DefaultLogger().Error(ctx, "create tasks dir failed: %v, dir: %s", err, s.dir)
		return err
	}
	lockPath := filepath.Join(s.dir, ".lock")
	f, err := os.OpenFile(lockPath, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		log4go.DefaultLogger().Error(ctx, "open task lock failed: %v, path: %s", err, lockPath)
		return err
	}
	defer f.Close()
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		log4go.DefaultLogger().Error(ctx, "lock task store failed: %v, path: %s", err, lockPath)
		return err
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	return fn()
}

func (s *TaskStore) path(id string) (string, error) {
	if !taskIDRe.MatchString(id) {
		return "", fmt.Errorf("invalid task ID %q", id)
	}
	return filepath.Join(s.dir, id+".json"), nil
}

// Create allocates a random ID and writes the task exclusively, retrying on the unlikely collision.
func (s *TaskStore) Create(subject, description string) (Task, error) {
	ctx := context.Background()
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return Task{}, errors.New("task subject cannot be empty")
	}
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		log4go.DefaultLogger().Error(ctx, "create tasks dir failed: %v, dir: %s", err, s.dir)
		return Task{}, err
	}
	for attempt := 0; attempt < 100; attempt++ {
		var raw [4]byte
		if _, err := rand.Read(raw[:]); err != nil {
			log4go.DefaultLogger().Error(ctx, "generate task ID failed: %v", err)
			return Task{}, err
		}
		task := Task{ID: "task_" + hex.EncodeToString(raw[:]), Subject: subject, Description: description, Status: TaskPending, BlockedBy: []string{}}
		path, _ := s.path(task.ID)
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if errors.Is(err, os.ErrExist) {
			continue
		}
		if err != nil {
			log4go.DefaultLogger().Error(ctx, "create task file failed: %v, path: %s", err, path)
			return Task{}, err
		}
		_, err = f.Write(taskJSON(task))
		if cerr := f.Close(); err == nil {
			err = cerr
		}
		if err != nil {
			log4go.DefaultLogger().Error(ctx, "write task file failed: %v, path: %s", err, path)
			return Task{}, err
		}
		return task, nil
	}
	return Task{}, errors.New("could not allocate a unique task ID")
}

func taskJSON(task Task) []byte {
	out, _ := json.MarshalIndent(task, "", "  ")
	return out
}

// Load reads one task; the file's ID must match the name it was asked for.
func (s *TaskStore) Load(id string) (Task, error) {
	path, err := s.path(id)
	if err != nil {
		return Task{}, err
	}
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Task{}, fmt.Errorf("task not found: %s", id)
		}
		log4go.DefaultLogger().Error(context.Background(), "read task file failed: %v, path: %s", err, path)
		return Task{}, err
	}
	var task Task
	if err := json.Unmarshal(content, &task); err != nil {
		log4go.DefaultLogger().Error(context.Background(), "unmarshal task failed: %v, path: %s", err, path)
		return Task{}, fmt.Errorf("task %s is corrupt: %w", id, err)
	}
	if task.ID != id {
		return Task{}, fmt.Errorf("task file %s holds ID %q", id, task.ID)
	}
	if _, ok := taskMarkers[task.Status]; !ok {
		return Task{}, fmt.Errorf("task %s has invalid status %q", id, task.Status)
	}
	if task.BlockedBy == nil {
		task.BlockedBy = []string{}
	}
	return task, nil
}

// Save replaces the task's file atomically, so a reader in another process never sees a half-written record.
func (s *TaskStore) Save(task Task) error {
	ctx := context.Background()
	path, err := s.path(task.ID)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, taskJSON(task), 0o644); err != nil {
		log4go.DefaultLogger().Error(ctx, "write task temp file failed: %v, path: %s", err, tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		log4go.DefaultLogger().Error(ctx, "rename task file failed: %v, from: %s, to: %s", err, tmp, path)
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// List returns every task sorted by ID. A missing directory is an empty store.
func (s *TaskStore) List() ([]Task, error) {
	files, err := filepath.Glob(filepath.Join(s.dir, "task_*.json"))
	if err != nil {
		log4go.DefaultLogger().Error(context.Background(), "glob tasks dir failed: %v, dir: %s", err, s.dir)
		return nil, err
	}
	sort.Strings(files)
	var tasks []Task
	for _, file := range files {
		task, err := s.Load(strings.TrimSuffix(filepath.Base(file), ".json"))
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}

// AddDependencies appends deps to the task's blockedBy after checking the whole change: the task must still be
// pending and unowned, every dependency must exist, and no edge may point at the task itself or close a cycle.
// Repeating an existing edge is a no-op.
func (s *TaskStore) AddDependencies(id string, deps []string) (out Task, err error) {
	err = s.withLock(func() error {
		out, err = s.addDependencies(id, deps)
		return err
	})
	return out, err
}

func (s *TaskStore) addDependencies(id string, deps []string) (Task, error) {
	task, err := s.Load(id)
	if err != nil {
		return Task{}, err
	}
	if task.Status != TaskPending || task.Owner != "" {
		return Task{}, fmt.Errorf("task %s dependencies can only be updated while pending and unowned", id)
	}
	have := map[string]bool{}
	for _, d := range task.BlockedBy {
		have[d] = true
	}
	seen := map[string]bool{}
	for _, dep := range deps {
		if seen[dep] {
			continue
		}
		seen[dep] = true
		if dep == id {
			return Task{}, errors.New("task cannot depend on itself")
		}
		if _, err := s.Load(dep); err != nil {
			return Task{}, fmt.Errorf("dependency not found: %s", dep)
		}
		if have[dep] {
			continue
		}
		cyclic, err := s.dependsOn(dep, id)
		if err != nil {
			return Task{}, err
		}
		if cyclic {
			return Task{}, fmt.Errorf("dependency cycle detected: %s -> %s", id, dep)
		}
		task.BlockedBy = append(task.BlockedBy, dep)
		have[dep] = true
	}
	if err := s.Save(task); err != nil {
		return Task{}, err
	}
	return task, nil
}

// dependsOn reports whether from transitively depends on target.
func (s *TaskStore) dependsOn(from, target string) (bool, error) {
	stack := []string{from}
	visited := map[string]bool{}
	for len(stack) > 0 {
		cur := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if cur == target {
			return true, nil
		}
		if visited[cur] {
			continue
		}
		visited[cur] = true
		task, err := s.Load(cur)
		if err != nil {
			return false, err
		}
		stack = append(stack, task.BlockedBy...)
	}
	return false, nil
}

// IncompleteDependencies lists the prerequisites that are not completed; a missing prerequisite counts as incomplete.
func (s *TaskStore) IncompleteDependencies(task Task) []string {
	var blocked []string
	for _, dep := range task.BlockedBy {
		d, err := s.Load(dep)
		if err != nil || d.Status != TaskCompleted {
			blocked = append(blocked, dep)
		}
	}
	return blocked
}

// CanStart reports whether every prerequisite of the task is completed.
func (s *TaskStore) CanStart(task Task) bool { return len(s.IncompleteDependencies(task)) == 0 }

// Claim moves a pending, unblocked task to in_progress under owner.
func (s *TaskStore) Claim(id, owner string) (out Task, err error) {
	err = s.withLock(func() error {
		out, err = s.claim(id, owner)
		return err
	})
	return out, err
}

func (s *TaskStore) claim(id, owner string) (Task, error) {
	task, err := s.Load(id)
	if err != nil {
		return Task{}, err
	}
	if task.Status != TaskPending {
		return Task{}, fmt.Errorf("task %s is %s, cannot claim", id, task.Status)
	}
	if blocked := s.IncompleteDependencies(task); len(blocked) > 0 {
		return Task{}, fmt.Errorf("task %s is blocked by: %s", id, strings.Join(blocked, ", "))
	}
	if busy, err := s.inProgressFor(owner); err != nil {
		return Task{}, err
	} else if busy != "" {
		return Task{}, fmt.Errorf("%s must complete %s before claiming another task", owner, busy)
	}
	if _, err := taskWorktreeCwd(task); err != nil {
		return Task{}, fmt.Errorf("cannot claim %s: %w", id, err)
	}
	task.Owner = owner
	task.Status = TaskInProgress
	if err := s.Save(task); err != nil {
		return Task{}, err
	}
	return task, nil
}

// Complete marks an in_progress task owned by owner as completed and returns the tasks that just became startable.
func (s *TaskStore) Complete(id, owner string) (task Task, unblocked []Task, err error) {
	err = s.withLock(func() error {
		task, unblocked, err = s.complete(id, owner)
		return err
	})
	return task, unblocked, err
}

func (s *TaskStore) complete(id, owner string) (Task, []Task, error) {
	task, err := s.Load(id)
	if err != nil {
		return Task{}, nil, err
	}
	if task.Status != TaskInProgress {
		return Task{}, nil, fmt.Errorf("task %s is %s, cannot complete", id, task.Status)
	}
	if task.Owner != owner {
		return Task{}, nil, fmt.Errorf("task %s is owned by %s, not %s", id, task.Owner, owner)
	}
	before, err := s.List()
	if err != nil {
		return Task{}, nil, err
	}
	readyBefore := map[string]bool{}
	for _, t := range before {
		if t.Status == TaskPending && len(t.BlockedBy) > 0 && s.CanStart(t) {
			readyBefore[t.ID] = true
		}
	}

	task.Status = TaskCompleted
	if err := s.Save(task); err != nil {
		return Task{}, nil, err
	}

	after, err := s.List()
	if err != nil {
		return Task{}, nil, err
	}
	var unblocked []Task
	for _, t := range after {
		if t.Status == TaskPending && len(t.BlockedBy) > 0 && !readyBefore[t.ID] && s.CanStart(t) {
			unblocked = append(unblocked, t)
		}
	}
	return task, unblocked, nil
}

/* vvvvvvvvvvvvvvvvvvvvv tool handlers vvvvvvvvvvvvvvvvvvvvv */

const taskSystemPromptGuidance = `For work whose steps depend on each other or that must survive this session, use the task tools instead of todo_write: create every task with create_task first, then add dependencies with update_task using the exact IDs create_task returned. Claim a task with claim_task before working on it and call complete_task when it is done.`

func (a *agent) runCreateTask(ctx context.Context, input map[string]interface{}) (string, error) {
	subject, err := stringArg(input, "subject")
	if err != nil {
		return "", err
	}
	description, err := optionalStringArg(input, "description", "")
	if err != nil {
		return "", err
	}
	task, err := a.tasks.Create(subject, description)
	if err != nil {
		log4go.DefaultLogger().Error(ctx, "[%s] create_task failed: %v, input: %+v", agentNameFrom(ctx), err, input)
		return "", err
	}
	return fmt.Sprintf("Created %s: %s", task.ID, task.Subject), nil
}

func (a *agent) runUpdateTask(ctx context.Context, input map[string]interface{}) (string, error) {
	id, err := stringArg(input, "task_id")
	if err != nil {
		return "", err
	}
	deps, err := stringSliceArg(input, "addBlockedBy")
	if err != nil {
		return "", err
	}
	if len(deps) == 0 {
		return "", errors.New("addBlockedBy must list at least one task ID")
	}
	task, err := a.tasks.AddDependencies(id, deps)
	if err != nil {
		log4go.DefaultLogger().Error(ctx, "[%s] update_task failed: %v, input: %+v", agentNameFrom(ctx), err, input)
		return "", err
	}
	return fmt.Sprintf("Updated %s blockedBy: %s", task.ID, strings.Join(task.BlockedBy, ", ")), nil
}

func (a *agent) runListTasks(ctx context.Context, _ map[string]interface{}) (string, error) {
	tasks, err := a.tasks.List()
	if err != nil {
		log4go.DefaultLogger().Error(ctx, "[%s] list_tasks failed: %v", agentNameFrom(ctx), err)
		return "", err
	}
	if len(tasks) == 0 {
		return "No tasks. Use create_task to add some.", nil
	}
	var lines []string
	for _, t := range tasks {
		line := fmt.Sprintf("%s %s: %s [%s]", taskMarkers[t.Status], t.ID, t.Subject, t.Status)
		if t.Owner != "" {
			line += " [" + t.Owner + "]"
		}
		if len(t.BlockedBy) > 0 {
			line += " (blockedBy: " + strings.Join(t.BlockedBy, ", ") + ")"
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n"), nil
}

func (a *agent) runGetTask(ctx context.Context, input map[string]interface{}) (string, error) {
	id, err := stringArg(input, "task_id")
	if err != nil {
		return "", err
	}
	task, err := a.tasks.Load(id)
	if err != nil {
		log4go.DefaultLogger().Error(ctx, "[%s] get_task failed: %v, id: %s", agentNameFrom(ctx), err, id)
		return "", err
	}
	return string(taskJSON(task)), nil
}

func (a *agent) runClaimTask(ctx context.Context, input map[string]interface{}) (string, error) {
	id, err := stringArg(input, "task_id")
	if err != nil {
		return "", err
	}
	task, err := a.tasks.Claim(id, agentNameFrom(ctx))
	if err != nil {
		log4go.DefaultLogger().Error(ctx, "[%s] claim_task failed: %v, id: %s", agentNameFrom(ctx), err, id)
		return "", err
	}
	return fmt.Sprintf("Claimed %s (%s)", task.ID, task.Subject), nil
}

func (a *agent) runCompleteTask(ctx context.Context, input map[string]interface{}) (string, error) {
	id, err := stringArg(input, "task_id")
	if err != nil {
		return "", err
	}
	task, unblocked, err := a.tasks.Complete(id, agentNameFrom(ctx))
	if err != nil {
		log4go.DefaultLogger().Error(ctx, "[%s] complete_task failed: %v, id: %s", agentNameFrom(ctx), err, id)
		return "", err
	}
	msg := fmt.Sprintf("Completed %s (%s)", task.ID, task.Subject)
	if len(unblocked) > 0 {
		var subjects []string
		for _, t := range unblocked {
			subjects = append(subjects, t.Subject)
		}
		msg += "\nUnblocked: " + strings.Join(subjects, ", ")
	}
	return msg, nil
}

// stringSliceArg reads a JSON array of strings; a JSON-encoded string holding such an array is accepted too, as
// some models serialize array arguments that way.
func stringSliceArg(input map[string]interface{}, key string) ([]string, error) {
	v, ok := input[key]
	if !ok {
		return nil, fmt.Errorf("missing required argument %q", key)
	}
	if s, ok := v.(string); ok {
		var decoded []interface{}
		if err := json.Unmarshal([]byte(s), &decoded); err != nil {
			return nil, fmt.Errorf("argument %q must be an array of strings", key)
		}
		v = decoded
	}
	items, ok := v.([]interface{})
	if !ok {
		return nil, fmt.Errorf("argument %q must be an array of strings, got %T", key, v)
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		s, ok := item.(string)
		if !ok {
			return nil, fmt.Errorf("argument %q must contain only strings, got %T", key, item)
		}
		out = append(out, s)
	}
	return out, nil
}

// inProgressFor returns the ID of the task owner is already working on, "" when it is free.
func (s *TaskStore) inProgressFor(owner string) (string, error) {
	tasks, err := s.List()
	if err != nil {
		return "", err
	}
	for _, t := range tasks {
		if t.Status == TaskInProgress && t.Owner == owner {
			return t.ID, nil
		}
	}
	return "", nil
}

// ClaimNext claims the first task that is pending, unowned and unblocked, and reports whether it found one. Scanning
// only produces candidates: the claim itself happens under the store lock, so when several teammates see the same
// task exactly one of them gets it.
func (s *TaskStore) ClaimNext(owner string) (claimed Task, ok bool, err error) {
	err = s.withLock(func() error {
		if busy, err := s.inProgressFor(owner); err != nil || busy != "" {
			return err
		}
		tasks, err := s.List()
		if err != nil {
			return err
		}
		for _, t := range tasks {
			if t.Status != TaskPending || t.Owner != "" || !s.CanStart(t) {
				continue
			}
			claimed, err = s.claim(t.ID, owner)
			if err != nil {
				return err
			}
			ok = true
			return nil
		}
		return nil
	})
	return claimed, ok, err
}

// Release hands a claimed task back to the board, used when a teammate could not finish it.
func (s *TaskStore) Release(id, owner string) error {
	return s.withLock(func() error {
		task, err := s.Load(id)
		if err != nil {
			return err
		}
		if task.Status != TaskInProgress || task.Owner != owner {
			return fmt.Errorf("task %s is not in progress for %s", id, owner)
		}
		task.Status, task.Owner = TaskPending, ""
		return s.Save(task)
	})
}
