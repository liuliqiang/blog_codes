package agentloop

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/liuliqiang/log4go"
)

// backgroundOut is where background task activity is printed; tests swap it.
var backgroundOut io.Writer = os.Stdout

// BackgroundConfig bounds background bash commands. It is read when an agent is built.
type BackgroundConfig struct {
	MaxConcurrent int           // commands running at once; further tasks wait in a queue. <= 0 means no limit
	Timeout       time.Duration // per command, from the moment it starts running
}

// Background is the active config.
var Background = BackgroundConfig{
	MaxConcurrent: 4,
	Timeout:       10 * time.Minute,
}

const (
	// backgroundSummaryLimit is how much of a finished command's output goes into its notification.
	backgroundSummaryLimit = 500
	// backgroundOutputLimit caps what a background command may hand back at all.
	backgroundOutputLimit = 50_000
)

type backgroundStatus string

const (
	backgroundQueued    backgroundStatus = "queued"
	backgroundRunning   backgroundStatus = "running"
	backgroundCompleted backgroundStatus = "completed"
	backgroundFailed    backgroundStatus = "failed"
)

type backgroundTask struct {
	id      string
	command string
	status  backgroundStatus
	result  string
}

// BackgroundManager runs bash commands in goroutines and queues their results until the loop collects them. At most
// Background.MaxConcurrent commands run at once, the rest wait their turn. Each command gets its own process group so
// the whole tree can be stopped when the agent shuts down.
type BackgroundManager struct {
	mu      sync.Mutex
	tasks   map[string]*backgroundTask
	ready   []string // finished task ids not yet collected, in completion order
	counter int
	wg      sync.WaitGroup
	sem     chan struct{} // nil when unlimited
	timeout time.Duration

	// ctx is the lifetime of the current batch of tasks: Shutdown cancels it, which kills running commands, makes
	// commands that have not started yet fail on Start, and releases queued tasks.
	ctx    context.Context
	cancel context.CancelFunc
}

func NewBackgroundManager() *BackgroundManager {
	m := &BackgroundManager{
		tasks:   map[string]*backgroundTask{},
		timeout: Background.Timeout,
	}
	m.ctx, m.cancel = context.WithCancel(context.Background())
	if Background.MaxConcurrent > 0 {
		m.sem = make(chan struct{}, Background.MaxConcurrent)
	}
	return m
}

// Start launches command and returns its task id right away.
func (m *BackgroundManager) Start(ctx context.Context, command string) (string, error) {
	if strings.TrimSpace(command) == "" {
		return "", errors.New("bash command cannot be empty")
	}
	m.mu.Lock()
	m.counter++
	id := fmt.Sprintf("bg_%04d", m.counter)
	task := &backgroundTask{id: id, command: command, status: backgroundQueued}
	m.tasks[id] = task
	life := m.ctx
	m.mu.Unlock()

	m.wg.Add(1)
	go m.run(ctx, life, task)
	fmt.Fprintf(
		backgroundOut,
		"\033[36m[background] started %s: %s\033[0m\n",
		id,
		excerpt(command, 60), // command preview
	)
	return id, nil
}

// run executes one task. ctx is the tool call's ctx and is only used for logging; the command's own lifetime is life,
// so it outlives the tool call that started it but not the manager.
func (m *BackgroundManager) run(ctx, life context.Context, task *backgroundTask) {
	defer m.wg.Done()
	if m.sem != nil {
		select {
		case m.sem <- struct{}{}:
			defer func() { <-m.sem }()
		case <-life.Done():
			m.finish(task, backgroundFailed, "Error: agent shut down before the task started")
			return
		}
	}
	m.mu.Lock()
	task.status = backgroundRunning
	m.mu.Unlock()

	runCtx, cancel := context.WithTimeout(life, m.timeout)
	defer cancel()
	cmd := exec.CommandContext(runCtx, "bash", "-c", task.command)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return killGroup(cmd) }

	output, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(output))
	if len(text) > backgroundOutputLimit {
		text = text[:backgroundOutputLimit]
	}
	if text == "" {
		text = "(no output)"
	}
	status := backgroundCompleted
	switch {
	case life.Err() != nil:
		status = backgroundFailed
		text = "Error: agent shut down while the task was running\n" + text
	case runCtx.Err() == context.DeadlineExceeded:
		status = backgroundFailed
		text = fmt.Sprintf("Error: timeout after %s\n%s", m.timeout, text)
		log4go.DefaultLogger().Error(ctx, "background task %s timed out, command: %s", task.id, task.command)
	case err != nil:
		status = backgroundFailed
		text = fmt.Sprintf("Error: %v\n%s", err, text)
		log4go.DefaultLogger().Error(ctx, "background task %s failed: %v, command: %s, output: %s", task.id, err, task.command, output)
	}
	m.finish(task, status, text)
}

// finish records the outcome and queues the task for collection.
func (m *BackgroundManager) finish(task *backgroundTask, status backgroundStatus, result string) {
	m.mu.Lock()
	task.status, task.result = status, result
	m.ready = append(m.ready, task.id)
	m.mu.Unlock()
}

// killGroup stops the command's whole process group; a command that already exited is not an error.
func killGroup(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM); err != nil && !errors.Is(err, syscall.ESRCH) {
		return err
	}
	return nil
}

// Collect removes every finished task from the queue and renders each as a notification for the model.
func (m *BackgroundManager) Collect() []string {
	m.mu.Lock()
	var done []*backgroundTask
	for _, id := range m.ready {
		if task, ok := m.tasks[id]; ok {
			done = append(done, task)
			delete(m.tasks, id)
		}
	}
	m.ready = nil
	m.mu.Unlock()

	var notes []string
	for _, task := range done {
		notes = append(notes, fmt.Sprintf(
			"<task_notification>\n  <task_id>%s</task_id>\n  <status>%s</status>\n  <command>%s</command>\n  <summary>%s</summary>\n</task_notification>",
			task.id, task.status, task.command, excerpt(task.result, backgroundSummaryLimit)))
		fmt.Fprintf(backgroundOut, "\033[36m[background] collected %s: %s\033[0m\n", task.id, task.status)
	}
	return notes
}

// Pending reports how many tasks are still running or finished but not yet collected.
func (m *BackgroundManager) Pending() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.tasks)
}

// Wait blocks until every running task has finished or timeout passes; it reports whether all of them did.
func (m *BackgroundManager) Wait(timeout time.Duration) bool {
	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return true
	case <-time.After(timeout):
		return false
	}
}

// Shutdown stops every running command's process group, releases queued tasks and drops all state. The manager can
// be used again afterwards.
func (m *BackgroundManager) Shutdown() {
	m.mu.Lock()
	m.cancel()
	m.mu.Unlock()
	m.wg.Wait()
	m.mu.Lock()
	m.tasks = map[string]*backgroundTask{}
	m.ready = nil
	m.ctx, m.cancel = context.WithCancel(context.Background())
	m.mu.Unlock()
}

/* vvvvvvvvvvvvvvvvvvvvv loop integration vvvvvvvvvvvvvvvvvvvvv */

// shouldRunBackground reports whether a tool call asked to run in the background.
func shouldRunBackground(toolUse MessagesBlock) bool {
	if toolUse.Name != "run_bash" {
		return false
	}
	flag, _ := toolUse.Input["run_in_background"].(bool)
	return flag
}

// startBackground is the handler path for a background bash call: it returns the placeholder tool result.
func (a *agent) startBackground(ctx context.Context, toolUse MessagesBlock) string {
	command, err := stringArg(toolUse.Input, "command")
	if err != nil {
		return "error: " + err.Error()
	}
	id, err := a.background.Start(ctx, command)
	if err != nil {
		log4go.DefaultLogger().Error(ctx, "[%s] start background task failed: %v, command: %s", agentNameFrom(ctx), err, command)
		return "error: " + err.Error()
	}
	return fmt.Sprintf("[Background task %s started] Its result will arrive in a later turn as a <task_notification>; carry on with work that does not depend on it.", id)
}

// injectBackgroundResults appends finished background results to the conversation before the next LLM call. They
// go into the trailing user message when there is one, otherwise into a new one, so one tool_use still gets exactly
// one tool_result.
func (a *agent) injectBackgroundResults() {
	notes := a.background.Collect()
	if len(notes) == 0 {
		return
	}
	var blocks []MessagesBlock
	for _, note := range notes {
		blocks = append(blocks, MessagesBlock{Type: MessagesBlockTypeText, Text: note})
	}
	if n := len(a.messages); n > 0 && a.messages[n-1].Role == MessageRoleUser {
		last := &a.messages[n-1]
		switch content := last.Content.(type) {
		case []MessagesBlock:
			last.Content = append(content, blocks...)
		case string:
			last.Content = append([]MessagesBlock{{Type: MessagesBlockTypeText, Text: content}}, blocks...)
		default:
			a.messages = append(a.messages, Message{Role: MessageRoleUser, Content: blocks})
		}
		return
	}
	a.messages = append(a.messages, Message{Role: MessageRoleUser, Content: blocks})
}

// backgroundWaitHint is what the model sees when it tries to stop while background tasks are still pending.
const backgroundWaitHint = "<reminder>%d background task(s) are still pending. Wait for their <task_notification> before finishing, or do other work meanwhile.</reminder>"

// BackgroundTasksHook keeps the loop alive while background tasks are pending: the model would otherwise stop and
// their results would be lost. Finished tasks are collected first so the model sees them instead of the reminder.
func BackgroundTasksHook() StopHook {
	return func(ctx context.Context, _ []Message) (*Message, error) {
		a := agentFrom(ctx)
		if a == nil || a.background.Pending() == 0 {
			return nil, nil
		}
		if notes := a.background.Collect(); len(notes) > 0 {
			var blocks []MessagesBlock
			for _, note := range notes {
				blocks = append(blocks, MessagesBlock{Type: MessagesBlockTypeText, Text: note})
			}
			return &Message{Role: MessageRoleUser, Content: blocks}, nil
		}
		// nothing finished yet: give the running commands a moment before nudging the model again
		a.background.Wait(backgroundPollInterval)
		hint := fmt.Sprintf(
			backgroundWaitHint,
			a.background.Pending(), // pending count
		)
		return &Message{Role: MessageRoleUser, Content: hint}, nil
	}
}

// backgroundPollInterval is how long BackgroundTasksHook waits for running tasks before re-prompting the model.
var backgroundPollInterval = 5 * time.Second
