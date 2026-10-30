## 6. Effort, team aur timeline — honest math

### 6.1 LOC se effort (teen scenarios)

Base: **76,560 LOC** hand-written Rust (tests/docs alag). Industry-observed net production throughput for senior systems engineers with AI assistance (2026): **40–70 LOC/day** (reviewed, tested, merged). Hard subsystems (sandbox, TUI, patch engine, protocol) isse slow — 20–35 LOC/day.

| Scenario | Team | Effective LOC/day | Duration | Reality check |
|---|---|---:|---:|---|
| **A. Recommended** | 6 engineers (4 core + 1 TUI + 1 platform/CI) | 6 × 50 = 300 | **~310 working days ≈ 14.5 months** (buffer ke saath) | P0 pehle 3.5 mahine |
| B. Lean | 3 engineers | 150 | ~510 days ≈ **22–24 months** | Scope cut to P0+selected P1 (~185 files) warna burnout |
| C. Aggressive | 9 engineers | 450 | ~170 days ≈ **8 months** | Coordination cost; 19 crates me ownership clash risk |
| D. Solo + AI | 1 senior | 45 | ~1,700 days ≈ **6+ years** | Sirf Tier A/B realistic |

**Cross-check:** Codex ~549k LOC / 5,706 files, OpenAI team + 1.5 years. Ratio-wise 76.5k LOC ≈ 14% of Codex — 6 engineers × ~1 saal ka sensible output (Codex 1.5 saal me 549k LOC, bade team ke saath).

### 6.2 Phase plan (calendar, Scenario A)

| Phase | Weeks | Files added | LOC | Exit criteria |
|---|---:|---:|---:|---|
| **P0 — Walking skeleton** | 0–15 | 129 | 33,040 | One prompt end-to-end: TUI → loop → 1 provider → read/edit/shell → sandbox → JSONL. 20 golden transcripts green. Sandbox probe works on 3 OS. |
| **P1 — Usable product** | 15–30 | 92 | 22,730 | 4 providers, compaction ladder, MCP client, hooks, resume, approvals + audit, cost display, ACP endpoint, `exec` mode. Internal dogfooding daily. |
| **P2 — Enterprise GA** | 30–48 | 73 | 17,280 | Managed policy, SSO, SIEM export, AppContainer/Seatbelt hardened, plugin signing, LSP diagnostics, worktrees, docs complete, SBOM + signed releases, eval suite integrated. |
| **P3 — Differentiation** | 48–60 | 14 | 3,510 | Vim mode, embeddings index, tool search, coordinator multi-agent, image rendering, SDK schemas. |

**P0 detail (sabse critical 15 weeks)** — 6 vertical slices, har slice demoable:

1. **Weeks 1–3:** workspace + `protocol` + `core` loop + mock provider + JSONL + reducer tests. *Demo: scripted conversation replays in CLI.*
2. **Weeks 3–6:** TUI shell (transcript, composer, spinner, approvals) — fake events se. *Demo: fake streaming agent, 60 fps.*
3. **Weeks 5–8:** model gateway + 1 real provider + SSE + tool-call assembly. *Demo: real model, read-only tools.*
4. **Weeks 7–10:** tools (read/write/edit/patch/grep/glob/shell) + policy + approvals. *Demo: real bug fix in a sample repo.*
5. **Weeks 9–12:** exec + sandbox (Linux first, macOS second) + resource limits. *Demo: sandboxed shell, escape test suite.*
6. **Weeks 12–15:** context assembly + compaction (snip/summarize) + resume + `doctor` + CI matrix. *Demo: 200-turn session without overflow.*

### 6.3 Team topology (6 logins)

| Role | Owns | Crates |
|---|---|---|
| **Tech lead / architect** | protocol, core loop, ADRs, reviews | `protocol`, `core` |
| **Agent engineer** | context, model gateway, prompts, evals | `context`, `model`, `index` |
| **Tools engineer** | tool surface, patch engine, LSP, git | `tools`, `git`, `ext` |
| **Platform/security engineer** | sandbox, exec, policy, audit, auth | `exec`, `sandbox`, `policy`, `auth` |
| **TUI engineer** | renderer, input, themes, a11y | `tui` |
| **Infra/DevEx engineer** | CI, release, telemetry, server, docs, benchmarks | `server`, `state`, `telemetry`, `cli`, `testkit` |

Rule: **ek crate ka ek owner.** Shared crate (`protocol`, `core`) changes pe tech lead review mandatory.

### 6.4 Risk register

| # | Risk | P | Impact | Mitigation | Early signal |
|---|---|---|---|---|---|
| 1 | Scope creep ("Devin jaisa") | High | Fatal | Tier C scope freeze; P3 backlog me sab | Phase slip > 2 weeks |
| 2 | TUI eats the schedule | High | High | TUI ko P0 me fake events pe build karo; polish P1 | Frame budget red |
| 3 | Sandbox escapes / bypass | Med | Fatal (trust) | Deny-by-default, escape test suite, external audit | Any `--no-sandbox` in wild |
| 4 | Provider API churn | High | Med | IR + adapter isolation, contract tests, mock | Adapter LOC growing |
| 5 | Context cost blowup | High | High | Ledger + budgets + cache-boundary tests | tokens/turn regression |
| 6 | Prompt injection via repo/MCP | High | High | Untrusted-content marking, tool allowlist, no auto-approve on injected instructions, hooks | Agent doing unrequested network calls |
| 7 | Cross-platform terminal bugs | Med | Med | Capability tiers + snapshot tests + real VMs | "works in iTerm only" reports |
| 8 | Windows support slips | Med | Med | AppContainer P2, but **CI on Windows from week 1** | No Windows CI = guaranteed slip |
| 9 | Supply-chain compromise | Low | Fatal | `cargo-deny`, vendored lockfile, signed releases, plugin signatures | New transitive deps without review |
| 10 | Team Rust ramp-up | Med | Med | Pair on hard crates, ADRs, clippy lints, no `unsafe` outside `sandbox`/`exec` | Rising review latency |
| 11 | Compile times kill velocity | Med | Med | 19 crates, sccache, mold/lld, `cargo-nextest` | > 90 s incremental |
| 12 | Enterprise audit retrofit | Med | High | Audit/policy crates P0–P1, not P2 | "we'll add logging later" |

### 6.5 Architecture Decision Records (15, jo repo me hone chahiye)

| # | Decision | Rationale | Revisit trigger |
|---|---|---|---|
| 1 | One binary, many surfaces (arg0 dispatch) | distribution simplicity; Codex pattern | binary > 60 MB |
| 2 | Event-sourced turn + actor per thread | replay, resume, multi-surface, testability | event schema churn > 1/sprint |
| 3 | `protocol` crate on day 1 | SDK/IDE/serialisation stability | never |
| 4 | 18-crate workspace | compile time, ownership, embedding | workspace build > 3 min |
| 5 | Primitive tools, not integrations | MCP covers long tail | — |
| 6 | Search/replace + diff patching, validated + transactional | token cost, safety | model-native patch formats converge |
| 7 | Deny-first in-process policy + OS sandbox | independent layers | — |
| 8 | JSONL truth + SQLite index | crash safety + query speed | SQLite WAL insufficient at scale |
| 9 | tree-sitter repo map, no embeddings by default | 1–4k tokens, zero infra | map+grep recall < 80% on evals |
| 10 | `rmcp` for MCP, no bespoke client | spec churn | rmcp missing critical feature |
| 11 | ACP adapter in `server` | editor reach without plugins | ACP v2 breaking |
| 12 | ratatui immediate mode + virtualised list | predictable perf, small binary | cannot hit 60 fps |
| 13 | Cache-aware prompt ordering enforced by test | 3–10× cost delta | providers remove prefix caching |
| 14 | Subagents return summaries only (sidecar transcripts) | context isolation | — |
| 15 | Telemetry off by default, local logs on | enterprise trust | — |

### 6.6 Dependency shortlist (pin exact versions at kickoff)

| Purpose | Crate | License | Risk |
|---|---|---|---|
| Async runtime | `tokio`, `tokio-util` | MIT | low |
| HTTP/streaming | `reqwest`, `hyper`, `rustls`, `eventsource-stream` | MIT/Apache-2.0 | low |
| Serde/JSON | `serde`, `serde_json`, `toml`, `schemars` | MIT/Apache-2.0 | low |
| CLI | `clap` (derive) | MIT/Apache-2.0 | low |
| TUI | `ratatui`, `crossterm` | MIT | low |
| Unicode | `unicode-width`, `unicode-segmentation` | MIT/Apache-2.0 | low |
| Markdown | `pulldown-cmark` | MIT | low |
| Syntax | `tree-sitter` + curated grammars, `two-face`/`syntect` | MIT | med (grammar ABI) |
| Diff | `similar`, `diffy` | Apache-2.0/MIT | low |
| Files | `ignore`, `globset`, `walkdir`, `encoding_rs` | MIT/Apache-2.0 | low |
| Regex | `regex` | MIT/Apache-2.0 | low |
| DB | `sqlx` (sqlite) or `rusqlite` | MIT/Apache-2.0 | low |
| Git | `gix` | MIT/Apache-2.0 | med (API churn) |
| PTY/process | `portable-pty`, `nix`, `windows-sys`, `signal-hook` | MIT | med |
| Sandbox | `landlock`, `seccompiler` (+ own bwrap/seatbelt/AppContainer code) | MIT/Apache-2.0 | med |
| Keyring | `keyring` | MIT/Apache-2.0 | med (platform backends) |
| OAuth | `oauth2` | MIT/Apache-2.0 | low |
| MCP | `rmcp` | MIT/Apache-2.0 | med (spec churn) |
| Tracing | `tracing`, `tracing-subscriber`, `opentelemetry*` | MIT | low |
| Logging/files | `tracing-appender`, `directories` | MIT | low |
| Testing | `insta`, `proptest`, `criterion`, `rstest` | Apache-2.0/MIT | low |
| Dev tools | `cargo-nextest`, `cargo-deny`, `cargo-audit`, `cargo-dist`, `cargo-llvm-cov`, `cargo-mutants`, `typos` | — | — |

Policy: **no crate without a named owner + license check + advisory check.** `unsafe` only permitted in `sandbox`, `exec` (and even there: documented invariant + test).

### 6.7 Definition of Done (per file, non-negotiable)

Ek file "done" hai jab:
1. Public items documented (`#![warn(missing_docs)]` in libs).
2. Kam se kam ek unit test ya snapshot (pure function ho to property test).
3. Error paths `thiserror` variants ke saath, panic-free (no `unwrap` outside tests; `#![deny(clippy::unwrap_used)]`).
4. Koi blocking I/O async context me nahi.
5. Tracing spans emit karta hai (turn/tool/model me correlate).
6. Budget impact documented (agar hot path hai).
7. ADR/comment me *why*, code me *what*.
8. `cargo clippy -- -D warnings` clean, `cargo fmt` clean.

### 6.8 Aapke current Go ZED se migration strategy

Aapke repo me abhi **93 `.go` files** hain (`cmd/` 12 KB, `internal/` 772 KB) aur docs me "enterprise phase" already likha hai. Rust rewrite ke liye pragmatic path:

| Step | Action | Why |
|---|---|---|
| 1 | **Protocol pehle, port baad me.** Go version ka event/tool JSON shape freeze karo → `protocol` crate ka starting point | Behavior parity measurable hoti hai |
| 2 | **Vertical slice port karo** (TUI + loop + read/edit/shell + 1 provider), poora rewrite nahi | 3 mahine me demoable |
| 3 | **Go version ko oracle banao** — same transcript dono pe chalao, event diff nikalo | Silent regressions pakadta hai |
| 4 | **Enterprise features dobara mat likho, dobara design karo.** Go `internal/enterprise` (secret scan, SBOM, PII) Rust me `policy` + `tools` me split hoga | Warna 20 files ka dead weight |
| 5 | Feature freeze Go pe; Rust P1 tak Go maintenance-only | Team focus |

**Warning:** 1:1 porting ka trap — Go ka goroutine/channel design Rust me actor model ke saath mapping nahi karega. Architecture dobara socho, code dobara nahi.

---

## 7. Sources

1. OpenAI Codex architecture teardown — `codex.danielvaughan.com/2026/03/28/codex-rs-rust-rewrite-architecture/`
2. Codex deepwiki (crate org, ThreadManager/Session/ContextManager) — `deepwiki.com/openai/codex`
3. Codex internals: queue-pair protocol, guardian, 3-OS sandbox — `codex.danielvaughan.com/2026/04/10/codex-cli-internals-queue-pair-guardian-sandbox/`
4. Codex sandbox analysis (Seatbelt / bwrap+Landlock+seccomp / restricted token) — `agent-safehouse.dev/docs/agent-investigations/codex`
5. "Dive into Claude Code: The Design Space…" (arXiv 2604.14228) — permission modes, 5-layer compaction, 4 extensibility mechanisms
6. Claude Code architecture deep dive (gist, source analysis) — 27 hook events, 7 permission layers, sidechain transcripts
7. Reverse-engineering Claude Code — `sathwick.xyz/blog/claude-code` (renderer, query loop, tool pool)
8. Ratatui / terminal capability matrix — `terminfo.dev/framework/ratatui`
9. MCP specification 2026-07-28 (stateless core) + changelog — `modelcontextprotocol.io/specification/2026-07-28`
10. Official Rust MCP SDK (`rmcp`) — `github.com/modelcontextprotocol/rust-sdk`
11. Agent Client Protocol (Zed) — `morphllm.com/agent-client-protocol`, `rywalker.com/research/zed-agent-client-protocol`
12. "The Scaffold Effect in Coding Agents" (KDD 2026 eval workshop) — 40× tokens/solved-task across harnesses
13. Terminal-Bench 2.0 / SWE-bench Pro reporting — `openhands.dev/blog/ai-coding-benchmarks-explained`, `codersera.com` roundup
14. Cross-platform sandbox crates & primitives — `docs.rs/skarn-sandbox`, `docs.rs/proc-sandbox`, ai-sandbox
15. Enterprise AI coding agent deployment (7 mandatory controls) — `northflank.com/blog/enterprise-ai-coding-agent-deployment`, `digitalapplied.com/.../enterprise-coding-agent-deployment-playbook-2026`
16. "Enterprise AI Agents: Everyone Passes the Checklist" — `heym.run/blog/enterprise-ai-agents`
17. Aider repo map internals (tree-sitter + personalised PageRank) — `anishgandhi.com/aider-pagerank-codebase-ranking/`, `ggprompts.com/architecture/aider/`
18. Goose architecture (crate layout, extension-first MCP) — `deepwiki.com/aaif-goose/goose`, `pyshine.com/goose/`
19. Gemini CLI package structure & context engineering — `deepwiki.com/google-gemini/gemini-cli/1.2-package-structure`
20. Direct measurements via GitHub API (30 Oct 2026): file counts for `openai/codex` (5,706 `.rs` / 9,791 total), `aaif-goose/goose` (598 `.rs`), `charmbracelet/crush` (844 `.go`), `sst/opencode` (4,787 `.ts`).

> **Note on source reliability:** items 1–4, 6–7, 12–17 are third-party analyses of closed or fast-moving codebases; numbers may drift. Structural conclusions (crate split, compaction ladders, sandbox matrix) are corroborated across multiple independent sources. File counts in item 20 are first-party measurements of public repos on the stated date.

---
