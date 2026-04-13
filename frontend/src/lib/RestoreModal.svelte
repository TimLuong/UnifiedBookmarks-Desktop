<script lang="ts">
  import { createEventDispatcher, onMount } from 'svelte';
  import { profiles } from './store';
  import type { Profile, Snapshot } from './store';

  const dispatch = createEventDispatcher();

  export let visible = false;

  let snapshots: Snapshot[] = [];
  let loadingSnapshots = false;
  let backingUp = false;
  let confirmAction: { type: 'restore' | 'delete'; snapshot: Snapshot; targetProfile: number } | null = null;
  let statusMessage = '';

  // ── Snapshot filter state ──────────────────────────────
  let snapSearch = '';
  let snapDateRange: 'all' | '1h' | 'today' | 'week' | 'month' = 'all';
  const snapDateOptions: { key: 'all' | '1h' | 'today' | 'week' | 'month'; label: string }[] = [
    { key: 'all',   label: 'All' },
    { key: '1h',    label: '1h' },
    { key: 'today', label: 'Today' },
    { key: 'week',  label: '7d' },
    { key: 'month', label: '30d' },
  ];

  function snapMatchesDate(ts: string, range: typeof snapDateRange): boolean {
    if (range === 'all') return true;
    try {
      const d = new Date(ts.replace(' ', 'T'));
      if (isNaN(d.getTime())) return true;
      const diff = Date.now() - d.getTime();
      if (range === '1h')    return diff < 3_600_000;
      if (range === 'today') return diff < 86_400_000;
      if (range === 'week')  return diff < 604_800_000;
      if (range === 'month') return diff < 2_592_000_000;
    } catch {}
    return true;
  }

  $: filteredSnapshots = snapshots.filter(s => {
    const q = snapSearch.toLowerCase().trim();
    if (q && !`${s.browser} ${s.profile}`.toLowerCase().includes(q)) return false;
    if (!snapMatchesDate(s.timestamp, snapDateRange)) return false;
    return true;
  });

  // ── Group filtered snapshots by browser+profile ────────
  // Shows one row per profile (latest backup), expandable to see older ones
  let expandedGroups: Record<string, boolean> = {};

  function toggleGroup(key: string) {
    expandedGroups = { ...expandedGroups, [key]: !expandedGroups[key] };
  }

  $: snapGroups = (() => {
    const map: Map<string, { browser: string; profile: string; snaps: typeof filteredSnapshots }> = new Map();
    for (const s of filteredSnapshots) {
      const k = s.browser + '|||' + s.profile;
      if (!map.has(k)) map.set(k, { browser: s.browser, profile: s.profile, snaps: [] });
      map.get(k)!.snaps.push(s);
    }
    return [...map.values()];
  })();

  // @ts-ignore — Wails auto-generated
  async function loadSnapshots() {
    loadingSnapshots = true;
    statusMessage = '';
    try {
      // @ts-ignore
      snapshots = await window['go']['main']['App']['ListSnapshots']() || [];
    } catch (e: any) {
      statusMessage = '❌ ' + (e.message || e);
      snapshots = [];
    }
    loadingSnapshots = false;
  }

  async function doForceBackup() {
    backingUp = true;
    statusMessage = '';
    try {
      // @ts-ignore
      const snaps: Snapshot[] = await window['go']['main']['App']['ForceBackupAll']() || [];
      statusMessage = `✅ Backed up ${snaps.length} profile(s).`;
      await loadSnapshots();
    } catch (e: any) {
      statusMessage = '❌ ' + (e.message || e);
    }
    backingUp = false;
  }

  async function doRestore(snap: Snapshot, profileIdx: number) {
    statusMessage = '';
    try {
      // @ts-ignore
      await window['go']['main']['App']['RestoreSnapshot'](snap.id, profileIdx);
      const p = $profiles[profileIdx];
      statusMessage = `✅ Restored "${snap.browser} — ${snap.profile}" → ${p ? p.displayName : 'profile #' + profileIdx}`;
      confirmAction = null;
    } catch (e: any) {
      statusMessage = '❌ ' + (e.message || e);
    }
  }

  async function doDelete(snap: Snapshot) {
    statusMessage = '';
    try {
      // @ts-ignore
      await window['go']['main']['App']['DeleteSnapshot'](snap.id);
      statusMessage = '🗑 Snapshot deleted.';
      confirmAction = null;
      await loadSnapshots();
    } catch (e: any) {
      statusMessage = '❌ ' + (e.message || e);
    }
  }

  function formatSize(bytes: number): string {
    if (bytes < 1024) return bytes + ' B';
    if (bytes < 1048576) return (bytes / 1024).toFixed(1) + ' KB';
    return (bytes / 1048576).toFixed(1) + ' MB';
  }

  function formatDate(ts: string): string {
    if (!ts) return '';
    try {
      // Go sends "2006-01-02 15:04:05" — replace space with T for ISO 8601
      const d = new Date(ts.replace(' ', 'T'));
      if (isNaN(d.getTime())) return ts;
      return d.toLocaleString('en-US', { year: 'numeric', month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' });
    } catch {
      return ts;
    }
  }

  function close() {
    confirmAction = null;
    statusMessage = '';
    dispatch('close');
  }

  $: if (visible) loadSnapshots();
</script>

{#if visible}
  <div class="modal-overlay" on:click|self={close} on:keydown|self={(e) => e.key === 'Escape' && close()}>
    <div class="modal">
      <div class="modal-header">
        <h2>
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/><path d="M3 3v5h5"/><path d="M12 7v5l4 2"/></svg>
          backups & restore
        </h2>
        <div class="modal-header-actions">
          <button class="backup-now-btn" disabled={backingUp} on:click={doForceBackup}>
            {#if backingUp}
              <span class="backup-spinner"></span>
            {:else}
              <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7,10 12,15 17,10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
            {/if}
            Backup Now
          </button>
          <button class="modal-close" on:click={close}>
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
        </button>        </div>      </div>

      {#if statusMessage}
        <div class="modal-status" class:is-error={statusMessage.startsWith('❌')}>
          {statusMessage}
        </div>
      {/if}

      <div class="modal-body">
        {#if loadingSnapshots}
          <div class="snapshots-loading">loading snapshots...</div>
        {:else if snapshots.length === 0}
          <div class="snapshots-empty">
            <pre class="ascii-empty">┌──────┐
│  --  │
└──────┘</pre>
            <p>no snapshots yet. created automatically before each sync.</p>
          </div>
        {:else}
          <div class="snap-filter-bar">
            <div class="snap-search-wrap">
              <svg class="snap-search-icon" width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
              <input class="snap-search" type="text" placeholder="Search browser or profile…" bind:value={snapSearch} />
              {#if snapSearch}<button class="snap-search-clear" on:click={() => snapSearch = ''}>×</button>{/if}
            </div>
            <div class="snap-date-btns">
              {#each snapDateOptions as opt}
                <button class="snap-date-btn" class:active={snapDateRange === opt.key} on:click={() => snapDateRange = opt.key}>
                  {opt.label}
                </button>
              {/each}
            </div>
          </div>
          <div class="snapshot-list">
            {#each snapGroups as group}
              {@const key = group.browser + '|||' + group.profile}
              <div class="snap-group" class:is-expanded={expandedGroups[key]}>
                <!-- Group header row: shows the latest snapshot for this profile -->
                <div class="snap-group-header">
                  <div class="snap-group-info">
                    <div class="snap-group-title">{group.browser} — {group.profile}</div>
                    <div class="snap-group-meta">
                      <span>{formatDate(group.snaps[0].timestamp)}</span>
                      <span>·</span>
                      <span>{group.snaps[0].count} bookmarks</span>
                      <span>·</span>
                      <span>{formatSize(group.snaps[0].sizeBytes)}</span>
                      {#if group.snaps.length > 1}
                        <span class="snap-more-badge">+{group.snaps.length - 1}</span>
                      {/if}
                    </div>
                  </div>
                  <div class="snap-group-actions">
                    {#if group.snaps.length > 1}
                      <button class="snap-expand-btn" title="Show history" on:click={() => toggleGroup(key)}>
                        <svg class="expand-chevron" class:rotated={expandedGroups[key]} width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><polyline points="6,9 12,15 18,9"/></svg>
                      </button>
                    {/if}
                    <button class="snap-btn snap-restore" on:click={() => { confirmAction = { type: 'restore', snapshot: group.snaps[0], targetProfile: 0 }; }}>
                      <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><polyline points="1,4 1,10 7,10"/><path d="M3.51 15a9 9 0 1 0 2.13-9.36L1 10"/></svg>
                      restore
                    </button>
                    <button class="snap-btn snap-delete" on:click={() => { confirmAction = { type: 'delete', snapshot: group.snaps[0], targetProfile: 0 }; }}>
                      <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><polyline points="3,6 5,6 21,6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>
                    </button>
                  </div>
                </div>
                <!-- Older backups for this profile (collapsed by default) -->
                {#if expandedGroups[key] && group.snaps.length > 1}
                  <div class="snap-history">
                    {#each group.snaps.slice(1) as snap}
                      <div class="snap-history-row">
                        <div class="snap-hist-info">
                          <span class="snap-hist-time">{formatDate(snap.timestamp)}</span>
                          <span class="snap-hist-sep">·</span>
                          <span class="snap-hist-meta">{snap.count} bm · {formatSize(snap.sizeBytes)}</span>
                        </div>
                        <div class="snap-hist-actions">
                          <button class="snap-btn snap-restore" on:click={() => { confirmAction = { type: 'restore', snapshot: snap, targetProfile: 0 }; }}>
                            <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><polyline points="1,4 1,10 7,10"/><path d="M3.51 15a9 9 0 1 0 2.13-9.36L1 10"/></svg>
                            restore
                          </button>
                          <button class="snap-btn snap-delete" on:click={() => { confirmAction = { type: 'delete', snapshot: snap, targetProfile: 0 }; }}>
                            <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><polyline points="3,6 5,6 21,6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>
                          </button>
                        </div>
                      </div>
                    {/each}
                  </div>
                {/if}
              </div>
            {/each}
            {#if snapGroups.length === 0}
              <div class="snapshots-empty"><p>no snapshots match the filter.</p></div>
            {/if}
          </div>
        {/if}
      </div>

      {#if confirmAction}
        <div class="confirm-overlay" on:click|self={() => confirmAction = null} on:keydown|self={(e) => e.key === 'Escape' && (confirmAction = null)}>
          <div class="confirm-box">
            {#if confirmAction.type === 'restore'}
              <div class="confirm-snap-info">
                <div class="confirm-snap-title">{confirmAction.snapshot.browser} — {confirmAction.snapshot.profile}</div>
                <div class="confirm-snap-meta">{formatDate(confirmAction.snapshot.timestamp)} · {confirmAction.snapshot.count} bookmarks · {formatSize(confirmAction.snapshot.sizeBytes)}</div>
              </div>
              <div class="confirm-target-row">
                <label for="confirm-profile">Restore to:</label>
                <select id="confirm-profile" bind:value={confirmAction.targetProfile}>
                  {#each $profiles as p, i}
                    <option value={i}>{p.browser} — {p.displayName}</option>
                  {/each}
                </select>
              </div>
              <p class="confirm-warn">This will <strong>overwrite</strong> the target profile's bookmarks.</p>
              <div class="confirm-btns">
                <button class="cbtn cbtn-cancel" on:click={() => confirmAction = null}>Cancel</button>
                <button class="cbtn cbtn-confirm" on:click={() => confirmAction && doRestore(confirmAction.snapshot, confirmAction.targetProfile)}>Yes, Restore</button>
              </div>
            {:else}
              <p>delete this snapshot permanently?</p>
              <div class="confirm-btns">
                <button class="cbtn cbtn-cancel" on:click={() => confirmAction = null}>Cancel</button>
                <button class="cbtn cbtn-danger" on:click={() => confirmAction && doDelete(confirmAction.snapshot)}>Delete</button>
              </div>
            {/if}
          </div>
        </div>
      {/if}
    </div>
  </div>
{/if}

<style>
  .modal-overlay {
    position: fixed;
    inset: 0;
    background: rgba(0,0,0,0.6);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
    backdrop-filter: blur(3px);
  }
  .modal {
    background: var(--bg);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    width: 620px;
    max-height: 80vh;
    display: flex;
    flex-direction: column;
    box-shadow: 0 20px 60px rgba(0,0,0,0.5);
    position: relative;
  }
  .modal-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 16px 20px;
    border-bottom: 1px solid var(--border);
  }
  .modal-header h2 {    font-size: 13px; color: var(--text-secondary); margin: 0; font-weight: 600;
    display: flex; align-items: center; gap: 6px;
    text-transform: lowercase; letter-spacing: 0.3px;
  }
  .modal-header h2 svg { color: var(--text-dim); }
  .modal-close {
    background: none; border: none; color: var(--text-dim); cursor: pointer;
    padding: 4px; border-radius: var(--radius-sm); transition: all .12s;
    display: flex; align-items: center;
  }
  .modal-close:hover { background: var(--bg-hover); color: var(--text-muted); }

  .modal-header-actions { display: flex; align-items: center; gap: 8px; }

  .backup-now-btn {
    display: flex;
    align-items: center;
    gap: 5px;
    padding: 5px 10px;
    border-radius: var(--radius);
    border: 1px solid var(--border);
    background: transparent;
    color: var(--text-secondary);
    font-size: 12px;
    font-weight: 500;
    cursor: pointer;
    transition: all .12s;
  }
  .backup-now-btn:hover { border-color: var(--accent); color: var(--accent); background: var(--accent-dim); }
  .backup-now-btn:disabled { opacity: 0.4; pointer-events: none; }
  .backup-spinner {
    width: 10px; height: 10px; border-radius: 50%;
    border: 1.5px solid var(--text-dim); border-top-color: var(--accent);
    animation: spin .6s linear infinite; flex-shrink: 0;
  }
  @keyframes spin { to { transform: rotate(360deg); } }

  .modal-status {
    padding: 6px 20px;
    font-size: 11px;
    color: var(--green);
    background: var(--green-dim);
    border-bottom: 1px solid var(--border);
  }
  .modal-status.is-error { color: var(--red); background: var(--red-dim); }

  .modal-body {
    padding: 16px 20px;
    overflow-y: auto;
    flex: 1;
  }

  .restore-target {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 14px;
    padding: 8px 10px;
    background: var(--bg-card);
    border-radius: var(--radius-sm);
    border: 1px solid var(--border);
  }
  .restore-target label { font-size: 11px; color: var(--text-muted); white-space: nowrap; }
  .restore-target select {
    flex: 1;
    background: var(--bg-deep);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    color: var(--text);
    padding: 4px 6px;
    font-size: 11px;
  }

  .snapshots-loading, .snapshots-empty {
    text-align: center;
    padding: 24px;
    color: var(--text-dim);
    font-size: 11px;
  }
  .ascii-empty {
    color: var(--text-dim);
    font-size: 10px;
    margin: 0 0 6px;
    line-height: 1.3;
  }

  .snapshot-list { display: flex; flex-direction: column; gap: 6px; }

  /* ── Grouped snapshot card ─────────────────────────────── */
  .snap-group {
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    background: var(--bg-card);
    overflow: hidden;
    transition: border-color .12s;
  }
  .snap-group:hover { border-color: var(--border-hover); }
  .snap-group.is-expanded { border-color: var(--border-hover); }

  .snap-group-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 10px;
    padding: 8px 10px;
    transition: background .1s;
  }
  .snap-group.is-expanded .snap-group-header {
    border-bottom: 1px solid var(--border);
    background: var(--bg-hover);
  }

  .snap-group-info { flex: 1; min-width: 0; }
  .snap-group-title { font-size: 12px; font-weight: 600; color: var(--text); }
  .snap-group-meta {
    display: flex;
    gap: 5px;
    font-size: 11px;
    color: var(--text-muted);
    margin-top: 2px;
    align-items: center;
    flex-wrap: wrap;
  }
  .snap-more-badge {
    font-size: 10px;
    font-weight: 600;
    padding: 1px 6px;
    border-radius: 10px;
    background: var(--accent-dim);
    color: var(--accent);
    flex-shrink: 0;
  }

  .snap-group-actions { display: flex; gap: 4px; flex-shrink: 0; align-items: center; }

  .snap-expand-btn {
    width: 24px;
    height: 24px;
    display: flex;
    align-items: center;
    justify-content: center;
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    background: var(--bg-deep);
    color: var(--text-muted);
    cursor: pointer;
    transition: all .12s;
    flex-shrink: 0;
  }
  .snap-expand-btn:hover { border-color: var(--border-hover); color: var(--text-secondary); }
  .expand-chevron { transition: transform .18s ease; }
  .expand-chevron.rotated { transform: rotate(180deg); }

  /* ── Older-backup history rows (expanded) ────────────── */
  .snap-history { display: flex; flex-direction: column; }
  .snap-history-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
    padding: 6px 10px 6px 20px; /* indent to show hierarchy */
    border-top: 1px solid var(--border);
    background: var(--bg);
    transition: background .1s;
  }
  .snap-history-row:hover { background: var(--bg-hover); }
  .snap-hist-info {
    display: flex;
    align-items: center;
    gap: 5px;
    flex: 1;
    min-width: 0;
  }
  .snap-hist-time { font-size: 11px; color: var(--text-secondary); white-space: nowrap; }
  .snap-hist-sep  { font-size: 11px; color: var(--text-dim); flex-shrink: 0; }
  .snap-hist-meta { font-size: 11px; color: var(--text-muted); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .snap-hist-actions { display: flex; gap: 4px; flex-shrink: 0; }
  .snap-btn {
    padding: 4px 8px;
    border-radius: var(--radius-sm);
    border: 1px solid var(--border);
    background: var(--bg-deep);
    color: var(--text-muted);
    cursor: pointer;
    font-size: 10px;
    font-weight: 500;
    transition: all .1s;
    display: flex;
    align-items: center;
    gap: 4px;
  }
  .snap-restore:hover { background: var(--blue-dim); color: var(--blue); border-color: var(--blue); }
  .snap-delete:hover { background: var(--red-dim); color: var(--red); border-color: var(--red); }

  .confirm-overlay {
    position: absolute;
    inset: 0;
    background: rgba(0,0,0,0.5);
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: var(--radius);
    z-index: 10;
  }
  .confirm-box {
    background: var(--bg);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    padding: 18px 20px;
    max-width: 400px;
    width: 100%;
    text-align: left;
    box-shadow: 0 10px 30px rgba(0,0,0,0.4);
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  /* Snapshot summary at top of confirm dialog */
  .confirm-snap-info {
    padding: 8px 10px;
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
  }
  .confirm-snap-title { font-size: 12px; font-weight: 600; color: var(--text); }
  .confirm-snap-meta { font-size: 11px; color: var(--text-muted); margin-top: 2px; }

  /* Target profile picker in confirm */
  .confirm-target-row {
    display: flex;
    align-items: center;
    gap: 8px;
  }
  .confirm-target-row label { font-size: 11px; color: var(--text-muted); white-space: nowrap; }
  .confirm-target-row select {
    flex: 1;
    background: var(--bg-deep);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    color: var(--text);
    padding: 5px 8px;
    font-size: 12px;
    font-family: inherit;
  }

  .confirm-warn { font-size: 11px; color: var(--text-muted); margin: 0; }
  .confirm-warn strong { color: var(--orange); }

  .confirm-box p { font-size: 12px; color: var(--text-secondary); margin: 0; line-height: 1.5; }
  .confirm-btns { display: flex; gap: 6px; justify-content: flex-end; }
  .cbtn {
    padding: 5px 12px;
    border-radius: var(--radius-sm);
    border: 1px solid var(--border);
    font-size: 11px;
    font-weight: 500;
    cursor: pointer;
    transition: all .1s;
  }
  .cbtn-cancel { background: var(--bg-card); color: var(--text-muted); }
  .cbtn-cancel:hover { background: var(--bg-hover); color: var(--text-secondary); }
  .cbtn-confirm { background: var(--blue-dim); color: var(--blue); border-color: var(--blue); }
  .cbtn-confirm:hover { background: var(--blue); color: #fff; }
  .cbtn-danger { background: var(--red-dim); color: var(--red); border-color: var(--red); }
  .cbtn-danger:hover { background: var(--red); color: #fff; }

  /* ── Snapshot filter bar ─────────────────────────────── */
  .snap-filter-bar {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 10px;
  }
  .snap-search-wrap {
    flex: 1;
    position: relative;
    display: flex;
    align-items: center;
  }
  .snap-search-icon {
    position: absolute;
    left: 7px;
    color: var(--text-dim);
    pointer-events: none;
  }
  .snap-search {
    width: 100%;
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    color: var(--text);
    font-size: 11px;
    padding: 4px 24px 4px 22px;
    font-family: inherit;
    outline: none;
    transition: border-color .12s;
  }
  .snap-search:focus { border-color: var(--accent); }
  .snap-search-clear {
    position: absolute;
    right: 5px;
    background: none;
    border: none;
    color: var(--text-dim);
    cursor: pointer;
    font-size: 12px;
    padding: 0 2px;
    line-height: 1;
  }
  .snap-date-btns { display: flex; gap: 3px; flex-shrink: 0; }
  .snap-date-btn {
    padding: 4px 8px;
    font-size: 10px;
    font-weight: 500;
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    background: var(--bg-card);
    color: var(--text-muted);
    cursor: pointer;
    transition: all .1s;
    white-space: nowrap;
  }
  .snap-date-btn:hover { border-color: var(--border-hover); color: var(--text-secondary); }
  .snap-date-btn.active { background: var(--accent-dim); color: var(--accent); border-color: var(--accent); }
</style>
