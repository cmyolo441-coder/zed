package tui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/cmyolo441-coder/zed/internal/enterprise"
	"github.com/cmyolo441-coder/zed/internal/agentos"
	"github.com/cmyolo441-coder/zed/internal/mission"
	"github.com/cmyolo441-coder/zed/internal/promax"
)

type planStep struct { ID int; Title, Risk, Command, Rollback string; Approval bool; Done bool }
type approvalItem struct { ID int; Action, Detail, Risk, Status, Approver string; Created time.Time }
type toolActivity struct { Name, Target, Status string; Started time.Time; Duration time.Duration }

type tuiMode int
const (
	modeChat tuiMode = iota
	modePlan
	modeReview
	modeDashboard
	modePalette
)

func (m *Model) openCommandPalette() {
	items := []string{
		"Run enterprise release gate",
		"Generate evidence pack",
		"Run architecture brain",
		"Create ADR",
		"Generate patch bundle",
		"Run API guardian",
		"Run Go concurrency audit",
		"Open dashboard",
		"Review diff",
		"Open setup wizard",
	}
	var b strings.Builder
	b.WriteString(m.theme.Title.Render("⌘ Enterprise Command Palette") + "\n")
	for i, item := range items { fmt.Fprintf(&b, "  %s %s\n", m.theme.Prompt.Render(fmt.Sprintf("%d", i+1)), item) }
	b.WriteString(m.theme.Dim.Render("\nType a slash command, e.g. /dashboard, /review, /plan <goal>") + "\n")
	m.appendLine(m.theme.Panel.Render(b.String()) + "\n")
}

func (m *Model) renderPlanMode(goal string) string {
	_, milestones, _ := enterprise.MilestonePlan(m.cfg.WorkDir, goal)
	m.activeGoal = goal
	m.planSteps = nil
	for _, ms := range milestones {
		cmd := ""
		if len(ms.Commands) > 0 { cmd = strings.Join(ms.Commands, " → ") }
		m.planSteps = append(m.planSteps, planStep{ID:ms.ID, Title:ms.Name, Risk:ms.Risk, Command:cmd, Rollback:ms.Rollback, Approval:ms.NeedsReview})
		if ms.NeedsReview { m.enqueueApproval("milestone_review", ms.Name+" for goal: "+goal, ms.Risk) }
	}
	var b strings.Builder
	b.WriteString(m.theme.Title.Render("PLAN MODE") + "\n")
	b.WriteString(m.theme.Dim.Render("Goal: "+goal) + "\n\n")
	for i, step := range m.planSteps {
		marker := "○"
		if i == m.currentPlanStep { marker = "◉" }
		approval := ""
		if step.Approval { approval = m.theme.ToolErr.Render(" approval") }
		fmt.Fprintf(&b, "%s %d. %s  %s%s\n", m.theme.Prompt.Render(marker), step.ID, step.Title, riskBadge(m.theme, step.Risk), approval)
		fmt.Fprintf(&b, "   command: %s\n", m.theme.Dim.Render(step.Command))
		fmt.Fprintf(&b, "   rollback: %s\n", m.theme.Dim.Render(step.Rollback))
	}
	return m.theme.Panel.Width(max(m.width-4, 60)).Render(b.String())
}

func (m *Model) renderReviewDiff() string {
	out, err := gitOutputTUI(m.cfg.WorkDir, "diff", "HEAD")
	if err != nil || strings.TrimSpace(out) == "" { out, _ = gitOutputTUI(m.cfg.WorkDir, "status", "--short") }
	if strings.TrimSpace(out) == "" { return m.theme.Panel.Render(m.theme.Title.Render("Review Diff")+"\n"+m.theme.Dim.Render("No git diff or pending changes found.")) }
	var b strings.Builder
	b.WriteString(m.theme.Title.Render("Review Diff") + "\n")
	b.WriteString(m.theme.Dim.Render("approve: /approve <id>   deny: /deny <id>") + "\n\n")
	for _, line := range strings.Split(out, "\n") {
		switch {
		case strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---") || strings.HasPrefix(line, "diff --git") || strings.HasPrefix(line, "@@"):
			b.WriteString(m.theme.Title.Render(line) + "\n")
		case strings.HasPrefix(line, "+"):
			b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#7CFF9B")).Render(line) + "\n")
		case strings.HasPrefix(line, "-"):
			b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5C7A")).Render(line) + "\n")
		default:
			b.WriteString(m.theme.Dim.Render(line) + "\n")
		}
	}
	m.enqueueApproval("review_diff", "Apply reviewed working-tree diff", "HIGH")
	return m.theme.Panel.Width(max(m.width-4, 60)).Render(firstNRunes(b.String(), 12000))
}

func (m *Model) renderApprovals() string {
	var b strings.Builder
	b.WriteString(m.theme.Title.Render("Enterprise Approvals") + "\n")
	if len(m.approvals) == 0 { b.WriteString(m.theme.Dim.Render("No pending approvals.") + "\n"); return m.theme.Panel.Render(b.String()) }
	for _, a := range m.approvals {
		fmt.Fprintf(&b, "  #%d %-14s %-8s %s\n", a.ID, a.Action, a.Risk, a.Status)
		fmt.Fprintf(&b, "     %s\n", a.Detail)
		if a.Approver != "" { fmt.Fprintf(&b, "     approver: %s\n", a.Approver) }
	}
	b.WriteString(m.theme.Dim.Render("\n/approve <id> [approver]  ·  /deny <id> [reason]") + "\n")
	return m.theme.Panel.Render(b.String())
}

func (m *Model) approve(id int, approver string) string {
	for i := range m.approvals { if m.approvals[i].ID == id { m.approvals[i].Status="approved"; m.approvals[i].Approver=approver; enterprise.AppendAudit(m.cfg.WorkDir, approver, "approve", m.approvals[i].Action, m.approvals[i].Detail); return fmt.Sprintf("approved #%d", id) } }
	return fmt.Sprintf("approval #%d not found", id)
}
func (m *Model) deny(id int, reason string) string { for i := range m.approvals { if m.approvals[i].ID == id { m.approvals[i].Status="denied"; enterprise.AppendAudit(m.cfg.WorkDir, "user", "deny", m.approvals[i].Action, reason); return fmt.Sprintf("denied #%d", id) } }; return fmt.Sprintf("approval #%d not found", id) }
func (m *Model) enqueueApproval(action, detail, risk string) { for _, a := range m.approvals { if a.Action==action && a.Detail==detail && a.Status=="pending" { return } }; m.nextApprovalID++; m.approvals=append(m.approvals, approvalItem{ID:m.nextApprovalID,Action:action,Detail:detail,Risk:risk,Status:"pending",Created:time.Now()}) }

func (m *Model) renderTimeline() string {
	entries, _ := enterprise.ReadWorkJournal(m.cfg.WorkDir)
	var b strings.Builder
	b.WriteString(m.theme.Title.Render("Agent Memory Timeline") + "\n")
	if len(entries)==0 { b.WriteString(m.theme.Dim.Render("No work journal entries yet. Use enterprise_work_journal or /plan workflows.") + "\n"); return m.theme.Panel.Render(b.String()) }
	for _, e := range entries { fmt.Fprintf(&b,"  %s  %s\n", e.Time.Format("2006-01-02 15:04"), m.theme.Prompt.Render(e.Task)); if e.Evidence!="" { fmt.Fprintf(&b,"     evidence: %s\n", e.Evidence) }; if e.Risks!="" { fmt.Fprintf(&b,"     risks: %s\n", e.Risks) } }
	return m.theme.Panel.Render(b.String())
}

func (m *Model) renderDashboard() string {
	changed, _ := gitOutputTUI(m.cfg.WorkDir, "status", "--short")
	_, heat, _ := enterprise.RiskHeatmap(m.cfg.WorkDir)
	risk := "LOW"; if len(heat)>0 { risk = heat[0].Risk }
	var b strings.Builder
	b.WriteString(m.theme.Title.Render("Autonomous Work Dashboard") + "\n")
	fmt.Fprintf(&b,"  mode:        %s\n", m.layoutMode)
	fmt.Fprintf(&b,"  active goal: %s\n", emptyDash(m.activeGoal))
	fmt.Fprintf(&b,"  step:        %d/%d\n", m.currentPlanStep+1, len(m.planSteps))
	fmt.Fprintf(&b,"  changes:     %d files\n", countNonEmptyLines(changed))
	fmt.Fprintf(&b,"  risk:        %s\n", riskBadge(m.theme, risk))
	fmt.Fprintf(&b,"  tokens:      %d input / %d output\n", m.inputTokens, m.outputTokens)
	fmt.Fprintf(&b,"  cost:        $%s\n", formatCost(m.totalCost))
	fmt.Fprintf(&b,"  approvals:   %d pending\n", m.pendingApprovalCount())
	if len(m.toolEvents)>0 { b.WriteString("\nRecent tool activity:\n"); start:=len(m.toolEvents)-5; if start<0 { start=0 }; for _, ev := range m.toolEvents[start:] { fmt.Fprintf(&b,"  ⚙ %-24s %-8s %s\n", ev.Name, ev.Status, ev.Duration.Round(time.Millisecond)) } }
	return m.theme.Panel.Render(b.String())
}

func (m *Model) renderRiskHeatmap() string {
	_, entries, err := enterprise.RiskHeatmap(m.cfg.WorkDir)
	if err != nil { return m.theme.ToolErr.Render(err.Error()) }
	var b strings.Builder
	b.WriteString(m.theme.Title.Render("Enterprise Risk Heatmap") + "\n")
	for _, e := range entries { fmt.Fprintf(&b,"  %-34s %s score=%d\n", e.Path, riskBadge(m.theme,e.Risk), e.Score) }
	return m.theme.Panel.Render(b.String())
}

func (m *Model) renderReplay(id string) string {
	if id == "" { return m.theme.Dim.Render("Usage: /replay <session-id>") }
	if m.sessions == nil { return m.theme.ToolErr.Render("Session store unavailable") }
	sess, err := m.sessions.Load(id); if err != nil { return m.theme.ToolErr.Render(err.Error()) }
	var b strings.Builder
	b.WriteString(m.theme.Title.Render("Work Replay: "+id) + "\n")
	for i, msg := range sess.Messages { role := strings.ToUpper(string(msg.Role)); content := msg.Content; if len(content)>400 { content=content[:400]+"…" }; fmt.Fprintf(&b,"\n%d. %s\n%s\n", i+1, role, content) }
	return m.theme.Panel.Render(b.String())
}

func (m *Model) renderAuthUI() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(m.theme.Glow.Render("        ◢◤\n     ◢██◤   ◢◤\n   ◢████◤ ◢██\n     ◥██◣◢██◤\n        ◥◤") + "\n\n")
	b.WriteString(m.theme.Title.Render("BITTU CHAUHAN AUTH") + "\n\n")
	b.WriteString(m.theme.Dim.Render("A browser window will open for authentication when provider supports OAuth.") + "\n")
	b.WriteString(m.theme.Dim.Render("For API-key mode, paste/set your token with environment variables:") + "\n\n")
	b.WriteString("  export ZED_API_KEY=...\n  export OPENAI_API_KEY=...\n  export ANTHROPIC_API_KEY=...\n\n")
	b.WriteString(m.theme.Hint.Render("/setup shows provider/model configuration. enter submit · ctrl+c quit") + "\n")
	return m.theme.Panel.Width(max(m.width-4, 60)).Render(b.String())
}

func (m *Model) renderSetupWizard() string {
	var b strings.Builder
	b.WriteString(m.theme.Title.Render("Model/Provider Setup Wizard") + "\n")
	fmt.Fprintf(&b,"  provider: %s\n  model:    %s\n  workdir:  %s\n\n", m.cfg.Provider, m.cfg.Model, m.cfg.WorkDir)
	b.WriteString("Steps:\n")
	b.WriteString("  1. Choose provider with /model or CLI --provider\n")
	b.WriteString("  2. Export API key: ZED_API_KEY / OPENAI_API_KEY / ANTHROPIC_API_KEY\n")
	b.WriteString("  3. Use /models to list supported models\n")
	b.WriteString("  4. Run /dashboard and /plan <goal>\n")
	return m.theme.Panel.Render(b.String())
}

func (m *Model) setTheme(name string) string { if name=="" { name="bittu" }; m.theme = DraculaTheme(); return "theme set: "+name+" (enterprise dark)" }
func (m *Model) setLayout(name string) string { switch name { case "split","focus","minimal": m.layoutMode=name; default: if name=="" { name="focus"; m.layoutMode=name } else { return "unknown layout: "+name } }; return "layout set: "+m.layoutMode }
func (m *Model) pendingApprovalCount() int { n:=0; for _, a := range m.approvals { if a.Status=="pending" { n++ } }; return n }

func riskBadge(t Theme, risk string) string { switch strings.ToUpper(risk) { case "CRITICAL": return t.ToolErr.Render("CRITICAL"); case "HIGH": return lipgloss.NewStyle().Foreground(lipgloss.Color("#FF9F43")).Bold(true).Render("HIGH"); case "MEDIUM": return lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD166")).Render("MEDIUM"); case "LOW": return t.Tool.Render("LOW"); default: return t.Dim.Render(risk) } }
func gitOutputTUI(root string, args ...string) (string,error) { cmd:=exec.Command("git",args...); cmd.Dir=root; var b bytes.Buffer; cmd.Stdout=&b; cmd.Stderr=&b; err:=cmd.Run(); return b.String(),err }
func firstNRunes(s string, n int) string { r:=[]rune(s); if len(r)<=n { return s }; return string(r[:n])+"\n…truncated…" }
func countNonEmptyLines(s string) int { c:=0; for _, l:=range strings.Split(s,"\n") { if strings.TrimSpace(l)!="" { c++ } }; return c }
func emptyDash(s string) string { if s=="" { return "—" }; return s }
func parseID(s string) int { fields:=strings.Fields(s); if len(fields)==0 { return 0 }; n,_:=strconv.Atoi(fields[0]); return n }
func writeJSON(path string, v any) error { _=os.MkdirAll(filepath.Dir(path),0755); b,_:=json.MarshalIndent(v,"","  "); return os.WriteFile(path,b,0644) }
func sortedKeys(m map[string]bool) []string { var out []string; for k:=range m { out=append(out,k) }; sort.Strings(out); return out }

func (m Model) sidePanel() string {
	var b strings.Builder
	b.WriteString(m.theme.Title.Render("Enterprise Panel") + "\n\n")
	fmt.Fprintf(&b, "mode: %s\n", m.layoutMode)
	fmt.Fprintf(&b, "goal: %s\n", emptyDash(m.activeGoal))
	fmt.Fprintf(&b, "approvals: %d\n", m.pendingApprovalCount())
	fmt.Fprintf(&b, "tokens: %d\n", m.inputTokens+m.outputTokens)
	if len(m.planSteps) > 0 {
		b.WriteString("\nPlan:\n")
		for _, s := range m.planSteps { fmt.Fprintf(&b, " %d. %s [%s]\n", s.ID, s.Title, s.Risk) }
	}
	if len(m.toolEvents) > 0 {
		b.WriteString("\nTools:\n")
		start := len(m.toolEvents)-4; if start < 0 { start = 0 }
		for _, ev := range m.toolEvents[start:] { fmt.Fprintf(&b, " ⚙ %s %s\n", ev.Name, ev.Status) }
	}
	return b.String()
}


func (m *Model) runAgentOSCommand(name, arg string) string {
	var r agentos.Result
	var err error
	switch name {
	case "vision-ui":
		r, err = agentos.VisionUIBuilder(m.cfg.WorkDir, arg)
	case "desktop":
		r, err = agentos.DesktopView(m.cfg.WorkDir)
	case "app":
		r, err = agentos.AppBuilder(m.cfg.WorkDir, arg)
	case "theme-generate":
		r, err = agentos.ThemeGenerator(m.cfg.WorkDir, arg)
	case "persona":
		r, err = agentos.PersonaEngine(m.cfg.WorkDir, arg)
	case "tutorial":
		r, err = agentos.TutorialBuilder(m.cfg.WorkDir)
	case "prompt-os":
		r, err = agentos.PromptOS(m.cfg.WorkDir, "init", "", nil)
	case "skills":
		r, err = agentos.SkillMarketplace(m.cfg.WorkDir, "init", agentos.Skill{})
	case "knowledge":
		r, err = agentos.KnowledgePackBuilder(m.cfg.WorkDir)
	case "readme-design":
		r, err = agentos.ReadmeDesigner(m.cfg.WorkDir)
	case "self-upgrade":
		r, err = agentos.SelfUpgradeKernel(m.cfg.WorkDir)
	case "code-movie":
		r, err = agentos.CodeMovie(m.cfg.WorkDir, arg)
	case "animation":
		r, err = agentos.AnimationEngine(arg)
	case "voice":
		r, err = agentos.VoiceCommandMode(arg)
	default:
		return m.theme.ToolErr.Render("unknown Agent OS command: "+name)
	}
	if err != nil { return m.theme.ToolErr.Render(err.Error()) }
	return m.theme.Panel.Render(r.Text())
}


func (m *Model) renderMissionControl(args string) string {
	fields := strings.Fields(args)
	action := "list"
	if len(fields) > 0 { action = fields[0] }
	switch action {
	case "start":
		goal := strings.TrimSpace(strings.TrimPrefix(args, "start"))
		ms, err := mission.Start(m.cfg.WorkDir, strings.Trim(goal, "\""))
		if err != nil { return m.theme.ToolErr.Render(err.Error()) }
		return m.theme.Panel.Render(fmt.Sprintf("🚀 Mission #%d started\n%s", ms.ID, ms.Goal))
	case "pause":
		ms, err := mission.SetStatus(m.cfg.WorkDir, parseID(strings.TrimPrefix(args, "pause")), mission.StatusPaused, "paused from TUI")
		if err != nil { return m.theme.ToolErr.Render(err.Error()) }
		return m.theme.Panel.Render(fmt.Sprintf("⏸ Mission #%d paused", ms.ID))
	case "resume":
		ms, err := mission.SetStatus(m.cfg.WorkDir, parseID(strings.TrimPrefix(args, "resume")), mission.StatusActive, "resumed from TUI")
		if err != nil { return m.theme.ToolErr.Render(err.Error()) }
		return m.theme.Panel.Render(fmt.Sprintf("▶ Mission #%d resumed", ms.ID))
	case "complete":
		ms, err := mission.SetStatus(m.cfg.WorkDir, parseID(strings.TrimPrefix(args, "complete")), mission.StatusCompleted, "completed from TUI")
		if err != nil { return m.theme.ToolErr.Render(err.Error()) }
		return m.theme.Panel.Render(fmt.Sprintf("✅ Mission #%d completed", ms.ID))
	default:
		s, err := mission.List(m.cfg.WorkDir)
		if err != nil { return m.theme.ToolErr.Render(err.Error()) }
		return m.theme.Panel.Render(mission.Render(s))
	}
}


func (m *Model) runProMaxCommand(name, arg string) string {
	var r promax.Result
	var err error
	switch name {
	case "map": r,err=promax.CodebaseMapTool(m.cfg.WorkDir)
	case "memory-graph": r,err=promax.MemoryGraphTool(m.cfg.WorkDir,"build",arg)
	case "workspace": r,err=promax.NeuralWorkspace(m.cfg.WorkDir)
	case "artifacts": r,err=promax.ArtifactBrowser(m.cfg.WorkDir)
	case "jobs": r,err=promax.JobManager(m.cfg.WorkDir,"list","",0)
	case "replay2": r,err=promax.ReplayEngine(m.cfg.WorkDir,arg)
	case "hypervisor": r,err=promax.AgentHypervisor(m.cfg.WorkDir,arg)
	case "project-pro": r,err=promax.ProjectGeneratorPro(m.cfg.WorkDir,arg,"go-cli")
	case "preview": r,err=promax.PreviewServer(m.cfg.WorkDir,arg)
	case "web-ui": r,err=promax.WebUIExporter(m.cfg.WorkDir)
	case "plugin-build": r,err=promax.PluginBuilder(m.cfg.WorkDir,arg,"generated by BITTU CHAUHAN")
	case "self-tests": r,err=promax.AgentSelfTestHarness(m.cfg.WorkDir)
	case "bench": r,err=promax.BenchmarkLab(m.cfg.WorkDir)
	case "kernel-plan": r,err=promax.KernelUpgradePlanner(m.cfg.WorkDir)
	default: return m.theme.ToolErr.Render("unknown Pro Max command: "+name)
	}
	if err != nil { return m.theme.ToolErr.Render(err.Error()) }
	return m.theme.Panel.Render(r.Text())
}
