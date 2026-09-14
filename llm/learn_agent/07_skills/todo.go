package agentloop

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/liuliqiang/log4go"
)

// todoOut is where todo_write echoes the list; tests swap it.
var todoOut io.Writer = os.Stdout

type TodoStatus string

const (
	TodoPending    TodoStatus = "pending"
	TodoInProgress TodoStatus = "in_progress"
	TodoCompleted  TodoStatus = "completed"
)

var todoMarkers = map[TodoStatus]string{
	TodoPending:    "[ ]",
	TodoInProgress: "[>]",
	TodoCompleted:  "[x]",
}

type TodoItem struct {
	Content string     `json:"content"`
	Status  TodoStatus `json:"status"`
}

// TodoManager owns the in-memory todo list. Update validates the whole new
// list before replacing the current one, so a bad update never leaves the
// list half-changed.
type TodoManager struct {
	items []TodoItem
}

func NewTodoManager() *TodoManager {
	return &TodoManager{}
}

// Update replaces the list. todos may be the decoded JSON array
// ([]interface{}) or a JSON string containing that array.
func (m *TodoManager) Update(todos interface{}) (string, error) {
	items, err := parseTodos(todos)
	if err != nil {
		return "", err
	}
	m.items = items
	return m.Render(), nil
}

func (m *TodoManager) Items() []TodoItem {
	return append([]TodoItem(nil), m.items...)
}

// Render draws one line per item: [ ] pending, [>] in progress, [x] completed.
func (m *TodoManager) Render() string {
	if len(m.items) == 0 {
		return "(no todos)"
	}
	lines := make([]string, 0, len(m.items))
	for _, item := range m.items {
		lines = append(lines, todoMarkers[item.Status]+" "+item.Content)
	}
	return strings.Join(lines, "\n")
}

func parseTodos(todos interface{}) ([]TodoItem, error) {
	var raw []byte
	switch v := todos.(type) {
	case string:
		raw = []byte(v)
	case []interface{}:
		var err error
		if raw, err = json.Marshal(v); err != nil {
			return nil, fmt.Errorf("encode todos: %w", err)
		}
	default:
		return nil, fmt.Errorf("todos must be an array or a JSON string, got %T", todos)
	}

	var items []TodoItem
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, fmt.Errorf("todos must be a JSON array of {content, status}: %w", err)
	}
	for i, item := range items {
		if strings.TrimSpace(item.Content) == "" {
			return nil, fmt.Errorf("todo #%d: content must not be empty", i+1)
		}
		if _, ok := todoMarkers[item.Status]; !ok {
			return nil, fmt.Errorf("todo #%d: status must be one of pending, in_progress, completed; got %q", i+1, item.Status)
		}
	}
	return items, nil
}

// todoManager backs the todo_write tool. One list per process, like the
// other tools' shared state (the working directory).
var todoManager = NewTodoManager()

// runTodoWrite updates the list and echoes the rendered state to the terminal
// as well as returning it to the model.
func runTodoWrite(ctx context.Context, input map[string]interface{}) (string, error) {
	todos, ok := input["todos"]
	if !ok {
		return "", fmt.Errorf("missing required argument %q", "todos")
	}
	output, err := todoManager.Update(todos)
	if err != nil {
		log4go.DefaultLogger().Error(ctx, "todo_write rejected: %v, input: %+v", err, input)
		return "", err
	}
	fmt.Fprintln(todoOut, output)
	return output, nil
}
