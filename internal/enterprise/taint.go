package enterprise

import (
	"regexp"
	"strings"
	"time"
)

type TaintRule struct { Name, Severity string; Source *regexp.Regexp; Sink *regexp.Regexp; Remediation string }

var defaultTaintRules = []TaintRule{
	{Name: "TAINT_SQL_INJECTION", Severity: "HIGH", Source: regexp.MustCompile(`(?i)(req\.|request\.|params|query|body|userInput|os\.Args)`), Sink: regexp.MustCompile(`(?i)(db\.Query|db\.Exec|SELECT |INSERT |UPDATE |DELETE |DROP )`), Remediation: "Use parameterized queries and validate input schema."},
	{Name: "TAINT_COMMAND_INJECTION", Severity: "CRITICAL", Source: regexp.MustCompile(`(?i)(req\.|request\.|params|query|body|userInput|os\.Args)`), Sink: regexp.MustCompile(`(?i)(exec\.Command|system\(|shell_exec|Runtime\.exec|child_process)`), Remediation: "Use fixed command paths and argument arrays; never concatenate user input."},
	{Name: "TAINT_PATH_TRAVERSAL", Severity: "HIGH", Source: regexp.MustCompile(`(?i)(req\.|request\.|params|query|body|userInput|os\.Args)`), Sink: regexp.MustCompile(`(?i)(os\.Open|os\.ReadFile|os\.WriteFile|http\.ServeFile|send_file)`), Remediation: "Canonicalize paths and enforce an allowlisted root directory."},
	{Name: "TAINT_XSS", Severity: "HIGH", Source: regexp.MustCompile(`(?i)(req\.|request\.|params|query|body|userInput)`), Sink: regexp.MustCompile(`(?i)(innerHTML|dangerouslySetInnerHTML|document\.write|template\.HTML)`), Remediation: "Escape output and use safe templating APIs."},
}

// TaintAnalysis is a deterministic local source-to-sink proximity analysis. It
// is not a compiler data-flow engine, but it uses real code evidence and errs on
// reviewable findings instead of pretending to prove exploitability.
func TaintAnalysis(root string) (Report, error) {
	report := Report{Name: "Taint Analysis", Root: abs(root), GeneratedAt: time.Now()}
	err := walkText(root, func(path string, lineNo int, line string) {
		for _, rule := range defaultTaintRules {
			if !rule.Source.MatchString(line) { continue }
			window := collectLineWindow(root, path, lineNo, 8)
			for offset, wline := range window {
				if rule.Sink.MatchString(wline) {
					report.Findings = append(report.Findings, Finding{Control: rule.Name, Severity: rule.Severity, File: rel(root, path), Line: lineNo+offset, Message: "User-controlled input appears near a sensitive sink", Evidence: strings.TrimSpace(wline), Remediation: rule.Remediation})
				}
			}
		}
	})
	return AttachCompliance(report), err
}

func collectLineWindow(root, path string, startLine, maxLines int) []string {
	var lines []string
	_ = walkText(root, func(p string, lineNo int, line string) {
		if p == path && lineNo >= startLine && lineNo < startLine+maxLines { lines = append(lines, line) }
	})
	return lines
}
