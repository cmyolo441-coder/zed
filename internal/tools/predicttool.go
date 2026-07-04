package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/cmyolo441-coder/zed/internal/predict"
)

type PredictSuggest struct {
	Engine *predict.Engine
}

func (t *PredictSuggest) Name() string { return "predict_suggest" }
func (t *PredictSuggest) Description() string {
	return "Return proactive suggestions based on learned user patterns. " +
		"Call this to anticipate what the user might ask next. " +
		"Args: {\"input\": \"current user input (optional)\"}"
}
func (t *PredictSuggest) RequiresApproval() bool { return false }
func (t *PredictSuggest) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"input": map[string]any{"type": "string", "description": "Current user input for context."},
		},
	}
}
func (t *PredictSuggest) Execute(_ context.Context, args string) (string, error) {
	var a struct {
		Input string `json:"input"`
	}
	if err := parseArgs(args, &a); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	var b strings.Builder
	b.WriteString("📊 Intent Prediction:\n\n")
	if a.Input != "" {
		predictions := t.Engine.Predict(a.Input)
		if len(predictions) > 0 {
			b.WriteString("  Predicted next actions:\n")
			for _, p := range predictions {
				fmt.Fprintf(&b, "    • %s\n", p)
			}
		} else {
			b.WriteString("  (no predictions yet — learning in progress)\n")
		}
	}
	suggestions := t.Engine.Suggestions()
	if len(suggestions) > 0 {
		b.WriteString("\n  Learned patterns:\n")
		for _, s := range suggestions {
			fmt.Fprintf(&b, "    • %s\n", s)
		}
	}
	return b.String(), nil
}

var _ Tool = (*PredictSuggest)(nil)
