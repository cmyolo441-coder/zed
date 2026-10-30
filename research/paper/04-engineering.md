## 4. Engineering discipline — jo 300+ files ko maintainable banata hai

74k LOC Rust akele likhna possible hai; 74k LOC **team ke saath 2 saal tak** maintain karna alag skill hai. Ye section wahi decide karta hai.

### 4.1 Test strategy (pyramid, agent-specific)

```
                ┌─────────────────────────────┐
   ~40 evals    │ Terminal-Bench / SWE-bench  │  nightly, cost-capped
                │ adapter + internal task set │
            ┌───┴─────────────────────────────┴───┐
   ~180       │ Golden transcripts (insta)          │  every PR, < 60 s
              │ mock provider, real tool loopback   │
          ┌───┴─────────────────────────────────────┴───┐
  ~600 unit │ Reducers, parsers, policy, ledger, patch   │  every save, < 5 s
            └────────────────────────────────────────────┘
```

| Layer | What | Tool | Gate |
|---|---|---|---|
| Unit | pure functions: reducers, parsers, policy eval, token math | `cargo test` | PR, < 5 s |
| Snapshot | TUI frames, prompt assembly, compaction output, JSON schema | `insta` | PR |
| Property | patch apply ∘ reverse = identity; truncate is UTF-8 valid; policy monotonicity | `proptest` | PR (slow subset nightly) |
| Fuzz | patch parser, ANSI parser, markdown parser, config TOML, SSE decoder | `cargo-fuzz` | nightly 30 min |
| Integration | full turn with mock provider + temp repo + real sandbox | `testkit` | PR |
| Chaos | kill -9 mid-turn, disk full, network drop, terminal resize storm | script | nightly |
| Eval | 20–40 real tasks, pass@1 + tokens + turns | custom harness | nightly/weekly |
| Perf | startup, frame time, RSS, tokens/turn | `criterion` + budgets | PR (regression > 10% fail) |

**Golden transcripts** is blueprint ka secret weapon hai: ek transcript = user prompt + scripted mock responses + expected event sequence + expected file tree. Isse model-facing behavior (prompt, tool choice, compaction) *deterministically* test hota hai — bina API cost ke.

Mock provider requirements (`model/src/mock.rs` + `testkit/src/mock_provider.rs`):
- scripted SSE streams (including malformed chunks),
- delayed/hanging responses (timeout tests),
- tool-call argument fragmentation,
- 429/500/auth-expiry scenarios,
- token-usage lies (to test ledger drift).

### 4.2 Eval harness (apna, vendor-neutral)

| Benchmark | Kya measure karta hai | Kaise integrate |
|---|---|---|
| **Terminal-Bench 2.0** (89 tasks, Docker) | shell-driving, env recovery | container per task, `exec` mode, hidden tests |
| **SWE-bench Verified / Pro** | multi-file patch quality | repo checkout + patch apply + test run |
| **Internal suite (40 tasks)** | aapke product ke real workflows | aapke fixtures, fast, cheap |

Report card (har run): pass@1, **tokens/task**, turns, tool calls, wall time, cost, failure-mode histogram. `tokens/task` ko first-class metric banao — harness quality ka sabse honest signal (40× variance real hai [12]).

Nightly budget: 40 tasks × 3 attempts × ~150k tokens ≈ manageable; weekly full Terminal-Bench run.

### 4.3 Performance budgets (CI-enforced)

| Metric | Budget | Measurement |
|---|---:|---|
| Cold start (`--version` → exit) | < 60 ms | hyperfine in CI |
| TUI first frame | < 120 ms | instrumented |
| Idle RSS | < 120 MB | `/usr/bin/time -v` |
| Transcript scroll frame (10k msgs) | < 16 ms p95 | synthetic bench |
| SSE event → UI update | < 2 ms | in-process bench |
| Tool dispatch overhead | < 5 ms | bench |
| SQLite write (turn end) | < 8 ms | bench |
| Binary size (stripped, LTO) | < 38 MB | release CI |
| Prompt assembly | < 5 ms | bench |

Budget fail = PR red. Ye "Rust hone ka fayda" ko measurable banata hai, warna 18 mahine baad pata chalta hai ki binary 120 MB hai aur startup 900 ms.

### 4.4 Distribution & release engineering

| Channel | Mechanism | Notes |
|---|---|---|
| Direct download | `cargo-dist` artifacts (linux x86_64/aarch64 musl, macOS universal, Windows x64/arm64) | static musl for Linux |
| npm | thin JS shim + postinstall binary fetch | Codex pattern; JS me business logic zero |
| Homebrew | tap with formula + bottle | macOS/Linux |
| Windows | Scoop manifest + winget | ARM64 support |
| Containers | `FROM scratch`-ish image, non-root, no shell | CI/agent-in-container |
| Self-update | signed manifest, staged rollout, **rollback slot** | `cli/commands/update.rs` |

Security gates in CI:
1. `cargo-deny` (licenses, advisories, bans, sources) — allowlist: MIT, Apache-2.0, BSD-3, ISC, Unicode-3.0. `ring`/`aws-lc` jaise licenses case-by-case.
2. `cargo-audit` + `osv-scanner` daily.
3. `cargo-cyclonedx` SBOM per release; Sigstore/cosign signature; SLSA provenance (optional but sales-friendly).
4. macOS notarization + hardened runtime; Windows Authenticode.
5. Reproducible build check (`SOURCE_DATE_EPOCH`, `--remap-path-prefix`).
6. `cargo-mutants` weekly on `policy`, `sandbox`, `patch` (mutation score > 80% for security crates).

### 4.5 Cross-platform reality

| Area | Risk | Mitigation |
|---|---|---|
| Windows ConPTY | PTY behavior differs; Ctrl+C semantics | `portable-pty` + real Windows CI VM tests |
| Windows paths | `\\?\` prefixes, case-insensitivity, MAX_PATH | path normalisation in `core/util/paths.rs`, tests |
| macOS Seatbelt | deprecated `sandbox-exec`, SIP interactions | probe + profile test at startup; fallback ask-mode + warn |
| Linux userns | some distros disable unprivileged userns | bwrap probe → Landlock fallback → fail-closed |
| Terminals | 30+ emulators, wildly different caps | capability probe + 5 profile tiers, snapshot tests per tier |
| Locales/encoding | non-UTF8 files, CRLF, BOM | encoding detection (`encoding_rs`), preserve on write |
| Signals | SIGWINCH/SIGTSTP/SIGCONT handling | `signal-hook`, suspend/resume tests |

### 4.6 Documentation & developer experience (yes, ye bhi files hain)

Code ke alawa ~60–90 non-code files chahiye (manifest me count nahi kiye):

- `docs/` — quickstart, config reference (generated from schema), sandbox matrix, MCP guide, ACP guide, hooks reference, security model, threat model, troubleshooting.
- `docs/adr/` — 15 ADRs (section 6.5).
- `.github/workflows/` — 8–12 workflows (ci, release, nightly eval, fuzz, audit, docs).
- `prompts/` — 10–15 embedded prompt fragments (system, compact, summarizer, plan, review, classifier).
- `examples/` — 3 sample plugins, 5 hooks, 2 MCP configs.

**Prompt files ko code treat karo:** versioned, snapshot-tested, aur eval se gated. Prompt change = behavior change = regression risk.

### 4.7 Observability (support ke bina enterprise nahi)

| Signal | Where | Retention |
|---|---|---|
| Structured logs | `~/.agent/logs/*.jsonl` ring buffer | 7 din / 50 MB |
| Turn traces (spans: turn → model call → tool) | local, OTLP optional | 24 h |
| Latency/token/tool-failure metrics | in-memory + periodic summary | session |
| `agent doctor` | env + sandbox + provider + config validation | on demand |
| `agent support-bundle` | redacted tar.gz with preview | on demand |

Rule: **code/prompt content kabhi default telemetry me nahi.** Metadata (counts, timings, error codes) allowed; content opt-in only.

---
