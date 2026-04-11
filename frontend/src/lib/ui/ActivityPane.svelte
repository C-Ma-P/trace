<script lang="ts">
  import { onMount, tick } from 'svelte';
  import { slide } from 'svelte/transition';
  import { cubicOut } from 'svelte/easing';
  import { type ActivityDomain, type ActivityEvent } from '../activityEvents';
  import { getActivityEvents, getPhoneIntakeInfo } from '../backend';

  type ConsoleDomain = 'activity' | 'sourcing' | 'phone';

  let activeTab: ActivityDomain = $state('activity');
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

  function toggleEventDetails(id: string) {
    expandedEventId = expandedEventId === id ? null : id;
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

<div class="activity-pane" class:expanded={expanded}>
  <div class="strip">
    <div class="strip-left" aria-hidden="true"></div>
    <div class="strip-actions" role="tablist" aria-label="Activity console selectors">
      <button
        type="button"
        class="action-btn {activeTab === 'activity' ? 'active' : ''}"
        aria-label="Open activity console"
        on:click={() => toggleConsole('activity')}
      >
        <svg viewBox="0 0 20 20" fill="currentColor" class="action-icon">
          <path d="M4 14h3V6H4zm5 0h3V9H9zm5 0h3V4h-3z" />
        </svg>
        {#if unreadCounts.activity > 0}
          <span class="action-count">{unreadCounts.activity}</span>
        {/if}
      </button>
      <button
        type="button"
        class="action-btn {activeTab === 'sourcing' ? 'active' : ''}"
        aria-label="Open sourcing console"
        on:click={() => toggleConsole('sourcing')}
      >
        <svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" class="action-icon">
          <circle cx="8" cy="8" r="4" />
          <path d="M12.5 12.5l4 4" />
        </svg>
        {#if unreadCounts.sourcing > 0}
          <span class="action-count">{unreadCounts.sourcing}</span>
        {/if}
      </button>
      <button
        type="button"
        class="action-btn phone-btn {activeTab === 'phone' ? 'active' : ''} {phoneIntakeActive ? 'phone-enabled' : ''}"
        aria-label="Open phone intake console"
        on:click={() => toggleConsole('phone')}
      >
        <svg viewBox="0 0 20 20" fill="currentColor" class="action-icon">
          <path d="M7 2a2 2 0 00-2 2v12a2 2 0 002 2h6a2 2 0 002-2V4a2 2 0 00-2-2H7zm3 14a1 1 0 100-2 1 1 0 000 2z" />
        </svg>
        {#if unreadCounts.phone > 0}
          <span class="action-count">{unreadCounts.phone}</span>
        {/if}
      </button>
    </div>
  </div>

  {#if expanded}
    <div class="panel" transition:slide={{ duration: 240, easing: cubicOut }}>
      <div class="panel-header">
        <div class="panel-title">
          {#if activeTab === 'activity'}
            Activity Console
          {:else if activeTab === 'sourcing'}
            Sourcing Console
          {:else}
            Phone Intake Console
          {/if}
        </div>
      </div>
      <div class="panel-content" bind:this={panelContentEl} on:scroll={handlePanelScroll}>
        {#if visibleEventsForTab().length === 0}
          <div class="empty-msg">{emptyMessageForTab(activeTab)}</div>
        {:else}
          <div class="event-list">
            {#each visibleEventsForTab() as event}
              <button
                type="button"
                class="event-line {badgeClass(event.severity)} {expandedEventId === event.id ? 'selected' : ''}"
                aria-expanded={expandedEventId === event.id}
                on:click={() => toggleEventDetails(event.id)}
              >
                <span class="event-text">{formatLogLine(event)}</span>
              </button>
              {#if expandedEventId === event.id && event.metadata}
                <div class="event-details">
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
  .activity-pane {
    position: relative;
    background: var(--color-bg-app);
    border-top: 1px solid var(--color-border);
    transition: height 0.24s ease;
  }

  .strip {
    display: flex;
    align-items: center;
    justify-content: space-between;
    min-height: 44px;
    padding: 0 14px;
    gap: 12px;
    color: var(--color-text-secondary);
    background: var(--color-bg-surface);
    border-bottom: 1px solid var(--color-border);
  }

  .strip-left {
    flex: 1;
  }

  .strip-actions {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .action-btn {
    position: relative;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 38px;
    height: 38px;
    padding: 0;
    border: none;
    border-radius: 0;
    background: transparent;
    color: var(--color-text-secondary);
    cursor: pointer;
    transition: background 0.15s ease, color 0.15s ease;
  }

  .action-btn:hover {
    background: var(--color-bg-surface);
  }

  .action-btn.active {
    color: var(--color-text-primary);
  }

  .phone-btn.phone-enabled {
    color: var(--color-success);
  }

  .phone-btn.phone-enabled .action-icon {
    filter: drop-shadow(0 0 6px rgba(52, 211, 153, 0.32));
  }

  .action-icon {
    width: 18px;
    height: 18px;
  }

  .action-count {
    position: absolute;
    top: 4px;
    right: 4px;
    min-width: 16px;
    padding: 0 4px;
    border-radius: 999px;
    font-size: 10px;
    line-height: 1.4;
    color: var(--color-text-primary);
    background: rgba(255, 255, 255, 0.08);
    border: 1px solid rgba(255, 255, 255, 0.1);
  }

  .panel {
    border-top: 1px solid var(--color-border);
    background: var(--color-bg-app);
    display: flex;
    flex-direction: column;
    min-height: 240px;
    max-height: 280px;
    overflow: hidden;
  }

  .panel-header {
    padding: 10px 14px;
    background: var(--color-bg-surface);
    border-bottom: 1px solid var(--color-border);
  }

  .panel-title {
    font-size: 13px;
    font-weight: 600;
    color: var(--color-text-primary);
  }

  .panel-content {
    padding: 12px 14px 14px;
    overflow: auto;
    flex: 1;
    background: var(--color-bg-muted);
    font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
    color: var(--color-text-primary);
  }

  .event-list {
    display: flex;
    flex-direction: column;
    gap: 1px;
  }

  .event-line {
    all: unset;
    box-sizing: border-box;
    display: block;
    width: 100%;
    padding: 5px 10px;
    cursor: pointer;
    background: transparent;
    border: 1px solid transparent;
    font-size: 12px;
    line-height: 1.4;
    color: var(--color-text-primary);
    white-space: pre-wrap;
    word-break: break-word;
    transition: background 0.15s ease, border-color 0.15s ease;
  }

  .event-line:hover {
    background: rgba(255, 255, 255, 0.04);
  }

  .event-line.selected {
    background: rgba(255, 255, 255, 0.08);
    border-color: var(--color-border);
  }

  .event-line.badge-info {
    color: var(--color-text-secondary);
  }

  .event-line.badge-success {
    color: var(--color-success);
  }

  .event-line.badge-warning {
    color: var(--color-warning);
  }

  .event-line.badge-error {
    color: var(--color-danger);
  }

  .event-text {
    display: block;
    min-height: 1.2em;
  }

  .event-details {
    padding: 10px 12px;
    margin: 0 0 0 1px;
    background: rgba(0, 0, 0, 0.08);
    border-left: 3px solid rgba(255, 255, 255, 0.08);
    font-size: 12px;
    line-height: 1.5;
    color: var(--color-text-secondary);
    overflow-x: auto;
    border-radius: 0 0 6px 6px;
  }

  .event-details pre {
    margin: 0;
    padding: 0;
    font-family: inherit;
    font-size: 12px;
    line-height: 1.5;
    white-space: pre-wrap;
    word-break: break-word;
  }

  .empty-msg {
    color: var(--color-text-secondary);
    font-size: 13px;
    padding: 22px 0;
    text-align: center;
  }

  .empty-msg {
    color: var(--color-text-secondary);
    font-size: 13px;
    padding: 28px 0;
    text-align: center;
  }

  .panel-loading {
    color: var(--color-text-secondary);
    font-size: 13px;
    padding: 18px 0;
  }
</style>
