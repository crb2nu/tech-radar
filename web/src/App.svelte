<script lang="ts">
  import { onMount } from 'svelte';
  import { fade, fly } from 'svelte/transition';

  interface HistoryEntry {
    date: string;
    ring: string;
    reason: string;
  }

  interface Entry {
    name: string;
    quadrant: string;
    ring: string;
    description: string;
    moved: number; // -1, 0, 1
    tags?: string[];
    history?: HistoryEntry[];
  }

  interface Radar {
    meta: {
      name: string;
    };
    entries: Entry[];
  }

  let radar: Radar | null = $state(null);
  let error: string | null = $state(null);
  let searchQuery = $state('');
  let selectedQuadrant = $state('all');
  let selectedRing = $state('all');
  let selectedTag = $state('all');
  let hoveredEntry: string | null = $state(null);
  let selectedEntry: Entry | null = $state(null);

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

  onMount(async () => {
    // @ts-ignore
    if (window.RADAR_DATA) {
      // @ts-ignore
      radar = window.RADAR_DATA;
      return;
    }
    try {
      const response = await fetch('/api/radar');
      if (!response.ok) {
        throw new Error('Failed to fetch radar data');
      }
      radar = await response.json();
    } catch (e: any) {
      error = e.message;
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
    <p>Loading...</p>
  {:else}
    <header>
      <h1>{radar.meta.name}</h1>
      
      <div class="controls">
        <input 
          type="text" 
          placeholder="Search technologies..." 
          bind:value={searchQuery}
          class="search-input"
        />
        
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
      </div>
    </header>

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
            role="button"
            tabindex="0"
            aria-label={entry.name}
          >
            {#if entry.moved === 1}
              <!-- Triangle pointing towards center (approximate) -->
              <path d="M 0 -8 L 7 6 L -7 6 Z" fill={colors[entry.ring as keyof typeof colors]} stroke="#333" stroke-width="1" />
            {:else if entry.moved === -1}
              <!-- Triangle pointing away from center (approximate) -->
              <path d="M 0 8 L 7 -6 L -7 -6 Z" fill={colors[entry.ring as keyof typeof colors]} stroke="#333" stroke-width="1" />
            {:else}
              <circle r="6" fill={colors[entry.ring as keyof typeof colors]} stroke="#333" stroke-width="1" />
            {/if}
            <text y="-12" text-anchor="middle" class="blip-text">{entry.name}</text>
            <title>{entry.name} ({entry.ring}): {entry.description}</title>
          </g>
        {/each}
      </svg>
    </div>

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
                      {#each entry.tags || [] as tag}
                        <span class="tag">{tag}</span>
                      {/each}
                      <span class="quadrant-tag">{entry.quadrant}</span>
                    </div>
                  </div>
                  <p class="description">{entry.description}</p>
                  
                  {#if entry.history && entry.history.length > 1}
                    <div class="history">
                      <h4>History</h4>
                      <ul>
                        {#each entry.history.slice().reverse() as h}
                          <li>
                            <span class="date">{formatDate(h.date)}</span>: 
                            <span class="ring-tag" style="background: {colors[h.ring as keyof typeof colors]}15; color: {colors[h.ring as keyof typeof colors]}">{h.ring}</span>
                            {#if h.reason}
                              <span class="reason">— {h.reason}</span>
                            {/if}
                          </li>
                        {/each}
                      </ul>
                    </div>
                  {/if}
                </div>
              {/each}
            </div>
          </section>
        {/if}
      {/each}
    </div>
  {/if}

  {#if selectedEntry}
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
  /* ... previous styles ... */

  .modal-backdrop {
    position: fixed;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    background: rgba(0, 0, 0, 0.6);
    display: flex;
    justify-content: center;
    align-items: center;
    z-index: 1000;
    backdrop-filter: blur(4px);
    padding: 2rem;
  }

  .modal-content {
    background: var(--bg-card);
    width: 100%;
    max-width: 800px;
    max-height: 90vh;
    border-radius: 24px;
    position: relative;
    padding: 3rem;
    overflow-y: auto;
    box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.5);
    border: 1px solid var(--border-color);
  }

  .close-btn {
    position: absolute;
    top: 1.5rem;
    right: 1.5rem;
    background: none;
    border: none;
    font-size: 2rem;
    color: var(--text-muted);
    cursor: pointer;
    line-height: 1;
    padding: 0.5rem;
    transition: color 0.2s;
  }

  .close-btn:hover {
    color: var(--text-primary);
  }

  .modal-header {
    margin-bottom: 2.5rem;
    text-align: left;
  }

  .modal-title-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 1rem;
    flex-wrap: wrap;
    gap: 1rem;
  }

  .modal-title-row h1 {
    margin-bottom: 0;
    text-align: left;
  }

  .ring-tag-large {
    font-size: 1rem;
    font-weight: 800;
    padding: 8px 20px;
    border-radius: 9999px;
    letter-spacing: 0.05em;
  }

  .modal-meta {
    display: flex;
    gap: 0.75rem;
    flex-wrap: wrap;
  }

  .modal-body {
    display: flex;
    flex-direction: column;
    gap: 3rem;
  }

  h3 {
    font-size: 0.875rem;
    text-transform: uppercase;
    color: var(--text-muted);
    letter-spacing: 0.15em;
    margin-bottom: 1.25rem;
    font-weight: 800;
  }

  .description-section p {
    font-size: 1.15rem;
    line-height: 1.7;
    color: var(--text-secondary);
  }

  .external-link {
    display: inline-block;
    margin-top: 1.5rem;
    color: #4299e1;
    text-decoration: none;
    font-weight: 600;
    transition: color 0.2s;
  }

  .external-link:hover {
    color: #3182ce;
    text-decoration: underline;
  }

  .history-timeline {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }

  .history-card {
    padding: 1.25rem;
    background: var(--bg-secondary);
    border-radius: 12px;
    border-left: 4px solid var(--border-color);
  }

  .history-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 0.5rem;
  }

  .history-date {
    font-weight: 700;
    font-size: 0.9rem;
    color: var(--text-primary);
  }

  .history-reason {
    font-size: 0.95rem;
    color: var(--text-secondary);
    margin: 0;
  }

  .ring-tag-small {
    font-size: 0.75rem;
    font-weight: 800;
    padding: 2px 10px;
    border-radius: 6px;
    text-transform: uppercase;
  }

  :root {
    --bg-primary: #ffffff;
    --bg-secondary: #f7fafc;
    --bg-card: #ffffff;
    --text-primary: #1a202c;
    --text-secondary: #4a5568;
    --text-muted: #718096;
    --border-color: #edf2f7;
    --radar-grid: #ffffff;
    --radar-label: #a0aec0;
    --shadow: 0 4px 15px rgba(0, 0, 0, 0.05);
    --shadow-inset: inset 0 2px 10px rgba(0,0,0,0.03);
  }

  @media (prefers-color-scheme: dark) {
    :root {
      --bg-primary: #1a202c;
      --bg-secondary: #171923;
      --bg-card: #2d3748;
      --text-primary: #f7fafc;
      --text-secondary: #e2e8f0;
      --text-muted: #a0aec0;
      --border-color: #4a5568;
      --radar-grid: #2d3748;
      --radar-label: #718096;
      --shadow: 0 4px 15px rgba(0, 0, 0, 0.3);
      --shadow-inset: inset 0 2px 10px rgba(0,0,0,0.2);
    }
  }

  main {
    font-family: 'Inter', system-ui, -apple-system, sans-serif;
    max-width: 1400px;
    margin: 0 auto;
    padding: 2rem;
    color: var(--text-primary);
    background-color: var(--bg-primary);
    min-height: 100vh;
    transition: background-color 0.3s, color 0.3s;
  }

  header {
    text-align: center;
    margin-bottom: 3rem;
  }

  h1 {
    font-weight: 800;
    font-size: 3rem;
    letter-spacing: -0.04em;
    margin-bottom: 2rem;
    background: linear-gradient(135deg, var(--text-primary) 0%, var(--text-secondary) 100%);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
  }

  .controls {
    display: flex;
    gap: 1.5rem;
    justify-content: center;
    align-items: center;
    flex-wrap: wrap;
    background: var(--bg-card);
    padding: 1rem 2rem;
    border-radius: 16px;
    box-shadow: var(--shadow);
    border: 1px solid var(--border-color);
  }

  .search-input {
    padding: 0.75rem 1.25rem;
    border: 2px solid var(--border-color);
    background: var(--bg-primary);
    color: var(--text-primary);
    border-radius: 10px;
    min-width: 300px;
    font-size: 1rem;
    outline: none;
    transition: all 0.2s;
  }

  .search-input:focus {
    border-color: #4299e1;
    box-shadow: 0 0 0 3px rgba(66, 153, 225, 0.15);
  }

  .filter-group {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .filter-group label {
    font-size: 0.875rem;
    font-weight: 700;
    color: var(--text-muted);
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  select {
    padding: 0.6rem 2.5rem 0.6rem 1rem;
    border: 2px solid var(--border-color);
    border-radius: 10px;
    background-color: var(--bg-primary);
    color: var(--text-primary);
    font-size: 0.875rem;
    cursor: pointer;
    font-weight: 600;
    appearance: none;
    background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' fill='none' viewBox='0 0 24 24' stroke='%23718096'%3E%3Cpath stroke-linecap='round' stroke-linejoin='round' stroke-width='2' d='M19 9l-7 7-7-7'%3E%3C/path%3E%3C/svg%3E");
    background-repeat: no-repeat;
    background-position: right 0.75rem center;
    background-size: 1rem;
  }

  .radar-container {
    display: flex;
    justify-content: center;
    margin-bottom: 4rem;
    background: var(--bg-secondary);
    border-radius: 32px;
    padding: 3rem;
    box-shadow: var(--shadow-inset);
  }

  .quad-label {
    font-size: 14px;
    fill: var(--radar-label);
    font-weight: 900;
    pointer-events: none;
    letter-spacing: 0.15em;
  }

  .radar-ring {
    stroke: var(--radar-grid);
  }

  .radar-divider {
    stroke: var(--radar-grid);
  }

  .blip path, .blip circle {
    cursor: pointer;
    transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
  }

  .blip.hovered circle, .blip.hovered path {
    stroke: var(--text-primary);
    stroke-width: 2;
    transform: scale(1.2);
  }

  .blip-text {
    font-size: 10px;
    fill: var(--text-primary);
    font-weight: 700;
    pointer-events: none;
    opacity: 0;
    transition: opacity 0.2s;
  }

  .blip.hovered .blip-text {
    opacity: 1;
  }

  .entries-list {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
    gap: 3rem;
  }

  h2 {
    font-size: 1.75rem;
    font-weight: 800;
    border-bottom: 4px solid currentColor;
    padding-bottom: 0.75rem;
    margin-bottom: 2rem;
    letter-spacing: -0.03em;
  }

  .entries-group {
    display: flex;
    flex-direction: column;
    gap: 1.5rem;
  }

  .entry-item {
    padding: 1.25rem;
    border-radius: 16px;
    transition: all 0.2s;
    background: var(--bg-card);
    border: 2px solid var(--border-color);
  }

  .entry-item.active {
    border-color: #4299e1;
    box-shadow: var(--shadow);
    transform: translateY(-4px);
  }

  .entry-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    margin-bottom: 0.75rem;
    gap: 1rem;
  }

  strong {
    font-size: 1.25rem;
    color: var(--text-primary);
    letter-spacing: -0.01em;
  }

  .description {
    font-size: 0.95rem;
    color: var(--text-secondary);
    line-height: 1.6;
    margin-bottom: 1rem;
  }

  .quadrant-tag {
    display: inline-block;
    background: var(--bg-secondary);
    padding: 4px 12px;
    border-radius: 9999px;
    font-size: 0.7rem;
    font-weight: 800;
    color: var(--text-secondary);
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .tag {
    display: inline-block;
    background: var(--bg-secondary);
    padding: 4px 10px;
    border-radius: 6px;
    font-size: 0.7rem;
    font-weight: 700;
    color: var(--text-secondary);
    margin-right: 0.5rem;
  }

  .history {
    margin-top: 1rem;
    padding-top: 1rem;
    border-top: 1px dashed var(--border-color);
  }

  .history h4 {
    font-size: 0.75rem;
    text-transform: uppercase;
    color: var(--text-muted);
    margin-bottom: 0.5rem;
    letter-spacing: 0.1em;
  }

  .history ul {
    list-style: none;
    padding: 0;
    margin: 0;
  }

  .history li {
    font-size: 0.8rem;
    color: var(--text-muted);
    margin-bottom: 0.4rem;
  }

  .date {
    font-weight: 600;
    color: var(--text-secondary);
  }

  .ring-tag {
    padding: 1px 6px;
    border-radius: 4px;
    font-weight: 700;
    font-size: 0.7rem;
    text-transform: uppercase;
  }

  .error {
    color: #c53030;
    text-align: center;
    background: #fff5f5;
    padding: 2rem;
    border-radius: 16px;
    font-weight: 700;
    box-shadow: 0 4px 6px rgba(0,0,0,0.05);
  }
</style>
