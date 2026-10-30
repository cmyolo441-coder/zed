# -*- coding: utf-8 -*-
"""plan.md spec: git, telemetry, testkit."""

SPEC = {}

SPEC["git"] = [
    ("src/lib.rs",
     "repo discovery (walk up), worktree detection, facade over gix + git CLI fallback",
     "bare repo / submodule / worktree edge cases handled"),
    ("src/repo.rs",
     "open repo, branch/HEAD/status summary, ahead-behind, dirty file list for prompt",
     "summary < 50 ms; huge repo me limited scope"),
    ("src/diff.rs",
     "working tree/staged/commit-range diff extraction, numstat, per-file hunks",
     "binary detect; 5 MB diff cap with file-level summary"),
    ("src/commit.rs",
     "commit creation: staged path selection, message, trailer, signing passthrough",
     "never commits unrequested files; trailer me session id"),
    ("src/worktree.rs",
     "create/remove ephemeral worktrees, branch naming, lockfile, cleanup on crash",
     "10 workers = 10 worktrees; disk cap + GC"),
    ("src/checkpoint.rs",
     "shadow-git checkpoint per turn: tree hash, restore, list, prune",
     "undo one turn < 300 ms; restore conflict-free (isolated refs)"),
]

SPEC["telemetry"] = [
    ("src/lib.rs",
     "init/guard: subscriber install, flush on exit, panic capture hook",
     "crash pe last 200 log lines preserved"),
    ("src/tracing_setup.rs",
     "layered tracing: rolling file (JSONL), stderr (TTY-aware), env filters, sampling",
     "log volume cap; secrets never logged (redaction layer)"),
    ("src/otel.rs",
     "OTLP spans: turn → model call → tool → worker; attributes schema; batched export",
     "export failure = non-blocking; cardinality controlled"),
    ("src/metrics.rs",
     "counters/histograms: latency, tokens, tool success, cache hit, worker parallelism",
     "metric names stable/documented; local aggregation for `/cost`"),
    ("src/privacy.rs",
     "scrubber: paths, prompts, hostnames, emails; opt-in surface; preview command",
     "default off; `telemetry preview` me exact payload dikhe"),
    ("src/support_bundle.rs",
     "one-command bundle: logs, config (redacted), doctor output, last session metadata",
     "user ko preview; size < 5 MB; no code content"),
]

SPEC["testkit"] = [
    ("src/lib.rs",
     "test harness exports: `TestAgent`, `TestRepo`, `assert_events`",
     "ek line me agent + mock model + temp repo"),
    ("src/fixtures.rs",
     "temp workspaces, sample polyglot repo, permission/policy fixtures, worker plans",
     "fixtures deterministic; parallel tests safe (unique temp dirs)"),
    ("src/mock_provider.rs",
     "in-process HTTP mock: scripted SSE, fragmentation, delays, 429/500, usage lies",
     "zero network; millisecond-fast tests; recorded fixtures replay"),
    ("src/golden.rs",
     "golden transcript comparison, event-diff output, insta integration, update flow",
     "diff readable (line-level); `INSTA_UPDATE=always` documented"),
]
