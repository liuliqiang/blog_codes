package agentloop

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/liuliqiang/log4go"
)

// CompactionConfig decides when the conversation is compacted and how much
// of it survives verbatim. ContextWindow <= 0 disables compaction.
type CompactionConfig struct {
	ContextWindow    int // model context window, in tokens
	ReserveTokens    int // headroom kept free below the window
	KeepRecentTokens int // newest messages that are never summarized
}

// compaction is the active config; tests swap it.
var compaction = CompactionConfig{
	ContextWindow:    128_000,
	ReserveTokens:    16_384,
	KeepRecentTokens: 20_000,
}

const (
	summaryOpenTag  = "<summary>\n"
	summaryCloseTag = "\n</summary>"

	// toolResultExcerpt bounds how much of each tool result reaches the
	// summarizer; the summary never needs the full output.
	toolResultExcerpt = 2000
)

const summarizeSystemPrompt = `You are compacting the context of a coding agent. Summarize the conversation below so the agent can continue the task without the original messages. Be specific: keep file paths, commands, error messages, decisions and open questions. Use exactly this structure:

## Goal
## Constraints & Preferences
## Progress
### Done
### In Progress
### Blocked
## Key Decisions
## Next Steps
## Critical Context

Output only the summary.`

const summarizeUpdateInstruction = `A summary of the earlier part of the conversation already exists. Update it with the new messages below instead of rewriting it from scratch; keep everything from it that is still relevant.`

// shouldCompact reports whether the last response left the context above the
// compaction line.
func (a *agent) shouldCompact() bool {
	if compaction.ContextWindow <= 0 || a.lastUsage == (Usage{}) {
		return false
	}
	return a.lastUsage.InputTokens+a.lastUsage.OutputTokens > compaction.ContextWindow-compaction.ReserveTokens
}

// compact replaces the older part of a.messages with a summary. Failures,
// including a Compact hook returning an error, are logged and leave the
// messages untouched so the loop can carry on.
func (a *agent) compact(ctx context.Context) {
	start := 0
	if a.summary != "" {
		start = 1 // messages[0] is the previous summary
	}
	cut := findCutPoint(a.messages[start:], compaction.KeepRecentTokens) + start
	if cut <= start {
		return
	}
	span := a.messages[start:cut]

	summary, err := a.summarize(ctx, span)
	if err != nil {
		log4go.DefaultLogger().Error(ctx, "[%s] compaction failed, keeping %d messages: %v", a.name, len(a.messages), err)
		return
	}
	summary, err = a.runCompactHooks(ctx, span, a.messages[cut:], summary)
	if err != nil {
		log4go.DefaultLogger().Error(ctx, "[%s] compaction cancelled by hook, keeping %d messages: %v", a.name, len(a.messages), err)
		return
	}
	a.collectFiles(span)
	a.summary = summary + a.renderFiles()

	a.messages = append([]Message{summaryMessage(a.summary)}, a.messages[cut:]...)
}

// findCutPoint returns cut such that messages[:cut] get summarized and
// messages[cut:] stay verbatim. It walks from the newest message backwards
// until keepRecentTokens are accumulated, then moves further back past any
// tool-result message so a kept result is never separated from the assistant
// message that requested it. 0 means nothing should be compacted.
func findCutPoint(messages []Message, keepRecentTokens int) int {
	cut := 0
	total := 0
	for i := len(messages) - 1; i >= 0; i-- {
		total += estimateTokens(messages[i])
		if total >= keepRecentTokens {
			cut = i
			break
		}
	}
	for cut > 0 && isToolResultMessage(messages[cut]) {
		cut--
	}
	return cut
}

func isToolResultMessage(msg Message) bool {
	blocks, ok := msg.Content.([]MessagesBlock)
	if !ok {
		return false
	}
	for _, b := range blocks {
		if b.Type == MessagesBlockTypeToolResult {
			return true
		}
	}
	return false
}

// estimateTokens is a rough chars/4 count over everything in the messages;
// it only steers the cut point, the trigger uses the model's real usage.
func estimateTokens(messages ...Message) int {
	chars := 0
	for _, msg := range messages {
		switch content := msg.Content.(type) {
		case string:
			chars += len(content)
		case []MessagesBlock:
			for _, b := range content {
				chars += len(b.Text) + len(b.Content)
				if b.Input != nil {
					input, _ := json.Marshal(b.Input)
					chars += len(input)
				}
			}
		}
	}
	return (chars + 3) / 4
}

// summarize asks the model for a summary of span, updating a.summary when
// there is one. Tools and hooks are deliberately bypassed.
func (a *agent) summarize(ctx context.Context, span []Message) (string, error) {
	var b strings.Builder
	if a.summary != "" {
		b.WriteString(summarizeUpdateInstruction)
		b.WriteString("\n\n<previous-summary>\n")
		b.WriteString(a.summary)
		b.WriteString("\n</previous-summary>\n\n")
	}
	b.WriteString("<conversation>\n")
	b.WriteString(renderConversation(span))
	b.WriteString("</conversation>")

	resp, err := a.llmClient.SendMessages(
		ctx,
		a.llmClient.GetModel(),
		Message{Role: MessageRoleSystem, Content: summarizeSystemPrompt},
		[]Message{{Role: MessageRoleUser, Content: b.String()}},
		nil,
		NewSendMessagesOpts())
	if err != nil {
		log4go.DefaultLogger().Error(ctx, "[%s] summarize request failed: %v, span: %d messages", a.name, err, len(span))
		return "", err
	}
	var parts []string
	for _, block := range resp.GetContent() {
		if block.Type == MessagesBlockTypeText && block.Text != "" {
			parts = append(parts, block.Text)
		}
	}
	summary := strings.TrimSpace(strings.Join(parts, "\n"))
	if summary == "" {
		log4go.DefaultLogger().Error(ctx, "[%s] summarize returned no text, response: %+v", a.name, resp)
		return "", fmt.Errorf("summarize returned no text")
	}
	return summary, nil
}

// renderConversation flattens messages into the plain-text transcript the
// summarizer reads.
func renderConversation(messages []Message) string {
	var b strings.Builder
	for _, msg := range messages {
		switch content := msg.Content.(type) {
		case string:
			fmt.Fprintf(&b, "%s: %s\n", msg.Role, content)
		case []MessagesBlock:
			for _, block := range content {
				switch block.Type {
				case MessagesBlockTypeText:
					fmt.Fprintf(&b, "%s: %s\n", msg.Role, block.Text)
				case MessagesBlockTypeToolUse:
					input, _ := json.Marshal(block.Input)
					fmt.Fprintf(&b, "%s: [tool_use %s %s %s]\n", msg.Role, block.ID, block.Name, input)
				case MessagesBlockTypeToolResult:
					fmt.Fprintf(&b, "%s: [tool_result %s] %s\n", msg.Role, block.ToolUseID, excerpt(block.Content, toolResultExcerpt))
				}
				// reasoning is the model's scratch work, not worth summarizing
			}
		}
	}
	return b.String()
}

func excerpt(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + fmt.Sprintf("... [%d more bytes]", len(s)-max)
}

// collectFiles records the files the compacted span touched. The lists
// accumulate across compactions: the conversation may be forgotten, which
// files were read or changed may not.
func (a *agent) collectFiles(span []Message) {
	for _, msg := range span {
		blocks, _ := msg.Content.([]MessagesBlock)
		for _, b := range blocks {
			if b.Type != MessagesBlockTypeToolUse {
				continue
			}
			path, _ := b.Input["path"].(string)
			if path == "" {
				continue
			}
			switch b.Name {
			case "read_file":
				a.readFiles[path] = true
			case "write_file", "edit_file":
				a.modifiedFiles[path] = true
			}
		}
	}
}

func (a *agent) renderFiles() string {
	var b strings.Builder
	for _, list := range []struct {
		tag   string
		files map[string]bool
	}{{"read-files", a.readFiles}, {"modified-files", a.modifiedFiles}} {
		if len(list.files) == 0 {
			continue
		}
		names := make([]string, 0, len(list.files))
		for name := range list.files {
			names = append(names, name)
		}
		sort.Strings(names)
		fmt.Fprintf(&b, "\n<%s>\n%s\n</%s>", list.tag, strings.Join(names, "\n"), list.tag)
	}
	return b.String()
}

// summaryMessage wraps the summary as the user message that opens the
// compacted conversation.
func summaryMessage(summary string) Message {
	return Message{Role: MessageRoleUser, Content: summaryOpenTag + summary + summaryCloseTag}
}
