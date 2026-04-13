<script lang="ts">
  import { consoleLogs, showConsole } from './store';
  import type { LogEntry } from './store';
  import { afterUpdate } from 'svelte';

  let logContainer: HTMLDivElement;
  let autoScroll = true;

  afterUpdate(() => {
    if (autoScroll && logContainer) {
      logContainer.scrollTop = logContainer.scrollHeight;
    }
  });

  function handleScroll() {
    if (!logContainer) return;
    const { scrollTop, scrollHeight, clientHeight } = logContainer;
    autoScroll = scrollHeight - scrollTop - clientHeight < 40;
  }

  function clearLogs() {
    consoleLogs.set([]);
  }

  function levelIcon(level: LogEntry['level']): string {
    switch (level) {
      case 'info': return '›';
      case 'warn': return '⚠';
      case 'error': return '✖';
      case 'success': return '✔';
      case 'debug': return '…';
      default: return '›';
    }
  }
</script>

{#if $showConsole}
  <div class="console-panel">
    <div class="console-header">
      <div class="console-title">
        <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><polyline points="4,17 10,11 4,5"/><line x1="12" y1="19" x2="20" y2="19"/></svg>
        <span>terminal</span>
        <span class="log-count">{$consoleLogs.length}</span>
      </div>
      <div class="console-actions">
        <button class="console-btn" on:click={clearLogs} title="Clear">
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><circle cx="12" cy="12" r="10"/><line x1="15" y1="9" x2="9" y2="15"/><line x1="9" y1="9" x2="15" y2="15"/></svg>
        </button>
        <button class="console-btn" on:click={() => showConsole.set(false)} title="Close">
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><polyline points="6,9 12,15 18,9"/></svg>
        </button>
      </div>
    </div>
    <div class="console-body" bind:this={logContainer} on:scroll={handleScroll}>
      {#if $consoleLogs.length === 0}
        <div class="console-empty">No logs yet. Run a pipeline action to see output.</div>
      {:else}
        {#each $consoleLogs as entry}
          <div class="log-line level-{entry.level}">
            <span class="log-time">{entry.time}</span>
            <span class="log-icon">{levelIcon(entry.level)}</span>
            <span class="log-msg">{entry.message}</span>
          </div>
        {/each}
      {/if}
    </div>
  </div>
{/if}

<style>
  .console-panel {
    border-top: 1px solid var(--border);
    background: var(--bg-deep);
    display: flex;
    flex-direction: column;
    height: 170px;
    flex-shrink: 0;
  }
  .console-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 3px 12px;
    background: var(--bg-deep);
    border-bottom: 1px solid var(--border);
    min-height: 24px;
  }
  .console-title {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 11px;
    font-weight: 600;
    color: var(--text-muted);
  }
  .console-title svg { color: var(--text-dim); }
  .log-count {
    font-size: 8px;
    background: var(--border-active);
    color: var(--text-muted);
    padding: 0 4px;
    border-radius: 2px;
    font-weight: 600;
  }
  .console-actions { display: flex; gap: 2px; }
  .console-btn {
    background: none;
    border: none;
    color: var(--text-dim);
    cursor: pointer;
    padding: 2px 4px;
    border-radius: var(--radius-sm);
    display: flex;
    align-items: center;
    transition: all .1s;
  }
  .console-btn:hover { color: var(--text-muted); background: var(--bg-hover); }

  .console-body {
    flex: 1;
    overflow-y: auto;
    padding: 2px 0;
    font-size: 12px;
    line-height: 1.55;
  }
  .console-body::-webkit-scrollbar { width: 4px; }
  .console-body::-webkit-scrollbar-track { background: transparent; }
  .console-body::-webkit-scrollbar-thumb { background: var(--border-hover); border-radius: 2px; }

  .console-empty {
    color: var(--text-dim);
    padding: 12px 14px;
    font-size: 12px;
  }

  .log-line {
    display: flex;
    align-items: baseline;
    gap: 6px;
    padding: 0 12px;
    transition: background .06s;
  }
  .log-line:hover { background: var(--bg-hover); }

  .log-time {
    color: var(--text-dim);
    font-size: 9px;
    flex-shrink: 0;
    min-width: 56px;
  }
  .log-icon {
    flex-shrink: 0;
    width: 12px;
    font-size: 10px;
    text-align: center;
  }
  .log-msg {
    color: var(--text);
    word-break: break-word;
  }

  /* Level colors — muted */
  .level-info .log-icon { color: var(--blue); }
  .level-info .log-msg { color: var(--text); }

  .level-warn .log-icon { color: var(--yellow); }
  .level-warn .log-msg { color: var(--yellow); }

  .level-error .log-icon { color: var(--red); }
  .level-error .log-msg { color: var(--red); }

  .level-success .log-icon { color: var(--green); }
  .level-success .log-msg { color: var(--green); }

  .level-debug .log-icon { color: var(--text-dim); }
  .level-debug .log-msg { color: var(--text-muted); }
</style>
