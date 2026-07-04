package enterprise

import "strings"

// ComplianceMapping connects a finding to common enterprise control families.
type ComplianceMapping struct {
	Control string   `json:"control"`
	SOC2    []string `json:"soc2,omitempty"`
	ISO27001 []string `json:"iso27001,omitempty"`
	NIST    []string `json:"nist,omitempty"`
	OWASP   []string `json:"owasp,omitempty"`
}

var defaultComplianceMappings = []ComplianceMapping{
	{Control: "HARDCODED_SECRET", SOC2: []string{"CC6.1", "CC6.6"}, ISO27001: []string{"A.5.17", "A.8.9"}, NIST: []string{"IA-5", "SC-12"}, OWASP: []string{"A02:2021"}},
	{Control: "PRIVATE_KEY", SOC2: []string{"CC6.1", "CC6.6"}, ISO27001: []string{"A.5.17", "A.8.24"}, NIST: []string{"IA-5", "SC-12"}, OWASP: []string{"A02:2021"}},
	{Control: "SQL_INJECTION", SOC2: []string{"CC7.1"}, ISO27001: []string{"A.8.28"}, NIST: []string{"SI-10"}, OWASP: []string{"A03:2021"}},
	{Control: "COMMAND_INJECTION", SOC2: []string{"CC7.1"}, ISO27001: []string{"A.8.28"}, NIST: []string{"SI-10"}, OWASP: []string{"A03:2021"}},
	{Control: "TLS_VERIFY_DISABLED", SOC2: []string{"CC6.7"}, ISO27001: []string{"A.8.20", "A.8.24"}, NIST: []string{"SC-8", "SC-13"}, OWASP: []string{"A02:2021"}},
	{Control: "PRIVILEGED_CONTAINER", SOC2: []string{"CC6.3"}, ISO27001: []string{"A.8.18"}, NIST: []string{"AC-6", "CM-6"}, OWASP: []string{"A05:2021"}},
	{Control: "UNPINNED_DEPENDENCY", SOC2: []string{"CC8.1"}, ISO27001: []string{"A.8.8", "A.8.9"}, NIST: []string{"SA-10", "SI-7"}, OWASP: []string{"A06:2021"}},
	{Control: "MISSING_NODE_LOCKFILE", SOC2: []string{"CC8.1"}, ISO27001: []string{"A.8.8"}, NIST: []string{"SA-10"}, OWASP: []string{"A06:2021"}},
	{Control: "PII", SOC2: []string{"P4.1", "P5.1"}, ISO27001: []string{"A.5.34"}, NIST: []string{"PT-2", "PT-3"}, OWASP: []string{"A01:2021"}},
	{Control: "AUDIT_RECORD_TAMPERED", SOC2: []string{"CC7.2", "CC7.3"}, ISO27001: []string{"A.8.15", "A.8.16"}, NIST: []string{"AU-9", "AU-10"}},
}

// MapCompliance returns a stable compliance mapping for a finding control.
func MapCompliance(control string) ComplianceMapping {
	for _, m := range defaultComplianceMappings {
		if strings.EqualFold(m.Control, control) || strings.Contains(strings.ToUpper(control), strings.ToUpper(m.Control)) {
			return m
		}
	}
	return ComplianceMapping{Control: control, SOC2: []string{"CC7.1"}, ISO27001: []string{"A.8.8"}, NIST: []string{"RA-5"}}
}

// AttachCompliance enriches findings with compliance metadata in Report.Metadata.
func AttachCompliance(r Report) Report {
	mappings := make(map[string]ComplianceMapping)
	for _, f := range r.Findings {
		mappings[f.Control] = MapCompliance(f.Control)
	}
	if r.Metadata == nil { r.Metadata = map[string]any{} }
	r.Metadata["compliance_mappings"] = mappings
	return r
}
