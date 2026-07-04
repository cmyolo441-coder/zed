package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	appctx "github.com/gjkjk/zed/internal/context"
	"github.com/gjkjk/zed/internal/config"
	"github.com/gjkjk/zed/internal/llm"
)

func effortBudget(model string, effort config.Effort) appctx.Budget {
	b := appctx.BudgetForModel(config.LookupModel(model).MaxTokens)
	if effort.ContextBudgetMultiplier > 1 {
		maxModel := config.LookupModel(model).MaxTokens
		grown := int(float64(b.MaxContextTokens) * effort.ContextBudgetMultiplier)
		if maxModel > 0 && grown > maxModel {
			grown = maxModel
		}
		b.MaxContextTokens = grown
	}
	return b
}

func (a *Agent) runEffortPreflight(ctx context.Context, userInput string, emit func(Event)) {
	if !a.effort.EnterprisePreflight && !a.effort.MilestonePlanning && !a.effort.RiskHeatmap {
		return
	}
	emit(Event{Kind: EventNotice, Text: fmt.Sprintf("%s real-effort preflight: architecture, impact, milestones, risk", a.effort.Label)})
	var blocks []string
	// In Dream Mode, suppress verbose preflight tool output — results still
	// get injected into context but the terminal stays clean.
	quiet := a.effort.DreamMode
	if a.effort.MilestonePlanning {
		blocks = append(blocks, a.runEffortToolQuiet(ctx, emit, "enterprise_milestone_executor", fmt.Sprintf(`{"goal":%q}`, userInput), quiet))
	}
	if a.effort.EnterprisePreflight {
		blocks = append(blocks, a.runEffortToolQuiet(ctx, emit, "enterprise_architecture_brain", `{}`, quiet))
		blocks = append(blocks, a.runEffortToolQuiet(ctx, emit, "enterprise_change_impact", `{}`, quiet))
	}
	if a.effort.RiskHeatmap {
		blocks = append(blocks, a.runEffortToolQuiet(ctx, emit, "enterprise_risk_heatmap", `{}`, quiet))
	}
	if a.effort.WorkJournal {
		_ = a.runEffortToolQuiet(ctx, emit, "enterprise_work_journal", fmt.Sprintf(`{"action":"append","task":%q,"evidence":"effort preflight started","risks":"auto-computed by enterprise analyzers","next":"execute plan and verify"}`, userInput), quiet)
	}
	if a.effort.DreamMode {
		blocks = append(blocks, a.runDreamControlPlane(ctx, userInput, emit)...)
	}
	a.history = append(a.history, llm.Message{Role: llm.RoleUser, Content: "[REAL EFFORT PREFLIGHT]\n" + strings.Join(blocks, "\n\n---\n\n") + "\n\nUse this real local evidence to plan and execute the task. Address high/critical risks before final delivery."})
}

func (a *Agent) runGoalBootstrap(ctx context.Context, userInput string, emit func(Event)) {
	if !a.effort.GoalMode {
		return
	}
	emit(Event{Kind: EventNotice, Text: "Goal mode max autonomy: milestone execution + verification gates enabled"})
	goal := strings.TrimPrefix(userInput, "[GOAL]")
	goal = strings.TrimSpace(goal)
	if goal == "" { goal = userInput }
	content := "[GOAL MODE EXECUTION CONTRACT]\n" +
		"You must work end-to-end: analyze, plan, implement, verify, document, and produce evidence.\n" +
		"Before final answer, ensure: build/test attempted, enterprise_policy_gate considered, patch/evidence/work journal updated when files changed.\n" +
		"Goal: " + goal
	a.history = append(a.history, llm.Message{Role: llm.RoleUser, Content: content})
}

func (a *Agent) runEnterpriseVerification(ctx context.Context, emit func(Event)) {
	if !a.effort.EnterpriseReleaseGate && !a.effort.PatchBundle && !a.effort.EvidencePack && !a.effort.WorkJournal {
		return
	}
	emit(Event{Kind: EventNotice, Text: "enterprise effort verification gates running"})
	var blocks []string
	quiet := a.effort.DreamMode
	if a.effort.EnterpriseReleaseGate {
		blocks = append(blocks, a.runEffortToolQuiet(ctx, emit, "enterprise_policy_gate", `{}`, quiet))
	}
	if a.effort.PatchBundle {
		blocks = append(blocks, a.runEffortToolQuiet(ctx, emit, "enterprise_patch_bundle", `{"output":"zed-patch-bundle.md"}`, quiet))
	}
	if a.effort.EvidencePack {
		blocks = append(blocks, a.runEffortToolQuiet(ctx, emit, "enterprise_evidence_pack", `{"output":".zed-evidence-pack.json"}`, quiet))
	}
	if a.effort.WorkJournal {
		_ = a.runEffortToolQuiet(ctx, emit, "enterprise_work_journal", `{"action":"append","task":"enterprise effort verification","evidence":"post-change gates executed","next":"summarize verified result"}`, quiet)
	}
	if a.effort.DreamMode {
		blocks = append(blocks, a.runDreamVerificationPlane(ctx, emit)...)
	}
	if len(blocks) > 0 {
		a.history = append(a.history, llm.Message{Role: llm.RoleUser, Content: "[ENTERPRISE VERIFICATION RESULTS]\n" + strings.Join(blocks, "\n\n---\n\n") + "\n\nFix blocking failures before final response. If a gate cannot run because the environment lacks tooling, report that clearly."})
	}
}

func (a *Agent) runEffortTool(ctx context.Context, emit func(Event), name, args string) string {
	return a.runEffortToolQuiet(ctx, emit, name, args, false)
}

// runEffortToolQuiet executes an effort tool. When quiet is true, tool call/done
// events are suppressed in the UI — the tool still runs and its output is
// injected into context, but the terminal stays clean.
func (a *Agent) runEffortToolQuiet(ctx context.Context, emit func(Event), name, args string, quiet bool) string {
	t, ok := a.registry.Get(name)
	if !ok {
		return name + ": tool not registered"
	}
	if !quiet {
		emit(Event{Kind: EventToolCall, ToolName: name, ToolArgs: args})
	}
	start := time.Now()
	result, err := t.Execute(ctx, args)
	failed := err != nil
	if err != nil {
		result = "Error: " + err.Error()
	}
	if len(result) > 12000 {
		result = result[:12000] + "\n…truncated for context…"
	}
	if !quiet {
		emit(Event{Kind: EventToolDone, ToolName: name, Result: result + fmt.Sprintf("\n(duration: %s)", time.Since(start).Round(time.Millisecond)), IsError: failed})
	}
	return "## " + name + "\n" + result
}


func (a *Agent) runDreamControlPlane(ctx context.Context, userInput string, emit func(Event)) []string {
	// Dream control plane runs silently — no notice, no per-tool output.
	goal := strings.TrimSpace(strings.TrimPrefix(userInput, "[DREAM]"))
	goal = strings.TrimSpace(strings.TrimPrefix(goal, "[GOAL]"))
	if goal == "" { goal = userInput }
	tools := []struct{ name, args string }{
		{"mission_control", fmt.Sprintf(`{"action":"start","goal":%q}`, goal)},
		{"agent_hypervisor", fmt.Sprintf(`{"goal":%q}`, goal)},
		{"memory_graph", `{"action":"build"}`},
		{"codebase_map", `{}`},
		{"neural_workspace", `{}`},
		{"artifact_browser", `{}`},
		{"prompt_os", `{"action":"init"}`},
		{"skill_marketplace", `{"action":"init"}`},
		{"self_upgrade_kernel", `{}`},
	}
	out := make([]string, 0, len(tools))
	for _, t := range tools { out = append(out, a.runEffortToolQuiet(ctx, emit, t.name, t.args, true)) }
	return out
}

func (a *Agent) runDreamVerificationPlane(ctx context.Context, emit func(Event)) []string {
	// Dream verification plane runs silently — no notice, no per-tool output.
	tools := []struct{ name, args string }{
		{"timeline_db", `{"action":"append","name":"dream_verification","text":"Dream Mode verification plane executed"}`},
		{"tool_reliability_score", `{}`},
		{"benchmark_lab", `{}`},
		{"artifact_browser", `{}`},
		{"replay_engine", `{"name":"dream-mode"}`},
		{"kernel_upgrade_planner", `{}`},
	}
	out := make([]string, 0, len(tools))
	for _, t := range tools { out = append(out, a.runEffortToolQuiet(ctx, emit, t.name, t.args, true)) }
	return out
}
