<script lang="ts">
  import { onMount } from 'svelte';
  import { slide } from 'svelte/transition';
  import { cubicOut } from 'svelte/easing';
  import QRCode from 'qrcode';
  import { getPhoneIntakeInfo, setPhoneIntakeEnabled, setPhoneIntakeHostOverride, clearPhoneIntakeHostOverride, type PhoneIntakeInfo } from '../backend';

  let { collapsed = false }: { collapsed?: boolean } = $props();

  let info: PhoneIntakeInfo | null = $state(null);
  let localActive = $state(false); // optimistic, synced from info.active
  let toggling = $state(false);
  let qrDataURL = $state<string | null>(null);
  let pollTimer: ReturnType<typeof setInterval> | null = null;
  let showOverrideInput = $state(false);
  let overrideInputValue = $state('');

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

  function hostDiagLabel(h: PhoneIntakeInfo['hostInfo'] | undefined): string {
    if (!h) return '';
    if (h.source === 'override') return `${h.host} · override`;
    if (h.source === 'auto') return h.iface ? `${h.host} · ${h.iface}` : h.host;
    return `${h.host} · no LAN address found`;
  }

  async function applyHostOverride() {
    const v = overrideInputValue.trim();
    if (!v) return;
    try {
      await setPhoneIntakeHostOverride(v);
      showOverrideInput = false;
      overrideInputValue = '';
      await refresh();
    } catch { /* ignore */ }
  }

  async function resetHostOverride() {
    try {
      await clearPhoneIntakeHostOverride();
      await refresh();
    } catch { /* ignore */ }
  }
</script>

{#if info?.available}
  <div class="intake-section">
    <div class="intake-collapse-clip" class:open={!collapsed}>
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
            <div class="host-diag">
              <span
                class="host-diag-text"
                class:fallback={info.hostInfo?.source === 'fallback'}
                class:override={info.hostInfo?.source === 'override'}
                title="Host used in phone URL"
              >{hostDiagLabel(info.hostInfo)}</span>
              {#if info.hostInfo?.source === 'override'}
                <button class="host-diag-btn" onclick={resetHostOverride}>use auto</button>
              {:else}
                <button class="host-diag-btn" onclick={() => { showOverrideInput = !showOverrideInput; overrideInputValue = ''; }}>override</button>
              {/if}
            </div>
            {#if showOverrideInput}
              <div class="override-row">
                <input
                  class="override-input"
                  bind:value={overrideInputValue}
                  placeholder="e.g. 192.168.1.50"
                  onkeydown={(e) => { if (e.key === 'Enter') applyHostOverride(); if (e.key === 'Escape') showOverrideInput = false; }}
                />
                <button class="override-set-btn" onclick={applyHostOverride} disabled={!overrideInputValue.trim()}>Set</button>
                <button class="override-cancel-btn" onclick={() => showOverrideInput = false}>✕</button>
              </div>
            {/if}
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

    <button
      class="intake-header"
      onclick={toggle}
      disabled={toggling}
      title={localActive ? 'Turn off phone intake' : 'Turn on phone intake'}
    >
      <svg viewBox="0 0 20 20" fill="currentColor" class="intake-icon" class:on={localActive}>
        <path d="M7 2a2 2 0 00-2 2v12a2 2 0 002 2h6a2 2 0 002-2V4a2 2 0 00-2-2H7zm3 14a1 1 0 100-2 1 1 0 000 2z" />
      </svg>
      <span class="intake-label" class:on={localActive}>Phone Intake</span>
      <span class="toggle-track" class:on={localActive}>
        <span class="toggle-thumb"></span>
      </span>
    </button>
  </div>
{/if}

<style>
  .intake-section {
    padding: 4px 8px;
  }

  /* Header row — single clickable button */
  .intake-header {
    display: flex;
    align-items: center;
    gap: 8px;
    width: 100%;
    padding: 7px 10px;
    border-radius: var(--radius-sm);
    color: var(--color-text-secondary);
    font-size: 13px;
    font-weight: 500;
    text-align: left;
    cursor: pointer;
    transition: background 0.1s;
    white-space: nowrap;
    overflow: hidden;
  }
  .intake-header:hover:not(:disabled) {
    background: var(--color-bg-sidebar-hover);
  }
  .intake-header:disabled {
    opacity: 0.5;
    cursor: default;
  }
  .intake-icon {
    width: 16px;
    height: 16px;
    flex-shrink: 0;
    color: var(--color-text-muted);
    transition: color 0.15s;
  }
  .intake-icon.on {
    color: var(--color-success);
    filter: drop-shadow(0 0 4px rgba(52, 211, 153, 0.4));
  }
  .intake-label {
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .intake-label.on {
    color: var(--color-text-primary);
  }

  /* Inline server toggle */
  .toggle-track {
    position: relative;
    display: block;
    width: 28px;
    height: 16px;
    flex-shrink: 0;
    border-radius: 8px;
    background: var(--color-bg-muted);
    border: 1px solid var(--color-border);
    transition: background 0.15s, border-color 0.15s;
  }
  .toggle-track.on {
    background: var(--color-success);
    border-color: var(--color-success);
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

  .intake-collapse-clip {
    overflow: hidden;
    max-height: 600px;
    transition: max-height 0.18s ease;
  }
  .intake-collapse-clip:not(.open) {
    max-height: 0;
    transition: max-height 0.18s ease 0.18s;
  }

  .intake-panel {
    padding: 2px 10px 8px;
    min-width: 184px;
    transform: translateX(-100%);
    transition: transform 0.18s ease;
    box-sizing: border-box;
  }
  .intake-collapse-clip.open .intake-panel {
    transform: translateX(0);
    transition-delay: 0.05s;
  }

  /* Connect info block */
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
    width: 100%;
    margin-bottom: 6px;
  }
  .qr-wrap img,
  .qr-wrap canvas {
    border-radius: var(--radius-md);
    display: block;
    margin: 0 auto;
  }
  .intake-hint {
    font-size: 10px;
    color: var(--color-text-muted);
    text-align: center;
    margin-bottom: 4px;
  }

  /* Host diagnostics row */
  .host-diag {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 6px;
    margin-bottom: 4px;
    padding: 3px 4px;
    background: var(--color-bg-muted);
    border-radius: var(--radius-sm);
  }
  .host-diag-text {
    font-size: 10px;
    font-family: var(--font-mono);
    color: var(--color-text-muted);
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .host-diag-text.fallback {
    color: var(--color-warning-text, #fbbf24);
  }
  .host-diag-text.override {
    color: var(--color-accent-text);
  }
  .host-diag-btn {
    font-size: 9px;
    font-weight: 500;
    color: var(--color-text-muted);
    padding: 1px 5px;
    border-radius: 3px;
    background: rgba(255,255,255,.06);
    white-space: nowrap;
    flex-shrink: 0;
    cursor: pointer;
  }
  .host-diag-btn:hover {
    color: var(--color-text-primary);
    background: rgba(255,255,255,.12);
  }

  /* Override input row */
  .override-row {
    display: flex;
    align-items: center;
    gap: 4px;
    margin-bottom: 6px;
  }
  .override-input {
    flex: 1;
    min-width: 0;
    background: var(--color-bg-input, rgba(255,255,255,.07));
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
    padding: 4px 6px;
    font-size: 11px;
    font-family: var(--font-mono);
    color: var(--color-text-primary);
    outline: none;
  }
  .override-input:focus {
    border-color: var(--color-accent, #3b82f6);
  }
  .override-set-btn {
    font-size: 11px;
    font-weight: 600;
    color: #fff;
    background: var(--color-accent, #3b82f6);
    border-radius: var(--radius-sm);
    padding: 3px 8px;
    cursor: pointer;
    flex-shrink: 0;
  }
  .override-set-btn:disabled {
    opacity: 0.4;
    cursor: default;
  }
  .override-cancel-btn {
    font-size: 12px;
    color: var(--color-text-muted);
    padding: 2px 5px;
    border-radius: var(--radius-sm);
    cursor: pointer;
    flex-shrink: 0;
  }
  .override-cancel-btn:hover {
    color: var(--color-text-primary);
    background: rgba(255,255,255,.08);
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
