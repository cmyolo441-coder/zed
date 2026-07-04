package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cmyolo441-coder/zed/internal/enterprise"
)

type EnterprisePolicyGate struct{ WorkDir string }
func (t *EnterprisePolicyGate) Name() string { return "enterprise_policy_gate" }
func (t *EnterprisePolicyGate) Description() string { return "Evaluate enterprise_release_gate against .zed-policy.json and .zed-risk-waivers.json. Args: {\"path\":\"optional path\",\"json\":false}" }
func (t *EnterprisePolicyGate) RequiresApproval() bool { return false }
func (t *EnterprisePolicyGate) Schema() map[string]any { return enterprisePathSchema() }
func (t *EnterprisePolicyGate) Execute(_ context.Context, args string) (string, error) { a, err := enterpriseArgs(args); if err != nil { return "", err }; r, decision, err := enterprise.PolicyGate(resolveEnterprisePath(t.WorkDir, a.Path)); if err != nil { return "", err }; if a.JSON { buf, _ := json.MarshalIndent(decision, "", "  "); return string(buf), nil }; return r.Summary(), nil }

type EnterpriseRiskWaiverAudit struct{ WorkDir string }
func (t *EnterpriseRiskWaiverAudit) Name() string { return "enterprise_risk_waiver_audit" }
func (t *EnterpriseRiskWaiverAudit) Description() string { return "Validate .zed-risk-waivers.json for expiry, approver, reason, and max-duration policy. Args: {\"path\":\"optional path\",\"json\":false}" }
func (t *EnterpriseRiskWaiverAudit) RequiresApproval() bool { return false }
func (t *EnterpriseRiskWaiverAudit) Schema() map[string]any { return enterprisePathSchema() }
func (t *EnterpriseRiskWaiverAudit) Execute(_ context.Context, args string) (string, error) { a, err := enterpriseArgs(args); if err != nil { return "", err }; r, err := enterprise.RiskWaiverAudit(resolveEnterprisePath(t.WorkDir, a.Path)); if err != nil { return "", err }; return formatEnterpriseReport(r, a.JSON), nil }

type EnterpriseSARIFExport struct{ WorkDir string }
func (t *EnterpriseSARIFExport) Name() string { return "enterprise_sarif_export" }
func (t *EnterpriseSARIFExport) Description() string { return "Export enterprise policy-gate findings as SARIF 2.1.0 for GitHub/GitLab security dashboards. Args: {\"path\":\"optional path\",\"output\":\"zed-enterprise.sarif\"}" }
func (t *EnterpriseSARIFExport) RequiresApproval() bool { return true }
func (t *EnterpriseSARIFExport) Schema() map[string]any { s := enterprisePathSchema(); props := s["properties"].(map[string]any); props["output"] = map[string]any{"type":"string"}; return s }
func (t *EnterpriseSARIFExport) Execute(_ context.Context, args string) (string, error) { a, err := enterpriseArgs(args); if err != nil { return "", err }; out, err := enterprise.WriteSARIF(resolveEnterprisePath(t.WorkDir, a.Path), a.Output); if err != nil { return "", err }; return "✅ SARIF written: " + out, nil }

type EnterpriseCITemplates struct{ WorkDir string }
func (t *EnterpriseCITemplates) Name() string { return "enterprise_ci_templates" }
func (t *EnterpriseCITemplates) Description() string { return "Write real GitHub Actions and GitLab CI enterprise gate templates that run tests and export SARIF. Args: {\"path\":\"optional path\"}" }
func (t *EnterpriseCITemplates) RequiresApproval() bool { return true }
func (t *EnterpriseCITemplates) Schema() map[string]any { return enterprisePathSchema() }
func (t *EnterpriseCITemplates) Execute(_ context.Context, args string) (string, error) { a, err := enterpriseArgs(args); if err != nil { return "", err }; files, err := enterprise.WriteCITemplates(resolveEnterprisePath(t.WorkDir, a.Path)); if err != nil { return "", err }; return "✅ CI templates written:\n  - " + strings.Join(files, "\n  - "), nil }

type EnterprisePolicyInit struct{ WorkDir string }
func (t *EnterprisePolicyInit) Name() string { return "enterprise_policy_init" }
func (t *EnterprisePolicyInit) Description() string { return "Return starter .zed-policy.json and .zed-risk-waivers.json content for enterprise governance. This does not write files. Args: {}" }
func (t *EnterprisePolicyInit) RequiresApproval() bool { return false }
func (t *EnterprisePolicyInit) Schema() map[string]any { return map[string]any{"type":"object","properties":map[string]any{}} }
func (t *EnterprisePolicyInit) Execute(_ context.Context, _ string) (string, error) { policy, _ := json.MarshalIndent(enterprise.DefaultReleasePolicy(), "", "  "); waivers := `[
  {
    "control": "EXAMPLE_CONTROL",
    "file": "optional/path.go",
    "reason": "Business-approved temporary exception with mitigation details.",
    "approver": "security@example.com",
    "expires_at": "2026-09-30",
    "ticket": "SEC-123"
  }
]`; return fmt.Sprintf(".zed-policy.json:\n```json\n%s\n```\n\n.zed-risk-waivers.json:\n```json\n%s\n```\n", string(policy), waivers), nil }

var (
	_ Tool = (*EnterprisePolicyGate)(nil)
	_ Tool = (*EnterpriseRiskWaiverAudit)(nil)
	_ Tool = (*EnterpriseSARIFExport)(nil)
	_ Tool = (*EnterpriseCITemplates)(nil)
	_ Tool = (*EnterprisePolicyInit)(nil)
)
