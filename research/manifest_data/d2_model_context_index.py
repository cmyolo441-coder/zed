# -*- coding: utf-8 -*-
"""Manifest data: model gateway, context engine, code index."""

DATA = {}

DATA["model"] = {
    "desc": "Provider-neutral gateway: one internal IR, N adapters, SSE streaming, caching, cost, failover.",
    "files": [
        ("src/lib.rs", "Public API: Gateway, ModelClient, ModelId, usage types", 60, "P0"),
        ("src/message_convert.rs", "Bidirectional conversion between IR and provider payloads", 340, "P0"),
        ("src/gateway.rs", "Model selection, fallback chains, per-role routing (main/summarizer/classifier)", 300, "P0"),
        ("src/client.rs", "HTTP client: pooling, proxy, TLS, timeouts, idempotent retries", 320, "P0"),
        ("src/stream.rs", "SSE framing, event decode, keepalive, graceful abort", 400, "P0"),
        ("src/stream_assembly.rs", "Incremental tool-call JSON accumulation + repair of partial args", 320, "P0"),
        ("src/providers/mod.rs", "Provider trait + registry + shared helpers", 160, "P0"),
        ("src/providers/anthropic.rs", "Messages API: cache_control breakpoints, thinking blocks", 520, "P0"),
        ("src/providers/openai_responses.rs", "Responses API: item stream, tool outputs, reasoning effort", 540, "P0"),
        ("src/providers/openai_chat.rs", "Chat Completions fallback for OpenAI-compatible hosts", 380, "P1"),
        ("src/providers/gemini.rs", "generateContent + function calling + 1M context handling", 440, "P1"),
        ("src/providers/bedrock.rs", "AWS Bedrock Converse + SigV4 credential chain", 300, "P2"),
        ("src/providers/vertex.rs", "Vertex AI endpoint + ADC auth", 280, "P2"),
        ("src/providers/azure.rs", "Azure OpenAI deployment routing and api-version pinning", 240, "P2"),
        ("src/providers/ollama.rs", "Local models: capability probing, no-stream fallback", 240, "P1"),
        ("src/providers/openrouter.rs", "Aggregator routing headers, provider preference", 200, "P3"),
        ("src/providers/custom.rs", "Generic OpenAI-compatible BYO endpoint with schema probing", 220, "P2"),
        ("src/auth_headers.rs", "Auth strategies: API key, bearer, signed, OAuth token refresh", 220, "P0"),
        ("src/capabilities.rs", "Feature matrix per model: tools, images, cache, reasoning, limits", 260, "P1"),
        ("src/token_count.rs", "Fast tokenizer + provider-authoritative usage reconciliation", 280, "P0"),
        ("src/pricing.rs", "Price table, cache-aware cost math, per-session spend", 220, "P1"),
        ("src/cache.rs", "Prompt-cache breakpoint placement and invalidation guard", 260, "P1"),
        ("src/reasoning.rs", "Reasoning/thinking budget control and redaction from history", 220, "P2"),
        ("src/ratelimit.rs", "429/overloaded handling, retry-after parsing, queueing", 220, "P0"),
        ("src/failover.rs", "Cross-provider failover with conversation replay safety", 260, "P2"),
        ("src/mock.rs", "Scriptable mock model used by every integration test", 300, "P0"),
    ],
}

DATA["context"] = {
    "desc": "The context economy: what the model sees, in which order, and what gets thrown away.",
    "files": [
        ("src/lib.rs", "ContextEngine facade used by the turn loop", 70, "P0"),
        ("src/assembler.rs", "Builds the final request: sections, order, cache boundaries", 360, "P0"),
        ("src/sections.rs", "Section model (stable/volatile) with token accounting", 240, "P0"),
        ("src/system_sections.rs", "Ordered system fragments: identity, rules, tools, env, policy", 300, "P0"),
        ("src/memory/mod.rs", "Layered memory loader and merge policy", 220, "P1"),
        ("src/memory/hierarchy.rs", "AGENTS.md/CLAUDE.md upward discovery, scoped rules, size caps", 280, "P0"),
        ("src/memory/auto.rs", "Agent-written notes with provenance and TTL", 240, "P2"),
        ("src/memory/conflicts.rs", "Deterministic precedence when user/project/managed rules collide", 200, "P2"),
        ("src/attachments.rs", "Explicit @file/@dir/@url references, images, pasted blobs", 240, "P1"),
        ("src/imports.rs", "Inline file imports with depth/size guards", 180, "P2"),
        ("src/compaction/mod.rs", "Compaction driver + trigger thresholds + hooks", 300, "P0"),
        ("src/compaction/ladder.rs", "Escalating strategy chain; stop as soon as budget is met", 260, "P0"),
        ("src/compaction/snip.rs", "Stage 1: clip oversized tool outputs and old images", 240, "P0"),
        ("src/compaction/dedupe.rs", "Stage 2: remove duplicated reads and repeated outputs", 220, "P1"),
        ("src/compaction/collapse.rs", "Stage 3: reversible folding of inactive segments", 300, "P2"),
        ("src/compaction/summarize.rs", "Stage 4: model-assisted summary with pinned facts", 400, "P0"),
        ("src/compaction/restore.rs", "Re-inject recently edited files and active plan after compaction", 240, "P1"),
        ("src/compaction/prompt.rs", "Summariser prompt + structured output contract", 200, "P0"),
        ("src/ledger.rs", "Token ledger: per-section estimate vs provider truth", 280, "P0"),
        ("src/pressure.rs", "Watermarks (soft/hard) and pre-flight overflow prediction", 200, "P0"),
        ("src/tool_budget.rs", "Tool schema budget, deferred/lazy schema disclosure", 260, "P2"),
        ("src/scratchpad.rs", "Agent-maintained plan/scratch state that survives compaction", 220, "P1"),
    ],
}

DATA["index"] = {
    "desc": "Code understanding without shipping the whole repo: tree-sitter tags, graph ranking, search.",
    "files": [
        ("src/lib.rs", "Index facade + configuration", 70, "P1"),
        ("src/repo_map/mod.rs", "Builds a token-budgeted map of the repository", 300, "P1"),
        ("src/repo_map/tags.rs", "tree-sitter queries for definitions/references across languages", 420, "P1"),
        ("src/repo_map/graph.rs", "Symbol/file graph construction with weighted edges", 320, "P1"),
        ("src/repo_map/pagerank.rs", "Personalised PageRank with chat-seeded restart vector", 260, "P1"),
        ("src/repo_map/cache.rs", "SQLite tag cache keyed by mtime + grammar version", 240, "P2"),
        ("src/search.rs", "ripgrep/grep integration, glob filters, result ranking", 320, "P0"),
        ("src/file_tree.rs", "gitignore-aware traversal with depth/size limits", 220, "P0"),
    ],
}
