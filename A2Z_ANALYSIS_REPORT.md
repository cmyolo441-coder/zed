# A2Z Source Analysis Report

Generated: 2026-07-03

## Scope

- Go source files analyzed: 91
- Go source lines: 15717
- Packages/directories: 36
- Sandbox package: removed from Go source
- Remaining sandbox text references: documentation only (`README.md`, `ENTERPRISE_PLAN.md`, this report) explaining that the package was removed

## Package map

- `cmd/zed` — 1 file(s)
- `internal/agent` — 3 file(s)
- `internal/ast` — 1 file(s)
- `internal/cache` — 1 file(s)
- `internal/ci` — 1 file(s)
- `internal/collab` — 1 file(s)
- `internal/config` — 2 file(s)
- `internal/context` — 1 file(s)
- `internal/diagram` — 1 file(s)
- `internal/diff` — 1 file(s)
- `internal/dist` — 1 file(s)
- `internal/enterprise` — 2 file(s)
- `internal/fuzz` — 1 file(s)
- `internal/index` — 1 file(s)
- `internal/llm` — 5 file(s)
- `internal/logging` — 1 file(s)
- `internal/memory` — 1 file(s)
- `internal/meta` — 1 file(s)
- `internal/metrics` — 1 file(s)
- `internal/planner` — 1 file(s)
- `internal/plugin` — 1 file(s)
- `internal/predict` — 1 file(s)
- `internal/profiler` — 1 file(s)
- `internal/quality` — 1 file(s)
- `internal/reason` — 1 file(s)
- `internal/retry` — 1 file(s)
- `internal/scaffold` — 1 file(s)
- `internal/security` — 1 file(s)
- `internal/selftest` — 1 file(s)
- `internal/session` — 1 file(s)
- `internal/snapshot` — 2 file(s)
- `internal/swarm` — 1 file(s)
- `internal/tools` — 26 file(s)
- `internal/tui` — 2 file(s)
- `internal/vcs` — 1 file(s)
- `internal/watcher` — 1 file(s)

## Main upgrade findings

1. The old sandbox implementation was coupled into `cmd/zed`, `tools.Shell`, and TUI `/sandbox`. It has been removed from Go source. Shell remains explicit-approval based and governed by the agent security policy.
2. Enterprise controls are now evidence-driven: reports are built from actual files, manifests, hashes, routes, dependencies, config, and code ownership rules.
3. Test generation is honest: no skipped placeholder tests are emitted; executable tests require concrete signatures and expected values.
4. Release readiness can now be evaluated with a combined gate instead of isolated ad-hoc checks.

## Registered enterprise tools

- `enterprise_secret_scan`
- `enterprise_sbom`
- `enterprise_license_audit`
- `enterprise_supply_chain_audit`
- `enterprise_pii_scan`
- `enterprise_config_audit`
- `enterprise_policy_audit`
- `enterprise_slo_audit`
- `enterprise_integrity_manifest`
- `enterprise_audit_trail`
- `enterprise_backup_readiness`
- `enterprise_api_surface`
- `enterprise_observability_audit`
- `enterprise_container_audit`
- `enterprise_code_ownership_audit`
- `enterprise_threat_model`
- `enterprise_release_gate`
- `enterprise_evidence_pack`


## Phase 4 implementation added

- `--enterprise-sarif` non-interactive CLI command for CI.
- `--enterprise-policy-gate` non-interactive CLI command for CI.
- `.zed-policy.json` loader with strict release thresholds.
- `.zed-risk-waivers.json` validator with expiry/approver/reason checks.
- SARIF 2.1.0 exporter.
- GitHub Actions and GitLab CI template writer.

## Recommended next validation

Run these commands on a machine with Go installed:

```bash
go mod tidy
go test ./...
go vet ./...
go build -o zed ./cmd/zed
```

## Notes

The current execution environment does not include the Go toolchain, so runtime compilation/tests could not be executed here.

## Brain feature update

Added the requested Phase 5 tools: architecture brain, change impact, patch bundle, build doctor, ADR generator, critical path analyzer, API guardian, Go concurrency audit, performance hotspot predictor, and work journal.

Current Go files: 82
Current Go lines: 14345

## UI update

Added BITTU CHAUHAN branded enterprise TUI with dark header chrome, centered startup mark, bordered transcript panel, premium prompt box, footer shortcuts, and improved status bar.

## Advanced workflow UI update

Added plan mode, diff review, command palette, approvals, timeline, dashboard, heatmap, milestone executor, replay, auth/setup panels, theme/layout controls, live tool activity, and deterministic demo mode.

Current Go files: 84
Current Go lines: 14691

## Real effort engine update

Added actual runtime behavior for ultraeffort, ultramax, ultracombomax, and goal mode: deterministic enterprise preflight tools, milestone planning, risk heatmap, architecture/change-impact evidence, post-change policy gates, patch bundles, evidence packs, work journal entries, context budget scaling, CLI `--effort`, and serial/parallel tool execution based on effort profile.

## Agent OS power layer

Added all requested new-new power features: vision UI builder, prompt OS, role studio, skill marketplace, UI composer, macro recorder, workflow engine, desktop mode, code movie, terminal animations, app builder, blueprint engine, theme generator, personas, knowledge pack, learning system, tutorial builder, README designer, voice command mode, and self-upgrade kernel.

Current Go files: 87
Current Go lines: 15114

## Mission Control and tool reliability update

Added persistent mission control with active/paused/completed missions, artifacts, and mission journal. Upgraded read_file/write_file/edit_file/append_file/list_dir/run_shell for robust JSON handling, safer path resolution, atomic writes, binary detection, fuzzy edits, configurable shell cwd/timeouts/env, and structured shell exit metadata.

Current Go files: 89
Current Go lines: 15419

## Agent OS Pro Max layer

Added agent hypervisor, memory graph, codebase map, neural workspace, merge engine, patch queue, timeline DB, replay engine, artifact browser, project generator pro, preview server, web UI exporter, plugin builder, self-test harness, benchmark lab, tool reliability score, tool auto repair, job manager, workspace sync, and kernel upgrade planner.

Current Go files: 91
Current Go lines: 15616

## Dream Mode update

Added `dream` effort mode with 1000× multiplier, `/dream` TUI command, `--effort dream` CLI support, Dream Mode prompt contract, runtime control plane, and runtime verification plane. Dream Mode coordinates mission_control, agent_hypervisor, memory_graph, codebase_map, neural_workspace, artifact_browser, prompt_os, skill_marketplace, self_upgrade_kernel, timeline_db, tool_reliability_score, benchmark_lab, replay_engine, kernel_upgrade_planner, and existing enterprise gates.

Build/test attempt: `go version`, `go test ./...`, and `go build -o zed ./cmd/zed` were executed but this environment does not have the Go toolchain installed (`go: command not found`).

Current Go files: 91
Current Go lines: 15717
