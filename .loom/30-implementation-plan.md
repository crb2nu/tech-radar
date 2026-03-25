# Implementation Plan

## Scope

Implement the next product slice as **Decision Workbench Foundations**:

- persist evidence and observations that are currently dropped
- add project and technology drill-down data shapes
- replace shallow recommendation rows with a more useful action queue
- keep the existing radar/dashboard shell, but make it a launch point into real governance context

## Milestones

1. Evidence persistence foundation
2. Detail APIs and domain model enrichment
3. UI drill-down transformation
4. Action queue and overview refocus
5. Validation, docs, and follow-on discovery hooks

## Plan

1. Evidence persistence foundation
   - Extend DB persistence beyond `tech_name`, `version`, and `source`. `internal/db/db.go:192-201`, `infrastructure/radar/schema.sql:41-47`
   - Decide whether to enrich `project_technologies` directly or add a new observation/evidence table.
   - Preserve at least:
     - evidence path
     - evidence title
     - evidence kind/source classification
     - first/last seen timestamps if feasible
   - Add DB tests that prove scanner-captured evidence survives ingestion. `pkg/types/types.go:115-123`, `internal/scanner/scanner.go:371-395`

2. Detail APIs and domain model enrichment
   - Add detail-oriented API shapes for:
     - a single radar entry dossier
     - a single project drill-down
   - Enrich the domain model so project views can return more than a score and top six names. `pkg/types/types.go:166-182`
   - Keep existing list endpoints stable where practical, but introduce additive response shapes for richer drill-downs. `internal/api/api.go:161-215`

3. UI drill-down transformation
   - Turn the selected-entry modal into a dossier view that includes:
     - affected projects
     - source/evidence list
     - versions where known
     - ring history and recommendation context
   - Make project cards open a real project diagnostic panel/page instead of staying as summary tiles. `web/src/App.svelte:943-993`, `web/src/App.svelte:1335-1405`
   - Preserve the existing radar canvas and command-deck framing; do not restart the UI from scratch. `web/src/App.svelte:771-939`

4. Action queue and overview refocus
   - Replace or augment the current ephemeral recommendations with a more durable action model.
   - At minimum, actions should be structured objects with stable IDs, linked entry/project context, and a status.
   - Refocus the overview from “how many things” toward:
     - what changed
     - what needs review now
     - which projects got worse
     - which technologies need evidence or decisions
   - Reuse current recommendation heuristics as seed logic where helpful, but stop treating them as the final product. `web/src/lib/dashboard.ts:932-1049`

5. Validation, docs, and follow-on discovery hooks
   - Update README and architecture notes to explain the richer workbench behavior.
   - Add or refresh sample/demo data so the richer UI does not render as mostly empty states. `.tech-radar/radar.yaml:1-24`
   - Keep AI/MCP discovery improvements as the next follow-on slice once the new dossier/action model is ready.
   - Run:
     - `go test ./...`
     - `cd web && pnpm check`
     - `cd web && pnpm build`

## Suggested Execution Order

1. Schema + persistence
2. Go types + DB reads
3. New detail endpoints
4. Svelte dossier/project drill-down UI
5. Action queue refactor
6. Sample data/docs polish

This order keeps the foundational data changes ahead of the UX work so we do not build another shallow UI layer on top of incomplete persistence.

## Test Plan

- Persistence tests
  - Verify `EvidencePath` and `EvidenceTitle` survive scan ingestion and are returned through detail APIs.
- API tests
  - Add coverage for new project/entry detail routes or handlers.
- UI validation
  - Confirm a user can move from overview -> action -> dossier -> project drill-down without losing context.
- Regression checks
  - Re-run existing Go and Svelte validation after each milestone.

## Acceptance Criteria

- Selecting a technology explains where it is observed, in which projects, with what evidence, and why it is currently in its ring.
- Selecting a project explains which technologies are driving its risk/alignment and what action is needed next.
- The action queue links to durable context objects rather than acting as a pure search/filter shortcut.
- DB-backed mode preserves scanner evidence that is currently only available in YAML-first flows.
- Local validation passes: `go test ./...`, `cd web && pnpm check`, `cd web && pnpm build`.

## Rollout / Backout

- Rollout
  - Ship evidence persistence first, even if the UI lags behind.
  - Ship detail APIs second so the frontend can build against stable data.
  - Ship dossier/project/action UI third.
- Backout
  - If the full workbench slice is too large, keep the evidence persistence and detail APIs; they are independently valuable and reduce future rework.

## Risks / Dependencies

- Risk: schema work creates migration complexity.
  - Mitigation: add additive tables/columns first and keep legacy list endpoints working.
- Risk: persisted actions sprawl into project management.
  - Mitigation: keep the first action model intentionally lightweight and governance-focused.
- Risk: sparse sample data keeps the UX looking empty.
  - Mitigation: update seed/demo data as part of the same slice.
- Dependency: helper scripts in `$CODEX_HOME` remain broken, so planning docs should continue to be maintained manually for now.

## Sources

- [S1] `README.md:5-22`
- [S2] `.tech-radar/radar.yaml:1-24`
- [S3] `pkg/types/types.go:115-123`
- [S4] `pkg/types/types.go:166-182`
- [S5] `internal/scanner/scanner.go:371-395`
- [S6] `internal/db/db.go:112-201`
- [S7] `internal/api/api.go:161-215`
- [S8] `infrastructure/radar/schema.sql:41-47`
- [S9] `web/src/App.svelte:771-939`
- [S10] `web/src/App.svelte:943-993`
- [S11] `web/src/App.svelte:1335-1405`
- [S12] `web/src/lib/dashboard.ts:932-1049`
- [S13] `go test ./...`
- [S14] `cd web && pnpm check`
- [S15] `cd web && pnpm build`
- [S16] `codebase_memory__codebase_stats(repo_id="tech-radar")`
