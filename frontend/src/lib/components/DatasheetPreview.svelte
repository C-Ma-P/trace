<script lang="ts">
  import { onMount, tick } from 'svelte';
  import { getDocument, GlobalWorkerOptions } from 'pdfjs-dist';
  import type { PDFDocumentProxy } from 'pdfjs-dist/types/src/display/api';
  import { readAssetFile, type ComponentAsset } from '../backend';
  import { openAssetExternally, openDatasheetWindow } from '../windowService';

  GlobalWorkerOptions.workerSrc = new URL('pdfjs-dist/build/pdf.worker.min.mjs', import.meta.url).toString();

  let {
    asset = null,
    allowPopout = true,
    fullWindow = false,
  }: {
    asset?: ComponentAsset | null;
    allowPopout?: boolean;
    fullWindow?: boolean;
  } = $props();

  type PreviewState =
    | { kind: 'empty' }
    | { kind: 'loading'; message: string }
    | { kind: 'ready' }
    | { kind: 'error'; message: string }
    | { kind: 'unsupported'; message: string };

  let previewState: PreviewState = $state({ kind: 'empty' });
  let zoom = $state(1);
  let pdfDocument: PDFDocumentProxy | null = null;
  let pageNumbers: number[] = $state([]);
  let pageCanvases: Array<HTMLCanvasElement | null> = $state([]);
  let objectURL: string | null = null;
  let loadSequence = 0;
  let renderSequence = 0;

  const zoomMin = 0.5;
  const zoomMax = 3;
  const zoomStep = 0.25;

  function getDisplayName(assetData: ComponentAsset | null): string {
    if (!assetData) {
      return 'Datasheet';
    }
    if (assetData.label.trim()) {
      return assetData.label.trim();
    }
    const value = assetData.urlOrPath.trim();
    if (!value) {
      return 'Datasheet';
    }
    const normalized = value.split('?')[0].split('#')[0];
    const pieces = normalized.split('/').filter(Boolean);
    return pieces.at(-1) || 'Datasheet';
  }

  function describeSource(assetData: ComponentAsset): string {
    return assetData.source ? assetData.source : 'attached asset';
  }

  function base64ToUint8Array(value: string): Uint8Array {
    const binary = atob(value);
    const bytes = new Uint8Array(binary.length);
    for (let index = 0; index < binary.length; index += 1) {
      bytes[index] = binary.charCodeAt(index);
    }
    return bytes;
  }

  function cleanupObjectURL(): void {
    if (objectURL) {
      URL.revokeObjectURL(objectURL);
      objectURL = null;
    }
  }

  function resetDocument(): void {
    renderSequence += 1;
    pageNumbers = [];
    pageCanvases = [];
    if (pdfDocument) {
      void pdfDocument.destroy();
      pdfDocument = null;
    }
    cleanupObjectURL();
  }

  function isRemoteURL(value: string): boolean {
    try {
      const parsed = new URL(value);
      return parsed.protocol === 'http:' || parsed.protocol === 'https:';
    } catch {
      return false;
    }
  }

  function inferExtension(assetData: ComponentAsset): string {
    const candidates = [assetData.label, assetData.urlOrPath];
    for (const candidate of candidates) {
      const trimmed = candidate.trim();
      if (!trimmed) {
        continue;
      }
      const normalized = trimmed.split('?')[0].split('#')[0];
      const dot = normalized.lastIndexOf('.');
      if (dot >= 0) {
        return normalized.substring(dot).toLowerCase();
      }
    }
    return '';
  }

  function isPDF(assetData: ComponentAsset): boolean {
    return inferExtension(assetData) === '.pdf';
  }

  async function resolvePDFSource(assetData: ComponentAsset): Promise<string> {
    const target = assetData.urlOrPath.trim();
    if (isRemoteURL(target)) {
      return target;
    }

    const response = await readAssetFile(assetData.id);
    const bytes = base64ToUint8Array(response.data);
    const blobBytes = new Uint8Array(bytes.length);
    blobBytes.set(bytes);
    const blob = new Blob([
      blobBytes,
    ], { type: 'application/pdf' });
    objectURL = URL.createObjectURL(blob);
    return objectURL;
  }

  async function loadDocument(assetData: ComponentAsset): Promise<void> {
    const sequence = ++loadSequence;
    resetDocument();
    zoom = 1;

    if (!isPDF(assetData)) {
      previewState = { kind: 'unsupported', message: 'Only PDF datasheets can be previewed inline.' };
      return;
    }

    previewState = { kind: 'loading', message: 'Loading datasheet…' };

    try {
      const source = await resolvePDFSource(assetData);
      if (sequence !== loadSequence) {
        return;
      }

      previewState = { kind: 'loading', message: 'Rendering pages…' };
      const task = getDocument({ url: source, withCredentials: false });
      const document = await task.promise;
      if (sequence !== loadSequence) {
        void document.destroy();
        return;
      }

      pdfDocument = document;
      pageNumbers = Array.from({ length: document.numPages }, (_, index) => index + 1);
      previewState = { kind: 'ready' };
    } catch (error) {
      console.error('datasheet preview error', error);
      const message = error instanceof Error ? error.message : String(error);
      previewState = {
        kind: 'error',
        message: isRemoteURL(assetData.urlOrPath)
          ? `Inline preview failed. The supplier PDF may block embedding or cross-origin access. ${message}`
          : `Failed to load the managed PDF asset. ${message}`,
      };
    }
  }

  async function renderPages(document: PDFDocumentProxy, scale: number): Promise<void> {
    const sequence = ++renderSequence;
    await tick();

    for (let index = 0; index < pageNumbers.length; index += 1) {
      if (sequence !== renderSequence) {
        return;
      }

      const canvas = pageCanvases[index];
      if (!canvas) {
        continue;
      }

      const page = await document.getPage(pageNumbers[index]);
      if (sequence !== renderSequence) {
        return;
      }

      const viewport = page.getViewport({ scale });
      const outputScale = window.devicePixelRatio || 1;
      const context = canvas.getContext('2d');
      if (!context) {
        continue;
      }

      canvas.width = Math.ceil(viewport.width * outputScale);
      canvas.height = Math.ceil(viewport.height * outputScale);
      canvas.style.width = `${viewport.width}px`;
      canvas.style.height = `${viewport.height}px`;
      context.setTransform(outputScale, 0, 0, outputScale, 0, 0);
      context.clearRect(0, 0, viewport.width, viewport.height);

      await page.render({ canvasContext: context, viewport }).promise;
    }
  }

  function adjustZoom(direction: -1 | 1): void {
    zoom = Math.min(zoomMax, Math.max(zoomMin, Number((zoom + direction * zoomStep).toFixed(2))));
  }

  function resetZoom(): void {
    zoom = 1;
  }

  async function handlePopout(): Promise<void> {
    if (!asset) {
      return;
    }
    await openDatasheetWindow(asset.id);
  }

  async function handleOpenExternal(): Promise<void> {
    if (!asset) {
      return;
    }
    await openAssetExternally(asset.id);
  }

  onMount(() => resetDocument);

  $effect(() => {
    if (!asset) {
      resetDocument();
      previewState = { kind: 'empty' };
      return;
    }
    void loadDocument(asset);
  });

  $effect(() => {
    const document = pdfDocument;
    const isReady = previewState.kind === 'ready';
    const scale = zoom;
    if (!document || !isReady) {
      return;
    }
    void renderPages(document, scale);
  });
</script>

<div class:full-window={fullWindow} class="datasheet-preview">
  {#if asset}
    <div class="datasheet-toolbar">
      <div class="datasheet-summary">
        <div class="datasheet-title">{getDisplayName(asset)}</div>
        <div class="datasheet-subtitle">{describeSource(asset)}</div>
      </div>
      <div class="datasheet-actions">
        <div class="zoom-group">
          <button class="toolbar-btn" onclick={() => adjustZoom(-1)} disabled={previewState.kind !== 'ready' || zoom <= zoomMin} title="Zoom out">
            −
          </button>
          <button class="toolbar-btn zoom-readout" onclick={resetZoom} disabled={previewState.kind !== 'ready'} title="Reset zoom">
            {Math.round(zoom * 100)}%
          </button>
          <button class="toolbar-btn" onclick={() => adjustZoom(1)} disabled={previewState.kind !== 'ready' || zoom >= zoomMax} title="Zoom in">
            +
          </button>
        </div>
        <button class="toolbar-btn action-btn" onclick={handlePopout} disabled={!allowPopout || !asset} title={allowPopout ? 'Open in a separate window' : 'Already in a separate window'}>
          {allowPopout ? 'Pop out' : 'Popped out'}
        </button>
        <button class="toolbar-btn action-btn" onclick={handleOpenExternal} disabled={!asset}>
          Open external
        </button>
      </div>
    </div>
  {/if}

  {#if previewState.kind === 'empty'}
    <div class="datasheet-state">
      <div class="state-icon">📄</div>
      <div class="state-title">No Datasheet Selected</div>
      <div class="state-message">Attach and select a datasheet asset below.</div>
    </div>
  {:else if previewState.kind === 'loading'}
    <div class="datasheet-state">
      <div class="spinner"></div>
      <div class="state-title">Loading Datasheet</div>
      <div class="state-message">{previewState.message}</div>
    </div>
  {:else if previewState.kind === 'unsupported'}
    <div class="datasheet-state">
      <div class="state-icon">⚠</div>
      <div class="state-title">Unsupported File</div>
      <div class="state-message">{previewState.message}</div>
      {#if asset}
        <button class="fallback-btn" onclick={handleOpenExternal}>Open external</button>
      {/if}
    </div>
  {:else if previewState.kind === 'error'}
    <div class="datasheet-state">
      <div class="state-icon">⚠</div>
      <div class="state-title">Inline Preview Unavailable</div>
      <div class="state-message">{previewState.message}</div>
      {#if asset}
        <button class="fallback-btn" onclick={handleOpenExternal}>Open external</button>
      {/if}
    </div>
  {:else}
    <div class="datasheet-scroll">
      <div class="datasheet-pages">
        {#each pageNumbers as pageNumber, index}
          <section class="page-frame">
            <div class="page-label">Page {pageNumber}</div>
            <canvas bind:this={pageCanvases[index]} class="page-canvas"></canvas>
          </section>
        {/each}
      </div>
    </div>
  {/if}
</div>

<style>
  .datasheet-preview {
    display: flex;
    flex-direction: column;
    width: 100%;
    height: 100%;
    min-height: 0;
    background: var(--color-bg-muted);
  }
  .datasheet-preview.full-window {
    background: var(--color-bg-app);
  }
  .datasheet-toolbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    padding: 12px 14px;
    border-bottom: 1px solid var(--color-border);
    background: var(--color-bg-surface);
  }
  .datasheet-summary {
    min-width: 0;
  }
  .datasheet-title {
    font-size: 13px;
    font-weight: 600;
    color: var(--color-text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .datasheet-subtitle {
    font-size: 11px;
    color: var(--color-text-secondary);
    text-transform: capitalize;
  }
  .datasheet-actions {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-wrap: wrap;
    justify-content: flex-end;
  }
  .zoom-group {
    display: inline-flex;
    align-items: center;
    gap: 1px;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    overflow: hidden;
  }
  .toolbar-btn,
  .fallback-btn {
    border: 1px solid var(--color-border);
    background: var(--color-bg-surface);
    color: var(--color-text-primary);
    font: inherit;
    cursor: pointer;
    transition: background 0.12s ease, opacity 0.12s ease;
  }
  .toolbar-btn:hover,
  .fallback-btn:hover {
    background: var(--color-bg-hover);
  }
  .toolbar-btn:disabled {
    cursor: default;
    opacity: 0.55;
  }
  .zoom-group .toolbar-btn {
    min-width: 34px;
    min-height: 32px;
    border: 0;
    border-radius: 0;
  }
  .zoom-readout {
    min-width: 62px !important;
    font-size: 12px;
  }
  .action-btn,
  .fallback-btn {
    padding: 8px 12px;
    border-radius: var(--radius-md);
    font-size: 12px;
  }
  .datasheet-scroll {
    flex: 1;
    min-height: 0;
    overflow: auto;
    padding: 18px;
  }
  .datasheet-pages {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 18px;
  }
  .page-frame {
    display: flex;
    flex-direction: column;
    gap: 8px;
    align-items: center;
    width: 100%;
  }
  .page-label {
    font-size: 11px;
    color: var(--color-text-muted);
  }
  .page-canvas {
    max-width: 100%;
    background: white;
    box-shadow: 0 10px 24px rgb(0 0 0 / 0.18);
    border-radius: 4px;
  }
  .datasheet-state {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 10px;
    padding: 24px;
    text-align: center;
    color: var(--color-text-secondary);
  }
  .state-icon {
    font-size: 30px;
  }
  .state-title {
    font-size: 14px;
    font-weight: 600;
    color: var(--color-text-primary);
  }
  .state-message {
    max-width: 520px;
    font-size: 12px;
    line-height: 1.5;
  }
  .spinner {
    width: 24px;
    height: 24px;
    border: 2px solid var(--color-border);
    border-top-color: var(--color-accent);
    border-radius: 999px;
    animation: spin 0.8s linear infinite;
  }
  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
  }
  @media (max-width: 720px) {
    .datasheet-toolbar {
      align-items: flex-start;
      flex-direction: column;
    }
    .datasheet-actions {
      width: 100%;
      justify-content: flex-start;
    }
    .datasheet-scroll {
      padding: 12px;
    }
  }
</style>