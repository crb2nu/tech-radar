// Codebase visualizer types and utilities.

export interface CodebaseAnalysis {
  nodes: TechNode[];
  edges: DependencyEdge[];
  stacks: ProjectStack[];
  patterns: PatternTag[];
  findings?: ProjectFinding[];
  file_tree: ProjectTree[];
  summary: CodebaseSummary;
}

export interface TechNode {
  name: string;
  quadrant: string;
  ring: string;
  usage: number;
  is_tracked: boolean;
}

export interface DependencyEdge {
  source: string;
  target: string;
  weight: number;
  projects: string[];
}

export interface ProjectStack {
  project_id: string;
  project_name: string;
  last_scan: string;
  categories: Record<string, string[]>;
  patterns: string[];
  tech_count: number;
}

export interface PatternTag {
  pattern: string;
  projects: string[];
  signal: string;
  icon: string;
}

export type FindingKind = 'pattern' | 'anti_pattern';
export type FindingSeverity = 'low' | 'medium' | 'high';

export interface ProjectIdentity {
  id: string;
  name: string;
}

export interface FindingEvidence {
  technology: string;
  source: string;
  version?: string;
  evidence_path?: string;
  evidence_title?: string;
}

export interface ProjectFinding {
  id: string;
  kind: FindingKind;
  title: string;
  summary: string;
  severity: FindingSeverity;
  confidence: number;
  project: ProjectIdentity;
  evidence: FindingEvidence[];
  alternatives?: string[];
}

export interface ProjectTree {
  project_id: string;
  project_name: string;
  files: FileNode[];
}

export interface FileNode {
  path: string;
  technologies: string[];
}

export interface CodebaseSummary {
  total_projects: number;
  total_technologies: number;
  total_edges: number;
  total_patterns: number;
  tracked_percent: number;
}

/* ---------- fetch ---------- */

export async function fetchCodebaseAnalysis(): Promise<CodebaseAnalysis | null> {
  try {
    const res = await fetch('/api/v1/codebase');
    if (!res.ok) return null;
    return (await res.json()) as CodebaseAnalysis;
  } catch {
    return null;
  }
}

/* ---------- force-directed graph (simple velocity-Verlet) ---------- */

export interface GraphNode extends TechNode {
  x: number;
  y: number;
  vx: number;
  vy: number;
  fx?: number | null;
  fy?: number | null;
  radius: number;
}

export type GraphDensityMode = 'focused' | 'full';

const QUADRANT_COLORS: Record<string, string> = {
  languages: '#6366f1',
  frameworks: '#0ea5e9',
  tools: '#f59e0b',
  platforms: '#10b981',
  techniques: '#ec4899',
};

const RING_OPACITY: Record<string, number> = {
  adopt: 1.0,
  trial: 0.85,
  assess: 0.65,
  hold: 0.4,
};

export function nodeColor(node: GraphNode): string {
  return QUADRANT_COLORS[node.quadrant] ?? '#64748b';
}

export function nodeOpacity(node: GraphNode): number {
  return RING_OPACITY[node.ring] ?? 0.6;
}

export function nodeRadius(node: GraphNode): number {
  return Math.max(8, Math.min(28, 6 + node.usage * 4));
}

export function initGraph(
  analysis: CodebaseAnalysis,
  width: number,
  height: number
): { nodes: GraphNode[]; edges: DependencyEdge[] } {
  const cx = width / 2;
  const cy = height / 2;

  // Only include nodes that appear in at least one edge for graph clarity
  const edgeNames = new Set<string>();
  for (const e of analysis.edges) {
    edgeNames.add(e.source);
    edgeNames.add(e.target);
  }

  const nodes: GraphNode[] = analysis.nodes
    .filter((n) => edgeNames.has(n.name) || n.usage >= 2)
    .map((n, i) => {
      const angle = (i / Math.max(1, analysis.nodes.length)) * Math.PI * 2;
      const spread = Math.min(width, height) * 0.35;
      return {
        ...n,
        x: cx + Math.cos(angle) * spread * (0.5 + Math.random() * 0.5),
        y: cy + Math.sin(angle) * spread * (0.5 + Math.random() * 0.5),
        vx: 0,
        vy: 0,
        radius: nodeRadius({ ...n, x: 0, y: 0, vx: 0, vy: 0, radius: 0 }),
      };
    });

  // Only keep edges where both nodes exist in filtered set
  const nodeNames = new Set(nodes.map((n) => n.name));
  const edges = analysis.edges.filter((e) => nodeNames.has(e.source) && nodeNames.has(e.target));

  return { nodes, edges };
}

export function tickSimulation(
  nodes: GraphNode[],
  edges: DependencyEdge[],
  width: number,
  height: number,
  alpha = 0.3
): void {
  const nodeMap = new Map<string, GraphNode>();
  for (const n of nodes) nodeMap.set(n.name, n);

  // Repulsion (charge)
  for (let i = 0; i < nodes.length; i++) {
    for (let j = i + 1; j < nodes.length; j++) {
      const a = nodes[i];
      const b = nodes[j];
      let dx = b.x - a.x;
      let dy = b.y - a.y;
      let dist = Math.sqrt(dx * dx + dy * dy) || 1;
      const force = (-800 * alpha) / (dist * dist);
      const fx = (dx / dist) * force;
      const fy = (dy / dist) * force;
      a.vx -= fx;
      a.vy -= fy;
      b.vx += fx;
      b.vy += fy;
    }
  }

  // Attraction (links)
  for (const edge of edges) {
    const a = nodeMap.get(edge.source);
    const b = nodeMap.get(edge.target);
    if (!a || !b) continue;
    let dx = b.x - a.x;
    let dy = b.y - a.y;
    let dist = Math.sqrt(dx * dx + dy * dy) || 1;
    const idealDist = 120;
    const force = ((dist - idealDist) * 0.05 * alpha * Math.min(edge.weight, 4));
    const fx = (dx / dist) * force;
    const fy = (dy / dist) * force;
    a.vx += fx;
    a.vy += fy;
    b.vx -= fx;
    b.vy -= fy;
  }

  // Center gravity
  const cx = width / 2;
  const cy = height / 2;
  for (const n of nodes) {
    n.vx += (cx - n.x) * 0.01 * alpha;
    n.vy += (cy - n.y) * 0.01 * alpha;
  }

  // Apply velocities
  for (const n of nodes) {
    if (n.fx != null) {
      n.x = n.fx;
      n.vx = 0;
    } else {
      n.vx *= 0.6;
      n.x += n.vx;
    }
    if (n.fy != null) {
      n.y = n.fy;
      n.vy = 0;
    } else {
      n.vy *= 0.6;
      n.y += n.vy;
    }

    // Bound to viewport
    const pad = n.radius + 4;
    n.x = Math.max(pad, Math.min(width - pad, n.x));
    n.y = Math.max(pad, Math.min(height - pad, n.y));
  }
}

function edgeStrengthByNode(edges: DependencyEdge[]): Map<string, number> {
  const strength = new Map<string, number>();
  for (const edge of edges) {
    const weight = Math.min(edge.weight, 4);
    strength.set(edge.source, (strength.get(edge.source) ?? 0) + weight);
    strength.set(edge.target, (strength.get(edge.target) ?? 0) + weight);
  }
  return strength;
}

export function getGraphView(
  nodes: GraphNode[],
  edges: DependencyEdge[],
  mode: GraphDensityMode,
  focusedNames: string[] = []
): { nodes: GraphNode[]; edges: DependencyEdge[] } {
  if (mode === 'full') {
    return { nodes, edges };
  }

  const sticky = new Set(focusedNames.filter(Boolean));
  const strength = edgeStrengthByNode(edges);
  const scored = [...nodes]
    .map((node) => ({
      node,
      score:
        node.usage * 2 +
        (node.is_tracked ? 5 : 0) +
        (strength.get(node.name) ?? 0) +
        (sticky.has(node.name) ? 100 : 0),
    }))
    .sort((a, b) => b.score - a.score);

  const keep = new Set(scored.slice(0, 42).map(({ node }) => node.name));
  for (const name of sticky) {
    keep.add(name);
    for (const edge of edges) {
      if (edge.source === name) keep.add(edge.target);
      if (edge.target === name) keep.add(edge.source);
    }
  }

  const nextEdges = edges.filter((edge) => {
    if (!keep.has(edge.source) || !keep.has(edge.target)) return false;
    return edge.weight >= 2 || sticky.has(edge.source) || sticky.has(edge.target);
  });

  const connected = new Set<string>();
  for (const edge of nextEdges) {
    connected.add(edge.source);
    connected.add(edge.target);
  }
  for (const name of sticky) connected.add(name);

  const nextNodes = nodes.filter((node) => connected.has(node.name));
  return { nodes: nextNodes, edges: nextEdges };
}

export function shouldShowGraphLabel(
  node: GraphNode,
  edges: DependencyEdge[],
  hoveredName: string | null,
  selectedName: string | null,
  mode: GraphDensityMode
): boolean {
  if (node.name === hoveredName || node.name === selectedName) return true;

  const strength = edgeStrengthByNode(edges).get(node.name) ?? 0;
  if (mode === 'full') {
    return node.is_tracked && (node.usage >= 7 || strength >= 8);
  }

  return node.is_tracked && (node.usage >= 5 || strength >= 6);
}

/* ---------- Stack explorer helpers ---------- */

export const CATEGORY_ORDER = ['languages', 'frameworks', 'tools', 'platforms', 'techniques'];
export const CATEGORY_LABELS: Record<string, string> = {
  languages: 'Languages',
  frameworks: 'Frameworks',
  tools: 'Tools',
  platforms: 'Platforms',
  techniques: 'Techniques',
};

export function categoryColor(cat: string): string {
  return QUADRANT_COLORS[cat] ?? '#64748b';
}

export function allCategories(stacks: ProjectStack[]): string[] {
  const cats = new Set<string>();
  for (const s of stacks) {
    for (const k of Object.keys(s.categories)) {
      cats.add(k);
    }
  }
  return CATEGORY_ORDER.filter((c) => cats.has(c));
}

export function patternIcon(pattern: string): string {
  const icons: Record<string, string> = {
    'GitOps': '🔄',
    'MCP-enabled': '🤖',
    'AI/ML pipeline': '🧠',
    'Containerized': '📦',
    'Monorepo': '🏗️',
    'Full-stack': '🌐',
  };
  return icons[pattern] ?? '🏷️';
}

export function findingSeverityTone(severity: FindingSeverity): 'good' | 'watch' | 'danger' {
  switch (severity) {
    case 'high':
      return 'danger';
    case 'medium':
      return 'watch';
    default:
      return 'good';
  }
}
