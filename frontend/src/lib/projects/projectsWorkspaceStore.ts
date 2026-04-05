import { writable } from 'svelte/store';
import {
  getCategories,
  getProject,
  type CategoryInfo,
  type Project,
} from '../backend';

export function createProjectsWorkspaceStore() {
  const categories = writable<CategoryInfo[]>([]);
  const selectedProject = writable<Project | null>(null);
  const loading = writable(false);
  const error = writable('');

  async function init(requestedProjectId: string | null) {
    try {
      categories.set(await getCategories());
      if (requestedProjectId) {
        await loadProject(requestedProjectId);
      }
    } catch (e: any) {
      error.set(e?.message ?? String(e));
    }
  }

  async function loadProject(id: string) {
    loading.set(true);
    error.set('');
    try {
      selectedProject.set(await getProject(id));
    } catch (e: any) {
      error.set(e?.message ?? String(e));
      selectedProject.set(null);
    } finally {
      loading.set(false);
    }
  }

  return {
    categories,
    selectedProject,
    loading,
    error,
    init,
    loadProject,
  };
}
