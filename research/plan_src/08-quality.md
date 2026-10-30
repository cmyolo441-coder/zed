## 8. Quality bar — tests, evals, budgets, CI

### 8.1 Test matrix (kaunsi crate ka kaunsa test mandatory)

| Crate | Unit | Snapshot (`insta`) | Property (`proptest`) | Fuzz | Integration | Cross-platform |
|---|---|---|---|---|---|---|
| `protocol` | schema round-trip | ✅ event JSON | ✅ serde round-trip | — | — | — |
| `core` | state machine, budgets, approval table | ✅ prompt assembly | ✅ budget invariants | — | ✅ full turn (mock) | — |
| `model` | converters, token math | ✅ request payloads | ✅ IR↔provider round-trip | ✅ SSE parser | ✅ 4 provider contract tests | — |
| `context` | ledger, ladder selection | ✅ assembled prefix | ✅ token monotonicity | — | ✅ 30-turn survival | — |
| `index` | ranking, tags | ✅ repo map | — | ✅ tree-sitter query parser | ✅ 500k LOC fixture | — |
| `tools` | per-tool contract | ✅ tool output shape | ✅ **patch apply∘reverse = identity** | ✅ **patch parser** | ✅ temp repo | ✅ path semantics |
| `exec` | spec, timeout | — | — | — | ✅ PTY session | ✅ **ConPTY**, signals |
| `sandbox` | policy mapping | — | — | — | ✅ **escape suite** | ✅ 3 OS |
| `policy` | deny-first order, matcher | ✅ decision tables | ✅ monotonicity (more deny = never more allow) | ✅ pattern parser | ✅ audit chain | — |
| `state` | repos, migrations | — | ✅ crash consistency (kill -9 loop) | — | ✅ 100 MB DB | — |
| `tui` | reducer (pure) | ✅ **frames** per terminal tier | — | ✅ markdown/ANSI parser | — | ✅ 3 terminal profiles |
| `server` | jsonrpc framing | ✅ payloads | — | ✅ frame parser | ✅ ACP conformance | — |
| `orchestrator` | dag, scheduler, leases | ✅ board states | ✅ **no-clobber invariant** | — | ✅ 5-task run | — |
| `ext` | frontmatter, hook matcher | ✅ payloads | — | — | ✅ plugin verify | — |
| `auth` | token logic | — | — | — | ✅ mock IdP | — |
| `git` | diff, worktree | — | — | — | ✅ real repos | ✅ worktree paths |
| `telemetry` | scrubber | ✅ redaction cases | — | — | — | — |

**Counts (estimates):** ~179 test/snapshot files, ~600 unit tests, ~250 snapshots, 8 fuzz targets, 40 eval tasks.

### 8.2 Fuzz targets (nightly, 30 min each)

1. `patch/parser` — garbled model output (highest risk).
2. `tui/render/markdown` + ANSI stripper — **escape injection**.
3. `model/stream` SSE decoder — malformed framing.
4. `config` TOML + schema — unknown keys, huge files.
5. `index/tags` tree-sitter queries — malformed source.
6. `policy/rules` pattern parser — pathological globs.
7. `server/jsonrpc` framing — truncated frames.
8. `state/rollout` JSONL reader — corrupt lines.

Rule: har fuzz crash = regression test (fix ke saath commit).

### 8.3 Eval suite (product quality, not just correctness)

| Tier | Tasks | Cadence | Cost cap | Purpose |
|---|---|---:|---:|---|
| Smoke | 5 | every PR (mock) | $0 | plumbing |
| Internal | 40 | nightly | ~$8 | real fixes: bug, feature, refactor, docs, test-gen |
| External | Terminal-Bench 2.0 subset (20) | weekly | ~$25 | shell competence |
| External | SWE-bench Verified subset (20) | weekly | ~$30 | patch quality |
| Parallel | 5-task DAG corpus | weekly | ~$12 | §4.15 metrics |

Report card per run: `pass@1`, **tokens/solved task**, turns, tool calls, wall time, cost, failure-mode histogram. **Gate:** tokens/solved-task 10% se zyada na badhe, warna PR block (harness regression).

### 8.4 Performance budgets (CI-enforced, fail = red)

| Metric | Budget | Job |
|---|---:|---|
| Cold start (`--version`) | < 60 ms | hyperfine |
| Startup → first TUI frame | < 120 ms | instrumented |
| Idle RSS (empty session) | < 120 MB | cgroup measure |
| Transcript scroll frame (10k msgs) | p95 < 16 ms | bench |
| SSE event → UI update | < 2 ms | bench |
| Prompt assembly | < 5 ms | bench |
| SQLite write (turn end) | < 8 ms | bench |
| Binary size (release, stripped) | < 38 MB | size check |
| Incremental build (1 crate) | < 25 s | CI timing |
| Parallel run overhead (vs serial) | < 2.2× tokens | eval |

### 8.5 CI pipeline (stages + budgets)

| Stage | Command | Timeout | Blocks merge? |
|---|---|---|---|
| fmt | `cargo fmt --check` | 1 min | ✅ |
| clippy | `cargo clippy --all-targets -- -D warnings` | 8 min | ✅ |
| test (unit+snapshot) | `cargo nextest run --profile ci` | 10 min | ✅ |
| doc | `cargo doc --no-deps` (warnings = error) | 5 min | ✅ |
| deny/audit | `cargo deny check` + `cargo audit` | 3 min | ✅ |
| typos + markdown lint | `typos`, `markdownlint` | 1 min | ✅ |
| platform matrix | linux x64/arm64, macos arm64, **windows x64** | 20 min | ✅ |
| perf budgets | bench job on main only | 15 min | ✅ (main) |
| fuzz smoke | 2 min per target | 16 min | nightly |
| eval smoke | 5 tasks, mock | 5 min | nightly |
| mutation (security crates) | `cargo-mutants` on `policy`, `sandbox`, `patch` | weekly | advisory → then blocking |
| cross-platform PTY | Windows VM + 3 terminal emulators | 10 min | ✅ |

### 8.6 Release gates (har release)

1. All CI green on tag commit; reproducible build hash match.
2. SBOM attached (`cargo-cyclonedx`), signatures (cosign) + notarization.
3. Changelog with **breaking-change** section (config/CLI/protocol).
4. Upgrade + downgrade test: previous version ke session resumption.
5. Smoke install on 3 OS, `agent doctor` clean.
6. Eval delta vs last release documented (pass rate + tokens).
7. Security: no new `unsafe`, `cargo-deny` clean, advisory triage done.
8. Rollback plan tested (update slot revert).

### 8.7 Snapshot policy (kyunki 250 snapshots maintain karni hain)

- Snapshots **intent** document karte hain, implementation nahi — ek snapshot fail ho to pehle poochho "behaviour badla ya formatting?"
- `.snap` me sirf rendered output; metadata minimal (diff noise kam).
- Har snapshot ka ek owner crate (CODEOWNERS) — random updates banned.
- `INSTA_FORCE_UPDATE` PR me justify karna hoga (reviewer checklist).
- Flaky snapshots (timing, IDs) allowed nahi — injectable clock/IDs use karo.
