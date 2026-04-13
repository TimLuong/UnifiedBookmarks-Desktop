<script lang="ts">
  import { activeFolder, activeFolderUrls } from './store';
  import type { FolderNode } from './store';

  export let nodes: FolderNode[];
  export let depth: number = 0;
  export let profileKey: string = '';

  // Track expanded state per node (keyed by profileKey + path)
  let expanded: Record<string, boolean> = {};

  function toggle(key: string) {
    expanded[key] = !expanded[key];
    expanded = expanded;
  }

  function nodeKey(node: FolderNode) {
    return `${profileKey}__${node.path}`;
  }

  // Collect all URLs from a node and its descendants recursively
  function collectAllUrls(node: FolderNode): string[] {
    const urls = [...(node.urls || [])];
    for (const child of (node.children || [])) {
      urls.push(...collectAllUrls(child));
    }
    return urls;
  }

  function setFolder(node: FolderNode) {
    const path = node.path;
    activeFolder.update(v => {
      if (v === path) {
        // deselect
        activeFolderUrls.set(null);
        return null;
      }
      activeFolderUrls.set(new Set(collectAllUrls(node)));
      return path;
    });
  }
</script>

{#each nodes as node (nodeKey(node))}
  <div class="folder-item" style="--depth:{depth}">
    <button
      class="folder-btn"
      class:active={$activeFolder === node.path}
      on:click={() => setFolder(node)}
    >
      <span class="folder-indent" style="width:{depth * 12}px"></span>
      {#if node.children && node.children.length > 0}
        <button class="expand-btn" on:click|stopPropagation={() => toggle(nodeKey(node))}>
          <svg class="exp-chevron" class:open={expanded[nodeKey(node)]}
            width="9" height="9" viewBox="0 0 24 24" fill="none"
            stroke="currentColor" stroke-width="2.5" stroke-linecap="round">
            <polyline points="9,18 15,12 9,6"/>
          </svg>
        </button>
      {:else}
        <span class="expand-spacer"></span>
      {/if}
      <svg class="folder-icon" width="13" height="13" viewBox="0 0 24 24"
        fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round">
        <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2v11z"/>
      </svg>
      <span class="folder-name">{node.name}</span>
      <span class="folder-count">{node.count}</span>
    </button>
    {#if node.children && node.children.length > 0 && expanded[nodeKey(node)]}
      <svelte:self nodes={node.children} depth={depth + 1} {profileKey} />
    {/if}
  </div>
{/each}

<style>
  .folder-item { display: flex; flex-direction: column; }

  .folder-btn {
    display: flex;
    align-items: center;
    gap: 4px;
    width: 100%;
    padding: 4px 8px 4px 6px;
    background: none;
    border: none;
    color: var(--text-secondary);
    font-size: 13px;
    cursor: pointer;
    text-align: left;
    border-radius: var(--radius-sm);
    transition: background .07s;
    min-height: 28px;
  }
  .folder-btn:hover { background: var(--bg-hover); color: var(--text); }
  .folder-btn.active { background: var(--bg-active); color: var(--text); }

  .expand-btn {
    background: none;
    border: none;
    cursor: pointer;
    padding: 0;
    display: flex;
    align-items: center;
    color: var(--text-dim);
    flex-shrink: 0;
  }
  .expand-btn:hover { color: var(--text-muted); }
  .expand-spacer { width: 9px; flex-shrink: 0; }

  .exp-chevron { transition: transform .15s; }
  .exp-chevron.open { transform: rotate(90deg); }

  .folder-icon { color: var(--text-dim); flex-shrink: 0; }
  .folder-btn.active .folder-icon { color: var(--accent); }

  .folder-name {
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: 13px;
  }
  .folder-count {
    font-size: 11px;
    color: var(--text-dim);
    margin-left: auto;
    flex-shrink: 0;
  }
</style>
