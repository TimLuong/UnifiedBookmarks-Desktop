<script lang="ts">
  import { openInApp } from './store';

  // @ts-ignore
  const go = window['go']?.['main']?.['App'];
  // @ts-ignore
  const runtime = window['runtime'];

  let pageHtml = '';
  let loading = false;
  let fetchError = '';

  // Reactive: fetch page content via Go RPC whenever URL changes
  $: if ($openInApp) {
    loading = true;
    pageHtml = '';
    fetchError = '';
    if (go?.FetchPageHTML) {
      go.FetchPageHTML($openInApp.url)
        .then((r: { html: string; error: string }) => {
          loading = false;
          if (r.error) {
            fetchError = r.error;
          } else {
            pageHtml = r.html;
          }
        })
        .catch((e: unknown) => {
          loading = false;
          fetchError = String(e);
        });
    } else {
      loading = false;
      fetchError = 'Wails bridge not available';
    }
  }

  function openExternal() {
    if (!$openInApp) return;
    if (runtime?.BrowserOpenURL) {
      runtime.BrowserOpenURL($openInApp.url);
    } else {
      window.open($openInApp.url, '_blank', 'noopener,noreferrer');
    }
  }

  function close() {
    openInApp.set(null);
    pageHtml = '';
    loading = false;
    fetchError = '';
  }

  function getDomain(url: string): string {
    try { return new URL(url).hostname.replace('www.', ''); }
    catch { return url; }
  }
</script>

{#if $openInApp}
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <!-- svelte-ignore a11y-no-static-element-interactions -->
  <div class="browser-overlay" on:click|self={close}>
    <div class="browser-panel">
      <div class="browser-bar">
        <div class="browser-favicon">
          <img src={`https://www.google.com/s2/favicons?domain=${getDomain($openInApp.url)}&sz=16`} width="16" height="16" alt="" on:error={(e) => (e.currentTarget.style.display='none')} />
        </div>
        <div class="browser-url-area">
          <span class="browser-title">{$openInApp.title}</span>
          <span class="browser-domain">{getDomain($openInApp.url)}</span>
        </div>
        <button class="browser-btn-ext" title="Open in browser" on:click={openExternal}>
          <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/>
            <polyline points="15 3 21 3 21 9"/>
            <line x1="10" y1="14" x2="21" y2="3"/>
          </svg>
          Open in browser
        </button>
        <button class="browser-btn-close" title="Close" on:click={close}>
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round">
            <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
          </svg>
        </button>
      </div>
      <div class="browser-content">
        {#if loading}
          <div class="frame-state">
            <div class="frame-spinner"></div>
            <span class="frame-state-msg">Loading…</span>
          </div>
        {/if}

        {#if fetchError}
          <div class="frame-state">
            <div class="block-icon">
              <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round">
                <circle cx="12" cy="12" r="10"/>
                <line x1="4.93" y1="4.93" x2="19.07" y2="19.07"/>
              </svg>
            </div>
            <p class="block-title">Could not load {getDomain($openInApp?.url ?? '')}</p>
            <p class="block-hint">The site may require login or have strict access controls.<br/>Open it in your system browser instead.</p>
            <button class="block-open-btn" on:click={openExternal}>
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
                <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/>
                <polyline points="15 3 21 3 21 9"/>
                <line x1="10" y1="14" x2="21" y2="3"/>
              </svg>
              Open in browser
            </button>
          </div>
        {/if}

        {#if pageHtml}
          <iframe
            srcdoc={pageHtml}
            title={$openInApp?.title ?? ''}
            class="browser-iframe"
            sandbox="allow-scripts allow-same-origin allow-forms allow-popups"
          ></iframe>
        {/if}
      </div>
    </div>
  </div>
{/if}

<style>
  .browser-overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.55);
    z-index: 1000;
    display: flex;
    align-items: stretch;
    justify-content: flex-end;
  }

  .browser-panel {
    width: 72%;
    max-width: 1100px;
    height: 100%;
    background: var(--bg);
    display: flex;
    flex-direction: column;
    border-left: 1px solid var(--border);
    box-shadow: -8px 0 32px rgba(0,0,0,0.4);
    animation: slide-in .15s ease-out;
  }

  @keyframes slide-in {
    from { transform: translateX(40px); opacity: 0; }
    to   { transform: translateX(0);    opacity: 1; }
  }

  .browser-bar {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px 12px;
    border-bottom: 1px solid var(--border);
    background: var(--bg-secondary);
    min-height: 44px;
    flex-shrink: 0;
  }

  .browser-favicon { flex-shrink: 0; width: 16px; height: 16px; }

  .browser-url-area {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 1px;
  }
  .browser-title {
    font-size: 13px;
    font-weight: 500;
    color: var(--text);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .browser-domain {
    font-size: 11px;
    color: var(--text-muted);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .browser-btn-ext {
    display: flex;
    align-items: center;
    gap: 5px;
    padding: 5px 10px;
    border-radius: var(--radius-sm);
    background: var(--bg-hover);
    border: 1px solid var(--border);
    color: var(--text-secondary);
    font-size: 12px;
    cursor: pointer;
    white-space: nowrap;
    flex-shrink: 0;
    transition: background .1s, color .1s;
  }
  .browser-btn-ext:hover { background: var(--accent); color: #fff; border-color: var(--accent); }

  .browser-btn-close {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 28px;
    height: 28px;
    border-radius: var(--radius-sm);
    background: none;
    border: none;
    color: var(--text-dim);
    cursor: pointer;
    flex-shrink: 0;
    transition: background .1s, color .1s;
  }
  .browser-btn-close:hover { background: var(--red, #e55); color: #fff; }

  .browser-content {
    flex: 1;
    display: flex;
    position: relative;
    overflow: hidden;
  }

  .browser-iframe {
    position: absolute;
    inset: 0;
    width: 100%;
    height: 100%;
    border: none;
    display: block;
    background: #fff;
  }
  .browser-iframe.hidden { visibility: hidden; pointer-events: none; }

  /* Loading / blocked overlay */
  .frame-state {
    position: absolute;
    inset: 0;
    z-index: 2;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 14px;
    background: var(--bg);
    padding: 32px;
    text-align: center;
  }

  .frame-spinner {
    width: 32px;
    height: 32px;
    border: 3px solid var(--border);
    border-top-color: var(--accent);
    border-radius: 50%;
    animation: spin .7s linear infinite;
  }
  @keyframes spin { to { transform: rotate(360deg); } }

  .frame-state-msg { font-size: 13px; color: var(--text-muted); }

  .block-icon { color: var(--text-dim); opacity: .5; }

  .block-title {
    font-size: 15px;
    font-weight: 600;
    color: var(--text);
    margin: 0;
  }

  .block-hint {
    font-size: 13px;
    color: var(--text-muted);
    line-height: 1.5;
    margin: 0;
    max-width: 320px;
  }

  .block-open-btn {
    display: flex;
    align-items: center;
    gap: 7px;
    padding: 9px 20px;
    border-radius: var(--radius-sm);
    background: var(--accent);
    border: none;
    color: #fff;
    font-size: 13px;
    font-weight: 500;
    cursor: pointer;
    transition: opacity .1s;
    font-family: inherit;
  }
  .block-open-btn:hover { opacity: .85; }
</style>
