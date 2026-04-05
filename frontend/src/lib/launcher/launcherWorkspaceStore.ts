import { derived, writable } from 'svelte/store';
import {
  listRecentProjects,
  type RecentProject,
} from '../backend';
import { listOpenProjectIDs } from '../windowService';

function isSubsequence(needle: string, haystack: string): boolean {
  let j = 0;
  for (let i = 0; i < haystack.length && j < needle.length; i++) {
    if (haystack[i] === needle[j]) j++;
  }
  return j === needle.length;
}

function fuzzyMatch(query: string, text: string): boolean {
  const q = query.trim().toLowerCase();
  if (q === '') return true;
  const t = text.toLowerCase();

  const tokens = q.split(/\s+/).filter(Boolean);
  for (const tok of tokens) {
    if (!isSubsequence(tok, t)) return false;
  }
  return true;
}

export function createLauncherWorkspaceStore() {
  const recent = writable<RecentProject[]>([]);
  const openProjectIDs = writable<string[]>([]);
  const filterText = writable('');
  const loading = writable(false);

  const visibleProjects = derived([recent, filterText], ([$recent, $filterText]) => {
    return $recent.filter((p) => fuzzyMatch($filterText, p.name || p.id));
  });

  async function init() {
    loading.set(true);
    try {
	  const [recentProjects, openIDs] = await Promise.all([listRecentProjects(), listOpenProjectIDs()]);
	  recent.set(recentProjects);
	  openProjectIDs.set(openIDs);
    } finally {
      loading.set(false);
    }
  }

  async function refreshRecent() {
    recent.set(await listRecentProjects());
  }

  async function refreshOpenProjects() {
    openProjectIDs.set(await listOpenProjectIDs());
  }

  return {
    recent,
    openProjectIDs,
    filterText,
    loading,
    visibleProjects,
    init,
    refreshRecent,
    refreshOpenProjects,
  };
}
