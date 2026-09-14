package agentloop

import (
	"context"
	"errors"
	"strings"

	"github.com/liuliqiang/log4go"
)

const subagentSystemPrompt = `You are a senior software engineer working on a single delegated task.

Complete the task with the tools available, then reply with a concise summary of what you found or did. Your final message is the only thing returned to the caller, so make it self-contained.`

// subagentMaxLoop caps a subagent's rounds so a runaway delegate cannot burn
// the whole session.
const subagentMaxLoop = 30

// subagentExcludedTools are not handed to a subagent: task would allow
// unbounded recursion, and todo_write would clobber the parent's shared list.
var subagentExcludedTools = map[string]bool{
	"task":       true,
	"todo_write": true,
}

// newSubagent builds a child agent that shares the parent's LLM client and
// hooks but starts with fresh messages, no recorders and a reduced tool set.
func (a *agent) newSubagent() *agent {
	sub := &agent{
		name:         "subagent",
		maxLoop:      subagentMaxLoop,
		systemPrompt: subagentSystemPrompt,
		llmClient:    a.llmClient,
		hooks:        a.hooks.snapshot(),
	}
	for _, tool := range a.tools {
		if subagentExcludedTools[tool.Name] {
			continue
		}
		sub.tools = append(sub.tools, tool)
	}
	sub.toolIndex = indexTools(sub.tools)
	return sub
}

// runTask is the handler of the task tool: it runs the prompt in a fresh
// subagent and returns that subagent's final text.
func (a *agent) runTask(ctx context.Context, input map[string]interface{}) (string, error) {
	prompt, err := stringArg(input, "prompt")
	if err != nil {
		return "", err
	}
	sub := a.newSubagent()
	err = sub.RunLoop(ctx, []Message{{Role: MessageRoleUser, Content: prompt}})
	if err != nil {
		log4go.DefaultLogger().Error(ctx, "subagent failed: %v, prompt: %s", err, prompt)
		return "", err
	}
	answer, ok := sub.finalText()
	if !ok {
		log4go.DefaultLogger().Error(ctx, "subagent stopped without a final answer after %d rounds, prompt: %s", sub.maxLoop, prompt)
		return "", errors.New("subagent stopped without a final answer")
	}
	return answer, nil
}

// finalText returns the text of the last assistant message. It reports false
// when that message still asks for tools, which means the loop was cut off by
// maxLoop rather than finished by the model.
func (a *agent) finalText() (string, bool) {
	for i := len(a.messages) - 1; i >= 0; i-- {
		if a.messages[i].Role != MessageRoleAssistant {
			continue
		}
		blocks, _ := a.messages[i].Content.([]MessagesBlock)
		var parts []string
		for _, b := range blocks {
			switch b.Type {
			case MessagesBlockTypeToolUse:
				return "", false
			case MessagesBlockTypeText:
				parts = append(parts, b.Text)
			}
		}
		return strings.Join(parts, "\n"), true
	}
	return "", false
}
