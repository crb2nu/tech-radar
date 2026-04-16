# Product Spec

## Summary

Build **Project Pattern Intelligence** into Tech Radar.

The app should not only show what technologies exist across projects, but also identify the architectural patterns each project is exhibiting, flag anti-patterns or risky combinations, and recommend source-backed alternatives.

This is an expansion of the existing radar, insights, and codebase experience, not a replacement for it. The radar remains the map; pattern intelligence becomes the guidance layer. `README.md:9-22`, `web/src/App.svelte:1696-1915`

## Product Goals

- Identify meaningful design patterns at the project level.
- Flag anti-patterns and risky architecture combinations with explicit rationale.
- Show the evidence behind each finding.
- Suggest safer defaults, tracked standards, or next-best alternatives.
- Preserve deterministic, repo-local behavior so users can trust and reproduce findings.

## Non-Goals

- Replacing the radar metaphor or ring model.
- Building a full static-analysis platform or language-specific AST engine in the first slice.
- Making autonomous architecture decisions without evidence or human review.
- Turning the app into a generic ticketing or project-management system.

## Users

- Architecture owner maintaining standards across many repos.
- Project maintainer trying to understand why a project is risky or inconsistent.
- Future agent/operator needing evidence-backed reasoning for a recommendation.

## User Problems

1. “I can see that a project is risky, but not what architecture pattern it falls into.”
2. “I can see technologies, but not whether the combination is sensible or fragmented.”
3. “I can see drift or uncharted usage, but not what the preferred alternative should be.”
4. “I can see a pattern badge, but not what evidence triggered it or how confident the app is.”

## Functional Requirements

1. Project finding model
   - The system must produce first-class project findings, not only aggregate stats.
   - Each finding should include at least:
     - `id`
     - `kind` (`pattern` or `anti_pattern`)
     - `title`
     - `summary`
     - `severity`
     - `confidence`
     - `project`
     - `evidence`
     - `alternatives`

2. Pattern catalog
   - Extend the current positive pattern heuristics into a richer catalog.
   - Preserve existing patterns where they still make sense:
     - GitOps
     - MCP-enabled
     - AI/ML pipeline
     - Containerized
     - Monorepo
     - Full-stack `internal/codebase/analyzer.go:21-93`
   - Add additional project-level patterns that are still deterministically inferable from current scan inputs.

3. Anti-pattern catalog
   - Introduce a paired anti-pattern catalog with explicit severity and remediation hints.
   - Initial anti-pattern candidates should stay grounded in current scan evidence, for example:
     - hold technology still active
     - high untracked concentration
     - fragmented AI/MCP tooling
     - excessive polyglot spread
     - stack divergence between otherwise similar projects

4. Evidence-backed explanations
   - Every finding must show the observations that triggered it.
   - Observation detail should include real evidence paths when available, not just manifest type labels.
   - The UI should let a user answer “why did you say this?” quickly.

5. Alternative suggestions
   - Findings should recommend the next best action:
     - use an `adopt` standard
     - evaluate a `trial` option
     - create a new radar entry
     - document an exception
   - Suggestions should prefer tracked radar entries where possible.

6. Project drill-down
   - Each project should have an intelligence view that combines:
     - detected patterns
     - anti-pattern warnings
     - relevant technologies
     - supporting evidence
     - recommended alternatives

7. Portfolio-level pattern view
   - The app should show which patterns are common, which anti-patterns are spreading, and which projects need the next governance intervention.

8. Rule authoring path
   - The system should allow a future path toward repo-local rule authoring.
   - A good first step is Go-defined defaults with a clear extension seam for `.tech-radar/patterns.yaml`.

## Non-Functional Requirements

- Deterministic by default.
- Explainable with source-backed evidence.
- Backward-compatible with the existing radar/dashboard shell.
- Safe against noisy over-warning by supporting severity and confidence.
- Green validation on:
  - `go test ./...`
  - `pnpm check`
  - `pnpm build`

## UX / Core Flows

1. Review a project
   - User opens a project card.
   - They see pattern badges, anti-pattern warnings, severity, and evidence.
   - Each warning includes a suggested alternative or next step.

2. Inspect a warning
   - User opens a flagged anti-pattern.
   - The app shows which technologies and files triggered the rule.
   - The app explains why the current setup is risky and what standard to prefer instead.

3. Standardize a recurring project shape
   - User notices a pattern spreading across multiple repos.
   - The app suggests formalizing that pattern as an `adopt` or `trial` standard, or documenting it as an approved exception.

## Data / API Shape

### Current State

- The scanner can capture `EvidencePath` and `EvidenceTitle`. `pkg/types/types.go:115-123`, `internal/scanner/scanner.go:384-541`
- The DB now persists those fields for project technologies. `internal/db/db.go:139-160`, `internal/db/db.go:184-225`
- The codebase analyzer can emit portfolio pattern tags, but not per-project findings with severity or remediation. `internal/codebase/analyzer.go:154-179`, `pkg/types/codebase.go:43-49`
- The frontend project technology model does not carry evidence fields forward. `web/src/lib/dashboard.ts:31-44`

### Needed State

- A richer observation model that can represent multiple observations for the same technology in the same project.
- A project-finding model for positive and negative architecture signals.
- Project-detail and finding-detail API shapes.
- Alternative-suggestion payloads that can point to tracked radar entries or recommended next actions.

## Rollout

- Phase 1:
  - observation model + richer rule engine
- Phase 2:
  - project findings API and codebase-analysis enrichment
- Phase 3:
  - project intelligence UI with warnings and alternatives
- Phase 4:
  - configurable rule catalog and onboarding polish

## Risks

- Warning quality will suffer if we try to infer too much from one-row-per-tech project storage. `infrastructure/radar/schema.sql:41-49`
- Overly aggressive anti-pattern rules can produce alarm fatigue.
- Project intelligence could become visually heavy unless findings are ranked and grouped well.

## Open Questions

- Should findings be returned only from `/api/v1/codebase`, or should there be dedicated `/api/v1/projects/:id/intelligence` and `/api/v1/findings` endpoints?
- Should alternatives be free text first, or typed references to existing radar entries plus fallback text?
- Which first anti-patterns are valuable enough to ship without requiring deeper source-code parsing?

## Acceptance Criteria

- A user can open a project and see at least one clear architecture profile, any applicable anti-pattern warnings, and the evidence behind them.
- A warning includes a recommended alternative or next governance action.
- Findings are explainable and deterministic.
- Existing radar, insights, and codebase views still work.
- Validation remains green:
  - `go test ./...`
  - `pnpm check`
  - `pnpm build`

## Sources

- [S1] `README.md:9-22`
- [S2] `internal/scanner/scanner.go:384-541`
- [S3] `internal/db/db.go:139-225`
- [S4] `internal/codebase/analyzer.go:21-179`
- [S5] `pkg/types/codebase.go:43-49`
- [S6] `pkg/types/types.go:115-123`
- [S7] `infrastructure/radar/schema.sql:41-49`
- [S8] `web/src/App.svelte:1696-1915`
- [S9] `web/src/lib/dashboard.ts:31-44`
- [S10] `go test ./...`
- [S11] `pnpm check`
- [S12] `pnpm build`
