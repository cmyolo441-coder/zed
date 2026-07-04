package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/cmyolo441-coder/zed/internal/agent"
	"github.com/cmyolo441-coder/zed/internal/cache"
	"github.com/cmyolo441-coder/zed/internal/collab"
	"github.com/cmyolo441-coder/zed/internal/config"
	"github.com/cmyolo441-coder/zed/internal/dist"
	"github.com/cmyolo441-coder/zed/internal/index"
	"github.com/cmyolo441-coder/zed/internal/metrics"
	"github.com/cmyolo441-coder/zed/internal/plugin"
	"github.com/cmyolo441-coder/zed/internal/predict"
	"github.com/cmyolo441-coder/zed/internal/session"
	"github.com/cmyolo441-coder/zed/internal/snapshot"
	"github.com/cmyolo441-coder/zed/internal/watcher"
)

// --- messages passed through the Bubble Tea runtime -------------------------

type agentEventMsg agent.Event
type agentChannelMsg struct{ ch <-chan agent.Event }

// Model is the root Bubble Tea model for the ZED terminal UI.
type Model struct {
	cfg   *config.Config
	agent *agent.Agent
	theme Theme

	snapshots *snapshot.Manager
	sessions  *session.Store
	index     *index.Index
	curSess   *session.Session

	// Enterprise subsystems (formerly orphaned, now fully wired).
	cch     *cache.Cache
	dash    *metrics.Dashboard
	eng     *predict.Engine
	pm      *plugin.Manager
	net     *dist.Network
	wtr     *watcher.Watcher
	collab  *collab.Session

	viewport viewport.Model
	input    textarea.Model
	spinner  spinner.Model

	width, height int
	transcript    *strings.Builder // rendered chat history
	streaming     bool             // agent is currently working
	curAssistant  *strings.Builder // in-progress assistant text
	tokens        int              // cumulative output tokens
	inputTokens   int              // cumulative input tokens
	outputTokens  int              // cumulative output tokens (same as tokens but kept separate for clarity)
	totalCost     float64          // cumulative cost in USD
	err           string

	events <-chan agent.Event
	cancel context.CancelFunc

	// Paste detection: tracks recent paste activity so we don't auto-submit
	// when a user pastes multi-line text that contains newlines.
	lastPasteTime time.Time // time of the most recent bracketed-paste event
	lastKeyAt     time.Time // wall-clock time of the previous key event (burst detection)

	uiMode          tuiMode
	layoutMode      string
	activeGoal      string
	currentPlanStep int
	planSteps       []planStep
	approvals       []approvalItem
	nextApprovalID  int
	toolEvents      []toolActivity
	toolStarted     map[string]time.Time

	// /login multi-step flow state. When loginStep != "", the next Enter
	// submits the typed text as the value for the current login step instead
	// of running a prompt. This lets us capture API keys / base URLs / model
	// names interactively — like Claude Code's /login.
	loginStep  string // "" = inactive; otherwise one of loginStep* constants
	loginDraft loginDraft
}

// loginStep* are the stages of the /login flow.
const (
	loginStepProvider  = "provider" // choose: nvidia / opencode / cloudflare / custom
	loginStepAPIKey    = "apikey"
	loginStepAccountID = "accountid" // Cloudflare only
	loginStepBaseURL   = "baseurl"
	loginStepModel     = "model"
)

// loginDraft accumulates the values collected during a /login flow.
type loginDraft struct {
	provider  string // nvidia | opencode | cloudflare | custom
	apiKey    string
	baseURL   string
	model     string
	choice    int    // numeric provider choice (1-4)
	accountID string // Cloudflare account ID
}

// NewModel wires up the TUI with an agent and all enterprise subsystems.
func NewModel(cfg *config.Config, ag *agent.Agent, snapshots *snapshot.Manager, sessions *session.Store, idx *index.Index, cch *cache.Cache, dash *metrics.Dashboard, eng *predict.Engine, pm *plugin.Manager, net *dist.Network, wtr *watcher.Watcher, collabSess *collab.Session) Model {
	th := DraculaTheme()

	ta := textarea.New()
	ta.Placeholder = "Type a command for BITTU CHAUHAN…"
	ta.Prompt = "› "
	ta.CharLimit = 0
	ta.SetHeight(3)
	ta.ShowLineNumbers = false
	ta.MaxHeight = 30
	ta.Focus()

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = th.Prompt

	vp := viewport.New(0, 0)

	m := Model{
		cfg:          cfg,
		agent:        ag,
		theme:        th,
		snapshots:    snapshots,
		sessions:     sessions,
		index:        idx,
		curSess:      session.NewSession(cfg.Provider, cfg.Model, cfg.WorkDir),
		cch:          cch,
		dash:         dash,
		eng:          eng,
		pm:           pm,
		net:          net,
		wtr:          wtr,
		collab:       collabSess,
		viewport:     vp,
		input:        ta,
		spinner:      sp,
		transcript:   &strings.Builder{},
		curAssistant: &strings.Builder{},
		layoutMode:   "focus",
		toolStarted:  map[string]time.Time{},
	}
	m.transcript.WriteString(m.banner())
	return m
}

func (m Model) banner() string {
	t := m.theme
	info := config.LookupModel(m.cfg.Model)
	mark := t.Glow.Render("        ◢◤") + "\n" +
		t.Glow.Render("     ◢██◤   ◢◤") + "\n" +
		t.Glow.Render("   ◢████◤ ◢██") + "\n" +
		t.Glow.Render("     ◥██◣◢██◤") + "\n" +
		t.Glow.Render("        ◥◤")
	title := t.Title.Render("BITTU CHAUHAN")
	sub := t.Dim.Render(fmt.Sprintf("Enterprise Agent CLI · %s · %s · max %dk tokens", m.cfg.Provider, m.cfg.Model, info.MaxTokens/1000))
	return "\n" + lipgloss.NewStyle().Align(lipgloss.Center).Width(max(m.width-2, 60)).Render(mark+"\n"+title+"\n"+sub) + "\n\n"
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, m.spinner.Tick)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		inputHeight := 6
		chromeHeight := 5
		m.viewport.Width = max(msg.Width-4, 20)
		m.viewport.Height = max(msg.Height-inputHeight-chromeHeight, 5)
		m.input.SetWidth(max(msg.Width-8, 20))
		m.refreshViewport()

	case tea.KeyMsg:
		// --- paste-safe input handling ------------------------------------
		// A paste can reach us two ways:
		//   1. Bracketed paste: one KeyMsg with msg.Paste=true carrying the
		//      whole clipboard (Windows Terminal, iTerm, most modern emulators).
		//   2. Character-by-character burst: legacy ConHost / some SSH setups
		//      replay the clipboard as individual key events with no Paste flag.
		//      Here the FIRST embedded newline arrives as a lone KeyEnter while
		//      the textarea still holds only line 1 — the old code submitted
		//      early and dropped the rest of the paste.
		//
		// We defend against both. `now` and the gap since the previous key let
		// us recognise burst #2: keys arriving faster than a human can type
		// (<~20ms apart) are part of a paste, so their Enter must insert a
		// newline rather than submit.
		now := time.Now()
		gap := now.Sub(m.lastKeyAt)
		m.lastKeyAt = now
		// Keys within this window of each other are considered a paste burst.
		const pasteBurstGap = 20 * time.Millisecond
		inBurst := !m.lastPasteTime.IsZero() && now.Sub(m.lastPasteTime) < 300*time.Millisecond

		if msg.Paste {
			// Bracketed paste: the textarea absorbs the whole payload below.
			// Record the time so a trailing lone KeyEnter is also suppressed.
			m.lastPasteTime = now
			break
		}
		// Rapid consecutive keys ⇒ paste burst. Remember it so the Enter case
		// (and a brief window afterwards) treats newlines as insertion.
		if gap > 0 && gap < pasteBurstGap {
			m.lastPasteTime = now
			inBurst = true
		}

		switch msg.Type {
		case tea.KeyCtrlC:
			if m.streaming {
				// Cancel the running agent turn.
				if m.cancel != nil {
					m.cancel()
					m.cancel = nil
				}
				m.streaming = false
				m.appendLine(m.theme.Dim.Render("\n✋ Cancelled by user (Esc)") + "\n")
				return m, nil
			}
			return m, tea.Quit
		case tea.KeyEsc:
			// Cancel an in-progress /login flow.
			if m.loginStep != "" {
				m.loginStep = ""
				m.loginDraft = loginDraft{}
				m.input.Reset()
				m.appendLine(m.theme.Dim.Render("✋ /login cancelled.") + "\n")
				return m, nil
			}
			if m.streaming {
				// Stop the running agent turn — like Claude Code / Cursor.
				if m.cancel != nil {
					m.cancel()
					m.cancel = nil
				}
				m.streaming = false
				m.appendLine(m.theme.Dim.Render("\n✋ Stopped by user (Esc)") + "\n")
				return m, nil
			}
		case tea.KeyCtrlP:
			m.openCommandPalette()
			return m, nil
		case tea.KeyEnter:
			if !m.streaming {
				// If this Enter is part of a paste burst (or arrived just
				// after one), it's an embedded newline, not a submit — forward
				// it to the textarea so the full paste is preserved intact.
				// This is the crucial fix for the "long paste auto-sends and
				// truncates" bug on Windows ConHost.
				if inBurst {
					break
				}
				// Alt+Enter always inserts a newline (explicit multi-line).
				if msg.Alt {
					break
				}
				// Otherwise this is a real, human Enter → submit.
				return m.submit()
			}
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case agentChannelMsg:
		m.events = msg.ch
		cmds = append(cmds, m.waitForEvent())

	case agentEventMsg:
		cmds = append(cmds, m.handleAgentEvent(agent.Event(msg)))
	}

	// Always forward messages to textarea (even during paste when not streaming)
	// so it can absorb pasted text character-by-character.
	if !m.streaming {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		cmds = append(cmds, cmd)
		// Auto-resize textarea height based on content (min 3, max 30 lines).
		lines := strings.Split(m.input.Value(), "\n")
		needed := len(lines)
		if needed < 3 {
			needed = 3
		} else if needed > 30 {
			needed = 30
		}
		m.input.SetHeight(needed)
	}
	var vpCmd tea.Cmd
	m.viewport, vpCmd = m.viewport.Update(msg)
	cmds = append(cmds, vpCmd)

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	t := m.theme
	width := max(m.width, 60)

	// Full-page login form — takes over the entire screen.
	if m.loginStep != "" {
		return m.renderLoginForm()
	}

	header := m.topBar(width)
	inputBox := t.InputBox.Width(max(width-4, 40)).Render(m.input.View())
	hints := m.footerHints(width)
	status := m.statusLine()

	if m.layoutMode == "minimal" {
		return lipgloss.JoinVertical(lipgloss.Left, m.viewport.View(), inputBox, status)
	}

	if m.layoutMode == "split" {
		leftW := max((width*62)/100, 40)
		rightW := max(width-leftW-5, 28)
		left := t.Panel.Width(leftW).Height(max(m.viewport.Height, 5)).Render(m.viewport.View())
		right := t.Panel.Width(rightW).Height(max(m.viewport.Height, 5)).Render(m.sidePanel())
		body := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
		return lipgloss.JoinVertical(lipgloss.Left, header, body, inputBox, hints, status)
	}

	panel := t.Panel.Width(max(width-4, 40)).Height(max(m.viewport.Height, 5)).Render(m.viewport.View())
	return lipgloss.JoinVertical(lipgloss.Left, header, panel, inputBox, hints, status)
}

func (m Model) topBar(width int) string {
	t := m.theme
	traffic := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5F57")).Render("●") + " " +
		lipgloss.NewStyle().Foreground(lipgloss.Color("#FFBD2E")).Render("●") + " " +
		lipgloss.NewStyle().Foreground(lipgloss.Color("#28C840")).Render("●")
	brand := t.Title.Render("▣ BITTU CHAUHAN")
	mode := t.Dim.Render("enterprise agent")
	right := t.Dim.Render("+" + fmt.Sprintf("%d", len(m.agent.History())))
	gap := max(width-lipgloss.Width(traffic)-lipgloss.Width(brand)-lipgloss.Width(mode)-lipgloss.Width(right)-8, 1)
	line := traffic + "  " + brand + "  " + mode + strings.Repeat(" ", gap) + right + " "
	return t.Header.Width(width).Render(line)
}

func (m Model) footerHints(width int) string {
	t := m.theme
	run := t.Title.Render("enter") + t.Dim.Render(" run")
	swap := t.Title.Render("ctrl+p") + t.Dim.Render(" prompt")
	stop := t.Title.Render("esc") + t.Dim.Render(" stop")
	quit := t.Title.Render("ctrl+c") + t.Dim.Render(" quit")
	line := lipgloss.JoinHorizontal(lipgloss.Top, run, "   ", swap, "   ", stop, "   ", quit)
	return t.Hint.Width(width).Align(lipgloss.Center).Render(line)
}

func (m Model) statusLine() string {
	t := m.theme
	model := config.EffectiveModel(m.cfg, m.cfg.Effort)
	modelInfo := config.LookupModel(model)
	left := fmt.Sprintf("  %s  •  %s  ", model, m.cfg.Effort)
	state := "idle"
	if m.streaming {
		state = m.spinner.View() + " thinking"
	}
	totalTok := m.inputTokens + m.outputTokens
	costStr := formatCost(m.totalCost)
	maxTokStr := fmt.Sprintf("%dk", modelInfo.MaxTokens/1000)
	pct := 0.0
	if modelInfo.MaxTokens > 0 {
		pct = float64(totalTok) / float64(modelInfo.MaxTokens) * 100
	}
	bar := budgetBar(pct, 8)
	right := fmt.Sprintf(" %s  %.1f%%  tok:%d  max:%s  $%s  %s ", bar, pct, totalTok, maxTokStr, costStr, state)
	gap := max(m.width-lipgloss.Width(left)-lipgloss.Width(right), 1)
	return t.Status.Width(max(m.width, 60)).Render(left + strings.Repeat(" ", gap) + right)
}

// budgetBar returns a 10-char progress bar for the given percentage.
func budgetBar(pct float64, width int) string {
	if pct > 100 {
		pct = 100
	}
	filled := int(pct / 100 * float64(width))
	if filled > width {
		filled = width
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	return bar
}

// formatCost formats a dollar amount to a human-readable string.
func formatCost(c float64) string {
	if c == 0 {
		return "0.00"
	}
	if c < 0.01 {
		return fmt.Sprintf("%.6f", c)
	}
	return fmt.Sprintf("%.4f", c)
}

// --- interaction ------------------------------------------------------------

func (m Model) submit() (tea.Model, tea.Cmd) {
	prompt := strings.TrimSpace(m.input.Value())
	if prompt == "" {
		return m, nil
	}

	// /login multi-step capture: when a login step is active, the typed text
	// is captured as the value for that step (not run as a prompt).
	if m.loginStep != "" {
		return m.captureLoginStep(prompt)
	}

	// slash commands
	if strings.HasPrefix(prompt, "/") {
		return m.handleSlash(prompt)
	}

	m.input.Reset()
	return m.runPrompt(prompt)
}

// runPrompt echoes the user's prompt and kicks off an agent turn.
func (m Model) runPrompt(prompt string) (tea.Model, tea.Cmd) {
	m.appendLine(m.theme.User.Render("› You") + "\n" + prompt + "\n")
	m.streaming = true
	m.curAssistant.Reset()

	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel

	ch := make(chan agent.Event, 64)
	go func() {
		m.agent.Run(ctx, prompt, func(e agent.Event) { ch <- e })
		close(ch)
	}()

	return m, func() tea.Msg { return agentChannelMsg{ch: ch} }
}

// setEffort switches the active effort level and reports the result.
func (m Model) setEffort(name string) (tea.Model, tea.Cmd) {
	prof, ok := config.LookupEffort(name)
	if !ok {
		m.appendLine(m.theme.ToolErr.Render("Unknown effort level: "+name) +
			m.theme.Dim.Render("  (try /effort to list)") + "\n")
		return m, nil
	}
	m.cfg.Effort = prof.Name
	m.agent.SetEffort(prof)
	// Persist so effort survives restart.
	_ = m.cfg.Save()
	mult := ""
	if prof.Multiplier > 1 {
		mult = fmt.Sprintf(" (≈%d×)", prof.Multiplier)
	}
	goalNote := ""
	if prof.GoalMode {
		goalNote = "\n" + m.theme.Dim.Render("   ⚡ Goal-mode autonomy is ACTIVE — use /goal <task> for end-to-end autonomous execution.")
	}
	m.appendLine(m.theme.Tool.Render("⚡ Effort: "+prof.Label+mult) + "\n" +
		m.theme.Dim.Render("   "+prof.Description) + goalNote + "\n")
	return m, nil
}

// handleSlash processes built-in slash commands.
func (m Model) handleSlash(prompt string) (tea.Model, tea.Cmd) {
	fields := strings.Fields(prompt)
	cmd := fields[0]
	arg := ""
	if len(fields) > 1 {
		arg = fields[1]
	}
	// rest is everything after the command, preserving spaces (for /goal).
	rest := strings.TrimSpace(strings.TrimPrefix(prompt, cmd))

	switch cmd {

	case "/effort":
		m.input.Reset()
		if arg == "" {
			var b strings.Builder
			b.WriteString(m.theme.Prompt.Render("Effort levels:") + "\n")
			for _, e := range config.Efforts() {
				marker := "  "
				if e.Name == m.cfg.Effort {
					marker = m.theme.Tool.Render("• ")
				}
				goalTag := ""
				if e.GoalMode {
					goalTag = " ⚡goal"
				}
				fmt.Fprintf(&b, "%s%-14s %s%s\n", marker, e.Name, m.theme.Dim.Render(e.Description), m.theme.Tool.Render(goalTag))
			}
			b.WriteString(m.theme.Dim.Render("Switch with: /effort <name>   (or /ultraeffort, /ultramax, /ultracombomax, /goal, /normal)") + "\n")
			m.appendLine(b.String())
			return m, nil
		}
		return m.setEffort(arg)

	case "/normal":
		m.input.Reset()
		return m.setEffort(config.EffortNormal)

	case "/ultraeffort":
		m.input.Reset()
		return m.setEffort(config.EffortUltra)

	case "/ultramax":
		m.input.Reset()
		return m.setEffort(config.EffortUltraMax)

	case "/ultracombomax":
		m.input.Reset()
		prof, _ := config.LookupEffort(config.EffortUltraComboMax)
		m.cfg.Effort = prof.Name
		m.agent.SetEffort(prof)
		_ = m.cfg.Save()
		mult := ""
		if prof.Multiplier > 1 {
			mult = fmt.Sprintf(" (≈%d×)", prof.Multiplier)
		}
		goalNote := ""
		if prof.GoalMode {
			goalNote = "\n" + m.theme.Dim.Render("   ⚡ Goal-mode autonomy is ACTIVE — use /goal <task> for end-to-end autonomous execution.")
		}
		m.appendLine(m.theme.Tool.Render("⚡ Effort: "+prof.Label+mult) + "\n" +
			m.theme.Dim.Render("   "+prof.Description) + goalNote + "\n")
		return m, nil

	case "/goal":
		m.input.Reset()
		if rest == "" {
			m.appendLine(m.theme.Dim.Render("Usage: /goal <describe what you want built end-to-end>") + "\n")
			return m, nil
		}
		// Switch to autonomous goal mode (which is ultracombomax + goal autonomy),
		// then run the goal as an agent turn.
		prof := config.EffortProfile(config.EffortGoal)
		m.cfg.Effort = prof.Name
		m.agent.SetEffort(prof)
		_ = m.cfg.Save()
		m.appendLine(m.theme.Tool.Render("◎ Goal mode engaged ("+prof.Label+" ≈"+fmt.Sprintf("%d", prof.Multiplier)+"×) — Ultra Combo Max + autonomous goal execution.") + "\n" +
			m.theme.Dim.Render("   Researching → planning → building → testing → debugging → verifying — fully autonomous.") + "\n" +
			m.theme.Dim.Render("   Swarm: "+fmt.Sprintf("%d", prof.SwarmSize)+" agents · Auto-debug retries: "+fmt.Sprintf("%d", prof.MaxDebugRetries)+" · Context: "+fmt.Sprintf("%.1f×", prof.ContextBudgetMultiplier)) + "\n")
		return m.runPrompt("[GOAL] " + rest)


	case "/dream":
		m.input.Reset()
		prof := config.EffortProfile(config.EffortDream)
		m.cfg.Effort = prof.Name
		m.agent.SetEffort(prof)
		_ = m.cfg.Save()
		m.appendLine(m.theme.Tool.Render("☁ Dream Mode engaged ("+prof.Label+" ≈"+fmt.Sprintf("%d", prof.Multiplier)+"×) — Agent OS Pro Max controller active.") + "\n" +
			m.theme.Dim.Render("   Controls all enterprise + Agent OS + Pro Max + mission + memory + artifacts + verification planes.") + "\n")
		if rest == "" {
			return m, nil
		}
		return m.runPrompt("[DREAM] " + rest)

	case "/cyber":
		m.input.Reset()
		m.agent.SetCyberMode(true)
		m.appendLine(m.theme.Tool.Render("🛡️  Cybersecurity mode ACTIVATED.") + "\n" +
			m.theme.Dim.Render("   BITTU CHAUHAN is now a security-focused agent: penetration tester + defensive engineer.") + "\n" +
			m.theme.Dim.Render("   OWASP Top 10 scanning, STRIDE threat modeling, secure coding, vulnerability assessment.") + "\n" +
			m.theme.Dim.Render("   Use /normal-mode to return to regular coding mode.") + "\n")
		return m, nil

	case "/normal-mode":
		m.input.Reset()
		m.agent.SetCyberMode(false)
		m.appendLine(m.theme.Tool.Render("💻 Cybersecurity mode deactivated. Back to regular coding mode.") + "\n")
		return m, nil

	case "/clear":
		m.agent.Reset()
		m.transcript.Reset()
		m.transcript.WriteString(m.banner())
		m.input.Reset()
		m.refreshViewport()
		return m, nil

	case "/stop":
		m.input.Reset()
		if m.streaming && m.cancel != nil {
			m.cancel()
			m.cancel = nil
			m.streaming = false
			m.appendLine(m.theme.Dim.Render("✋ Stopped by user.") + "\n")
		} else {
			m.appendLine(m.theme.Dim.Render("Nothing is running.") + "\n")
		}
		return m, nil

	case "/quit", "/exit":
		return m, tea.Quit

	case "/models":
		m.input.Reset()
		var b strings.Builder
		b.WriteString(m.theme.Prompt.Render("Available models:") + "\n")
		for _, name := range config.AvailableModels {
			marker := "  "
			if name == m.cfg.Model {
				marker = m.theme.Tool.Render("• ")
			}
			info := config.LookupModel(name)
			b.WriteString(fmt.Sprintf("%s%-28s max:%-7d $%.4f/1M\n", marker, name, info.MaxTokens, info.PricePer1M))
		}
		b.WriteString(m.theme.Dim.Render("Switch with: /model <name>") + "\n")
		m.appendLine(b.String())
		return m, nil

	case "/model":
		m.input.Reset()
		if arg == "" {
			m.appendLine(m.theme.Dim.Render("Current model: "+m.cfg.Model) + "\n")
			return m, nil
		}
		config.ApplyModel(m.cfg, arg)
		// Persist model choice across restarts.
		_ = m.cfg.Save()
		note := m.theme.Tool.Render("Model set to: " + arg)
		if _, ok := config.KnownModels[arg]; !ok {
			note = m.theme.Dim.Render("Model set to: " + arg + " (not in known list)")
		} else {
			note += m.theme.Dim.Render("\n   → endpoint: "+m.cfg.BaseURL)
		}
		m.appendLine(note + "\n")
		return m, nil

	case "/undo":
		m.input.Reset()
		if m.snapshots == nil || !m.snapshots.CanUndo() {
			m.appendLine(m.theme.Dim.Render("Nothing to undo.") + "\n")
			return m, nil
		}
		desc, err := m.snapshots.Undo()
		if err != nil {
			m.appendLine(m.theme.ToolErr.Render("Undo failed: "+err.Error()) + "\n")
		} else {
			m.appendLine(m.theme.Tool.Render("↺ "+desc) + "\n")
		}
		return m, nil

	case "/redo":
		m.input.Reset()
		if m.snapshots == nil || !m.snapshots.CanRedo() {
			m.appendLine(m.theme.Dim.Render("Nothing to redo.") + "\n")
			return m, nil
		}
		desc, err := m.snapshots.Redo()
		if err != nil {
			m.appendLine(m.theme.ToolErr.Render("Redo failed: "+err.Error()) + "\n")
		} else {
			m.appendLine(m.theme.Tool.Render("↻ "+desc) + "\n")
		}
		return m, nil

	case "/history":
		m.input.Reset()
		if m.snapshots == nil || m.snapshots.Depth() == 0 {
			m.appendLine(m.theme.Dim.Render("No file changes yet.") + "\n")
			return m, nil
		}
		var b strings.Builder
		b.WriteString(m.theme.Prompt.Render(fmt.Sprintf("Change history (%d):", m.snapshots.Depth())) + "\n")
		for i, h := range m.snapshots.History() {
			fmt.Fprintf(&b, "  %d. %s\n", i+1, h)
		}
		m.appendLine(b.String())
		return m, nil

	case "/save":
		m.input.Reset()
		if m.sessions == nil {
			m.appendLine(m.theme.ToolErr.Render("Session store unavailable.") + "\n")
			return m, nil
		}
		m.curSess.Messages = m.agent.History()
		m.curSess.Tokens = m.tokens
		m.curSess.Model = m.cfg.Model
		if err := m.sessions.Save(m.curSess); err != nil {
			m.appendLine(m.theme.ToolErr.Render("Save failed: "+err.Error()) + "\n")
		} else {
			m.appendLine(m.theme.Tool.Render("Saved session: "+m.curSess.ID) + "\n")
		}
		return m, nil

	case "/sessions":
		m.input.Reset()
		if m.sessions == nil {
			m.appendLine(m.theme.ToolErr.Render("Session store unavailable.") + "\n")
			return m, nil
		}
		metas, err := m.sessions.List()
		if err != nil || len(metas) == 0 {
			m.appendLine(m.theme.Dim.Render("No saved sessions.") + "\n")
			return m, nil
		}
		var b strings.Builder
		b.WriteString(m.theme.Prompt.Render("Saved sessions:") + "\n")
		for _, meta := range metas {
			fmt.Fprintf(&b, "  %s  %s  (%d msgs)\n", meta.ID, meta.Title, meta.Messages)
		}
		b.WriteString(m.theme.Dim.Render("Resume with: /resume <id>") + "\n")
		m.appendLine(b.String())
		return m, nil

	case "/resume":
		m.input.Reset()
		if m.sessions == nil || arg == "" {
			m.appendLine(m.theme.Dim.Render("Usage: /resume <session-id>  (see /sessions)") + "\n")
			return m, nil
		}
		sess, err := m.sessions.Load(arg)
		if err != nil {
			m.appendLine(m.theme.ToolErr.Render("Load failed: "+err.Error()) + "\n")
			return m, nil
		}
		m.agent.LoadHistory(sess.Messages)
		m.curSess = sess
		m.tokens = sess.Tokens
		m.transcript.Reset()
		m.transcript.WriteString(m.banner())
		m.appendLine(m.theme.Tool.Render(fmt.Sprintf("Resumed %q (%d messages)", sess.Title, len(sess.Messages))) + "\n")
		return m, nil

	case "/index":
		m.input.Reset()
		if m.index == nil {
			m.appendLine(m.theme.ToolErr.Render("Index unavailable.") + "\n")
			return m, nil
		}
		s := m.index.Stats(time.Now())
		m.appendLine(m.theme.Prompt.Render("Codebase index:") + "\n" +
			fmt.Sprintf("  files:   %d\n  symbols: %d\n  lines:   %d\n", s.Files, s.Symbols, s.Lines))
		return m, nil

	case "/tokens":
		m.input.Reset()
		model := config.EffectiveModel(m.cfg, m.cfg.Effort)
		info := config.LookupModel(model)
		total := m.inputTokens + m.outputTokens
		b := m.theme.Prompt.Render("Token Usage:") + "\n" +
			fmt.Sprintf("  Input:      %d\n", m.inputTokens) +
			fmt.Sprintf("  Output:     %d\n", m.outputTokens) +
			fmt.Sprintf("  Total:      %d\n", total) +
			fmt.Sprintf("  Max Output: %d (%s)\n", info.MaxTokens, model) +
			fmt.Sprintf("  Cost:       $%s\n", formatCost(m.totalCost))
		m.appendLine(m.theme.Dim.Render(b) + "\n")
		return m, nil

	case "/cache":
		m.input.Reset()
		if m.cch == nil {
			m.appendLine(m.theme.Dim.Render("Cache not available.") + "\n")
			return m, nil
		}
		s := m.cch.Stats()
		m.appendLine(m.theme.Prompt.Render("LLM Response Cache:") + "\n" +
			fmt.Sprintf("  Entries: %d\n", s.Size) +
			fmt.Sprintf("  Hits:    %d\n", s.Hits) +
			fmt.Sprintf("  Hit rate: %.1f%%\n", s.HitRate))
		return m, nil


	case "/collab":
		m.input.Reset()
		if m.collab == nil {
			m.appendLine(m.theme.Dim.Render("Collaboration not available.") + "\n")
			return m, nil
		}
		m.appendLine(m.theme.Dim.Render(m.collab.Status()) + "\n")
		return m, nil

	case "/dist":
		m.input.Reset()
		if m.net == nil {
			m.appendLine(m.theme.Dim.Render("Distributed network not available.") + "\n")
			return m, nil
		}
		m.appendLine(m.theme.Dim.Render(m.net.Status()) + "\n")
		return m, nil

	case "/metrics":
		m.input.Reset()
		if m.dash == nil {
			m.appendLine(m.theme.Dim.Render("Metrics dashboard not available.") + "\n")
			return m, nil
		}
		m.appendLine(m.theme.Dim.Render(m.dash.Render()) + "\n")
		return m, nil

	case "/watcher":
		m.input.Reset()
		if m.wtr == nil {
			m.appendLine(m.theme.Dim.Render("File watcher not available.") + "\n")
			return m, nil
		}
		m.appendLine(m.theme.Dim.Render(m.wtr.Summary()) + "\n")
		return m, nil

	case "/plugin":
		m.input.Reset()
		if m.pm == nil {
			m.appendLine(m.theme.Dim.Render("Plugin system not available.") + "\n")
			return m, nil
		}
		if arg == "" {
			plugins := m.pm.List()
			if len(plugins) == 0 {
				m.appendLine(m.theme.Dim.Render("No plugins registered.") + "\n")
			} else {
				var b strings.Builder
				b.WriteString(m.theme.Prompt.Render("Registered plugins:") + "\n")
				for _, name := range plugins {
					fmt.Fprintf(&b, "  • %s\n", name)
				}
				m.appendLine(b.String())
			}
			return m, nil
		}
		m.appendLine(m.theme.Dim.Render("Plugin management: use /plugin to list, register plugins via the plugin system API.") + "\n")
		return m, nil



	case "/mission":
		m.input.Reset()
		m.appendLine(m.renderMissionControl(rest) + "\n")
		return m, nil

	case "/plan":
		m.input.Reset()
		goal := rest
		if goal == "" { goal = "enterprise engineering goal" }
		m.uiMode = modePlan
		m.appendLine(m.renderPlanMode(goal) + "\n")
		return m, nil

	case "/review":
		m.input.Reset()
		m.uiMode = modeReview
		m.appendLine(m.renderReviewDiff() + "\n")
		return m, nil

	case "/palette":
		m.input.Reset()
		m.uiMode = modePalette
		m.openCommandPalette()
		return m, nil

	case "/approvals":
		m.input.Reset()
		m.appendLine(m.renderApprovals() + "\n")
		return m, nil

	case "/approve":
		m.input.Reset()
		id := parseID(rest)
		approver := strings.TrimSpace(strings.TrimPrefix(rest, arg))
		if approver == "" { approver = "user" }
		m.appendLine(m.theme.Tool.Render(m.approve(id, approver)) + "\n")
		return m, nil

	case "/deny":
		m.input.Reset()
		id := parseID(rest)
		reason := strings.TrimSpace(strings.TrimPrefix(rest, arg))
		if reason == "" { reason = "denied by user" }
		m.appendLine(m.theme.ToolErr.Render(m.deny(id, reason)) + "\n")
		return m, nil

	case "/timeline":
		m.input.Reset()
		m.appendLine(m.renderTimeline() + "\n")
		return m, nil

	case "/dashboard":
		m.input.Reset()
		m.uiMode = modeDashboard
		m.appendLine(m.renderDashboard() + "\n")
		return m, nil

	case "/heatmap":
		m.input.Reset()
		m.appendLine(m.renderRiskHeatmap() + "\n")
		return m, nil

	case "/replay":
		m.input.Reset()
		m.appendLine(m.renderReplay(arg) + "\n")
		return m, nil

	case "/login":
		m.input.Reset()
		m.loginStep = loginStepProvider
		m.loginDraft = loginDraft{}
		return m, nil

	case "/auth":
		m.input.Reset()
		m.appendLine(m.renderAuthUI() + "\n")
		return m, nil

	case "/setup":
		m.input.Reset()
		m.appendLine(m.renderSetupWizard() + "\n")
		return m, nil

	case "/theme":
		m.input.Reset()
		m.appendLine(m.theme.Tool.Render(m.setTheme(arg)) + "\n")
		return m, nil

	case "/layout":
		m.input.Reset()
		m.appendLine(m.theme.Tool.Render(m.setLayout(arg)) + "\n")
		return m, nil


	case "/vision-ui":
		m.input.Reset()
		m.appendLine(m.runAgentOSCommand("vision-ui", rest) + "\n")
		return m, nil

	case "/desktop":
		m.input.Reset()
		m.appendLine(m.runAgentOSCommand("desktop", rest) + "\n")
		return m, nil

	case "/app":
		m.input.Reset()
		m.appendLine(m.runAgentOSCommand("app", rest) + "\n")
		return m, nil

	case "/theme-generate":
		m.input.Reset()
		m.appendLine(m.runAgentOSCommand("theme-generate", rest) + "\n")
		return m, nil

	case "/persona":
		m.input.Reset()
		m.appendLine(m.runAgentOSCommand("persona", arg) + "\n")
		return m, nil

	case "/tutorial":
		m.input.Reset()
		m.appendLine(m.runAgentOSCommand("tutorial", rest) + "\n")
		return m, nil

	case "/prompt-os":
		m.input.Reset()
		m.appendLine(m.runAgentOSCommand("prompt-os", rest) + "\n")
		return m, nil

	case "/skills":
		m.input.Reset()
		m.appendLine(m.runAgentOSCommand("skills", rest) + "\n")
		return m, nil

	case "/knowledge":
		m.input.Reset()
		m.appendLine(m.runAgentOSCommand("knowledge", rest) + "\n")
		return m, nil

	case "/readme-design":
		m.input.Reset()
		m.appendLine(m.runAgentOSCommand("readme-design", rest) + "\n")
		return m, nil

	case "/self-upgrade":
		m.input.Reset()
		m.appendLine(m.runAgentOSCommand("self-upgrade", rest) + "\n")
		return m, nil

	case "/code-movie":
		m.input.Reset()
		m.appendLine(m.runAgentOSCommand("code-movie", rest) + "\n")
		return m, nil

	case "/animation":
		m.input.Reset()
		m.appendLine(m.runAgentOSCommand("animation", arg) + "\n")
		return m, nil

	case "/voice":
		m.input.Reset()
		m.appendLine(m.runAgentOSCommand("voice", rest) + "\n")
		return m, nil


	case "/map":
		m.input.Reset()
		m.appendLine(m.runProMaxCommand("map", rest) + "\n")
		return m, nil

	case "/memory-graph":
		m.input.Reset()
		m.appendLine(m.runProMaxCommand("memory-graph", rest) + "\n")
		return m, nil

	case "/workspace":
		m.input.Reset()
		m.appendLine(m.runProMaxCommand("workspace", rest) + "\n")
		return m, nil

	case "/artifacts":
		m.input.Reset()
		m.appendLine(m.runProMaxCommand("artifacts", rest) + "\n")
		return m, nil

	case "/jobs":
		m.input.Reset()
		m.appendLine(m.runProMaxCommand("jobs", rest) + "\n")
		return m, nil

	case "/hypervisor":
		m.input.Reset()
		m.appendLine(m.runProMaxCommand("hypervisor", rest) + "\n")
		return m, nil

	case "/project-pro":
		m.input.Reset()
		m.appendLine(m.runProMaxCommand("project-pro", rest) + "\n")
		return m, nil

	case "/preview":
		m.input.Reset()
		m.appendLine(m.runProMaxCommand("preview", rest) + "\n")
		return m, nil

	case "/web-ui":
		m.input.Reset()
		m.appendLine(m.runProMaxCommand("web-ui", rest) + "\n")
		return m, nil

	case "/plugin-build":
		m.input.Reset()
		m.appendLine(m.runProMaxCommand("plugin-build", rest) + "\n")
		return m, nil

	case "/self-tests":
		m.input.Reset()
		m.appendLine(m.runProMaxCommand("self-tests", rest) + "\n")
		return m, nil

	case "/bench":
		m.input.Reset()
		m.appendLine(m.runProMaxCommand("bench", rest) + "\n")
		return m, nil

	case "/kernel-plan":
		m.input.Reset()
		m.appendLine(m.runProMaxCommand("kernel-plan", rest) + "\n")
		return m, nil

	case "/help":
		m.input.Reset()
		help := m.theme.Prompt.Render("Commands:") + "\n" +
			"  /effort [name]   show / set effort level\n" +
			"  /ultraeffort     deep reasoning + plan + self-verify\n" +
			"  /ultramax        40× — research + exhaustive verify\n" +
			"  /ultracombomax   100× — enterprise autonomous engineering + goal mode\n" +
			"  /goal <task>     autonomous end-to-end: research→build→test→fix\n" +
			"  /dream [task]    1000× Agent OS Pro Max controller mode\n" +
			"  /normal          back to fast, balanced effort\n" +
			"  /cyber           🛡️  activate cybersecurity mode (pentest + defense)\n" +
			"  /normal-mode     deactivate cybersecurity mode\n" +
			"  /model <name>    switch model\n" +
			"  /models          list available models\n" +
			"  /login           set API key / provider interactively (nvidia · opencode · cloudflare · custom)\n" +
			"  /undo /redo      revert or reapply file changes\n" +
			"  /history         list file changes\n" +
			"  /save            save this session\n" +
			"  /sessions        list saved sessions\n" +
			"  /resume <id>     resume a saved session\n" +
			"  /index           codebase index stats\n" +
			"  /cache           LLM response cache stats\n" +
			"  /collab          collaboration session status\n" +
			"  /dist            distributed agent network status\n" +
			"  /metrics         code metrics dashboard\n" +
			"  /watcher         file watcher status\n" +
			"  /plugin [name]   list or load plugins\n" +
			"  /mission         mission control goals\n" +
			"  /plan <goal>     open enterprise plan mode UI\n" +
			"  /review          review current git diff\n" +
			"  /palette         command palette (Ctrl+P)\n" +
			"  /approvals       pending approvals\n" +
			"  /dashboard       autonomous work dashboard\n" +
			"  /heatmap         enterprise risk heatmap\n" +
			"  /timeline        agent memory timeline\n" +
			"  /auth /setup     auth and setup screens\n" +
			"  /vision-ui <img>  screenshot-to-UI builder\n" +
			"  /desktop         Agent OS desktop panel\n" +
			"  /app <idea>      autonomous app builder\n" +
			"  /prompt-os       initialize prompt operating system\n" +
			"  /skills          local skill marketplace\n" +
			"  /persona <role>  persona engine\n" +
			"  /knowledge       build local knowledge pack\n" +
			"  /tutorial        generate project tutorial\n" +
			"  /readme-design   generate designed README\n" +
			"  /self-upgrade    self-upgrading agent kernel\n" +
			"  /map             codebase map UI\n" +
			"  /memory-graph    memory graph query/build\n" +
			"  /workspace       neural workspace state\n" +
			"  /artifacts       artifact browser\n" +
			"  /jobs            long-running job manager\n" +
			"  /hypervisor      agent orchestration kernel\n" +
			"  /project-pro     autonomous project generator pro\n" +
			"  /web-ui          export web dashboard\n" +
			"  /plugin-build    generate tool plugin skeleton\n" +
			"  /theme /layout   UI theme and layout controls\n" +
			"  /tokens          token usage\n" +
			"  /clear           reset the conversation\n" +
			"  /stop            stop the current agent turn\n" +
			"  /help            show this help\n" +
			"  /quit            exit BITTU CHAUHAN\n" +
			"\n  Shortcuts:  Esc = stop  ·  Ctrl+C = quit  ·  Ctrl+V = paste"
		m.appendLine(help)
		return m, nil

	default:
		m.input.Reset()
		m.appendLine(m.theme.ToolErr.Render("Unknown command: "+cmd) + m.theme.Dim.Render("  (try /help)") + "\n")
		return m, nil
	}
}

func (m Model) waitForEvent() tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-m.events
		if !ok {
			return agentEventMsg{Kind: agent.EventDone}
		}
		return agentEventMsg(ev)
	}
}

func (m *Model) handleAgentEvent(e agent.Event) tea.Cmd {
	t := m.theme
	switch e.Kind {
	case agent.EventText:
		if m.curAssistant.Len() == 0 {
			m.appendLine(t.Prompt.Render("✦ BITTU CHAUHAN") + "\n")
		}
		m.curAssistant.WriteString(e.Text)
		m.appendRaw(e.Text)

	case agent.EventToolCall:
		m.curAssistant.Reset()
		if m.toolStarted == nil { m.toolStarted = map[string]time.Time{} }
		m.toolStarted[e.ToolName] = time.Now()
		m.toolEvents = append(m.toolEvents, toolActivity{Name:e.ToolName, Target:summarizeArgs(e.ToolArgs), Status:"running", Started:time.Now()})
		m.appendLine("\n" + t.Tool.Render(fmt.Sprintf("⚙ %s", e.ToolName)) +
			t.Dim.Render(" "+summarizeArgs(e.ToolArgs)) + "\n")

	case agent.EventToolDone:
		style := t.Dim
		status := "done"
		if e.IsError {
			style = t.ToolErr
			status = "error"
		}
		if len(m.toolEvents) > 0 {
			idx := len(m.toolEvents)-1
			m.toolEvents[idx].Status = status
			m.toolEvents[idx].Duration = time.Since(m.toolEvents[idx].Started)
		}
		m.appendLine(style.Render("  ↳ "+firstLines(e.Result, 6)) + "\n")

	case agent.EventDone:
		if e.Usage != nil {
			m.inputTokens += e.Usage.InputTokens
			m.outputTokens += e.Usage.OutputTokens
			m.tokens = m.inputTokens + m.outputTokens
			// Calculate cost from output tokens using model's price info.
			model := config.EffectiveModel(m.cfg, m.cfg.Effort)
			info := config.LookupModel(model)
			m.totalCost += float64(e.Usage.OutputTokens) * info.PricePer1M / 1_000_000
		}
		m.streaming = false
		m.appendRaw("\n")
		if m.cancel != nil {
			m.cancel()
			m.cancel = nil
		}
		return nil

	case agent.EventNotice:
		// Transient status (rate-limit backoff etc.) — show dimmed, keep streaming.
		m.appendLine(t.Dim.Render("⏳ "+e.Text) + "\n")
		return m.waitForEvent()

	case agent.EventError:
		m.streaming = false
		m.cancel = nil
		m.appendLine("\n" + t.ToolErr.Render("✖ "+e.Text) + "\n")
		return nil
	}
	return m.waitForEvent()
}

// --- rendering helpers ------------------------------------------------------

func (m *Model) appendLine(s string) {
	m.transcript.WriteString(s)
	m.refreshViewport()
}

func (m *Model) appendRaw(s string) {
	m.transcript.WriteString(s)
	m.refreshViewport()
}

func (m *Model) refreshViewport() {
	wrapped := lipgloss.NewStyle().Width(max(m.width-1, 20)).Render(m.transcript.String())
	m.viewport.SetContent(wrapped)
	m.viewport.GotoBottom()
}

func summarizeArgs(args string) string {
	s := strings.ReplaceAll(args, "\n", " ")
	if len(s) > 80 {
		s = s[:80] + "…"
	}
	return s
}

func firstLines(s string, n int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) <= n {
		return strings.Join(lines, "\n  ")
	}
	return strings.Join(lines[:n], "\n  ") + fmt.Sprintf("\n  … (%d more lines)", len(lines)-n)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
