<script lang="ts">
  import { onMount } from 'svelte';

  interface Entry {
    name: string;
    quadrant: string;
    ring: string;
    description: string;
  }

  interface Radar {
    meta: {
      name: string;
    };
    entries: Entry[];
  }

  let radar: Radar | null = $state(null);
  let error: string | null = $state(null);

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

  const quadrants = ['languages', 'frameworks', 'tools', 'platforms', 'techniques'];
  const rings = ['adopt', 'trial', 'assess', 'hold'];

  function getEntriesByRing(ring: string) {
    return radar?.entries.filter(e => e.ring === ring) || [];
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
    </header>

    <div class="radar-container">
      <!-- Placeholder for SVG Radar -->
      <svg viewBox="0 0 600 600" width="600" height="600">
        <circle cx="300" cy="300" r="280" fill="none" stroke="#ccc" stroke-width="1" />
        <circle cx="300" cy="300" r="210" fill="none" stroke="#ccc" stroke-width="1" />
        <circle cx="300" cy="300" r="140" fill="none" stroke="#ccc" stroke-width="1" />
        <circle cx="300" cy="300" r="70" fill="none" stroke="#ccc" stroke-width="1" />
        
        <line x1="300" y1="20" x2="300" y2="580" stroke="#ccc" stroke-width="1" />
        <line x1="20" y1="300" x2="580" y2="300" stroke="#ccc" stroke-width="1" />

        <text x="300" y="45" text-anchor="middle" class="ring-label">ADOPT</text>
        <text x="300" y="115" text-anchor="middle" class="ring-label">TRIAL</text>
        <text x="300" y="185" text-anchor="middle" class="ring-label">ASSESS</text>
        <text x="300" y="255" text-anchor="middle" class="ring-label">HOLD</text>
      </svg>
    </div>

    <div class="entries-list">
      {#each rings as ring}
        <section>
          <h2>{ring.toUpperCase()}</h2>
          <ul>
            {#each getEntriesByRing(ring) as entry}
              <li>
                <strong>{entry.name}</strong>: {entry.description}
                <span class="quadrant-tag">{entry.quadrant}</span>
              </li>
            {/each}
          </ul>
        </section>
      {/each}
    </div>
  {/if}
</main>

<style>
  main {
    font-family: sans-serif;
    max-width: 1000px;
    margin: 0 auto;
    padding: 2rem;
  }

  header {
    text-align: center;
    margin-bottom: 2rem;
  }

  .radar-container {
    display: flex;
    justify-content: center;
    margin-bottom: 3rem;
  }

  .ring-label {
    font-size: 12px;
    fill: #999;
    font-weight: bold;
  }

  .entries-list {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
    gap: 2rem;
  }

  h2 {
    border-bottom: 2px solid #eee;
    padding-bottom: 0.5rem;
  }

  ul {
    list-style: none;
    padding: 0;
  }

  li {
    margin-bottom: 1rem;
    font-size: 0.9rem;
  }

  .quadrant-tag {
    display: inline-block;
    background: #f0f0f0;
    padding: 2px 6px;
    border-radius: 4px;
    font-size: 0.7rem;
    margin-left: 0.5rem;
    color: #666;
  }

  .error {
    color: red;
    text-align: center;
  }
</style>
