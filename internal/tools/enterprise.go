package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gjkjk/zed/internal/enterprise"
)

type EnterpriseSecretScan struct{ WorkDir string }
func (t *EnterpriseSecretScan) Name() string { return "enterprise_secret_scan" }
func (t *EnterpriseSecretScan) Description() string { return "Run a real high-signal secret scan over source/config files using credential regexes and entropy analysis. Args: {\"path\":\"optional path\",\"json\":false}" }
func (t *EnterpriseSecretScan) RequiresApproval() bool { return false }
func (t *EnterpriseSecretScan) Schema() map[string]any { return enterprisePathSchema() }
func (t *EnterpriseSecretScan) Execute(_ context.Context, args string) (string, error) { a, err := enterpriseArgs(args); if err != nil { return "", err }; r, err := enterprise.SecretScan(resolveEnterprisePath(t.WorkDir, a.Path)); if err != nil { return "", err }; return formatEnterpriseReport(r, a.JSON), nil }

type EnterpriseSBOM struct{ WorkDir string }
func (t *EnterpriseSBOM) Name() string { return "enterprise_sbom" }
func (t *EnterpriseSBOM) Description() string { return "Generate a real SBOM from go.mod, package.json, requirements.txt, and Cargo.toml. Args: {\"path\":\"optional path\",\"json\":false}" }
func (t *EnterpriseSBOM) RequiresApproval() bool { return false }
func (t *EnterpriseSBOM) Schema() map[string]any { return enterprisePathSchema() }
func (t *EnterpriseSBOM) Execute(_ context.Context, args string) (string, error) { a, err := enterpriseArgs(args); if err != nil { return "", err }; r, deps, err := enterprise.SBOM(resolveEnterprisePath(t.WorkDir, a.Path)); if err != nil { return "", err }; if a.JSON { return r.JSON(), nil }; var b strings.Builder; b.WriteString(r.Summary()); if len(deps) > 0 { b.WriteString("\nDependencies:\n"); for _, d := range deps { fmt.Fprintf(&b, "  - %s:%s@%s (%s)\n", d.Ecosystem, d.Name, d.Version, d.Source) } }; return b.String(), nil }

type EnterpriseLicenseAudit struct{ WorkDir string }
func (t *EnterpriseLicenseAudit) Name() string { return "enterprise_license_audit" }
func (t *EnterpriseLicenseAudit) Description() string { return "Audit repository and package license declarations, including missing/root and risky copyleft flags. Args: {\"path\":\"optional path\",\"json\":false}" }
func (t *EnterpriseLicenseAudit) RequiresApproval() bool { return false }
func (t *EnterpriseLicenseAudit) Schema() map[string]any { return enterprisePathSchema() }
func (t *EnterpriseLicenseAudit) Execute(_ context.Context, args string) (string, error) { a, err := enterpriseArgs(args); if err != nil { return "", err }; r, err := enterprise.LicenseAudit(resolveEnterprisePath(t.WorkDir, a.Path)); if err != nil { return "", err }; return formatEnterpriseReport(r, a.JSON), nil }

type EnterpriseSupplyChainAudit struct{ WorkDir string }
func (t *EnterpriseSupplyChainAudit) Name() string { return "enterprise_supply_chain_audit" }
func (t *EnterpriseSupplyChainAudit) Description() string { return "Audit dependency manifests for real supply-chain risk signals: unpinned versions, remote sources, and missing lockfiles. Args: {\"path\":\"optional path\",\"json\":false}" }
func (t *EnterpriseSupplyChainAudit) RequiresApproval() bool { return false }
func (t *EnterpriseSupplyChainAudit) Schema() map[string]any { return enterprisePathSchema() }
func (t *EnterpriseSupplyChainAudit) Execute(_ context.Context, args string) (string, error) { a, err := enterpriseArgs(args); if err != nil { return "", err }; r, err := enterprise.SupplyChainRisk(resolveEnterprisePath(t.WorkDir, a.Path)); if err != nil { return "", err }; return formatEnterpriseReport(r, a.JSON), nil }

type EnterprisePIIScan struct{ WorkDir string }
func (t *EnterprisePIIScan) Name() string { return "enterprise_pii_scan" }
func (t *EnterprisePIIScan) Description() string { return "Classify potential PII in source/config/test files: email, India PAN/Aadhaar, and Luhn-valid payment cards. Args: {\"path\":\"optional path\",\"json\":false}" }
func (t *EnterprisePIIScan) RequiresApproval() bool { return false }
func (t *EnterprisePIIScan) Schema() map[string]any { return enterprisePathSchema() }
func (t *EnterprisePIIScan) Execute(_ context.Context, args string) (string, error) { a, err := enterpriseArgs(args); if err != nil { return "", err }; r, err := enterprise.PIIScan(resolveEnterprisePath(t.WorkDir, a.Path)); if err != nil { return "", err }; return formatEnterpriseReport(r, a.JSON), nil }

type EnterpriseConfigAudit struct{ WorkDir string }
func (t *EnterpriseConfigAudit) Name() string { return "enterprise_config_audit" }
func (t *EnterpriseConfigAudit) Description() string { return "Audit config files for production-hardening mistakes such as debug enabled, TLS verification disabled, and exposed admin binds. Args: {\"path\":\"optional path\",\"json\":false}" }
func (t *EnterpriseConfigAudit) RequiresApproval() bool { return false }
func (t *EnterpriseConfigAudit) Schema() map[string]any { return enterprisePathSchema() }
func (t *EnterpriseConfigAudit) Execute(_ context.Context, args string) (string, error) { a, err := enterpriseArgs(args); if err != nil { return "", err }; r, err := enterprise.ConfigAudit(resolveEnterprisePath(t.WorkDir, a.Path)); if err != nil { return "", err }; return formatEnterpriseReport(r, a.JSON), nil }

type EnterprisePolicyAudit struct{ WorkDir string }
func (t *EnterprisePolicyAudit) Name() string { return "enterprise_policy_audit" }
func (t *EnterprisePolicyAudit) Description() string { return "Audit policy-as-code readiness: CODEOWNERS, SECURITY.md, and dangerous policy bypass markers. Args: {\"path\":\"optional path\",\"json\":false}" }
func (t *EnterprisePolicyAudit) RequiresApproval() bool { return false }
func (t *EnterprisePolicyAudit) Schema() map[string]any { return enterprisePathSchema() }
func (t *EnterprisePolicyAudit) Execute(_ context.Context, args string) (string, error) { a, err := enterpriseArgs(args); if err != nil { return "", err }; r, err := enterprise.PolicyAudit(resolveEnterprisePath(t.WorkDir, a.Path)); if err != nil { return "", err }; return formatEnterpriseReport(r, a.JSON), nil }

type EnterpriseSLOAudit struct{ WorkDir string }
func (t *EnterpriseSLOAudit) Name() string { return "enterprise_slo_audit" }
func (t *EnterpriseSLOAudit) Description() string { return "Validate .zed-slo.json service objectives and error-budget burn rates. Args: {\"path\":\"optional path\",\"json\":false}" }
func (t *EnterpriseSLOAudit) RequiresApproval() bool { return false }
func (t *EnterpriseSLOAudit) Schema() map[string]any { return enterprisePathSchema() }
func (t *EnterpriseSLOAudit) Execute(_ context.Context, args string) (string, error) { a, err := enterpriseArgs(args); if err != nil { return "", err }; r, err := enterprise.SLOAudit(resolveEnterprisePath(t.WorkDir, a.Path)); if err != nil { return "", err }; return formatEnterpriseReport(r, a.JSON), nil }

type EnterpriseIntegrityManifest struct{ WorkDir string }
func (t *EnterpriseIntegrityManifest) Name() string { return "enterprise_integrity_manifest" }
func (t *EnterpriseIntegrityManifest) Description() string { return "Create or verify a SHA-256 integrity manifest for the project tree. Args: {\"path\":\"optional path\",\"output\":\".zed-manifest.json\",\"verify\":\"manifest path\",\"json\":false}" }
func (t *EnterpriseIntegrityManifest) RequiresApproval() bool { return true }
func (t *EnterpriseIntegrityManifest) Schema() map[string]any { s := enterprisePathSchema(); props := s["properties"].(map[string]any); props["output"] = map[string]any{"type":"string"}; props["verify"] = map[string]any{"type":"string"}; return s }
func (t *EnterpriseIntegrityManifest) Execute(_ context.Context, args string) (string, error) { a, err := enterpriseArgs(args); if err != nil { return "", err }; root := resolveEnterprisePath(t.WorkDir, a.Path); if a.Verify != "" { vp := resolveEnterprisePath(t.WorkDir, a.Verify); buf, err := os.ReadFile(vp); if err != nil { return "", err }; r, err := enterprise.VerifyManifest(root, buf); if err != nil { return "", err }; return formatEnterpriseReport(r, a.JSON), nil }; out, err := enterprise.WriteManifest(root, a.Output); if err != nil { return "", err }; return "✅ Integrity manifest written: " + out, nil }

type EnterpriseAuditTrail struct{ WorkDir string }
func (t *EnterpriseAuditTrail) Name() string { return "enterprise_audit_trail" }
func (t *EnterpriseAuditTrail) Description() string { return "Append or verify a tamper-evident hash-chained audit trail in .zed-audit.jsonl. Args: {\"action\":\"verify|append\",\"actor\":\"name\",\"event\":\"action name\",\"target\":\"file/task\",\"details\":\"text\",\"json\":false}" }
func (t *EnterpriseAuditTrail) RequiresApproval() bool { return true }
func (t *EnterpriseAuditTrail) Schema() map[string]any { return map[string]any{"type":"object","properties":map[string]any{"action":map[string]any{"type":"string"},"actor":map[string]any{"type":"string"},"event":map[string]any{"type":"string"},"target":map[string]any{"type":"string"},"details":map[string]any{"type":"string"},"json":map[string]any{"type":"boolean"}},"required":[]string{"action"}} }
func (t *EnterpriseAuditTrail) Execute(_ context.Context, args string) (string, error) { a, err := enterpriseArgs(args); if err != nil { return "", err }; switch a.Action { case "verify", "": r, err := enterprise.AuditVerify(t.WorkDir); if err != nil { return "", err }; return formatEnterpriseReport(r, a.JSON), nil; case "append": rec, err := enterprise.AppendAudit(t.WorkDir, a.Actor, a.Event, a.Target, a.Details); if err != nil { return "", err }; if a.JSON { buf, _ := json.MarshalIndent(rec, "", "  "); return string(buf), nil }; return fmt.Sprintf("✅ Audit event appended: %s hash=%s", rec.Action, rec.Hash), nil; default: return "", fmt.Errorf("unknown action %q", a.Action) } }

type EnterpriseBackupReadiness struct{ WorkDir string }
func (t *EnterpriseBackupReadiness) Name() string { return "enterprise_backup_readiness" }
func (t *EnterpriseBackupReadiness) Description() string { return "Check backup/restore readiness using real file inventory size and integrity manifest presence. Args: {\"path\":\"optional path\",\"json\":false}" }
func (t *EnterpriseBackupReadiness) RequiresApproval() bool { return false }
func (t *EnterpriseBackupReadiness) Schema() map[string]any { return enterprisePathSchema() }
func (t *EnterpriseBackupReadiness) Execute(_ context.Context, args string) (string, error) { a, err := enterpriseArgs(args); if err != nil { return "", err }; r, err := enterprise.BackupPlan(resolveEnterprisePath(t.WorkDir, a.Path)); if err != nil { return "", err }; return formatEnterpriseReport(r, a.JSON), nil }

type enterpriseToolArgs struct { Path string `json:"path"`; JSON bool `json:"json"`; Output string `json:"output"`; Verify string `json:"verify"`; Action string `json:"action"`; Actor string `json:"actor"`; Event string `json:"event"`; Target string `json:"target"`; Details string `json:"details"` }
func enterpriseArgs(args string) (enterpriseToolArgs, error) { var a enterpriseToolArgs; if strings.TrimSpace(args) == "" || strings.TrimSpace(args) == "{}" { return a, nil }; if err := parseArgs(args, &a); err != nil { return a, fmt.Errorf("invalid arguments: %w", err) }; return a, nil }
func enterprisePathSchema() map[string]any { return map[string]any{"type":"object","properties":map[string]any{"path":map[string]any{"type":"string","description":"Optional file or directory path. Defaults to work dir."},"json":map[string]any{"type":"boolean","description":"Return machine-readable JSON."}}} }
func resolveEnterprisePath(workDir, p string) string { if p == "" { return workDir }; if filepath.IsAbs(p) { return p }; return filepath.Join(workDir, p) }
func formatEnterpriseReport(r enterprise.Report, asJSON bool) string { if asJSON { return r.JSON() }; return r.Summary() }

var (
	_ Tool = (*EnterpriseSecretScan)(nil)
	_ Tool = (*EnterpriseSBOM)(nil)
	_ Tool = (*EnterpriseLicenseAudit)(nil)
	_ Tool = (*EnterpriseSupplyChainAudit)(nil)
	_ Tool = (*EnterprisePIIScan)(nil)
	_ Tool = (*EnterpriseConfigAudit)(nil)
	_ Tool = (*EnterprisePolicyAudit)(nil)
	_ Tool = (*EnterpriseSLOAudit)(nil)
	_ Tool = (*EnterpriseIntegrityManifest)(nil)
	_ Tool = (*EnterpriseAuditTrail)(nil)
	_ Tool = (*EnterpriseBackupReadiness)(nil)
)
