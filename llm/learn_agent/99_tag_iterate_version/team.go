package agentloop

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/liuliqiang/log4go"
)

// mailboxDir holds one JSONL inbox per agent, relative to the working directory like tasksDir; tests swap it.
var mailboxDir = ".mailboxes"

// teamOut is where team activity is printed; tests swap it.
var teamOut io.Writer = os.Stdout

// TeamConfig controls the teammates a lead may run.
type TeamConfig struct {
	Model       Model         // "" uses the lead's model
	MaxMates    int           // how many teammates may run at once
	IdleScan    time.Duration // how long an idle teammate waits for a message before checking the task board
	StopTimeout time.Duration // how long Shutdown waits for teammates to stop on their own
}

// Team is the active config.
var Team = TeamConfig{
	MaxMates:    4,
	IdleScan:    2 * time.Second,
	StopTimeout: 30 * time.Second,
}

// Team message types. Ordinary collaboration is free text; shutdown is a typed request/response pair so a reply can
// never be mistaken for anything else.
const (
	teamMessage          = "message"
	teamResult           = "result"
	teamIdleNotification = "idle_notification"
	teamShutdownRequest  = "shutdown_request"
	teamShutdownResponse = "shutdown_response"
)

// leadName is the mailbox the coordinator reads; it is reserved and cannot be used as a teammate name.
const leadName = "lead"

// TeamMessage is one entry in an agent's mailbox.
type TeamMessage struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Type      string `json:"type"`
	Content   string `json:"content"`
	RequestID string `json:"request_id,omitempty"` // correlates a response with its request
	TaskID    string `json:"task_id,omitempty"`
}

/* vvvvvvvvvvvvvvvvvvvvv message bus vvvvvvvvvvvvvvvvvvvvv */

// MessageBus delivers messages between the lead and its teammates through one file per agent. They never share a
// messages array: that would leak one teammate's tool results into another's reasoning.
type MessageBus struct {
	mu      sync.Mutex
	dir     string
	waiters map[string][]chan struct{}
}

func NewMessageBus(dir string) *MessageBus {
	return &MessageBus{dir: dir, waiters: map[string][]chan struct{}{}}
}

func (b *MessageBus) path(agent string) (string, error) {
	if agent == "" || filepath.Base(agent) != agent || strings.ContainsAny(agent, `/\`) {
		return "", fmt.Errorf("invalid agent name %q", agent)
	}
	return filepath.Join(b.dir, agent+".jsonl"), nil
}

// Send appends msg to the recipient's mailbox and wakes anyone waiting on it.
func (b *MessageBus) Send(msg TeamMessage) error {
	ctx := context.Background()
	path, err := b.path(msg.To)
	if err != nil {
		return err
	}
	line, err := json.Marshal(msg)
	if err != nil {
		log4go.DefaultLogger().Error(ctx, "marshal team message failed: %v, message: %+v", err, msg)
		return err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := os.MkdirAll(b.dir, 0o755); err != nil {
		log4go.DefaultLogger().Error(ctx, "create mailbox dir failed: %v, dir: %s", err, b.dir)
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		log4go.DefaultLogger().Error(ctx, "open mailbox failed: %v, path: %s", err, path)
		return err
	}
	_, err = f.Write(append(line, '\n'))
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		log4go.DefaultLogger().Error(ctx, "write mailbox failed: %v, path: %s", err, path)
		return err
	}
	b.notifyLocked(msg.To)
	return nil
}

func (b *MessageBus) notifyLocked(agent string) {
	for _, ch := range b.waiters[agent] {
		close(ch)
	}
	delete(b.waiters, agent)
}

// Peek reports whether the agent has anything waiting.
func (b *MessageBus) Peek(agent string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	path, err := b.path(agent)
	if err != nil {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && info.Size() > 0
}

// ReadInbox consumes the agent's mailbox: everything in it is returned and the file is removed, so each message has
// exactly one reader.
func (b *MessageBus) ReadInbox(agent string) []TeamMessage {
	ctx := context.Background()
	b.mu.Lock()
	defer b.mu.Unlock()
	path, err := b.path(agent)
	if err != nil {
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		if !os.IsNotExist(err) {
			log4go.DefaultLogger().Error(ctx, "open mailbox failed: %v, path: %s", err, path)
		}
		return nil
	}
	var msgs []TeamMessage
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var msg TeamMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			log4go.DefaultLogger().Error(ctx, "skipping unreadable mailbox line: %v, path: %s, line: %s", err, path, line)
			continue
		}
		msgs = append(msgs, msg)
	}
	if err := scanner.Err(); err != nil {
		log4go.DefaultLogger().Error(ctx, "read mailbox failed: %v, path: %s", err, path)
	}
	f.Close()
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		log4go.DefaultLogger().Error(ctx, "clear mailbox failed: %v, path: %s", err, path)
	}
	return msgs
}

// Wait returns the agent's messages as soon as there are any, or nil when timeout passes or ctx is done.
func (b *MessageBus) Wait(ctx context.Context, agent string, timeout time.Duration) []TeamMessage {
	deadline := time.After(timeout)
	for {
		if msgs := b.ReadInbox(agent); len(msgs) > 0 {
			return msgs
		}
		b.mu.Lock()
		ch := make(chan struct{})
		b.waiters[agent] = append(b.waiters[agent], ch)
		b.mu.Unlock()
		// a message may have landed between the read and the registration
		if b.Peek(agent) {
			b.drop(agent, ch)
			continue
		}
		select {
		case <-ch:
		case <-deadline:
			b.drop(agent, ch)
			return nil
		case <-ctx.Done():
			b.drop(agent, ch)
			return nil
		}
	}
}

func (b *MessageBus) drop(agent string, ch chan struct{}) {
	b.mu.Lock()
	defer b.mu.Unlock()
	waiters := b.waiters[agent][:0:0]
	for _, w := range b.waiters[agent] {
		if w != ch {
			waiters = append(waiters, w)
		}
	}
	if len(waiters) == 0 {
		delete(b.waiters, agent)
	} else {
		b.waiters[agent] = waiters
	}
}

/* vvvvvvvvvvvvvvvvvvvvv teammates vvvvvvvvvvvvvvvvvvvvv */

type teammateState string

const (
	teammateWorking teammateState = "working"
	teammateIdle    teammateState = "idle"
	teammateStopped teammateState = "stopped"
)

// Teammate is a persistent agent: unlike a subagent it survives one assignment, keeps its own conversation, and
// alternates between working and idle until it is shut down.
type Teammate struct {
	Name string

	mu     sync.Mutex
	state  teammateState
	agent  *agent
	role   string
	taskID string

	// plan is the approval gate; workVersion changes whenever the teammate picks up or drops work, which
	// invalidates a plan that was written for the old assignment.
	plan        planState
	workVersion int
}

func (m *Teammate) setPlan(p planState) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.plan = p
}

// planNow is the teammate's gate right now.
func (m *Teammate) planNow() planState {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.plan
}

func (m *Teammate) setState(s teammateState) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state = s
}

func (m *Teammate) snapshot() (teammateState, string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.state, m.taskID
}

func (m *Teammate) setTask(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.taskID != id {
		m.workVersion++
		if m.plan == planApproved {
			m.plan = planRequired // the approval was for the previous assignment
		}
	}
	m.taskID = id
}

// shutdownRequest is a shutdown the lead asked for and is still waiting on.
type shutdownRequest struct {
	target    string
	approved  bool
	requested time.Time
}

// TeamRuntime owns the lead's teammates and the bus they talk over.
type TeamRuntime struct {
	mu       sync.Mutex
	lead     *agent
	bus      *MessageBus
	mates    map[string]*Teammate
	order    []string // spawn order, for stable listings
	requests map[string]*shutdownRequest
	plans    map[string]*planRequest
	wg       sync.WaitGroup
	ctx      context.Context
	cancel   context.CancelFunc
}

func newTeamRuntime(lead *agent) *TeamRuntime {
	t := &TeamRuntime{
		lead:     lead,
		bus:      NewMessageBus(mailboxDir),
		mates:    map[string]*Teammate{},
		requests: map[string]*shutdownRequest{},
		plans:    map[string]*planRequest{},
	}
	t.ctx, t.cancel = context.WithCancel(context.Background())
	return t
}

const teammateSystemPrompt = `You are a teammate on a software team, working under a lead agent.

You are given one task at a time. Work on it with the tools available, then call complete_task and report what you did; your final message is sent to the lead as the result of the assignment. Keep the report short and specific. You may message the lead with send_message when you need a decision. Do not wait or poll: when you have nothing to do, finish your turn and the runtime will bring you the next assignment.`

// Active reports how many teammates have not stopped.
func (t *TeamRuntime) Active() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	n := 0
	for _, m := range t.mates {
		if state, _ := m.snapshot(); state != teammateStopped {
			n++
		}
	}
	return n
}

// Working reports how many teammates are running a turn right now.
func (t *TeamRuntime) Working() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	n := 0
	for _, m := range t.mates {
		if state, _ := m.snapshot(); state == teammateWorking {
			n++
		}
	}
	return n
}

// Spawn starts a teammate. taskID, when given, is claimed before the teammate starts: a teammate that cannot get its
// first task never runs.
func (t *TeamRuntime) Spawn(name, role, taskID string, requirePlan bool) (*Teammate, error) {
	name = strings.TrimSpace(name)
	if name == "" || name == leadName || name == "agent" || name == "main" || name == "subagent" {
		return nil, fmt.Errorf("invalid teammate name %q", name)
	}
	if _, err := t.bus.path(name); err != nil {
		return nil, err
	}
	t.mu.Lock()
	if _, exists := t.mates[name]; exists {
		t.mu.Unlock()
		return nil, fmt.Errorf("teammate %s already exists", name)
	}
	if Team.MaxMates > 0 && len(t.mates) >= Team.MaxMates {
		t.mu.Unlock()
		return nil, fmt.Errorf("the team already has %d teammates", len(t.mates))
	}
	t.mu.Unlock()

	var first []Message
	if taskID != "" {
		task, err := t.lead.tasks.Claim(taskID, name)
		if err != nil {
			return nil, err
		}
		first = []Message{{Role: MessageRoleUser, Content: assignmentPrompt(task)}}
	}

	gate := planNotRequired
	if requirePlan {
		gate = planRequired
	}
	mate := &Teammate{Name: name, state: teammateIdle, role: role, taskID: taskID, plan: gate}
	mate.agent = t.newTeammateAgent(name, role)
	t.mu.Lock()
	t.mates[name] = mate
	t.order = append(t.order, name)
	t.mu.Unlock()

	t.wg.Add(1)
	go t.run(mate, first)
	fmt.Fprintf(teamOut, "\033[32m[team] spawned %s (%s)\033[0m\n", name, role)
	return mate, nil
}

func assignmentPrompt(task Task) string {
	prompt := fmt.Sprintf("[Task %s] %s", task.ID, task.Subject)
	if task.Description != "" {
		prompt += "\n" + task.Description
	}
	return prompt
}

// teammateExcludedTools are lead-only or would collide across agents: only the lead changes the task graph, schedules
// work or spawns more agents, and todo_write is a single shared list.
var teammateExcludedTools = map[string]bool{
	"update_task":       true,
	"todo_write":        true,
	"task":              true,
	"schedule_cron":     true,
	"list_crons":        true,
	"cancel_cron":       true,
	"spawn_teammate":    true,
	"list_teammates":    true,
	"shutdown_teammate": true,
	"create_worktree":   true,
	"request_plan":      true,
	"review_plan":       true,
}

// newTeammateAgent builds the agent behind a teammate: the lead's client, hooks, skills and task board, but its own
// conversation, tools and background manager.
func (t *TeamRuntime) newTeammateAgent(name, role string) *agent {
	lead := t.lead
	prompt := teammateSystemPrompt
	if role != "" {
		prompt += "\n\nYour area of responsibility: " + role
	}
	a := &agent{
		name:           name,
		model:          Team.Model.orModel(lead.model),
		maxLoop:        -1,
		systemPrompt:   withSkillCatalog(prompt, lead.skills.Catalog()),
		skills:         lead.skills,
		tasks:          lead.tasks,
		team:           t,
		background:     NewBackgroundManager(),
		llmClient:      lead.llmClient,
		hooks:          lead.hooks.snapshot(),
		persistHistory: true,
		readFiles:      map[string]bool{},
		modifiedFiles:  map[string]bool{},
	}
	for _, tool := range lead.tools {
		if teammateExcludedTools[tool.Name] {
			continue
		}
		if workspaceTools[tool.Name] {
			tool = a.inAssignment(tool)
		}
		a.tools = append(a.tools, tool)
	}
	a.tools = append(a.tools, a.sendMessageTool(), a.submitPlanTool())
	a.toolIndex = indexTools(a.tools)
	return a
}

// run is one teammate's life: work on what it was given, report, then idle — checking its mailbox first and the task
// board second — until it is shut down.
func (t *TeamRuntime) run(mate *Teammate, pending []Message) {
	defer t.wg.Done()
	defer func() {
		mate.setState(teammateStopped)
		mate.agent.background.Shutdown()
	}()

	for {
		if len(pending) > 0 {
			t.work(mate, pending)
			pending = nil
		}
		mate.setState(teammateIdle)

		if msgs := t.bus.Wait(t.ctx, mate.Name, Team.IdleScan); len(msgs) > 0 {
			stop, next := t.handleTeammateInbox(mate, msgs)
			if stop {
				return
			}
			pending = next
			continue
		}
		if t.ctx.Err() != nil {
			return
		}
		// nothing waiting: look for ready work on the shared board
		if task, ok, err := mate.agent.tasks.ClaimNext(mate.Name); err != nil {
			log4go.DefaultLogger().Error(t.ctx, "[%s] claiming next task failed: %v", mate.Name, err)
		} else if ok {
			mate.setTask(task.ID)
			fmt.Fprintf(teamOut, "\033[32m[team] %s auto-claimed %s: %s\033[0m\n", mate.Name, task.ID, task.Subject)
			pending = []Message{{Role: MessageRoleUser, Content: "[Auto-claimed] " + assignmentPrompt(task)}}
		}
	}
}

// work runs one assignment and reports it to the lead as two events: what the assignment produced, and that the
// teammate can take more work. One vague "done" cannot carry both facts.
func (t *TeamRuntime) work(mate *Teammate, pending []Message) {
	mate.setState(teammateWorking)
	result := ""
	if err := mate.agent.RunLoop(t.ctx, pending); err != nil {
		log4go.DefaultLogger().Error(t.ctx, "[%s] assignment failed: %v", mate.Name, err)
		result = "Assignment failed: " + err.Error()
	} else if text, ok := mate.agent.finalText(); ok {
		result = text
	} else {
		result = "(no final answer)"
	}

	// a task the teammate did not complete goes back on the board: leaving it in progress would block that teammate
	// from ever claiming anything else
	_, taskID := mate.snapshot()
	if taskID != "" {
		if task, err := mate.agent.tasks.Load(taskID); err == nil && task.Status == TaskInProgress && task.Owner == mate.Name {
			if err := mate.agent.tasks.Release(taskID, mate.Name); err != nil {
				log4go.DefaultLogger().Error(t.ctx, "[%s] releasing %s failed: %v", mate.Name, taskID, err)
			} else {
				result += fmt.Sprintf("\n(task %s was not completed and is back on the board)", taskID)
				fmt.Fprintf(teamOut, "\033[32m[team] %s released %s\033[0m\n", mate.Name, taskID)
			}
		}
	}
	t.send(TeamMessage{From: mate.Name, To: leadName, Type: teamResult, Content: result, TaskID: taskID})
	mate.setTask("")
	t.send(TeamMessage{From: mate.Name, To: leadName, Type: teamIdleNotification, Content: "Waiting for more work."})
}

// handleTeammateInbox turns inbox messages into the next assignment, and answers a shutdown request. It reports
// whether the teammate should stop.
func (t *TeamRuntime) handleTeammateInbox(mate *Teammate, msgs []TeamMessage) (bool, []Message) {
	var next []Message
	for _, msg := range msgs {
		switch msg.Type {
		case teamShutdownRequest:
			t.send(TeamMessage{From: mate.Name, To: leadName, Type: teamShutdownResponse, RequestID: msg.RequestID, Content: "Stopping."})
			fmt.Fprintf(teamOut, "\033[32m[team] %s stopping\033[0m\n", mate.Name)
			return true, nil
		case teamPlanRequest:
			mate.setPlan(planRequired)
			next = append(next, Message{Role: MessageRoleUser, Content: "[Plan required] " + msg.Content})
		case teamPlanApprovalResponse:
			next = append(next, Message{Role: MessageRoleUser, Content: fmt.Sprintf("[Plan %s] %s", mate.planNow(), msg.Content)})
		default:
			next = append(next, Message{Role: MessageRoleUser, Content: fmt.Sprintf("[Message from %s] %s", msg.From, msg.Content)})
		}
	}
	return false, next
}

func (t *TeamRuntime) send(msg TeamMessage) {
	if err := t.bus.Send(msg); err != nil {
		log4go.DefaultLogger().Error(context.Background(), "send team message failed: %v, message: %+v", err, msg)
	}
}

// RequestShutdown asks a teammate to stop once it finishes what it is doing, and returns the request id the reply
// will carry.
func (t *TeamRuntime) RequestShutdown(name string) (string, error) {
	t.mu.Lock()
	mate, ok := t.mates[name]
	t.mu.Unlock()
	if !ok {
		return "", fmt.Errorf("teammate %s not found", name)
	}
	if state, _ := mate.snapshot(); state == teammateStopped {
		return "", fmt.Errorf("teammate %s already stopped", name)
	}
	var raw [4]byte
	if _, err := rand.Read(raw[:]); err != nil {
		log4go.DefaultLogger().Error(context.Background(), "generate request ID failed: %v", err)
		return "", err
	}
	id := "req_" + hex.EncodeToString(raw[:])
	t.mu.Lock()
	t.requests[id] = &shutdownRequest{target: name, requested: time.Now()}
	t.mu.Unlock()
	if err := t.bus.Send(TeamMessage{From: leadName, To: name, Type: teamShutdownRequest, RequestID: id, Content: "Please finish up and stop."}); err != nil {
		t.mu.Lock()
		delete(t.requests, id)
		t.mu.Unlock()
		return "", err
	}
	return id, nil
}

// matchShutdownResponse applies a reply to the request it names. A reply with an unknown id, the wrong type or a
// request that was already answered changes nothing.
func (t *TeamRuntime) matchShutdownResponse(msg TeamMessage) {
	if msg.Type != teamShutdownResponse {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	req, ok := t.requests[msg.RequestID]
	if !ok || req.approved || req.target != msg.From {
		log4go.DefaultLogger().Error(context.Background(), "ignoring shutdown response: %+v", msg)
		return
	}
	req.approved = true
}

// Shutdown asks every teammate to stop, waits for them, then cancels whatever is left.
func (t *TeamRuntime) Shutdown() {
	t.mu.Lock()
	names := append([]string(nil), t.order...)
	t.mu.Unlock()
	for _, name := range names {
		if _, err := t.RequestShutdown(name); err != nil {
			continue
		}
	}
	done := make(chan struct{})
	go func() {
		t.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(Team.StopTimeout):
		log4go.DefaultLogger().Error(context.Background(), "teammates did not stop within %s, cancelling", Team.StopTimeout)
	}
	t.cancel()
	t.wg.Wait()
}

/* vvvvvvvvvvvvvvvvvvvvv lead side vvvvvvvvvvvvvvvvvvvvv */

// consumeTeamEvents drains the lead's mailbox and renders the events for the model.
func (t *TeamRuntime) consumeTeamEvents() []string {
	msgs := t.bus.ReadInbox(leadName)
	var notes []string
	for _, msg := range msgs {
		t.matchShutdownResponse(msg)
		if msg.Type == teamShutdownResponse {
			fmt.Fprintf(teamOut, "\033[32m[team] %s acknowledged shutdown\033[0m\n", msg.From)
		}
		note := fmt.Sprintf("<team_event from=%q type=%q", msg.From, msg.Type)
		if msg.TaskID != "" {
			note += fmt.Sprintf(" task=%q", msg.TaskID)
		}
		notes = append(notes, note+">\n"+msg.Content+"\n</team_event>")
		fmt.Fprintf(teamOut, "\033[32m[team] %s -> lead (%s)\033[0m\n", msg.From, msg.Type)
	}
	return notes
}

// injectTeamEvents appends whatever the teammates reported to the conversation before the next model call.
func (a *agent) injectTeamEvents() {
	if a.team == nil || a.name != "main" {
		return
	}
	notes := a.team.consumeTeamEvents()
	if len(notes) == 0 {
		return
	}
	var blocks []MessagesBlock
	for _, note := range notes {
		blocks = append(blocks, MessagesBlock{Type: MessagesBlockTypeText, Text: note})
	}
	a.messages = appendUserBlocks(a.messages, blocks)
}

// appendUserBlocks adds blocks to the trailing user message, or starts a new one when the conversation does not end
// with one, so a tool_use never gains a second tool_result.
func appendUserBlocks(messages []Message, blocks []MessagesBlock) []Message {
	if n := len(messages); n > 0 && messages[n-1].Role == MessageRoleUser {
		last := &messages[n-1]
		switch content := last.Content.(type) {
		case []MessagesBlock:
			last.Content = append(content, blocks...)
			return messages
		case string:
			last.Content = append([]MessagesBlock{{Type: MessagesBlockTypeText, Text: content}}, blocks...)
			return messages
		}
	}
	return append(messages, Message{Role: MessageRoleUser, Content: blocks})
}

// teamWaitHint is what the lead sees when it tries to stop while teammates are still working.
const teamWaitHint = "<reminder>%d teammate(s) are still working. Wait for their <team_event> reports before finishing, or send them instructions meanwhile.</reminder>"

// TeamEventsHook keeps the lead's loop alive while teammates are working: the lead would otherwise stop as soon as it
// has nothing to say and their reports would never be read.
func TeamEventsHook() StopHook {
	return func(ctx context.Context, _ []Message) (*Message, error) {
		a := agentFrom(ctx)
		if a == nil || a.team == nil || a.name != "main" {
			return nil, nil
		}
		if notes := a.team.consumeTeamEvents(); len(notes) > 0 {
			var blocks []MessagesBlock
			for _, note := range notes {
				blocks = append(blocks, MessagesBlock{Type: MessagesBlockTypeText, Text: note})
			}
			return &Message{Role: MessageRoleUser, Content: blocks}, nil
		}
		working := a.team.Working()
		if working == 0 {
			return nil, nil
		}
		a.team.bus.Wait(ctx, leadName, Team.IdleScan)
		return &Message{Role: MessageRoleUser, Content: fmt.Sprintf(teamWaitHint, working)}, nil
	}
}

/* vvvvvvvvvvvvvvvvvvvvv tools vvvvvvvvvvvvvvvvvvvvv */

const teamSystemPromptGuidance = `When a job is large enough that working on its parts in parallel would help, first propose a small team — one teammate per area, with the tasks each would own — and wait for the user to confirm before calling spawn_teammate. Create the tasks first, then spawn a teammate per area with its first task_id. After spawning, end your turn: teammate results arrive on their own as <team_event> and start your next turn. Idle teammates pick up unblocked tasks from the board themselves.`

func (a *agent) runSpawnTeammate(ctx context.Context, input map[string]interface{}) (string, error) {
	name, err := stringArg(input, "name")
	if err != nil {
		return "", err
	}
	role, err := optionalStringArg(input, "role", "")
	if err != nil {
		return "", err
	}
	taskID, err := optionalStringArg(input, "task_id", "")
	if err != nil {
		return "", err
	}
	requirePlan, err := optionalBoolArg(input, "require_plan", false)
	if err != nil {
		return "", err
	}
	mate, err := a.team.Spawn(name, role, taskID, requirePlan)
	if err != nil {
		log4go.DefaultLogger().Error(ctx, "[%s] spawn_teammate failed: %v, input: %+v", agentNameFrom(ctx), err, input)
		return "", err
	}
	msg := fmt.Sprintf("Spawned teammate %s", mate.Name)
	if taskID != "" {
		msg += " on task " + taskID
	}
	if requirePlan {
		msg += "; it must get a plan approved before it can change anything"
	}
	return msg + ". Its result will arrive as a <team_event>; end your turn instead of waiting.", nil
}

func (a *agent) runListTeammates(_ context.Context, _ map[string]interface{}) (string, error) {
	a.team.mu.Lock()
	names := append([]string(nil), a.team.order...)
	mates := make([]*Teammate, 0, len(names))
	for _, name := range names {
		mates = append(mates, a.team.mates[name])
	}
	a.team.mu.Unlock()
	if len(mates) == 0 {
		return "No teammates. Use spawn_teammate after the user confirms the plan.", nil
	}
	var lines []string
	for _, mate := range mates {
		state, taskID := mate.snapshot()
		line := fmt.Sprintf("%s [%s]", mate.Name, state)
		if gate := mate.planNow(); gate != planNotRequired {
			line += " [plan " + string(gate) + "]"
		}
		if mate.role != "" {
			line += " " + mate.role
		}
		if taskID != "" {
			line += " (task " + taskID + ")"
		}
		lines = append(lines, line)
	}
	sort.Strings(lines)
	return strings.Join(lines, "\n"), nil
}

func (a *agent) runSendMessage(ctx context.Context, input map[string]interface{}) (string, error) {
	content, err := stringArg(input, "content")
	if err != nil {
		return "", err
	}
	to := leadName
	if a.name == "main" {
		if to, err = stringArg(input, "to"); err != nil {
			return "", err
		}
		a.team.mu.Lock()
		_, ok := a.team.mates[to]
		a.team.mu.Unlock()
		if !ok {
			return "", fmt.Errorf("teammate %s not found", to)
		}
	}
	msg := TeamMessage{From: a.name, To: to, Type: teamMessage, Content: content}
	if err := a.team.bus.Send(msg); err != nil {
		log4go.DefaultLogger().Error(ctx, "[%s] send_message failed: %v, to: %s", agentNameFrom(ctx), err, to)
		return "", err
	}
	return "Message sent to " + to + ".", nil
}

func (a *agent) runShutdownTeammate(ctx context.Context, input map[string]interface{}) (string, error) {
	name, err := stringArg(input, "name")
	if err != nil {
		return "", err
	}
	id, err := a.team.RequestShutdown(name)
	if err != nil {
		log4go.DefaultLogger().Error(ctx, "[%s] shutdown_teammate failed: %v, name: %s", agentNameFrom(ctx), err, name)
		return "", err
	}
	return fmt.Sprintf("Asked %s to stop (request %s); it replies when it has finished its current step.", name, id), nil
}

// sendMessageTool is the teammate's version: it can only write to the lead.
func (a *agent) sendMessageTool() Tool {
	return Tool{
		Name:        "send_message",
		Handler:     a.runSendMessage,
		Description: "Send a message to the lead agent.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"content": map[string]interface{}{"type": "string"},
			},
			"required": []string{"content"},
		},
	}
}
