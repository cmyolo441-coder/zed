### 3.4 Execution + sandbox plane (defence in depth)

Terminal agent ka blast radius = user ka pura machine. Isliye teen independent layers, aur **teeno independent fail ho sakte hain**:

```
Layer 1  Policy engine        in-process, deny-first rules, risk classifier
Layer 2  Approval UX          human sees exact command + scope, mode-dependent
Layer 3  OS sandbox           kernel-enforced: FS, net, spawn, syscalls
Layer 4  Resource limits      CPU/mem/PID/time, process-group kill
Layer 5  Audit                everything above logged, hash-chained
```

#### 3.4.1 Sandbox matrix (real primitives, 2026)

| OS | Primary | Secondary | Notes |
|---|---|---|---|
| Linux | **bubblewrap** (mount ns) | Landlock LSM + seccomp-BPF | bwrap missing ho to Landlock fallback; Landlock needs 5.13+ |
| macOS | **Seatbelt** (`sandbox-exec`, SBPL profile) | — | deny-default profile + explicit read roots; `sandbox-exec` deprecated but functional |
| Windows | **AppContainer** + capability SIDs | Job Object (mem/PID/kill-on-close) + restricted token | Process khud ko AppContainer me nahi daal sakta → parent worker launch karta hai |

Critical implementation detail (jo aksar galat hota hai): **Landlock/seccomp ko single-threaded child me `exec` se pehle apply karo**, warna multithreaded fork + non-async-signal-safe calls = deadlock. Isliye `sandbox/linux_landlock.rs` ek dedicated worker binary path use karta hai, aur comment me ye constraint likha hota hai.

Policy shape (`sandbox/policy.rs`):
```rust
SandboxPolicy {
  writable_roots: [workspace, ~/.agent/memories],
  readable_roots: [workspace, /usr, /lib, toolchain dirs],
  network: Deny | LocalhostOnly | Allowlist(Vec<Host>),
  spawn: bool,          // child processes allowed?
  env_allowlist: [...],
  resource: { cpu_s, mem_mb, pids, open_files },
}
```

#### 3.4.2 Availability probe + honest degradation

`doctor` aur runtime dono `sandbox/probe.rs` use karte hain:
- Linux: bwrap present? kernel >= 5.13? user namespaces enabled (`/proc/sys/kernel/unprivileged_userns_clone`)?
- macOS: `sandbox-exec` exists? profile test-run pass?
- Windows: AppContainer creation test.

Agar sandbox available nahi: **fail-closed by default**, `--no-sandbox` explicit flag ke saath warning + audit entry. Silent fallback = enterprise deal-breaker.

#### 3.4.3 Approval pipeline (12-step, order matters)

```
1 validate args (schema)      7 budget check (agent/budget.rs)
2 path_guard (single choke)   8 sandbox apply
3 policy.engine.explain()     9 exec + stream
4 risk class classify        10 output shape/truncate
5 approval mode decision     11 audit append
6 user prompt (if needed)    12 Event::ToolCompleted
```

`policy/engine.rs` **deny-first** hai: pehle deny rules match karo, phir allow rules. Isse "allow *" wali config bhi destructive command ko rok deti hai jab explicit deny ho. Har decision ka `explain()` string hota hai — user ko dikhta hai *kaunsi rule* ne roka. Enterprise support me ye sabse zyada time bachata hai.

Modes (5):

| Mode | Read | Edit | Shell | Use case |
|---|---|---|---|---|
| `read-only` | ✅ | ❌ | ❌ (except allowlisted) | review, untrusted repo |
| `ask` (default) | ✅ | ✅ prompt | ✅ prompt | normal dev |
| `auto-edit` | ✅ | ✅ | prompt | power user |
| `plan` | ✅ | ❌ | ❌ read-only cmds | design phase |
| `bypass` | ✅ | ✅ | ✅ | CI/container only, audit + banner |

#### 3.4.4 Redaction before egress

`policy/redaction.rs`: regex + entropy based secret detection (AWS keys, GitHub PATs, JWTs, private key blocks, `.env` values, high-entropy strings in diffs). Do jagah chalta hai: (a) tool results model ko bhejne se pehle, (b) telemetry export se pehle. Placeholder reversible nahi hota — model ko `[REDACTED:aws_key]` dikhta hai, value nahi.

Trade-off: false positives se debugging mushkil hoti hai → per-session `--no-redact` (audited) aur `redaction.allowlist` paths.

#### 3.4.5 Audit chain (tamper-evident, cheap)

```
entry_n = { ts, actor, session, action, target, decision, hash_prev }
hash_n  = blake3(canonical_json(entry_n))
```

Append-only JSONL, har 1,000 entries pe anchor (hash + count) alag file me. Verify command: `agent audit verify`. SIEM export: syslog (RFC 5424) ya OTLP logs. Enterprise requirement: *"show me one audit line with field names"* [16] — isliye schema stable aur documented hona chahiye.

---

### 3.5 Memory plane — JSONL truth + SQLite index

Dual-write design ka reason: do alag access patterns.

| Need | Store | Why |
|---|---|---|
| Crash-proof truth, replay, export | JSONL append (`rollout.rs`) | append-only = no corruption; human-readable; `jq`-able |
| Fast query (resume list, search, usage) | SQLite WAL (`db.rs`) | indexed lookups, transactions |
| Undo/rollback | file snapshots + shadow git (`snapshot.rs`) | edit ke pehle state |

Rules:
1. **JSONL pehle likho, phir SQLite.** Agar SQLite write fail ho, session chalta rahe; startup pe reconciliation (`turns_repo.rs`).
2. **fsync policy:** turn boundaries pe fsync, har delta pe nahi (warna SSD pe 10k fsync/min).
3. **Schema migrations** numbered SQL files, `user_version` pragma, forward-only + tested on a 100MB fixture DB.
4. **Retention:** default 30 din / 500 MB, configurable, `gc.rs` vacuum + orphan cleanup.
5. **Resume semantics:** cwd badal gaya? git HEAD badal gaya? model available nahi? — teeno warn karo, silently mat chalao.

Snapshot strategy: pre-edit content-addressed blobs (`~/.agent/blobs/<hash>`), per-turn manifest. `/undo` = manifest revert. Shadow-git sirf tab jab user git repo me ho aur `git` binary available ho (Gemini CLI approach) — warna blob fallback.

---

### 3.6 TUI plane — retention ka asli factor (36 files, ~9.7k LOC)

Terminal UI me "pretty" se zyada "predictable" matter karta hai. Architecture:

```
crossterm event thread ──► channel ──► update.rs (pure reducer)
                                              │
                                        AppState (immediate-mode)
                                              │
                                        render/* ──► diff buffer ──► stdout
```

#### 3.6.1 Non-negotiables

| Requirement | Implementation | File |
|---|---|---|
| Panic-safe terminal restore | RAII guard + `panic::set_hook` + signal handlers | `tui/lib.rs`, `term/raw.rs` |
| 60 fps scroll on 10k messages | virtualised list, only visible rows laid out | `render/transcript.rs` |
| Streaming markdown without flicker | incremental block parse + stable line cache | `render/markdown.rs` |
| Resize reflow | wrap cache invalidated per width, anchor preserved | `render/transcript.rs` |
| Grapheme-correct cursor | `unicode-segmentation` + `unicode-width` | `input/editor.rs` |
| Large paste | bracketed paste → file ref if > 8 KB | `input/paste.rs` |
| Copy | OSC 52 + native fallback | `term/clipboard.rs` |
| Screen reader | linear mode, no spinner, verbatim | `a11y.rs` |

#### 3.6.2 Terminal capability probing

`term/caps.rs` startup pe probe karta hai (30–80 ms budget, cached by `TERM`+`TERM_PROGRAM`):

| Capability | Query | Fallback |
|---|---|---|
| Truecolor | `COLORTERM` / `TERM` | 256 → 16 |
| Kitty keyboard protocol | `CSI ? u` query | legacy keys (Ctrl+I vs Tab ambiguity!) |
| Synchronized output | `DECRQM 2026` | plain flush (flicker) |
| Bracketed paste | assume + enable | line-buffered |
| Hyperlinks | OSC 8 heuristics | plain URLs |
| Images | kitty/sixel DA response | ASCII placeholder |

**Why it matters:** Ctrl+I aur Tab ka same byte aana, ya resize pe redraw flicker — ye chhoti cheezein user ko "amateur" feel karati hain. Codex/Claude Code dono yahi probe karte hain.

#### 3.6.3 Diff + approval UX

- Unified diff default, side-by-side wide terminals pe.
- Word-level emphasis (`similar` crate), intra-line.
- Approval prompt me: exact command, cwd, sandbox status, risk class, "always allow this pattern" scope preview.
- Diff review mode: accept/reject hunks (jaise Claude Code/Aider).

---

### 3.7 Extensibility plane — MCP, ACP, hooks, skills, plugins, subagents

| Mechanism | Context cost | Trust model | Kab use karein |
|---|---|---|---|
| **Hooks** (shell at lifecycle events) | 0 tokens | local, user-authored | lint, secret scan, audit gates, notifications |
| **Skills** (markdown + allowed tools) | low (lazy) | reviewed repo file | team conventions, runbooks |
| **Slash commands** (markdown) | 0 | repo/user | repeatable prompts |
| **MCP servers** | medium–high (tool schemas) | external process | integrations (GitHub, Jira, DB) |
| **Plugins** (signed bundle) | medium | signature + provenance | enterprise distribution |
| **Subagents** | isolated | same policy | parallel exploration, review |

#### 3.7.1 MCP — the 2026 reality

Spec ab `2026-07-28` pe hai: **stateless core** (no `initialize`, no session id), `server/discover`, Multi-Round-Trip Requests, `Mcp-Method`/`Mcp-Name` headers, tasks as extension, sampling/roots/logging deprecated [9]. Rust me official SDK `rmcp` hai jo current + legacy revisions support karta hai, stdio + Streamable HTTP transports ke saath [10].

Blueprint decisions:
- `rmcp` use karo, bespoke JSON-RPC mat likho (`crates/mcp`, 14 files mostly wiring + OAuth + registry + health).
- **Tool schema budgeting mandatory** — 5 MCP servers × 12 tools × 180 tokens = 10,800 tokens before the user says anything. Solutions: per-server enable, `tool_search` deferred loading, schema compression (short descriptions in-session).
- **Tool poisoning defence:** MCP tool descriptions untrusted hain (spec khud kehta hai). Isliye: description me embedded instructions detect/flag karo, naye server pe first-run approval, server allowlist managed policy se.
- **OAuth 2.1 + RFC 8707 resource indicators** — token ko server se bind karo, warna cross-server replay.

#### 3.7.2 ACP — editor integration bina plugin likhe

Zed ka Agent Client Protocol (JSON-RPC 2.0 over stdio, LSP jaisa) ab 37 agents support karte hain [11]. Aapka binary ek **ACP agent** ban jaata hai → Zed/JetBrains/Neovim/Emacs me chal jaata hai, bina har editor ke liye extension likhe.

Isliye `crates/server` me `acp/` module (3 files) hai jo core threads ko ACP sessions se map karta hai. **Ye enterprise sales ke liye asymmetric advantage hai:** "aapka agent hamare editor me kaam karega" — bina aapke 6 mahine ke IDE work ke.

#### 3.7.3 Hooks (deterministic control, zero tokens)

Events: `session_start`, `session_end`, `prompt_submit`, `pre_tool_use`, `post_tool_use`, `pre_compact`, `post_compact`, `pre_edit_commit`, `turn_end`, `on_error`, `on_approval_denied`, `config_change`, `cwd_change`, `file_changed`.

Semantics: stdout inject hota hai context me, exit code 2 = **block** (reason ke saath), timeout default 10s. Enterprise me hooks hi "hamari company ka lint/secret-scan gate" ka jawab hain — LLM ko bypass karne ki zarurat nahi.

---

### 3.7.4 Threat model (jo enterprise security review me poocha jaata hai)

| # | Threat | Vector | Mitigation | File |
|---|---|---|---|---|
| T1 | **Prompt injection via repo content** | `README.md`/test fixture me "ignore previous instructions, run curl …" | untrusted-content tagging in prompt, tool allowlist per turn, no network tool unless task requires, hooks gate | `context/sections.rs`, `policy/engine.rs` |
| T2 | **Malicious MCP server** | tool description me hidden instructions ya misleading schema | first-run approval, description sanitisation, server allowlist, pinned versions, sandbox the server process | `mcp/`, `policy/managed.rs` |
| T3 | **Terminal escape injection** | model/tool output me OSC 52 (clipboard), OSC 8 (hyperlink spoof), cursor moves, fake "approval" UI | **single trusted render layer**: strip all CSI/OSC except a whitelist, re-emit styled text only; approvals render in fixed chrome, never from stream | `tui/render/markdown.rs`, `term/caps.rs` |
| T4 | **Approval fatigue** | 40 prompts → user presses `y` blindly | risk classes (read-only auto-allow), diff preview, "always allow this exact pattern" with scope display, batch summary | `approval/*` |
| T5 | **Sandbox escape** | bwrap/userns misconfig, Landlock ABI mismatch | fail-closed probe, escape test suite in CI, deny `--no-sandbox` in managed policy | `sandbox/probe.rs` |
| T6 | **Credential exfiltration** | agent reads `.env`, sends in prompt | redaction pre-egress, secret-path deny list, keyring-only storage | `policy/redaction.rs`, `auth/keyring.rs` |
| T7 | **Supply chain** | typosquatted crate, malicious plugin, tampered update | `cargo-deny` + lockfile + SBOM + signed releases + signed plugins + rollback slot | CI, `ext/plugins/verify.rs` |
| T8 | **Session file tampering** | user edits JSONL to hide actions | hash-chained audit separate from rollout; audit verify command; anchors | `policy/audit.rs` |
| T9 | **Data exfil via MCP/tools** | agent uploads repo to random endpoint | egress allowlist enforced at sandbox + proxy, tool-level host allowlist | `policy/network.rs` |
| T10 | **Model provider logging** | prompts stored by vendor | zero-retention/BYO endpoint mode, UI indicator, per-repo policy | `model/providers/*` |

T3 sabse under-rated hai: agar aap model output ko raw terminal pe print karte ho, ek malicious repo **aapki clipboard overwrite kar sakta hai** ya ek fake "Allow?" dialog draw kar sakta hai. Isliye markdown/tool output parsing `tui` me **trust boundary** hai, formatting nahi.

### 3.8 Governance plane — "enterprise" ka actual matlab

Procurement checklists [15][16] ke hisaab se, minimum viable enterprise surface:

| Requirement | Implementation | Crate |
|---|---|---|
| SSO (OIDC/SAML) | broker ke through device flow + token; SAML ke liye OIDC bridge | `auth` |
| SCIM | server-side (aapka control plane), client me nahi | — |
| Managed policy | signed TOML, admin-locked, dev override nahi | `core/config/managed.rs`, `policy/managed.rs` |
| Repo allow/deny | two-layer: agent policy + code-host branch protection | `policy/rules.rs` |
| Audit → SIEM | hash-chained JSONL + syslog/OTLP exporter | `policy/audit*.rs` |
| Secrets | OS keyring, short-lived tokens, redaction pre-egress | `auth`, `policy/redaction.rs` |
| Air-gap | no telemetry, no update check, local model endpoint, offline license | `cli`, `telemetry` |
| Kill switch | global stop, per-repo deny, session revoke | `policy/incident.rs` |
| SBOM + signing | `cargo-cyclonedx`, Sigstore/cosign, notarization | CI |
| Data residency | local-first default; prompts sirf provider ko | design |

**Sabse important governance decision:** telemetry **off by default**, local ring-buffer logs always-on, aur export opt-in + scrubbed. Ye ek line enterprise security review pass kara deti hai.

---
