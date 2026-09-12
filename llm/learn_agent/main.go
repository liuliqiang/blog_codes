package main

import (
	"context"
	"os"

	agentloop "github.com/liuliqiang/llmagent/01_agent_loop"
	"github.com/liuliqiang/llmagent/01_agent_loop/llm/deepseek"
)

func main() {
	llmOpts := deepseek.NewDeepseekClientOptions(agentloop.ModelDeepseekFlash)
	llmOpts.WithAPIKey(os.Getenv("DS_API_KEY"))
	llmClient := deepseek.NewDeepseekClient(llmOpts)
	agent := agentloop.NewAgent(llmClient)

	agent.RunLoop(context.Background(), []agentloop.Message{
		{
			Role:    agentloop.MessageRoleUser,
			Content: "Write a bash script to print 'Hello, World!' to the console.",
		},
	})
}
