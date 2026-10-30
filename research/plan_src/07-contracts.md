## 7. Cross-cutting contracts (jo sab crates share karte hain)

Ye section "single source of truth" hai. Yahan jo shape decide hoga, wo 308 files me repeat hoga — isliye ise week 1 me freeze karna zaroori hai.

### 7.1 Op catalogue (client → agent)

| Op | Payload | Ack | Notes |
|---|---|---|---|
| `Submit` | `{ prompt, attachments[], mode?, model? }` | `TurnId` | idempotent by `op_id` |
| `Steer` | `{ turn_id, text }` | — | next tool boundary pe inject |
| `Interrupt` | `{ turn_id }` | `TurnId` | cancel tree, children kill |
| `Approve` | `{ request_id, decision, remember? }` | — | request_id single-use |
| `Compact` | `{ turn_id?, focus? }` | — | manual compaction |
| `SetMode` | `{ mode }` | — | audited |
| `SpawnWorkers` | `{ plan }` | `RunId` | parallel mode (§4) |
| `CancelWorkers` | `{ run_id, worker_id? }` | — | kill switch path |
| `Shutdown` | `{}` | — | graceful, flush state |

### 7.2 Event catalogue (agent → client) — 30 events, 6 families

| Family | Events |
|---|---|
| **Turn** | `TurnStarted`, `TurnCompleted`, `TurnFailed`, `TurnCancelled`, `StateChanged` |
| **Content** | `TextDelta`, `ThinkingDelta`, `TextCompleted`, `ContentReplaced` (compaction) |
| **Tools** | `ToolRequested`, `ToolApprovalNeeded`, `ToolApprovalDecided`, `ToolStarted`, `ToolOutputDelta`, `ToolCompleted`, `ToolFailed`, `ToolDenied` |
| **Context** | `ContextAssembled` (sizes only), `CompactStarted`, `CompactStage`, `CompactCompleted`, `MemoryLoaded` |
| **Workers** | `WorkerSpawned`, `WorkerProgress`, `WorkerReported`, `WorkerCancelled`, `MergeQueued`, `MergeCompleted`, `VerifyResult` |
| **System** | `UsageUpdated`, `CostUpdated`, `Warning`, `Error`, `ConfigChanged`, `SessionResumed` |

Rules: every event has `{ id, thread_id, turn_id?, ts, schema_version }`; events are **additive-only**; UI must ignore unknown variants (forward compat).

### 7.3 Tool catalogue (v1)

| Tool | Args (summary) | Risk | Approval in `ask` | Result shape |
|---|---|---|---|---|
| `read_file` | path, offset, limit | read | no | numbered lines + truncation hint |
| `write_file` | path, content | mutate | yes | bytes written, backup ref |
| `edit_file` | path, old, new, occurrence? | mutate | yes | diff + match count |
| `apply_patch` | patch (3 formats), dry_run? | mutate | yes | per-file hunks applied/failed |
| `glob` | pattern, sort?, limit? | read | no | paths + count |
| `grep` | pattern, glob?, context?, max? | read | no | grouped matches |
| `list_dir` | path, depth? | read | no | tree + sizes |
| `shell` | command, cwd?, timeout?, session? | **exec** | yes (mode-dependent*) | exit code, stdout/stderr (capped) |
| `plan` | items[], update? | state | no | plan ids + status |
| `task` | goal, profile, parallel? | spawn | yes (first time) | WorkerReport summary |
| `web_fetch` | url, max_bytes? | net | yes | text + title + bytes |
| `ask_user` | question, options[] | meta | n/a | chosen option |
| `git_status` | — | read | no | branch, dirty files |
| `git_diff` | scope, path? | read | no | unified diff (capped) |
| `diagnostics` | path? | read | no | LSP diagnostics |
| `mcp__*` | server-defined | server-declared | per policy | server-defined |

\* `shell` read-only commands (allowlisted) auto-approve in `ask`; mutating/destructive prompt.

### 7.4 Policy pattern grammar

```
pattern   := tool "(" selector ")"
tool      := "shell" | "read" | "write" | "net" | "mcp" | "*"
selector  := path-glob | argv-pattern | host | tool-name
argv-pattern := token (" " token)*           # token "=" literal, "*" = any single, "**" = rest
examples
  shell(git status)            # exact argv match
  shell(cargo test * )         # fixed prefix + one arg
  shell(rm -rf **)             # destructive prefix (deny)
  read(src/**)                 # path glob within workspace
  net(api.anthropic.com)       # host
  mcp(github.*)                # MCP server tool
```

Evaluation order: **deny → ask → allow** (deny-first). Specificity tie-break: longer literal prefix wins. Every decision returns `explain()` string: `denied by rule #3 shell(rm -rf **) at .agent/config.toml:12`.

### 7.5 Persistence schema (SQLite, 5 migrations)

| Table | Key columns | Purpose |
|---|---|---|
| `threads` | id, parent_id, cwd, model, created, archived | session tree |
| `turns` | id, thread_id, status, started, ended, cost_micros | turn ledger |
| `messages` | id, turn_id, role, parts_json, tokens | ordered history |
| `tool_calls` | id, turn_id, name, args_hash, decision, exit, duration_ms | audit-able actions |
| `workers` | id, run_id, node_id, state, worktree, tokens, cost | parallel runs |
| `leases` | run_id, worker_id, paths_hash, acquired, expires | conflict record |
| `usage` | turn_id, model, in/out/cache/reasoning | cost attribution |
| `approvals` | id, pattern, scope, expires | session grants |
| `audit_anchors` | seq, hash, ts | tamper evidence |
| `messages_fts` | rowid, text | search |

JSONL rollout line (append-only truth):

```json
{"v":1,"ts":"2026-10-30T09:12:44.118Z","thread":"01JB...","turn":"01JB...","type":"ToolCompleted","data":{...},"checksum":"blake3:..."}
```

### 7.6 Error taxonomy (user-visible codes)

| Code | Class | Example | Recovery |
|---|---|---|---|
| `E1xxx` | config | unknown key, invalid mode | `agent config explain`, doctor |
| `E2xxx` | provider | 429, auth, model not found | retry hint, `auth login` |
| `E3xxx` | tool | path denied, patch mismatch, timeout | model-facing hint |
| `E4xxx` | sandbox | unavailable, denied path, net blocked | `--no-sandbox`, doctor |
| `E5xxx` | state | disk full, db locked, corrupt rollout | resume, prune |
| `E6xxx` | protocol | version mismatch, bad frame | upgrade hint |
| `E7xxx` | orchestration | cycle, budget exhausted, conflict | re-plan, reduce nodes |

Har error me: code, one-line message, **next action**, aur `--verbose` me detail. Support ko code se search karna aasan.

### 7.7 Metric names (stable, documented)

`turn_duration_ms`, `ttft_ms`, `tokens_in/out/cache_read`, `tool_calls_total{tool,status}`, `tool_duration_ms`, `compact_stage_tokens_saved{stage}`, `context_tokens{section}`, `cache_hit_ratio`, `sandbox_denials_total`, `approval_latency_ms`, `worker_active`, `merge_conflicts_total`, `verify_failures_total`, `crash_total`.

Cardinality rule: labels sirf bounded sets (tool name, status, stage). Paths/prompts **kabhi** label me nahi.
