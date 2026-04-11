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
  let lastViewedAt: Record<ConsoleDomain, string> = {
    activity: '',
    sourcing: '',
    phone: '',
  };
  let refreshTimer: ReturnType<typeof setInterval> | null = null;

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

  function formatMetadataValue(value: unknown): string {
    if (value === null || value === undefined) {
      return '';
    }
    if (Array.isArray(value)) {
      return value.map((item) => String(item)).join(', ');
    }
    if (typeof value === 'object') {
      return Object.entries(value)
        .map(([key, val]) => `${key}: ${String(val)}`)
        .join(', ');
    }
    return String(value);
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

      if (expanded && expandedEventId === null) {
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
        onclick={() => toggleConsole('activity')}
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
        onclick={() => toggleConsole('sourcing')}
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
        onclick={() => toggleConsole('phone')}
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
      <div class="panel-content" bind:this={panelContentEl}>
        {#if activeTab === 'activity'}
          <div class="panel-summary">All recent workspace activity and alerts.</div>
          {#if events.activity.length === 0}
            <div class="empty-msg">No structured activity events yet.</div>
          {:else}
            <div class="event-list">
              {#each events.activity.slice().reverse() as event}
                <button type="button" class="event-line {badgeClass(event.severity)} {expandedEventId === event.id ? 'selected' : ''}" onclick={() => toggleEventDetails(event.id)}>
                  <span class="event-time">{formatTime(event.timestamp)}</span>
                  <span class="event-severity">{event.severity}</span>
                  <span class="event-domain">{event.domain}</span>
                  {#if event.kind}
                    <span class="event-kind">{event.kind}</span>
                  {/if}
                  <span class="event-message">{event.message}</span>
                </button>
                {#if expandedEventId === event.id && event.metadata}
                  <div class="event-details">
                    {#each Object.entries(event.metadata) as [key, value]}
                      <div class="metadata-row">
                        <span class="metadata-key">{key}</span>
                        <span class="metadata-value">{formatMetadataValue(value)}</span>
                      </div>
                    {/each}
                  </div>
                {/if}
              {/each}
            </div>
          {/if}
        {:else if activeTab === 'sourcing'}
          <div class="panel-summary">Supplier and sourcing-related history, problems, and state changes.</div>
          {#if events.sourcing.length === 0}
            <div class="empty-msg">No sourcing events are available.</div>
          {:else}
            <div class="event-list">
              {#each events.sourcing.slice().reverse() as event}
                <button type="button" class="event-line {badgeClass(event.severity)} {expandedEventId === event.id ? 'selected' : ''}" onclick={() => toggleEventDetails(event.id)}>
                  <span class="event-time">{formatTime(event.timestamp)}</span>
                  <span class="event-severity">{event.severity}</span>
                  <span class="event-domain">{event.metadata?.provider ?? 'sourcing'}</span>
                  {#if event.kind}
                    <span class="event-kind">{event.kind}</span>
                  {/if}
                  <span class="event-message">{event.message}</span>
                </button>
                {#if expandedEventId === event.id && event.metadata}
                  <div class="event-details">
                    {#each Object.entries(event.metadata) as [key, value]}
                      <div class="metadata-row">
                        <span class="metadata-key">{key}</span>
                        <span class="metadata-value">{formatMetadataValue(value)}</span>
                      </div>
                    {/each}
                  </div>
                {/if}
              {/each}
            </div>
          {/if}
        {:else}
          <div class="panel-summary">Phone intake event history and recent mobile scan activity.</div>
          {#if events.phone.length === 0}
            <div class="empty-msg">No phone intake events are available.</div>
          {:else}
            <div class="event-list">
              {#each events.phone.slice().reverse() as event}
                <button type="button" class="event-line {badgeClass(event.severity)} {expandedEventId === event.id ? 'selected' : ''}" onclick={() => toggleEventDetails(event.id)}>
                  <span class="event-time">{formatTime(event.timestamp)}</span>
                  <span class="event-severity">{event.severity}</span>
                  <span class="event-domain">Phone</span>
                  {#if event.kind}
                    <span class="event-kind">{event.kind}</span>
                  {/if}
                  <span class="event-message">{event.message}</span>
                </button>
                {#if expandedEventId === event.id && event.metadata}
                  <div class="event-details">
                    {#each Object.entries(event.metadata) as [key, value]}
                      <div class="metadata-row">
                        <span class="metadata-key">{key}</span>
                        <span class="metadata-value">{formatMetadataValue(value)}</span>
                      </div>
                    {/each}
                  </div>
                {/if}
              {/each}
            </div>
          {/if}
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
    background: var(--color-bg-surface);
    display: flex;
    flex-direction: column;
    min-height: 240px;
    max-height: 280px;
    overflow: hidden;
  }

  .panel-header {
    padding: 12px 16px;
    background: var(--color-bg-app);
    border-bottom: 1px solid var(--color-border);
  }

  .panel-title {
    font-size: 13px;
    font-weight: 600;
    color: var(--color-text-primary);
  }

  .panel-content {
    padding: 14px 16px 18px;
    overflow: auto;
    flex: 1;
  }

  .panel-summary {
    margin-bottom: 12px;
    font-size: 12px;
    color: var(--color-text-secondary);
  }

  .event-list {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .event-line {
    all: unset;
    box-sizing: border-box;
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 8px;
    padding: 8px 12px;
    cursor: pointer;
    border-radius: 6px;
    background: var(--color-bg-muted);
    transition: background 0.15s ease, transform 0.15s ease;
    font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
  }

  .event-line:hover {
    background: var(--color-bg-hover);
  }

  .event-line.selected {
    background: var(--color-bg-surface);
  }

  .event-line.badge-info .event-severity {
    color: var(--color-text-secondary);
  }
  .event-line.badge-success .event-severity {
    color: var(--color-success);
  }
  .event-line.badge-warning .event-severity {
    color: var(--color-warning);
  }
  .event-line.badge-error .event-severity {
    color: var(--color-danger);
  }

  .event-time,
  .event-domain,
  .event-kind,
  .event-severity {
    font-size: 11px;
    color: var(--color-text-tertiary);
    white-space: nowrap;
  }

  .event-message {
    flex: 1 1 100%;
    font-size: 13px;
    line-height: 1.3;
    color: var(--color-text-primary);
    overflow-wrap: anywhere;
  }

  .event-kind {
    padding: 2px 8px;
    border-radius: 999px;
    border: 1px solid var(--color-border);
    color: var(--color-text-secondary);
    background: var(--color-bg-muted);
  }

  .event-details {
    padding: 10px 14px 14px 14px;
    background: var(--color-bg-app);
    border-bottom: 1px solid var(--color-border);
    border-left: 3px solid var(--color-border);
  }

  .metadata-row {
    display: flex;
    justify-content: space-between;
    gap: 20px;
    padding: 4px 0;
    font-size: 12px;
    color: var(--color-text-secondary);
  }

  .metadata-key {
    color: var(--color-text-primary);
    font-weight: 600;
  }

  .metadata-value {
    text-align: right;
    flex: 1;
    min-width: 80px;
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
