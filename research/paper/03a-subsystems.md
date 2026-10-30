## 3. Subsystems — design se implementation tak

### 3.1 Context economy (sabse zyada ROI wala subsystem)

Context window ek budget hai, memory nahi. Isliye `crates/context` (23 files, 5,630 LOC) ek **ledger + scheduler** hai, string concatenation nahi.

#### 3.1.1 Token ledger

```
Window: 200,000 tokens (model dependent)
├── reserve_output        16,000   (model ke liye chhoda)
├── reserve_safety         4,000   (tokenizer error margin — hamesha underestimate karo)
├── system_core            2,500   (identity + hard rules)
├── tools_schema           6,000–18,000  ← sabse zyada ignore ki gayi line item
├── memory                1,000–6,000
├── repo_map              1,024–4,096
├── history            remaining
└── attachments        history me se
```

Rules jo code me enforce hote hain (`context/ledger.rs`):
- **Model-facing tools ka schema budget fixed hai.** 40 tools × 150 tokens = 6,000 tokens *har turn* — chat se pehle hi. Isliye `tool_budget.rs` (P2) me **deferred schema disclosure**: 8 core tools eagerly, baaki `tool_search` se on-demand.
- Ledger estimate vs provider truth ka **drift tracking**: agar 3 turns tak estimate > 8% off hai, safety reserve auto-badhta hai.
- Prompt cache ke saath ye critical: stable prefix me sirf wahi cheez daalo jo badalti nahi. **Ek changing byte prefix me = poora cache miss = 3–10× cost.**

#### 3.1.2 Compaction ladder (5 stages, cheap se expensive)

Claude Code ka public analysis 5-stage pipeline batata hai [5][6]; design yahan is principle pe hai — *sasta stage pehle, aur jitna kaafi ho wahi ruk jao*.

| Stage | Kya karta hai | Cost | Reversible? | File |
|---|---|---|---|---|
| 0. Budget reduce | tool output caps tighten, images drop | 0 | Nahi (lossy) | `compaction/snip.rs` |
| 1. Snip | purane bade tool results clip (head+tail) | 0 | Nahi | `compaction/snip.rs` |
| 2. Dedupe | same file ke repeated reads, duplicate outputs | ~0 | Nahi | `compaction/dedupe.rs` |
| 3. Collapse | inactive segments fold (summary stub + pointer) | 0 | **Haan** | `compaction/collapse.rs` |
| 4. Summarize | model se structured summary, pinned facts preserve | 1 call | Nahi | `compaction/summarize.rs` |
| 5. Subagent isolate | exploration child me, sirf summary parent me | 1 call | Nahi | `core/subagent/` |

Summary prompt ka contract (structured, not prose): *goal, decisions made, files touched + why, failing tests, open questions, next 3 steps*. Isse "amnesia" fix hota hai — Claude Code ke analysis me bhi yahi pattern hai.

**Post-compaction restore:** last 5 edited files ka content + active plan re-inject (`compaction/restore.rs`). Bina iske model bhool jaata hai ki wo kya kar raha tha — empirically sabse common "agent got dumber" complaint.

#### 3.1.3 Cache-aware assembly

Provider prompt caching (Anthropic `cache_control`, OpenAI automatic prefix cache) sirf tab kaam karta hai jab **prefix byte-identical** ho.

Assembly order (fixed):
```
[ system: identity + rules ]        ← never changes in a session
[ system: tool schemas ]            ← stable per toolset
[ system: memory (AGENTS.md) ]      ← stable per directory set
[ user: env snapshot ]              ← changes rarely (branch, date)
────────────── cache boundary ──────────────
[ history: compacted prefix ]
[ history: recent turns ]           ← grows
[ user: current prompt ]            ← changes every turn
```

`context/assemble.rs` is order ko *enforce* karta hai — koi bhi code jo env ko system se pehle daalna chahe, test fail karega (`cache_aware_order` snapshot test).

#### 3.1.4 Memory hierarchy (file-based, transparent)

| Layer | Path | Loaded | Size cap |
|---|---|---|---|
| Managed | `/etc/<agent>/AGENTS.md` | always | 4 KB |
| User | `~/.config/<agent>/AGENTS.md` | always | 8 KB |
| Project root | `./AGENTS.md` | always | 16 KB |
| Nested | `./sub/AGENTS.md` | **lazy**, us dir ko read karne pe | 8 KB |
| Local | `./AGENTS.local.md` (gitignored) | always | 8 KB |
| Rules | `./.agent/rules/*.md` (glob-scoped) | lazy | 4 KB each |

Why file-based: audit-able, diff-able, review-able. Why not embeddings-only: opaque, stale, aur "why did the agent do that" ka jawab nahi deta.

#### 3.1.5 Repo map (bade codebase ka jawab)

Aider ka proven design [17]: tree-sitter se definitions/references nikalo → file graph banao → **personalised PageRank** → token budget me top symbols render karo.

Numbers jo design decide karte hain:
- Cost: 1,024–4,096 tokens for a 500k-LOC repo map — full-file dumping se 100× sasta.
- Cold build: 0.5–3 s (parallel tree-sitter, rayon), incremental: 10–40 ms (mtime+hash cache in SQLite).
- Ranking knobs: chat me mentioned symbols ×10, already-open files ×50, long identifiers ×10, edge weight = `sqrt(ref_count)`.
- Failure mode: low-PageRank peripheral code map me nahi aata → isliye **grep tool mandatory hai** (map = prior, grep = recall).

`index` crate 8 files / 2,150 LOC — yahi 80% "codebase understanding" hai. Embeddings (P3, optional) sirf tab jab map+grep fail karein.

---

### 3.2 Tool plane — kam tools, gehri quality

#### 3.2.1 Tool set (v1: 14 core tools)

| Tool | Risk class | Notes |
|---|---|---|
| `read_file` | read | paged (line ranges), binary detect, image passthrough |
| `write_file` | mutate | atomic write, backup, encoding preserve |
| `edit_file` | mutate | exact string replace + occurrence disambiguation |
| `apply_patch` | mutate | unified diff/V4A, validated, transactional |
| `glob` | read | gitignore-aware, capped |
| `grep` | read | regex + context lines, type filters |
| `shell` | **exec** | persistent session, risk-scanned, sandboxed |
| `list_dir` | read | tree with sizes |
| `todo_write` | state | plan visibility (bahut under-rated) |
| `task` | spawn | subagent with budget |
| `web_fetch` | net | SSRF guard, size cap, HTML→text |
| `ask_user` | meta | structured elicitation |
| `git_status/diff` | read | prompt-friendly summaries |
| `diagnostics` (LSP) | read | post-edit errors |

**Design rule:** naya tool tabhi jab primitive composition se possible na ho. "Jira tool" nahi — `shell` + `web_fetch` + MCP.

#### 3.2.2 Patch strategy (token economics ka core)

Teen options, measured reality:

| Format | Tokens (100-line edit) | Failure mode | Verdict |
|---|---:|---|---|
| Whole-file rewrite | 3,000–15,000 | truncation, silent loss | ❌ |
| Search/replace blocks | 150–400 | ambiguous match | ✅ default |
| Unified diff | 200–500 | context mismatch | ✅ fallback |
| V4A / custom patch | 180–450 | parser bugs | ✅ internal |

`tools/patch/` (5 files, 1,740 LOC) isliye bada hai: parser (480) + apply (420) + validate (300) + rollback (260) + facade. Har patch apply hone se pehle **current bytes ke against validate** hota hai; mismatch pe model ko exact context wapas jaata hai (self-correct loop). Apply transactional hai: journal + partial-failure rollback, kyunki multi-file patch ka aadha apply hona worst-case hai.

#### 3.2.3 Result shaping (ye 80% context bugs ka fix hai)

`core/tools/result_policy.rs` per-tool policy:

```rust
match tool {
  Read   => { max_lines: 2000, head_tail: true,  dedupe: true },
  Grep   => { max_matches: 200, group_by_file: true, elide_repeats: true },
  Shell  => { max_bytes: 30_000, ansi: strip, exit_code: always, tail_bias: true },
  Lsp    => { max_diagnostics: 50, group_by_severity: true },
  ...
}
```

Principles:
- **Truncation informative ho:** "showing lines 1–200 of 4,312; use offset=200" — model ko next action pata chale.
- **Exit code kabhi na kaato** — model ka pura error-recovery isi pe depend karta hai.
- **Repeated identical output** ko `[unchanged since last read]` se replace karo (dedupe) — token bachta hai aur model confuse nahi hota.

#### 3.2.4 Parallel tool calls

Model ek turn me 3 reads maang sakta hai. Rules (`tools/router.rs`):
- **Read-only tools** parallel (JoinSet, cap 8).
- **Mutating tools** serial, aur agar same file pe do mutating calls hain → second reject with explanation.
- **Order preservation** in results, warna model ka reasoning todta hai (provider ko results call order me chahiye).
- Har call ka apna timeout + cancellation token.

---

### 3.3 Model gateway — N providers, ek IR

#### 3.3.1 Internal IR (protocol crate)

Provider-agnostic message model: `Role`, `ContentPart::{Text, Image, Thinking, ToolUse, ToolResult}`, `ToolCall{id, name, args_json}`, `Usage{in, out, cache_read, cache_write, reasoning}`. Har provider adapter **sirf translation** karta hai — retries, caching, cost, accounting gateway me hai.

Why: providers ka API shape badalta rehta hai (Responses API vs Chat Completions vs Messages vs Converse). Agar har provider apna loop likhe, 4 agents ban jaate hain.

#### 3.3.2 Streaming + tool-call assembly

Sabse bug-prone jagah: **streamed tool arguments JSON** aksar tootey hue chunks me aata hai.

`model/stream_assembly.rs` (320 LOC) rules:
1. Buffer deltas per call-id.
2. Har chunk pe incremental parse attempt — success pe emit `ToolCallDelta`, failure pe silent continue.
3. Stream end pe **strict** parse; fail = structured error with raw buffer (model ko dikhta hai, retry hota hai).
4. `finish_reason` mismatch (tool_calls bheja par args empty) detect karke ek repair-retry.

#### 3.3.3 Resilience

| Failure | Handling | File |
|---|---|---|
| 429 / overloaded | `retry-after` respect, jittered backoff, queue | `ratelimit.rs` |
| 5xx transient | 3 retries, exponential + jitter, idempotent key | `client.rs` |
| Stream drop mid-turn | resume from last event index if provider allows, else replay turn | `stream.rs` |
| Model refuses / empty | one re-ask with stricter instruction, then surface | `retry.rs` |
| Context overflow | compact ladder (3.1.2), then retry once | `core/context_mgr` |
| Provider down | failover chain (same-family first), warn on capability loss | `failover.rs` |
| Auth expired | refresh token, single retry, then interactive re-login hint | `auth/token_store.rs` |

#### 3.3.4 Cost & cache telemetry

`pricing.rs` + `usage.rs`: per-turn cost, per-session spend, cache hit rate, tokens/tool-call. UI status bar me live (`tui/render/status.rs`), `exec` mode me JSON me. Enterprise me ye "budget enforcement" ban jaata hai (`agent/budget.rs` hard stop).

---
