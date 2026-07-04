package tools

import (
	"context"
	"encoding/json"
	"os"

	"github.com/cmyolo441-coder/zed/internal/enterprise"
)

type EnterpriseTaintAnalysis struct{ WorkDir string }
func (t *EnterpriseTaintAnalysis) Name() string { return "enterprise_taint_analysis" }
func (t *EnterpriseTaintAnalysis) Description() string { return "Run real local source-to-sink taint proximity analysis for SQL, command, path traversal, and XSS risks. Args: {\"path\":\"optional path\",\"json\":false}" }
func (t *EnterpriseTaintAnalysis) RequiresApproval() bool { return false }
func (t *EnterpriseTaintAnalysis) Schema() map[string]any { return enterprisePathSchema() }
func (t *EnterpriseTaintAnalysis) Execute(_ context.Context, args string) (string, error) { a, err := enterpriseArgs(args); if err != nil { return "", err }; r, err := enterprise.TaintAnalysis(resolveEnterprisePath(t.WorkDir, a.Path)); if err != nil { return "", err }; return formatEnterpriseReport(r, a.JSON), nil }

type EnterpriseMigrationAudit struct{ WorkDir string }
func (t *EnterpriseMigrationAudit) Name() string { return "enterprise_migration_audit" }
func (t *EnterpriseMigrationAudit) Description() string { return "Audit SQL/database migrations for unsafe production operations such as DROP, unbounded UPDATE/DELETE, and blocking indexes. Args: {\"path\":\"optional path\",\"json\":false}" }
func (t *EnterpriseMigrationAudit) RequiresApproval() bool { return false }
func (t *EnterpriseMigrationAudit) Schema() map[string]any { return enterprisePathSchema() }
func (t *EnterpriseMigrationAudit) Execute(_ context.Context, args string) (string, error) { a, err := enterpriseArgs(args); if err != nil { return "", err }; r, err := enterprise.MigrationSafetyAudit(resolveEnterprisePath(t.WorkDir, a.Path)); if err != nil { return "", err }; return formatEnterpriseReport(r, a.JSON), nil }

type EnterpriseEvidenceKeygen struct{ WorkDir string }
func (t *EnterpriseEvidenceKeygen) Name() string { return "enterprise_evidence_keygen" }
func (t *EnterpriseEvidenceKeygen) Description() string { return "Generate an Ed25519 keypair for signing evidence packs. Returns base64 public/private keys; store private key securely. Args: {}" }
func (t *EnterpriseEvidenceKeygen) RequiresApproval() bool { return false }
func (t *EnterpriseEvidenceKeygen) Schema() map[string]any { return map[string]any{"type":"object","properties":map[string]any{}} }
func (t *EnterpriseEvidenceKeygen) Execute(_ context.Context, _ string) (string, error) { pub, priv, err := enterprise.GenerateEvidenceKeypair(); if err != nil { return "", err }; buf, _ := json.MarshalIndent(map[string]string{"public_key": pub, "private_key": priv}, "", "  "); return string(buf), nil }

type EnterpriseEvidenceSign struct{ WorkDir string }
func (t *EnterpriseEvidenceSign) Name() string { return "enterprise_evidence_sign" }
func (t *EnterpriseEvidenceSign) Description() string { return "Sign an evidence pack with an Ed25519 private key. Args: {\"file\":\".zed-evidence-pack.json\",\"private_key\":\"base64\"}" }
func (t *EnterpriseEvidenceSign) RequiresApproval() bool { return true }
func (t *EnterpriseEvidenceSign) Schema() map[string]any { return map[string]any{"type":"object","properties":map[string]any{"file":map[string]any{"type":"string"},"private_key":map[string]any{"type":"string"}},"required":[]string{"file","private_key"}} }
func (t *EnterpriseEvidenceSign) Execute(_ context.Context, args string) (string, error) { var a struct{ File string `json:"file"`; PrivateKey string `json:"private_key"` }; if err := parseArgs(args, &a); err != nil { return "", err }; out, err := enterprise.WriteEvidenceSignature(resolveEnterprisePath(t.WorkDir, a.File), a.PrivateKey); if err != nil { return "", err }; return "✅ Evidence signature written: " + out, nil }

type EnterpriseEvidenceVerify struct{ WorkDir string }
func (t *EnterpriseEvidenceVerify) Name() string { return "enterprise_evidence_verify" }
func (t *EnterpriseEvidenceVerify) Description() string { return "Verify an evidence pack signature JSON. Args: {\"file\":\".zed-evidence-pack.json\",\"signature\":\".zed-evidence-pack.json.sig.json\"}" }
func (t *EnterpriseEvidenceVerify) RequiresApproval() bool { return false }
func (t *EnterpriseEvidenceVerify) Schema() map[string]any { return map[string]any{"type":"object","properties":map[string]any{"file":map[string]any{"type":"string"},"signature":map[string]any{"type":"string"}},"required":[]string{"file","signature"}} }
func (t *EnterpriseEvidenceVerify) Execute(_ context.Context, args string) (string, error) { var a struct{ File string `json:"file"`; Signature string `json:"signature"` }; if err := parseArgs(args, &a); err != nil { return "", err }; buf, err := os.ReadFile(resolveEnterprisePath(t.WorkDir, a.Signature)); if err != nil { return "", err }; var sig enterprise.EvidenceSignature; if err := json.Unmarshal(buf, &sig); err != nil { return "", err }; if err := enterprise.VerifyEvidenceFile(resolveEnterprisePath(t.WorkDir, a.File), sig); err != nil { return "", err }; return "✅ Evidence signature verified", nil }

type EnterpriseComplianceMap struct{ WorkDir string }
func (t *EnterpriseComplianceMap) Name() string { return "enterprise_compliance_map" }
func (t *EnterpriseComplianceMap) Description() string { return "Map a finding control to SOC2, ISO27001, NIST, and OWASP references. Args: {\"control\":\"HARDCODED_SECRET\"}" }
func (t *EnterpriseComplianceMap) RequiresApproval() bool { return false }
func (t *EnterpriseComplianceMap) Schema() map[string]any { return map[string]any{"type":"object","properties":map[string]any{"control":map[string]any{"type":"string"}},"required":[]string{"control"}} }
func (t *EnterpriseComplianceMap) Execute(_ context.Context, args string) (string, error) { var a struct{ Control string `json:"control"` }; if err := parseArgs(args, &a); err != nil { return "", err }; m := enterprise.MapCompliance(a.Control); buf, _ := json.MarshalIndent(m, "", "  "); return string(buf), nil }

var (
	_ Tool = (*EnterpriseTaintAnalysis)(nil)
	_ Tool = (*EnterpriseMigrationAudit)(nil)
	_ Tool = (*EnterpriseEvidenceKeygen)(nil)
	_ Tool = (*EnterpriseEvidenceSign)(nil)
	_ Tool = (*EnterpriseEvidenceVerify)(nil)
	_ Tool = (*EnterpriseComplianceMap)(nil)
)
