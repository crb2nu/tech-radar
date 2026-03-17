<script lang="ts">
  import { onMount } from 'svelte';
  import { fade, fly } from 'svelte/transition';

  interface HistoryEntry {
    date: string;
    ring: string;
    reason: string;
    author?: string;
  }

  interface Entry {
    id: string;
    name: string;
    quadrant: string;
    ring: string;
    description: string;
    moved: number; // -1, 0, 1
    tags?: string[];
    history?: HistoryEntry[];
    project_ids?: string[];
  }

  interface Radar {
    id: string;
    meta: {
      name: string;
    };
    entries: Entry[];
  }

  let radar: Radar | null = $state(null);
  let projects: any[] = $state([]);
  let insights: any = $state(null);
  let error: string | null = $state(null);
  let searchQuery = $state('');
  let selectedQuadrant = $state('all');
  let selectedRing = $state('all');
  let selectedTag = $state('all');
  let hoveredEntry: string | null = $state(null);
  let selectedEntry: Entry | null = $state(null);
  let activeTab = $state('radar');

  const quadrants = ['languages', 'frameworks', 'tools', 'platforms'];
  const rings = ['adopt', 'trial', 'assess', 'hold'];

  const colors = {
    adopt: '#10b981', // Emerald 500
    trial: '#3b82f6', // Blue 500
    assess: '#f59e0b', // Amber 500
    hold: '#ef4444'   // Red 500
  };

  const ringRadius = [70, 140, 210, 280];

  const allTags = $derived(
    Array.from(new Set(radar?.entries.flatMap(e => e.tags || []) || []))
  );

  const filteredEntries = $derived(
    radar?.entries.filter(e => {
      const matchesSearch = e.name.toLowerCase().includes(searchQuery.toLowerCase()) || 
                          e.description.toLowerCase().includes(searchQuery.toLowerCase());
      const matchesQuadrant = selectedQuadrant === 'all' || e.quadrant === selectedQuadrant;
      const matchesRing = selectedRing === 'all' || e.ring === selectedRing;
      const matchesTag = selectedTag === 'all' || (e.tags && e.tags.includes(selectedTag));
      return matchesSearch && matchesQuadrant && matchesRing && matchesTag;
    }) || []
  );

  const filteredProjects = $derived(
    projects.filter(p => {
      const matchesSearch = p.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
                          p.technologies?.some((t: any) => t.name.toLowerCase().includes(searchQuery.toLowerCase()));
      return matchesSearch;
    })
  );

  // Statistics
  const stats = $derived({
    totalTechs: radar?.entries.length || 0,
    totalProjects: projects.length || 0,
    driftCount: radar?.entries.filter(e => e.ring === 'hold' && e.project_ids && e.project_ids.length > 0).length || 0
  });

  async function handleSearch() {
    if (searchQuery.length < 2) return;
    try {
      const response = await fetch(`/api/v1/search?q=${encodeURIComponent(searchQuery)}`);
      if (response.ok) {
        const results = await response.json();
        console.log('Search results:', results);
      }
    } catch (e) {
      console.error('Search failed', e);
    }
  }

  onMount(async () => {
    try {
      let response = await fetch('/api/v1/radar');
      if (!response.ok || response.status === 404) {
        response = await fetch('/api/radar');
      }
      if (response.ok) {
        radar = await response.json();
      } else {
        error = "Failed to fetch radar data";
      }
    } catch (e: any) {
      error = e.message;
    }

    try {
      const pResponse = await fetch('/api/v1/projects');
      if (pResponse.ok) {
        projects = await pResponse.json();
      }
    } catch (e) {
      console.error('Failed to fetch projects', e);
    }

    try {
      const iResponse = await fetch('/api/v1/insights');
      if (iResponse.ok) {
        insights = await iResponse.json();
      }
    } catch (e) {
      console.error('Failed to fetch insights', e);
    }
  });

  function getEntriesByRing(ring: string) {
    return filteredEntries.filter(e => e.ring === ring);
  }

  function getBlipPosition(entry: Entry, index: number) {
    const quadIndex = quadrants.indexOf(entry.quadrant);
    const ringIndex = rings.indexOf(entry.ring);
    
    if (quadIndex === -1 || ringIndex === -1) return { x: 0, y: 0 };

    const baseAngle = quadIndex * (Math.PI / 2);
    const innerR = ringIndex === 0 ? 20 : ringRadius[ringIndex - 1];
    const outerR = ringRadius[ringIndex];

    const seed = entry.name.split('').reduce((acc, char) => acc + char.charCodeAt(0), 0);
    const randomR = innerR + (Math.abs(Math.sin(seed)) * (outerR - innerR - 20)) + 10;
    const randomAngle = baseAngle + (Math.abs(Math.cos(seed)) * (Math.PI / 2 - 0.4)) + 0.2;

    return {
      x: 300 + randomR * Math.cos(randomAngle),
      y: 300 + randomR * Math.sin(randomAngle)
    };
  }

  function formatDate(dateStr: string) {
    if (!dateStr) return 'Never';
    return new Date(dateStr).toLocaleDateString();
  }
</script>

<main>
  {#if error}
    <div class="full-screen-error" in:fade>
      <div class="error-box">
        <div class="error-icon">🛑</div>
        <h2>System Connection Error</h2>
        <p>{error}</p>
        <button onclick={() => window.location.reload()}>Retry Connection</button>
      </div>
    </div>
  {:else if !radar}
    <div class="full-screen-loading">
      <div class="loader"></div>
      <p>Synchronizing Technology Ecosystem...</p>
    </div>
  {:else}
    <div class="page-layout">
      <nav class="sidebar">
        <div class="brand">
          <div class="logo">TR</div>
          <span>TechRadar</span>
        </div>
        
        <div class="nav-links">
          <button class:active={activeTab === 'radar'} onclick={() => activeTab = 'radar'}>
            <span class="icon">🎯</span> Radar
          </button>
          <button class:active={activeTab === 'projects'} onclick={() => activeTab = 'projects'}>
            <span class="icon">📦</span> Projects
          </button>
          <button class:active={activeTab === 'insights'} onclick={() => activeTab = 'insights'}>
            <span class="icon">🧠</span> Insights
          </button>
          {#if stats.driftCount > 0}
            <button class:active={activeTab === 'drift'} onclick={() => activeTab = 'drift'} class="drift-nav">
              <span class="icon">⚠️</span> Drift
            </button>
          {/if}
        </div>

        <div class="sidebar-stats">
          <div class="stat-item">
            <span class="label">Technologies</span>
            <span class="value">{stats.totalTechs}</span>
          </div>
          <div class="stat-item">
            <span class="label">Projects</span>
            <span class="value">{stats.totalProjects}</span>
          </div>
          {#if stats.driftCount > 0}
            <div class="stat-item drift">
              <span class="label">Drift Detected</span>
              <span class="value">{stats.driftCount}</span>
            </div>
          {/if}
        </div>
      </nav>

      <div class="main-content">
        <header>
          <div class="header-top">
            <h1>{radar.meta.name}</h1>
            <div class="search-container">
              <input 
                type="text" 
                placeholder="Quick Search..." 
                bind:value={searchQuery}
                oninput={handleSearch}
              />
              <span class="shortcut">⌘K</span>
            </div>
          </div>

          {#if activeTab === 'radar'}
            <div class="header-filters">
              <div class="filter">
                <label>Quadrant</label>
                <select bind:value={selectedQuadrant}>
                  <option value="all">All Sections</option>
                  {#each quadrants as q}<option value={q}>{q.toUpperCase()}</option>{/each}
                </select>
              </div>
              <div class="filter">
                <label>Ring</label>
                <select bind:value={selectedRing}>
                  <option value="all">All Rings</option>
                  {#each rings as r}<option value={r}>{r.toUpperCase()}</option>{/each}
                </select>
              </div>
              <div class="filter">
                <label>Tag</label>
                <select bind:value={selectedTag}>
                  <option value="all">All Tags</option>
                  {#each allTags as t}<option value={t}>{t}</option>{/each}
                </select>
              </div>
            </div>
          {/if}
        </header>

        <div class="scroll-area">
          {#if activeTab === 'radar'}
            <div class="radar-dashboard" in:fade>
              <div class="radar-svg-container">
                <svg viewBox="0 0 600 600">
                  {#each rings.slice().reverse() as ring, i}
                    {@const r = ringRadius[3 - i]}
                    <circle cx="300" cy="300" r={r} class="ring-bg" style="--ring-color: {colors[ring as keyof typeof colors]}" />
                  {/each}
                  
                  <line x1="300" y1="20" x2="300" y2="580" class="divider" />
                  <line x1="20" y1="300" x2="580" y2="300" class="divider" />

                  <text x="40" y="40" class="quad-label">LANGUAGES</text>
                  <text x="560" y="40" text-anchor="end" class="quad-label">FRAMEWORKS</text>
                  <text x="40" y="560" class="quad-label">TOOLS</text>
                  <text x="560" y="560" text-anchor="end" class="quad-label">PLATFORMS</text>

                  {#each filteredEntries as entry, i}
                    {@const pos = getBlipPosition(entry, i)}
                    <g 
                      class="blip" 
                      class:hovered={hoveredEntry === entry.name}
                      transform="translate({pos.x}, {pos.y})"
                      onmouseenter={() => hoveredEntry = entry.name}
                      onmouseleave={() => hoveredEntry = null}
                      onclick={() => selectedEntry = entry}
                    >
                      {#if entry.moved === 1}
                        <path d="M 0 -8 L 7 6 L -7 6 Z" fill={colors[entry.ring as keyof typeof colors]} />
                      {:else if entry.moved === -1}
                        <path d="M 0 8 L 7 -6 L -7 -6 Z" fill={colors[entry.ring as keyof typeof colors]} />
                      {:else}
                        <circle r="6" fill={colors[entry.ring as keyof typeof colors]} />
                      {/if}
                      
                      {#if entry.project_ids && entry.project_ids.length > 0}
                        <circle cx="8" cy="-8" r="5" fill="#3b82f6" stroke="#0f172a" stroke-width="1" />
                        <text x="8" y="-6" text-anchor="middle" font-size="6" fill="white" font-weight="900">{entry.project_ids.length}</text>
                      {/if}
                      
                      <text y="-18" text-anchor="middle" class="blip-text" class:visible={hoveredEntry === entry.name}>{entry.name}</text>
                    </g>
                  {/each}
                </svg>
              </div>

              <div class="entries-grid-area">
                {#each rings as ring}
                  {@const entries = getEntriesByRing(ring)}
                  {#if entries.length > 0}
                    <div class="ring-group">
                      <div class="group-header">
                        <span class="dot" style="background: {colors[ring as keyof typeof colors]}"></span>
                        <h3>{ring.toUpperCase()}</h3>
                      </div>
                      <div class="items">
                        {#each entries as entry}
                          <div 
                            class="item-card" 
                            class:active={hoveredEntry === entry.name}
                            onclick={() => selectedEntry = entry}
                            onmouseenter={() => hoveredEntry = entry.name}
                            onmouseleave={() => hoveredEntry = null}
                          >
                            <div class="item-info">
                              <span class="name">{entry.name}</span>
                              {#if entry.project_ids && entry.project_ids.length > 0}
                                <span class="count">{entry.project_ids.length}</span>
                              {/if}
                            </div>
                            <div class="item-meta">
                              <span class="quad">{entry.quadrant}</span>
                            </div>
                          </div>
                        {/each}
                      </div>
                    </div>
                  {/if}
                {/each}
              </div>
            </div>
          {:else if activeTab === 'projects'}
            <div class="project-view" in:fade>
              <div class="project-grid">
                {#each filteredProjects as project}
                  <div class="p-card">
                    <div class="p-header">
                      <h3>{project.name}</h3>
                      <span class="p-date">Scan: {formatDate(project.last_scan)}</span>
                    </div>
                    <div class="p-stats">
                      <div class="p-stat">
                        <span class="v">{project.technologies?.length || 0}</span>
                        <span class="l">Deps</span>
                      </div>
                    </div>
                    <div class="p-techs">
                      {#each (project.technologies || []).slice(0, 12) as tech}
                        <span class="t-tag">{tech.name}</span>
                      {/each}
                      {#if project.technologies?.length > 12}
                        <span class="t-tag more">+{project.technologies.length - 12} more</span>
                      {/if}
                    </div>
                  </div>
                {/each}
              </div>
            </div>
          {:else if activeTab === 'insights'}
            <div class="insights-view" in:fade>
              <div class="insights-header">
                <h2>🧠 Technology Intelligence</h2>
                <p>Ecosystem analytics driven by live CI/CD pipeline scans.</p>
              </div>

              {#if insights}
                <div class="insights-grid">
                  <!-- Uncharted Tech (Shadow IT) -->
                  <div class="insight-card highlight">
                    <h3>Top Uncharted Technologies</h3>
                    <p class="subtitle">Widely used but not officially tracked on the Radar.</p>
                    <div class="stats-list">
                      {#each insights.top_uncharted || [] as stat}
                        <div class="stat-row">
                          <span class="t-name">{stat.name}</span>
                          <span class="t-count" title="{stat.count} projects">
                            <span class="bar" style="width: {Math.max(5, Math.min(100, (stat.count / insights.projects_count) * 100))}%"></span>
                            <span class="num">{stat.count}</span>
                          </span>
                        </div>
                      {/each}
                      {#if (!insights.top_uncharted || insights.top_uncharted.length === 0)}
                        <p class="empty">No uncharted technologies found. Great job!</p>
                      {/if}
                    </div>
                  </div>

                  <!-- Top Adopted -->
                  <div class="insight-card">
                    <h3>Most Adopted Technologies</h3>
                    <p class="subtitle">Radar technologies with the highest internal usage.</p>
                    <div class="stats-list">
                      {#each insights.top_adopted || [] as stat}
                        <div class="stat-row">
                          <span class="t-name">{stat.name}</span>
                          <span class="t-count adopted" title="{stat.count} projects">
                            <span class="bar" style="width: {Math.max(5, Math.min(100, (stat.count / insights.projects_count) * 100))}%"></span>
                            <span class="num">{stat.count}</span>
                          </span>
                        </div>
                      {/each}
                      {#if (!insights.top_adopted || insights.top_adopted.length === 0)}
                        <p class="empty">No adopted technologies found yet.</p>
                      {/if}
                    </div>
                  </div>
                </div>
              {:else}
                 <div class="loading-state">
                  <div class="spinner"></div>
                  <p>Calculating ecosystem insights...</p>
                </div>
              {/if}
            </div>
          {:else if activeTab === 'drift'}
            <div class="drift-view" in:fade>
              <div class="drift-header">
                <h2>🚨 Technology Drift Detected</h2>
                <p>The following technologies are marked as <strong>HOLD</strong> but are still being used in active projects.</p>
              </div>
              <div class="drift-grid">
                {#each radar?.entries.filter(e => e.ring === 'hold' && e.project_ids && e.project_ids.length > 0) || [] as entry}
                  <div class="drift-card" onclick={() => { selectedEntry = entry; activeTab = 'radar'; }} role="button" tabindex="0">
                    <div class="drift-card-header">
                      <h3>{entry.name}</h3>
                      <span class="drift-badge">HOLD</span>
                    </div>
                    <p>{entry.description}</p>
                    <div class="drift-projects">
                      <span class="label">Detected in {entry.project_ids?.length} projects:</span>
                      <div class="p-list">
                        {#each entry.project_ids || [] as pid}
                          <span class="p-tag">{pid}</span>
                        {/each}
                      </div>
                    </div>
                  </div>
                {/each}
              </div>
            </div>
          {/if}
        </div>
      </div>
    </div>
  {/if}

  {#if selectedEntry}
    <div class="modal-overlay" transition:fade onclick={() => selectedEntry = null}>
      <div class="modal" transition:fly={{ y: 20 }} onclick={e => e.stopPropagation()}>
        <button class="close" onclick={() => selectedEntry = null}>&times;</button>
        
        <div class="modal-header">
          <div class="title-line">
            <h2>{selectedEntry.name}</h2>
            <span class="ring-label" style="--ring-color: {colors[selectedEntry.ring as keyof typeof colors]}">
              {selectedEntry.ring.toUpperCase()}
            </span>
          </div>
          <div class="meta-line">
            <span class="quad-label">{selectedEntry.quadrant}</span>
            {#each selectedEntry.tags || [] as tag}
              <span class="tag">#{tag}</span>
            {/each}
          </div>
        </div>

        <div class="modal-content-inner">
          <div class="main-info">
            <p class="desc">{selectedEntry.description}</p>
            {#if selectedEntry.link}
              <a href={selectedEntry.link} target="_blank" class="docs-btn">Technical Docs ↗</a>
            {/if}
          </div>

          <div class="adoption-info">
            <h3>Internal Adoption</h3>
            {#if selectedEntry.project_ids && selectedEntry.project_ids.length > 0}
              <div class="project-list">
                {#each selectedEntry.project_ids as pid}
                  <span class="p-chip">{pid}</span>
                {/each}
              </div>
            {:else}
              <p class="empty-adoption">No detected usage in monitored projects.</p>
            {/if}
          </div>

          {#if selectedEntry.history && selectedEntry.history.length > 0}
            <div class="history-info">
              <h3>Governance History</h3>
              <div class="timeline">
                {#each selectedEntry.history.slice().reverse() as h}
                  <div class="history-step" style="--ring-color: {colors[h.ring as keyof typeof colors]}">
                    <div class="step-top">
                      <span class="date">{formatDate(h.date)}</span>
                      <span class="status">{h.ring.toUpperCase()}</span>
                    </div>
                    {#if h.author}
                      <div class="author">By <strong>{h.author}</strong></div>
                    {/if}
                    {#if h.reason}
                      <p class="reason">{h.reason}</p>
                    {/if}
                  </div>
                {/each}
              </div>
            </div>
          {/if}
        </div>
      </div>
    </div>
  {/if}
</main>

<style>
  :global(body) {
    margin: 0;
    padding: 0;
    background-color: #0f172a;
    color: #f1f5f9;
    font-family: 'Inter', system-ui, -apple-system, sans-serif;
    overflow: hidden;
  }

  main {
    height: 100vh;
    width: 100vw;
    display: flex;
    background: #0f172a;
  }

  .page-layout {
    display: flex;
    width: 100%;
    height: 100%;
  }

  /* Sidebar */
  .sidebar {
    width: 280px;
    background: #0b1120;
    border-right: 1px solid #1e293b;
    display: flex;
    flex-direction: column;
    padding: 2rem;
  }

  .brand {
    display: flex;
    align-items: center;
    gap: 1rem;
    font-weight: 900;
    font-size: 1.25rem;
    margin-bottom: 4rem;
  }

  .logo {
    background: #3b82f6;
    width: 40px;
    height: 40px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 10px;
    color: white;
    font-size: 1rem;
  }

  .nav-links {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    flex: 1;
  }

  .nav-links button {
    background: transparent;
    border: none;
    padding: 1rem 1.5rem;
    color: #94a3b8;
    text-align: left;
    font-weight: 700;
    cursor: pointer;
    border-radius: 12px;
    display: flex;
    align-items: center;
    gap: 1rem;
    transition: all 0.2s;
  }

  .nav-links button:hover {
    background: rgba(255,255,255,0.05);
    color: white;
  }

  .nav-links button.active {
    background: #3b82f615;
    color: #3b82f6;
  }

  .sidebar-stats {
    background: rgba(255,255,255,0.03);
    padding: 1.5rem;
    border-radius: 16px;
    display: flex;
    flex-direction: column;
    gap: 1.25rem;
  }

  .stat-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .stat-item .label {
    font-size: 0.75rem;
    font-weight: 700;
    color: #64748b;
    text-transform: uppercase;
  }

  .stat-item .value {
    font-weight: 900;
    color: white;
  }

  .stat-item.drift .value {
    color: #ef4444;
  }

  /* Main Content */
  .main-content {
    flex: 1;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  header {
    padding: 2.5rem 4rem;
    background: #0f172a;
    border-bottom: 1px solid #1e293b;
  }

  .header-top {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 2rem;
  }

  h1 {
    margin: 0;
    font-size: 2rem;
    font-weight: 900;
    letter-spacing: -0.04em;
  }

  .search-container {
    position: relative;
    width: 400px;
  }

  .search-container input {
    width: 100%;
    background: #1e293b;
    border: 1px solid #334155;
    padding: 0.8rem 1.5rem;
    border-radius: 12px;
    color: white;
    outline: none;
  }

  .shortcut {
    position: absolute;
    right: 1rem;
    top: 50%;
    transform: translateY(-50%);
    font-size: 0.7rem;
    font-weight: 800;
    color: #64748b;
    background: #0f172a;
    padding: 0.2rem 0.4rem;
    border-radius: 4px;
  }

  .header-filters {
    display: flex;
    gap: 3rem;
  }

  .filter {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .filter label {
    font-size: 0.65rem;
    font-weight: 900;
    color: #64748b;
    text-transform: uppercase;
    letter-spacing: 0.1em;
  }

  .filter select {
    background: transparent;
    border: none;
    color: white;
    font-weight: 700;
    cursor: pointer;
    font-size: 0.9rem;
    padding: 0;
  }

  /* Scroll Area */
  .scroll-area {
    flex: 1;
    overflow-y: auto;
    padding: 4rem;
  }

  /* Radar View */
  .radar-dashboard {
    display: grid;
    grid-template-columns: 600px 1fr;
    gap: 6rem;
    align-items: start;
  }

  .radar-svg-container {
    background: #0b1120;
    padding: 2rem;
    border-radius: 32px;
    border: 1px solid #1e293b;
    box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.5);
  }

  .ring-bg {
    fill: var(--ring-color);
    fill-opacity: 0.03;
    stroke: var(--ring-color);
    stroke-opacity: 0.15;
    stroke-width: 1;
  }

  .divider {
    stroke: #1e293b;
    stroke-width: 1;
    stroke-dasharray: 4 4;
  }

  .quad-label {
    font-size: 10px;
    font-weight: 900;
    fill: #475569;
    letter-spacing: 0.2em;
  }

  .blip {
    cursor: pointer;
    outline: none;
  }

  .blip circle, .blip path {
    transition: all 0.2s;
  }

  .blip.hovered circle, .blip.hovered path {
    filter: drop-shadow(0 0 10px rgba(59, 130, 246, 0.5));
    stroke: white;
    stroke-width: 2;
  }

  .blip-text {
    font-size: 9px;
    fill: white;
    font-weight: 800;
    opacity: 0;
    pointer-events: none;
  }

  .blip-text.visible {
    opacity: 1;
  }

  /* Entries Area */
  .entries-grid-area {
    display: flex;
    flex-direction: column;
    gap: 4rem;
  }

  .ring-group h3 {
    margin: 0 0 1.5rem 0;
    font-size: 1.25rem;
    font-weight: 900;
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }

  .ring-group .dot {
    width: 10px;
    height: 10px;
    border-radius: 50%;
  }

  .items {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
    gap: 1rem;
  }

  .item-card {
    background: #1e293b;
    padding: 1.25rem;
    border-radius: 16px;
    border: 1px solid #334155;
    cursor: pointer;
    transition: all 0.2s;
  }

  .item-card:hover, .item-card.active {
    border-color: #3b82f6;
    background: #24334d;
    transform: translateY(-2px);
  }

  .item-info {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 0.75rem;
  }

  .item-info .name {
    font-weight: 800;
    font-size: 1.1rem;
  }

  .item-info .count {
    background: #3b82f620;
    color: #60a5fa;
    font-size: 0.7rem;
    font-weight: 900;
    padding: 0.2rem 0.5rem;
    border-radius: 6px;
  }

  .item-meta .quad {
    font-size: 0.6rem;
    font-weight: 900;
    text-transform: uppercase;
    color: #64748b;
    letter-spacing: 0.05em;
  }

  /* Project Grid */
  .project-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(400px, 1fr));
    gap: 2.5rem;
  }

  .p-card {
    background: #1e293b;
    padding: 2.5rem;
    border-radius: 24px;
    border: 1px solid #334155;
    box-shadow: 0 10px 15px -3px rgba(0, 0, 0, 0.3);
  }

  .p-header {
    display: flex;
    justify-content: space-between;
    margin-bottom: 1.5rem;
  }

  .p-header h3 {
    margin: 0;
    font-size: 1.5rem;
    font-weight: 900;
  }

  .p-date {
    font-size: 0.7rem;
    font-weight: 800;
    color: #64748b;
  }

  .p-stats {
    margin-bottom: 2rem;
  }

  .p-stat {
    display: flex;
    flex-direction: column;
  }

  .p-stat .v {
    font-size: 2rem;
    font-weight: 900;
    color: #3b82f6;
  }

  .p-stat .l {
    font-size: 0.7rem;
    font-weight: 900;
    text-transform: uppercase;
    color: #64748b;
  }

  .p-techs {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
  }

  .t-tag {
    background: #0f172a;
    padding: 0.4rem 0.8rem;
    border-radius: 8px;
    font-size: 0.75rem;
    font-weight: 700;
    border: 1px solid #334155;
  }

  .t-tag.more {
    color: #3b82f6;
    border-color: #3b82f630;
  }

  /* Modal */
  .modal-overlay {
    position: fixed;
    inset: 0;
    background: rgba(15, 23, 42, 0.9);
    backdrop-filter: blur(12px);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
  }

  .modal {
    background: #1e293b;
    width: 900px;
    max-height: 85vh;
    border-radius: 32px;
    padding: 4rem;
    position: relative;
    border: 1px solid #334155;
    overflow-y: auto;
  }

  .close {
    position: absolute;
    top: 2rem;
    right: 2rem;
    background: none;
    border: none;
    color: #64748b;
    font-size: 2rem;
    cursor: pointer;
  }

  .modal-header {
    margin-bottom: 3rem;
  }

  .title-line {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 1rem;
  }

  .title-line h2 {
    margin: 0;
    font-size: 2.5rem;
    font-weight: 900;
  }

  .ring-label {
    padding: 0.5rem 1.5rem;
    border-radius: 9999px;
    background: color-mix(in srgb, var(--ring-color), transparent 85%);
    color: var(--ring-color);
    font-weight: 900;
    font-size: 0.8rem;
    letter-spacing: 0.1em;
  }

  .meta-line {
    display: flex;
    gap: 1rem;
  }

  .meta-line .quad-label {
    color: #3b82f6;
    font-weight: 900;
    text-transform: uppercase;
    font-size: 0.8rem;
  }

  .meta-line .tag {
    color: #64748b;
    font-weight: 700;
    font-size: 0.8rem;
  }

  .modal-content-inner {
    display: flex;
    flex-direction: column;
    gap: 4rem;
  }

  .desc {
    font-size: 1.1rem;
    line-height: 1.7;
    color: #cbd5e1;
    margin-bottom: 2rem;
  }

  .docs-btn {
    display: inline-block;
    background: #3b82f6;
    color: white;
    padding: 1rem 2rem;
    border-radius: 14px;
    text-decoration: none;
    font-weight: 800;
  }

  .adoption-info h3, .history-info h3 {
    font-size: 1.25rem;
    font-weight: 900;
    margin-bottom: 1.5rem;
  }

  .project-list {
    display: flex;
    flex-wrap: wrap;
    gap: 0.75rem;
  }

  .p-chip {
    background: #0f172a;
    padding: 0.6rem 1.25rem;
    border-radius: 10px;
    font-weight: 700;
    border: 1px solid #334155;
  }

  .timeline {
    display: flex;
    flex-direction: column;
    gap: 1.5rem;
  }

  .history-step {
    background: #0b1120;
    padding: 2rem;
    border-radius: 20px;
    border-left: 4px solid var(--ring-color);
  }

  .step-top {
    display: flex;
    justify-content: space-between;
    margin-bottom: 1rem;
  }

  .step-top .date {
    font-family: monospace;
    color: #64748b;
  }

  .step-top .status {
    font-weight: 900;
    font-size: 0.7rem;
    color: var(--ring-color);
  }

  /* States */
  .full-screen-loading, .full-screen-error {
    height: 100vh;
    width: 100vw;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    background: #0f172a;
  }

  .loader {
    width: 48px;
    height: 48px;
    border: 5px solid #1e293b;
    border-top-color: #3b82f6;
    border-radius: 50%;
    animation: spin 1s linear infinite;
    margin-bottom: 2rem;
  }

  @keyframes spin { to { transform: rotate(360deg); } }

  .error-box {
    text-align: center;
    background: #1e293b;
    padding: 4rem;
    border-radius: 32px;
    border: 1px solid #ef444430;
  }

  .error-icon { font-size: 4rem; margin-bottom: 2rem; }

  .error-box button {
    margin-top: 2rem;
    background: #3b82f6;
    color: white;
    border: none;
    padding: 1rem 2.5rem;
    border-radius: 12px;
    font-weight: 800;
    cursor: pointer;
  }

  /* Drift View */
  .drift-header {
    margin-bottom: 4rem;
  }

  .drift-header h2 {
    font-size: 2.5rem;
    font-weight: 900;
    color: #ef4444;
    margin-bottom: 1rem;
  }

  .drift-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(400px, 1fr));
    gap: 2rem;
  }

  .drift-card {
    background: #1e293b;
    padding: 2.5rem;
    border-radius: 24px;
    border: 2px solid #ef444430;
    cursor: pointer;
    transition: all 0.2s;
  }

  .drift-card:hover {
    border-color: #ef4444;
    transform: translateY(-4px);
  }

  .drift-badge {
    background: #ef4444;
    color: white;
    padding: 0.25rem 0.75rem;
    border-radius: 6px;
    font-size: 0.7rem;
    font-weight: 900;
  }

  .drift-projects {
    margin-top: 2rem;
    padding-top: 2rem;
    border-top: 1px solid #334155;
  }

  .drift-projects .label {
    display: block;
    font-size: 0.7rem;
    font-weight: 800;
    color: #64748b;
    margin-bottom: 1rem;
    text-transform: uppercase;
  }

  .p-list {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
  }

  .p-tag {
    background: #0f172a;
    padding: 0.4rem 0.8rem;
    border-radius: 8px;
    font-size: 0.8rem;
    font-weight: 700;
    color: #ef4444;
    border: 1px solid #ef444430;
  }

  /* Insights View */
  .insights-header {
    margin-bottom: 4rem;
  }

  .insights-header h2 {
    font-size: 2.5rem;
    font-weight: 900;
    margin-bottom: 1rem;
    background: linear-gradient(to right, #a855f7, #3b82f6);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
  }

  .insights-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(450px, 1fr));
    gap: 3rem;
  }

  .insight-card {
    background: #1e293b;
    padding: 2.5rem;
    border-radius: 24px;
    border: 1px solid #334155;
  }

  .insight-card.highlight {
    border-color: #a855f750;
    box-shadow: 0 10px 30px -10px rgba(168, 85, 247, 0.1);
  }

  .insight-card h3 {
    margin: 0;
    font-size: 1.5rem;
    font-weight: 800;
  }

  .insight-card .subtitle {
    color: #64748b;
    font-size: 0.9rem;
    margin-bottom: 2rem;
  }

  .stats-list {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }

  .stat-row {
    display: grid;
    grid-template-columns: 140px 1fr;
    align-items: center;
    gap: 1rem;
  }

  .t-name {
    font-family: monospace;
    font-size: 0.85rem;
    color: #cbd5e1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .t-count {
    display: flex;
    align-items: center;
    gap: 1rem;
  }

  .t-count .bar {
    height: 8px;
    background: #a855f7;
    border-radius: 9999px;
    transition: width 1s ease-out;
  }

  .t-count.adopted .bar {
    background: #10b981;
  }

  .t-count .num {
    font-weight: 800;
    font-size: 0.85rem;
    min-width: 20px;
  }

  .empty {
    color: #64748b;
    font-style: italic;
  }
</style>
