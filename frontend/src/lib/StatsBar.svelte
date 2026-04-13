<script lang="ts">
  import { bookmarks, statusMsg, loading, showConsole, consoleLogs } from './store';
  import type { Bookmark } from './store';

  export let progress: string = '';

  function getStats(bms: Bookmark[]): { total: number; categorized: number; paraCount: number; avgConf: number; tagCount: number } {
    const categorized = bms.filter(b => b.category).length;
    const paraCount = bms.filter(b => b.paraType).length;
    const allTags = new Set<string>();
    let totalConf = 0;
    let confCount = 0;
    for (const b of bms) {
      for (const t of (b.tags || [])) allTags.add(t);
      if (b.confidence) { totalConf += b.confidence; confCount++; }
    }
    return {
      total: bms.length,
      categorized,
      paraCount,
      avgConf: confCount > 0 ? totalConf / confCount : 0,
      tagCount: allTags.size,
    };
  }

  $: stats = getStats($bookmarks);
</script>

<div class="statsbar">
  <div class="stats-left">
    {#if $loading}
      <span class="stat-proc">
        <span class="proc-dot"></span>
        {progress || 'processing...'}
      </span>
    {:else if $statusMsg}
      <span class="stat-msg">{$statusMsg}</span>
    {:else}
      <span class="stat-chip">{stats.total} items</span>
      {#if stats.categorized > 0}
        <span class="stat-sep">·</span>
        <span class="stat-chip">{stats.categorized} categorized</span>
      {/if}
      {#if stats.paraCount > 0}
        <span class="stat-sep">·</span>
        <span class="stat-chip">{stats.paraCount} para</span>
      {/if}
      {#if stats.tagCount > 0}
        <span class="stat-sep">·</span>
        <span class="stat-chip">{stats.tagCount} tags</span>
      {/if}
      {#if stats.avgConf > 0}
        <span class="stat-sep">·</span>
        <span class="stat-chip stat-conf">{Math.round(stats.avgConf * 100)}% conf</span>
      {/if}
    {/if}
  </div>
  <div class="stats-right">
    <button class="console-toggle" class:active={$showConsole} on:click={() => showConsole.update(v => !v)} title="Toggle Console (Ctrl+`)">
      <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><polyline points="4,17 10,11 4,5"/><line x1="12" y1="19" x2="20" y2="19"/></svg>
      <span>terminal</span>
      {#if $consoleLogs.length > 0}
        <span class="toggle-count">{$consoleLogs.length}</span>
      {/if}
    </button>
  </div>
</div>

<style>
  .statsbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 3px 12px;
    box-shadow: inset 0 1px 0 var(--shadow-clr);
    background: var(--bg-deep);
    font-size: 12px;
    min-height: 26px;
    flex-shrink: 0;
  }
  .stats-left { display: flex; align-items: center; gap: 6px; flex: 1; }
  .stats-right { display: flex; align-items: center; flex-shrink: 0; }
  .stat-chip { color: var(--text-muted); white-space: nowrap; font-weight: 500; }
  .stat-conf { color: var(--green); }
  .stat-sep { color: var(--text-dim); }
  .stat-proc {
    display: flex;
    align-items: center;
    gap: 5px;
    color: var(--yellow);
    font-weight: 500;
  }
  .proc-dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: var(--yellow);
    animation: blink 1s infinite;
  }
  @keyframes blink { 0%,100% { opacity: 1; } 50% { opacity: 0.2; } }
  .stat-msg { color: var(--green); font-weight: 500; }

  .console-toggle {
    display: flex;
    align-items: center;
    gap: 4px;
    padding: 1px 6px;
    border: 1px solid transparent;
    border-radius: var(--radius-sm);
    background: none;
    color: var(--text-dim);
    cursor: pointer;
    font-size: 10px;
    font-weight: 500;
    transition: all .1s;
  }
  .console-toggle:hover { color: var(--text-muted); background: var(--bg-hover); }
  .console-toggle.active { border-color: var(--border); color: var(--blue); background: var(--bg-card); }
  .toggle-count {
    font-size: 8px;
    background: var(--border-active);
    color: var(--text-secondary);
    padding: 0 4px;
    border-radius: 2px;
    font-weight: 700;
  }
</style>
