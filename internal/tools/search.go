package tools

import (
	"bufio"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// skipDir returns true for directories that should never be scanned.
func skipDir(name string) bool {
	switch name {
	case ".git", "node_modules", "vendor", "dist", "build", ".idea", ".vscode":
		return true
	}
	return false
}

// ---------------------------------------------------------------------------
// Grep: search file contents for a substring.
// ---------------------------------------------------------------------------

type Grep struct{ WorkDir string }

func (t *Grep) Name() string { return "grep" }
func (t *Grep) Description() string {
	return "Search file contents recursively for a substring. Returns matching file:line: text. Optionally restrict by file extension."
}
func (t *Grep) RequiresApproval() bool { return false }
func (t *Grep) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{"type": "string", "description": "Text to search for."},
			"ext":   map[string]any{"type": "string", "description": "Optional file extension filter, e.g. '.go'."},
		},
		"required": []string{"query"},
	}
}

func (t *Grep) Execute(_ context.Context, args string) (string, error) {
	var a struct {
		Query string `json:"query"`
		Ext   string `json:"ext"`
	}
	if err := parseArgs(args, &a); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if a.Query == "" {
		return "", fmt.Errorf("empty query")
	}

	var b strings.Builder
	matches := 0
	const maxMatches = 200

	err := filepath.WalkDir(t.WorkDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if skipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if a.Ext != "" && !strings.HasSuffix(d.Name(), a.Ext) {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()

		rel, _ := filepath.Rel(t.WorkDir, path)
		scanner := bufio.NewScanner(f)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		line := 0
		for scanner.Scan() {
			line++
			text := scanner.Text()
			if strings.Contains(text, a.Query) {
				fmt.Fprintf(&b, "%s:%d: %s\n", rel, line, strings.TrimSpace(text))
				matches++
				if matches >= maxMatches {
					return fmt.Errorf("__stop__")
				}
			}
		}
		return nil
	})
	if err != nil && err.Error() != "__stop__" {
		return "", err
	}
	if matches == 0 {
		return "No matches found.", nil
	}
	if matches >= maxMatches {
		b.WriteString(fmt.Sprintf("\n(stopped at %d matches)\n", maxMatches))
	}
	return b.String(), nil
}

// ---------------------------------------------------------------------------
// FindFiles: find files by glob pattern.
// ---------------------------------------------------------------------------

type FindFiles struct{ WorkDir string }

func (t *FindFiles) Name() string { return "find_files" }
func (t *FindFiles) Description() string {
	return "Find files by name using a glob pattern (e.g. '*.go', 'internal/*.go'). Returns matching paths."
}
func (t *FindFiles) RequiresApproval() bool { return false }
func (t *FindFiles) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"pattern": map[string]any{"type": "string", "description": "Glob pattern to match filenames."},
		},
		"required": []string{"pattern"},
	}
}

func (t *FindFiles) Execute(_ context.Context, args string) (string, error) {
	var a struct {
		Pattern string `json:"pattern"`
	}
	if err := parseArgs(args, &a); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if a.Pattern == "" {
		return "", fmt.Errorf("empty pattern")
	}

	var b strings.Builder
	count := 0
	err := filepath.WalkDir(t.WorkDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if skipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		ok, _ := filepath.Match(a.Pattern, d.Name())
		if !ok {
			// also try matching against the relative path
			rel, _ := filepath.Rel(t.WorkDir, path)
			ok, _ = filepath.Match(a.Pattern, filepath.ToSlash(rel))
		}
		if ok {
			rel, _ := filepath.Rel(t.WorkDir, path)
			fmt.Fprintf(&b, "%s\n", filepath.ToSlash(rel))
			count++
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if count == 0 {
		return "No files found.", nil
	}
	return b.String(), nil
}
