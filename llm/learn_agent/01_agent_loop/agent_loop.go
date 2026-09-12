package agentloop

import (
	"context"
	"fmt"
)

type Agent interface {
	RunLoop(ctx context.Context, messages []Message) error
}

func NewAgent(llmClient LLMClient) Agent {
	return &agent{
		currLoop:  0,
		maxLoop:   -1,
		llmClient: llmClient,
	}
}

type agent struct {
	currLoop int
	maxLoop  int

	messages  []Message
	llmClient LLMClient
}

func (a *agent) RunLoop(ctx context.Context, messages []Message) error {
	a.messages = messages

	for a.shouldContinue() {
		resp, err := a.llmClient.SendMessages(
			ctx,
			a.llmClient.GetModel(),
			Message{
				Role:    MessageRoleSystem,
				Content: "You are a senior software engineer.",
			},
			a.messages,
			nil,
			NewSendMessagesOpts())
		if err != nil {
			return err
		}

		a.showResponse(resp)

		a.messages = append(a.messages, Message{
			Role:    MessageRoleAssistant,
			Content: resp.GetContent(),
		})

		toolUses := resp.GetToolUsesBlocks()
		if len(toolUses) == 0 {
			return nil
		}

		if err := a.runTools(toolUses); err != nil {
			return err
		}

		a.currLoop++
	}
	return nil
}

func (a *agent) runTools(toolUses []MessagesBlock) error {
	var results []MessagesBlock
	for _, toolUse := range toolUses {
		output, err := runBash(toolUse.Input["command"].(string))
		if err != nil {
			return err
		}

		results = append(results, MessagesBlock{
			Type:      MessagesBlockTypeToolResult,
			ToolUseID: toolUse.ID,
			Content:   output,
		})
	}

	a.messages = append(a.messages, Message{
		Role:    MessageRoleUser,
		Content: results,
	})
	return nil
}

func (a *agent) showResponse(resp SendMessagesResponse) {
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

/* vvvvvvvvvvvvvvvv agentLoop control logics vvvvvvvvvvvvvvvv */

func (a *agent) resetLoop() bool {
	a.currLoop = 0
	return true
}

func (a *agent) shouldContinue() bool {
	if a.maxLoop >= 0 {
		return a.currLoop < a.maxLoop
	}
	return true
}

/* ^^^^^^^^^^^^^^^^ agentLoop control logics ^^^^^^^^^^^^^^^^ */
