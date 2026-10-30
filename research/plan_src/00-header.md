# plan.md — Enterprise Terminal AI Agent in Rust
## Implementation Plan · File-by-File Spec · Parallel Subagent System

| Field | Value |
|---|---|
| **Status** | Planning complete — ready for kickoff (no code written yet) |
| **Product** | `agent` — terminal-native AI coding agent (Claude Code / Codex / Auggie-class) |
| **Language** | 100% Rust, edition 2024, MSRV pinned per release |
| **Scope** | Enterprise v1 (P0–P2) + parallel subagents (P1–P3) |
| **Size** | **308 hand-written source files · ~76,560 LOC** across **19 crates** |
| **Timeline** | 6 engineers × ~14.5 months (P0 skeleton in 15 weeks) |
| **This document** | **Sab kuch ek hi file me** — research + architecture + subagents + delivery + 308-file spec + contracts + quality + enterprise + sources |
| **Date** | 30 October 2026 |

---

## Document map

| § | Section | Kya milega |
|---|---|---|
| **1** | Mission & scope | Product definition, users, non-goals, 12 golden rules, success metrics |
| **2** | Product surface | Commands, modes, slash commands, config keys, exit codes, UX flows |
| **3** | Architecture | Crate map, dependency rules, op/event flow, concurrency, data flow |
| **4** | **Parallel subagent system** | Advanced multi-agent design: DAG, scheduler, leases, isolation, merge gates, metrics |
| **5** | Delivery plan | Phases, 2-week sprints, exit criteria, staffing, dependency order |
| **6** | **File-by-file specification** | 308 files: exactly kya code + kaunse features/acceptance |
| **7** | Cross-cutting contracts | Event/op catalogue, tool catalogue, policy grammar, DB schema, error codes |
| **8** | Quality bar | Test matrix, eval suite, perf budgets, CI stages, release gates |
| **9** | Enterprise readiness | 12 pillars → files mapping, threat model, deployment modes |
| **10** | Execution | DoD, ownership/RACI, risk register, backlog, traceability |

## Padhne ka tareeka

| Aap kaun ho | Padho |
|---|---|
| Founder / decision maker | §1, §5, §10 (risk register) |
| Tech lead | Sab, especially §3, §4, §6, §8 |
| Implementer (day 1) | §6 (apni crate), §7 (contracts), §8 (DoD/tests), phir §5 (apna sprint) |
| Reviewer / architect | §3 invariants, §4, §8 budgets, §9 checklist |

## Teen non-negotiables (poore plan ka summary)

1. **Event-sourced core** — UI, server, replay, audit sab ek hi event stream ke projections hain. Isse resume, testing aur multi-surface free me milte hain.
2. **Context economy first-class** — token ledger + cache-aware ordering + 5-stage compaction. Yahi 3–10× cost delta aur "agent got dumb" complaints ka fix hai.
3. **Deny-first safety** — policy → approval → OS sandbox → resource limits → audit, paanch independent layers. Sandbox P0 me hai, retrofit me nahi.

---
