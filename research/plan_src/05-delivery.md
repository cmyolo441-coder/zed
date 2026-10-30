## 5. Delivery plan — 4 phases, 15 sprints, gated milestones

### 5.1 Phases

| Phase | Weeks | Files | LOC | Goal | Gate (exit criteria) |
|---|---:|---:|---:|---|---|
| **P0** Walking skeleton | 0–15 | 129 | 33,040 | 1 prompt → TUI → 1 provider → read/edit/shell → sandbox → JSONL | **M1**: 20 golden transcripts green, 3-OS sandbox probe, cancel-safe, demo recording |
| **P1** Usable product | 15–30 | +92 (221) | +22,730 (55,770) | 4+ providers, compaction, MCP, hooks, resume, approvals+audit, ACP, board foundation | **M3**: internal dogfood 20 devs, 8h/day, no P0 bugs, eval ≥ 55% |
| **P2** Enterprise GA | 30–48 | +73 (294) | +17,280 (73,050) | managed policy, SSO, SIEM, AppContainer, plugin signing, LSP, merge gates, signed release | **M4**: external audit clean, SBOM, 3 pilot enterprises |
| **P3** Differentiation | 48–60 | +14 (308) | +3,510 (76,560) | vim mode, embeddings, tool search, ACP v2, PR automation, multi-repo | **M5**: parallel mode default-on for ≥ 1.8× speedup cases |

### 5.2 P0 sprints (2 weeks each, 6 engineers)

| Sprint | Focus | Main files | Demo | Exit test |
|---|---|---|---|---|
| **S0** (w1–2) | Skeleton: workspace, 19 crates stubs, CI, lint policy, protocol types, config default, testkit | `protocol/*`, `core/config/*`, `testkit/*`, `cli/main.rs` | `agent --version`, `agent doctor` (stub), CI green | build < 2 min, clippy clean |
| **S1** (w3–4) | Agent loop on mock provider + state machine + events + JSONL rollout + exec CLI | `core/agent/*`, `core/events.rs`, `state/rollout.rs`, `model/mock.rs`, `cli/exec.rs` | scripted 6-turn conversation replays identically | 10 golden transcripts |
| **S2** (w5–6) | TUI shell on fake events: transcript, composer, spinner, status bar, term caps | `tui/app.rs`, `update.rs`, `render/transcript.rs`, `render/layout.rs`, `term/caps.rs` | 10k-message scroll at 60 fps | frame p95 < 16 ms |
| **S3** (w7–8) | Real provider (Anthropic) + SSE + tool-call assembly + read-only tools | `model/stream*.rs`, `providers/anthropic.rs`, `tools/fs/read.rs`, `tools/search/grep.rs`, `tools/fs/glob.rs` | ask a question on a real repo, streamed answer | cancel mid-stream safe |
| **S4** (w9–10) | Patch engine + write/edit + policy engine + approval pipeline + ask mode | `tools/patch/*`, `tools/fs/{edit,write}.rs`, `policy/{rules,engine}.rs`, `core/approval/*`, `tui/render/approval.rs` | fix a real bug, approve 3 calls, diff shown | patch fuzz 1M iterations clean |
| **S5** (w11–12) | exec + Linux sandbox (bwrap/Landlock/seccomp) + shell tool + risk scan | `exec/*`, `sandbox/{lib,policy,linux_*}.rs`, `tools/shell/*` | sandboxed `cargo test`; escape suite blocked | escape tests: 0 bypass |
| **S6** (w13–14) | Context assembly + ledger + compaction (snip/summarize) + resume + macOS Seatbelt | `context/{assembler,ledger,compaction/*}.rs`, `state/resume.rs`, `sandbox/macos_seatbelt.rs` | 200-turn session, 2 compactions, no amnesia | cache-stability snapshot |
| **S7** (w15) | Hardening: doctor, signals, panic safety, Windows probe, docs, demo | `cli/commands/doctor.rs`, `signals.rs`, `tui/term/raw.rs`, `docs/*` | full demo: fix → test → commit-ready diff | **M1 gate** |

**P0 ke saath ye bhi lazmi (parallel tracks):**
- CI matrix week 1 se: linux-x64, linux-arm64, macos-arm64, **windows-x64** (ConPTY + paths), 2 terminal emulators in TUI snapshot tests.
- Perf budgets week 2 se CI me (startup, RSS, binary size) — baad me add karna painful hai.
- `unsafe` lint, `cargo-deny`, `typos` week 1 se.

### 5.3 P1 sprints (weeks 16–30)

| Sprint | Focus | Key files | Exit |
|---|---|---|---|
| **S8** | Providers 2–4 (OpenAI Responses, Gemini, Ollama) + failover + cost UI | `model/providers/*`, `model/{gateway,failover,pricing}.rs`, `tui/render/status.rs` | 4 providers pass contract tests |
| **S9** | MCP client (rmcp) + dynamic tool registry + tool schema budget + hooks | `mcp/*`, `core/tools/{mod,router}.rs`, `context/tool_budget.rs`, `ext/hooks/*` | 3 MCP servers, schema tokens < 8k |
| **S10** | Repo map + index + search ranking | `index/*`, `context/memory/hierarchy.rs` | 500k LOC repo map < 3 s cold, < 40 ms incr |
| **S11** | Full compaction ladder + memory + scratchpad + restore | `context/compaction/*`, `context/memory/*`, `context/scratchpad.rs` | 30-turn task survives 2 compactions |
| **S12** | State depth + app-server JSON-RPC + ACP | `state/{snapshot,replay,resume}.rs`, `server/*` | Zed me agent chalta hai via ACP |
| **S13** | Auth + audit + redaction + managed config | `auth/*`, `policy/{audit,redaction,managed}.rs` | audit verify + SIEM export demo |
| **S14** | Orchestrator foundation (dag/scheduler/leases/board) + profiles | `orchestrator/*` (7 files), `core/subagent/*`, `tui/render/board.rs` | 4 workers, 0 clobber, 1.5× speedup |
| **S15** | Beta hardening, eval suite, perf, docs, migration guide | `testkit/*`, `docs/*`, `evals/*` | **M3 gate** |

### 5.4 P2 items (weeks 31–48, GA)

AppContainer + Job Object (Windows parity), plugin signing + trust store, LSP diagnostics, merge queue + verify gates, worktrees + GC, SIEM exporters, SSO/device flow hardening, SBOM + notarization + reproducible builds, `cargo-mutants` on security crates, external pentest, docs site, support bundle.

### 5.5 Critical path aur parallelism

```
protocol → core/loop → TUI(fake events) ┐
                    → model gateway     ├─► P0 demo
                    → tools/patch       │
                    → exec → sandbox    ┘
context ──► index ──► (map quality) ──► P1 eval
policy ──► audit ──► enterprise gate ──► P2
state ──► replay ──► orchestrator ──► parallel gate
```

**Critical path:** `protocol → core loop → patch engine → sandbox → compaction`. Inme se koi bhi slip = poora P0 slip. Isliye:
- `protocol` week 1 me freeze (baad me sirf additive).
- `patch engine` ko sabse senior engineer de (fuzz + property tests).
- `sandbox` Linux-first; Windows probe-only P0 me, full AppContainer P2.
- TUI fake events pe chalta hai — model ke intezaar me block nahi hota. **Ye scheduling ka sabse bada unlock hai.**

### 5.6 Ownership (RACI-lite)

| Crate | Owner | Reviewers |
|---|---|---|
| `protocol` | Tech lead | all leads |
| `core` | Tech lead + agent engineer | tools owner |
| `model`, `context`, `index` | Agent engineer | tech lead |
| `tools`, `git` | Tools engineer | security engineer |
| `exec`, `sandbox`, `policy`, `auth` | Platform/security engineer | tech lead |
| `tui` | TUI engineer | design-minded reviewer |
| `state`, `server`, `telemetry`, `cli`, `testkit` | Infra/DevEx engineer | tech lead |
| `orchestrator` | Agent engineer + tech lead | tools + security |
| `ext` | Tools engineer | platform |

### 5.7 Kickoff checklist (pehle 10 working days)

1. Workspace + 19 crate stubs + `Cargo.toml` workspace deps + `rust-toolchain.toml`.
2. CI: fmt, clippy (`-D warnings`), nextest, `cargo-deny`, `typos`, 4-platform matrix, perf budget job.
3. `protocol` v0: Op/Event enums + serde + schema export + snapshot tests.
4. `testkit`: mock provider + temp repo fixture + golden helper.
5. ADRs 1–6 likho (repo me `docs/adr/`), baaki sprint ke saath.
6. Perf harness: hyperfine startup, RSS script, TUI frame benchmark skeleton.
7. Branch protection + CODEOWNERS + PR template (checklist = §10.2 DoD).
8. Threat model v0 (`docs/security/threat-model.md`, §9.2 se).
9. Secrets baseline: `gitleaks` pre-commit + CI.
10. Demo environment: sample polyglot repo + 10 scripted tasks (eval seed).

### 5.8 Milestone gates (go/no-go)

| Gate | Kab | Kya chahiye | No-go matlab |
|---|---|---|---|
| **M0** Architecture review | week 2 | protocol frozen, crate graph approved, ADRs 1–6 | redesign (sasta ab) |
| **M1** Skeleton demo | week 15 | P0 exit criteria + perf budgets green | P1 start nahi |
| **M2** Alpha (internal) | week 22 | 4 providers, compaction, approvals, no data loss | dogfood start nahi |
| **M3** Beta | week 30 | eval ≥ 55%, 20 devs dogfood, ACP working | external pilots nahi |
| **M4** GA | week 48 | audit clean, SBOM, 3 pilots, docs, SSO | enterprise launch nahi |
| **M5** Parallel default-on | week 58 | §4.15 metrics pass | feature off rehta hai |
