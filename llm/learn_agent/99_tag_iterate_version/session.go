package agentloop

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/liuliqiang/log4go"
)

// A session is one ongoing conversation with its own main agent: a history that carries over between turns, its own
// cron schedule, whose jobs fire back into the same session, and its own teammates. Memory, the task board and skills
// stay shared by the whole project. Every session is saved under the manager's root, one directory each, after every
// turn, so the list and each conversation survive a restart.

const (
	defaultSessionTitle = "新会话"
	sessionTitleRunes   = 40
	sessionOutputBytes  = 4000 // how much of a tool output a turn keeps for display
	sessionFileName     = "session.json"
)

// ErrSessionBusy is returned when a turn is sent to a session that is still working on one.
var ErrSessionBusy = errors.New("session is busy")

// ErrSessionNotFound is returned for an unknown session id.
var ErrSessionNotFound = errors.New("session not found")

// SessionTurn is what the user sees of one turn: the prompt, the steps the loop took and the final reply.
type SessionTurn struct {
	ID        int           `json:"id"`
	Source    string        `json:"source"` // "user", or "cron" for a scheduled turn
	Prompt    string        `json:"prompt"`
	Steps     []SessionStep `json:"steps"`
	Reply     string        `json:"reply"`
	Error     string        `json:"error,omitempty"`
	Status    string        `json:"status"` // running, done, failed, or interrupted by a restart
	StartedAt time.Time     `json:"started_at"`
	EndedAt   time.Time     `json:"ended_at"`
}

// SessionStep is one step inside a turn: something the model said on the way, or one tool call.
type SessionStep struct {
	Kind      string `json:"kind"` // "model" or "tool"
	Text      string `json:"text,omitempty"`
	Reasoning string `json:"reasoning,omitempty"`
	ToolUseID string `json:"tool_use_id,omitempty"`
	Tool      string `json:"tool,omitempty"`
	Input     string `json:"input,omitempty"`
	Output    string `json:"output,omitempty"`
	Status    string `json:"status,omitempty"` // for a tool: running or done
	TookMS    int64  `json:"took_ms,omitempty"`
}

// PermissionRequest is a tool call waiting for the user's decision.
type PermissionRequest struct {
	ID    string `json:"id"`
	Agent string `json:"agent"`
	Tool  string `json:"tool"`
	Input string `json:"input"`
}

// SessionView is a session as the UI sees it. Turns are only filled in for a single session, not in the list.
type SessionView struct {
	ID          string              `json:"id"`
	Title       string              `json:"title"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
	Busy        bool                `json:"busy"`
	Turns       []SessionTurn       `json:"turns,omitempty"`
	Permissions []PermissionRequest `json:"permissions,omitempty"`
}

// SessionEvent is pushed to subscribers whenever the session changes. A "turn" event carries the whole turn, so a
// client that missed one only has to apply the next.
type SessionEvent struct {
	Type         string             `json:"type"` // snapshot, turn, permission, permission_resolved, session
	Session      *SessionView       `json:"session,omitempty"`
	Turn         *SessionTurn       `json:"turn,omitempty"`
	Permission   *PermissionRequest `json:"permission,omitempty"`
	PermissionID string             `json:"permission_id,omitempty"`
}

/* vvvvvvvvvvvvvvvvvvvvv manager vvvvvvvvvvvvvvvvvvvvv */

// SessionManager owns every session under root.
type SessionManager struct {
	ctx   context.Context
	root  string
	llm   LLMClient
	hooks *Hooks

	mu       sync.Mutex
	sessions map[string]*Session
}

// NewSessionManager loads the sessions saved under root and starts their schedulers. They run until ctx is done.
func NewSessionManager(ctx context.Context, root string, llm LLMClient, hooks *Hooks) (*SessionManager, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		log4go.DefaultLogger().Error(ctx, "create session root failed: %v, root: %s", err, root)
		return nil, err
	}
	m := &SessionManager{ctx: ctx, root: root, llm: llm, hooks: hooks, sessions: map[string]*Session{}}
	entries, err := os.ReadDir(root)
	if err != nil {
		log4go.DefaultLogger().Error(ctx, "read session root failed: %v, root: %s", err, root)
		return nil, err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		s, err := m.load(entry.Name())
		if err != nil {
			// one broken session must not take the others down with it
			log4go.DefaultLogger().Error(ctx, "load session failed: %v, id: %s", err, entry.Name())
			continue
		}
		m.sessions[s.id] = s
	}
	return m, nil
}

// Create starts an empty session.
func (m *SessionManager) Create() (*Session, error) {
	id, err := newSessionID()
	if err != nil {
		log4go.DefaultLogger().Error(m.ctx, "generate session id failed: %v", err)
		return nil, err
	}
	now := time.Now()
	s := m.newSession(id, sessionFile{ID: id, Title: defaultSessionTitle, CreatedAt: now, UpdatedAt: now})
	if err := s.save(); err != nil {
		s.stop()
		return nil, err
	}
	m.mu.Lock()
	m.sessions[id] = s
	m.mu.Unlock()
	s.start()
	return s, nil
}

// Get returns the session with id, or ErrSessionNotFound.
func (m *SessionManager) Get(id string) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[id]
	if !ok {
		return nil, ErrSessionNotFound
	}
	return s, nil
}

// List returns every session, most recently active first.
func (m *SessionManager) List() []SessionView {
	m.mu.Lock()
	sessions := make([]*Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		sessions = append(sessions, s)
	}
	m.mu.Unlock()
	views := make([]SessionView, 0, len(sessions))
	for _, s := range sessions {
		views = append(views, s.View(false))
	}
	sort.Slice(views, func(i, j int) bool { return views[i].UpdatedAt.After(views[j].UpdatedAt) })
	return views
}

// Delete stops the session, cancelling whatever it is doing, and removes it from disk.
func (m *SessionManager) Delete(id string) error {
	m.mu.Lock()
	s, ok := m.sessions[id]
	delete(m.sessions, id)
	m.mu.Unlock()
	if !ok {
		return ErrSessionNotFound
	}
	s.stop()
	if err := os.RemoveAll(s.dir); err != nil {
		log4go.DefaultLogger().Error(m.ctx, "remove session dir failed: %v, dir: %s", err, s.dir)
		return err
	}
	return nil
}

func (m *SessionManager) load(id string) (*Session, error) {
	content, err := os.ReadFile(filepath.Join(m.root, id, sessionFileName))
	if err != nil {
		log4go.DefaultLogger().Error(m.ctx, "read session file failed: %v, id: %s", err, id)
		return nil, err
	}
	var saved sessionFile
	if err := json.Unmarshal(content, &saved); err != nil {
		log4go.DefaultLogger().Error(m.ctx, "decode session file failed: %v, id: %s", err, id)
		return nil, err
	}
	if saved.ID != id {
		return nil, fmt.Errorf("session file id %q does not match its directory", saved.ID)
	}
	s := m.newSession(id, saved)
	s.start()
	return s, nil
}

func (m *SessionManager) newSession(id string, saved sessionFile) *Session {
	dir := filepath.Join(m.root, id)
	ctx, cancel := context.WithCancel(m.ctx)
	s := &Session{
		id:          id,
		dir:         dir,
		ctx:         ctx,
		cancel:      cancel,
		title:       saved.Title,
		createdAt:   saved.CreatedAt,
		updatedAt:   saved.UpdatedAt,
		turns:       saved.Turns,
		permissions: map[string]*pendingPermission{},
		subscribers: map[chan SessionEvent]struct{}{},
	}
	for _, turn := range s.turns {
		if turn.Status == "running" {
			turn.Status = "interrupted"
		}
	}
	a := newAgent(m.llm, m.hooks, filepath.Join(dir, "cron.json"), filepath.Join(dir, "mailboxes"), &sessionRecorder{s: s})
	a.persistHistory = true
	a.messages = decodeMessages(saved.Messages)
	a.summary = saved.Summary
	// teammates run on the team's own context: route their confirmations to the page too
	a.team.ctx = WithApprover(a.team.ctx, s.approve)
	s.agent = a
	s.sched = &Scheduler{agent: a, store: a.cron}
	return s
}

func newSessionID() (string, error) {
	suffix := make([]byte, 3)
	if _, err := rand.Read(suffix); err != nil {
		return "", err
	}
	return time.Now().Format("20060102-150405") + "-" + hex.EncodeToString(suffix), nil
}

/* ^^^^^^^^^^^^^^^^^^^^^ manager ^^^^^^^^^^^^^^^^^^^^^ */

/* vvvvvvvvvvvvvvvvvvvvv session vvvvvvvvvvvvvvvvvvvvv */

// Session is one conversation. Its turns, user or scheduled, run one at a time through its scheduler.
type Session struct {
	id     string
	dir    string
	agent  *agent
	sched  *Scheduler
	ctx    context.Context
	cancel context.CancelFunc

	mu          sync.Mutex
	title       string
	createdAt   time.Time
	updatedAt   time.Time
	turns       []*SessionTurn
	current     *SessionTurn // the turn the agent is running, user or scheduled
	finalStep   int          // index in current.Steps of the latest tool-free reply, -1 when there is none
	userPrompt  string       // set once a user turn holds the agent, until it starts
	queued      bool         // a user turn was accepted and hasn't returned yet
	deleted     bool
	nextPermID  int
	permissions map[string]*pendingPermission
	subscribers map[chan SessionEvent]struct{}
}

type pendingPermission struct {
	request PermissionRequest
	answer  chan bool
}

// ID is the session's id.
func (s *Session) ID() string { return s.id }

func (s *Session) start() {
	go func() {
		if err := s.sched.Run(s.ctx); err != nil {
			log4go.DefaultLogger().Error(s.ctx, "[session %s] scheduler stopped: %v", s.id, err)
		}
	}()
}

func (s *Session) stop() {
	s.mu.Lock()
	s.deleted = true
	s.mu.Unlock()
	s.cancel()
	go s.agent.team.Shutdown()
}

// View returns the session's current state; withTurns includes the conversation and pending permissions.
func (s *Session) View(withTurns bool) SessionView {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.viewLocked(withTurns)
}

func (s *Session) viewLocked(withTurns bool) SessionView {
	view := SessionView{
		ID:        s.id,
		Title:     s.title,
		CreatedAt: s.createdAt,
		UpdatedAt: s.updatedAt,
		Busy:      s.queued || s.current != nil,
	}
	if !withTurns {
		return view
	}
	view.Turns = make([]SessionTurn, 0, len(s.turns))
	for _, turn := range s.turns {
		view.Turns = append(view.Turns, turn.clone())
	}
	ids := make([]string, 0, len(s.permissions))
	for id := range s.permissions {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		view.Permissions = append(view.Permissions, s.permissions[id].request)
	}
	return view
}

// Submit starts a user turn in the background and returns at once; its progress arrives as events. A session works
// on one turn at a time, so a prompt sent while it is busy is refused with ErrSessionBusy.
func (s *Session) Submit(prompt string) error {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return errors.New("prompt is empty")
	}
	s.mu.Lock()
	if s.queued || s.current != nil {
		s.mu.Unlock()
		return ErrSessionBusy
	}
	s.queued = true
	if s.title == defaultSessionTitle && len(s.turns) == 0 {
		s.title = truncateRunes(prompt, sessionTitleRunes)
	}
	s.updatedAt = time.Now()
	s.broadcastLocked(SessionEvent{Type: "session", Session: ptr(s.viewLocked(false))})
	s.mu.Unlock()

	go func() {
		ctx := WithApprover(s.ctx, s.approve)
		err := s.sched.runTurn(ctx, []Message{{Role: MessageRoleUser, Content: prompt}}, func() {
			s.mu.Lock()
			s.userPrompt = prompt
			s.mu.Unlock()
		})
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.userPrompt != "" {
			// refused before the loop started, e.g. an invalid /goal: still show the user why
			log4go.DefaultLogger().Error(s.ctx, "[session %s] turn rejected: %v, prompt: %s", s.id, err, prompt)
			now := time.Now()
			turn := &SessionTurn{ID: len(s.turns) + 1, Source: "user", Prompt: prompt, Status: "failed", StartedAt: now, EndedAt: now}
			if err != nil {
				turn.Error = err.Error()
			}
			s.turns = append(s.turns, turn)
			s.userPrompt = ""
			s.broadcastLocked(SessionEvent{Type: "turn", Turn: ptr(turn.clone())})
		}
		s.queued = false
		s.broadcastLocked(SessionEvent{Type: "session", Session: ptr(s.viewLocked(false))})
	}()
	return nil
}

// Subscribe returns the session's current state and a channel of the changes after it. A subscriber that falls too
// far behind has its channel closed and should subscribe again. Call cancel when done.
func (s *Session) Subscribe() (SessionView, <-chan SessionEvent, func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ch := make(chan SessionEvent, 64)
	s.subscribers[ch] = struct{}{}
	cancel := func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if _, ok := s.subscribers[ch]; ok {
			delete(s.subscribers, ch)
			close(ch)
		}
	}
	return s.viewLocked(true), ch, cancel
}

func (s *Session) broadcastLocked(event SessionEvent) {
	for ch := range s.subscribers {
		select {
		case ch <- event:
		default:
			// too slow: drop it rather than block the agent; it reconnects and gets a fresh snapshot
			delete(s.subscribers, ch)
			close(ch)
		}
	}
}

// ResolvePermission answers a pending permission request.
func (s *Session) ResolvePermission(id string, allow bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	pending, ok := s.permissions[id]
	if !ok {
		return fmt.Errorf("no pending permission request %q", id)
	}
	pending.answer <- allow // buffered: never blocks
	return nil
}

// approve is the session's Approver: it shows the request to subscribers and waits for ResolvePermission.
func (s *Session) approve(ctx context.Context, tool MessagesBlock) bool {
	input, err := json.MarshalIndent(tool.Input, "", "  ")
	if err != nil {
		log4go.DefaultLogger().Error(ctx, "marshal tool input failed: %v, input: %+v", err, tool.Input)
		input = []byte(fmt.Sprintf("%+v", tool.Input))
	}
	s.mu.Lock()
	s.nextPermID++
	pending := &pendingPermission{
		request: PermissionRequest{ID: fmt.Sprintf("p%d", s.nextPermID), Agent: agentNameFrom(ctx), Tool: tool.Name, Input: string(input)},
		answer:  make(chan bool, 1),
	}
	s.permissions[pending.request.ID] = pending
	s.broadcastLocked(SessionEvent{Type: "permission", Permission: ptr(pending.request)})
	s.mu.Unlock()

	allow := false
	select {
	case allow = <-pending.answer:
	case <-ctx.Done():
	}

	s.mu.Lock()
	delete(s.permissions, pending.request.ID)
	s.broadcastLocked(SessionEvent{Type: "permission_resolved", PermissionID: pending.request.ID})
	s.mu.Unlock()
	return allow
}

/* ^^^^^^^^^^^^^^^^^^^^^ session ^^^^^^^^^^^^^^^^^^^^^ */

/* vvvvvvvvvvvvvvvvvvvvv recorder vvvvvvvvvvvvvvvvvvvvv */

// sessionRecorder turns the agent's run into the session's turns. RunLoop calls it on the goroutine running the turn,
// which is also the only one touching the agent's messages, so OnEnd can save them.
type sessionRecorder struct {
	s *Session
}

func (r *sessionRecorder) OnStart(_ Model, _ string, userMessages []Message) {
	s := r.s
	s.mu.Lock()
	defer s.mu.Unlock()
	turn := &SessionTurn{ID: len(s.turns) + 1, Status: "running", StartedAt: time.Now(), Steps: []SessionStep{}}
	if s.userPrompt != "" {
		turn.Source, turn.Prompt = "user", s.userPrompt
		s.userPrompt = ""
	} else {
		turn.Source, turn.Prompt = "cron", messagesText(userMessages)
	}
	s.turns = append(s.turns, turn)
	s.current = turn
	s.finalStep = -1
	s.updatedAt = turn.StartedAt
	s.broadcastLocked(SessionEvent{Type: "turn", Turn: ptr(turn.clone())})
	s.broadcastLocked(SessionEvent{Type: "session", Session: ptr(s.viewLocked(false))})
}

func (r *sessionRecorder) OnResponse(_ int, resp SendMessagesResponse) {
	s := r.s
	s.mu.Lock()
	defer s.mu.Unlock()
	turn := s.current
	if turn == nil {
		return
	}
	step := SessionStep{Kind: "model"}
	var tools []SessionStep
	for _, block := range resp.Content {
		switch block.Type {
		case MessagesBlockTypeText:
			step.Text = joinNonEmpty(step.Text, block.Text)
		case MessagesBlockTypeReasoning:
			step.Reasoning = joinNonEmpty(step.Reasoning, block.Text)
		case MessagesBlockTypeToolUse:
			input, _ := json.Marshal(block.Input)
			tools = append(tools, SessionStep{Kind: "tool", ToolUseID: block.ID, Tool: block.Name, Input: string(input), Status: "running"})
		}
	}
	s.finalStep = -1
	if step.Text != "" || step.Reasoning != "" {
		turn.Steps = append(turn.Steps, step)
		if len(tools) == 0 {
			s.finalStep = len(turn.Steps) - 1
		}
	}
	turn.Steps = append(turn.Steps, tools...)
	s.broadcastLocked(SessionEvent{Type: "turn", Turn: ptr(turn.clone())})
}

func (r *sessionRecorder) OnToolResult(_ int, toolUse MessagesBlock, output string, took time.Duration) {
	s := r.s
	s.mu.Lock()
	defer s.mu.Unlock()
	turn := s.current
	if turn == nil {
		return
	}
	for i := range turn.Steps {
		step := &turn.Steps[i]
		if step.Kind == "tool" && step.ToolUseID == toolUse.ID && step.Status == "running" {
			step.Output = excerpt(output, sessionOutputBytes)
			step.Status = "done"
			step.TookMS = took.Milliseconds()
			break
		}
	}
	s.broadcastLocked(SessionEvent{Type: "turn", Turn: ptr(turn.clone())})
}

func (r *sessionRecorder) OnEnd(err error) {
	s := r.s
	s.mu.Lock()
	turn := s.current
	if turn != nil {
		// the last thing the model said without calling a tool is the reply, not a step
		if s.finalStep >= 0 && s.finalStep < len(turn.Steps) {
			turn.Reply = turn.Steps[s.finalStep].Text
			turn.Steps = append(turn.Steps[:s.finalStep], turn.Steps[s.finalStep+1:]...)
		}
		turn.Status, turn.EndedAt = "done", time.Now()
		if err != nil {
			turn.Status, turn.Error = "failed", err.Error()
		}
		s.updatedAt = turn.EndedAt
		s.current = nil
		s.broadcastLocked(SessionEvent{Type: "turn", Turn: ptr(turn.clone())})
		s.broadcastLocked(SessionEvent{Type: "session", Session: ptr(s.viewLocked(false))})
	}
	s.mu.Unlock()
	if err := s.save(); err != nil {
		log4go.DefaultLogger().Error(s.ctx, "[session %s] save after turn failed: %v", s.id, err)
	}
}

func (t *SessionTurn) clone() SessionTurn {
	c := *t
	c.Steps = append([]SessionStep{}, t.Steps...)
	return c
}

/* ^^^^^^^^^^^^^^^^^^^^^ recorder ^^^^^^^^^^^^^^^^^^^^^ */

/* vvvvvvvvvvvvvvvvvvvvv persistence vvvvvvvvvvvvvvvvvvvvv */

// sessionFile is a session on disk: the agent's history, so the conversation continues after a restart, and the
// turns, so the page can show it.
type sessionFile struct {
	ID        string          `json:"id"`
	Title     string          `json:"title"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
	Summary   string          `json:"summary,omitempty"`
	Messages  []storedMessage `json:"messages"`
	Turns     []*SessionTurn  `json:"turns"`
}

// storedMessage is a Message with its Content typed: either plain text or blocks.
type storedMessage struct {
	Role   MessageRole     `json:"role"`
	Text   *string         `json:"text,omitempty"`
	Blocks []MessagesBlock `json:"blocks,omitempty"`
}

// save writes the session atomically. It reads the agent's history, so it is only called when no turn is running or
// from the turn's own goroutine.
func (s *Session) save() error {
	s.mu.Lock()
	if s.deleted {
		s.mu.Unlock()
		return nil
	}
	saved := sessionFile{
		ID:        s.id,
		Title:     s.title,
		CreatedAt: s.createdAt,
		UpdatedAt: s.updatedAt,
		Summary:   s.agent.summary,
		Messages:  encodeMessages(s.agent.messages),
		Turns:     s.turns,
	}
	content, err := json.MarshalIndent(saved, "", "  ")
	s.mu.Unlock()
	if err != nil {
		log4go.DefaultLogger().Error(s.ctx, "encode session failed: %v, id: %s", err, s.id)
		return err
	}
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		log4go.DefaultLogger().Error(s.ctx, "create session dir failed: %v, dir: %s", err, s.dir)
		return err
	}
	tmp := filepath.Join(s.dir, sessionFileName+".tmp")
	if err := os.WriteFile(tmp, content, 0o644); err != nil {
		log4go.DefaultLogger().Error(s.ctx, "write session file failed: %v, path: %s", err, tmp)
		return err
	}
	if err := os.Rename(tmp, filepath.Join(s.dir, sessionFileName)); err != nil {
		log4go.DefaultLogger().Error(s.ctx, "replace session file failed: %v, dir: %s", err, s.dir)
		return err
	}
	return nil
}

func encodeMessages(messages []Message) []storedMessage {
	out := make([]storedMessage, 0, len(messages))
	for _, msg := range messages {
		stored := storedMessage{Role: msg.Role}
		switch content := msg.Content.(type) {
		case string:
			stored.Text = &content
		case []MessagesBlock:
			stored.Blocks = content
		default:
			text := fmt.Sprintf("%v", content)
			stored.Text = &text
		}
		out = append(out, stored)
	}
	return out
}

func decodeMessages(stored []storedMessage) []Message {
	out := make([]Message, 0, len(stored))
	for _, msg := range stored {
		if msg.Text != nil {
			out = append(out, Message{Role: msg.Role, Content: *msg.Text})
			continue
		}
		out = append(out, Message{Role: msg.Role, Content: msg.Blocks})
	}
	return out
}

/* ^^^^^^^^^^^^^^^^^^^^^ persistence ^^^^^^^^^^^^^^^^^^^^^ */

// messagesText is the text of the messages that started a turn.
func messagesText(messages []Message) string {
	var parts []string
	for _, msg := range messages {
		switch content := msg.Content.(type) {
		case string:
			parts = append(parts, content)
		case []MessagesBlock:
			for _, block := range content {
				if block.Type == MessagesBlockTypeText {
					parts = append(parts, block.Text)
				}
			}
		}
	}
	return strings.Join(parts, "\n")
}

func joinNonEmpty(a, b string) string {
	if a == "" {
		return b
	}
	if b == "" {
		return a
	}
	return a + "\n" + b
}

func truncateRunes(s string, max int) string {
	runes := []rune(strings.Join(strings.Fields(s), " "))
	if len(runes) <= max {
		return string(runes)
	}
	return string(runes[:max]) + "…"
}

func ptr[T any](v T) *T { return &v }
