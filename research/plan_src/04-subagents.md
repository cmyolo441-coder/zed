## 4. Parallel subagent system (advanced)

> Ye section poore plan ka sabse high-risk aur sabse high-differentiation part hai. Isliye ise **deterministic orchestration + isolated workers + gated merge** ki tarah design kiya gaya hai — "kai agents ko chat karwa do" ki tarah nahi.

### 4.1 Pehle honest baat: parallel kab fayda deta hai, kab nuksan

Public data do cheezein clear kehta hai:

- **Agent teams me token cost ~7× badh jaata hai** ek normal session ke muqable (Claude Code analysis), aur harness badalne se tokens/solved-task me **40× tak** farq padta hai.
- Same model, different scaffold: **52.1% vs 57.8%**, par input tokens **256.9M vs 3.9M** (Terminal-Bench 2.0).

Toh parallel ka rule:

| Situation | Parallel? | Kyun |
|---|---|---|
| 5 independent files/modules me same-pattern change | ✅ haan | write sets disjoint, coordination ~0 |
| Repo-wide "find all call sites" + "fix" | ✅ haan (explore parallel, fix serial) | read-only fan-out sasta hai |
| Ek bug jiske 6 possible causes hain | ✅ haan (4 hypothesis workers, 1 verifier) | search width, not work width |
| Multi-stage feature (schema → API → UI) | ⚠️ limited | dependencies hain; DAG chain banao, fan-out nahi |
| Ek file me bade refactor | ❌ nahi | merge conflicts guaranteed; single worker better |
| Design decision / architecture | ❌ nahi | parallel opinions = confusion; 1 planner + 1 critic |

**Product decision:** parallel mode **opt-in** (`--parallel`, `/board`), default serial. Har run pe orchestrator cost-vs-time report deta hai, aur agar speedup < 1.3× ho to CLI warning deta hai ("parallel ne 2.4× tokens liye, 1.1× speedup — serial try karo").

### 4.2 Topology — 5 roles, 1 deterministic core

```
                    ┌──────────────────────────────┐
   user prompt ───► │  PLANNER (LLM, 1 call)        │  proposes DAG (JSON, schema-validated)
                    └──────────────┬───────────────┘
                                   │ plan
                    ┌──────────────▼───────────────┐
                    │  SUPERVISOR (deterministic)   │  ← koi LLM nahi; pure Rust
                    │  scheduler · leases · budgets │
                    └───┬──────────┬───────────┬───┘
             ┌──────────┘          │           └────────────┐
             ▼                     ▼                        ▼
      ┌────────────┐        ┌────────────┐           ┌────────────┐
      │ WORKER 1   │        │ WORKER 2   │    ...    │ WORKER N   │
      │ worktree   │        │ worktree   │           │ worktree   │
      │ own budget │        │ own budget │           │ own budget │
      └──────┬─────┘        └──────┬─────┘           └──────┬─────┘
             └──────────┬──────────┴────────────────────────┘
                        ▼  WorkerReport (summary + patch ref + evidence)
              ┌───────────────────────┐
              │ MERGE QUEUE (serial)  │  3-way merge, rebase onto tip
              └──────────┬────────────┘
                         ▼
              ┌───────────────────────┐
              │ VERIFY GATE           │  build + affected tests + lint
              └──────────┬────────────┘
                    pass │ fail → quarantine (worker ko evidence wapas)
                         ▼
                 consolidated diff → user review
```

**Design rule:** planner *proposes*, supervisor *decides*. Planner LLM kabhi scheduling, file locks, budgets ya merge order decide nahi karta — warna non-determinism aur deadlocks aate hain.

### 4.3 Task DAG — node schema

Planner ko ek **strict JSON schema** se output dena hota hai (invalid = 1 retry, phir serial fallback):

```json
{
  "goal": "Add rate limiting to the API layer",
  "nodes": [
    {
      "id": "n1",
      "kind": "implement",              // explore | implement | test | review | doc | fix
      "goal": "Add token bucket middleware in src/api/middleware/",
      "write_set": ["src/api/middleware/**", "src/api/mod.rs"],
      "read_set": ["src/config/**", "Cargo.toml"],
      "deps": [],
      "acceptance": ["cargo test api::middleware"],
      "budget_tokens": 90000,
      "model_tier": "strong"
    },
    {
      "id": "n2",
      "kind": "test",
      "goal": "Add integration tests for 429 + header behaviour",
      "write_set": ["tests/api/rate_limit.rs"],
      "read_set": ["src/api/**"],
      "deps": ["n1"],
      "acceptance": ["cargo test --test api"],
      "budget_tokens": 40000,
      "model_tier": "cheap"
    },
    {
      "id": "n3",
      "kind": "review",
      "goal": "Security review: bypass paths, clock skew, config validation",
      "write_set": [],
      "read_set": ["src/api/**", "tests/api/**"],
      "deps": ["n1", "n2"],
      "budget_tokens": 30000,
      "model_tier": "strong"
    }
  ]
}
```

Validation rules (supervisor me, LLM me nahi):

1. **Cycle** = reject. **Self-dep** = reject. `deps` sirf pehle declare kiye gaye ids pe.
2. **Width cap** — `max_workers` (default 4) se zyada ready nodes → queue.
3. **Node count cap** — default 6, hard cap 12. Isse zyada = plan ko chhota karo (task decomposition fail).
4. **Write-set disjointness check** — ready nodes ka intersection empty hona chahiye, warna serial + warning.
5. **Root containment** — har path workspace root ke andar, `path_guard` se validate.
6. **Acceptance mandatory** — `implement`/`fix`/`test` node ke paas kam se kam ek acceptance command.
7. **Budget sum** ≤ task budget × 1.2 (planning overhead).

### 4.4 Isolation — 4 levels, default L2

| Level | Workspace | Context | Policy | Kiske liye |
|---|---|---|---|---|
| **L0** | shared, read-only | own | parent | explore/research workers (safest, cheapest) |
| **L1** | shared, leased writes | own | parent | same-file sequential tasks (no parallel) |
| **L2** *(default)* | **git worktree per worker** | own | parent + narrowed | implement/test workers |
| **L3** | worktree + container/VM | own | narrowed + no net | untrusted repo, third-party MCP, "run this sketchy branch" |

Isolation ka matlab sirf files nahi:

- **Context isolation** — worker ko parent transcript nahi milta; sirf `goal + constraints + relevant files + acceptance`. Return sirf summary (300–600 tokens). Isse parent context pollution nahi hoti (research: subagent summary isolation).
- **Policy isolation** — worker ke paas parent se **kam** permissions, kabhi zyada nahi. `net` default deny, `git commit` default deny, `git push` **always** deny.
- **Budget isolation** — per-worker token/time/cost cap; exceed = worker stopped with partial report, parent decide kare.
- **Process isolation** — worker ke child processes uska cancel tree follow karein; worker die = uske processes bhi die.
- **Audit isolation** — har action me `worker_id` + `parent_turn` (attribution chain §9).

### 4.5 Scheduler — deterministic, fair, adaptive

Koi LLM nahi. Pure function: `(graph_state, budgets, limits) → next_node`.

```
loop:
  ready   = nodes whose deps are all Done and leases grantable
  ordered = sort by (priority_class, dag_depth_asc, created_seq)   // deterministic
  while running < max_workers and ordered not empty:
      n = ordered.pop_front()
      if not leases.try_acquire(n.write_set): requeue(n); continue
      if budget_pool.remaining() < n.reserve: wait_for_budget(); break
      spawn_worker(n); running += 1
  wait_for_any(worker_exit | budget_event | cancel)

on 429/overloaded(provider):
      concurrency = max(1, concurrency / 2)        // AIMD: multiplicative decrease
on 5 consecutive successes:
      concurrency = min(max_workers, concurrency + 1)
```

Rules:
- **Priority classes:** `verify` > `review` > `implement` > `test` > `explore`. (Verification never starves — warna poora parallel run waste ho jaata hai.)
- **Aging:** 2 minute wait pe priority ek class upar (starvation zero).
- **Determinism:** `created_seq` + seeded tie-break; same plan → same schedule → replayable.
- **Adaptive concurrency:** `min(cpu_cores/2, provider_rpm/6, max_workers)` se start; 429 pe half; success pe +1.
- **Cost ceiling:** run-level `max_cost`; 80% pe warning event, 100% pe stop new spawns (running workers finish).
- **Backpressure:** provider queue depth > threshold → naye workers spawn nahi; board pe "waiting on rate limit" dikhe.

### 4.6 File leases (conflict ka asli solution)

Merge conflicts ko *baad me* solve karna costly hai; **pehle hi rokna** sasta hai.

- Worker declare karta hai `write_set` (globs). Supervisor usse expand karke **lease** deta hai.
- Do leases overlap = second worker **queue** me (spawn hi nahi hota) — token waste zero.
- **Deadlock-free:** multiple leases ek saath chahiye to sorted path order me acquire (classic lock ordering).
- **Heartbeat:** har 30 s; 2 miss = stale.
- **Expiry/steal:** 5 min no progress → lease revoke + worker cancel → requeue with note ("previous attempt stalled at X").
- **Dynamic expansion:** worker naya file chhuना chahe to `request_lease` — grant ya wait; **bina lease ke write = policy deny** (tool-level enforcement, prompt-level nahi).

Ye wahi mechanism hai jo "worker A ne B ki file overwrite kar di" class ke bugs khatam karta hai.

### 4.7 Worker prompt contract (har worker ko same shape)

| Section | Content | Why |
|---|---|---|
| Role | "You are an implementation worker. You own only these paths: …" | scope clarity |
| Goal | node goal + parent goal (1 para) | alignment |
| Constraints | no network, no policy changes, no git push, no new deps without approval | safety |
| Context | relevant files (repo-map slice), interfaces, conventions | no transcript dump |
| Acceptance | exact commands + expected outcome | verifiable done |
| Output | strict `WorkerReport` JSON + patch via normal edit tools | machine-readable return |

**Worker ke paas kya nahi hota:** parent chat, other workers ke reports, approval rights, policy edit, memory write, MCP config changes.

### 4.8 WorkerReport — return protocol

```json
{
  "node_id": "n1",
  "status": "done",                  // done | partial | blocked | failed
  "summary": "Added TokenBucket middleware + config parsing; 2 files, 140 LOC.",
  "files_changed": ["src/api/middleware/rate_limit.rs", "src/api/mod.rs"],
  "commits": ["worktree:3f9a1c2"],
  "tests": [{"cmd": "cargo test api::middleware", "result": "pass", "duration_ms": 41200}],
  "acceptance_met": true,
  "confidence": 0.8,
  "blockers": [],
  "evidence": {"log_ref": "worker-n1.log", "diff_stat": "2 files, +140 -3"},
  "cost": {"tokens_in": 61200, "tokens_out": 8400, "usd_micros": 184000}
}
```

Validation: schema + `files_changed ⊆ write_set` (violation = report rejected, worker quarantined). Parent ko sirf `summary` jaata hai; baaki board/detail me.

### 4.9 Merge + verify pipeline

1. **Queue, not race** — merge strictly sequential (ek time pe ek), `dag` order me nahi, **completion order + dependency order** me (deps pehle).
2. **Rebase onto tip** — worker branch ko current tip pe rebase; conflict = `merge.rs` 3-way; auto-resolve only whitespace-level, warna worker ko retry ("conflict in X, rebase your change").
3. **Verify gate** — `build` + **affected tests** (file → test mapping via repo map) + `lint`. Timeout 10 min. Flaky retry ek baar.
4. **Quarantine** — fail pe merge revert + worker ko evidence; retry budget 1; doosri fail = node `failed`, plan partially complete.
5. **Consolidated diff** — user ko ek hi diff review milta hai, per-worker commits squashed with attribution trailers.
6. **Never push/PR automatically** — v1 me sirf local branch + diff. PR creation policy-gated (P3).

### 4.10 Failure taxonomy aur recovery

| Failure | Detection | Recovery | Escalation |
|---|---|---|---|
| Wrong assumption | acceptance fails, self-report `blocked` | 1 retry with failure evidence | node failed → parent |
| Test fail | gate | worker retry (max 1) | quarantine + report |
| Merge conflict | rebase | 1 rebase-retry | re-plan node (serial) |
| Budget exhausted | budget ledger | partial report return | parent decides continue/stop |
| Tool denied by policy | policy engine | worker told; alternate approach | if repeated, node blocked |
| Infinite loop | limits (identical call ×3) | worker cancel | requeue with tighter goal |
| Environment break | gate infra error (disk/net) | run aborted, state saved | resume support |
| Worker crash/panic | task join error | requeue once | node failed |
| Provider outage | gateway | AIMD + failover | run paused, resumable |

### 4.11 Budgets, rate limits, cost control

| Budget | Default | Scope | Enforcement |
|---|---|---|---|
| `task_budget_tokens` | 400k | whole parallel run | pool; spawn blocked at 100% |
| `worker_budget_tokens` | 120k | per worker | hard stop, partial report |
| `max_cost_usd` | $3.00 | run | 80% warn, 100% no new spawns |
| `max_workers` | 4 | concurrency | scheduler |
| `max_wall_time` | 30 min | run | cancel tree |
| `provider_rpm` | config | per provider | token bucket (shared with main turn) |

**Model tiering:** planner + verifier = strong model; implement = strong; test/explore/doc = cheap. Measured effect: ~35–50% cost reduction vs all-strong (to be validated in eval).

### 4.12 Safety — parallel mode me 6 extra rules

1. Workers **approve nahi kar sakte** — koi bhi mutating tool call parent policy + (agar ask mode) user ke through jaata hai, **ya** pre-approved `auto-edit` scope me hota hai. Silent escalation impossible.
2. Worker ke paas `git push`, `policy edit`, `mcp add`, `config write`, `memory write` tools **by default nahi**.
3. Network default **deny** (L3 me definitively).
4. Har worker apne sandbox policy ke saath chalta hai; parent se broader nahi.
5. **Kill switch** (`policy/incident.rs`) — ek command me saare workers cancel + worktrees cleanup + audit entry.
6. Parallel run ka **poora transcript sidecar** me; parent context me sirf summaries (prompt-injection blast radius chhota).

### 4.13 Config keys (orchestration)

```toml
[agents]
enabled        = false        # opt-in
max_workers    = 4
max_nodes      = 6
isolation      = "worktree"   # none | worktree | container
merge_policy   = "verify"     # manual | verify | auto
plan_model     = "strong"
worker_model   = "strong"
cheap_model    = "cheap"
task_budget    = 400000       # tokens
worker_budget  = 120000
max_cost_usd   = 3.00
max_wall_time  = "30m"
require_confirm = true        # user plan dekhe aur approve kare
```

### 4.14 Observability + TUI board

Board state (`orchestrator/board_model.rs`) → `tui/render/board.rs`:

```
┌─ workers ──────────────────────────────────────────────┐
│ n1 implement  ● running   62%  $0.18   files: 2        │
│ n2 test       ○ blocked   dep n1                        │
│ n3 review     · queued                                  │
├─ merge queue ──────────────────────────────────────────┤
│ 1/3 merged · conflict 0 · verify 2 pass 0 fail          │
├─ budget ───────────────────────────────────────────────┤
│ tokens 214k/400k · cost $0.61/$3.00 · 4m12s elapsed     │
└────────────────────────────────────────────────────────┘
```

Metrics (telemetry crate): `orch_workers_active`, `orch_node_duration_ms`, `orch_tokens_per_node`, `orch_merge_conflicts_total`, `orch_verify_failures_total`, `orch_speedup_ratio`, `orch_rework_rate`.

### 4.15 Success criteria (parallel mode ship karne se pehle)

| Metric | Target | Test |
|---|---:|---|
| Speedup on 5 independent tasks | ≥ 1.8× vs serial wall time | benchmark repo, 3 runs |
| Merge conflict rate | < 10% | 30-task corpus |
| Rework rate (worker redo) | < 20% | same corpus |
| Token overhead vs serial | < 2.2× | measured |
| Determinism | same plan → same merge order | replay test |
| Zero cross-worker file clobber | 0 incidents | lease stress test (50 runs) |
| Kill switch completeness | 0 orphan worktrees/processes | chaos test |

Agar ye numbers miss ho, feature **default off** rehta hai — ship nahi hota. (Yahi honest engineering hai.)

### 4.16 File map — kaunsi file kya karti hai

| File | Role in this system |
|---|---|
| `orchestrator/src/lib.rs` | `run(plan)` facade, budgets/cancel wiring |
| `orchestrator/src/dag.rs` | node/edge model, topo order, ready-set, validation rules |
| `orchestrator/src/scheduler.rs` | deterministic queue, priority, AIMD, budget pool |
| `orchestrator/src/leases.rs` | write-set leases, ordering, heartbeat, steal |
| `orchestrator/src/worker.rs` | worktree + session + narrowed policy + prompt + cancel |
| `orchestrator/src/report.rs` | report schema/validation + summary extraction |
| `orchestrator/src/merge.rs` | serial merge queue, rebase, conflict policy, revert |
| `orchestrator/src/verify_gate.rs` | build/test/lint gate, quarantine, retry |
| `orchestrator/src/board_model.rs` | live board state for TUI/server |
| `core/src/subagent/*` | single-subagent path (legacy/simple), profiles |
| `tools/src/task.rs` | model-facing `task` tool → orchestrator (parallel) |
| `git/src/worktree.rs` | worktree create/remove/GC |
| `state/src/migrations/0005_approvals.sql` | worker leases + approvals persistence |
| `tui/src/render/board.rs` | board rendering |
| `policy/src/attribution.rs` | worker → parent audit chain |

### 4.17 Anti-patterns (jo hum *nahi* karenge)

1. **LLM as scheduler** — non-deterministic, deadlocks, unreproducible costs.
2. **Workers chatting with each other** — context explosion; coordination only via artifacts/reports.
3. **Shared workspace writes** — clobber risk; always leases + worktrees.
4. **Unbounded fan-out** ("spawn 20 agents") — 7× token cost, 40× variance; caps mandatory.
5. **Merge without gates** — broken mainline, lost trust, worse than serial.
6. **Approval amplification** — 4 workers × 10 prompts = fatigue → auto-approve → breach. Isliye workers ko pre-approved narrow scope.
7. **Same model for all roles** — cost blowup; tiering mandatory.
8. **No kill switch / no leases / no budgets** — teeno ke bina parallel mode production me nahi jaata.
