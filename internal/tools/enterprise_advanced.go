package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/cmyolo441-coder/zed/internal/enterprise"
)

type EnterpriseAPISurface struct{ WorkDir string }
func (t *EnterpriseAPISurface) Name() string { return "enterprise_api_surface" }
func (t *EnterpriseAPISurface) Description() string { return "Discover real HTTP/API endpoints in Go, Node, and Python source and flag sensitive or unconstrained routes. Args: {\"path\":\"optional path\",\"json\":false}" }
func (t *EnterpriseAPISurface) RequiresApproval() bool { return false }
func (t *EnterpriseAPISurface) Schema() map[string]any { return enterprisePathSchema() }
func (t *EnterpriseAPISurface) Execute(_ context.Context, args string) (string, error) { a, err := enterpriseArgs(args); if err != nil { return "", err }; r, eps, err := enterprise.APISurfaceInventory(resolveEnterprisePath(t.WorkDir, a.Path)); if err != nil { return "", err }; if a.JSON { return r.JSON(), nil }; var b strings.Builder; b.WriteString(r.Summary()); if len(eps) > 0 { b.WriteString("\nEndpoints:\n"); for _, ep := range eps { fmt.Fprintf(&b, "  - %s %s (%s:%d)\n", ep.Method, ep.Path, ep.File, ep.Line) } }; return b.String(), nil }

type EnterpriseObservabilityAudit struct{ WorkDir string }
func (t *EnterpriseObservabilityAudit) Name() string { return "enterprise_observability_audit" }
func (t *EnterpriseObservabilityAudit) Description() string { return "Audit real observability readiness: structured logs, metrics, traces, health checks, and panic recovery. Args: {\"path\":\"optional path\",\"json\":false}" }
func (t *EnterpriseObservabilityAudit) RequiresApproval() bool { return false }
func (t *EnterpriseObservabilityAudit) Schema() map[string]any { return enterprisePathSchema() }
func (t *EnterpriseObservabilityAudit) Execute(_ context.Context, args string) (string, error) { a, err := enterpriseArgs(args); if err != nil { return "", err }; r, err := enterprise.ObservabilityAudit(resolveEnterprisePath(t.WorkDir, a.Path)); if err != nil { return "", err }; return formatEnterpriseReport(r, a.JSON), nil }

type EnterpriseContainerAudit struct{ WorkDir string }
func (t *EnterpriseContainerAudit) Name() string { return "enterprise_container_audit" }
func (t *EnterpriseContainerAudit) Description() string { return "Audit Dockerfile and Kubernetes YAML hardening risks: latest tags, root users, privileged containers, escalation, writable root FS. Args: {\"path\":\"optional path\",\"json\":false}" }
func (t *EnterpriseContainerAudit) RequiresApproval() bool { return false }
func (t *EnterpriseContainerAudit) Schema() map[string]any { return enterprisePathSchema() }
func (t *EnterpriseContainerAudit) Execute(_ context.Context, args string) (string, error) { a, err := enterpriseArgs(args); if err != nil { return "", err }; r, err := enterprise.ContainerAudit(resolveEnterprisePath(t.WorkDir, a.Path)); if err != nil { return "", err }; return formatEnterpriseReport(r, a.JSON), nil }

type EnterpriseCodeOwnershipAudit struct{ WorkDir string }
func (t *EnterpriseCodeOwnershipAudit) Name() string { return "enterprise_code_ownership_audit" }
func (t *EnterpriseCodeOwnershipAudit) Description() string { return "Check CODEOWNERS coverage for real source/config files and report unowned files. Args: {\"path\":\"optional path\",\"json\":false}" }
func (t *EnterpriseCodeOwnershipAudit) RequiresApproval() bool { return false }
func (t *EnterpriseCodeOwnershipAudit) Schema() map[string]any { return enterprisePathSchema() }
func (t *EnterpriseCodeOwnershipAudit) Execute(_ context.Context, args string) (string, error) { a, err := enterpriseArgs(args); if err != nil { return "", err }; r, err := enterprise.CodeOwnershipAudit(resolveEnterprisePath(t.WorkDir, a.Path)); if err != nil { return "", err }; return formatEnterpriseReport(r, a.JSON), nil }

type EnterpriseThreatModel struct{ WorkDir string }
func (t *EnterpriseThreatModel) Name() string { return "enterprise_threat_model" }
func (t *EnterpriseThreatModel) Description() string { return "Generate a real STRIDE threat model from discovered API endpoints and concrete code locations. Args: {\"path\":\"optional path\",\"json\":false}" }
func (t *EnterpriseThreatModel) RequiresApproval() bool { return false }
func (t *EnterpriseThreatModel) Schema() map[string]any { return enterprisePathSchema() }
func (t *EnterpriseThreatModel) Execute(_ context.Context, args string) (string, error) { a, err := enterpriseArgs(args); if err != nil { return "", err }; r, err := enterprise.ThreatModel(resolveEnterprisePath(t.WorkDir, a.Path)); if err != nil { return "", err }; return formatEnterpriseReport(r, a.JSON), nil }

type EnterpriseReleaseGate struct{ WorkDir string }
func (t *EnterpriseReleaseGate) Name() string { return "enterprise_release_gate" }
func (t *EnterpriseReleaseGate) Description() string { return "Run combined enterprise release gate across secrets, supply chain, config, policy, observability, containers, and ownership. Args: {\"path\":\"optional path\",\"json\":false}" }
func (t *EnterpriseReleaseGate) RequiresApproval() bool { return false }
func (t *EnterpriseReleaseGate) Schema() map[string]any { return enterprisePathSchema() }
func (t *EnterpriseReleaseGate) Execute(_ context.Context, args string) (string, error) { a, err := enterpriseArgs(args); if err != nil { return "", err }; r, err := enterprise.ReleaseGate(resolveEnterprisePath(t.WorkDir, a.Path)); if err != nil { return "", err }; return formatEnterpriseReport(r, a.JSON), nil }

type EnterpriseEvidencePack struct{ WorkDir string }
func (t *EnterpriseEvidencePack) Name() string { return "enterprise_evidence_pack" }
func (t *EnterpriseEvidencePack) Description() string { return "Write a real JSON compliance/evidence bundle containing multiple enterprise audit reports. Args: {\"path\":\"optional path\",\"output\":\".zed-evidence-pack.json\"}" }
func (t *EnterpriseEvidencePack) RequiresApproval() bool { return true }
func (t *EnterpriseEvidencePack) Schema() map[string]any { s := enterprisePathSchema(); props := s["properties"].(map[string]any); props["output"] = map[string]any{"type":"string"}; return s }
func (t *EnterpriseEvidencePack) Execute(_ context.Context, args string) (string, error) { a, err := enterpriseArgs(args); if err != nil { return "", err }; out, err := enterprise.EvidencePack(resolveEnterprisePath(t.WorkDir, a.Path), resolveOutput(t.WorkDir, a.Output)); if err != nil { return "", err }; if a.JSON { buf, _ := json.MarshalIndent(map[string]string{"output": out}, "", "  "); return string(buf), nil }; return "✅ Evidence pack written: " + out, nil }

func resolveOutput(workDir, p string) string { if p == "" { return "" }; if filepath.IsAbs(p) { return p }; return filepath.Join(workDir, p) }

var (
	_ Tool = (*EnterpriseAPISurface)(nil)
	_ Tool = (*EnterpriseObservabilityAudit)(nil)
	_ Tool = (*EnterpriseContainerAudit)(nil)
	_ Tool = (*EnterpriseCodeOwnershipAudit)(nil)
	_ Tool = (*EnterpriseThreatModel)(nil)
	_ Tool = (*EnterpriseReleaseGate)(nil)
	_ Tool = (*EnterpriseEvidencePack)(nil)
)
