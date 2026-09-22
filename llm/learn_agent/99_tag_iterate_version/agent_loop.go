package agentloop

import "context"

const defaultSystemPrompt = `You are a senior software engineer.

When a task needs more than a couple of steps, start by calling todo_write to break it down. Mark the step you are working on in_progress, mark steps completed as soon as they are done, and keep the list current as the plan changes.

` + taskSystemPromptGuidance

type Agent interface {
	RunLoop(ctx context.Context, messages []Message) error
}

// NewAgent builds an agent. hooks may be nil; the set is copied so later
// registrations on the caller's Hooks don't affect this agent.
func NewAgent(llmClient LLMClient, hooks *Hooks, recorders ...Recorder) Agent {
	a := &agent{
		name:          "main",
		currLoop:      0,
		maxLoop:       -1,
		skills:        NewSkillLoader(skillsDir),
		memory:        NewMemoryStore(memoryDir),
		tasks:         NewTaskStore(tasksDir),
		llmClient:     llmClient,
		hooks:         hooks.snapshot(),
		recorders:     recorders,
		readFiles:     map[string]bool{},
		modifiedFiles: map[string]bool{},
	}
	if llmClient != nil {
		a.model = llmClient.GetModel()
	}
	a.systemPrompt = withSkillCatalog(defaultSystemPrompt, a.skills.Catalog())
	a.tools = a.generateTools()
	a.toolIndex = indexTools(a.tools)
	return a
}

func indexTools(tools []Tool) map[string]Tool {
	index := make(map[string]Tool, len(tools))
	for _, tool := range tools {
		index[tool.Name] = tool
	}
	return index
}

type agent struct {
	name     string
	currLoop int
	maxLoop  int

	systemPrompt string
	messages     []Message
	llmClient    LLMClient
	model        Model // what the loop itself talks to; compaction and memory may use their own

	tools     []Tool
	toolIndex map[string]Tool

	hooks     Hooks
	recorders []Recorder
	skills    *SkillLoader
	memory    *MemoryStore // nil on subagents: memory belongs to the main conversation
	tasks     *TaskStore   // shared with subagents so they can claim work from the same graph

	// roundsSinceTodo counts consecutive tool rounds without a todo_write call;
	// runTools nags the model once it reaches todoReminderRounds.
	roundsSinceTodo int

	// compaction state, see compact.go
	lastUsage     Usage
	summary       string // current summary; "" until the first compaction
	readFiles     map[string]bool
	modifiedFiles map[string]bool
}

func (a *agent) RunLoop(ctx context.Context, messages []Message) (err error) {
	ctx = withAgentName(ctx, a.name)
	systemPrompt := a.systemPrompt
	if a.memory != nil {
		systemPrompt = withMemory(
			systemPrompt,     // base
			a.memory.Index(), // index
			recallMemories(ctx, a.llmClient, Memory.Model.orModel(a.model), a.memory, messages), // recalled
		)
	}
	for _, r := range a.recorders {
		r.OnStart(a.model, systemPrompt, messages)
	}
	defer func() {
		for _, r := range a.recorders {
			r.OnEnd(err)
		}
		a.resetLoop()
	}()

	messages, err = a.runUserPromptSubmitHooks(ctx, messages)
	if err != nil {
		return err
	}
	a.messages = messages

	for a.shouldContinue() {
		if a.shouldCompact() {
			a.compact(ctx)
		}

		resp, err := a.llmClient.SendMessages(
			ctx,
			a.model,
			Message{
				Role:    MessageRoleSystem,
				Content: systemPrompt,
			},
			a.messages,
			a.tools,
			NewSendMessagesOpts())
		if err != nil {
			return err
		}
		a.lastUsage = resp.Usage

		for _, r := range a.recorders {
			r.OnResponse(a.turn(), resp)
		}

		a.messages = append(a.messages, Message{
			Role:    MessageRoleAssistant,
			Content: resp.GetContent(),
		})

		toolUses := resp.GetToolUsesBlocks()
		if len(toolUses) == 0 {
			followUp, err := a.runStopHooks(ctx, a.messages)
			if err != nil {
				return err
			}
			if followUp == nil {
				return nil
			}
			a.messages = append(a.messages, *followUp)
			a.currLoop++
			continue
		}

		a.runTools(ctx, toolUses)

		a.currLoop++
	}

	return nil
}

func (a *agent) resetLoop() bool {
	a.currLoop = 0
	a.roundsSinceTodo = 0
	a.lastUsage = Usage{}
	a.summary = ""
	a.readFiles = map[string]bool{}
	a.modifiedFiles = map[string]bool{}
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

// agentNameKey carries the running agent's name through ctx so hooks and
// tool handlers can tell the main agent and subagents apart.
type agentNameKey struct{}

func withAgentName(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, agentNameKey{}, name)
}

// agentNameFrom returns the agent name stored in ctx, or "agent" when none is.
func agentNameFrom(ctx context.Context) string {
	if name, ok := ctx.Value(agentNameKey{}).(string); ok {
		return name
	}
	return "agent"
}
