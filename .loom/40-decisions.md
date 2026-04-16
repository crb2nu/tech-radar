# Decisions

Record decisions as they are made, with date, rationale, and sources.

## 2026-03-29: Prioritize Project Pattern Intelligence as the next expansion

- Decision:
  - The next major product slice should focus on project pattern identification, anti-pattern warnings, and alternative suggestions rather than another round of dashboard polish.
- Rationale:
  - The app already has a broad visual shell and a starter pattern analyzer, but it still cannot tell a user what architecture shape a project is in, what is risky about it, or what the better default should be.
  - This is now the clearest path to making the app more useful in day-to-day governance work.
- Alternatives considered:
  - Keep expanding summary cards and charts.
  - Add warnings on top of the current shallow project model without fixing the underlying observation shape.
- Consequences:
  - The implementation plan now centers on richer observations, rule-backed project findings, and UI surfaces that explain warnings and alternatives.
  - Existing radar and codebase views stay in place; they become launch points for guidance rather than the whole product.
- Sources:
  - [S1] `README.md:9-22`
  - [S2] `internal/codebase/analyzer.go:21-179`
  - [S3] `web/src/App.svelte:1522-1655`
  - [S4] `web/src/App.svelte:1696-1915`
  - [S5] `web/src/lib/dashboard.ts:982-1099`

## 2026-03-29: Keep the first finding engine deterministic and evidence-backed

- Decision:
  - The first pattern/anti-pattern engine should stay deterministic and rely on manifest/config/ADR evidence already available from scans.
- Rationale:
  - The scanner and DB already capture reproducible signals such as source file type, evidence path, and ADR references.
  - Deterministic findings will be easier to trust, test, and explain than an early probabilistic or LLM-first warning system.
- Alternatives considered:
  - Introduce AI-generated architecture warnings immediately.
  - Wait for deeper source-code parsing before shipping any guidance.
- Consequences:
  - The first rule catalog will be narrower, but much more explainable.
  - Future AI-assisted guidance can layer on top of a strong deterministic baseline instead of replacing it.
- Sources:
  - [S1] `internal/scanner/scanner.go:64-85`
  - [S2] `internal/scanner/scanner.go:384-541`
  - [S3] `internal/db/db.go:139-225`
  - [S4] `web/src/lib/dashboard.ts:31-44`

## 2026-03-29: Replace one-row-per-project-tech storage with richer observations

- Decision:
  - The current `project_technologies` shape should no longer be the sole backing model for project intelligence.
- Rationale:
  - `PRIMARY KEY (project_id, tech_name)` prevents storing multiple observations for the same technology in a single project.
  - That blocks higher-confidence findings that need to combine manifest, ADR, and config evidence.
- Alternatives considered:
  - Keep the existing table and collapse multiple observations into one synthesized row.
  - Build warnings only from global aggregates without richer observations.
- Consequences:
  - Schema and DB work move to the front of the implementation order.
  - The app will be able to justify findings with multiple pieces of evidence rather than one collapsed summary row.
- Sources:
  - [S1] `infrastructure/radar/schema.sql:41-49`
  - [S2] `internal/db/db.go:184-225`
  - [S3] `pkg/types/types.go:115-123`

## 2026-03-18: Use loom-mode as the MCP inventory baseline

- Decision:
  - Treat loom proxy resources (`loom://config`, `loom://servers`, `loom://tools/*`) as the primary MCP inventory source for this repo.
- Rationale:
  - Loom-mode is active, exposes the managed server inventory, and is sufficient for planning inventory.
- Consequences:
  - Planning docs can cite one consistent inventory path.
- Sources:
  - [S1] `loom://config`
  - [S2] `loom://servers`
  - [S3] `loom://tools/index`
