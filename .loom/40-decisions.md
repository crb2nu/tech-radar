# Decisions

Record decisions as they are made, with date, rationale, and sources.

## 2026-03-19: Shift the next slice from discovery-only to Decision Workbench foundations

- Decision:
  - Reposition the next roadmap slice around evidence persistence, drill-downs, and action workflows rather than treating AI/MCP discovery as the sole next milestone.
- Rationale:
  - The current UI shell is already broader than a basic dashboard, but most of its value still collapses into summaries and count-derived queues.
  - The scanner already captures richer evidence than the DB-backed product preserves, so usefulness is now gated more by persistence and workflow depth than by another category of discovery.
- Alternatives considered:
  - Continue with AI/MCP discovery as the primary next slice.
  - Add more visual polish and charts without changing the underlying model.
- Consequences:
  - AI/MCP discovery remains important, but becomes a follow-on amplifier on top of a better workbench foundation.
  - The implementation plan now prioritizes schema/API/UI changes that unlock dossiers, project drill-downs, and durable action context.
- Sources:
  - [S1] `README.md:5-22`
  - [S2] `pkg/types/types.go:115-123`
  - [S3] `internal/db/db.go:192-201`
  - [S4] `web/src/App.svelte:455-767`
  - [S5] `web/src/App.svelte:1335-1405`
  - [S6] `web/src/lib/dashboard.ts:932-1049`

## 2026-03-18: Use loom-mode as the MCP inventory baseline

- Decision:
  - Treat loom proxy resources (`loom://config`, `loom://servers`, `loom://tools/*`) as the primary MCP inventory source for this repo.
- Rationale:
  - Loom-mode is active, exposes the full managed server inventory, and supports paged tool inventory.
- Alternatives considered:
  - Direct per-server resource enumeration only.
  - CLI-only inventory without loom resource evidence.
- Consequences:
  - Planning docs can cite one consistent inventory path.
  - Server-specific deep dives still happen where needed.
- Sources:
  - [S1] `loom://config`
  - [S2] `loom://servers`
  - [S3] `loom://tools/index`

## 2026-03-18: Accept no-embedding codebase indexing as the current planning baseline

- Decision:
  - Use `codebase_memory` in `embeddings: false` mode until Morph embedding DNS is healthy again.
- Rationale:
  - The embedding-backed index failed on `api.morphllm.com` resolution, but no-embedding indexing completed and lexical search is already sufficient for planning/navigation.
- Alternatives considered:
  - Block planning until embeddings are restored.
  - Ignore codebase-memory entirely and rely only on shell/file inspection.
- Consequences:
  - We retain repo search readiness with reduced semantic quality.
  - Future planning should revisit full embedding mode when network conditions improve.
- Sources:
  - [S1] `codebase_memory__codebase_index_poll(job_id="aaa84db9975c352e")`
  - [S2] `codebase_memory__codebase_index_poll(job_id="44e3d6e56d4829b7")`
  - [S3] `codebase_memory__codebase_stats(repo_id="tech-radar")`

## 2026-03-18: Plan AI / MCP Discovery as the provisional next roadmap slice

- Decision:
  - Treat AI/MCP discovery as the next recommended feature slice after the rich dashboard/intelligence merge.
- Rationale:
  - The dashboard and backend insights are now rich enough to consume better upstream signals, while scan coverage remains narrow and the roadmap already lists AI/MCP discovery first.
- Alternatives considered:
  - ADR integration first.
  - Motion vectors first.
- Consequences:
  - This recommendation is now superseded by the 2026-03-19 decision to prioritize Decision Workbench foundations first.
  - AI/MCP discovery remains a follow-on slice once evidence and workflow depth improve.
- Sources:
  - [S1] `plan.md:1-13`
  - [S2] `.tech-radar/config.yaml:1-15`
  - [S3] `internal/scanner/scanner.go:92-105`
  - [S4] `internal/db/insights.go:26-171`
  - [S5] `web/src/lib/dashboard.ts:203-260`
