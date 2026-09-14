package agentloop

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
)

// hookOut is where the built-in hooks print; tests swap it.
var hookOut io.Writer = os.Stdout

// PermissionHook enforces permissionRules and then asks the user on the
// terminal. Register it after any hook that modifies tool input so it
// checks what will actually run.
func PermissionHook() PreToolUseHook {
	return func(ctx context.Context, toolUse MessagesBlock) (MessagesBlock, error) {
		if allowed, message := checkToolPermission(ctx, toolUse); !allowed {
			return toolUse, errors.New(message)
		}
		return toolUse, nil
	}
}

// LogToolUseHook prints every tool call before it runs.
func LogToolUseHook() PreToolUseHook {
	return func(_ context.Context, toolUse MessagesBlock) (MessagesBlock, error) {
		fmt.Fprintf(hookOut, "[HOOK] %s(...)\n", toolUse.Name)
		return toolUse, nil
	}
}

// largeOutputThreshold is the output size (bytes) above which
// LargeOutputHook warns.
const largeOutputThreshold = 100_000

// LargeOutputHook warns when a tool returns an unusually large output. It
// never changes the output.
func LargeOutputHook() PostToolUseHook {
	return func(_ context.Context, toolUse MessagesBlock, output string) (string, error) {
		if len(output) > largeOutputThreshold {
			fmt.Fprintf(hookOut, "[HOOK] \u26a0 Large output from %s (%d bytes)\n", toolUse.Name, len(output))
		}
		return output, nil
	}
}

// SummaryHook prints how many tool calls the session used when the loop is
// about to stop. It always allows the stop.
func SummaryHook() StopHook {
	return func(_ context.Context, messages []Message) (*Message, error) {
		toolCount := 0
		for _, m := range messages {
			blocks, ok := m.Content.([]MessagesBlock)
			if !ok {
				continue
			}
			for _, b := range blocks {
				if b.Type == MessagesBlockTypeToolResult {
					toolCount++
				}
			}
		}
		fmt.Fprintf(hookOut, "\033[90m[HOOK] Stop: session used %d tool calls\033[0m\n", toolCount)
		return nil, nil
	}
}
