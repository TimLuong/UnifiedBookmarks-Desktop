<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import { searchQuery, view, bookmarks, activePara, activeTag, activeCat, activeFolder, loading, sortBy, showInList, folderTree, activeProfiles, uniqueUrlsOnly } from './store';
  import type { Bookmark } from './store';

  const dispatch = createEventDispatcher();

  let showSortMenu = false;
  let showViewPanel = false;
  let showProfileFilter = false;

  // Build flat profile list from folderTree
  $: profileOptions = $folderTree.map(pt => ({
    key: `${pt.browser}__${pt.profileDir}`,
    label: pt.displayName,
    sub: pt.browserLabel,
    browser: pt.browser,
  }));

  function profileKey(browser: string, profileDir: string) {
    return `${browser}__${profileDir}`;
  }

  function isProfileActive(key: string): boolean {
    return $activeProfiles === null || $activeProfiles.has(key);
  }

  function toggleProfile(key: string) {
    activeProfiles.update(cur => {
      // Start from all if currently "all"
      const base = cur === null
        ? new Set(profileOptions.map(p => p.key))
        : new Set(cur);
      if (base.has(key)) {
        base.delete(key);
      } else {
        base.add(key);
      }
      // If all selected → null (no filter)
      if (base.size === profileOptions.length) return null;
      return base;
    });
  }

  function clearProfileFilter() {
    activeProfiles.set(null);
  }

  // Badge = number of ACTIVE profiles when subset is selected, 0 when all shown
  $: activeProfileCount = $activeProfiles === null ? 0 : $activeProfiles.size;

  function countFiltered(bms: Bookmark[], para: string | null, tag: string | null, cat: string | null, folder: string | null, search: string): number {
    let out = bms;
    if (para) {
      if (para === 'inbox') out = out.filter(b => !b.paraType);
      else out = out.filter(b => b.paraType === para);
    }
    if (tag) out = out.filter(b => (b.tags || []).includes(tag));
    if (cat) out = out.filter(b => (b.category || '').startsWith(cat));
    if (folder) out = out.filter(b => (b.folderPath || '').startsWith(folder));
    if (search) {
      const q = search.toLowerCase();
      out = out.filter(b =>
        b.title.toLowerCase().includes(q) || b.url.toLowerCase().includes(q) || (b.category || '').toLowerCase().includes(q)
      );
    }
    return out.length;
  }

  const sortOptions: { key: typeof $sortBy; label: string; icon: string }[] = [
    { key: 'date-asc',  label: 'By date ↑', icon: 'clock' },
    { key: 'date-desc', label: 'By date ↓', icon: 'clock' },
    { key: 'name-asc',  label: 'By name (A–Z)', icon: 'az' },
    { key: 'name-desc', label: 'By name (Z–A)', icon: 'za' },
    { key: 'site-asc',  label: 'Sites (A–Z)', icon: 'site' },
    { key: 'site-desc', label: 'Sites (Z–A)', icon: 'site' },
  ];

  function sortLabel(s: typeof $sortBy): string {
    return sortOptions.find(o => o.key === s)?.label ?? 'Sort';
  }

  function closeMenus() {
    showSortMenu = false;
    showViewPanel = false;
    showProfileFilter = false;
  }

  $: filteredCount = countFiltered($bookmarks, $activePara, $activeTag, $activeCat, $activeFolder, $searchQuery);
</script>

<svelte:window on:click={closeMenus} />

<div class="toolbar">
  <div class="toolbar-left">
    <div class="search-box">
      <svg class="search-icon" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
      <input type="text" placeholder="Search..." bind:value={$searchQuery} />
      {#if $searchQuery}
        <button class="search-clear" on:click={() => searchQuery.set('')}>×</button>
      {/if}
    </div>
    <span class="result-count">{filteredCount}<span class="count-sep">/</span>{$bookmarks.length}</span>
  </div>

  <div class="toolbar-right">
    <!-- Filter Profiles dropdown -->
    {#if profileOptions.length > 1}
      <div class="tb-menu-wrap" on:click|stopPropagation>
        <button class="tbtn tbtn-profiles" class:active={showProfileFilter} class:has-filter={activeProfileCount > 0}
          on:click={() => { showProfileFilter = !showProfileFilter; showSortMenu = false; }}>
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><polygon points="22,3 2,3 10,12.46 10,19 14,21 14,12.46"/></svg>
          Profiles
          {#if activeProfileCount > 0}<span class="filter-badge">{activeProfileCount}</span>{/if}
          <svg class="chevron" width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><polyline points="6,9 12,15 18,9"/></svg>
        </button>
        {#if showProfileFilter}
          <div class="dropdown profile-dropdown">
            <div class="dropdown-header">
              <span class="dropdown-label">Filter by profile</span>
              {#if activeProfileCount > 0}
                <button class="clear-filter" on:click={clearProfileFilter}>clear</button>
              {/if}
            </div>
            {#each profileOptions as opt}
              <button class="dropdown-item profile-item" on:click={() => toggleProfile(opt.key)}>
                <span class="check-box" class:checked={$activeProfiles === null || $activeProfiles.has(opt.key)}>
                  {#if $activeProfiles === null || $activeProfiles.has(opt.key)}
                    <svg width="9" height="9" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round"><polyline points="20,6 9,17 4,12"/></svg>
                  {/if}
                </span>
                <span class="prof-badge prof-{opt.browser}">{opt.browser === 'chrome' ? 'C' : opt.browser === 'edge' ? 'E' : opt.browser[0]?.toUpperCase()}</span>
                <span class="prof-name">{opt.label}</span>
                <span class="prof-sub">{opt.sub}</span>
              </button>
            {/each}
          </div>
        {/if}
      </div>
    {/if}

    <!-- Sort dropdown -->
    <div class="tb-menu-wrap" on:click|stopPropagation>
      <button class="tbtn tbtn-sort" class:active={showSortMenu} on:click={() => { showSortMenu = !showSortMenu; showViewPanel = false; }}>
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><circle cx="12" cy="12" r="10"/><polyline points="12,6 12,12 16,14"/></svg>
        {sortLabel($sortBy)}
        <svg class="chevron" width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><polyline points="6,9 12,15 18,9"/></svg>
      </button>
      {#if showSortMenu}
        <div class="dropdown sort-dropdown">
          <div class="dropdown-label">Sort by</div>
          {#each sortOptions as opt}
            <button
              class="dropdown-item"
              class:selected={$sortBy === opt.key}
              on:click={() => { sortBy.set(opt.key); showSortMenu = false; }}
            >
              <span class="radio-dot" class:radio-on={$sortBy === opt.key}></span>
              <svg class="sort-icon" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round">
                {#if opt.icon === 'clock'}<circle cx="12" cy="12" r="10"/><polyline points="12,6 12,12 16,14"/>
                {:else if opt.icon === 'az' || opt.icon === 'za'}<line x1="3" y1="6" x2="21" y2="6"/><line x1="3" y1="12" x2="15" y2="12"/><line x1="3" y1="18" x2="9" y2="18"/>
                {:else}<rect x="3" y="3" width="18" height="18" rx="2"/><line x1="3" y1="9" x2="21" y2="9"/><line x1="3" y1="15" x2="21" y2="15"/>{/if}
              </svg>
              {opt.label}
            </button>
          {/each}
        </div>
      {/if}
    </div>

    <!-- View toggle (List / Cards) -->
    <div class="view-toggle">
      <button class:active={$view === 'list'} on:click={() => view.set('list')} title="List">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
          <line x1="4" y1="6" x2="20" y2="6"/><line x1="4" y1="12" x2="20" y2="12"/><line x1="4" y1="18" x2="20" y2="18"/>
        </svg>
        List
      </button>
      <button class:active={$view === 'cards'} on:click={() => view.set('cards')} title="Cards">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
          <rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/><rect x="3" y="14" width="7" height="7"/><rect x="14" y="14" width="7" height="7"/>
        </svg>
        Cards
      </button>
    </div>

    <div class="toolbar-sep"></div>

    <!-- Unique URLs toggle -->
    <button class="tbtn tbtn-unique" class:active={$uniqueUrlsOnly}
      title={$uniqueUrlsOnly ? 'Showing unique URLs only (click to show all)' : 'Show all cross-profile duplicates'}
      on:click={() => uniqueUrlsOnly.update(v => !v)}>
      <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M8 3H5a2 2 0 0 0-2 2v3m18 0V5a2 2 0 0 0-2-2h-3m0 18h3a2 2 0 0 0 2-2v-3M3 16v3a2 2 0 0 0 2 2h3"/></svg>
      Unique
      {#if $uniqueUrlsOnly}<span class="filter-badge active">on</span>{/if}
    </button>

    <div class="toolbar-sep"></div>

    <!-- Export -->
    <button class="tbtn" disabled={$loading === 'analyze'} on:click={() => dispatch('export')}>
      <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="17,8 12,3 7,8"/><line x1="12" y1="3" x2="12" y2="15"/></svg>
      Export
    </button>
  </div>
</div>

<style>
  .toolbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 8px 14px;
    box-shadow: inset 0 -1px 0 var(--shadow-clr);
    background: var(--bg);
    gap: 10px;
  }
  .toolbar-left { display: flex; align-items: center; gap: 10px; flex: 1; min-width: 150px; }
  .toolbar-right { display: flex; align-items: center; gap: 5px; }
  .toolbar-sep { width: 1px; height: 18px; background: var(--border); margin: 0 4px; }

  .search-box {
    display: flex;
    align-items: center;
    gap: 6px;
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: 20px;
    padding: 5px 12px;
    flex: 1;
    max-width: 320px;
    transition: border-color .15s;
  }
  .search-box:focus-within { border-color: var(--accent); }
  .search-icon { color: var(--text-muted); flex-shrink: 0; }
  .search-box input {
    flex: 1;
    background: none;
    border: none;
    color: var(--text);
    font-size: 13px;
    outline: none;
    min-width: 0;
  }
  .search-box input::placeholder { color: var(--text-dim); }
  .search-clear {
    background: none;
    border: none;
    color: var(--text-muted);
    cursor: pointer;
    font-size: 15px;
    padding: 0 2px;
    line-height: 1;
  }
  .search-clear:hover { color: var(--text); }

  .result-count { font-size: 12px; color: var(--text-muted); white-space: nowrap; font-weight: 500; }
  .count-sep { color: var(--text-dim); margin: 0 1px; }

  /* ── Sort / View dropdowns ────── */
  .tb-menu-wrap { position: relative; }

  .tbtn {
    display: flex;
    align-items: center;
    gap: 5px;
    padding: 5px 10px;
    border-radius: var(--radius);
    border: 1px solid var(--border);
    background: transparent;
    color: var(--text-secondary);
    font-size: 12px;
    cursor: pointer;
    font-weight: 500;
    transition: all .12s;
    white-space: nowrap;
  }
  .tbtn:hover, .tbtn.active { border-color: var(--border-hover); color: var(--text); background: var(--bg-hover); }
  .tbtn-sort { color: var(--text-muted); }
  .chevron { flex-shrink: 0; color: var(--text-dim); }
  .tbtn:disabled { opacity: 0.3; pointer-events: none; }

  /* View toggle */
  .view-toggle {
    display: flex;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    overflow: hidden;
  }
  .view-toggle button {
    background: none;
    border: none;
    border-right: 1px solid var(--border);
    color: var(--text-muted);
    padding: 5px 9px;
    cursor: pointer;
    transition: all .1s;
    display: flex;
    align-items: center;
    gap: 4px;
    font-size: 12px;
  }
  .view-toggle button:last-child { border-right: none; }
  .view-toggle button:hover { color: var(--text-secondary); background: var(--bg-hover); }
  .view-toggle button.active { background: var(--accent-dim); color: var(--accent); }

  /* Dropdown */
  .dropdown {
    position: absolute;
    top: calc(100% + 6px);
    left: 0;
    min-width: 190px;
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    box-shadow: 0 8px 24px rgba(0,0,0,.28);
    z-index: 100;
    padding: 6px 0;
  }
  .dropdown-label {
    font-size: 11px;
    color: var(--text-dim);
    font-weight: 600;
    padding: 4px 14px 6px;
  }
  .dropdown-item {
    display: flex;
    align-items: center;
    gap: 8px;
    width: 100%;
    padding: 7px 14px;
    background: none;
    border: none;
    color: var(--text-secondary);
    font-size: 13px;
    cursor: pointer;
    text-align: left;
    transition: background .08s;
  }
  .dropdown-item:hover { background: var(--bg-hover); color: var(--text); }
  .dropdown-item.selected { color: var(--text); }

  .radio-dot {
    width: 14px;
    height: 14px;
    border-radius: 50%;
    border: 1.5px solid var(--border-hover);
    flex-shrink: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: border-color .1s;
  }
  .radio-on {
    border-color: var(--accent);
    background: var(--accent);
    box-shadow: inset 0 0 0 3px var(--bg-card);
  }
  .sort-icon { color: var(--text-dim); flex-shrink: 0; }

  /* ── Profile filter ─────────────────────────── */
  .tbtn-profiles { gap: 5px; }
  .has-filter { border-color: var(--accent); color: var(--accent); }
  .has-filter svg { opacity: 1; }
  .filter-badge {
    background: var(--accent);
    color: var(--bg);
    font-size: 10px;
    font-weight: 700;
    line-height: 1;
    padding: 1px 5px;
    border-radius: 8px;
    min-width: 16px;
    text-align: center;
  }

  /* Unique URLs toggle */
  .tbtn-unique.active {
    background: var(--accent-dim);
    color: var(--accent);
    border-color: var(--accent);
  }
  .tbtn-unique.active svg { opacity: 1; }
  .profile-dropdown { min-width: 230px; }
  .dropdown-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 4px 14px 6px;
  }
  .clear-filter {
    background: none;
    border: none;
    font-size: 11px;
    color: var(--accent);
    cursor: pointer;
    padding: 0;
    font-weight: 500;
  }
  .clear-filter:hover { text-decoration: underline; }

  .profile-item { gap: 7px; }
  .check-box {
    width: 15px;
    height: 15px;
    border-radius: 3px;
    border: 1.5px solid var(--border-hover);
    flex-shrink: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all .1s;
  }
  .check-box.checked {
    background: var(--accent);
    border-color: var(--accent);
    color: var(--bg);
  }

  .prof-badge {
    width: 17px;
    height: 17px;
    border-radius: 3px;
    font-size: 9px;
    font-weight: 700;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }
  .prof-chrome { background: #e8f0fe; color: #1a73e8; }
  .prof-edge   { background: #e3f2fd; color: #0078d4; }
  :global([data-theme="dark"]) .prof-chrome { background: rgba(66,133,244,.18); color: #8ab4f8; }
  :global([data-theme="dark"]) .prof-edge   { background: rgba(0,120,212,.18);  color: #6caef5; }
  :global([data-theme="sand"]) .prof-chrome { background: hsl(215,60%,88%); color: hsl(215,70%,35%); }
  :global([data-theme="sand"]) .prof-edge   { background: hsl(208,60%,88%); color: hsl(208,70%,32%); }

  .prof-name { flex: 1; font-size: 13px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .prof-sub  { font-size: 11px; color: var(--text-dim); white-space: nowrap; }
</style>