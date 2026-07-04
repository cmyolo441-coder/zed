package enterprise

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Endpoint is a discovered API/HTTP surface.
type Endpoint struct {
	Method string `json:"method"`
	Path   string `json:"path"`
	File   string `json:"file"`
	Line   int    `json:"line"`
}

// APISurfaceInventory discovers real HTTP routes in common Go/Node/Python code.
func APISurfaceInventory(root string) (Report, []Endpoint, error) {
	report := Report{Name: "API Surface Inventory", Root: abs(root), GeneratedAt: time.Now()}
	var endpoints []Endpoint
	patterns := []struct{ re *regexp.Regexp; methodIdx, pathIdx int }{
		{regexp.MustCompile(`HandleFunc\(\s*"([^"]+)"`), 0, 1},
		{regexp.MustCompile(`\.Methods\(\s*"([A-Z]+)"\s*\)`), 1, 0},
		{regexp.MustCompile(`app\.(get|post|put|patch|delete)\(\s*['"]([^'"]+)['"]`), 1, 2},
		{regexp.MustCompile(`@(app|router)\.(get|post|put|patch|delete)\(\s*['"]([^'"]+)['"]`), 2, 3},
	}
	lastGoPath := ""
	err := walkText(root, func(path string, lineNo int, line string) {
		if m := patterns[0].re.FindStringSubmatch(line); m != nil {
			lastGoPath = m[1]
			endpoints = append(endpoints, Endpoint{Method: "ANY", Path: m[1], File: rel(root, path), Line: lineNo})
		}
		if m := patterns[1].re.FindStringSubmatch(line); m != nil && lastGoPath != "" && len(endpoints) > 0 {
			endpoints[len(endpoints)-1].Method = m[1]
		}
		for _, pat := range patterns[2:] {
			if m := pat.re.FindStringSubmatch(line); m != nil {
				endpoints = append(endpoints, Endpoint{Method: strings.ToUpper(m[pat.methodIdx]), Path: m[pat.pathIdx], File: rel(root, path), Line: lineNo})
			}
		}
	})
	sort.Slice(endpoints, func(i, j int) bool { if endpoints[i].File != endpoints[j].File { return endpoints[i].File < endpoints[j].File }; return endpoints[i].Line < endpoints[j].Line })
	report.Metadata = map[string]any{"endpoints": endpoints, "count": len(endpoints)}
	for _, ep := range endpoints {
		low := strings.ToLower(ep.Path)
		if strings.Contains(low, "admin") || strings.Contains(low, "debug") {
			report.Findings = append(report.Findings, Finding{Control: "SENSITIVE_ENDPOINT", Severity: "HIGH", File: ep.File, Line: ep.Line, Message: "Sensitive endpoint exposed in API surface", Evidence: ep.Method + " " + ep.Path, Remediation: "Require strong authentication, authorization, audit logging, and network restrictions."})
		}
		if ep.Method == "ANY" {
			report.Findings = append(report.Findings, Finding{Control: "HTTP_METHOD_UNSCOPED", Severity: "MEDIUM", File: ep.File, Line: ep.Line, Message: "Route does not constrain HTTP method", Evidence: ep.Path, Remediation: "Bind handlers to explicit methods."})
		}
	}
	return report, endpoints, err
}

// ObservabilityAudit checks for real logging, metrics, tracing, and health-readiness signals.
func ObservabilityAudit(root string) (Report, error) {
	report := Report{Name: "Observability Audit", Root: abs(root), GeneratedAt: time.Now()}
	var hasStructuredLog, hasMetrics, hasTracing, hasHealth, hasPanicRecovery bool
	_ = walkText(root, func(path string, lineNo int, line string) {
		low := strings.ToLower(line)
		if strings.Contains(low, "zap.") || strings.Contains(low, "zerolog") || strings.Contains(low, "logrus") || strings.Contains(low, "slog.") { hasStructuredLog = true }
		if strings.Contains(low, "prometheus") || strings.Contains(low, "opentelemetry") || strings.Contains(low, "statsd") || strings.Contains(low, "metrics") { hasMetrics = true }
		if strings.Contains(low, "trace") || strings.Contains(low, "otel") || strings.Contains(low, "jaeger") { hasTracing = true }
		if strings.Contains(low, "/health") || strings.Contains(low, "/ready") || strings.Contains(low, "healthcheck") { hasHealth = true }
		if strings.Contains(low, "recover()") || strings.Contains(low, "recovery") { hasPanicRecovery = true }
	})
	checks := []struct{ ok bool; ctrl, msg, fix string }{
		{hasStructuredLog, "STRUCTURED_LOGGING_MISSING", "No structured logging signal found", "Use slog/zap/zerolog/logrus with request IDs and JSON output."},
		{hasMetrics, "METRICS_MISSING", "No metrics instrumentation signal found", "Expose RED/USE metrics via OpenTelemetry or Prometheus."},
		{hasTracing, "TRACING_MISSING", "No distributed tracing signal found", "Add OpenTelemetry tracing and propagate trace context."},
		{hasHealth, "HEALTH_ENDPOINT_MISSING", "No health/readiness endpoint signal found", "Expose liveness and readiness checks."},
		{hasPanicRecovery, "PANIC_RECOVERY_MISSING", "No panic/recovery middleware signal found", "Add recovery middleware around HTTP/background workers."},
	}
	for _, c := range checks { if !c.ok { report.Findings = append(report.Findings, Finding{Control: c.ctrl, Severity: "MEDIUM", Message: c.msg, Remediation: c.fix}) } }
	return report, nil
}

// ContainerAudit scans Docker/Kubernetes manifests for concrete hardening risks.
func ContainerAudit(root string) (Report, error) {
	report := Report{Name: "Container/Kubernetes Hardening Audit", Root: abs(root), GeneratedAt: time.Now()}
	err := walkText(root, func(path string, lineNo int, line string) {
		name := strings.ToLower(filepath.Base(path))
		if !(strings.Contains(name, "dockerfile") || strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml")) { return }
		low := strings.ToLower(strings.TrimSpace(line))
		switch {
		case strings.HasPrefix(low, "from ") && strings.Contains(low, ":latest"):
			report.Findings = append(report.Findings, Finding{Control: "IMAGE_TAG_LATEST", Severity: "HIGH", File: rel(root, path), Line: lineNo, Message: "Container base image uses latest tag", Remediation: "Pin immutable versions or digests."})
		case strings.HasPrefix(low, "user root") || strings.Contains(low, "runasuser: 0"):
			report.Findings = append(report.Findings, Finding{Control: "RUNS_AS_ROOT", Severity: "HIGH", File: rel(root, path), Line: lineNo, Message: "Container workload runs as root", Remediation: "Run as non-root UID and drop capabilities."})
		case strings.Contains(low, "privileged: true"):
			report.Findings = append(report.Findings, Finding{Control: "PRIVILEGED_CONTAINER", Severity: "CRITICAL", File: rel(root, path), Line: lineNo, Message: "Privileged container detected", Remediation: "Disable privileged mode and grant narrow capabilities only."})
		case strings.Contains(low, "allowprivilegeescalation: true"):
			report.Findings = append(report.Findings, Finding{Control: "PRIVILEGE_ESCALATION_ALLOWED", Severity: "HIGH", File: rel(root, path), Line: lineNo, Message: "Privilege escalation allowed", Remediation: "Set allowPrivilegeEscalation: false."})
		case strings.Contains(low, "readonlyrootfilesystem: false"):
			report.Findings = append(report.Findings, Finding{Control: "WRITABLE_ROOT_FILESYSTEM", Severity: "MEDIUM", File: rel(root, path), Line: lineNo, Message: "Root filesystem is writable", Remediation: "Use readOnlyRootFilesystem: true and explicit writable volumes."})
		}
	})
	return report, err
}

// CodeOwnershipAudit checks whether real source files are covered by CODEOWNERS.
func CodeOwnershipAudit(root string) (Report, error) {
	report := Report{Name: "Code Ownership Audit", Root: abs(root), GeneratedAt: time.Now()}
	ownersPath := findFirst(root, "CODEOWNERS", filepath.Join(".github", "CODEOWNERS"))
	if ownersPath == "" {
		report.Findings = append(report.Findings, Finding{Control: "CODEOWNERS_MISSING", Severity: "HIGH", Message: "No CODEOWNERS file found", Remediation: "Add CODEOWNERS with owners for critical paths."})
		return report, nil
	}
	buf, err := os.ReadFile(ownersPath); if err != nil { return report, err }
	patterns := parseCodeownersPatterns(string(buf))
	unowned := 0
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() { return err }
		if skipDir(d.Name()) || !isTextFile(path) { return nil }
		r := rel(root, path)
		if !matchesAnyOwnerPattern(r, patterns) {
			unowned++
			if unowned <= 50 { report.Findings = append(report.Findings, Finding{Control: "SOURCE_FILE_UNOWNED", Severity: "LOW", File: r, Message: "Source/config file has no matching CODEOWNERS rule"}) }
		}
		return nil
	})
	report.Metadata = map[string]any{"codeowners": rel(root, ownersPath), "patterns": len(patterns), "unowned_files": unowned}
	return report, nil
}

// ThreatModel builds a lightweight STRIDE model from discovered API endpoints and sensitive markers.
func ThreatModel(root string) (Report, error) {
	report, endpoints, err := APISurfaceInventory(root)
	if err != nil { return report, err }
	report.Name = "STRIDE Threat Model"
	if len(endpoints) == 0 { report.Findings = append(report.Findings, Finding{Control: "NO_API_SURFACE", Severity: "INFO", Message: "No API routes detected for threat modeling"}); return report, nil }
	for _, ep := range endpoints {
		controls := []Finding{
			{Control: "STRIDE_SPOOFING", Severity: "MEDIUM", File: ep.File, Line: ep.Line, Message: "Endpoint requires authentication design review", Evidence: ep.Method + " " + ep.Path, Remediation: "Document authN, token validation, session lifetime, and MFA needs."},
			{Control: "STRIDE_TAMPERING", Severity: "MEDIUM", File: ep.File, Line: ep.Line, Message: "Endpoint requires input validation and integrity review", Evidence: ep.Method + " " + ep.Path, Remediation: "Validate schemas, enforce authorization, and use parameterized persistence."},
			{Control: "STRIDE_REPUDIATION", Severity: "LOW", File: ep.File, Line: ep.Line, Message: "Endpoint requires audit logging decision", Evidence: ep.Method + " " + ep.Path, Remediation: "Log actor, action, target, result, request ID, and trace ID."},
		}
		report.Findings = append(report.Findings, controls...)
	}
	return report, nil
}

// ReleaseGate combines real enterprise controls into a pass/fail release report.
func ReleaseGate(root string) (Report, error) {
	report := Report{Name: "Enterprise Release Gate", Root: abs(root), GeneratedAt: time.Now()}
	audits := []func(string) (Report, error){SecretScan, SupplyChainRisk, ConfigAudit, PolicyAudit, ObservabilityAudit, ContainerAudit, CodeOwnershipAudit, TaintAnalysis, MigrationSafetyAudit}
	critical, high, medium := 0, 0, 0
	for _, audit := range audits {
		r, err := audit(root); if err != nil { return report, err }
		for _, f := range r.Findings {
			switch strings.ToUpper(f.Severity) { case "CRITICAL": critical++; case "HIGH": high++; case "MEDIUM": medium++ }
			report.Findings = append(report.Findings, f)
		}
	}
	status := "PASS"
	if critical > 0 || high > 0 { status = "BLOCK" } else if medium > 5 { status = "WARN" }
	report.Metadata = map[string]any{"status": status, "critical": critical, "high": high, "medium": medium}
	if status != "PASS" { report.Findings = append([]Finding{{Control: "RELEASE_GATE_" + status, Severity: "CRITICAL", Message: "Release gate did not pass", Evidence: fmt.Sprintf("critical=%d high=%d medium=%d", critical, high, medium), Remediation: "Fix blocking findings or record approved risk exceptions."}}, report.Findings...) }
	return report, nil
}

// EvidencePack writes a real JSON evidence bundle for audits.
func EvidencePack(root, output string) (string, error) {
	if output == "" { output = filepath.Join(root, ".zed-evidence-pack.json") }
	if !filepath.IsAbs(output) { output = filepath.Join(root, output) }
	secret, _ := SecretScan(root)
	sbom, _, _ := SBOM(root)
	license, _ := LicenseAudit(root)
	supply, _ := SupplyChainRisk(root)
	pii, _ := PIIScan(root)
	config, _ := ConfigAudit(root)
	policy, _ := PolicyAudit(root)
	obs, _ := ObservabilityAudit(root)
	container, _ := ContainerAudit(root)
	owners, _ := CodeOwnershipAudit(root)
	taint, _ := TaintAnalysis(root)
	migrations, _ := MigrationSafetyAudit(root)
	release, _ := ReleaseGate(root)
	pack := map[string]any{"generated_at": time.Now(), "root": abs(root), "reports": []Report{secret, sbom, license, supply, pii, config, policy, obs, container, owners, taint, migrations, release}}
	buf, err := json.MarshalIndent(pack, "", "  "); if err != nil { return "", err }
	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil { return "", err }
	if err := os.WriteFile(output, buf, 0644); err != nil { return "", err }
	return output, nil
}

func findFirst(root string, names ...string) string { for _, n := range names { p := filepath.Join(root, n); if _, err := os.Stat(p); err == nil { return p } }; return "" }
func parseCodeownersPatterns(s string) []string { var out []string; for _, line := range strings.Split(s, "\n") { line = strings.TrimSpace(line); if line == "" || strings.HasPrefix(line, "#") { continue }; fields := strings.Fields(line); if len(fields) > 1 { out = append(out, fields[0]) } }; return out }
func matchesAnyOwnerPattern(path string, patterns []string) bool { for _, p := range patterns { p = strings.TrimPrefix(p, "/"); if p == "*" || p == "**" { return true }; if strings.HasSuffix(p, "/") && strings.HasPrefix(path, p) { return true }; if ok, _ := filepath.Match(p, path); ok { return true }; if strings.Contains(p, "*") { if ok, _ := filepath.Match(strings.ReplaceAll(p, "**/", "*"), path); ok { return true } }; if path == p || strings.HasPrefix(path, strings.TrimSuffix(p, "*") ) { return true } }; return false }
