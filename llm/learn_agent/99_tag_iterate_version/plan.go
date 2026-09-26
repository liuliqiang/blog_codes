package agentloop

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/liuliqiang/log4go"
)

// Plan protocol message types. A plan runs in the opposite direction to a shutdown: the lead asks for one, the
// teammate submits it, the lead approves or rejects it.
const (
	teamPlanRequest          = "plan_request"
	teamPlanApprovalRequest  = "plan_approval_request"
	teamPlanApprovalResponse = "plan_approval_response"
)

// planState is how far a teammate is through the approval gate.
type planState string

const (
	planNotRequired planState = "not_required"
	planRequired    planState = "required"
	planPending     planState = "pending"
	planApproved    planState = "approved"
	planRejected    planState = "rejected"
)

// planGatedTools may not run until the teammate's plan is approved. Reading and planning stay open so it can study
// the code and revise its proposal.
var planGatedTools = map[string]bool{
	"run_bash":   true,
	"write_file": true,
	"edit_file":  true,
}

// planRequest is a submitted plan the lead has not answered yet.
type planRequest struct {
	teammate string
	plan     string
	version  int // the teammate's work version when it submitted; a later claim or release invalidates the approval
	answered bool
}

func newRequestID(ctx context.Context, prefix string) (string, error) {
	var raw [4]byte
	if _, err := rand.Read(raw[:]); err != nil {
		log4go.DefaultLogger().Error(ctx, "generate request ID failed: %v", err)
		return "", err
	}
	return prefix + "_" + hex.EncodeToString(raw[:]), nil
}

// planStateFor reports the gate of the named agent; anything that is not a teammate is ungated.
func (t *TeamRuntime) planStateFor(name string) planState {
	if t == nil {
		return planNotRequired
	}
	t.mu.Lock()
	mate, ok := t.mates[name]
	t.mu.Unlock()
	if !ok {
		return planNotRequired
	}
	mate.mu.Lock()
	defer mate.mu.Unlock()
	return mate.plan
}

// requirePlan puts a running teammate behind the gate and tells it to submit a plan.
func (t *TeamRuntime) requirePlan(name string) error {
	t.mu.Lock()
	mate, ok := t.mates[name]
	t.mu.Unlock()
	if !ok {
		return fmt.Errorf("teammate %s not found", name)
	}
	if state, _ := mate.snapshot(); state == teammateStopped {
		return fmt.Errorf("teammate %s already stopped", name)
	}
	mate.setPlan(planRequired)
	return t.bus.Send(TeamMessage{
		From:    leadName,
		To:      name,
		Type:    teamPlanRequest,
		Content: "Submit a plan with submit_plan before changing anything. You can read files until it is approved.",
	})
}

// submitPlan records a teammate's plan and returns the request id the lead's answer will carry.
func (t *TeamRuntime) submitPlan(ctx context.Context, name, plan string) (string, error) {
	t.mu.Lock()
	mate, ok := t.mates[name]
	t.mu.Unlock()
	if !ok {
		return "", fmt.Errorf("teammate %s not found", name)
	}
	id, err := newRequestID(ctx, "plan")
	if err != nil {
		return "", err
	}
	mate.mu.Lock()
	version := mate.workVersion
	mate.plan = planPending
	mate.mu.Unlock()

	t.mu.Lock()
	t.plans[id] = &planRequest{teammate: name, plan: plan, version: version}
	t.mu.Unlock()

	if err := t.bus.Send(TeamMessage{From: name, To: leadName, Type: teamPlanApprovalRequest, RequestID: id, Content: plan}); err != nil {
		return "", err
	}
	return id, nil
}

// reviewPlan applies the lead's answer to one submitted plan. An unknown id, a second answer, or a plan whose
// teammate has moved on to other work since submitting it changes nothing.
func (t *TeamRuntime) reviewPlan(requestID string, approve bool, feedback string) error {
	t.mu.Lock()
	req, ok := t.plans[requestID]
	if !ok {
		t.mu.Unlock()
		return fmt.Errorf("plan request %s not found", requestID)
	}
	if req.answered {
		t.mu.Unlock()
		return fmt.Errorf("plan request %s was already answered", requestID)
	}
	mate, running := t.mates[req.teammate]
	req.answered = true
	t.mu.Unlock()
	if !running {
		return fmt.Errorf("teammate %s is gone", req.teammate)
	}

	mate.mu.Lock()
	stale := mate.workVersion != req.version
	switch {
	case stale:
		mate.plan = planRequired
	case approve:
		mate.plan = planApproved
	default:
		mate.plan = planRejected
	}
	state := mate.plan
	mate.mu.Unlock()

	content := feedback
	switch {
	case stale:
		content = "This plan was written for earlier work and no longer applies; submit a new one. " + feedback
	case approve:
		content = "Plan approved. " + feedback
	default:
		content = "Plan rejected. " + feedback
	}
	fmt.Fprintf(teamOut, "\033[32m[team] plan %s for %s: %s\033[0m\n", requestID, req.teammate, state)
	return t.bus.Send(TeamMessage{From: leadName, To: req.teammate, Type: teamPlanApprovalResponse, RequestID: requestID, Content: content})
}

/* vvvvvvvvvvvvvvvvvvvvv tools vvvvvvvvvvvvvvvvvvvvv */

func (a *agent) runRequestPlan(ctx context.Context, input map[string]interface{}) (string, error) {
	name, err := stringArg(input, "name")
	if err != nil {
		return "", err
	}
	if err := a.team.requirePlan(name); err != nil {
		log4go.DefaultLogger().Error(ctx, "[%s] request_plan failed: %v, name: %s", agentNameFrom(ctx), err, name)
		return "", err
	}
	return fmt.Sprintf("Asked %s for a plan; it cannot run commands or change files until you approve it.", name), nil
}

func (a *agent) runReviewPlan(ctx context.Context, input map[string]interface{}) (string, error) {
	requestID, err := stringArg(input, "request_id")
	if err != nil {
		return "", err
	}
	approve, ok := input["approve"].(bool)
	if !ok {
		return "", fmt.Errorf("argument %q must be a boolean", "approve")
	}
	feedback, err := optionalStringArg(input, "feedback", "")
	if err != nil {
		return "", err
	}
	if err := a.team.reviewPlan(requestID, approve, feedback); err != nil {
		log4go.DefaultLogger().Error(ctx, "[%s] review_plan failed: %v, request: %s", agentNameFrom(ctx), err, requestID)
		return "", err
	}
	verdict := "rejected"
	if approve {
		verdict = "approved"
	}
	return fmt.Sprintf("Plan %s %s.", requestID, verdict), nil
}

func (a *agent) runSubmitPlan(ctx context.Context, input map[string]interface{}) (string, error) {
	plan, err := stringArg(input, "plan")
	if err != nil {
		return "", err
	}
	id, err := a.team.submitPlan(ctx, a.name, plan)
	if err != nil {
		log4go.DefaultLogger().Error(ctx, "[%s] submit_plan failed: %v", agentNameFrom(ctx), err)
		return "", err
	}
	return fmt.Sprintf("Plan submitted as %s. Wait for the lead's answer; you can keep reading in the meantime.", id), nil
}

// submitPlanTool is teammate-only: the lead never submits plans.
func (a *agent) submitPlanTool() Tool {
	return Tool{
		Name:        "submit_plan",
		Handler:     a.runSubmitPlan,
		Description: "Submit a plan for the lead to approve. Until it is approved you cannot run commands or change files.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"plan": map[string]interface{}{"type": "string"},
			},
			"required": []string{"plan"},
		},
	}
}
