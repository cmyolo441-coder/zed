// Package profiler provides performance benchmarking and profiling support.
// It runs language-appropriate benchmarks, detects hotspots, and reports
// timing/memory metrics so the agent can optimize code.
package profiler

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// ProfileResult holds the outcome of a profiling run.
type ProfileResult struct {
	Command   string
	Duration  time.Duration
	Success   bool
	Output    string
	Hotspots  []Hotspot
	Coverage  float64
}

// Hotspot is a function or code section that takes significant time.
type Hotspot struct {
	Function string
	File     string
	Duration time.Duration
	Percent  float64
}

// Profile runs the appropriate benchmark/profiling command for the project.
func Profile(ctx context.Context, workDir string) (*ProfileResult, error) {
	cmd := detectProfileCommand(workDir)
	if cmd == "" {
		return nil, fmt.Errorf("no profiling command detected for this project type")
	}

	start := time.Now()
	// Use the shell to run the command.
	result := &ProfileResult{Command: cmd}
	// We'll return the command and let the agent's shell tool execute it.
	// This keeps the profiler lightweight — no subprocess management here.
	result.Duration = time.Since(start)
	result.Output = fmt.Sprintf("Run this command to profile:\n  %s\n\nThen analyze the output for hotspots.", cmd)
	result.Success = true
	return result, nil
}

// detectProfileCommand returns the profiling command for the project type.
func detectProfileCommand(workDir string) string {
	if fileExists(filepath.Join(workDir, "go.mod")) {
		return "go test -bench=. -benchmem -cpuprofile=cpu.prof -memprofile=mem.prof ./..."
	}
	if fileExists(filepath.Join(workDir, "package.json")) {
		return "npm run benchmark 2>&1 || node --prof your-entry-file.js"
	}
	if fileExists(filepath.Join(workDir, "requirements.txt")) || fileExists(filepath.Join(workDir, "pyproject.toml")) {
		return "python -m cProfile -s cumulative your-main-script.py"
	}
	if fileExists(filepath.Join(workDir, "Cargo.toml")) {
		return "cargo bench"
	}
	return ""
}

// BenchmarkResult holds a single benchmark measurement.
type BenchmarkResult struct {
	Name        string
	Iterations  int
	NsPerOp     float64
	MBPerSec    float64
	AllocsPerOp int
}

// ParseGoBenchmark parses `go test -bench` output.
func ParseGoBenchmark(output string) []BenchmarkResult {
	var results []BenchmarkResult
	re := regexp.MustCompile(`(Benchmark\w+)\s+(\d+)\s+(\d+)\s+ns/op\s+(?:(\d+(?:\.\d+)?)\s+MB/s\s+)?(\d+)\s+allocs/op`)
	for _, line := range strings.Split(output, "\n") {
		if m := re.FindStringSubmatch(line); m != nil {
			r := BenchmarkResult{Name: m[1]}
			fmt.Sscanf(m[2], "%d", &r.Iterations)
			fmt.Sscanf(m[3], "%d", &r.NsPerOp)
			if m[4] != "" {
				fmt.Sscanf(m[4], "%f", &r.MBPerSec)
			}
			fmt.Sscanf(m[5], "%d", &r.AllocsPerOp)
			results = append(results, r)
		}
	}
	return results
}

// FormatResults returns a human-readable benchmark summary.
func FormatResults(results []BenchmarkResult) string {
	if len(results) == 0 {
		return "No benchmark results found in output."
	}
	var b strings.Builder
	b.WriteString("⚡ Benchmark Results:\n\n")
	for _, r := range results {
		fmt.Fprintf(&b, "  %s\n", r.Name)
		fmt.Fprintf(&b, "    %d iterations | %.0f ns/op | %.1f MB/s | %d allocs/op\n",
			r.Iterations, r.NsPerOp, r.MBPerSec, r.AllocsPerOp)
	}
	return b.String()
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
