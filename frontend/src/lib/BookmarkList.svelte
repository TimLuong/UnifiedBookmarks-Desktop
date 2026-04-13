<script lang="ts">
  import { bookmarks, activePara, activeTag, activeCat, activeFolder, activeFolderUrls, searchQuery, view, sortBy, activeProfiles, uniqueUrlsOnly, openInApp } from './store';
  import type { Bookmark } from './store';

  function filterBookmarks(
    bms: Bookmark[], para: string | null, tag: string | null,
    cat: string | null, folder: string | null, folderUrls: Set<string> | null, search: string,
    profiles: Set<string> | null, uniqueOnly: boolean
  ): Bookmark[] {
    let out = bms;
    if (para) {
      if (para === 'inbox') out = out.filter(b => !b.paraType);
      else out = out.filter(b => b.paraType === para);
    }
    if (tag) out = out.filter(b => (b.tags || []).includes(tag));
    if (cat) out = out.filter(b => (b.category || '').startsWith(cat));
    if (folder) {
      if (folderUrls !== null) {
        // URL-set match: catches deduped bookmarks regardless of folderPath
        out = out.filter(b => folderUrls.has(b.url));
      } else {
        out = out.filter(b => (b.folderPath || '') === folder || (b.folderPath || '').startsWith(folder + '/'));
      }
    }
    if (profiles !== null) out = out.filter(b => profiles.has(`${b.browser}__${b.profileDir}`));
    if (search) {
      const q = search.toLowerCase();
      out = out.filter(b =>
        b.title.toLowerCase().includes(q) ||
        b.url.toLowerCase().includes(q) ||
        (b.category || '').toLowerCase().includes(q)
      );
    }
    // Deduplicate by URL across profiles — keep first occurrence (newest profile first by default)
    if (uniqueOnly) {
      const seen = new Set<string>();
      out = out.filter(b => {
        if (seen.has(b.url)) return false;
        seen.add(b.url);
        return true;
      });
    }
    return out;
  }

  function browserLabel(browser: string): string {
    if (browser === 'chrome') return 'Chrome';
    if (browser === 'edge') return 'Edge';
    return browser;
  }

  function browserBadgeChar(browser: string): string {
    if (browser === 'chrome') return 'C';
    if (browser === 'edge') return 'E';
    return browser[0]?.toUpperCase() ?? '?';
  }

  function sortBookmarks(bms: Bookmark[], sort: typeof $sortBy): Bookmark[] {
    const arr = [...bms];
    switch (sort) {
      case 'date-asc':
        return arr.sort((a, b) => (a.dateAdded || '').localeCompare(b.dateAdded || ''));
      case 'date-desc':
        return arr.sort((a, b) => (b.dateAdded || '').localeCompare(a.dateAdded || ''));
      case 'name-asc':
        return arr.sort((a, b) => a.title.localeCompare(b.title));
      case 'name-desc':
        return arr.sort((a, b) => b.title.localeCompare(a.title));
      case 'site-asc':
        return arr.sort((a, b) => getDomain(a.url).localeCompare(getDomain(b.url)));
      case 'site-desc':
        return arr.sort((a, b) => getDomain(b.url).localeCompare(getDomain(a.url)));
      default:
        return arr;
    }
  }

  function getFaviconUrl(url: string): string {
    try {
      const u = new URL(url);
      return `https://www.google.com/s2/favicons?domain=${u.hostname}&sz=32`;
    } catch { return ''; }
  }

  function getDomain(url: string): string {
    try { return new URL(url).hostname.replace('www.', ''); }
    catch { return ''; }
  }

  function getDomainColor(url: string): string {
    const domain = getDomain(url);
    let hash = 5381;
    for (let i = 0; i < domain.length; i++) hash = ((hash << 5) + hash) ^ domain.charCodeAt(i);
    const hue = Math.abs(hash) % 360;
    return `hsl(${hue}, 30%, 32%)`;
  }

  function getParaLabel(paraType: string): { label: string; cls: string } {
    switch (paraType) {
      case 'project': return { label: 'Project', cls: 'p-project' };
      case 'area': return { label: 'Area', cls: 'p-area' };
      case 'resource': return { label: 'Resource', cls: 'p-resource' };
      case 'archive': return { label: 'Archive', cls: 'p-archive' };
      default: return { label: 'Inbox', cls: 'p-inbox' };
    }
  }

  function confidenceColor(c: number | undefined): string {
    if (!c) return 'var(--text-dim)';
    if (c >= 0.8) return 'var(--green)';
    if (c >= 0.5) return 'var(--yellow)';
    return 'var(--red)';
  }

  function formatDate(dateAdded: string): string {
    if (!dateAdded) return '';
    try {
      const ts = parseInt(dateAdded);
      if (ts > 1e15) {
        const epoch = (ts / 1000) - 11644473600000;
        const d = new Date(epoch);
        if (d.getFullYear() > 2000) return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: '2-digit' });
      }
      const d = new Date(dateAdded);
      if (!isNaN(d.getTime())) return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: '2-digit' });
    } catch {}
    return '';
  }

  $: filtered = sortBookmarks(
    filterBookmarks($bookmarks, $activePara, $activeTag, $activeCat, $activeFolder, $activeFolderUrls, $searchQuery, $activeProfiles, $uniqueUrlsOnly),
    $sortBy
  );

  let expandedIndex: number | null = null;
  let copiedIndex: number | null = null;

  function toggleRow(i: number) {
    expandedIndex = expandedIndex === i ? null : i;
  }

  function copyUrl(url: string, i: number) {
    navigator.clipboard.writeText(url).then(() => {
      copiedIndex = i;
      setTimeout(() => { copiedIndex = null; }, 1500);
    });
  }

  function formatDateFull(dateAdded: string): string {
    if (!dateAdded) return '';
    try {
      const ts = parseInt(dateAdded);
      if (ts > 1e15) {
        const epoch = (ts / 1000) - 11644473600000;
        const d = new Date(epoch);
        if (d.getFullYear() > 2000) return d.toLocaleString('en-US', { month: 'short', day: 'numeric', year: 'numeric', hour: '2-digit', minute: '2-digit' });
      }
      const d = new Date(dateAdded);
      if (!isNaN(d.getTime())) return d.toLocaleString('en-US', { month: 'short', day: 'numeric', year: 'numeric', hour: '2-digit', minute: '2-digit' });
    } catch {}
    return '';
  }
</script>

<div class="bookmark-list" class:cards-view={$view === 'cards'}>
  {#if filtered.length === 0}
    <div class="empty-state">
      {#if $bookmarks.length === 0}
        <div class="empty-icon">
          <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1" stroke-linecap="round" stroke-linejoin="round"><path d="M19 21l-7-5-7 5V5a2 2 0 0 1 2-2h10a2 2 0 0 1 2 2z"/></svg>
        </div>
        <p class="empty-title">No bookmarks yet</p>
        <p class="empty-hint">Run <span class="cmd">Collect</span> then <span class="cmd">Analyze</span> to get started</p>
      {:else}
        <div class="empty-icon">
          <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
        </div>
        <p class="empty-title">No results</p>
        <p class="empty-hint">Try adjusting your filters or search query</p>
      {/if}
    </div>

  {:else if $view === 'list'}
    <!-- Feed-style list: no column headers, thumbnail + info -->
    {#each filtered as bm, i (bm.url + i)}
      <!-- svelte-ignore a11y-click-events-have-key-events -->
      <!-- svelte-ignore a11y-no-static-element-interactions -->
      <div class="bm-row" class:bm-row-expanded={expandedIndex === i} on:click={() => toggleRow(i)}>
        <!-- Thumbnail placeholder: domain-colored box with favicon -->
        <div class="bm-thumb" style="background:{getDomainColor(bm.url)}">
          <img class="bm-thumb-fav"
               src={getFaviconUrl(bm.url)} alt=""
               width="20" height="20"
               on:error={(e) => (e.currentTarget.style.display = 'none')} />
        </div>

        <!-- Info block: title + meta row + tags -->
        <div class="bm-info">
          <!-- svelte-ignore a11y-click-events-have-key-events -->
          <button class="bm-title" on:click|stopPropagation={() => openInApp.set({ url: bm.url, title: bm.title })}>{bm.title}</button>
          <div class="bm-meta">
            <span class="bm-domain">{getDomain(bm.url)}</span>
            {#if formatDate(bm.dateAdded)}
              <span class="bm-sep">·</span>
              <span class="bm-date">{formatDate(bm.dateAdded)}</span>
            {/if}
            {#if bm.folderPath}
              <span class="bm-sep">·</span>
              <span class="bm-cat">{bm.folderPath}</span>
            {/if}
            {#if bm.browser}
              <span class="bm-sep">·</span>
              <span class="bm-profile-badge bm-prof-{bm.browser}">{browserBadgeChar(bm.browser)}</span>
              <span class="bm-profile-name">{bm.displayName}</span>
            {/if}
          </div>
          {#if (bm.tags && bm.tags.length > 0) || bm.paraType}
            <div class="bm-tags-row">
              {#if bm.paraType}
                {@const p = getParaLabel(bm.paraType)}
                <span class="para-badge {p.cls}">{p.label}</span>
              {/if}
              {#each (bm.tags || []).slice(0, 3) as tag}
                <span class="mini-tag">#{tag}</span>
              {/each}
            </div>
          {/if}
        </div>

        <!-- Right: confidence + expand chevron -->
        <div class="bm-right">
          {#if bm.confidence}
            <span class="bm-conf" style="color:{confidenceColor(bm.confidence)}">
              {Math.round(bm.confidence * 100)}%
            </span>
          {/if}
          <svg class="bm-chevron" class:bm-chevron-open={expandedIndex === i}
               width="12" height="12" viewBox="0 0 24 24" fill="none"
               stroke="currentColor" stroke-width="2.5" stroke-linecap="round">
            <polyline points="6 9 12 15 18 9"/>
          </svg>
        </div>
      </div>

      {#if expandedIndex === i}
        <!-- svelte-ignore a11y-click-events-have-key-events -->
        <!-- svelte-ignore a11y-no-static-element-interactions -->
        <div class="bm-detail" on:click|stopPropagation>
          <!-- Full URL row -->
          <div class="bm-detail-url-row">
            <span class="bm-detail-full-url">{bm.url}</span>
            <button class="bm-copy-btn" title="Copy URL"
                    on:click={() => copyUrl(bm.url, i)}>
              {#if copiedIndex === i}
                <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round"><polyline points="20 6 9 17 4 12"/></svg>
                Copied
              {:else}
                <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>
                Copy
              {/if}
            </button>
            <button class="bm-open-btn" title="Open in app"
                    on:click={() => openInApp.set({ url: bm.url, title: bm.title })}>
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" y1="14" x2="21" y2="3"/></svg>
              Open
            </button>
          </div>
          <!-- Detail chips -->
          <div class="bm-detail-chips">
            {#if bm.folderPath}
              <span class="bm-detail-chip">
                <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/></svg>
                {bm.folderPath}
              </span>
            {/if}
            {#if formatDateFull(bm.dateAdded)}
              <span class="bm-detail-chip">
                <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><rect x="3" y="4" width="18" height="18" rx="2" ry="2"/><line x1="16" y1="2" x2="16" y2="6"/><line x1="8" y1="2" x2="8" y2="6"/><line x1="3" y1="10" x2="21" y2="10"/></svg>
                {formatDateFull(bm.dateAdded)}
              </span>
            {/if}
            {#if bm.browser}
              <span class="bm-detail-chip">
                <span class="bm-profile-badge bm-prof-{bm.browser}">{browserBadgeChar(bm.browser)}</span>
                {browserLabel(bm.browser)} · {bm.displayName}
              </span>
            {/if}
            {#if bm.category}
              <span class="bm-detail-chip">
                <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M20.59 13.41l-7.17 7.17a2 2 0 0 1-2.83 0L2 12V2h10l8.59 8.59a2 2 0 0 1 0 2.82z"/><line x1="7" y1="7" x2="7.01" y2="7"/></svg>
                {bm.category}
              </span>
            {/if}
            {#each (bm.tags || []) as tag}
              <span class="bm-detail-chip bm-detail-tag">#{tag}</span>
            {/each}
          </div>
        </div>
      {/if}
    {/each}

  {:else}
    <!-- Card grid view: thumbnail at top, info below -->
    {#each filtered as bm, i (bm.url + i)}
      <div class="bm-card">
        <!-- Card thumbnail area -->
        <div class="card-thumb" style="background:{getDomainColor(bm.url)}; min-height:80px">
          <img class="card-thumb-fav"
               src={getFaviconUrl(bm.url)} alt=""
               width="28" height="28"
               on:error={(e) => (e.currentTarget.style.display = 'none')} />
          {#if bm.paraType}
            {@const p = getParaLabel(bm.paraType)}
            <span class="card-para-badge {p.cls}">{p.label}</span>
          {/if}
        </div>
        <!-- Card body -->
        <div class="card-body">
          <div class="card-head">
            <img class="card-fav" src={getFaviconUrl(bm.url)} alt=""
                 width="14" height="14"
                 on:error={(e) => (e.currentTarget.style.display = 'none')} />
            <span class="card-domain">{getDomain(bm.url)}</span>
            {#if bm.confidence}
              <span class="card-conf" style="color:{confidenceColor(bm.confidence)}">
                {Math.round(bm.confidence * 100)}%
              </span>
            {/if}
          </div>
          <button class="card-title" on:click={() => openInApp.set({ url: bm.url, title: bm.title })}>{bm.title}</button>
          {#if bm.category}
            <div class="card-cat">{bm.category}</div>
          {/if}
          {#if bm.tags && bm.tags.length > 0}
            <div class="card-tags">
              {#each bm.tags.slice(0, 3) as tag}
                <span class="mini-tag">#{tag}</span>
              {/each}
            </div>
          {/if}
          {#if formatDate(bm.dateAdded)}
            <div class="card-date">{formatDate(bm.dateAdded)}</div>
          {/if}
        </div>
      </div>
    {/each}
  {/if}
</div>

<style>
  .bookmark-list {
    flex: 1;
    min-height: 0;
    overflow-y: auto;
    padding: 0;
  }
  .bookmark-list.cards-view {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
    grid-auto-rows: max-content;
    gap: 12px;
    padding: 16px;
    align-content: start;
  }

  /* ── EMPTY STATE ───────────────────────────────────────── */
  .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 60%;
    color: var(--text-dim);
    gap: 8px;
  }
  .empty-icon { opacity: 0.25; }
  .empty-title { font-size: 15px; font-weight: 600; color: var(--text-muted); margin: 0; }
  .empty-hint { font-size: 13px; color: var(--text-dim); margin: 0; }
  .cmd {
    color: var(--accent);
    background: var(--accent-dim);
    padding: 1px 6px;
    border-radius: 4px;
    font-weight: 500;
  }

  /* ── LIST VIEW: FEED ROWS ──────────────────────────────── */
  .bm-row {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 10px 16px;
    box-shadow: inset 0 -1px 0 var(--divider-clr);
    transition: background .08s;
    cursor: pointer;
    min-height: 48px;
  }
  .bm-row:hover { background: var(--bg-hover); }

  /* Thumbnail: colored domain box with favicon centered */
  .bm-thumb {
    width: 56px;
    height: 40px;
    border-radius: var(--radius);
    flex-shrink: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    overflow: hidden;
    opacity: 0.85;
  }
  .bm-thumb-fav {
    filter: brightness(0) invert(1);
    opacity: 0.7;
  }

  /* Info block */
  .bm-info {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 3px;
  }
  .bm-title {
    font-size: 13px;
    font-weight: 500;
    color: var(--text);
    text-decoration: none;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    line-height: 1.35;
    background: none;
    border: none;
    padding: 0;
    margin: 0;
    cursor: pointer;
    text-align: left;
    font-family: inherit;
    max-width: 100%;
  }
  .bm-title:hover { color: var(--accent); }

  /* Meta row: domain · date · category */
  .bm-meta {
    display: flex;
    align-items: center;
    gap: 4px;
    flex-wrap: nowrap;
    overflow: hidden;
  }
  .bm-domain {
    font-size: 12px;
    color: var(--text-muted);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    max-width: 140px;
  }
  .bm-sep { font-size: 11px; color: var(--text-dim); flex-shrink: 0; }
  .bm-date { font-size: 11px; color: var(--text-dim); white-space: nowrap; flex-shrink: 0; }
  .bm-cat {
    font-size: 11px;
    color: var(--text-dim);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    max-width: 160px;
  }

  /* Profile badge inline in meta row */
  .bm-profile-badge {
    width: 14px;
    height: 14px;
    border-radius: 3px;
    font-size: 8px;
    font-weight: 700;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    line-height: 1;
  }
  .bm-prof-chrome { background: #e8f0fe; color: #1a73e8; }
  .bm-prof-edge   { background: #e3f2fd; color: #0078d4; }
  :global([data-theme="dark"]) .bm-prof-chrome { background: rgba(66,133,244,.18); color: #8ab4f8; }
  :global([data-theme="dark"]) .bm-prof-edge   { background: rgba(0,120,212,.18);  color: #6caef5; }
  :global([data-theme="sand"]) .bm-prof-chrome { background: hsl(215,60%,88%); color: hsl(215,70%,35%); }
  :global([data-theme="sand"]) .bm-prof-edge   { background: hsl(208,60%,88%); color: hsl(208,70%,32%); }
  .bm-profile-name {
    font-size: 11px;
    color: var(--text-dim);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    max-width: 100px;
  }

  /* Tags row */
  .bm-tags-row {
    display: flex;
    align-items: center;
    gap: 4px;
    flex-wrap: nowrap;
    overflow: hidden;
    margin-top: 1px;
  }
  .mini-tag {
    font-size: 11px;
    color: var(--text-muted);
    padding: 1px 7px;
    border: 1px solid var(--border);
    border-radius: 20px;
    white-space: nowrap;
    flex-shrink: 0;
  }

  /* Para badges */
  .para-badge {
    font-size: 10px;
    font-weight: 600;
    padding: 1px 6px;
    border-radius: 4px;
    white-space: nowrap;
    flex-shrink: 0;
  }
  .p-project { color: var(--accent); background: var(--accent-dim); }
  .p-area { color: var(--green); background: var(--green-dim); }
  .p-resource { color: var(--cyan); background: var(--cyan-dim); }
  .p-archive { color: var(--text-muted); background: var(--bg-hover); }
  .p-inbox { color: var(--orange); background: var(--orange-dim); }

  /* Right column: conf + chevron */
  .bm-right {
    display: flex;
    align-items: center;
    gap: 6px;
    flex-shrink: 0;
  }

  /* Confidence score */
  .bm-conf {
    font-size: 11px;
    font-weight: 600;
    flex-shrink: 0;
    min-width: 32px;
    text-align: right;
  }

  /* Chevron */
  .bm-chevron {
    color: var(--text-dim);
    flex-shrink: 0;
    transition: transform .15s ease;
    opacity: 0.5;
  }
  .bm-chevron-open { transform: rotate(180deg); opacity: 1; color: var(--accent); }
  .bm-row:hover .bm-chevron { opacity: 0.8; }
  .bm-row-expanded { background: var(--bg-hover); }

  /* ── DETAIL PANEL ──────────────────────── */
  .bm-detail {
    padding: 8px 16px 12px 84px; /* indent past thumb */
    background: var(--bg-secondary);
    border-bottom: 1px solid var(--divider-clr);
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .bm-detail-url-row {
    display: flex;
    align-items: center;
    gap: 6px;
    flex-wrap: nowrap;
    overflow: hidden;
  }

  .bm-detail-full-url {
    flex: 1;
    min-width: 0;
    font-size: 11.5px;
    color: var(--text-muted);
    font-family: monospace;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    user-select: text;
  }

  .bm-copy-btn, .bm-open-btn {
    display: flex;
    align-items: center;
    gap: 4px;
    padding: 3px 9px;
    border-radius: var(--radius-sm);
    font-size: 11px;
    font-weight: 500;
    cursor: pointer;
    white-space: nowrap;
    flex-shrink: 0;
    font-family: inherit;
    transition: background .1s, color .1s;
  }
  .bm-copy-btn {
    background: var(--bg-hover);
    border: 1px solid var(--border);
    color: var(--text-secondary);
  }
  .bm-copy-btn:hover { background: var(--border); }
  .bm-open-btn {
    background: var(--accent);
    border: 1px solid var(--accent);
    color: #fff;
  }
  .bm-open-btn:hover { opacity: .85; }

  .bm-detail-chips {
    display: flex;
    align-items: center;
    gap: 6px;
    flex-wrap: wrap;
  }

  .bm-detail-chip {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    font-size: 11px;
    color: var(--text-muted);
    background: var(--bg-card);
    border: 1px solid var(--border);
    padding: 2px 8px;
    border-radius: 20px;
    white-space: nowrap;
  }

  .bm-detail-tag {
    color: var(--accent);
    border-color: var(--accent-dim);
    background: var(--accent-dim);
  }

  /* ── CARD VIEW ─────────────────────────────────────────── */
  .bm-card {
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    display: flex;
    flex-direction: column;
    min-height: 160px;
    transition: border-color .12s, box-shadow .12s;
  }
  .bm-card:hover {
    border-color: var(--border-hover);
    box-shadow: 0 4px 16px rgba(0,0,0,.18);
  }

  /* Card thumbnail: colored area with favicon centered */
  .card-thumb {
    width: 100%;
    min-height: 80px;
    display: flex;
    align-items: center;
    justify-content: center;
    position: relative;
    flex-shrink: 0;
    border-radius: var(--radius) var(--radius) 0 0;
    overflow: hidden;
  }
  .card-thumb-fav {
    filter: brightness(0) invert(1);
    opacity: 0.5;
    width: 32px;
    height: 32px;
  }
  .card-para-badge {
    position: absolute;
    top: 8px;
    right: 8px;
    font-size: 10px;
    font-weight: 600;
    padding: 2px 7px;
    border-radius: 4px;
    backdrop-filter: blur(4px);
  }

  /* Card body */
  .card-body {
    padding: 10px 12px;
    display: flex;
    flex-direction: column;
    gap: 4px;
    flex: 1;
  }
  .card-head {
    display: flex;
    align-items: center;
    gap: 5px;
    margin-bottom: 2px;
  }
  .card-fav { border-radius: 2px; flex-shrink: 0; }
  .card-domain {
    font-size: 11px;
    color: var(--text-muted);
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .card-conf { font-size: 11px; font-weight: 600; flex-shrink: 0; }
  .card-title {
    font-size: 13px;
    color: var(--text);
    font-weight: 500;
    text-decoration: none;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
    line-height: 1.4;
    background: none;
    border: none;
    padding: 0;
    margin: 0;
    cursor: pointer;
    text-align: left;
    font-family: inherit;
    width: 100%;
  }
  .card-title:hover { color: var(--accent); }
  .card-cat { font-size: 11px; color: var(--text-dim); }
  .card-tags { display: flex; gap: 4px; flex-wrap: wrap; }
  .card-date { font-size: 11px; color: var(--text-dim); margin-top: auto; padding-top: 4px; }
</style>