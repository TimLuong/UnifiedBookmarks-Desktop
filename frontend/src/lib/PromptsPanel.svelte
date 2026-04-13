<script lang="ts">
  import { showPrompts } from './store';
  import type { PromptProfile } from './store';

  function callGo(method: string, ...args: any[]): Promise<any> {
    return (window as any)['go']['main']['App'][method](...args);
  }

  let profiles: PromptProfile[] = [];
  let activeId = 'default';
  let selectedId = 'default';
  let editName = '';
  let editContent = '';
  let status = '';
  let saving = false;

  // Folder structure settings
  let fsMaxDepth = 3;
  let fsMinItems = 3;
  let fsMaxItems = 30;
  let fsSmartRename = false;
  let fsSortAlpha = false;
  let fsStatus = '';
  let fsSaving = false;

  $: selected = profiles.find(p => p.id === selectedId) ?? null;

  function selectProfile(id: string) {
    selectedId = id;
    status = '';
    const p = profiles.find(p => p.id === id);
    if (p) { editName = p.name; editContent = p.content; }
  }

  async function load() {
    try {
      const res = await callGo('GetPromptProfiles');
      profiles = res.profiles || [];
      activeId = res.activeId || 'default';
      selectProfile(activeId);
    } catch {}
    try {
      const fs = await callGo('GetFolderSettings');
      fsMaxDepth = fs.maxDepth ?? 3;
      fsMinItems = fs.minFolderItems ?? 3;
      fsMaxItems = fs.maxFolderItems ?? 30;
      fsSmartRename = fs.smartRenamePrefix ?? false;
      fsSortAlpha = fs.sortAlphaInFolder ?? false;
    } catch {}
  }

  async function saveFolderSettings() {
    fsSaving = true;
    fsStatus = '';
    try {
      await callGo('SaveFolderSettings', { maxDepth: fsMaxDepth, minFolderItems: fsMinItems, maxFolderItems: fsMaxItems, smartRenamePrefix: fsSmartRename, sortAlphaInFolder: fsSortAlpha });
      fsStatus = '✓ saved — takes effect on next analysis';
    } catch (e: any) { fsStatus = `✗ ${e.message}`; }
    fsSaving = false;
  }

  async function activate() {
    if (!selected) return;
    try {
      await callGo('SetActivePromptID', selected.id);
      activeId = selected.id;
      status = '✓ activated';
    } catch (e: any) { status = `✗ ${e.message}`; }
  }

  async function saveEdit() {
    if (!selected || selected.isBuiltin) return;
    saving = true;
    const updated = profiles.map(p =>
      p.id === selectedId ? { ...p, name: editName, content: editContent } : p
    );
    try {
      await callGo('SaveCustomProfiles', updated.filter(p => !p.isBuiltin));
      profiles = updated;
      status = '✓ saved';
    } catch (e: any) { status = `✗ ${e.message}`; }
    saving = false;
  }

  async function duplicate() {
    if (!selected) return;
    const newId = 'custom-' + Date.now();
    const np: PromptProfile = {
      id: newId, name: `${selected.name} (copy)`,
      description: '', content: selected.content, isBuiltin: false,
    };
    const custom = [...profiles.filter(p => !p.isBuiltin), np];
    try {
      await callGo('SaveCustomProfiles', custom);
      profiles = [...profiles, np];
      selectProfile(newId);
      status = '✓ duplicated';
    } catch (e: any) { status = `✗ ${e.message}`; }
  }

  async function deleteProfile() {
    if (!selected || selected.isBuiltin) return;
    if (!confirm(`Delete "${selected.name}"?`)) return;
    const custom = profiles.filter(p => !p.isBuiltin && p.id !== selectedId);
    try {
      await callGo('SaveCustomProfiles', custom);
      profiles = profiles.filter(p => p.id !== selectedId);
      selectProfile('default');
      status = '✓ deleted';
    } catch (e: any) { status = `✗ ${e.message}`; }
  }

  async function addNew() {
    const newId = 'custom-' + Date.now();
    const np: PromptProfile = { id: newId, name: 'New Profile', description: '', content: '', isBuiltin: false };
    const custom = [...profiles.filter(p => !p.isBuiltin), np];
    try {
      await callGo('SaveCustomProfiles', custom);
      profiles = [...profiles, np];
      selectProfile(newId);
    } catch {}
  }

  $: if ($showPrompts) load();
</script>

{#if $showPrompts}
  <div class="pp-overlay"
    on:click|self={() => showPrompts.set(false)}
    on:keydown|self={(e) => e.key === 'Escape' && showPrompts.set(false)}
    role="dialog" aria-modal="true" tabindex="-1">
    <div class="pp-panel">
      <div class="pp-header">
        <div class="pp-title">
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
            <polygon points="13,2 3,14 12,14 11,22 21,10 12,10 13,2"/>
          </svg>
          prompt profiles
        </div>
        <button class="pp-close" on:click={() => showPrompts.set(false)}>
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
            <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
          </svg>
        </button>
      </div>

      <div class="pp-body">
        <!-- Left: profile list -->
        <div class="pp-list">
          {#each profiles as p (p.id)}
            <button class="pp-item"
              class:pp-selected={selectedId === p.id}
              class:pp-item-active={activeId === p.id}
              on:click={() => selectProfile(p.id)}>
              <div class="pp-item-row">
                <span class="pp-item-name">{p.name}</span>
                {#if activeId === p.id}
                  <span class="pp-badge">active</span>
                {/if}
              </div>
              <span class="pp-item-sub">{p.isBuiltin ? 'built-in' : 'custom'}{p.description ? ' · ' + p.description : ''}</span>
            </button>
          {/each}
          <button class="pp-add-btn" on:click={addNew}>
            <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
              <line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>
            </svg>
            new profile
          </button>
        </div>

        <!-- Right: editor -->
        <div class="pp-editor">
          {#if selected}
            <div class="pp-field">
              <label class="pp-lbl">name</label>
              {#if selected.isBuiltin}
                <div class="pp-val">{selected.name} <span class="pp-builtin-tag">built-in</span></div>
              {:else}
                <input class="pp-input" bind:value={editName} spellcheck="false" />
              {/if}
            </div>

            <div class="pp-field pp-field-grow">
              <div class="pp-lbl-row">
                <label class="pp-lbl">instructions</label>
                <span class="pp-hint">prepended to built-in prompt · JSON format always preserved</span>
              </div>
              {#if selected.isBuiltin && !selected.content}
                <div class="pp-info">Uses the built-in PARA classifier with no extra instructions.<br/>Click <strong>duplicate</strong> to create an editable copy.</div>
              {:else}
                <textarea
                  class="pp-textarea"
                  class:is-readonly={selected.isBuiltin}
                  readonly={selected.isBuiltin}
                  bind:value={editContent}
                  placeholder="Enter your instructions for the AI…&#10;&#10;Example: Use exactly 5 categories: Work, Dev, Learning, Health, Other."
                  spellcheck="false"
                ></textarea>
              {/if}
            </div>

            {#if status}
              <div class="pp-status" class:pp-ok={status.startsWith('✓')}>{status}</div>
            {/if}

            <div class="pp-actions">
              <button class="pp-btn" on:click={duplicate} title="Create editable copy">duplicate</button>
              {#if !selected.isBuiltin}
                <button class="pp-btn pp-danger" on:click={deleteProfile}>delete</button>
                <button class="pp-btn pp-save" disabled={saving} on:click={saveEdit}>{saving ? '…' : 'save'}</button>
              {/if}
              <span class="pp-spacer"></span>
              <button class="pp-btn pp-activate"
                class:is-active={activeId === selected.id}
                on:click={activate}>
                {#if activeId === selected.id}
                  <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round"><polyline points="20,6 9,17 4,12"/></svg>
                  active
                {:else}
                  use for analysis
                {/if}
              </button>
            </div>
          {/if}
        </div>
      </div>

      <!-- Folder Structure Rules -->
      <div class="pp-folder-rules">
        <div class="pp-fr-header">
          <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
            <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
          </svg>
          <span class="pp-fr-title">folder structure rules</span>
          <span class="pp-hint">injected into AI prompt + applied as post-processing</span>
        </div>
        <div class="pp-fr-row">
          <div class="pp-fr-field">
            <label class="pp-lbl" for="fr-depth">max depth</label>
            <div class="pp-fr-input-wrap">
              <input id="fr-depth" class="pp-input pp-fr-num" type="number" min="1" max="6" bind:value={fsMaxDepth} />
              <span class="pp-fr-unit">levels</span>
            </div>
          </div>
          <div class="pp-fr-field">
            <label class="pp-lbl" for="fr-min">min URLs / folder</label>
            <div class="pp-fr-input-wrap">
              <input id="fr-min" class="pp-input pp-fr-num" type="number" min="1" max="20" bind:value={fsMinItems} />
              <span class="pp-fr-unit">bookmarks</span>
            </div>
          </div>
          <div class="pp-fr-field">
            <label class="pp-lbl" for="fr-max">max URLs / folder</label>
            <div class="pp-fr-input-wrap">
              <input id="fr-max" class="pp-input pp-fr-num" type="number" min="5" max="500" bind:value={fsMaxItems} />
              <span class="pp-fr-unit">bookmarks</span>
            </div>
          </div>
          <button class="pp-btn pp-save" style="align-self:flex-end" disabled={fsSaving} on:click={saveFolderSettings}>
            {fsSaving ? '…' : 'save'}
          </button>
        </div>
        <div class="pp-fr-checks">
          <label class="pp-fr-check">
            <input type="checkbox" bind:checked={fsSmartRename} />
            <span class="pp-fr-check-label">
              <strong>smart rename</strong> — prepend type prefix to title
              <span class="pp-hint">[PBI], [SP], [DOC], [SHEET], [GH], [YT]…</span>
            </span>
          </label>
          <label class="pp-fr-check">
            <input type="checkbox" bind:checked={fsSortAlpha} />
            <span class="pp-fr-check-label">
              <strong>sort A→Z</strong> — sort bookmarks alphabetically within each folder on sync
            </span>
          </label>
        </div>
        {#if fsStatus}
          <div class="pp-status" class:pp-ok={fsStatus.startsWith('✓')} style="margin-top:4px">{fsStatus}</div>
        {/if}
      </div>
    </div>
  </div>
{/if}

<style>
  .pp-overlay {
    position: fixed; inset: 0; background: rgba(0,0,0,0.45);
    display: flex; align-items: center; justify-content: center;
    z-index: 1000; backdrop-filter: blur(2px);
  }
  .pp-panel {
    background: var(--bg); border: 1px solid var(--border); border-radius: var(--radius);
    width: 640px; max-height: 85vh; display: flex; flex-direction: column;
    box-shadow: 0 20px 60px rgba(0,0,0,0.5);
  }
  .pp-header {
    display: flex; align-items: center; justify-content: space-between;
    padding: 10px 14px; border-bottom: 1px solid var(--border); flex-shrink: 0;
  }
  .pp-title {
    display: flex; align-items: center; gap: 6px;
    font-size: 11px; font-weight: 600; color: var(--text-secondary);
    text-transform: lowercase; letter-spacing: 0.3px;
  }
  .pp-title svg { color: var(--accent); }
  .pp-close {
    background: none; border: none; color: var(--text-dim); cursor: pointer;
    padding: 3px; border-radius: var(--radius-sm); display: flex; align-items: center; transition: all .1s;
  }
  .pp-close:hover { color: var(--text-muted); background: var(--bg-hover); }

  .pp-body { display: flex; flex: 1; overflow: hidden; min-height: 0; }

  /* Left list */
  .pp-list {
    width: 215px; flex-shrink: 0; border-right: 1px solid var(--border);
    overflow-y: auto; padding: 6px; display: flex; flex-direction: column; gap: 2px;
  }
  .pp-item {
    width: 100%; text-align: left; padding: 7px 9px;
    border: 1px solid transparent; border-radius: var(--radius);
    background: none; cursor: pointer; transition: all .1s; color: var(--text-secondary);
  }
  .pp-item:hover { background: var(--bg-hover); }
  .pp-item.pp-selected { background: var(--bg-card); border-color: var(--border); }
  .pp-item.pp-item-active .pp-item-name { color: var(--green); }
  .pp-item-row { display: flex; align-items: center; gap: 4px; }
  .pp-item-name { font-size: 11px; font-weight: 500; flex: 1; min-width: 0; }
  .pp-badge {
    font-size: 9px; padding: 1px 5px; background: var(--green-dim); color: var(--green);
    border-radius: 10px; font-weight: 600; flex-shrink: 0;
  }
  .pp-item-sub { font-size: 9px; color: var(--text-dim); margin-top: 2px; display: block;
    white-space: nowrap; overflow: hidden; text-overflow: ellipsis; max-width: 190px; }
  .pp-add-btn {
    display: flex; align-items: center; gap: 5px; width: 100%; padding: 6px 9px; margin-top: 4px;
    border: 1px dashed var(--border); border-radius: var(--radius);
    background: none; cursor: pointer; color: var(--text-dim); font-size: 10px; transition: all .1s;
  }
  .pp-add-btn:hover { border-color: var(--accent); color: var(--accent); }

  /* Right editor */
  .pp-editor {
    flex: 1; display: flex; flex-direction: column; padding: 12px 14px;
    overflow-y: auto; gap: 8px; min-width: 0;
  }
  .pp-field { display: flex; flex-direction: column; gap: 3px; }
  .pp-field-grow { flex: 1; min-height: 0; display: flex; flex-direction: column; gap: 3px; }
  .pp-lbl-row { display: flex; align-items: center; gap: 6px; }
  .pp-lbl {
    font-size: 9px; color: var(--text-dim); text-transform: uppercase;
    letter-spacing: 0.8px; font-weight: 600;
  }
  .pp-hint { font-size: 9px; color: var(--text-dim); opacity: 0.6; }
  .pp-val { font-size: 11px; color: var(--text-secondary); padding: 2px 0; display: flex; align-items: center; gap: 6px; }
  .pp-builtin-tag {
    font-size: 9px; padding: 1px 5px; border: 1px solid var(--border);
    border-radius: 10px; color: var(--text-dim);
  }
  .pp-input {
    padding: 5px 8px; border: 1px solid var(--border); border-radius: var(--radius);
    background: var(--bg-card); color: var(--text); font-size: 11px;
    outline: none; transition: border-color .1s; box-sizing: border-box; width: 100%;
  }
  .pp-input:focus { border-color: var(--blue); }
  .pp-textarea {
    flex: 1; padding: 7px 9px; border: 1px solid var(--border); border-radius: var(--radius);
    background: var(--bg-card); color: var(--text); font-size: 10px; font-family: monospace;
    resize: none; outline: none; transition: border-color .1s; line-height: 1.55; min-height: 140px;
    box-sizing: border-box; width: 100%;
  }
  .pp-textarea:not(.is-readonly):focus { border-color: var(--blue); }
  .pp-textarea.is-readonly { opacity: 0.65; cursor: default; }
  .pp-info {
    font-size: 10px; color: var(--text-dim); padding: 12px; line-height: 1.6;
    border: 1px solid var(--border); border-radius: var(--radius); background: var(--bg-card);
  }
  .pp-status {
    font-size: 10px; padding: 4px 8px; border-radius: var(--radius);
    color: var(--red); background: var(--red-dim); font-weight: 500; flex-shrink: 0;
  }
  .pp-ok { color: var(--green); background: var(--green-dim); }
  .pp-actions { display: flex; align-items: center; gap: 5px; flex-shrink: 0; }
  .pp-spacer { flex: 1; }
  .pp-btn {
    display: flex; align-items: center; gap: 4px; padding: 5px 10px;
    border: 1px solid var(--border); border-radius: var(--radius);
    background: var(--bg-card); color: var(--text-muted); font-size: 10px;
    font-weight: 500; cursor: pointer; transition: all .1s;
  }
  .pp-btn:hover { border-color: var(--border-hover); color: var(--text-secondary); }
  .pp-btn:disabled { opacity: 0.4; pointer-events: none; }
  .pp-danger:hover { border-color: var(--red); color: var(--red); }
  .pp-save:hover { border-color: var(--green); color: var(--green); background: var(--green-dim); }
  .pp-activate { border-color: var(--accent-dim); color: var(--accent); font-weight: 600; padding: 5px 12px; }
  .pp-activate:hover { background: var(--accent-dim); border-color: var(--accent); }
  .pp-activate.is-active { border-color: var(--green-dim); color: var(--green); background: var(--green-dim); }

  /* Folder rules section */
  .pp-folder-rules {
    border-top: 1px solid var(--border); padding: 10px 14px 12px;
    flex-shrink: 0; display: flex; flex-direction: column; gap: 6px;
  }
  .pp-fr-header {
    display: flex; align-items: center; gap: 5px;
  }
  .pp-fr-header svg { color: var(--accent); flex-shrink: 0; }
  .pp-fr-title {
    font-size: 9px; font-weight: 600; color: var(--text-dim);
    text-transform: uppercase; letter-spacing: 0.8px;
  }
  .pp-fr-row {
    display: flex; align-items: flex-end; gap: 10px; flex-wrap: wrap;
  }
  .pp-fr-field {
    display: flex; flex-direction: column; gap: 3px;
  }
  .pp-fr-input-wrap {
    display: flex; align-items: center; gap: 5px;
  }
  .pp-fr-num {
    width: 56px; text-align: center; padding: 4px 6px;
  }
  .pp-fr-unit {
    font-size: 9px; color: var(--text-dim);
  }
  .pp-fr-checks {
    display: flex; flex-direction: column; gap: 5px; margin-top: 2px;
  }
  .pp-fr-check {
    display: flex; align-items: flex-start; gap: 7px; cursor: pointer;
    font-size: 10px; color: var(--text-secondary); user-select: none;
  }
  .pp-fr-check input[type="checkbox"] {
    margin-top: 2px; flex-shrink: 0; accent-color: var(--accent);
    cursor: pointer;
  }
  .pp-fr-check-label { display: flex; flex-direction: column; gap: 1px; line-height: 1.4; }
  .pp-fr-check-label strong { font-weight: 600; color: var(--text); }
</style>
