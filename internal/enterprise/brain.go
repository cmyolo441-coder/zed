package enterprise

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

type ArchitectureNode struct { ID string `json:"id"`; Kind string `json:"kind"`; File string `json:"file,omitempty"` }
type ArchitectureEdge struct { From string `json:"from"`; To string `json:"to"`; Kind string `json:"kind"` }
type ArchitectureModel struct { Root string `json:"root"`; Nodes []ArchitectureNode `json:"nodes"`; Edges []ArchitectureEdge `json:"edges"`; Domains []string `json:"domains"`; ExternalIntegrations []string `json:"external_integrations"` }

// ArchitectureBrain builds a local architecture knowledge graph from real files.
func ArchitectureBrain(root string) (Report, ArchitectureModel, error) {
	model := ArchitectureModel{Root: abs(root)}
	nodes := map[string]ArchitectureNode{}
	addNode := func(id, kind, file string) { if id != "" { nodes[id] = ArchitectureNode{ID:id, Kind:kind, File:file} } }
	importsByPkg := map[string]map[string]bool{}
	domainSet := map[string]bool{}
	external := map[string]bool{}
	goImportRe := regexp.MustCompile(`^\s*"([^"]+)"|^\s*import\s+"([^"]+)"`)
	pkgRe := regexp.MustCompile(`^\s*package\s+(\w+)`)
	err := walkText(root, func(path string, lineNo int, line string) {
		r := rel(root, path)
		parts := strings.Split(r, "/")
		if len(parts) > 1 && (parts[0] == "internal" || parts[0] == "pkg" || parts[0] == "cmd") { domainSet[parts[1]] = true; addNode(parts[0]+"/"+parts[1], "domain", parts[0]+"/"+parts[1]) }
		if strings.HasSuffix(path, ".go") {
			if m := pkgRe.FindStringSubmatch(line); m != nil { addNode(m[1], "go_package", r); if importsByPkg[m[1]] == nil { importsByPkg[m[1]] = map[string]bool{} } }
			if m := goImportRe.FindStringSubmatch(line); m != nil { imp := firstNonEmptyLocal(m[1], m[2]); if imp != "" { external[imp] = true; addNode(imp, "import", ""); for pkg := range importsByPkg { importsByPkg[pkg][imp] = true } } }
		}
		low := strings.ToLower(line)
		if strings.Contains(low, "http://") || strings.Contains(low, "https://") || strings.Contains(low, "grpc") || strings.Contains(low, "kafka") || strings.Contains(low, "redis") || strings.Contains(low, "postgres") || strings.Contains(low, "mysql") { external[strings.TrimSpace(line)] = true }
	})
	for id, n := range nodes { _ = id; model.Nodes = append(model.Nodes, n) }
	for pkg, imps := range importsByPkg { for imp := range imps { model.Edges = append(model.Edges, ArchitectureEdge{From:pkg, To:imp, Kind:"imports"}) } }
	for d := range domainSet { model.Domains = append(model.Domains, d) }
	for e := range external { if strings.Contains(e, ".") || strings.Contains(e, "://") || strings.Contains(e, "redis") || strings.Contains(e, "kafka") { model.ExternalIntegrations = append(model.ExternalIntegrations, e) } }
	sort.Slice(model.Nodes, func(i,j int) bool { return model.Nodes[i].ID < model.Nodes[j].ID })
	sort.Strings(model.Domains); sort.Strings(model.ExternalIntegrations)
	report := Report{Name:"Enterprise Architecture Brain", Root:abs(root), GeneratedAt:time.Now(), Metadata:map[string]any{"model":model}}
	if len(model.Domains) == 0 { report.Findings = append(report.Findings, Finding{Control:"ARCHITECTURE_DOMAINS_UNCLEAR", Severity:"MEDIUM", Message:"No clear internal/pkg/cmd domain boundaries detected", Remediation:"Organize code around explicit bounded contexts/modules."}) }
	if len(model.Edges) > 150 { report.Findings = append(report.Findings, Finding{Control:"ARCHITECTURE_HIGH_COUPLING", Severity:"MEDIUM", Message:"Import graph has high edge count", Evidence:fmt.Sprintf("edges=%d", len(model.Edges)), Remediation:"Extract interfaces and reduce cross-domain imports."}) }
	return report, model, err
}

type ChangeImpactResult struct { ChangedFiles []string `json:"changed_files"`; ImpactedTests []string `json:"impacted_tests"`; ImpactedEndpoints []Endpoint `json:"impacted_endpoints"`; Risk string `json:"risk"` }

func ChangeImpact(root string) (Report, ChangeImpactResult, error) {
	files, _ := gitOutputLines(root, "diff", "--name-only", "HEAD")
	if len(files) == 0 { files, _ = gitOutputLines(root, "status", "--porcelain") }
	var changed []string
	for _, f := range files { fields := strings.Fields(f); if len(fields) > 0 { changed = append(changed, fields[len(fields)-1]) } }
	_, endpoints, _ := APISurfaceInventory(root)
	changedSet := map[string]bool{}; for _, f := range changed { changedSet[filepath.ToSlash(f)] = true }
	res := ChangeImpactResult{ChangedFiles:changed, Risk:"LOW"}
	for _, f := range changed { res.ImpactedTests = append(res.ImpactedTests, impactedTestsFor(f)...); if isCriticalPathName(f) { res.Risk = "HIGH" } }
	for _, ep := range endpoints { if changedSet[ep.File] { res.ImpactedEndpoints = append(res.ImpactedEndpoints, ep); res.Risk = maxRisk(res.Risk, "MEDIUM") } }
	if len(changed) > 20 { res.Risk = "HIGH" }
	report := Report{Name:"Enterprise Change Impact", Root:abs(root), GeneratedAt:time.Now(), Metadata:map[string]any{"impact":res}}
	if res.Risk == "HIGH" { report.Findings = append(report.Findings, Finding{Control:"CHANGE_IMPACT_HIGH", Severity:"HIGH", Message:"Change has high blast radius", Evidence:fmt.Sprintf("files=%d endpoints=%d", len(res.ChangedFiles), len(res.ImpactedEndpoints)), Remediation:"Require focused review, targeted tests, and rollback plan."}) }
	return report, res, nil
}

func WritePatchBundle(root, output string) (string, error) {
	if output == "" { output = filepath.Join(root, "zed-patch-bundle.md") }
	if !filepath.IsAbs(output) { output = filepath.Join(root, output) }
	stat, _ := gitOutput(root, "diff", "--stat", "HEAD")
	diff, _ := gitOutput(root, "diff", "HEAD")
	impact, _, _ := ChangeImpact(root)
	content := "# ZED Enterprise Patch Bundle\n\nGenerated: "+time.Now().Format(time.RFC3339)+"\n\n## Impact\n\n"+impact.Summary()+"\n\n## Diff Stat\n\n```\n"+stat+"\n```\n\n## Unified Diff\n\n```diff\n"+diff+"\n```\n\n## Verification\n\n- go test ./...\n- go vet ./...\n- ./zed --enterprise-policy-gate\n\n## Rollback\n\n- git checkout -- <changed files>\n- restore database/config from approved rollback plan when applicable\n"
	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil { return "", err }
	return output, os.WriteFile(output, []byte(content), 0644)
}

func BuildDoctor(buildOutput string) Report {
	report := Report{Name:"Enterprise Build Doctor", Root:"", GeneratedAt:time.Now()}
	rules := []struct{ control, sev string; re *regexp.Regexp; msg, fix string }{
		{"BUILD_UNDEFINED_SYMBOL", "HIGH", regexp.MustCompile(`undefined:|cannot find symbol|is not defined`), "Undefined symbol detected", "Check renamed/deleted identifiers and imports."},
		{"BUILD_UNUSED_IMPORT", "MEDIUM", regexp.MustCompile(`imported and not used|unused import`), "Unused import detected", "Remove unused imports or use the referenced package."},
		{"BUILD_TYPE_MISMATCH", "HIGH", regexp.MustCompile(`cannot use .* as .* value|type mismatch|incompatible types`), "Type mismatch detected", "Update function signatures or conversions."},
		{"BUILD_PACKAGE_CYCLE", "CRITICAL", regexp.MustCompile(`import cycle not allowed|cyclic dependency`), "Package cycle detected", "Extract interfaces into a lower-level package."},
		{"BUILD_MODULE_MISSING", "HIGH", regexp.MustCompile(`no required module provides package|Cannot find module|module not found`), "Missing dependency/module", "Add dependency with package manager and commit lockfile."},
		{"TEST_ASSERTION_FAILED", "MEDIUM", regexp.MustCompile(`FAIL:|--- FAIL|AssertionError|expected .* got`), "Test failure detected", "Inspect assertion diff and fix behavior or test expectation."},
	}
	for i, line := range strings.Split(buildOutput, "\n") { for _, r := range rules { if r.re.MatchString(line) { report.Findings = append(report.Findings, Finding{Control:r.control, Severity:r.sev, Line:i+1, Message:r.msg, Evidence:strings.TrimSpace(line), Remediation:r.fix}) } } }
	if len(report.Findings) == 0 { report.Findings = append(report.Findings, Finding{Control:"BUILD_DOCTOR_NO_MATCH", Severity:"INFO", Message:"No known build failure pattern matched"}) }
	return report
}

func GenerateADR(root, title, contextText, decision, alternatives, consequences string) (string, error) {
	if title == "" { title = "Architecture Decision" }
	dir := filepath.Join(root, "docs", "adr"); _ = os.MkdirAll(dir, 0755)
	n := 1; entries, _ := os.ReadDir(dir); for _, e := range entries { if strings.HasSuffix(e.Name(), ".md") { n++ } }
	file := filepath.Join(dir, fmt.Sprintf("%04d-%s.md", n, slug(title)))
	content := fmt.Sprintf("# ADR %04d: %s\n\nDate: %s\n\n## Status\n\nProposed\n\n## Context\n\n%s\n\n## Decision\n\n%s\n\n## Alternatives\n\n%s\n\n## Consequences\n\n%s\n\n## Verification\n\n- go test ./...\n- enterprise_policy_gate\n\n## Rollback\n\nDocument the reverse migration or git revert strategy before approval.\n", n, title, time.Now().Format("2006-01-02"), contextText, decision, alternatives, consequences)
	return file, os.WriteFile(file, []byte(content), 0644)
}

func CriticalPath(root string) (Report, error) {
	report := Report{Name:"Enterprise Critical Path Analyzer", Root:abs(root), GeneratedAt:time.Now()}
	keywords := map[string]string{"auth":"authentication/authorization", "admin":"administration", "payment":"payments", "billing":"billing", "delete":"destructive data action", "export":"data export", "crypto":"cryptography", "secret":"secret handling", "migration":"database migration"}
	err := walkText(root, func(path string, lineNo int, line string) { low := strings.ToLower(path+" "+line); for k, desc := range keywords { if strings.Contains(low, k) { report.Findings = append(report.Findings, Finding{Control:"CRITICAL_PATH_"+strings.ToUpper(k), Severity:"MEDIUM", File:rel(root,path), Line:lineNo, Message:"Business-critical path detected: "+desc, Evidence:strings.TrimSpace(line), Remediation:"Require owner review, tests, audit logging, and rollback plan."}); break } } })
	return report, err
}

func APIGuardian(root, baseline string, writeBaseline bool) (Report, error) {
	if baseline == "" { baseline = filepath.Join(root, ".zed-api-baseline.json") }
	_, endpoints, err := APISurfaceInventory(root); if err != nil { return Report{}, err }
	if writeBaseline { buf,_ := json.MarshalIndent(endpoints,"","  "); return Report{Name:"API Guardian Baseline", Root:abs(root), GeneratedAt:time.Now()}, os.WriteFile(baseline, buf, 0644) }
	report := Report{Name:"Enterprise API Guardian", Root:abs(root), GeneratedAt:time.Now()}
	buf, err := os.ReadFile(baseline); if err != nil { report.Findings = append(report.Findings, Finding{Control:"API_BASELINE_MISSING", Severity:"MEDIUM", Message:"API baseline missing", Evidence:baseline, Remediation:"Run api guardian with write_baseline=true."}); return report, nil }
	var old []Endpoint; _ = json.Unmarshal(buf, &old)
	oldSet := map[string]Endpoint{}; newSet := map[string]Endpoint{}; for _, e := range old { oldSet[e.Method+" "+e.Path]=e }; for _, e := range endpoints { newSet[e.Method+" "+e.Path]=e }
	for k, e := range oldSet { if _, ok := newSet[k]; !ok { report.Findings = append(report.Findings, Finding{Control:"API_BREAKING_REMOVED_ENDPOINT", Severity:"HIGH", File:e.File, Line:e.Line, Message:"Endpoint from baseline is missing", Evidence:k, Remediation:"Version the API or update baseline with approval."}) } }
	for k, e := range newSet { if _, ok := oldSet[k]; !ok { report.Findings = append(report.Findings, Finding{Control:"API_NEW_ENDPOINT", Severity:"LOW", File:e.File, Line:e.Line, Message:"New endpoint not in baseline", Evidence:k, Remediation:"Review auth, tests, and update baseline."}) } }
	return report, nil
}

func GoConcurrencyAudit(root string) (Report, error) {
	report := Report{Name:"Enterprise Go Concurrency Audit", Root:abs(root), GeneratedAt:time.Now()}
	err := walkText(root, func(path string, lineNo int, line string) { if !strings.HasSuffix(path,".go") { return }; low := strings.ToLower(line); switch { case strings.Contains(line,"go ") && !strings.Contains(low,"context") && !strings.Contains(low,"ctx"): report.Findings=append(report.Findings, Finding{Control:"GOROUTINE_WITHOUT_CONTEXT_SIGNAL", Severity:"MEDIUM", File:rel(root,path), Line:lineNo, Message:"Goroutine launch has no visible context/cancellation signal", Evidence:strings.TrimSpace(line), Remediation:"Pass context and ensure shutdown path."}); case strings.Contains(line,"time.Tick("): report.Findings=append(report.Findings, Finding{Control:"TIME_TICK_LEAK_RISK", Severity:"HIGH", File:rel(root,path), Line:lineNo, Message:"time.Tick can leak ticker resources", Evidence:strings.TrimSpace(line), Remediation:"Use time.NewTicker and Stop()."}); case strings.Contains(line,"sync.Mutex") && strings.Contains(line,"="): report.Findings=append(report.Findings, Finding{Control:"MUTEX_COPY_RISK", Severity:"MEDIUM", File:rel(root,path), Line:lineNo, Message:"Mutex assignment/copy risk", Evidence:strings.TrimSpace(line), Remediation:"Do not copy structs containing mutexes."}) } })
	return report, err
}

func PerfHotspotPredictor(root string) (Report, error) {
	report := Report{Name:"Enterprise Performance Hotspot Predictor", Root:abs(root), GeneratedAt:time.Now()}
	loopDepth := map[string]int{}
	err := walkText(root, func(path string, lineNo int, line string) { trimmed := strings.TrimSpace(line); if strings.HasPrefix(trimmed,"for ") || strings.HasPrefix(trimmed,"for{") || strings.HasPrefix(trimmed,"while ") { loopDepth[path]++ }; if loopDepth[path] > 1 { report.Findings=append(report.Findings, Finding{Control:"NESTED_LOOP_HOTSPOT", Severity:"MEDIUM", File:rel(root,path), Line:lineNo, Message:"Nested loop may be a performance hotspot", Evidence:trimmed, Remediation:"Check complexity and benchmark with realistic data."}) }; if strings.Contains(line,"regexp.MustCompile") && loopDepth[path] > 0 { report.Findings=append(report.Findings, Finding{Control:"REGEX_COMPILE_IN_LOOP", Severity:"HIGH", File:rel(root,path), Line:lineNo, Message:"Regex compilation inside loop", Evidence:trimmed, Remediation:"Compile regex once outside hot path."}) }; if strings.Contains(line," += ") && loopDepth[path] > 0 { report.Findings=append(report.Findings, Finding{Control:"STRING_CONCAT_IN_LOOP", Severity:"MEDIUM", File:rel(root,path), Line:lineNo, Message:"String concatenation in loop", Evidence:trimmed, Remediation:"Use strings.Builder or bytes.Buffer."}) }; if strings.Contains(line,"ReadFile(") || strings.Contains(line,"io.ReadAll") { report.Findings=append(report.Findings, Finding{Control:"UNBOUNDED_READ_MEMORY_RISK", Severity:"LOW", File:rel(root,path), Line:lineNo, Message:"Whole-file/read-all memory pattern", Evidence:trimmed, Remediation:"Use streaming or size limits for large/untrusted inputs."}) }; if strings.Contains(trimmed,"}") && loopDepth[path] > 0 { loopDepth[path]-- } })
	return report, err
}

type WorkJournalEntry struct { Time time.Time `json:"time"`; Task string `json:"task,omitempty"`; Action string `json:"action,omitempty"`; Files string `json:"files,omitempty"`; Evidence string `json:"evidence,omitempty"`; Risks string `json:"risks,omitempty"`; Next string `json:"next,omitempty"` }
func AppendWorkJournal(root string, e WorkJournalEntry) (string, error) { if e.Time.IsZero() { e.Time = time.Now().UTC() }; path := filepath.Join(root,".zed-work-journal.jsonl"); f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644); if err != nil { return "", err }; defer f.Close(); buf,_ := json.Marshal(e); _, err = f.Write(append(buf,'\n')); return path, err }
func ReadWorkJournal(root string) ([]WorkJournalEntry, error) { path := filepath.Join(root,".zed-work-journal.jsonl"); buf, err := os.ReadFile(path); if os.IsNotExist(err) { return nil, nil }; if err != nil { return nil, err }; var out []WorkJournalEntry; for _, line := range bytes.Split(bytes.TrimSpace(buf), []byte("\n")) { var e WorkJournalEntry; if len(line)>0 && json.Unmarshal(line,&e)==nil { out=append(out,e) } }; return out, nil }

func gitOutput(root string, args ...string) (string, error) { cmd := exec.Command("git", args...); cmd.Dir=root; var b bytes.Buffer; cmd.Stdout=&b; cmd.Stderr=&b; err:=cmd.Run(); return b.String(), err }
func gitOutputLines(root string, args ...string) ([]string,error) { out, err := gitOutput(root,args...); var lines []string; for _, l := range strings.Split(out,"\n") { if strings.TrimSpace(l)!="" { lines=append(lines,l) } }; return lines, err }
func impactedTestsFor(path string) []string { dir := filepath.Dir(path); base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)); return []string{filepath.ToSlash(filepath.Join(dir, base+"_test.go")), filepath.ToSlash(filepath.Join(dir, base+".test.ts")), filepath.ToSlash(filepath.Join(dir, "test_"+base+".py"))} }
func isCriticalPathName(s string) bool { low:=strings.ToLower(s); for _, k := range []string{"auth","admin","payment","billing","crypto","secret","migration","delete","export"} { if strings.Contains(low,k) { return true } }; return false }
func maxRisk(a,b string) string { rank:=map[string]int{"LOW":1,"MEDIUM":2,"HIGH":3,"CRITICAL":4}; if rank[b]>rank[a] { return b }; return a }
func firstNonEmptyLocal(vals ...string) string { for _, v := range vals { if v!="" { return v } }; return "" }
func slug(s string) string { s=strings.ToLower(s); re:=regexp.MustCompile(`[^a-z0-9]+`); s=strings.Trim(re.ReplaceAllString(s,"-"),"-"); if s=="" { return "decision" }; return s }
