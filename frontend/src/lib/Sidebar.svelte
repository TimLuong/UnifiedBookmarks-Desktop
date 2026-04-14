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
  let collectionsCollapsed = false;
  let paraCollapsed = false;
  let tagsCollapsed = true;

  function toggleProfile(key: string) {
    expandedProfiles[key] = !expandedProfiles[key];
    expandedProfiles = expandedProfiles;
  }
  function toggleFolder(key: string) {
    expandedFolders[key] = !expandedFolders[key];
    expandedFolders = expandedFolders;
  }

  const paraTypes = [
    { key: null, label: 'Inbox', prefix: '📥', cls: 'inbox' },
    { key: 'project', label: 'Projects', prefix: '🚀', cls: 'project' },
    { key: 'area', label: 'Areas', prefix: '🗂️', cls: 'area' },
    { key: 'resource', label: 'Resources', prefix: '📚', cls: 'resource' },
    { key: 'archive', label: 'Archives', prefix: '🗃️', cls: 'archive' },
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
  <div class="brand">
    <span class="brand-emoji">🔖</span>
    <span class="brand-text">unified<span class="brand-accent">bookmarks</span></span>
  </div>
  <div class="sidebar-scroll">

  <!-- Pipeline actions -->
  <section class="sidebar-section">
    <h3 class="section-title">⚡ Pipeline</h3>
    <div class="pipeline-steps">
      <button class="pbtn" class:is-loading={$loading === 'scan'} class:done={$pipelineStep >= 1}
        disabled={!!$loading && $loading !== 'scan'} on:click={() => dispatch('scan')}>
        {#if $pipelineStep >= 1}✅{:else}<span class="step-num">1</span>{/if}
        🔍 scan
      </button>

      <button class="pbtn" class:is-loading={$loading === 'collect'} class:done={$pipelineStep >= 2}
        class:next-step={$pipelineStep === 1}
        disabled={!!$loading && $loading !== 'collect'} on:click={() => dispatch('collect')}>
        {#if $pipelineStep >= 2}✅{:else}<span class="step-num">2</span>{/if}
        📥 collect
      </button>

      <button class="pbtn pbtn-ai" class:is-loading={$loading === 'analyze'} class:done={$pipelineStep >= 3}
        class:next-step={$pipelineStep === 2}
        disabled={!!$loading && $loading !== 'analyze'} on:click={() => dispatch('analyze')}>
        {#if $pipelineStep >= 3}✅{:else}<span class="step-num">3</span>{/if}
        🧠 analyze
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
        ⏳ syncing…
      {:else if $pipelineStep >= 4}
        ✅ synced
      {:else}
        🔄 {$pipelineStep === 3 ? 'sync to browsers' : 'sync'}
      {/if}
    </button>
  </section>

  <!-- Browser / Profile folder tree -->
  {#if $folderTree.length > 0}
    <section class="sidebar-section">
      <button class="section-title section-title-btn" on:click={() => collectionsCollapsed = !collectionsCollapsed}>
        <span>🗂 Collections</span>
        <svg class="collapse-chevron" class:collapsed={collectionsCollapsed} width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><polyline points="18,15 12,9 6,15"/></svg>
      </button>

    {#if !collectionsCollapsed}
      <!-- All bookmarks -->
      <button class="nav-item" class:active={$activeFolder === null && $activePara === null && $activeTag === null}
        on:click={() => { activeFolder.set(null); activePara.set(null); activeTag.set(null); }}>
        <span class="nav-emoji">🌐</span>
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
    {/if}
    </section>
  {/if}

  <!-- PARA -->
  <section class="sidebar-section">
    <button class="section-title section-title-btn" on:click={() => paraCollapsed = !paraCollapsed}>
      <span>
        <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/><rect x="3" y="14" width="7" height="7"/><rect x="14" y="14" width="7" height="7"/></svg>
        PARA
      </span>
      <svg class="collapse-chevron" class:collapsed={paraCollapsed} width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><polyline points="18,15 12,9 6,15"/></svg>
    </button>
    {#if !paraCollapsed}
      {#each paraTypes as p}
        {@const count = paraCount(p.key, $bookmarks)}
        <button class="nav-item" class:active={$activePara === (p.key === null ? 'inbox' : p.key)}
          on:click={() => setPara(p.key)}>
          <span class="nav-prefix {p.cls}">{p.prefix}</span>
          <span class="nav-label">{p.label}</span>
          <span class="nav-count" class:count-warn={p.key === null && count > 0}>{count}</span>
        </button>
      {/each}
    {/if}
  </section>

  <!-- Filters: tags -->
  {#if tags.length > 0}
    <section class="sidebar-section">
      <button class="section-title section-title-btn" on:click={() => tagsCollapsed = !tagsCollapsed}>
        <span>🏷️ Tags ({tags.length})</span>
        <svg class="collapse-chevron" class:collapsed={tagsCollapsed} width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><polyline points="18,15 12,9 6,15"/></svg>
      </button>
      {#if !tagsCollapsed}
        <div class="tag-cloud">
          {#each tags as t}
            <button class="tag-pill" class:active={$activeTag === t.tag} on:click={() => setTag(t.tag)}>
              <span class="tag-hash">#</span>{t.tag}
              <span class="tag-cnt">{t.count}</span>
            </button>
          {/each}
        </div>
      {/if}
    </section>
  {/if}
  </div>

  <div class="sidebar-footer">
    <button class="footer-btn" disabled={$loading === 'analyze'} on:click={() => dispatch('restore')}>
      💾 Backups
    </button>
    <button class="footer-btn" on:click={() => showPrompts.set(true)}>
      ⚡ Prompts
    </button>
    <button class="footer-btn" on:click={() => showSettings.set(true)}>
      ⚙️ Settings
    </button>
  </div>
</aside>

<style>
  .sidebar {
    width: 280px;
    background: var(--bg-sidebar);
    border-right: 1px solid var(--border);
    display: flex;
    flex-direction: column;
    flex-shrink: 0;
    padding: 0;
  }
  .sidebar-scroll {
    flex: 1;
    overflow-y: auto;
    padding: 4px 0;
    min-height: 0;
  }
  .brand {
    display: flex;
    align-items: center;
    gap: 8px;
    height: 48px;
    padding: 0 12px;
    flex-shrink: 0;
    border-bottom: 1px solid var(--border);
  }
  .brand-emoji { font-size: 18px; flex-shrink: 0; }
  .brand-text { font-size: 13px; font-weight: 700; color: var(--text-secondary); letter-spacing: -0.02em; }
  .brand-accent { color: var(--accent); }

  .sidebar-section { margin-bottom: 4px; }
  .section-title {
    font-size: 10px;
    color: var(--text-muted);
    letter-spacing: 0.5px;
    text-transform: uppercase;
    font-weight: 700;
    padding: 6px 12px 4px;
    opacity: 0.75;
  }
  .section-title-btn {
    display: flex;
    align-items: center;
    justify-content: space-between;
    width: 100%;
    background: none;
    border: none;
    cursor: pointer;
    border-radius: var(--radius-sm);
    transition: background .12s, opacity .12s;
  }
  .section-title-btn:hover { background: var(--bg-hover); opacity: 1; }
  .collapse-chevron { color: var(--text-dim); transition: transform .2s; flex-shrink: 0; }
  .collapse-chevron.collapsed { transform: rotate(180deg); }

  .pipeline-steps {
    display: flex;
    flex-direction: column;
    gap: 3px;
    padding: 0 12px;
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
    justify-content: flex-start;
    gap: 7px;
    padding: 7px 10px;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: rgba(255,255,255,.03);
    color: var(--text-secondary);
    cursor: pointer;
    font-size: 12px;
    font-weight: 600;
    text-transform: lowercase;
    transition: all .15s;
    white-space: nowrap;
    width: 100%;
  }
  .pbtn:hover { border-color: var(--border-hover); color: var(--text); background: var(--bg-hover); }
  .pbtn.done { color: var(--green); border-color: var(--green-dim); background: var(--green-dim); }
  .pbtn.next-step { border-color: var(--accent); color: var(--accent); animation: step-pulse 1.8s infinite; }
  @keyframes step-pulse { 0%,100% { box-shadow: 0 0 0 0 var(--accent-dim); } 50% { box-shadow: 0 0 0 4px var(--accent-dim); } }

  .pbtn-ai { border-color: var(--accent-dim); color: var(--accent); }
  .pbtn-ai:hover { background: var(--accent-dim); border-color: var(--accent); }
  .pbtn-ai.done { color: var(--green); border-color: var(--green-dim); background: var(--green-dim); }

  /* Sync: full-width prominent button */
  .pbtn-sync {
    width: calc(100% - 24px);
    margin: 4px 12px 0;
    flex: none;
    padding: 8px 10px;
    font-size: 12px;
    font-weight: 700;
    justify-content: center;
    gap: 6px;
    border-color: var(--border);
    background: rgba(255,255,255,.03);
    color: var(--text-muted);
  }
  .pbtn-sync:not(:disabled):hover { border-color: var(--accent); color: var(--accent); background: var(--bg-hover); }
  .pbtn-sync.next-step {
    background: var(--accent);
    color: #fff;
    border-color: var(--accent);
    animation: none;
    box-shadow: 0 2px 16px rgba(255, 77, 77, .35);
  }
  .pbtn-sync.next-step:hover { filter: brightness(1.1); }
  .pbtn-sync.done { background: var(--green-dim); color: var(--green); border-color: var(--green-dim); }
  .pbtn-sync:disabled { opacity: 0.28; pointer-events: none; }
  .pbtn-sync.is-loading { opacity: 0.6; pointer-events: none; }

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
    width: calc(100% - 24px);
    margin: 3px 12px 0;
    padding: 5px 8px;
    border: 1px dashed var(--border);
    border-radius: var(--radius-sm);
    background: transparent;
    color: var(--text-dim);
    font-size: 10px;
    font-weight: 600;
    cursor: pointer;
    text-align: left;
    transition: all .15s;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .load-last-btn:hover { border-color: var(--accent); color: var(--accent); background: var(--accent-dim); box-shadow: 0 0 8px var(--accent-dim); }
  .load-last-btn:disabled { opacity: 0.3; pointer-events: none; }

  .nav-emoji { font-size: 14px; flex-shrink: 0; width: 20px; text-align: center; }

  .nav-item {
    display: flex;
    align-items: center;
    gap: 8px;
    width: 100%;
    padding: 7px 12px;
    border: none;
    border-radius: var(--radius);
    background: transparent;
    color: var(--text-secondary);
    cursor: pointer;
    font-size: 13px;
    text-align: left;
    transition: background .15s, color .15s;
  }
  .nav-item:hover { background: var(--bg-hover); color: var(--text); }
  .nav-item.active {
    background: var(--bg-active);
    color: var(--text);
    box-shadow: inset 2px 0 0 var(--accent);
  }
  .nav-item.has-warning { color: var(--orange); }

  .nav-prefix {
    font-size: 16px;
    width: 20px;
    text-align: center;
    flex-shrink: 0;
  }

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

  .tag-cloud { display: flex; flex-wrap: wrap; gap: 4px; padding: 0 12px; }
  .tag-pill {
    padding: 2px 8px;
    border-radius: 20px;
    font-size: 11px;
    background: transparent;
    color: var(--text-muted);
    border: 1px solid var(--border);
    cursor: pointer;
    font-weight: 500;
    transition: all .15s;
    display: flex;
    align-items: center;
    gap: 2px;
  }
  .tag-pill:hover { border-color: var(--border-hover); color: var(--text-secondary); background: var(--bg-hover); }
  .tag-pill.active { border-color: var(--accent); color: var(--accent); background: var(--accent-dim); box-shadow: 0 0 8px var(--accent-dim); }
  .tag-hash { color: var(--accent); opacity: 0.5; font-weight: 400; }
  .tag-cnt { font-size: 10px; color: var(--text-dim); }

  .sidebar-footer { flex-shrink: 0; padding: 6px 12px 10px; border-top: 1px solid var(--border); }
  .footer-btn {
    display: flex;
    align-items: center;
    gap: 6px;
    width: 100%;
    padding: 7px 0;
    border: none;
    border-radius: var(--radius);
    background: transparent;
    color: var(--text-muted);
    cursor: pointer;
    font-size: 12px;
    font-weight: 600;
    transition: background .15s, color .15s;
  }
  .footer-btn:hover { color: var(--text); background: var(--bg-hover); }
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
