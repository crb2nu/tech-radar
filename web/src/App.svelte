<script lang="ts">
  import { onMount } from 'svelte';
  import { fade, fly } from 'svelte/transition';
  import {
    fetchCodebaseAnalysis,
    getGraphView,
    initGraph,
    tickSimulation,
    nodeColor,
    nodeOpacity,
    nodeRadius,
    categoryColor,
    allCategories,
    patternIcon,
    shouldShowGraphLabel,
    CATEGORY_LABELS,
    type CodebaseAnalysis,
    type GraphDensityMode,
    type GraphNode,
    type DependencyEdge as GraphEdge,
  } from './lib/codebase';
  import {
    canvasQuadrants,
    computeDashboardModel,
    formatDate,
    formatRelativeDate,
    getEntryMotion,
    getBlipPosition,
    isADRReference,
    motionLabel,
    motionTone,
    quadrantMeta,
    ringMeta,
    ringOrder,
    uniqueTags,
    type DependencyHotspot,
    type EcosystemStat,
    type Entry,
      type GovernanceRecommendation,
      type Insights,
      type MotionVector,
      type PolyglotProfile,
      type Project,
      type SummaryMetric,
      type Radar,
      type StackSimilarity
    } from './lib/dashboard';

  type Tab = 'overview' | 'radar' | 'projects' | 'insights' | 'drift' | 'codebase';

  type EntryDraft = {
    id?: string;
    name: string;
    quadrant: string;
    ring: string;
    description: string;
    link: string;
    tags: string[];
  };

  type ThemeMode = 'system' | 'light' | 'dark';

  const THEME_STORAGE_KEY = 'tr-theme';

  function canUseDOM() {
    return typeof window !== 'undefined' && typeof document !== 'undefined';
  }

  function readStoredTheme(): ThemeMode {
    if (!canUseDOM()) return 'system';
    const stored = window.localStorage.getItem(THEME_STORAGE_KEY);
    return stored === 'light' || stored === 'dark' || stored === 'system' ? stored : 'system';
  }

  function systemPrefersDark() {
    return canUseDOM() && window.matchMedia('(prefers-color-scheme: dark)').matches;
  }

  const tabs: Array<{ id: Tab; label: string }> = [
    { id: 'overview', label: 'Command deck' },
    { id: 'radar', label: 'Radar' },
    { id: 'codebase', label: 'Codebase' },
    { id: 'projects', label: 'Projects' },
    { id: 'insights', label: 'Intelligence' },
    { id: 'drift', label: 'Drift' }
  ];

  let radar = $state<Radar | null>(null);
  let projects = $state<Project[]>([]);
  let insights = $state<Insights | null>(null);
  let codebaseData = $state<CodebaseAnalysis | null>(null);
  let error = $state<string | null>(null);
  let loading = $state(true);

  // Codebase graph state
  let graphNodes = $state<GraphNode[]>([]);
  let graphEdges = $state<GraphEdge[]>([]);
  let graphAnimFrame = $state<number | null>(null);
  let graphHovered = $state<string | null>(null);
  let graphSelected = $state<string | null>(null);
  let graphDragging = $state<GraphNode | null>(null);
  let cbGraphMode = $state<GraphDensityMode>('focused');
  let cbSubTab = $state<'graph' | 'stacks' | 'files'>('graph');
  let expandedProjects = $state<Set<string>>(new Set());

  let searchQuery = $state('');
  let selectedQuadrant = $state('all');
  let selectedRing = $state('all');
  let selectedTag = $state('all');
  let hoveredEntry = $state<string | null>(null);
  let selectedEntry = $state<Entry | null>(null);
  let activeTab = $state<Tab>('overview');

  let isAdminMode = $state(false);
  let showEntryForm = $state(false);
  let formLoading = $state(false);
  let formError = $state<string | null>(null);
  let formReason = $state('');
  let formTagsInput = $state('');
  let editEntryData = $state<EntryDraft>(emptyEntryDraft());

  let themeMode = $state<ThemeMode>(readStoredTheme());
  let prefersDark = $state(systemPrefersDark());
  const isDark = $derived(themeMode === 'dark' || (themeMode === 'system' && prefersDark));
  const activeThemeLabel = $derived(themeMode === 'system' ? (isDark ? 'System dark' : 'System light') : themeMode === 'dark' ? 'Dark mode' : 'Light mode');
  const themeDescription = $derived(
    themeMode === 'system'
      ? 'Following your device appearance automatically.'
      : themeMode === 'dark'
        ? 'Dark mode stays on until you change it.'
        : 'Light mode stays on until you change it.'
  );

  $effect(() => {
    if (!canUseDOM()) return;
    document.body.classList.toggle('dark', isDark);
    document.body.dataset.theme = isDark ? 'dark' : 'light';
    document.documentElement.style.colorScheme = isDark ? 'dark' : 'light';
    if (themeMode === 'system') {
      window.localStorage.removeItem(THEME_STORAGE_KEY);
    } else {
      window.localStorage.setItem(THEME_STORAGE_KEY, themeMode);
    }
  });

  let searchField = $state<HTMLInputElement | null>(null);

  const allTags = $derived(uniqueTags(radar?.entries ?? []));
  const allQuadrants = $derived(
    Array.from(new Set([...canvasQuadrants, ...((radar?.entries ?? []).map((entry: Entry) => entry.quadrant))])).sort((a, b) => {
      const ai = [...canvasQuadrants, 'techniques'].indexOf(a as (typeof canvasQuadrants)[number]);
      const bi = [...canvasQuadrants, 'techniques'].indexOf(b as (typeof canvasQuadrants)[number]);
      if (ai !== -1 && bi !== -1) return ai - bi;
      if (ai !== -1) return -1;
      if (bi !== -1) return 1;
      return a.localeCompare(b);
    })
  );

  const dashboard = $derived(
    computeDashboardModel(radar, projects, insights, {
      searchQuery,
      selectedQuadrant,
      selectedRing,
      selectedTag
    })
  );
  const motionByEntryId = $derived(new Map(dashboard.motionEntries.map((motion) => [motion.entryId, motion])));
  const recommendationByEntryId = $derived(
    new Map(dashboard.recommendations.filter((recommendation) => recommendation.entryId).map((recommendation) => [recommendation.entryId!, recommendation]))
  );
  const selectedEntryMotion = $derived(selectedEntry ? entryMotion(selectedEntry) : null);
  const selectedEntryRecommendation = $derived(
    selectedEntry ? recommendationByEntryId.get(selectedEntry.id) ?? null : null
  );

  const hasActiveFilters = $derived(
    Boolean(searchQuery.trim()) || selectedQuadrant !== 'all' || selectedRing !== 'all' || selectedTag !== 'all'
  );
  const activeFilterTokens = $derived([
    searchQuery.trim() ? `Search: ${searchQuery.trim()}` : null,
    selectedQuadrant !== 'all' ? `Quadrant: ${quadrantMeta[selectedQuadrant]?.label ?? selectedQuadrant}` : null,
    selectedRing !== 'all' ? `Ring: ${ringMeta[selectedRing]?.label ?? selectedRing}` : null,
    selectedTag !== 'all' ? `Tag: #${selectedTag}` : null
  ].filter(Boolean) as string[]);
  const headlineMetrics = $derived([
    dashboard.summaryMetrics.find((metric) => metric.label === 'Action queue'),
    dashboard.summaryMetrics.find((metric) => metric.label === 'Uncharted demand'),
    dashboard.summaryMetrics.find((metric) => metric.label === 'Active drift'),
    dashboard.summaryMetrics.find((metric) => metric.label === 'Radar coverage')
  ].filter(Boolean) as SummaryMetric[]);
  const activeViewSummary = $derived((() => {
    if (activeTab === 'overview') {
      return {
        label: 'Portfolio pulse',
        title: 'Start with the next governance call',
        description: 'Triage the most urgent interventions, then work down into strategic follow-ups and project outliers.',
        metric: `${dashboard.criticalRecommendations.length} urgent`,
        detail: `${dashboard.strategicRecommendations.length} strategic follow-ups queued`
      };
    }
    if (activeTab === 'radar') {
      return {
        label: 'Visual map',
        title: 'Scan the portfolio before you drill in',
        description: 'Use the radar to spot movement, hover for context, and jump straight into the decision card for any blip.',
        metric: `${dashboard.renderableEntries.length} plotted`,
        detail: `${dashboard.offCanvasEntries.length} off-canvas domains`
      };
    }
    if (activeTab === 'projects') {
      return {
        label: 'Team alignment',
        title: 'Compare project health side by side',
        description: 'Review coverage, untracked technologies, and hold exposure to find teams that need the next governance nudge.',
        metric: `${dashboard.filteredProjects.length} matching projects`,
        detail: `${dashboard.projectSpotlight.length} spotlighted teams`
      };
    }
    if (activeTab === 'insights') {
      return {
        label: 'Signal review',
        title: 'Work the recommendation queue',
        description: 'This lane ranks the technologies and decisions that need evidence, formal calls, or cleanup next.',
        metric: `${dashboard.recommendations.length} live recommendations`,
        detail: `${dashboard.aiMcpProjectCount} projects with AI or MCP traces`
      };
    }
    if (activeTab === 'drift') {
      return {
        label: 'Risk watch',
        title: 'Contain technologies that are drifting out of policy',
        description: 'Focus the teams still shipping hold technologies and close the gap between standards and real usage.',
        metric: `${dashboard.driftEntries.length} drift entries`,
        detail: `${dashboard.driftLeaders.length} hold leaders in flight`
      };
    }
    return {
      label: 'Codebase map',
      title: 'Inspect how the stack fits together',
      description: 'Trace dependency clusters, compare project stacks, and review file-level signals across the scanned workspace.',
      metric: `${codebaseData?.summary.total_technologies ?? 0} technologies`,
      detail: `${codebaseData?.summary.total_projects ?? 0} projects indexed`
    };
  })());

  onMount(() => {
    void loadData();

    const media = window.matchMedia('(prefers-color-scheme: dark)');
    const handleThemeChange = (event: MediaQueryListEvent) => {
      prefersDark = event.matches;
    };
    prefersDark = media.matches;
    media.addEventListener('change', handleThemeChange);

    const handleKeydown = (event: KeyboardEvent) => {
      const isSearchShortcut = (event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k';
      if (isSearchShortcut) {
        event.preventDefault();
        focusSearch();
      }
      if (event.key === 'Escape') {
        if (showEntryForm) showEntryForm = false;
        if (selectedEntry) selectedEntry = null;
      }
    };

    window.addEventListener('keydown', handleKeydown);
    return () => {
      media.removeEventListener('change', handleThemeChange);
      window.removeEventListener('keydown', handleKeydown);
    };
  });

  async function loadData() {
    loading = true;
    error = null;

    try {
      const radarData = await fetchRadar();
      radar = radarData;
    } catch (err) {
      error = err instanceof Error ? err.message : 'Unable to load radar data';
      loading = false;
      return;
    }

    const [projectsResult, insightsResult, codebaseResult] = await Promise.allSettled([
      fetchProjects(),
      fetchInsights(),
      fetchCodebaseAnalysis(),
    ]);

    if (projectsResult.status === 'fulfilled') {
      projects = projectsResult.value;
    } else {
      projects = [];
      console.error(projectsResult.reason);
    }

    if (insightsResult.status === 'fulfilled') {
      insights = insightsResult.value;
    } else {
      insights = null;
      console.error(insightsResult.reason);
    }

    if (codebaseResult.status === 'fulfilled') {
      codebaseData = codebaseResult.value;
    } else {
      codebaseData = null;
    }

    loading = false;
  }

  async function fetchRadar(): Promise<Radar> {
    let response = await fetch('/api/v1/radar');
    if (!response.ok) {
      response = await fetch('/api/radar');
    }
    if (!response.ok) {
      throw new Error('Failed to load radar data');
    }
    return (await response.json()) as Radar;
  }

  async function fetchProjects(): Promise<Project[]> {
    const response = await fetch('/api/v1/projects');
    if (!response.ok) return [];
    return (await response.json()) as Project[];
  }

  async function fetchInsights(): Promise<Insights | null> {
    const response = await fetch('/api/v1/insights');
    if (!response.ok) return null;
    return (await response.json()) as Insights;
  }

  function focusSearch() {
    searchField?.focus();
    searchField?.select();
  }

  function openEntryById(entryId: string) {
    selectedEntry = radar?.entries.find((entry) => entry.id === entryId) ?? null;
  }

  function entryMotion(entry: Entry): MotionVector {
    return motionByEntryId.get(entry.id) ?? getEntryMotion(entry);
  }

  function applyRecommendation(recommendation: GovernanceRecommendation) {
    if (recommendation.entryId) {
      openEntryById(recommendation.entryId);
      activeTab = recommendation.category === 'contain' ? 'drift' : 'insights';
      return;
    }

    if (isAdminMode) {
      openEntryForm(null, recommendation.targetName);
      return;
    }

    searchQuery = recommendation.targetName;
    activeTab = 'insights';
  }

  function clearFilters() {
    searchQuery = '';
    selectedQuadrant = 'all';
    selectedRing = 'all';
    selectedTag = 'all';
  }

  function emptyEntryDraft(name = ''): EntryDraft {
    return {
      name,
      quadrant: 'frameworks',
      ring: 'assess',
      description: '',
      link: '',
      tags: []
    };
  }

  function openEntryForm(existingEntry: Entry | null = null, defaultName = '') {
    if (existingEntry) {
      editEntryData = {
        id: existingEntry.id,
        name: existingEntry.name,
        quadrant: existingEntry.quadrant,
        ring: existingEntry.ring,
        description: existingEntry.description ?? '',
        link: existingEntry.link ?? '',
        tags: existingEntry.tags ?? []
      };
      formTagsInput = (existingEntry.tags ?? []).join(', ');
    } else {
      editEntryData = emptyEntryDraft(defaultName);
      formTagsInput = '';
    }

    formReason = '';
    formError = null;
    showEntryForm = true;
  }

  async function submitEntry() {
    formLoading = true;
    formError = null;

    const nextName = editEntryData.name;
    const payload = {
      ...editEntryData,
      tags: formTagsInput
        .split(',')
        .map((tag) => tag.trim())
        .filter(Boolean),
      description: formReason.trim() || editEntryData.description
    };

    try {
      const response = await fetch('/api/v1/entry', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify(payload)
      });

      if (!response.ok) {
        throw new Error(await response.text());
      }

      await loadData();
      selectedEntry = radar?.entries.find((entry) => entry.name === nextName) ?? null;
      showEntryForm = false;
    } catch (err) {
      formError = err instanceof Error ? err.message : 'Unable to save entry';
    } finally {
      formLoading = false;
    }
  }

  function riskLabel(level: string) {
    if (level === 'critical') return 'Needs attention';
    if (level === 'watch') return 'Watch list';
    return 'Healthy';
  }

  function movementLabel(entry: Entry) {
    if (entry.moved > 0) return 'Promoted';
    if (entry.moved < 0) return 'Downgraded';
    return 'Steady';
  }

  function setThemeMode(nextTheme: ThemeMode) {
    themeMode = nextTheme;
  }

  // --- Codebase graph helpers ---
  function startGraphSimulation() {
    if (!codebaseData) return;
    const w = 800;
    const h = 560;
    const init = initGraph(codebaseData, w, h);
    graphNodes = init.nodes;
    graphEdges = init.edges;
    let ticks = 0;
    const maxTicks = 300;

    function frame() {
      if (ticks >= maxTicks) return;
      const alpha = Math.max(0.01, 0.8 * Math.pow(0.99, ticks));
      tickSimulation(graphNodes, graphEdges, w, h, alpha);
      graphNodes = [...graphNodes]; // trigger reactivity
      ticks++;
      graphAnimFrame = requestAnimationFrame(frame);
    }
    if (graphAnimFrame) cancelAnimationFrame(graphAnimFrame);
    graphAnimFrame = requestAnimationFrame(frame);
  }

  function onGraphNodeMouseDown(node: GraphNode, event: MouseEvent) {
    event.preventDefault();
    graphDragging = node;
    node.fx = node.x;
    node.fy = node.y;
  }

  function onGraphMouseMove(event: MouseEvent) {
    if (!graphDragging) return;
    const canvas = event.currentTarget as HTMLDivElement;
    const rect = canvas.getBoundingClientRect();
    const scaleX = 800 / rect.width;
    const scaleY = 560 / rect.height;
    graphDragging.fx = (event.clientX - rect.left) * scaleX;
    graphDragging.fy = (event.clientY - rect.top) * scaleY;
    graphDragging.x = graphDragging.fx;
    graphDragging.y = graphDragging.fy;
    graphNodes = [...graphNodes];
  }

  function onGraphMouseUp() {
    if (graphDragging) {
      graphDragging.fx = null;
      graphDragging.fy = null;
      graphDragging = null;
    }
  }

  function toggleProjectExpand(id: string) {
    const next = new Set(expandedProjects);
    if (next.has(id)) {
      next.delete(id);
    } else {
      next.add(id);
    }
    expandedProjects = next;
  }

  function getNodeEdges(name: string): GraphEdge[] {
    return graphEdges.filter(e => e.source === name || e.target === name);
  }

  function getTopNodeEdges(name: string, limit = 6): GraphEdge[] {
    return getNodeEdges(name).sort((a, b) => b.weight - a.weight).slice(0, limit);
  }

  let cbCategories = $derived(codebaseData ? allCategories(codebaseData.stacks) : []);
  const cbGraphView = $derived(
    getGraphView(graphNodes, graphEdges, cbGraphMode, [graphSelected, graphHovered].filter(Boolean) as string[])
  );

  $effect(() => {
    if (activeTab === 'codebase' && codebaseData && graphNodes.length === 0) {
      startGraphSimulation();
    }
  });
</script>

<main class="shell">
  {#if error}
    <section class="state-screen" in:fade>
      <div class="state-card">
        <span class="eyebrow">Telemetry offline</span>
        <h2>We couldn't load the radar workspace.</h2>
        <p>{error}</p>
        <button class="primary-button" onclick={() => void loadData()}>Retry</button>
      </div>
    </section>
  {:else if loading && !radar}
    <section class="state-screen">
      <div class="state-card loading-card">
        <div class="signal-loader" aria-hidden="true"></div>
        <span class="eyebrow">Booting dashboard</span>
        <h2>Mapping the current technology estate...</h2>
        <p>Pulling radar entries, project scans, and governance signals into one view.</p>
      </div>
    </section>
  {:else}
    <div class="app-layout">
      <aside class="sidebar">
        <div class="brand-block">
          <div class="brand-mark">TR</div>
          <div>
            <p class="eyebrow">Technology intelligence</p>
            <h2>Radar cockpit</h2>
          </div>
        </div>

        <nav class="nav-stack" aria-label="Primary navigation">
          {#each tabs as tab}
            <button
              class:active={activeTab === tab.id}
              class="nav-button"
              onclick={() => activeTab = tab.id}
            >
              <span>{tab.label}</span>
              {#if tab.id === 'drift' && dashboard.driftEntries.length > 0}
                <strong>{dashboard.driftEntries.length}</strong>
              {/if}
            </button>
          {/each}
        </nav>

        <section class="sidebar-panel">
          <header>
            <span class="eyebrow">Snapshot</span>
            <h3>Current estate</h3>
          </header>
          <dl class="mini-stats">
            <div>
              <dt>Technologies</dt>
              <dd>{dashboard.techCount}</dd>
            </div>
            <div>
              <dt>Projects</dt>
              <dd>{dashboard.projectCount}</dd>
            </div>
            <div>
              <dt>Coverage</dt>
              <dd>{dashboard.coveragePercent.toFixed(1)}%</dd>
            </div>
            <div>
              <dt>Drift</dt>
              <dd>{dashboard.driftEntries.length}</dd>
            </div>
          </dl>
        </section>

        <section class="sidebar-panel">
          <header>
            <span class="eyebrow">Admin controls</span>
            <h3>Governance mode</h3>
          </header>
          <label class="toggle-row">
            <span>Admin mode</span>
            <span class="toggle">
              <input type="checkbox" bind:checked={isAdminMode} />
              <span></span>
            </span>
          </label>
          <p class="sidebar-copy">
            Turn this on to add missing technologies, update ring decisions, and keep the radar honest.
          </p>
          {#if isAdminMode}
            <button class="secondary-button wide-button" onclick={() => openEntryForm()}>Add technology</button>
          {/if}
        </section>

      </aside>

      <section class="workspace">
        <header class="hero">
          <div class="hero-copy">
            <span class="eyebrow">Governance command deck</span>
            <h1>{radar?.meta.name ?? 'Tech Radar'}</h1>
            <p>
              A single place to see what's standard, what's emerging, where drift is growing, and which
              projects need a governance decision next.
            </p>
            <div class="hero-notes">
              <span>{dashboard.projectCount} projects scanned</span>
              <span>{dashboard.recentDecisionText}</span>
              {#if radar?.meta.version}
                <span>Version {radar.meta.version}</span>
              {/if}
            </div>
          </div>

          <div class="hero-actions">
            <section class="focus-card">
              <div class="focus-card-header">
                <div>
                  <span class="eyebrow">{activeViewSummary.label}</span>
                  <strong>{activeViewSummary.title}</strong>
                </div>
                <span class="theme-status-dot" class:is-dark={isDark} aria-hidden="true"></span>
              </div>
              <p class="focus-copy">{activeViewSummary.description}</p>
              <div class="focus-metrics" aria-label="Current view summary">
                <span>{activeViewSummary.metric}</span>
                <span>{activeViewSummary.detail}</span>
              </div>
              <div class="focus-actions">
                <button class="primary-button" onclick={() => activeTab = 'overview'}>Action board</button>
                <button class="ghost-button" onclick={() => activeTab = 'radar'}>Open radar</button>
                <button class="ghost-button" onclick={() => activeTab = 'projects'}>Review projects</button>
                <button class="ghost-button" onclick={() => activeTab = 'insights'}>AI / MCP lane</button>
              </div>
            </section>

            <section class="control-card" aria-label="Workspace controls">
              <div class="control-bar">
                <div class="control-copy">
                  <span class="eyebrow">Workspace controls</span>
                  <strong>Search, filter, and set the theme in one place</strong>
                  <p>{themeDescription}</p>
                </div>

                <div class="theme-inline">
                  <div class="theme-inline-copy">
                    <span class="eyebrow">Appearance</span>
                    <strong>{activeThemeLabel}</strong>
                  </div>
                  <div class="theme-segmented" role="group" aria-label="Color theme picker">
                    {#each ['system', 'light', 'dark'] as option}
                      <button
                        type="button"
                        class:active={themeMode === option}
                        aria-pressed={themeMode === option}
                        onclick={() => setThemeMode(option as ThemeMode)}
                      >
                        {option}
                      </button>
                    {/each}
                  </div>
                </div>
              </div>

              <label class="search-card compact-card">
                <div class="search-head">
                  <span class="eyebrow">Search</span>
                  <span class="search-meta">Jump straight to a technology, project, or tag</span>
                </div>
                <div class="search-shell">
                  <input
                    bind:this={searchField}
                    bind:value={searchQuery}
                    type="text"
                    placeholder="React, GitOps, infra..."
                    aria-label="Search technologies, projects, and tags"
                  />
                  <kbd>⌘K</kbd>
                </div>
              </label>

              <div class="toolbar compact-card">
                <label>
                  <span>Quadrant</span>
                  <select bind:value={selectedQuadrant}>
                    <option value="all">All</option>
                    {#each allQuadrants as quadrant}
                      <option value={quadrant}>{quadrantMeta[quadrant]?.label ?? quadrant}</option>
                    {/each}
                  </select>
                </label>
                <label>
                  <span>Ring</span>
                  <select bind:value={selectedRing}>
                    <option value="all">All</option>
                    {#each ringOrder as ring}
                      <option value={ring}>{ringMeta[ring].label}</option>
                    {/each}
                  </select>
                </label>
                <label>
                  <span>Tag</span>
                  <select bind:value={selectedTag}>
                    <option value="all">All</option>
                    {#each allTags as tag}
                      <option value={tag}>{tag}</option>
                    {/each}
                  </select>
                </label>
                <button class="ghost-button" onclick={() => activeTab = 'insights'}>
                  AI / MCP watch
                </button>
                <button class="ghost-button" onclick={clearFilters} disabled={!hasActiveFilters}>Reset</button>
              </div>
            </section>
          </div>
        </header>

        <section class="filter-summary" aria-label="Current filters and workspace status">
          <div class="filter-summary-copy">
            <span class="eyebrow">Current focus</span>
            <strong>
              {#if activeTab === 'projects'}
                {dashboard.filteredProjects.length} projects in view
              {:else if activeTab === 'codebase'}
                {codebaseData?.summary.total_technologies ?? 0} technologies mapped
              {:else if activeTab === 'insights'}
                {dashboard.recommendations.length} active recommendations
              {:else if activeTab === 'drift'}
                {dashboard.driftEntries.length} drift entries need attention
              {:else}
                {dashboard.filteredEntries.length} radar entries in the current scope
              {/if}
            </strong>
          </div>

          <div class="filter-chip-row">
            {#if hasActiveFilters}
              {#each activeFilterTokens as token}
                <span class="filter-chip">{token}</span>
              {/each}
            {:else}
              <span class="filter-chip quiet">No filters applied. You're seeing the full portfolio.</span>
            {/if}
          </div>

          <button class="ghost-button" onclick={clearFilters} disabled={!hasActiveFilters}>Clear filters</button>
        </section>

        <section class="metric-grid">
          {#each headlineMetrics as metric}
            <article class="metric-card" data-tone={metric.tone}>
              <span class="eyebrow">{metric.label}</span>
              <strong>{metric.value}</strong>
              <p>{metric.caption}</p>
            </article>
          {/each}
        </section>

        <section class="content-scroll">
          {#if activeTab === 'overview'}
            <div class="page-grid overview-grid" in:fade>
              <article class="panel panel-large">
                <div class="panel-header">
                  <div>
                    <span class="eyebrow">Action board</span>
                    <h2>What the team should do next</h2>
                  </div>
                  {#if isAdminMode}
                    <button class="ghost-button" onclick={() => activeTab = 'insights'}>Open intelligence</button>
                  {/if}
                </div>

                <div class="split-grid recommendation-grid">
                  <section>
                    <h3>Urgent calls</h3>
                    {#if dashboard.criticalRecommendations.length > 0}
                      <div class="signal-stack">
                        {#each dashboard.criticalRecommendations as recommendation}
                          <button class="signal-row actionable" onclick={() => applyRecommendation(recommendation)}>
                            <div>
                              <strong>{recommendation.title}</strong>
                              <span>{recommendation.summary}</span>
                            </div>
                            <span class={`signal-pill ${recommendation.tone}`}>{recommendation.actionLabel}</span>
                          </button>
                        {/each}
                      </div>
                    {:else}
                      <div class="empty-card">No urgent governance interventions are standing out right now.</div>
                    {/if}
                  </section>

                  <section>
                    <h3>Strategic follow-ups</h3>
                    {#if dashboard.strategicRecommendations.length > 0}
                      <div class="signal-stack">
                        {#each dashboard.strategicRecommendations.slice(0, 4) as recommendation}
                          <button class="signal-row actionable" onclick={() => applyRecommendation(recommendation)}>
                            <div>
                              <strong>{recommendation.title}</strong>
                              <span>{recommendation.summary}</span>
                            </div>
                            <span class={`signal-pill ${recommendation.tone}`}>{recommendation.actionLabel}</span>
                          </button>
                        {/each}
                      </div>
                    {:else}
                      <div class="empty-card">No medium-priority follow-ups are waiting behind the urgent queue.</div>
                    {/if}
                  </section>
                </div>
              </article>

              <article class="panel overview-rail">
                <div class="panel-header">
                  <div>
                    <span class="eyebrow">Signal ledger</span>
                    <h2>Context behind the next governance call</h2>
                  </div>
                  <p>Keep the summary tight here, then jump into the deeper tabs when a signal needs a full review.</p>
                </div>

                <div class="overview-rail-sections">
                  <section class="overview-subsection">
                    <div class="overview-subsection-header">
                      <h3>Portfolio balance</h3>
                      <span class="muted">{dashboard.techCount} tracked decisions</span>
                    </div>
                    <h3>By ring</h3>
                    {#each dashboard.ringBreakdown as stat}
                      <div class="bar-row">
                        <span>{ringMeta[stat.label]?.label ?? stat.label}</span>
                        <div class="bar-track">
                          <div class="bar-fill" style={`width:${Math.max(10, (stat.count / Math.max(1, dashboard.techCount)) * 100)}%; --bar-color: ${ringMeta[stat.label]?.accent ?? '#1d4ed8'}`}></div>
                        </div>
                        <strong>{stat.count}</strong>
                      </div>
                    {/each}

                    <h3>By quadrant</h3>
                    {#each dashboard.quadrantBreakdown as stat}
                      <div class="bar-row">
                        <span>{quadrantMeta[stat.label]?.label ?? stat.label}</span>
                        <div class="bar-track">
                          <div class="bar-fill neutral" style={`width:${Math.max(10, (stat.count / Math.max(1, dashboard.techCount)) * 100)}%`}></div>
                        </div>
                        <strong>{stat.count}</strong>
                      </div>
                    {/each}
                  </section>

                  <section class="overview-subsection">
                    <div class="overview-subsection-header">
                      <h3>AI / MCP frontier</h3>
                      <button class="ghost-button subtle-button" onclick={() => activeTab = 'insights'}>Open lane</button>
                    </div>

                    {#if dashboard.aiMcpOpportunityQueue.length > 0}
                      <div class="signal-stack">
                        {#each dashboard.aiMcpOpportunityQueue.slice(0, 3) as item}
                          <button class="signal-row actionable" onclick={() => {
                            if (isAdminMode) {
                              openEntryForm(null, item.name);
                            } else {
                              searchQuery = item.name;
                              activeTab = 'insights';
                            }
                          }}>
                            <div>
                              <strong>{item.name}</strong>
                              <span>{item.count} projects are already experimenting without a radar call.</span>
                            </div>
                            <span class="signal-pill watch">{item.count}</span>
                          </button>
                        {/each}
                      </div>
                    {:else if dashboard.aiMcpAdoptionLeaders.length > 0}
                      <div class="signal-stack">
                        {#each dashboard.aiMcpAdoptionLeaders.slice(0, 3) as item}
                          <div class="signal-row">
                            <div>
                              <strong>{item.name}</strong>
                              <span>Tracked agentic tooling already present in {item.count} monitored projects.</span>
                            </div>
                            <span class="signal-pill good">{item.count}</span>
                          </div>
                        {/each}
                      </div>
                    {:else}
                      <div class="empty-card">No AI or MCP signals are showing up in scans yet.</div>
                    {/if}
                  </section>

                  <section class="overview-subsection">
                    <div class="overview-subsection-header">
                      <h3>Lifecycle motion</h3>
                      <span class="muted">{dashboard.motionEntries.length} entries scored</span>
                    </div>

                    {#if dashboard.motionEntries.length > 0}
                      <div class="signal-stack">
                        {#each dashboard.motionEntries.slice(0, 3) as motion}
                          <button class="signal-row actionable" onclick={() => {
                            openEntryById(motion.entryId);
                            activeTab = 'insights';
                          }}>
                            <div>
                              <strong>{motion.name}</strong>
                              <span>{motion.summary}</span>
                            </div>
                            <span class={`signal-pill ${motionTone(motion)}`}>{motionLabel(motion)}</span>
                          </button>
                        {/each}
                      </div>
                    {:else}
                      <div class="empty-card">No entries have enough movement history to score motion yet.</div>
                    {/if}
                  </section>

                  <section class="overview-subsection">
                    <div class="overview-subsection-header">
                      <h3>Recent movement</h3>
                      <span class="muted">{dashboard.recentMoves.length} recorded changes</span>
                    </div>

                    {#if dashboard.recentMoves.length > 0}
                      <div class="timeline-list">
                        {#each dashboard.recentMoves.slice(0, 4) as move}
                          <div class="timeline-item">
                            <div class="timeline-dot" style={`--dot-color:${ringMeta[move.to_ring]?.accent ?? '#1d4ed8'}`}></div>
                            <div>
                              <strong>{move.name}</strong>
                              <p>
                                {move.from_ring ? `${move.from_ring.toUpperCase()} -> ${move.to_ring.toUpperCase()}` : `Entered ${move.to_ring.toUpperCase()}`}
                                {' '}on {formatDate(move.date)}
                              </p>
                              {#if move.reason}
                                <span>{move.reason}</span>
                              {/if}
                            </div>
                          </div>
                        {/each}
                      </div>
                    {:else}
                      <div class="empty-card">No movement history yet. New ring changes will start building a governance timeline.</div>
                    {/if}
                  </section>
                </div>
              </article>

              <article class="panel panel-wide">
                <div class="panel-header">
                  <div>
                    <span class="eyebrow">Project spotlight</span>
                    <h2>Which teams need the next governance nudge</h2>
                  </div>
                  <button class="ghost-button" onclick={() => activeTab = 'projects'}>View all projects</button>
                </div>

                {#if dashboard.projectSpotlight.length > 0}
                  <div class="project-spotlight-list">
                    {#each dashboard.projectSpotlight.slice(0, 4) as project}
                      <button class="project-row actionable" data-risk={project.riskLevel} onclick={() => {
                        searchQuery = project.name;
                        activeTab = 'projects';
                      }}>
                        <div class="project-row-head">
                          <div>
                            <h3>{project.name}</h3>
                            <p>Last scan {formatRelativeDate(project.last_scan)}</p>
                          </div>
                          <span class={`risk-badge ${project.riskLevel}`}>{riskLabel(project.riskLevel)}</span>
                        </div>

                        <div class="project-row-summary">
                          <div class="score-row compact">
                            <strong>{project.radarAlignmentScore}</strong>
                            <span>alignment score</span>
                          </div>
                          <div class="project-row-metrics">
                            <span>{project.coveragePercent.toFixed(1)}% covered</span>
                            <span>{project.untrackedTechs} uncharted</span>
                            <span>{project.holdTechs} hold</span>
                          </div>
                        </div>

                        <div class="meter">
                          <div class="meter-fill" style={`width:${project.coveragePercent}%`}></div>
                        </div>

                        <div class="chip-row">
                          {#each project.topTechs as tech}
                            <span>{tech}</span>
                          {/each}
                        </div>
                      </button>
                    {/each}
                  </div>
                {:else}
                  <div class="empty-card">No projects match the current search. Try clearing filters.</div>
                {/if}
              </article>
            </div>
          {/if}

          {#if activeTab === 'radar'}
            <div class="page-grid radar-page" in:fade>
              <article class="panel panel-large radar-panel">
                <div class="panel-header">
                  <div>
                    <span class="eyebrow">Visual radar</span>
                    <h2>Live portfolio map</h2>
                  </div>
                  <p>Hover to inspect. Click a blip to open the full decision card.</p>
                </div>

                <div class="radar-canvas">
                  <svg viewBox="0 0 680 680" aria-label="Technology radar">
                    {#each ringOrder.slice().reverse() as ring, reverseIndex}
                      {@const radius = [80, 156, 232, 300][3 - reverseIndex]}
                      <circle
                        cx="340"
                        cy="340"
                        r={radius}
                        class="ring-circle"
                        style={`--ring-color:${ringMeta[ring].accent}`}
                      />
                    {/each}

                    <line x1="340" y1="32" x2="340" y2="648" class="divider-line" />
                    <line x1="32" y1="340" x2="648" y2="340" class="divider-line" />

                    <text x="54" y="54" class="quadrant-label">LANGUAGES</text>
                    <text x="626" y="54" text-anchor="end" class="quadrant-label">FRAMEWORKS</text>
                    <text x="54" y="626" class="quadrant-label">TOOLS</text>
                    <text x="626" y="626" text-anchor="end" class="quadrant-label">PLATFORMS</text>

                    {#each dashboard.renderableEntries as entry, index}
                      {@const pos = getBlipPosition(entry, index)}
                      {@const motion = entryMotion(entry)}
                      <g
                        class="blip"
                        data-motion={motion.status}
                        class:hovered={hoveredEntry === entry.name || selectedEntry?.name === entry.name}
                        transform={`translate(${pos.x}, ${pos.y})`}
                        role="button"
                        tabindex="0"
                        onmouseenter={() => hoveredEntry = entry.name}
                        onmouseleave={() => hoveredEntry = null}
                        onclick={() => selectedEntry = entry}
                        onkeydown={(event) => {
                          if (event.key === 'Enter' || event.key === ' ') {
                            event.preventDefault();
                            selectedEntry = entry;
                          }
                        }}
                      >
                        {#if motion.status === 'accelerating' || motion.status === 'heating' || motion.status === 'cooling'}
                          <circle r={motion.status === 'accelerating' ? 15 : 13} class={`motion-halo ${motion.status}`}></circle>
                        {:else if motion.status === 'new'}
                          <rect x="-12" y="-12" width="24" height="24" rx="7" class="motion-frame"></rect>
                        {/if}

                        {#if entry.moved > 0}
                          <path class="blip-core" d="M 0 -11 L 10 7 L -10 7 Z" fill={ringMeta[entry.ring]?.accent ?? '#1d4ed8'} />
                        {:else if entry.moved < 0}
                          <path class="blip-core" d="M 0 11 L 10 -7 L -10 -7 Z" fill={ringMeta[entry.ring]?.accent ?? '#b42318'} />
                        {:else}
                          <circle class="blip-core" r="8" fill={ringMeta[entry.ring]?.accent ?? '#1d4ed8'} />
                        {/if}

                        {#if (entry.project_ids?.length ?? 0) > 0}
                          <circle cx="12" cy="-12" r="8" class="project-count-dot"></circle>
                          <text x="12" y="-9" text-anchor="middle" class="project-count-text">{entry.project_ids?.length}</text>
                        {/if}

                        <text y="-22" text-anchor="middle" class="blip-label">{entry.name}</text>
                      </g>
                    {/each}
                  </svg>
                </div>
              </article>

              <article class="panel">
                <div class="panel-header">
                  <div>
                    <span class="eyebrow">Field guide</span>
                    <h2>How to read this radar</h2>
                  </div>
                </div>

                <div class="legend-stack">
                  {#each ringOrder as ring}
                    <div class="legend-row">
                      <span class="legend-swatch" style={`--swatch:${ringMeta[ring].accent}`}></span>
                      <div>
                        <strong>{ringMeta[ring].label}</strong>
                        <p>{ringMeta[ring].description}</p>
                      </div>
                    </div>
                  {/each}
                </div>

                <div class="legend-note">
                  Motion halos spotlight lifecycle change: teal means a technology is heating up, amber frames mark new arrivals, and dashed red halos flag cooling decisions.
                </div>

                {#if dashboard.offCanvasEntries.length > 0}
                  <div class="offcanvas-block">
                    <h3>Off-canvas domains</h3>
                    <p>These entries use quadrants outside the classic four-quadrant radar.</p>
                    <div class="chip-cloud">
                      {#each dashboard.offCanvasEntries as entry}
                        <button class="tag-chip actionable" onclick={() => selectedEntry = entry}>{entry.name}</button>
                      {/each}
                    </div>
                  </div>
                {/if}
              </article>

              <article class="panel panel-wide">
                <div class="panel-header">
                  <div>
                    <span class="eyebrow">Ring collections</span>
                    <h2>Decision inventory</h2>
                  </div>
                </div>

                <div class="ring-section-grid">
                  {#each dashboard.radarSections as section}
                    <section class="ring-section">
                      <header>
                        <div class="ring-headline">
                          <span class="legend-swatch" style={`--swatch:${ringMeta[section.ring].accent}`}></span>
                          <div>
                            <h3>{section.label}</h3>
                            <p>{section.description}</p>
                          </div>
                        </div>
                      </header>

                      {#if section.entries.length > 0}
                        <div class="entry-list">
                          {#each section.entries as entry}
                            {@const motion = entryMotion(entry)}
                            <button class="entry-card" onclick={() => selectedEntry = entry}>
                              <div class="entry-card-top">
                                <strong>{entry.name}</strong>
                                <span class="entry-meta">{entry.project_ids?.length ?? 0} projects</span>
                              </div>
                              <p>{entry.description}</p>
                              {#if motion.moveCount > 0}
                                <p class="entry-note">{motion.summary}</p>
                              {/if}
                              <div class="chip-row">
                                <span>{quadrantMeta[entry.quadrant]?.label ?? entry.quadrant}</span>
                                <span>{movementLabel(entry)}</span>
                                {#if motion.moveCount > 0}
                                  <span>{motionLabel(motion)}</span>
                                {/if}
                                {#if entry.tags?.[0]}
                                  <span>#{entry.tags[0]}</span>
                                {/if}
                              </div>
                            </button>
                          {/each}
                        </div>
                      {:else}
                        <div class="empty-card">No entries match the current filters in this ring.</div>
                      {/if}
                    </section>
                  {/each}
                </div>
              </article>
            </div>
          {/if}

          {#if activeTab === 'projects'}
            <div class="page-grid" in:fade>
              <article class="panel panel-wide">
                <div class="panel-header">
                  <div>
                    <span class="eyebrow">Project intelligence</span>
                    <h2>Alignment across the monitored portfolio</h2>
                  </div>
                  <p>{dashboard.filteredProjects.length} projects match the current search and filters.</p>
                </div>

                {#if dashboard.filteredProjects.length > 0}
                  <div class="project-grid project-grid-wide">
                    {#each dashboard.filteredProjects as project}
                      <article class="project-card" data-risk={project.riskLevel}>
                        <div class="project-card-header">
                          <div>
                            <h3>{project.name}</h3>
                            <p>Scanned {formatRelativeDate(project.last_scan)}</p>
                          </div>
                          <span class={`risk-badge ${project.riskLevel}`}>{riskLabel(project.riskLevel)}</span>
                        </div>

                        <div class="score-row">
                          <strong>{project.radarAlignmentScore}</strong>
                          <span>alignment score</span>
                        </div>

                        <div class="meter">
                          <div class="meter-fill" style={`width:${project.coveragePercent}%`}></div>
                        </div>

                        <div class="project-metrics">
                          <span>{project.coveragePercent.toFixed(1)}% covered</span>
                          <span>{project.totalTechs} technologies</span>
                          <span>{project.untrackedTechs} uncharted</span>
                          <span>{project.holdTechs} hold</span>
                        </div>

                        <div class="chip-row">
                          {#each project.topTechs as tech}
                            <span>{tech}</span>
                          {/each}
                        </div>
                      </article>
                    {/each}
                  </div>
                {:else}
                  <div class="empty-card">No projects match the current search. Try clearing filters.</div>
                {/if}
              </article>
            </div>
          {/if}

          {#if activeTab === 'insights'}
            <div class="page-grid" in:fade>
              <article class="panel panel-wide">
                <div class="panel-header">
                  <div>
                    <span class="eyebrow">Governance recommendations</span>
                    <h2>Ranked actions from the current signal set</h2>
                  </div>
                  <p>{dashboard.recommendations.length} recommendations are currently active.</p>
                </div>

                {#if dashboard.recommendations.length > 0}
                  <div class="signal-stack">
                    {#each dashboard.recommendations as recommendation}
                      <button class="signal-row actionable" onclick={() => applyRecommendation(recommendation)}>
                        <div>
                          <strong>{recommendation.title}</strong>
                          <span>{recommendation.summary}</span>
                        </div>
                        <span class={`signal-pill ${recommendation.tone}`}>{recommendation.actionLabel}</span>
                      </button>
                    {/each}
                  </div>
                {:else}
                  <div class="empty-card">No recommendations are active. The current signals look balanced enough to monitor without immediate intervention.</div>
                {/if}
              </article>

              <article class="panel">
                <div class="panel-header">
                  <div>
                    <span class="eyebrow">AI / MCP lane</span>
                    <h2>Agentic signals to govern next</h2>
                  </div>
                  <p>{dashboard.aiMcpProjectCount} projects currently carry AI or MCP traces.</p>
                </div>

                <div class="signal-stack">
                  {#if dashboard.aiMcpOpportunityQueue.length > 0}
                    {#each dashboard.aiMcpOpportunityQueue as item}
                      <button class="signal-row actionable" onclick={() => {
                        if (isAdminMode) {
                          openEntryForm(null, item.name);
                        } else {
                          searchQuery = item.name;
                        }
                      }}>
                        <div>
                          <strong>{item.name}</strong>
                          <span>{item.count} projects show this AI/MCP signal without a formal decision.</span>
                        </div>
                        <span class="signal-pill watch">{item.count}</span>
                      </button>
                    {/each}
                  {:else if dashboard.aiMcpAdoptionLeaders.length > 0}
                    {#each dashboard.aiMcpAdoptionLeaders as item}
                      <div class="signal-row">
                        <div>
                          <strong>{item.name}</strong>
                          <span>Tracked AI/MCP standard in {item.count} monitored projects.</span>
                        </div>
                        <span class="signal-pill good">{item.count}</span>
                      </div>
                    {/each}
                  {:else}
                    <div class="empty-card">No AI or MCP discoveries are showing up in the latest scans.</div>
                  {/if}
                </div>
              </article>

              <article class="panel">
                <div class="panel-header">
                  <div>
                    <span class="eyebrow">Opportunity queue</span>
                    <h2>Uncharted technologies</h2>
                  </div>
                </div>
                {#if dashboard.opportunityQueue.length > 0}
                  <div class="signal-stack">
                    {#each dashboard.opportunityQueue as item}
                      <button class="signal-row actionable" onclick={() => {
                        if (isAdminMode) {
                          openEntryForm(null, item.name);
                        } else {
                          searchQuery = item.name;
                        }
                      }}>
                        <div>
                          <strong>{item.name}</strong>
                          <span>{item.count} projects are already using it.</span>
                        </div>
                        <span class="signal-pill watch">{item.count}</span>
                      </button>
                    {/each}
                  </div>
                {:else}
                  <div class="empty-card">No uncharted technologies are showing up in scans right now.</div>
                {/if}
              </article>

              <article class="panel">
                <div class="panel-header">
                  <div>
                    <span class="eyebrow">Adoption leaders</span>
                    <h2>Most entrenched standards</h2>
                  </div>
                </div>
                {#if dashboard.adoptionLeaders.length > 0}
                  <div class="signal-stack">
                    {#each dashboard.adoptionLeaders as item}
                      <div class="signal-row">
                        <div>
                          <strong>{item.name}</strong>
                          <span>Used in {item.count} monitored projects.</span>
                        </div>
                        <span class="signal-pill good">{item.count}</span>
                      </div>
                    {/each}
                  </div>
                {:else}
                  <div class="empty-card">No adoption leaders yet because tracked usage data is still sparse.</div>
                {/if}
              </article>

              <article class="panel">
                <div class="panel-header">
                  <div>
                    <span class="eyebrow">Drift exposure</span>
                    <h2>Hold technologies still in flight</h2>
                  </div>
                </div>
                {#if dashboard.driftLeaders.length > 0}
                  <div class="signal-stack">
                    {#each dashboard.driftLeaders as item}
                      <div class="signal-row">
                        <div>
                          <strong>{item.name}</strong>
                          <span>Still present in {item.count} projects.</span>
                        </div>
                        <span class="signal-pill danger">{item.count}</span>
                      </div>
                    {/each}
                  </div>
                {:else}
                  <div class="empty-card">No active hold exposure is showing up in project scans.</div>
                {/if}
              </article>

              <article class="panel">
                <div class="panel-header">
                  <div>
                    <span class="eyebrow">ADR evidence</span>
                    <h2>Decision documents tied to the radar</h2>
                  </div>
                </div>
                {#if dashboard.adrEntries.length > 0}
                  <div class="signal-stack">
                    {#each dashboard.adrEntries as entry}
                      <button class="signal-row actionable" onclick={() => selectedEntry = entry}>
                        <div>
                          <strong>{entry.name}</strong>
                          <span>{entry.link}</span>
                        </div>
                        <span class="signal-pill good">ADR</span>
                      </button>
                    {/each}
                  </div>
                {:else}
                  <div class="empty-card">No ADR-linked radar entries yet. Scan a repo with ADRs or attach one in admin mode.</div>
                {/if}
              </article>

              <article class="panel">
                <div class="panel-header">
                  <div>
                    <span class="eyebrow">Lifecycle vectors</span>
                    <h2>Momentum behind each decision</h2>
                  </div>
                </div>

                <div class="split-grid motion-split">
                  <section>
                    <h3>Heating up</h3>
                    {#if dashboard.heatingUpEntries.length > 0}
                      <div class="signal-stack">
                        {#each dashboard.heatingUpEntries as motion}
                          <button class="signal-row actionable" onclick={() => openEntryById(motion.entryId)}>
                            <div>
                              <strong>{motion.name}</strong>
                              <span>{motion.summary}</span>
                            </div>
                            <span class={`signal-pill ${motionTone(motion)}`}>{motionLabel(motion)}</span>
                          </button>
                        {/each}
                      </div>
                    {:else}
                      <div class="empty-card">No technologies are showing upward motion beyond the current baseline.</div>
                    {/if}
                  </section>

                  <section>
                    <h3>Cooling down</h3>
                    {#if dashboard.coolingEntries.length > 0}
                      <div class="signal-stack">
                        {#each dashboard.coolingEntries as motion}
                          <button class="signal-row actionable" onclick={() => openEntryById(motion.entryId)}>
                            <div>
                              <strong>{motion.name}</strong>
                              <span>{motion.summary}</span>
                            </div>
                            <span class={`signal-pill ${motionTone(motion)}`}>{motionLabel(motion)}</span>
                          </button>
                        {/each}
                      </div>
                    {:else}
                      <div class="empty-card">No cooling technologies are currently standing out.</div>
                    {/if}
                  </section>
                </div>
              </article>

              <article class="panel">
                <div class="panel-header">
                  <div>
                    <span class="eyebrow">Governance tempo</span>
                    <h2>Recent ring changes</h2>
                  </div>
                </div>
                {#if dashboard.recentMoves.length > 0}
                  <div class="timeline-list">
                    {#each dashboard.recentMoves as move}
                      <div class="timeline-item">
                        <div class="timeline-dot" style={`--dot-color:${ringMeta[move.to_ring]?.accent ?? '#1d4ed8'}`}></div>
                        <div>
                          <strong>{move.name}</strong>
                          <p>
                            {move.from_ring ? `${move.from_ring.toUpperCase()} -> ${move.to_ring.toUpperCase()}` : `Added to ${move.to_ring.toUpperCase()}`}
                          </p>
                          <span>{formatDate(move.date)}</span>
                        </div>
                      </div>
                    {/each}
                  </div>
                {:else}
                  <div class="empty-card">No history yet. Start recording ring changes to unlock movement analytics.</div>
                {/if}
              </article>

              <!-- ============ NEW INTELLIGENCE PANELS ============ -->

              <article class="panel panel-wide">
                <div class="panel-header">
                  <div>
                    <span class="eyebrow">Dependency intelligence</span>
                    <h2>Cross-project hotspots</h2>
                  </div>
                  <p>Dependencies shared across the most projects — these are your foundational standards.</p>
                </div>
                {#if dashboard.dependencyHotspots.length > 0}
                  <div class="intel-hotspot-grid">
                    {#each dashboard.dependencyHotspots.slice(0, 12) as hotspot}
                      <div class="intel-hotspot-card">
                        <div class="intel-hotspot-header">
                          <strong>{hotspot.name}</strong>
                          <span class="signal-pill good">{hotspot.projectCount} projects</span>
                        </div>
                        <span class="intel-source-tag">{hotspot.source}</span>
                        <div class="intel-project-tags">
                          {#each hotspot.projects.slice(0, 4) as pname}
                            <span class="cb-pill small">{pname}</span>
                          {/each}
                          {#if hotspot.projects.length > 4}
                            <span class="cb-pill small">+{hotspot.projects.length - 4}</span>
                          {/if}
                        </div>
                      </div>
                    {/each}
                  </div>
                {:else}
                  <div class="empty-card">No cross-project dependency data available yet. Run scans to populate.</div>
                {/if}
              </article>

              <article class="panel">
                <div class="panel-header">
                  <div>
                    <span class="eyebrow">Language ecosystems</span>
                    <h2>Technology landscape by runtime</h2>
                  </div>
                </div>
                {#if dashboard.ecosystemBreakdown.length > 0}
                  <div class="intel-eco-stack">
                    {#each dashboard.ecosystemBreakdown as eco}
                      <div class="intel-eco-row">
                        <div class="intel-eco-label">
                          <span class="intel-eco-icon">{eco.icon}</span>
                          <strong>{eco.ecosystem}</strong>
                        </div>
                        <div class="intel-eco-bars">
                          <div class="intel-eco-bar">
                            <div class="bar-track">
                              <div class="bar-fill neutral" style={`width:${Math.min(100, (eco.projectCount / Math.max(1, dashboard.projectCount)) * 100)}%`}></div>
                            </div>
                            <span class="muted">{eco.projectCount} projects</span>
                          </div>
                          <div class="intel-eco-bar">
                            <div class="bar-track">
                              <div class="bar-fill" style={`width:${Math.min(100, (eco.techCount / Math.max(1, 100)) * 100)}%; --bar-color: rgba(76, 199, 187, 0.7)`}></div>
                            </div>
                            <span class="muted">{eco.techCount} unique deps</span>
                          </div>
                        </div>
                      </div>
                    {/each}
                  </div>
                {:else}
                  <div class="empty-card">No ecosystem data yet — scan your workspace to see the breakdown.</div>
                {/if}
              </article>

              <article class="panel">
                <div class="panel-header">
                  <div>
                    <span class="eyebrow">Polyglot complexity</span>
                    <h2>Project runtime breadth</h2>
                  </div>
                  <p>How many language ecosystems each project spans.</p>
                </div>
                {#if dashboard.polyglotProfiles.length > 0}
                  <div class="signal-stack">
                    {#each dashboard.polyglotProfiles.slice(0, 10) as profile}
                      <div class="signal-row">
                        <div>
                          <strong>{profile.name}</strong>
                          <span>{profile.ecosystems.join(' · ')} — {profile.totalTechs} dependencies</span>
                        </div>
                        <span class={`signal-pill ${profile.complexityLabel === 'Complex' ? 'danger' : profile.complexityLabel === 'Polyglot' ? 'watch' : 'good'}`}>
                          {profile.complexityLabel}
                        </span>
                      </div>
                    {/each}
                  </div>
                {:else}
                  <div class="empty-card">Polyglot analysis requires project scan data.</div>
                {/if}
              </article>

              <article class="panel panel-wide">
                <div class="panel-header">
                  <div>
                    <span class="eyebrow">Stack convergence</span>
                    <h2>Which projects share the most DNA</h2>
                  </div>
                  <p>Jaccard similarity of technology stacks — higher means more shared dependencies.</p>
                </div>
                {#if dashboard.stackSimilarities.length > 0}
                  <div class="intel-similarity-grid">
                    {#each dashboard.stackSimilarities.slice(0, 8) as sim}
                      <div class="intel-similarity-card">
                        <div class="intel-sim-header">
                          <strong>{sim.projectA}</strong>
                          <span class="intel-sim-arrow">⇄</span>
                          <strong>{sim.projectB}</strong>
                        </div>
                        <div class="intel-sim-meter">
                          <div class="bar-track">
                            <div class="bar-fill neutral" style={`width:${Math.round(sim.similarity * 100)}%`}></div>
                          </div>
                          <span class="muted">{Math.round(sim.similarity * 100)}% overlap · {sim.sharedCount} shared</span>
                        </div>
                        <div class="intel-project-tags">
                          {#each sim.sharedTechs.slice(0, 6) as tech}
                            <span class="cb-pill small">{tech}</span>
                          {/each}
                          {#if sim.sharedTechs.length > 6}
                            <span class="cb-pill small">+{sim.sharedTechs.length - 6}</span>
                          {/if}
                        </div>
                      </div>
                    {/each}
                  </div>
                {:else}
                  <div class="empty-card">Stack convergence requires at least 2 projects with shared dependencies.</div>
                {/if}
              </article>
            </div>
          {/if}

          {#if activeTab === 'drift'}
            <div class="page-grid" in:fade>
              <article class="panel panel-wide">
                <div class="panel-header">
                  <div>
                    <span class="eyebrow">Drift watch</span>
                    <h2>Hold decisions that still have active usage</h2>
                  </div>
                </div>

                {#if dashboard.driftEntries.length > 0}
                  <div class="drift-grid">
                    {#each dashboard.driftEntries as entry}
                      <button class="drift-card" onclick={() => selectedEntry = entry}>
                        <div class="drift-card-header">
                          <div>
                            <h3>{entry.name}</h3>
                            <p>{entry.description}</p>
                          </div>
                          <span class="signal-pill danger">Hold</span>
                        </div>
                        <div class="chip-row">
                          {#each entry.project_ids ?? [] as projectId}
                            <span>{projectId}</span>
                          {/each}
                        </div>
                      </button>
                    {/each}
                  </div>
                {:else}
                  <div class="empty-card">No drift detected. The hold ring is currently clean.</div>
                {/if}
              </article>
            </div>
          {/if}

          {#if activeTab === 'codebase'}
            <div class="page-grid" in:fade>
              <article class="panel panel-wide">
                <div class="panel-header">
                  <div>
                    <span class="eyebrow">Codebase intelligence</span>
                    <h2>Architecture & technology landscape</h2>
                  </div>
                  {#if codebaseData}
                    <div class="cb-summary-pills">
                      <span class="cb-pill">{codebaseData.summary.total_technologies} technologies</span>
                      <span class="cb-pill">{codebaseData.summary.total_projects} projects</span>
                      <span class="cb-pill">{codebaseData.summary.tracked_percent}% tracked</span>
                      <span class="cb-pill">{codebaseData.summary.total_patterns} patterns</span>
                    </div>
                  {/if}
                </div>

                {#if !codebaseData}
                  <div class="empty-card">No codebase data available. Scan projects and publish results to populate this view.</div>
                {:else}
                  <nav class="cb-sub-tabs">
                    <button class:active={cbSubTab === 'graph'} onclick={() => cbSubTab = 'graph'}>Dependency graph</button>
                    <button class:active={cbSubTab === 'stacks'} onclick={() => cbSubTab = 'stacks'}>Tech stacks</button>
                    <button class:active={cbSubTab === 'files'} onclick={() => cbSubTab = 'files'}>File explorer</button>
                  </nav>

                  {#if cbSubTab === 'graph'}
                    <div class="cb-graph-wrap" in:fade>
                      <div class="cb-graph-toolbar">
                        <div class="cb-graph-summary">
                          <strong>{cbGraphView.nodes.length} of {graphNodes.length} nodes shown</strong>
                          <span>
                            {cbGraphMode === 'focused'
                              ? 'Focused mode keeps tracked and high-signal dependencies readable.'
                              : 'Full mode shows the entire dependency graph with restrained labels.'}
                          </span>
                        </div>
                        <div class="cb-graph-mode" role="group" aria-label="Dependency graph density">
                          <button
                            type="button"
                            class:active={cbGraphMode === 'focused'}
                            aria-pressed={cbGraphMode === 'focused'}
                            onclick={() => cbGraphMode = 'focused'}
                          >
                            Focused
                          </button>
                          <button
                            type="button"
                            class:active={cbGraphMode === 'full'}
                            aria-pressed={cbGraphMode === 'full'}
                            onclick={() => cbGraphMode = 'full'}
                          >
                            Full
                          </button>
                        </div>
                      </div>

                      {#if codebaseData.patterns.length > 0}
                        <div class="cb-pattern-strip">
                          {#each codebaseData.patterns as pat}
                            <span class="cb-pattern-badge" title={pat.signal}>
                              <span>{pat.icon}</span>
                              <span>{pat.pattern}</span>
                              <strong>{pat.projects.length}</strong>
                            </span>
                          {/each}
                        </div>
                      {/if}

                      <!-- svelte-ignore a11y_no_noninteractive_element_interactions a11y_no_noninteractive_tabindex -->
                      <div
                        class="cb-graph-canvas"
                        role="application"
                        aria-label="Technology dependency graph canvas"
                        onmousemove={onGraphMouseMove}
                        onmouseup={onGraphMouseUp}
                        onmouseleave={onGraphMouseUp}
                      >
                        <svg
                          viewBox="0 0 800 560"
                          class="dep-graph"
                          role="img"
                          aria-label="Technology dependency graph"
                        >
                          <defs>
                            <filter id="graph-glow">
                              <feGaussianBlur in="SourceGraphic" stdDeviation="3" />
                            </filter>
                          </defs>

                          {#each cbGraphView.edges as edge}
                            {@const sn = cbGraphView.nodes.find(n => n.name === edge.source)}
                            {@const tn = cbGraphView.nodes.find(n => n.name === edge.target)}
                            {#if sn && tn}
                              <line
                                x1={sn.x} y1={sn.y}
                                x2={tn.x} y2={tn.y}
                                class="graph-edge"
                                class:highlighted={graphHovered === edge.source || graphHovered === edge.target || graphSelected === edge.source || graphSelected === edge.target}
                                style={`stroke-width:${Math.max(1, Math.min(4, edge.weight))};opacity:${(graphHovered === edge.source || graphHovered === edge.target) ? 0.7 : 0.15}`}
                              />
                            {/if}
                          {/each}

                          {#each cbGraphView.nodes as node}
                            <g
                              class="graph-node"
                              class:hovered={graphHovered === node.name}
                              class:selected={graphSelected === node.name}
                              transform={`translate(${node.x},${node.y})`}
                              onmouseenter={() => graphHovered = node.name}
                              onmouseleave={() => graphHovered = null}
                              onmousedown={(e) => onGraphNodeMouseDown(node, e)}
                              onclick={() => graphSelected = graphSelected === node.name ? null : node.name}
                              onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); graphSelected = graphSelected === node.name ? null : node.name; } }}
                              role="button"
                              tabindex="0"
                            >
                              {#if graphHovered === node.name || graphSelected === node.name}
                                <circle r={node.radius + 4} fill={nodeColor(node)} opacity="0.15" filter="url(#graph-glow)" />
                              {/if}
                              <circle
                                r={node.radius}
                                fill={nodeColor(node)}
                                opacity={nodeOpacity(node)}
                                stroke={node.is_tracked ? nodeColor(node) : '#94a3b8'}
                                stroke-width={node.is_tracked ? 2 : 1}
                                stroke-dasharray={node.is_tracked ? 'none' : '3,2'}
                              />
                              {#if shouldShowGraphLabel(node, cbGraphView.edges, graphHovered, graphSelected, cbGraphMode)}
                                <text
                                  y={-node.radius - 8}
                                  text-anchor="middle"
                                  class="graph-label"
                                  style={`font-size:${node.usage >= 5 ? 11 : 9}px`}
                                >{node.name}</text>
                              {/if}
                            </g>
                          {/each}
                        </svg>
                      </div>

                      {#if graphSelected}
                        {@const sel = graphNodes.find(n => n.name === graphSelected)}
                        {#if sel}
                          <div class="cb-node-detail" in:fly={{ y: 10, duration: 200 }}>
                            <div class="cb-node-detail-header">
                              <span class="cb-node-dot" style={`background:${nodeColor(sel)}`}></span>
                              <strong>{sel.name}</strong>
                              <span class="cb-pill small">{sel.quadrant}</span>
                              {#if sel.ring}
                                <span class="cb-pill small">{sel.ring}</span>
                              {/if}
                              <span class="cb-pill small">{sel.is_tracked ? 'Tracked' : 'Uncharted'}</span>
                            </div>
                            <p>Used in {sel.usage} project{sel.usage !== 1 ? 's' : ''}. Connected to {getNodeEdges(sel.name).length} other technologies.</p>
                            {#if getNodeEdges(sel.name).length > 0}
                              <div class="cb-connections">
                                {#each getTopNodeEdges(sel.name) as edge}
                                  {@const other = edge.source === sel.name ? edge.target : edge.source}
                                  <span class="cb-connection-chip">{other} <strong>×{edge.weight}</strong></span>
                                {/each}
                              </div>
                            {/if}
                          </div>
                        {/if}
                      {/if}

                      <div class="cb-graph-legend">
                        <span><span class="legend-dot" style="background:#6366f1"></span> Languages</span>
                        <span><span class="legend-dot" style="background:#0ea5e9"></span> Frameworks</span>
                        <span><span class="legend-dot" style="background:#f59e0b"></span> Tools</span>
                        <span><span class="legend-dot" style="background:#10b981"></span> Platforms</span>
                        <span>Solid border = tracked · Dashed = uncharted</span>
                      </div>
                    </div>
                  {/if}

                  {#if cbSubTab === 'stacks'}
                    <div class="cb-stacks-wrap" in:fade>
                      <div class="cb-stack-grid">
                        <div class="cb-stack-header">
                          <div class="cb-stack-project-col">Project</div>
                          {#each cbCategories as cat}
                            <div class="cb-stack-cat-col" style={`border-bottom: 3px solid ${categoryColor(cat)}`}>
                              {CATEGORY_LABELS[cat] ?? cat}
                            </div>
                          {/each}
                        </div>

                        {#each codebaseData.stacks as stack}
                          <div class="cb-stack-row">
                            <div class="cb-stack-project-col">
                              <strong>{stack.project_name}</strong>
                              <div class="chip-row">
                                {#each (stack.patterns ?? []) as pat}
                                  <span class="cb-mini-badge" title={pat}>{patternIcon(pat)}</span>
                                {/each}
                              </div>
                              <span class="cb-stack-count">{stack.tech_count} techs</span>
                            </div>
                            {#each cbCategories as cat}
                              <div class="cb-stack-cell">
                                {#each (stack.categories[cat] ?? []) as tech}
                                  <span class="cb-tech-chip" style={`border-left: 3px solid ${categoryColor(cat)}`}>{tech}</span>
                                {/each}
                                {#if !(stack.categories[cat]?.length)}
                                  <span class="cb-empty-cell">—</span>
                                {/if}
                              </div>
                            {/each}
                          </div>
                        {/each}
                      </div>
                    </div>
                  {/if}

                  {#if cbSubTab === 'files'}
                    <div class="cb-files-wrap" in:fade>
                      {#each codebaseData.file_tree as tree}
                        <div class="cb-file-project">
                          <button class="cb-file-project-header" onclick={() => toggleProjectExpand(tree.project_id)}>
                            <span class="cb-expand-icon">{expandedProjects.has(tree.project_id) ? '▾' : '▸'}</span>
                            <strong>{tree.project_name}</strong>
                            <span class="cb-file-count">{tree.files.length} manifest{tree.files.length !== 1 ? 's' : ''}</span>
                          </button>

                          {#if expandedProjects.has(tree.project_id)}
                            <div class="cb-file-list" in:fly={{ y: -8, duration: 150 }}>
                              {#each tree.files as file}
                                <div class="cb-file-node">
                                  <span class="cb-file-icon">📄</span>
                                  <span class="cb-file-path">{file.path}</span>
                                  <div class="cb-file-techs">
                                    {#each file.technologies as tech}
                                      <span class="cb-file-tech">{tech}</span>
                                    {/each}
                                  </div>
                                </div>
                              {/each}
                            </div>
                          {/if}
                        </div>
                      {/each}
                      {#if codebaseData.file_tree.length === 0}
                        <div class="empty-card">No file scan data available yet.</div>
                      {/if}
                    </div>
                  {/if}
                {/if}
              </article>
            </div>
          {/if}
        </section>
      </section>
    </div>
  {/if}

  {#if selectedEntry}
    <div
      class="modal-overlay"
      transition:fade
      role="button"
      tabindex="0"
      onclick={() => selectedEntry = null}
      onkeydown={(event) => {
        if (event.key === 'Escape' || event.key === 'Enter' || event.key === ' ') {
          event.preventDefault();
          selectedEntry = null;
        }
      }}
    >
      <div
        class="modal-card"
        transition:fly={{ y: 24, duration: 220 }}
        role="dialog"
        aria-modal="true"
        tabindex="0"
        onclick={(event) => event.stopPropagation()}
        onkeydown={(event) => {
          if (event.key === 'Escape') {
            event.preventDefault();
            selectedEntry = null;
          } else {
            event.stopPropagation();
          }
        }}
      >
        <button class="close-button" onclick={() => selectedEntry = null} aria-label="Close detail panel">×</button>

        <header class="modal-header">
          <div>
            <span class="eyebrow">{quadrantMeta[selectedEntry.quadrant]?.label ?? selectedEntry.quadrant}</span>
            <h2>{selectedEntry.name}</h2>
            <p>{selectedEntry.description}</p>
          </div>
          <div class="modal-actions">
            <span class="ring-chip" style={`--ring-color:${ringMeta[selectedEntry.ring]?.accent ?? '#1d4ed8'}`}>
              {ringMeta[selectedEntry.ring]?.label ?? selectedEntry.ring}
            </span>
            {#if isAdminMode}
              <button class="secondary-button" onclick={() => openEntryForm(selectedEntry)}>Edit</button>
            {/if}
          </div>
        </header>

        <section class="modal-grid">
          <article class="modal-panel">
            <h3>Adoption footprint</h3>
            {#if (selectedEntry.project_ids?.length ?? 0) > 0}
              <div class="chip-row">
                {#each selectedEntry.project_ids ?? [] as projectId}
                  <span>{projectId}</span>
                {/each}
              </div>
            {:else}
              <p class="muted">No active project scans are currently reporting this technology.</p>
            {/if}

            <div class="chip-row">
              <span>{movementLabel(selectedEntry)}</span>
              {#if selectedEntryMotion && selectedEntryMotion.moveCount > 0}
                <span>{motionLabel(selectedEntryMotion)}</span>
              {/if}
              {#each selectedEntry.tags ?? [] as tag}
                <span>#{tag}</span>
              {/each}
            </div>

            {#if selectedEntryMotion && selectedEntryMotion.moveCount > 0}
              <div class="motion-callout" data-tone={motionTone(selectedEntryMotion)}>
                <div class="motion-callout-header">
                  <span class={`signal-pill ${motionTone(selectedEntryMotion)}`}>{motionLabel(selectedEntryMotion)}</span>
                  <strong>{selectedEntryMotion.summary}</strong>
                </div>
                <div class="chip-row">
                  <span>{selectedEntryMotion.moveCount} checkpoints</span>
                  <span>{selectedEntryMotion.recentMoveCount} in 180 days</span>
                  <span>{selectedEntryMotion.streak}-step streak</span>
                </div>
              </div>
            {/if}

            {#if selectedEntryRecommendation}
              <div class="motion-callout recommendation-callout" data-tone={selectedEntryRecommendation.tone}>
                <div class="motion-callout-header">
                  <span class={`signal-pill ${selectedEntryRecommendation.tone}`}>{selectedEntryRecommendation.actionLabel}</span>
                  <strong>{selectedEntryRecommendation.title}</strong>
                </div>
                <p class="muted">{selectedEntryRecommendation.summary}</p>
              </div>
            {/if}

            {#if selectedEntry.link}
              {#if isADRReference(selectedEntry.link)}
                <div class="reference-block">
                  <span class="eyebrow">ADR reference</span>
                  <code>{selectedEntry.link}</code>
                </div>
              {:else}
                <a class="text-link" href={selectedEntry.link} target="_blank" rel="noreferrer">Open technical context</a>
              {/if}
            {/if}
          </article>

          <article class="modal-panel">
            <h3>Governance history</h3>
            {#if (selectedEntry.history?.length ?? 0) > 0}
              <div class="timeline-list">
                {#each selectedEntry.history ?? [] as item}
                  <div class="timeline-item">
                    <div class="timeline-dot" style={`--dot-color:${ringMeta[item.ring]?.accent ?? '#1d4ed8'}`}></div>
                    <div>
                      <strong>{ringMeta[item.ring]?.label ?? item.ring}</strong>
                      <p>{formatDate(item.date)}</p>
                      {#if item.reason}
                        <span>{item.reason}</span>
                      {/if}
                      {#if item.author}
                        <span>Author: {item.author}</span>
                      {/if}
                    </div>
                  </div>
                {/each}
              </div>
            {:else}
              <p class="muted">No history has been logged yet for this decision.</p>
            {/if}
          </article>
        </section>
      </div>
    </div>
  {/if}

  {#if showEntryForm}
    <div
      class="modal-overlay"
      transition:fade
      role="button"
      tabindex="0"
      onclick={() => showEntryForm = false}
      onkeydown={(event) => {
        if (event.key === 'Escape' || event.key === 'Enter' || event.key === ' ') {
          event.preventDefault();
          showEntryForm = false;
        }
      }}
    >
      <div
        class="modal-card form-card"
        transition:fly={{ y: 24, duration: 220 }}
        role="dialog"
        aria-modal="true"
        tabindex="0"
        onclick={(event) => event.stopPropagation()}
        onkeydown={(event) => {
          if (event.key === 'Escape') {
            event.preventDefault();
            showEntryForm = false;
          } else {
            event.stopPropagation();
          }
        }}
      >
        <button class="close-button" onclick={() => showEntryForm = false} aria-label="Close form">×</button>

        <header class="modal-header">
          <div>
            <span class="eyebrow">Admin editor</span>
            <h2>{editEntryData.id ? 'Update technology' : 'Add technology'}</h2>
            <p>Record the decision, explain why it matters, and keep the dashboard signal-rich.</p>
          </div>
        </header>

        <form class="form-grid" onsubmit={(event) => { event.preventDefault(); void submitEntry(); }}>
          <label>
            <span>Name</span>
            <input bind:value={editEntryData.name} required type="text" placeholder="React" disabled={Boolean(editEntryData.id)} />
          </label>

          <div class="split-grid">
            <label>
              <span>Quadrant</span>
              <select bind:value={editEntryData.quadrant}>
                {#each allQuadrants as quadrant}
                  <option value={quadrant}>{quadrantMeta[quadrant]?.label ?? quadrant}</option>
                {/each}
              </select>
            </label>

            <label>
              <span>Ring</span>
              <select bind:value={editEntryData.ring}>
                {#each ringOrder as ring}
                  <option value={ring}>{ringMeta[ring].label}</option>
                {/each}
              </select>
            </label>
          </div>

          <label>
            <span>Assessment</span>
            <textarea bind:value={editEntryData.description} rows="5" required placeholder="Why are we adopting, trialing, assessing, or holding this?"></textarea>
          </label>

          <label>
            <span>Decision rationale</span>
            <input bind:value={formReason} type="text" placeholder="Optional log entry for this update" />
          </label>

          <div class="split-grid">
            <label>
              <span>Tags</span>
              <input bind:value={formTagsInput} type="text" placeholder="frontend, platform, migration" />
            </label>

            <label>
              <span>Reference link or ADR path</span>
              <input bind:value={editEntryData.link} type="text" placeholder="https://... or docs/adrs/0001-example.md" />
            </label>
          </div>

          {#if formError}
            <div class="form-error">{formError}</div>
          {/if}

          <div class="form-actions">
            <button class="ghost-button" type="button" onclick={() => showEntryForm = false}>Cancel</button>
            <button class="primary-button" type="submit" disabled={formLoading}>
              {formLoading ? 'Saving...' : 'Save decision'}
            </button>
          </div>
        </form>
      </div>
    </div>
  {/if}
</main>

<style>
  :global(body) {
    --paper: #f6f1e8;
    --panel: rgba(255, 252, 247, 0.92);
    --panel-strong: #fff9f1;
    --ink: #18243a;
    --muted: #5d6778;
    --line: rgba(24, 36, 58, 0.12);
    --shadow: 0 24px 60px rgba(18, 29, 48, 0.12);
    --good: #0b8f5d;
    --watch: #c97a10;
    --danger: #b42318;
    --accent: #1d4ed8;
    margin: 0;
    background:
      radial-gradient(circle at top left, rgba(29, 78, 216, 0.12), transparent 32%),
      radial-gradient(circle at top right, rgba(201, 122, 16, 0.18), transparent 24%),
      linear-gradient(180deg, #f9f4ec 0%, #f4eee4 100%);
    color: var(--ink);
    font-family: "Space Grotesk", "Avenir Next", "Segoe UI", sans-serif;
  }

  :global(*) {
    box-sizing: border-box;
  }

  main {
    min-height: 100vh;
  }

  button,
  input,
  select,
  textarea {
    font: inherit;
  }

  button {
    cursor: pointer;
  }

  button:focus-visible,
  input:focus-visible,
  select:focus-visible,
  textarea:focus-visible {
    outline: 3px solid color-mix(in srgb, var(--accent) 30%, white);
    outline-offset: 2px;
  }

  .shell {
    min-height: 100vh;
  }

  .app-layout {
    display: grid;
    grid-template-columns: 290px minmax(0, 1fr);
    min-height: 100vh;
  }

  .sidebar {
    position: sticky;
    top: 0;
    height: 100vh;
    padding: 1.75rem;
    background:
      linear-gradient(180deg, rgba(19, 31, 53, 0.97), rgba(27, 41, 66, 0.96)),
      radial-gradient(circle at top left, rgba(14, 191, 165, 0.22), transparent 35%);
    color: #f9fbff;
    display: flex;
    flex-direction: column;
    gap: 1.25rem;
  }

  .brand-block,
  .sidebar-panel,
  .nav-stack {
    border: 1px solid rgba(255, 255, 255, 0.1);
    background: rgba(255, 255, 255, 0.05);
    backdrop-filter: blur(16px);
    border-radius: 22px;
  }

  .brand-block {
    padding: 1.2rem;
    display: flex;
    align-items: center;
    gap: 1rem;
  }

  .brand-mark {
    width: 54px;
    height: 54px;
    border-radius: 18px;
    display: grid;
    place-items: center;
    font-weight: 700;
    font-size: 1rem;
    letter-spacing: 0.12em;
    background: linear-gradient(135deg, #4cc7bb, #1d4ed8);
    color: #fff;
    box-shadow: 0 14px 26px rgba(19, 78, 216, 0.25);
  }

  .eyebrow {
    display: inline-block;
    font-size: 0.72rem;
    font-weight: 700;
    letter-spacing: 0.12em;
    text-transform: uppercase;
    color: inherit;
    opacity: 0.72;
  }

  .brand-block h2,
  .sidebar-panel h3,
  .panel-header h2,
  .hero h1,
  .state-card h2,
  .modal-header h2 {
    margin: 0;
  }

  .nav-stack {
    padding: 0.6rem;
    display: grid;
    gap: 0.3rem;
  }

  .nav-button {
    width: 100%;
    border: none;
    background: transparent;
    color: inherit;
    padding: 0.95rem 1rem;
    border-radius: 16px;
    display: flex;
    justify-content: space-between;
    align-items: center;
    font-weight: 600;
    transition: transform 160ms ease, background 160ms ease;
  }

  .nav-button:hover,
  .nav-button.active {
    background: rgba(255, 255, 255, 0.11);
    transform: translateX(3px);
  }

  .nav-button strong {
    min-width: 1.9rem;
    padding: 0.2rem 0.45rem;
    border-radius: 999px;
    background: rgba(255, 255, 255, 0.14);
    text-align: center;
    font-size: 0.74rem;
  }

  .sidebar-panel {
    padding: 1.2rem;
  }

  .sidebar-panel header {
    display: grid;
    gap: 0.25rem;
    margin-bottom: 0.9rem;
  }

  .mini-stats {
    display: grid;
    gap: 0.8rem;
    margin: 0;
  }

  .mini-stats div {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    font-size: 0.92rem;
  }

  .mini-stats dt {
    opacity: 0.72;
  }

  .mini-stats dd {
    margin: 0;
    font-weight: 700;
  }

  .toggle-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 1rem;
    margin-bottom: 0.9rem;
    font-weight: 600;
  }

  .toggle {
    position: relative;
    width: 52px;
    height: 30px;
  }

  .toggle input {
    position: absolute;
    inset: 0;
    opacity: 0;
  }

  .toggle span {
    display: block;
    width: 100%;
    height: 100%;
    background: rgba(255, 255, 255, 0.2);
    border-radius: 999px;
    transition: background 160ms ease;
  }

  .toggle span::after {
    content: "";
    position: absolute;
    top: 4px;
    left: 4px;
    width: 22px;
    height: 22px;
    border-radius: 50%;
    background: #fff;
    transition: transform 160ms ease;
  }

  .toggle input:checked + span {
    background: rgba(76, 199, 187, 0.7);
  }

  .toggle input:checked + span::after {
    transform: translateX(22px);
  }

  .sidebar-copy {
    margin: 0 0 1rem;
    font-size: 0.92rem;
    line-height: 1.5;
    opacity: 0.76;
  }

  .theme-card,
  .focus-card,
  .control-card {
    border: 1px solid var(--line);
    border-radius: 22px;
    background: rgba(255, 255, 255, 0.76);
    padding: 0.95rem 1rem 1rem;
    display: grid;
    gap: 0.75rem;
  }

  .theme-card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 0.8rem;
  }

  .theme-status-dot {
    width: 0.8rem;
    height: 0.8rem;
    border-radius: 999px;
    background: rgba(201, 122, 16, 0.75);
    box-shadow: 0 0 0 0.35rem rgba(201, 122, 16, 0.12);
    flex-shrink: 0;
  }

  .theme-status-dot.is-dark {
    background: rgba(29, 78, 216, 0.82);
    box-shadow: 0 0 0 0.35rem rgba(29, 78, 216, 0.14);
  }

  .theme-segmented {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 0.45rem;
  }

  .theme-segmented button {
    border: 1px solid var(--line);
    border-radius: 999px;
    background: rgba(24, 36, 58, 0.05);
    color: inherit;
    min-height: 2.9rem;
    padding: 0.8rem 1rem;
    font-size: 0.95rem;
    font-weight: 700;
    text-transform: capitalize;
    transition: transform 160ms ease, background 160ms ease, border-color 160ms ease;
  }

  .theme-segmented button:hover {
    transform: translateY(-1px);
    border-color: rgba(29, 78, 216, 0.24);
  }

  .theme-segmented button.active {
    background: linear-gradient(135deg, rgba(29, 78, 216, 0.16), rgba(76, 199, 187, 0.22));
    border-color: rgba(29, 78, 216, 0.32);
  }

  .theme-copy {
    margin: 0;
    color: var(--muted);
    line-height: 1.5;
    font-size: 0.92rem;
  }

  .focus-card {
    background:
      linear-gradient(180deg, rgba(255, 255, 255, 0.9), rgba(247, 243, 236, 0.82)),
      radial-gradient(circle at top right, rgba(76, 199, 187, 0.14), transparent 36%);
  }

  .control-card {
    background:
      linear-gradient(180deg, rgba(255, 255, 255, 0.84), rgba(249, 245, 238, 0.84)),
      radial-gradient(circle at top right, rgba(29, 78, 216, 0.08), transparent 30%);
  }

  .control-bar {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    gap: 1rem;
    flex-wrap: wrap;
  }

  .control-copy {
    display: grid;
    gap: 0.35rem;
    max-width: 26rem;
  }

  .control-copy strong,
  .theme-inline-copy strong {
    display: block;
    font-size: 1rem;
    line-height: 1.3;
  }

  .control-copy p {
    margin: 0;
    color: var(--muted);
    font-size: 0.92rem;
    line-height: 1.5;
  }

  .theme-inline {
    display: grid;
    gap: 0.65rem;
    min-width: min(100%, 22rem);
  }

  .theme-inline-copy {
    display: flex;
    justify-content: space-between;
    align-items: baseline;
    gap: 0.9rem;
  }

  .compact-card {
    border-radius: 18px;
  }

  .focus-card-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    gap: 0.8rem;
  }

  .focus-card-header strong {
    display: block;
    font-size: 1.15rem;
    line-height: 1.2;
    margin-top: 0.25rem;
  }

  .focus-copy {
    margin: 0;
    color: var(--muted);
    line-height: 1.6;
  }

  .focus-metrics,
  .focus-actions {
    display: flex;
    flex-wrap: wrap;
    gap: 0.65rem;
  }

  .focus-metrics span {
    padding: 0.5rem 0.75rem;
    border-radius: 999px;
    border: 1px solid var(--line);
    background: rgba(255, 255, 255, 0.78);
    font-size: 0.84rem;
    font-weight: 600;
    color: var(--ink);
  }

  .focus-actions .ghost-button,
  .focus-actions .primary-button {
    min-height: 2.75rem;
  }

  .workspace {
    min-width: 0;
    padding: 1.5rem 1.5rem 2rem;
  }

  .hero {
    display: grid;
    grid-template-columns: minmax(0, 1.4fr) minmax(320px, 1fr);
    gap: 1rem;
    padding: 1.35rem 1.45rem;
    border: 1px solid var(--line);
    border-radius: 28px;
    background:
      linear-gradient(135deg, rgba(255, 255, 255, 0.82), rgba(255, 247, 236, 0.78)),
      radial-gradient(circle at top right, rgba(76, 199, 187, 0.14), transparent 32%);
    box-shadow: var(--shadow);
  }

  .hero h1 {
    font-size: clamp(2rem, 3vw, 3.5rem);
    letter-spacing: -0.04em;
    margin-top: 0.3rem;
  }

  .hero-copy p {
    max-width: 62ch;
    margin: 0.8rem 0 0;
    color: var(--muted);
    line-height: 1.6;
  }

  .hero-notes {
    display: flex;
    flex-wrap: wrap;
    gap: 0.7rem;
    margin-top: 1rem;
  }

  .hero-notes span {
    padding: 0.45rem 0.7rem;
    border-radius: 999px;
    border: 1px solid var(--line);
    background: rgba(255, 255, 255, 0.75);
    font-size: 0.88rem;
  }

  .hero-actions {
    display: grid;
    gap: 0.95rem;
    align-content: start;
  }

  .search-card,
  .toolbar {
    border: 1px solid var(--line);
    border-radius: 22px;
    background: rgba(255, 255, 255, 0.76);
  }

  .search-card {
    padding: 0.95rem 1rem 1rem;
    display: grid;
    gap: 0.7rem;
  }

  .search-head {
    display: flex;
    justify-content: space-between;
    gap: 0.75rem;
    align-items: baseline;
    flex-wrap: wrap;
  }

  .search-meta {
    color: var(--muted);
    font-size: 0.82rem;
    font-weight: 600;
  }

  .search-shell {
    display: flex;
    align-items: center;
    gap: 0.6rem;
    padding: 0.35rem 0.45rem 0.35rem 0.9rem;
    border-radius: 16px;
    background: rgba(24, 36, 58, 0.05);
  }

  .search-shell input {
    flex: 1;
    border: none;
    background: transparent;
    outline: none;
    color: var(--ink);
  }

  .search-shell kbd {
    border-radius: 10px;
    border: 1px solid rgba(24, 36, 58, 0.1);
    padding: 0.25rem 0.5rem;
    background: rgba(255, 255, 255, 0.8);
    font-size: 0.8rem;
  }

  .toolbar {
    padding: 0.8rem;
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr)) repeat(2, auto);
    gap: 0.75rem;
    align-items: end;
  }

  .toolbar label,
  .form-grid label {
    display: grid;
    gap: 0.45rem;
    color: var(--muted);
    font-size: 0.9rem;
  }

  .toolbar span,
  .form-grid span {
    font-weight: 600;
    color: var(--ink);
  }

  .toolbar select,
  .form-grid select,
  .form-grid input,
  .form-grid textarea {
    width: 100%;
    border: 1px solid var(--line);
    border-radius: 14px;
    background: #fffefb;
    padding: 0.75rem 0.9rem;
    color: var(--ink);
    outline: none;
  }

  .form-grid textarea {
    resize: vertical;
    min-height: 7.5rem;
  }

  .metric-grid {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 1rem;
    margin-top: 1rem;
  }

  .filter-summary {
    margin-top: 1rem;
    padding: 0.95rem 1rem;
    border: 1px solid var(--line);
    border-radius: 22px;
    background: rgba(255, 255, 255, 0.68);
    box-shadow: var(--shadow);
    display: grid;
    grid-template-columns: auto minmax(0, 1fr) auto;
    gap: 0.85rem;
    align-items: center;
  }

  .filter-summary-copy {
    display: grid;
    gap: 0.15rem;
  }

  .filter-summary-copy strong {
    font-size: 1rem;
    line-height: 1.3;
  }

  .filter-chip-row {
    display: flex;
    flex-wrap: wrap;
    gap: 0.55rem;
  }

  .filter-chip {
    display: inline-flex;
    align-items: center;
    padding: 0.48rem 0.78rem;
    border-radius: 999px;
    background: rgba(24, 36, 58, 0.06);
    border: 1px solid rgba(24, 36, 58, 0.08);
    color: var(--ink);
    font-size: 0.84rem;
    font-weight: 600;
  }

  .filter-chip.quiet {
    color: var(--muted);
    background: rgba(255, 255, 255, 0.74);
  }

  .metric-card {
    padding: 1.1rem 1.2rem;
    border-radius: 22px;
    border: 1px solid var(--line);
    background: var(--panel);
    box-shadow: var(--shadow);
    display: grid;
    gap: 0.45rem;
  }

  .metric-card strong {
    font-size: 1.75rem;
    letter-spacing: -0.04em;
  }

  .metric-card p {
    margin: 0;
    color: var(--muted);
    line-height: 1.5;
  }

  .metric-card[data-tone="good"] strong {
    color: var(--good);
  }

  .metric-card[data-tone="watch"] strong {
    color: var(--watch);
  }

  .metric-card[data-tone="danger"] strong {
    color: var(--danger);
  }

  .content-scroll {
    margin-top: 1rem;
    padding-bottom: 2rem;
  }

  .page-grid {
    display: grid;
    grid-template-columns: minmax(0, 1.25fr) minmax(320px, 0.95fr);
    gap: 1rem;
    align-items: start;
  }

  .overview-grid {
    grid-template-columns: minmax(0, 1.2fr) minmax(320px, 0.92fr);
  }

  .radar-page {
    grid-template-columns: minmax(0, 1.45fr) minmax(320px, 0.8fr);
  }

  .panel,
  .project-card,
  .drift-card {
    border: 1px solid var(--line);
    border-radius: 28px;
    background: var(--panel);
    box-shadow: var(--shadow);
  }

  .panel {
    padding: 1.35rem;
  }

  .panel-large {
    grid-column: span 1;
  }

  .panel-wide {
    grid-column: 1 / -1;
  }

  .panel-header {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    align-items: flex-start;
    margin-bottom: 1rem;
  }

  .panel-header p {
    margin: 0;
    color: var(--muted);
    max-width: 28ch;
    line-height: 1.5;
  }

  .overview-rail-sections,
  .project-spotlight-list {
    display: grid;
    gap: 1rem;
  }

  .overview-subsection {
    display: grid;
    gap: 0.8rem;
  }

  .overview-subsection + .overview-subsection {
    padding-top: 1rem;
    border-top: 1px solid var(--line);
  }

  .overview-subsection-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    gap: 0.75rem;
  }

  .overview-subsection-header h3 {
    margin: 0;
    font-size: 1rem;
  }

  .queue-grid,
  .breakdown-stack,
  .signal-stack,
  .timeline-list,
  .entry-list {
    display: grid;
    gap: 0.85rem;
  }

  .queue-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 1rem;
  }

  .queue-column {
    display: grid;
    gap: 0.7rem;
  }

  .overview-subsection h3,
  .ring-section h3,
  .modal-panel h3 {
    margin: 0;
    font-size: 1rem;
  }

  .signal-row,
  .entry-card {
    width: 100%;
    border: 1px solid var(--line);
    background: rgba(255, 255, 255, 0.72);
    padding: 0.95rem 1rem;
    border-radius: 18px;
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 1rem;
    text-align: left;
  }

  .signal-row strong,
  .entry-card strong,
  .drift-card h3,
  .project-card h3 {
    font-size: 1rem;
  }

  .signal-row span,
  .entry-card p,
  .project-card p,
  .timeline-item span,
  .muted {
    color: var(--muted);
    line-height: 1.45;
  }

  .actionable {
    transition: transform 160ms ease, border-color 160ms ease, background 160ms ease;
  }

  .actionable:hover {
    transform: translateY(-1px);
    border-color: rgba(29, 78, 216, 0.24);
    background: rgba(255, 255, 255, 0.96);
  }

  .signal-pill,
  .risk-badge,
  .ring-chip {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-width: 2.5rem;
    padding: 0.35rem 0.7rem;
    border-radius: 999px;
    font-size: 0.8rem;
    font-weight: 700;
  }

  .signal-pill.good,
  .risk-badge.healthy {
    background: rgba(11, 143, 93, 0.12);
    color: var(--good);
  }

  .signal-pill.neutral {
    background: rgba(29, 78, 216, 0.1);
    color: var(--accent);
  }

  .signal-pill.watch,
  .risk-badge.watch {
    background: rgba(201, 122, 16, 0.14);
    color: var(--watch);
  }

  .signal-pill.danger,
  .risk-badge.critical {
    background: rgba(180, 35, 24, 0.12);
    color: var(--danger);
  }

  .subtle-button {
    min-height: auto;
    padding: 0.45rem 0.8rem;
    border-radius: 999px;
    font-size: 0.82rem;
  }

  .breakdown-stack {
    gap: 1.15rem;
  }

  .breakdown-stack.compact {
    gap: 1rem;
  }

  .bar-row {
    display: grid;
    grid-template-columns: 120px minmax(0, 1fr) auto;
    gap: 0.75rem;
    align-items: center;
    margin-top: 0.7rem;
    font-size: 0.92rem;
  }

  .bar-track {
    height: 0.6rem;
    border-radius: 999px;
    background: rgba(24, 36, 58, 0.08);
    overflow: hidden;
  }

  .bar-fill {
    height: 100%;
    border-radius: inherit;
    background: var(--bar-color, var(--accent));
  }

  .bar-fill.neutral {
    background: linear-gradient(90deg, rgba(29, 78, 216, 0.75), rgba(76, 199, 187, 0.9));
  }

  .project-grid,
  .drift-grid,
  .ring-section-grid {
    display: grid;
    gap: 1rem;
  }

  .project-grid {
    grid-template-columns: repeat(auto-fit, minmax(270px, 1fr));
  }

  .project-grid-wide {
    grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  }

  .project-card,
  .drift-card {
    padding: 1.15rem;
    text-align: left;
  }

  .project-row {
    width: 100%;
    border: 1px solid var(--line);
    border-radius: 22px;
    background: rgba(255, 255, 255, 0.68);
    padding: 1rem 1.1rem;
    text-align: left;
    display: grid;
    gap: 0.85rem;
  }

  .project-card {
    display: grid;
    gap: 0.85rem;
  }

  .project-card[data-risk="critical"] {
    border-color: rgba(180, 35, 24, 0.24);
  }

  .project-card[data-risk="watch"] {
    border-color: rgba(201, 122, 16, 0.22);
  }

  .project-row[data-risk="critical"] {
    border-color: rgba(180, 35, 24, 0.24);
  }

  .project-row[data-risk="watch"] {
    border-color: rgba(201, 122, 16, 0.22);
  }

  .project-card-header,
  .drift-card-header,
  .modal-header,
  .entry-card-top,
  .ring-headline,
  .form-actions,
  .score-row {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    align-items: flex-start;
  }

  .score-row strong {
    font-size: 2rem;
    letter-spacing: -0.05em;
  }

  .score-row span {
    color: var(--muted);
    margin-top: 0.45rem;
  }

  .project-row-head,
  .project-row-summary {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    gap: 1rem;
  }

  .project-row h3 {
    margin: 0;
    font-size: 1rem;
  }

  .project-row p {
    margin: 0.25rem 0 0;
    color: var(--muted);
    line-height: 1.45;
  }

  .project-row-metrics {
    display: flex;
    flex-wrap: wrap;
    justify-content: flex-end;
    gap: 0.55rem 0.9rem;
    color: var(--muted);
    font-size: 0.92rem;
  }

  .score-row.compact {
    align-items: baseline;
    gap: 0.65rem;
  }

  .score-row.compact strong {
    font-size: 1.65rem;
  }

  .score-row.compact span {
    margin-top: 0;
  }

  .meter {
    height: 0.72rem;
    border-radius: 999px;
    background: rgba(24, 36, 58, 0.08);
    overflow: hidden;
  }

  .meter-fill {
    height: 100%;
    background: linear-gradient(90deg, #1d4ed8 0%, #4cc7bb 100%);
  }

  .project-metrics {
    display: flex;
    flex-wrap: wrap;
    gap: 0.55rem 0.9rem;
    color: var(--muted);
    font-size: 0.92rem;
  }

  .chip-row,
  .chip-cloud {
    display: flex;
    flex-wrap: wrap;
    gap: 0.55rem;
  }

  .chip-row span,
  .tag-chip {
    display: inline-flex;
    align-items: center;
    gap: 0.35rem;
    padding: 0.45rem 0.7rem;
    border-radius: 999px;
    border: 1px solid var(--line);
    background: rgba(255, 255, 255, 0.78);
    color: var(--ink);
    font-size: 0.84rem;
  }

  .tag-chip.actionable {
    border: 1px solid rgba(29, 78, 216, 0.15);
  }

  .timeline-item {
    display: grid;
    grid-template-columns: auto minmax(0, 1fr);
    gap: 0.85rem;
    align-items: flex-start;
    padding: 0.3rem 0;
  }

  .timeline-dot {
    width: 0.85rem;
    height: 0.85rem;
    border-radius: 50%;
    margin-top: 0.4rem;
    background: var(--dot-color, var(--accent));
    box-shadow: 0 0 0 8px color-mix(in srgb, var(--dot-color, var(--accent)) 18%, transparent);
  }

  .timeline-item p {
    margin: 0.15rem 0;
    color: var(--ink);
    font-size: 0.95rem;
  }

  .empty-card,
  .empty-inline {
    padding: 1rem;
    border-radius: 18px;
    border: 1px dashed rgba(24, 36, 58, 0.18);
    color: var(--muted);
    background: rgba(255, 255, 255, 0.56);
  }

  .radar-panel {
    padding-bottom: 1rem;
  }

  .radar-canvas {
    border-radius: 24px;
    background:
      radial-gradient(circle at center, rgba(76, 199, 187, 0.08), transparent 30%),
      linear-gradient(180deg, rgba(255, 255, 255, 0.9), rgba(247, 241, 232, 0.92));
    padding: 1rem;
    border: 1px solid var(--line);
  }

  .radar-canvas svg {
    width: 100%;
    display: block;
  }

  .ring-circle {
    fill: color-mix(in srgb, var(--ring-color), white 90%);
    stroke: color-mix(in srgb, var(--ring-color), #fff 20%);
    stroke-width: 1;
  }

  .divider-line {
    stroke: rgba(24, 36, 58, 0.15);
    stroke-dasharray: 4 6;
  }

  .quadrant-label {
    font-size: 12px;
    letter-spacing: 0.18em;
    font-weight: 700;
    fill: rgba(24, 36, 58, 0.56);
  }

  .blip {
    cursor: pointer;
  }

  .blip path,
  .blip circle,
  .blip rect {
    transition: transform 150ms ease, filter 150ms ease;
  }

  .blip.hovered .blip-core {
    filter: drop-shadow(0 10px 10px rgba(29, 78, 216, 0.26));
    transform: scale(1.12);
  }

  .motion-halo,
  .motion-frame {
    fill: none;
    stroke-width: 2;
    pointer-events: none;
  }

  .motion-halo.accelerating,
  .motion-halo.heating {
    stroke: rgba(11, 143, 93, 0.78);
    filter: drop-shadow(0 0 12px rgba(11, 143, 93, 0.3));
  }

  .motion-halo.cooling {
    stroke: rgba(180, 35, 24, 0.7);
    stroke-dasharray: 4 5;
  }

  .motion-frame {
    stroke: rgba(201, 122, 16, 0.78);
    stroke-dasharray: 3 3;
  }

  .project-count-dot {
    fill: #18243a;
  }

  .project-count-text {
    fill: #fff;
    font-size: 8px;
    font-weight: 700;
  }

  .blip-label {
    font-size: 10px;
    font-weight: 700;
    fill: rgba(24, 36, 58, 0);
    pointer-events: none;
    transition: fill 150ms ease;
  }

  .blip.hovered .blip-label {
    fill: rgba(24, 36, 58, 0.82);
  }

  .legend-stack,
  .offcanvas-block {
    display: grid;
    gap: 0.95rem;
  }

  .legend-row {
    display: grid;
    grid-template-columns: auto minmax(0, 1fr);
    gap: 0.8rem;
    align-items: flex-start;
  }

  .legend-swatch {
    width: 0.9rem;
    height: 0.9rem;
    border-radius: 50%;
    background: var(--swatch, var(--accent));
    margin-top: 0.35rem;
  }

  .legend-row p,
  .offcanvas-block p,
  .ring-section p,
  .drift-card p {
    margin: 0.3rem 0 0;
    color: var(--muted);
    line-height: 1.5;
  }

  .entry-note,
  .legend-note {
    margin: 0;
    color: var(--muted);
    line-height: 1.55;
    font-size: 0.9rem;
  }

  .ring-section-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .ring-section {
    display: grid;
    gap: 0.9rem;
    border: 1px solid var(--line);
    border-radius: 22px;
    padding: 1rem;
    background: rgba(255, 255, 255, 0.54);
  }

  .entry-card {
    display: grid;
    gap: 0.7rem;
  }

  .entry-meta {
    color: var(--muted);
    font-size: 0.84rem;
  }

  .drift-grid {
    grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  }

  .drift-card {
    border-color: rgba(180, 35, 24, 0.2);
    display: grid;
    gap: 0.9rem;
  }

  .state-screen {
    min-height: 100vh;
    display: grid;
    place-items: center;
    padding: 2rem;
  }

  .state-card {
    max-width: 560px;
    padding: 2rem;
    border-radius: 28px;
    border: 1px solid var(--line);
    background: var(--panel-strong);
    box-shadow: var(--shadow);
    display: grid;
    gap: 0.9rem;
    text-align: center;
  }

  .state-card p {
    margin: 0;
    color: var(--muted);
    line-height: 1.6;
  }

  .loading-card {
    justify-items: center;
  }

  .signal-loader {
    width: 64px;
    height: 64px;
    border-radius: 50%;
    border: 4px solid rgba(29, 78, 216, 0.12);
    border-top-color: var(--accent);
    animation: spin 1s linear infinite;
  }

  .modal-overlay {
    position: fixed;
    inset: 0;
    background: rgba(18, 29, 48, 0.45);
    backdrop-filter: blur(12px);
    display: grid;
    place-items: center;
    padding: 1.5rem;
    z-index: 10;
  }

  .modal-card {
    position: relative;
    width: min(1040px, 100%);
    max-height: min(90vh, 960px);
    overflow: auto;
    padding: 1.4rem;
    border-radius: 30px;
    border: 1px solid var(--line);
    background: var(--panel-strong);
    box-shadow: 0 40px 90px rgba(18, 29, 48, 0.24);
  }

  .form-card {
    width: min(780px, 100%);
  }

  .close-button {
    position: absolute;
    top: 1rem;
    right: 1rem;
    width: 2.5rem;
    height: 2.5rem;
    border: none;
    border-radius: 50%;
    background: rgba(24, 36, 58, 0.06);
    color: var(--ink);
    font-size: 1.5rem;
  }

  .modal-header {
    margin-bottom: 1rem;
    padding-right: 3rem;
  }

  .modal-header p {
    margin: 0.55rem 0 0;
    color: var(--muted);
    line-height: 1.6;
  }

  .modal-actions {
    display: flex;
    gap: 0.7rem;
    align-items: center;
  }

  .ring-chip {
    background: color-mix(in srgb, var(--ring-color), white 88%);
    color: var(--ring-color);
  }

  .modal-grid,
  .form-grid {
    display: grid;
    gap: 1rem;
  }

  .modal-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .modal-panel {
    padding: 1rem;
    border-radius: 22px;
    border: 1px solid var(--line);
    background: rgba(255, 255, 255, 0.64);
  }

  .motion-callout {
    margin-top: 1rem;
    padding: 1rem;
    border-radius: 18px;
    border: 1px solid var(--line);
    background: rgba(255, 255, 255, 0.72);
    display: grid;
    gap: 0.85rem;
  }

  .motion-callout[data-tone="good"] {
    border-color: rgba(11, 143, 93, 0.2);
    background: rgba(11, 143, 93, 0.06);
  }

  .motion-callout[data-tone="danger"] {
    border-color: rgba(180, 35, 24, 0.18);
    background: rgba(180, 35, 24, 0.06);
  }

  .motion-callout[data-tone="watch"] {
    border-color: rgba(201, 122, 16, 0.22);
    background: rgba(201, 122, 16, 0.08);
  }

  .motion-callout-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 0.85rem;
  }

  .text-link {
    color: var(--accent);
    font-weight: 600;
    text-decoration: none;
  }

  .split-grid {
    display: grid;
    gap: 1rem;
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .recommendation-grid section,
  .motion-split section {
    display: grid;
    gap: 0.8rem;
  }

  .recommendation-callout {
    border-style: solid;
  }

  .form-error {
    padding: 0.85rem 0.95rem;
    border-radius: 16px;
    background: rgba(180, 35, 24, 0.1);
    color: var(--danger);
    border: 1px solid rgba(180, 35, 24, 0.15);
  }

  .primary-button,
  .secondary-button,
  .ghost-button {
    border-radius: 14px;
    padding: 0.75rem 1rem;
    border: 1px solid transparent;
    font-weight: 600;
  }

  .primary-button {
    background: linear-gradient(135deg, #1d4ed8, #4cc7bb);
    color: #fff;
  }

  .secondary-button {
    background: rgba(29, 78, 216, 0.08);
    border-color: rgba(29, 78, 216, 0.12);
    color: var(--accent);
  }

  .ghost-button {
    background: rgba(255, 255, 255, 0.72);
    border-color: var(--line);
    color: var(--ink);
  }

  .ghost-button:disabled,
  .primary-button:disabled {
    opacity: 0.55;
    cursor: not-allowed;
  }

  .wide-button {
    width: 100%;
  }

  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
  }

  @media (max-width: 1180px) {
    .app-layout {
      grid-template-columns: 1fr;
    }

    .sidebar {
      position: static;
      height: auto;
      display: grid;
      grid-template-columns: repeat(2, minmax(0, 1fr));
      align-items: start;
    }

    .hero,
    .page-grid,
    .radar-page {
      grid-template-columns: 1fr;
    }

    .metric-grid {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }

    .brand-block,
    .nav-stack {
      grid-column: 1 / -1;
    }

    .filter-summary {
      grid-template-columns: 1fr;
      align-items: start;
    }

    .ring-section-grid,
    .queue-grid,
    .modal-grid {
      grid-template-columns: 1fr;
    }
  }

  @media (max-width: 760px) {
    .workspace {
      padding: 1rem;
    }

    .sidebar {
      grid-template-columns: 1fr;
      padding: 1rem;
      border-bottom: 1px solid rgba(255, 255, 255, 0.08);
    }

    .hero,
    .panel,
    .project-card,
    .drift-card,
    .modal-card,
    .state-card {
      border-radius: 24px;
    }

    .toolbar,
    .metric-grid {
      grid-template-columns: 1fr;
    }

    .nav-stack {
      display: flex;
      overflow-x: auto;
      padding: 0.5rem;
      gap: 0.45rem;
      scroll-snap-type: x proximity;
    }

    .nav-button {
      min-width: max-content;
      scroll-snap-align: start;
    }

    .search-head,
    .focus-card-header,
    .filter-summary-copy,
    .control-bar,
    .theme-inline-copy,
    .project-row-head,
    .project-row-summary {
      align-items: flex-start;
    }

    .panel-header,
    .project-card-header,
    .drift-card-header,
    .modal-header,
    .form-actions,
    .score-row,
    .entry-card-top {
      flex-direction: column;
    }

    .project-row-metrics {
      justify-content: flex-start;
    }

    .split-grid {
      grid-template-columns: 1fr;
    }

    .bar-row {
      grid-template-columns: 1fr;
    }
  }

  /* ========= Codebase Visualizer ========= */
  .cb-summary-pills {
    display: flex; gap: 8px; flex-wrap: wrap; align-items: center;
  }
  .cb-pill {
    font-size: 0.72rem; font-weight: 600;
    padding: 3px 10px; border-radius: 20px;
    background: var(--ink-faint, rgba(92, 71, 50, 0.06));
    color: var(--ink-secondary, #6b5e50);
    white-space: nowrap;
  }
  .cb-pill.small { font-size: 0.65rem; padding: 2px 7px; }
  .cb-sub-tabs {
    display: flex; gap: 0; border-bottom: 1.5px solid var(--rule, rgba(92, 71, 50, 0.1));
    margin-bottom: 20px; margin-top: 4px;
  }
  .cb-sub-tabs button {
    font-size: 0.82rem; font-weight: 600;
    padding: 8px 18px; background: none; border: none;
    color: var(--ink-secondary, #6b5e50);
    cursor: pointer; position: relative;
    transition: color 0.2s;
  }
  .cb-sub-tabs button.active {
    color: var(--ink, #3a2f24);
  }
  .cb-sub-tabs button.active::after {
    content: ''; position: absolute;
    bottom: -1.5px; left: 8px; right: 8px;
    height: 2.5px; background: var(--accent, #1d4ed8);
    border-radius: 2px;
  }
  .cb-sub-tabs button:hover { color: var(--ink, #3a2f24); }

  .cb-graph-wrap { display: flex; flex-direction: column; gap: 12px; }
  .cb-graph-toolbar {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    align-items: flex-start;
    flex-wrap: wrap;
  }
  .cb-graph-summary {
    display: grid;
    gap: 0.3rem;
  }
  .cb-graph-summary strong {
    font-size: 0.98rem;
  }
  .cb-graph-summary span {
    color: var(--muted);
    font-size: 0.9rem;
    line-height: 1.5;
  }
  .cb-graph-mode {
    display: inline-grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 0.45rem;
  }
  .cb-graph-mode button {
    min-width: 6.5rem;
    min-height: 2.7rem;
    border-radius: 999px;
    border: 1px solid var(--line);
    background: rgba(255, 255, 255, 0.7);
    color: var(--ink);
    padding: 0.7rem 1rem;
    font-size: 0.9rem;
    font-weight: 700;
    transition: transform 160ms ease, background 160ms ease, border-color 160ms ease;
  }
  .cb-graph-mode button:hover {
    transform: translateY(-1px);
    border-color: rgba(29, 78, 216, 0.22);
  }
  .cb-graph-mode button.active {
    background: linear-gradient(135deg, rgba(29, 78, 216, 0.14), rgba(76, 199, 187, 0.2));
    border-color: rgba(29, 78, 216, 0.28);
  }
  .cb-graph-canvas {
    background: var(--panel, rgba(255, 252, 247, 0.92));
    border-radius: 12px; border: 1px solid var(--rule, rgba(92, 71, 50, 0.08));
    overflow: hidden;
    min-height: 560px;
  }
  .dep-graph { width: 100%; height: auto; display: block; cursor: grab; }
  .dep-graph:active { cursor: grabbing; }
  .graph-edge { stroke: var(--ink-faint, #c4b9a8); stroke-linecap: round; }
  .graph-edge.highlighted { stroke: var(--accent, #1d4ed8); }
  .graph-node { cursor: pointer; }
  .graph-node:hover circle, .graph-node.hovered circle { filter: brightness(1.15); }
  .graph-node.selected circle { stroke-width: 3 !important; }
  .graph-label {
    fill: color-mix(in srgb, var(--ink, #3a2f24) 84%, transparent); font-weight: 600; pointer-events: none;
    text-shadow: 0 1px 2px rgba(255,255,255,0.92);
  }

  .cb-node-detail {
    background: var(--panel, rgba(255, 252, 247, 0.92));
    border: 1px solid var(--rule, rgba(92, 71, 50, 0.1));
    border-radius: 10px; padding: 14px 18px;
    font-size: 0.82rem;
  }
  .cb-node-detail-header {
    display: flex; gap: 8px; align-items: center;
    flex-wrap: wrap; margin-bottom: 6px;
  }
  .cb-node-dot {
    width: 10px; height: 10px; border-radius: 50%;
    display: inline-block; flex-shrink: 0;
  }
  .cb-node-detail p { margin: 0 0 8px; color: var(--ink-secondary, #6b5e50); }
  .cb-connections { display: flex; gap: 6px; flex-wrap: wrap; }
  .cb-connection-chip {
    font-size: 0.72rem; padding: 2px 8px;
    background: var(--ink-faint, rgba(92, 71, 50, 0.06));
    border-radius: 6px; color: var(--ink-secondary, #6b5e50);
  }
  .cb-connection-chip strong { color: var(--ink, #3a2f24); margin-left: 2px; }

  .cb-pattern-strip { display: flex; gap: 8px; flex-wrap: wrap; }
  .cb-pattern-badge {
    display: inline-flex; gap: 5px; align-items: center;
    font-size: 0.75rem; padding: 4px 12px;
    background: var(--panel, rgba(255, 252, 247, 0.92));
    border: 1px solid var(--rule, rgba(92, 71, 50, 0.1));
    border-radius: 20px; cursor: help;
    transition: box-shadow 0.2s;
  }
  .cb-pattern-badge:hover { box-shadow: 0 2px 8px rgba(0,0,0,0.08); }
  .cb-pattern-badge strong {
    background: var(--accent, #1d4ed8); color: white;
    border-radius: 50%; width: 18px; height: 18px;
    display: inline-flex; align-items: center; justify-content: center;
    font-size: 0.65rem;
  }

  .cb-graph-legend {
    display: flex; gap: 14px; flex-wrap: wrap;
    font-size: 0.72rem; color: var(--ink-secondary, #6b5e50);
    align-items: center;
  }
  .legend-dot {
    display: inline-block; width: 10px; height: 10px;
    border-radius: 50%; margin-right: 4px; vertical-align: middle;
  }

  .cb-stacks-wrap { overflow-x: auto; }
  .cb-stack-grid { min-width: 600px; }
  .cb-stack-header {
    display: grid;
    grid-template-columns: 180px repeat(auto-fill, minmax(130px, 1fr));
    gap: 1px; font-size: 0.75rem; font-weight: 700;
    color: var(--ink-secondary, #6b5e50);
    padding-bottom: 8px; border-bottom: 1px solid var(--rule, rgba(92, 71, 50, 0.1));
  }
  .cb-stack-header .cb-stack-cat-col { text-align: center; padding-bottom: 6px; }
  .cb-stack-row {
    display: grid;
    grid-template-columns: 180px repeat(auto-fill, minmax(130px, 1fr));
    gap: 1px; padding: 10px 0;
    border-bottom: 1px solid var(--rule, rgba(92, 71, 50, 0.05));
    transition: background 0.15s;
  }
  .cb-stack-row:hover { background: rgba(92, 71, 50, 0.02); }
  .cb-stack-project-col {
    display: flex; flex-direction: column; gap: 2px; font-size: 0.82rem;
  }
  .cb-stack-count { font-size: 0.7rem; color: var(--ink-secondary, #6b5e50); }
  .cb-stack-cell {
    display: flex; flex-wrap: wrap; gap: 4px; align-items: flex-start;
    padding: 2px 4px;
  }
  .cb-tech-chip {
    font-size: 0.7rem; padding: 2px 6px;
    background: var(--ink-faint, rgba(92, 71, 50, 0.04));
    border-radius: 4px; white-space: nowrap;
  }
  .cb-empty-cell { color: var(--ink-faint, #c4b9a8); font-size: 0.7rem; }
  .cb-mini-badge { font-size: 0.9rem; cursor: help; }

  .cb-files-wrap { display: flex; flex-direction: column; gap: 4px; }
  .cb-file-project {
    border: 1px solid var(--rule, rgba(92, 71, 50, 0.08));
    border-radius: 8px; overflow: hidden;
  }
  .cb-file-project-header {
    display: flex; gap: 10px; align-items: center;
    width: 100%; padding: 10px 14px;
    background: var(--ink-faint, rgba(92, 71, 50, 0.03));
    border: none; cursor: pointer; font-size: 0.85rem;
    text-align: left; transition: background 0.15s;
  }
  .cb-file-project-header:hover { background: rgba(92, 71, 50, 0.06); }
  .cb-expand-icon { font-size: 0.8rem; width: 12px; opacity: 0.5; }
  .cb-file-count {
    margin-left: auto; font-size: 0.72rem;
    color: var(--ink-secondary, #6b5e50);
  }
  .cb-file-list { padding: 4px 14px 10px 36px; }
  .cb-file-node {
    display: flex; gap: 8px; align-items: flex-start;
    padding: 5px 0; border-bottom: 1px solid var(--rule, rgba(92, 71, 50, 0.04));
    flex-wrap: wrap;
  }
  .cb-file-node:last-child { border-bottom: none; }
  .cb-file-icon { font-size: 0.9rem; flex-shrink: 0; }
  .cb-file-path {
    font-family: monospace; font-size: 0.78rem;
    font-weight: 600; flex-shrink: 0; min-width: 130px;
  }
  .cb-file-techs { display: flex; gap: 4px; flex-wrap: wrap; }
  .cb-file-tech {
    font-size: 0.68rem; padding: 1px 6px;
    background: var(--ink-faint, rgba(92, 71, 50, 0.04));
    border-radius: 4px;
  }

  /* ========= Intelligence panels ========= */
  .intel-hotspot-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
    gap: 0.85rem;
  }
  .intel-hotspot-card {
    border: 1px solid var(--line);
    border-radius: 16px;
    padding: 0.85rem 1rem;
    background: rgba(255, 255, 255, 0.54);
    display: grid;
    gap: 0.5rem;
    transition: transform 160ms ease, border-color 160ms ease;
  }
  .intel-hotspot-card:hover {
    transform: translateY(-1px);
    border-color: rgba(29, 78, 216, 0.22);
  }
  .intel-hotspot-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 0.5rem;
  }
  .intel-source-tag {
    font-size: 0.72rem;
    color: var(--muted);
    opacity: 0.8;
  }
  .intel-project-tags {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
  }

  .intel-eco-stack {
    display: grid;
    gap: 1rem;
  }
  .intel-eco-row {
    display: grid;
    grid-template-columns: 140px minmax(0, 1fr);
    gap: 0.85rem;
    align-items: center;
  }
  .intel-eco-label {
    display: flex;
    align-items: center;
    gap: 0.55rem;
    font-size: 0.95rem;
  }
  .intel-eco-icon {
    font-size: 1.2rem;
  }
  .intel-eco-bars {
    display: grid;
    gap: 0.35rem;
  }
  .intel-eco-bar {
    display: grid;
    grid-template-columns: minmax(0, 1fr) auto;
    gap: 0.65rem;
    align-items: center;
    font-size: 0.84rem;
  }

  .intel-similarity-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
    gap: 0.85rem;
  }
  .intel-similarity-card {
    border: 1px solid var(--line);
    border-radius: 16px;
    padding: 0.85rem 1rem;
    background: rgba(255, 255, 255, 0.54);
    display: grid;
    gap: 0.55rem;
  }
  .intel-sim-header {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    flex-wrap: wrap;
    font-size: 0.92rem;
  }
  .intel-sim-arrow {
    color: var(--accent);
    font-size: 1.1rem;
    font-weight: 700;
  }
  .intel-sim-meter {
    display: grid;
    gap: 0.3rem;
    font-size: 0.84rem;
  }

  /* ========= Dark mode ========= */
  :global(body.dark) {
    --paper: #0d1117;
    --panel: rgba(22, 27, 34, 0.92);
    --panel-strong: #161b22;
    --ink: #e6edf3;
    --muted: #8b949e;
    --line: rgba(240, 246, 252, 0.12);
    --shadow: 0 24px 60px rgba(0, 0, 0, 0.45);
    --good: #3fb950;
    --watch: #d29922;
    --danger: #f85149;
    --accent: #58a6ff;
    background:
      radial-gradient(circle at top left, rgba(88, 166, 255, 0.08), transparent 32%),
      radial-gradient(circle at top right, rgba(210, 153, 34, 0.1), transparent 24%),
      linear-gradient(180deg, #0d1117 0%, #10161d 100%);
    color: var(--ink);
  }

  :global(body.dark) .sidebar {
    background:
      linear-gradient(180deg, rgba(8, 13, 21, 0.97), rgba(13, 17, 23, 0.96)),
      radial-gradient(circle at top left, rgba(63, 185, 80, 0.15), transparent 35%);
  }

  :global(body.dark) .hero {
    background:
      linear-gradient(135deg, rgba(22, 27, 34, 0.92), rgba(30, 37, 48, 0.88)),
      radial-gradient(circle at top right, rgba(63, 185, 80, 0.09), transparent 32%);
  }

  :global(body.dark) .signal-row,
  :global(body.dark) .entry-card {
    background: rgba(22, 27, 34, 0.72);
  }

  :global(body.dark) .actionable:hover {
    border-color: rgba(88, 166, 255, 0.24);
    background: rgba(30, 40, 56, 0.92);
  }

  :global(body.dark) .search-shell {
    background: rgba(240, 246, 252, 0.06);
  }

  :global(body.dark) .search-shell input {
    color: var(--ink);
  }

  :global(body.dark) .search-shell kbd {
    border-color: rgba(240, 246, 252, 0.1);
    background: rgba(22, 27, 34, 0.8);
    color: var(--muted);
  }

  :global(body.dark) .toolbar select,
  :global(body.dark) .form-grid select,
  :global(body.dark) .form-grid input,
  :global(body.dark) .form-grid textarea {
    background: #0d1117;
    border-color: var(--line);
    color: var(--ink);
  }

  :global(body.dark) .hero-notes span {
    background: rgba(22, 27, 34, 0.75);
    border-color: var(--line);
  }

  :global(body.dark) .search-card,
  :global(body.dark) .toolbar,
  :global(body.dark) .theme-card,
  :global(body.dark) .control-card,
  :global(body.dark) .focus-card,
  :global(body.dark) .filter-summary {
    background: rgba(22, 27, 34, 0.76);
    border-color: var(--line);
  }

  :global(body.dark) .focus-card {
    background:
      linear-gradient(180deg, rgba(22, 27, 34, 0.94), rgba(20, 28, 38, 0.9)),
      radial-gradient(circle at top right, rgba(63, 185, 80, 0.08), transparent 34%);
  }

  :global(body.dark) .control-card {
    background:
      linear-gradient(180deg, rgba(22, 27, 34, 0.94), rgba(17, 24, 34, 0.9)),
      radial-gradient(circle at top right, rgba(88, 166, 255, 0.08), transparent 32%);
  }

  :global(body.dark) .focus-metrics span,
  :global(body.dark) .filter-chip {
    background: rgba(240, 246, 252, 0.06);
    border-color: rgba(240, 246, 252, 0.08);
    color: var(--ink);
  }

  :global(body.dark) .filter-chip.quiet {
    color: var(--muted);
    background: rgba(22, 27, 34, 0.7);
  }

  :global(body.dark) .theme-status-dot {
    background: rgba(210, 153, 34, 0.82);
    box-shadow: 0 0 0 0.35rem rgba(210, 153, 34, 0.12);
  }

  :global(body.dark) .theme-status-dot.is-dark {
    background: rgba(63, 185, 80, 0.9);
    box-shadow: 0 0 0 0.35rem rgba(63, 185, 80, 0.14);
  }

  :global(body.dark) .theme-segmented button {
    background: rgba(240, 246, 252, 0.04);
    border-color: rgba(240, 246, 252, 0.08);
    color: var(--ink);
  }

  :global(body.dark) .theme-segmented button.active {
    background: linear-gradient(135deg, rgba(88, 166, 255, 0.18), rgba(63, 185, 80, 0.18));
    border-color: rgba(88, 166, 255, 0.28);
  }

  :global(body.dark) .ring-section {
    background: rgba(22, 27, 34, 0.54);
  }

  :global(body.dark) .ring-circle {
    fill: color-mix(in srgb, var(--ring-color), #0d1117 82%);
    stroke: color-mix(in srgb, var(--ring-color), #1a2332 40%);
  }

  :global(body.dark) .radar-canvas {
    background:
      radial-gradient(circle at center, rgba(63, 185, 80, 0.06), transparent 30%),
      linear-gradient(180deg, rgba(22, 27, 34, 0.92), rgba(16, 22, 29, 0.95));
  }

  :global(body.dark) .divider-line {
    stroke: rgba(240, 246, 252, 0.12);
  }

  :global(body.dark) .quadrant-label {
    fill: rgba(230, 237, 243, 0.42);
  }

  :global(body.dark) .blip-label {
    fill: rgba(230, 237, 243, 0);
  }

  :global(body.dark) .blip.hovered .blip-label {
    fill: rgba(230, 237, 243, 0.85);
  }

  :global(body.dark) .project-count-dot {
    fill: #e6edf3;
  }

  :global(body.dark) .project-count-text {
    fill: #0d1117;
  }

  :global(body.dark) .bar-track {
    background: rgba(240, 246, 252, 0.08);
  }

  :global(body.dark) .meter {
    background: rgba(240, 246, 252, 0.08);
  }

  :global(body.dark) .empty-card,
  :global(body.dark) .empty-inline {
    border-color: rgba(240, 246, 252, 0.15);
    background: rgba(22, 27, 34, 0.56);
  }

  :global(body.dark) .ghost-button {
    background: rgba(22, 27, 34, 0.72);
    border-color: var(--line);
    color: var(--ink);
  }

  :global(body.dark) .chip-row span,
  :global(body.dark) .tag-chip {
    background: rgba(22, 27, 34, 0.78);
    border-color: var(--line);
    color: var(--ink);
  }

  :global(body.dark) .project-row {
    background: rgba(22, 27, 34, 0.72);
  }

  :global(body.dark) .modal-overlay {
    background: rgba(1, 4, 9, 0.65);
  }

  :global(body.dark) .modal-card {
    box-shadow: 0 40px 90px rgba(0, 0, 0, 0.55);
  }

  :global(body.dark) .modal-panel {
    background: rgba(22, 27, 34, 0.64);
  }

  :global(body.dark) .motion-callout {
    background: rgba(22, 27, 34, 0.72);
  }

  :global(body.dark) .close-button {
    background: rgba(240, 246, 252, 0.06);
    color: var(--ink);
  }

  :global(body.dark) .signal-loader {
    border-color: rgba(88, 166, 255, 0.12);
    border-top-color: var(--accent);
  }

  :global(body.dark) .intel-hotspot-card,
  :global(body.dark) .intel-similarity-card {
    background: rgba(22, 27, 34, 0.54);
  }

  :global(body.dark) .intel-hotspot-card:hover {
    border-color: rgba(88, 166, 255, 0.22);
  }

  :global(body.dark) .cb-pill {
    background: rgba(240, 246, 252, 0.06);
    color: var(--muted);
  }

  :global(body.dark) .cb-graph-mode button {
    background: rgba(240, 246, 252, 0.04);
    border-color: rgba(240, 246, 252, 0.1);
    color: var(--ink);
  }

  :global(body.dark) .cb-graph-mode button.active {
    background: linear-gradient(135deg, rgba(88, 166, 255, 0.16), rgba(63, 185, 80, 0.18));
    border-color: rgba(88, 166, 255, 0.3);
  }

  :global(body.dark) .cb-graph-canvas {
    background: var(--panel);
  }

  :global(body.dark) .graph-label {
    fill: var(--ink);
    text-shadow: 0 1px 2px rgba(0,0,0,0.5);
  }

  :global(body.dark) .cb-node-detail {
    background: var(--panel);
  }

  :global(body.dark) .cb-tech-chip {
    background: rgba(240, 246, 252, 0.06);
  }

  :global(body.dark) .cb-file-project-header {
    background: rgba(240, 246, 252, 0.03);
  }

  :global(body.dark) .cb-file-project-header:hover {
    background: rgba(240, 246, 252, 0.06);
  }

  :global(body.dark) .cb-pattern-badge {
    background: var(--panel);
    border-color: var(--line);
  }

  :global(body.dark) .blip.hovered .blip-core {
    filter: drop-shadow(0 10px 10px rgba(88, 166, 255, 0.35));
  }

  :global(body.dark) .ring-chip {
    background: color-mix(in srgb, var(--ring-color), #0d1117 82%);
  }

  :global(body.dark) .signal-pill.good,
  :global(body.dark) .risk-badge.healthy {
    background: rgba(63, 185, 80, 0.15);
    color: var(--good);
  }

  :global(body.dark) .signal-pill.watch,
  :global(body.dark) .risk-badge.watch {
    background: rgba(210, 153, 34, 0.15);
    color: var(--watch);
  }

  :global(body.dark) .signal-pill.danger,
  :global(body.dark) .risk-badge.critical {
    background: rgba(248, 81, 73, 0.15);
    color: var(--danger);
  }

  :global(body.dark) .signal-pill.neutral {
    background: rgba(88, 166, 255, 0.12);
    color: var(--accent);
  }

  :global(body.dark) .primary-button {
    background: linear-gradient(135deg, #1f6feb, #3fb950);
  }

  :global(body.dark) .secondary-button {
    background: rgba(88, 166, 255, 0.1);
    border-color: rgba(88, 166, 255, 0.15);
    color: var(--accent);
  }

  :global(body.dark) .form-error {
    background: rgba(248, 81, 73, 0.12);
    border-color: rgba(248, 81, 73, 0.18);
  }

  :global(body.dark) .graph-edge {
    stroke: rgba(139, 148, 158, 0.3);
  }
  :global(body.dark) .graph-edge.highlighted {
    stroke: var(--accent);
  }

  @media (max-width: 760px) {
    .theme-segmented {
      grid-template-columns: 1fr;
    }

    .cb-graph-toolbar {
      display: grid;
    }

    .cb-graph-mode {
      grid-template-columns: 1fr;
    }

    .intel-eco-row {
      grid-template-columns: 1fr;
    }
  }
</style>
