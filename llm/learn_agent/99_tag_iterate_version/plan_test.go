package agentloop

import (
	"context"
	"strings"
	"testing"
)

// gatedTeam spawns one teammate behind the plan gate and returns the lead, the teammate and a lead ctx.
func gatedTeam(t *testing.T, requirePlan bool) (*agent, *Teammate, context.Context) {
	t.Helper()
	useTasksDir(t)
	useTeam(t, testTeamConfig())
	lead := NewAgent(newTeamLLM(), nil).(*agent)
	t.Cleanup(lead.team.Shutdown)
	mate, err := lead.team.Spawn("alice", "", "", requirePlan)
	if err != nil {
		t.Fatal(err)
	}
	return lead, mate, withAgent(withAgentName(context.Background(), "main"), lead)
}

func TestPlanGate_BlocksMutatingToolsUntilApproved(t *testing.T) {
	lead, mate, _ := gatedTeam(t, true)
	if mate.planNow() != planRequired {
		t.Fatalf("gate = %s, want required", mate.planNow())
	}

	ctx := withAgentName(context.Background(), "alice")
	blocked := []string{"run_bash", "write_file", "edit_file"}
	for _, name := range blocked {
		out := mate.agent.runTool(ctx, MessagesBlock{Name: name, Input: map[string]interface{}{"command": "echo hi", "path": "a.txt", "content": "x"}})
		if !strings.HasPrefix(out, "Blocked: your plan status is required") {
			t.Errorf("%s = %q, want blocked", name, out)
		}
	}
	// reading and planning stay open
	if out := mate.agent.runTool(ctx, MessagesBlock{Name: "read_file", Input: map[string]interface{}{"path": "a.txt"}}); strings.HasPrefix(out, "Blocked") {
		t.Errorf("read_file should not be gated: %q", out)
	}
	if _, ok := mate.agent.toolIndex["submit_plan"]; !ok {
		t.Fatal("teammate has no submit_plan")
	}
	if _, ok := lead.toolIndex["submit_plan"]; ok {
		t.Error("the lead should not submit plans")
	}

	// submit → pending, still blocked
	out, err := mate.agent.toolIndex["submit_plan"].Handler(ctx, map[string]interface{}{"plan": "1. read 2. edit"})
	if err != nil || !strings.Contains(out, "Plan submitted as plan_") {
		t.Fatalf("submit_plan = %q, %v", out, err)
	}
	reqID := planIDFrom(out)
	if mate.planNow() != planPending {
		t.Errorf("gate = %s, want pending", mate.planNow())
	}
	if got := mate.agent.runTool(ctx, MessagesBlock{Name: "run_bash", Input: map[string]interface{}{"command": "echo hi"}}); !strings.Contains(got, "plan status is pending") {
		t.Errorf("run_bash while pending = %q", got)
	}
	// the lead sees the submitted plan as an event
	waitFor(t, "the plan request", func() bool { return lead.team.bus.Peek(leadName) })
	notes := strings.Join(lead.team.consumeTeamEvents(), "\n")
	if !strings.Contains(notes, `type="plan_approval_request"`) || !strings.Contains(notes, "1. read 2. edit") {
		t.Errorf("plan event = %s", notes)
	}

	// rejection keeps the gate shut
	leadCtx := withAgent(withAgentName(context.Background(), "main"), lead)
	if _, err := lead.toolIndex["review_plan"].Handler(leadCtx, map[string]interface{}{"request_id": reqID, "approve": false, "feedback": "too vague"}); err != nil {
		t.Fatal(err)
	}
	if mate.planNow() != planRejected {
		t.Errorf("gate = %s, want rejected", mate.planNow())
	}
	if got := mate.agent.runTool(ctx, MessagesBlock{Name: "write_file", Input: map[string]interface{}{"path": "a.txt", "content": "x"}}); !strings.Contains(got, "plan status is rejected") {
		t.Errorf("write_file after rejection = %q", got)
	}

	// a second answer to the same request changes nothing
	if _, err := lead.toolIndex["review_plan"].Handler(leadCtx, map[string]interface{}{"request_id": reqID, "approve": true}); err == nil {
		t.Error("answering the same plan twice should fail")
	}
	if _, err := lead.toolIndex["review_plan"].Handler(leadCtx, map[string]interface{}{"request_id": "plan_deadbeef", "approve": true}); err == nil {
		t.Error("an unknown request id should fail")
	}
	if _, err := lead.toolIndex["review_plan"].Handler(leadCtx, map[string]interface{}{"request_id": reqID}); err == nil {
		t.Error("approve must be a boolean")
	}

	// resubmit and approve: the gate opens
	out, _ = mate.agent.toolIndex["submit_plan"].Handler(ctx, map[string]interface{}{"plan": "detailed plan"})
	if _, err := lead.toolIndex["review_plan"].Handler(leadCtx, map[string]interface{}{"request_id": planIDFrom(out), "approve": true, "feedback": "go"}); err != nil {
		t.Fatal(err)
	}
	if mate.planNow() != planApproved {
		t.Fatalf("gate = %s, want approved", mate.planNow())
	}
	if got := mate.agent.runTool(ctx, MessagesBlock{Name: "run_bash", Input: map[string]interface{}{"command": "echo hi"}}); strings.HasPrefix(got, "Blocked") {
		t.Errorf("run_bash after approval = %q", got)
	}
}

func planIDFrom(out string) string {
	_, rest, _ := strings.Cut(out, "submitted as ")
	id, _, _ := strings.Cut(rest, ".")
	return id
}

func TestPlanGate_NewWorkInvalidatesApproval(t *testing.T) {
	lead, mate, leadCtx := gatedTeam(t, true)
	mateCtx := withAgentName(context.Background(), "alice")

	// approved for the current assignment
	out, _ := mate.agent.toolIndex["submit_plan"].Handler(mateCtx, map[string]interface{}{"plan": "plan A"})
	if _, err := lead.toolIndex["review_plan"].Handler(leadCtx, map[string]interface{}{"request_id": planIDFrom(out), "approve": true}); err != nil {
		t.Fatal(err)
	}
	if mate.planNow() != planApproved {
		t.Fatalf("gate = %s", mate.planNow())
	}

	// picking up different work invalidates it
	mate.setTask("task_11111111")
	if mate.planNow() != planRequired {
		t.Errorf("gate after new work = %s, want required", mate.planNow())
	}

	// a plan answered after the teammate moved on is not applied either
	out, _ = mate.agent.toolIndex["submit_plan"].Handler(mateCtx, map[string]interface{}{"plan": "plan B"})
	mate.setTask("task_22222222")
	if _, err := lead.toolIndex["review_plan"].Handler(leadCtx, map[string]interface{}{"request_id": planIDFrom(out), "approve": true}); err != nil {
		t.Fatal(err)
	}
	if mate.planNow() != planRequired {
		t.Errorf("stale approval applied: gate = %s", mate.planNow())
	}
	waitFor(t, "the stale-plan notice", func() bool { return mate.agent.team.bus.Peek("alice") })
	msgs := mate.agent.team.bus.ReadInbox("alice")
	if len(msgs) == 0 || !strings.Contains(msgs[len(msgs)-1].Content, "no longer applies") {
		t.Errorf("teammate not told the plan was stale: %+v", msgs)
	}
}

func TestPlanGate_RequestPlanOnRunningTeammate(t *testing.T) {
	lead, mate, leadCtx := gatedTeam(t, false)
	if mate.planNow() != planNotRequired {
		t.Fatalf("gate = %s, want not_required", mate.planNow())
	}
	ctx := withAgentName(context.Background(), "alice")
	if got := mate.agent.runTool(ctx, MessagesBlock{Name: "run_bash", Input: map[string]interface{}{"command": "echo hi"}}); strings.HasPrefix(got, "Blocked") {
		t.Fatalf("ungated teammate was blocked: %q", got)
	}

	res, err := lead.toolIndex["request_plan"].Handler(leadCtx, map[string]interface{}{"name": "alice"})
	if err != nil || !strings.Contains(res, "Asked alice for a plan") {
		t.Fatalf("request_plan = %q, %v", res, err)
	}
	if mate.planNow() != planRequired {
		t.Errorf("gate = %s, want required", mate.planNow())
	}
	if got := mate.agent.runTool(ctx, MessagesBlock{Name: "run_bash", Input: map[string]interface{}{"command": "echo hi"}}); !strings.Contains(got, "plan status is required") {
		t.Errorf("run_bash = %q", got)
	}
	if _, err := lead.toolIndex["request_plan"].Handler(leadCtx, map[string]interface{}{"name": "nobody"}); err == nil {
		t.Error("requesting a plan from an unknown teammate should fail")
	}

	// the lead's own tools are never gated
	if lead.team.planStateFor("main") != planNotRequired {
		t.Error("the lead should not be behind a gate")
	}
}

func TestSpawnTeammate_RequirePlanFlag(t *testing.T) {
	useTasksDir(t)
	useTeam(t, testTeamConfig())
	lead := NewAgent(newTeamLLM(), nil).(*agent)
	defer lead.team.Shutdown()
	ctx := withAgent(withAgentName(context.Background(), "main"), lead)

	out, err := lead.toolIndex["spawn_teammate"].Handler(ctx, map[string]interface{}{"name": "alice", "require_plan": true})
	if err != nil || !strings.Contains(out, "must get a plan approved") {
		t.Fatalf("spawn = %q, %v", out, err)
	}
	if lead.team.planStateFor("alice") != planRequired {
		t.Errorf("gate = %s", lead.team.planStateFor("alice"))
	}
	if list, _ := lead.toolIndex["list_teammates"].Handler(ctx, nil); !strings.Contains(list, "[plan required]") {
		t.Errorf("list_teammates = %q", list)
	}
	if _, err := lead.toolIndex["spawn_teammate"].Handler(ctx, map[string]interface{}{"name": "bob", "require_plan": "yes"}); err == nil {
		t.Error("require_plan must be a boolean")
	}
}
