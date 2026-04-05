<script lang="ts">
  import { onMount } from 'svelte';
  import { Clipboard } from '@wailsio/runtime';
  import KiCadImportWorkspace from './KiCadImportWorkspace.svelte';
  import Modal from '../ui/Modal.svelte';
  import { createLauncherWorkspaceStore } from './launcherWorkspaceStore';
  import {
    createProjectWithDisk,
    deleteProjectAndDisk,
    formatDate,
    getProjectDiskPath,
    revealProjectInFileBrowser,
    setRecentProjectPinned,
    type RecentProject,
  } from '../backend';
  import { setLauncherView } from '../windowService';

  let {
    onOpenProject,
    onOpenProjectKeepLauncher,
  }: {
    onOpenProject?: (id: string) => void;
    onOpenProjectKeepLauncher?: (id: string) => void;
  } = $props();

  const {
    recent,
    openProjectIDs,
    filterText,
    loading,
    visibleProjects,
    init,
    refreshRecent,
    refreshOpenProjects,
  } = createLauncherWorkspaceStore();

  let menuOpen = $state(false);
  let menuX = $state(0);
  let menuY = $state(0);
  let menuProject: RecentProject | null = $state(null);
  let menuBusy = $state(false);

  let newProjectOpen = $state(false);
  let newProjectName = $state('');
  let newProjectDescription = $state('');
  let creatingProject = $state(false);
  let createProjectError = $state('');
  let currentView: 'launcher' | 'kicad' = $state('launcher');

  $effect(() => {
    if (newProjectOpen) {
      newProjectName = '';
      newProjectDescription = '';
      createProjectError = '';
      creatingProject = false;
    }
  });

  onMount(async () => {
    await init();
  });

  onMount(() => {
    const onFocus = () => {
      void refreshOpenProjects();
    };
    window.addEventListener('focus', onFocus);
    return () => window.removeEventListener('focus', onFocus);
  });

  onMount(() => {
    const onMouseDown = (e: MouseEvent) => {
      if (!menuOpen) return;
      const target = e.target as HTMLElement | null;
      if (target && target.closest('.context-menu')) return;
      menuOpen = false;
      menuProject = null;
    };

    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && menuOpen) {
        menuOpen = false;
        menuProject = null;
      }
    };

    window.addEventListener('mousedown', onMouseDown);
    window.addEventListener('keydown', onKeyDown);
    return () => {
      window.removeEventListener('mousedown', onMouseDown);
      window.removeEventListener('keydown', onKeyDown);
    };
  });

  const comingSoon = (e: MouseEvent) => {
    e.preventDefault();
    console.log('coming-soon');
  };

  function nextFrame(): Promise<void> {
    return new Promise((resolve) => window.requestAnimationFrame(() => resolve()));
  }

  async function resizeLauncherWindow(view: 'launcher' | 'kicad-import') {
    try {
      await setLauncherView(view);
    } catch (err) {
      console.error('failed to resize launcher window', err);
    }
  }

  async function transitionToView(view: 'launcher' | 'kicad') {
    if (currentView === view) {
      return;
    }

    const launcherView = view === 'kicad' ? 'kicad-import' : 'launcher';
    await resizeLauncherWindow(launcherView);
    await nextFrame();
    await nextFrame();
    currentView = view;
  }

  function revealLabel(): string {
    const ua = navigator.userAgent;
    if (/Windows/i.test(ua)) return 'Show in Explorer';
    if (/Mac/i.test(ua)) return 'Reveal in Finder';
    return 'Show in File Manager';
  }

  function openMenu(e: MouseEvent, p: RecentProject) {
    e.preventDefault();
    menuOpen = true;
    menuProject = p;
    menuX = e.clientX;
    menuY = e.clientY;
  }

  function closeMenu() {
    menuOpen = false;
    menuProject = null;
  }

  function handleOpen() {
    if (!menuProject) return;
    const id = menuProject.id;
    closeMenu();
    void openWithAlreadyOpenDialog(id, false);
  }

  function handleOpenInNewWindow() {
    if (!menuProject) return;
    const id = menuProject.id;
    closeMenu();
    void openWithAlreadyOpenDialog(id, true);
  }

  async function openWithAlreadyOpenDialog(projectId: string, keepLauncher: boolean) {
    try {
      await refreshOpenProjects();
    } catch {
    }
    if ($openProjectIDs.includes(projectId)) {
      alert('That project is already open.');
      onOpenProjectKeepLauncher?.(projectId);
      void refreshOpenProjects();
      return;
    }
    if (keepLauncher) {
      onOpenProjectKeepLauncher?.(projectId);
    } else {
      onOpenProject?.(projectId);
    }
    void refreshOpenProjects();
  }

  async function handleCopyPath() {
    if (!menuProject) return;
    menuBusy = true;
    try {
      const path = await getProjectDiskPath(menuProject.id);
      if (Clipboard?.SetText) {
        await Clipboard.SetText(path);
      } else if (navigator.clipboard?.writeText) {
        await navigator.clipboard.writeText(path);
      }
      closeMenu();
    } finally {
      menuBusy = false;
    }
  }

  async function handleReveal() {
    if (!menuProject) return;
    menuBusy = true;
    try {
      await revealProjectInFileBrowser(menuProject.id);
      closeMenu();
    } finally {
      menuBusy = false;
    }
  }

  async function handleDelete() {
    if (!menuProject) return;
    const name = menuProject.name || menuProject.id;
    if (!confirm(`Delete project "${name}"? This will remove it from disk and the database.`)) return;
    menuBusy = true;
    try {
      const deletingId = menuProject.id;
      await deleteProjectAndDisk(deletingId);
      recent.update((r) => r.filter((p) => p.id !== deletingId));
      closeMenu();
      await refreshOpenProjects();
    } finally {
      menuBusy = false;
    }
  }

  async function handleTogglePin() {
    if (!menuProject) return;
    menuBusy = true;
    try {
      await setRecentProjectPinned(menuProject.id, !menuProject.pinned);
      closeMenu();
      await refreshRecent();
    } finally {
      menuBusy = false;
    }
  }

  async function handleCreateProject(e: SubmitEvent) {
    e.preventDefault();
    const name = newProjectName.trim();
    if (!name) return;

    creatingProject = true;
    createProjectError = '';
    try {
      const p = await createProjectWithDisk({ name, description: newProjectDescription });
      newProjectOpen = false;
      await refreshRecent();
      onOpenProject?.(p.id);
    } catch (err: any) {
      createProjectError = err?.message ?? String(err);
    } finally {
      creatingProject = false;
    }
  }

  function openKiCadImport() {
    closeMenu();
    void transitionToView('kicad');
  }

  async function handleImportedProject(projectId: string) {
	await refreshRecent();
	await openWithAlreadyOpenDialog(projectId, false);
  }
</script>

<div class="launcher-shell" class:show-import={currentView === 'kicad'}>
    <section class="workspace-pane workspace-pane-main" aria-hidden={currentView === 'kicad'}>
      <div class="launcher">
        <section class="left">
          <header class="left-header">
            <div>
              <h1 class="title">Launcher</h1>
              <p class="subtitle">Pick up where you left off</p>
            </div>
          </header>

          <div class="recent">
            <div class="search-row">
              <input
                class="search-input"
                type="text"
                placeholder="Search projects…"
                bind:value={$filterText}
              />
            </div>
            {#if $loading}
              <div class="empty">Loading recent projects…</div>
            {:else if $recent.length === 0}
              <div class="empty">No recent projects yet</div>
            {:else}
              <div class="recent-list">
                {#each $visibleProjects as p (p.id)}
                  <button
                    class="recent-item"
                    onclick={() => void openWithAlreadyOpenDialog(p.id, false)}
                    oncontextmenu={(e) => openMenu(e, p)}
                  >
                    <div class="recent-row">
                      <div class="recent-name">{p.name}</div>
                      <div class="recent-badges">
                        {#if p.pinned}
                          <span class="badge">Pinned</span>
                        {/if}
                        {#if $openProjectIDs.includes(p.id)}
                          <span class="badge">Open</span>
                        {/if}
                      </div>
                    </div>
                    {#if p.subtitle}
                      <div class="recent-sub">{p.subtitle}</div>
                    {/if}
                    {#if p.openedAtUtc}
                      <div class="recent-meta">Opened {formatDate(p.openedAtUtc)}</div>
                    {/if}
                  </button>
                {/each}
              </div>
            {/if}
          </div>
        </section>

        <section class="right">
          <div class="actions">
            <div class="actions-label">New Project</div>
            <button class="action-row" onclick={() => (newProjectOpen = true)}>
              <div class="action-icon">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
                  <path d="M14 2v6h6" />
                  <path d="M12 11v6" />
                  <path d="M9 14h6" />
                </svg>
              </div>
              <div class="action-text">
                <span class="action-title">Start blank</span>
                <span class="action-desc">Create a new project from scratch</span>
              </div>
            </button>

            <button class="action-row" disabled onclick={comingSoon}>
              <div class="action-icon muted">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <path d="M16 4h2a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2v-2" />
                  <path d="M8 2h10v12H8z" />
                  <path d="M10 9h6" />
                  <path d="M10 6h6" />
                </svg>
              </div>
              <div class="action-text">
                <span class="action-title">Paste rough parts list</span>
                <span class="action-soon">Soon</span>
              </div>
            </button>

            <button class="action-row" onclick={openKiCadImport}>
              <div class="action-icon">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
                  <path d="M7 10l5 5 5-5" />
                  <path d="M12 15V3" />
                </svg>
              </div>
              <div class="action-text">
                <span class="action-title">Import from KiCad</span>
                <span class="action-desc">Discover `.kicad_pro` projects and review the generated requirements first</span>
              </div>
            </button>

            <button class="action-row" disabled onclick={comingSoon}>
              <div class="action-icon muted">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <rect x="7" y="7" width="10" height="10" rx="2" />
                  <path d="M9 1v4" />
                  <path d="M15 1v4" />
                  <path d="M9 19v4" />
                  <path d="M15 19v4" />
                  <path d="M1 9h4" />
                  <path d="M1 15h4" />
                  <path d="M19 9h4" />
                  <path d="M19 15h4" />
                </svg>
              </div>
              <div class="action-text">
                <span class="action-title">Start from core IC</span>
                <span class="action-soon">Soon</span>
              </div>
            </button>
          </div>
        </section>
      </div>
    </section>

    <section class="workspace-pane workspace-pane-import" aria-hidden={currentView !== 'kicad'}>
      <KiCadImportWorkspace
        active={currentView === 'kicad'}
        onBack={() => void transitionToView('launcher')}
        onImportedProject={(id) => void handleImportedProject(id)}
      />
    </section>
</div>

<Modal
  open={newProjectOpen}
  title="New Project"
  width="460px"
  onclose={() => {
    if (!creatingProject) newProjectOpen = false;
  }}
>
  <form onsubmit={handleCreateProject} class="create-form">
    <div class="form-group">
      <label for="np-name">Name</label>
      <input
        id="np-name"
        class="form-input"
        type="text"
        bind:value={newProjectName}
        placeholder="Project name"
        required
      />
    </div>

    <div class="form-group">
      <label for="np-desc">Description</label>
      <textarea
        id="np-desc"
        class="form-input"
        bind:value={newProjectDescription}
        rows="3"
        placeholder="Optional description"
      ></textarea>
    </div>

    {#if createProjectError}
      <div class="error-text">{createProjectError}</div>
    {/if}

    <div class="form-actions">
      <button type="button" class="btn btn-secondary" onclick={() => (newProjectOpen = false)} disabled={creatingProject}>
        Cancel
      </button>
      <button type="submit" class="btn btn-primary" disabled={creatingProject || !newProjectName.trim()}>
        {creatingProject ? 'Creating…' : 'Create Project'}
      </button>
    </div>
  </form>
</Modal>

{#if menuOpen && menuProject}
  <div class="context-menu" style={`left:${menuX}px;top:${menuY}px;`}>
    <button class="menu-item" onclick={handleOpen} disabled={menuBusy}>
      Open
    </button>
    <button class="menu-item" onclick={handleOpenInNewWindow} disabled={menuBusy}>
      Open in New Window
    </button>
    <div class="menu-sep"></div>
    <button class="menu-item" onclick={handleCopyPath} disabled={menuBusy}>
      Copy Path
    </button>
    <button class="menu-item" onclick={handleReveal} disabled={menuBusy}>
      {revealLabel()}
    </button>
    <button class="menu-item" onclick={handleTogglePin} disabled={menuBusy}>
      {menuProject.pinned ? 'Unpin' : 'Pin'}
    </button>
    <div class="menu-sep"></div>
    <button class="menu-item danger" onclick={handleDelete} disabled={menuBusy}>
      Delete
    </button>
  </div>
{/if}

<style>
  .launcher-shell {
    position: relative;
    height: 100%;
    overflow: hidden;
    background: var(--color-bg-app);
  }

  .workspace-pane {
    position: absolute;
    inset: 0;
    min-width: 0;
    min-height: 0;
    overflow: hidden;
    background: var(--color-bg-app);
    transition:
      transform 280ms cubic-bezier(0.22, 1, 0.36, 1),
      opacity 220ms ease;
    will-change: transform, opacity;
    backface-visibility: hidden;
    contain: layout paint;
  }

  .workspace-pane-main {
    z-index: 1;
    transform: translate3d(0, 0, 0);
  }

  .workspace-pane-import {
    z-index: 2;
    transform: translate3d(100%, 0, 0);
  }

  .launcher-shell.show-import .workspace-pane-main {
    transform: translate3d(-100%, 0, 0);
  }

  .launcher-shell.show-import .workspace-pane-import {
    transform: translate3d(0, 0, 0);
  }

  .launcher-shell:not(.show-import) .workspace-pane-import {
    pointer-events: none;
  }

  .launcher-shell.show-import .workspace-pane-main {
    pointer-events: none;
  }

  .launcher {
    position: relative;
    height: 100%;
    display: grid;
    grid-template-columns: 360px 1fr;
    gap: 0;
    padding: 0;
  }

  .context-menu {
    position: fixed;
    z-index: 9999;
    width: 220px;
    background: var(--color-bg-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    overflow: hidden;
    box-shadow: var(--shadow-md);
  }

  .menu-item {
    width: 100%;
    text-align: left;
    padding: 10px 12px;
    background: transparent;
    border: 0;
    color: var(--color-text-primary);
    cursor: pointer;
    font-size: 13px;
  }

  .menu-item:hover:not(:disabled) {
    background: var(--color-bg-hover);
  }

  .menu-item:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  .menu-item.danger {
    color: var(--color-danger);
  }

  .menu-sep {
    height: 1px;
    background: var(--color-border);
  }

  .create-form {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .form-actions {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
    padding-top: 8px;
  }

  .error-text {
    font-size: 12px;
    color: var(--color-danger);
  }

  .left,
  .right {
    background: var(--color-bg-app);
    border: 0;
    border-radius: 0;
    overflow: hidden;
  }

  .left {
    display: flex;
    flex-direction: column;
    border-right: 1px solid var(--color-border);
    background: var(--color-bg-sidebar);
  }

  .left-header {
    padding: 14px 14px 10px 14px;
    display: flex;
    justify-content: space-between;
    gap: 12px;
    align-items: flex-start;
    border-bottom: 1px solid var(--color-border);
  }

  .title {
    font-size: 13px;
    font-weight: 600;
    color: var(--color-text-primary);
  }

  .subtitle {
    margin-top: 2px;
    color: var(--color-text-muted);
    font-size: 11px;
  }

  .recent {
    padding: 8px;
    display: flex;
    flex-direction: column;
    gap: 6px;
    flex: 1 1 0;
    min-height: 0;
    overflow: hidden;
  }

  .recent > .empty,
  .recent > .search-row,
  .recent > div {
    flex-shrink: 0;
  }

  .recent-list {
    display: flex;
    flex-direction: column;
    gap: 0;
    flex: 1 1 0;
    overflow-y: auto;
    overflow-x: hidden;
    min-height: 0;
    border-top: 1px solid var(--color-border);
  }

  .search-row {
    padding: 2px;
  }

  .search-input {
    width: 100%;
    padding: 8px 10px;
    border-radius: var(--radius-md);
    border: 1px solid var(--color-border);
    background: var(--color-bg-surface);
    color: var(--color-text-primary);
    font-size: 13px;
  }

  .search-input:focus {
    outline: none;
    border-color: var(--color-border-strong);
  }

  .empty {
    padding: 14px 12px;
    color: var(--color-text-muted);
    font-size: 12px;
  }

  .recent-item {
    text-align: left;
    padding: 8px 12px;
    border-radius: 0;
    background: transparent;
    border: none;
    border-bottom: 1px solid var(--color-border);
    transition: background 0.1s;
  }

  .recent-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 10px;
  }

  .recent-badges {
    display: flex;
    align-items: center;
    gap: 6px;
    flex-shrink: 0;
  }

  .badge {
    font-size: 10px;
    padding: 1px 5px;
    border-radius: 2px;
    border: 1px solid var(--color-border);
    background: transparent;
    color: var(--color-text-muted);
    text-transform: uppercase;
    letter-spacing: 0.04em;
    font-weight: 500;
  }

  .recent-item:hover {
    background: var(--color-bg-hover);
  }

  .recent-name {
    font-weight: 600;
  }

  .recent-sub {
    margin-top: 3px;
    color: var(--color-text-secondary);
    font-size: 12px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .recent-meta {
    margin-top: 4px;
    color: var(--color-text-muted);
    font-size: 11px;
  }

  .right {
    background: var(--color-bg-app);
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .actions {
    padding: 16px;
    display: flex;
    flex-direction: column;
  }

  .actions-label {
    font-size: 10px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.08em;
    color: var(--color-text-muted);
    padding: 0 2px 10px;
  }

  .action-row {
    display: flex;
    align-items: center;
    gap: 12px;
    width: 100%;
    text-align: left;
    padding: 10px 4px;
    border-top: 1px solid var(--color-border);
    transition: background 0.1s;
  }

  .action-row:last-child {
    border-bottom: 1px solid var(--color-border);
  }

  .action-row:hover:not([disabled]) {
    background: var(--color-bg-hover);
  }

  .action-row[disabled] {
    opacity: 0.4;
    cursor: not-allowed;
  }

  .action-icon {
    width: 20px;
    height: 20px;
    flex-shrink: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--color-accent-text);
  }

  .action-icon.muted {
    color: var(--color-text-muted);
  }

  .action-icon svg {
    width: 16px;
    height: 16px;
  }

  .action-text {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .action-title {
    font-size: 13px;
    color: var(--color-text-primary);
    font-weight: 500;
  }

  .action-desc {
    font-size: 11px;
    color: var(--color-text-muted);
  }

  .action-soon {
    font-size: 10px;
    color: var(--color-text-muted);
    text-transform: uppercase;
    letter-spacing: 0.06em;
    font-weight: 500;
  }

  @media (max-width: 1000px) {
    .launcher {
      grid-template-columns: 1fr;
    }
  }
</style>
