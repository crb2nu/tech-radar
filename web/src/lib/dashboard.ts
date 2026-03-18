export interface HistoryEntry {
  date: string;
  ring: string;
  reason: string;
  author?: string;
}

export interface Entry {
  id: string;
  name: string;
  quadrant: string;
  ring: string;
  description: string;
  moved: number;
  link?: string;
  tags?: string[];
  history?: HistoryEntry[];
  project_ids?: string[];
}

export interface Radar {
  id: string;
  meta: {
    name: string;
    updated?: string;
    version?: string;
  };
  entries: Entry[];
}

export interface ProjectTechnology {
  name: string;
  source: string;
  version?: string;
  quadrant?: string;
}

export interface Project {
  id: string;
  name: string;
  url?: string;
  last_scan: string;
  technologies?: ProjectTechnology[];
}

export interface UsageStat {
  name: string;
  count: number;
}

export interface LabelStat {
  label: string;
  count: number;
}

export interface RecentMove {
  name: string;
  from_ring?: string;
  to_ring: string;
  date: string;
  reason?: string;
  author?: string;
  project_count: number;
}

export interface ProjectHealth {
  id: string;
  name: string;
  last_scan: string;
  total_techs: number;
  tracked_techs: number;
  untracked_techs: number;
  adopt_techs: number;
  trial_techs: number;
  assess_techs: number;
  hold_techs: number;
  coverage_percent: number;
  radar_alignment_score: number;
  risk_level: 'healthy' | 'watch' | 'critical' | string;
  top_techs: string[];
}

export interface InsightSummary {
  radar_coverage_percent: number;
  tracked_usage_count: number;
  uncharted_usage_count: number;
  active_drift_count: number;
  recent_decision_count: number;
  ai_mcp_project_count: number;
  ai_mcp_uncharted_count: number;
}

export interface Insights {
  top_uncharted?: UsageStat[];
  top_adopted?: UsageStat[];
  top_drift?: UsageStat[];
  top_ai_mcp_uncharted?: UsageStat[];
  top_ai_mcp_tracked?: UsageStat[];
  top_tags?: UsageStat[];
  quadrant_breakdown?: LabelStat[];
  ring_breakdown?: LabelStat[];
  source_breakdown?: LabelStat[];
  recent_moves?: RecentMove[];
  project_health?: ProjectHealth[];
  summary?: InsightSummary;
  projects_count?: number;
  tech_count?: number;
}

export interface ProjectIntelligence extends Project {
  totalTechs: number;
  trackedTechs: number;
  untrackedTechs: number;
  holdTechs: number;
  coveragePercent: number;
  radarAlignmentScore: number;
  riskLevel: 'healthy' | 'watch' | 'critical';
  topTechs: string[];
}

export interface SummaryMetric {
  label: string;
  value: string;
  caption: string;
  tone: 'good' | 'watch' | 'danger' | 'neutral';
}

export type MotionStatus = 'accelerating' | 'heating' | 'cooling' | 'new' | 'steady';

export interface MotionVector {
  entryId: string;
  name: string;
  status: MotionStatus;
  direction: 'up' | 'down' | 'flat';
  score: number;
  moveCount: number;
  recentMoveCount: number;
  netRingDelta: number;
  streak: number;
  latestDate?: string;
  latestRing: string;
  previousRing?: string;
  projectCount: number;
  summary: string;
}

export interface RadarSection {
  ring: string;
  label: string;
  description: string;
  entries: Entry[];
}

export interface DashboardModel {
  renderableEntries: Entry[];
  offCanvasEntries: Entry[];
  filteredEntries: Entry[];
  filteredProjects: ProjectIntelligence[];
  summaryMetrics: SummaryMetric[];
  coveragePercent: number;
  projectCount: number;
  techCount: number;
  driftEntries: Entry[];
  opportunityQueue: UsageStat[];
  adoptionLeaders: UsageStat[];
  driftLeaders: UsageStat[];
  aiMcpOpportunityQueue: UsageStat[];
  aiMcpAdoptionLeaders: UsageStat[];
  aiMcpProjectCount: number;
  adrEntries: Entry[];
  motionEntries: MotionVector[];
  heatingUpEntries: MotionVector[];
  coolingEntries: MotionVector[];
  recentMoves: RecentMove[];
  quadrantBreakdown: LabelStat[];
  ringBreakdown: LabelStat[];
  sourceBreakdown: LabelStat[];
  topTags: UsageStat[];
  projectSpotlight: ProjectIntelligence[];
  radarSections: RadarSection[];
  recentDecisionText: string;
}

export const canvasQuadrants = ['languages', 'frameworks', 'tools', 'platforms'] as const;
export const ringOrder = ['adopt', 'trial', 'assess', 'hold'] as const;

export const ringMeta: Record<string, { label: string; description: string; accent: string }> = {
  adopt: {
    label: 'Adopt',
    description: 'Preferred defaults with strong internal confidence.',
    accent: '#0b8f5d'
  },
  trial: {
    label: 'Trial',
    description: 'Active bets that need evidence and adoption signal.',
    accent: '#1d4ed8'
  },
  assess: {
    label: 'Assess',
    description: 'Interesting candidates worth structured evaluation.',
    accent: '#c97a10'
  },
  hold: {
    label: 'Hold',
    description: 'Known risks or migration targets that need attention.',
    accent: '#b42318'
  }
};

export const quadrantMeta: Record<string, { label: string; summary: string }> = {
  languages: {
    label: 'Languages',
    summary: 'Execution substrate and platform fluency.'
  },
  frameworks: {
    label: 'Frameworks',
    summary: 'Product delivery stack and primary developer surface.'
  },
  tools: {
    label: 'Tools',
    summary: 'Build, automation, and operational productivity.'
  },
  platforms: {
    label: 'Platforms',
    summary: 'Infrastructure, cloud, and runtime estate.'
  },
  techniques: {
    label: 'Techniques',
    summary: 'Ways of working that deserve explicit governance.'
  }
};

export function computeDashboardModel(
  radar: Radar | null,
  projects: Project[],
  insights: Insights | null,
  filters: {
    searchQuery: string;
    selectedQuadrant: string;
    selectedRing: string;
    selectedTag: string;
  }
): DashboardModel {
  const entries = radar?.entries ?? [];
  const normalizedQuery = filters.searchQuery.trim().toLowerCase();

  const filteredEntries = entries.filter((entry) => {
    const haystack = [entry.name, entry.description, ...(entry.tags ?? [])].join(' ').toLowerCase();
    const matchesSearch = !normalizedQuery || haystack.includes(normalizedQuery);
    const matchesQuadrant = filters.selectedQuadrant === 'all' || entry.quadrant === filters.selectedQuadrant;
    const matchesRing = filters.selectedRing === 'all' || entry.ring === filters.selectedRing;
    const matchesTag = filters.selectedTag === 'all' || (entry.tags ?? []).includes(filters.selectedTag);
    return matchesSearch && matchesQuadrant && matchesRing && matchesTag;
  });

  const entryByName = new Map(entries.map((entry) => [entry.name.toLowerCase(), entry]));
  const computedProjectHealth = computeProjectHealth(projects, entryByName);
  const allProjectsIntelligence = projects.map((project) =>
    mergeProjectHealth(project, computedProjectHealth.get(project.name.toLowerCase()), undefined)
  );

  const projectHealthByName = new Map(
    (insights?.project_health ?? []).map((project) => [project.name.toLowerCase(), project])
  );

  const mergedProjects = projects.map((project) =>
    mergeProjectHealth(project, computedProjectHealth.get(project.name.toLowerCase()), projectHealthByName.get(project.name.toLowerCase()))
  );

  const filteredProjects = mergedProjects
    .filter((project) => {
      if (!normalizedQuery) return true;
      const techMatch = (project.technologies ?? []).some((tech) => tech.name.toLowerCase().includes(normalizedQuery));
      return project.name.toLowerCase().includes(normalizedQuery) || techMatch;
    })
    .sort((a, b) => {
      if (a.riskLevel === b.riskLevel) {
        return a.radarAlignmentScore - b.radarAlignmentScore;
      }
      return riskPriority(a.riskLevel) - riskPriority(b.riskLevel);
    });

  const renderableEntries = filteredEntries.filter((entry) => canvasQuadrants.includes(entry.quadrant as (typeof canvasQuadrants)[number]));
  const offCanvasEntries = filteredEntries.filter((entry) => !canvasQuadrants.includes(entry.quadrant as (typeof canvasQuadrants)[number]));
  const driftEntries = entries
    .filter((entry) => entry.ring === 'hold' && (entry.project_ids?.length ?? 0) > 0)
    .sort((a, b) => (b.project_ids?.length ?? 0) - (a.project_ids?.length ?? 0));

  const fallbackInsights = buildFallbackInsights(entries, allProjectsIntelligence);
  const summary = insights?.summary ?? fallbackInsights.summary;
  const coveragePercent = summary.radar_coverage_percent ?? fallbackInsights.summary.radar_coverage_percent;
  const opportunityQueue = insights?.top_uncharted?.length ? insights.top_uncharted : fallbackInsights.topUncharted;
  const adoptionLeaders = insights?.top_adopted?.length ? insights.top_adopted : fallbackInsights.topAdopted;
  const driftLeaders = insights?.top_drift?.length ? insights.top_drift : fallbackInsights.topDrift;
  const aiMcpOpportunityQueue = insights?.top_ai_mcp_uncharted?.length
    ? insights.top_ai_mcp_uncharted
    : fallbackInsights.topAIMCPUncharted;
  const aiMcpAdoptionLeaders = insights?.top_ai_mcp_tracked?.length
    ? insights.top_ai_mcp_tracked
    : fallbackInsights.topAIMCPTracked;
  const adrEntries = filteredEntries
    .filter((entry) => isADRReference(entry.link))
    .sort((a, b) => (b.history?.length ?? 0) - (a.history?.length ?? 0) || a.name.localeCompare(b.name));
  const motionEntries = filteredEntries
    .map((entry) => getEntryMotion(entry))
    .filter((motion) => motion.moveCount > 0)
    .sort(compareMotionVectors);
  const activeMotionEntries = motionEntries.filter((motion) => motion.status !== 'steady');
  const heatingUpEntries = activeMotionEntries.filter((motion) =>
    motion.status === 'accelerating' || motion.status === 'heating' || motion.status === 'new'
  );
  const coolingEntries = activeMotionEntries.filter((motion) => motion.status === 'cooling');
  const recentMoves = insights?.recent_moves?.length ? insights.recent_moves : fallbackInsights.recentMoves;
  const quadrantBreakdown = insights?.quadrant_breakdown?.length ? insights.quadrant_breakdown : fallbackInsights.quadrantBreakdown;
  const ringBreakdown = insights?.ring_breakdown?.length ? insights.ring_breakdown : fallbackInsights.ringBreakdown;
  const sourceBreakdown = insights?.source_breakdown?.length ? insights.source_breakdown : fallbackInsights.sourceBreakdown;
  const topTags = insights?.top_tags?.length ? insights.top_tags : fallbackInsights.topTags;
  const aiMcpProjectCount = insights?.summary?.ai_mcp_project_count ?? fallbackInsights.summary.ai_mcp_project_count;

  const projectSpotlight = filteredProjects.slice(0, 6);
  const radarSections = ringOrder.map((ring) => ({
    ring,
    label: ringMeta[ring].label,
    description: ringMeta[ring].description,
    entries: filteredEntries
      .filter((entry) => entry.ring === ring)
      .sort((a, b) => {
        const projectsDiff = (b.project_ids?.length ?? 0) - (a.project_ids?.length ?? 0);
        if (projectsDiff !== 0) return projectsDiff;
        return a.name.localeCompare(b.name);
      })
  }));

  const summaryMetrics: SummaryMetric[] = [
    {
      label: 'Radar coverage',
      value: `${coveragePercent.toFixed(1)}%`,
      caption: `${summary.tracked_usage_count} reviewed signals across the estate`,
      tone: coveragePercent >= 80 ? 'good' : coveragePercent >= 60 ? 'watch' : 'danger'
    },
    {
      label: 'Uncharted demand',
      value: `${summary.uncharted_usage_count}`,
      caption: 'Technologies showing up in projects without an explicit decision',
      tone: summary.uncharted_usage_count === 0 ? 'good' : summary.uncharted_usage_count < 4 ? 'watch' : 'danger'
    },
    {
      label: 'Active drift',
      value: `${summary.active_drift_count}`,
      caption: 'Hold items still present in live project scans',
      tone: summary.active_drift_count === 0 ? 'good' : 'danger'
    },
    {
      label: 'Decision momentum',
      value: `${summary.recent_decision_count}`,
      caption: 'Recent governance moves captured in history',
      tone: summary.recent_decision_count > 0 ? 'neutral' : 'watch'
    },
    {
      label: 'Motion vectors',
      value: `${activeMotionEntries.length}`,
      caption: activeMotionEntries[0]
        ? `${activeMotionEntries[0].name} is ${motionLabel(activeMotionEntries[0]).toLowerCase()}`
        : 'No active lifecycle shifts are standing out yet',
      tone: activeMotionEntries.length > 0 ? motionTone(activeMotionEntries[0]) : 'watch'
    },
    {
      label: 'AI / MCP frontier',
      value: `${summary.ai_mcp_uncharted_count}`,
      caption: `${aiMcpProjectCount} projects already show agentic tooling signals`,
      tone: summary.ai_mcp_uncharted_count === 0 ? 'good' : summary.ai_mcp_uncharted_count < 4 ? 'watch' : 'danger'
    },
    {
      label: 'ADR evidence',
      value: `${adrEntries.length}`,
      caption: 'Radar decisions that already point to ADR or markdown evidence',
      tone: adrEntries.length > 0 ? 'good' : 'watch'
    }
  ];

  return {
    renderableEntries,
    offCanvasEntries,
    filteredEntries,
    filteredProjects,
    summaryMetrics,
    coveragePercent,
    projectCount: projects.length,
    techCount: entries.length,
    driftEntries,
    opportunityQueue,
    adoptionLeaders,
    driftLeaders,
    aiMcpOpportunityQueue,
    aiMcpAdoptionLeaders,
    aiMcpProjectCount,
    adrEntries,
    motionEntries,
    heatingUpEntries,
    coolingEntries,
    recentMoves,
    quadrantBreakdown,
    ringBreakdown,
    sourceBreakdown,
    topTags,
    projectSpotlight,
    radarSections,
    recentDecisionText:
      recentMoves[0] ? `${recentMoves[0].name} most recently moved to ${recentMoves[0].to_ring.toUpperCase()}.` : 'No recent governance movement has been recorded yet.'
  };
}

export function uniqueTags(entries: Entry[]): string[] {
  return Array.from(new Set(entries.flatMap((entry) => entry.tags ?? []))).sort((a, b) => a.localeCompare(b));
}

export function getBlipPosition(entry: Entry, index: number): { x: number; y: number } {
  const quadIndex = canvasQuadrants.indexOf(entry.quadrant as (typeof canvasQuadrants)[number]);
  const ringIndex = ringOrder.indexOf(entry.ring as (typeof ringOrder)[number]);
  const ringRadius = [80, 156, 232, 300];

  if (quadIndex === -1 || ringIndex === -1) {
    return { x: 340, y: 340 };
  }

  const baseAngle = quadIndex * (Math.PI / 2);
  const inner = ringIndex === 0 ? 34 : ringRadius[ringIndex - 1];
  const outer = ringRadius[ringIndex];
  const seed = `${entry.name}-${index}`.split('').reduce((total, char) => total + char.charCodeAt(0), 0);
  const radial = inner + (Math.abs(Math.sin(seed)) * (outer - inner - 28)) + 14;
  const angle = baseAngle + (Math.abs(Math.cos(seed * 1.7)) * (Math.PI / 2 - 0.42)) + 0.21;

  return {
    x: 340 + radial * Math.cos(angle),
    y: 340 + radial * Math.sin(angle)
  };
}

export function formatDate(value?: string): string {
  if (!value) return 'No date';
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) return 'No date';
  return parsed.toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' });
}

export function formatRelativeDate(value?: string): string {
  if (!value) return 'No scan recorded';
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) return 'No scan recorded';
  const diffDays = Math.round((Date.now() - parsed.getTime()) / (1000 * 60 * 60 * 24));
  if (diffDays <= 0) return 'today';
  if (diffDays === 1) return '1 day ago';
  if (diffDays < 30) return `${diffDays} days ago`;
  const diffMonths = Math.round(diffDays / 30);
  if (diffMonths === 1) return '1 month ago';
  return `${diffMonths} months ago`;
}

export function isADRReference(link?: string): boolean {
  if (!link) return false;
  const normalized = link.toLowerCase();
  return normalized.endsWith('.md') || normalized.includes('/adr') || normalized.includes('/adrs');
}

export function getEntryMotion(entry: Entry, now = new Date()): MotionVector {
  const history = (entry.history ?? [])
    .filter((item) => !Number.isNaN(new Date(item.date).getTime()))
    .slice()
    .sort((a, b) => +new Date(b.date) - +new Date(a.date));

  const latestRing = history[0]?.ring ?? entry.ring;
  if (history.length === 0) {
    return {
      entryId: entry.id,
      name: entry.name,
      status: 'steady',
      direction: 'flat',
      score: 0,
      moveCount: 0,
      recentMoveCount: 0,
      netRingDelta: 0,
      streak: 0,
      latestRing,
      projectCount: entry.project_ids?.length ?? 0,
      summary: 'No governance history has been recorded yet.'
    };
  }

  const latest = history[0];
  const previous = history[1];
  const latestDate = new Date(latest.date);
  const daysSinceLatest = differenceInDays(now, latestDate);
  const recentMoveCount = history.filter((item) => differenceInDays(now, new Date(item.date)) <= 180).length;
  const latestRank = ringWeight(latest.ring);
  const previousRank = previous ? ringWeight(previous.ring) : latestRank;
  const oldestRank = ringWeight(history[history.length - 1]?.ring ?? latest.ring);
  const latestDelta = latestRank - previousRank;
  const netRingDelta = latestRank - oldestRank;
  const streak = previous ? movementStreak(history) : 1;

  let status: MotionStatus = 'steady';
  let direction: MotionVector['direction'] = 'flat';

  if (history.length === 1) {
    status = daysSinceLatest <= 120 ? 'new' : 'steady';
  } else if (latestDelta > 0) {
    direction = 'up';
    status = recentMoveCount >= 3 || streak >= 2 ? 'accelerating' : 'heating';
  } else if (latestDelta < 0) {
    direction = 'down';
    status = 'cooling';
  } else if (netRingDelta > 0) {
    direction = 'up';
    status = 'heating';
  } else if (netRingDelta < 0) {
    direction = 'down';
    status = 'cooling';
  }

  const recencyBoost = Math.max(0, 90 - daysSinceLatest);
  const score = Math.max(
    0,
    recentMoveCount * 14 +
      Math.abs(latestDelta) * 24 +
      Math.abs(netRingDelta) * 10 +
      streak * 8 +
      Math.round(recencyBoost / 3) +
      (entry.project_ids?.length ?? 0) * 2
  );

  return {
    entryId: entry.id,
    name: entry.name,
    status,
    direction,
    score,
    moveCount: history.length,
    recentMoveCount,
    netRingDelta,
    streak,
    latestDate: latest.date,
    latestRing,
    previousRing: previous?.ring,
    projectCount: entry.project_ids?.length ?? 0,
    summary: describeMotion(status, {
      latestRing,
      previousRing: previous?.ring,
      recentMoveCount,
      streak,
      daysSinceLatest
    })
  };
}

export function motionLabel(motion: MotionVector): string {
  switch (motion.status) {
    case 'accelerating':
      return 'Accelerating';
    case 'heating':
      return 'Heating up';
    case 'cooling':
      return 'Cooling down';
    case 'new':
      return 'Newly added';
    default:
      return 'Steady';
  }
}

export function motionTone(motion: MotionVector): SummaryMetric['tone'] {
  switch (motion.status) {
    case 'accelerating':
    case 'heating':
      return 'good';
    case 'cooling':
      return 'danger';
    case 'new':
      return 'watch';
    default:
      return 'neutral';
  }
}

function buildFallbackInsights(entries: Entry[], projects: ProjectIntelligence[]) {
  const trackedNames = new Set(entries.map((entry) => entry.name.toLowerCase()));
  const topUncharted = rankMap(
    countByName(
      projects.flatMap((project) =>
        (project.technologies ?? [])
          .filter((tech) => !trackedNames.has(tech.name.toLowerCase()))
          .map((tech) => tech.name)
      )
    )
  );

  const topAdopted = rankMap(
    countByName(
      projects.flatMap((project) =>
        (project.technologies ?? [])
          .filter((tech) => entries.some((entry) => entry.name.toLowerCase() === tech.name.toLowerCase()))
          .map((tech) => tech.name)
      )
    )
  );

  const topDrift = rankMap(
    Object.fromEntries(
      entries
        .filter((entry) => entry.ring === 'hold' && (entry.project_ids?.length ?? 0) > 0)
        .map((entry) => [entry.name, entry.project_ids?.length ?? 0])
    )
  );

  const topTags = rankMap(countByName(entries.flatMap((entry) => entry.tags ?? [])));
  const quadrantBreakdown = toLabelStats(countByName(entries.map((entry) => entry.quadrant)));
  const ringBreakdown = toLabelStats(countByName(entries.map((entry) => entry.ring)));
  const sourceBreakdown = toLabelStats(countByName(projects.flatMap((project) => (project.technologies ?? []).map((tech) => tech.source))));
  const topAIMCPUncharted = topUncharted.filter((item) => isAIMCPSignal(item.name));
  const topAIMCPTracked = topAdopted.filter((item) => isAIMCPSignal(item.name));
  const aiMcpProjectCount = projects.filter((project) =>
    (project.technologies ?? []).some((tech) => isAIMCPSignal(tech.name, tech.source))
  ).length;
  const recentMoves = entries
    .flatMap((entry) => {
      const history = (entry.history ?? []).slice().sort((a, b) => +new Date(b.date) - +new Date(a.date));
      if (history.length === 0) return [];
      return [{
        name: entry.name,
        to_ring: history[0].ring,
        from_ring: history[1]?.ring,
        date: history[0].date,
        reason: history[0].reason,
        author: history[0].author,
        project_count: entry.project_ids?.length ?? 0
      }];
    })
    .sort((a, b) => +new Date(b.date) - +new Date(a.date))
    .slice(0, 8);

  const uniqueTracked = new Set(topAdopted.map((item) => item.name)).size;
  const uniqueUncharted = new Set(topUncharted.map((item) => item.name)).size;
  const summary: InsightSummary = {
    radar_coverage_percent: uniqueTracked + uniqueUncharted === 0 ? 0 : Number(((uniqueTracked / (uniqueTracked + uniqueUncharted)) * 100).toFixed(1)),
    tracked_usage_count: uniqueTracked,
    uncharted_usage_count: uniqueUncharted,
    active_drift_count: topDrift.length,
    recent_decision_count: recentMoves.length,
    ai_mcp_project_count: aiMcpProjectCount,
    ai_mcp_uncharted_count: new Set(topAIMCPUncharted.map((item) => item.name.toLowerCase())).size
  };

  return {
    topUncharted,
    topAdopted,
    topDrift,
    topAIMCPUncharted,
    topAIMCPTracked,
    topTags,
    quadrantBreakdown,
    ringBreakdown,
    sourceBreakdown,
    recentMoves,
    summary
  };
}

function isAIMCPSignal(name: string, source = ''): boolean {
  const haystack = `${name} ${source}`.toLowerCase();
  return [
    'mcp',
    'model context protocol',
    'modelcontextprotocol',
    'openai',
    'anthropic',
    'claude',
    'langchain',
    'langgraph',
    'llamaindex',
    'llama-index',
    'ollama',
    'vllm',
    'promptfoo',
    'litellm',
    'autogen',
    'crewai',
    'dspy'
  ].some((keyword) => haystack.includes(keyword));
}

function mergeProjectHealth(project: Project, computed?: ProjectIntelligence, apiHealth?: ProjectHealth): ProjectIntelligence {
  const fallback = computed ?? {
    ...project,
    totalTechs: project.technologies?.length ?? 0,
    trackedTechs: 0,
    untrackedTechs: project.technologies?.length ?? 0,
    holdTechs: 0,
    coveragePercent: 0,
    radarAlignmentScore: 0,
    riskLevel: 'critical' as const,
    topTechs: (project.technologies ?? []).slice(0, 6).map((tech) => tech.name)
  };

  if (!apiHealth) {
    return fallback;
  }

  return {
    ...project,
    totalTechs: apiHealth.total_techs,
    trackedTechs: apiHealth.tracked_techs,
    untrackedTechs: apiHealth.untracked_techs,
    holdTechs: apiHealth.hold_techs,
    coveragePercent: apiHealth.coverage_percent,
    radarAlignmentScore: apiHealth.radar_alignment_score,
    riskLevel: (apiHealth.risk_level as ProjectIntelligence['riskLevel']) ?? fallback.riskLevel,
    topTechs: apiHealth.top_techs?.length ? apiHealth.top_techs : fallback.topTechs
  };
}

function computeProjectHealth(projects: Project[], entryByName: Map<string, Entry>): Map<string, ProjectIntelligence> {
  const output = new Map<string, ProjectIntelligence>();

  for (const project of projects) {
    let trackedTechs = 0;
    let untrackedTechs = 0;
    let holdTechs = 0;
    let adoptBoost = 0;

    for (const tech of project.technologies ?? []) {
      const match = entryByName.get(tech.name.toLowerCase());
      if (!match) {
        untrackedTechs += 1;
        continue;
      }

      trackedTechs += 1;
      if (match.ring === 'hold') holdTechs += 1;
      if (match.ring === 'adopt') adoptBoost += 1;
    }

    const totalTechs = project.technologies?.length ?? 0;
    const coveragePercent = totalTechs === 0 ? 0 : Number(((trackedTechs / totalTechs) * 100).toFixed(1));
    const radarAlignmentScore = Math.max(0, Math.min(100, Math.round(coveragePercent + adoptBoost * 4 - untrackedTechs * 8 - holdTechs * 15)));

    output.set(project.name.toLowerCase(), {
      ...project,
      totalTechs,
      trackedTechs,
      untrackedTechs,
      holdTechs,
      coveragePercent,
      radarAlignmentScore,
      riskLevel: holdTechs > 0 || coveragePercent < 45 ? 'critical' : untrackedTechs > 0 || coveragePercent < 75 ? 'watch' : 'healthy',
      topTechs: (project.technologies ?? []).slice(0, 6).map((tech) => tech.name)
    });
  }

  return output;
}

function riskPriority(level: string): number {
  if (level === 'critical') return 0;
  if (level === 'watch') return 1;
  return 2;
}

function rankMap(values: Record<string, number>): UsageStat[] {
  return Object.entries(values)
    .filter(([, count]) => count > 0)
    .map(([name, count]) => ({ name, count }))
    .sort((a, b) => {
      if (a.count === b.count) return a.name.localeCompare(b.name);
      return b.count - a.count;
    })
    .slice(0, 8);
}

function countByName(values: string[]): Record<string, number> {
  return values.reduce<Record<string, number>>((acc, value) => {
    acc[value] = (acc[value] ?? 0) + 1;
    return acc;
  }, {});
}

function toLabelStats(values: Record<string, number>): LabelStat[] {
  return Object.entries(values)
    .filter(([label, count]) => !!label && count > 0)
    .map(([label, count]) => ({ label, count }))
    .sort((a, b) => b.count - a.count || a.label.localeCompare(b.label));
}

function compareMotionVectors(a: MotionVector, b: MotionVector): number {
  if (a.score === b.score) {
    return a.name.localeCompare(b.name);
  }
  return b.score - a.score;
}

function ringWeight(ring?: string): number {
  switch (ring) {
    case 'adopt':
      return 4;
    case 'trial':
      return 3;
    case 'assess':
      return 2;
    case 'hold':
      return 1;
    default:
      return 0;
  }
}

function movementStreak(history: HistoryEntry[]): number {
  let streak = 0;
  let direction = 0;

  for (let index = 0; index < history.length - 1; index += 1) {
    const delta = ringWeight(history[index].ring) - ringWeight(history[index + 1].ring);
    if (delta === 0) {
      break;
    }

    const sign = Math.sign(delta);
    if (direction === 0) {
      direction = sign;
      streak = 1;
      continue;
    }
    if (sign !== direction) {
      break;
    }
    streak += 1;
  }

  return streak;
}

function differenceInDays(left: Date, right: Date): number {
  return Math.max(0, Math.round((left.getTime() - right.getTime()) / (1000 * 60 * 60 * 24)));
}

function describeMotion(
  status: MotionStatus,
  detail: {
    latestRing: string;
    previousRing?: string;
    recentMoveCount: number;
    streak: number;
    daysSinceLatest: number;
  }
): string {
  const latest = detail.latestRing.toUpperCase();
  const previous = detail.previousRing?.toUpperCase();

  switch (status) {
    case 'accelerating':
      return previous
        ? `Accelerating from ${previous} to ${latest} with a ${detail.streak}-step upward streak and ${detail.recentMoveCount} checkpoints in 180 days.`
        : `Accelerating into ${latest} with repeated governance movement in the last 180 days.`;
    case 'heating':
      return previous
        ? `Heating up from ${previous} to ${latest} with fresh supporting evidence in the last 180 days.`
        : `Heating up in ${latest} with new governance activity.`;
    case 'cooling':
      return previous
        ? `Cooling down from ${previous} to ${latest} after a recent downgrade signal.`
        : `Cooling down in ${latest} with a recent negative movement signal.`;
    case 'new':
      return `Newly added to ${latest} ${relativeDaysLabel(detail.daysSinceLatest)}.`;
    default:
      return detail.recentMoveCount > 1
        ? `Holding steady in ${latest} after ${detail.recentMoveCount} recent governance checkpoints.`
        : `Holding steady in ${latest}.`;
  }
}

function relativeDaysLabel(days: number): string {
  if (days <= 0) return 'today';
  if (days === 1) return '1 day ago';
  if (days < 30) return `${days} days ago`;
  const months = Math.max(1, Math.round(days / 30));
  if (months === 1) return '1 month ago';
  return `${months} months ago`;
}
