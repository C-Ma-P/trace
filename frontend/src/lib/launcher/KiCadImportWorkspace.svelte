<script lang="ts">
  import { onMount } from 'svelte';
  import RequirementEditor from '../projects/RequirementEditor.svelte';
  import {
    categoryDisplayName,
    getCategories,
    getKiCadPreferences,
    importKiCadProject,
    listKiCadProjects,
    listProjects,
    previewKiCadImport,
    type CategoryInfo,
    type KiCadImportPreview,
    type KiCadImportPreviewRow,
    type KiCadProjectCandidate,
    type Project,
  } from '../backend';
  import { pickDirectory } from '../windowService';

  let {
    active = false,
    onBack,
    onImportedProject,
  }: {
    active?: boolean;
    onBack?: () => void;
    onImportedProject?: (projectId: string) => void;
  } = $props();

  let categories: CategoryInfo[] = $state([]);
  let existingProjects: Project[] = $state([]);

  let defaultRoots: string[] = $state([]);
  let inlineRoots: string[] = $state([]);
  let query = $state('');
  let rootsReady = $state(false);
  let pickingRoot = $state(false);
  let scanError = $state('');
  let scanning = $state(false);
  let candidates: KiCadProjectCandidate[] = $state([]);
  let selectedProjectPath = $state('');
  let activeScanToken = 0;

  let preview: KiCadImportPreview | null = $state(null);
  let previewCache: Record<string, KiCadImportPreview> = $state({});
  let previewErrorCache: Record<string, any> = $state({});
  let previewBusy = $state(false);
  let previewError = $state('');
  let previewErrorData: any = $state(null);
  let selectedRowIndex: number | null = $state(null);
  let previewLoadToken = 0;

  let targetMode: 'new' | 'existing' = $state('new');
  let newProjectName = $state('');
  let newProjectDescription = $state('');
  let existingProjectId = $state('');
  let projectNameManuallyEdited = $state(false);

  let importBusy = $state(false);
  let importError = $state('');

  const scanRoots = $derived([...defaultRoots, ...inlineRoots]);

  async function loadKiCadRoots() {
    try {
      const prefs = await getKiCadPreferences();
      defaultRoots = prefs.projectRoots ?? [];
    } catch (err: any) {
      scanError = err?.message ?? String(err);
    } finally {
      rootsReady = true;
    }
  }

  onMount(async () => {
    categories = (await getCategories()) ?? [];
    existingProjects = (await listProjects()) ?? [];
    await loadKiCadRoots();
  });

  onMount(() => {
    const onFocus = () => {
      void loadKiCadRoots();
    };
    window.addEventListener('focus', onFocus);
    return () => window.removeEventListener('focus', onFocus);
  });

  $effect(() => {
    if (!rootsReady) {
      return;
    }

    void scanRoots;
    void query;

    const handle = window.setTimeout(() => {
      void handleScan();
    }, 180);

    return () => window.clearTimeout(handle);
  });

  $effect(() => {
    if (!active) {
      projectNameManuallyEdited = false;
      return;
    }

    if (projectNameManuallyEdited) {
      return;
    }

    newProjectName = suggestedProjectName();
  });

  function selectedRow(): KiCadImportPreviewRow | null {
    if (!preview || selectedRowIndex === null) {
      return null;
    }
    return preview.rows[selectedRowIndex] ?? null;
  }

  function summary() {
    if (!preview) {
      return { totalRows: 0, includedRows: 0, warningRows: 0 };
    }
    return preview.rows.reduce(
      (acc, row) => {
        acc.totalRows += 1;
        if (row.included) acc.includedRows += 1;
        if (row.hasWarning) acc.warningRows += 1;
        return acc;
      },
      { totalRows: 0, includedRows: 0, warningRows: 0 }
    );
  }

  function resetPreview() {
    preview = null;
    previewBusy = false;
    previewError = '';
    importError = '';
    selectedRowIndex = null;
  }

  function parsePreviewError(err: any): any {
    const message = err?.message ?? String(err);
    if (typeof message !== 'string') {
      return message;
    }

    try {
      return JSON.parse(message);
    } catch {
      return message;
    }
  }

  function previewErrorDataToText(value: any, indent = 0): string {
    const spacer = '  '.repeat(indent);
    if (value === null) {
      return `${spacer}null`;
    }
    if (Array.isArray(value)) {
      return value
        .map((item, index) => `${spacer}[${index}]: ${typeof item === 'object' ? '\n' + previewErrorDataToText(item, indent + 1) : String(item)}`)
        .join('\n');
    }
    if (typeof value === 'object') {
      return Object.entries(value)
        .map(([key, nested]) => `${spacer}${key}: ${typeof nested === 'object' ? '\n' + previewErrorDataToText(nested, indent + 1) : String(nested)}`)
        .join('\n');
    }
    return `${spacer}${String(value)}`;
  }

  function setPreviewForSelected() {
    if (!selectedProjectPath) {
      preview = null;
      previewError = '';
      selectedRowIndex = null;
      return;
    }

    const cached = previewCache[selectedProjectPath];
    if (cached) {
      preview = cached;
      previewErrorData = previewErrorCache[selectedProjectPath] ?? null;
      previewError = previewErrorData ? 'Preview load failed' : '';
      selectedRowIndex = cached.rows.length > 0 ? 0 : null;
      return;
    }

    preview = null;
    previewErrorData = previewErrorCache[selectedProjectPath] ?? null;
    previewError = previewErrorData ? 'Preview load failed' : '';
    selectedRowIndex = null;
  }

  async function refreshPreview(projectPath: string) {
    const loadToken = ++previewLoadToken;
    previewBusy = true;
    previewError = '';
    importError = '';

    try {
      const loaded = await previewKiCadImport(projectPath);
      if (loadToken !== previewLoadToken) {
        return;
      }
      previewCache[projectPath] = loaded;
      delete previewErrorCache[projectPath];
      if (selectedProjectPath === projectPath) {
        preview = loaded;
        previewError = '';
        selectedRowIndex = loaded.rows.length > 0 ? 0 : null;
      }
    } catch (err: any) {
      const parsed = parsePreviewError(err);
      previewErrorCache[projectPath] = parsed;
      if (selectedProjectPath === projectPath) {
        previewErrorData = parsed;
        previewError = 'Preview load failed';
        preview = null;
        selectedRowIndex = null;
      }
    } finally {
      if (loadToken === previewLoadToken) {
        previewBusy = false;
      }
    }
  }

  async function prefetchPreview(projectPath: string) {
    if (!projectPath || previewCache[projectPath] || previewErrorCache[projectPath]) {
      return;
    }

    try {
      const loaded = await previewKiCadImport(projectPath);
      previewCache[projectPath] = loaded;
      if (projectPath === selectedProjectPath && !preview) {
        preview = loaded;
        previewError = '';
        selectedRowIndex = loaded.rows.length > 0 ? 0 : null;
      }
    } catch (err: any) {
      const parsed = parsePreviewError(err);
      previewErrorCache[projectPath] = parsed;
      if (projectPath === selectedProjectPath) {
        previewErrorData = parsed;
        previewError = 'Preview load failed';
        preview = null;
        selectedRowIndex = null;
      }
    }
  }

  function selectedCandidate(): KiCadProjectCandidate | null {
    return candidates.find((candidate) => candidate.projectPath === selectedProjectPath) ?? null;
  }

  function suggestedProjectName(): string {
    if (preview && preview.selectedProject.projectPath === selectedProjectPath) {
      return preview.selectedProject.name;
    }
    return selectedCandidate()?.name ?? '';
  }

  function handleNewProjectNameInput(event: Event) {
    const value = (event.currentTarget as HTMLInputElement).value;
    newProjectName = value;
    projectNameManuallyEdited = value !== suggestedProjectName();
  }

  async function handleScan() {
    const nextRoots = [...scanRoots];
    const nextQuery = query;
    const scanToken = ++activeScanToken;

    if (nextRoots.length === 0) {
      scanError = 'No KiCad scan roots configured. Add a folder here or set defaults in Preferences > Global > Integrations > KiCad.';
      candidates = [];
      selectedProjectPath = '';
      resetPreview();
      return;
    }

    scanning = true;
    scanError = '';
    try {
      const nextCandidates = await listKiCadProjects(nextRoots, nextQuery);
      if (scanToken !== activeScanToken) {
        return;
      }
      candidates = nextCandidates;
      if (!nextCandidates.find((candidate) => candidate.projectPath === selectedProjectPath)) {
        selectedProjectPath = nextCandidates[0]?.projectPath ?? '';
        resetPreview();
      }
      setPreviewForSelected();
      nextCandidates.forEach((candidate) => {
        void prefetchPreview(candidate.projectPath);
      });
    } catch (err: any) {
      if (scanToken !== activeScanToken) {
        return;
      }
      scanError = err?.message ?? String(err);
      candidates = [];
      selectedProjectPath = '';
      resetPreview();
    } finally {
      if (scanToken === activeScanToken) {
        scanning = false;
      }
    }
  }

  async function handleAddRoot() {
    const startDir = scanRoots.at(-1) ?? '';
    pickingRoot = true;
    try {
      const selectedRoot = (await pickDirectory(startDir)).trim();
      if (!selectedRoot || scanRoots.includes(selectedRoot)) {
        return;
      }
      inlineRoots = [...inlineRoots, selectedRoot];
    } catch (err: any) {
      scanError = err?.message ?? String(err);
    } finally {
      pickingRoot = false;
    }
  }

  function addRootFromInput(trimmed: string) {
    if (!trimmed) {
      return;
    }
    if (!scanRoots.includes(trimmed)) {
      inlineRoots = [...inlineRoots, trimmed];
    }
  }

  function removeInlineRoot(root: string) {
    inlineRoots = inlineRoots.filter((value) => value !== root);
    if (scanRoots.length === 0) {
      candidates = [];
      selectedProjectPath = '';
      resetPreview();
    }
  }

  function selectCandidate(candidate: KiCadProjectCandidate) {
    if (candidate.projectPath === selectedProjectPath) {
      return;
    }
    selectedProjectPath = candidate.projectPath;
    setPreviewForSelected();
  }

  async function handlePreview() {
    if (!selectedProjectPath) {
      return;
    }
    await refreshPreview(selectedProjectPath);
  }

  function toggleIncluded(index: number, included: boolean) {
    if (!preview) {
      return;
    }
    preview.rows[index].included = included;
  }

  function removeRow(index: number) {
    toggleIncluded(index, false);
  }

  function canImport(): boolean {
    if (!preview || preview.rows.every((row) => !row.included)) {
      return false;
    }
    if (targetMode === 'new') {
      return newProjectName.trim().length > 0;
    }
    return existingProjectId.trim().length > 0;
  }

  async function handleImport() {
    if (!preview) {
      return;
    }
    importBusy = true;
    importError = '';
    try {
      const project = await importKiCadProject({
        targetMode,
        newProjectName,
        newProjectDescription,
        existingProjectId,
        sourceProjectPath: preview.selectedProject.projectPath,
        rows: preview.rows,
      });
      onBack?.();
      onImportedProject?.(project.id);
    } catch (err: any) {
      importError = err?.message ?? String(err);
    } finally {
      importBusy = false;
    }
  }
</script>

<div class="import-workspace">
  <header class="workspace-header">
    <div>
      <div class="eyebrow">Launcher / KiCad</div>
      <h1 class="workspace-title">Import from KiCad</h1>
      <p class="workspace-subtitle">Discover `.kicad_pro` projects, review every BOM row, and import into a new or existing Trace project.</p>
    </div>
    <button class="btn btn-secondary" onclick={() => onBack?.()}>
      Back
    </button>
  </header>

  <div class="workspace-grid">
    <section class="panel discovery-panel">
      <div class="panel-header">
        <div>
          <h2>Project Discovery</h2>
          <p>Defaults come from Preferences, and any changes here rescan automatically.</p>
        </div>
        <button class="btn btn-secondary btn-sm" onclick={() => void handleAddRoot()} disabled={pickingRoot}>
          {pickingRoot ? 'Opening…' : 'Add Folder'}
        </button>
      </div>

      <div class="discovery-content">
        <section class="discovery-section">
          <div class="section-copy">
            <div class="section-title">Scan Roots</div>
            <p>Folders checked for `.kicad_pro` projects in this import session.</p>
          </div>

          <div class="root-list">
            {#if scanRoots.length === 0}
              <div class="empty-block compact">No scan roots added for this session.</div>
            {:else}
              {#each scanRoots as root}
                <div class="root-chip" class:root-chip-default={defaultRoots.includes(root)}>
                  <span>{root}</span>
                  {#if defaultRoots.includes(root)}
                    <span class="root-chip-badge">Default</span>
                  {:else}
                    <button class="chip-remove" onclick={() => removeInlineRoot(root)} aria-label={`Remove ${root}`}>
                      ×
                    </button>
                  {/if}
                </div>
              {/each}
            {/if}
          </div>
        </section>

        <section class="discovery-section discovery-projects">
          <div class="projects-toolbar">
            <div class="section-copy">
              <div class="section-title">Discovered Projects</div>
              <p>Filter the current scan results and pick one project to preview. The selected project's preview loads automatically.</p>
            </div>

            <div class="filter-box">
              <input
                class="form-input"
                bind:value={query}
                placeholder="Filter discovered projects"
              />
              <div class="toolbar-status" aria-live="polite">
                {#if scanning}
                  <span class="scan-status">Scanning…</span>
                {:else if candidates.length > 0}
                  <span>{candidates.length} found</span>
                {/if}
              </div>
            </div>
          </div>

          {#if scanError}
            <div class="notice-card notice-card-error">{scanError}</div>
          {/if}

          <div class="candidate-list">
            {#if candidates.length === 0}
              <div class="empty-block roomy">No KiCad projects discovered.</div>
            {:else}
              {#each candidates as candidate}
                <button
                  class="candidate-item"
                  class:selected={candidate.projectPath === selectedProjectPath}
                  class:warning={!!previewErrorCache[candidate.projectPath]}
                  class:loading={candidate.projectPath === selectedProjectPath && previewBusy}
                  onclick={() => selectCandidate(candidate)}
                >
                  <div class="candidate-name">
                    {candidate.name}
                    {#if previewErrorCache[candidate.projectPath]}
                      <span class="badge badge-warning candidate-warning" aria-label="Preview failed">Warning</span>
                    {/if}
                  </div>
                  <div class="candidate-path">{candidate.projectPath}</div>
                </button>
              {/each}
            {/if}
          </div>
        </section>
      </div>
    </section>

    <section class="panel preview-panel">
      <div class="panel-header preview-header">
        <div>
          <h2>Preview</h2>
          {#if selectedProjectPath}
            <p>{selectedProjectPath}</p>
          {:else}
            <p>Select a KiCad project to preview.</p>
          {/if}
        </div>
        <button class="btn btn-primary btn-sm" onclick={() => void handlePreview()} disabled={!selectedProjectPath || previewBusy}>
          {previewBusy ? 'Refreshing…' : 'Refresh Preview'}
        </button>
      </div>

      {#if previewError}
        <div class="notice-card notice-card-error error-block">
          <div class="error-block-title">Preview load failed</div>
          <pre>{previewErrorData ? previewErrorDataToText(previewErrorData) : previewError}</pre>
        </div>
      {/if}

      {#if !preview}
        <div class="empty-block tall">Run a preview to inspect import rows and edit the generated requirements.</div>
      {:else}
        <div class="summary-strip">
          <span class="badge">{summary().totalRows} rows</span>
          <span class="badge">{summary().includedRows} included</span>
          <span class="badge badge-warning">{summary().warningRows} warnings</span>
        </div>

        <div class="preview-layout">
          <div class="row-list">
            {#each preview.rows as row, index}
              <button class="row-card" class:selected={selectedRowIndex === index} onclick={() => (selectedRowIndex = index)}>
                <div class="row-topline">
                  <div class="include-toggle">
                    <input
                      type="checkbox"
                      aria-label={`Include ${row.sourceRefs || row.rowId}`}
                      checked={row.included}
                      onclick={(event) => event.stopPropagation()}
                      onchange={(event) => toggleIncluded(index, event.currentTarget.checked)}
                    />
                    <span>{row.sourceRefs || row.rowId}</span>
                  </div>
                  {#if row.hasWarning}
                    <span class="badge badge-warning">Warning</span>
                  {/if}
                </div>
                <div class="row-name">{row.requirement.name || 'Unnamed requirement'}</div>
                <div class="row-meta">
                  <span>{categoryDisplayName(categories, row.requirement.category)}</span>
                  <span>Qty {row.requirement.quantity}</span>
                </div>
                <div class="row-source">{row.rawValue || 'No value'} · {row.rawFootprint || 'No footprint'}</div>
              </button>
            {/each}
          </div>

          <div class="row-editor">
            {#if selectedRow()}
              <div class="row-editor-scroll">
                <div class="row-editor-header">
                  <div>
                    <h3>{selectedRow()?.sourceRefs || selectedRow()?.rowId}</h3>
                    <p>{selectedRow()?.rawDescription || 'No description provided'}</p>
                  </div>
                  <button class="btn btn-secondary btn-sm" onclick={() => removeRow(selectedRowIndex ?? 0)}>
                    Exclude Row
                  </button>
                </div>

                {#if selectedRow()?.hasWarning}
                  <div class="warning-box">
                    {#each selectedRow()?.warningMessages ?? [] as warning}
                      <div>{warning}</div>
                    {/each}
                  </div>
                {/if}

                <div class="source-grid">
                  <div>
                    <span class="field-label">Refs</span>
                    <span>{selectedRow()?.sourceRefs || '—'}</span>
                  </div>
                  <div>
                    <span class="field-label">Value</span>
                    <span>{selectedRow()?.rawValue || '—'}</span>
                  </div>
                  <div>
                    <span class="field-label">Footprint</span>
                    <span>{selectedRow()?.rawFootprint || '—'}</span>
                  </div>
                  <div>
                    <span class="field-label">Manufacturer</span>
                    <span>{selectedRow()?.manufacturer || '—'}</span>
                  </div>
                  <div>
                    <span class="field-label">MPN</span>
                    <span>{selectedRow()?.mpn || '—'}</span>
                  </div>
                  <div>
                    <span class="field-label">Source Qty</span>
                    <span>{selectedRow()?.sourceQuantity ?? 0}</span>
                  </div>
                </div>

                {#if Object.keys(selectedRow()?.otherFields ?? {}).length > 0}
                  <div class="other-fields">
                    <div class="field-label">Other Fields</div>
                    {#each Object.entries(selectedRow()?.otherFields ?? {}) as [key, value]}
                      <div class="other-field-row">
                        <span>{key}</span>
                        <span>{value}</span>
                      </div>
                    {/each}
                  </div>
                {/if}

                <div class="editor-block">
                  <div class="editor-label">Editable requirement</div>
                  <RequirementEditor bind:requirement={preview.rows[selectedRowIndex ?? 0].requirement} {categories} />
                </div>
              </div>
            {:else}
              <div class="empty-block tall">Select a preview row to edit its mapped requirement.</div>
            {/if}
          </div>
        </div>
      {/if}
    </section>

    <section class="panel target-panel">
      <div class="panel-header">
        <div>
          <h2>Import Target</h2>
          <p>Create a new project or append requirements onto an existing one.</p>
        </div>
      </div>

      <div class="target-switch">
        <button class="target-pill" class:active={targetMode === 'new'} onclick={() => (targetMode = 'new')}>
          New Project
        </button>
        <button class="target-pill" class:active={targetMode === 'existing'} onclick={() => (targetMode = 'existing')}>
          Existing Project
        </button>
      </div>

      {#if targetMode === 'new'}
        <div class="target-copy">A new project will be created and seeded with the included requirements.</div>
        <div class="target-fields">
          <div class="form-group">
            <label for="kicad-import-name">Name</label>
            <input id="kicad-import-name" class="form-input" value={newProjectName} oninput={handleNewProjectNameInput} placeholder="Imported project name" />
          </div>
          <div class="form-group">
            <label for="kicad-import-description">Description</label>
            <textarea id="kicad-import-description" class="form-input" bind:value={newProjectDescription} rows="3" placeholder="Optional project description"></textarea>
          </div>
        </div>
      {:else}
        <div class="target-copy">Included requirements will be appended to the selected project. Existing requirements stay in place.</div>
        <div class="form-group">
          <label for="kicad-import-project">Project</label>
          <select id="kicad-import-project" class="form-input" bind:value={existingProjectId}>
            <option value="">Select a project</option>
            {#each existingProjects as project}
              <option value={project.id}>{project.name}</option>
            {/each}
          </select>
        </div>
      {/if}

      {#if importError}
        <div class="error-text">{importError}</div>
      {/if}

      <div class="target-actions">
        <button class="btn btn-secondary" onclick={() => onBack?.()} disabled={importBusy}>Cancel</button>
        <button class="btn btn-primary" onclick={() => void handleImport()} disabled={!canImport() || importBusy}>
          {importBusy ? 'Importing…' : targetMode === 'new' ? 'Create Project from Import' : 'Append Requirements'}
        </button>
      </div>
    </section>
  </div>
</div>

<style>
  .import-workspace {
    height: 100%;
    display: flex;
    flex-direction: column;
    background: linear-gradient(180deg, rgba(59, 130, 246, 0.06), transparent 180px), var(--color-bg-app);
  }

  .workspace-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 16px;
    padding: 24px;
    border-bottom: 1px solid var(--color-border);
  }

  .eyebrow {
    font-size: 10px;
    text-transform: uppercase;
    letter-spacing: 0.12em;
    color: var(--color-text-muted);
    margin-bottom: 8px;
  }

  .workspace-title {
    font-size: 22px;
    font-weight: 600;
    margin-bottom: 6px;
  }

  .workspace-subtitle {
    max-width: 760px;
    color: var(--color-text-secondary);
    line-height: 1.5;
  }

  .workspace-grid {
    flex: 1;
    min-height: 0;
    display: grid;
    grid-template-columns: 320px minmax(0, 1fr) 320px;
    gap: 1px;
    background: var(--color-border);
  }

  .panel {
    min-height: 0;
    display: flex;
    flex-direction: column;
    background: var(--color-bg-surface);
  }

  .panel-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 12px;
    padding: 16px;
    border-bottom: 1px solid var(--color-border);
  }

  .panel-header h2 {
    font-size: 13px;
    font-weight: 600;
    margin-bottom: 3px;
  }

  .panel-header p,
  .target-copy {
    color: var(--color-text-secondary);
    line-height: 1.5;
  }

  .preview-header p {
    word-break: break-word;
  }

  .discovery-content {
    display: flex;
    flex: 1;
    min-height: 0;
    flex-direction: column;
  }

  .discovery-section {
    display: flex;
    flex-direction: column;
    gap: 12px;
    padding: 16px;
    border-bottom: 1px solid var(--color-border);
  }

  .discovery-projects {
    flex: 1;
    min-height: 0;
    border-bottom: none;
  }

  .section-copy {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .section-title {
    font-size: 11px;
    font-weight: 700;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    color: var(--color-text-muted);
  }

  .section-copy p {
    color: var(--color-text-secondary);
    line-height: 1.45;
    font-size: 12px;
  }

  .root-list {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
  }

  .root-chip {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    padding: 6px 10px;
    border-radius: 999px;
    border: 1px solid var(--color-border);
    background: var(--color-bg-muted);
    max-width: 100%;
  }

  .root-chip-default {
    background: var(--color-accent-soft);
    border-color: var(--color-accent);
  }

  .root-chip span {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .root-chip-badge {
    flex: 0 0 auto;
    padding: 2px 7px;
    border-radius: 999px;
    background: var(--color-bg-surface);
    color: var(--color-text-secondary);
    font-size: 10px;
    font-weight: 700;
    letter-spacing: 0.06em;
    text-transform: uppercase;
  }

  .chip-remove {
    color: var(--color-text-muted);
  }

  .candidate-list {
    flex: 1;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: 8px;
    min-height: 0;
    padding-top: 1px;
    padding-right: 16px;
    scrollbar-gutter: stable both-edges;
  }

  .candidate-item,
  .row-card {
    width: 100%;
    text-align: left;
    padding: 12px;
    border-radius: var(--radius-md);
    border: 1px solid var(--color-border);
    background: var(--color-bg-app);
    transition: border-color 0.16s ease, background 0.16s ease, transform 0.16s ease;
  }

  .candidate-item:hover,
  .row-card:hover {
    border-color: var(--color-border-strong);
    background: var(--color-bg-hover);
  }

  .candidate-item.selected,
  .row-card.selected {
    border-color: var(--color-accent);
    background: var(--color-accent-soft);
    transform: translateY(-1px);
  }

  .candidate-name,
  .row-name {
    font-weight: 600;
    margin-bottom: 4px;
  }

  .projects-toolbar {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .filter-box {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .toolbar-status {
    min-height: 18px;
    color: var(--color-text-muted);
    font-size: 12px;
  }

  .scan-status {
    color: var(--color-text-muted);
    font-size: 12px;
    white-space: nowrap;
  }

  .notice-card {
    padding: 12px 14px;
    border-radius: var(--radius-md);
    border: 1px solid var(--color-border);
    background: var(--color-bg-app);
    color: var(--color-text-secondary);
    line-height: 1.45;
  }

  .notice-card-error {
    border-color: var(--color-danger-border);
    background: var(--color-danger-soft);
    color: var(--color-danger-text);
  }

  .candidate-path,
  .row-source {
    color: var(--color-text-secondary);
    line-height: 1.4;
    word-break: break-word;
  }

  .candidate-item.warning {
    border-color: var(--color-danger-border);
    background: rgba(251, 191, 36, 0.06);
  }

  .candidate-warning {
    margin-left: 8px;
    font-size: 0.75em;
    display: inline-flex;
    align-items: center;
  }

  .candidate-item.loading {
    box-shadow: 0 0 0 1px rgba(59, 130, 246, 0.2), 0 0 0 4px rgba(59, 130, 246, 0.1);
    animation: glow-border 1.4s ease-in-out infinite;
  }

  @keyframes glow-border {
    0%, 100% { box-shadow: 0 0 0 1px rgba(59, 130, 246, 0.2), 0 0 0 4px rgba(59, 130, 246, 0.04); }
    50% { box-shadow: 0 0 0 1px rgba(59, 130, 246, 0.35), 0 0 0 8px rgba(59, 130, 246, 0.12); }
  }

  .error-block {
    padding: 16px;
    border-left: none;
    border-right: none;
    border-top: 1px solid var(--color-danger-border);
    border-bottom: 1px solid var(--color-danger-border);
    border-radius: 0;
    background: var(--color-danger-soft);
    color: var(--color-danger-text);
    margin-bottom: 16px;
  }

  .error-block-title {
    font-weight: 700;
    margin-bottom: 8px;
  }

  .error-block pre {
    white-space: pre-wrap;
    word-break: break-word;
    margin: 0;
    font-size: 13px;
    line-height: 1.5;
  }

  .summary-strip {
    display: flex;
    gap: 8px;
    padding: 12px 16px 0;
  }

  .empty-block.compact {
    width: 100%;
    padding: 12px 14px;
  }

  .empty-block.roomy {
    padding: 18px;
  }

  .preview-layout {
    flex: 1;
    min-height: 0;
    display: grid;
    grid-template-columns: 320px minmax(0, 1fr);
    border-top: 1px solid var(--color-border);
    margin-top: 12px;
  }

  .row-list {
    min-height: 0;
    overflow-y: auto;
    padding: 16px;
    border-right: 1px solid var(--color-border);
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .row-editor {
    min-height: 0;
    display: flex;
    flex-direction: column;
  }

  .row-editor-scroll {
    min-height: 0;
    overflow-y: auto;
    padding: 16px;
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .row-topline,
  .row-meta,
  .target-switch,
  .target-actions,
  .row-editor-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
  }

  .row-meta {
    justify-content: flex-start;
    color: var(--color-text-secondary);
  }

  .include-toggle {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    font-weight: 500;
    cursor: pointer;
  }

  .warning-box {
    padding: 12px;
    border-radius: var(--radius-md);
    border: 1px solid var(--color-warning-border);
    background: var(--color-warning-soft);
    color: var(--color-warning-text);
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .source-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 12px;
    padding: 14px;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    background: var(--color-bg-app);
  }

  .source-grid > div,
  .other-field-row {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .field-label,
  .editor-label {
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 0.08em;
    color: var(--color-text-muted);
  }

  .other-fields,
  .editor-block {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .other-fields {
    padding: 14px;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    background: var(--color-bg-app);
  }

  .other-field-row {
    padding-top: 8px;
    border-top: 1px solid var(--color-border);
  }

  .other-field-row:first-of-type {
    border-top: 0;
    padding-top: 0;
  }

  .target-panel {
    padding-bottom: 16px;
  }

  .target-switch {
    padding: 16px;
  }

  .target-pill {
    flex: 1;
    padding: 8px 10px;
    border-radius: var(--radius-md);
    border: 1px solid var(--color-border);
    background: var(--color-bg-app);
    color: var(--color-text-secondary);
  }

  .target-pill.active {
    border-color: var(--color-accent);
    background: var(--color-accent-soft);
    color: var(--color-text-primary);
  }

  .target-copy,
  .target-fields,
  .target-actions,
  .section-error {
    padding-left: 16px;
    padding-right: 16px;
  }

  .target-fields {
    padding-top: 12px;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .target-actions {
    margin-top: auto;
    padding-top: 16px;
  }

  .empty-block {
    padding: 16px;
    color: var(--color-text-muted);
  }

  .empty-block.tall {
    height: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
    text-align: center;
  }

  @media (max-width: 1320px) {
    .workspace-grid {
      grid-template-columns: 280px minmax(0, 1fr);
      grid-template-rows: minmax(0, 1fr) auto;
    }

    .target-panel {
      grid-column: 1 / -1;
      border-top: 1px solid var(--color-border);
    }
  }

  @media (max-width: 960px) {
    .workspace-header {
      padding: 18px;
    }

    .workspace-grid,
    .preview-layout {
      grid-template-columns: 1fr;
    }

    .row-list {
      border-right: 0;
      border-bottom: 1px solid var(--color-border);
      max-height: 280px;
    }
  }
</style>
