# Enterprise-Grade Terminal AI Agent in Rust
## Deep Research Blueprint — File Count, Architecture, Effort aur Build Plan

**Document type:** Deep-research architecture paper (code nahi — sirf research, design aur planning)
**Target:** Claude Code / Codex CLI / Gemini CLI / Auggie / Goose / Crush jaisa enterprise terminal agent, 100% Rust
**Date:** 30 October 2026
**Prepared for:** ZED project (current Go codebase: 93 `.go` files) — Rust rewrite/replacement ke liye
**Machine-readable data:** `research/file-manifest.csv` (308 rows: id, crate, path, purpose, est LOC, phase)

---

## 0. Sawaal ka seedha jawab (TL;DR)

Aapne poocha: *"200–300 files ka code to zarur hoga?"*

**Haan — exactly sahi andaaza hai.** Lekin sirf tab, jab aap "enterprise-level v1" bana rahe ho, "Codex ke barabar" nahi.

| Level | Hand-written source files | Est. LOC | Kya milta hai | Real-world anchor |
|---|---:|---:|---|---|
| **Tier A — Demo / prototype** | 45–65 | 9k–14k | Read/edit/shell + single-provider loop + plain REPL | Weekend-to-4-week projects |
| **Tier B — Serious OSS tool** | 140–170 | 32k–40k | Multi-provider, approvals, sessions/resume, decent TUI, tests | Chhote OSS agents |
| **Tier C — Enterprise v1 (Aapka target)** | **308** | **~76,500** | 19 crates, 3-OS sandbox, MCP+ACP, hooks/skills, audit, SSO, parallel subagents, evals, signed releases | Goose ka ~52% Rust surface, Crush ke barabar se thoda bada |
| **Tier D — Frontier parity** | 1,200–2,500 | 250k–450k | Multi-surface (TUI+IDE+desktop+cloud), multi-agent orchestration, telemetry mesh | Codex (5,706 `.rs` files, ~549k LOC) |

**Is paper ka concrete deliverable:** ek 308-file / 76,560-LOC blueprint — har file ka naam, role, estimated LOC aur delivery phase ke saath. Plus effort math, team plan, risk register, ADRs aur dependency list.

### Ek line me core thesis

> Terminal agent ka **model-facing logic 1–2% hota hai**. Baaki 98% "harness" hai — context economy, permission plane, state machine, sandbox, renderer, replay, audit. Isi liye 300+ files lagti hain: model ek API call hai, product uske *aas-paas ka operating system* hai.

Is thesis ka evidence section 1.3 me hai (Claude Code ke reverse-engineering se 1.6% figure, aur Terminal-Bench ka "scaffold effect" — same model, different harness = same task pe **40× token difference**, accuracy sirf 0–8 pp badalti hai [12][13]).

### Numbers jo yaad rakhne hain

| Metric | Value | Note |
|---|---:|---|
| Source files (hand-written) | **308** | 19 crates, avg 249 LOC/file |
| Test + snapshot files | ~179 + ~250 | insta snapshots Rust me default practice hai |
| Total tracked files | **~750** | Codex me 1,671 `.snap` files hain — snapshots normal hain |
| P0 (walking skeleton) | 129 files / 33,040 LOC | Pehle 3–4 mahine |
| P1 (beta) | +90 files / 22,210 LOC | Agle 3 mahine |
| P2 (GA) | +67 files / 15,630 LOC | Agle 4–5 mahine |
| P3 (differentiation) | +14 files / 3,290 LOC | Post-GA |
| Effort (6 seniors + AI assist) | ~10–12 mahine | 5.5–6.5 person-year |
| Effort (3 engineers) | ~20–26 mahine | Scope cut karna padega |
| Binary size target | 28–38 MB stripped | Codex single binary isi range me |
| Cold start target | < 60 ms | Rust ka pura fayda yahan hai |

### Kya aapko *nahi* karna chahiye

1. **300 files ko ek hi crate me mat dalo.** 300 files / 1 crate = 40+ minute incremental builds, merge conflicts, aur test suite jo har PR pe 6 minute leta hai. 19 crates ka split compile time ka *feature* hai.
2. **Model se free-form "poora file rewrite" karwana band karo.** Patch-based editing (diff + strict validation + rollback) hi token-efficient aur safe hai.
3. **Sandbox ko baad me mat jodo.** Ye baad me retrofit karne pe har tool signature badalta hai.
4. **Embeddings/RAG ko default mat banao.** tree-sitter repo-map + grep 90% kaam karta hai, zero infra me. (Aider ka repo-map ka data point section 3.1.5 me.)
5. **"Enterprise" ka matlab SSO + audit + air-gap hai — extra features nahi.** Wo 4 crates (policy/auth/telemetry/state) design ke *pehle din* se chahiye, warna audit trail fake ban jaata hai — jaisa aapke current `ENTERPRISE_PLAN.md` me "Phase 2" me retrofit hua.

---

## 1. Research method (isse maine kaise nikala)

Guessing nahi — teen tarah ke evidence use kiye:

**(a) Direct measurement (GitHub API, 30 Oct 2026).** Real agents ke repos ka file tree pull karke count kiya:

| Repo | Language | Source files | Total files | Notes |
|---|---|---:|---:|---|
| `openai/codex` | Rust | **5,706 `.rs`** | 9,791 | +1,671 `.snap`, 784 `.ts`, 211 Bazel |
| `aaif-goose/goose` | Rust | **598 `.rs`** | 2,559 | +340 `.tsx` (Electron UI) |
| `charmbracelet/crush` | Go | **844 `.go`** | 1,357 | +363 golden files |
| `sst/opencode` | TS | 4,787 `.ts` + 832 `.tsx` | 9,242 | Rust sirf 34 files (TUI plugin) |
| `Aider-AI/aider` | Python | — | — | 140 MB repo, ~90 modules |

Isse do cheezein clear hoti hain: (i) **Rust agents 500–5,700 files ke range me hote hain**, (ii) aapka 300-file target *upar* ka realistic v1 hai, neeche ka nahi.

**(b) Architecture teardown.** Codex ka crate split, Goose ka crate layout, Gemini CLI ke 7 packages, Claude Code ka post-hoc analysis (UCL ke reverse-engineering paper [5], community gists [6][7]), Aider ka repo-map internals, Auggie/Devin ki feature surface.

**(c) Standards + ecosystem state.** MCP spec revision `2026-07-28` (stateless core) [9], official Rust SDK `rmcp` [10], ACP (Zed, JSON-RPC over stdio, 37 agents) [11], ratatui/crossterm terminal capability matrix [8], OS sandbox primitives (Landlock/seccomp/Seatbelt/AppContainer) [14].

**Honesty note:** web sources me Codex ke crate count pe disagreement hai (52 vs 70 vs 130 vs "208 crates in binary"). Ye isliye hai ki log *workspace members*, *dependency count*, aur *linked crates* ko mix karte hain. Main is paper me "70–160 internal crates" range bolta hoon aur file counts pe bharosa karta hoon, kyunki wo maine khud measure kiye.

### 1.1 Product landscape — jo aapko beat karna hai

| Product | Type | Harness ka differentiator | Rust ke liye lesson |
|---|---|---|---|
| **Claude Code** | Closed, TS/Node | 7 permission modes, 5-layer compaction, 27 hook events, subagent sidechains, ML permission classifier | Harness hi product hai; hooks/compaction ko first-class banao |
| **Codex CLI** | Open, Rust | 5,700 files, event/op protocol, app-server JSON-RPC, 3-OS sandbox, apply-patch crate | Crate split = parallelism; protocol crate alag rakho |
| **Gemini CLI** | Open, TS | 7 packages, policy TOML engine, checkpointing (shadow git), 5 sandbox variants | Policy engine ko declarative + testable rakho |
| **Goose** | Open, Rust | MCP-native, 15+ providers, recipes, LLM "adversary inspector" | Extension-first design; built-in tools bhi MCP jaisa shape |
| **Crush** | Open, Go | Provider-agnostic, polished TUI, LSP awareness | TUI quality hi retention hai |
| **Aider** | Open, Python | tree-sitter + PageRank repo map, 6 edit formats, auto-commit | Repo map = best context/₹ ratio |
| **Auggie (Augment)** | Closed | Persistent semantic index, ACP, CI/headless modes | Index ko background service samjho, per-prompt nahi |
| **Devin** | Closed, cloud | Long-horizon autonomy, browser+shell, PR loop | Autonomy = recovery + verification, intelligence nahi |

### 1.2 "Enterprise-grade" ka matlab (12 pillars)

Enterprise buyers ko model quality se zyada **controls** se matlab hai [15][16]. Ye 12 pillars is blueprint ke acceptance criteria hain:

1. **Identity** — har session ek named human se bound (OIDC/SAML device flow), CI ke liye service accounts.
2. **Policy-as-code** — admin-managed, signed, developer-override-proof settings file.
3. **Permission plane** — deny-first rules + risk classifier + approval modes; session-scoped grants.
4. **OS-level sandbox** — Linux (bubblewrap/Landlock/seccomp), macOS (Seatbelt), Windows (AppContainer + Job Object).
5. **Egress control** — default-deny network, allowlist via proxy; SSRF guard on fetch.
6. **Audit trail** — tamper-evident (hash-chained), SIEM-exportable (syslog/OTLP), per-action attribution.
7. **Data governance** — local-first by default, redaction before egress, retention policy, zero-retention mode.
8. **Air-gap mode** — BYO endpoint (vLLM/Ollama/Bedrock/Vertex), no phone-home, offline license check.
9. **Supply chain** — signed binaries + notarization, SBOM, `cargo-deny`, reproducible builds, updater rollback.
10. **Observability** — opt-in OTel traces, local ring-buffer logs, one-command redacted support bundle.
11. **Extensibility without forking** — MCP, ACP, hooks, skills, plugins (signed), custom slash commands.
12. **Operability** — headless/CI mode with stable exit codes, JSONL streaming, `doctor`, deterministic replay.

Agar inme se 4 bhi missing hain, aap "enterprise" nahi keh sakte — chahe model frontier ho.

### 1.3 Kyun harness par sab kuch depend karta hai (evidence)

- Claude Code ke source analysis me **~1.6% code AI decision logic** hai; baaki permission gates, tool routing, compaction, recovery, persistence [5][6].
- Terminal-Bench 2.0 pe **same model, do harness** = 52.1% vs 57.8%, par input tokens **256.9M vs 3.9M (65×)** [13].
- Ek controlled study (3 harnesses × 2 models × 50 tasks) me **tokens per solved task me 40× gap**, pass-rate me sirf 0–8 pp [12].
- Same model pe custom scaffold 62.3% → 70.3% (SWE-bench Verified) [13].

**Interpretation:** model badalne se zyada farq padta hai ki aap uska *context, tools, retries, verification aur stop-condition* kaise design karte ho. Isi liye 74k LOC ka 90% "boring" infrastructure hai — aur wahi moat hai.

---
