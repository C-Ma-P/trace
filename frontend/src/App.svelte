<script lang="ts">
  import { onMount } from 'svelte';
  import Sidebar from './lib/Sidebar.svelte';
  import ComponentsWorkspace from './lib/components/ComponentsWorkspace.svelte';
  import LauncherWorkspace from './lib/launcher/LauncherWorkspace.svelte';
  import PreferencesWorkspace from './lib/preferences/PreferencesWorkspace.svelte';
  import ProjectsWorkspace from './lib/projects/ProjectsWorkspace.svelte';
  import { notifyAppReady } from './lib/appReady';
  import { getStartupStatus } from './lib/backend';
  import { openProjectWindow, openProjectWindowKeepLauncher } from './lib/windowService';

  type WindowMode = 'launcher' | 'project' | 'preferences';
  type StartupState = 'ready' | 'failed' | 'unknown';

  const params = new URLSearchParams(window.location.search);
  const mode = (params.get('mode') as WindowMode) || 'launcher';
  const initialProjectId = params.get('projectId');

  function parseStartupState(value: string | null): StartupState {
    if (value === 'ready' || value === 'failed') {
      return value;
    }
    return 'unknown';
  }

  const startup = parseStartupState(params.get('startup'));

  let currentSection: 'home' | 'components' = $state(mode === 'project' ? 'home' : 'components');
  // Always start as false so the workspace gate is in place until onMount runs.
  // onMount sets this true either directly (from URL param) or after the RPC.
  let startupChecked = $state(false);
  let startupError = $state('');
  let startupErrorTitle = $state('');
  let startupErrorBody = $state('');
  let appReadyNotified = $state(false);

  function timeout(ms: number): Promise<never> {
    return new Promise((_, reject) => {
      setTimeout(() => reject(new Error(`Startup check timed out after ${ms}ms`)), ms);
    });
  }

  async function hydrateStartupStatus() {
    try {
      const status = await Promise.race([getStartupStatus(), timeout(10000)]);
      if (status.ready) {
        startupError = '';
        startupErrorTitle = '';
        startupErrorBody = '';
      } else {
        startupError = status.error;
        startupErrorTitle = 'Database Unavailable';
        startupErrorBody =
          'Trace could not initialize the database. The app cannot be used until this is resolved.';
      }
    } catch (err) {
      startupError = err instanceof Error ? err.message : String(err);
      startupErrorTitle = 'Backend Unavailable';
      startupErrorBody =
        'Trace could not reach the backend runtime. The app cannot be used until this is resolved.';
    } finally {
      startupChecked = true;
    }
  }

  onMount(() => {
    if (startup === 'ready') {
      // Backend already confirmed ready via URL param — skip the RPC.
      startupChecked = true;
      return;
    }
    if (startup === 'failed') {
      startupError = 'Database initialization failed before the window opened.';
      startupErrorTitle = 'Database Unavailable';
      startupErrorBody =
        'Trace could not initialize the database. The app cannot be used until this is resolved.';
      startupChecked = true;
      return;
    }
    void hydrateStartupStatus();
  });

  // Dismiss the boot shell once the workspace or error screen has rendered.
  // notifyAppReady() uses tick() + double-rAF so the content is painted first.
  $effect(() => {
    if (!startupChecked || appReadyNotified) {
      return;
    }
    appReadyNotified = true;
    void notifyAppReady();
  });
</script>

{#if !startupChecked}
  <div class="startup-loading"></div>
{:else if startupError}
  <div class="startup-error-screen">
    <div class="startup-error-card">
      <h2 class="startup-error-title">{startupErrorTitle || 'Startup Error'}</h2>
      <p class="startup-error-body">
        {startupErrorBody || 'Trace could not start.'}
      </p>
      <pre class="startup-error-detail">{startupError}</pre>
      <p class="startup-error-hint">
        Ensure PostgreSQL is running and the database exists, then restart the app.<br />
        Set <code>DATABASE_URL</code> to override the default connection string.
      </p>
    </div>
  </div>
{:else}
    {#if mode === 'launcher'}
    <LauncherWorkspace
      onOpenProject={(id) => void openProjectWindow(id)}
      onOpenProjectKeepLauncher={(id) => void openProjectWindowKeepLauncher(id)}
    />
  {:else if mode === 'preferences'}
    <PreferencesWorkspace projectId={initialProjectId ?? null} />
  {:else}
    <div class="app-layout">
      <Sidebar bind:currentSection />
      <main class="main-content">
        {#if currentSection === 'home'}
          <ProjectsWorkspace requestedProjectId={initialProjectId ?? null} />
        {:else}
          <ComponentsWorkspace />
        {/if}
      </main>
    </div>
  {/if}
{/if}

<style>
  .startup-loading {
    height: 100vh;
    background: var(--color-bg-app);
  }
  .app-layout {
    display: flex;
    height: 100vh;
    overflow: hidden;
  }
  .main-content {
    flex: 1;
    overflow: hidden;
    background: var(--color-bg-app);
  }
  .startup-error-screen {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 100vh;
    background: var(--color-bg-app);
    padding: 40px;
  }
  .startup-error-card {
    max-width: 560px;
    width: 100%;
    background: var(--color-bg-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-lg);
    padding: 32px;
    display: flex;
    flex-direction: column;
    gap: 16px;
  }
  .startup-error-title {
    font-size: 18px;
    font-weight: 600;
    color: var(--color-danger);
  }
  .startup-error-body {
    font-size: 13px;
    color: var(--color-text-primary);
    line-height: 1.6;
  }
  .startup-error-detail {
    font-family: var(--font-mono);
    font-size: 11px;
    background: var(--color-bg-muted);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    padding: 12px;
    white-space: pre-wrap;
    word-break: break-all;
    color: var(--color-text-primary);
  }
  .startup-error-hint {
    font-size: 12px;
    color: var(--color-text-secondary);
    line-height: 1.6;
  }
  .startup-error-hint code {
    font-family: var(--font-mono);
    background: var(--color-bg-muted);
    padding: 1px 4px;
    border-radius: var(--radius-sm);
  }
</style>
