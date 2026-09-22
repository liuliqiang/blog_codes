package agentloop

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/liuliqiang/log4go"
)

// cronPath is the file durable cron jobs are saved to, relative to the working directory like tasksDir; tests swap it.
var cronPath = ".scheduled_tasks.json"

// cronOut is where scheduler activity is printed; tests swap it.
var cronOut io.Writer = os.Stdout

// CronJob is one scheduled prompt. Recurring jobs fire on every match, one-shot jobs are removed once delivered.
// Durable jobs survive a restart; the others live only as long as the process.
type CronJob struct {
	ID        string `json:"id"`
	Cron      string `json:"cron"`
	Prompt    string `json:"prompt"`
	Recurring bool   `json:"recurring"`
	Durable   bool   `json:"durable"`

	// PendingDelivery is set from the moment the job becomes due until the model has accepted its prompt, so the
	// same firing is never queued twice. LastFired is the "YYYY-MM-DD HH:MM" it last became due, so a match does
	// not re-fire within the same minute.
	PendingDelivery bool   `json:"pending_delivery"`
	LastFired       string `json:"last_fired,omitempty"`
}

/* vvvvvvvvvvvvvvvvvvvvv cron expressions vvvvvvvvvvvvvvvvvvvvv */

// cronFields are the five fields of an expression with their allowed ranges. Day of week is 0-6 with 0 = Sunday,
// matching time.Weekday.
var cronFields = []struct {
	name     string
	min, max int
}{
	{"minute", 0, 59},
	{"hour", 0, 23},
	{"day-of-month", 1, 31},
	{"month", 1, 12},
	{"day-of-week", 0, 6},
}

// validateCron checks a five-field expression supporting *, */N, N, N-M and comma lists of those.
func validateCron(expr string) error {
	fields := strings.Fields(expr)
	if len(fields) != len(cronFields) {
		return fmt.Errorf("expected 5 fields, got %d", len(fields))
	}
	for i, field := range fields {
		if err := validateCronField(field, cronFields[i].min, cronFields[i].max); err != nil {
			return fmt.Errorf("%s: %w", cronFields[i].name, err)
		}
	}
	return nil
}

func validateCronField(field string, min, max int) error {
	switch {
	case field == "*":
		return nil
	case strings.HasPrefix(field, "*/"):
		step, err := strconv.Atoi(field[2:])
		if err != nil || step <= 0 {
			return fmt.Errorf("invalid step %q", field)
		}
		return nil
	case strings.Contains(field, ","):
		for _, part := range strings.Split(field, ",") {
			if err := validateCronField(strings.TrimSpace(part), min, max); err != nil {
				return err
			}
		}
		return nil
	case strings.Contains(field, "-"):
		lo, hi, _ := strings.Cut(field, "-")
		start, err1 := strconv.Atoi(lo)
		end, err2 := strconv.Atoi(hi)
		if err1 != nil || err2 != nil {
			return fmt.Errorf("invalid range %q", field)
		}
		if start > end {
			return fmt.Errorf("range start is greater than end in %q", field)
		}
		if start < min || end > max {
			return fmt.Errorf("range %q is outside [%d-%d]", field, min, max)
		}
		return nil
	default:
		v, err := strconv.Atoi(field)
		if err != nil {
			return fmt.Errorf("invalid field %q", field)
		}
		if v < min || v > max {
			return fmt.Errorf("value %d is outside [%d-%d]", v, min, max)
		}
		return nil
	}
}

// cronMatches reports whether a validated expression matches the minute of t. As in cron, when both day-of-month
// and day-of-week are restricted, either one matching is enough.
func cronMatches(expr string, t time.Time) bool {
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return false
	}
	minute, hour, dom, month, dow := fields[0], fields[1], fields[2], fields[3], fields[4]
	if !cronFieldMatches(minute, t.Minute()) || !cronFieldMatches(hour, t.Hour()) || !cronFieldMatches(month, int(t.Month())) {
		return false
	}
	domOK := cronFieldMatches(dom, t.Day())
	dowOK := cronFieldMatches(dow, int(t.Weekday()))
	switch {
	case dom == "*" && dow == "*":
		return true
	case dom == "*":
		return dowOK
	case dow == "*":
		return domOK
	default:
		return domOK || dowOK
	}
}

func cronFieldMatches(field string, value int) bool {
	switch {
	case field == "*":
		return true
	case strings.HasPrefix(field, "*/"):
		step, _ := strconv.Atoi(field[2:])
		return step > 0 && value%step == 0
	case strings.Contains(field, ","):
		for _, part := range strings.Split(field, ",") {
			if cronFieldMatches(strings.TrimSpace(part), value) {
				return true
			}
		}
		return false
	case strings.Contains(field, "-"):
		lo, hi, _ := strings.Cut(field, "-")
		start, _ := strconv.Atoi(lo)
		end, _ := strconv.Atoi(hi)
		return start <= value && value <= end
	default:
		v, _ := strconv.Atoi(field)
		return value == v
	}
}

func minuteMarker(t time.Time) string { return t.Format("2006-01-02 15:04") }

/* vvvvvvvvvvvvvvvvvvvvv store vvvvvvvvvvvvvvvvvvvvv */

// CronStore keeps the scheduled jobs and the queue of jobs that are due but not yet delivered. Durable jobs are
// mirrored to a JSON file; every change is written before it takes effect in memory, and rolled back if the write
// fails, so the file never claims something the process did not do.
type CronStore struct {
	mu      sync.Mutex
	path    string
	jobs    map[string]*CronJob
	queue   []string // ids of due jobs in firing order
	wake    chan struct{}
	loadErr error
}

// NewCronStore loads the durable jobs from path. A missing file is an empty store; a corrupt one is remembered as
// loadErr so the scheduler can refuse to start instead of silently dropping jobs.
func NewCronStore(path string) *CronStore {
	s := &CronStore{path: path, jobs: map[string]*CronJob{}, wake: make(chan struct{}, 1)}
	s.loadErr = s.load()
	return s
}

func (s *CronStore) load() error {
	ctx := context.Background()
	content, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		log4go.DefaultLogger().Error(ctx, "read cron file failed: %v, path: %s", err, s.path)
		return err
	}
	var saved []CronJob
	if err := json.Unmarshal(content, &saved); err != nil {
		log4go.DefaultLogger().Error(ctx, "unmarshal cron file failed: %v, path: %s", err, s.path)
		return fmt.Errorf("%s is corrupt: %w", s.path, err)
	}
	for i := range saved {
		job := saved[i]
		if !strings.HasPrefix(job.ID, "cron_") || strings.TrimSpace(job.Prompt) == "" || validateCron(job.Cron) != nil {
			log4go.DefaultLogger().Error(ctx, "skipping invalid saved cron job: %+v", job)
			continue
		}
		job.Durable = true
		s.jobs[job.ID] = &job
		if job.PendingDelivery {
			// it became due before the last shutdown and was never acknowledged: deliver it again (at least once)
			s.queue = append(s.queue, job.ID)
		}
	}
	if len(s.jobs) > 0 {
		fmt.Fprintf(cronOut, "\033[35m[cron] loaded %d durable job(s)\033[0m\n", len(s.jobs))
	}
	if len(s.queue) > 0 {
		s.signal()
	}
	return nil
}

// save writes every durable job through a temp file and rename. Callers hold s.mu.
func (s *CronStore) save() error {
	ctx := context.Background()
	var durable []CronJob
	for _, id := range s.sortedIDs() {
		if job := s.jobs[id]; job.Durable {
			durable = append(durable, *job)
		}
	}
	if len(durable) == 0 {
		if err := os.Remove(s.path); err != nil && !os.IsNotExist(err) {
			log4go.DefaultLogger().Error(ctx, "remove cron file failed: %v, path: %s", err, s.path)
			return err
		}
		return nil
	}
	payload, err := json.MarshalIndent(durable, "", "  ")
	if err != nil {
		log4go.DefaultLogger().Error(ctx, "marshal cron jobs failed: %v", err)
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, payload, 0o644); err != nil {
		log4go.DefaultLogger().Error(ctx, "write cron temp file failed: %v, path: %s", err, tmp)
		return err
	}
	if err := os.Rename(tmp, s.path); err != nil {
		log4go.DefaultLogger().Error(ctx, "rename cron file failed: %v, from: %s, to: %s", err, tmp, s.path)
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func (s *CronStore) sortedIDs() []string {
	ids := make([]string, 0, len(s.jobs))
	for id := range s.jobs {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// signal wakes a scheduler waiting for queued jobs; it never blocks.
func (s *CronStore) signal() {
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

// Schedule validates and registers a job.
func (s *CronStore) Schedule(expr, prompt string, recurring, durable bool) (CronJob, error) {
	if err := validateCron(expr); err != nil {
		return CronJob{}, err
	}
	if strings.TrimSpace(prompt) == "" {
		return CronJob{}, errors.New("prompt cannot be empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var id string
	for attempt := 0; ; attempt++ {
		var raw [4]byte
		if _, err := rand.Read(raw[:]); err != nil {
			log4go.DefaultLogger().Error(context.Background(), "generate cron ID failed: %v", err)
			return CronJob{}, err
		}
		id = "cron_" + hex.EncodeToString(raw[:])
		if _, taken := s.jobs[id]; !taken {
			break
		}
		if attempt == 100 {
			return CronJob{}, errors.New("could not allocate a cron job ID")
		}
	}
	job := &CronJob{ID: id, Cron: strings.Join(strings.Fields(expr), " "), Prompt: prompt, Recurring: recurring, Durable: durable}
	s.jobs[id] = job
	if durable {
		if err := s.save(); err != nil {
			delete(s.jobs, id)
			return CronJob{}, err
		}
	}
	fmt.Fprintf(
		cronOut,
		"\033[35m[cron] scheduled %s: %s -> %s\033[0m\n",
		id,
		job.Cron,
		excerpt(prompt, 60), // prompt preview
	)
	return *job, nil
}

// Cancel removes a job, including a pending delivery of it.
func (s *CronStore) Cancel(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, ok := s.jobs[id]
	if !ok {
		return fmt.Errorf("job %s not found", id)
	}
	prevQueue := append([]string(nil), s.queue...)
	delete(s.jobs, id)
	s.queue = removeID(s.queue, id)
	if job.Durable {
		if err := s.save(); err != nil {
			s.jobs[id] = job
			s.queue = prevQueue
			return err
		}
	}
	fmt.Fprintf(cronOut, "\033[35m[cron] cancelled %s\033[0m\n", id)
	return nil
}

func removeID(ids []string, id string) []string {
	out := ids[:0:0]
	for _, x := range ids {
		if x != id {
			out = append(out, x)
		}
	}
	return out
}

// List returns every job sorted by ID.
func (s *CronStore) List() []CronJob {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []CronJob
	for _, id := range s.sortedIDs() {
		out = append(out, *s.jobs[id])
	}
	return out
}

// PollDue queues every job whose expression matches now and that has neither fired this minute nor is still waiting
// for delivery. It returns the jobs it queued.
func (s *CronStore) PollDue(now time.Time) []CronJob {
	marker := minuteMarker(now)
	s.mu.Lock()
	defer s.mu.Unlock()
	var due []CronJob
	for _, id := range s.sortedIDs() {
		job := s.jobs[id]
		if job.PendingDelivery || job.LastFired == marker || !cronMatches(job.Cron, now) {
			continue
		}
		prevPending, prevFired := job.PendingDelivery, job.LastFired
		job.PendingDelivery, job.LastFired = true, marker
		if job.Durable {
			if err := s.save(); err != nil {
				job.PendingDelivery, job.LastFired = prevPending, prevFired
				log4go.DefaultLogger().Error(context.Background(), "could not queue cron job %s: %v", id, err)
				continue
			}
		}
		s.queue = append(s.queue, id)
		due = append(due, *job)
		fmt.Fprintf(
			cronOut,
			"\033[35m[cron] due %s: %s\033[0m\n",
			id,
			excerpt(job.Prompt, 60), // prompt preview
		)
	}
	if len(due) > 0 {
		s.signal()
	}
	return due
}

// Consume takes every queued job out of the queue.
func (s *CronStore) Consume() []CronJob {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []CronJob
	for _, id := range s.queue {
		if job, ok := s.jobs[id]; ok {
			out = append(out, *job)
		}
	}
	s.queue = nil
	return out
}

// Acknowledge records that the model accepted the prompts: one-shot jobs are removed, recurring ones become eligible
// for their next match.
func (s *CronStore) Acknowledge(delivered []CronJob) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	type change struct {
		job     *CronJob
		pending bool
		removed bool
	}
	var changes []change
	durable := false
	for _, d := range delivered {
		job, ok := s.jobs[d.ID]
		if !ok {
			continue
		}
		c := change{job: job, pending: job.PendingDelivery}
		if job.Recurring {
			job.PendingDelivery = false
		} else {
			delete(s.jobs, job.ID)
			c.removed = true
		}
		changes = append(changes, c)
		durable = durable || job.Durable
	}
	if !durable {
		return nil
	}
	if err := s.save(); err != nil {
		for _, c := range changes {
			if c.removed {
				s.jobs[c.job.ID] = c.job
			}
			c.job.PendingDelivery = c.pending
		}
		return err
	}
	return nil
}

// Restore puts delivered jobs back in the queue after the model could not be reached.
func (s *CronStore) Restore(delivered []CronJob) {
	s.mu.Lock()
	defer s.mu.Unlock()
	queued := map[string]bool{}
	for _, id := range s.queue {
		queued[id] = true
	}
	for _, d := range delivered {
		job, ok := s.jobs[d.ID]
		if !ok {
			continue
		}
		job.PendingDelivery = true
		if !queued[job.ID] {
			s.queue = append(s.queue, job.ID)
			queued[job.ID] = true
		}
	}
	if len(s.queue) > 0 {
		s.signal()
	}
}

/* vvvvvvvvvvvvvvvvvvvvv delivery vvvvvvvvvvvvvvvvvvvvv */

// cronDeliveryKey carries the jobs a scheduled RunLoop was started for, so the loop can acknowledge them once the
// model accepts the prompt or restore them if it never does.
type cronDeliveryKey struct{}

func withCronDelivery(ctx context.Context, jobs []CronJob) context.Context {
	return context.WithValue(ctx, cronDeliveryKey{}, jobs)
}

func cronDeliveryFrom(ctx context.Context) []CronJob {
	jobs, _ := ctx.Value(cronDeliveryKey{}).([]CronJob)
	return jobs
}

// schedulerPollInterval is how often the scheduler checks the clock; tests shorten it.
var schedulerPollInterval = time.Second

// Scheduler fires due cron jobs into the agent. It polls the clock in one goroutine and delivers queued prompts from
// Run's goroutine. Every turn on the agent — scheduled or the user's via RunTurn — takes the same lock, so turns never
// overlap while jobs that come due during one keep queuing and are delivered as soon as it ends.
type Scheduler struct {
	agent Agent
	store *CronStore
	turn  sync.Mutex
}

// NewScheduler binds a scheduler to the agent's cron store. Only agents built by NewAgent have one.
func NewScheduler(a Agent) (*Scheduler, error) {
	impl, ok := a.(*agent)
	if !ok || impl.cron == nil {
		return nil, errors.New("agent has no cron store")
	}
	return &Scheduler{agent: a, store: impl.cron}, nil
}

// HasJobs reports whether anything is scheduled or waiting for delivery.
func (s *Scheduler) HasJobs() bool {
	s.store.mu.Lock()
	defer s.store.mu.Unlock()
	return len(s.store.jobs) > 0 || len(s.store.queue) > 0
}

// RunTurn runs the user's own turn on the agent, waiting for any scheduled turn in progress to finish first.
func (s *Scheduler) RunTurn(ctx context.Context, messages []Message) error {
	s.turn.Lock()
	defer s.turn.Unlock()
	return s.agent.RunLoop(ctx, messages)
}

// Run blocks until ctx is done, delivering every due job as a fresh `[Scheduled] <prompt>` turn. Scheduled turns run
// non-interactively: a tool call that would need the user's approval is denied instead of waiting on the terminal.
func (s *Scheduler) Run(ctx context.Context) error {
	if s.store.loadErr != nil {
		return s.store.loadErr
	}
	go s.poll(ctx)
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-s.store.wake:
		}
		for jobs := s.store.Consume(); len(jobs) > 0; jobs = s.store.Consume() {
			s.deliver(ctx, jobs)
		}
	}
}

func (s *Scheduler) poll(ctx context.Context) {
	ticker := time.NewTicker(schedulerPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			s.store.PollDue(now)
		}
	}
}

func (s *Scheduler) deliver(ctx context.Context, jobs []CronJob) {
	var messages []Message
	for _, job := range jobs {
		messages = append(messages, Message{Role: MessageRoleUser, Content: "[Scheduled] " + job.Prompt})
		fmt.Fprintf(
			cronOut,
			"\033[35m[cron] delivered %s: %s\033[0m\n",
			job.ID,
			excerpt(job.Prompt, 60), // prompt preview
		)
	}
	runCtx := withCronDelivery(
		withNonInteractive(ctx), // ctx
		jobs,                    // jobs
	)
	s.turn.Lock()
	defer s.turn.Unlock()
	if ctx.Err() != nil {
		// the scheduler was stopped while we waited for the lock; leave the jobs queued for the next start
		s.store.Restore(jobs)
		return
	}
	if err := s.agent.RunLoop(runCtx, messages); err != nil {
		log4go.DefaultLogger().Error(ctx, "scheduled turn failed: %v, jobs: %d", err, len(jobs))
		fmt.Fprintf(cronOut, "\033[35m[cron] scheduled turn failed: %v\033[0m\n", err)
	}
}

/* vvvvvvvvvvvvvvvvvvvvv tool handlers vvvvvvvvvvvvvvvvvvvvv */

const cronSystemPromptGuidance = `When the user wants something to happen on a schedule ("every morning at 9", "every 30 minutes"), register it with schedule_cron instead of doing it once; the prompt you give it is what a future turn will be asked to do.`

func (a *agent) runScheduleCron(ctx context.Context, input map[string]interface{}) (string, error) {
	expr, err := stringArg(input, "cron")
	if err != nil {
		return "", err
	}
	prompt, err := stringArg(input, "prompt")
	if err != nil {
		return "", err
	}
	recurring, err := optionalBoolArg(input, "recurring", true)
	if err != nil {
		return "", err
	}
	durable, err := optionalBoolArg(input, "durable", true)
	if err != nil {
		return "", err
	}
	job, err := a.cron.Schedule(expr, prompt, recurring, durable)
	if err != nil {
		log4go.DefaultLogger().Error(ctx, "[%s] schedule_cron failed: %v, input: %+v", agentNameFrom(ctx), err, input)
		return "", err
	}
	return fmt.Sprintf("Scheduled %s: %s -> %s", job.ID, job.Cron, job.Prompt), nil
}

func (a *agent) runListCrons(ctx context.Context, _ map[string]interface{}) (string, error) {
	jobs := a.cron.List()
	if len(jobs) == 0 {
		if a.cron.loadErr != nil {
			return "", a.cron.loadErr
		}
		return "No cron jobs. Use schedule_cron to add one.", nil
	}
	var lines []string
	for _, job := range jobs {
		kind := "one-shot"
		if job.Recurring {
			kind = "recurring"
		}
		storage := "memory"
		if job.Durable {
			storage = "durable"
		}
		line := fmt.Sprintf("%s  %s  [%s, %s]", job.ID, job.Cron, kind, storage)
		if job.PendingDelivery {
			line += " [pending]"
		}
		lines = append(lines, line+"  -> "+job.Prompt)
	}
	return strings.Join(lines, "\n"), nil
}

func (a *agent) runCancelCron(ctx context.Context, input map[string]interface{}) (string, error) {
	id, err := stringArg(input, "job_id")
	if err != nil {
		return "", err
	}
	if err := a.cron.Cancel(id); err != nil {
		log4go.DefaultLogger().Error(ctx, "[%s] cancel_cron failed: %v, id: %s", agentNameFrom(ctx), err, id)
		return "", err
	}
	return "Cancelled " + id, nil
}

func optionalBoolArg(input map[string]interface{}, key string, defaultValue bool) (bool, error) {
	v, ok := input[key]
	if !ok {
		return defaultValue, nil
	}
	b, ok := v.(bool)
	if !ok {
		return false, fmt.Errorf("argument %q must be a boolean, got %T", key, v)
	}
	return b, nil
}

/* vvvvvvvvvvvvvvvvvvvvv non-interactive turns vvvvvvvvvvvvvvvvvvvvv */

// nonInteractiveKey marks a ctx whose turn has nobody at the terminal to answer prompts.
type nonInteractiveKey struct{}

func withNonInteractive(ctx context.Context) context.Context {
	return context.WithValue(ctx, nonInteractiveKey{}, true)
}

func isNonInteractive(ctx context.Context) bool {
	v, _ := ctx.Value(nonInteractiveKey{}).(bool)
	return v
}
