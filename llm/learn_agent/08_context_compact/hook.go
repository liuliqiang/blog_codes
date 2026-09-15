package agentloop

import (
	"context"
	"fmt"

	"github.com/liuliqiang/log4go"
)

// Hook signatures. Each one states exactly what the hook may change.

// UserPromptSubmitHook runs before the first LLM call and may rewrite the
// initial messages. Returning an error rejects the prompt: RunLoop returns it.
type UserPromptSubmitHook func(ctx context.Context, messages []Message) ([]Message, error)

// PreToolUseHook runs before a tool executes and may modify the call.
// Returning an error denies the call; the error text is sent back to the
// model as the tool result and the loop continues.
type PreToolUseHook func(ctx context.Context, toolUse MessagesBlock) (MessagesBlock, error)

// PostToolUseHook runs after a tool executes and may rewrite its output.
// Returning an error replaces the output with "error: ..." for the model.
type PostToolUseHook func(ctx context.Context, toolUse MessagesBlock, output string) (string, error)

// StopHook runs when the model produced no tool calls and the loop is about
// to end. Returning a non-nil message appends it as a user message and keeps
// the loop going; nil lets the loop stop. Returning an error ends RunLoop
// with that error. A hook that always continues will loop forever unless the
// agent has a maxLoop.
type StopHook func(ctx context.Context, messages []Message) (*Message, error)

// Hooks is an append-only registry. Fields are unexported so callers can add
// hooks but never remove or replace ones registered by someone else; the
// zero value is ready to use.
type Hooks struct {
	userPromptSubmit []UserPromptSubmitHook
	preToolUse       []PreToolUseHook
	postToolUse      []PostToolUseHook
	stop             []StopHook
}

func (h *Hooks) OnUserPromptSubmit(fn UserPromptSubmitHook) *Hooks {
	h.userPromptSubmit = append(h.userPromptSubmit, fn)
	return h
}

func (h *Hooks) OnPreToolUse(fn PreToolUseHook) *Hooks {
	h.preToolUse = append(h.preToolUse, fn)
	return h
}

func (h *Hooks) OnPostToolUse(fn PostToolUseHook) *Hooks {
	h.postToolUse = append(h.postToolUse, fn)
	return h
}

func (h *Hooks) OnStop(fn StopHook) *Hooks {
	h.stop = append(h.stop, fn)
	return h
}

// snapshot returns an independent copy so the agent's hook set is frozen at
// construction; later registrations on the caller's Hooks don't leak in.
func (h *Hooks) snapshot() Hooks {
	if h == nil {
		return Hooks{}
	}
	return Hooks{
		userPromptSubmit: append([]UserPromptSubmitHook(nil), h.userPromptSubmit...),
		preToolUse:       append([]PreToolUseHook(nil), h.preToolUse...),
		postToolUse:      append([]PostToolUseHook(nil), h.postToolUse...),
		stop:             append([]StopHook(nil), h.stop...),
	}
}

/* vvvvvvvvvvvvvvvvvvvvv dispatch vvvvvvvvvvvvvvvvvvvvv */

// Hooks of one event run in registration order; each receives the previous
// one's output and the first error short-circuits the chain. A panicking
// hook is turned into an error so third-party code can't take the agent down.

func (a *agent) runUserPromptSubmitHooks(ctx context.Context, messages []Message) ([]Message, error) {
	for i, fn := range a.hooks.userPromptSubmit {
		next, err := safeCall(ctx, "UserPromptSubmit", i, func() ([]Message, error) { return fn(ctx, messages) })
		if err != nil {
			return nil, err
		}
		messages = next
	}
	return messages, nil
}

func (a *agent) runPreToolUseHooks(ctx context.Context, toolUse MessagesBlock) (MessagesBlock, error) {
	for i, fn := range a.hooks.preToolUse {
		next, err := safeCall(ctx, "PreToolUse", i, func() (MessagesBlock, error) { return fn(ctx, toolUse) })
		if err != nil {
			return toolUse, err
		}
		toolUse = next
	}
	return toolUse, nil
}

func (a *agent) runPostToolUseHooks(ctx context.Context, toolUse MessagesBlock, output string) (string, error) {
	for i, fn := range a.hooks.postToolUse {
		next, err := safeCall(ctx, "PostToolUse", i, func() (string, error) { return fn(ctx, toolUse, output) })
		if err != nil {
			return output, err
		}
		output = next
	}
	return output, nil
}

// runStopHooks returns the first follow-up message any hook wants to inject.
func (a *agent) runStopHooks(ctx context.Context, messages []Message) (*Message, error) {
	for i, fn := range a.hooks.stop {
		msg, err := safeCall(ctx, "Stop", i, func() (*Message, error) { return fn(ctx, messages) })
		if err != nil {
			return nil, err
		}
		if msg != nil {
			return msg, nil
		}
	}
	return nil, nil
}

func safeCall[T any](ctx context.Context, event string, index int, fn func() (T, error)) (result T, err error) {
	defer func() {
		if r := recover(); r != nil {
			log4go.DefaultLogger().Error(ctx, "%s hook #%d panicked: %v", event, index, r)
			err = fmt.Errorf("%s hook #%d panicked: %v", event, index, r)
		}
	}()
	result, err = fn()
	if err != nil {
		log4go.DefaultLogger().Error(ctx, "%s hook #%d failed: %v", event, index, err)
	}
	return result, err
}

/* ^^^^^^^^^^^^^^^^^^^^^ dispatch ^^^^^^^^^^^^^^^^^^^^^ */
