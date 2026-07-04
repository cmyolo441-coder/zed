# Full Enterprise Upgrade Plan for ZED

Date: 2026-07-03

## Mission

Turn ZED into a production-grade enterprise autonomous engineering agent with real controls, auditable evidence, deterministic local analyzers, strong governance, and no fake/skipped/simulated implementation paths.

## Completed in this upgrade

### Phase 1 — Remove sandbox coupling

- Removed `internal/sandbox` package from active source tree.
- Removed sandbox import/initialization from `cmd/zed/main.go`.
- Removed sandbox field from `tools.Shell`.
- Removed `/sandbox` TUI command and help entry.
- Shell execution remains explicit-approval based and is still governed by agent security policy.

### Phase 2 — Real enterprise audit foundation

Implemented evidence-driven enterprise package and tool wrappers:

- Secret scanning with credential regexes and entropy checks.
- SBOM generation from real dependency manifests.
- License audit.
- Supply-chain risk audit.
- PII classification.
- Configuration hardening audit.
- Policy-as-code audit.
- SLO/error-budget audit.
- SHA-256 integrity manifests.
- Tamper-evident audit trails.
- Backup/restore readiness.

### Phase 3 — Advanced complex enterprise controls

Added advanced real controls:

- API surface discovery from Go/Node/Python route patterns.
- STRIDE threat model generation from discovered endpoints.
- Observability audit for logs, metrics, traces, health, and recovery.
- Container/Kubernetes hardening audit.
- CODEOWNERS/code-ownership coverage audit.
- Combined enterprise release gate.
- Compliance/evidence pack JSON writer.

## Next-level roadmap

### Phase 4 — CI/CD enterprise enforcement — started/implemented

- Added SARIF 2.1.0 output for enterprise findings through `enterprise_sarif_export` and `--enterprise-sarif`.
- Added GitHub Actions/GitLab CI templates through `enterprise_ci_templates`.
- Added risk-waiver files with expiry and approver validation through `enterprise_risk_waiver_audit`.
- Added policy configuration `.zed-policy.json` with severity thresholds through `enterprise_policy_gate`.
- Added `enterprise_policy_init` starter JSON generator.

### Phase 4 next tasks

- Add SARIF rule metadata taxonomy for SOC2/ISO/NIST mapping.
- Add branch protection checks for GitHub/GitLab APIs.
- Add signed evidence packs.

### Phase 5 — Deep code intelligence — started/implemented

- Added real local taint analysis for SQL injection, command injection, path traversal, and XSS proximity.
- Added database migration safety audit for destructive and blocking migration patterns.
- Added compliance mapping for SOC2, ISO27001, NIST, and OWASP.
- Added Ed25519 evidence signing and verification.


### Phase 5 additional update implemented

- Added `enterprise_architecture_brain` for architecture graph/domain/import/integration extraction.
- Added `enterprise_change_impact` for real git-diff blast-radius analysis.
- Added `enterprise_patch_bundle` for reviewable unified diff bundles with verification and rollback notes.
- Added `enterprise_build_doctor` for build/test failure classification.
- Added `enterprise_adr_generator` for ADR creation.
- Added `enterprise_critical_path` for critical business path detection.
- Added `enterprise_api_guardian` for API baseline and breaking-change detection.
- Added `enterprise_go_concurrency_audit` for Go concurrency risks.
- Added `enterprise_perf_hotspot_predictor` for static performance risk detection.
- Added `enterprise_work_journal` for auditable agent work logs.

### Phase 5 next tasks

- Add language-specific AST route extraction for Go, TS, Python, Java, and C#.
- Add call graph risk scoring for sensitive sinks.
- Add deeper interprocedural data-flow for secret/PII propagation.

### Phase 6 — Runtime and production readiness

- Add OpenTelemetry instrumentation templates.
- Add SLO burn-rate report parser for Prometheus JSON exports.
- Add deployment manifest diff risk scoring.
- Add rollback-plan validation and migration safety checks.

### Phase 7 — Governance and compliance

- Add SOC2/ISO27001/NIST control mapping for each finding.
- Add evidence bundle signing.
- Add audit-log verification command in CI.
- Add dependency provenance attestation support.

### Phase 8 — Enterprise UX

- Add `/enterprise` TUI dashboard.
- Add release-gate summary in status line.
- Add export buttons for JSON, Markdown, and SARIF evidence.
- Add guided remediation workflows.

## Definition of done

- `go test ./...` passes on a Go-enabled machine.
- `go vet ./...` passes.
- `enterprise_release_gate` returns PASS or documented risk exceptions.
- No skipped placeholder tests are generated.
- No sandbox package remains in Go source.
- Enterprise evidence pack can be generated reproducibly.

## Phase 5 UI workflow update implemented

- Full plan mode UI.
- Review diff UI.
- Command palette.
- Approval workflow UI.
- Timeline UI.
- Dashboard UI.
- Risk heatmap tool and UI.
- Milestone executor tool.
- Replay UI.
- Auth/setup panels.
- Theme/layout commands.
- Live tool activity stream.
- Deterministic `--demo` mode.
