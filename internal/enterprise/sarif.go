package enterprise

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type sarifLog struct {
	Version string     `json:"version"`
	Schema  string     `json:"$schema"`
	Runs    []sarifRun `json:"runs"`
}
type sarifRun struct { Tool sarifTool `json:"tool"`; Results []sarifResult `json:"results"` }
type sarifTool struct { Driver sarifDriver `json:"driver"` }
type sarifDriver struct { Name string `json:"name"`; InformationURI string `json:"informationUri,omitempty"`; Rules []sarifRule `json:"rules"` }
type sarifRule struct { ID string `json:"id"`; Name string `json:"name"`; ShortDescription sarifText `json:"shortDescription"`; Help sarifText `json:"help"`; Properties map[string]any `json:"properties,omitempty"` }
type sarifText struct { Text string `json:"text"` }
type sarifResult struct { RuleID string `json:"ruleId"`; Level string `json:"level"`; Message sarifText `json:"message"`; Locations []sarifLocation `json:"locations,omitempty"` }
type sarifLocation struct { PhysicalLocation sarifPhysicalLocation `json:"physicalLocation"` }
type sarifPhysicalLocation struct { ArtifactLocation sarifArtifactLocation `json:"artifactLocation"`; Region sarifRegion `json:"region,omitempty"` }
type sarifArtifactLocation struct { URI string `json:"uri"` }
type sarifRegion struct { StartLine int `json:"startLine,omitempty"` }

// SARIF builds a SARIF 2.1.0 document from the enterprise policy gate findings.
func SARIF(root string) ([]byte, error) {
	report, _, err := PolicyGate(root)
	if err != nil { return nil, err }
	ruleMap := map[string]Finding{}
	var results []sarifResult
	for _, f := range report.Findings {
		ruleMap[f.Control] = f
		res := sarifResult{RuleID: f.Control, Level: sarifLevel(f.Severity), Message: sarifText{Text: f.Message}}
		if f.File != "" {
			res.Locations = []sarifLocation{{PhysicalLocation: sarifPhysicalLocation{ArtifactLocation: sarifArtifactLocation{URI: filepath.ToSlash(f.File)}, Region: sarifRegion{StartLine: f.Line}}}}
		}
		results = append(results, res)
	}
	var rules []sarifRule
	for id, f := range ruleMap {
		rules = append(rules, sarifRule{ID: id, Name: id, ShortDescription: sarifText{Text: f.Message}, Help: sarifText{Text: f.Remediation}, Properties: map[string]any{"severity": strings.ToUpper(f.Severity)}})
	}
	log := sarifLog{Version: "2.1.0", Schema: "https://json.schemastore.org/sarif-2.1.0.json", Runs: []sarifRun{{Tool: sarifTool{Driver: sarifDriver{Name: "ZED Enterprise Gate", InformationURI: "https://github.com/cmyolo441-coder/zed", Rules: rules}}, Results: results}}}
	return json.MarshalIndent(log, "", "  ")
}

func WriteSARIF(root, output string) (string, error) {
	if output == "" { output = filepath.Join(root, "zed-enterprise.sarif") }
	if !filepath.IsAbs(output) { output = filepath.Join(root, output) }
	buf, err := SARIF(root)
	if err != nil { return "", err }
	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil { return "", err }
	if err := os.WriteFile(output, buf, 0644); err != nil { return "", err }
	return output, nil
}

func sarifLevel(sev string) string { switch strings.ToUpper(sev) { case "CRITICAL", "HIGH": return "error"; case "MEDIUM": return "warning"; default: return "note" } }
