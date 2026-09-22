package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"

	agentloop "github.com/liuliqiang/llmagent/99_tag_iterate_version"
	"github.com/liuliqiang/llmagent/99_tag_iterate_version/llm/deepseek"
	"github.com/liuliqiang/llmagent/99_tag_iterate_version/trace"
)

const traceHTMLPath = "trace.html"

func main() {
	prompt := flag.String("p", "", "prompt for an initial agent turn; without it only the cron scheduler runs")
	flag.Parse()

	llmOpts := deepseek.NewDeepseekClientOptions(agentloop.ModelDeepseekFlash)
	llmOpts.WithAPIKey(os.Getenv("DS_API_KEY"))
	llmClient := deepseek.NewDeepseekClient(llmOpts)

	agentloop.Compaction.Model = agentloop.ModelDeepseekFlash
	agentloop.Memory.Model = agentloop.ModelDeepseekFlash
	agentloop.Subagent.Model = agentloop.ModelDeepseekFlash

	rec := trace.NewRecorder()
	hooks := new(agentloop.Hooks).
		OnPreToolUse(agentloop.LogToolUseHook()).
		OnPreToolUse(agentloop.PermissionHook()). // last, so it checks the final input
		OnPostToolUse(agentloop.LargeOutputHook()).
		OnStop(agentloop.TeamEventsHook()).      // first: keeps the loop alive until teammates report
		OnStop(agentloop.BackgroundTasksHook()). // then background tasks
		OnStop(agentloop.SummaryHook()).
		OnStop(agentloop.MemoryHook(llmClient)). // last: only when the session really ends
		OnCompact(agentloop.CompactLogHook())
	agent := agentloop.NewAgent(llmClient, hooks, rec)

	sched, err := agentloop.NewScheduler(agent)
	if err != nil {
		fmt.Fprintf(os.Stderr, "scheduler: %v\n", err)
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// the scheduler starts first so jobs that come due during the user's turn queue up and run right after it
	schedDone := make(chan error, 1)
	go func() { schedDone <- sched.Run(ctx) }()

	if *prompt != "" {
		err := sched.RunTurn(ctx, []agentloop.Message{{Role: agentloop.MessageRoleUser, Content: *prompt}})
		if err != nil {
			fmt.Fprintf(os.Stderr, "agent failed: %v\n", err)
		}
		if sched.HasJobs() {
			fmt.Println("cron jobs scheduled, scheduler keeps running (Ctrl-C to stop)")
		} else {
			stop() // nothing scheduled: no reason to linger
		}
	} else {
		fmt.Println("no -p prompt given, running the cron scheduler only (Ctrl-C to stop)")
	}

	if err := <-schedDone; err != nil {
		fmt.Fprintf(os.Stderr, "scheduler: %v\n", err)
	}

	if err := rec.WriteHTML(traceHTMLPath); err != nil {
		fmt.Fprintf(os.Stderr, "write trace: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("trace written to %s\n", traceHTMLPath)
}
