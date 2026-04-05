import { get, writable } from 'svelte/store';
import {
  getCategories,
  getComponentDetail,
  listComponents,
  type CategoryInfo,
  type Component,
  type ComponentDetail,
  type ComponentFilter,
} from '../backend';

export function createComponentsWorkspaceStore() {
  const categories = writable<CategoryInfo[]>([]);
  const components = writable<Component[]>([]);
  const selectedId = writable<string | null>(null);
  const selectedDetail = writable<ComponentDetail | null>(null);
  const filter = writable<Partial<ComponentFilter>>({});
  const loading = writable(false);
  const error = writable('');

  async function init() {
    try {
      categories.set(await getCategories());
      await loadComponents();
    } catch (e: any) {
      error.set(e?.message ?? String(e));
    }
  }

  async function loadComponents() {
    loading.set(true);
    error.set('');
    try {
      components.set(await listComponents(get(filter)));
    } catch (e: any) {
      error.set(e?.message ?? String(e));
    } finally {
      loading.set(false);
    }
  }

  async function selectComponent(id: string) {
    selectedId.set(id);
    try {
      selectedDetail.set(await getComponentDetail(id));
    } catch (e: any) {
      error.set(e?.message ?? String(e));
    }
  }

  async function afterCreated(comp: Component) {
    await loadComponents();
    await selectComponent(comp.id);
  }

  async function afterUpdated() {
    const id = get(selectedId);
    if (id) {
      await selectComponent(id);
    }
    await loadComponents();
  }

  async function afterDeleted() {
    selectedId.set(null);
    selectedDetail.set(null);
    await loadComponents();
  }

  async function setFilterAndReload(f: Partial<ComponentFilter>) {
    filter.set(f);
    await loadComponents();
  }

  return {
    categories,
    components,
    selectedId,
    selectedDetail,
    filter,
    loading,
    error,
    init,
    loadComponents,
    selectComponent,
    afterCreated,
    afterUpdated,
    afterDeleted,
    setFilterAndReload,
  };
}
