// Package enterprise contains production-grade, deterministic engineering
// controls used by ZED tools. The code in this package uses real local evidence rather than mocked
// external platforms; every analyzer works from real project files, command
// manifests, cryptographic hashes, and auditable JSON records.
package enterprise

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const maxScannedFileBytes = 2 << 20

// Finding is a normalized enterprise-control result.
type Finding struct {
	Control     string  `json:"control"`
	Severity    string  `json:"severity"`
	File        string  `json:"file,omitempty"`
	Line        int     `json:"line,omitempty"`
	Message     string  `json:"message"`
	Evidence    string  `json:"evidence,omitempty"`
	Remediation string  `json:"remediation,omitempty"`
	Score       float64 `json:"score,omitempty"`
}

// Report is returned by every enterprise analyzer.
type Report struct {
	Name        string    `json:"name"`
	Root        string    `json:"root"`
	GeneratedAt time.Time `json:"generated_at"`
	Findings    []Finding `json:"findings"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// Summary renders a deterministic human-readable report.
func (r Report) Summary() string {
	var b strings.Builder
	fmt.Fprintf(&b, "🏢 %s\n", r.Name)
	fmt.Fprintf(&b, "Root: %s\n", r.Root)
	fmt.Fprintf(&b, "Generated: %s\n", r.GeneratedAt.Format(time.RFC3339))
	counts := map[string]int{}
	for _, f := range r.Findings {
		counts[strings.ToUpper(f.Severity)]++
	}
	fmt.Fprintf(&b, "Findings: %d critical=%d high=%d medium=%d low=%d info=%d\n\n",
		len(r.Findings), counts["CRITICAL"], counts["HIGH"], counts["MEDIUM"], counts["LOW"], counts["INFO"])
	if len(r.Findings) == 0 {
		b.WriteString("✅ No findings for this control.\n")
		return b.String()
	}
	for i, f := range r.Findings {
		loc := f.File
		if f.Line > 0 {
			loc = fmt.Sprintf("%s:%d", f.File, f.Line)
		}
		fmt.Fprintf(&b, "%d. [%s] %s", i+1, strings.ToUpper(f.Severity), f.Control)
		if loc != "" {
			fmt.Fprintf(&b, " — %s", loc)
		}
		fmt.Fprintf(&b, "\n   %s\n", f.Message)
		if f.Evidence != "" {
			fmt.Fprintf(&b, "   Evidence: %s\n", f.Evidence)
		}
		if f.Remediation != "" {
			fmt.Fprintf(&b, "   Fix: %s\n", f.Remediation)
		}
	}
	return b.String()
}

// JSON renders the full machine-readable report.
func (r Report) JSON() string {
	buf, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"error":%q}`, err.Error())
	}
	return string(buf)
}

// FileHash identifies a real file by cryptographic digest.
type FileHash struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

// Manifest captures integrity metadata for a project tree.
type Manifest struct {
	Root        string     `json:"root"`
	GeneratedAt time.Time  `json:"generated_at"`
	Files       []FileHash `json:"files"`
}

// BuildManifest walks a directory and hashes source/config files.
func BuildManifest(root string) (*Manifest, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	m := &Manifest{Root: root, GeneratedAt: time.Now()}
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if skipDir(d.Name()) && path != root {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := d.Info()
		if err != nil || info.Size() > 50<<20 {
			return err
		}
		h, err := hashFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		m.Files = append(m.Files, FileHash{Path: filepath.ToSlash(rel), Size: info.Size(), SHA256: h})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(m.Files, func(i, j int) bool { return m.Files[i].Path < m.Files[j].Path })
	return m, nil
}

// VerifyManifest compares the current tree against a saved manifest JSON.
func VerifyManifest(root string, manifestJSON []byte) (Report, error) {
	var old Manifest
	if err := json.Unmarshal(manifestJSON, &old); err != nil {
		return Report{}, err
	}
	current, err := BuildManifest(root)
	if err != nil {
		return Report{}, err
	}
	report := Report{Name: "Integrity Manifest Verification", Root: current.Root, GeneratedAt: time.Now()}
	before := map[string]FileHash{}
	for _, f := range old.Files {
		before[f.Path] = f
	}
	after := map[string]FileHash{}
	for _, f := range current.Files {
		after[f.Path] = f
	}
	for path, oldHash := range before {
		newHash, ok := after[path]
		if !ok {
			report.Findings = append(report.Findings, Finding{Control: "FILE_REMOVED", Severity: "HIGH", File: path, Message: "File existed in manifest but is missing now", Evidence: oldHash.SHA256})
			continue
		}
		if oldHash.SHA256 != newHash.SHA256 {
			report.Findings = append(report.Findings, Finding{Control: "FILE_MODIFIED", Severity: "MEDIUM", File: path, Message: "File content hash changed", Evidence: oldHash.SHA256 + " -> " + newHash.SHA256})
		}
	}
	for path, newHash := range after {
		if _, ok := before[path]; !ok {
			report.Findings = append(report.Findings, Finding{Control: "FILE_ADDED", Severity: "LOW", File: path, Message: "File is not present in saved manifest", Evidence: newHash.SHA256})
		}
	}
	return report, nil
}

// WriteManifest writes an integrity manifest to disk.
func WriteManifest(root, output string) (string, error) {
	m, err := BuildManifest(root)
	if err != nil {
		return "", err
	}
	buf, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return "", err
	}
	if output == "" {
		output = filepath.Join(root, ".zed-manifest.json")
	}
	if !filepath.IsAbs(output) {
		output = filepath.Join(root, output)
	}
	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		return "", err
	}
	if err := os.WriteFile(output, buf, 0644); err != nil {
		return "", err
	}
	return output, nil
}

// SecretScan detects real high-signal secrets and high-entropy tokens.
func SecretScan(root string) (Report, error) {
	report := Report{Name: "Secret Scanner", Root: abs(root), GeneratedAt: time.Now()}
	patterns := []struct{ name, sev, expr, fix string }{
		{"AWS_ACCESS_KEY", "CRITICAL", `AKIA[0-9A-Z]{16}`, "Rotate the AWS key and move credentials into a secrets manager."},
		{"GITHUB_TOKEN", "CRITICAL", `gh[pousr]_[A-Za-z0-9_]{36,}`, "Revoke the GitHub token and use environment-based credentials."},
		{"SLACK_TOKEN", "CRITICAL", `xox[baprs]-[A-Za-z0-9-]{20,}`, "Revoke the Slack token and store it outside source control."},
		{"PRIVATE_KEY", "CRITICAL", `-----BEGIN (?:RSA |EC |OPENSSH |DSA )?PRIVATE KEY-----`, "Remove private keys from the repository and rotate impacted credentials."},
		{"GENERIC_SECRET_ASSIGNMENT", "HIGH", `(?i)(api[_-]?key|secret|password|token|client[_-]?secret)\s*[:=]\s*["'][^"'\s]{12,}["']`, "Use environment variables or a vault-backed secret provider."},
	}
	compiled := make([]struct{ name, sev string; re *regexp.Regexp; fix string }, 0, len(patterns))
	for _, p := range patterns {
		compiled = append(compiled, struct{ name, sev string; re *regexp.Regexp; fix string }{p.name, p.sev, regexp.MustCompile(p.expr), p.fix})
	}
	err := walkText(root, func(path string, lineNo int, line string) {
		for _, p := range compiled {
			if match := p.re.FindString(line); match != "" {
				report.Findings = append(report.Findings, Finding{Control: p.name, Severity: p.sev, File: rel(root, path), Line: lineNo, Message: "Potential credential material found", Evidence: redact(match), Remediation: p.fix})
			}
		}
		for _, token := range candidateTokens(line) {
			if len(token) >= 32 && shannon(token) >= 4.2 && !looksLikeHash(token) {
				report.Findings = append(report.Findings, Finding{Control: "HIGH_ENTROPY_TOKEN", Severity: "MEDIUM", File: rel(root, path), Line: lineNo, Message: "High-entropy token-like value found", Evidence: redact(token), Remediation: "Confirm whether this is a secret; if yes, rotate it and load from a secrets manager."})
			}
		}
	})
	return report, err
}

// Dependency represents a discovered third-party package.
type Dependency struct {
	Ecosystem string `json:"ecosystem"`
	Name      string `json:"name"`
	Version   string `json:"version,omitempty"`
	Source    string `json:"source"`
}

// SBOM builds a real software bill of materials from common manifest files.
func SBOM(root string) (Report, []Dependency, error) {
	var deps []Dependency
	collectGoMod(root, &deps)
	collectPackageJSON(root, &deps)
	collectRequirements(root, &deps)
	collectCargo(root, &deps)
	sort.Slice(deps, func(i, j int) bool {
		if deps[i].Ecosystem != deps[j].Ecosystem { return deps[i].Ecosystem < deps[j].Ecosystem }
		return deps[i].Name < deps[j].Name
	})
	report := Report{Name: "Software Bill of Materials", Root: abs(root), GeneratedAt: time.Now(), Metadata: map[string]any{"dependencies": deps, "count": len(deps)}}
	if len(deps) == 0 {
		report.Findings = append(report.Findings, Finding{Control: "SBOM_EMPTY", Severity: "INFO", Message: "No supported dependency manifests found"})
	}
	seen := map[string]bool{}
	for _, d := range deps {
		key := d.Ecosystem + ":" + d.Name
		if seen[key] {
			report.Findings = append(report.Findings, Finding{Control: "DUPLICATE_DEPENDENCY", Severity: "LOW", File: d.Source, Message: "Dependency appears multiple times", Evidence: key})
		}
		seen[key] = true
	}
	return report, deps, nil
}

// LicenseAudit identifies declared package licenses and flags missing or risky terms.
func LicenseAudit(root string) (Report, error) {
	report := Report{Name: "License Audit", Root: abs(root), GeneratedAt: time.Now()}
	// Root license.
	if !existsAny(root, "LICENSE", "LICENSE.md", "LICENSE.txt", "COPYING") {
		report.Findings = append(report.Findings, Finding{Control: "MISSING_ROOT_LICENSE", Severity: "MEDIUM", Message: "Repository has no root LICENSE file", Remediation: "Add an approved license file or document proprietary ownership."})
	}
	// package.json license.
	filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() { return err }
		if skipDir(d.Name()) { return nil }
		if d.Name() == "package.json" {
			var pkg struct{ License any `json:"license"` }
			if buf, err := os.ReadFile(path); err == nil && json.Unmarshal(buf, &pkg) == nil {
				license := fmt.Sprint(pkg.License)
				if license == "<nil>" || strings.TrimSpace(license) == "" {
					report.Findings = append(report.Findings, Finding{Control: "PACKAGE_LICENSE_MISSING", Severity: "MEDIUM", File: rel(root, path), Message: "package.json has no license field"})
				} else if riskyLicense(license) {
					report.Findings = append(report.Findings, Finding{Control: "RISKY_LICENSE", Severity: "HIGH", File: rel(root, path), Message: "License may impose copyleft/commercial obligations", Evidence: license, Remediation: "Review with legal/security before release."})
				}
			}
		}
		return nil
	})
	return report, nil
}

// SupplyChainRisk checks dependency manifests for risky patterns without using mocked vulnerability data.
func SupplyChainRisk(root string) (Report, error) {
	report := Report{Name: "Supply Chain Risk Audit", Root: abs(root), GeneratedAt: time.Now()}
	_, deps, _ := SBOM(root)
	for _, d := range deps {
		if d.Version == "" || strings.EqualFold(d.Version, "latest") || strings.ContainsAny(d.Version, "*xX^~<>") || strings.Contains(d.Version, ">=") || strings.Contains(d.Version, "<=") {
			report.Findings = append(report.Findings, Finding{Control: "UNPINNED_DEPENDENCY", Severity: "HIGH", File: d.Source, Message: "Dependency is not pinned to an exact immutable version", Evidence: d.Ecosystem + ":" + d.Name + "@" + d.Version, Remediation: "Pin exact versions and commit lockfiles."})
		}
		if strings.HasPrefix(d.Version, "http://") || strings.Contains(d.Version, "git+") || strings.Contains(d.Version, "github:") {
			report.Findings = append(report.Findings, Finding{Control: "REMOTE_DEPENDENCY_SOURCE", Severity: "MEDIUM", File: d.Source, Message: "Dependency references remote source outside normal registry", Evidence: d.Version, Remediation: "Prefer verified registries and lock immutable checksums."})
		}
	}
	if existsAny(root, "package.json") && !existsAny(root, "package-lock.json", "npm-shrinkwrap.json", "yarn.lock", "pnpm-lock.yaml") {
		report.Findings = append(report.Findings, Finding{Control: "MISSING_NODE_LOCKFILE", Severity: "HIGH", Message: "Node project has package.json but no lockfile", Remediation: "Generate and commit a package manager lockfile."})
	}
	return report, nil
}

// PIIScan detects likely personal data in source/config files.
func PIIScan(root string) (Report, error) {
	report := Report{Name: "PII/Data Classification Scan", Root: abs(root), GeneratedAt: time.Now()}
	patterns := []struct{ name, sev, expr string }{
		{"EMAIL_ADDRESS", "LOW", `[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}`},
		{"INDIA_PAN", "HIGH", `\b[A-Z]{5}[0-9]{4}[A-Z]\b`},
		{"INDIA_AADHAAR", "CRITICAL", `\b[2-9][0-9]{3}\s?[0-9]{4}\s?[0-9]{4}\b`},
		{"CREDIT_CARD", "CRITICAL", `\b(?:4[0-9]{12}(?:[0-9]{3})?|5[1-5][0-9]{14}|3[47][0-9]{13}|6(?:011|5[0-9]{2})[0-9]{12})\b`},
	}
	compiled := make([]struct{ name, sev string; re *regexp.Regexp }, 0, len(patterns))
	for _, p := range patterns { compiled = append(compiled, struct{ name, sev string; re *regexp.Regexp }{p.name, p.sev, regexp.MustCompile(p.expr)}) }
	err := walkText(root, func(path string, lineNo int, line string) {
		for _, p := range compiled {
			for _, match := range p.re.FindAllString(line, -1) {
				if p.name == "CREDIT_CARD" && !luhn(match) { continue }
				report.Findings = append(report.Findings, Finding{Control: p.name, Severity: p.sev, File: rel(root, path), Line: lineNo, Message: "Potential regulated personal data found", Evidence: redact(match), Remediation: "Do not commit production PII; use synthetic fixtures and access controls."})
			}
		}
	})
	return report, err
}

// ConfigAudit validates production hardening settings in common config files.
func ConfigAudit(root string) (Report, error) {
	report := Report{Name: "Configuration Hardening Audit", Root: abs(root), GeneratedAt: time.Now()}
	err := walkText(root, func(path string, lineNo int, line string) {
		low := strings.ToLower(line)
		switch {
		case strings.Contains(low, "debug=true") || strings.Contains(low, "debug: true"):
			report.Findings = append(report.Findings, Finding{Control: "DEBUG_ENABLED", Severity: "HIGH", File: rel(root, path), Line: lineNo, Message: "Debug mode appears enabled", Remediation: "Disable debug in production profiles."})
		case strings.Contains(low, "tls_verify=false") || strings.Contains(low, "insecureskipverify: true"):
			report.Findings = append(report.Findings, Finding{Control: "TLS_VERIFY_DISABLED", Severity: "CRITICAL", File: rel(root, path), Line: lineNo, Message: "TLS certificate verification appears disabled", Remediation: "Enable certificate verification and manage trusted CAs."})
		case strings.Contains(low, "0.0.0.0") && (strings.Contains(low, "admin") || strings.Contains(low, "management")):
			report.Findings = append(report.Findings, Finding{Control: "ADMIN_BIND_ALL_INTERFACES", Severity: "HIGH", File: rel(root, path), Line: lineNo, Message: "Admin/management endpoint may bind all interfaces", Remediation: "Bind admin endpoints to localhost/private networks and require auth."})
		}
	})
	return report, err
}

// PolicyAudit scans for minimum enterprise policy files and unsafe exemptions.
func PolicyAudit(root string) (Report, error) {
	report := Report{Name: "Policy-as-Code Audit", Root: abs(root), GeneratedAt: time.Now()}
	required := []string{"CODEOWNERS", "SECURITY.md"}
	for _, f := range required {
		if !existsAny(root, f, filepath.Join(".github", f)) {
			report.Findings = append(report.Findings, Finding{Control: "MISSING_POLICY_FILE", Severity: "MEDIUM", Message: "Required governance file missing", Evidence: f, Remediation: "Add the governance file with ownership and vulnerability reporting rules."})
		}
	}
	err := walkText(root, func(path string, lineNo int, line string) {
		low := strings.ToLower(line)
		if strings.Contains(low, "allow_all") || strings.Contains(low, "skip_security") || strings.Contains(low, "no_verify") {
			report.Findings = append(report.Findings, Finding{Control: "POLICY_BYPASS", Severity: "HIGH", File: rel(root, path), Line: lineNo, Message: "Potential policy bypass marker found", Evidence: strings.TrimSpace(line), Remediation: "Remove broad exemptions; document narrowly scoped exceptions with expiry."})
		}
	})
	return report, err
}

// SLOAudit validates service-level objectives from .zed-slo.json when present.
func SLOAudit(root string) (Report, error) {
	report := Report{Name: "SLO/Error Budget Audit", Root: abs(root), GeneratedAt: time.Now()}
	path := filepath.Join(root, ".zed-slo.json")
	buf, err := os.ReadFile(path)
	if err != nil {
		report.Findings = append(report.Findings, Finding{Control: "SLO_FILE_MISSING", Severity: "INFO", Message: "No .zed-slo.json file found", Remediation: "Add SLO definitions for production services."})
		return report, nil
	}
	var doc struct { Services []struct{ Name string `json:"name"`; Availability float64 `json:"availability"`; LatencyP95MS int `json:"latency_p95_ms"`; ErrorBudgetBurn float64 `json:"error_budget_burn"` } `json:"services"` }
	if err := json.Unmarshal(buf, &doc); err != nil { return report, err }
	for _, svc := range doc.Services {
		if svc.Name == "" { report.Findings = append(report.Findings, Finding{Control: "SLO_SERVICE_UNNAMED", Severity: "MEDIUM", File: ".zed-slo.json", Message: "SLO service entry has no name"}) }
		if svc.Availability < 99.0 { report.Findings = append(report.Findings, Finding{Control: "LOW_AVAILABILITY_TARGET", Severity: "MEDIUM", File: ".zed-slo.json", Message: "Availability target is below 99%", Evidence: svc.Name}) }
		if svc.ErrorBudgetBurn > 1.0 { report.Findings = append(report.Findings, Finding{Control: "ERROR_BUDGET_EXHAUSTED", Severity: "HIGH", File: ".zed-slo.json", Message: "Error-budget burn rate is above 1.0", Evidence: svc.Name}) }
		if svc.LatencyP95MS <= 0 { report.Findings = append(report.Findings, Finding{Control: "LATENCY_SLO_MISSING", Severity: "LOW", File: ".zed-slo.json", Message: "Latency p95 objective is missing or invalid", Evidence: svc.Name}) }
	}
	return report, nil
}

// AuditRecord is a tamper-evident JSONL audit event.
type AuditRecord struct {
	Timestamp time.Time `json:"timestamp"`
	Actor     string    `json:"actor"`
	Action    string    `json:"action"`
	Target    string    `json:"target"`
	Details   string    `json:"details,omitempty"`
	PrevHash  string    `json:"prev_hash"`
	Hash      string    `json:"hash"`
}

// AppendAudit writes a hash-chained audit event.
func AppendAudit(root, actor, action, target, details string) (AuditRecord, error) {
	if actor == "" { actor = "zed" }
	if action == "" { return AuditRecord{}, fmt.Errorf("action is required") }
	path := filepath.Join(root, ".zed-audit.jsonl")
	prev := lastAuditHash(path)
	rec := AuditRecord{Timestamp: time.Now().UTC(), Actor: actor, Action: action, Target: target, Details: details, PrevHash: prev}
	rec.Hash = auditHash(rec)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil { return rec, err }
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil { return rec, err }
	defer f.Close()
	buf, _ := json.Marshal(rec)
	_, err = f.Write(append(buf, '\n'))
	return rec, err
}

// AuditVerify validates the audit hash chain.
func AuditVerify(root string) (Report, error) {
	path := filepath.Join(root, ".zed-audit.jsonl")
	report := Report{Name: "Audit Trail Verification", Root: abs(root), GeneratedAt: time.Now()}
	f, err := os.Open(path)
	if os.IsNotExist(err) { report.Findings = append(report.Findings, Finding{Control: "AUDIT_LOG_MISSING", Severity: "INFO", Message: "No .zed-audit.jsonl audit log exists"}); return report, nil }
	if err != nil { return report, err }
	defer f.Close()
	prev := ""
	s := bufio.NewScanner(f)
	lineNo := 0
	for s.Scan() {
		lineNo++
		var rec AuditRecord
		if err := json.Unmarshal(s.Bytes(), &rec); err != nil { report.Findings = append(report.Findings, Finding{Control: "AUDIT_JSON_INVALID", Severity: "HIGH", File: ".zed-audit.jsonl", Line: lineNo, Message: err.Error()}); continue }
		hash := rec.Hash
		rec.Hash = ""
		if rec.PrevHash != prev { report.Findings = append(report.Findings, Finding{Control: "AUDIT_CHAIN_BROKEN", Severity: "CRITICAL", File: ".zed-audit.jsonl", Line: lineNo, Message: "Previous hash does not match chain"}) }
		if auditHash(rec) != hash { report.Findings = append(report.Findings, Finding{Control: "AUDIT_RECORD_TAMPERED", Severity: "CRITICAL", File: ".zed-audit.jsonl", Line: lineNo, Message: "Record hash is invalid"}) }
		prev = hash
	}
	return report, s.Err()
}

// BackupPlan reports files that should be protected by backups.
func BackupPlan(root string) (Report, error) {
	manifest, err := BuildManifest(root)
	if err != nil { return Report{}, err }
	total := int64(0)
	for _, f := range manifest.Files { total += f.Size }
	report := Report{Name: "Backup/Restore Readiness", Root: abs(root), GeneratedAt: time.Now(), Metadata: map[string]any{"files": len(manifest.Files), "bytes": total}}
	if !existsAny(root, ".zed-manifest.json") {
		report.Findings = append(report.Findings, Finding{Control: "RESTORE_MANIFEST_MISSING", Severity: "MEDIUM", Message: "No integrity manifest is committed", Remediation: "Run integrity_manifest to create .zed-manifest.json for restore verification."})
	}
	if total > 100<<20 {
		report.Findings = append(report.Findings, Finding{Control: "BACKUP_SIZE_LARGE", Severity: "LOW", Message: "Project backup is larger than 100 MiB", Evidence: fmt.Sprintf("%d bytes", total), Remediation: "Exclude generated artifacts and vendor caches from backups."})
	}
	return report, nil
}

func walkText(root string, fn func(path string, lineNo int, line string)) error {
	root = abs(root)
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil { return err }
		if d.IsDir() {
			if path != root && skipDir(d.Name()) { return filepath.SkipDir }
			return nil
		}
		info, err := d.Info(); if err != nil || info.Size() > maxScannedFileBytes { return err }
		if !isTextFile(path) { return nil }
		f, err := os.Open(path); if err != nil { return err }
		defer f.Close()
		s := bufio.NewScanner(f)
		s.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		line := 0
		for s.Scan() { line++; fn(path, line, s.Text()) }
		return s.Err()
	})
}

func collectGoMod(root string, deps *[]Dependency) {
	path := filepath.Join(root, "go.mod")
	buf, err := os.ReadFile(path); if err != nil { return }
	inBlock := false
	for _, line := range strings.Split(string(buf), "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "// indirect"))
		if strings.HasPrefix(line, "require (") { inBlock = true; continue }
		if inBlock && line == ")" { inBlock = false; continue }
		if strings.HasPrefix(line, "require ") { line = strings.TrimSpace(strings.TrimPrefix(line, "require ")) }
		if inBlock || strings.Contains(line, " ") {
			fields := strings.Fields(line)
			if len(fields) >= 2 && strings.Contains(fields[0], ".") {
				*deps = append(*deps, Dependency{Ecosystem: "go", Name: fields[0], Version: fields[1], Source: "go.mod"})
			}
		}
	}
}

func collectPackageJSON(root string, deps *[]Dependency) {
	filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() { return err }
		if skipDir(d.Name()) { return nil }
		if d.Name() != "package.json" { return nil }
		var pkg struct { Dependencies map[string]string `json:"dependencies"`; DevDependencies map[string]string `json:"devDependencies"`; PeerDependencies map[string]string `json:"peerDependencies"` }
		buf, err := os.ReadFile(path); if err != nil || json.Unmarshal(buf, &pkg) != nil { return nil }
		source := rel(root, path)
		for name, ver := range pkg.Dependencies { *deps = append(*deps, Dependency{Ecosystem: "npm", Name: name, Version: ver, Source: source}) }
		for name, ver := range pkg.DevDependencies { *deps = append(*deps, Dependency{Ecosystem: "npm-dev", Name: name, Version: ver, Source: source}) }
		for name, ver := range pkg.PeerDependencies { *deps = append(*deps, Dependency{Ecosystem: "npm-peer", Name: name, Version: ver, Source: source}) }
		return nil
	})
}

func collectRequirements(root string, deps *[]Dependency) {
	filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() { return err }
		if d.Name() != "requirements.txt" { return nil }
		buf, err := os.ReadFile(path); if err != nil { return nil }
		for _, line := range strings.Split(string(buf), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") { continue }
			name, ver := line, ""
			for _, sep := range []string{"==", ">=", "<=", "~=", ">", "<"} { if i := strings.Index(line, sep); i >= 0 { name, ver = strings.TrimSpace(line[:i]), strings.TrimSpace(line[i+len(sep):]); break } }
			*deps = append(*deps, Dependency{Ecosystem: "pypi", Name: name, Version: ver, Source: rel(root, path)})
		}
		return nil
	})
}

func collectCargo(root string, deps *[]Dependency) {
	path := filepath.Join(root, "Cargo.toml")
	buf, err := os.ReadFile(path); if err != nil { return }
	inDeps := false
	for _, line := range strings.Split(string(buf), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[") { inDeps = line == "[dependencies]" || line == "[dev-dependencies]"; continue }
		if !inDeps || line == "" || strings.HasPrefix(line, "#") { continue }
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 { *deps = append(*deps, Dependency{Ecosystem: "cargo", Name: strings.TrimSpace(parts[0]), Version: strings.Trim(strings.TrimSpace(parts[1]), `"`), Source: "Cargo.toml"}) }
	}
}

func hashFile(path string) (string, error) { f, err := os.Open(path); if err != nil { return "", err }; defer f.Close(); h := sha256.New(); if _, err := io.Copy(h, f); err != nil { return "", err }; return hex.EncodeToString(h.Sum(nil)), nil }
func candidateTokens(line string) []string { return regexp.MustCompile(`[A-Za-z0-9_\-+/=]{20,}`).FindAllString(line, -1) }
func shannon(s string) float64 { counts := map[rune]float64{}; for _, r := range s { counts[r]++ }; var e float64; l := float64(len(s)); for _, c := range counts { p := c/l; e -= p*math.Log2(p) }; return e }
func looksLikeHash(s string) bool { if len(s) == 32 || len(s) == 40 || len(s) == 64 { _, err := hex.DecodeString(s); return err == nil }; return false }
func redact(s string) string { if len(s) <= 8 { return "[redacted]" }; return s[:4] + strings.Repeat("*", min(12, len(s)-8)) + s[len(s)-4:] }
func isTextFile(path string) bool { ext := strings.ToLower(filepath.Ext(path)); text := map[string]bool{".go":true,".js":true,".ts":true,".tsx":true,".jsx":true,".py":true,".rs":true,".java":true,".cs":true,".rb":true,".php":true,".json":true,".yaml":true,".yml":true,".toml":true,".md":true,".txt":true,".env":true,".sh":true,".ps1":true,".sql":true,".xml":true,".html":true,".css":true}; return text[ext] }
func skipDir(name string) bool { switch name { case ".git","node_modules","vendor","dist","build","target","coverage",".cache",".next",".venv","__pycache__": return true }; return false }
func abs(path string) string { a, err := filepath.Abs(path); if err != nil { return path }; return filepath.ToSlash(a) }
func rel(root, path string) string { r, err := filepath.Rel(abs(root), abs(path)); if err != nil { return filepath.ToSlash(path) }; return filepath.ToSlash(r) }
func existsAny(root string, names ...string) bool { for _, n := range names { if _, err := os.Stat(filepath.Join(root, n)); err == nil { return true } }; return false }
func riskyLicense(s string) bool { up := strings.ToUpper(s); return strings.Contains(up, "GPL") || strings.Contains(up, "AGPL") || strings.Contains(up, "SSPL") }
func luhn(s string) bool { digits := regexp.MustCompile(`\D`).ReplaceAllString(s, ""); sum, alt := 0, false; for i := len(digits)-1; i >= 0; i-- { n, _ := strconv.Atoi(string(digits[i])); if alt { n *= 2; if n > 9 { n -= 9 } }; sum += n; alt = !alt }; return len(digits) >= 13 && sum%10 == 0 }
func min(a, b int) int { if a < b { return a }; return b }
func lastAuditHash(path string) string { buf, err := os.ReadFile(path); if err != nil { return "" }; lines := strings.Split(strings.TrimSpace(string(buf)), "\n"); if len(lines) == 0 || lines[0] == "" { return "" }; var rec AuditRecord; if json.Unmarshal([]byte(lines[len(lines)-1]), &rec) == nil { return rec.Hash }; return "" }
func auditHash(rec AuditRecord) string { rec.Hash = ""; buf, _ := json.Marshal(rec); sum := sha256.Sum256(buf); return hex.EncodeToString(sum[:]) }
