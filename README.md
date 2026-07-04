# ZED — Real Enterprise Autonomous Software Engineering Agent

## Install

**Option A — `go install` (Go users ke liye sabse easy):**

```bash
go install github.com/gjkjk/zed/cmd/zed@latest
```

Iske baad `zed` command kahin se bhi chalega (agar `$GOPATH/bin` ya `$HOME/go/bin` PATH me hai).

**Option B — Ready binary download karo (Go install ki zaroorat nahi):**

1. [Releases page](https://github.com/gjkjk/zed/releases) par jao.
2. Apne OS ka binary download karo:
   - Windows: `zed-windows-amd64.exe`
   - Linux: `zed-linux-amd64` (ya `zed-linux-arm64`)
   - macOS Intel: `zed-macos-amd64` — Apple Silicon: `zed-macos-arm64`
3. Linux/macOS par executable banao aur chalao:
   ```bash
   chmod +x zed-linux-amd64
   ./zed-linux-amd64
   ```

**Setup (pehli baar):** ek API key chahiye. Set karo:

```bash
export ZED_API_KEY="your-key"   # Windows PowerShell: $env:ZED_API_KEY="your-key"
zed
```

---

This repository was upgraded from the supplied ZIP source into a real, local-evidence enterprise agent. Fake/skipped test scaffolds and pretend controls were removed or replaced with deterministic Go implementations that inspect actual project files.

## What is real now

- Real OpenAI-compatible and Anthropic API clients with cache support.
- Real filesystem/search/shell tools; shell execution uses explicit approval and the agent security policy. The sandbox package has been removed as requested.
- Real AST, index, diff, snapshot, session, memory, CI-detection, quality, security, profiling, watcher, plugin, collaboration, and distributed-queue code.
- Honest generated tests: `generate_tests` does not emit skipped placeholder tests. It generates a plan only until concrete function signatures and expected cases are supplied, then emits runnable assertions.
- Valid Go module directive (`go 1.23`) instead of a non-existent future version.

## 10+ new enterprise-grade real features

The agent now registers these additional tools in `cmd/zed/main.go`:

1. `enterprise_secret_scan` — credential regex + entropy scanner for real source/config files.
2. `enterprise_sbom` — SBOM generation from `go.mod`, `package.json`, `requirements.txt`, and `Cargo.toml`.
3. `enterprise_license_audit` — root/package license audit and risky license flags.
4. `enterprise_supply_chain_audit` — unpinned dependency, remote source, and missing lockfile detection.
5. `enterprise_pii_scan` — PII/data classification for email, India PAN/Aadhaar, and Luhn-valid card numbers.
6. `enterprise_config_audit` — production hardening checks for debug mode, TLS verification, and exposed admin binds.
7. `enterprise_policy_audit` — CODEOWNERS/SECURITY.md and policy-bypass marker audit.
8. `enterprise_slo_audit` — validates `.zed-slo.json` availability/latency/error-budget definitions.
9. `enterprise_integrity_manifest` — creates/verifies SHA-256 file integrity manifests.
10. `enterprise_audit_trail` — tamper-evident hash-chained JSONL audit log.
11. `enterprise_backup_readiness` — restore-manifest and backup inventory readiness check.
12. `enterprise_api_surface` — route/API surface discovery from real code.
13. `enterprise_observability_audit` — logs/metrics/traces/health/recovery readiness.
14. `enterprise_container_audit` — Docker/Kubernetes hardening checks.
15. `enterprise_code_ownership_audit` — CODEOWNERS coverage for source/config files.
16. `enterprise_threat_model` — STRIDE threat model from discovered endpoints.
17. `enterprise_release_gate` — combined blocking enterprise release gate.
18. `enterprise_evidence_pack` — JSON compliance evidence bundle.
19. `enterprise_policy_gate` — policy/waiver-aware release gate from `.zed-policy.json`.
20. `enterprise_risk_waiver_audit` — validates `.zed-risk-waivers.json` expiry/approver/reason.
21. `enterprise_sarif_export` — SARIF 2.1.0 export for security dashboards.
22. `enterprise_ci_templates` — writes GitHub Actions/GitLab CI enterprise gate templates.
23. `enterprise_policy_init` — returns starter policy and waiver JSON.
24. `enterprise_taint_analysis` — source-to-sink security analysis for injection/path/XSS risks.
25. `enterprise_migration_audit` — database migration safety audit.
26. `enterprise_evidence_keygen` — Ed25519 keypair generation for evidence signing.
27. `enterprise_evidence_sign` — sign evidence packs.
28. `enterprise_evidence_verify` — verify evidence pack signatures.
29. `enterprise_compliance_map` — SOC2/ISO27001/NIST/OWASP mapping.
30. `enterprise_architecture_brain` — architecture graph/domains/imports/integrations brain.
31. `enterprise_change_impact` — git-diff blast-radius and impacted tests/endpoints.
32. `enterprise_patch_bundle` — reviewable markdown patch bundle with diff/rollback.
33. `enterprise_build_doctor` — classifies real build/test output failures.
34. `enterprise_adr_generator` — writes Architecture Decision Records.
35. `enterprise_critical_path` — detects business-critical paths.
36. `enterprise_api_guardian` — API baseline create/compare for breaking changes.
37. `enterprise_go_concurrency_audit` — Go goroutine/ticker/mutex risk analyzer.
38. `enterprise_perf_hotspot_predictor` — static performance hotspot predictor.
39. `enterprise_work_journal` — agent engineering journal JSONL.

All reports support human-readable output and most support JSON via `{ "json": true }`.


## BITTU CHAUHAN enterprise UI

The terminal UI has been restyled into a premium dark agent console inspired by modern build/plan CLIs:

- top chrome with traffic-light dots and `BITTU CHAUHAN` brand
- centered startup mark and enterprise agent subtitle
- bordered conversation panel
- large bottom prompt box with `›` prompt
- footer shortcuts: `enter run`, `ctrl+p prompt`, `esc stop`, `ctrl+c quit`
- token/cost/model status bar



## Real Effort Engine

BITTU CHAUHAN now has real runtime effort levels, not just prompt labels:

| Effort | Runtime behavior |
|---|---|
| `normal` | 25 ReAct steps, serial tools, fast everyday mode. |
| `ultraeffort` | 10×: milestone preflight, risk heatmap, memory/journal, self-verify, auto-debug. |
| `ultramax` | 50×: deep codebase research, architecture brain, change impact, CI, security, policy gate, patch bundle. |
| `ultracombomax` | 120×: max enterprise mode with swarm, release gates, evidence pack, patch bundle, work journal. |
| `goal` | 200×: autonomous end-to-end goal execution with milestone plan, runtime gates, evidence pack, patch bundle, and journal. |

Commands:

```text
/effort ultraeffort
/ultramax
/ultracombomax
/goal build the complete feature end-to-end
```

CLI override:

```bash
./zed --effort ultracombomax
```

## Advanced Phase 5 UI workflow additions

Added real UI/workflow capabilities:

- `/plan <goal>` — full plan-mode panel with steps, risk, approvals, commands, rollback.
- `/review` — git diff review UI with added/removed line coloring and approval queue entry.
- `Ctrl+P` or `/palette` — enterprise command palette.
- `/approvals`, `/approve <id>`, `/deny <id>` — approval workflow with audit trail writes.
- `/timeline` — work-journal memory timeline.
- `/dashboard` — autonomous work dashboard.
- `/heatmap` — enterprise risk heatmap UI.
- `enterprise_risk_heatmap` — tool for path-level risk scoring.
- `enterprise_milestone_executor` — tool for enterprise milestone plans.
- `/replay <session>` — replay saved session messages.
- `/auth` — beautiful authentication/token guidance panel.
- `/setup` — model/provider setup wizard panel.
- `/theme <name>` — theme command hook.
- `/layout split|focus|minimal` — layout modes, including split-pane enterprise side panel.
- live tool activity stream in dashboard/split panel.
- `./zed --demo` — deterministic local demo mode without API key.

## Build

```bash
go mod download
go build -o zed ./cmd/zed
./zed --enterprise-policy-gate
./zed --enterprise-sarif zed-enterprise.sarif
```

## Run

```bash
export ZED_API_KEY="your-key"
./zed
```

Optional provider/model overrides:

```bash
./zed --provider openai --model gpt-4o
./zed --provider anthropic --model claude-3-5-sonnet-latest
```

## Enterprise tool examples

Inside the TUI, ask the agent to run:

```text
Run enterprise_secret_scan on the repo and return JSON.
Generate an enterprise_sbom for this project.
Create an enterprise_integrity_manifest at .zed-manifest.json.
Verify enterprise_audit_trail.
Run enterprise_policy_audit and enterprise_supply_chain_audit.
```

## Notes

I could not execute `go test` in this execution environment because the Go toolchain is not installed here. The source has been updated directly and structured to use only standard-library code for the new enterprise package.

## Additional documents

- `ENTERPRISE_PLAN.md` — complete enterprise roadmap and completed work.
- `A2Z_ANALYSIS_REPORT.md` — source analysis summary and validation checklist.

## Agent OS power features

Added a full BITTU CHAUHAN Agent OS layer with real local artifacts and commands:

1. `enterprise_vision_ui_builder` + `/vision-ui <image>` — image/screenshot to UI spec using real file hash, dimensions when supported, and implementation suggestions.
2. `prompt_os` + `/prompt-os` — `.zed-prompts/` prompt packs, roles, workflows, variables, rendering.
3. `agent_role_studio` — multi-agent role split for architect/builder/tester/UI/docs.
4. `skill_marketplace` + `/skills` — `.zed-skills/*.skill.json` local skill plugins.
5. `ui_composer` — JSON UI spec to generated Go component spec.
6. `macro_recorder` — `.zed-macros/*.json` repeated workflow recording.
7. `workflow_engine` — `.zed-workflows.json` local automation rules.
8. `agent_desktop` + `/desktop` — terminal desktop layout model.
9. `code_movie` + `/code-movie` — HTML timeline from diff and work journal.
10. `terminal_animation_engine` + `/animation` — terminal animation frames.
11. `autonomous_app_builder` + `/app <idea>` — creates app scaffold with code, tests, docs, run script.
12. `blueprint_engine` — parses Agent Blueprint DSL into `.zed-blueprint.json`.
13. `theme_generator` + `/theme-generate <prompt>` — generates theme palette JSON.
14. `persona_engine` + `/persona <role>` — persistent assistant persona file.
15. `knowledge_pack_builder` + `/knowledge` — `.zed-knowledge/` architecture, commands, conventions, glossary, onboarding.
16. `learning_system` — `.zed-learning.json` learning events.
17. `tutorial_builder` + `/tutorial` — docs/tutorials/getting-started.md.
18. `readme_designer` + `/readme-design` — professional README_DESIGNED.md.
19. `voice_command_mode` + `/voice <text>` — voice-ready command grammar.
20. `self_upgrade_kernel` + `/self-upgrade` — analyzes agent structure and recommends upgrades.

## Mission Control and max-level tool reliability

Added persistent multi-goal mission control:

- `/mission` — list missions
- `/mission start <goal>` — start mission and auto-pause other active missions
- `/mission pause <id>` — pause mission
- `/mission resume <id>` — resume mission
- `/mission complete <id>` — complete mission
- `mission_control` tool — start/list/pause/resume/complete/cancel/journal/artifact
- persistent file: `.zed-missions.json`

File and shell tools were upgraded for reliability:

- robust path resolution using `filepath.Rel` to prevent prefix/path traversal bugs
- stronger JSON argument repair for fenced JSON, smart quotes, embedded JSON objects, literal newlines, trailing commas
- binary/non-UTF8 read detection with safe metadata response
- atomic file writes via temp-file + rename
- newline normalization for edits
- fuzzy whitespace-tolerant edit fallback when exact block differs by formatting
- better edit errors with actionable hints
- shell supports `cwd`, `timeout_seconds`, `env`, `max_output_bytes`
- shell returns command, cwd, status, exit code, duration, truncation state, and output instead of hiding failures

## Agent OS Pro Max features

Added next-generation Pro Max capabilities:

- `agent_hypervisor` + `/hypervisor <goal>` — orchestration kernel for architect/builder/tester/UI/docs/release/self-upgrade roles.
- `memory_graph` + `/memory-graph <query>` — persistent `.zed-memory-graph.json` graph of files/artifacts/project facts.
- `codebase_map` + `/map` — packages/files/imports/tools/Agent OS map.
- `neural_workspace` + `/workspace` — live `.zed-workspace-state.json` with git status and artifacts.
- `merge_engine` — conflict marker detector and optional resolver.
- `patch_queue` — `.zed-patch-queue.json` queue/approve/apply/rollback model.
- `timeline_db` — `.zed-timeline.jsonl` structured event database.
- `replay_engine` — docs/replays HTML replay from timeline.
- `artifact_browser` + `/artifacts` — browser for .zed artifacts, docs, apps, web UI, dist.
- `project_generator_pro` + `/project-pro <name>` — full Go CLI project with docs, tests, Makefile.
- `preview_server` + `/preview <file>` — preview command/url generator.
- `web_ui_exporter` + `/web-ui` — exports web-ui/index.html artifact dashboard.
- `plugin_builder` + `/plugin-build <name>` — Go tool plugin skeleton generator.
- `agent_self_test_harness` + `/self-tests` — generates internal tool self-test harness.
- `benchmark_lab` + `/bench` — local benchmark timing output.
- `tool_reliability_score` — reliability counts from timeline DB.
- `tool_auto_repair` — repairs malformed tool argument candidates.
- `job_manager` + `/jobs` — persistent background job metadata, logs and stop support.
- `workspace_sync` — syncs prompts/skills/macros/workflows/knowledge/missions.
- `kernel_upgrade_planner` + `/kernel-plan` — writes docs/kernel-upgrade-plan.md.

## Dream Mode

Added `dream` effort mode — the 1000× Agent OS Pro Max controller mode.

Commands:

```text
/dream
/dream build the complete feature end-to-end
/effort dream
```

CLI:

```bash
./zed --effort dream
```

Dream Mode is real runtime behavior, not only a prompt label. It controls and coordinates:

- mission control
- agent hypervisor
- memory graph
- codebase map
- neural workspace
- artifact browser
- prompt OS
- skill marketplace
- self-upgrade kernel
- enterprise milestones
- architecture brain
- change impact
- risk heatmap
- policy gate
- patch bundle
- evidence pack
- timeline DB
- tool reliability score
- benchmark lab
- replay engine
- kernel upgrade planner

Dream Mode preflight runs real local tools before LLM execution and injects the evidence into the context. After file changes, Dream Mode runs the verification plane and artifact generation.
