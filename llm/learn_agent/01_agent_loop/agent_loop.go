package agentloop

import "context"

const defaultSystemPrompt = "You are a senior software engineer."

type Agent interface {
	RunLoop(ctx context.Context, messages []Message) error
}

func NewAgent(llmClient LLMClient, recorders ...Recorder) Agent {
	a := &agent{
		currLoop:  0,
		maxLoop:   -1,
		llmClient: llmClient,
		recorders: recorders,
	}
	a.tools = a.generateTools()
	a.toolIndex = make(map[string]Tool, len(a.tools))
	for _, tool := range a.tools {
		a.toolIndex[tool.Name] = tool
	}
	return a
}

type agent struct {
	currLoop int
	maxLoop  int

	messages  []Message
	llmClient LLMClient

	tools     []Tool
	toolIndex map[string]Tool

	recorders []Recorder
}

func (a *agent) RunLoop(ctx context.Context, messages []Message) (err error) {
	a.messages = messages

	for _, r := range a.recorders {
		r.OnStart(a.llmClient.GetModel(), defaultSystemPrompt, messages)
	}
	defer func() {
		for _, r := range a.recorders {
			r.OnEnd(err)
		}
	}()

	for a.shouldContinue() {
		resp, err := a.llmClient.SendMessages(
			ctx,
			a.llmClient.GetModel(),
			Message{
				Role:    MessageRoleSystem,
				Content: defaultSystemPrompt,
			},
			a.messages,
			a.tools,
			NewSendMessagesOpts())
		if err != nil {
			return err
		}

		for _, r := range a.recorders {
			r.OnResponse(a.turn(), resp)
		}

		a.messages = append(a.messages, Message{
			Role:    MessageRoleAssistant,
			Content: resp.GetContent(),
		})

		toolUses := resp.GetToolUsesBlocks()
		if len(toolUses) == 0 {
			return nil
		}

		a.runTools(ctx, toolUses)

		a.currLoop++
	}
	return nil
}

func (a *agent) resetLoop() bool {
	a.currLoop = 0
	return true
}

// turn is the 1-based index of the LLM call currently being processed.
func (a *agent) turn() int {
	return a.currLoop + 1
}

func (a *agent) shouldContinue() bool {
	if a.maxLoop >= 0 {
		return a.currLoop < a.maxLoop
	}
	return true
}

/* ^^^^^^^^^^^^^^^^ agentLoop control logics ^^^^^^^^^^^^^^^^ */
