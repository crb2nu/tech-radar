Based on the research, here are 3 high-impact feature expansions for our Tech Radar:

1. **AI/LLMOps Quadrant & MCP Discovery**
   *   **Concept**: Add specific tracking for GenAI tools, LLMs, and MCP (Model Context Protocol) servers.
   *   **Implementation**: Update the CLI scanner to detect `mcp-go` and python MCP SDKs. Add a new conceptual quadrant or tag filter for "Agentic AI".

2. **Architecture Decision Records (ADR) Integration**
   *   **Concept**: Link radar moves directly to markdown ADR files in a repository.
   *   **Implementation**: When adding/moving a tech, instead of just a text "reason", allow linking to a specific GitOps/ADR URL. The scanner could even parse `docs/adrs/` and auto-suggest updates to the radar.

3. **Dynamic Lifecycle "Motion" Vectors**
   *   **Concept**: Visually show the momentum of a technology on the radar (e.g., accelerating towards Adopt, or slowly drifting to Hold).
   *   **Implementation**: We have `moved: 1 | -1` in the backend. We can enhance the UI to show velocity based on the `entry_history` table (e.g., if it moved from Assess -> Trial -> Adopt in 3 months, it gets a "hot" vector icon).
