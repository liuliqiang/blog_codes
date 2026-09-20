package agentloop

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// useTasksDir points the task store at a temp dir for the test.
func useTasksDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), ".tasks")
	orig := tasksDir
	tasksDir = dir
	t.Cleanup(func() { tasksDir = orig })
	return dir
}

func mustCreate(t *testing.T, s *TaskStore, subject string) Task {
	t.Helper()
	task, err := s.Create(subject, "")
	if err != nil {
		t.Fatal(err)
	}
	return task
}

func TestTaskStore_CreateLoadList(t *testing.T) {
	dir := useTasksDir(t)
	s := NewTaskStore(dir)

	if tasks, err := s.List(); err != nil || len(tasks) != 0 {
		t.Fatalf("empty store: %v %v", tasks, err)
	}
	if _, err := s.Create("  ", ""); err == nil {
		t.Error("blank subject must be rejected")
	}

	task, err := s.Create("  setup schema ", "create the tables")
	if err != nil {
		t.Fatal(err)
	}
	if !taskIDRe.MatchString(task.ID) || task.Subject != "setup schema" || task.Status != TaskPending || task.Owner != "" || len(task.BlockedBy) != 0 {
		t.Errorf("created = %+v", task)
	}

	raw, err := os.ReadFile(filepath.Join(dir, task.ID+".json"))
	if err != nil {
		t.Fatal(err)
	}
	var onDisk map[string]interface{}
	if err := json.Unmarshal(raw, &onDisk); err != nil {
		t.Fatal(err)
	}
	if onDisk["id"] != task.ID || onDisk["status"] != "pending" || onDisk["owner"] != "" || !reflect.DeepEqual(onDisk["blockedBy"], []interface{}{}) {
		t.Errorf("on disk = %v", onDisk)
	}

	loaded, err := s.Load(task.ID)
	if err != nil || !reflect.DeepEqual(loaded, task) {
		t.Errorf("Load = %+v, %v", loaded, err)
	}
	second := mustCreate(t, s, "second")
	tasks, err := s.List()
	if err != nil || len(tasks) != 2 {
		t.Fatalf("List = %v, %v", tasks, err)
	}
	if tasks[0].ID > tasks[1].ID {
		t.Error("List must be sorted by ID")
	}
	_ = second
}

func TestTaskStore_LoadRejectsBadFiles(t *testing.T) {
	dir := useTasksDir(t)
	s := NewTaskStore(dir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	for _, id := range []string{"nope", "task_xyz", "../task_deadbeef", "task_DEADBEEF"} {
		if _, err := s.Load(id); err == nil || !strings.Contains(err.Error(), "invalid task ID") {
			t.Errorf("Load(%q) err = %v", id, err)
		}
	}
	if _, err := s.Load("task_00000000"); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("missing err = %v", err)
	}

	write := func(id, content string) {
		if err := os.WriteFile(filepath.Join(dir, id+".json"), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("task_aaaaaaaa", `{"id":"task_bbbbbbbb","subject":"x","status":"pending"}`)
	if _, err := s.Load("task_aaaaaaaa"); err == nil || !strings.Contains(err.Error(), "holds ID") {
		t.Errorf("mismatched id err = %v", err)
	}
	write("task_cccccccc", `{"id":"task_cccccccc","subject":"x","status":"done"}`)
	if _, err := s.Load("task_cccccccc"); err == nil || !strings.Contains(err.Error(), "invalid status") {
		t.Errorf("bad status err = %v", err)
	}
	write("task_dddddddd", `not json`)
	if _, err := s.Load("task_dddddddd"); err == nil || !strings.Contains(err.Error(), "corrupt") {
		t.Errorf("corrupt err = %v", err)
	}
	// a bad file poisons List too, rather than being silently dropped
	if _, err := s.List(); err == nil {
		t.Error("List should surface the corrupt file")
	}
}

func TestTaskStore_AddDependencies(t *testing.T) {
	s := NewTaskStore(useTasksDir(t))
	schema := mustCreate(t, s, "schema")
	api := mustCreate(t, s, "api")
	tests := mustCreate(t, s, "tests")

	got, err := s.AddDependencies(api.ID, []string{schema.ID, schema.ID})
	if err != nil || !reflect.DeepEqual(got.BlockedBy, []string{schema.ID}) {
		t.Fatalf("AddDependencies = %+v, %v", got, err)
	}
	// repeating an edge is a no-op
	if got, err := s.AddDependencies(api.ID, []string{schema.ID}); err != nil || len(got.BlockedBy) != 1 {
		t.Errorf("repeat = %+v, %v", got, err)
	}
	if _, err := s.AddDependencies(tests.ID, []string{api.ID}); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name string
		id   string
		deps []string
		want string
	}{
		{"self", schema.ID, []string{schema.ID}, "depend on itself"},
		{"missing dep", schema.ID, []string{"task_00000000"}, "dependency not found"},
		{"bad dep id", schema.ID, []string{"x"}, "dependency not found"},
		{"direct cycle", schema.ID, []string{api.ID}, "cycle"},
		{"transitive cycle", schema.ID, []string{tests.ID}, "cycle"},
		{"unknown task", "task_00000000", []string{schema.ID}, "not found"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := s.AddDependencies(c.id, c.deps); err == nil || !strings.Contains(err.Error(), c.want) {
				t.Errorf("err = %v, want %q", err, c.want)
			}
		})
	}
	// a rejected change must not be partially saved
	if got, _ := s.Load(schema.ID); len(got.BlockedBy) != 0 {
		t.Errorf("schema deps changed after rejected updates: %v", got.BlockedBy)
	}

	// once claimed, dependencies are frozen
	if _, err := s.Claim(schema.ID, "main"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AddDependencies(schema.ID, []string{tests.ID}); err == nil || !strings.Contains(err.Error(), "pending and unowned") {
		t.Errorf("claimed err = %v", err)
	}
}

func TestTaskStore_ClaimAndComplete(t *testing.T) {
	s := NewTaskStore(useTasksDir(t))
	schema := mustCreate(t, s, "schema")
	api := mustCreate(t, s, "api")
	tests := mustCreate(t, s, "tests")
	docs := mustCreate(t, s, "docs")
	for _, e := range [][2]string{{api.ID, schema.ID}, {tests.ID, api.ID}, {docs.ID, schema.ID}} {
		if _, err := s.AddDependencies(e[0], []string{e[1]}); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := s.Claim(api.ID, "main"); err == nil || !strings.Contains(err.Error(), "blocked by: "+schema.ID) {
		t.Errorf("blocked claim err = %v", err)
	}
	got, err := s.Claim(schema.ID, "main")
	if err != nil || got.Status != TaskInProgress || got.Owner != "main" {
		t.Fatalf("Claim = %+v, %v", got, err)
	}
	if _, err := s.Claim(schema.ID, "main"); err == nil || !strings.Contains(err.Error(), "is in_progress") {
		t.Errorf("double claim err = %v", err)
	}

	if _, _, err := s.Complete(api.ID, "main"); err == nil || !strings.Contains(err.Error(), "is pending") {
		t.Errorf("complete pending err = %v", err)
	}
	if _, _, err := s.Complete(schema.ID, "subagent"); err == nil || !strings.Contains(err.Error(), "owned by main, not subagent") {
		t.Errorf("wrong owner err = %v", err)
	}
	done, unblocked, err := s.Complete(schema.ID, "main")
	if err != nil || done.Status != TaskCompleted {
		t.Fatalf("Complete = %+v, %v", done, err)
	}
	if names := subjects(unblocked); !reflect.DeepEqual(sorted(names), []string{"api", "docs"}) {
		t.Errorf("unblocked = %v, want api, docs", names)
	}
	if _, _, err := s.Complete(schema.ID, "main"); err == nil || !strings.Contains(err.Error(), "is completed") {
		t.Errorf("double complete err = %v", err)
	}

	// completing api unblocks only tests; docs was already ready
	if _, err := s.Claim(api.ID, "main"); err != nil {
		t.Fatal(err)
	}
	if _, unblocked, err := s.Complete(api.ID, "main"); err != nil || !reflect.DeepEqual(subjects(unblocked), []string{"tests"}) {
		t.Errorf("unblocked after api = %v, %v", subjects(unblocked), err)
	}

	// a prerequisite whose file vanished keeps the task blocked
	if err := os.Remove(filepath.Join(tasksDir, api.ID+".json")); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Claim(tests.ID, "main"); err == nil || !strings.Contains(err.Error(), "blocked by") {
		t.Errorf("missing prerequisite should block: %v", err)
	}
}

func subjects(tasks []Task) []string {
	var out []string
	for _, t := range tasks {
		out = append(out, t.Subject)
	}
	return out
}

func sorted(s []string) []string {
	out := append([]string(nil), s...)
	for i := range out {
		for j := i + 1; j < len(out); j++ {
			if out[j] < out[i] {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

func TestStringSliceArg(t *testing.T) {
	cases := []struct {
		name  string
		input map[string]interface{}
		want  []string
		ok    bool
	}{
		{"array", map[string]interface{}{"k": []interface{}{"a", "b"}}, []string{"a", "b"}, true},
		{"json string", map[string]interface{}{"k": `["a"]`}, []string{"a"}, true},
		{"empty", map[string]interface{}{"k": []interface{}{}}, []string{}, true},
		{"missing", map[string]interface{}{}, nil, false},
		{"not array", map[string]interface{}{"k": 42}, nil, false},
		{"bad json", map[string]interface{}{"k": `a`}, nil, false},
		{"non-string item", map[string]interface{}{"k": []interface{}{1}}, nil, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := stringSliceArg(c.input, "k")
			if (err == nil) != c.ok || (c.ok && !reflect.DeepEqual(got, c.want)) {
				t.Errorf("got %v, %v; want %v, ok=%v", got, err, c.want, c.ok)
			}
		})
	}
}

func TestTaskTools_Registered(t *testing.T) {
	useTasksDir(t)
	a := NewAgent(nil, nil).(*agent)
	for _, name := range []string{"create_task", "update_task", "list_tasks", "get_task", "claim_task", "complete_task"} {
		if _, ok := a.toolIndex[name]; !ok {
			t.Errorf("%s not registered", name)
		}
		if !isAllowListed(MessagesBlock{Name: name}) {
			t.Errorf("%s should be allow-listed", name)
		}
	}
	if !strings.Contains(a.systemPrompt, "create every task with create_task first") {
		t.Error("system prompt lacks task guidance")
	}
	if sub := a.newSubagent(); sub.tasks != a.tasks {
		t.Error("subagent must share the task store")
	} else if _, ok := sub.toolIndex["claim_task"]; !ok {
		t.Error("subagent should get the task tools")
	}
}

func TestTaskTools_Handlers(t *testing.T) {
	useTasksDir(t)
	a := NewAgent(nil, nil).(*agent)
	ctx := withAgentName(context.Background(), "main")
	call := func(name string, input map[string]interface{}) (string, error) {
		return a.toolIndex[name].Handler(ctx, input)
	}
	idOf := func(out string) string { return strings.Fields(out)[1][:13] }

	out, err := call("list_tasks", nil)
	if err != nil || out != "No tasks. Use create_task to add some." {
		t.Errorf("empty list = %q, %v", out, err)
	}

	out, err = call("create_task", map[string]interface{}{"subject": "schema", "description": "tables"})
	if err != nil || !strings.HasPrefix(out, "Created task_") || !strings.HasSuffix(out, ": schema") {
		t.Fatalf("create = %q, %v", out, err)
	}
	schema := idOf(out)
	out, _ = call("create_task", map[string]interface{}{"subject": "api"})
	api := idOf(out)
	if _, err := call("create_task", map[string]interface{}{}); err == nil {
		t.Error("create without subject should fail")
	}

	out, err = call("update_task", map[string]interface{}{"task_id": api, "addBlockedBy": []interface{}{schema}})
	if err != nil || out != "Updated "+api+" blockedBy: "+schema {
		t.Errorf("update = %q, %v", out, err)
	}
	if _, err := call("update_task", map[string]interface{}{"task_id": api, "addBlockedBy": []interface{}{}}); err == nil {
		t.Error("empty addBlockedBy should fail")
	}

	out, _ = call("list_tasks", nil)
	for _, want := range []string{"[ ] " + schema + ": schema [pending]", "[ ] " + api + ": api [pending] (blockedBy: " + schema + ")"} {
		if !strings.Contains(out, want) {
			t.Errorf("list missing %q:\n%s", want, out)
		}
	}

	out, err = call("get_task", map[string]interface{}{"task_id": schema})
	var got Task
	if err != nil || json.Unmarshal([]byte(out), &got) != nil || got.Description != "tables" {
		t.Errorf("get = %q, %v", out, err)
	}

	if _, err := call("claim_task", map[string]interface{}{"task_id": api}); err == nil {
		t.Error("claiming a blocked task should fail")
	}
	if out, err := call("claim_task", map[string]interface{}{"task_id": schema}); err != nil || out != "Claimed "+schema+" (schema)" {
		t.Errorf("claim = %q, %v", out, err)
	}
	out, _ = call("list_tasks", nil)
	if !strings.Contains(out, "[>] "+schema+": schema [in_progress] [main]") {
		t.Errorf("list after claim:\n%s", out)
	}

	// another agent may not complete main's task
	if _, err := a.toolIndex["complete_task"].Handler(withAgentName(context.Background(), "subagent"), map[string]interface{}{"task_id": schema}); err == nil {
		t.Error("wrong owner should fail")
	}
	out, err = call("complete_task", map[string]interface{}{"task_id": schema})
	if err != nil || out != "Completed "+schema+" (schema)\nUnblocked: api" {
		t.Errorf("complete = %q, %v", out, err)
	}
	out, err = call("claim_task", map[string]interface{}{"task_id": api})
	if err != nil || out != "Claimed "+api+" (api)" {
		t.Errorf("claim unblocked = %q, %v", out, err)
	}
	if out, err := call("complete_task", map[string]interface{}{"task_id": api}); err != nil || out != "Completed "+api+" (api)" {
		t.Errorf("complete last = %q, %v", out, err)
	}
}

func TestTaskTools_SurviveAcrossAgents(t *testing.T) {
	useTasksDir(t)
	ctx := withAgentName(context.Background(), "main")
	first := NewAgent(nil, nil).(*agent)
	out, err := first.toolIndex["create_task"].Handler(ctx, map[string]interface{}{"subject": "persisted"})
	if err != nil {
		t.Fatal(err)
	}
	id := strings.Fields(out)[1][:13]

	// a fresh agent (a later session) sees the same task
	second := NewAgent(nil, nil).(*agent)
	list, err := second.toolIndex["list_tasks"].Handler(ctx, nil)
	if err != nil || !strings.Contains(list, id+": persisted [pending]") {
		t.Errorf("second session list = %q, %v", list, err)
	}
}
