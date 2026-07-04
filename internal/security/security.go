// Package security enforces ZED's safety policy: it decides which tool actions
// are allowed, which require explicit user approval, and which are blocked
// outright. This is the guardrail layer between the LLM's requests and the
// user's real filesystem and shell.
package security

import (
	"path/filepath"
	"regexp"
	"strings"
)

// Decision is the outcome of evaluating an action against the policy.
type Decision int

const (
	// Allow means the action may run without prompting.
	Allow Decision = iota
	// Confirm means the user must approve before the action runs.
	Confirm
	// Deny means the action is blocked and must not run.
	Deny
)

func (d Decision) String() string {
	switch d {
	case Allow:
		return "allow"
	case Confirm:
		return "confirm"
	case Deny:
		return "deny"
	default:
		return "unknown"
	}
}

// Policy holds configurable safety rules.
type Policy struct {
	WorkDir       string
	AutoApprove   bool     // if true, write/edit/shell run without confirmation
	AllowOutside  bool     // if true, allow paths outside the working directory
	BlockedCmds   []*regexp.Regexp
	DangerousCmds []*regexp.Regexp
	ProtectedPath []string // path suffixes that always require confirmation
}

// Verdict is a policy decision plus a human-readable reason.
type Verdict struct {
	Decision Decision
	Reason   string
}

// DefaultPolicy returns a hardened policy suitable for interactive use.
func DefaultPolicy(workDir string, autoApprove bool) *Policy {
	return &Policy{
		WorkDir:     workDir,
		AutoApprove: autoApprove,
		BlockedCmds: compileAll([]string{
			`(?i)\brm\s+-rf\s+/(\s|$)`,      // rm -rf /
			`(?i)\brm\s+-rf\s+~`,            // rm -rf ~
			`(?i):\(\)\s*\{.*\}\s*;`,        // fork bomb
			`(?i)\bmkfs\b`,                  // formatting filesystems
			`(?i)\bdd\s+if=.*of=/dev/`,      // writing to raw devices
			`(?i)>\s*/dev/sd[a-z]`,          // clobbering disks
			`(?i)\bformat\s+[a-z]:`,         // windows format C:
			`(?i)Remove-Item.*-Recurse.*[Cc]:\\`, // powershell wipe of C:
		}),
		DangerousCmds: compileAll([]string{
			`(?i)\bsudo\b`,
			`(?i)\bcurl\b.*\|\s*(sh|bash|pwsh|powershell)`, // curl | sh
			`(?i)\bwget\b.*\|\s*(sh|bash)`,
			`(?i)\bgit\s+push\b.*--force`,
			`(?i)\bshutdown\b`,
			`(?i)\breboot\b`,
			`(?i)\bkill(all)?\b`,
			`(?i)\bchmod\s+777\b`,
			`(?i)\bnpm\s+publish\b`,
			`(?i)Invoke-WebRequest.*\|\s*(iex|Invoke-Expression)`,
		}),
		ProtectedPath: []string{
			".env", ".git", "id_rsa", ".ssh", "credentials",
			".npmrc", ".aws", ".kube", "secrets",
		},
	}
}

// EvaluatePath decides whether a filesystem path may be accessed/written.
// readonly=true relaxes the check to allow reading protected paths but still
// blocks reads outside the working directory unless AllowOutside is set.
func (p *Policy) EvaluatePath(path string, readonly bool) Verdict {
	abs := p.abs(path)
	root := p.abs(p.WorkDir)

	inside := strings.HasPrefix(abs, root)
	if !inside && !p.AllowOutside {
		return Verdict{Deny, "path is outside the working directory: " + path}
	}

	// Protected files always require confirmation (or denial for writes).
	if p.isProtected(abs) {
		if readonly {
			return Verdict{Confirm, "reading a sensitive path: " + path}
		}
		return Verdict{Deny, "refusing to modify a sensitive path: " + path}
	}

	if readonly {
		return Verdict{Allow, ""}
	}
	// Writes: allow silently only when auto-approve is on.
	if p.AutoApprove {
		return Verdict{Allow, ""}
	}
	return Verdict{Confirm, "write to " + path}
}

// EvaluateCommand decides whether a shell command may run.
func (p *Policy) EvaluateCommand(cmd string) Verdict {
	trimmed := strings.TrimSpace(cmd)
	if trimmed == "" {
		return Verdict{Deny, "empty command"}
	}
	for _, re := range p.BlockedCmds {
		if re.MatchString(trimmed) {
			return Verdict{Deny, "command matches a blocked destructive pattern"}
		}
	}
	for _, re := range p.DangerousCmds {
		if re.MatchString(trimmed) {
			return Verdict{Confirm, "command looks dangerous and needs approval"}
		}
	}
	if p.AutoApprove {
		return Verdict{Allow, ""}
	}
	return Verdict{Confirm, "run shell command"}
}

// isProtected reports whether a path touches a protected/secret file or dir.
func (p *Policy) isProtected(abs string) bool {
	lower := strings.ToLower(filepath.ToSlash(abs))
	for _, marker := range p.ProtectedPath {
		m := strings.ToLower(marker)
		if strings.Contains(lower, "/"+m) || strings.HasSuffix(lower, "/"+m) ||
			strings.HasSuffix(lower, m) || strings.Contains(lower, "/"+m+"/") {
			return true
		}
	}
	return false
}

func (p *Policy) abs(path string) string {
	if !filepath.IsAbs(path) {
		path = filepath.Join(p.WorkDir, path)
	}
	a, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(a)
}

func compileAll(patterns []string) []*regexp.Regexp {
	out := make([]*regexp.Regexp, 0, len(patterns))
	for _, pat := range patterns {
		if re, err := regexp.Compile(pat); err == nil {
			out = append(out, re)
		}
	}
	return out
}

// AddBlocked adds a custom blocked command pattern at runtime.
func (p *Policy) AddBlocked(pattern string) error {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}
	p.BlockedCmds = append(p.BlockedCmds, re)
	return nil
}
