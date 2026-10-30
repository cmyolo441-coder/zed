## 2. Product surface — user ko kya dikhta hai

### 2.1 Commands (top-level)

| Command | Purpose | Mode |
|---|---|---|
| `agent` | interactive TUI (default) | TTY |
| `agent "prompt"` | one-shot with TUI streaming, phir exit | TTY |
| `agent exec -f task.md --json` | headless, CI-friendly, JSON/JSONL out | non-TTY |
| `agent exec --resume <id>` | session continue | non-TTY |
| `agent review --base main --fail-on high` | diff review + exit code | CI |
| `agent mcp serve` | agent ko MCP server banao (doosre agents ke liye) | stdio |
| `agent acp` | ACP agent mode (editors spawn karte hain) | stdio |
| `agent auth login\|logout\|status` | credentials | — |
| `agent config get\|set\|list\|edit\|explain` | config | — |
| `agent session list\|show\|export\|rm` | sessions | — |
| `agent doctor [--json]` | environment diagnostics | — |
| `agent audit verify\|export` | tamper check + SIEM export | — |
| `agent update [--check]` | signed self-update | — |
| `agent completions <shell>` | shell completions | — |

### 2.2 Interaction modes (permission modes)

`--mode` (ya `/mode` se live switch):

| Mode | Reads | Edits | Shell | Network | Kiske liye |
|---|---|---|---|---|---|
| `plan` | ✅ | ❌ | ❌ (ro commands) | ❌ | design/analysis, untrusted repos |
| `read-only` | ✅ | ❌ | ❌ | ❌ | review, audit |
| `ask` *(default)* | ✅ | ✅ prompt | ✅ prompt | prompt | daily dev |
| `auto-edit` | ✅ | ✅ auto | ✅ prompt | prompt | power users |
| `bypass` | ✅ | ✅ | ✅ | ✅ | containers/CI (banner + audit) |

**UX detail:** approval prompt me hamesha — exact command/argv, cwd, risk badge (read/mutate/destructive/network), sandbox status, aur "always allow this pattern" ka expanded preview. `Esc` = deny. Default focus = **Deny** (Enter pe deny, `y` pe allow) — safe default.

### 2.3 Slash commands (TUI)

| Command | Kya karta hai |
|---|---|
| `/model [name]` | model picker (cost + context ke saath) |
| `/mode [name]` | permission mode switch (audit) |
| `/context` | token ledger breakdown (system/tools/memory/history) |
| `/compact [focus]` | manual compaction with focus hint |
| `/cost` | session spend, per-turn, cache hit rate |
| `/permissions` | rules list, add/remove, scope preview |
| `/undo` | last turn ke file changes revert |
| `/resume [id]` | session list + resume |
| `/board` | parallel workers board (§4) |
| `/workers [kill <id>]` | worker management |
| `/diff` | pending diff review (accept/reject hunks) |
| `/help [topic]` | help + keybindings + concepts |
| `/quit` | graceful exit (children cleanup) |

### 2.4 Keybindings (defaults)

| Key | Action |
|---|---|
| `Enter` | submit (in composer) |
| `Alt+Enter` / `Ctrl+J` | newline |
| `Ctrl+C` | interrupt turn (twice = quit) |
| `Esc` | deny/cancel modal, clear input |
| `Ctrl+R` | history search |
| `Ctrl+O` | expand/collapse tool card |
| `Ctrl+B` | board toggle |
| `Ctrl+D` | debug overlay (last request, ledger, timings) |
| `Tab` | completion (`@file`, `/cmd`) |
| `PgUp/PgDn`, `Ctrl+U/D` | transcript scroll |
| `?` | help overlay |

Vim mode optional (`keymap/vim.rs`), off by default.

### 2.5 On-disk layout

```
~/.config/agent/        config.toml · AGENTS.md · mcp.toml · hooks/ · skills/ · commands/
~/.local/share/agent/   sessions/*.jsonl · index.sqlite · blobs/ · logs/ · worktrees/
~/.cache/agent/         repo-map tags · schema cache · completions
<repo>/.agent/          config.toml · rules/*.md · commands/*.md · agents/*.md
<repo>/AGENTS.md        project memory (git-tracked)
```

### 2.6 Config surface (v1 keys)

| Section | Keys | Notes |
|---|---|---|
| `model` | provider, name, max_tokens, temperature, reasoning_effort, fallback[] | `provider/model` shortcut |
| `permissions` | mode, allow[], deny[], ask[], remember_ttl | pattern grammar §7.4 |
| `sandbox` | enabled, writable[], network, allowlist[], resource{cpu,mem,pids} | fail-closed |
| `context` | compact_at, repo_map_tokens, keep_recent_files, memory_cap | fractions 0–1 |
| `tools` | enabled[], disabled[], shell_session, lsp{servers[]} | per-tool toggles |
| `agents` | max_workers, task_budget, worker_model, merge_policy, isolation | §4.13 |
| `hooks` | pre_tool_use[], post_tool_use[], ... | paths to executables |
| `telemetry` | enabled, otlp_endpoint, sample_rate | default **false** |
| `ui` | theme, keymap, a11y, mouse, images | per-terminal |
| `enterprise` | managed_policy_path, audit_sink, identity_mode | admin-locked |

`agent config explain <key>` batata hai kaunsi file se value aayi (6-layer precedence).

### 2.7 Exit codes (stable contract for CI)

| Code | Meaning |
|---:|---|
| 0 | success |
| 1 | task/agent error |
| 2 | CLI usage error |
| 3 | policy denied (nothing executed) |
| 4 | sandbox unavailable (fail-closed) |
| 5 | provider/auth error |
| 42 | invalid input (bad prompt file, empty stdin) |
| 53 | turn/budget limit exceeded |
| 130 | interrupted (SIGINT) |

### 2.8 Core user flows

**Flow A — normal fix (ask mode):** prompt → plan card → tool calls with inline diffs → approval per mutating call → verification (tests) → summary with file list + cost. Esc at any point cancels safely.

**Flow B — headless CI:** `agent exec -f task.md --json` → JSON stream (`turn.delta`, `tool.call`, `result`) → exit code → SARIF/JUnit conversion via `--output-format`. No prompts; policy deny = exit 3.

**Flow C — parallel:** `/board` ya `agent exec --parallel` → planner proposes DAG → user sees task cards → workers run in worktrees → merge queue → verification gate → single consolidated diff for review.

**Flow D — editor (ACP):** editor spawns `agent acp` → prompt → streaming updates → permission request editor UI me → diff editor me render → session resume on reopen.
