// Package diff computes and renders textual differences between two versions
// of a file. It implements a Myers-style longest-common-subsequence diff and
// produces unified-diff output suitable for display in the terminal.
package diff

import (
	"fmt"
	"strings"
)

// Op is the kind of change for a single line.
type Op int

const (
	OpEqual Op = iota
	OpInsert
	OpDelete
)

// Line is one line of diff output with its operation.
type Line struct {
	Op   Op
	Text string
}

// Hunk is a contiguous group of changes with surrounding context.
type Hunk struct {
	OldStart, OldCount int
	NewStart, NewCount int
	Lines              []Line
}

// Result is the full diff between two texts.
type Result struct {
	Path        string
	Hunks       []Hunk
	Insertions  int
	Deletions   int
	Unchanged   int
	Binary      bool
}

// Compute returns the line-level diff between old and new text.
func Compute(path, oldText, newText string) Result {
	res := Result{Path: path}
	if isBinary(oldText) || isBinary(newText) {
		res.Binary = true
		return res
	}
	oldLines := splitLines(oldText)
	newLines := splitLines(newText)

	ops := myers(oldLines, newLines)

	// Count stats.
	for _, o := range ops {
		switch o.Op {
		case OpInsert:
			res.Insertions++
		case OpDelete:
			res.Deletions++
		case OpEqual:
			res.Unchanged++
		}
	}
	res.Hunks = buildHunks(ops, 3)
	return res
}

// change is an intermediate op tied to its source/target line numbers.
type change struct {
	Op       Op
	Text     string
	OldIndex int // 0-based index into old lines (-1 for pure inserts)
	NewIndex int // 0-based index into new lines (-1 for pure deletes)
}

// myers computes the diff using the classic dynamic-programming LCS. This is
// O(n*m) memory but simple and correct; ZED diffs single files so this is fine.
func myers(a, b []string) []change {
	n, m := len(a), len(b)
	// lcs[i][j] = length of LCS of a[i:] and b[j:]
	lcs := make([][]int, n+1)
	for i := range lcs {
		lcs[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if a[i] == b[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else if lcs[i+1][j] >= lcs[i][j+1] {
				lcs[i][j] = lcs[i+1][j]
			} else {
				lcs[i][j] = lcs[i][j+1]
			}
		}
	}

	var out []change
	i, j := 0, 0
	for i < n && j < m {
		switch {
		case a[i] == b[j]:
			out = append(out, change{Op: OpEqual, Text: a[i], OldIndex: i, NewIndex: j})
			i++
			j++
		case lcs[i+1][j] >= lcs[i][j+1]:
			out = append(out, change{Op: OpDelete, Text: a[i], OldIndex: i, NewIndex: -1})
			i++
		default:
			out = append(out, change{Op: OpInsert, Text: b[j], OldIndex: -1, NewIndex: j})
			j++
		}
	}
	for ; i < n; i++ {
		out = append(out, change{Op: OpDelete, Text: a[i], OldIndex: i, NewIndex: -1})
	}
	for ; j < m; j++ {
		out = append(out, change{Op: OpInsert, Text: b[j], OldIndex: -1, NewIndex: j})
	}
	return out
}

// buildHunks groups changes into hunks with the given amount of context.
func buildHunks(ops []change, context int) []Hunk {
	var hunks []Hunk

	// Mark which ops are changes vs equal.
	changed := func(o change) bool { return o.Op != OpEqual }

	i := 0
	for i < len(ops) {
		if !changed(ops[i]) {
			i++
			continue
		}
		// Found a change; expand backwards for context.
		start := i
		for start > 0 && !changed(ops[start-1]) && i-start < context {
			start--
		}
		// Walk forward to the end of this change cluster (allowing small gaps).
		end := i
		for end < len(ops) {
			if changed(ops[end]) {
				end++
				continue
			}
			// look ahead: if another change is within 2*context, keep going
			gap := 0
			k := end
			for k < len(ops) && !changed(ops[k]) {
				gap++
				k++
			}
			if k < len(ops) && gap <= context*2 {
				end = k
			} else {
				break
			}
		}
		// Add trailing context.
		tail := 0
		for end < len(ops) && !changed(ops[end]) && tail < context {
			end++
			tail++
		}

		h := makeHunk(ops[start:end])
		hunks = append(hunks, h)
		i = end
	}
	return hunks
}

func makeHunk(ops []change) Hunk {
	h := Hunk{OldStart: -1, NewStart: -1}
	for _, o := range ops {
		var op Op
		switch o.Op {
		case OpEqual:
			op = OpEqual
		case OpInsert:
			op = OpInsert
		case OpDelete:
			op = OpDelete
		}
		h.Lines = append(h.Lines, Line{Op: op, Text: o.Text})

		if o.OldIndex >= 0 {
			if h.OldStart == -1 {
				h.OldStart = o.OldIndex + 1
			}
			h.OldCount++
		}
		if o.NewIndex >= 0 {
			if h.NewStart == -1 {
				h.NewStart = o.NewIndex + 1
			}
			h.NewCount++
		}
	}
	if h.OldStart == -1 {
		h.OldStart = 0
	}
	if h.NewStart == -1 {
		h.NewStart = 0
	}
	return h
}

// Unified renders the diff in unified-diff text format (no color).
func (r Result) Unified() string {
	if r.Binary {
		return fmt.Sprintf("Binary file %s differs\n", r.Path)
	}
	if len(r.Hunks) == 0 {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "--- a/%s\n", r.Path)
	fmt.Fprintf(&b, "+++ b/%s\n", r.Path)
	for _, h := range r.Hunks {
		fmt.Fprintf(&b, "@@ -%d,%d +%d,%d @@\n", h.OldStart, h.OldCount, h.NewStart, h.NewCount)
		for _, ln := range h.Lines {
			switch ln.Op {
			case OpEqual:
				fmt.Fprintf(&b, " %s\n", ln.Text)
			case OpInsert:
				fmt.Fprintf(&b, "+%s\n", ln.Text)
			case OpDelete:
				fmt.Fprintf(&b, "-%s\n", ln.Text)
			}
		}
	}
	return b.String()
}

// Colorizer renders a diff line with ANSI colors. The caller supplies styling
// functions so this package stays free of UI dependencies.
type Colorizer struct {
	Add     func(string) string
	Del     func(string) string
	Context func(string) string
	Header  func(string) string
}

// Render produces a colorized unified diff using the given colorizer.
func (r Result) Render(c Colorizer) string {
	if r.Binary {
		return c.Header(fmt.Sprintf("Binary file %s differs", r.Path)) + "\n"
	}
	if len(r.Hunks) == 0 {
		return c.Context("(no changes)") + "\n"
	}
	var b strings.Builder
	b.WriteString(c.Header(fmt.Sprintf("%s  +%d -%d", r.Path, r.Insertions, r.Deletions)) + "\n")
	for _, h := range r.Hunks {
		b.WriteString(c.Header(fmt.Sprintf("@@ -%d,%d +%d,%d @@", h.OldStart, h.OldCount, h.NewStart, h.NewCount)) + "\n")
		for _, ln := range h.Lines {
			switch ln.Op {
			case OpEqual:
				b.WriteString(c.Context("  " + ln.Text))
			case OpInsert:
				b.WriteString(c.Add("+ " + ln.Text))
			case OpDelete:
				b.WriteString(c.Del("- " + ln.Text))
			}
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// Summary returns a one-line "+X -Y" change summary.
func (r Result) Summary() string {
	return fmt.Sprintf("+%d -%d", r.Insertions, r.Deletions)
}

// HasChanges reports whether the diff contains any modifications.
func (r Result) HasChanges() bool {
	return r.Insertions > 0 || r.Deletions > 0 || r.Binary
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	s = strings.ReplaceAll(s, "\r\n", "\n")
	lines := strings.Split(s, "\n")
	// Drop a trailing empty element from a final newline.
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func isBinary(s string) bool {
	// Heuristic: presence of a NUL byte in the first 8KB.
	limit := len(s)
	if limit > 8192 {
		limit = 8192
	}
	for i := 0; i < limit; i++ {
		if s[i] == 0 {
			return true
		}
	}
	return false
}
