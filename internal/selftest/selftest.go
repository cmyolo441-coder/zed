// Package selftest provides honest test-plan and test-code generation. It never
// emits skipped or pretend tests; when executable test cases are not supplied it
// returns a coverage plan only and asks the caller for concrete inputs/outputs.
package selftest

import (
	"fmt"
	"strings"
)

type TestPlan struct {
	Target       string
	Language     string
	Package      string
	CoverageGoal float64
	ParamType    string
	ReturnType   string
	Cases        []TestCase
}

type TestCase struct {
	Name        string
	Description string
	Input       string
	Expected    string
	Type        string
}

func Generate(funcName, language string, coverageGoal float64) *TestPlan {
	return &TestPlan{
		Target:       funcName,
		Language:     language,
		CoverageGoal: coverageGoal,
		Cases: []TestCase{
			{Name: "basic", Description: "Representative valid input and exact expected output", Type: "basic"},
			{Name: "empty_input", Description: "Empty/nil/minimum input behavior", Type: "edge"},
			{Name: "max_input", Description: "Maximum practical size or upper boundary", Type: "edge"},
			{Name: "invalid_input", Description: "Invalid input returns a documented error/result", Type: "error"},
			{Name: "deterministic", Description: "Same input returns the same output", Type: "property"},
		},
	}
}

func (p *TestPlan) WithExecutableDetails(pkg, paramType, returnType string, cases []TestCase) *TestPlan {
	p.Package = pkg
	p.ParamType = paramType
	p.ReturnType = returnType
	if len(cases) > 0 {
		p.Cases = cases
	}
	return p
}

func (p *TestPlan) RenderGo() string {
	ready := p.Package != "" && p.ParamType != "" && len(p.executableCases()) > 0
	if !ready {
		return p.nonExecutableNotice("Go")
	}
	pkg := p.Package
	var b strings.Builder
	fmt.Fprintf(&b, "package %s\n\n", pkg)
	b.WriteString("import \"testing\"\n\n")
	fmt.Fprintf(&b, "func Test%s_GeneratedContract(t *testing.T) {\n", exportedName(p.Target))
	fmt.Fprintf(&b, "\ttests := []struct {\n\t\tname string\n\t\tinput %s\n\t\twant %s\n\t}{\n", p.ParamType, defaultReturnType(p.ReturnType))
	for _, tc := range p.executableCases() {
		fmt.Fprintf(&b, "\t\t{name: %q, input: %s, want: %s},\n", safeName(tc.Name), tc.Input, tc.Expected)
	}
	b.WriteString("\t}\n\tfor _, tt := range tests {\n\t\tt.Run(tt.name, func(t *testing.T) {\n")
	fmt.Fprintf(&b, "\t\t\tgot := %s(tt.input)\n", p.Target)
	b.WriteString("\t\t\tif got != tt.want {\n\t\t\t\tt.Fatalf(\"got %v, want %v\", got, tt.want)\n\t\t\t}\n")
	b.WriteString("\t\t})\n\t}\n}\n")
	return b.String()
}

func (p *TestPlan) RenderPython() string {
	cases := p.executableCases()
	if len(cases) == 0 {
		return p.nonExecutableNotice("Python")
	}
	var b strings.Builder
	b.WriteString("import pytest\n\n")
	fmt.Fprintf(&b, "@pytest.mark.parametrize(\"value, expected\", [\n")
	for _, tc := range cases {
		fmt.Fprintf(&b, "    (%s, %s),  # %s\n", tc.Input, tc.Expected, safeName(tc.Name))
	}
	b.WriteString("])\n")
	fmt.Fprintf(&b, "def test_%s_generated_contract(value, expected):\n", strings.ToLower(p.Target))
	fmt.Fprintf(&b, "    assert %s(value) == expected\n", p.Target)
	return b.String()
}

func (p *TestPlan) Render() string {
	switch strings.ToLower(p.Language) {
	case "python":
		return p.RenderPython()
	default:
		return p.RenderGo()
	}
}

func (p *TestPlan) Summary() string {
	var b strings.Builder
	fmt.Fprintf(&b, "🧪 Self-Test Plan for %s (%s)\n", p.Target, p.Language)
	fmt.Fprintf(&b, "Coverage goal: %.0f%%\n", p.CoverageGoal)
	fmt.Fprintf(&b, "Cases: %d\n\n", len(p.Cases))
	for _, tc := range p.Cases {
		emoji := "✓"
		if tc.Type == "edge" { emoji = "🔍" } else if tc.Type == "error" { emoji = "⚠️" } else if tc.Type == "property" { emoji = "🧬" }
		fmt.Fprintf(&b, "  %s %s: %s", emoji, tc.Name, tc.Description)
		if tc.Input != "" || tc.Expected != "" { fmt.Fprintf(&b, " [input=%s expected=%s]", tc.Input, tc.Expected) }
		b.WriteString("\n")
	}
	return b.String()
}

func (p *TestPlan) executableCases() []TestCase {
	out := make([]TestCase, 0, len(p.Cases))
	for _, tc := range p.Cases { if tc.Input != "" && tc.Expected != "" { out = append(out, tc) } }
	return out
}

func (p *TestPlan) nonExecutableNotice(language string) string {
	return fmt.Sprintf("No executable %s test file was generated because concrete function signature and input/expected cases were not provided. Supply package/param_type/return_type and cases to generate runnable assertions.\n", language)
}

func exportedName(s string) string { if s == "" { return "Target" }; return strings.ToUpper(s[:1]) + s[1:] }
func safeName(s string) string { if s == "" { return "case" }; return strings.ReplaceAll(s, "\"", "'") }
func defaultReturnType(s string) string { if s == "" { return "any" }; return s }
