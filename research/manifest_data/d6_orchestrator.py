# -*- coding: utf-8 -*-
"""Manifest data: parallel subagent orchestration crate (added on request)."""

DATA = {}

DATA["orchestrator"] = {
    "desc": "Advanced parallel subagents: DAG planning, file leases, deterministic scheduler, merge + verify gates.",
    "files": [
        ("src/lib.rs", "Facade: Orchestrator::run(plan) with config, budgets and cancellation handle", 90, "P2"),
        ("src/dag.rs", "Task graph: nodes, edges, topo order, cycle detection, ready-set, critical path", 280, "P1"),
        ("src/scheduler.rs", "Deterministic ready-queue: caps, priority, fairness, adaptive concurrency, budget pool", 420, "P2"),
        ("src/leases.rs", "File write-set leases: conflict prevention, deadlock-free ordering, steal/expire", 220, "P2"),
        ("src/worker.rs", "Worker lifecycle: worktree + session + narrowed policy + prompt contract + cancel", 380, "P2"),
        ("src/report.rs", "Structured worker report schema, validation, parent-facing summary extraction", 240, "P1"),
        ("src/merge.rs", "Sequential rebase merge queue: 3-way merge, conflict policy, revert", 360, "P3"),
        ("src/verify_gate.rs", "Per-merge gate: build/test/lint evidence, quarantine on failure", 280, "P3"),
        ("src/board_model.rs", "State model for the live board (worker states, progress, tokens, cost)", 220, "P2"),
    ],
}
