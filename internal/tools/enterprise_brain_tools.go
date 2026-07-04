package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gjkjk/zed/internal/enterprise"
)

type EnterpriseArchitectureBrain struct{ WorkDir string }
func (t *EnterpriseArchitectureBrain) Name() string { return "enterprise_architecture_brain" }
func (t *EnterpriseArchitectureBrain) Description() string { return "Build a real architecture knowledge graph from project files: domains, imports, integrations, coupling findings. Args: {\"path\":\"optional\",\"json\":false}" }
func (t *EnterpriseArchitectureBrain) RequiresApproval() bool { return false }
func (t *EnterpriseArchitectureBrain) Schema() map[string]any { return enterprisePathSchema() }
func (t *EnterpriseArchitectureBrain) Execute(_ context.Context, args string) (string, error) { a, err := enterpriseArgs(args); if err != nil { return "", err }; r, model, err := enterprise.ArchitectureBrain(resolveEnterprisePath(t.WorkDir,a.Path)); if err != nil { return "", err }; if a.JSON { b,_ := json.MarshalIndent(model,"","  "); return string(b), nil }; return r.Summary(), nil }

type EnterpriseChangeImpact struct{ WorkDir string }
func (t *EnterpriseChangeImpact) Name() string { return "enterprise_change_impact" }
func (t *EnterpriseChangeImpact) Description() string { return "Analyze git diff blast radius: changed files, impacted tests, endpoints, and risk. Args: {\"path\":\"optional\",\"json\":false}" }
func (t *EnterpriseChangeImpact) RequiresApproval() bool { return false }
func (t *EnterpriseChangeImpact) Schema() map[string]any { return enterprisePathSchema() }
func (t *EnterpriseChangeImpact) Execute(_ context.Context, args string) (string, error) { a, err := enterpriseArgs(args); if err != nil { return "", err }; r, impact, err := enterprise.ChangeImpact(resolveEnterprisePath(t.WorkDir,a.Path)); if err != nil { return "", err }; if a.JSON { b,_ := json.MarshalIndent(impact,"","  "); return string(b), nil }; return r.Summary(), nil }

type EnterprisePatchBundle struct{ WorkDir string }
func (t *EnterprisePatchBundle) Name() string { return "enterprise_patch_bundle" }
func (t *EnterprisePatchBundle) Description() string { return "Write a reviewable patch bundle markdown with git diff, impact, verification, and rollback notes. Args: {\"output\":\"zed-patch-bundle.md\"}" }
func (t *EnterprisePatchBundle) RequiresApproval() bool { return true }
func (t *EnterprisePatchBundle) Schema() map[string]any { s:=enterprisePathSchema(); s["properties"].(map[string]any)["output"]=map[string]any{"type":"string"}; return s }
func (t *EnterprisePatchBundle) Execute(_ context.Context, args string) (string, error) { a, err := enterpriseArgs(args); if err != nil { return "", err }; out, err := enterprise.WritePatchBundle(resolveEnterprisePath(t.WorkDir,a.Path), a.Output); if err != nil { return "", err }; return "✅ Patch bundle written: "+out, nil }

type EnterpriseBuildDoctor struct{ WorkDir string }
func (t *EnterpriseBuildDoctor) Name() string { return "enterprise_build_doctor" }
func (t *EnterpriseBuildDoctor) Description() string { return "Parse real build/test output and classify failures: undefined symbol, unused import, type mismatch, cycles, missing module, test failures. Args: {\"output\":\"build output text\",\"json\":false}" }
func (t *EnterpriseBuildDoctor) RequiresApproval() bool { return false }
func (t *EnterpriseBuildDoctor) Schema() map[string]any { return map[string]any{"type":"object","properties":map[string]any{"output":map[string]any{"type":"string"},"json":map[string]any{"type":"boolean"}},"required":[]string{"output"}} }
func (t *EnterpriseBuildDoctor) Execute(_ context.Context, args string) (string, error) { var a struct{ Output string `json:"output"`; JSON bool `json:"json"` }; if err:=parseArgs(args,&a); err!=nil { return "", err }; r:=enterprise.BuildDoctor(a.Output); if a.JSON { return r.JSON(), nil }; return r.Summary(), nil }

type EnterpriseADRGenerator struct{ WorkDir string }
func (t *EnterpriseADRGenerator) Name() string { return "enterprise_adr_generator" }
func (t *EnterpriseADRGenerator) Description() string { return "Create a real Architecture Decision Record under docs/adr. Args: {title, context, decision, alternatives, consequences}" }
func (t *EnterpriseADRGenerator) RequiresApproval() bool { return true }
func (t *EnterpriseADRGenerator) Schema() map[string]any { return map[string]any{"type":"object","properties":map[string]any{"title":map[string]any{"type":"string"},"context":map[string]any{"type":"string"},"decision":map[string]any{"type":"string"},"alternatives":map[string]any{"type":"string"},"consequences":map[string]any{"type":"string"}},"required":[]string{"title","decision"}} }
func (t *EnterpriseADRGenerator) Execute(_ context.Context, args string) (string, error) { var a struct{ Title, Context, Decision, Alternatives, Consequences string }; if err:=parseArgs(args,&a); err!=nil { return "", err }; out, err := enterprise.GenerateADR(t.WorkDir,a.Title,a.Context,a.Decision,a.Alternatives,a.Consequences); if err!=nil { return "", err }; return "✅ ADR written: "+out, nil }

type EnterpriseCriticalPath struct{ WorkDir string }
func (t *EnterpriseCriticalPath) Name() string { return "enterprise_critical_path" }
func (t *EnterpriseCriticalPath) Description() string { return "Identify business-critical code paths such as auth, admin, payment, billing, crypto, secrets, migrations, delete/export. Args: {\"path\":\"optional\",\"json\":false}" }
func (t *EnterpriseCriticalPath) RequiresApproval() bool { return false }
func (t *EnterpriseCriticalPath) Schema() map[string]any { return enterprisePathSchema() }
func (t *EnterpriseCriticalPath) Execute(_ context.Context, args string) (string, error) { a,err:=enterpriseArgs(args); if err!=nil { return "",err }; r,err:=enterprise.CriticalPath(resolveEnterprisePath(t.WorkDir,a.Path)); if err!=nil { return "",err }; return formatEnterpriseReport(r,a.JSON), nil }

type EnterpriseAPIGuardian struct{ WorkDir string }
func (t *EnterpriseAPIGuardian) Name() string { return "enterprise_api_guardian" }
func (t *EnterpriseAPIGuardian) Description() string { return "Compare current API surface to .zed-api-baseline.json or write a baseline. Args: {\"baseline\":\"path\",\"write_baseline\":false,\"json\":false}" }
func (t *EnterpriseAPIGuardian) RequiresApproval() bool { return true }
func (t *EnterpriseAPIGuardian) Schema() map[string]any { return map[string]any{"type":"object","properties":map[string]any{"baseline":map[string]any{"type":"string"},"write_baseline":map[string]any{"type":"boolean"},"json":map[string]any{"type":"boolean"}}} }
func (t *EnterpriseAPIGuardian) Execute(_ context.Context, args string) (string, error) { var a struct{ Baseline string `json:"baseline"`; Write bool `json:"write_baseline"`; JSON bool `json:"json"` }; if strings.TrimSpace(args)!="" && strings.TrimSpace(args)!="{}" { if err:=parseArgs(args,&a); err!=nil { return "",err } }; baseline := a.Baseline; if baseline != "" { baseline = resolveEnterprisePath(t.WorkDir, baseline) }; r,err:=enterprise.APIGuardian(t.WorkDir, baseline, a.Write); if err!=nil { return "",err }; return formatEnterpriseReport(r,a.JSON), nil }

type EnterpriseGoConcurrencyAudit struct{ WorkDir string }
func (t *EnterpriseGoConcurrencyAudit) Name() string { return "enterprise_go_concurrency_audit" }
func (t *EnterpriseGoConcurrencyAudit) Description() string { return "Audit Go concurrency risks: goroutines without cancellation signals, time.Tick leaks, mutex copy risks. Args: {\"path\":\"optional\",\"json\":false}" }
func (t *EnterpriseGoConcurrencyAudit) RequiresApproval() bool { return false }
func (t *EnterpriseGoConcurrencyAudit) Schema() map[string]any { return enterprisePathSchema() }
func (t *EnterpriseGoConcurrencyAudit) Execute(_ context.Context, args string) (string, error) { a,err:=enterpriseArgs(args); if err!=nil { return "",err }; r,err:=enterprise.GoConcurrencyAudit(resolveEnterprisePath(t.WorkDir,a.Path)); if err!=nil { return "",err }; return formatEnterpriseReport(r,a.JSON), nil }

type EnterprisePerfHotspotPredictor struct{ WorkDir string }
func (t *EnterprisePerfHotspotPredictor) Name() string { return "enterprise_perf_hotspot_predictor" }
func (t *EnterprisePerfHotspotPredictor) Description() string { return "Predict performance hotspots from real code: nested loops, regex compile in loops, string concat in loops, unbounded reads. Args: {\"path\":\"optional\",\"json\":false}" }
func (t *EnterprisePerfHotspotPredictor) RequiresApproval() bool { return false }
func (t *EnterprisePerfHotspotPredictor) Schema() map[string]any { return enterprisePathSchema() }
func (t *EnterprisePerfHotspotPredictor) Execute(_ context.Context, args string) (string, error) { a,err:=enterpriseArgs(args); if err!=nil { return "",err }; r,err:=enterprise.PerfHotspotPredictor(resolveEnterprisePath(t.WorkDir,a.Path)); if err!=nil { return "",err }; return formatEnterpriseReport(r,a.JSON), nil }

type EnterpriseWorkJournal struct{ WorkDir string }
func (t *EnterpriseWorkJournal) Name() string { return "enterprise_work_journal" }
func (t *EnterpriseWorkJournal) Description() string { return "Append or read the agent engineering work journal .zed-work-journal.jsonl. Args: {\"action\":\"append|read\",\"task\":\"...\",\"evidence\":\"...\"}" }
func (t *EnterpriseWorkJournal) RequiresApproval() bool { return true }
func (t *EnterpriseWorkJournal) Schema() map[string]any { return map[string]any{"type":"object","properties":map[string]any{"action":map[string]any{"type":"string"},"task":map[string]any{"type":"string"},"evidence":map[string]any{"type":"string"},"files":map[string]any{"type":"string"},"risks":map[string]any{"type":"string"},"next":map[string]any{"type":"string"}}} }
func (t *EnterpriseWorkJournal) Execute(_ context.Context, args string) (string, error) { var a struct{ Action, Task, Evidence, Files, Risks, Next string }; if err:=parseArgs(args,&a); err!=nil { return "",err }; if a.Action=="read" { entries,err:=enterprise.ReadWorkJournal(t.WorkDir); if err!=nil { return "",err }; b,_:=json.MarshalIndent(entries,"","  "); return string(b),nil }; path,err:=enterprise.AppendWorkJournal(t.WorkDir, enterprise.WorkJournalEntry{Task:a.Task,Action:"append",Evidence:a.Evidence,Files:a.Files,Risks:a.Risks,Next:a.Next}); if err!=nil { return "",err }; return fmt.Sprintf("✅ Work journal updated: %s", path), nil }

var (
	_ Tool = (*EnterpriseArchitectureBrain)(nil)
	_ Tool = (*EnterpriseChangeImpact)(nil)
	_ Tool = (*EnterprisePatchBundle)(nil)
	_ Tool = (*EnterpriseBuildDoctor)(nil)
	_ Tool = (*EnterpriseADRGenerator)(nil)
	_ Tool = (*EnterpriseCriticalPath)(nil)
	_ Tool = (*EnterpriseAPIGuardian)(nil)
	_ Tool = (*EnterpriseGoConcurrencyAudit)(nil)
	_ Tool = (*EnterprisePerfHotspotPredictor)(nil)
	_ Tool = (*EnterpriseWorkJournal)(nil)
)
