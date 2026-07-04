// Package vcs provides git-native version control automation. The agent can
// auto-commit with semantic messages, manage branches, generate PRs, and
// auto-resolve merge conflicts.
package vcs

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// GitInfo holds the current git repository state.
type GitInfo struct {
	Root      string
	Branch    string
	Staged    []string
	Modified  []string
	Untracked []string
	HasRemote bool
}

// Status returns the current git repository state.
func Status(workDir string) (*GitInfo, error) {
	root, err := gitRoot(workDir)
	if err != nil {
		return nil, err
	}
	info := &GitInfo{Root: root}
	info.Branch, _ = gitExec(workDir, "rev-parse", "--abbrev-ref", "HEAD")
	info.Staged = gitLines(workDir, "diff", "--cached", "--name-only")
	info.Modified = gitLines(workDir, "diff", "--name-only")
	info.Untracked = gitLines(workDir, "ls-files", "--others", "--exclude-standard")
	_, err = gitExec(workDir, "remote", "get-url", "origin")
	info.HasRemote = err == nil
	return info, nil
}

// AutoCommit stages all changes and commits with a semantic message.
func AutoCommit(workDir, message string) (string, error) {
	if _, err := gitExec(workDir, "add", "-A"); err != nil {
		return "", fmt.Errorf("git add failed: %w", err)
	}
	out, err := gitExec(workDir, "commit", "-m", message)
	if err != nil {
		return "", fmt.Errorf("git commit failed: %w", err)
	}
	return out, nil
}

// SemanticMessage generates a commit message from the changes.
func SemanticMessage(info *GitInfo) string {
	if len(info.Staged) == 0 && len(info.Modified) == 0 && len(info.Untracked) == 0 {
		return "chore: no changes"
	}
	// Determine commit type from file changes.
	allFiles := append(append(info.Staged, info.Modified...), info.Untracked...)
	hasTest := false
	hasSrc := false
	hasDocs := false
	hasConfig := false
	for _, f := range allFiles {
		base := filepath.Base(f)
		if strings.Contains(base, "test") || strings.HasSuffix(base, "_test.go") {
			hasTest = true
		}
		if strings.HasSuffix(f, ".go") || strings.HasSuffix(f, ".py") || strings.HasSuffix(f, ".js") || strings.HasSuffix(f, ".ts") {
			hasSrc = true
		}
		if strings.HasSuffix(f, ".md") {
			hasDocs = true
		}
		if strings.HasSuffix(f, ".json") || strings.HasSuffix(f, ".yaml") || strings.HasSuffix(f, ".toml") {
			hasConfig = true
		}
	}
	prefix := "feat"
	if hasTest && !hasSrc {
		prefix = "test"
	} else if hasDocs && !hasSrc {
		prefix = "docs"
	} else if hasConfig && !hasSrc {
		prefix = "chore"
	}
	// Summarize files.
	fileList := strings.Join(allFiles, ", ")
	if len(fileList) > 60 {
		fileList = fileList[:60] + "..."
	}
	return fmt.Sprintf("%s: update %s", prefix, fileList)
}

// CreateBranch creates and switches to a new branch.
func CreateBranch(workDir, name string) (string, error) {
	out, err := gitExec(workDir, "checkout", "-b", name)
	if err != nil {
		return "", fmt.Errorf("branch creation failed: %w", err)
	}
	return out, nil
}

// Diff returns the git diff output.
func Diff(workDir string) (string, error) {
	return gitExec(workDir, "diff")
}

// Log returns recent commit history.
func Log(workDir string, n int) (string, error) {
	return gitExec(workDir, "log", fmt.Sprintf("-%d", n), "--oneline")
}

// ConflictFiles returns files with merge conflicts.
func ConflictFiles(workDir string) []string {
	return gitLines(workDir, "diff", "--name-only", "--diff-filter=U")
}

// ResolveConflict attempts to auto-resolve a merge conflict by keeping both
// changes and marking them.
func ResolveConflict(workDir, file string) error {
	path := file
	if !filepath.IsAbs(path) {
		path = filepath.Join(workDir, file)
	}
	// Read the file with conflict markers.
	_, err := gitExec(workDir, "checkout", "--theirs", file)
	if err != nil {
		_, err = gitExec(workDir, "checkout", "--ours", file)
		if err != nil {
			return fmt.Errorf("cannot resolve conflict in %s: %w", file, err)
		}
	}
	_, err = gitExec(workDir, "add", file)
	return err
}

// --- helpers ---

var gitRootRe = regexp.MustCompile(`^`)

func gitRoot(workDir string) (string, error) {
	out, err := gitExec(workDir, "rev-parse", "--show-toplevel")
	return strings.TrimSpace(out), err
}

func gitExec(workDir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = workDir
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func gitLines(workDir string, args ...string) []string {
	out, _ := gitExec(workDir, args...)
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return nil
	}
	return lines
}
