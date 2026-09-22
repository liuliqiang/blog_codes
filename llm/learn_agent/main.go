package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	agentloop "github.com/liuliqiang/llmagent/99_tag_iterate_version"
	"github.com/liuliqiang/llmagent/99_tag_iterate_version/llm/deepseek"
	"github.com/liuliqiang/llmagent/99_tag_iterate_version/trace"
)

const traceHTMLPath = "trace.html"

func main() {
	llmOpts := deepseek.NewDeepseekClientOptions(agentloop.ModelDeepseekFlash)
	llmOpts.WithAPIKey(os.Getenv("DS_API_KEY"))
	llmClient := deepseek.NewDeepseekClient(llmOpts)

	// side tasks don't need the main model; leave any of these "" to fall back to it
	agentloop.Compaction.Model = agentloop.ModelDeepseekFlash
	agentloop.Memory.Model = agentloop.ModelDeepseekFlash
	agentloop.Subagent.Model = agentloop.ModelDeepseekFlash

	rec := trace.NewRecorder()
	hooks := new(agentloop.Hooks).
		OnPreToolUse(agentloop.LogToolUseHook()).
		OnPreToolUse(agentloop.PermissionHook()). // last, so it checks the final input
		OnPostToolUse(agentloop.LargeOutputHook()).
		OnStop(agentloop.BackgroundTasksHook()). // before the rest: keeps the loop alive until background tasks report
		OnStop(agentloop.SummaryHook()).
		OnStop(agentloop.MemoryHook(llmClient)). // last: only when the session really ends
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

	// keep running while there are cron jobs; Ctrl-C stops the scheduler
	if sched, err := agentloop.NewScheduler(agent); err == nil && sched.HasJobs() {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
		fmt.Println("cron jobs scheduled, scheduler running (Ctrl-C to stop)")
		if err := sched.Run(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "scheduler: %v\n", err)
		}
		stop()
	}

	if err := rec.WriteHTML(traceHTMLPath); err != nil {
		fmt.Fprintf(os.Stderr, "write trace: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("trace written to %s\n", traceHTMLPath)
}
