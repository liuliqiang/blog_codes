package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"

	agentloop "github.com/liuliqiang/llmagent/99_tag_iterate_version"
	"github.com/liuliqiang/llmagent/99_tag_iterate_version/llm/deepseek"
	"github.com/liuliqiang/llmagent/99_tag_iterate_version/trace"
	"github.com/liuliqiang/llmagent/99_tag_iterate_version/web"
)

const (
	traceHTMLPath = "trace.html"
	sessionsDir   = ".sessions"
)

func main() {
	prompt := flag.String("p", "", "prompt for an initial agent turn; without it only the cron scheduler runs")
	webAddr := flag.String("web", "", "serve the session web UI on this address (e.g. :8080) instead of running in the terminal")
	flag.Parse()

	llmOpts := deepseek.NewDeepseekClientOptions(agentloop.ModelDeepseekFlash)
	llmOpts.WithAPIKey(os.Getenv("DS_API_KEY"))
	llmClient := deepseek.NewDeepseekClient(llmOpts)

	agentloop.Compaction.Model = agentloop.ModelDeepseekFlash
	agentloop.Memory.Model = agentloop.ModelDeepseekFlash
	agentloop.Subagent.Model = agentloop.ModelDeepseekFlash
	agentloop.Goal.Model = agentloop.ModelDeepseekFlash

	// MCP servers the model may connect to, and what the host lets their tools do. A tool with no policy entry needs
	// the user's confirmation; a server's own "readOnly" hint never grants access.
	agentloop.MCP.Servers = []agentloop.MCPServerConfig{
		{Name: "filesystem", Command: "npx", Args: []string{"-y", "@modelcontextprotocol/server-filesystem", "."}},
	}
	agentloop.MCP.Policy = map[string]string{
		"mcp__filesystem__read_file":      "allow",
		"mcp__filesystem__list_directory": "allow",
		"mcp__filesystem__write_file":     "confirm",
	}

	rec := trace.NewRecorder()
	hooks := new(agentloop.Hooks).
		OnPreToolUse(agentloop.LogToolUseHook()).
		OnPreToolUse(agentloop.PermissionHook()). // last, so it checks the final input
		OnPostToolUse(agentloop.LargeOutputHook()).
		OnStop(agentloop.TeamEventsHook()).      // first: keeps the loop alive until teammates report
		OnStop(agentloop.BackgroundTasksHook()). // then background tasks
		OnStop(agentloop.GoalHook()).            // judged only once teammates and background work have reported
		OnStop(agentloop.SummaryHook()).
		OnStop(agentloop.MemoryHook(llmClient)). // last: only when the session really ends
		OnCompact(agentloop.CompactLogHook())
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if *webAddr != "" {
		if err := serveWeb(ctx, *webAddr, llmClient, hooks); err != nil {
			fmt.Fprintf(os.Stderr, "web: %v\n", err)
			os.Exit(1)
		}
		return
	}

	agent := agentloop.NewAgent(llmClient, hooks, rec)

	sched, err := agentloop.NewScheduler(agent)
	if err != nil {
		fmt.Fprintf(os.Stderr, "scheduler: %v\n", err)
		os.Exit(1)
	}

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

// serveWeb runs every session, with its own agent and cron schedule, behind the web UI until ctx is done.
func serveWeb(ctx context.Context, addr string, llmClient agentloop.LLMClient, hooks *agentloop.Hooks) error {
	sessions, err := agentloop.NewSessionManager(ctx, sessionsDir, llmClient, hooks)
	if err != nil {
		return err
	}
	srv := &http.Server{Addr: addr, Handler: web.NewHandler(sessions)}
	go func() {
		<-ctx.Done()
		srv.Shutdown(context.Background())
	}()
	fmt.Printf("session web UI on http://localhost%s (Ctrl-C to stop)\n", addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
