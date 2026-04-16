# Research Brief

## Problem

Tech Radar already does more than a classic radar: it has project scans, insights, codebase analysis, hotspot views, runtime breadth, stack similarity, and basic pattern detection. But the app still stops short of telling someone:

- what architectural pattern a project is actually exhibiting
- which combinations look risky or off-policy
- why that warning was raised
- what a better default or alternative should be

That is the biggest remaining gap between “interesting portfolio telemetry” and “useful governance tool.” `README.md:9-22`, `web/src/App.svelte:1522-1655`, `web/src/App.svelte:1696-1915`

## Questions

- Q1: What feature expansion would most increase practical usefulness without replacing the current radar shell?
- Q2: How much of project pattern identification already exists in the codebase?
- Q3: What prevents the current app from surfacing anti-patterns, warnings, and alternatives today?

## Method

- Refreshed the workspace snapshot and MCP/tooling baseline.
- Read the README, scan config, scanner, DB schema, API handlers, codebase analyzer, dashboard model, and main Svelte app.
- Verified the local validation baseline:
  - `go test ./...`
  - `pnpm check`
  - `pnpm build`

## Findings

1. The product promise already points beyond “just a radar.”
   - The README positions the app around tracked decisions, auto-discovery, documentation, and cross-project analysis. `README.md:9-22`
   - That means “project pattern guidance” is an expansion of the current mission, not a pivot away from it.

2. The app already has a meaningful codebase-intelligence surface.
   - The codebase tab exposes dependency graph, tech stacks, and file explorer views. `web/src/App.svelte:1717-1915`
   - The overview also includes cross-project hotspots, runtime ecosystem breakdown, polyglot complexity, and stack convergence panels. `web/src/App.svelte:1522-1655`
   - This is a good staging area for richer architecture guidance rather than a brand-new product surface.

3. Pattern identification already exists, but only as a shallow heuristic layer.
   - `internal/codebase/analyzer.go` ships six positive pattern rules today:
     - `GitOps`
     - `MCP-enabled`
     - `AI/ML pipeline`
     - `Containerized`
     - `Monorepo`
     - `Full-stack` `internal/codebase/analyzer.go:21-93`
   - The output model for those findings is only `{ pattern, projects, signal, icon }`, with no severity, confidence, remediation, or alternatives. `pkg/types/codebase.go:43-49`
   - Tests prove the positive-pattern layer works, but they also show the scope is still limited to presence/absence detection. `internal/codebase/analyzer_test.go:90-135`, `internal/codebase/analyzer_test.go:232-254`

4. There is no anti-pattern or warning model yet.
   - There are no anti-pattern types in `pkg/types/codebase.go` or `pkg/types/types.go`. `pkg/types/codebase.go:6-70`, `pkg/types/types.go:107-193`
   - The recommendation engine only reasons about drift, motion, ADR evidence, uncharted technologies, and AI/MCP standardization. It never evaluates project pattern combinations or emits “do this instead” guidance. `web/src/lib/dashboard.ts:982-1099`
   - Current project risk scoring is broad-brush coverage math, not rule-backed diagnosis. `internal/db/insights.go:89-210`, `web/src/lib/dashboard.ts:808-847`

5. Evidence is persisted in the DB now, but still not carried through to the user-facing models that would power warnings.
   - `SaveScan` stores `evidence_path` and `evidence_title`, and `GetProjects` reads them back. `internal/db/db.go:139-160`, `internal/db/db.go:184-225`
   - But the frontend `ProjectTechnology` type only includes `name`, `source`, `version`, and `quadrant`. `web/src/lib/dashboard.ts:31-44`
   - The entry detail modal shows project names, movement, tags, recommendation, and one link, but not a list of concrete observation files or evidence items. `web/src/App.svelte:1955-2075`

6. The current data model is too coarse for richer guidance.
   - `project_technologies` uses `PRIMARY KEY (project_id, tech_name)`. `infrastructure/radar/schema.sql:41-49`
   - That means one project cannot store multiple separate observations for the same technology, such as:
     - manifest occurrence
     - ADR mention
     - config-file detection
     - manual annotation
   - For pattern and anti-pattern warnings, that is a real limitation because good guidance usually depends on multiple corroborating signals.

7. The current codebase analyzer leaks some unfinished intent.
   - `ProjectStack` has a `Patterns []string` field. `pkg/types/codebase.go:33-41`
   - The UI tries to render per-project pattern badges in the stack matrix. `web/src/App.svelte:1887-1896`
   - But `buildStacks` never assigns `Patterns`, so that signal path is currently inert. `internal/codebase/analyzer.go:241-275`
   - This is a strong indicator that project-level pattern attribution was already on the product path and is a good place to resume.

8. File-level evidence is still abstracted away in the codebase view.
   - `buildFileTrees` groups findings by `Source`, which is usually a manifest type like `go.mod` or `package.json`, not a real path. `internal/codebase/analyzer.go:320-355`
   - The scanner does know concrete evidence paths for ADR and MCP detections. `internal/scanner/scanner.go:64-85`, `internal/scanner/scanner.go:384-541`
   - That mismatch reduces trust when a warning needs to answer “what file made you think this?”

## Option Space

### Option A: Add more charts and summary cards

- Pros:
  - Fastest incremental improvement.
  - Builds on the current shell.
- Cons:
  - Does not close the usefulness gap.
  - Risks making the app even more telemetry-heavy.

### Option B: Add anti-pattern warnings on top of the current shallow project model

- Pros:
  - Faster than a deeper model refactor.
  - Can prove the UX concept quickly.
- Cons:
  - Warnings will be noisy or under-explained because the evidence model is too thin.
  - Alternative suggestions will be hard to justify with only one observation per project/tech.

### Option C: Build a project pattern intelligence foundation

- Pros:
  - Reuses the existing codebase view and pattern analyzer direction.
  - Gives the app a new “reasoning layer” instead of another summary layer.
  - Creates the right substrate for both positive patterns and anti-pattern warnings.
- Cons:
  - Requires type/schema/API/UI work together.
  - Slightly bigger first slice than a pure UI enhancement.

## Recommendation

Choose **Option C: project pattern intelligence foundation**.

That means the next expansion should make these first-class:

- project design pattern findings
- anti-pattern warnings
- evidence-backed rationale
- alternative suggestions mapped to tracked standards or better defaults

This direction fits the current product and codebase better than another round of visualization work because the repo already has:

- project scans
- stored evidence
- a codebase analysis endpoint
- a partial pattern engine
- a recommendation layer looking for higher-value signals

It is missing the explanatory and prescriptive layer that turns those building blocks into guidance. `internal/api/api.go:279-299`, `internal/codebase/analyzer.go:95-179`, `web/src/lib/dashboard.ts:982-1099`

## Suggested Expansion Themes

1. Project pattern profiles
   - Detect and explain patterns such as GitOps, monorepo, full-stack, platform-heavy service, AI/MCP-enabled workspace, and ADR-backed architecture.

2. Anti-pattern warnings
   - Start with deterministic warnings the current scan model can justify, such as:
     - hold technology still active
     - high untracked concentration
     - polyglot sprawl without clear standards
     - fragmented AI/MCP adoption
     - duplicated stack divergence across similar projects

3. Alternative suggestions
   - Every warning should point to one of:
     - an existing `adopt` radar standard
     - a `trial` candidate
     - a new radar entry suggestion
     - “document the exception” when there is no good substitute

4. Source-backed diagnostics
   - Show the exact manifest/config/ADR evidence that triggered the finding.

## Sources

- [S1] `README.md:9-22`
- [S2] `.tech-radar/config.yaml:1-69`
- [S3] `internal/scanner/scanner.go:64-85`
- [S4] `internal/scanner/scanner.go:384-541`
- [S5] `internal/api/api.go:226-299`
- [S6] `internal/db/db.go:139-225`
- [S7] `internal/db/insights.go:89-210`
- [S8] `internal/codebase/analyzer.go:21-179`
- [S9] `internal/codebase/analyzer.go:241-355`
- [S10] `internal/codebase/analyzer_test.go:90-135`
- [S11] `internal/codebase/analyzer_test.go:232-254`
- [S12] `pkg/types/codebase.go:6-70`
- [S13] `pkg/types/types.go:107-193`
- [S14] `infrastructure/radar/schema.sql:41-49`
- [S15] `web/src/App.svelte:1522-1655`
- [S16] `web/src/App.svelte:1717-1915`
- [S17] `web/src/App.svelte:1955-2075`
- [S18] `web/src/lib/dashboard.ts:31-44`
- [S19] `web/src/lib/dashboard.ts:808-847`
- [S20] `web/src/lib/dashboard.ts:982-1099`
- [S21] `go test ./...`
- [S22] `pnpm check`
- [S23] `pnpm build`
