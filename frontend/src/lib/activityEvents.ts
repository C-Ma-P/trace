import { derived, writable } from 'svelte/store';

export type ActivityDomain = 'activity' | 'sourcing' | 'phone' | 'import' | 'asset-probe' | 'export';
export type ActivitySeverity = 'info' | 'success' | 'warning' | 'error';

export interface ActivityEvent {
  id: string;
  timestamp: string;
  domain: ActivityDomain;
  severity: ActivitySeverity;
  kind?: string;
  message: string;
  metadata?: Record<string, unknown>;
}

export const activityEvents = writable<ActivityEvent[]>([]);

function makeEvent(event: Omit<ActivityEvent, 'id' | 'timestamp'>): ActivityEvent {
  return {
    ...event,
    id: `${event.domain}-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
    timestamp: new Date().toISOString(),
  };
}

export const activityCounts = derived(activityEvents, ($events) => ({
  all: $events.length,
  activity: $events.filter((event) => event.domain === 'activity').length,
  sourcing: $events.filter((event) => event.domain === 'sourcing').length,
  phone: $events.filter((event) => event.domain === 'phone').length,
  warning: $events.filter((event) => event.severity === 'warning').length,
  error: $events.filter((event) => event.severity === 'error').length,
  recent: $events.slice(-5).reverse(),
}));

export function pushActivityEvent(event: Omit<ActivityEvent, 'id' | 'timestamp'>) {
  activityEvents.update((existing) => [makeEvent(event), ...existing].slice(0, 50));
}

export const activitySummary = derived(activityEvents, ($events) => ({
  activeDomains: Array.from(new Set($events.map((event) => event.domain))),
  mostSevere: ['error', 'warning', 'success', 'info'].find((level) =>
    $events.some((event) => event.severity === level),
  ) as ActivitySeverity,
}));
