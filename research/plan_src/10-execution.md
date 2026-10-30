## 10. Execution discipline — DoD, risks, backlog, traceability

### 10.1 Definition of Done — per file

Ek file "done" hai jab:

1. Public items documented; crate me `#![warn(missing_docs)]`.
2. Kam se kam ek test (pure logic ho to property test, UI ho to snapshot).
3. Panic-free: `#![deny(clippy::unwrap_used, clippy::expect_used)]` (tests me allowed).
4. `unsafe` nahi (except `sandbox`/`exec` with documented invariant + test).
5. Async context me blocking I/O nahi.
6. Tracing spans emit karta hai (`turn`, `tool`, `model`, `worker` correlate kar sakein).
7. Errors typed (`thiserror`) + user-facing remediation string.
8. Hot path ho to perf budget comment + bench.
9. Role ek line me explain ho jaye (warna split).
10. `clippy -- -D warnings` + `fmt` clean.

### 10.2 PR checklist (template me hi daal do)

```
[ ] Scope: ek crate, ek concern (crates: protocol/core/tools/...)
[ ] Tests: unit/snapshot added; golden transcript updated if prompt/tool changed
[ ] Budgets: startup/RSS/frame impact measured (bench link)
[ ] Security: no new path join outside path_guard; no raw output to terminal
[ ] Contracts: protocol/event changes are additive + schema_version bumped
[ ] Prompts: snapshot diff reviewed; eval smoke run
[ ] Docs: config key → docs table; CLI flag → --help + docs
[ ] Changelog entry (breaking? migration?)
[ ] ADR added if architectural decision taken
```

### 10.3 Review rules

- **Ek crate ka ek owner** approve karta hai; `protocol`, `policy`, `sandbox` ke liye **do** approvals (tech lead + security).
- 400 LOC se bada diff = split (except generated/snapshot).
- Prompt changes alag PR me (behavior vs code separate) — warna regression blame mushkil.
- Snapshot updates alag commit me reason ke saath.

### 10.4 Risk register (12 risks, owners ke saath)

| # | Risk | P | Impact | Mitigation | Early signal | Owner |
|---|---|---|---|---|---|---|
| 1 | Scope creep (Devin-jaisa) | H | Fatal | Scope freeze §1.4; P3 backlog me | phase slip > 2 weeks | Tech lead |
| 2 | TUI schedule eat kare | H | High | Fake events se P0 build; polish P1 | frame budget red | TUI eng |
| 3 | Sandbox escape/bypass | M | Fatal (trust) | fail-closed, escape suite, external audit | `--no-sandbox` usage in telemetry | Security eng |
| 4 | Provider API churn | H | Med | IR + adapters + contract tests | adapter LOC growth | Agent eng |
| 5 | Context cost blowup | H | High | ledger + budgets + cache tests | tokens/solved-task up 10% | Agent eng |
| 6 | Prompt injection (repo/MCP) | H | High | untrusted tagging, tool allowlist, hooks | unrequested net calls | Security eng |
| 7 | Parallel mode merges break mainline | M | High | leases + verify gates + default off | conflict rate > 10% | Agent eng |
| 8 | Windows parity slips | M | Med | Windows CI week 1, probe in P0 | no Windows CI = guaranteed slip | Infra |
| 9 | Compile times kill velocity | M | Med | 19 crates, sccache, mold | incr build > 25 s | Infra |
| 10 | Audit/policy retrofit | M | High | policy crate P0, audit P1 | "add logging later" talk | Security eng |
| 11 | Team Rust ramp-up | M | Med | pair on hard crates, ADRs, lint gates | review latency up | Tech lead |
| 12 | Parallel token cost shock users | M | Med | cost report + warning at < 1.3× speedup | support tickets | Agent eng |

### 10.5 Backlog (v1 ke baad, priority order)

1. **PR automation** — branch push, PR create, review comment loop (policy-gated, P3 in blueprint).
2. **Browser tool** — Playwright-class capability behind sandbox + allowlist.
3. **Remote/cloud sessions** — same core, app-server over TLS; team-shared sessions.
4. **Embeddings index** — only if repo-map + grep recall < 80% on evals (measured, not assumed).
5. **ACP v2** + editor registry listing.
6. **Cost optimizer** — auto model tiering per turn, cache-aware batching.
7. **Team policy server** — central managed policy distribution + telemetry aggregation.
8. **Voice/dictation input**, **image-to-code**, **diagram render** in TUI.
9. **Multi-repo / monorepo scale mode** — partial index, per-package context.
10. **Fine-tuned routing classifier** — replace heuristics for approvals (only after data).

### 10.6 Traceability matrix (requirement → crate → files → test)

| Requirement | Crates | Key files | Test |
|---|---|---|---|
| R1 One prompt end-to-end | `core` | `agent/loop.rs`, `agent/state.rs` | golden transcripts |
| R2 Safe multi-file editing | `tools` | `patch/*`, `fs/edit.rs` | fuzz + property (apply∘reverse) |
| R3 Sandboxed execution | `exec`, `sandbox` | `sandbox/*`, `exec/spawn.rs` | escape suite (3 OS) |
| R4 Context never overflows | `context` | `compaction/*`, `ledger.rs` | 30-turn survival test |
| R5 Great terminal UX | `tui` | `render/*`, `input/*` | frame snapshots + perf |
| R6 Integrations via MCP | `mcp`, `core` | `mcp/*`, `tools/mod.rs` | 3 servers + schema budget |
| R7 Editor integration | `server` | `acp/*` | ACP conformance |
| R8 Tamper-evident audit | `policy` | `audit.rs`, `audit_export.rs` | bit-flip verify test |
| R9 Parallel subagents | `orchestrator` | `dag/scheduler/leases/merge` | 5-task corpus + no-clobber |
| R10 CI/headless | `cli` | `commands/exec.rs` | exit-code contract test |
| R11 Resume / undo | `state` | `resume.rs`, `snapshot.rs` | kill -9 + undo tests |
| R12 Model freedom | `model` | `providers/*` | 4 provider contract tests |

### 10.7 Decisions jo aap se chahiye (kickoff se pehle)

1. **Primary OS priority?** Default: macOS + Linux first, Windows parity P2 (lekin CI week 1).
2. **Default provider/model?** Config-driven; recommendation: Anthropic (cache + tools maturity) + OpenAI fallback.
3. **License?** Apache-2.0 (enterprise-friendly, patent grant) — recommend.
4. **Team reality?** 6 engineers ka plan hai; agar 3 hain to §5 ka P0-only slice lo (129 files) aur P1 12 mahine me.
5. **Monetization?** OSS core + paid enterprise (SSO/audit/policy server) ya pure OSS? Iska asar `ext/plugins` + `server` scope pe padta hai.
6. **ZED Go codebase ka future?** Recommend: feature freeze + protocol parity oracle (§blueprint 6.8), phir archive.
7. **Telemetry vendor?** Default: none (local only). Enterprise: OTLP to customer's collector.
8. **Parallel mode default?** Recommend `off` until §4.15 metrics pass.

### 10.8 Glossary

| Term | Meaning |
|---|---|
| **Op / Event** | Client→agent commands / agent→client facts |
| **Thread / Turn** | Conversation stream / one user prompt's execution |
| **Worker** | Isolated subagent with own worktree + budget |
| **Lease** | Exclusive write permission on a path set |
| **Repo map** | Token-budgeted tree-sitter + PageRank summary of the codebase |
| **Compaction ladder** | 5-stage context reduction (snip → dedupe → collapse → summarise → isolate) |
| **Cache boundary** | Point after which prompt bytes may change without invalidating cache |
| **Verify gate** | Build/test/lint gate before a worker's change is merged |
| **Escape suite** | Tests that attempt to break out of the sandbox |
| **Bundle** | Redacted diagnostic archive for support |

---

### Final word

Ye plan ka pura point ek hi hai: **agent ka dimaag model hai, product uska harness hai.** 308 files me se ~300 files harness ki hain — aur wahi 300 files decide karti hain ki aapka agent reliable lagega ya jua. Agar aap P0 (129 files / 33k LOC) 15 hafte me ship kar dete ho, to aapke paas ek asli agent hoga; baaki sab uske upar additive hai.
