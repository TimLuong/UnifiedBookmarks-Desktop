<script lang="ts">
  import { onMount } from 'svelte';
  import { settings, showSettings } from './store';
  import type { ThemeMode, FontSize, AIConfig, TestResult } from './store';

  const themes: { value: ThemeMode; label: string }[] = [
    { value: 'dark', label: 'dark' },
    { value: 'light', label: 'light' },
    { value: 'sand', label: 'sand' },
  ];

  const fontSizes: { value: FontSize; label: string; desc: string }[] = [
    { value: 'small', label: '12px', desc: 'compact' },
    { value: 'medium', label: '13px', desc: 'default' },
    { value: 'large', label: '14px', desc: 'comfortable' },
  ];

  function setTheme(t: ThemeMode) {
    settings.update(s => ({ ...s, theme: t }));
  }

  function setFontSize(f: FontSize) {
    settings.update(s => ({ ...s, fontSize: f }));
  }

  // @ts-ignore — Wails auto-generated
  function callGo(method: string, ...args: any[]): Promise<any> {
    return (window as any)['go']['main']['App'][method](...args);
  }

  // ── AI Config state ──
  let aiEndpoint = '';
  let aiApiKey = '';
  let aiModel = '';
  let aiTesting = false;
  let aiSaving = false;
  let aiStatus: { type: 'success' | 'error' | 'info' | ''; msg: string } = { type: '', msg: '' };
  let showApiKey = false;

  async function loadAIConfig() {
    try {
      const cfg: AIConfig = await callGo('GetAIConfig');
      aiEndpoint = cfg.endpoint || '';
      aiApiKey = cfg.apiKey || '';
      aiModel = cfg.model || '';
    } catch {}
  }

  async function testAI() {
    aiTesting = true;
    aiStatus = { type: 'info', msg: 'testing connection...' };
    try {
      const r: TestResult = await callGo('TestAIConnection', { endpoint: aiEndpoint, apiKey: aiApiKey, model: aiModel });
      aiStatus = r.ok
        ? { type: 'success', msg: `✓ ${r.message}` }
        : { type: 'error', msg: `✗ ${r.message}` };
    } catch (e: any) {
      aiStatus = { type: 'error', msg: `✗ ${e.message || e}` };
    }
    aiTesting = false;
  }

  async function saveAI() {
    aiSaving = true;
    aiStatus = { type: 'info', msg: 'saving...' };
    try {
      await callGo('SaveAIConfig', { endpoint: aiEndpoint, apiKey: aiApiKey, model: aiModel });
      aiStatus = { type: 'success', msg: '✓ saved' };
    } catch (e: any) {
      aiStatus = { type: 'error', msg: `✗ ${e.message || e}` };
    }
    aiSaving = false;
  }

  function maskKey(key: string): string {
    if (!key || key.length < 8) return '••••••••';
    return key.slice(0, 4) + '••••' + key.slice(-4);
  }

  // Load AI config when settings panel opens
  $: if ($showSettings) { loadAIConfig(); }
</script>

{#if $showSettings}
  <div class="settings-overlay" on:click|self={() => showSettings.set(false)}
       on:keydown|self={(e) => e.key === 'Escape' && showSettings.set(false)}>
    <div class="settings-panel">
      <div class="sp-header">
        <div class="sp-title">
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
            <circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"/>
          </svg>
          settings
        </div>
        <button class="sp-close" on:click={() => showSettings.set(false)}>
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
        </button>
      </div>

      <div class="sp-body">
        <div class="sp-section">
          <div class="sp-label">
            <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><circle cx="12" cy="12" r="5"/><line x1="12" y1="1" x2="12" y2="3"/><line x1="12" y1="21" x2="12" y2="23"/><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/><line x1="1" y1="12" x2="3" y2="12"/><line x1="21" y1="12" x2="23" y2="12"/><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/></svg>
            theme
          </div>
          <div class="sp-options">
            {#each themes as t}
              <button
                class="sp-opt"
                class:active={$settings.theme === t.value}
                on:click={() => setTheme(t.value)}
              >
                {#if t.value === 'dark'}
                  <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/></svg>
                {:else if t.value === 'light'}
                  <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><circle cx="12" cy="12" r="5"/><line x1="12" y1="1" x2="12" y2="3"/><line x1="12" y1="21" x2="12" y2="23"/><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/><line x1="1" y1="12" x2="3" y2="12"/><line x1="21" y1="12" x2="23" y2="12"/><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/></svg>
                {:else}
                  <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M17 8C8 10 5.9 16.17 3.82 19.7A1 1 0 0 0 5 21C21 21 21 8 17 8z"/><path d="M3.82 19.7a9 9 0 0 1 9-9.7"/></svg>
                {/if}
                {t.label}
              </button>
            {/each}
          </div>
        </div>

        <div class="sp-section">
          <div class="sp-label">
            <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><polyline points="4,7 4,4 20,4 20,7"/><line x1="9" y1="20" x2="15" y2="20"/><line x1="12" y1="4" x2="12" y2="20"/></svg>
            font size
          </div>
          <div class="sp-options">
            {#each fontSizes as f}
              <button
                class="sp-opt"
                class:active={$settings.fontSize === f.value}
                on:click={() => setFontSize(f.value)}
              >
                <span class="opt-sz">{f.label}</span>
                <span class="opt-desc">{f.desc}</span>
              </button>
            {/each}
          </div>
        </div>

        <div class="sp-divider"></div>

        <div class="sp-section">
          <div class="sp-label">
            <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M12 2a4 4 0 0 1 4 4c0 1.95-1.4 3.58-3.25 3.93"/><path d="M12 2a4 4 0 0 0-4 4c0 1.95 1.4 3.58 3.25 3.93"/><circle cx="12" cy="14" r="4"/></svg>
            ai configuration
          </div>

          <div class="ai-field">
            <label class="ai-label" for="ai-endpoint">endpoint</label>
            <input id="ai-endpoint" class="ai-input" type="text" bind:value={aiEndpoint} placeholder="https://api.openai.com/v1" spellcheck="false" />
          </div>

          <div class="ai-field">
            <label class="ai-label" for="ai-key">api key</label>
            <div class="ai-key-wrap">
              {#if showApiKey}
                <input id="ai-key" class="ai-input ai-key-input" type="text" bind:value={aiApiKey} placeholder="sk-..." spellcheck="false" />
              {:else}
                <input id="ai-key" class="ai-input ai-key-input" type="password" bind:value={aiApiKey} placeholder="sk-..." />
              {/if}
              <button class="ai-eye" on:click={() => showApiKey = !showApiKey} title={showApiKey ? 'Hide' : 'Show'}>
                {#if showApiKey}
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24"/><line x1="1" y1="1" x2="23" y2="23"/></svg>
                {:else}
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/></svg>
                {/if}
              </button>
            </div>
          </div>

          <div class="ai-field">
            <label class="ai-label" for="ai-model">model</label>
            <input id="ai-model" class="ai-input" type="text" bind:value={aiModel} placeholder="gpt-4o-mini" spellcheck="false" />
          </div>

          {#if aiStatus.msg}
            <div class="ai-status" class:status-success={aiStatus.type === 'success'} class:status-error={aiStatus.type === 'error'} class:status-info={aiStatus.type === 'info'}>
              {aiStatus.msg}
            </div>
          {/if}

          <div class="ai-actions">
            <button class="ai-btn ai-btn-test" disabled={aiTesting || aiSaving} on:click={testAI}>
              {#if aiTesting}
                <span class="ai-spinner"></span>
              {:else}
                <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><polyline points="20,6 9,17 4,12"/></svg>
              {/if}
              test
            </button>
            <button class="ai-btn ai-btn-save" disabled={aiTesting || aiSaving} on:click={saveAI}>
              {#if aiSaving}
                <span class="ai-spinner"></span>
              {:else}
                <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z"/><polyline points="17,21 17,13 7,13 7,21"/><polyline points="7,3 7,8 15,8"/></svg>
              {/if}
              save
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
{/if}

<style>
  .settings-overlay {
    position: fixed;
    inset: 0;
    background: rgba(0,0,0,0.4);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
    backdrop-filter: blur(2px);
  }
  .settings-panel {
    background: var(--bg);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    width: 440px;
    max-height: 90vh;
    overflow-y: auto;
    box-shadow: 0 16px 48px rgba(0,0,0,0.4);
  }
  .sp-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 10px 14px;
    border-bottom: 1px solid var(--border);
  }
  .sp-title {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 11px;
    font-weight: 600;
    color: var(--text-secondary);
    text-transform: lowercase;
    letter-spacing: 0.3px;
  }
  .sp-title svg { color: var(--text-dim); }
  .sp-close {
    background: none;
    border: none;
    color: var(--text-dim);
    cursor: pointer;
    padding: 3px;
    border-radius: var(--radius-sm);
    display: flex;
    align-items: center;
    transition: all .1s;
  }
  .sp-close:hover { color: var(--text-muted); background: var(--bg-hover); }

  .sp-body { padding: 12px 14px; }

  .sp-section { margin-bottom: 14px; }
  .sp-section:last-child { margin-bottom: 0; }

  .sp-label {
    display: flex;
    align-items: center;
    gap: 5px;
    font-size: 10px;
    color: var(--text-dim);
    text-transform: uppercase;
    letter-spacing: 1px;
    font-weight: 600;
    margin-bottom: 6px;
  }
  .sp-label svg { color: var(--text-dim); }

  .sp-options { display: flex; gap: 4px; }

  .sp-opt {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 5px;
    padding: 6px 8px;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--bg-card);
    color: var(--text-muted);
    cursor: pointer;
    font-size: 10px;
    font-weight: 500;
    transition: all .1s;
    flex-direction: column;
  }
  .sp-opt:hover { border-color: var(--border-hover); color: var(--text-secondary); }
  .sp-opt.active {
    border-color: var(--blue);
    color: var(--blue);
    background: var(--blue-dim);
  }
  .sp-opt svg { opacity: 0.7; }
  .sp-opt.active svg { opacity: 1; }

  .opt-sz { font-weight: 600; }
  .opt-desc { font-size: 9px; color: var(--text-dim); }
  .sp-opt.active .opt-desc { color: var(--blue); opacity: 0.7; }

  /* ── Divider ── */
  .sp-divider {
    height: 1px;
    background: var(--border);
    margin: 4px 0 10px;
  }

  /* ── AI Config fields ── */
  .ai-field { margin-bottom: 8px; }
  .ai-label {
    display: block;
    font-size: 9px;
    color: var(--text-dim);
    text-transform: uppercase;
    letter-spacing: 0.8px;
    font-weight: 600;
    margin-bottom: 3px;
  }
  .ai-label-row {
    display: flex; align-items: center; justify-content: space-between;
    margin-bottom: 3px;
  }
  .ai-label-row .ai-label { margin-bottom: 0; }
  .ai-reset {
    font-size: 9px; padding: 1px 6px;
    background: none; border: 1px solid var(--border);
    border-radius: var(--radius-sm); color: var(--text-dim);
    cursor: pointer; transition: all .1s;
  }
  .ai-reset:hover { border-color: var(--border-hover); color: var(--text-muted); }

  .ai-prompt-hint {
    font-size: 9px; color: var(--text-dim); margin-bottom: 5px;
    line-height: 1.4;
  }
  .ai-textarea {
    width: 100%; padding: 6px 8px;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--bg-card); color: var(--text);
    font-size: 10px; font-family: monospace;
    resize: vertical; outline: none;
    transition: border-color .1s; box-sizing: border-box;
    line-height: 1.5; min-height: 100px;
  }
  .ai-textarea::placeholder { color: var(--text-dim); white-space: pre-wrap; }
  .ai-textarea:focus { border-color: var(--blue); }

  .ai-input {
    width: 100%;
    padding: 5px 8px;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--bg-card);
    color: var(--text);
    font-size: 11px;
    outline: none;
    transition: border-color .1s;
    box-sizing: border-box;
  }
  .ai-input::placeholder { color: var(--text-dim); }
  .ai-input:focus { border-color: var(--blue); }

  .ai-key-wrap { display: flex; gap: 4px; }
  .ai-key-input { flex: 1; }
  .ai-eye {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 30px;
    flex-shrink: 0;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--bg-card);
    color: var(--text-dim);
    cursor: pointer;
    transition: all .1s;
  }
  .ai-eye:hover { border-color: var(--border-hover); color: var(--text-secondary); }

  .ai-status {
    font-size: 10px;
    padding: 4px 8px;
    border-radius: var(--radius);
    margin-bottom: 8px;
    font-weight: 500;
  }
  .status-success { color: var(--green); background: var(--green-dim); }
  .status-error { color: var(--red); background: var(--red-dim); }
  .status-info { color: var(--yellow); background: var(--yellow-dim); }

  .ai-actions { display: flex; gap: 6px; }
  .ai-btn {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 5px;
    padding: 6px 10px;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--bg-card);
    color: var(--text-muted);
    font-size: 10px;
    font-weight: 500;
    cursor: pointer;
    transition: all .1s;
  }
  .ai-btn:hover { border-color: var(--border-hover); color: var(--text-secondary); }
  .ai-btn:disabled { opacity: 0.4; pointer-events: none; }
  .ai-btn-test:hover { border-color: var(--blue); color: var(--blue); }
  .ai-btn-save:hover { border-color: var(--green); color: var(--green); background: var(--green-dim); }

  .ai-spinner {
    width: 10px;
    height: 10px;
    border: 2px solid var(--border-active);
    border-top-color: var(--text-secondary);
    border-radius: 50%;
    animation: spin 0.6s linear infinite;
  }
  @keyframes spin { to { transform: rotate(360deg); } }
</style>
