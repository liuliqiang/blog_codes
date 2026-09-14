package agentloop

import (
	"fmt"
	"time"
)

// Recorder observes an agent run. RunLoop calls the hooks in order:
// OnStart, then per turn OnResponse and OnToolResult (one per tool use),
// and finally OnEnd exactly once, even when the run fails.
type Recorder interface {
	OnStart(model Model, systemPrompt string, userMessages []Message)
	OnResponse(turn int, resp SendMessagesResponse)
	OnToolResult(turn int, toolUse MessagesBlock, output string, took time.Duration)
	OnEnd(err error)
}

// NewStdoutRecorder prints every step of the run to stdout.
func NewStdoutRecorder() Recorder {
	return stdoutRecorder{}
}

type stdoutRecorder struct{}

func (stdoutRecorder) OnStart(model Model, _ string, _ []Message) {
	fmt.Printf("Model: %s\n", model)
}

func (stdoutRecorder) OnResponse(_ int, resp SendMessagesResponse) {
	for _, block := range resp.Content {
		switch block.Type {
		case MessagesBlockTypeText:
			fmt.Printf("Text: %s\n", block.Text)
		case MessagesBlockTypeReasoning:
			fmt.Printf("Assistant (reasoning): %s\n", block.Text)
		case MessagesBlockTypeToolUse:
			fmt.Printf("Assistant (tool use): %s %v\n", block.Name, block.Input)
		case MessagesBlockTypeToolResult:
			fmt.Printf("Assistant (tool result): %s %s\n", block.ToolUseID, block.Content)
		default:
			fmt.Printf("Unknown block type: %s\n", block.Type)
		}
	}
}

func (stdoutRecorder) OnToolResult(_ int, toolUse MessagesBlock, output string, took time.Duration) {
	fmt.Printf("Tool result (%s, %s): %s\n", toolUse.Name, took.Round(time.Millisecond), output)
}

func (stdoutRecorder) OnEnd(err error) {
	if err != nil {
		fmt.Printf("Agent failed: %v\n", err)
	}
}
