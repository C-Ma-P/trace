<script lang="ts">
  import { onMount } from 'svelte';
  import { slide } from 'svelte/transition';
  import { cubicOut } from 'svelte/easing';
  import QRCode from 'qrcode';
  import { getPhoneIntakeInfo, setPhoneIntakeEnabled, type PhoneIntakeInfo } from '../backend';

  let info: PhoneIntakeInfo | null = $state(null);
  let localActive = $state(false); // optimistic, synced from info.active
  let toggling = $state(false);
  let qrDataURL = $state<string | null>(null);
  let pollTimer: ReturnType<typeof setInterval> | null = null;

  async function refresh() {
    try {
      info = await getPhoneIntakeInfo();
      if (!toggling) localActive = info.active;
    } catch {
      // ignore — service may not be available
    }
  }

  onMount(() => {
    refresh();
    pollTimer = setInterval(refresh, 3000);
    return () => {
      if (pollTimer) clearInterval(pollTimer);
    };
  });

  async function toggle(e: MouseEvent) {
    e.stopPropagation();
    if (!info?.available || toggling) return;
    const next = !localActive;
    localActive = next;
    toggling = true;
    try {
      await setPhoneIntakeEnabled(next);
      await refresh();
    } catch {
      localActive = !next;
    } finally {
      toggling = false;
    }
  }

  $effect(() => {
    const url = info?.url;
    if (!url) {
      qrDataURL = null;
      return;
    }
    // Pre-generate whenever we have a URL so it's ready before the animation runs
    QRCode.toDataURL(url, {
      width: 164,
      margin: 1,
      color: { dark: '#e2e5f0', light: '#1a1d27' },
    }).then(d => { qrDataURL = d; });
  });

  function formatTime(ts: string): string {
    const d = new Date(ts);
    return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
  }

  function copyURL() {
    if (info?.url) navigator.clipboard.writeText(info.url);
  }
</script>

{#if info?.available}
  <div class="intake-section">
    <div class="intake-header">
      <button
        class="intake-expand"
        title="Phone Intake"
        disabled
      >
        <svg viewBox="0 0 20 20" fill="currentColor" class="intake-icon">
          <path d="M7 2a2 2 0 00-2 2v12a2 2 0 002 2h6a2 2 0 002-2V4a2 2 0 00-2-2H7zm3 14a1 1 0 100-2 1 1 0 000 2z" />
        </svg>
        <span class="intake-label">Phone Intake</span>
      </button>
      <button
        class="server-toggle-btn"
        onclick={toggle}
        disabled={toggling}
        title={localActive ? 'Turn off phone intake' : 'Turn on phone intake'}
      >
        <span class="toggle-track" class:on={localActive}>
          <span class="toggle-thumb"></span>
        </span>
      </button>
    </div>

    {#if localActive}
      <div class="intake-panel" transition:slide={{ duration: 220, easing: cubicOut }}>
        <div class="connect-info">
          <div class="intake-url-row">
            <code class="intake-url">{info.url}</code>
            <button class="copy-btn" onclick={copyURL} title="Copy URL">
              <svg viewBox="0 0 16 16" fill="currentColor" width="12" height="12">
                <path d="M0 6.75C0 5.784.784 5 1.75 5h1.5a.75.75 0 010 1.5h-1.5a.25.25 0 00-.25.25v7.5c0 .138.112.25.25.25h7.5a.25.25 0 00.25-.25v-1.5a.75.75 0 011.5 0v1.5A1.75 1.75 0 019.25 16h-7.5A1.75 1.75 0 010 14.25v-7.5z"/>
                <path d="M5 1.75C5 .784 5.784 0 6.75 0h7.5C15.216 0 16 .784 16 1.75v7.5A1.75 1.75 0 0114.25 11h-7.5A1.75 1.75 0 015 9.25v-7.5zm1.75-.25a.25.25 0 00-.25.25v7.5c0 .138.112.25.25.25h7.5a.25.25 0 00.25-.25v-7.5a.25.25 0 00-.25-.25h-7.5z"/>
              </svg>
            </button>
          </div>
          <div class="qr-wrap">
            {#if qrDataURL}
              <img src={qrDataURL} alt="Scan to open phone intake" width="164" height="164" />
            {/if}
          </div>
          <div class="intake-hint">Open on your phone (same Wi-Fi)</div>
        </div>

        {#if info.recent.length > 0}
          <div class="recent-header">Recent</div>
          <div class="recent-list">
            {#each info.recent.slice(0, 8) as ev}
              <div class="recent-item" class:error={!ev.success}>
                <span class="recent-time">{formatTime(ev.timestamp)}</span>
                <span class="recent-name">{ev.displayName || ev.qrData}</span>
                <span class="recent-action">
                  {#if ev.action === 'submit' && ev.success}
                    → {ev.newQuantity ?? '?'}
                  {:else if ev.action === 'lookup'}
                    scanned
                  {:else}
                    {ev.error || 'error'}
                  {/if}
                </span>
              </div>
            {/each}
          </div>
        {/if}
      </div>
    {/if}
  </div>
{/if}

<style>
  .intake-section {
    padding: 4px 8px;
  }

  /* Header row: expand button + inline toggle */
  .intake-header {
    display: flex;
    align-items: center;
    gap: 4px;
    padding: 0 2px;
  }
  .intake-expand {
    display: flex;
    align-items: center;
    gap: 8px;
    flex: 1;
    min-width: 0;
    padding: 7px 8px;
    border-radius: var(--radius-sm);
    color: var(--color-text-secondary);
    font-size: 13px;
    font-weight: 500;
    text-align: left;
    white-space: nowrap;
    overflow: hidden;
    cursor: default;
  }
  .intake-expand:disabled {
    opacity: 1;
  }
  .intake-icon {
    width: 16px;
    height: 16px;
    flex-shrink: 0;
  }
  .intake-label {
    overflow: hidden;
    text-overflow: ellipsis;
  }

  /* Inline server toggle */
  .server-toggle-btn {
    flex-shrink: 0;
    padding: 4px 6px;
    border-radius: var(--radius-sm);
    cursor: pointer;
    transition: background 0.1s;
  }
  .server-toggle-btn:hover:not(:disabled) {
    background: var(--color-bg-sidebar-hover);
  }
  .server-toggle-btn:disabled {
    opacity: 0.5;
    cursor: default;
  }
  .toggle-track {
    position: relative;
    display: block;
    width: 28px;
    height: 16px;
    border-radius: 8px;
    background: var(--color-bg-muted);
    border: 1px solid var(--color-border);
    transition: background 0.15s, border-color 0.15s;
  }
  .toggle-track.on {
    background: var(--color-accent);
    border-color: var(--color-accent);
  }
  .toggle-thumb {
    position: absolute;
    top: 2px;
    left: 2px;
    width: 10px;
    height: 10px;
    border-radius: 50%;
    background: var(--color-text-muted);
    transition: transform 0.15s, background 0.15s;
  }
  .toggle-track.on .toggle-thumb {
    transform: translateX(12px);
    background: #fff;
  }

  .intake-panel {
    padding: 2px 10px 8px;
    overflow: hidden;
    will-change: height;
  }

  /* Connect info block — slides in/out via Svelte transition */
  .connect-info {
    margin-bottom: 8px;
    overflow: hidden;
  }
  .intake-url-row {
    display: flex;
    align-items: center;
    gap: 6px;
    margin-bottom: 8px;
    padding-top: 6px;
  }
  .intake-url {
    font-family: var(--font-mono);
    font-size: 10px;
    color: var(--color-accent-text);
    background: var(--color-bg-muted);
    padding: 4px 6px;
    border-radius: var(--radius-sm);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    flex: 1;
  }
  .copy-btn {
    padding: 4px;
    border-radius: var(--radius-sm);
    color: var(--color-text-muted);
    flex-shrink: 0;
    cursor: pointer;
  }
  .copy-btn:hover {
    color: var(--color-text-primary);
    background: var(--color-bg-hover);
  }
  .qr-wrap {
    display: flex;
    justify-content: center;
    margin-bottom: 6px;
  }
  .qr-wrap canvas {
    border-radius: var(--radius-md);
    display: block;
  }
  .intake-hint {
    font-size: 10px;
    color: var(--color-text-muted);
    text-align: center;
    margin-bottom: 4px;
  }

  /* Recent events */
  .recent-header {
    font-size: 10px;
    font-weight: 600;
    color: var(--color-text-muted);
    text-transform: uppercase;
    letter-spacing: 0.04em;
    margin-bottom: 4px;
    margin-top: 4px;
  }
  .recent-list {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }
  .recent-item {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 11px;
    padding: 2px 0;
  }
  .recent-item.error {
    color: var(--color-danger-text);
  }
  .recent-time {
    color: var(--color-text-muted);
    font-family: var(--font-mono);
    font-size: 10px;
    flex-shrink: 0;
  }
  .recent-name {
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--color-text-primary);
  }
  .recent-action {
    flex-shrink: 0;
    color: var(--color-text-secondary);
    font-size: 10px;
  }
</style>
