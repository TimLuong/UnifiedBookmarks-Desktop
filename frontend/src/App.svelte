<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import Sidebar from './lib/Sidebar.svelte';
  import Toolbar from './lib/Toolbar.svelte';
  import BookmarkList from './lib/BookmarkList.svelte';
  import StatsBar from './lib/StatsBar.svelte';
  import RestoreModal from './lib/RestoreModal.svelte';
  import ConsolePanel from './lib/ConsolePanel.svelte';
  import SettingsPanel from './lib/SettingsPanel.svelte';
  import PromptsPanel from './lib/PromptsPanel.svelte';
  import SyncPreviewPanel from './lib/SyncPreviewPanel.svelte';
  import BookmarkBrowser from './lib/BookmarkBrowser.svelte';
  import { bookmarks, profiles, loading, statusMsg, showConsole, addLog, settings, folderTree, pipelineStep } from './lib/store';
  import type { Bookmark, Profile, SyncPreviewResult, ProfileTree } from './lib/store';

  let showRestore = false;
  let showSyncPreview = false;
  let syncPreviewData: SyncPreviewResult | null = null;
  let progress = '';
  let unsubEvents: (() => void)[] = [];

  interface AnalysisCacheMeta {
    exists: boolean;
    timestamp: string;
    count: number;
    model: string;
    tokens: number;
  }
  let lastAnalysisMeta: AnalysisCacheMeta | null = null;

  // @ts-ignore — Wails runtime injected globally
  const runtime = window['runtime'];

  // Apply theme + font size to <html> element reactively
  $: {
    const root = document.documentElement;
    root.setAttribute('data-theme', $settings.theme);
    root.setAttribute('data-fontsize', $settings.fontSize);
    // Use WebView2-compatible zoom via document.body.style.zoom
    const zoomMap: Record<string, string> = { small: '1', medium: '1.1', large: '1.2' };
    document.body.style.zoom = zoomMap[$settings.fontSize] || '1';
  }

  onMount(async () => {
    addLog('info', 'UnifiedBookmarks Desktop started');
    // Load last analysis meta (non-blocking)
    try {
      lastAnalysisMeta = await callGo('GetLastAnalysisMeta');
      if (lastAnalysisMeta?.exists) {
        addLog('info', `💾 Cached analysis: ${lastAnalysisMeta.count} bm · ${lastAnalysisMeta.model} · ${lastAnalysisMeta.timestamp}`);
      }
    } catch {}
    if (runtime && runtime.EventsOn) {
      const off = runtime.EventsOn('analyze:progress', (info: any) => {
        if (typeof info === 'string') {
          progress = info;
          addLog('info', info);
        } else if (info && typeof info === 'object') {
          if (info.message && !info.batch) {
            // Message-only event (e.g. "Processing 469 bookmarks in 1 batch(es)...")
            progress = info.message;
            addLog('info', info.message);
          } else if (info.batch) {
            // Batch progress event
            const pct = Math.round((info.batch / info.total) * 100);
            progress = `Batch ${info.batch}/${info.total} (${pct}%) — ${info.batchItems} items`;
            addLog('info', `[${info.elapsed || '?'}s] Batch ${info.batch}/${info.total} — ${info.batchItems} items, ${info.tokens} tokens total`);
          } else {
            progress = info.message || JSON.stringify(info);
            addLog('debug', JSON.stringify(info));
          }
        }
      });
      unsubEvents.push(off);
    }
  });

  onDestroy(() => {
    for (const off of unsubEvents) off();
  });

  async function callGo(method: string, ...args: any[]): Promise<any> {
    // @ts-ignore
    return window['go']['main']['App'][method](...args);
  }

  async function handleScan() {
    loading.set('scan');
    statusMsg.set('');
    addLog('info', '🔍 Scanning browser profiles...');
    try {
      const result = await callGo('ScanProfiles');
      const p: Profile[] = result.profiles || [];
      profiles.set(p);
      for (const prof of p) {
        addLog('debug', `  ${prof.browserLabel} — ${prof.displayName} (${prof.hasBookmarks ? 'has bookmarks' : 'empty'})`);
      }
      addLog('success', `Found ${p.length} browser profile(s) on ${result.os}`);
      statusMsg.set(`✅ Found ${p.length} browser profile(s)`);
      pipelineStep.update(s => s < 1 ? 1 : s);
    } catch (e: any) {
      addLog('error', `Scan failed: ${e.message || e}`);
      statusMsg.set('❌ Scan failed: ' + (e.message || e));
    }
    loading.set(null);
  }

  async function handleCollect() {
    loading.set('collect');
    statusMsg.set('');
    addLog('info', '📥 Collecting bookmarks from all profiles...');
    try {
      const stats = await callGo('CollectBookmarks');
      const bms: Bookmark[] = await callGo('GetBookmarks') || [];
      bookmarks.set(bms);
      // Populate the folder tree from collected bookmarks
      const tree: ProfileTree[] = await callGo('GetFolderTree') || [];
      folderTree.set(tree);
      addLog('success', `Collected ${stats.totalDeduped} bookmarks from ${stats.totalRaw} raw`);
      addLog('debug', `  URL duplicates removed: ${stats.duplicatesUrl}`);
      addLog('debug', `  Fuzzy duplicates removed: ${stats.duplicatesFuzzy}`);
      statusMsg.set(`✅ Collected ${stats.totalDeduped} bookmarks (${stats.duplicatesUrl} URL dupes, ${stats.duplicatesFuzzy} fuzzy dupes removed)`);
      pipelineStep.update(s => s < 2 ? 2 : s);
    } catch (e: any) {
      addLog('error', `Collect failed: ${e.message || e}`);
      statusMsg.set('❌ Collect failed: ' + (e.message || e));
    }
    loading.set(null);
  }

  async function handleAnalyze() {
    loading.set('analyze');
    statusMsg.set('');
    progress = 'Preparing AI analysis...';
    addLog('info', '🧠 Starting AI categorization...');
    try {
      const result = await callGo('Analyze');
      const bms: Bookmark[] = result.bookmarks || await callGo('GetBookmarks') || [];
      bookmarks.set(bms);
      addLog('success', `AI analysis complete — ${bms.length} bookmarks, ${result.totalBatches} batches, ${result.totalTokens} tokens`);
      addLog('debug', `  Model: ${result.model}`);
      statusMsg.set(`✅ Analyzed ${bms.length} bookmarks — ${result.totalBatches} batches, model: ${result.model}`);
      pipelineStep.update(s => s < 3 ? 3 : s);
      progress = '';
      // Refresh cache meta
      try { lastAnalysisMeta = await callGo('GetLastAnalysisMeta'); } catch {}
    } catch (e: any) {
      addLog('error', `Analyze failed: ${e.message || e}`);
      statusMsg.set('❌ Analyze failed: ' + (e.message || e));
      progress = '';
    }
    loading.set(null);
  }

  async function handleSync() {
    // Show sync preview first
    addLog('info', 'Generating sync preview...');
    try {
      const preview = await callGo('GetSyncPreview');
      syncPreviewData = preview;
      showSyncPreview = true;
      addLog('success', `Preview ready — ${preview.diffs?.length || 0} profile(s)`);
    } catch (e: any) {
      addLog('error', `Preview failed: ${e.message || e}`);
      statusMsg.set('❌ Preview failed: ' + (e.message || e));
    }
  }

  async function handleConfirmSync(event: CustomEvent) {
    const selectedProfiles: string[] = event.detail?.selectedProfiles || [];
    showSyncPreview = false;
    syncPreviewData = null;
    loading.set('sync');
    statusMsg.set('');
    addLog('info', `Syncing bookmarks to ${selectedProfiles.length} profile(s)...`);
    try {
      const resp = await callGo('SyncToSelectedProfiles', selectedProfiles);
      const results = resp.results || [];
      const ok = results.filter((r: any) => r.status === 'ok' || r.status === 'success').length;
      const fail = results.filter((r: any) => r.status === 'error').length;
      const backups = (resp.backups || []).length;
      for (const r of results) {
        if (r.status === 'ok' || r.status === 'success') {
          addLog('success', `  ✓ ${r.browser} — ${r.profile}: synced (${r.written} bm)`);
        } else {
          addLog('error', `  ✗ ${r.browser} — ${r.profile}: ${r.reason ?? 'unknown error'}`);
        }
      }
      addLog('info', `${backups} backup(s) created before sync`);
      if (ok > 0) {
        addLog('warn', `⚠️ If Chrome reverts in a few seconds, disable Chrome Sync (chrome://settings/syncSetup) then sync again`);
      }
      statusMsg.set(`✅ Synced to ${ok} profile(s)${fail ? `, ${fail} failed` : ''} (${backups} backup(s) created)`);
      pipelineStep.set(4);
    } catch (e: any) {
      addLog('error', `Sync failed: ${e.message || e}`);
      statusMsg.set('❌ Sync failed: ' + (e.message || e));
    }
    loading.set(null);
  }

  async function handleExport() {
    addLog('info', '📤 Exporting bookmarks...');
    try {
      const path = await callGo('ExportBookmarks');
      addLog('success', `Exported to ${path}`);
      statusMsg.set(`✅ Exported to ${path}`);
    } catch (e: any) {
      addLog('error', `Export failed: ${e.message || e}`);
      statusMsg.set('❌ Export failed: ' + (e.message || e));
    }
  }

  async function handleLoadLastAnalysis() {
    loading.set('analyze');
    statusMsg.set('');
    progress = 'Loading cached analysis...';
    addLog('info', '💾 Loading last analysis from cache...');
    try {
      const result = await callGo('LoadLastAnalysis');
      const bms: Bookmark[] = result.bookmarks || [];
      const profs: Profile[] = result.profiles || [];
      bookmarks.set(bms);
      if (profs.length > 0) {
        profiles.set(profs);
        // Restore folder tree from bookmarks
        try {
          const tree: ProfileTree[] = await callGo('GetFolderTree') || [];
          folderTree.set(tree);
        } catch {}
      }
      addLog('success', `Loaded ${bms.length} bookmarks · ${profs.length} profile(s) from cache (model: ${result.model})`);
      statusMsg.set(`✅ Loaded cached analysis — ${bms.length} bookmarks · ${result.model}`);
      pipelineStep.update(s => s < 3 ? 3 : s);
      progress = '';
    } catch (e: any) {
      addLog('error', `Load cache failed: ${e.message || e}`);
      statusMsg.set('❌ ' + (e.message || e));
      progress = '';
    }
    loading.set(null);
  }
</script>

<div class="app-layout">
  <Sidebar on:scan={handleScan} on:collect={handleCollect} on:analyze={handleAnalyze} on:sync={handleSync} on:restore={() => showRestore = true}
    on:loadlast={handleLoadLastAnalysis}
    {lastAnalysisMeta} />
  <div class="main-panel">
    <Toolbar on:sync={handleSync} on:export={handleExport} />
    <BookmarkList />
    <ConsolePanel />
    <StatsBar {progress} />
  </div>
</div>

<RestoreModal bind:visible={showRestore} on:close={() => showRestore = false} />
<SettingsPanel />
<PromptsPanel />
<BookmarkBrowser />
<SyncPreviewPanel
  visible={showSyncPreview}
  preview={syncPreviewData}
  on:confirm={handleConfirmSync}
  on:close={() => { showSyncPreview = false; syncPreviewData = null; }}
/>

<style>
  :global(html) {
    height: 100%;
    margin: 0;
  }
  :global(body) {
    margin: 0;
    padding: 0;
    height: 100%;
    background: var(--bg);
    color: var(--text);
    overflow: hidden;
  }
  .app-layout {
    display: flex;
    height: 100%;
    width: 100%;
    overflow: hidden;
  }
  .main-panel {
    flex: 1;
    display: flex;
    flex-direction: column;
    min-width: 0;
  }
</style>
