# Loom Context Pack

## Quick Links

- Workspace snapshot: `00-workspace-snapshot.md`
- MCP inventory: `00-mcp-inventory.md`
- Research: `10-research.md`
- Product spec: `20-product-spec.md`
- Implementation plan: `30-implementation-plan.md`
- Decisions: `40-decisions.md`
- Worklog: `50-worklog.md`

## Current Goal

- Expand `/Users/cblevins/workspace/services/tech-radar` from a radar plus summary dashboard into a more useful governance copilot for real projects.
- Keep the existing radar, insights, and codebase views, but add project pattern identification, anti-pattern warnings, and source-backed alternative suggestions.
- Ground the next slice in deterministic scan evidence so the app can explain why it flagged a project and what the safer default should be.

## Success Criteria

- The planning pack reflects the current `main` branch instead of the older “evidence persistence is missing” assumption.
- The product spec defines first-class pattern findings, anti-pattern warnings, and alternative suggestions for projects.
- The implementation plan identifies a concrete first slice that can ship without rebuilding the product shell.
- Planning and validation baselines are current:
  - `go test ./...`
  - `pnpm check`
  - `pnpm build`

## Open Questions

- Should pattern rules live only in Go code first, or also be authorable in repo config such as `.tech-radar/patterns.yaml`?
- Do we want findings to be fully derived at request time, or persisted as materialized analysis snapshots after each scan?
- How strict should the first anti-pattern warnings be for polyglot breadth and toolchain fragmentation so we avoid noisy false positives?

## Risks

- The current `project_technologies` primary key allows only one row per `project_id + tech_name`, which is too narrow for multiple observations of the same technology in one project. `infrastructure/radar/schema.sql:41-49`
- Pattern detection exists today, but it is heuristic-only and positive-only; there is no severity, confidence, remediation, or alternative suggestion model yet. `internal/codebase/analyzer.go:13-93`, `pkg/types/codebase.go:43-49`
- The frontend still treats project technology data as shallow fields (`name`, `source`, `version`, `quadrant`), so persisted evidence cannot yet drive useful project warnings. `web/src/lib/dashboard.ts:31-44`
- The codebase stack UI expects per-project pattern badges, but `buildStacks` does not currently populate `ProjectStack.Patterns`, so one intended signal path is effectively empty. `pkg/types/codebase.go:33-41`, `internal/codebase/analyzer.go:241-275`, `web/src/App.svelte:1887-1896`

## Sources

- [S1] `README.md:5-22`
- [S2] `internal/codebase/analyzer.go:13-93`
- [S3] `pkg/types/codebase.go:33-49`
- [S4] `infrastructure/radar/schema.sql:41-49`
- [S5] `web/src/lib/dashboard.ts:31-44`
- [S6] `web/src/App.svelte:1887-1896`
- [S7] `go test ./...`
- [S8] `pnpm check`
- [S9] `pnpm build`
