package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/cmyolo441-coder/zed/internal/agent"
	"github.com/cmyolo441-coder/zed/internal/cache"
	"github.com/cmyolo441-coder/zed/internal/collab"
	"github.com/cmyolo441-coder/zed/internal/config"
	"github.com/cmyolo441-coder/zed/internal/dist"
	"github.com/cmyolo441-coder/zed/internal/enterprise"
	"github.com/cmyolo441-coder/zed/internal/index"
	"github.com/cmyolo441-coder/zed/internal/llm"
	"github.com/cmyolo441-coder/zed/internal/logging"
	"github.com/cmyolo441-coder/zed/internal/memory"
	"github.com/cmyolo441-coder/zed/internal/meta"
	"github.com/cmyolo441-coder/zed/internal/metrics"
	"github.com/cmyolo441-coder/zed/internal/plugin"
	"github.com/cmyolo441-coder/zed/internal/predict"
	"github.com/cmyolo441-coder/zed/internal/security"
	"github.com/cmyolo441-coder/zed/internal/session"
	"github.com/cmyolo441-coder/zed/internal/snapshot"
	"github.com/cmyolo441-coder/zed/internal/tools"
	"github.com/cmyolo441-coder/zed/internal/tui"
	"github.com/cmyolo441-coder/zed/internal/watcher"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "zed:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Allow simple CLI overrides: zed --model X --provider Y
	parseFlags(cfg)
	if handled, err := runEnterpriseCLI(cfg); handled || err != nil {
		return err
	}

	// No API-key gate here: the agent opens immediately (like Claude Code /
	// Codex). If no key is configured, the user runs /login inside the TUI to
	// set one. We only warn at startup — we never block the UI.
	if cfg.APIKey == "" {
		fmt.Fprintln(os.Stderr, "zed: no API key configured — run /login inside the agent to set one.")
	}

	// Structured file logging (never writes to the TUI's terminal).
	logger, err := logging.New(logging.Options{Level: logging.LevelInfo})
	if err != nil {
		return fmt.Errorf("init logging: %w", err)
	}
	defer logger.Close()
	logger.Info("zed starting", logging.F("provider", cfg.Provider), logging.F("model", cfg.Model))

	// Cache layer for LLM responses (3-10x speedup).
	cch := cache.New(5000)
	logger.Info("cache initialized", logging.F("max_entries", "5000"))

	client, err := llm.New(cfg)
	if err != nil {
		return fmt.Errorf("init llm: %w", err)
	}
	client = llm.WithCache(client, cch)

	// Enterprise subsystems.
	snapshots := snapshot.NewManager(200)
	policy := security.DefaultPolicy(cfg.WorkDir, cfg.AutoApply)
	sessions, err := session.NewStore("")
	if err != nil {
		logger.Warn("session store unavailable", logging.F("err", err.Error()))
	}


	// Code metrics dashboard.
	dash := metrics.New()

	// Intent prediction engine.
	eng := predict.New()

	// Plugin manager.
	pm := plugin.New()

	// Distributed agent network.
	net := dist.New()
	net.Register(&dist.Node{ID: "zed-local", Address: "localhost", Capacity: 5})

	// Collaboration session.
	collabSess := collab.NewSession()

	// File watcher for hot-reload.
	wtr := watcher.New(cfg.WorkDir, []string{".go", ".py", ".js", ".ts", ".jsx", ".tsx", ".rs"})
	wtr.OnChange(func(ev watcher.Event) {
		tests := watcher.RelevantTests(ev.Path)
		if len(tests) > 0 {
			logger.Info("file changed, relevant tests",
				logging.F("file", ev.Path),
				logging.F("tests", fmt.Sprintf("%v", tests)))
		}
	})
	go wtr.Start(context.Background())

	// Build the codebase index in the background so startup stays fast.
	idx := index.New(cfg.WorkDir)
	go func() {
		stats, err := idx.Build()
		if err != nil {
			logger.Warn("index build failed", logging.F("err", err.Error()))
			return
		}
		logger.Info("index built",
			logging.F("files", stats.Files),
			logging.F("symbols", stats.Symbols),
			logging.F("ms", stats.Duration.Milliseconds()))
	}()

	registry := buildRegistry(cfg, snapshots, idx, client, cch, dash, eng, pm, net, wtr, collabSess)
	ag := agent.New(cfg, client, registry)
	ag.SetSnapshots(snapshots)
	ag.SetPolicy(policy)
	ag.SetLogger(logger)

	model := tui.NewModel(cfg, ag, snapshots, sessions, idx, cch, dash, eng, pm, net, wtr, collabSess)
	// Note: we deliberately do NOT enable mouse capture (WithMouseCellMotion).
	// Capturing the mouse prevents the terminal's native text selection, so
	// users couldn't select/copy text. Leaving it off restores normal
	// click-drag selection and copy (Ctrl/Cmd+C, or the terminal's copy).
	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err = p.Run()
	return err
}

// buildRegistry registers every tool the agent can use.
func buildRegistry(cfg *config.Config, snapshots *snapshot.Manager, idx *index.Index, client llm.Client, cch *cache.Cache, dash *metrics.Dashboard, eng *predict.Engine, pm *plugin.Manager, net *dist.Network, wtr *watcher.Watcher, collabSess *collab.Session) *tools.Registry {
	r := tools.NewRegistry()
	wd := cfg.WorkDir

	// File operations.
	r.Register(&tools.ReadFile{WorkDir: wd})
	r.Register(&tools.WriteFile{WorkDir: wd, Snapshots: snapshots})
	r.Register(&tools.AppendFile{WorkDir: wd, Snapshots: snapshots})
	r.Register(&tools.EditFile{WorkDir: wd, Snapshots: snapshots})
	r.Register(&tools.ListDir{WorkDir: wd})
	r.Register(&tools.MissionControl{WorkDir: wd})

	// Search tools.
	r.Register(&tools.Grep{WorkDir: wd})
	r.Register(&tools.FindFiles{WorkDir: wd})
	r.Register(&tools.CodeSearch{Index: idx})
	r.Register(&tools.SymbolLookup{Index: idx})
	r.Register(&tools.SemanticSearch{Index: idx})

	// Shell execution is guarded by the agent security policy and explicit approval.
	r.Register(&tools.Shell{WorkDir: wd, Timeout: 4000 * time.Second})

	// Web search.
	r.Register(tools.NewWebSearch())

	// Persistent memory.
	mem := memory.New()
	r.Register(&tools.MemoryRemember{Store: mem})
	r.Register(&tools.MemoryRecall{Store: mem})

	// Multi-agent orchestration.
	r.Register(&tools.SpawnSwarm{Client: client, Registry: r})

	// AST code intelligence.
	r.Register(&tools.AnalyzeCode{WorkDir: wd})

	// Snapshot history + diff.
	r.Register(&tools.SnapshotHistory{Snapshots: snapshots})

	// Task decomposition.
	r.Register(&tools.DecomposeTask{})

	// Code quality analyzer.
	r.Register(&tools.CodeQuality{WorkDir: wd})

	// CI pipeline.
	r.Register(&tools.CIPipeline{WorkDir: wd})

	// Code scaffolding.
	r.Register(&tools.Scaffold{WorkDir: wd})

	// Profiling & benchmarks.
	r.Register(&tools.ProfileCode{WorkDir: wd})
	r.Register(&tools.ParseBenchmark{})

	// Code metrics dashboard.
	r.Register(&tools.CodeMetrics{Dashboard: dash})

	// Intent prediction.
	r.Register(&tools.PredictSuggest{Engine: eng})

	// Distributed network status.
	r.Register(&tools.DistStatus{Network: net})
	r.Register(&tools.DistSubmit{Network: net})

	// File watcher status.
	r.Register(&tools.WatcherStatus{Watcher: wtr})

	// Advanced enterprise tools.
	r.Register(&tools.SelfAnalyze{Profile: meta.New()})
	r.Register(&tools.ReasonTool{})
	r.Register(&tools.GitTool{WorkDir: wd})
	r.Register(&tools.FuzzTool{})
	r.Register(&tools.DiagramTool{WorkDir: wd})
	r.Register(&tools.SelfTestTool{})

	// Real enterprise governance, security, compliance, and operations controls.
	r.Register(&tools.EnterpriseSecretScan{WorkDir: wd})
	r.Register(&tools.EnterpriseSBOM{WorkDir: wd})
	r.Register(&tools.EnterpriseLicenseAudit{WorkDir: wd})
	r.Register(&tools.EnterpriseSupplyChainAudit{WorkDir: wd})
	r.Register(&tools.EnterprisePIIScan{WorkDir: wd})
	r.Register(&tools.EnterpriseConfigAudit{WorkDir: wd})
	r.Register(&tools.EnterprisePolicyAudit{WorkDir: wd})
	r.Register(&tools.EnterpriseSLOAudit{WorkDir: wd})
	r.Register(&tools.EnterpriseIntegrityManifest{WorkDir: wd})
	r.Register(&tools.EnterpriseAuditTrail{WorkDir: wd})
	r.Register(&tools.EnterpriseBackupReadiness{WorkDir: wd})
	r.Register(&tools.EnterpriseAPISurface{WorkDir: wd})
	r.Register(&tools.EnterpriseObservabilityAudit{WorkDir: wd})
	r.Register(&tools.EnterpriseContainerAudit{WorkDir: wd})
	r.Register(&tools.EnterpriseCodeOwnershipAudit{WorkDir: wd})
	r.Register(&tools.EnterpriseThreatModel{WorkDir: wd})
	r.Register(&tools.EnterpriseReleaseGate{WorkDir: wd})
	r.Register(&tools.EnterpriseEvidencePack{WorkDir: wd})
	r.Register(&tools.EnterprisePolicyGate{WorkDir: wd})
	r.Register(&tools.EnterpriseRiskWaiverAudit{WorkDir: wd})
	r.Register(&tools.EnterpriseSARIFExport{WorkDir: wd})
	r.Register(&tools.EnterpriseCITemplates{WorkDir: wd})
	r.Register(&tools.EnterprisePolicyInit{WorkDir: wd})
	r.Register(&tools.EnterpriseTaintAnalysis{WorkDir: wd})
	r.Register(&tools.EnterpriseMigrationAudit{WorkDir: wd})
	r.Register(&tools.EnterpriseEvidenceKeygen{WorkDir: wd})
	r.Register(&tools.EnterpriseEvidenceSign{WorkDir: wd})
	r.Register(&tools.EnterpriseEvidenceVerify{WorkDir: wd})
	r.Register(&tools.EnterpriseComplianceMap{WorkDir: wd})
	r.Register(&tools.EnterpriseArchitectureBrain{WorkDir: wd})
	r.Register(&tools.EnterpriseChangeImpact{WorkDir: wd})
	r.Register(&tools.EnterprisePatchBundle{WorkDir: wd})
	r.Register(&tools.EnterpriseBuildDoctor{WorkDir: wd})
	r.Register(&tools.EnterpriseADRGenerator{WorkDir: wd})
	r.Register(&tools.EnterpriseCriticalPath{WorkDir: wd})
	r.Register(&tools.EnterpriseAPIGuardian{WorkDir: wd})
	r.Register(&tools.EnterpriseGoConcurrencyAudit{WorkDir: wd})
	r.Register(&tools.EnterprisePerfHotspotPredictor{WorkDir: wd})
	r.Register(&tools.EnterpriseWorkJournal{WorkDir: wd})
	r.Register(&tools.EnterpriseRiskHeatmap{WorkDir: wd})
	r.Register(&tools.EnterpriseMilestoneExecutor{WorkDir: wd})
	for _, tool := range tools.NewAgentOSTools(wd) {
		r.Register(tool)
	}
	for _, tool := range tools.NewProMaxTools(wd) {
		r.Register(tool)
	}

	// Plugin executor — wraps plugin.Manager as a tool so the LLM can call plugins.
	r.Register(&tools.PluginExecute{Manager: pm})
	r.Register(&tools.PluginList{Manager: pm})

	return r
}

// parseFlags applies minimal command-line overrides.
func parseFlags(cfg *config.Config) {
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--model":
			if i+1 < len(args) {
				cfg.Model = args[i+1]
				i++
			}
		case "--provider":
			if i+1 < len(args) {
				cfg.Provider = args[i+1]
				i++
			}
		case "--effort":
			if i+1 < len(args) {
				cfg.Effort = args[i+1]
				i++
			}
		}
	}
}


// runEnterpriseCLI supports non-interactive enterprise checks used by CI. These
// modes do not require an LLM API key because they run deterministic local analyzers.
func runEnterpriseCLI(cfg *config.Config) (bool, error) {
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--enterprise-sarif":
			out := "zed-enterprise.sarif"
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
				out = args[i+1]
			}
			path, err := enterprise.WriteSARIF(cfg.WorkDir, out)
			if err != nil { return true, err }
			fmt.Println(path)
			return true, nil
		case "--enterprise-policy-gate":
			report, _, err := enterprise.PolicyGate(cfg.WorkDir)
			if err != nil { return true, err }
			fmt.Print(report.Summary())
			if strings.Contains(report.Summary(), "POLICY_GATE_BLOCK") { return true, fmt.Errorf("enterprise policy gate blocked release") }
			return true, nil
		case "--demo":
			fmt.Println("BITTU CHAUHAN Enterprise Demo Mode")
			fmt.Println("This is deterministic local demo output; no API key or network is used.")
			fmt.Println("\nPLAN MODE")
			for _, m := range enterprise.BuildMilestones("demo enterprise upgrade") {
				fmt.Printf("%d. %s [%s] review=%v\n", m.ID, m.Name, m.Risk, m.NeedsReview)
			}
			report, entries, _ := enterprise.RiskHeatmap(cfg.WorkDir)
			fmt.Println("\n" + report.Name)
			for i, e := range entries { if i < 8 { fmt.Printf("%-32s %-8s score=%d\n", e.Path, e.Risk, e.Score) } }
			fmt.Println("\nTry inside TUI: /plan <goal>, /review, /dashboard, /heatmap, /auth, /setup, Ctrl+P")
			return true, nil
		}
	}
	return false, nil
}
