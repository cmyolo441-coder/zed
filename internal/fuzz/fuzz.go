// Package fuzz provides fuzz testing and property-based test generation.
// It auto-generates fuzz tests for functions, discovers edge cases, and
// runs mutation testing to verify test quality.
package fuzz

import (
	"fmt"
	"strings"
)

// FuzzTest represents a generated fuzz test.
type FuzzTest struct {
	Target    string // function name to fuzz
	Language  string
	Code      string
	Property  string // property being tested
}

// GenerateGoFuzz creates a Go fuzz test for a function.
func GenerateGoFuzz(funcName, paramType string) *FuzzTest {
	code := fmt.Sprintf(`package fuzz_test

import "testing"

func Fuzz%s(f *testing.F) {
	// Seed with known edge cases.
	f.Add(%s) // zero value
	f.Add(%s) // typical value

	f.Fuzz(func(t *testing.T, input %s) {
		// Property: the function should not panic on any input.
		// Add specific properties based on the function's contract.
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("%s panicked on input %%v: %%v", input, r)
			}
		}()

		// result := %s(input)
		// Property 1: output should be deterministic
		// Property 2: output should satisfy invariants
		// Property 3: no side effects on input
	})
}
`, funcName, zeroValue(paramType), typicalValue(paramType), paramType, funcName, funcName)

	return &FuzzTest{
		Target:   funcName,
		Language: "go",
		Code:     code,
		Property: "no panic on any input; deterministic output",
	}
}

// GeneratePythonFuzz creates a Python property-based test using hypothesis.
func GeneratePythonFuzz(funcName, paramType string) *FuzzTest {
	code := fmt.Sprintf(`from hypothesis import given, strategies as st, settings
import pytest

# Property-based test: the function should handle any valid input.
@given(st_%s())
@settings(max_examples=100)
def test_%s_property(input_val):
    # Property: function should not raise on valid inputs.
    result = %s(input_val)
    # Add specific property assertions:
    # assert result is not None
    # assert isinstance(result, expected_type)
    pass

def st_%s():
    # Define strategy based on parameter type.
    # Adjust this to match the function's expected input domain.
    if "%s" == "int":
        return st.integers()
    elif "%s" == "str":
        return st.text()
    elif "%s" == "float":
        return st.floats()
    else:
        return st.text()
`, paramType, funcName, funcName, paramType, paramType, paramType, paramType)

	return &FuzzTest{
		Target:   funcName,
		Language: "python",
		Code:     code,
		Property: "handles any valid input without error",
	}
}

// MutationTest represents a mutation testing result.
type MutationTest struct {
	Function  string
	Mutants   int
	Killed    int // mutants caught by tests
	Survived  int // mutants NOT caught (test quality gap)
	Score     float64 // killed/total * 100
}

// MutationScore calculates test quality: what % of mutations are caught.
func MutationScore(funcName string, mutants, killed int) MutationTest {
	score := 0.0
	if mutants > 0 {
		score = float64(killed) / float64(mutants) * 100
	}
	return MutationTest{
		Function: funcName,
		Mutants:  mutants,
		Killed:   killed,
		Survived: mutants - killed,
		Score:    score,
	}
}

func (m MutationTest) Summary() string {
	return fmt.Sprintf("🧬 Mutation Testing for %s:\n  Mutants: %d | Killed: %d | Survived: %d\n  Score: %.1f%%\n  %s",
		m.Function, m.Mutants, m.Killed, m.Survived, m.Score,
		mutationVerdict(m.Score))
}

func mutationVerdict(score float64) string {
	if score >= 90 {
		return "✅ Excellent test quality"
	} else if score >= 70 {
		return "⚠️  Good, but some gaps"
	} else if score >= 50 {
		return "🧡 Moderate — tests miss many mutations"
	}
	return "❌ Poor — tests are not catching mutations"
}

// EdgeCases returns common edge case inputs for a type.
func EdgeCases(typeName string) []string {
	switch strings.ToLower(typeName) {
	case "int", "int64", "int32":
		return []string{"0", "-1", "1", "2147483647", "-2147483648", "999999999"}
	case "string", "str":
		return []string{`""`, `"a"`, `"hello world"`, `"<script>alert(1)</script>"`, `"null"`, strings.Repeat("x", 10000)}
	case "float", "float64":
		return []string{"0.0", "-0.0", "1.0", "-1.0", "3.14159", "1e308", "-1e308"}
	case "bool":
		return []string{"true", "false"}
	default:
		return []string{"nil", "empty", "max", "min"}
	}
}

func zeroValue(t string) string {
	switch strings.ToLower(t) {
	case "int", "int64", "int32", "float", "float64":
		return "0"
	case "string":
		return `""`
	case "bool":
		return "false"
	default:
		return "nil"
	}
}

func typicalValue(t string) string {
	switch strings.ToLower(t) {
	case "int", "int64", "int32":
		return "42"
	case "string":
		return `"example"`
	case "float", "float64":
		return "3.14"
	case "bool":
		return "true"
	default:
		return "nil"
	}
}
