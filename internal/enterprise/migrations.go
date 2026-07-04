package enterprise

import (
	"regexp"
	"strings"
	"time"
)

// MigrationSafetyAudit scans SQL migrations for production-dangerous operations.
func MigrationSafetyAudit(root string) (Report, error) {
	report := Report{Name: "Database Migration Safety Audit", Root: abs(root), GeneratedAt: time.Now()}
	patterns := []struct{ rule, severity string; re *regexp.Regexp; fix string }{
		{"DROP_TABLE", "CRITICAL", regexp.MustCompile(`(?i)\bDROP\s+TABLE\b`), "Use a phased migration with backups and explicit approval."},
		{"DROP_COLUMN", "HIGH", regexp.MustCompile(`(?i)\bDROP\s+COLUMN\b`), "Use expand/contract deployment and remove reads before dropping columns."},
		{"ALTER_NOT_NULL_WITHOUT_DEFAULT", "HIGH", regexp.MustCompile(`(?i)ALTER\s+TABLE.*SET\s+NOT\s+NULL`), "Backfill data first, add default/constraint in a separate migration."},
		{"CREATE_INDEX_NON_CONCURRENT", "MEDIUM", regexp.MustCompile(`(?i)\bCREATE\s+INDEX\b`), "For PostgreSQL production tables, use CREATE INDEX CONCURRENTLY."},
		{"DELETE_WITHOUT_WHERE", "CRITICAL", regexp.MustCompile(`(?i)^\s*DELETE\s+FROM\s+\w+\s*;?\s*$`), "Add a WHERE clause and dry-run row counts."},
		{"UPDATE_WITHOUT_WHERE", "CRITICAL", regexp.MustCompile(`(?i)^\s*UPDATE\s+\w+\s+SET\s+.*;?\s*$`), "Add a WHERE clause and bounded batches."},
	}
	err := walkText(root, func(path string, lineNo int, line string) {
		low := strings.ToLower(path)
		if !(strings.HasSuffix(low, ".sql") || strings.Contains(low, "migration")) { return }
		for _, p := range patterns {
			// CREATE INDEX is only unsafe when it is NOT concurrent; RE2 lacks
			// negative lookahead, so exclude the CONCURRENTLY case here.
			if p.rule == "CREATE_INDEX_NON_CONCURRENT" && strings.Contains(strings.ToUpper(line), "CONCURRENTLY") { continue }
			if p.re.MatchString(line) { report.Findings = append(report.Findings, Finding{Control: p.rule, Severity: p.severity, File: rel(root, path), Line: lineNo, Message: "Potentially unsafe database migration operation", Evidence: strings.TrimSpace(line), Remediation: p.fix}) }
		}
	})
	return AttachCompliance(report), err
}
