package agentloop

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/liuliqiang/log4go"
)

var (
	denyList = map[string]bool{
		"rm -rf /":           true,
		"sudo rm -rf /":      true,
		"sudo rm -rf *":      true,
		"sudo rm -rf .":      true,
		"sudo rm -rf ..":     true,
		"sudo":               true,
		"shutdown":           true,
		"reboot":             true,
		"mkfs":               true,
		"dd if=":             true,
		"dd of=":             true,
		"format":             true,
		"kill":               true,
		"killall":            true,
		"pkill":              true,
		"halt":               true,
		"poweroff":           true,
		"init 0":             true,
		"init 6":             true,
		"telinit 0":          true,
		"telinit 6":          true,
		"systemctl poweroff": true,
		"systemctl reboot":   true,
	}

	// allowList holds tool names and bash programs that run without rule
	// checks or a user prompt. A bash command matches on its first word and
	// only when it is a single plain command (no ; && | > $ etc.), so
	// "ls -la" is allowed while "ls; rm -rf /" is not.
	allowList = map[string]bool{
		"ls":         true,
		"pwd":        true,
		"whoami":     true,
		"todo_write": true,
		"task":       true,
	}

	// denySubstrings rejects any command containing one of these fragments,
	// regardless of what surrounds it (pipes, &&, subshells, ...).
	denySubstrings = []string{"rm ", "> /etc/", "chmod 777"}

	permissionRules = []struct {
		Tools map[string]bool
		// Check returns true if the command is allowed, false otherwise.
		Check   func(map[string]interface{}) bool
		Message string
	}{
		{
			Tools: map[string]bool{
				"read_file":  true,
				"write_file": true,
				"edit_file":  true,
			},
			// only allow reading/writing files in the WORKSPACE directory
			Check:   checkPathInWorkspace,
			Message: "You are only allowed to read/write files in the WORKSPACE directory. Please provide a path that is within the current working directory.",
		},
		{
			Tools: map[string]bool{
				"run_bash": true,
			},
			// only allow running bash commands that are not in the deny list
			Check:   checkCommandAllowed,
			Message: "The command you are trying to run is not allowed. Please provide a different command.",
		},
	}
)

func checkRules(tool MessagesBlock) (bool, string) {
	for _, rule := range permissionRules {
		if rule.Tools[tool.Name] {
			if !rule.Check(tool.Input) {
				return false, rule.Message
			}
		}
	}

	return true, ""
}

// promptIn / promptOut are where checkUserAllow talks to the user; tests swap them.
var (
	promptIn  io.Reader = os.Stdin
	promptOut io.Writer = os.Stdout
)

// checkUserAllow asks the user on the terminal whether the tool call may run.
// Only an explicit "y" / "yes" allows it; anything else, including EOF, denies.
func checkUserAllow(ctx context.Context, tool MessagesBlock) bool {
	input, err := json.MarshalIndent(tool.Input, "  ", "  ")
	if err != nil {
		log4go.DefaultLogger().Error(ctx, "marshal tool input failed: %v, input: %+v", err, tool.Input)
		input = []byte(fmt.Sprintf("%+v", tool.Input))
	}
	fmt.Fprintf(promptOut, "\n[%s] wants to run tool %q with input:\n  %s\nAllow? [y/N] ", agentNameFrom(ctx), tool.Name, input)

	answer, err := bufio.NewReader(promptIn).ReadString('\n')
	if err != nil && err != io.EOF {
		log4go.DefaultLogger().Error(ctx, "read user answer failed: %v", err)
		return false
	}
	switch strings.ToLower(strings.TrimSpace(answer)) {
	case "y", "yes":
		return true
	default:
		return false
	}
}

// checkPathInWorkspace applies isInWorkspace to the tool's "path" argument.
// A missing or non-string path is denied rather than let through unchecked.
func checkPathInWorkspace(input map[string]interface{}) bool {
	path, ok := input["path"].(string)
	if !ok {
		log4go.DefaultLogger().Error(context.Background(), "permission check: missing or non-string path, input: %+v", input)
		return false
	}
	return isInWorkspace(path)
}

// checkCommandAllowed applies isCommandAllowed to the tool's "command" argument.
func checkCommandAllowed(input map[string]interface{}) bool {
	command, ok := input["command"].(string)
	if !ok {
		log4go.DefaultLogger().Error(context.Background(), "permission check: missing or non-string command, input: %+v", input)
		return false
	}
	return isCommandAllowed(command)
}

// isInWorkspace reports whether path resolves to a location inside the
// current working directory. Relative paths are resolved against it, so
// "a.txt" and "./sub/b.txt" pass while "../x", "/etc/passwd" and symlinks
// pointing outside the workspace do not.
func isInWorkspace(path string) bool {
	ctx := context.Background()

	cwd, err := os.Getwd()
	if err != nil {
		log4go.DefaultLogger().Error(ctx, "get working directory failed: %v", err)
		return false
	}
	cwd, err = filepath.EvalSymlinks(cwd)
	if err != nil {
		log4go.DefaultLogger().Error(ctx, "resolve working directory failed: %v, cwd: %s", err, cwd)
		return false
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		log4go.DefaultLogger().Error(ctx, "resolve path failed: %v, path: %s", err, path)
		return false
	}
	// The target may not exist yet (write_file), so resolve symlinks on the
	// deepest existing ancestor and re-attach the rest.
	abs = resolveExisting(abs)

	rel, err := filepath.Rel(cwd, abs)
	if err != nil {
		log4go.DefaultLogger().Error(ctx, "relative path failed: %v, cwd: %s, path: %s", err, cwd, abs)
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func resolveExisting(abs string) string {
	var tail []string
	for {
		resolved, err := filepath.EvalSymlinks(abs)
		if err == nil {
			return filepath.Join(append([]string{resolved}, tail...)...)
		}
		parent, base := filepath.Split(filepath.Clean(abs))
		if parent == abs || base == "" {
			return abs
		}
		tail = append([]string{base}, tail...)
		abs = filepath.Clean(parent)
	}
}

// isCommandAllowed rejects commands that exactly match the deny list or
// contain any of the denied substrings.
func isCommandAllowed(command string) bool {
	if denyList[command] {
		return false
	}
	for _, fragment := range denySubstrings {
		if strings.Contains(command, fragment) {
			return false
		}
	}
	return true
}

func checkToolPermission(ctx context.Context, tool MessagesBlock) (bool, string) {
	if isAllowListed(tool) {
		return true, ""
	}

	allowed, message := checkRules(tool)
	if !allowed {
		return false, message
	}

	if !checkUserAllow(ctx, tool) {
		return false, "User denied permission to run the tool."
	}

	return true, ""
}

// shellControlChars are the characters that let one command string run more
// than one program; a command containing any of them never matches allowList.
const shellControlChars = ";&|<>$`()\n"

// isAllowListed reports whether the tool itself, or for run_bash the program
// it invokes, is on allowList.
func isAllowListed(tool MessagesBlock) bool {
	if allowList[tool.Name] {
		return true
	}
	if tool.Name != "run_bash" {
		return false
	}
	command, ok := tool.Input["command"].(string)
	if !ok || strings.ContainsAny(command, shellControlChars) {
		return false
	}
	fields := strings.Fields(command)
	return len(fields) > 0 && allowList[fields[0]]
}
