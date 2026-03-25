# Research Brief

## Problem

The current app is no longer a literal one-page dashboard, but it still feels underwhelming because most of the experience collapses into counts, ranked lists, and shallow heuristics. The repo needs a next slice that makes the product genuinely useful for technology governance work, not just more visually polished telemetry.

## Questions

- Q1: Why does the current product still feel like “a dashboard of counts” even after the richer UI merge?
- Q2: Which architectural limits keep the product from becoming a decision workspace?
- Q3: What is the highest-leverage next slice if the goal is usefulness rather than more visual summary?

## Constraints

- The bundled `plan-loom-core` helper scripts are broken by merge markers and cannot be trusted for automation.
- `codebase_memory` is currently usable only in no-embedding mode.
- Search still depends on Meilisearch at runtime.
- Ingestion still lacks API-key validation.

## Method

- Read the product framing in `README.md`.
- Reviewed the current dashboard shell, dashboard derivation logic, scanner, DB schema, and API layers.
- Compared YAML-first radar behavior with DB-backed server behavior.
- Verified loom-mode MCP inventory and current codebase index readiness.

## Findings

1. The UI shell is broader, but most surfaces still reduce to counts or count-derived summaries.
   - The sidebar snapshot, hero metrics, portfolio balance, AI/MCP watch, opportunity queue, adoption leaders, drift exposure, and motion panels are all rendered as aggregate cards, bars, or ranked rows. `web/src/App.svelte:341-363`, `web/src/App.svelte:455-767`, `web/src/App.svelte:997-1243`
   - `computeDashboardModel` centers the experience on `summaryMetrics`, `breakdowns`, `opportunityQueue`, `driftLeaders`, `projectSpotlight`, and other summarized collections. `web/src/lib/dashboard.ts:307-447`
   - Recommendation generation is based on slicing top-N lists (`slice(0, 3)`, `slice(0, 4)`, `slice(0, 2)`) rather than operating on persisted tasks, workflows, or campaigns. `web/src/lib/dashboard.ts:952-1049`

2. The repo already captures richer evidence than the product actually preserves.
   - `DiscoveredTech` includes `EvidencePath` and `EvidenceTitle`. `pkg/types/types.go:115-123`
   - The scanner fills those fields for ADR discoveries and merges them across duplicate detections. `internal/scanner/scanner.go:371-395`, `internal/scanner/scanner.go:589-605`
   - But the DB schema for `project_technologies` stores only `tech_name`, `version`, and `source`, and `SaveScan` writes only those fields. `infrastructure/radar/schema.sql:41-47`, `internal/db/db.go:192-201`
   - That means the evidence needed for a useful dossier-level experience is lost in DB-backed mode.

3. Detail views are still shallow for both technologies and projects.
   - The selected-entry modal shows project names, a motion summary, tags, history, and one link, but not versions by project, scan sources, evidence files, first/last seen dates, exceptions, owners, or migrations. `web/src/App.svelte:1335-1405`
   - `ProjectHealth` only exposes totals, counts by ring, a score, risk level, and a short list of top tech names. `pkg/types/types.go:166-182`
   - The projects page renders those shallow fields directly and does not open a deeper project-level diagnostic view. `web/src/App.svelte:943-993`

4. The current system has no durable concept of an action, campaign, or decision workflow.
   - Recommendations are ephemeral derived objects created in memory. `web/src/lib/dashboard.ts:149-160`, `web/src/lib/dashboard.ts:932-1049`
   - The schema has tables for radars, entries, history, projects, and project technologies, but nothing for owners, migration plans, exceptions, tasks, due dates, or review checkpoints. `infrastructure/radar/schema.sql:3-65`
   - This is why the app can tell us “chart X” or “contain Y drift” without giving us a place to work the problem.

5. There is a product-path mismatch between YAML-first and server-backed modes.
   - The CLI `addToRadar` path uses scanner evidence to populate entry descriptions, links, and ADR tags in local YAML. `cmd/radar/main.go:383-469`
   - The server-backed APIs return DB-backed radar/projects/insights, where project evidence is no longer persisted. `internal/api/api.go:161-215`, `internal/db/db.go:112-201`
   - The frontend falls back to `/api/radar` only for the radar payload, not for projects or insights, so the “rich” experience implicitly depends on the DB-backed path. `web/src/App.svelte:154-174`

6. Sparse seed data amplifies the feeling of emptiness.
   - The default radar has only three entries. `.tech-radar/radar.yaml:1-24`
   - Many “advanced” panels therefore degrade into empty states or tiny ranked lists, which makes the product feel conceptually thin even when the UI is visually ambitious.

## Options

### Option A: Continue with AI/MCP discovery as the next primary slice

- Pros:
  - Improves upstream signal breadth.
  - Builds on the recent AI/MCP framing already present in config and insights. `.tech-radar/config.yaml:19-51`, `internal/db/insights.go:26-51`
- Cons:
  - Still leaves the product mostly as a better telemetry layer.
  - Risks producing more items in queues without a better place to investigate or act on them.

### Option B: Build Decision Workbench foundations first

- Pros:
  - Directly addresses the “underwhelming” problem by making the app useful for actual decision work.
  - Unlocks richer use of the evidence the scanner already knows how to capture.
  - Makes future discovery expansions more valuable because new signals can feed dossiers and action flows instead of just counts.
- Cons:
  - Requires schema/API/UI work, not just scanner/config expansion.
  - Is a broader slice than a pure discovery improvement.

### Option C: Add more visual polish and charts on top of the current model

- Pros:
  - Fastest to ship.
  - Could improve first impressions.
- Cons:
  - Does not fix the structural reason the experience feels shallow.
  - Risks over-investing in presentation without improving decision support.

## Recommendation

Choose **Option B: Decision Workbench foundations first**.

Rationale:
- The product promise in `README.md` is about decisions, documentation, and cross-project analysis, not just telemetry. `README.md:5-22`
- The current gaps are primarily in persisted evidence, drill-down depth, and action workflows, not the absence of more scorecards. `internal/db/db.go:192-201`, `web/src/App.svelte:1335-1405`, `web/src/lib/dashboard.ts:952-1049`
- AI/MCP discovery should become a follow-on amplifier for a more useful foundation, not the only story.

## Sources

- [S1] `README.md:5-22`
- [S2] `.tech-radar/config.yaml:19-51`
- [S3] `.tech-radar/radar.yaml:1-24`
- [S4] `pkg/types/types.go:115-123`
- [S5] `pkg/types/types.go:166-182`
- [S6] `internal/scanner/scanner.go:371-395`
- [S7] `internal/scanner/scanner.go:589-605`
- [S8] `internal/api/api.go:161-215`
- [S9] `internal/db/db.go:112-201`
- [S10] `internal/db/insights.go:26-51`
- [S11] `infrastructure/radar/schema.sql:3-65`
- [S12] `cmd/radar/main.go:383-469`
- [S13] `web/src/App.svelte:341-363`
- [S14] `web/src/App.svelte:455-767`
- [S15] `web/src/App.svelte:943-993`
- [S16] `web/src/App.svelte:997-1243`
- [S17] `web/src/App.svelte:1335-1405`
- [S18] `web/src/lib/dashboard.ts:307-447`
- [S19] `web/src/lib/dashboard.ts:932-1049`
- [S20] `python /Users/cblevins/.codex/skills/plan-loom-core/scripts/workspace_snapshot.py --root .`
- [S21] `read_mcp_resource(server="loom", uri="loom://config")`
- [S22] `codebase_memory__codebase_stats(repo_id="tech-radar")`
