// Package quality provides static analysis and code quality scoring.
// It detects security vulnerabilities (OWASP Top 10), code smells, duplication,
// and complexity issues — then suggests or applies fixes.
package quality

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Severity levels.
type Severity int

const (
	SeverityError   Severity = iota // critical: security, crash risk
	SeverityWarning                 // should fix: code smell, anti-pattern
	SeverityInfo                    // nice to have: style, readability
)

func (s Severity) String() string {
	switch s {
	case SeverityError:
		return "ERROR"
	case SeverityWarning:
		return "WARN"
	case SeverityInfo:
		return "INFO"
	default:
		return "?"
	}
}

// Finding is a single code quality issue.
type Finding struct {
	File     string
	Line     int
	Severity Severity
	Rule     string
	Message  string
	Fix      string // suggested fix (empty if no auto-fix)
}

// Report holds the results of analyzing a file or project.
type Report struct {
	Files    int
	Findings []Finding
	Score    float64 // 0-10 quality score
}

// Analyze runs all quality checks on a file or directory.
func Analyze(path string) (*Report, error) {
	var findings []Finding
	var fileCount int

	err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(p))
		if !isSourceFile(ext) {
			return nil
		}
		// Skip common ignore dirs.
		for _, skip := range []string{".git", "node_modules", "vendor", "dist", "build"} {
			if strings.Contains(p, skip) {
				return nil
			}
		}
		fileFindings, err := analyzeFile(p)
		if err != nil {
			return nil
		}
		findings = append(findings, fileFindings...)
		fileCount++
		return nil
	})
	if err != nil {
		return nil, err
	}

	report := &Report{
		Files:    fileCount,
		Findings: findings,
	}
	report.Score = calculateScore(findings, fileCount)
	return report, nil
}

// analyzeFile runs all rules on a single file.
func analyzeFile(path string) ([]Finding, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	content := string(src)
	lines := strings.Split(content, "\n")

	var findings []Finding
	findings = append(findings, checkSecurity(path, lines)...)
	findings = append(findings, checkCodeSmells(path, lines)...)
	findings = append(findings, checkDuplication(path, content)...)
	findings = append(findings, checkComplexity(path, lines)...)
	return findings, nil
}

// --- Security checks (OWASP Top 10 patterns) ---

var securityPatterns = []struct {
	rule    string
	pattern *regexp.Regexp
	message string
	fix     string
}{
	{
		"SQL_INJECTION",
		regexp.MustCompile(`(?i)(?:SELECT|INSERT|UPDATE|DELETE|DROP).*\+\s*(?:req|params|query|input|user|data)`),
		"Potential SQL injection: string concatenation in SQL query",
		"Use parameterized queries instead of string concatenation",
	},
	{
		"XSS_RISK",
		regexp.MustCompile(`(?i)(?:innerHTML|document\.write|dangerouslySetInnerHTML)\s*[=(]`),
		"Potential XSS: unescaped HTML output",
		"Use textContent or escape HTML before rendering",
	},
	{
		"HARDCODED_SECRET",
		regexp.MustCompile(`(?i)(?:password|secret|api_key|apikey|token|auth)\s*[:=]\s*["'][^"']{8,}["']`),
		"Hardcoded secret/credential detected",
		"Move secrets to environment variables or a secrets manager",
	},
	{
		"PATH_TRAVERSAL",
		regexp.MustCompile(`(?:\.\.\/|\.\.\\).*(?:open|read|write|file|path)`),
		"Potential path traversal: user input in file path",
		"Validate and sanitize file paths, restrict to allowed directories",
	},
	{
		"COMMAND_INJECTION",
		regexp.MustCompile(`(?:exec|system|popen|shell_exec|Runtime\.exec)\s*\([^)]*(?:req|params|input|user|args)`),
		"Potential command injection: user input in shell command",
		"Use argument arrays instead of string concatenation for commands",
	},
	{
		"WEAK_CRYPTO",
		regexp.MustCompile(`(?i)(?:md5|sha1)\s*\(`),
		"Weak cryptographic hash (MD5/SHA1)",
		"Use SHA-256 or stronger for security-sensitive hashing",
	},
	{
		"INSECURE_RANDOM",
		regexp.MustCompile(`(?i)math\.random|Math\.random`),
		"Insecure random number generator for security context",
		"Use crypto/rand or crypto.getRandomValues for security-sensitive randomness",
	},
	{
		"EVAL_USAGE",
		regexp.MustCompile(`\beval\s*\(`),
		"Use of eval() is dangerous",
		"Avoid eval(); use JSON.parse or proper parsing instead",
	},
}

func checkSecurity(file string, lines []string) []Finding {
	var findings []Finding
	for i, line := range lines {
		for _, p := range securityPatterns {
			if p.pattern.FindString(line) != "" {
				findings = append(findings, Finding{
					File:     file,
					Line:     i + 1,
					Severity: SeverityError,
					Rule:     p.rule,
					Message:  p.message,
					Fix:      p.fix,
				})
			}
		}
	}
	return findings
}

// --- Code smell checks ---

var smellPatterns = []struct {
	rule    string
	pattern *regexp.Regexp
	message string
	fix     string
}{
	{
		"TODO_LEFT",
		regexp.MustCompile(`(?i)(?:TODO|FIXME|HACK|XXX|BUG)`),
		"Unresolved TODO/FIXME comment",
		"Complete or remove the TODO before shipping",
	},
	{
		"LONG_LINE",
		regexp.MustCompile(`^.{121,}$`),
		"Line exceeds 120 characters",
		"Break long lines for readability",
	},
	{
		"EMPTY_CATCH",
		regexp.MustCompile(`catch\s*\([^)]*\)\s*\{\s*\}`),
		"Empty catch block — errors silently swallowed",
		"Log or handle the error instead of ignoring it",
	},
	{
		"PRINT_DEBUG",
		regexp.MustCompile(`(?:fmt\.Print|console\.log|print\()`),
		"Debug print statement left in code",
		"Remove debug prints or use a proper logger",
	},
	{
		"MAGIC_NUMBER",
		regexp.MustCompile(`(?:if|while|for|return)\s+[^=]*\b\d{3,}\b`),
		"Magic number — unexplained numeric literal",
		"Extract to a named constant for clarity",
	},
}

func checkCodeSmells(file string, lines []string) []Finding {
	var findings []Finding
	for i, line := range lines {
		for _, p := range smellPatterns {
			if p.pattern.FindString(line) != "" {
				findings = append(findings, Finding{
					File:     file,
					Line:     i + 1,
					Severity: SeverityWarning,
					Rule:     p.rule,
					Message:  p.message,
					Fix:      p.fix,
				})
			}
		}
	}
	return findings
}

// --- Duplication detection ---

func checkDuplication(file, content string) []Finding {
	var findings []Finding
	lines := strings.Split(content, "\n")
	seen := make(map[string][]int) // line content → line numbers
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) < 20 || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#") {
			continue
		}
		seen[trimmed] = append(seen[trimmed], i+1)
	}
	for line, occurrences := range seen {
		if len(occurrences) > 2 {
			findings = append(findings, Finding{
				File:     file,
				Line:     occurrences[0],
				Severity: SeverityInfo,
				Rule:     "DUPLICATION",
				Message:  fmt.Sprintf("Code duplicated %d times: %q", len(occurrences), truncate(line, 60)),
				Fix:      "Extract to a shared function or constant",
			})
		}
	}
	return findings
}

// --- Complexity check ---

func checkComplexity(file string, lines []string) []Finding {
	var findings []Finding
	depth := 0
	maxDepth := 0
	for i, line := range lines {
		depth += strings.Count(line, "{") - strings.Count(line, "}")
		depth += strings.Count(line, "(") - strings.Count(line, ")")
		if depth > maxDepth {
			maxDepth = depth
		}
		if depth > 5 {
			findings = append(findings, Finding{
				File:     file,
				Line:     i + 1,
				Severity: SeverityWarning,
				Rule:     "HIGH_NESTING",
				Message:  fmt.Sprintf("Deep nesting (%d levels) — hard to read", depth),
				Fix:      "Extract nested logic into separate functions or use early returns",
			})
		}
	}
	return findings
}

// calculateScore computes a 0-10 quality score based on findings.
func calculateScore(findings []Finding, fileCount int) float64 {
	if fileCount == 0 {
		return 10.0
	}
	score := 10.0
	for _, f := range findings {
		switch f.Severity {
		case SeverityError:
			score -= 1.0
		case SeverityWarning:
			score -= 0.3
		case SeverityInfo:
			score -= 0.1
		}
	}
	if score < 0 {
		score = 0
	}
	return score
}

// Summary returns a human-readable report.
func (r *Report) Summary() string {
	var b strings.Builder
	fmt.Fprintf(&b, "📊 Code Quality Report\n")
	fmt.Fprintf(&b, "   Files analyzed: %d\n", r.Files)
	fmt.Fprintf(&b, "   Quality score: %.1f/10\n\n", r.Score)

	errors, warns, infos := 0, 0, 0
	for _, f := range r.Findings {
		switch f.Severity {
		case SeverityError:
			errors++
		case SeverityWarning:
			warns++
		case SeverityInfo:
			infos++
		}
	}
	fmt.Fprintf(&b, "   🔴 Errors: %d | 🟡 Warnings: %d | 🔵 Info: %d\n\n", errors, warns, infos)

	for _, f := range r.Findings {
		fmt.Fprintf(&b, "  %s %s:%d [%s] %s\n", f.Severity, f.File, f.Line, f.Rule, f.Message)
		if f.Fix != "" {
			fmt.Fprintf(&b, "    → Fix: %s\n", f.Fix)
		}
	}
	return b.String()
}

func isSourceFile(ext string) bool {
	switch ext {
	case ".go", ".py", ".js", ".ts", ".jsx", ".tsx", ".java", ".c", ".cpp", ".h", ".rb", ".php", ".cs", ".rs":
		return true
	default:
		return false
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
