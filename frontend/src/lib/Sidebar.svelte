<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import { bookmarks, activePara, activeTag, activeFolder, activeFolderUrls, loading, showSettings, showPrompts, folderTree, activeProfiles, pipelineStep } from './store';
  import type { Bookmark, ProfileTree } from './store';
  import FolderTree from './FolderTree.svelte';

  const dispatch = createEventDispatcher();

  interface AnalysisCacheMeta {
    exists: boolean;
    timestamp: string;
    count: number;
    model: string;
    tokens: number;
  }
  export let lastAnalysisMeta: AnalysisCacheMeta | null = null;

  function formatCacheDate(ts: string): string {
    try {
      const d = new Date(ts.replace(' ', 'T'));
      if (isNaN(d.getTime())) return ts;
      const now = new Date();
      const diff = now.getTime() - d.getTime();
      if (diff < 3_600_000) return `${Math.round(diff / 60000)}m ago`;
      if (diff < 86_400_000) return `${Math.round(diff / 3_600_000)}h ago`;
      return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
    } catch { return ts; }
  }

  // expanded state for profile trees
  let expandedProfiles: Record<string, boolean> = {};
  let expandedFolders: Record<string, boolean> = {};

  function toggleProfile(key: string) {
    expandedProfiles[key] = !expandedProfiles[key];
    expandedProfiles = expandedProfiles;
  }
  function toggleFolder(key: string) {
    expandedFolders[key] = !expandedFolders[key];
    expandedFolders = expandedFolders;
  }

  const paraTypes = [
    { key: null, label: 'Inbox', prefix: 'IN', cls: 'inbox' },
    { key: 'project', label: 'Projects', prefix: 'PR', cls: 'project' },
    { key: 'area', label: 'Areas', prefix: 'AR', cls: 'area' },
    { key: 'resource', label: 'Resources', prefix: 'RS', cls: 'resource' },
    { key: 'archive', label: 'Archives', prefix: 'AV', cls: 'archive' },
  ];

  function paraCount(key: string | null, bms: Bookmark[]): number {
    if (key === null) return bms.filter(b => !b.paraType).length;
    return bms.filter(b => b.paraType === key).length;
  }

  function getAllTags(bms: Bookmark[]): { tag: string; count: number }[] {
    const map = new Map<string, number>();
    for (const b of bms) {
      for (const t of (b.tags || [])) map.set(t, (map.get(t) || 0) + 1);
    }
    return [...map.entries()].sort((a, b) => b[1] - a[1]).slice(0, 30).map(([tag, count]) => ({ tag, count }));
  }

  function setPara(key: string | null) {
    activePara.update(v => v === (key === null ? 'inbox' : key) ? null : (key === null ? 'inbox' : key));
    activeTag.set(null);
    activeFolder.set(null);
    activeFolderUrls.set(null);
  }

  function setTag(tag: string) {
    activeTag.update(v => v === tag ? null : tag);
  }

  function setFolder(path: string) {
    activeFolder.update(v => v === path ? null : path);
    activePara.set(null);
    activeTag.set(null);
  }

  function profileKey(pt: ProfileTree) {
    return `${pt.browser}__${pt.profileDir}`;
  }

  // Browser icons
  function browserIcon(browser: string): string {
    if (browser === 'chrome') return 'C';
    if (browser === 'edge') return 'E';
    return browser[0]?.toUpperCase() ?? '?';
  }

  // Sidebar only shows profiles that pass the activeProfiles filter
  $: visibleTree = $activeProfiles === null
    ? $folderTree
    : $folderTree.filter(pt => ($activeProfiles as Set<string>).has(profileKey(pt)));

  $: hiddenTree = $activeProfiles === null
    ? ([] as typeof $folderTree)
    : $folderTree.filter(pt => !($activeProfiles as Set<string>).has(profileKey(pt)));

  function isProfileChecked(pt: ProfileTree): boolean {
    return $activeProfiles === null || $activeProfiles.has(profileKey(pt));
  }

  function toggleActiveProfile(pt: ProfileTree) {
    const k = profileKey(pt);
    activeProfiles.update(cur => {
      const allKeys = $folderTree.map(p => profileKey(p));
      const base = cur === null ? new Set(allKeys) : new Set(cur);
      if (base.has(k)) { base.delete(k); } else { base.add(k); }
      return base.size === allKeys.length ? null : base;
    });
  }

  $: tags = getAllTags($bookmarks);

  // Auto-expand profiles when tree first loads
  $: if ($folderTree.length > 0) {
    for (const pt of $folderTree) {
      const k = profileKey(pt);
      if (!(k in expandedProfiles)) expandedProfiles[k] = true;
    }
    expandedProfiles = expandedProfiles;
  }
</script>

<aside class="sidebar">
  <div class="sidebar-scroll">
  <div class="brand">
    <svg class="brand-icon" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
      <path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20"/><path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"/>
    </svg>
    <span class="brand-text">unified<span class="brand-accent">bookmarks</span></span>
  </div>

  <!-- Pipeline actions -->
  <section class="sidebar-section">
    <h3 class="section-title">
      <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="4,17 10,11 4,5"/><line x1="12" y1="19" x2="20" y2="19"/></svg>
      Pipeline
    </h3>
    <div class="pipeline-steps">
      <button class="pbtn" class:is-loading={$loading === 'scan'} class:done={$pipelineStep >= 1}
        disabled={!!$loading && $loading !== 'scan'} on:click={() => dispatch('scan')}>
        {#if $pipelineStep >= 1}
          <svg class="step-check" width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round"><polyline points="20,6 9,17 4,12"/></svg>
        {:else}
          <span class="step-num">1</span>
        {/if}
        scan
      </button>
      <span class="step-arrow">→</span>

      <button class="pbtn" class:is-loading={$loading === 'collect'} class:done={$pipelineStep >= 2}
        class:next-step={$pipelineStep === 1}
        disabled={!!$loading && $loading !== 'collect'} on:click={() => dispatch('collect')}>
        {#if $pipelineStep >= 2}
          <svg class="step-check" width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round"><polyline points="20,6 9,17 4,12"/></svg>
        {:else}
          <span class="step-num">2</span>
        {/if}
        collect
      </button>
      <span class="step-arrow">→</span>

      <button class="pbtn pbtn-ai" class:is-loading={$loading === 'analyze'} class:done={$pipelineStep >= 3}
        class:next-step={$pipelineStep === 2}
        disabled={!!$loading && $loading !== 'analyze'} on:click={() => dispatch('analyze')}>
        {#if $pipelineStep >= 3}
          <svg class="step-check" width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round"><polyline points="20,6 9,17 4,12"/></svg>
        {:else}
          <span class="step-num">3</span>
        {/if}
        analyze
      </button>
    </div>

    <!-- Load last analysis cache hint -->
    {#if lastAnalysisMeta?.exists && $pipelineStep < 3}
      <button class="load-last-btn" disabled={!!$loading} on:click={() => dispatch('loadlast')}>
        <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/><path d="M3 3v5h5"/></svg>
        load last · {lastAnalysisMeta.count} bm · {formatCacheDate(lastAnalysisMeta.timestamp)}
      </button>
    {/if}

    <!-- Sync: full-width, visually prominent after analyze -->
    <button class="pbtn pbtn-sync"
      class:is-loading={$loading === 'sync'}
      class:next-step={$pipelineStep === 3}
      class:done={$pipelineStep >= 4}
      disabled={$pipelineStep < 2 || (!!$loading && $loading !== 'sync')}
      on:click={() => dispatch('sync')}>
      {#if $loading === 'sync'}
        <span class="sync-spinner"></span>
        syncing…
      {:else if $pipelineStep >= 4}
        <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round"><polyline points="20,6 9,17 4,12"/></svg>
        synced
      {:else}
        <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><polyline points="17,1 21,5 17,9"/><path d="M3 11V9a4 4 0 0 1 4-4h14"/><polyline points="7,23 3,19 7,15"/><path d="M21 13v2a4 4 0 0 1-4 4H3"/></svg>
        {$pipelineStep === 3 ? '↑ sync to browsers' : 'sync'}
      {/if}
    </button>
  </section>

  <!-- Browser / Profile folder tree -->
  {#if $folderTree.length > 0}
    <section class="sidebar-section">
      <h3 class="section-title">
        <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/></svg>
        Collections
      </h3>

      <!-- All bookmarks -->
      <button class="nav-item" class:active={$activeFolder === null && $activePara === null && $activeTag === null}
        on:click={() => { activeFolder.set(null); activePara.set(null); activeTag.set(null); }}>
        <svg class="nav-folder-icon" width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M3 6h18M3 6a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v10a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V6z"/></svg>
        <span class="nav-label">All bookmarks</span>
        <span class="nav-count">{$bookmarks.length}</span>
      </button>

      {#each visibleTree as pt (profileKey(pt))}
        <!-- Profile row (collapsed/expanded) -->
        <div class="profile-group">
          <div class="profile-header">
            <button class="profile-check" class:checked={$activeProfiles === null || $activeProfiles.has(profileKey(pt))}
              on:click|stopPropagation={() => toggleActiveProfile(pt)}
              title={$activeProfiles === null || $activeProfiles.has(profileKey(pt)) ? 'Uncheck to hide profile' : 'Check to show profile'}>
              {#if $activeProfiles === null || $activeProfiles.has(profileKey(pt))}
                <svg width="8" height="8" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round"><polyline points="20,6 9,17 4,12"/></svg>
              {/if}
            </button>
            <button class="profile-expand" on:click={() => toggleProfile(profileKey(pt))}>
              <span class="browser-badge browser-{pt.browser}">{browserIcon(pt.browser)}</span>
              <span class="profile-name">{pt.displayName}</span>
              <span class="profile-sublabel">{pt.browserLabel}</span>
              <span class="nav-count">{pt.totalCount}</span>
              <svg class="expand-chevron" class:expanded={expandedProfiles[profileKey(pt)]}
                width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
                <polyline points="9,18 15,12 9,6"/>
              </svg>
            </button>
          </div>

          {#if expandedProfiles[profileKey(pt)]}
            <FolderTree nodes={pt.roots} depth={1} profileKey={profileKey(pt)} />
          {/if}
        </div>
      {/each}

      <!-- Hidden profiles (grayed out, click to re-enable) -->
      {#if hiddenTree.length > 0}
        {#each hiddenTree as pt (profileKey(pt))}
          <div class="profile-group profile-hidden">
            <div class="profile-header">
              <button class="profile-check"
                on:click|stopPropagation={() => toggleActiveProfile(pt)}
                title="Click to show profile">
              </button>
              <button class="profile-expand profile-expand-disabled">
                <span class="browser-badge browser-{pt.browser}">{browserIcon(pt.browser)}</span>
                <span class="profile-name">{pt.displayName}</span>
                <span class="profile-sublabel">{pt.browserLabel}</span>
              </button>
            </div>
          </div>
        {/each}
      {/if}
    </section>
  {/if}

  <!-- PARA -->
  <section class="sidebar-section">
    <h3 class="section-title">
      <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/><rect x="3" y="14" width="7" height="7"/><rect x="14" y="14" width="7" height="7"/></svg>
      PARA
    </h3>
    {#each paraTypes as p}
      {@const count = paraCount(p.key, $bookmarks)}
      <button class="nav-item" class:active={$activePara === (p.key === null ? 'inbox' : p.key)}
        on:click={() => setPara(p.key)}>
        <span class="nav-prefix {p.cls}">{p.prefix}</span>
        <span class="nav-label">{p.label}</span>
        <span class="nav-count" class:count-warn={p.key === null && count > 0}>{count}</span>
      </button>
    {/each}
  </section>

  <!-- Filters: tags -->
  {#if tags.length > 0}
    <section class="sidebar-section">
      <h3 class="section-title">
        <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M20.59 13.41l-7.17 7.17a2 2 0 0 1-2.83 0L2 12V2h10l8.59 8.59a2 2 0 0 1 0 2.82z"/><line x1="7" y1="7" x2="7.01" y2="7"/></svg>
        Tags ({tags.length})
      </h3>
      <div class="tag-cloud">
        {#each tags as t}
          <button class="tag-pill" class:active={$activeTag === t.tag} on:click={() => setTag(t.tag)}>
            <span class="tag-hash">#</span>{t.tag}
            <span class="tag-cnt">{t.count}</span>
          </button>
        {/each}
      </div>
    </section>
  {/if}
  </div>

  <div class="sidebar-footer">
    <button class="footer-btn" disabled={$loading === 'analyze'} on:click={() => dispatch('restore')}>
      <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/><path d="M3 3v5h5"/></svg>
      Backups
    </button>
    <button class="footer-btn" on:click={() => showPrompts.set(true)}>
      <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
        <polygon points="13,2 3,14 12,14 11,22 21,10 12,10 13,2"/>
      </svg>
      Prompts
    </button>
    <button class="footer-btn" on:click={() => showSettings.set(true)}>
      <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
        <circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"/>
      </svg>
      Settings
    </button>
  </div>
</aside>

<style>
  .sidebar {
    width: 248px;
    background: var(--bg-sidebar);
    /* Raindrop: shadow-based right edge, not border */
    box-shadow: inset -1px 0 0 var(--shadow-clr);
    display: flex;
    flex-direction: column;
    flex-shrink: 0;
    padding: 0;
  }
  .sidebar-scroll {
    flex: 1;
    overflow-y: auto;
    padding: 8px 6px 4px;
    min-height: 0;
  }
  .brand {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 10px 10px 12px;
    margin-bottom: 4px;
  }
  .brand-icon { color: var(--accent); flex-shrink: 0; }
  .brand-text { font-size: 13px; font-weight: 600; color: var(--text-secondary); letter-spacing: -0.01em; }
  .brand-accent { color: var(--text); }

  .sidebar-section { margin-bottom: 8px; }
  .section-title {
    display: flex;
    align-items: center;
    gap: 5px;
    font-size: 11px;
    color: var(--text-muted);
    text-transform: none;
    letter-spacing: 0.1px;
    font-weight: 600;
    padding: 2px 10px 5px;
    opacity: 0.7;
  }
  .section-title svg { color: var(--text-muted); }

  .pipeline-steps {
    display: flex;
    align-items: center;
    gap: 2px;
    padding: 0 4px;
    margin-bottom: 5px;
  }
  .step-arrow {
    color: var(--text-dim);
    font-size: 10px;
    flex-shrink: 0;
    padding: 0 1px;
    opacity: 0.5;
  }
  .step-num {
    width: 14px;
    height: 14px;
    border-radius: 50%;
    background: var(--bg-hover);
    color: var(--text-dim);
    font-size: 9px;
    font-weight: 700;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }
  .step-check { flex-shrink: 0; }

  .pbtn {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 3px;
    padding: 6px 2px;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--bg-card);
    color: var(--text-secondary);
    cursor: pointer;
    font-size: 11px;
    font-weight: 500;
    text-transform: lowercase;
    transition: all .12s;
    white-space: nowrap;
  }
  .pbtn:hover { border-color: var(--border-hover); color: var(--text); background: var(--bg-hover); }
  .pbtn.done { color: var(--green); border-color: var(--green-dim); background: var(--green-dim); }
  .pbtn.done svg { color: var(--green); }
  .pbtn.next-step { border-color: var(--accent); color: var(--accent); animation: step-pulse 2s infinite; }
  @keyframes step-pulse { 0%,100% { box-shadow: 0 0 0 0 var(--accent-dim); } 50% { box-shadow: 0 0 0 3px var(--accent-dim); } }

  .pbtn-ai { border-color: var(--accent-dim); color: var(--accent); }
  .pbtn-ai:hover { background: var(--accent-dim); border-color: var(--accent); }
  .pbtn-ai.done { color: var(--green); border-color: var(--green-dim); background: var(--green-dim); }

  /* Sync: full-width prominent button */
  .pbtn-sync {
    width: calc(100% - 8px);
    margin: 0 4px;
    flex: none;
    padding: 7px 10px;
    font-size: 12px;
    font-weight: 600;
    justify-content: center;
    gap: 6px;
    border-color: var(--border);
    background: var(--bg-card);
    color: var(--text-muted);
  }
  .pbtn-sync:not(:disabled):hover { border-color: var(--accent); color: var(--accent); background: var(--accent-dim); }
  .pbtn-sync.next-step {
    background: var(--accent);
    color: #fff;
    border-color: var(--accent);
    animation: none;
    box-shadow: 0 2px 12px rgba(0,0,0,.25);
  }
  .pbtn-sync.next-step:hover { opacity: 0.9; }
  .pbtn-sync.done { background: var(--green-dim); color: var(--green); border-color: var(--green-dim); }
  .pbtn-sync:disabled { opacity: 0.3; pointer-events: none; }
  .pbtn-sync.is-loading { opacity: 0.7; pointer-events: none; }

  .sync-spinner {
    width: 10px; height: 10px; border-radius: 50%;
    border: 1.5px solid rgba(255,255,255,0.4); border-top-color: #fff;
    animation: spin .6s linear infinite; flex-shrink: 0;
  }
  @keyframes spin { to { transform: rotate(360deg); } }

  .pbtn.is-loading { opacity: 0.5; pointer-events: none; animation: pulse 1.5s infinite; }
  .pbtn:disabled:not(.pbtn-sync) { opacity: 0.3; pointer-events: none; cursor: not-allowed; }
  @keyframes pulse { 0%,100% { opacity: 0.5; } 50% { opacity: 0.2; } }

  /* Load last analysis cache button */
  .load-last-btn {
    display: flex;
    align-items: center;
    gap: 5px;
    width: calc(100% - 8px);
    margin: 3px 4px 0;
    padding: 4px 8px;
    border: 1px dashed var(--border);
    border-radius: var(--radius-sm);
    background: transparent;
    color: var(--text-dim);
    font-size: 10px;
    font-weight: 500;
    cursor: pointer;
    text-align: left;
    transition: all .12s;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .load-last-btn:hover { border-color: var(--accent); color: var(--accent); background: var(--accent-dim); }
  .load-last-btn:disabled { opacity: 0.3; pointer-events: none; }

  .nav-item {
    display: flex;
    align-items: center;
    gap: 8px;
    width: 100%;
    padding: 6px 10px;
    border: none;
    border-radius: var(--radius);
    background: transparent;
    color: var(--text-secondary);
    cursor: pointer;
    font-size: 13px;
    text-align: left;
    transition: background .1s, color .1s;
  }
  .nav-item:hover { background: var(--bg-hover); color: var(--text); }
  /* Raindrop active = subtle full-row background, no colored left border */
  .nav-item.active { background: var(--bg-active); color: var(--text); }
  .nav-item.has-warning { color: var(--orange); }

  .nav-prefix {
    font-size: 10px;
    font-weight: 600;
    letter-spacing: 0.3px;
    width: 20px;
    text-align: center;
    flex-shrink: 0;
    opacity: 0.6;
  }
  .nav-prefix.inbox   { color: var(--orange);      opacity: 0.9; }
  .nav-prefix.project { color: var(--accent);      opacity: 0.9; }
  .nav-prefix.area    { color: var(--green);        opacity: 0.9; }
  .nav-prefix.resource{ color: var(--cyan);         opacity: 0.9; }
  .nav-prefix.archive { color: var(--text-muted); }

  .nav-dot {
    width: 6px;
    height: 6px;
    border-radius: 1px;
    background: var(--border-active);
    flex-shrink: 0;
  }

  .nav-label { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .nav-count { font-size: 11px; color: var(--text-dim); min-width: 20px; text-align: right; font-weight: 500; }
  .count-warn {
    background: var(--orange-dim);
    color: var(--orange);
    padding: 0 5px;
    border-radius: 3px;
    font-weight: 600;
    font-size: 10px;
  }

  .tag-cloud { display: flex; flex-wrap: wrap; gap: 3px; padding: 0 6px; }
  .tag-pill {
    padding: 2px 7px;
    border-radius: var(--radius-sm);
    font-size: 11px;
    background: transparent;
    color: var(--text-muted);
    border: 1px solid var(--border);
    cursor: pointer;
    font-weight: 500;
    transition: all .1s;
    display: flex;
    align-items: center;
    gap: 2px;
  }
  .tag-pill:hover { border-color: var(--border-hover); color: var(--text-secondary); }
  .tag-pill.active { border-color: var(--accent); color: var(--accent); background: var(--accent-dim); }
  .tag-hash { color: var(--text-dim); font-weight: 400; }
  .tag-cnt { font-size: 10px; color: var(--text-dim); }

  .sidebar-footer { flex-shrink: 0; padding: 6px 6px 8px; box-shadow: inset 0 1px 0 var(--shadow-clr); }
  .footer-btn {
    display: flex;
    align-items: center;
    gap: 6px;
    width: 100%;
    padding: 7px 10px;
    border: none;
    border-radius: var(--radius);
    background: transparent;
    color: var(--text-muted);
    cursor: pointer;
    font-size: 12px;
    font-weight: 500;
    transition: background .1s, color .1s;
  }
  .footer-btn:hover { color: var(--text-secondary); background: var(--bg-hover); }
  .footer-btn:disabled { opacity: 0.3; pointer-events: none; cursor: not-allowed; }

  /* ── Profile / Folder tree ─────────────────────────── */
  .nav-folder-icon { color: var(--text-dim); flex-shrink: 0; }

  .profile-group { margin-bottom: 2px; }
  .profile-hidden { opacity: 0.38; }

  /* profile-header is now a flex row wrapping check + expand */
  .profile-header {
    display: flex;
    align-items: center;
    gap: 0;
    width: 100%;
    border-radius: var(--radius-sm);
  }
  .profile-header:hover { background: var(--bg-hover); }

  .profile-check {
    width: 24px;
    height: 28px;
    flex-shrink: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    background: none;
    border: none;
    cursor: pointer;
    color: var(--text-dim);
    border-radius: var(--radius-sm) 0 0 var(--radius-sm);
    transition: color .1s;
  }
  .profile-check:hover { color: var(--text-muted); }
  .profile-check .check-box-inner {
    width: 13px;
    height: 13px;
    border-radius: 3px;
    border: 1.5px solid var(--border-hover);
  }
  .profile-check.checked { color: var(--accent); }

  .profile-expand {
    flex: 1;
    display: flex;
    align-items: center;
    gap: 7px;
    padding: 5px 8px 5px 2px;
    background: none;
    border: none;
    cursor: pointer;
    min-width: 0;
    border-radius: 0 var(--radius-sm) var(--radius-sm) 0;
    transition: background .07s;
  }
  .profile-expand-disabled { cursor: default; pointer-events: none; }

  .browser-badge {
    width: 18px;
    height: 18px;
    border-radius: 4px;
    font-size: 9px;
    font-weight: 700;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    letter-spacing: 0;
  }
  .browser-chrome { background: #e8f0fe; color: #1a73e8; }
  .browser-edge   { background: #e3f2fd; color: #0078d4; }

  /* Dark mode overrides */
  :global([data-theme="dark"]) .browser-chrome { background: rgba(66,133,244,.15); color: #8ab4f8; }
  :global([data-theme="dark"]) .browser-edge   { background: rgba(0,120,212,.15);  color: #6caef5; }
  /* Sand mode overrides */
  :global([data-theme="sand"]) .browser-chrome { background: hsl(215,60%,88%); color: hsl(215,70%,35%); }
  :global([data-theme="sand"]) .browser-edge   { background: hsl(208,60%,88%); color: hsl(208,70%,32%); }

  .profile-name {
    font-size: 13px;
    font-weight: 600;
    color: var(--text-secondary);
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    text-align: left;
  }
  .profile-sublabel {
    font-size: 10px;
    color: var(--text-dim);
    white-space: nowrap;
  }

  .expand-chevron { color: var(--text-dim); transition: transform .15s; flex-shrink: 0; }
  .expand-chevron.expanded { transform: rotate(90deg); }
</style>
