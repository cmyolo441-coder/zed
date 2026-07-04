// Package ci provides a continuous integration loop that runs lint, build,
// test, and security checks after every file change. It detects the project
// language(s) and runs the appropriate toolchain for each.
package ci

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Language represents a programming language and its toolchain.
type Language struct {
	Name       string
	Extensions []string
	BuildCmd   string
	TestCmd    string
	LintCmd    string
	FmtCmd     string
}

// DetectedProject holds info about the project's language(s) and commands.
type DetectedProject struct {
	Languages []Language
	Root      string
	HasGo     bool
	HasNode   bool
	HasPython bool
	HasRust   bool
	HasMake   bool
}

// Detect identifies the project's language(s) by looking for marker files.
func Detect(workDir string) *DetectedProject {
	p := &DetectedProject{Root: workDir}

	if fileExists(filepath.Join(workDir, "go.mod")) {
		p.HasGo = true
		p.Languages = append(p.Languages, Language{
			Name:       "Go",
			Extensions: []string{".go"},
			BuildCmd:   "go build ./...",
			TestCmd:    "go test ./... -v -count=1",
			LintCmd:    "go vet ./...",
			FmtCmd:     "gofmt -w .",
		})
	}
	if fileExists(filepath.Join(workDir, "package.json")) {
		p.HasNode = true
		p.Languages = append(p.Languages, Language{
			Name:       "Node.js",
			Extensions: []string{".js", ".jsx", ".ts", ".tsx"},
			BuildCmd:   "npm run build 2>&1",
			TestCmd:    "npm test 2>&1",
			LintCmd:    "npx eslint . 2>&1 || true",
			FmtCmd:     "npx prettier --write . 2>&1 || true",
		})
	}
	if fileExists(filepath.Join(workDir, "requirements.txt")) || fileExists(filepath.Join(workDir, "pyproject.toml")) || fileExists(filepath.Join(workDir, "setup.py")) {
		p.HasPython = true
		p.Languages = append(p.Languages, Language{
			Name:       "Python",
			Extensions: []string{".py"},
			BuildCmd:   "python -m py_compile . 2>&1",
			TestCmd:    "python -m pytest -v 2>&1",
			LintCmd:    "python -m flake8 . 2>&1 || true",
			FmtCmd:     "python -m black . 2>&1 || true",
		})
	}
	if fileExists(filepath.Join(workDir, "Cargo.toml")) {
		p.HasRust = true
		p.Languages = append(p.Languages, Language{
			Name:       "Rust",
			Extensions: []string{".rs"},
			BuildCmd:   "cargo build 2>&1",
			TestCmd:    "cargo test 2>&1",
			LintCmd:    "cargo clippy 2>&1",
			FmtCmd:     "cargo fmt 2>&1",
		})
	}
	if fileExists(filepath.Join(workDir, "Makefile")) {
		p.HasMake = true
	}
	return p
}

// PipelineStage represents one step in the CI pipeline.
type PipelineStage struct {
	Name     string // "lint", "build", "test", "security"
	Command  string
	Required bool
}

// Pipeline returns the full CI pipeline for the project.
func (p *DetectedProject) Pipeline() []PipelineStage {
	var stages []PipelineStage
	for _, lang := range p.Languages {
		if lang.LintCmd != "" {
			stages = append(stages, PipelineStage{
				Name:     fmt.Sprintf("lint (%s)", lang.Name),
				Command:  lang.LintCmd,
				Required: false,
			})
		}
		if lang.BuildCmd != "" {
			stages = append(stages, PipelineStage{
				Name:     fmt.Sprintf("build (%s)", lang.Name),
				Command:  lang.BuildCmd,
				Required: true,
			})
		}
		if lang.TestCmd != "" {
			stages = append(stages, PipelineStage{
				Name:     fmt.Sprintf("test (%s)", lang.Name),
				Command:  lang.TestCmd,
				Required: true,
			})
		}
	}
	// Security scan is always included.
	stages = append(stages, PipelineStage{
		Name:     "security scan",
		Command:  "", // handled by quality analyzer
		Required: false,
	})
	return stages
}

// FormatCmd returns the format command for the project.
func (p *DetectedProject) FormatCmd() string {
	var cmds []string
	for _, lang := range p.Languages {
		if lang.FmtCmd != "" {
			cmds = append(cmds, lang.FmtCmd)
		}
	}
	return strings.Join(cmds, " && ")
}

// Summary returns a description of the detected project.
func (p *DetectedProject) Summary() string {
	var b strings.Builder
	b.WriteString("🔍 Project Detection:\n")
	for _, lang := range p.Languages {
		fmt.Fprintf(&b, "   %s: build=%q test=%q lint=%q\n", lang.Name, lang.BuildCmd, lang.TestCmd, lang.LintCmd)
	}
	stages := p.Pipeline()
	fmt.Fprintf(&b, "\n📋 CI Pipeline (%d stages):\n", len(stages))
	for i, s := range stages {
		req := ""
		if s.Required {
			req = " (required)"
		}
		fmt.Fprintf(&b, "   %d. %s%s\n", i+1, s.Name, req)
	}
	return b.String()
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
