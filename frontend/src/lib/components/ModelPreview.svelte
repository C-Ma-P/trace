<script lang="ts">
  import { onMount } from 'svelte';
  import * as THREE from 'three';
  import { OrbitControls } from 'three/examples/jsm/controls/OrbitControls.js';
  import { readAssetFile } from '../backend';
  import type { ComponentAsset } from '../backend';
  import {
    loadStepFileToGroup,
    getGroupBounds,
    disposeGroup,
    base64ToUint8Array,
  } from './stepLoader';

  let { asset }: { asset: ComponentAsset } = $props();

  type PreviewState =
    | { kind: 'loading'; message: string }
    | { kind: 'ready' }
    | { kind: 'error'; message: string }
    | { kind: 'unsupported'; extension: string };

  let state = $state<PreviewState>({ kind: 'loading', message: 'Initializing…' });
  let canvasContainer: HTMLDivElement | undefined = $state();

  // Three.js state
  let renderer: THREE.WebGLRenderer | null = null;
  let scene: THREE.Scene | null = null;
  let camera: THREE.PerspectiveCamera | null = null;
  let controls: OrbitControls | null = null;
  let modelGroup: THREE.Group | null = null;
  let animationFrameId: number | null = null;
  let resizeObserver: ResizeObserver | null = null;

  // Cache to avoid reparsing the same asset
  let loadedAssetId: string | null = null;

  const SUPPORTED_EXTENSIONS = ['.step', '.stp'];

  function getExtension(pathOrLabel: string): string {
    const dot = pathOrLabel.lastIndexOf('.');
    return dot >= 0 ? pathOrLabel.substring(dot).toLowerCase() : '';
  }

  function isSupported(ext: string): boolean {
    return SUPPORTED_EXTENSIONS.includes(ext);
  }

  function setupScene(container: HTMLDivElement): void {
    const width = container.clientWidth;
    const height = container.clientHeight;

    // Renderer
    renderer = new THREE.WebGLRenderer({ antialias: true, alpha: false });
    renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));
    renderer.setSize(width, height);
    renderer.setClearColor(0x1e2130); // matches --color-bg-muted
    renderer.toneMapping = THREE.ACESFilmicToneMapping;
    renderer.toneMappingExposure = 1.0;
    container.appendChild(renderer.domElement);

    // Scene
    scene = new THREE.Scene();

    // Camera
    camera = new THREE.PerspectiveCamera(45, width / height, 0.01, 10000);
    camera.position.set(50, 40, 60);

    // Lights
    const ambient = new THREE.AmbientLight(0xffffff, 0.6);
    scene.add(ambient);

    const dirLight1 = new THREE.DirectionalLight(0xffffff, 0.8);
    dirLight1.position.set(50, 80, 60);
    scene.add(dirLight1);

    const dirLight2 = new THREE.DirectionalLight(0x8899cc, 0.4);
    dirLight2.position.set(-40, 20, -30);
    scene.add(dirLight2);

    // Controls
    controls = new OrbitControls(camera, renderer.domElement);
    controls.enableDamping = true;
    controls.dampingFactor = 0.08;
    controls.enablePan = true;
    controls.minDistance = 0.1;
    controls.maxDistance = 5000;

    // Animation loop
    function animate() {
      animationFrameId = requestAnimationFrame(animate);
      controls!.update();
      renderer!.render(scene!, camera!);
    }
    animate();

    // Resize observer
    resizeObserver = new ResizeObserver(() => {
      if (!renderer || !camera || !container) return;
      const w = container.clientWidth;
      const h = container.clientHeight;
      renderer.setSize(w, h);
      camera.aspect = w / h;
      camera.updateProjectionMatrix();
    });
    resizeObserver.observe(container);
  }

  function fitCameraToModel(group: THREE.Group): void {
    if (!camera || !controls) return;
    const { center, size } = getGroupBounds(group);
    const maxDim = Math.max(size.x, size.y, size.z);
    const fov = camera.fov * (Math.PI / 180);
    let cameraDistance = (maxDim / 2) / Math.tan(fov / 2);
    cameraDistance *= 1.8; // padding

    camera.position.set(
      center.x + cameraDistance * 0.5,
      center.y + cameraDistance * 0.4,
      center.z + cameraDistance * 0.7
    );
    camera.near = maxDim * 0.001;
    camera.far = maxDim * 100;
    camera.updateProjectionMatrix();

    controls.target.copy(center);
    controls.update();
  }

  function clearModel(): void {
    if (modelGroup && scene) {
      scene.remove(modelGroup);
      disposeGroup(modelGroup);
      modelGroup = null;
    }
  }

  function cleanup(): void {
    if (animationFrameId !== null) {
      cancelAnimationFrame(animationFrameId);
      animationFrameId = null;
    }
    if (resizeObserver) {
      resizeObserver.disconnect();
      resizeObserver = null;
    }
    clearModel();
    if (controls) {
      controls.dispose();
      controls = null;
    }
    if (renderer) {
      renderer.dispose();
      if (renderer.domElement.parentElement) {
        renderer.domElement.parentElement.removeChild(renderer.domElement);
      }
      renderer = null;
    }
    scene = null;
    camera = null;
    loadedAssetId = null;
  }

  async function loadAsset(assetData: ComponentAsset): Promise<void> {
    if (!scene || !camera || !controls) return;

    const ext = getExtension(assetData.urlOrPath || assetData.label);
    if (!isSupported(ext)) {
      state = { kind: 'unsupported', extension: ext || 'unknown' };
      return;
    }

    // Skip if already loaded
    if (loadedAssetId === assetData.id) return;

    state = { kind: 'loading', message: 'Fetching model file…' };

    try {
      const response = await readAssetFile(assetData.id);
      state = { kind: 'loading', message: 'Parsing STEP geometry…' };

      const fileBytes = base64ToUint8Array(response.data);
      const group = await loadStepFileToGroup(fileBytes);

      // Remove previous model
      clearModel();

      modelGroup = group;
      scene.add(group);
      fitCameraToModel(group);
      loadedAssetId = assetData.id;
      state = { kind: 'ready' };
    } catch (err: any) {
      console.error('3D preview error:', err);
      state = { kind: 'error', message: err?.message || 'Failed to load model' };
    }
  }

  function resetView(): void {
    if (modelGroup) {
      fitCameraToModel(modelGroup);
    }
  }

  // Setup scene on mount, cleanup on destroy
  onMount(() => {
    if (canvasContainer) {
      setupScene(canvasContainer);
      loadAsset(asset);
    }
    return cleanup;
  });

  // React to asset changes
  $effect(() => {
    if (asset && scene) {
      loadAsset(asset);
    }
  });
</script>

<div class="model-preview">
  <div class="canvas-container" bind:this={canvasContainer}></div>

  {#if state.kind === 'loading'}
    <div class="overlay">
      <div class="loading-indicator">
        <div class="spinner"></div>
        <div class="loading-text">{state.message}</div>
      </div>
    </div>
  {:else if state.kind === 'error'}
    <div class="overlay">
      <div class="error-state">
        <div class="error-icon">⚠</div>
        <div class="error-title">Preview unavailable</div>
        <div class="error-message">{state.message}</div>
      </div>
    </div>
  {:else if state.kind === 'unsupported'}
    <div class="overlay">
      <div class="error-state">
        <div class="error-icon">◇</div>
        <div class="error-title">Format not supported</div>
        <div class="error-message">{state.extension} preview is not yet available</div>
      </div>
    </div>
  {/if}

  {#if state.kind === 'ready'}
    <div class="toolbar">
      <button class="toolbar-btn" onclick={resetView} title="Reset view">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M3 12a9 9 0 0 1 9-9 9.75 9.75 0 0 1 6.74 2.74L21 8" />
          <path d="M21 3v5h-5" />
          <path d="M21 12a9 9 0 0 1-9 9 9.75 9.75 0 0 1-6.74-2.74L3 16" />
          <path d="M3 21v-5h5" />
        </svg>
      </button>
    </div>
  {/if}
</div>

<style>
  .model-preview {
    position: relative;
    width: 100%;
    height: 100%;
    min-height: 200px;
    overflow: hidden;
    border-radius: var(--radius-md);
  }

  .canvas-container {
    width: 100%;
    height: 100%;
    min-height: 200px;
  }

  .canvas-container :global(canvas) {
    display: block;
    width: 100% !important;
    height: 100% !important;
  }

  .overlay {
    position: absolute;
    inset: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--color-bg-muted);
    z-index: 2;
  }

  .loading-indicator {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 12px;
  }

  .spinner {
    width: 28px;
    height: 28px;
    border: 2px solid var(--color-border-strong);
    border-top-color: var(--color-accent);
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  .loading-text {
    font-size: 12px;
    color: var(--color-text-secondary);
  }

  .error-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 6px;
    text-align: center;
    padding: 24px;
  }

  .error-icon {
    font-size: 28px;
    color: var(--color-text-muted);
    opacity: 0.6;
  }

  .error-title {
    font-size: 13px;
    font-weight: 500;
    color: var(--color-text-secondary);
  }

  .error-message {
    font-size: 11px;
    color: var(--color-text-muted);
    max-width: 240px;
  }

  .toolbar {
    position: absolute;
    top: 8px;
    right: 8px;
    display: flex;
    gap: 4px;
    z-index: 3;
  }

  .toolbar-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 28px;
    height: 28px;
    border-radius: var(--radius-md);
    background: var(--color-bg-surface);
    border: 1px solid var(--color-border);
    color: var(--color-text-secondary);
    cursor: pointer;
    transition: background 0.12s, color 0.12s;
  }

  .toolbar-btn:hover {
    background: var(--color-bg-hover);
    color: var(--color-text-primary);
  }
</style>
