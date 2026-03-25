# MCP Inventory

## Runtime Mode Detection

- Detection method: `list_mcp_resources`, `list_mcp_resource_templates`, and `read_mcp_resource(server="loom", uri="loom://config|servers|tools/index")`.
- Loom-mode detected: `yes`.
- Evidence:
  - `loom://config` reports active profile `full`, `serverCount: 46`, and `toolCount: 498`.
  - `loom://servers` returns the managed server inventory through the loom proxy.
  - `loom://tools/index` reports `pageSize: 100`, `totalTools: 498`, and `totalPages: 5`.

## Server Inventory

- Total servers reported by loom: `46`.
- Relevant planning/delivery servers for this repo:
  - `codebase_memory`: semantic/lexical code index and code navigation.
  - `git` and `gitlab`: local SCM plus GitLab MR/pipeline operations.
  - `quality`: lint/test/security/coverage gates for Go packages.
  - `browserkit`: screenshot capture for UI verification.
  - `devbox`: sandbox execution when local toolchains drift.
  - `context7` and `tavily`: external documentation and research.
  - `agent_context`: session/task/handoff persistence.

## Tool Inventory Summary

- Inventory source: loom proxy resources plus CLI fallback `loom tools list --json`.
- Total tools: `498`.
- Count by relevant server:
  - `agent_context`: `59`
  - `codebase_memory`: `17`
  - `gitlab`: `33`
  - `git`: `11`
  - `devbox`: `11`
  - `quality`: `6`
  - `browserkit`: `1`
  - `context7`: `2`
  - `tavily`: `4`
- Full server-count summary was derived with:
  - `loom tools list --json | jq '.tools | map(.name | split("__")[0]) | group_by(.) | map({server: .[0], count: length}) | sort_by(.server)'`

## Paged Deep-Dive Capture

- Paged strategy:
  - Used `loom://tools/index` to establish global pagination (`totalPages: 5`).
  - Sampled server-specific pages for the tools most relevant to this repo:
    - `loom://tools/server/codebase_memory/page/1`
    - `loom://tools/server/gitlab/page/1`
    - `loom://tools/server/quality/page/1`
    - `loom://tools/server/browserkit/page/1`
- Deep-dive findings:
  - `codebase_memory` exposes indexing, polling, lexical search, definition/context, call graph, and module graph flows.
  - `gitlab` exposes MR, pipeline, job, artifact, and merge automation flows.
  - `quality` exposes lint, test, security, coverage, and arch checks.
  - `browserkit` currently exposes a single `screenshot` tool.

## Auth / Permissions / Constraints

- Loom health is broadly good, but `gitlab` hub health shows intermittent lock timeout while the local target remains healthy enough for normal operations.
- Initial codebase index with embeddings failed because `api.morphllm.com` could not be resolved:
  - Error excerpt: `flush chunks: Post "https://api.morphllm.com/v1/embeddings": dial tcp: lookup api.morphllm.com: no such host`
- Fallback index with `embeddings: false` completed successfully:
  - `repo_id: tech-radar`
  - `files_total: 15`
  - `chunks_total: 117`
  - `status: done`
- Bundled planning helpers are currently broken by merge markers:
  - `python /Users/cblevins/.codex/skills/plan-loom-core/scripts/init_loom_context.py --root .`
  - `python /Users/cblevins/.codex/skills/plan-loom-core/scripts/workspace_snapshot.py --root .`
  - Both fail with `SyntaxError` at `<<<<<<< Updated upstream`.

## Codebase Index Readiness

- Current recommended mode: `usable with lexical/no-embedding search`.
- Working evidence:
  - `codebase_memory__codebase_stats(repo_id="tech-radar")` reports `117` chunks:
    - Go: `86`
    - TypeScript: `30`
    - JavaScript: `1`
  - `codebase_text_search(query="GetInsights")` resolves `internal/api/api.go` and `internal/db/db.go`.
  - `codebase_text_search(query="computeDashboardModel")` resolves `web/src/lib/dashboard.ts`.

## Delegation Plan Used and Why

- Strategy used: single coordinator plus parallel read-only calls.
- Why:
  - The repo is small enough that sub-agents were unnecessary.
  - MCP inventory, file inspection, and codebase polling are naturally parallel and low-risk.
- Residual risk:
  - The planning pack is accurate for the current environment, but helper-script breakage and embedding DNS issues should be treated as environment debt.

## Notes

- For this repo, the most useful MCP path is:
  1. `codebase_memory` for navigation/search
  2. `quality` for Go validation
  3. `gitlab` for CI/MR lifecycle
  4. `browserkit` for UI proof when the local screenshot dependency chain is healthy

## Sources

- [S1] `loom://config`
- [S2] `loom://servers`
- [S3] `loom://tools/index`
- [S4] `loom://tools/server/codebase_memory/page/1`
- [S5] `loom://tools/server/gitlab/page/1`
- [S6] `loom://tools/server/quality/page/1`
- [S7] `loom://tools/server/browserkit/page/1`
- [S8] `loom://health`
- [S9] `loom tools list --json | jq '.tools | map(.name | split("__")[0]) | group_by(.) | map({server: .[0], count: length}) | sort_by(.server)'`
- [S10] `codebase_memory__codebase_index_poll(job_id="aaa84db9975c352e")`
- [S11] `codebase_memory__codebase_index_poll(job_id="44e3d6e56d4829b7")`
- [S12] `codebase_memory__codebase_stats(repo_id="tech-radar")`
- [S13] `codebase_memory__codebase_text_search(repo_id="tech-radar", query="GetInsights")`
- [S14] `codebase_memory__codebase_text_search(repo_id="tech-radar", query="computeDashboardModel")`
- [S15] `python /Users/cblevins/.codex/skills/plan-loom-core/scripts/init_loom_context.py --root .`
- [S16] `python /Users/cblevins/.codex/skills/plan-loom-core/scripts/workspace_snapshot.py --root .`
