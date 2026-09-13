// Package trace records an agent run and renders it as a self-contained HTML timeline.
package trace

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	agentloop "github.com/liuliqiang/llmagent/03_permission"
	"github.com/liuliqiang/log4go"
)

//go:embed viewer.html
var viewerHTML string

const jsonPlaceholder = "/*__TRACE_JSON__*/"

type Event struct {
	Seq  int       `json:"seq"`
	Time time.Time `json:"time"`
	// Turn is the 1-based LLM call the event belongs to; the initial user input is turn 0.
	Turn int    `json:"turn"`
	Kind string `json:"kind"` // user | reasoning | text | tool_use | tool_result | error

	Text string `json:"text,omitempty"`

	ToolName   string                 `json:"tool_name,omitempty"`
	ToolID     string                 `json:"tool_id,omitempty"`
	ToolInput  map[string]interface{} `json:"tool_input,omitempty"`
	DurationMS int64                  `json:"duration_ms,omitempty"`
}

type Trace struct {
	Model        string    `json:"model"`
	SystemPrompt string    `json:"system_prompt"`
	StartedAt    time.Time `json:"started_at"`
	EndedAt      time.Time `json:"ended_at"`
	Events       []Event   `json:"events"`
}

// Recorder collects a run into a Trace. It is safe to use from one goroutine;
// the mutex only guards against reading Trace() while the run is still going.
type Recorder struct {
	mu    sync.Mutex
	trace Trace
	now   func() time.Time
}

func NewRecorder() *Recorder {
	return &Recorder{now: time.Now}
}

func (r *Recorder) OnStart(model agentloop.Model, systemPrompt string, userMessages []agentloop.Message) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.trace.Model = string(model)
	r.trace.SystemPrompt = systemPrompt
	r.trace.StartedAt = r.now()
	for _, msg := range userMessages {
		r.add(Event{Turn: 0, Kind: "user", Text: fmt.Sprint(msg.Content)})
	}
}

func (r *Recorder) OnResponse(turn int, resp agentloop.SendMessagesResponse) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, block := range resp.Content {
		switch block.Type {
		case agentloop.MessagesBlockTypeReasoning:
			r.add(Event{Turn: turn, Kind: "reasoning", Text: block.Text})
		case agentloop.MessagesBlockTypeText:
			r.add(Event{Turn: turn, Kind: "text", Text: block.Text})
		case agentloop.MessagesBlockTypeToolUse:
			r.add(Event{Turn: turn, Kind: "tool_use", ToolName: block.Name, ToolID: block.ID, ToolInput: block.Input})
		}
	}
}

func (r *Recorder) OnToolResult(turn int, toolUse agentloop.MessagesBlock, output string, took time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.add(Event{
		Turn:       turn,
		Kind:       "tool_result",
		ToolName:   toolUse.Name,
		ToolID:     toolUse.ID,
		Text:       output,
		DurationMS: took.Milliseconds(),
	})
}

func (r *Recorder) OnEnd(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err != nil {
		r.add(Event{Turn: r.lastTurn(), Kind: "error", Text: err.Error()})
	}
	r.trace.EndedAt = r.now()
}

// Trace returns a copy of what has been recorded so far.
func (r *Recorder) Trace() Trace {
	r.mu.Lock()
	defer r.mu.Unlock()
	t := r.trace
	t.Events = append([]Event(nil), r.trace.Events...)
	return t
}

// WriteHTML renders the trace into a self-contained HTML file at path.
func (r *Recorder) WriteHTML(path string) error {
	html, err := RenderHTML(r.Trace())
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, html, 0o644); err != nil {
		log4go.DefaultLogger().Error(context.Background(), "write trace html failed: %v, path: %s", err, path)
		return fmt.Errorf("write trace html: %w", err)
	}
	return nil
}

// RenderHTML embeds the trace as JSON into the viewer page.
func RenderHTML(t Trace) ([]byte, error) {
	data, err := json.Marshal(t)
	if err != nil {
		log4go.DefaultLogger().Error(context.Background(), "marshal trace failed: %v", err)
		return nil, fmt.Errorf("marshal trace: %w", err)
	}
	// json.Marshal escapes "<" as \u003c, so a "</script>" inside a string
	// value can never terminate the data block early.
	return []byte(strings.Replace(viewerHTML, jsonPlaceholder, string(data), 1)), nil
}

func (r *Recorder) add(e Event) {
	e.Seq = len(r.trace.Events) + 1
	e.Time = r.now()
	r.trace.Events = append(r.trace.Events, e)
}

func (r *Recorder) lastTurn() int {
	if n := len(r.trace.Events); n > 0 {
		return r.trace.Events[n-1].Turn
	}
	return 0
}
