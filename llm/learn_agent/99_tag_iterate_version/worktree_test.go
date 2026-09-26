package agentloop

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// useWorktrees points the worktree directory at the current workspace.
func useWorktrees(t *testing.T, dir string) {
	t.Helper()
	orig := worktreesDir
	worktreesDir = dir
	t.Cleanup(func() { worktreesDir = orig })
}

// gitRepo makes a temp repository with one commit and chdirs into it.
func gitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q", "-b", "main"},
		{"config", "user.email", "t@example.com"},
		{"config", "user.name", "t"},
	} {
		if out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
			t.Skipf("git unavailable: %v %s", err, out)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("hi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "."}, {"commit", "-qm", "first"}} {
		if out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	t.Chdir(dir)
	return dir
}

func TestWorktreePath(t *testing.T) {
	useWorktrees(t, ".worktrees")
	for _, bad := range []string{"", ".", "..", "a/b", "../x", "-dash", strings.Repeat("x", 65)} {
		if _, err := worktreePath(bad); err == nil {
			t.Errorf("worktreePath(%q) should fail", bad)
		}
	}
	if got, err := worktreePath("auth-refactor"); err != nil || got != filepath.Join(".worktrees", "auth-refactor") {
		t.Errorf("worktreePath = %q, %v", got, err)
	}
}

func TestCreateWorktree(t *testing.T) {
	repo := gitRepo(t)
	useTasksDir(t)
	useWorktrees(t, filepath.Join(repo, ".worktrees"))
	store := NewTaskStore(tasksDir)
	ctx := context.Background()

	task, err := store.Create("refactor auth", "")
	if err != nil {
		t.Fatal(err)
	}
	bound, err := store.CreateWorktree(ctx, "auth", task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if bound.Worktree != "auth" {
		t.Errorf("binding = %+v", bound)
	}
	path := filepath.Join(worktreesDir, "auth")
	if info, err := os.Stat(filepath.Join(path, "README.md")); err != nil || info.IsDir() {
		t.Errorf("worktree not checked out: %v", err)
	}
	branches, _ := exec.Command("git", "-C", repo, "branch", "--list", "wt/auth").CombinedOutput()
	if !strings.Contains(string(branches), "wt/auth") {
		t.Errorf("branch wt/auth missing: %s", branches)
	}
	// the binding survives a reload
	if reloaded, err := store.Load(task.ID); err != nil || reloaded.Worktree != "auth" {
		t.Errorf("reloaded = %+v, %v", reloaded, err)
	}

	// a bound, claimed or missing task is refused, and so is a duplicate name
	if _, err := store.CreateWorktree(ctx, "auth2", task.ID); err == nil || !strings.Contains(err.Error(), "already bound") {
		t.Errorf("rebinding = %v", err)
	}
	other, _ := store.Create("other", "")
	if _, err := store.CreateWorktree(ctx, "auth", other.ID); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Errorf("duplicate name = %v", err)
	}
	if _, err := store.Claim(other.ID, "alice"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateWorktree(ctx, "other", other.ID); err == nil || !strings.Contains(err.Error(), "pending and unowned") {
		t.Errorf("claimed task = %v", err)
	}
	if _, err := store.CreateWorktree(ctx, "missing", "task_00000000"); err == nil {
		t.Error("unknown task should fail")
	}
}

func TestTaskWorktreeCwdAndClaim(t *testing.T) {
	repo := gitRepo(t)
	useTasksDir(t)
	useWorktrees(t, filepath.Join(repo, ".worktrees"))
	store := NewTaskStore(tasksDir)
	ctx := context.Background()

	plain, _ := store.Create("plain", "")
	if dir, err := taskWorktreeCwd(plain); err != nil || dir != repo {
		t.Errorf("unbound task should use the repository: %q, %v (want %q)", dir, err, repo)
	}

	task, _ := store.Create("bound", "")
	if _, err := store.CreateWorktree(ctx, "auth", task.ID); err != nil {
		t.Fatal(err)
	}
	bound, _ := store.Load(task.ID)
	dir, err := taskWorktreeCwd(bound)
	if err != nil || filepath.Base(dir) != "auth" {
		t.Fatalf("bound cwd = %q, %v", dir, err)
	}
	if _, err := store.Claim(task.ID, "alice"); err != nil {
		t.Fatal(err)
	}

	// a binding that no longer resolves fails closed instead of using the repository
	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := taskWorktreeCwd(bound); err == nil || !strings.Contains(err.Error(), "is missing") {
		t.Errorf("missing worktree = %v", err)
	}
	broken, _ := store.Create("broken", "")
	broken.Worktree = "gone"
	if err := store.Save(broken); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Claim(broken.ID, "bob"); err == nil || !strings.Contains(err.Error(), "cannot claim") {
		t.Errorf("claiming a task with a broken binding = %v", err)
	}
}

func TestRebaseToDir(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		name  string
		input map[string]interface{}
		check func(t *testing.T, out map[string]interface{}, err error)
	}{
		{"bash prefixes cd", map[string]interface{}{"command": "go test ./..."}, func(t *testing.T, out map[string]interface{}, err error) {
			if err != nil || out["command"] != "cd '"+dir+"' && go test ./..." {
				t.Errorf("command = %v, %v", out["command"], err)
			}
		}},
		{"relative path is anchored", map[string]interface{}{"path": "pkg/a.go"}, func(t *testing.T, out map[string]interface{}, err error) {
			if err != nil || out["path"] != filepath.Join(dir, "pkg/a.go") {
				t.Errorf("path = %v, %v", out["path"], err)
			}
		}},
		{"escape is refused", map[string]interface{}{"path": "../secrets"}, func(t *testing.T, _ map[string]interface{}, err error) {
			if err == nil || !strings.Contains(err.Error(), "outside the assigned directory") {
				t.Errorf("err = %v", err)
			}
		}},
		{"absolute escape is refused", map[string]interface{}{"path": "/etc/passwd"}, func(t *testing.T, _ map[string]interface{}, err error) {
			if err == nil {
				t.Error("want error")
			}
		}},
		{"missing path", map[string]interface{}{}, func(t *testing.T, _ map[string]interface{}, err error) {
			if err == nil {
				t.Error("want error")
			}
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			name := "read_file"
			if _, ok := c.input["command"]; ok {
				name = "run_bash"
			}
			out, err := rebaseToDir(name, c.input, dir)
			c.check(t, out, err)
		})
	}
	// grep_file and find_file default to the assignment itself
	for _, name := range []string{"grep_file", "find_file"} {
		out, err := rebaseToDir(name, map[string]interface{}{"pattern": "x"}, dir)
		if err != nil || out["path"] != dir {
			t.Errorf("%s = %v, %v", name, out["path"], err)
		}
	}
	// the original input is not mutated
	input := map[string]interface{}{"path": "a.go"}
	if _, err := rebaseToDir("read_file", input, dir); err != nil || input["path"] != "a.go" {
		t.Errorf("input mutated: %v", input)
	}
}

func TestTeammateToolsUseTheirWorktree(t *testing.T) {
	repo := gitRepo(t)
	useTasksDir(t)
	useWorktrees(t, filepath.Join(repo, ".worktrees"))
	useTeam(t, testTeamConfig())
	autoAllow(t)
	lead := NewAgent(newTeamLLM(), nil).(*agent)
	defer lead.team.Shutdown()
	ctx := withAgentName(context.Background(), "alice")

	task, _ := lead.tasks.Create("work in a worktree", "")
	if _, err := lead.tasks.CreateWorktree(context.Background(), "auth", task.ID); err != nil {
		t.Fatal(err)
	}
	mate, err := lead.team.Spawn("alice", "auth", task.ID, false)
	if err != nil {
		t.Fatal(err)
	}

	write := mate.agent.toolIndex["write_file"]
	if _, err := write.Handler(ctx, map[string]interface{}{"path": "new.txt", "content": "from alice"}); err != nil {
		t.Fatal(err)
	}
	// the file lands in the worktree, not the repository
	if _, err := os.Stat(filepath.Join(repo, ".worktrees", "auth", "new.txt")); err != nil {
		t.Errorf("file not written into the worktree: %v", err)
	}
	if _, err := os.Stat(filepath.Join(repo, "new.txt")); err == nil {
		t.Error("file leaked into the repository")
	}
	// bash runs there too
	out, err := mate.agent.toolIndex["run_bash"].Handler(ctx, map[string]interface{}{"command": "pwd"})
	if err != nil || !strings.Contains(out, filepath.Join(".worktrees", "auth")) {
		t.Errorf("pwd = %q, %v", out, err)
	}
	// and paths cannot escape it
	if _, err := write.Handler(ctx, map[string]interface{}{"path": "../../escaped.txt", "content": "x"}); err == nil {
		t.Error("escaping the worktree should fail")
	}

	// a teammate with no task has no directory to work in
	free, err := lead.team.Spawn("bob", "", "", false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := free.agent.toolIndex["read_file"].Handler(withAgentName(context.Background(), "bob"), map[string]interface{}{"path": "README.md"}); err == nil || !strings.Contains(err.Error(), "claim a task") {
		t.Errorf("unassigned teammate = %v", err)
	}
	// the lead is unaffected: it works in the repository
	if _, err := lead.toolIndex["read_file"].Handler(context.Background(), map[string]interface{}{"path": "README.md"}); err != nil {
		t.Errorf("lead read_file = %v", err)
	}
}

func TestRemoveWorktree(t *testing.T) {
	repo := gitRepo(t)
	useTasksDir(t)
	useWorktrees(t, filepath.Join(repo, ".worktrees"))
	store := NewTaskStore(tasksDir)
	ctx := context.Background()

	task, _ := store.Create("bound", "")
	if _, err := store.CreateWorktree(ctx, "auth", task.ID); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(worktreesDir, "auth")

	// a task still pending in that worktree blocks removal
	if err := RemoveWorktree(ctx, store, "auth", false); err == nil || !strings.Contains(err.Error(), "finish or release it first") {
		t.Errorf("pending task = %v", err)
	}
	if _, err := store.Claim(task.ID, "alice"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.Complete(task.ID, "alice"); err != nil {
		t.Fatal(err)
	}

	// local changes block removal unless the caller says to discard them
	if err := os.WriteFile(filepath.Join(path, "dirty.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := RemoveWorktree(ctx, store, "auth", false); err == nil || !strings.Contains(err.Error(), "local changes") {
		t.Errorf("dirty worktree = %v", err)
	}
	if err := RemoveWorktree(ctx, store, "auth", true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("worktree directory still there")
	}
	// the branch is kept and the binding is cleared
	branches, _ := exec.Command("git", "-C", repo, "branch", "--list", "wt/auth").CombinedOutput()
	if !strings.Contains(string(branches), "wt/auth") {
		t.Errorf("branch should be kept: %s", branches)
	}
	if reloaded, err := store.Load(task.ID); err != nil || reloaded.Worktree != "" {
		t.Errorf("binding not cleared: %+v, %v", reloaded, err)
	}
	if err := RemoveWorktree(ctx, store, "../escape", false); err == nil {
		t.Error("invalid name should fail")
	}
}

func TestCreateWorktreeTool(t *testing.T) {
	repo := gitRepo(t)
	useTasksDir(t)
	useWorktrees(t, filepath.Join(repo, ".worktrees"))
	useTeam(t, testTeamConfig())
	lead := NewAgent(newTeamLLM(), nil).(*agent)
	defer lead.team.Shutdown()
	ctx := withAgent(withAgentName(context.Background(), "main"), lead)

	task, _ := lead.tasks.Create("auth", "")
	out, err := lead.toolIndex["create_worktree"].Handler(ctx, map[string]interface{}{"name": "auth", "task_id": task.ID})
	if err != nil || !strings.Contains(out, "branch wt/auth") || !strings.Contains(out, task.ID) {
		t.Fatalf("create_worktree = %q, %v", out, err)
	}
	// it creates a branch and a checkout, so unlike the read-only team tools it is not allow-listed: the user is asked
	if isAllowListed(MessagesBlock{Name: "create_worktree"}) {
		t.Error("create_worktree changes the repository and should need approval")
	}
	if _, err := lead.toolIndex["create_worktree"].Handler(ctx, map[string]interface{}{"name": "x"}); err == nil {
		t.Error("missing task_id should fail")
	}
	// teammates cannot create or remove worktrees
	mate, err := lead.team.Spawn("alice", "", "", false)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := mate.agent.toolIndex["create_worktree"]; ok {
		t.Error("teammates must not create worktrees")
	}
	for _, name := range []string{"remove_worktree", "request_plan", "review_plan"} {
		if _, ok := mate.agent.toolIndex[name]; ok {
			t.Errorf("teammates must not have %s", name)
		}
	}
}
