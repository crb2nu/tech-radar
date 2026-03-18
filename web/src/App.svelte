<script lang="ts">
  import { onMount } from 'svelte';
  import { fade, fly } from 'svelte/transition';
  import {
    canvasQuadrants,
    computeDashboardModel,
    formatDate,
    formatRelativeDate,
    getBlipPosition,
    quadrantMeta,
    ringMeta,
    ringOrder,
    uniqueTags,
    type Entry,
    type Insights,
    type Project,
    type Radar
  } from './lib/dashboard';

  type Tab = 'overview' | 'radar' | 'projects' | 'insights' | 'drift';

  type EntryDraft = {
    id?: string;
    name: string;
    quadrant: string;
    ring: string;
    description: string;
    link: string;
    tags: string[];
  };

  const tabs: Array<{ id: Tab; label: string }> = [
    { id: 'overview', label: 'Command deck' },
    { id: 'radar', label: 'Radar' },
    { id: 'projects', label: 'Projects' },
    { id: 'insights', label: 'Intelligence' },
    { id: 'drift', label: 'Drift' }
  ];

  let radar = $state<Radar | null>(null);
  let projects = $state<Project[]>([]);
  let insights = $state<Insights | null>(null);
  let error = $state<string | null>(null);
  let loading = $state(true);

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

  const hasActiveFilters = $derived(
    Boolean(searchQuery.trim()) || selectedQuadrant !== 'all' || selectedRing !== 'all' || selectedTag !== 'all'
  );

  onMount(() => {
    void loadData();

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
    return () => window.removeEventListener('keydown', handleKeydown);
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

    const [projectsResult, insightsResult] = await Promise.allSettled([fetchProjects(), fetchInsights()]);

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
            <label class="search-card">
              <span class="eyebrow">Search</span>
              <div class="search-shell">
                <input
                  bind:this={searchField}
                  bind:value={searchQuery}
                  type="text"
                  placeholder="React, GitOps, infra..."
                />
                <kbd>⌘K</kbd>
              </div>
            </label>

            <div class="toolbar">
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
              <button class="ghost-button" onclick={clearFilters} disabled={!hasActiveFilters}>Reset</button>
            </div>
          </div>
        </header>

        <section class="metric-grid">
          {#each dashboard.summaryMetrics as metric}
            <article class="metric-card" data-tone={metric.tone}>
              <span class="eyebrow">{metric.label}</span>
              <strong>{metric.value}</strong>
              <p>{metric.caption}</p>
            </article>
          {/each}
        </section>

        <section class="content-scroll">
          {#if activeTab === 'overview'}
            <div class="page-grid" in:fade>
              <article class="panel panel-large">
                <div class="panel-header">
                  <div>
                    <span class="eyebrow">Priority queue</span>
                    <h2>Where the next governance work should go</h2>
                  </div>
                  {#if isAdminMode}
                    <button class="ghost-button" onclick={() => activeTab = 'insights'}>Open intelligence</button>
                  {/if}
                </div>

                <div class="queue-grid">
                  <div class="queue-column">
                    <h3>Uncharted demand</h3>
                    {#if dashboard.opportunityQueue.length > 0}
                      {#each dashboard.opportunityQueue.slice(0, 4) as item}
                        <button class="signal-row actionable" onclick={() => {
                          if (isAdminMode) {
                            openEntryForm(null, item.name);
                          } else {
                            searchQuery = item.name;
                          }
                        }}>
                          <div>
                            <strong>{item.name}</strong>
                            <span>Appears in {item.count} project{item.count === 1 ? '' : 's'} without an explicit decision.</span>
                          </div>
                          <span class="signal-pill watch">{item.count}</span>
                        </button>
                      {/each}
                    {:else}
                      <div class="empty-card">Everything discovered in project scans already has a radar decision.</div>
                    {/if}
                  </div>

                  <div class="queue-column">
                    <h3>Drift pressure</h3>
                    {#if dashboard.driftEntries.length > 0}
                      {#each dashboard.driftEntries.slice(0, 4) as entry}
                        <button class="signal-row actionable" onclick={() => { selectedEntry = entry; activeTab = 'drift'; }}>
                          <div>
                            <strong>{entry.name}</strong>
                            <span>{entry.project_ids?.length ?? 0} projects still depend on a hold item.</span>
                          </div>
                          <span class="signal-pill danger">Hold</span>
                        </button>
                      {/each}
                    {:else}
                      <div class="empty-card">No active drift. Hold technologies are currently out of the monitored estate.</div>
                    {/if}
                  </div>
                </div>
              </article>

              <article class="panel">
                <div class="panel-header">
                  <div>
                    <span class="eyebrow">Portfolio balance</span>
                    <h2>How the radar is distributed</h2>
                  </div>
                </div>

                <div class="breakdown-stack">
                  <section>
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
                  </section>

                  <section>
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
                </div>
              </article>

              <article class="panel panel-large">
                <div class="panel-header">
                  <div>
                    <span class="eyebrow">Project spotlight</span>
                    <h2>Which teams are aligned and which need attention</h2>
                  </div>
                  <button class="ghost-button" onclick={() => activeTab = 'projects'}>View all projects</button>
                </div>

                <div class="project-grid">
                  {#each dashboard.projectSpotlight as project}
                    <button class="project-card" data-risk={project.riskLevel} onclick={() => {
                      searchQuery = project.name;
                      activeTab = 'projects';
                    }}>
                      <div class="project-card-header">
                        <div>
                          <h3>{project.name}</h3>
                          <p>Last scan {formatRelativeDate(project.last_scan)}</p>
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
                        <span>{project.untrackedTechs} uncharted</span>
                        <span>{project.holdTechs} hold</span>
                      </div>
                      <div class="chip-row">
                        {#each project.topTechs as tech}
                          <span>{tech}</span>
                        {/each}
                      </div>
                    </button>
                  {/each}
                </div>
              </article>

              <article class="panel">
                <div class="panel-header">
                  <div>
                    <span class="eyebrow">Decision ledger</span>
                    <h2>Recent movement</h2>
                  </div>
                </div>

                {#if dashboard.recentMoves.length > 0}
                  <div class="timeline-list">
                    {#each dashboard.recentMoves.slice(0, 6) as move}
                      <div class="timeline-item">
                        <div class="timeline-dot" style={`--dot-color:${ringMeta[move.to_ring]?.accent ?? '#1d4ed8'}`}></div>
                        <div>
                          <strong>{move.name}</strong>
                          <p>
                            {move.from_ring ? `${move.from_ring.toUpperCase()} -> ${move.to_ring.toUpperCase()}` : `Entered ${move.to_ring.toUpperCase()}`}
                            on {formatDate(move.date)}
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
              </article>

              <article class="panel">
                <div class="panel-header">
                  <div>
                    <span class="eyebrow">Signal sources</span>
                    <h2>Where intelligence is coming from</h2>
                  </div>
                </div>

                <div class="breakdown-stack compact">
                  <section>
                    <h3>Scan sources</h3>
                    {#if dashboard.sourceBreakdown.length > 0}
                      {#each dashboard.sourceBreakdown as stat}
                        <div class="bar-row">
                          <span>{stat.label}</span>
                          <div class="bar-track">
                            <div class="bar-fill neutral" style={`width:${Math.max(10, (stat.count / Math.max(1, dashboard.projectCount * 6)) * 100)}%`}></div>
                          </div>
                          <strong>{stat.count}</strong>
                        </div>
                      {/each}
                    {:else}
                      <div class="empty-card">No project scan sources detected yet.</div>
                    {/if}
                  </section>
                  <section>
                    <h3>Most used tags</h3>
                    <div class="chip-cloud">
                      {#if dashboard.topTags.length > 0}
                        {#each dashboard.topTags as tag}
                          <span class="tag-chip">#{tag.name} <strong>{tag.count}</strong></span>
                        {/each}
                      {:else}
                        <span class="empty-inline">No governance tags recorded yet.</span>
                      {/if}
                    </div>
                  </section>
                </div>
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
                      <g
                        class="blip"
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
                        {#if entry.moved > 0}
                          <path d="M 0 -11 L 10 7 L -10 7 Z" fill={ringMeta[entry.ring]?.accent ?? '#1d4ed8'} />
                        {:else if entry.moved < 0}
                          <path d="M 0 11 L 10 -7 L -10 -7 Z" fill={ringMeta[entry.ring]?.accent ?? '#b42318'} />
                        {:else}
                          <circle r="8" fill={ringMeta[entry.ring]?.accent ?? '#1d4ed8'} />
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
                            <button class="entry-card" onclick={() => selectedEntry = entry}>
                              <div class="entry-card-top">
                                <strong>{entry.name}</strong>
                                <span class="entry-meta">{entry.project_ids?.length ?? 0} projects</span>
                              </div>
                              <p>{entry.description}</p>
                              <div class="chip-row">
                                <span>{quadrantMeta[entry.quadrant]?.label ?? entry.quadrant}</span>
                                <span>{movementLabel(entry)}</span>
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
              {#each selectedEntry.tags ?? [] as tag}
                <span>#{tag}</span>
              {/each}
            </div>

            {#if selectedEntry.link}
              <a class="text-link" href={selectedEntry.link} target="_blank" rel="noreferrer">Open technical context</a>
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
              <span>Reference link</span>
              <input bind:value={editEntryData.link} type="url" placeholder="https://..." />
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
    grid-template-columns: repeat(4, minmax(0, 1fr));
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
    font-size: 2rem;
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

  .queue-column h3,
  .breakdown-stack h3,
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
  .blip circle {
    transition: transform 150ms ease, filter 150ms ease;
  }

  .blip.hovered path,
  .blip.hovered circle:first-child {
    filter: drop-shadow(0 10px 10px rgba(29, 78, 216, 0.26));
    transform: scale(1.12);
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
    }

    .hero,
    .page-grid,
    .radar-page {
      grid-template-columns: 1fr;
    }

    .metric-grid {
      grid-template-columns: repeat(2, minmax(0, 1fr));
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

    .panel-header,
    .project-card-header,
    .drift-card-header,
    .modal-header,
    .form-actions,
    .score-row,
    .entry-card-top {
      flex-direction: column;
    }

    .split-grid {
      grid-template-columns: 1fr;
    }

    .bar-row {
      grid-template-columns: 1fr;
    }
  }
</style>
