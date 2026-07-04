package config

import "strings"

// Effort selects how hard ZED works on a task. Higher levels unlock deeper
// reasoning, more ReAct iterations, larger token budgets, up-front planning,
// and self-verification loops — trading speed and cost for capability on
// increasingly complex, real-world projects.
type Effort struct {
	// Name is the canonical identifier (also the /command alias).
	Name string
	// Label is a short human-friendly title shown in the UI.
	Label string
	// Multiplier is a rough "how much more advanced" factor vs. normal,
	// used purely for display (e.g. "40×").
	Multiplier int
	// MaxSteps caps the ReAct loop for this level.
	MaxSteps int
	// MaxTokens caps output tokens per model turn.
	MaxTokens int
	// FastModel overrides the primary model when set. Goal mode and similar
	// long-running levels benefit from a fast, cheap model for low latency.
	FastModel string
	// PlanFirst makes the agent produce a structured plan before acting.
	PlanFirst bool
	// SelfVerify makes the agent build/test its own work and fix regressions.
	SelfVerify bool
	// DeepResearch makes the agent investigate the codebase end-to-end first.
	DeepResearch bool
	// GoalMode turns on fully autonomous, end-to-end goal execution.
	GoalMode bool
	// DreamMode turns on the all-feature Agent OS Pro Max controller.
	DreamMode bool
	// Description is a one-line summary of when to use this level.
	Description string

	// === REAL EFFORT BEHAVIOR FIELDS ===
	// These control actual agent behavior — not just display.

	// ParallelTools allows parallel tool execution (multiple tools per step).
	ParallelTools bool
	// UseSwarm enables multi-agent orchestration for complex tasks.
	UseSwarm bool
	// SwarmSize is the max number of parallel sub-agents to spawn.
	SwarmSize int
	// AutoDebug enables self-healing: auto build/test after file changes.
	AutoDebug bool
	// MaxDebugRetries is how many times to retry fixing a failing build.
	MaxDebugRetries int
	// RunQualityCheck runs code quality analysis after changes.
	RunQualityCheck bool
	// RunSecurityScan runs security vulnerability scanning after changes.
	RunSecurityScan bool
	// ContextBudgetMultiplier scales the context window (1.0 = model default).
	ContextBudgetMultiplier float64
	// Temperature controls LLM creativity (0 = deterministic, 1 = creative).
	Temperature float64
	// EnableWebSearch allows the agent to search the web for information.
	EnableWebSearch bool
	// EnableMemory allows the agent to use persistent memory.
	EnableMemory bool
	// EnableAST enables AST-based code analysis.
	EnableAST bool
	// EnableCI runs the full CI pipeline after changes.
	EnableCI bool
	// VerbosePlanning produces detailed step-by-step plans.
	VerbosePlanning bool
	// MaxFileReads limits how many files to read per turn (0 = unlimited).
	MaxFileReads int

	// EnterprisePreflight runs deterministic local analyzers before the LLM acts.
	EnterprisePreflight bool
	// EnterpriseReleaseGate runs policy/release gates after meaningful changes.
	EnterpriseReleaseGate bool
	// EvidencePack writes compliance evidence after goal-mode work.
	EvidencePack bool
	// PatchBundle writes a reviewable patch bundle for changed work.
	PatchBundle bool
	// MilestonePlanning creates a real enterprise milestone plan for the task.
	MilestonePlanning bool
	// RiskHeatmap runs project risk heatmap analysis.
	RiskHeatmap bool
	// WorkJournal records autonomous progress in .zed-work-journal.jsonl.
	WorkJournal bool
}

// Effort level names. These double as slash-command aliases.
const (
	EffortNormal        = "normal"
	EffortUltra         = "ultraeffort"
	EffortUltraMax      = "ultramax"
	EffortUltraComboMax = "ultracombomax"
	EffortGoal          = "goal"
	EffortDream         = "dream"
)

// DefaultEffort is used when nothing is configured.
const DefaultEffort = EffortNormal

// efforts is the ordered catalog of available effort levels.
var efforts = []Effort{
	{
		Name:        EffortNormal,
		Label:       "Normal",
		Multiplier:  1,
		MaxSteps:    25,
		MaxTokens:   128000,
		Description: "Fast, balanced. Everyday edits, questions, and small tasks.",
		// Real behavior: minimal overhead, fast responses
		ParallelTools:            false,
		UseSwarm:                 false,
		SwarmSize:                0,
		AutoDebug:                false,
		MaxDebugRetries:          0,
		RunQualityCheck:          false,
		RunSecurityScan:          false,
		ContextBudgetMultiplier:  1.0,
		Temperature:              0.3,
		EnableWebSearch:          false,
		EnableMemory:             false,
		EnableAST:                false,
		EnableCI:                 false,
		VerbosePlanning:          false,
		MaxFileReads:             10,
		EnterprisePreflight:     false,
		EnterpriseReleaseGate:   false,
		EvidencePack:            false,
		PatchBundle:             false,
		MilestonePlanning:       false,
		RiskHeatmap:             false,
		WorkJournal:             false,
	},
	{
		Name:        EffortUltra,
		Label:       "Ultra Effort",
		Multiplier:  10,
		MaxSteps:    60,
		MaxTokens:   128000,
		PlanFirst:   true,
		SelfVerify:  true,
		Description: "10× — real planning, enterprise preflight, self-verify, and auto-debug for complex work.",
		// Real behavior: planning + self-verify + auto-debug
		ParallelTools:            true,
		UseSwarm:                 false,
		SwarmSize:                0,
		AutoDebug:                true,
		MaxDebugRetries:          3,
		RunQualityCheck:          true,
		RunSecurityScan:          false,
		ContextBudgetMultiplier:  1.0,
		Temperature:              0.2,
		EnableWebSearch:          true,
		EnableMemory:             true,
		EnableAST:                true,
		EnableCI:                 false,
		VerbosePlanning:          true,
		MaxFileReads:             20,
		EnterprisePreflight:     true,
		EnterpriseReleaseGate:   false,
		EvidencePack:            false,
		PatchBundle:             false,
		MilestonePlanning:       true,
		RiskHeatmap:             true,
		WorkJournal:             true,
	},
	{
		Name:         EffortUltraMax,
		Label:        "Ultra Max",
		Multiplier:   50,
		MaxSteps:     120,
		MaxTokens:    128000,
		PlanFirst:    true,
		SelfVerify:   true,
		DeepResearch: true,
		Description:  "50× — exhaustive codebase research, risk heatmap, CI, and enterprise verification.",
		// Real behavior: deep research + swarm + security + CI
		ParallelTools:            true,
		UseSwarm:                 true,
		SwarmSize:                3,
		AutoDebug:                true,
		MaxDebugRetries:          5,
		RunQualityCheck:          true,
		RunSecurityScan:          true,
		ContextBudgetMultiplier:  1.5,
		Temperature:              0.1,
		EnableWebSearch:          true,
		EnableMemory:             true,
		EnableAST:                true,
		EnableCI:                 true,
		VerbosePlanning:          true,
		MaxFileReads:             50,
		EnterprisePreflight:     true,
		EnterpriseReleaseGate:   true,
		EvidencePack:            false,
		PatchBundle:             true,
		MilestonePlanning:       true,
		RiskHeatmap:             true,
		WorkJournal:             true,
	},
	{
		Name:         EffortUltraComboMax,
		Label:        "Ultra Combo Max",
		Multiplier:   120,
		MaxSteps:     200,
		MaxTokens:    128000,
		PlanFirst:    true,
		SelfVerify:   true,
		DeepResearch: true,
		Description:  "120× — maximum enterprise engineering with swarm, release gates, patch bundles, and evidence.",
		// Real behavior: everything on, maximum swarm, maximum retries
		ParallelTools:            true,
		UseSwarm:                 true,
		SwarmSize:                5,
		AutoDebug:                true,
		MaxDebugRetries:          10,
		RunQualityCheck:          true,
		RunSecurityScan:          true,
		ContextBudgetMultiplier:  2.0,
		Temperature:              0.05,
		EnableWebSearch:          true,
		EnableMemory:             true,
		EnableAST:                true,
		EnableCI:                 true,
		VerbosePlanning:          true,
		MaxFileReads:             0, // unlimited
		EnterprisePreflight:     true,
		EnterpriseReleaseGate:   true,
		EvidencePack:            true,
		PatchBundle:             true,
		MilestonePlanning:       true,
		RiskHeatmap:             true,
		WorkJournal:             true,
	},
	{
		Name:         EffortGoal,
		Label:        "Goal Mode",
		Multiplier:   200,
		MaxSteps:     300,
		MaxTokens:    128000,
		PlanFirst:    true,
		SelfVerify:   true,
		DeepResearch: true,
		GoalMode:     true,
		Description:  "200× — autonomous goal execution with milestones, release gates, evidence packs, patch bundles, and work journal.",
		// Real behavior: same as ultracombomax + goal mode autonomy, maxed out
		ParallelTools:            true,
		UseSwarm:                 true,
		SwarmSize:                10,
		AutoDebug:                true,
		MaxDebugRetries:          20,
		RunQualityCheck:          true,
		RunSecurityScan:          true,
		ContextBudgetMultiplier:  3.0,
		Temperature:              0.02,
		EnableWebSearch:          true,
		EnableMemory:             true,
		EnableAST:                true,
		EnableCI:                 true,
		VerbosePlanning:          true,
		MaxFileReads:             0, // unlimited
		EnterprisePreflight:     true,
		EnterpriseReleaseGate:   true,
		EvidencePack:            true,
		PatchBundle:             true,
		MilestonePlanning:       true,
		RiskHeatmap:             true,
		WorkJournal:             true,
	},
	{
		Name:         EffortDream,
		Label:        "Dream Mode",
		Multiplier:   1000,
		MaxSteps:     500,
		MaxTokens:    128000,
		PlanFirst:    true,
		SelfVerify:   true,
		DeepResearch: true,
		GoalMode:     true,
		DreamMode:    true,
		Description:  "1000× — Dream Mode controls every enterprise, Agent OS, Pro Max, mission, memory, artifact, and verification feature.",
		ParallelTools:            true,
		UseSwarm:                 true,
		SwarmSize:                20,
		AutoDebug:                true,
		MaxDebugRetries:          50,
		RunQualityCheck:          true,
		RunSecurityScan:          true,
		ContextBudgetMultiplier:  4.0,
		Temperature:              0.01,
		EnableWebSearch:          true,
		EnableMemory:             true,
		EnableAST:                true,
		EnableCI:                 true,
		VerbosePlanning:          true,
		MaxFileReads:             0, // unlimited
		EnterprisePreflight:     true,
		EnterpriseReleaseGate:   true,
		EvidencePack:            true,
		PatchBundle:             true,
		MilestonePlanning:       true,
		RiskHeatmap:             true,
		WorkJournal:             true,
	},
}

// Efforts returns the catalog of effort levels in display order.
func Efforts() []Effort { return efforts }

// LookupEffort returns the effort profile for a name (case-insensitive).
// The second return value reports whether the name was recognized.
func LookupEffort(name string) (Effort, bool) {
	n := strings.ToLower(strings.TrimSpace(name))
	for _, e := range efforts {
		if e.Name == n {
			return e, true
		}
	}
	return efforts[0], false
}

// EffortProfile returns the profile for a name, falling back to Normal.
func EffortProfile(name string) Effort {
	e, _ := LookupEffort(name)
	return e
}

// IsGoalEffort reports whether the given effort level has goal-mode autonomy
// enabled. Goal and dream mode qualify for autonomous, end-to-end execution.
func IsGoalEffort(e Effort) bool {
	return e.GoalMode
}

// IsGoalEffortName is a convenience wrapper for callers that only have the
// effort name string (e.g. from config).
func IsGoalEffortName(name string) bool {
	e, ok := LookupEffort(name)
	if !ok {
		return false
	}
	return e.GoalMode
}
