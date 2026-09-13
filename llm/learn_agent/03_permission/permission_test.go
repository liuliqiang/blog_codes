package agentloop

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsInWorkspace(t *testing.T) {
	ws := t.TempDir()
	outside := t.TempDir()
	os.Mkdir(filepath.Join(ws, "sub"), 0o755)
	os.WriteFile(filepath.Join(ws, "a.txt"), nil, 0o644)
	os.WriteFile(filepath.Join(outside, "secret.txt"), nil, 0o644)
	// symlink inside the workspace that escapes it
	os.Symlink(outside, filepath.Join(ws, "escape"))
	t.Chdir(ws)

	cases := []struct {
		path string
		want bool
	}{
		{"a.txt", true},
		{"./a.txt", true},
		{"sub/new.txt", true},  // not yet existing, still inside
		{"sub/../a.txt", true}, // cleans back inside
		{".", true},
		{ws, true},
		{filepath.Join(ws, "sub", "x"), true},
		{"..", false},
		{"../x.txt", false},
		{"sub/../../x.txt", false},
		{"/etc/passwd", false},
		{filepath.Join(outside, "secret.txt"), false},
		{"escape/secret.txt", false},   // symlink pointing outside
		{"escape/new.txt", false},      // new file under escaping symlink
		{ws + "-sibling/a.txt", false}, // shares prefix string, different dir
	}
	for _, c := range cases {
		if got := isInWorkspace(c.path); got != c.want {
			t.Errorf("isInWorkspace(%q) = %v, want %v", c.path, got, c.want)
		}
	}
}

func TestIsCommandAllowed(t *testing.T) {
	cases := []struct {
		command string
		want    bool
	}{
		{"ls -la", true},
		{"echo hello > out.txt", true},
		{"chmod 755 hello.sh", true},
		{"cat /etc/hosts", true},        // reading /etc is fine, writing is not
		{"echo warm", true},             // "rm" without trailing space
		{"grep rm file.txt", false},     // "rm " matches even as a grep argument
		{"sudo", false},                 // exact deny list
		{"rm -rf /", false},             // exact deny list
		{"rm file.txt", false},          // "rm " fragment
		{"cd /tmp && rm -f x", false},   // fragment buried in a compound command
		{"echo x > /etc/hosts", false},  // "> /etc/" fragment
		{"chmod 777 hello.sh", false},   // "chmod 777" fragment
		{"ls | xargs chmod 777", false}, // fragment after a pipe
	}
	for _, c := range cases {
		if got := isCommandAllowed(c.command); got != c.want {
			t.Errorf("isCommandAllowed(%q) = %v, want %v", c.command, got, c.want)
		}
	}
}

func TestCheckRules(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	a := &agent{}

	cases := []struct {
		name  string
		tool  MessagesBlock
		allow bool
	}{
		{"read inside", MessagesBlock{Name: "read_file", Input: map[string]interface{}{"path": "a.txt"}}, true},
		{"write outside", MessagesBlock{Name: "write_file", Input: map[string]interface{}{"path": "/etc/passwd", "content": "x"}}, false},
		{"edit missing path", MessagesBlock{Name: "edit_file", Input: map[string]interface{}{}}, false},
		{"path not a string", MessagesBlock{Name: "read_file", Input: map[string]interface{}{"path": 42}}, false},
		{"safe command", MessagesBlock{Name: "run_bash", Input: map[string]interface{}{"command": "ls"}}, true},
		{"denied command", MessagesBlock{Name: "run_bash", Input: map[string]interface{}{"command": "rm -rf ."}}, false},
		{"command missing", MessagesBlock{Name: "run_bash", Input: map[string]interface{}{}}, false},
		{"unruled tool", MessagesBlock{Name: "list_directory", Input: map[string]interface{}{"path": "/"}}, true},
	}
	for _, c := range cases {
		allow, msg := a.checkRules(c.tool)
		if allow != c.allow {
			t.Errorf("%s: allow = %v, want %v", c.name, allow, c.allow)
		}
		if allow && msg != "" || !allow && msg == "" {
			t.Errorf("%s: message = %q for allow=%v", c.name, msg, allow)
		}
	}
}

func TestCheckUserAllow(t *testing.T) {
	origIn, origOut := promptIn, promptOut
	t.Cleanup(func() { promptIn, promptOut = origIn, origOut })

	tool := MessagesBlock{Name: "run_bash", Input: map[string]interface{}{"command": "ls"}}
	cases := []struct {
		answer string
		want   bool
	}{
		{"y\n", true},
		{"Y\n", true},
		{"yes\n", true},
		{"  yes  \n", true},
		{"n\n", false},
		{"no\n", false},
		{"\n", false}, // enter = default deny
		{"", false},   // EOF = deny
		{"whatever\n", false},
	}
	for _, c := range cases {
		var out bytes.Buffer
		promptIn, promptOut = strings.NewReader(c.answer), &out
		if got := (&agent{}).checkUserAllow(tool); got != c.want {
			t.Errorf("answer %q: got %v, want %v", c.answer, got, c.want)
		}
		if !strings.Contains(out.String(), `"run_bash"`) || !strings.Contains(out.String(), `"command": "ls"`) || !strings.Contains(out.String(), "[y/N]") {
			t.Errorf("prompt missing tool details: %q", out.String())
		}
	}
}

// yesReader answers "y" to every prompt. Each Read yields exactly one line so a
// fresh bufio.Reader per prompt never swallows the answers meant for later ones.
type yesReader struct{}

func (yesReader) Read(p []byte) (int, error) { return copy(p, "y\n"), nil }

// autoAllow makes checkUserAllow approve everything for the duration of the test.
func autoAllow(t *testing.T) {
	t.Helper()
	origIn, origOut := promptIn, promptOut
	promptIn, promptOut = yesReader{}, io.Discard
	t.Cleanup(func() { promptIn, promptOut = origIn, origOut })
}

func TestCheckToolPermission(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	a := &agent{}
	origIn, origOut := promptIn, promptOut
	t.Cleanup(func() { promptIn, promptOut = origIn, origOut })
	promptOut = io.Discard

	// rule denies before the user is even asked
	promptIn = yesReader{}
	allowed, msg := a.checkToolPermission(MessagesBlock{Name: "run_bash", Input: map[string]interface{}{"command": "rm -rf ."}})
	if allowed || !strings.Contains(msg, "not allowed") {
		t.Errorf("rule denial: allowed=%v msg=%q", allowed, msg)
	}

	// rule passes, user says no
	promptIn = strings.NewReader("n\n")
	allowed, msg = a.checkToolPermission(MessagesBlock{Name: "run_bash", Input: map[string]interface{}{"command": "ls"}})
	if allowed || msg != "User denied permission to run the tool." {
		t.Errorf("user denial: allowed=%v msg=%q", allowed, msg)
	}

	// rule passes, user says yes
	promptIn = yesReader{}
	allowed, msg = a.checkToolPermission(MessagesBlock{Name: "run_bash", Input: map[string]interface{}{"command": "ls"}})
	if !allowed || msg != "" {
		t.Errorf("approved: allowed=%v msg=%q", allowed, msg)
	}
}
