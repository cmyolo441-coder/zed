package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gjkjk/zed/internal/enterprise"
)

type EnterpriseRiskHeatmap struct{ WorkDir string }
func (t *EnterpriseRiskHeatmap) Name() string { return "enterprise_risk_heatmap" }
func (t *EnterpriseRiskHeatmap) Description() string { return "Build a directory-level enterprise risk heatmap from critical path, secrets, taint, concurrency, perf, ownership, and migration findings. Args: {\"path\":\"optional\",\"json\":false}" }
func (t *EnterpriseRiskHeatmap) RequiresApproval() bool { return false }
func (t *EnterpriseRiskHeatmap) Schema() map[string]any { return enterprisePathSchema() }
func (t *EnterpriseRiskHeatmap) Execute(_ context.Context, args string) (string, error) { a, err := enterpriseArgs(args); if err != nil { return "", err }; r, entries, err := enterprise.RiskHeatmap(resolveEnterprisePath(t.WorkDir,a.Path)); if err != nil { return "", err }; if a.JSON { b,_ := json.MarshalIndent(entries,"","  "); return string(b), nil }; var b strings.Builder; b.WriteString(r.Summary()); if len(entries)>0 { b.WriteString("\nHeatmap:\n"); for _, e := range entries { fmt.Fprintf(&b,"  %-32s %-8s score=%d findings=%d\n", e.Path, e.Risk, e.Score, len(e.Findings)) } }; return b.String(), nil }

type EnterpriseMilestoneExecutor struct{ WorkDir string }
func (t *EnterpriseMilestoneExecutor) Name() string { return "enterprise_milestone_executor" }
func (t *EnterpriseMilestoneExecutor) Description() string { return "Create a real milestone execution plan for an enterprise goal: analysis, implementation, verification, evidence, release review. Args: {\"goal\":\"...\",\"json\":false}" }
func (t *EnterpriseMilestoneExecutor) RequiresApproval() bool { return false }
func (t *EnterpriseMilestoneExecutor) Schema() map[string]any { return map[string]any{"type":"object","properties":map[string]any{"goal":map[string]any{"type":"string"},"json":map[string]any{"type":"boolean"}},"required":[]string{"goal"}} }
func (t *EnterpriseMilestoneExecutor) Execute(_ context.Context, args string) (string, error) { var a struct{ Goal string `json:"goal"`; JSON bool `json:"json"` }; if err:=parseArgs(args,&a); err!=nil { return "",err }; r, milestones, err := enterprise.MilestonePlan(t.WorkDir,a.Goal); if err!=nil { return "",err }; if a.JSON { b,_:=json.MarshalIndent(milestones,"","  "); return string(b),nil }; var b strings.Builder; b.WriteString(r.Summary()); b.WriteString("\nMilestones:\n"); for _, m := range milestones { fmt.Fprintf(&b,"  %d. %s [%s] review=%v\n", m.ID,m.Name,m.Risk,m.NeedsReview); for _, c := range m.Commands { fmt.Fprintf(&b,"     - %s\n", c) }; fmt.Fprintf(&b,"     rollback: %s\n", m.Rollback) }; return b.String(), nil }

var _ Tool = (*EnterpriseRiskHeatmap)(nil)
var _ Tool = (*EnterpriseMilestoneExecutor)(nil)
