# Implementation Plan

## Scope

Implement the next product slice as **Project Pattern Intelligence Foundations**.

The first delivery should make project-level architecture guidance real, not just implied:

- enrich observations so the app can explain findings
- extend pattern detection into a richer rule engine
- add anti-pattern warnings and alternative suggestions
- wire those findings into the existing codebase/projects experience

## Delivery Strategy

Ship this in layered slices so the existing app stays useful while the new guidance model comes online.

## Milestones

1. Observation model foundation
2. Pattern and anti-pattern engine
3. API and response-shape expansion
4. UI integration for project intelligence
5. Rule authoring, sample data, and docs

## Plan

### 1. Observation model foundation

- Fix the current one-row-per-tech limitation by introducing a richer observation structure.
- Preferred direction:
  - keep `project_technologies` for coarse summaries if needed
  - add a new observation table for per-project evidence rows
- Minimum observation payload:
  - `project_id`
  - `tech_name`
  - `source`
  - `evidence_path`
  - `evidence_title`
  - optional `observation_kind`
  - optional `observed_at`
- Primary files:
  - `infrastructure/radar/schema.sql`
  - `internal/db/db.go`
  - `pkg/types/types.go`

Why first:
- Pattern and anti-pattern guidance becomes much more trustworthy when we can point to more than one observation row. `infrastructure/radar/schema.sql:41-49`, `internal/db/db.go:184-225`

### 2. Pattern and anti-pattern engine

- Replace the current `patternRule` shape with a richer rule model.
- Add fields such as:
  - `kind`
  - `severity`
  - `confidence`
  - `why`
  - `alternatives`
  - `evidence selector`
- Preserve existing positive patterns, then add a first negative rule set:
  - active hold usage
  - untracked concentration
  - fragmented AI/MCP tooling
  - excessive polyglot spread
  - stack divergence between similar projects
- Populate per-project pattern badges while fixing the current `ProjectStack.Patterns` gap.
- Primary files:
  - `internal/codebase/analyzer.go`
  - `internal/codebase/analyzer_test.go`
  - `pkg/types/codebase.go`

### 3. API and response-shape expansion

- Keep `/api/v1/codebase`, but expand it to return richer per-project findings.
- Add dedicated detail endpoints if needed:
  - `/api/v1/projects/:id/intelligence`
  - `/api/v1/findings`
- Carry evidence through response types instead of dropping it at the frontend boundary.
- Primary files:
  - `internal/api/api.go`
  - `internal/db/db.go`
  - `pkg/types/codebase.go`
  - `pkg/types/types.go`

### 4. UI integration for project intelligence

- Add a project intelligence view or drawer that shows:
  - design patterns
  - anti-pattern warnings
  - severity/confidence
  - evidence list
  - suggested alternatives
- Upgrade the codebase tab pattern strip from passive badges to clickable diagnostics.
- Fix the stack grid so project pattern badges actually render from populated data.
- Extend `ProjectTechnology` on the frontend to include evidence fields.
- Primary files:
  - `web/src/App.svelte`
  - `web/src/lib/dashboard.ts`
  - `web/src/lib/codebase.ts`

### 5. Rule authoring, sample data, and docs

- Add a clear path for future configurable rules, likely via `.tech-radar/patterns.yaml`.
- Expand seed/demo data so the feature does not look empty on a fresh repo.
- Refresh README and product docs around “pattern intelligence” and “warnings with alternatives.”
- Primary files:
  - `README.md`
  - `.tech-radar/config.yaml`
  - `.tech-radar/radar.yaml`
  - optional new `.tech-radar/patterns.yaml`

## First Concrete Slice

If we want the smallest high-value first implementation, do this first:

1. Add an observation model that permits multiple rows per project/tech.
2. Extend `internal/codebase/analyzer.go` to emit a richer per-project finding structure.
3. Fix `ProjectStack.Patterns` population.
4. Surface one project detail panel in the UI with:
   - pattern badges
   - one or two anti-pattern warnings
   - evidence list
   - one suggested alternative

That gets us a visible usefulness jump without requiring the whole app to be rethought.

## Suggested Execution Order

1. Schema migration and Go types
2. DB persistence and reads
3. Analyzer refactor and tests
4. API expansion
5. Frontend models and project intelligence UI
6. Docs and sample data

## Test Plan

- DB tests
  - prove multiple observations for the same tech/project can coexist
  - prove evidence survives round-trip reads
- Analyzer tests
  - positive pattern coverage
  - anti-pattern coverage
  - severity/confidence consistency
  - `ProjectStack.Patterns` population
- API tests
  - project intelligence response shape
  - codebase analysis compatibility
- UI validation
  - project intelligence renders evidence and alternatives
  - codebase pattern badges are actionable
- Repo validation
  - `go test ./...`
  - `pnpm check`
  - `pnpm build`

## Acceptance Criteria

- The backend can represent multiple observations per technology per project.
- The analyzer emits both positive and negative findings with evidence and alternatives.
- The frontend can show project-level findings instead of only summary scores.
- The current pattern badge dead-end in the stack UI is fixed.
- The repo remains green on Go and web validation.

## Rollout / Backout

- Rollout
  - ship schema and analyzer changes first
  - ship project intelligence UI second
  - then expand the rule catalog
- Backout
  - if the full UX slice is too large, keep the richer observation model and analyzer output; both are independently valuable and reduce future rework

## Risks / Dependencies

- Dependency: the current schema shape must evolve before findings become trustworthy. `infrastructure/radar/schema.sql:41-49`
- Risk: heuristic rules become noisy.
  - Mitigation: add confidence, severity, and tightly scoped first rules.
- Risk: UI complexity rises too fast.
  - Mitigation: start with ranked cards and one project detail surface, not a full new IA.

## Sources

- [S1] `.tech-radar/config.yaml:1-69`
- [S2] `internal/db/db.go:139-225`
- [S3] `internal/api/api.go:226-299`
- [S4] `internal/codebase/analyzer.go:21-179`
- [S5] `internal/codebase/analyzer.go:241-355`
- [S6] `internal/codebase/analyzer_test.go:90-135`
- [S7] `internal/codebase/analyzer_test.go:232-254`
- [S8] `pkg/types/codebase.go:33-49`
- [S9] `pkg/types/types.go:115-123`
- [S10] `infrastructure/radar/schema.sql:41-49`
- [S11] `web/src/App.svelte:1717-1915`
- [S12] `web/src/App.svelte:1955-2075`
- [S13] `web/src/lib/dashboard.ts:31-44`
- [S14] `web/src/lib/dashboard.ts:982-1099`
