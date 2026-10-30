## 3. Crate map & runtime model — 19 crates

### 3.1 Crate map

| Crate | Files | LOC | Role | Depends on |
|---|---:|---:|---|---|
| `protocol` | 14 | 2,950 | Wire contracts: ops, events, IR messages, tool/permission/usage types | — |
| `testkit` | 4 | 840 | Fixtures, mock provider, golden helpers | protocol |
| `model` | 26 | 7,500 | Provider adapters, streaming, token/cost, cache, failover | protocol |
| `core` | 43 | 10,200 | Turn loop, sessions, approvals, budgets, prompt assembly, subagent (single) | protocol, model, policy, tools(late) |
| `policy` | 12 | 3,040 | Rules, deny-first engine, redaction, audit chain | protocol |
| `context` | 22 | 5,450 | Assembly, compaction ladder, memory, ledger, pressure | protocol, model |
| `index` | 8 | 2,150 | tree-sitter repo map, PageRank, grep, file tree | context |
| `tools` | 39 | 10,520 | File/patch/shell/LSP/web/plan/task tools | core, exec, policy, index, git, sandbox |
| `exec` | 10 | 2,420 | Spawn, PTY, env, timeouts, output, reaper | protocol |
| `sandbox` | 12 | 3,420 | bwrap/Landlock/seccomp, Seatbelt, AppContainer, egress | exec |
| `state` | 17 | 3,520 | JSONL rollout, SQLite, snapshots, resume, GC | protocol |
| `tui` | 36 | 10,100 | ratatui renderer, input, approvals, board, a11y | protocol, core (handle only) |
| `cli` | 14 | 2,230 | Subcommands, headless, doctor, update, completions | core, tui, server |
| `server` | 14 | 3,400 | JSON-RPC app-server, WebSocket, ACP agent | protocol, core |
| `orchestrator` | 9 | 2,490 | Parallel DAG, scheduler, leases, merge + verify gates | core, state, git |
| `ext` | 10 | 2,300 | Skills, hooks, plugins, custom commands, profiles | core, tools |
| `auth` | 6 | 1,310 | Keyring, OAuth PKCE, device code, CI identity | model |
| `git` | 6 | 1,400 | Repo discovery, diff, commit, worktrees, checkpoints | protocol |
| `telemetry` | 6 | 1,320 | Tracing, OTLP, metrics, privacy scrubber, support bundle | core |
| **Total** | **308** | **76,560** | | |

### 3.2 Dependency rule (forbidden edges)

```
protocol ──► (everyone)                       [types only, zero logic]
core     ──► model, policy, context, tools    [one-way]
tui      ──► protocol + core::handle          ✗ tools/model/state types nahi
server   ──► protocol + core                  ✗ tui, cli nahi
orchestrator ─► core, state, git              ✗ tui, server nahi
tools    ──► exec, sandbox, policy, index     ✗ tui, server nahi
```

`cargo` workspace me enforce: `tui` ke `Cargo.toml` me `tools` na ho, aur CI me ek custom check (`cargo metadata` graph) forbidden edges pe fail kare.

### 3.3 Op → Event flow (single source of truth)

```
  user/editor/CI
        │  Op { Submit, Steer, Interrupt, Approve, Compact, SetMode, SpawnWorkers }
        ▼
 ┌─────────────────┐    fold     ┌──────────────┐
 │  Agent actor    │ ──────────► │ Session state│
 │  (one per thread)│            └──────┬───────┘
 └────────┬────────┘                   │
          │ Event (broadcast + persist)│
    ┌─────┴──────┬──────────┬──────────┴─────┬──────────────┐
    ▼            ▼          ▼                ▼              ▼
  TUI         app-server  JSONL rollout   SQLite        telemetry
 (projection)  (projection) (truth)      (index)        (opt-in)
```

**Rule of thumb:** agar aap UI me kuch *calculate* kar rahe ho jo core se aana chahiye, galat ho. UI sirf format karta hai.

### 3.4 Threading / concurrency model

| Component | Thread/runtime | Rule |
|---|---|---|
| TUI event reader | dedicated OS thread (crossterm) | sends `UiEvent` via channel; never blocks |
| Render loop | TUI thread (16 ms tick) | pure draw, no I/O |
| Agent actor | tokio task per thread | owns `Session`, single writer |
| Tool executor | `JoinSet` (max 8 parallel) | read-only parallel, mutating serial |
| Shell/PTY | blocking pool + `spawn_blocking` | no `std::fs` on async runtime |
| Subagent/worker | separate actor + own task | cancel tree, own budget |
| SQLite | dedicated writer task | single writer, WAL readers |
| Telemetry exporter | background task | drop on backpressure |

### 3.5 Failure semantics (kya hota hai jab kuch toote)

| Failure | Behavior | User ko dikhta hai |
|---|---|---|
| Model 429/5xx | retry with backoff, then failover | "retrying (2/3)…" |
| Stream drops mid-tool-call | turn replay once, else error | "stream interrupted, retrying" |
| Context overflow | compact ladder then retry | "compacted (12k → 4k)" |
| Tool timeout | kill process group, return partial output | "timed out after 120s; showing output so far" |
| Sandbox denied | fail-closed, audit | "sandbox unavailable: enable bwrap or pass --no-sandbox" |
| Approval denied | tool not executed, model told why | "denied by policy: shell(rm -rf)" |
| Worker crash (parallel) | requeue once, then quarantine | board: red card + reason |
| Merge conflict | rebase retry once, then escalate | "conflict in src/a.rs; worker retried" |
| Provider auth expired | refresh, else interactive hint | "session expired — run `agent auth login`" |
| Disk full | stop writes, keep session in memory | "disk full: session not persisted" |
