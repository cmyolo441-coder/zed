package enterprise

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type RiskHeatmapEntry struct {
	Path     string   `json:"path"`
	Score    int      `json:"score"`
	Risk     string   `json:"risk"`
	Findings []string `json:"findings"`
}

type Milestone struct {
	ID          int      `json:"id"`
	Name        string   `json:"name"`
	Goal        string   `json:"goal"`
	Risk        string   `json:"risk"`
	Commands    []string `json:"commands"`
	Rollback    string   `json:"rollback"`
	NeedsReview bool     `json:"needs_review"`
}

func RiskHeatmap(root string) (Report, []RiskHeatmapEntry, error) {
	reports := []Report{}
	for _, audit := range []func(string) (Report, error){CriticalPath, SecretScan, TaintAnalysis, GoConcurrencyAudit, PerfHotspotPredictor, CodeOwnershipAudit, MigrationSafetyAudit} {
		r, err := audit(root)
		if err != nil { return Report{}, nil, err }
		reports = append(reports, r)
	}
	type bucket struct { score int; findings []string }
	buckets := map[string]*bucket{}
	add := func(path string, sev string, control string) {
		if path == "" { path = "." }
		parts := strings.Split(filepath.ToSlash(path), "/")
		key := path
		if len(parts) >= 2 { key = parts[0] + "/" + parts[1] } else if len(parts) == 1 { key = parts[0] }
		if buckets[key] == nil { buckets[key] = &bucket{} }
		buckets[key].score += severityPoints(sev)
		buckets[key].findings = append(buckets[key].findings, control)
	}
	for _, r := range reports { for _, f := range r.Findings { add(f.File, f.Severity, f.Control) } }
	var entries []RiskHeatmapEntry
	for path, b := range buckets {
		sort.Strings(b.findings)
		entries = append(entries, RiskHeatmapEntry{Path:path, Score:b.score, Risk:riskFromScore(b.score), Findings:dedupeStrings(b.findings)})
	}
	sort.Slice(entries, func(i,j int) bool { if entries[i].Score != entries[j].Score { return entries[i].Score > entries[j].Score }; return entries[i].Path < entries[j].Path })
	report := Report{Name:"Enterprise Risk Heatmap", Root:abs(root), GeneratedAt:time.Now(), Metadata:map[string]any{"heatmap": entries}}
	for _, e := range entries { if e.Risk == "HIGH" || e.Risk == "CRITICAL" { report.Findings = append(report.Findings, Finding{Control:"RISK_HEATMAP_"+e.Risk, Severity:e.Risk, File:e.Path, Message:"Directory risk is elevated", Evidence:fmt.Sprintf("score=%d findings=%d", e.Score, len(e.Findings)), Remediation:"Prioritize review, tests, ownership, and remediation for this path."}) } }
	return report, entries, nil
}

func BuildMilestones(goal string) []Milestone {
	if strings.TrimSpace(goal) == "" { goal = "enterprise engineering task" }
	return []Milestone{
		{ID:1, Name:"Analysis", Goal:goal, Risk:"LOW", Commands:[]string{"enterprise_architecture_brain", "enterprise_change_impact", "enterprise_risk_heatmap"}, Rollback:"No changes made in analysis stage", NeedsReview:false},
		{ID:2, Name:"Implementation", Goal:goal, Risk:"MEDIUM", Commands:[]string{"edit/write targeted files", "enterprise_work_journal append"}, Rollback:"git checkout -- <changed files>", NeedsReview:true},
		{ID:3, Name:"Verification", Goal:goal, Risk:"MEDIUM", Commands:[]string{"go test ./...", "go vet ./...", "enterprise_build_doctor on failures"}, Rollback:"Revert implementation milestone if verification fails", NeedsReview:false},
		{ID:4, Name:"Enterprise Evidence", Goal:goal, Risk:"LOW", Commands:[]string{"enterprise_policy_gate", "enterprise_evidence_pack", "enterprise_sarif_export"}, Rollback:"Regenerate evidence after fixes", NeedsReview:false},
		{ID:5, Name:"Release Review", Goal:goal, Risk:"HIGH", Commands:[]string{"enterprise_patch_bundle", "enterprise_adr_generator when architecture changed"}, Rollback:"Use patch bundle rollback section", NeedsReview:true},
	}
}

func MilestonePlan(root, goal string) (Report, []Milestone, error) {
	milestones := BuildMilestones(goal)
	report := Report{Name:"Enterprise Milestone Executor Plan", Root:abs(root), GeneratedAt:time.Now(), Metadata:map[string]any{"goal":goal, "milestones":milestones}}
	for _, m := range milestones { if m.NeedsReview { report.Findings = append(report.Findings, Finding{Control:"MILESTONE_REVIEW_REQUIRED", Severity:m.Risk, Message:"Milestone requires human review", Evidence:m.Name, Remediation:"Approve plan before applying changes."}) } }
	return report, milestones, nil
}

func severityPoints(sev string) int { switch strings.ToUpper(sev) { case "CRITICAL": return 12; case "HIGH": return 8; case "MEDIUM": return 4; case "LOW": return 2; default: return 1 } }
func riskFromScore(score int) string { switch { case score >= 35: return "CRITICAL"; case score >= 18: return "HIGH"; case score >= 8: return "MEDIUM"; case score >= 2: return "LOW"; default: return "INFO" } }
func dedupeStrings(in []string) []string { seen := map[string]bool{}; var out []string; for _, s := range in { if !seen[s] { seen[s]=true; out=append(out,s) } }; return out }
