# -*- coding: utf-8 -*-
"""Manifest data: policy/security, auth, extensibility, telemetry, git, testkit."""

DATA = {}

DATA["policy"] = {
    "desc": "The control plane: deny-first rules, risk classification, redaction, tamper-evident audit.",
    "files": [
        ("src/lib.rs", "Policy facade used by tools, approvals and server", 100, "P0"),
        ("src/rules.rs", "Rule model + glob/specifier matching (path, command, host, tool)", 340, "P0"),
        ("src/engine.rs", "Deny-first evaluation order, specificity tie-breaks, explain()", 320, "P0"),
        ("src/modes.rs", "Approval modes: read-only, ask, auto-edit, plan, bypass + transitions", 240, "P0"),
        ("src/classifier.rs", "Heuristic command risk scoring; optional model tie-breaker", 300, "P2"),
        ("src/redaction.rs", "Secret/PII scrubbing before any bytes leave the machine", 300, "P0"),
        ("src/audit.rs", "Append-only hash-chained audit log with periodic anchors", 320, "P1"),
        ("src/audit_export.rs", "SIEM sinks: syslog, OTLP logs, JSONL file rotation", 240, "P2"),
        ("src/managed.rs", "Admin-locked managed policy merge + integrity check", 260, "P2"),
        ("src/network.rs", "Egress allowlist evaluation and proxy decision", 220, "P2"),
        ("src/incident.rs", "Kill switch: global stop, per-repo deny, session revocation", 200, "P2"),
        ("src/attribution.rs", "Binds every action to identity, session, repo and commit", 200, "P1"),
    ],
}

DATA["auth"] = {
    "desc": "Credentials without leaking them: OS keyring, OAuth PKCE, short-lived tokens.",
    "files": [
        ("src/lib.rs", "Credential facade + provider selection", 90, "P1"),
        ("src/keyring.rs", "OS keychain wrapper with file fallback and 0600 perms", 220, "P1"),
        ("src/oauth_pkce.rs", "Browser + loopback callback PKCE flow", 300, "P1"),
        ("src/device_code.rs", "Headless/SSH device-code login", 240, "P2"),
        ("src/token_store.rs", "Token cache, expiry, refresh scheduling, revocation", 260, "P1"),
        ("src/service_account.rs", "CI mode: env/service-account credentials, no interactive prompts", 200, "P2"),
    ],
}

DATA["ext"] = {
    "desc": "Everything a customer can add without forking: skills, hooks, plugins, custom commands.",
    "files": [
        ("src/lib.rs", "Extension host facade + enable/disable resolution", 100, "P1"),
        ("src/skills/mod.rs", "Skill model: name, trigger description, allowed tools, budget", 220, "P2"),
        ("src/skills/loader.rs", "SKILL.md discovery, frontmatter parse, lazy injection", 280, "P2"),
        ("src/hooks/mod.rs", "Hook event catalogue + matcher syntax", 240, "P1"),
        ("src/hooks/exec.rs", "Hook execution: shell/webhook/verifier, timeouts, block semantics", 320, "P1"),
        ("src/plugins/manifest.rs", "Plugin manifest schema, component types, version constraints", 260, "P3"),
        ("src/plugins/verify.rs", "Signature + provenance verification before load", 240, "P3"),
        ("src/commands/custom.rs", "Markdown-defined slash commands with argument schema", 220, "P2"),
        ("src/agents_md.rs", "Agent profile files (markdown) for subagent definitions", 200, "P2"),
        ("src/watcher.rs", "Config/skill file watching with debounce and safe reload", 220, "P3"),
    ],
}

DATA["telemetry"] = {
    "desc": "Observability that is opt-in, scrubbed, and useful for support without leaking code.",
    "files": [
        ("src/lib.rs", "Init + global subscriber + shutdown flush", 120, "P1"),
        ("src/tracing_setup.rs", "Layered tracing: file ring buffer, stderr, env filters", 220, "P1"),
        ("src/otel.rs", "OTLP export of spans (turn, tool, model call) with sampling", 260, "P2"),
        ("src/metrics.rs", "Counters/histograms: latency, tokens, tool failures, cache hit rate", 240, "P2"),
        ("src/privacy.rs", "Scrubber: paths, prompts, hostnames; explicit opt-in surface", 260, "P1"),
        ("src/support_bundle.rs", "One-command diagnostic bundle with redaction preview", 220, "P2"),
    ],
}

DATA["git"] = {
    "desc": "Repository awareness: branch, dirty state, checkpointing, isolated worktrees.",
    "files": [
        ("src/lib.rs", "Repo discovery + facade", 100, "P1"),
        ("src/repo.rs", "Open repo via gix, branch/HEAD/status summary for prompts", 280, "P1"),
        ("src/diff.rs", "Working-tree diff extraction for review mode and UI", 260, "P1"),
        ("src/commit.rs", "Commit creation with trailer attribution and signing passthrough", 220, "P1"),
        ("src/worktree.rs", "Create/remove ephemeral worktrees for parallel subagents", 240, "P2"),
        ("src/checkpoint.rs", "Shadow-git checkpoints per turn for undo/rollback", 300, "P2"),
    ],
}

DATA["testkit"] = {
    "desc": "Shared test infrastructure: deterministic fixtures, mock provider, golden transcript tooling.",
    "files": [
        ("src/lib.rs", "Test harness exports", 60, "P0"),
        ("src/fixtures.rs", "Temp workspaces, sample repos, permission fixtures", 260, "P0"),
        ("src/mock_provider.rs", "In-process fake model server (SSE, tool calls, failures)", 320, "P0"),
        ("src/golden.rs", "Transcript/golden comparison helpers for insta snapshots", 200, "P1"),
    ],
}
