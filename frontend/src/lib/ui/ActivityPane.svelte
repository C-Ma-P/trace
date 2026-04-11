<script lang="ts">
  import { onMount, tick } from 'svelte';
  import { slide } from 'svelte/transition';
  import { cubicOut } from 'svelte/easing';
  import { type ActivityDomain, type ActivityEvent } from '../activityEvents';
  import { getActivityEvents, getPhoneIntakeInfo } from '../backend';

  type ConsoleDomain = 'activity' | 'sourcing' | 'phone';

  let activeTab: ConsoleDomain = $state('activity');
  let expanded = $state(false);
  let phoneIntakeActive = $state(false);

  let summary = $state({ activity: 0, sourcing: 0, phone: 0, warnings: 0, errors: 0 });
  let events = $state({ activity: [] as ActivityEvent[], sourcing: [] as ActivityEvent[], phone: [] as ActivityEvent[] });
  let expandedEventId = $state<string | null>(null);
  let unreadCounts = $state({ activity: 0, sourcing: 0, phone: 0 });
  let panelContentEl: HTMLDivElement | null = $state(null);
  let isNearBottom = true;
  let lastViewedAt: Record<ConsoleDomain, string> = {
    activity: '',
    sourcing: '',
    phone: '',
  };
  let refreshTimer: ReturnType<typeof setInterval> | null = null;

  const SCROLL_THRESHOLD = 36;

  function visibleEventsForTab() {
    return getSortedEvents(events[activeTab]);
  }

  function handlePanelScroll() {
    if (!panelContentEl) return;
    const distanceFromBottom = panelContentEl.scrollHeight - (panelContentEl.scrollTop + panelContentEl.clientHeight);
    isNearBottom = distanceFromBottom <= SCROLL_THRESHOLD;
  }

  function getSortedEvents(list: ActivityEvent[]) {
    return [...list].sort((a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime());
  }

  function formatSeverityLabel(event: ActivityEvent) {
    return event.severity ? event.severity.toUpperCase() : 'INFO';
  }

  function formatEventDomain(event: ActivityEvent) {
    if (event.domain === 'sourcing' && event.metadata?.provider) {
      return `${event.domain}/${event.metadata.provider}`;
    }
    if (event.domain === 'phone') {
      return 'phone';
    }
    return event.domain;
  }

  function formatLogLine(event: ActivityEvent) {
    const prefix = `[${formatTime(event.timestamp)}] ${formatSeverityLabel(event)} ${formatEventDomain(event)}`;
    const kind = event.kind ? ` ${event.kind}` : '';
    return `${prefix}${kind} :: ${event.message}`;
  }

  function formatMetadataJSON(metadata: unknown) {
    if (metadata === null || metadata === undefined) {
      return '';
    }
    try {
      return JSON.stringify(metadata, null, 2);
    } catch {
      return String(metadata);
    }
  }

  function emptyMessageForTab(domain: ConsoleDomain) {
    return {
      activity: 'No structured activity events yet.',
      sourcing: 'No sourcing events are available.',
      phone: 'No phone intake events are available.',
    }[domain];
  }

  function shouldAutoScrollToBottom() {
    return expanded && expandedEventId === null && isNearBottom;
  }

  function updateUnreadCounts() {
    unreadCounts = {
      activity:
        activeTab === 'activity' && expanded
          ? 0
          : events.activity.filter((event: ActivityEvent) => {
              const timestamp = new Date(event.timestamp).getTime();
              return lastViewedAt.activity
                ? timestamp > new Date(lastViewedAt.activity).getTime()
                : true;
            }).length,
      sourcing:
        activeTab === 'sourcing' && expanded
          ? 0
          : events.sourcing.filter((event: ActivityEvent) => {
              const timestamp = new Date(event.timestamp).getTime();
              return lastViewedAt.sourcing
                ? timestamp > new Date(lastViewedAt.sourcing).getTime()
                : true;
            }).length,
      phone:
        activeTab === 'phone' && expanded
          ? 0
          : events.phone.filter((event: ActivityEvent) => {
              const timestamp = new Date(event.timestamp).getTime();
              return lastViewedAt.phone
                ? timestamp > new Date(lastViewedAt.phone).getTime()
                : true;
            }).length,
    };
  }

  let copiedEventId: string | null = $state(null);

  function toggleEventDetails(id: string) {
    expandedEventId = expandedEventId === id ? null : id;
  }

  async function copyLogLine(event: ActivityEvent) {
    const { id, ...eventWithoutId } = event;
    const text = JSON.stringify(eventWithoutId, null, 2);
    try {
      await navigator.clipboard.writeText(text);
      copiedEventId = event.id;
      setTimeout(() => {
        if (copiedEventId === event.id) copiedEventId = null;
      }, 1500);
    } catch {
      // ignore failures silently; no copy feedback needed
    }
  }

  async function refreshPhoneIntakeInfo() {
    try {
      const info = await getPhoneIntakeInfo();
      phoneIntakeActive = info.active;
    } catch {
      phoneIntakeActive = false;
    }
  }

  function mapEvents(backendEvents: ActivityEvent[]) {
    events = {
      activity: backendEvents.filter((event) => event.domain === 'activity'),
      sourcing: backendEvents.filter((event) => event.domain === 'sourcing'),
      phone: backendEvents.filter((event) => event.domain === 'phone'),
    };
    summary = {
      activity: events.activity.length,
      sourcing: events.sourcing.length,
      phone: events.phone.length,
      warnings: backendEvents.filter((event) => event.severity === 'warning').length,
      errors: backendEvents.filter((event) => event.severity === 'error').length,
    };
  }

  function getLastEventId(list: ActivityEvent[]) {
    return list.length ? list[list.length - 1].id : null;
  }

  async function scrollToNewestEvent() {
    if (!panelContentEl) return;
    await tick();
    panelContentEl.scrollTo({ top: panelContentEl.scrollHeight, behavior: 'smooth' });
  }

  async function refreshActivityEvents() {
    try {
      const backendEvents = await getActivityEvents();
      const previousVisibleEvents = events[activeTab].slice();
      mapEvents(backendEvents);
      updateUnreadCounts();

      if (expanded && shouldAutoScrollToBottom()) {
        const previousId = getLastEventId(previousVisibleEvents);
        const currentId = getLastEventId(events[activeTab]);
        if (currentId && currentId !== previousId) {
          await scrollToNewestEvent();
        }
      }
    } catch {
      // Ignore backend errors; activity pane remains available when the service is ready.
    }
  }

  onMount(() => {
    refreshActivityEvents();
    refreshPhoneIntakeInfo();
    refreshTimer = setInterval(async () => {
      await refreshActivityEvents();
      await refreshPhoneIntakeInfo();
    }, 3000);
    return () => {
      if (refreshTimer) clearInterval(refreshTimer);
    };
  });

  function markConsoleViewed(domain: ConsoleDomain) {
    lastViewedAt = { ...lastViewedAt, [domain]: new Date().toISOString() };
    updateUnreadCounts();
  }

  async function toggleConsole(domain: ConsoleDomain) {
    if (activeTab === domain) {
      expanded = !expanded;
      if (expanded) {
        markConsoleViewed(domain);
      }
    } else {
      activeTab = domain;
      expanded = true;
      markConsoleViewed(domain);
    }

    if (expanded && expandedEventId === null) {
      await scrollToNewestEvent();
    }
  }

  function formatTime(ts: string) {
    const d = new Date(ts);
    return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
  }

  function badgeClass(severity: string) {
    return {
      info: 'badge-info',
      success: 'badge-success',
      warning: 'badge-warning',
      error: 'badge-error',
    }[severity] ?? 'badge-info';
  }
</script>

<div class="activity-dock" class:is-expanded={expanded}>
  <div class="dock-rail">
    <div class="dock-tabs" role="tablist" aria-label="Activity dock">
      <button
        type="button"
        role="tab"
        class="dock-tab"
        class:is-active={activeTab === 'activity' && expanded}
        aria-label="Activity"
        aria-selected={activeTab === 'activity' && expanded}
        onclick={() => toggleConsole('activity')}
      >
        <svg viewBox="0 0 20 20" fill="currentColor" class="tab-icon" aria-hidden="true">
          <path d="M4 14h3V6H4zm5 0h3V9H9zm5 0h3V4h-3z" />
        </svg>
        {#if unreadCounts.activity > 0}
          <span class="tab-badge">{unreadCounts.activity}</span>
        {/if}
      </button>
      <button
        type="button"
        role="tab"
        class="dock-tab"
        class:is-active={activeTab === 'sourcing' && expanded}
        aria-label="Sourcing"
        aria-selected={activeTab === 'sourcing' && expanded}
        onclick={() => toggleConsole('sourcing')}
      >
        <svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" class="tab-icon" aria-hidden="true">
          <circle cx="8" cy="8" r="4" />
          <path d="M12.5 12.5l4 4" />
        </svg>
        {#if unreadCounts.sourcing > 0}
          <span class="tab-badge">{unreadCounts.sourcing}</span>
        {/if}
      </button>
      <button
        type="button"
        role="tab"
        class="dock-tab"
        class:is-active={activeTab === 'phone' && expanded}
        class:phone-active={phoneIntakeActive}
        aria-label="Phone intake"
        aria-selected={activeTab === 'phone' && expanded}
        onclick={() => toggleConsole('phone')}
      >
        <svg viewBox="0 0 20 20" fill="currentColor" class="tab-icon" aria-hidden="true">
          <path d="M7 2a2 2 0 00-2 2v12a2 2 0 002 2h6a2 2 0 002-2V4a2 2 0 00-2-2H7zm3 14a1 1 0 100-2 1 1 0 000 2z" />
        </svg>
        {#if unreadCounts.phone > 0}
          <span class="tab-badge">{unreadCounts.phone}</span>
        {/if}
      </button>
    </div>
    <div class="dock-rail-end" aria-hidden="true"></div>
  </div>

  {#if expanded}
    <div class="dock-body" transition:slide={{ duration: 200, easing: cubicOut }}>
      <div class="dock-content" bind:this={panelContentEl} onscroll={handlePanelScroll}>
        {#if visibleEventsForTab().length === 0}
          <div class="empty-msg">{emptyMessageForTab(activeTab)}</div>
        {:else}
          <div class="event-list">
            {#each visibleEventsForTab() as event}
              <div
                class="event-row {badgeClass(event.severity)}"
                class:is-selected={expandedEventId === event.id}
                role="button"
                tabindex="0"
                aria-expanded={expandedEventId === event.id}
                onclick={() => toggleEventDetails(event.id)}
                onkeydown={(e) => {
                  if (e.key === 'Enter' || e.key === ' ') {
                    e.preventDefault();
                    toggleEventDetails(event.id);
                  }
                }}
              >
                <span class="event-text">{formatLogLine(event)}</span>
                <span
                  class="copy-btn"
                  role="button"
                  tabindex="0"
                  aria-label="Copy event JSON"
                  onclick={(e) => { e.stopPropagation(); copyLogLine(event); }}
                  onkeydown={(e) => {
                    e.stopPropagation();
                    if (e.key === 'Enter' || e.key === ' ') {
                      e.preventDefault();
                      copyLogLine(event);
                    }
                  }}
                >
                  {#if copiedEventId === event.id}
                    <span class="copy-feedback">copied</span>
                  {:else}
                    <svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" class="copy-icon" aria-hidden="true">
                      <rect x="8" y="4" width="8" height="10" rx="1" />
                      <path d="M6 8H5a2 2 0 00-2 2v6a2 2 0 002 2h6a2 2 0 002-2v-1" />
                    </svg>
                  {/if}
                </span>
              </div>
              {#if expandedEventId === event.id && event.metadata}
                <div class="event-meta">
                  <pre>{formatMetadataJSON(event.metadata)}</pre>
                </div>
              {/if}
            {/each}
          </div>
        {/if}
      </div>
    </div>
  {/if}
</div>

<style>
  /* ── Dock shell ──────────────────────────────────────────────── */
  .activity-dock {
    position: relative;
    background: var(--color-bg-surface);
    border-top: 1px solid var(--color-border-strong);
  }

  /* ── Dock rail (tab bar) ─────────────────────────────────────── */
  .dock-rail {
    position: relative;
    display: flex;
    align-items: stretch;
    height: 34px;
    background: var(--color-bg-surface);
    border-bottom: 1px solid var(--color-border);
  }

  .dock-tabs {
    display: flex;
    align-items: stretch;
  }

  .dock-rail-end {
    flex: 1;
  }

  /* ── Dock tab buttons ────────────────────────────────────────── */
  .dock-tab {
    position: relative;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-width: 40px;
    padding: 0 13px;
    border: none;
    border-top: 2px solid transparent;
    border-right: 1px solid var(--color-border);
    border-radius: 0;
    background: transparent;
    color: var(--color-text-muted);
    cursor: pointer;
    transition: color 0.12s ease, background 0.12s ease;
  }

  .dock-tab:hover {
    background: var(--color-bg-hover);
    color: var(--color-text-secondary);
  }

  .dock-tab.is-active {
    background: var(--color-bg-muted);
    color: var(--color-text-primary);
    border-top-color: var(--color-accent);
    /* Extend 1px downward to visually cover the rail's bottom border */
    transform: translateY(1px);
    z-index: 2;
  }

  .dock-tab.is-active:hover {
    background: var(--color-bg-muted);
  }

  /* Phone status-aware tab coloring */
  .dock-tab.phone-active {
    color: var(--color-success);
  }

  .dock-tab.phone-active.is-active {
    color: var(--color-success-text);
    border-top-color: var(--color-success);
  }

  /* ── Tab icon ────────────────────────────────────────────────── */
  .tab-icon {
    width: 16px;
    height: 16px;
    flex-shrink: 0;
  }

  /* ── Unread badge ────────────────────────────────────────────── */
  .tab-badge {
    position: absolute;
    top: 3px;
    right: 3px;
    min-width: 14px;
    padding: 0 3px;
    height: 13px;
    background: rgba(255, 255, 255, 0.07);
    border: 1px solid rgba(255, 255, 255, 0.12);
    border-radius: var(--radius-sm);
    font-size: 9px;
    line-height: 13px;
    color: var(--color-text-secondary);
    text-align: center;
    pointer-events: none;
  }

  /* ── Dock body ───────────────────────────────────────────────── */
  .dock-body {
    display: flex;
    flex-direction: column;
    min-height: 200px;
    max-height: 260px;
    overflow: hidden;
  }

  .dock-content {
    flex: 1;
    overflow: auto;
    padding: 4px 0;
    background: var(--color-bg-muted);
    font-family: var(--font-mono);
    font-size: 12px;
    color: var(--color-text-primary);
  }

  /* ── Event list ──────────────────────────────────────────────── */
  .event-list {
    display: flex;
    flex-direction: column;
  }

  .event-row {
    position: relative;
    display: flex;
    align-items: baseline;
    padding: 3px 38px 3px 10px;
    border-left: 2px solid transparent;
    cursor: pointer;
    white-space: pre-wrap;
    word-break: break-word;
    line-height: 1.45;
    transition: background 0.1s ease;
  }

  .event-row:hover {
    background: rgba(255, 255, 255, 0.04);
  }

  .event-row:focus-visible {
    outline: 1px solid rgba(255, 255, 255, 0.15);
    outline-offset: -1px;
  }

  .event-row.is-selected {
    background: rgba(255, 255, 255, 0.06);
    border-left-color: var(--color-accent);
  }

  /* Severity coloring */
  .event-row.badge-info    { color: var(--color-text-secondary); }
  .event-row.badge-success { color: var(--color-success-text); }
  .event-row.badge-warning { color: var(--color-warning-text); }
  .event-row.badge-error   { color: var(--color-danger-text); }

  .event-text {
    flex: 1;
    min-width: 0;
  }

  /* ── Copy affordance ─────────────────────────────────────────── */
  .copy-btn {
    position: absolute;
    right: 6px;
    top: 50%;
    transform: translateY(-50%);
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-width: 24px;
    height: 18px;
    padding: 0 4px;
    background: transparent;
    color: var(--color-text-muted);
    font-family: var(--font-mono);
    font-size: 9px;
    line-height: 1;
    cursor: pointer;
    opacity: 0;
    border-radius: var(--radius-sm);
    transition: opacity 0.1s ease, color 0.1s ease, background 0.1s ease;
  }

  .event-row:hover .copy-btn {
    opacity: 1;
  }

  .copy-btn:hover,
  .copy-btn:focus-visible {
    color: var(--color-text-secondary);
    background: rgba(255, 255, 255, 0.07);
    outline: none;
  }

  .copy-icon {
    width: 12px;
    height: 12px;
  }

  .copy-feedback {
    font-size: 9px;
    color: var(--color-text-muted);
    font-family: var(--font-mono);
  }

  /* ── Expanded event metadata ─────────────────────────────────── */
  .event-meta {
    padding: 5px 10px 5px 24px;
    background: rgba(0, 0, 0, 0.18);
    border-left: 2px solid var(--color-accent);
    overflow-x: auto;
  }

  .event-meta pre {
    margin: 0;
    padding: 0;
    font-family: var(--font-mono);
    font-size: 11px;
    line-height: 1.5;
    white-space: pre-wrap;
    word-break: break-word;
    color: var(--color-text-muted);
  }

  /* ── Empty state ─────────────────────────────────────────────── */
  .empty-msg {
    padding: 24px 0;
    text-align: center;
    font-family: var(--font-mono);
    font-size: 12px;
    color: var(--color-text-muted);
  }
</style>
