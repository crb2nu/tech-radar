# Product Spec

## Summary

Expand Tech Radar into a **Decision Workbench**: a product that helps someone understand where a technology is showing up, why it matters, what evidence supports a decision, which projects are affected, and what follow-up action is needed next.

This is not a “replace the radar” proposal. The radar remains the portfolio map. The workbench adds the missing depth behind it so the app becomes useful instead of merely informative.

## Product Goals

- Turn aggregate signals into drill-downable decision context.
- Preserve and surface evidence from scans so each technology can be inspected like a dossier.
- Make project-level alignment actionable, not just scored.
- Create a durable action queue for governance work instead of ephemeral recommendation rows.
- Keep future discovery work, including AI/MCP expansion, as a multiplier on top of a stronger foundation.

## Non-Goals

- Replacing the existing four-ring radar model.
- Building a full enterprise architecture platform or ticketing system.
- Requiring external SaaS enrichment or live internet lookups for core functionality.
- Solving every discovery gap in the same milestone.

## Users / Stakeholders

- Architecture owner / maintainer who sets standards and reviews exceptions.
- Project owner who needs to understand why their project is “risky” and what to do next.
- Future agent/operator who needs source-backed context for a technology decision without rereading multiple repos manually.

## User Problems

1. “I can see that something is uncharted, but I can’t inspect the evidence behind it.”
2. “I can see a project score, but I can’t tell which exact technologies are causing the score.”
3. “I can see a recommendation, but I can’t work it as a real piece of governance follow-up.”
4. “I can see a linked ADR, but I can’t tell how it connects to project adoption over time.”

## Requirements

### Functional

1. Evidence persistence
   - Preserve scanner evidence through the DB-backed ingestion path, not just in YAML-first flows.
   - Retain enough provenance to answer where a technology was seen, from what source, and with what supporting file/title.
   - Support multiple observations per project/technology when needed.

2. Technology dossier
   - Selecting a radar entry should open a useful dossier:
     - affected projects
     - observed versions and sources
     - linked ADR/evidence files
     - movement history
     - current governance recommendation
   - The dossier should explain why the technology is in its current ring and where its pressure is coming from.

3. Project drill-down
   - Selecting a project should reveal:
     - tracked vs untracked technologies
     - hold exposure
     - technologies likely ready for charting
     - evidence/source breakdown for each important technology
   - The project view should answer “what do we need to fix here?” rather than just “what is the score?”

4. Action queue
   - The app needs a durable notion of governance follow-up, even if lightweight at first.
   - Action items should link to the exact technology/project context that generated them.
   - Actions should support at least a status model such as `new`, `reviewing`, `decided`, `deferred`.

5. Portfolio change view
   - Keep the radar and top-level overview, but shift them toward change and focus:
     - what changed since last scan
     - new technologies entering the estate
     - projects whose risk materially changed
     - decisions lacking evidence

6. Discovery expansion compatibility
   - AI/MCP discovery and other future scan expansions should feed the same dossier/action model.
   - New discovery categories should not require inventing another special-case dashboard lane to become useful.

7. Demo/onboarding quality
   - The default experience should not look empty.
   - Seed/sample data or onboarding guidance should demonstrate the richer workflow with more than three entries.

### Non-Functional

- Preserve deterministic, repo-local scanning.
- Keep DB-backed and YAML-first behavior conceptually aligned, even if the richer workbench is DB-first.
- Avoid regressing the existing radar visualization.
- Maintain green validation with `go test ./...`, `pnpm check`, and `pnpm build`.

## UX / Core Flows

1. Investigate an uncharted technology
   - User sees `Chart X` in the action queue.
   - They open the technology dossier.
   - The dossier shows which projects use it, which files/configs/ADRs surfaced it, and whether usage is growing.
   - User records a decision or defers with rationale.

2. Review a risky project
   - User opens a project card from the portfolio view.
   - The project drill-down shows hold technologies, untracked technologies, evidence sources, and candidate standards.
   - User can jump directly to the relevant technology dossiers.

3. Audit a technology decision
   - User opens a radar entry from the radar map or action queue.
   - They see ring history, adoption footprint, evidence links, and open follow-ups in one place.
   - If the technology is drifting or heating up, the page explains the pressure rather than only showing a badge.

## Data / API Shape

The current codebase already has most of the raw building blocks but not the right persistence model.

- Today:
  - Scanner captures evidence. `pkg/types/types.go:115-123`, `internal/scanner/scanner.go:371-395`
  - DB ingestion drops it. `internal/db/db.go:192-201`, `infrastructure/radar/schema.sql:41-47`
  - Project/radar APIs return shallow project technology data and summary analytics. `internal/api/api.go:161-215`, `internal/db/db.go:112-159`

- Needed:
  - Extend project-technology persistence to retain evidence and observation metadata.
  - Add detail endpoints or equivalent API shapes for project and entry drill-downs.
  - Introduce a lightweight persisted action model or a clearly materialized derived-action store.

## Rollout

- Phase 1:
  - Evidence persistence and detail-oriented API shapes.
- Phase 2:
  - Technology dossiers and project drill-down UI.
- Phase 3:
  - Durable action queue and change-focused overview.
- Phase 4:
  - Discovery expansions such as richer AI/MCP coverage plugged into the same model.

Backout:
- If the richer workbench slice proves too large, ship Phase 1 independently because evidence persistence is valuable on its own.

## Risks

- Schema/API changes may create divergence between YAML-first local mode and DB-backed mode if not designed carefully.
- Persisting more observations can make scans noisier unless the UI is opinionated about surfacing only the most relevant evidence.
- Adding a durable action model can sprawl if we try to build project management instead of lightweight governance follow-up.

## Open Questions

- Should action items be persisted in the product database immediately, or materialized from deterministic rules first and persisted later?
- Do we want one generic evidence model, or explicit observation types such as `dependency`, `config`, `adr`, and `manual`?
- Should project drill-downs be modal/drawer-based or full-page routes?

## Sources

- [S1] `README.md:5-22`
- [S2] `README.md:91-165`
- [S3] `.tech-radar/radar.yaml:1-24`
- [S4] `pkg/types/types.go:115-123`
- [S5] `internal/scanner/scanner.go:371-395`
- [S6] `internal/db/db.go:112-201`
- [S7] `internal/api/api.go:161-215`
- [S8] `infrastructure/radar/schema.sql:3-65`
- [S9] `web/src/App.svelte:341-363`
- [S10] `web/src/App.svelte:455-767`
- [S11] `web/src/App.svelte:943-1243`
- [S12] `web/src/App.svelte:1335-1405`
- [S13] `web/src/lib/dashboard.ts:307-447`
- [S14] `web/src/lib/dashboard.ts:932-1049`
