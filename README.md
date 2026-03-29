![Banner](assets/banner.png)

# Tech Radar

Personal technology radar for tracking tech decisions across projects.

## Overview

Inspired by [ThoughtWorks Tech Radar](https://www.thoughtworks.com/radar), this tool helps you:

- **Track Decisions**: Record which technologies you've adopted, are trialing, or have rejected
- **Auto-Discover**: Scan projects for dependencies and infer technology usage
- **Visualize**: Interactive radar visualization
- **Document**: Capture reasoning behind decisions
- **Explain Patterns**: Identify recurring project patterns and anti-patterns from scan evidence
- **Suggest Alternatives**: Point to tracked standards or safer defaults when a project looks risky

## Features

- **YAML-based Data**: Easy to edit and version control
- **Auto-population**: Scan package.json, go.mod, pyproject.toml, Cargo.toml, requirements.txt, and MCP config files
- **Interactive Radar**: SVG/HTML visualization
- **Decision Log**: Timestamp and reasoning for each entry
- **Cross-project Analysis**: Aggregate across multiple repos
- **Project Intelligence**: Surface deterministic pattern and anti-pattern warnings with evidence

## Quadrants

Technologies are organized into quadrants:

1. **Languages & Frameworks**: Python, Go, React, etc.
2. **Tools**: Docker, Terraform, K8s operators
3. **Platforms**: AWS, GCP, K8s, Cloudflare
4. **Techniques**: GitOps, TDD, Event sourcing

The project pattern intelligence work treats the same quadrants as a governance vocabulary:

- **Patterns** describe healthy shapes such as GitOps, monorepo, containerized delivery, or an intentionally chosen AI/MCP footprint.
- **Anti-patterns** flag combinations that deserve review, such as hold technologies still appearing in live projects, fragmented AI/MCP tooling, or stack drift between similar repos.
- **Alternatives** should prefer existing `adopt` radar entries, then `trial` candidates, and fall back to “document the exception” when there is no clean substitute.

## Rings

Each technology is placed in a ring:

- **Adopt**: Use by default for appropriate problems
- **Trial**: Worth pursuing; understand trade-offs
- **Assess**: Worth exploring; not yet recommended
- **Hold**: Proceed with caution; actively migrating away

## Installation

```bash
go install gitlab.flexinfer.ai/services/tech-radar/cmd/radar@latest
```

## Usage

### Initialize Radar

```bash
radar init
# Creates .tech-radar/ directory with config and data files
```

### Add Entry

```bash
# Interactive
radar add

# Or directly
radar add --name "React" --quadrant "frameworks" --ring "adopt" \
  --reason "Primary frontend framework"
```

### Scan Projects

```bash
# Scan current project
radar scan .

# Scan multiple projects
radar scan /path/to/project1 /path/to/project2

# Auto-add discovered technologies
radar scan . --auto-add
```

To publish scan results to a central radar server, use `--publish --api-url ... --api-key ...`. The CLI sends the API key as `Authorization: Bearer <token>` and also keeps `X-API-Key` for compatibility with older servers.

### Generate Visualization

```bash
# Generate static HTML
radar render --output radar.html

# Serve interactive version
radar serve --port 8088
```

## Data Format

```yaml
# .tech-radar/radar.yaml
meta:
  name: "Personal Tech Radar"
  updated: 2026-03-29

entries:
  - name: Go
    quadrant: languages
    ring: adopt
    description: Primary backend language for CLI and service work
    moved: 0
    tags: [backend, cli, services]
    history:
      - date: 2023-01-01
        ring: trial
        reason: Evaluating for CLI tools
      - date: 2023-06-01
        ring: adopt
        reason: Proven for services and CLIs

  - name: TypeScript
    quadrant: languages
    ring: adopt
    description: Frontend and lightweight tooling language
    moved: 0
    tags: [frontend, tooling]
    history:
      - date: 2024-02-01
        ring: trial
        reason: Shared language for browser UI and tooling
      - date: 2024-10-01
        ring: adopt
        reason: Consistent across the frontend and utility scripts

  - name: GitOps
    quadrant: techniques
    ring: adopt
    description: Declarative delivery and environment management through Git
    moved: 0
    tags: [delivery, operations, platform]
    history:
      - date: 2024-03-15
        ring: assess
        reason: Needed a repeatable deployment path
      - date: 2025-01-15
        ring: adopt
        reason: Stable Git-backed operations across environments

  - name: Model Context Protocol
    quadrant: tools
    ring: trial
    description: Useful for agent integrations, but still requires governance and standardization
    moved: 1
    tags: [ai, agents, integration]
    history:
      - date: 2025-05-01
        ring: assess
        reason: Experimenting with local agent tooling
      - date: 2026-01-10
        ring: trial
        reason: Several repos now use MCP configs or SDKs

  - name: Python 2
    quadrant: languages
    ring: hold
    description: Legacy runtime to migrate away from
    moved: 0
    tags: [legacy, migration]

  - name: Ruby on Rails
    quadrant: frameworks
    ring: assess
    description: Still plausible for a few services, but not a default choice
    moved: 0
    tags: [web, legacy]
```

## Configuration

```yaml
# .tech-radar/config.yaml
scan:
  # Files to scan for dependencies
  files:
    - package.json
    - go.mod
    - pyproject.toml
    - Cargo.toml
    - requirements.txt
    - .mcp.json
    - mcp.json
    - claude_desktop_config.json
    - cline_mcp_settings.json
    - .cursor/mcp.json
    - .continue/mcp.json
    - .roo/mcp.json
    - .windsurf/mcp.json

  # Mappings from package names to radar entries
  mappings:
    react: { name: React, quadrant: frameworks }
    typescript: { name: TypeScript, quadrant: languages }
    openai: { name: OpenAI API, quadrant: platforms }
    anthropic: { name: Claude API, quadrant: platforms }
    "@anthropic-ai/sdk": { name: Claude API, quadrant: platforms }
    mcp: { name: Model Context Protocol, quadrant: tools }
    fastmcp: { name: FastMCP, quadrant: tools }
    "@modelcontextprotocol/sdk": { name: Model Context Protocol, quadrant: tools }
    "github.com/mark3labs/mcp-go": { name: mcp-go, quadrant: tools }
    gitops: { name: GitOps, quadrant: techniques }
    argocd: { name: GitOps, quadrant: techniques }
    flux: { name: GitOps, quadrant: techniques }
    kustomize: { name: GitOps, quadrant: techniques }
    docker: { name: Containerized, quadrant: techniques }
    containerd: { name: Containerized, quadrant: techniques }

visualization:
  colors:
    adopt: "#93c47d"
    trial: "#93d2c2"
    assess: "#fbdb84"
    hold: "#efafa9"
```

The scanner understands common MCP config files as repo-local discovery signals, including configs stored in hidden editor folders such as `.cursor/`, `.vscode/`, `.claude/`, `.continue/`, `.roo/`, and `.windsurf/`, so AI/agentic tooling can show up in project scans even when it is configured outside a language manifest.
ADR markdown files under common decision directories such as `docs/adrs/`, `docs/adr/`, `adrs/`, and `decisions/` are also scanned automatically. When an ADR mentions a mapped technology, `radar scan --auto-add` can attach that ADR path directly to the resulting radar entry as decision evidence.

## Pattern Intelligence

The next product slice adds deterministic project pattern intelligence on top of the current radar and codebase views.

- Pattern identification should explain why a project looks like GitOps, a monorepo, a containerized service, or an AI/MCP-enabled workspace.
- Anti-pattern warnings should call out combinations that deserve review, such as hold technologies still in live use, fragmented AI/MCP tooling, or overly broad polyglot sprawl.
- Suggestions should prefer existing tracked standards first, then trial candidates, then a documented exception when there is no clean substitute.
- The first implementation should stay evidence-backed and reproducible rather than AI-generated.

## Seed Rule Catalog

The repo now includes a human-readable starter catalog in `.tech-radar/patterns.yaml` for the deterministic first implementation. It is not yet wired into runtime behavior, but it documents the intended shape of the rule system and gives the project a concrete vocabulary for patterns, anti-patterns, and alternatives.

## Architecture

```
cmd/radar/main.go         # CLI entrypoint
internal/
  scanner/                # Project scanning
  codebase/               # Pattern and architecture intelligence
  radar/                  # Radar data management
pkg/types/                # Shared types
web/                      # Frontend (Svelte)
```

## Visualization

The radar is rendered as an interactive SVG with:
- Hover tooltips with entry details
- Click to expand history
- Filter by quadrant/ring
- Search functionality

The codebase and project intelligence views add:
- Pattern badges for healthy architecture shapes
- Warning cards for anti-patterns
- Evidence snippets and source references
- Suggested alternatives or next governance actions

## Integration with py-git-chronicles

Cross-reference with git history to see when technologies were introduced:

```bash
radar timeline --tech "React"
# Shows commits that introduced React and related changes
```

## Development

```bash
# Build CLI
make build

# Run dev server
make dev

# Build frontend
cd web && pnpm build
```

## License

MIT
