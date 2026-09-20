package agentloop

import "context"

type Model string

const (
	// anthropic models
	ModelClaudeFable51 Model = "claude-fable-5-1"
	ModelClaudeOpus48  Model = "claude-opus-4-8"
	ModelClaudeOpus50  Model = "claude-opus-5-0"
	ModelClaudeSonnet5 Model = "claude-sonnet-5"
	ModelClaudeHaiku45 Model = "claude-haiku-4-5-20251001"

	// deepseek models
	ModelDeepseekFlash Model = "deepseek-flash"
	ModelDeepseekV4Pro Model = "deepseek-v4-pro"
)

type SystemPrompt struct {
}

// orModel returns m, or fallback when no model was configured.
func (m Model) orModel(fallback Model) Model {
	if m == "" {
		return fallback
	}
	return m
}

type MessageRole string

const (
	MessageRoleSystem    MessageRole = "system"
	MessageRoleUser      MessageRole = "user"
	MessageRoleAssistant MessageRole = "assistant"
)

type Message struct {
	Role    MessageRole
	Content interface{}
}

// ToolHandler executes a tool locally with the input the LLM produced.
// The returned string is fed back to the LLM as the tool result.
type ToolHandler func(ctx context.Context, input map[string]interface{}) (string, error)

type Tool struct {
	Name        string
	Description string
	InputSchema map[string]interface{}
	Handler     ToolHandler `json:"-"`
}

/* vvvvvvvvvvvvvvvvvvvvv SendMessagesOpts vvvvvvvvvvvvvvvvvvvvv */
func NewSendMessagesOpts() SendMessagesOpts {
	return nil
}

type SendMessagesOpts interface {
}

/* ^^^^^^^^^^^^^^^^^^^^^ SendMessagesOpts ^^^^^^^^^^^^^^^^^^^^^ */

/* vvvvvvvvvvvvvvvvvvvvv SendMessagesResponse vvvvvvvvvvvvvvvvv */

type MessagesBlockType string

const (
	MessagesBlockTypeText       MessagesBlockType = "text"
	MessagesBlockTypeReasoning  MessagesBlockType = "reasoning"
	MessagesBlockTypeToolUse    MessagesBlockType = "tool_use"
	MessagesBlockTypeToolResult MessagesBlockType = "tool_result"
)

type MessagesBlock struct {
	ID   string
	Type MessagesBlockType

	// for text / reasoning only
	Text string

	// for tool use only
	Name  string
	Input map[string]interface{}

	// for tool result only
	ToolUseID string
	Content   string
}

// Usage is the token count the model reported for one request; the agent
// uses it to decide when the context needs compacting.
type Usage struct {
	InputTokens  int
	OutputTokens int
}

type SendMessagesResponse struct {
	Content []MessagesBlock
	Usage   Usage
}

func (r SendMessagesResponse) GetContent() []MessagesBlock {
	return r.Content
}

func (r SendMessagesResponse) GetToolUsesBlocks() []MessagesBlock {
	var toolUses []MessagesBlock
	for _, block := range r.Content {
		if block.Type == MessagesBlockTypeToolUse {
			toolUses = append(toolUses, block)
		}
	}
	return toolUses
}

/* ^^^^^^^^^^^^^^^^^^^^^ SendMessagesResponse ^^^^^^^^^^^^^^^^^ */
type LLMClient interface {
	GetModel() Model
	SendMessages(
		ctx context.Context,
		model Model,
		systemPrompt Message,
		messages []Message,
		tools []Tool,
		opts SendMessagesOpts,
	) (SendMessagesResponse, error)
}
