package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/cmyolo441-coder/zed/internal/diff"
	"github.com/cmyolo441-coder/zed/internal/snapshot"
)

// resolvePath keeps tool access within the working directory tree. It is
// symlink/path-traversal aware and accepts empty paths as the project root.
func resolvePath(workDir, p string) (string, error) {
	if strings.TrimSpace(p) == "" {
		p = "."
	}
	root, err := filepath.Abs(workDir)
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(p) {
		p = filepath.Join(root, p)
	}
	abs, err := filepath.Abs(filepath.Clean(p))
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(root, abs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q is outside the working directory", p)
	}
	return abs, nil
}

// sanitizeJSON attempts to repair common LLM JSON mistakes:
//   - literal newlines inside string values (should be \n)
//   - unescaped backslashes before special characters
//   - trailing commas before } or ]
//
// If the input is already valid JSON, it is returned unchanged.
func sanitizeJSON(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		parts := strings.Split(s, "```")
		if len(parts) >= 3 {
			s = parts[1]
			s = strings.TrimPrefix(strings.TrimSpace(s), "json")
		}
	}
	if !strings.HasPrefix(strings.TrimSpace(s), "{") {
		if i := strings.Index(s, "{"); i >= 0 {
			if j := strings.LastIndex(s, "}"); j > i {
				s = s[i : j+1]
			}
		}
	}
	s = strings.NewReplacer("“", "\"", "”", "\"", "‘", "'", "’", "'").Replace(s)
	// Quick check: if it parses already, return it.
	if json.Valid([]byte(s)) {
		return s
	}

	fixed := s

	// Fix literal newlines and carriage returns inside strings.
	// We process character-by-character to stay inside string boundaries.
	var buf strings.Builder
	inStr := false
	escaped := false
	for i := 0; i < len(fixed); i++ {
		c := fixed[i]
		if escaped {
			buf.WriteByte(c)
			escaped = false
			continue
		}
		if inStr {
			if c == '\\' {
				buf.WriteByte(c)
				escaped = true
				continue
			}
			if c == '"' {
				inStr = false
				buf.WriteByte(c)
				continue
			}
			// Replace literal control characters inside strings.
			if c == '\n' {
				buf.WriteString(`\n`)
				continue
			}
			if c == '\r' {
				buf.WriteString(`\r`)
				continue
			}
			if c == '\t' {
				buf.WriteString(`\t`)
				continue
			}
			buf.WriteByte(c)
			continue
		}
		// Not inside a string.
		if c == '"' {
			inStr = true
		}
		buf.WriteByte(c)
	}
	fixed = buf.String()

	// Remove trailing commas before } or ] (common LLM mistake).
	re := regexp.MustCompile(`,\s*([}\]])`)
	fixed = re.ReplaceAllString(fixed, "$1")

	return fixed
}

// parseArgs unmarshals JSON args, attempting sanitization if the first parse fails.
func parseArgs(args string, out any) error {
	trimmed := strings.TrimSpace(args)
	if trimmed == "" {
		return fmt.Errorf("no arguments were provided (empty). Re-issue the call with all required fields filled in")
	}
	if err := json.Unmarshal([]byte(args), out); err != nil {
		fixed := sanitizeJSON(args)
		if err2 := json.Unmarshal([]byte(fixed), out); err2 != nil {
			return fmt.Errorf("arguments are not valid JSON (possibly cut off mid-write); "+
				"for large content, write the file in smaller pieces: %w", err2)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// ReadFile
// ---------------------------------------------------------------------------

type ReadFile struct{ WorkDir string }

func (t *ReadFile) Name() string { return "read_file" }
func (t *ReadFile) Description() string {
	return "Read the contents of a file. Optionally limit to a line range with 'offset' (0-indexed start line) and 'limit' (number of lines)."
}
func (t *ReadFile) RequiresApproval() bool { return false }
func (t *ReadFile) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path":   map[string]any{"type": "string", "description": "File path relative to the project root."},
			"offset": map[string]any{"type": "integer", "description": "Start line (0-indexed). Optional."},
			"limit":  map[string]any{"type": "integer", "description": "Number of lines to read. Optional."},
		},
		"required": []string{"path"},
	}
}
func (t *ReadFile) Execute(_ context.Context, args string) (string, error) {
	var a struct {
		Path   string `json:"path"`
		Offset int    `json:"offset"`
		Limit  int    `json:"limit"`
	}
	if err := parseArgs(args, &a); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	path, err := resolvePath(t.WorkDir, a.Path)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	if !utf8.Valid(data) || looksBinary(data) {
		info, _ := os.Stat(path)
		size := int64(len(data))
		if info != nil { size = info.Size() }
		return fmt.Sprintf("%s is a binary/non-UTF8 file (%d bytes). Use specialized tooling or inspect metadata, not text read.", a.Path, size), nil
	}
	lines := strings.Split(string(data), "\n")
	start := a.Offset
	if start < 0 {
		start = 0
	}
	if start > len(lines) {
		start = len(lines)
	}
	end := len(lines)
	if a.Limit > 0 && start+a.Limit < end {
		end = start + a.Limit
	}
	var b strings.Builder
	for i := start; i < end; i++ {
		fmt.Fprintf(&b, "%d\t%s\n", i+1, lines[i])
	}
	return b.String(), nil
}

// ---------------------------------------------------------------------------
// WriteFile (create or overwrite)
// ---------------------------------------------------------------------------

type WriteFile struct {
	WorkDir   string
	Snapshots *snapshot.Manager
}

func (t *WriteFile) Name() string { return "write_file" }
func (t *WriteFile) Description() string {
	return "Create a new file or overwrite an existing file with the given content."
}
func (t *WriteFile) RequiresApproval() bool { return true }
func (t *WriteFile) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path":    map[string]any{"type": "string", "description": "File path relative to the project root."},
			"content": map[string]any{"type": "string", "description": "Full file content."},
		},
		"required": []string{"path", "content"},
	}
}
func (t *WriteFile) Execute(_ context.Context, args string) (string, error) {
	var a struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := parseArgs(args, &a); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	path, err := resolvePath(t.WorkDir, a.Path)
	if err != nil {
		return "", err
	}
	// Capture a snapshot before mutating so the change is undoable.
	var snap *snapshot.Snapshot
	if t.Snapshots != nil {
		snap, _ = snapshot.Capture(path, t.Name())
	}
	// Compute a diff for user-visible feedback.
	oldContent, _ := os.ReadFile(path)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	if err := atomicWriteFile(path, []byte(a.Content), 0o644); err != nil {
		return "", err
	}
	if t.Snapshots != nil && snap != nil {
		t.Snapshots.Commit(snap)
	}
	d := diff.Compute(a.Path, string(oldContent), a.Content)
	return fmt.Sprintf("Wrote %s (%s)\n%s", a.Path, d.Summary(), truncate(d.Unified(), 2000)), nil
}

// ---------------------------------------------------------------------------
// AppendFile (append content to a file — ideal for building large files in
// multiple small tool calls without hitting output token limits)
// ---------------------------------------------------------------------------

type AppendFile struct {
	WorkDir   string
	Snapshots *snapshot.Manager
}

func (t *AppendFile) Name() string { return "append_file" }
func (t *AppendFile) Description() string {
	return "Append content to the end of a file (creating it if needed). Use this to build a large file across several small calls: write_file the first chunk, then append_file the remaining chunks. Keeps each tool call small enough to avoid output-token truncation."
}
func (t *AppendFile) RequiresApproval() bool { return true }
func (t *AppendFile) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path":    map[string]any{"type": "string", "description": "File path relative to the project root."},
			"content": map[string]any{"type": "string", "description": "Content to append to the end of the file."},
		},
		"required": []string{"path", "content"},
	}
}
func (t *AppendFile) Execute(_ context.Context, args string) (string, error) {
	var a struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := parseArgs(args, &a); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	path, err := resolvePath(t.WorkDir, a.Path)
	if err != nil {
		return "", err
	}
	var snap *snapshot.Snapshot
	if t.Snapshots != nil {
		snap, _ = snapshot.Capture(path, t.Name())
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return "", err
	}
	n, werr := f.WriteString(a.Content)
	closeErr := f.Close()
	if werr != nil {
		return "", werr
	}
	if closeErr != nil {
		return "", closeErr
	}
	if t.Snapshots != nil && snap != nil {
		t.Snapshots.Commit(snap)
	}
	return fmt.Sprintf("Appended %d bytes to %s", n, a.Path), nil
}

// ---------------------------------------------------------------------------
// EditFile (precise old_str/new_str replacement)
// ---------------------------------------------------------------------------

type EditFile struct {
	WorkDir   string
	Snapshots *snapshot.Manager
}

func (t *EditFile) Name() string { return "edit_file" }
func (t *EditFile) Description() string {
	return "Edit a file by replacing an exact block of text. 'old_str' must match exactly once (include surrounding context to make it unique). 'new_str' is the replacement."
}
func (t *EditFile) RequiresApproval() bool { return true }
func (t *EditFile) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path":    map[string]any{"type": "string", "description": "File path relative to the project root."},
			"old_str": map[string]any{"type": "string", "description": "Exact text to replace (must be unique)."},
			"new_str": map[string]any{"type": "string", "description": "Replacement text."},
		},
		"required": []string{"path", "old_str", "new_str"},
	}
}
func (t *EditFile) Execute(_ context.Context, args string) (string, error) {
	var a struct {
		Path   string `json:"path"`
		OldStr string `json:"old_str"`
		NewStr string `json:"new_str"`
	}
	if err := parseArgs(args, &a); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	path, err := resolvePath(t.WorkDir, a.Path)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	content := string(data)
	old := normalizeNewlines(a.OldStr)
	base := normalizeNewlines(content)
	count := strings.Count(base, old)
	if count == 0 {
		if candidate, ok := fuzzyBlockMatch(base, old); ok {
			old = candidate
			count = strings.Count(base, old)
		}
	}
	if count == 0 {
		return "", fmt.Errorf("old_str not found in %s. Tip: read the exact current lines first or use a smaller unique block", a.Path)
	}
	if count > 1 {
		return "", fmt.Errorf("old_str matched %d times in %s; add more surrounding context to make it unique", count, a.Path)
	}
	updated := strings.Replace(base, old, normalizeNewlines(a.NewStr), 1)
	var snap *snapshot.Snapshot
	if t.Snapshots != nil {
		snap, _ = snapshot.Capture(path, t.Name())
	}
	if err := atomicWriteFile(path, []byte(updated), 0o644); err != nil {
		return "", err
	}
	if t.Snapshots != nil && snap != nil {
		t.Snapshots.Commit(snap)
	}
	d := diff.Compute(a.Path, content, updated)
	return fmt.Sprintf("Edited %s (%s)\n%s", a.Path, d.Summary(), truncate(d.Unified(), 2000)), nil
}


func atomicWriteFile(path string, data []byte, perm os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-write-*")
	if err != nil { return err }
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil { _ = tmp.Close(); return err }
	if err := tmp.Close(); err != nil { return err }
	if err := os.Chmod(tmpName, perm); err != nil { return err }
	return os.Rename(tmpName, path)
}

func normalizeNewlines(s string) string { return strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\r", "\n") }

func fuzzyBlockMatch(content, old string) (string, bool) {
	needle := compactWhitespace(old)
	if needle == "" { return "", false }
	lines := strings.Split(content, "\n")
	oldLines := strings.Split(old, "\n")
	window := len(oldLines)
	if window < 1 { window = 1 }
	for extra := 0; extra <= 2; extra++ {
		for i := 0; i+window+extra <= len(lines); i++ {
			candidate := strings.Join(lines[i:i+window+extra], "\n")
			if compactWhitespace(candidate) == needle { return candidate, true }
		}
	}
	return "", false
}

func compactWhitespace(s string) string { return strings.Join(strings.Fields(s), " ") }

func looksBinary(data []byte) bool {
	limit := len(data); if limit > 4096 { limit = 4096 }
	if limit == 0 { return false }
	nul := 0
	for i := 0; i < limit; i++ { if data[i] == 0 { nul++ } }
	return nul > 0
}

// truncate limits a string to n bytes, appending an ellipsis marker if cut.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "\n…[diff truncated]"
}

// ---------------------------------------------------------------------------
// ListDir
// ---------------------------------------------------------------------------

type ListDir struct{ WorkDir string }

func (t *ListDir) Name() string { return "list_dir" }
func (t *ListDir) Description() string {
	return "List files and directories at the given path (default: project root)."
}
func (t *ListDir) RequiresApproval() bool { return false }
func (t *ListDir) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{"type": "string", "description": "Directory path relative to project root. Optional."},
		},
	}
}
func (t *ListDir) Execute(_ context.Context, args string) (string, error) {
	var a struct {
		Path string `json:"path"`
	}
	_ = parseArgs(args, &a)
	if a.Path == "" {
		a.Path = "."
	}
	dir, err := resolvePath(t.WorkDir, a.Path)
	if err != nil {
		return "", err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for _, e := range entries {
		if e.IsDir() {
			fmt.Fprintf(&b, "%s/\n", e.Name())
		} else {
			info, _ := e.Info()
			size := int64(0)
			if info != nil {
				size = info.Size()
			}
			fmt.Fprintf(&b, "%s  (%d bytes)\n", e.Name(), size)
		}
	}
	if b.Len() == 0 {
		return "(empty directory)", nil
	}
	return b.String(), nil
}
