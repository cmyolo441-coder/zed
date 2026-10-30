## 9. Enterprise readiness — 12 pillars ka file-level mapping

Enterprise sale model quality se nahi, **controls** se hoti hai. Har pillar ka ek owner crate hai, aur ek evidence jo auditor ko dikha sakte ho.

| # | Pillar | Implementing files | Evidence auditor ko |
|---|---|---|---|
| 1 | **Identity** (OIDC/SAML device flow, CI service accounts) | `auth/{lib,oauth_pkce,device_code,token_store,service_account}.rs`, `cli/commands/auth.rs` | `agent auth status` output, token scope, expiry |
| 2 | **Policy-as-code** (admin-locked) | `core/config/managed.rs`, `policy/managed.rs`, `policy/rules.rs` | signed managed policy file + `config explain` showing managed source |
| 3 | **Permission plane** (deny-first, 5 modes) | `policy/{engine,modes,classifier}.rs`, `core/approval/*` | decision log with rule id + `explain()` string |
| 4 | **OS sandbox** (3 OS) | `sandbox/*` (12 files) | `agent doctor` sandbox matrix + escape test results |
| 5 | **Egress control** | `sandbox/network.rs`, `policy/network.rs`, `tools/net/fetch.rs` | blocked-host audit entries, allowlist diff |
| 6 | **Audit trail** (hash-chained) | `policy/{audit,audit_export}.rs`, `core/events.rs`, `state/rollout.rs` | `agent audit verify` + SIEM screenshot |
| 7 | **Data governance** (local-first, redaction, retention) | `policy/redaction.rs`, `telemetry/privacy.rs`, `state/gc.rs` | redaction test corpus, retention config, `telemetry preview` |
| 8 | **Air-gap / BYO model** | `model/providers/{bedrock,vertex,ollama,custom}.rs`, `cli/commands/doctor.rs` | offline demo, no phone-home (network trace) |
| 9 | **Supply chain** | CI (`cargo-deny`, SBOM, cosign), `cli/commands/update.rs`, `ext/plugins/verify.rs` | signed binary + SBOM + provenance attestation |
| 10 | **Observability** | `telemetry/*`, `cli/commands/doctor.rs` | OTLP trace screenshot, support bundle (redacted) |
| 11 | **Extensibility without forking** | `ext/*`, `mcp/*`, `server/acp/*` | custom skill + hook + MCP server demo |
| 12 | **Operability** (CI mode, exit codes, replay) | `cli/commands/exec.rs`, `state/replay.rs`, `protocol/*` | CI pipeline YAML + JSONL replay of a real incident |

### 9.1 Deployment modes

| Mode | Model endpoint | Telemetry | Update | Kiske liye |
|---|---|---|---|---|
| **Local dev** (default) | vendor API | off | auto (signed) | individuals |
| **Managed corporate** | vendor via proxy | OTLP to SIEM (opt-in) | pinned/off | enterprises |
| **VPC / self-hosted gateway** | internal proxy (`providers/custom.rs`) | SIEM | manual | regulated |
| **Air-gapped** | local vLLM/Ollama | none (local logs only) | offline bundle | defense/health |

Air-gap requirements (testable): no DNS on start, no update check, offline license file, no crash reporter, `--no-network` flag that **hard-fails** any outbound attempt (audited).

### 9.2 Threat model summary (detailed table: blueprint §3.7.4)

| # | Threat | Primary defence | File |
|---|---|---|---|
| T1 | Prompt injection via repo content | untrusted-content tagging, tool allowlist per turn | `context/sections.rs` |
| T2 | Malicious MCP server / tool description | first-run approval, description sanitisation, pinned versions | `mcp/*`, `policy/managed.rs` |
| T3 | **Terminal escape injection** (OSC 52 clipboard, fake approval UI) | single trusted render layer, whitelist CSI/OSC, fixed approval chrome | `tui/render/markdown.rs`, `term/caps.rs` |
| T4 | Approval fatigue → blind `y` | risk classes, default deny focus, scope preview, batch summaries | `approval/*`, `tui/render/approval.rs` |
| T5 | Sandbox escape | fail-closed probe, escape suite, managed `--no-sandbox` ban | `sandbox/probe.rs` |
| T6 | Credential exfiltration | redaction pre-egress, secret-path deny list, keyring only | `policy/redaction.rs`, `auth/keyring.rs` |
| T7 | Supply chain (crate/plugin/update) | `cargo-deny`, lockfile, SBOM, signing, rollback slot | CI, `ext/plugins/verify.rs` |
| T8 | Audit tampering | hash chain + anchors, separate from rollout | `policy/audit.rs` |
| T9 | Data exfil via tools/MCP | egress allowlist at sandbox + tool host allowlist | `policy/network.rs` |
| T10 | Provider-side retention | zero-retention mode, BYO endpoint, per-repo policy | `model/providers/*` |

### 9.3 Compliance evidence kit (SOC 2-ready)

Ye cheezein **product ke andar** honi chahiye, docs me nahi:

1. **Change evidence** — signed releases, SBOM, reproducible hash, changelog with security section.
2. **Access evidence** — `auth status` per session, SSO group mapping, service-account inventory.
3. **Action evidence** — `audit verify` + exported JSONL per repo/date range.
4. **Control evidence** — managed policy file + `config explain` showing locked values, `doctor --json`.
5. **Incident evidence** — kill switch audit entry, session revoke log, support bundle (redacted).

### 9.4 Security review demo script (15 minutes, jo buyer ko chahiye)

1. `agent doctor --json` → sandbox backends, caps, config provenance.
2. Repo me prompt-injection attempt → agent **refuses** (T1) aur rule id dikhata hai.
3. `rm -rf` request → policy deny, exit 3, audit entry with rule + actor.
4. Sandboxed command → `bwrap` proof (`/proc/self/status` output, no net).
5. `agent audit verify` → chain valid; ek byte mutate → verify fail (tamper demo).
6. `telemetry preview` → exactly kya bhejta (none by default).
7. Parallel run on 5 tasks → board, leases (no clobber), merge gate, consolidated diff.
8. Air-gap mode: `--no-network` + local model → task completes, network attempts logged as denied.

**Rule:** agar ye 8 steps demo nahi ho sakte, product "enterprise-ready" nahi hai — chahe features kitne bhi hon.
