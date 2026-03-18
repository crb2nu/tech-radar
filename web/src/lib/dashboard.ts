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
}

export interface Insights {
  top_uncharted?: UsageStat[];
  top_adopted?: UsageStat[];
  top_drift?: UsageStat[];
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
  const recentMoves = insights?.recent_moves?.length ? insights.recent_moves : fallbackInsights.recentMoves;
  const quadrantBreakdown = insights?.quadrant_breakdown?.length ? insights.quadrant_breakdown : fallbackInsights.quadrantBreakdown;
  const ringBreakdown = insights?.ring_breakdown?.length ? insights.ring_breakdown : fallbackInsights.ringBreakdown;
  const sourceBreakdown = insights?.source_breakdown?.length ? insights.source_breakdown : fallbackInsights.sourceBreakdown;
  const topTags = insights?.top_tags?.length ? insights.top_tags : fallbackInsights.topTags;

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
    recent_decision_count: recentMoves.length
  };

  return {
    topUncharted,
    topAdopted,
    topDrift,
    topTags,
    quadrantBreakdown,
    ringBreakdown,
    sourceBreakdown,
    recentMoves,
    summary
  };
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
