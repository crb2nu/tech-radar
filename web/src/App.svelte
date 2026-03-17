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

  const ringRadius = [70, 140, 210, 280]; // Radii for the 4 rings

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

  async function handleSearch() {
    if (searchQuery.length < 2) return;
    try {
      const response = await fetch(`/api/v1/search?q=${encodeURIComponent(searchQuery)}`);
      if (response.ok) {
        const results = await response.json();
        // Integration point for Meilisearch results
        console.log('Search results:', results);
      }
    } catch (e) {
      console.error('Search failed', e);
    }
  }

  onMount(async () => {
    // @ts-ignore
    if (window.RADAR_DATA) {
      // @ts-ignore
      radar = window.RADAR_DATA;
    } else {
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
    }

    try {
      const pResponse = await fetch('/api/v1/projects');
      if (pResponse.ok) {
        projects = await pResponse.json();
      }
    } catch (e) {
      console.error('Failed to fetch projects', e);
    }
  });

  function getEntriesByRing(ring: string) {
    return filteredEntries.filter(e => e.ring === ring);
  }

  // Blip placement logic
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
    <div class="error-container" in:fade>
      <div class="error-icon">⚠️</div>
      <p class="error">{error}</p>
      <button onclick={() => window.location.reload()}>Retry Connection</button>
    </div>
  {:else if !radar}
    <div class="loading-state">
      <div class="spinner"></div>
      <p>Loading Technology Ecosystem...</p>
    </div>
  {:else}
    <header>
      <div class="header-content">
        <h1>{radar.meta.name}</h1>
        <div class="tabs">
          <button class:active={activeTab === 'radar'} onclick={() => activeTab = 'radar'}>
            <span class="icon">🎯</span> Technology Radar
          </button>
          <button class:active={activeTab === 'projects'} onclick={() => activeTab = 'projects'}>
            <span class="icon">📦</span> Project Catalog
          </button>
        </div>
      </div>

      <div class="controls-wrapper">
        <div class="controls">
          <div class="search-box">
            <span class="search-icon">🔍</span>
            <input 
              type="text" 
              placeholder="Search {activeTab === 'radar' ? 'technologies' : 'projects'}..." 
              bind:value={searchQuery}
              oninput={handleSearch}
              class="search-input"
            />
          </div>
          
          {#if activeTab === 'radar'}
            <div class="filters">
              <div class="filter-group">
                <label for="quadrant-filter">Quadrant</label>
                <select id="quadrant-filter" bind:value={selectedQuadrant}>
                  <option value="all">All Sections</option>
                  {#each quadrants as q}
                    <option value={q}>{q.toUpperCase()}</option>
                  {/each}
                </select>
              </div>

              <div class="filter-group">
                <label for="ring-filter">Ring</label>
                <select id="ring-filter" bind:value={selectedRing}>
                  <option value="all">All Rings</option>
                  {#each rings as r}
                    <option value={r}>{r.toUpperCase()}</option>
                  {/each}
                </select>
              </div>

              {#if allTags.length > 0}
                <div class="filter-group">
                  <label for="tag-filter">Tag</label>
                  <select id="tag-filter" bind:value={selectedTag}>
                    <option value="all">All Tags</option>
                    {#each allTags as t}
                      <option value={t}>{t}</option>
                    {/each}
                  </select>
                </div>
              {/if}
            </div>
          {/if}
        </div>
      </div>
    </header>

    <div class="content-area">
      {#if activeTab === 'radar'}
        <div class="radar-view" in:fade>
          <div class="radar-container">
            <svg viewBox="0 0 600 600" width="600" height="600">
              <!-- Rings -->
              {#each rings.slice().reverse() as ring, i}
                {@const r = ringRadius[3 - i]}
                <circle cx="300" cy="300" r={r} fill={colors[ring as keyof typeof colors]} fill-opacity="0.05" class="radar-ring" stroke-width="1" />
              {/each}
              
              <!-- Quadrant dividers -->
              <line x1="300" y1="20" x2="300" y2="580" class="radar-divider" stroke-width="1" stroke-dasharray="4 4" />
              <line x1="20" y1="300" x2="580" y2="300" class="radar-divider" stroke-width="1" stroke-dasharray="4 4" />

              <!-- Quadrant Labels -->
              <text x="40" y="40" class="quad-label">LANGUAGES</text>
              <text x="560" y="40" text-anchor="end" class="quad-label">FRAMEWORKS</text>
              <text x="40" y="560" class="quad-label">TOOLS</text>
              <text x="560" y="560" text-anchor="end" class="quad-label">PLATFORMS</text>

              <!-- Blips -->
              {#each filteredEntries as entry, i}
                {@const pos = getBlipPosition(entry, i)}
                <g 
                  class="blip" 
                  class:hovered={hoveredEntry === entry.name}
                  transform="translate({pos.x}, {pos.y})"
                  onmouseenter={() => hoveredEntry = entry.name}
                  onmouseleave={() => hoveredEntry = null}
                  onclick={() => selectedEntry = entry}
                  onkeydown={(e) => e.key === 'Enter' && (selectedEntry = entry)}
                  role="button"
                  tabindex="0"
                  aria-label={entry.name}
                >
                  {#if entry.moved === 1}
                    <path d="M 0 -8 L 7 6 L -7 6 Z" fill={colors[entry.ring as keyof typeof colors]} stroke="rgba(0,0,0,0.3)" stroke-width="1" />
                  {:else if entry.moved === -1}
                    <path d="M 0 8 L 7 -6 L -7 -6 Z" fill={colors[entry.ring as keyof typeof colors]} stroke="rgba(0,0,0,0.3)" stroke-width="1" />
                  {:else}
                    <circle r="6" fill={colors[entry.ring as keyof typeof colors]} stroke="rgba(0,0,0,0.3)" stroke-width="1" />
                    {/if}
                    {#if entry.project_ids && entry.project_ids.length > 0}
                    <circle cx="8" cy="-8" r="5" fill="#3b82f6" stroke="white" stroke-width="1" />
                    <text x="8" y="-6" text-anchor="middle" font-size="6" fill="white" font-weight="900">{entry.project_ids.length}</text>
                    {/if}
                    <text y="-18" text-anchor="middle" class="blip-text" class:visible={hoveredEntry === entry.name}>{entry.name}</text>

                  <title>{entry.name} ({entry.ring}): {entry.description}</title>
                </g>
              {/each}
            </svg>
          </div>

          {#if filteredEntries.length === 0}
            <div class="empty-state">
              <div class="empty-icon">🔭</div>
              <p>No technologies match your current filters.</p>
              <button onclick={() => { searchQuery = ''; selectedQuadrant = 'all'; selectedRing = 'all'; selectedTag = 'all'; }}>Clear All Filters</button>
            </div>
          {/if}

          <div class="entries-list">
            {#each rings as ring}
              {@const entries = getEntriesByRing(ring)}
              {#if entries.length > 0}
                <section class="ring-section">
                  <div class="ring-header">
                    <span class="ring-dot" style="background: {colors[ring as keyof typeof colors]}"></span>
                    <h2 style="color: {colors[ring as keyof typeof colors]}">{ring.toUpperCase()}</h2>
                    <span class="count-badge">{entries.length}</span>
                  </div>
                  <div class="entries-grid">
                    {#each entries as entry}
                      <div 
                        class="entry-item"
                        class:active={hoveredEntry === entry.name}
                        onmouseenter={() => hoveredEntry = entry.name}
                        onmouseleave={() => hoveredEntry = null}
                        onclick={() => selectedEntry = entry}
                        onkeydown={(e) => e.key === 'Enter' && (selectedEntry = entry)}
                        role="listitem"
                        tabindex="0"
                      >
                        <div class="entry-header">
                          <strong>{entry.name}</strong>
                          {#if entry.project_ids && entry.project_ids.length > 0}
                            <span class="usage-count">Used by {entry.project_ids.length}</span>
                          {/if}
                        </div>
                        <p class="description">{entry.description}</p>
                        <div class="entry-footer">
                          <span class="quadrant-badge">{entry.quadrant}</span>
                          {#each (entry.tags || []).slice(0, 2) as tag}
                            <span class="tag-badge">{tag}</span>
                          {/each}
                        </div>
                      </div>
                    {/each}
                  </div>
                </section>
              {/if}
            {/each}
          </div>
        </div>
      {:else}
        <div class="projects-view" in:fade>
          {#if filteredProjects.length === 0}
            <div class="empty-state">
              <div class="empty-icon">📁</div>
              <p>No projects found matching your search.</p>
            </div>
          {:else}
            <div class="project-grid">
              {#each filteredProjects as project}
                <div class="project-card">
                  <div class="project-card-header">
                    <div class="project-info">
                      <h3>{project.name}</h3>
                      {#if project.url}
                        <a href={project.url} class="project-link" target="_blank">View Repository ↗</a>
                      {/if}
                    </div>
                    <div class="scan-meta">
                      <span class="scan-label">LAST SCAN</span>
                      <span class="scan-value">{formatDate(project.last_scan)}</span>
                    </div>
                  </div>
                  
                  <div class="project-stats">
                    <div class="stat">
                      <span class="stat-value">{project.technologies?.length || 0}</span>
                      <span class="stat-label">Dependencies</span>
                    </div>
                  </div>

                  <div class="project-tech-cloud">
                    {#if project.technologies && project.technologies.length > 0}
                      {#each project.technologies as tech}
                        <div class="tech-item" title="Source: {tech.source}">
                          <span class="tech-name">{tech.name}</span>
                          {#if tech.version}
                            <span class="tech-version">{tech.version}</span>
                          {/if}
                        </div>
                      {/each}
                    {:else}
                      <p class="no-techs">No technologies discovered yet.</p>
                    {/if}
                  </div>
                </div>
              {/each}
            </div>
          {/if}
        </div>
      {/if}
    </div>
  {/if}

  {#if selectedEntry}
    <div class="modal-backdrop" transition:fade onclick={() => selectedEntry = null}>
      <div class="modal-content" transition:fly={{ y: 20, duration: 300 }} onclick={(event) => event.stopPropagation()}>
        <button class="close-btn" onclick={() => selectedEntry = null}>&times;</button>
        
        <header class="modal-header">
          <div class="modal-title-row">
            <h1>{selectedEntry.name}</h1>
            <span class="ring-badge-large" style="background: {colors[selectedEntry.ring as keyof typeof colors]}20; color: {colors[selectedEntry.ring as keyof typeof colors]}">
              {selectedEntry.ring.toUpperCase()}
            </span>
          </div>
          <div class="modal-meta">
            <span class="quadrant-badge">{selectedEntry.quadrant}</span>
            {#each selectedEntry.tags || [] as tag}
              <span class="tag-badge">{tag}</span>
            {/each}
          </div>
        </header>

        <div class="modal-body">
          <section class="description-section">
            <h3>Assessment</h3>
            <p>{selectedEntry.description}</p>
            {#if selectedEntry.link}
              <a href={selectedEntry.link} target="_blank" rel="noopener noreferrer" class="external-link-btn">
                Technical Documentation ↗
              </a>
            {/if}
          </section>

          {#if selectedEntry.project_ids && selectedEntry.project_ids.length > 0}
            <section class="usage-section">
              <h3>Internal Adoption</h3>
              <div class="project-tags-large">
                {#each selectedEntry.project_ids as pid}
                  <span class="project-usage-tag">{pid}</span>
                {/each}
              </div>
            </section>
          {/if}

          {#if selectedEntry.history && selectedEntry.history.length > 0}
            <section class="history-section">
              <h3>Governance History</h3>
              <div class="history-timeline">
                {#each selectedEntry.history.slice().reverse() as h}
                  <div class="history-card" style="border-left-color: {colors[h.ring as keyof typeof colors]}">
                    <div class="history-card-header">
                      <span class="history-date">{formatDate(h.date)}</span>
                      <span class="ring-status" style="color: {colors[h.ring as keyof typeof colors]}">
                        {h.ring.toUpperCase()}
                      </span>
                    </div>
                    {#if h.author}
                      <div class="history-author">Decided by <strong>{h.author}</strong></div>
                    {/if}
                    {#if h.reason}
                      <p class="history-reason">{h.reason}</p>
                    {/if}
                  </div>
                {/each}
              </div>
            </section>
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
    min-height: 100vh;
  }

  :root {
    --bg-primary: #0f172a;
    --bg-secondary: #1e293b;
    --bg-card: #1e293b;
    --text-primary: #f1f5f9;
    --text-secondary: #cbd5e1;
    --text-muted: #64748b;
    --border-color: #334155;
    --accent-color: #3b82f6;
    --radar-grid: #334155;
    --radar-label: #94a3b8;
    --shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.3), 0 10px 10px -5px rgba(0, 0, 0, 0.2);
  }

  main {
    font-family: 'Inter', system-ui, -apple-system, sans-serif;
    color: var(--text-primary);
    background-color: var(--bg-primary);
    min-height: 100vh;
    padding: 0;
    display: flex;
    flex-direction: column;
  }

  header {
    background: #0b1120;
    padding: 4rem 2rem 2rem;
    border-bottom: 1px solid var(--border-color);
  }

  .header-content {
    max-width: 1200px;
    margin: 0 auto;
    text-align: center;
  }

  h1 {
    font-size: 4rem;
    font-weight: 900;
    letter-spacing: -0.06em;
    margin-bottom: 2.5rem;
    color: white;
  }

  .tabs {
    display: flex;
    gap: 0.5rem;
    justify-content: center;
    background: rgba(255,255,255,0.05);
    padding: 0.4rem;
    border-radius: 9999px;
    width: fit-content;
    margin: 0 auto 3rem;
    border: 1px solid rgba(255,255,255,0.1);
  }

  .tabs button {
    padding: 0.8rem 2rem;
    border-radius: 9999px;
    border: none;
    background: transparent;
    color: var(--text-muted);
    font-weight: 700;
    cursor: pointer;
    transition: all 0.2s;
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .tabs button.active {
    background: var(--accent-color);
    color: white;
    box-shadow: 0 4px 12px rgba(59, 130, 246, 0.4);
  }

  .controls-wrapper {
    max-width: 1200px;
    margin: 0 auto;
    width: 100%;
  }

  .controls {
    display: flex;
    flex-direction: column;
    gap: 2rem;
    background: var(--bg-secondary);
    padding: 2rem;
    border-radius: 24px;
    border: 1px solid var(--border-color);
    box-shadow: var(--shadow);
  }

  .search-box {
    position: relative;
    width: 100%;
  }

  .search-icon {
    position: absolute;
    left: 1.25rem;
    top: 50%;
    transform: translateY(-50%);
    color: var(--text-muted);
  }

  .search-input {
    width: 100%;
    padding: 1rem 1rem 1rem 3.5rem;
    background: #0f172a;
    border: 2px solid var(--border-color);
    border-radius: 16px;
    color: white;
    font-size: 1.1rem;
    outline: none;
    transition: all 0.2s;
    box-sizing: border-box;
  }

  .search-input:focus {
    border-color: var(--accent-color);
    background: #1e293b;
  }

  .filters {
    display: flex;
    gap: 2rem;
    flex-wrap: wrap;
  }

  .filter-group {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    flex: 1;
    min-width: 200px;
  }

  .filter-group label {
    font-size: 0.7rem;
    font-weight: 800;
    color: var(--text-muted);
    text-transform: uppercase;
    letter-spacing: 0.1em;
  }

  select {
    padding: 0.8rem 1rem;
    background: #0f172a;
    border: 1px solid var(--border-color);
    border-radius: 12px;
    color: white;
    font-weight: 600;
    cursor: pointer;
  }

  .content-area {
    flex: 1;
    max-width: 1400px;
    margin: 0 auto;
    width: 100%;
    padding: 4rem 2rem;
  }

  .radar-container {
    display: flex;
    justify-content: center;
    margin-bottom: 6rem;
    background: #0b1120;
    padding: 4rem;
    border-radius: 48px;
    border: 1px solid var(--border-color);
  }

  .quad-label {
    font-size: 12px;
    fill: var(--radar-label);
    font-weight: 900;
    letter-spacing: 0.2em;
  }

  .blip-text {
    font-size: 10px;
    fill: white;
    font-weight: 800;
    opacity: 0;
    pointer-events: none;
  }

  .blip-text.visible {
    opacity: 1;
  }

  .entries-list {
    display: flex;
    flex-direction: column;
    gap: 5rem;
  }

  .ring-header {
    display: flex;
    align-items: center;
    gap: 1rem;
    margin-bottom: 2.5rem;
    border-bottom: 1px solid var(--border-color);
    padding-bottom: 1rem;
  }

  .ring-dot {
    width: 12px;
    height: 12px;
    border-radius: 50%;
  }

  .ring-header h2 {
    margin: 0;
    font-size: 2rem;
    font-weight: 900;
    letter-spacing: -0.02em;
  }

  .count-badge {
    background: rgba(255,255,255,0.1);
    padding: 0.2rem 0.8rem;
    border-radius: 9999px;
    font-size: 0.9rem;
    font-weight: 700;
    color: var(--text-muted);
  }

  .entries-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
    gap: 1.5rem;
  }

  .entry-item {
    background: var(--bg-secondary);
    padding: 1.5rem;
    border-radius: 20px;
    border: 1px solid var(--border-color);
    cursor: pointer;
    transition: all 0.2s;
  }

  .entry-item:hover, .entry-item.active {
    border-color: var(--accent-color);
    transform: translateY(-4px);
    box-shadow: var(--shadow);
  }

  .entry-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    margin-bottom: 1rem;
  }

  .entry-item strong {
    font-size: 1.2rem;
    font-weight: 800;
  }

  .usage-count {
    font-size: 0.7rem;
    font-weight: 800;
    background: #3b82f620;
    color: #60a5fa;
    padding: 0.2rem 0.6rem;
    border-radius: 9999px;
  }

  .description {
    font-size: 0.9rem;
    color: var(--text-secondary);
    line-height: 1.6;
    margin-bottom: 1.5rem;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }

  .entry-footer {
    display: flex;
    gap: 0.5rem;
  }

  .quadrant-badge {
    font-size: 0.65rem;
    font-weight: 800;
    text-transform: uppercase;
    color: var(--accent-color);
    background: #3b82f615;
    padding: 0.2rem 0.6rem;
    border-radius: 4px;
  }

  .tag-badge {
    font-size: 0.65rem;
    font-weight: 700;
    color: var(--text-muted);
    border: 1px solid var(--border-color);
    padding: 0.2rem 0.6rem;
    border-radius: 4px;
  }

  .project-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(450px, 1fr));
    gap: 2rem;
  }

  .project-card {
    background: var(--bg-secondary);
    border-radius: 24px;
    padding: 2.5rem;
    border: 1px solid var(--border-color);
    box-shadow: var(--shadow);
  }

  .project-card-header {
    display: flex;
    justify-content: space-between;
    margin-bottom: 2rem;
  }

  .project-card h3 {
    margin: 0 0 0.5rem 0;
    font-size: 1.75rem;
    font-weight: 900;
  }

  .project-link {
    color: var(--accent-color);
    font-size: 0.8rem;
    font-weight: 700;
    text-decoration: none;
  }

  .scan-meta {
    text-align: right;
  }

  .scan-label {
    display: block;
    font-size: 0.6rem;
    font-weight: 900;
    color: var(--text-muted);
    letter-spacing: 0.1em;
  }

  .scan-value {
    font-weight: 700;
    font-size: 0.9rem;
  }

  .project-stats {
    margin-bottom: 2rem;
    padding: 1rem;
    background: #0f172a;
    border-radius: 12px;
    width: fit-content;
  }

  .stat {
    display: flex;
    flex-direction: column;
  }

  .stat-value {
    font-size: 1.5rem;
    font-weight: 900;
    color: var(--accent-color);
  }

  .stat-label {
    font-size: 0.65rem;
    font-weight: 800;
    color: var(--text-muted);
    text-transform: uppercase;
  }

  .project-tech-cloud {
    display: flex;
    flex-wrap: wrap;
    gap: 0.6rem;
  }

  .tech-item {
    background: #0f172a;
    border: 1px solid var(--border-color);
    padding: 0.4rem 0.8rem;
    border-radius: 8px;
    display: flex;
    gap: 0.5rem;
    align-items: center;
  }

  .tech-name {
    font-size: 0.8rem;
    font-weight: 700;
  }

  .tech-version {
    font-size: 0.7rem;
    color: var(--text-muted);
    font-family: monospace;
  }

  .no-techs {
    color: var(--text-muted);
    font-style: italic;
  }

  .empty-state {
    text-align: center;
    padding: 8rem 2rem;
    background: var(--bg-secondary);
    border-radius: 32px;
    border: 2px dashed var(--border-color);
  }

  .empty-icon {
    font-size: 4rem;
    margin-bottom: 1.5rem;
  }

  .loading-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 60vh;
  }

  .spinner {
    width: 60px;
    height: 60px;
    border: 6px solid rgba(255,255,255,0.1);
    border-top-color: var(--accent-color);
    border-radius: 50%;
    animation: spin 1s linear infinite;
    margin-bottom: 2rem;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  .modal-backdrop {
    position: fixed;
    top: 0;
    left: 0;
    width: 100vw;
    height: 100vh;
    background: rgba(15, 23, 42, 0.9);
    backdrop-filter: blur(12px);
    display: flex;
    justify-content: center;
    align-items: center;
    z-index: 2000;
  }

  .modal-content {
    background: #1e293b;
    width: 90%;
    max-width: 900px;
    max-height: 85vh;
    border-radius: 32px;
    padding: 4rem;
    position: relative;
    overflow-y: auto;
    border: 1px solid var(--border-color);
    box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.5);
  }

  .close-btn {
    position: absolute;
    top: 2rem;
    right: 2rem;
    width: 48px;
    height: 48px;
    border-radius: 50%;
    border: 1px solid var(--border-color);
    background: #0f172a;
    color: white;
    font-size: 1.5rem;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .ring-badge-large {
    padding: 0.5rem 1.5rem;
    border-radius: 9999px;
    font-weight: 900;
    letter-spacing: 0.05em;
  }

  .project-usage-tag {
    background: #3b82f615;
    color: #60a5fa;
    padding: 0.5rem 1rem;
    border-radius: 12px;
    font-size: 0.85rem;
    font-weight: 700;
    border: 1px solid rgba(59, 130, 246, 0.2);
  }

  .project-tags-large {
    display: flex;
    flex-wrap: wrap;
    gap: 0.75rem;
  }

  .history-timeline {
    display: flex;
    flex-direction: column;
    gap: 1.5rem;
  }

  .history-card {
    background: #0f172a;
    padding: 2rem;
    border-radius: 20px;
    border-left: 6px solid transparent;
  }

  .history-card-header {
    display: flex;
    justify-content: space-between;
    margin-bottom: 1rem;
  }

  .history-status {
    font-weight: 900;
    font-size: 0.8rem;
    letter-spacing: 0.1em;
  }

  .history-author {
    font-size: 0.85rem;
    margin-bottom: 1rem;
    color: var(--text-secondary);
  }

  .external-link-btn {
    display: inline-block;
    background: var(--accent-color);
    color: white;
    padding: 1rem 2rem;
    border-radius: 12px;
    text-decoration: none;
    font-weight: 700;
    margin-top: 2rem;
  }
</style>
