package main

import (
	"context"
	"fmt"
	"os"

	agentloop "github.com/liuliqiang/llmagent/03_permission"
	"github.com/liuliqiang/llmagent/03_permission/llm/deepseek"
	"github.com/liuliqiang/llmagent/03_permission/trace"
)

const traceHTMLPath = "trace.html"

func main() {
	llmOpts := deepseek.NewDeepseekClientOptions(agentloop.ModelDeepseekFlash)
	llmOpts.WithAPIKey(os.Getenv("DS_API_KEY"))
	llmClient := deepseek.NewDeepseekClient(llmOpts)

	rec := trace.NewRecorder()
	agent := agentloop.NewAgent(llmClient, rec)

	err := agent.RunLoop(context.Background(), []agentloop.Message{
		{
			Role:    agentloop.MessageRoleUser,
			Content: "Write a bash script to print 'Hello, World!' to the console.",
		},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "agent failed: %v\n", err)
	}

	if err := rec.WriteHTML(traceHTMLPath); err != nil {
		fmt.Fprintf(os.Stderr, "write trace: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("trace written to %s\n", traceHTMLPath)
}
