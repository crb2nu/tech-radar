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

  const quadrants = ['languages', 'frameworks', 'tools', 'platforms'];
  const rings = ['adopt', 'trial', 'assess', 'hold'];

  const colors = {
    adopt: '#93c47d',
    trial: '#93d2c2',
    assess: '#fbdb84',
    hold: '#efafa9'
  };

  const ringRadius = [70, 140, 210, 280]; // Radii for the 4 rings

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
    return radar?.entries.filter(e => e.ring === ring) || [];
  }

  // Blip placement logic
  function getBlipPosition(entry: Entry, index: number, totalInArea: number) {
    const quadIndex = quadrants.indexOf(entry.quadrant);
    const ringIndex = rings.indexOf(entry.ring);
    
    if (quadIndex === -1 || ringIndex === -1) return { x: 0, y: 0 };

    // Angles for the quadrants (0-90, 90-180, 180-270, 270-360)
    const baseAngle = quadIndex * (Math.PI / 2);
    
    // Radii for the ring
    const innerR = ringIndex === 0 ? 20 : ringRadius[ringIndex - 1];
    const outerR = ringRadius[ringIndex];

    // Simple deterministic placement within the quadrant slice and ring area
    // In a real app, we might use a more sophisticated collision detection algorithm
    const seed = entry.name.length + index;
    const randomR = innerR + (Math.abs(Math.sin(seed)) * (outerR - innerR - 20)) + 10;
    const randomAngle = baseAngle + (Math.abs(Math.cos(seed)) * (Math.PI / 2 - 0.4)) + 0.2;

    return {
      x: 300 + randomR * Math.cos(randomAngle),
      y: 300 + randomR * Math.sin(randomAngle)
    };
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
      <svg viewBox="0 0 600 600" width="600" height="600">
        <!-- Rings -->
        {#each rings.slice().reverse() as ring, i}
          {@const r = ringRadius[3 - i]}
          <circle cx="300" cy="300" r={r} fill={colors[ring as keyof typeof colors]} fill-opacity="0.1" stroke="#fff" stroke-width="2" />
        {/each}
        
        <!-- Quadrant dividers -->
        <line x1="300" y1="20" x2="300" y2="580" stroke="#fff" stroke-width="2" />
        <line x1="20" y1="300" x2="580" y2="300" stroke="#fff" stroke-width="2" />

        <!-- Quadrant Labels -->
        <text x="50" y="50" class="quad-label">LANGUAGES</text>
        <text x="550" y="50" text-anchor="end" class="quad-label">FRAMEWORKS</text>
        <text x="50" y="550" class="quad-label">TOOLS</text>
        <text x="550" y="550" text-anchor="end" class="quad-label">PLATFORMS</text>

        <!-- Ring labels -->
        <text x="300" y="25" text-anchor="middle" class="ring-label">ADOPT</text>
        <text x="300" y="95" text-anchor="middle" class="ring-label">TRIAL</text>
        <text x="300" y="165" text-anchor="middle" class="ring-label">ASSESS</text>
        <text x="300" y="235" text-anchor="middle" class="ring-label">HOLD</text>

        <!-- Blips -->
        {#each radar.entries as entry, i}
          {@const pos = getBlipPosition(entry, i, radar.entries.length)}
          <g class="blip" transform="translate({pos.x}, {pos.y})">
            <circle r="6" fill={colors[entry.ring as keyof typeof colors]} stroke="#333" stroke-width="1" />
            <text y="-10" text-anchor="middle" class="blip-text">{entry.name}</text>
            <title>{entry.name} ({entry.ring}): {entry.description}</title>
          </g>
        {/each}
      </svg>
    </div>

    <div class="entries-list">
      {#each rings as ring}
        <section>
          <h2 style="color: {colors[ring as keyof typeof colors]}">{ring.toUpperCase()}</h2>
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
    font-family: 'Inter', system-ui, -apple-system, sans-serif;
    max-width: 1200px;
    margin: 0 auto;
    padding: 2rem;
    color: #333;
  }

  header {
    text-align: center;
    margin-bottom: 3rem;
  }

  h1 {
    font-weight: 800;
    font-size: 2.5rem;
    letter-spacing: -0.02em;
  }

  .radar-container {
    display: flex;
    justify-content: center;
    margin-bottom: 4rem;
    background: #fafafa;
    border-radius: 12px;
    padding: 2rem;
    box-shadow: inset 0 2px 4px rgba(0,0,0,0.05);
  }

  .ring-label {
    font-size: 10px;
    fill: #666;
    font-weight: bold;
    pointer-events: none;
  }

  .quad-label {
    font-size: 14px;
    fill: #bbb;
    font-weight: 900;
    pointer-events: none;
  }

  .blip circle {
    cursor: pointer;
    transition: r 0.2s;
  }

  .blip:hover circle {
    r: 8;
  }

  .blip-text {
    font-size: 9px;
    fill: #333;
    font-weight: 600;
    pointer-events: none;
    opacity: 0;
    transition: opacity 0.2s;
  }

  .blip:hover .blip-text {
    opacity: 1;
  }

  .entries-list {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
    gap: 3rem;
  }

  h2 {
    font-size: 1.25rem;
    font-weight: 700;
    border-bottom: 2px solid currentColor;
    padding-bottom: 0.5rem;
    margin-bottom: 1.5rem;
  }

  ul {
    list-style: none;
    padding: 0;
  }

  li {
    margin-bottom: 1.25rem;
    font-size: 0.95rem;
    line-height: 1.5;
  }

  .quadrant-tag {
    display: inline-block;
    background: #eee;
    padding: 2px 8px;
    border-radius: 100px;
    font-size: 0.7rem;
    font-weight: 600;
    margin-left: 0.5rem;
    color: #666;
    text-transform: uppercase;
  }

  .error {
    color: #e53e3e;
    text-align: center;
    background: #fff5f5;
    padding: 1rem;
    border-radius: 8px;
  }
</style>
