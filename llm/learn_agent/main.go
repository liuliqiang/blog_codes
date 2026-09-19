package main

import (
	"context"
	"fmt"
	"os"

	agentloop "github.com/liuliqiang/llmagent/08_context_compact"
	"github.com/liuliqiang/llmagent/08_context_compact/llm/deepseek"
	"github.com/liuliqiang/llmagent/08_context_compact/trace"
)

const traceHTMLPath = "trace.html"

func main() {
	llmOpts := deepseek.NewDeepseekClientOptions(agentloop.ModelDeepseekFlash)
	llmOpts.WithAPIKey(os.Getenv("DS_API_KEY"))
	llmClient := deepseek.NewDeepseekClient(llmOpts)

	rec := trace.NewRecorder()
	hooks := new(agentloop.Hooks).
		OnPreToolUse(agentloop.LogToolUseHook()).
		OnPreToolUse(agentloop.PermissionHook()). // last, so it checks the final input
		OnPostToolUse(agentloop.LargeOutputHook()).
		OnStop(agentloop.SummaryHook()).
		OnCompact(agentloop.CompactLogHook())
	agent := agentloop.NewAgent(llmClient, hooks, rec)

	err := agent.RunLoop(context.Background(), []agentloop.Message{
		{
			Role:    agentloop.MessageRoleUser,
			Content: "/code-review https://github.com/liuliqiang/blog_codes/pull/6.",
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
