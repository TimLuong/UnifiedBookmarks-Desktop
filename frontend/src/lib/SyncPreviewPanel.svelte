<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import type { SyncPreviewResult } from './store';

  export let visible = false;
  export let preview: SyncPreviewResult | null = null;

  const dispatch = createEventDispatcher();
  // @ts-ignore
  const callGo = (method: string, ...args: any[]) => window['go']['main']['App'][method](...args);

  interface BookmarkDiffRow {
    title: string; url: string; domain: string; folderPath: string;
    category: string; status: 'added' | 'removed' | 'unchanged';
  }
  interface FolderSection {
    path: string; name: string; depth: number; items: BookmarkDiffRow[];
    totalItems: number;
  }
  interface ProfileDiffDetail {
    browser: string; browserLabel: string; profileDir: string; displayName: string;
    before: BookmarkDiffRow[]; after: BookmarkDiffRow[];
    added: number; removed: number; unchanged: number;
  }

  $: diffs = preview?.diffs || [];

  let selectedDir = '';
  let detail: ProfileDiffDetail | null = null;
  let loadingDetail = false;
  let leftFilter: 'all' | 'removed' | 'unchanged' = 'all';
  let rightFilter: 'all' | 'added' | 'unchanged' = 'all';
  let leftSearch = '';
  let rightSearch = '';
  let selectedProfiles = new Set<string>();

  // Composite key for unique profile identity across browsers (e.g. chrome:Default vs edge:Default)
  function profileKey(d: { browser: string; profileDir: string }) { return `${d.browser}:${d.profileDir}`; }

  // Auto-select first profile when panel opens
  $: if (visible && diffs.length > 0 && !selectedDir) {
    selectedDir = profileKey(diffs[0]);
    selectedProfiles = new Set(diffs.map(profileKey));
    loadDetail(selectedDir);
  }

  // Reset state when closed
  $: if (!visible) { selectedDir = ''; detail = null; leftFilter = 'all'; rightFilter = 'all'; leftSearch = ''; rightSearch = ''; highlightedUrl = ''; leftCrumb = ''; rightCrumb = ''; }

  async function loadDetail(profileKey: string) {
    if (!profileKey) return;
    loadingDetail = true;
    detail = null;
    try { detail = await callGo('GetProfileDiffDetail', profileKey); }
    catch(e) { console.error('GetProfileDiffDetail failed', e); }
    loadingDetail = false;
    // Trigger breadcrumb update after content renders
    requestAnimationFrame(() => {
      requestAnimationFrame(() => {
        updateCrumb(leftListEl, 'l');
        updateCrumb(rightListEl, 'r');
      });
    });
  }

  function pickProfile(key: string) {
    selectedDir = key;
    leftFilter = 'all'; rightFilter = 'all';
    leftSearch = ''; rightSearch = '';
    highlightedUrl = '';
    leftCrumb = ''; rightCrumb = '';
    loadDetail(key);
  }

  function toggleInclude(key: string) {
    const s = new Set(selectedProfiles);
    s.has(key) ? s.delete(key) : s.add(key);
    selectedProfiles = s;
  }

  function confirmSync() {
    dispatch('confirm', { selectedProfiles: [...selectedProfiles] });
  }

  $: activeDiff = diffs.find(d => profileKey(d) === selectedDir);

  $: filteredBefore = detail ? detail.before.filter(bm => {
    if (leftFilter === 'removed' && bm.status !== 'removed') return false;
    if (leftFilter === 'unchanged' && bm.status !== 'unchanged') return false;
    const q = leftSearch.toLowerCase();
    if (q && !bm.title.toLowerCase().includes(q) && !bm.domain.toLowerCase().includes(q)) return false;
    return true;
  }) : [];

  $: filteredAfter = detail ? detail.after.filter(bm => {
    if (rightFilter === 'added'     && bm.status !== 'added')     return false;
    if (rightFilter === 'unchanged' && bm.status !== 'unchanged') return false;
    const q = rightSearch.toLowerCase();
    if (q && !bm.title.toLowerCase().includes(q) && !bm.domain.toLowerCase().includes(q)) return false;
    return true;
  }) : [];

  // ── Folder tree helpers ──────────────────────────────

  function buildSections(rows: BookmarkDiffRow[]): FolderSection[] {
    const leafMap = new Map<string, BookmarkDiffRow[]>();
    for (const r of rows) {
      const p = r.folderPath || '';
      if (!leafMap.has(p)) leafMap.set(p, []);
      leafMap.get(p)!.push(r);
    }

    // Collect all paths including intermediate parent paths
    const allPaths = new Set<string>();
    for (const path of leafMap.keys()) {
      if (!path) { allPaths.add(''); continue; }
      const parts = path.split('/').filter(Boolean);
      for (let i = 1; i <= parts.length; i++) {
        allPaths.add(parts.slice(0, i).join('/'));
      }
    }

    const sections: FolderSection[] = [];
    for (const path of allPaths) {
      const parts = path ? path.split('/').filter(Boolean) : [];
      const directItems = leafMap.get(path) || [];
      let totalItems = 0;
      for (const [k, v] of leafMap) {
        if (k === path || k.startsWith(path + '/')) totalItems += v.length;
      }
      sections.push({
        path,
        name: parts.length > 0 ? parts[parts.length - 1] : '/',
        depth: Math.max(0, parts.length - 1),
        items: directItems,
        totalItems,
      });
    }
    sections.sort((a, b) => a.path.localeCompare(b.path));
    return sections;
  }

  // For AFTER pane: group by LLM-assigned category hierarchy
  function buildCategorySections(rows: BookmarkDiffRow[]): FolderSection[] {
    // Group items by their full category path
    const leafMap = new Map<string, BookmarkDiffRow[]>();
    for (const r of rows) {
      const p = r.category && r.category !== 'Uncategorized' ? r.category : 'Uncategorized';
      if (!leafMap.has(p)) leafMap.set(p, []);
      leafMap.get(p)!.push(r);
    }

    // Collect all paths including intermediate parent paths
    const allPaths = new Set<string>();
    for (const path of leafMap.keys()) {
      const parts = path.split('/').filter(Boolean);
      for (let i = 1; i <= parts.length; i++) {
        allPaths.add(parts.slice(0, i).join('/'));
      }
    }

    const sections: FolderSection[] = [];
    for (const path of allPaths) {
      const parts = path.split('/').filter(Boolean);
      const directItems = leafMap.get(path) || [];
      // Count all items in this path AND all descendant paths
      let totalItems = 0;
      for (const [k, v] of leafMap) {
        if (k === path || k.startsWith(path + '/')) totalItems += v.length;
      }
      sections.push({
        path,
        name: parts.length > 0 ? parts[parts.length - 1] : '(uncategorized)',
        depth: Math.max(0, parts.length - 1),
        items: directItems,
        totalItems,
      });
    }
    sections.sort((a, b) => a.path.localeCompare(b.path));
    return sections;
  }

  let leftCollapsed  = new Set<string>();
  let rightCollapsed = new Set<string>();
  let highlightedUrl = '';

  let leftListEl: HTMLElement;
  let rightListEl: HTMLElement;
  let leftCrumb = '';
  let rightCrumb = '';

  // When an item is highlighted, scroll-based crumb updates are suppressed —
  // the highlighted item's crumb takes priority over scroll position.
  function updateCrumb(listEl: HTMLElement, side: 'l' | 'r') {
    if (!listEl) return;
    // If this pane has a highlighted item, its crumb is already set — don't override.
    if (highlightedUrl) {
      const hasSideHl = Array.from(
        listEl.querySelectorAll<HTMLElement>('[data-url-side]')
      ).some(e => e.dataset.urlSide === `${side}-${highlightedUrl}`);
      if (hasSideHl) return;
    }
    const headers = listEl.querySelectorAll<HTMLElement>('[data-section-path]');
    const scrollTop = listEl.scrollTop;
    let current = '';
    for (const h of headers) {
      if (h.offsetTop <= scrollTop + 24) {
        current = h.dataset.sectionPath || '';
      }
    }
    if (side === 'l') leftCrumb = current;
    else rightCrumb = current;
  }

  function crumbParts(path: string): string[] {
    return path ? path.split('/').filter(Boolean) : [];
  }

  function highlightRow(url: string, fromSide: 'l' | 'r', sectionPath: string) {
    highlightedUrl = highlightedUrl === url ? '' : url;
    if (highlightedUrl) {
      // Update source pane breadcrumb immediately from passed sectionPath
      if (fromSide === 'l') leftCrumb = sectionPath;
      else rightCrumb = sectionPath;

      // Update target pane breadcrumb from the reactive map (no DOM lookup needed)
      const targetSide = fromSide === 'l' ? 'r' : 'l';
      const targetCrumb = (targetSide === 'r' ? rightUrlCrumb : leftUrlCrumb).get(highlightedUrl) ?? '';
      if (targetSide === 'l') leftCrumb = targetCrumb;
      else rightCrumb = targetCrumb;

      // Scroll target row into view
      const targetAttr = `${targetSide}-${highlightedUrl}`;
      requestAnimationFrame(() => {
        const all = document.querySelectorAll<HTMLElement>('[data-url-side]');
        const el = Array.from(all).find(e => e.dataset.urlSide === targetAttr);
        el?.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
      });
    }
  }

  function toggleFolder(side: 'l' | 'r', path: string) {
    if (side === 'l') {
      const s = new Set(leftCollapsed);
      s.has(path) ? s.delete(path) : s.add(path);
      leftCollapsed = s;
    } else {
      const s = new Set(rightCollapsed);
      s.has(path) ? s.delete(path) : s.add(path);
      rightCollapsed = s;
    }
  }

  function collapseAll(side: 'l' | 'r', sections: FolderSection[]) {
    const paths = new Set(sections.map(s => s.path));
    side === 'l' ? (leftCollapsed = paths) : (rightCollapsed = paths);
  }

  function expandAll(side: 'l' | 'r') {
    side === 'l' ? (leftCollapsed = new Set()) : (rightCollapsed = new Set());
  }

  $: leftSections  = buildSections(filteredBefore);
  $: rightSections = buildCategorySections(filteredAfter);

  // URL → section path lookup maps (one entry per url, last-write-wins for dupes
  // within a pane — same as what the user sees highlighted)
  $: leftUrlCrumb  = buildUrlCrumbMap(leftSections);
  $: rightUrlCrumb = buildUrlCrumbMap(rightSections);

  function buildUrlCrumbMap(sections: FolderSection[]): Map<string, string> {
    const m = new Map<string, string>();
    for (const sec of sections) {
      for (const item of sec.items) {
        m.set(item.url, sec.path);
      }
    }
    return m;
  }
</script>

{#if visible && preview}
<div class="sc-overlay"
     on:click|self={() => dispatch('close')}
     on:keydown|self={(e) => e.key === 'Escape' && dispatch('close')}
     role="dialog" aria-modal="true" tabindex="-1">
  <div class="sc-window">

    <!-- ── Header ── -->
    <div class="sc-header">
      <span class="sc-icon">⇄</span>
      <span class="sc-title">sync commander</span>
      <span class="sc-subtitle">review changes per profile, then confirm sync</span>
      <button class="sc-close" on:click={() => dispatch('close')}>✕</button>
    </div>

    <!-- ── Body: 3-pane ── -->
    <div class="sc-body">

      <!-- PROFILES SIDEBAR -->
      <div class="sc-profiles">
        <div class="sc-profiles-hdr">
          <span class="sc-profiles-label">BROWSER PROFILES</span>
          <div class="sc-profiles-selbtns">
            <button class="sc-selall" on:click={() => selectedProfiles = new Set(diffs.map(profileKey))}>all</button>
            <button class="sc-selall" on:click={() => selectedProfiles = new Set()}>none</button>
          </div>
        </div>
        <div class="sc-profiles-list">
          {#each diffs as diff}
            <div class="sc-profile-row" class:active={selectedDir === profileKey(diff)}
                 on:click={() => pickProfile(profileKey(diff))}
                 on:keydown={(e) => e.key === 'Enter' && pickProfile(profileKey(diff))}
                 role="button" tabindex="0">
              <div class="sc-pr-check">
                <input type="checkbox"
                       checked={selectedProfiles.has(profileKey(diff))}
                       on:change|stopPropagation={() => toggleInclude(profileKey(diff))}
                       on:click|stopPropagation />
              </div>
              <div class="sc-pr-info">
                <div class="sc-pr-browser">{diff.browserLabel}</div>
                <div class="sc-pr-name">{diff.displayName}</div>
                <div class="sc-pr-stats">
                  <span class="sc-pr-count">{diff.beforeCount} → {diff.afterCount}</span>
                  {#if diff.added > 0}<span class="sc-pr-add">+{diff.added}</span>{/if}
                  {#if diff.removed > 0}<span class="sc-pr-rm">−{diff.removed}</span>{/if}
                  {#if diff.added === 0 && diff.removed === 0}<span class="sc-pr-sync">in sync</span>{/if}
                </div>
              </div>
            </div>
          {/each}
        </div>
      </div>

      <!-- LEFT: before -->
      <div class="sc-pane sc-pane-l">
        <div class="sc-pane-head sc-pane-head-l">
          <div class="sc-pane-label">
            <span class="sc-pane-badge sc-pane-badge-l">B</span>
            BEFORE
          </div>
          <span class="sc-pane-count">{detail?.before.length ?? '—'} bm</span>
        </div>

        <div class="sc-pane-toolbar">
          <input class="sc-search" type="text" placeholder="search…" bind:value={leftSearch} />
          <div class="sc-fgroup">
            <button class="sc-f" class:active={leftFilter==='all'} on:click={() => leftFilter='all'}>all</button>
            <button class="sc-f sc-f-rm" class:active={leftFilter==='removed'} on:click={() => leftFilter='removed'}>−{detail?.removed ?? 0}</button>
            <button class="sc-f" class:active={leftFilter==='unchanged'} on:click={() => leftFilter='unchanged'}>=</button>
          </div>
          <div class="sc-fold-btns">
            <button class="sc-fold-btn" on:click={() => expandAll('l')} title="expand all">⊕</button>
            <button class="sc-fold-btn" on:click={() => collapseAll('l', leftSections)} title="collapse all">⊖</button>
          </div>
        </div>

        {#if leftCrumb}
          <div class="sc-breadcrumb">
            {#each crumbParts(leftCrumb) as part, i}
              {#if i > 0}<span class="sc-crumb-sep">›</span>{/if}
              <span class="sc-crumb-part">{part}</span>
            {/each}
          </div>
        {/if}

        <div class="sc-list" bind:this={leftListEl} on:scroll={() => { highlightedUrl = ''; updateCrumb(leftListEl, 'l'); }}>
          {#if loadingDetail}
            <div class="sc-spinner">loading…</div>
          {:else if leftSections.length === 0}
            <div class="sc-empty">no items</div>
          {:else}
            {#each leftSections as section}
              <button class="sc-folder-hdr"
                      style="padding-left: {8 + section.depth * 14}px"
                      class:has-change={section.items.some(i => i.status === 'removed')}
                      data-section-path={section.path}
                      on:click={() => toggleFolder('l', section.path)}>
                <span class="sc-fhdr-chevron">{leftCollapsed.has(section.path) ? '▶' : '▼'}</span>
                <span class="sc-fhdr-icon">📁</span>
                <span class="sc-fhdr-name">{section.name}</span>
                <span class="sc-fhdr-count">{section.totalItems}</span>
                {#if section.items.some(i => i.status === 'removed')}
                  <span class="sc-fhdr-badge sc-fhdr-badge-rm">−{section.items.filter(i=>i.status==='removed').length}</span>
                {/if}
              </button>
              {#if !leftCollapsed.has(section.path)}
                {#each section.items as bm}
                  <div class="sc-row"
                       class:sc-row-rm={bm.status==='removed'}
                       class:sc-row-ok={bm.status==='unchanged'}
                       class:sc-row-hl={highlightedUrl === bm.url}
                       style="padding-left: {22 + section.depth * 14}px"
                       data-url-side="l-{bm.url}"
                       data-crumb-path={section.path}
                       role="button" tabindex="0"
                       on:click={() => highlightRow(bm.url, 'l', section.path)}
                       on:keydown={(e) => e.key === 'Enter' && highlightRow(bm.url, 'l', section.path)}>
                    <span class="sc-row-icon">{bm.status==='removed' ? '✗' : '○'}</span>
                    <span class="sc-row-title">{bm.title || '(untitled)'}</span>
                    <span class="sc-row-domain">{bm.domain}</span>
                  </div>
                {/each}
              {/if}
            {/each}
          {/if}
        </div>
      </div>

      <!-- DIVIDER -->
      <div class="sc-divider">
        <div class="sc-div-line"></div>
        <span class="sc-div-icon">⟹</span>
        <div class="sc-div-line"></div>
      </div>

      <!-- RIGHT: after -->
      <div class="sc-pane sc-pane-r">
        <div class="sc-pane-head sc-pane-head-r">
          <div class="sc-pane-label">
            <span class="sc-pane-badge sc-pane-badge-r">A</span>
            AFTER
          </div>
          <span class="sc-pane-count">{detail?.after.length ?? '—'} bm</span>
        </div>

        <div class="sc-pane-toolbar">
          <input class="sc-search" type="text" placeholder="search…" bind:value={rightSearch} />
          <div class="sc-fgroup">
            <button class="sc-f" class:active={rightFilter==='all'} on:click={() => rightFilter='all'}>all</button>
            <button class="sc-f sc-f-add" class:active={rightFilter==='added'} on:click={() => rightFilter='added'}>+{detail?.added ?? 0}</button>
            <button class="sc-f" class:active={rightFilter==='unchanged'} on:click={() => rightFilter='unchanged'}>=</button>
          </div>
          <div class="sc-fold-btns">
            <button class="sc-fold-btn" on:click={() => expandAll('r')} title="expand all">⊕</button>
            <button class="sc-fold-btn" on:click={() => collapseAll('r', rightSections)} title="collapse all">⊖</button>
          </div>
        </div>

        {#if rightCrumb}
          <div class="sc-breadcrumb">
            {#each crumbParts(rightCrumb) as part, i}
              {#if i > 0}<span class="sc-crumb-sep">›</span>{/if}
              <span class="sc-crumb-part">{part}</span>
            {/each}
          </div>
        {/if}

        <div class="sc-list" bind:this={rightListEl} on:scroll={() => { highlightedUrl = ''; updateCrumb(rightListEl, 'r'); }}>
          {#if loadingDetail}
            <div class="sc-spinner">loading…</div>
          {:else if rightSections.length === 0}
            <div class="sc-empty">no items</div>
          {:else}
            {#each rightSections as section}
              <button class="sc-folder-hdr"
                      style="padding-left: {8 + section.depth * 14}px"
                      class:has-change={section.items.some(i => i.status === 'added')}
                      data-section-path={section.path}
                      on:click={() => toggleFolder('r', section.path)}>
                <span class="sc-fhdr-chevron">{rightCollapsed.has(section.path) ? '▶' : '▼'}</span>
                <span class="sc-fhdr-icon">📁</span>
                <span class="sc-fhdr-name">{section.name}</span>
                <span class="sc-fhdr-count">{section.totalItems}</span>
                {#if section.items.some(i => i.status === 'added')}
                  <span class="sc-fhdr-badge sc-fhdr-badge-add">+{section.items.filter(i=>i.status==='added').length}</span>
                {/if}
              </button>
              {#if !rightCollapsed.has(section.path)}
                {#each section.items as bm}
                  <div class="sc-row"
                       class:sc-row-add={bm.status==='added'}
                       class:sc-row-ok={bm.status==='unchanged'}
                       class:sc-row-hl={highlightedUrl === bm.url}
                       style="padding-left: {22 + section.depth * 14}px"
                       data-url-side="r-{bm.url}"
                       data-crumb-path={section.path}
                       role="button" tabindex="0"
                       on:click={() => highlightRow(bm.url, 'r', section.path)}
                       on:keydown={(e) => e.key === 'Enter' && highlightRow(bm.url, 'r', section.path)}>
                    <span class="sc-row-icon">{bm.status==='added' ? '+' : '○'}</span>
                    <span class="sc-row-title">{bm.title || '(untitled)'}</span>
                    <span class="sc-row-domain">{bm.domain}</span>
                  </div>
                {/each}
              {/if}
            {/each}
          {/if}
        </div>
      </div>

    </div><!-- end body -->

    <!-- ── Footer ── -->
    <div class="sc-footer">
      <span class="sc-footer-hint">
        {selectedProfiles.size === 0
          ? 'No profiles selected — check profiles in the sidebar'
          : `${selectedProfiles.size} of ${diffs.length} profile${diffs.length !== 1 ? 's' : ''} will be synced`}
      </span>
      <div class="sc-footer-btns">
        <button class="sc-btn sc-btn-cancel" on:click={() => dispatch('close')}>cancel</button>
        <button class="sc-btn sc-btn-confirm" disabled={selectedProfiles.size === 0} on:click={confirmSync}>
          sync {selectedProfiles.size} profile{selectedProfiles.size !== 1 ? 's' : ''} →
        </button>
      </div>
    </div>

  </div>
</div>
{/if}

<style>
  .sc-overlay {
    position: fixed; inset: 0;
    background: rgba(0,0,0,0.6);
    display: flex; align-items: center; justify-content: center;
    z-index: 1000;
    backdrop-filter: blur(3px);
  }
  .sc-window {
    background: var(--bg);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    width: 94vw; height: 90vh;
    display: flex; flex-direction: column;
    box-shadow: 0 24px 80px rgba(0,0,0,0.55);
    overflow: hidden;
  }

  /* ── Header ── */
  .sc-header {
    display: flex; align-items: center; gap: 8px;
    padding: 8px 12px;
    border-bottom: 1px solid var(--border);
    background: var(--bg-deep);
    flex-shrink: 0;
  }
  .sc-icon   { color: var(--blue); font-size: 14px; flex-shrink: 0; }
  .sc-title  {
    font-size: 11px; font-weight: 700; letter-spacing: .5px;
    color: var(--text-secondary); white-space: nowrap;
  }
  .sc-subtitle {
    font-size: 10px; color: var(--text-dim);
    flex: 1; white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
  }
  .sc-close {
    background: none; border: none; color: var(--text-dim);
    cursor: pointer; font-size: 12px; padding: 4px 6px;
    border-radius: var(--radius-sm); transition: all .1s; flex-shrink: 0;
  }
  .sc-close:hover { background: var(--bg-hover); color: var(--text-secondary); }

  /* ── Body ── */
  .sc-body {
    display: flex; flex: 1; min-height: 0;
  }

  /* ── Profiles sidebar ── */
  .sc-profiles {
    width: 190px; flex-shrink: 0;
    display: flex; flex-direction: column;
    border-right: 1px solid var(--border);
    background: var(--bg-deep);
  }
  .sc-profiles-hdr {
    display: flex; align-items: center; justify-content: space-between;
    padding: 6px 10px;
    border-bottom: 1px solid var(--border);
    flex-shrink: 0;
  }
  .sc-profiles-label {
    font-size: 9px; font-weight: 700; letter-spacing: .8px;
    color: var(--text-muted); text-transform: uppercase;
  }
  .sc-profiles-selbtns { display: flex; gap: 3px; }
  .sc-selall {
    font-size: 9px; padding: 1px 5px;
    background: none; border: 1px solid var(--border);
    border-radius: var(--radius-sm); color: var(--text-dim);
    cursor: pointer; transition: all .1s;
  }
  .sc-selall:hover { border-color: var(--border-hover); color: var(--text-muted); }

  .sc-profiles-list {
    flex: 1; overflow-y: auto;
  }
  .sc-profile-row {
    display: flex; align-items: center; gap: 6px;
    padding: 7px 8px;
    border-bottom: 1px solid var(--border);
    cursor: pointer; transition: background .1s;
    user-select: none;
  }
  .sc-profile-row:hover { background: var(--bg-hover); }
  .sc-profile-row.active {
    background: var(--blue-dim);
    border-left: 2px solid var(--blue);
  }
  .sc-pr-check { flex-shrink: 0; }
  .sc-pr-check input { cursor: pointer; accent-color: var(--blue); }
  .sc-pr-info { flex: 1; min-width: 0; }
  .sc-pr-browser {
    font-size: 10px; font-weight: 600; color: var(--text-secondary);
    white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
  }
  .sc-pr-name {
    font-size: 9px; color: var(--text-muted);
    white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
    margin-top: 1px;
  }
  .sc-profile-row.active .sc-pr-browser { color: var(--text); }
  .sc-profile-row.active .sc-pr-name    { color: var(--text-secondary); }

  .sc-pr-stats { display: flex; align-items: center; gap: 4px; margin-top: 3px; }
  .sc-pr-count { font-size: 8px; color: var(--text-dim); }
  .sc-pr-add   { font-size: 9px; font-weight: 700; color: var(--green); }
  .sc-pr-rm    { font-size: 9px; font-weight: 700; color: var(--red); }
  .sc-pr-sync  { font-size: 8px; color: var(--text-dim); font-style: italic; }

  /* ── Pane ── */
  .sc-pane {
    flex: 1; display: flex; flex-direction: column; min-width: 0;
  }
  .sc-pane-l { border-right: 1px solid var(--border); }

  .sc-pane-head {
    display: flex; align-items: center; justify-content: space-between;
    padding: 6px 12px;
    font-size: 10px; font-weight: 700; letter-spacing: .6px;
    border-bottom: 1px solid var(--border);
    flex-shrink: 0;
  }
  .sc-pane-head-l { background: hsla(3,80%,67%,.05); color: var(--red); }
  .sc-pane-head-r { background: hsla(145,38%,55%,.05); color: var(--green); }

  .sc-pane-label { display: flex; align-items: center; gap: 7px; }

  .sc-pane-badge {
    display: inline-flex; align-items: center; justify-content: center;
    width: 16px; height: 16px; border-radius: 3px;
    font-size: 9px; font-weight: 800; letter-spacing: 0;
  }
  .sc-pane-badge-l { background: var(--red-dim);   color: var(--red); }
  .sc-pane-badge-r { background: var(--green-dim); color: var(--green); }

  .sc-pane-count { font-size: 9px; color: var(--text-dim); font-weight: 400; }

  .sc-pane-toolbar {
    display: flex; align-items: center; gap: 6px;
    padding: 5px 10px;
    border-bottom: 1px solid var(--border);
    background: var(--bg-deep);
    flex-shrink: 0;
  }
  .sc-search {
    flex: 1; min-width: 0;
    background: var(--bg-card); border: 1px solid var(--border);
    border-radius: var(--radius-sm); color: var(--text);
    font-size: 10px; padding: 3px 7px; outline: none;
    transition: border-color .1s;
  }
  .sc-search:focus { border-color: var(--border-active); }

  .sc-fgroup { display: flex; gap: 2px; }
  .sc-f {
    padding: 2px 7px; border-radius: var(--radius-sm);
    border: 1px solid var(--border);
    background: var(--bg-card); color: var(--text-muted);
    font-size: 9px; font-weight: 600; cursor: pointer;
    transition: all .1s;
  }
  .sc-f:hover { border-color: var(--border-hover); color: var(--text-secondary); }
  .sc-f.active { background: var(--bg-hover); border-color: var(--border-active); color: var(--text); }
  .sc-f-rm.active { background: var(--red-dim);   border-color: var(--red);   color: var(--red); }
  .sc-f-add.active { background: var(--green-dim); border-color: var(--green); color: var(--green); }

  .sc-fold-btns { display: flex; gap: 2px; margin-left: auto; }
  .sc-fold-btn {
    background: none; border: none; color: var(--text-dim);
    cursor: pointer; font-size: 12px; padding: 1px 4px;
    border-radius: var(--radius-sm); transition: color .1s; line-height: 1;
  }
  .sc-fold-btn:hover { color: var(--text-secondary); }

  /* ── Folder headers ── */
  /* ── Breadcrumb ── */
  .sc-breadcrumb {
    display: flex; align-items: center; gap: 3px; flex-wrap: nowrap; overflow: hidden;
    padding: 4px 10px;
    background: var(--bg-deep);
    border-bottom: 1px solid var(--border-active);
    font-size: 9px; font-weight: 600; letter-spacing: .3px;
    color: var(--text-dim);
    flex-shrink: 0;
    min-height: 22px;
  }
  .sc-crumb-sep { color: var(--blue); opacity: .6; flex-shrink: 0; }
  .sc-crumb-part {
    white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
    color: var(--text-secondary);
    max-width: 120px;
  }
  .sc-crumb-part:last-child { color: var(--blue); }

  .sc-folder-hdr {
    display: flex; align-items: center; gap: 5px;
    width: 100%; min-height: 22px;
    padding-right: 10px; padding-top: 2px; padding-bottom: 2px;
    background: var(--bg-card); border: none; border-bottom: 1px solid var(--border);
    color: var(--text-muted); font-size: 10px; font-weight: 600;
    cursor: pointer; text-align: left; transition: background .1s;
    position: sticky; top: 0; z-index: 1;
  }
  .sc-folder-hdr:hover { background: var(--bg-hover); color: var(--text-secondary); }
  .sc-folder-hdr.has-change { border-left: 2px solid var(--border-active); }

  .sc-fhdr-chevron { font-size: 7px; color: var(--text-dim); flex-shrink: 0; width: 8px; }
  .sc-fhdr-icon    { font-size: 9px; flex-shrink: 0; }
  .sc-fhdr-name    { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .sc-fhdr-count   { font-size: 9px; color: var(--text-dim); flex-shrink: 0; font-weight: 400; }
  .sc-fhdr-badge   {
    font-size: 9px; font-weight: 700; padding: 0 4px; border-radius: 2px; flex-shrink: 0;
  }
  .sc-fhdr-badge-rm  { color: var(--red);   background: var(--red-dim); }
  .sc-fhdr-badge-add { color: var(--green); background: var(--green-dim); }

  /* ── List ── */
  .sc-list {
    flex: 1; overflow-y: auto; min-height: 0;
    position: relative; /* needed so offsetTop of children is relative to this container */
  }
  .sc-row {
    display: flex; align-items: center; gap: 6px;
    padding: 3px 10px 3px 22px; /* padding-left overridden inline per depth */
    min-height: 24px;
    border-bottom: 1px solid var(--border);
    font-size: 10px; transition: background .1s;
    cursor: pointer;
  }
  .sc-row:hover { background: var(--bg-hover); }
  .sc-row-hl {
    background: hsla(45, 90%, 55%, 0.18) !important;
    border-left: 2px solid hsl(45, 90%, 60%);
    outline: none;
  }
  .sc-row-hl .sc-row-title { color: hsl(45, 90%, 75%); font-weight: 600; }
  .sc-row-rm {
    background: var(--red-dim) !important;
    border-bottom-color: hsla(3,80%,67%,.12);
  }
  .sc-row-rm .sc-row-title { text-decoration: line-through; opacity: .7; }
  .sc-row-add { background: var(--green-dim) !important; border-bottom-color: hsla(145,38%,55%,.12); }

  .sc-row-icon {
    flex-shrink: 0; width: 12px; text-align: center;
    font-size: 9px; font-weight: 700;
  }
  .sc-row-rm .sc-row-icon { color: var(--red); }
  .sc-row-add .sc-row-icon { color: var(--green); }
  .sc-row-ok .sc-row-icon { color: var(--text-dim); }

  .sc-row-title {
    flex: 1; min-width: 0;
    overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
    color: var(--text-secondary);
  }
  .sc-row-cat {
    flex-shrink: 0; font-size: 8px; font-weight: 600;
    color: var(--blue); background: var(--blue-dim);
    padding: 1px 4px; border-radius: 2px; white-space: nowrap; max-width: 80px;
    overflow: hidden; text-overflow: ellipsis;
  }
  .sc-row-domain {
    flex-shrink: 0; color: var(--text-dim); font-size: 9px; white-space: nowrap;
    max-width: 100px; overflow: hidden; text-overflow: ellipsis;
  }

  .sc-spinner, .sc-empty {
    padding: 20px; text-align: center;
    color: var(--text-dim); font-size: 10px;
  }

  /* ── Divider ── */
  .sc-divider {
    display: flex; flex-direction: column; align-items: center;
    justify-content: center; width: 28px; flex-shrink: 0;
    background: var(--bg-deep); border-left: 1px solid var(--border);
    border-right: 1px solid var(--border); gap: 6px;
    color: var(--text-dim);
  }
  .sc-div-line { flex: 1; width: 1px; background: var(--border); }
  .sc-div-icon { font-size: 12px; writing-mode: vertical-lr; transform: rotate(0deg); color: var(--blue); }

  /* ── Footer ── */
  .sc-footer {
    display: flex; align-items: center; justify-content: flex-end;
    padding: 8px 14px; gap: 10px;
    border-top: 1px solid var(--border);
    background: var(--bg-deep); flex-shrink: 0;
  }
  .sc-footer-hint {
    flex: 1; font-size: 10px; color: var(--text-dim); font-style: italic;
  }
  .sc-footer-btns { display: flex; gap: 6px; flex-shrink: 0; }

  .sc-btn {
    padding: 6px 14px; border-radius: var(--radius);
    border: 1px solid var(--border);
    font-size: 11px; font-weight: 500; cursor: pointer;
    transition: all .1s; white-space: nowrap;
  }
  .sc-btn-cancel {
    background: var(--bg-card); color: var(--text-muted);
  }
  .sc-btn-cancel:hover { background: var(--bg-hover); color: var(--text-secondary); }
  .sc-btn-confirm {
    background: var(--blue); color: #fff; border-color: var(--blue);
    font-weight: 600;
  }
  .sc-btn-confirm:hover:not(:disabled) { filter: brightness(1.1); }
  .sc-btn-confirm:disabled { opacity: .4; cursor: not-allowed; }
</style>
