## 1. Mission, scope aur success criteria

### 1.1 Mission statement

> Ek aisa terminal agent banao jo **engineer ki jagah nahi, engineer ke haath tez** kare — local-first, auditable, sandboxed, aur itna fast ki `git status` se pehle khul jaye. Enterprise team isko rollout kar sake bina security review me atakne ke.

### 1.2 Target users (3 personas, teeno ka same binary)

| Persona | Kya chahiye | Kya todta hai |
|---|---|---|
| **Individual dev** (80%) | Fast TUI, acche diffs, model choice, cheap | Slow startup, prompts har 10 second, config hell |
| **Platform / DevEx team** (15%) | CI mode, policy, hooks, MCP integrations, stable JSON | Non-deterministic output, breaking CLI flags |
| **Security / compliance** (5%) | Sandbox, audit, SSO, air-gap, no data egress | "We'll add logging later", silent network calls |

### 1.3 Jobs to be done (v1)

1. "Is repo me bug fix karo aur tests chalao" — **long-horizon task**, 20+ turns.
2. "Ye error kya hai?" — **fast question**, < 5 s first token.
3. "PR review karo, sirf high-severity batao" — **headless**, CI me.
4. "Is codebase ka architecture samjhao" — **repo map + grep**, bade repo me bhi.
5. "Ye 5 independent cheezein parallel karo" — **parallel subagents** (§4).
6. "Editor se chalao" — **ACP**, Zed/JetBrains/Neovim me.

### 1.4 In scope (v1) / out of scope

| In scope | Out of scope (explicit non-goals) |
|---|---|
| Terminal TUI + headless exec + JSONL streaming | GUI/desktop app, Electron wrapper |
| 8+ model providers + local models | Training/fine-tuning models |
| 3-OS sandbox (Linux/macOS/Windows) | Cloud-hosted agent runtime (v2) |
| MCP client + MCP server mode + ACP agent | Editor plugins (ACP se free milta hai) |
| Skills, hooks, custom commands, plugins (signed) | Marketplace/billing infra |
| Parallel subagents (DAG + worktrees + merge gates) | Browser automation (v2) |
| Audit, policy, SSO, air-gap mode | SCIM server (control plane ka kaam) |
| Session persistence, replay, resume, undo | Cross-device sync service |

**Rule:** agar koi feature in "out of scope" me se aata hai, wo v1 ko *nahi* block karega — backlog §10 me jaayega.

### 1.5 Constraints

- **Rust only** — koi Node/Python runtime dependency nahi (npm sirf binary shim).
- **Terminal-first** — TUI 80×24 pe bhi usable; mouse optional.
- **Local-first** — bina internet ke start ho, config error pe bhi sane defaults (except provider calls).
- **Single static binary** — < 38 MB stripped; `scp` karke chale.
- **No `unsafe`** outside `sandbox` + `exec` (aur wahan documented invariant).
- **Backward compatibility** — CLI flags, config keys, protocol events: semver + deprecation cycle (2 minor releases).

### 1.6 12 golden rules (architectural invariants — CI-enforced jahan possible)

| # | Rule | Kyun | Kaise enforce |
|---|---|---|---|
| 1 | **Ek writer per thread** (actor model) | data races, torn state impossible | `Session` non-`Sync` handle; clippy lint + review |
| 2 | **Op in, Event out** — UI me business logic nahi | replay, testability, multi-surface | `tui` ka dependency sirf `protocol`+`core` handle |
| 3 | **Har mutation ek event** | crash consistency, audit | state changes only inside `core/agent/*` |
| 4 | **Path confinement ek hi function** | escape bugs ek jagah | `core/util/paths.rs` + `tools/path_guard.rs`, code review ban on other path joins |
| 5 | **Deny-first policy** | allow-all config bhi safe | `policy/engine.rs` order (deny → allow → ask) |
| 6 | **Sandbox fail-closed** | silent unsandboxed = breach | `sandbox/probe.rs` + explicit `--no-sandbox` + audit |
| 7 | **Truncation informative** | model ko agla step pata chale | `tools/result_policy.rs` test: har truncation me hint |
| 8 | **Cache-stable prefix** | 3–10× cost | snapshot test on assembler order |
| 9 | **Tool schema budget** | 40 tools = 6k tokens before prompt | `context/tool_budget.rs` + metric |
| 10 | **Every file has one owner** | review quality | `CODEOWNERS` per crate |
| 11 | **No silent fallback** | enterprise trust | every degrade = warning + audit entry |
| 12 | **Prompt files are code** | behavior regressions | snapshots + evals gate prompt changes |

### 1.7 Success metrics (v1 GA)

| Metric | Target | Kaise measure |
|---|---:|---|
| Cold start | < 60 ms (p95) | hyperfine CI |
| First visible token after Enter (network excl.) | < 120 ms | instrumented |
| Idle RSS | < 120 MB | CI memory test |
| Task success (internal 40-task suite) | > 70% pass@1 | eval harness §8 |
| Tokens per solved task | < 120k median | eval harness (harness quality metric) |
| Tool call error rate | < 4% | telemetry (opt-in) |
| Context compactions per 100 turns | < 15 | ledger metrics |
| Prompt cache hit rate (Anthropic/OpenAI) | > 60% | provider usage |
| Merge conflict rate (parallel mode) | < 10% of merges | orchestrator metrics |
| Parallel speedup (5 independent tasks) | > 1.8× serial time | benchmark job |
| Crash-free sessions | > 99.5% | telemetry |
| Sandbox escape findings (external audit) | 0 critical | pentest |

**Anti-metrics (jo hum *nahi* optimize karte):** lines of code, number of tools, number of providers, benchmark score without token cost. In sab ne agents ko kharab banaya hai.
