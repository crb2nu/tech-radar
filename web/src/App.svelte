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
    adopt: '#93c47d',
    trial: '#93d2c2',
    assess: '#fbdb84',
    hold: '#efafa9'
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
    return new Date(dateStr).toLocaleDateString();
  }
</script>

<main>
  {#if error}
    <p class="error">{error}</p>
  {:else if !radar}
    <div class="loading-state">
      <div class="spinner"></div>
      <p>Loading Tech Radar...</p>
    </div>
  {:else}
    <header>
      <h1>{radar.meta.name}</h1>
      
      <div class="tabs">
        <button class:active={activeTab === 'radar'} onclick={() => activeTab = 'radar'}>Technology Radar</button>
        <button class:active={activeTab === 'projects'} onclick={() => activeTab = 'projects'}>Project Catalog</button>
      </div>

      <div class="controls">
        <input 
          type="text" 
          placeholder="Search {activeTab === 'radar' ? 'technologies' : 'projects'}..." 
          bind:value={searchQuery}
          oninput={handleSearch}
          class="search-input"
        />
        
        {#if activeTab === 'radar'}
          <div class="filter-group">
            <label for="quadrant-filter">Quadrant:</label>
            <select id="quadrant-filter" bind:value={selectedQuadrant}>
              <option value="all">All Quadrants</option>
              {#each quadrants as q}
                <option value={q}>{q.toUpperCase()}</option>
              {/each}
            </select>
          </div>

          <div class="filter-group">
            <label for="ring-filter">Ring:</label>
            <select id="ring-filter" bind:value={selectedRing}>
              <option value="all">All Rings</option>
              {#each rings as r}
                <option value={r}>{r.toUpperCase()}</option>
              {/each}
            </select>
          </div>

          {#if allTags.length > 0}
            <div class="filter-group">
              <label for="tag-filter">Tag:</label>
              <select id="tag-filter" bind:value={selectedTag}>
                <option value="all">All Tags</option>
                {#each allTags as t}
                  <option value={t}>{t}</option>
                {/each}
              </select>
            </div>
          {/if}
        {/if}
      </div>
    </header>

    {#if activeTab === 'radar'}
      <div class="radar-view" in:fade>
        <div class="radar-container">
          <svg viewBox="0 0 600 600" width="600" height="600">
            <!-- Rings -->
            {#each rings.slice().reverse() as ring, i}
              {@const r = ringRadius[3 - i]}
              <circle cx="300" cy="300" r={r} fill={colors[ring as keyof typeof colors]} fill-opacity="0.1" class="radar-ring" stroke-width="2" />
            {/each}
            
            <!-- Quadrant dividers -->
            <line x1="300" y1="20" x2="300" y2="580" class="radar-divider" stroke-width="2" />
            <line x1="20" y1="300" x2="580" y2="300" class="radar-divider" stroke-width="2" />

            <!-- Quadrant Labels -->
            <text x="50" y="50" class="quad-label">LANGUAGES</text>
            <text x="550" y="50" text-anchor="end" class="quad-label">FRAMEWORKS</text>
            <text x="50" y="550" class="quad-label">TOOLS</text>
            <text x="550" y="550" text-anchor="end" class="quad-label">PLATFORMS</text>

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
                role="button"
                tabindex="0"
                aria-label={entry.name}
              >
                {#if entry.moved === 1}
                  <path d="M 0 -8 L 7 6 L -7 6 Z" fill={colors[entry.ring as keyof typeof colors]} stroke="#333" stroke-width="1" />
                {:else if entry.moved === -1}
                  <path d="M 0 8 L 7 -6 L -7 -6 Z" fill={colors[entry.ring as keyof typeof colors]} stroke="#333" stroke-width="1" />
                {:else}
                  <circle r="6" fill={colors[entry.ring as keyof typeof colors]} stroke="#333" stroke-width="1" />
                {/if}
                <text y="-12" text-anchor="middle" class="blip-text" class:visible={hoveredEntry === entry.name}>{entry.name}</text>
                <title>{entry.name} ({entry.ring}): {entry.description}</title>
              </g>
            {/each}
          </svg>
        </div>

        {#if filteredEntries.length === 0}
          <div class="empty-state">
            <p>No technologies match your filters.</p>
            <button onclick={() => { searchQuery = ''; selectedQuadrant = 'all'; selectedRing = 'all'; selectedTag = 'all'; }}>Reset Filters</button>
          </div>
        {/if}

        <div class="entries-list">
          {#each rings as ring}
            {@const entries = getEntriesByRing(ring)}
            {#if entries.length > 0}
              <section>
                <h2 style="color: {colors[ring as keyof typeof colors]}">{ring.toUpperCase()}</h2>
                <div class="entries-group" role="list">
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
                        <div>
                          {#if entry.project_ids && entry.project_ids.length > 0}
                            <span class="usage-tag">Used by {entry.project_ids.length}</span>
                          {/if}
                          {#each entry.tags || [] as tag}
                            <span class="tag">{tag}</span>
                          {/each}
                          <span class="quadrant-tag">{entry.quadrant}</span>
                        </div>
                      </div>
                      <p class="description">{entry.description}</p>
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
            <p>No projects found matching your search.</p>
          </div>
        {:else}
          <div class="project-grid">
            {#each filteredProjects as project}
              <div class="project-card">
                <div class="project-header">
                  <h3>{project.name}</h3>
                  <span class="scan-date">Last Scanned: {formatDate(project.last_scan)}</span>
                </div>
                {#if project.url}
                  <a href={project.url} class="project-url" target="_blank">{project.url}</a>
                {/if}
                <div class="project-techs">
                  {#each project.technologies || [] as tech}
                    <span class="tech-tag" title="Source: {tech.source}">
                      {tech.name} <small>{tech.version}</small>
                    </span>
                  {/each}
                </div>
              </div>
            {/each}
          </div>
        {/if}
      </div>
    {/if}
  {/if}

  {#if selectedEntry}
    <!-- ... same modal as before ... -->
    <div class="modal-backdrop" transition:fade onclick={() => selectedEntry = null}>
      <div class="modal-content" transition:fly={{ y: 20, duration: 300 }} onclick={(event) => event.stopPropagation()}>
        <button class="close-btn" onclick={() => selectedEntry = null}>&times;</button>
        
        <header class="modal-header">
          <div class="modal-title-row">
            <h1>{selectedEntry.name}</h1>
            <span class="ring-tag-large" style="background: {colors[selectedEntry.ring as keyof typeof colors]}20; color: {colors[selectedEntry.ring as keyof typeof colors]}">
              {selectedEntry.ring.toUpperCase()}
            </span>
          </div>
          <div class="modal-meta">
            <span class="quadrant-tag">{selectedEntry.quadrant}</span>
            {#each selectedEntry.tags || [] as tag}
              <span class="tag">{tag}</span>
            {/each}
          </div>
        </header>

        <div class="modal-body">
          <section class="description-section">
            <h3>About</h3>
            <p>{selectedEntry.description}</p>
            {#if selectedEntry.link}
              <a href={selectedEntry.link} target="_blank" rel="noopener noreferrer" class="external-link">
                Learn more ↗
              </a>
            {/if}
          </section>

          {#if selectedEntry.history && selectedEntry.history.length > 0}
            <section class="history-section">
              <h3>History</h3>
              <div class="history-timeline">
                {#each selectedEntry.history.slice().reverse() as h}
                  <div class="history-card">
                    <div class="history-header">
                      <span class="history-date">{formatDate(h.date)}</span>
                      <span class="ring-tag-small" style="background: {colors[h.ring as keyof typeof colors]}15; color: {colors[h.ring as keyof typeof colors]}">
                        {h.ring}
                      </span>
                      {#if h.author}
                        <span class="author">by {h.author}</span>
                      {/if}
                    </div>
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
  :root {
    --bg-primary: #f8fafc;
    --bg-secondary: #ffffff;
    --bg-card: #ffffff;
    --text-primary: #0f172a;
    --text-secondary: #475569;
    --text-muted: #94a3b8;
    --border-color: #e2e8f0;
    --accent-color: #3b82f6;
    --radar-grid: #e2e8f0;
    --radar-label: #64748b;
    --shadow: 0 10px 15px -3px rgba(0, 0, 0, 0.1), 0 4px 6px -4px rgba(0, 0, 0, 0.1);
    --shadow-sm: 0 1px 3px 0 rgba(0, 0, 0, 0.1);
  }

  @media (prefers-color-scheme: dark) {
    :root {
      --bg-primary: #0f172a;
      --bg-secondary: #1e293b;
      --bg-card: #1e293b;
      --text-primary: #f1f5f9;
      --text-secondary: #cbd5e1;
      --text-muted: #64748b;
      --border-color: #334155;
      --radar-grid: #334155;
      --radar-label: #94a3b8;
    }
  }

  main {
    font-family: 'Inter', system-ui, sans-serif;
    max-width: 1400px;
    margin: 0 auto;
    padding: 3rem 2rem;
    color: var(--text-primary);
    background-color: var(--bg-primary);
    min-height: 100vh;
  }

  header {
    text-align: center;
    margin-bottom: 4rem;
  }

  h1 {
    font-size: 3.5rem;
    font-weight: 900;
    letter-spacing: -0.05em;
    margin-bottom: 2rem;
    background: linear-gradient(to bottom right, var(--text-primary), var(--accent-color));
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
  }

  .tabs {
    display: flex;
    gap: 1rem;
    justify-content: center;
    margin-bottom: 2.5rem;
  }

  .tabs button {
    padding: 0.75rem 2rem;
    border-radius: 9999px;
    border: 2px solid var(--border-color);
    background: var(--bg-secondary);
    color: var(--text-secondary);
    font-weight: 700;
    cursor: pointer;
    transition: all 0.2s;
  }

  .tabs button.active {
    background: var(--accent-color);
    color: white;
    border-color: var(--accent-color);
    box-shadow: 0 4px 14px 0 rgba(59, 130, 246, 0.39);
  }

  .controls {
    display: flex;
    gap: 1.5rem;
    justify-content: center;
    align-items: center;
    flex-wrap: wrap;
    background: var(--bg-secondary);
    padding: 1.5rem;
    border-radius: 20px;
    box-shadow: var(--shadow-sm);
    border: 1px solid var(--border-color);
  }

  .search-input {
    padding: 0.75rem 1.5rem;
    border: 2px solid var(--border-color);
    background: var(--bg-primary);
    color: var(--text-primary);
    border-radius: 12px;
    min-width: 350px;
    font-size: 1rem;
    outline: none;
    transition: all 0.2s;
  }

  .search-input:focus {
    border-color: var(--accent-color);
    box-shadow: 0 0 0 4px rgba(59, 130, 246, 0.1);
  }

  .filter-group {
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }

  .filter-group label {
    font-size: 0.75rem;
    font-weight: 800;
    color: var(--text-muted);
    text-transform: uppercase;
    letter-spacing: 0.1em;
  }

  select {
    padding: 0.6rem 2.5rem 0.6rem 1rem;
    border: 2px solid var(--border-color);
    border-radius: 10px;
    background-color: var(--bg-primary);
    color: var(--text-primary);
    font-size: 0.875rem;
    font-weight: 600;
    cursor: pointer;
    appearance: none;
    background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' fill='none' viewBox='0 0 24 24' stroke='%2394a3b8'%3E%3Cpath stroke-linecap='round' stroke-linejoin='round' stroke-width='2' d='M19 9l-7 7-7-7'%3E%3C/path%3E%3C/svg%3E");
    background-repeat: no-repeat;
    background-position: right 0.75rem center;
    background-size: 1rem;
  }

  .radar-container {
    display: flex;
    justify-content: center;
    margin-bottom: 5rem;
    background: var(--bg-secondary);
    border-radius: 40px;
    padding: 4rem;
    box-shadow: var(--shadow-sm);
    border: 1px solid var(--border-color);
  }

  .blip circle, .blip path {
    cursor: pointer;
    transition: all 0.3s cubic-bezier(0.34, 1.56, 0.64, 1);
  }

  .blip.hovered circle, .blip.hovered path {
    stroke: var(--text-primary);
    stroke-width: 2.5;
    filter: drop-shadow(0 0 8px rgba(0,0,0,0.2));
  }

  .blip-text {
    font-size: 11px;
    fill: var(--text-primary);
    font-weight: 800;
    pointer-events: none;
    opacity: 0;
    transition: opacity 0.2s, transform 0.2s;
    transform: translateY(4px);
  }

  .blip-text.visible {
    opacity: 1;
    transform: translateY(0);
  }

  .project-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(400px, 1fr));
    gap: 2rem;
  }

  .project-card {
    background: var(--bg-card);
    border-radius: 20px;
    padding: 2rem;
    border: 1px solid var(--border-color);
    box-shadow: var(--shadow-sm);
    transition: all 0.3s;
  }

  .project-card:hover {
    transform: translateY(-5px);
    box-shadow: var(--shadow);
    border-color: var(--accent-color);
  }

  .project-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    margin-bottom: 1rem;
  }

  .project-header h3 {
    margin: 0;
    font-size: 1.5rem;
    font-weight: 800;
    color: var(--text-primary);
  }

  .scan-date {
    font-size: 0.75rem;
    color: var(--text-muted);
    font-weight: 600;
  }

  .project-techs {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
    margin-top: 1.5rem;
  }

  .tech-tag {
    background: var(--bg-primary);
    padding: 6px 12px;
    border-radius: 8px;
    font-size: 0.8rem;
    font-weight: 600;
    border: 1px solid var(--border-color);
    color: var(--text-secondary);
  }

  .tech-tag small {
    color: var(--text-muted);
    margin-left: 4px;
  }

  .empty-state {
    text-align: center;
    padding: 5rem;
    background: var(--bg-secondary);
    border-radius: 24px;
    border: 2px dashed var(--border-color);
  }

  .empty-state p {
    color: var(--text-muted);
    font-size: 1.25rem;
    margin-bottom: 1.5rem;
  }

  .empty-state button {
    padding: 0.5rem 1.5rem;
    background: var(--accent-color);
    color: white;
    border: none;
    border-radius: 8px;
    font-weight: 700;
    cursor: pointer;
  }

  .loading-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    min-height: 60vh;
  }

  .spinner {
    width: 50px;
    height: 50px;
    border: 5px solid var(--border-color);
    border-top-color: var(--accent-color);
    border-radius: 50%;
    animation: spin 1s linear infinite;
    margin-bottom: 1.5rem;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  .modal-backdrop {
    position: fixed;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    background: rgba(15, 23, 42, 0.8);
    display: flex;
    justify-content: center;
    align-items: center;
    z-index: 1000;
    backdrop-filter: blur(8px);
    padding: 2rem;
  }

  .modal-content {
    background: var(--bg-card);
    width: 100%;
    max-width: 850px;
    max-height: 90vh;
    border-radius: 32px;
    position: relative;
    padding: 4rem;
    overflow-y: auto;
    box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.5);
    border: 1px solid var(--border-color);
  }

  .close-btn {
    position: absolute;
    top: 2rem;
    right: 2rem;
    background: var(--bg-primary);
    border: 1px solid var(--border-color);
    width: 40px;
    height: 40px;
    border-radius: 50%;
    font-size: 1.5rem;
    color: var(--text-muted);
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.2s;
  }

  .close-btn:hover {
    background: #fee2e2;
    color: #ef4444;
    border-color: #fecaca;
  }

  .usage-tag {
    background: #dbeafe;
    color: #1e40af;
    padding: 4px 12px;
    border-radius: 9999px;
    font-size: 0.7rem;
    font-weight: 800;
    text-transform: uppercase;
    margin-right: 0.5rem;
  }

  .tag {
    background: var(--bg-primary);
    padding: 4px 12px;
    border-radius: 8px;
    font-size: 0.7rem;
    font-weight: 700;
    color: var(--text-secondary);
    border: 1px solid var(--border-color);
    margin-right: 0.5rem;
  }

  .quadrant-tag {
    background: var(--bg-primary);
    padding: 4px 12px;
    border-radius: 9999px;
    font-size: 0.7rem;
    font-weight: 800;
    color: var(--accent-color);
    border: 1px solid rgba(59, 130, 246, 0.2);
    text-transform: uppercase;
  }

  .entries-list {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(350px, 1fr));
    gap: 3rem;
  }

  .entry-item {
    background: var(--bg-card);
    border-radius: 20px;
    padding: 1.5rem;
    border: 1px solid var(--border-color);
    transition: all 0.2s;
    cursor: pointer;
  }

  .entry-item:hover {
    border-color: var(--accent-color);
    box-shadow: var(--shadow);
    transform: translateY(-2px);
  }

  .entry-item strong {
    font-size: 1.35rem;
    font-weight: 800;
  }

  .description {
    color: var(--text-secondary);
    font-size: 0.95rem;
    line-height: 1.6;
    margin-top: 1rem;
  }

  .history-card {
    background: var(--bg-primary);
    border-radius: 16px;
    padding: 1.5rem;
    border: 1px solid var(--border-color);
  }

  .history-date {
    font-family: monospace;
    font-size: 0.9rem;
    color: var(--accent-color);
  }

  .author {
    color: var(--text-muted);
    font-size: 0.8rem;
    margin-left: 1rem;
  }
</style>
