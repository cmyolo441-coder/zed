# -*- coding: utf-8 -*-
"""Manifest data: tools, process execution, OS sandboxing."""

DATA = {}

DATA["tools"] = {
    "desc": "The capability surface exposed to the model: primitives first, adapters second, all policy-checked.",
    "files": [
        ("src/lib.rs", "Tool trait, registry re-exports, feature gating", 120, "P0"),
        ("src/tool.rs", "Tool trait: schema, validation, execute, preview, risk class, UI hints", 300, "P0"),
        ("src/schema.rs", "JSON-Schema builder + token-cost estimator per tool", 240, "P0"),
        ("src/registry.rs", "Static + dynamic tool pool, name collision resolution, enable/disable", 280, "P0"),
        ("src/fs/read.rs", "Paged file read with line ranges, binary detection, image routing", 340, "P0"),
        ("src/fs/write.rs", "Atomic create/overwrite with backup and encoding preservation", 260, "P0"),
        ("src/fs/edit.rs", "Exact string replacement with occurrence disambiguation", 320, "P0"),
        ("src/fs/multi_edit.rs", "Batch edits with all-or-nothing semantics", 240, "P1"),
        ("src/fs/glob.rs", "Glob/gitignore-aware file discovery, sorted, capped", 200, "P0"),
        ("src/fs/ls.rs", "Directory tree listing with sizes and ignore rules", 180, "P0"),
        ("src/fs/move_copy.rs", "Rename/copy/delete with trash-by-default safety", 200, "P2"),
        ("src/fs/notebook.rs", "Jupyter cell editing as a first-class tool", 260, "P3"),
        ("src/patch/mod.rs", "Patch subsystem facade", 80, "P0"),
        ("src/patch/parser.rs", "Unified diff + V4A + search/replace block parser", 480, "P0"),
        ("src/patch/apply.rs", "Context-tolerant application with fuzzy anchors", 420, "P0"),
        ("src/patch/validate.rs", "Pre-flight validation against current file bytes", 300, "P0"),
        ("src/patch/rollback.rs", "Transactional apply: journal, partial failure rollback", 260, "P1"),
        ("src/search/grep.rs", "Regex search with context lines, type filters, result caps", 380, "P0"),
        ("src/search/engine.rs", "ripgrep binary vs in-process fallback selection", 240, "P1"),
        ("src/shell/mod.rs", "Shell tool: command spec, risk scan, approval binding", 320, "P0"),
        ("src/shell/session.rs", "Persistent shell session with cwd/env persistence", 380, "P1"),
        ("src/shell/background.rs", "Background jobs, log tailing, lifecycle ownership", 300, "P2"),
        ("src/shell/risk.rs", "Command classification: read-only / mutating / destructive / network", 340, "P0"),
        ("src/shell/format.rs", "Output post-processing: ANSI strip, exit code, truncation", 200, "P0"),
        ("src/net/fetch.rs", "URL fetch with SSRF guard, size cap, HTML->text extraction", 340, "P1"),
        ("src/net/search.rs", "Pluggable web search provider + citation formatting", 280, "P2"),
        ("src/lsp/client.rs", "LSP client lifecycle over stdio (rust-analyzer, tsserver, gopls)", 420, "P2"),
        ("src/lsp/diagnostics.rs", "Post-edit diagnostics as structured tool output", 300, "P2"),
        ("src/lsp/symbols.rs", "Definition/reference/hover queries for precise navigation", 260, "P3"),
        ("src/git/status.rs", "Status/diff/log summaries optimised for model consumption", 240, "P1"),
        ("src/git/commit.rs", "Commit with generated message + attribution trailer", 200, "P1"),
        ("src/git/worktree.rs", "Ephemeral worktrees for isolated parallel tasks", 260, "P2"),
        ("src/plan.rs", "todo_write / plan tool with stable IDs and status transitions", 260, "P1"),
        ("src/ask_user.rs", "Elicitation tool: structured questions with choices", 220, "P1"),
        ("src/task.rs", "Subagent spawn tool: prompt, profile, budget, summary return", 300, "P1"),
        ("src/memory_tool.rs", "Explicit memory read/write tool with review gate", 200, "P2"),
        ("src/render.rs", "Per-tool UI render hints (diff view, terminal, table, image)", 220, "P1"),
        ("src/errors.rs", "Tool error mapping to model-actionable messages", 200, "P0"),
        ("src/path_guard.rs", "Single choke-point for every model-supplied path", 180, "P0"),
    ],
}

DATA["exec"] = {
    "desc": "Process execution primitives: spawn, PTY, groups, env sanitising. No policy decisions here.",
    "files": [
        ("src/lib.rs", "Executor facade + public result types", 100, "P0"),
        ("src/spec.rs", "CommandSpec: argv, cwd, env, stdin, timeout, shell mode", 220, "P0"),
        ("src/spawn.rs", "tokio::process spawn with process-group ownership", 300, "P0"),
        ("src/pty.rs", "PTY allocation via portable-pty for interactive programs", 340, "P1"),
        ("src/pty_session.rs", "Long-lived PTY session: writes, reads, resize, expect-style waits", 360, "P2"),
        ("src/env.rs", "Env scrubbing: strip secrets, set deterministic vars, PATH policy", 220, "P0"),
        ("src/timeout.rs", "Deadline enforcement + SIGTERM->SIGKILL escalation", 180, "P0"),
        ("src/output.rs", "Ring-buffered stdout/stderr capture with size caps", 260, "P0"),
        ("src/stream.rs", "Live output streaming into events for the TUI", 240, "P1"),
        ("src/reaper.rs", "Zombie/orphan cleanup on cancel and on process exit", 200, "P1"),
    ],
}

DATA["sandbox"] = {
    "desc": "Defence in depth per OS. One policy type, four backends, one availability probe.",
    "files": [
        ("src/lib.rs", "Backend selection, capability probe, graceful degradation", 200, "P0"),
        ("src/policy.rs", "SandboxPolicy: readable/writable roots, network, spawn, env", 260, "P0"),
        ("src/backend.rs", "Backend trait + apply-to-child contract", 180, "P0"),
        ("src/linux_bwrap.rs", "bubblewrap argv construction with read-only binds", 340, "P1"),
        ("src/linux_landlock.rs", "Landlock ruleset built pre-exec in a single-threaded worker", 360, "P1"),
        ("src/linux_seccomp.rs", "seccomp-BPF filter: block mount/ptrace/raw sockets", 320, "P2"),
        ("src/macos_seatbelt.rs", "SBPL profile generation + sandbox-exec wrapper", 380, "P1"),
        ("src/windows_appcontainer.rs", "AppContainer + capability SIDs for child processes", 360, "P2"),
        ("src/windows_job.rs", "Job object limits: memory, process count, kill-on-close", 240, "P2"),
        ("src/network.rs", "Egress allowlist + loopback proxy wiring for sandboxed net", 320, "P2"),
        ("src/probe.rs", "Runtime detection: kernel version, bwrap present, entitlement checks", 220, "P0"),
        ("src/diagnostics.rs", "Explains WHY a sandboxed command failed (denied path, missing bind)", 240, "P1"),
    ],
}
