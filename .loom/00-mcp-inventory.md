# MCP Inventory

## Runtime Mode Detection

- Detection method:
  - `list_mcp_resources`
  - `list_mcp_resource_templates`
  - `read_mcp_resource(server="loom", uri="loom://config")`
  - `read_mcp_resource(server="loom", uri="loom://servers")`
  - `read_mcp_resource(server="loom", uri="loom://tools/index")`
- Loom-mode detected: `yes`
- Evidence:
  - `loom://config` reports active profile `full`, `serverCount: 46`, and `toolCount: 498`.
  - `loom://tools/index` reports `totalTools: 498` and `totalPages: 5`.

## Relevant Servers For This Repo

- `codebase_memory`
  - Current index for `repo_id="tech-radar"` is present and queryable.
- `agent_context`
  - Available if we want durable planning or handoff context.
- `gitlab`
  - Available for future MR / pipeline workflows.
- `devbox`
  - Available as a sandbox fallback for builds/tests.
- `quality`
  - Available for Go-focused lint/test/security checks if we want a structured quality gate.

## Current Planning Baseline

- Helper scripts are healthy in this environment:
  - `python /Users/cblevins/.codex/skills/plan-loom-core/scripts/workspace_snapshot.py --root .`
  - `python /Users/cblevins/.codex/skills/plan-loom-core/scripts/init_loom_context.py --root .`
- Codebase index readiness:
  - `codebase_memory__codebase_stats(repo_id="tech-radar")` reports `117` chunks.
  - Language split:
    - Go: `86`
    - TypeScript: `30`
    - JavaScript: `1`
- Local validation baseline on `2026-03-29`:
  - `go test ./...` passed.
  - `pnpm check` passed with `0 errors and 0 warnings`.
  - `pnpm build` passed.

## Constraints / Notes

- This repo’s most useful MCP path for planning is:
  1. `codebase_memory` for navigation and verification
  2. loom inventory resources for environment awareness
  3. local shell commands for line-precise source gathering
- I did not need sub-agents for this planning pass because the repo is small and the work was mostly read-only inventory plus document synthesis.
- I did not re-verify full paged server-by-server tool counts in this pass because the top-level loom inventory and current repo tasks were already sufficient.

## Sources

- [S1] `list_mcp_resources`
- [S2] `list_mcp_resource_templates`
- [S3] `loom://config`
- [S4] `loom://servers`
- [S5] `loom://tools/index`
- [S6] `codebase_memory__codebase_stats(repo_id="tech-radar")`
- [S7] `python /Users/cblevins/.codex/skills/plan-loom-core/scripts/workspace_snapshot.py --root .`
- [S8] `python /Users/cblevins/.codex/skills/plan-loom-core/scripts/init_loom_context.py --root .`
- [S9] `go test ./...`
- [S10] `pnpm check`
- [S11] `pnpm build`
