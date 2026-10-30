## 2. Architecture principles — 7 planes, event-sourced core

### 2.1 Planes (logical), crates (physical)

Har serious agent ka same logical shape hota hai. Difference sirf isme hai ki boundaries kitni saaf hain.

```
┌──────────────────────────────────────────────────────────────────────────────┐
│ P1  INTERFACE PLANE                                                          │
│   crates/tui (ratatui)   crates/cli (clap)   crates/server (JSON-RPC + ACP)  │
│   → teeno sirf Op bhejte hain aur Event padhte hain. Koi business logic nahi. │
├──────────────────────────────────────────────────────────────────────────────┤
│ P2  ORCHESTRATION PLANE                        crates/core                  │
│   ThreadManager → Session → Turn loop → tool router → subagents → budgets    │
├──────────────────────────────────────────────────────────────────────────────┤
│ P3  COGNITION PLANE            crates/context   crates/index   crates/model  │
│   prompt assembly · compaction ladder · repo map · token ledger · providers  │
├──────────────────────────────────────────────────────────────────────────────┤
│ P4  CAPABILITY PLANE           crates/tools   crates/ext   crates/mcp        │
│   read/write/patch/grep/glob/shell/lsp/web/plan/task · skills · hooks · MCP  │
├──────────────────────────────────────────────────────────────────────────────┤
│ P5  EXECUTION PLANE            crates/exec    crates/sandbox                 │
│   spawn · PTY · process groups · Seatbelt/Landlock/bwrap/AppContainer        │
├──────────────────────────────────────────────────────────────────────────────┤
│ P6  MEMORY PLANE               crates/state   crates/git                     │
│   JSONL rollout (truth) · SQLite (query) · snapshots · worktrees             │
├──────────────────────────────────────────────────────────────────────────────┤
│ P7  CONTROL PLANE              crates/policy  crates/auth  crates/telemetry  │
│   deny-first rules · approvals · redaction · audit chain · OIDC · OTel       │
└──────────────────────────────────────────────────────────────────────────────┘
        cross-cutting: crates/protocol (types) · crates/testkit (tests)
```

**Dependency rule (CI-enforced, `cargo-deny` + custom lint):**

```
protocol  ← (sab)
core      ← tui, cli, server
core      → model, context, tools, state, policy, exec
tools     → exec, sandbox, policy, index, git, mcp
tui       → protocol, core (read-only handle)      ❌ tui ko tools ka pata nahi
server    → protocol, core                          ❌ server ko tui ka pata nahi
```

Agar `tui` ko `tools` ka type chahiye, matlab aapne UI ko domain logic de diya hai — wo render *hints* ke through aana chahiye (`tools/src/render.rs`), type ke through nahi.

### 2.2 Kyun 19 crates, ek nahi

| Reason | Concrete effect |
|---|---|
| **Incremental compile** | `tui` change karne pe `model` compile nahi hota. 76k LOC monolith = 90–150s incremental; split = 8–20s. |
| **Testability** | `context` ke unit tests ko HTTP, TUI, ya SQLite nahi chahiye. |
| **Embedding** | `core` ek library hai — koi SDK/CI tool usse embed kar sakta hai (Codex ne `codex-core` exactly isliye publish kiya). |
| **Feature gating** | `server` aur `git` optional features ho sakte hain → minimal build 18 MB. |
| **Blast radius** | Sandbox bug sandbox crate me fail hota hai, poore binary me nahi. |

Trade-off: 19 crates = 19 `Cargo.toml` + workspace dependency management + version pinning. Ye cost 3–4 din ka hai, one-time.

### 2.3 Event-sourced turn (ye blueprint ka sabse important decision)

Purane agents (aur aapka Go version bhi) mutable state rakhte hain: `session.messages.push(...)`, phir UI ko callback. Iska matlab: resume brittle, tests flaky, replay impossible, aur multi-surface (TUI + IDE ek saath) me state divergence.

**Yahan design ye hai:** *op in, event out, state = fold(events)*.

```
       ┌──────────┐   Op::Submit{prompt}      ┌────────────────────────┐
 TUI ──┤          ├──────────────────────────►│  Thread actor          │
 CLI ──┤ Op queue │                           │  (single writer)       │
 IDE ──┤ (mpsc)   │◄──────────────────────────┤  state: Session        │
       └──────────┘   Event::*  (broadcast)   └───────────┬────────────┘
                                                          │
                    ┌─────────────────────────────────────┼──────────────────┐
                    ▼                     ▼               ▼                  ▼
              TUI projection      JSONL rollout     SQLite index       OTel span
              (pure render)       (source of truth)  (fast query)      (opt-in)
```

Rules:
1. **Sirf ek writer** per thread. Koi `Arc<Mutex<Session>>` nahi.
2. Har mutation ek `Event` emit karti hai (`TurnStarted`, `TextDelta`, `ToolCallRequested`, `ApprovalDecided`, `TokenUsage`, `CompactBoundary`, `TurnCompleted`, `TurnFailed`).
3. **UI ek pure function hai** `(AppState, Event) -> AppState` (`tui/src/update.rs`, 420 LOC, 100% unit-testable, no terminal needed).
4. **Replay** = JSONL padho, events re-emit karo, same UI state milega. Bug reports debuggable ho jaate hain — user `--replay session.jsonl` bhejta hai.
5. **Resume** = JSONL replay + provider context rebuild. Do alag cheezein, do alag code paths, isliye `state/src/resume.rs` sirf 260 LOC me fit hota hai.

Cost: events ka schema discipline (har event versioned, `#[serde(tag = "type")]`), aur thoda extra allocation. Benefit: crash-consistency, replay, test speed, multi-surface sync.

### 2.4 Turn state machine (explicit, not implied)

```
        ┌─────────┐  Submit   ┌───────────┐  context ok  ┌──────────┐
        │  Idle   ├──────────►│ Assembling├─────────────►│Streaming │
        └────┬────┘           └─────┬─────┘              └────┬─────┘
             ▲                     │ overflow                  │ tool_use
             │                     ▼                           ▼
             │              ┌────────────┐             ┌──────────────┐
             │              │ Compacting │             │  Approving   │
             │              └─────┬──────┘             └───┬──────┬───┘
             │                    │ retry                  │allow │deny
             │                    └────────────────────────┘      │
             │                                                    ▼
        ┌────┴─────┐   done   ┌──────────┐  result   ┌──────────────┐
        │ Completed│◄─────────┤ Verifying│◄──────────┤  Executing   │
        └──────────┘          └────┬─────┘           └──────┬───────┘
                                   │ fail                  │ cancel
                                   ▼                       ▼
                            ┌────────────┐          ┌────────────┐
                            │  Retrying  │          │ Cancelled  │
                            └────────────┘          └────────────┘
```

**Loop ka shape (illustrative pseudocode — implementation nahi, design intent):**

```
async fn run_turn(turn: &mut Turn, ops: &mut OpStream) -> Result<TurnOutcome> {
    loop {
        // 1. budget + context: explicit, cancelable, no hidden state
        if turn.budget.exhausted() { return Ok(TurnOutcome::BudgetStop) }
        let req = context.assemble(&turn.session)?;          // cache-aware sections
        turn.set(State::Streaming);

        // 2. model call: streaming, cancellable, retryable
        let mut stream = gateway.stream(req, &turn.cancel).await?;
        let mut calls: Vec<ToolCall> = vec![];
        while let Some(chunk) = stream.next().await? {
            match chunk {
                Chunk::Text(t)      => emit(Event::TextDelta(t)),
                Chunk::ToolDelta(d) => calls.assemble(d)?,       // partial JSON buffer
                Chunk::Usage(u)     => turn.ledger.reconcile(u),
                Chunk::Done(r)      => { turn.note_finish(r); break }
            }
        }

        // 3. no tools -> the turn is over (model decides, not us)
        if calls.is_empty() { return Ok(TurnOutcome::Completed) }

        // 4. tools: policy -> approval -> execute -> shape
        turn.set(State::Approving);
        for call in calls {
            let decision = policy.evaluate(&call, &turn.approvals).await?;
            if decision == Deny { emit(Event::ToolDenied(call)); continue }
            turn.set(State::Executing);
            let raw = tools.execute(call, &turn.cancel).await?;
            emit(Event::ToolCompleted(tools.shape(raw)));
        }
        // 5. loop again — new evidence enters the window
    }
}
```

Do baatein is shape me intentional hain: **(a)** loop model-driven hai (hardcoded workflow nahi), **(b)** har `await` point cancelable hai — isliye Ctrl-C kabhi bhi safe hai, aur `Drop` guards child processes kill karte hain.

Har state ka ek **owner** aur ek **timeout** hai. `agent/state.rs` (340 LOC) me transitions table-tested hain: illegal transition = compile-time error (enum + exhaustive match), runtime assertion nahi.

### 2.5 Concurrency model (Rust-specific)

| Concern | Mechanism | Kahan |
|---|---|---|
| Op queue | bounded `tokio::mpsc` (backpressure = UI freeze nahi, drop nahi) | `core/agent/mod.rs` |
| Event fanout | `tokio::sync::broadcast` + per-subscriber lag handling | `core/events.rs` |
| Cancellation | `CancellationToken` tree: turn → tool → subagent → child process | `core/agent/cancel.rs` |
| Parallel tools | `JoinSet` with per-call timeout + ordered result merge | `core/tools/executor.rs` |
| Blocking FS | `spawn_blocking` pool; kabhi async runtime pe `std::fs` nahi | `tools/fs/*` |
| Subagents | alag actor + alag thread id + summary-only return | `core/subagent/mod.rs` |
| TUI | apna OS thread (crossterm event reader) → channel → tokio | `tui/src/event.rs` |
| DB | `sqlx` pool, WAL, single-writer transaction | `state/db.rs` |

**Anti-patterns jo maine real agents me dekhe:** `Arc<RwLock<Conversation>>` (deadlock + torn reads), `tokio::spawn` bina handle (cancel impossible, orphan processes), `select!` me biased branches (starvation), aur process kill ke bina `Child::wait` drop (zombies — isliye `exec/src/reaper.rs`).

### 2.6 Ek prompt ka end-to-end journey (latency budget ke saath)

User: *"login bug fix karo"*

| # | Step | Component | Budget |
|---|---|---|---|
| 1 | Keystroke → editor buffer → submit | `tui/input/editor.rs` | < 1 ms |
| 2 | `Op::Submit` → thread actor | `core/agent/mod.rs` | < 1 ms |
| 3 | Env context refresh (git branch, cwd tree) | `context/sections.rs` | 5–20 ms |
| 4 | Memory load (AGENTS.md hierarchy, cached) | `context/memory/hierarchy.rs` | 0–10 ms |
| 5 | Repo map slice (cached tags, PageRank) | `index/repo_map/*` | 10–40 ms (cold 0.5–3 s) |
| 6 | Token ledger + pressure check | `context/ledger.rs` | < 2 ms |
| 7 | Prompt assembly with cache boundaries | `context/assembler.rs` | < 5 ms |
| 8 | Provider request (first byte) | `model/providers/*` | 300–1200 ms (network) |
| 9 | SSE decode → events → UI render | `model/stream.rs` → `tui/render` | < 1 ms/event |
| 10 | Tool call complete → policy → approval | `policy/engine.rs` | 1–5 ms (+ human time) |
| 11 | Sandboxed exec | `sandbox/*` + `exec/*` | command-dependent |
| 12 | Result shaping → next turn | `core/tools/result_policy.rs` | < 5 ms |

**Local overhead target: 60–80 ms** (steps 1–7 + 9 + 12). Baaki network aur tool time hai. Isi liye "Rust kyun" ka jawab "speed" nahi, "**predictable local latency + memory ceiling**" hai.

### 2.7 Rust: kahan jeetta hai, kahan maarta hai

**Jeetta hai:**
- Cold start 30–60 ms (Node agents 300–900 ms, warm cache ke baad bhi).
- Idle RSS 40–120 MB (Electron/Node 200–400 MB).
- Single static binary — `scp` karke chalao, npm postinstall drama nahi.
- Process/PTY/signal control first-class (`nix`, `windows-sys`, `portable-pty`).
- Fearless concurrency: actor model bina data-race anxiety ke.
- Type-safe protocol: `Event` enum me naya variant add karo → TUI, server, replay, tests sab compile-error se batate hain kahan handle karna hai. **Ye 300+ file codebase ko maintainable rakhta hai.**

**Maarta hai (aur iska mitigation blueprint me hai):**
| Problem | Mitigation |
|---|---|
| Compile times | 19 crates + `sccache` + `lld`/`mold`; CI me `cargo-nextest`; dev profile `opt-level=1` |
| Async cancellation bugs | CancellationToken discipline + `Drop` guards + `loom` tests for critical paths |
| Trait-object bloat / binary size | Generics at boundaries only; `strip`, LTO, `panic=abort` for release |
| Verbose error plumbing | `thiserror` in libs, `anyhow` only at bin edges; error taxonomy crate |
| TUI panic → broken terminal | Panic hook + RAII `TerminalGuard` + signal handlers (must-have, day 1) |
| Cross-platform PTY | `portable-pty` + ConPTY tests on Windows CI (real VM, not cross-compile) |
| Ecosystem churn | Pin exact versions, vendor `Cargo.lock`, `cargo-deny` advisories |
| Team ramp-up | ADRs + "one file = one responsibility" rule + golden transcripts for behavior |

---

### 2.8 Config model aur on-disk layout

Config 6 layers me resolve hoti hai (precedence: neeche wala upar wale ko override karta hai):

```
defaults (compiled) < global < project < project-local < env vars < CLI flags < managed (admin-locked)
```

Managed layer **sirf admin** set kar sakta hai aur developer usse override nahi kar sakta — enterprise ka core control yahi hai.

```
~/.config/<agent>/            config.toml · AGENTS.md · mcp.toml · hooks/ · skills/
~/.local/share/<agent>/        sessions/*.jsonl · index.sqlite · blobs/ · logs/
~/.cache/<agent>/              repo-map tags · completion cache · schema cache
<repo>/.agent/                 config.toml · rules/*.md · commands/*.md · agents/*.md
<repo>/AGENTS.md               project memory (git-tracked, review-able)
```

Minimal `config.toml` (jo 90% users chahiye):

```toml
[model]
provider   = "anthropic"          # anthropic | openai | gemini | ollama | custom
name       = "claude-sonnet-latest"
max_tokens = 16000
temperature = 0.2

[permissions]
mode = "ask"                      # read-only | ask | auto-edit | plan | bypass
allow = ["shell(git status)", "shell(git diff)", "read(**)"]
deny  = ["shell(rm -rf *)", "read(**/.env*)", "net(*)"]

[sandbox]
enabled = true
writable = ["."]
network  = "deny"                 # deny | localhost | allowlist
allowlist = ["api.anthropic.com", "github.com"]

[context]
compact_at = 0.72                 # window ka fraction
repo_map_tokens = 2048
keep_recent_files = 5

[telemetry]
enabled = false                   # default OFF
```

**Design rule:** har config key ka ek schema entry, ek default, ek doc line aur ek validation error message hona chahiye. `config/validate.rs` isi ko enforce karta hai — "unknown key" silently ignore karna enterprise me unacceptable hai (typo = security hole).

---
