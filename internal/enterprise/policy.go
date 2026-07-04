package enterprise

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ReleasePolicy is a local enterprise policy file. It is intentionally JSON so
// the agent remains standard-library only and deterministic.
type ReleasePolicy struct {
	MaxCritical       int      `json:"max_critical"`
	MaxHigh           int      `json:"max_high"`
	MaxMedium         int      `json:"max_medium"`
	RequireApprover   bool     `json:"require_approver"`
	AllowedApprovers  []string `json:"allowed_approvers"`
	WaiverMaxDays     int      `json:"waiver_max_days"`
	BlockControls     []string `json:"block_controls"`
	WarnControls      []string `json:"warn_controls"`
}

// DefaultReleasePolicy is strict enough for enterprise release gates while
// allowing teams to override thresholds in .zed-policy.json.
func DefaultReleasePolicy() ReleasePolicy {
	return ReleasePolicy{
		MaxCritical:      0,
		MaxHigh:          0,
		MaxMedium:        10,
		RequireApprover:  true,
		WaiverMaxDays:    90,
		BlockControls:    []string{"HARDCODED_SECRET", "PRIVATE_KEY", "PRIVILEGED_CONTAINER", "TLS_VERIFY_DISABLED", "AUDIT_RECORD_TAMPERED"},
		WarnControls:     []string{"STRUCTURED_LOGGING_MISSING", "TRACING_MISSING", "HEALTH_ENDPOINT_MISSING"},
	}
}

// RiskWaiver is a real auditable exception record.
type RiskWaiver struct {
	Control   string `json:"control"`
	File      string `json:"file,omitempty"`
	Reason    string `json:"reason"`
	Approver  string `json:"approver"`
	ExpiresAt string `json:"expires_at"`
	Ticket    string `json:"ticket,omitempty"`
}

// PolicyDecision is the release decision after policy and waiver evaluation.
type PolicyDecision struct {
	Status          string       `json:"status"`
	Policy          ReleasePolicy `json:"policy"`
	UnwaivedCritical int         `json:"unwaived_critical"`
	UnwaivedHigh     int         `json:"unwaived_high"`
	UnwaivedMedium   int         `json:"unwaived_medium"`
	AppliedWaivers   []RiskWaiver `json:"applied_waivers"`
	Findings         []Finding    `json:"findings"`
}

func LoadReleasePolicy(root string) (ReleasePolicy, error) {
	policy := DefaultReleasePolicy()
	path := filepath.Join(root, ".zed-policy.json")
	buf, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return policy, nil
	}
	if err != nil {
		return policy, err
	}
	if err := json.Unmarshal(buf, &policy); err != nil {
		return policy, fmt.Errorf("parse .zed-policy.json: %w", err)
	}
	if policy.WaiverMaxDays == 0 {
		policy.WaiverMaxDays = 90
	}
	return policy, nil
}

func LoadRiskWaivers(root string) ([]RiskWaiver, error) {
	path := filepath.Join(root, ".zed-risk-waivers.json")
	buf, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var waivers []RiskWaiver
	if err := json.Unmarshal(buf, &waivers); err != nil {
		return nil, fmt.Errorf("parse .zed-risk-waivers.json: %w", err)
	}
	return waivers, nil
}

// RiskWaiverAudit validates exception records for expiry, approver, reason, and ticket hygiene.
func RiskWaiverAudit(root string) (Report, error) {
	report := Report{Name: "Risk Waiver Audit", Root: abs(root), GeneratedAt: time.Now()}
	policy, err := LoadReleasePolicy(root)
	if err != nil {
		return report, err
	}
	waivers, err := LoadRiskWaivers(root)
	if err != nil {
		return report, err
	}
	if len(waivers) == 0 {
		report.Findings = append(report.Findings, Finding{Control: "NO_RISK_WAIVERS", Severity: "INFO", Message: "No .zed-risk-waivers.json file found or no waivers defined"})
		return report, nil
	}
	now := time.Now()
	allowed := map[string]bool{}
	for _, a := range policy.AllowedApprovers { allowed[strings.ToLower(a)] = true }
	for i, w := range waivers {
		line := i + 1
		if w.Control == "" {
			report.Findings = append(report.Findings, Finding{Control: "WAIVER_CONTROL_MISSING", Severity: "HIGH", File: ".zed-risk-waivers.json", Line: line, Message: "Waiver has no control"})
		}
		if strings.TrimSpace(w.Reason) == "" {
			report.Findings = append(report.Findings, Finding{Control: "WAIVER_REASON_MISSING", Severity: "HIGH", File: ".zed-risk-waivers.json", Line: line, Message: "Waiver has no reason"})
		}
		if policy.RequireApprover && strings.TrimSpace(w.Approver) == "" {
			report.Findings = append(report.Findings, Finding{Control: "WAIVER_APPROVER_MISSING", Severity: "HIGH", File: ".zed-risk-waivers.json", Line: line, Message: "Waiver requires an approver"})
		}
		if len(allowed) > 0 && !allowed[strings.ToLower(w.Approver)] {
			report.Findings = append(report.Findings, Finding{Control: "WAIVER_APPROVER_NOT_ALLOWED", Severity: "HIGH", File: ".zed-risk-waivers.json", Line: line, Message: "Waiver approver is not in .zed-policy.json allowed_approvers", Evidence: w.Approver})
		}
		exp, err := time.Parse("2006-01-02", w.ExpiresAt)
		if err != nil {
			report.Findings = append(report.Findings, Finding{Control: "WAIVER_EXPIRY_INVALID", Severity: "HIGH", File: ".zed-risk-waivers.json", Line: line, Message: "Waiver expires_at must use YYYY-MM-DD", Evidence: w.ExpiresAt})
			continue
		}
		if now.After(exp) {
			report.Findings = append(report.Findings, Finding{Control: "WAIVER_EXPIRED", Severity: "CRITICAL", File: ".zed-risk-waivers.json", Line: line, Message: "Risk waiver is expired", Evidence: w.Control + " expires " + w.ExpiresAt})
		}
		if policy.WaiverMaxDays > 0 && exp.Sub(now) > time.Duration(policy.WaiverMaxDays)*24*time.Hour {
			report.Findings = append(report.Findings, Finding{Control: "WAIVER_TOO_LONG", Severity: "MEDIUM", File: ".zed-risk-waivers.json", Line: line, Message: "Risk waiver exceeds maximum allowed duration", Evidence: w.ExpiresAt})
		}
	}
	return report, nil
}

// PolicyGate evaluates the release gate against .zed-policy.json and .zed-risk-waivers.json.
func PolicyGate(root string) (Report, PolicyDecision, error) {
	policy, err := LoadReleasePolicy(root)
	if err != nil { return Report{}, PolicyDecision{}, err }
	waivers, err := LoadRiskWaivers(root)
	if err != nil { return Report{}, PolicyDecision{}, err }
	gate, err := ReleaseGate(root)
	if err != nil { return gate, PolicyDecision{}, err }
	decision := PolicyDecision{Status: "PASS", Policy: policy}
	now := time.Now()
	for _, f := range gate.Findings {
		if isFindingWaived(f, waivers, now) {
			for _, w := range waivers { if waiverMatches(w, f) { decision.AppliedWaivers = append(decision.AppliedWaivers, w); break } }
			continue
		}
		decision.Findings = append(decision.Findings, f)
		switch strings.ToUpper(f.Severity) {
		case "CRITICAL": decision.UnwaivedCritical++
		case "HIGH": decision.UnwaivedHigh++
		case "MEDIUM": decision.UnwaivedMedium++
		}
		if containsFold(policy.BlockControls, f.Control) { decision.Status = "BLOCK" }
	}
	if decision.UnwaivedCritical > policy.MaxCritical || decision.UnwaivedHigh > policy.MaxHigh || decision.UnwaivedMedium > policy.MaxMedium {
		decision.Status = "BLOCK"
	}
	report := Report{Name: "Enterprise Policy Gate", Root: abs(root), GeneratedAt: time.Now(), Findings: decision.Findings, Metadata: map[string]any{"decision": decision}}
	if decision.Status == "BLOCK" {
		report.Findings = append([]Finding{{Control: "POLICY_GATE_BLOCK", Severity: "CRITICAL", Message: "Enterprise policy gate blocked release", Evidence: fmt.Sprintf("critical=%d high=%d medium=%d", decision.UnwaivedCritical, decision.UnwaivedHigh, decision.UnwaivedMedium), Remediation: "Fix findings or add valid approved risk waivers."}}, report.Findings...)
	}
	return report, decision, nil
}

func isFindingWaived(f Finding, waivers []RiskWaiver, now time.Time) bool {
	for _, w := range waivers {
		if !waiverMatches(w, f) { continue }
		exp, err := time.Parse("2006-01-02", w.ExpiresAt)
		if err != nil || now.After(exp) { continue }
		if strings.TrimSpace(w.Reason) == "" || strings.TrimSpace(w.Approver) == "" { continue }
		return true
	}
	return false
}

func waiverMatches(w RiskWaiver, f Finding) bool {
	if !strings.EqualFold(w.Control, f.Control) { return false }
	if w.File != "" && filepath.ToSlash(w.File) != filepath.ToSlash(f.File) { return false }
	return true
}

func containsFold(items []string, target string) bool { for _, item := range items { if strings.EqualFold(item, target) { return true } }; return false }
