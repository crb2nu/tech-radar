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

- Reframe `/Users/cblevins/workspace/services/tech-radar` from a passive intelligence dashboard into a useful decision workspace.
- Capture the current architectural limits, then define the next implementation slice so we can build workflows, evidence, and drill-downs instead of more top-line counters.

## Success Criteria

- The planning baseline reflects the current `main` branch and current MCP/index environment.
- Research clearly explains why the current experience still feels thin despite the richer UI shell.
- The product spec defines user-facing value around decision-making, not just telemetry display.
- The implementation plan identifies a concrete first slice that can be executed without rediscovery.

## Open Questions

- Should the first execution slice stop at evidence persistence plus drill-downs, or also include lightweight ownership/campaign tracking?
- Do we want to preserve YAML-first and DB-backed modes symmetrically, or accept a DB-first path for the richer experience?
- Should fixing the broken `plan-loom-core` helper scripts in `$CODEX_HOME` happen alongside product work or as separate tooling debt?

## Risks

- The bundled `plan-loom-core` helper scripts currently contain merge markers and cannot be trusted as-is.
- The current DB ingestion path drops scanner evidence fields, which can make a richer governance UI impossible until fixed.
- Default/sample radar data is still extremely sparse, so the product can feel empty even when the UI shell is capable.
- Codebase indexing with embeddings is still unavailable; lexical/no-embedding index mode is the current baseline.
