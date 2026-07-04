# 🚀 ZED — World's Most Advanced Terminal AI Agent (Go)

> **Vision:** A pure Go, terminal-only AI coding agent — more powerful and beautiful than Cursor & Claude Code.
> Reads files, edits files, runs shell commands, understands entire codebases, and streams responses in a
> gorgeous TUI with a Cursor-style prompt box.
>
> **Constraints (STRICT):**
> - ✅ Go only (no Python, no Node, no web, no IDE plugin)
> - ✅ Terminal-only (TUI, runs in any terminal)
> - ✅ Real, production-grade, complex architecture
> - ✅ File read / file edit / shell exec / codebase understanding
> - ✅ Best-in-class UI + prompt box (better than Cursor/Claude Code)

---

## 📐 High-Level Architecture

```
┌──────────────────────────────────────────────────────────────────────┐
│                          TERMINAL (TUI Layer)                          │
│   Bubble Tea + Lip Gloss + Glamour  →  Prompt box, chat, diff viewer   │
└───────────────────────────────┬──────────────────────────────────────┘
                                │
┌───────────────────────────────▼──────────────────────────────────────┐
│                         AGENT ORCHESTRATOR                             │
│  ReAct loop • Planning • Tool routing • Context building • Memory      │
└───────┬─────────────┬──────────────┬──────────────┬──────────────────┘
        │             │              │              │
   ┌────▼───┐   ┌─────▼────┐   ┌─────▼─────┐   ┌────▼──────┐
   │  LLM   │   │  TOOLS   │   │  CONTEXT  │   │  MEMORY   │
   │ Client │   │ Registry │   │  Engine   │   │  Store    │
   └────────┘   └────┬─────┘   └─────┬─────┘   └───────────┘
                     │               │
         ┌───────────┼───────────┐   │
     ┌───▼──┐  ┌─────▼───┐  ┌────▼─┐ │
     │ File │  │  Shell  │  │ Grep │ │
     │ Tool │  │  Tool   │  │ Tool │ │
     └──────┘  └─────────┘  └──────┘ │
                                     │
                            ┌────────▼─────────┐
                            │  Codebase Index  │
                            │ (AST + Embeddings)│
                            └──────────────────┘
```

---

## 🗂️ Complete File & Directory Structure

```
zed/
├── cmd/
│   └── zed/
│       └── main.go                     # Entry point, CLI flags, bootstraps TUI
│
├── internal/
│   │
│   ├── app/
│   │   ├── app.go                      # App wiring, dependency injection container
│   │   ├── config.go                   # Config loading (yaml/env), defaults
│   │   └── lifecycle.go                # Startup / graceful shutdown, signal handling
│   │
│   ├── tui/                            # ===== TERMINAL UI LAYER =====
│   │   ├── model.go                    # Root Bubble Tea model (state machine)
│   │   ├── update.go                   # Update() — message/event handling
│   │   ├── view.go                     # View() — composes full-screen render
│   │   ├── keys.go                     # Keybindings (vim-style + custom)
│   │   ├── theme.go                    # Color themes, adaptive light/dark
│   │   ├── components/
│   │   │   ├── promptbox.go            # ⭐ Cursor-style multi-line input box
│   │   │   ├── chat.go                 # Streaming chat viewport
│   │   │   ├── message.go              # Single message renderer (markdown)
│   │   │   ├── diffview.go             # Side-by-side / unified diff viewer
│   │   │   ├── filetree.go             # Interactive file tree sidebar
│   │   │   ├── statusbar.go            # Bottom status bar (model, tokens, mode)
│   │   │   ├── spinner.go              # Loading / thinking animations
│   │   │   ├── toast.go                # Notifications / errors
│   │   │   ├── autocomplete.go         # @file / /command autocomplete popup
│   │   │   └── codeblock.go            # Syntax-highlighted code blocks
│   │   ├── syntax/
│   │   │   └── highlighter.go          # Chroma-based syntax highlighting
│   │   └── markdown/
│   │       └── renderer.go             # Glamour markdown -> terminal
│   │
│   ├── agent/                          # ===== AGENT BRAIN =====
│   │   ├── agent.go                    # Core agent, owns the ReAct loop
│   │   ├── loop.go                     # Reason → Act → Observe iteration
│   │   ├── planner.go                  # Task decomposition / multi-step plans
│   │   ├── router.go                   # Decides which tool(s) to call
│   │   ├── prompt.go                   # System prompt builder + templates
│   │   ├── streaming.go                # Token streaming into TUI
│   │   └── state.go                    # Agent run state, cancellation
│   │
│   ├── llm/                            # ===== LLM PROVIDER ABSTRACTION =====
│   │   ├── client.go                   # Provider-agnostic interface
│   │   ├── message.go                  # Chat message / role types
│   │   ├── stream.go                   # SSE streaming reader
│   │   ├── tokenizer.go                # Token counting / budgeting
│   │   ├── retry.go                    # Backoff, rate-limit handling
│   │   └── providers/
│   │       ├── openai.go               # OpenAI-compatible
│   │       ├── anthropic.go            # Claude
│   │       ├── gemini.go               # Google Gemini
│   │       ├── ollama.go               # Local models
│   │       └── openrouter.go           # OpenRouter gateway
│   │
│   ├── tools/                          # ===== AGENT TOOLS =====
│   │   ├── tool.go                     # Tool interface + JSON schema
│   │   ├── registry.go                 # Tool registration & lookup
│   │   ├── executor.go                 # Safe execution + result capture
│   │   ├── permissions.go              # Approval gating for risky actions
│   │   ├── file/
│   │   │   ├── read.go                 # Read file (with line ranges)
│   │   │   ├── write.go                # Create new file
│   │   │   ├── edit.go                 # ⭐ Precise old_str/new_str edit
│   │   │   ├── multiedit.go            # Batched edits across files
│   │   │   ├── delete.go               # Delete file/dir
│   │   │   └── list.go                 # List dir / tree
│   │   ├── shell/
│   │   │   ├── exec.go                 # ⭐ Run shell commands (cross-platform)
│   │   │   ├── session.go              # Persistent shell session (pty)
│   │   ├── search/
│   │   │   ├── grep.go                 # ripgrep-style content search
│   │   │   ├── find.go                 # Glob file search
│   │   │   └── semantic.go             # Embedding-based semantic search
│   │   ├── git/
│   │   │   ├── status.go               # git status / diff
│   │   │   ├── commit.go               # Stage + commit
│   │   │   └── log.go                  # History
│   │   └── web/
│   │       └── fetch.go                # Fetch URL / docs (optional)
│   │
│   ├── context/                        # ===== CODEBASE UNDERSTANDING =====
│   │   ├── engine.go                   # Builds context window per turn
│   │   ├── indexer.go                  # Walks repo, builds file index
│   │   ├── ast/
│   │   │   ├── parser.go               # Tree-sitter parsing
│   │   │   └── symbols.go              # Extract funcs/types/imports
│   │   ├── embeddings/
│   │   │   ├── embedder.go             # Generate embeddings
│   │   │   └── vectorstore.go          # Local vector DB (in-memory/bbolt)
│   │   ├── chunker.go                  # Smart code chunking
│   │   ├── ranker.go                   # Relevance ranking (hybrid BM25+vec)
│   │   └── ignore.go                   # .gitignore / .zedignore respect
│   │
│   ├── memory/                         # ===== MEMORY / PERSISTENCE =====
│   │   ├── store.go                    # Session + long-term memory iface
│   │   ├── conversation.go             # Chat history persistence
│   │   ├── summarizer.go               # Context compaction / summarizing
│   │   └── sqlite.go                   # SQLite-backed store
│   │
│   ├── session/
│   │   ├── manager.go                  # Multiple sessions, switching
│   │   └── snapshot.go                 # Undo/redo of file changes
│   │
│   ├── config/
│   │   ├── config.go                   # Typed config struct
│   │   ├── keys.go                     # API key management (secure)
│   │   └── defaults.go
│   │
│   └── util/
│       ├── logger.go                   # Structured logging (to file)
│       ├── diff.go                     # Diff generation
│       ├── fs.go                       # Filesystem helpers
│       ├── platform.go                 # Windows/Linux/macOS specifics
│       └── errors.go                   # Error types
│
├── pkg/
│   └── ptyx/
│       └── ptyx.go                     # Cross-platform pseudo-terminal wrapper
│
├── configs/
│   ├── zed.example.yaml                # Example user config
│   └── themes/
│       ├── dracula.yaml
│       └── tokyonight.yaml
│
├── scripts/
│   ├── build.sh
│   └── install.sh
│
├── test/
│   ├── integration/
│   └── fixtures/
│
├── .zedignore                          # Files to skip during indexing
├── go.mod
├── go.sum
├── Makefile
├── README.md
└── LICENSE
```

---

## 🧩 Core Go Dependencies

| Purpose | Library |
|---|---|
| TUI framework | `github.com/charmbracelet/bubbletea` |
| Styling / layout | `github.com/charmbracelet/lipgloss` |
| Markdown render | `github.com/charmbracelet/glamour` |
| Prebuilt widgets | `github.com/charmbracelet/bubbles` |
| Syntax highlight | `github.com/alecthomas/chroma/v2` |
| AST parsing | `github.com/smacker/go-tree-sitter` |
| Fast search | `github.com/BurntSushi/ripgrep` bindings / native walk |
| Vector store | `go.etcd.io/bbolt` or in-memory |
| SQLite | `modernc.org/sqlite` (pure Go, no CGO) |
| PTY | `github.com/creack/pty` (+ `github.com/UserExistsError/conpty` for Windows) |
| HTTP/SSE | stdlib `net/http` + custom SSE reader |
| Config | `github.com/spf13/viper` |
| CLI | `github.com/spf13/cobra` |

---

## 🔑 Key Design Decisions

### 1. The ReAct Agent Loop (`internal/agent/loop.go`)
The heart of the agent. Each turn:
1. **Reason** — LLM decides next action based on goal + context + history.
2. **Act** — Call a tool (read/edit/shell/search) via the registry.
3. **Observe** — Feed tool result back into context.
4. Repeat until the task is complete or user cancels.

### 2. Cursor-Style Prompt Box (`internal/tui/components/promptbox.go`)
- Multi-line input with soft-wrap
- `@` triggers file autocomplete, `/` triggers slash-commands
- Inline token counter + model badge
- History (↑/↓), paste-safe, syntax hint preview

### 3. Precise File Editing (`internal/tools/file/edit.go`)
- Uses `old_str` / `new_str` matching (like Claude Code) for surgical edits
- Generates a diff preview shown in the TUI **before** applying
- Every edit is snapshotted for undo (`internal/session/snapshot.go`)

### 4. Safe Shell Execution (`internal/tools/shell/exec.go`)
- Cross-platform (PowerShell on Windows, sh/bash on Unix)
- Streams stdout/stderr live into the chat viewport
- Risky commands gated behind approval (`tools/permissions.go`)
- Persistent session support via PTY (`pkg/ptyx`)

### 5. Codebase Understanding (`internal/context/`)
- On startup, index the repo: walk files → parse AST → extract symbols → embed
- Hybrid retrieval: **BM25 keyword** + **vector semantic** → rank → inject top-K
- Respects `.gitignore` + `.zedignore`
- Incremental re-index on file changes (fsnotify)

### 6. Context Compaction (`internal/memory/summarizer.go`)
- When context nears the token budget, older turns are summarized
- Keeps the agent effective on very long sessions

---

## 🎨 UI/UX Highlights (better than Cursor/Claude Code)

- **Full-screen adaptive TUI** with sidebar file tree + chat + status bar
- **Live streaming** tokens with a "thinking" spinner and tool-call badges
- **Rich diff viewer** — accept/reject edits with a keypress
- **Syntax-highlighted** code blocks (Chroma) in every message
- **Themeable** (Dracula, Tokyo Night, custom yaml themes)
- **Slash commands**: `/clear`, `/model`, `/undo`, `/diff`, `/reindex`, `/help`
- **@-mentions** to pull specific files into context instantly
- **Keyboard-first**, vim-style navigation, mouse optional

---

## 🛠️ Build & Run

```bash
# Build
go build -o zed ./cmd/zed

# Run in any project
./zed

# With a specific model
./zed --model claude-3-5-sonnet --provider anthropic
```

`Makefile`:
```make
build:
	go build -o bin/zed ./cmd/zed
run:
	go run ./cmd/zed
test:
	go test ./...
install:
	go install ./cmd/zed
```

---

## 🧭 Development Roadmap (Phased)

**Phase 1 — Foundation**
- [ ] `cmd/zed/main.go`, config loading, logger
- [ ] LLM client + one provider (Anthropic/OpenAI) with streaming
- [ ] Minimal TUI: prompt box + chat viewport

**Phase 2 — Tools**
- [ ] File read/write/edit tools + diff preview
- [ ] Shell exec tool (cross-platform)
- [ ] Grep/find search tools
- [ ] Tool registry + executor + permissions

**Phase 3 — Agent Brain**
- [ ] ReAct loop + router + planner
- [ ] Streaming tool calls into TUI
- [ ] Undo/redo snapshots

**Phase 4 — Codebase Intelligence**
- [ ] Repo indexer + AST symbols
- [ ] Embeddings + vector store + hybrid ranker
- [ ] Auto context injection

**Phase 5 — Polish**
- [ ] Themes, autocomplete, slash commands
- [ ] Session management + memory compaction
- [ ] Multi-provider support, tests, docs

---

## 🔒 Safety

- All file writes and risky shell commands require explicit approval (configurable).
- Every change is snapshotted → instant `/undo`.
- Sandboxing limits (timeout, output size) on shell execution.
- API keys stored securely, never logged.

---

**ZED** — Pure Go. Terminal-only. Real, production-grade. Built to beat Cursor & Claude Code. 🦀→🐹
