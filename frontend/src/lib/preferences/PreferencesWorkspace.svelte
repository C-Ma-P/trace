<script lang="ts">
  import { onMount } from 'svelte';
  import { getProject, getProjectDiskPath, type Project } from '../backend';
  import KiCadIntegrationsPage from './KiCadIntegrationsPage.svelte';
  import PreferencesShell from './PreferencesShell.svelte';
  import ProjectGeneralSettingsPage from './ProjectGeneralSettingsPage.svelte';
  import ProjectSourcingSettingsPage from './ProjectSourcingSettingsPage.svelte';
  import SuppliersSettingsPage from './SuppliersSettingsPage.svelte';

  type PageKey =
    | 'global-suppliers'
    | 'global-supplier-digikey'
    | 'global-supplier-mouser'
    | 'global-supplier-lcsc'
    | 'global-integration-kicad'
    | 'project-general'
    | 'project-sourcing';

  type NavigationNode = {
    id: string;
    label: string;
    hint?: string;
    key?: PageKey;
    children?: NavigationNode[];
    defaultExpanded?: boolean;
  };

  type NavigationGroup = {
    label: string;
    nodes: NavigationNode[];
  };

  let { projectId = null }: { projectId?: string | null } = $props();

  let currentProject = $state<Project | null>(null);
  let currentProjectPath = $state('');
  let projectLoading = $state(false);
  let projectError = $state('');
  let selectedPage = $state<PageKey>('global-suppliers');
  let loadedProjectId = $state<string | null>(null);

  onMount(async () => {
    await loadProjectContext(projectId);
  });

  $effect(() => {
    if (projectId !== loadedProjectId) {
      void loadProjectContext(projectId);
    }
  });

  async function loadProjectContext(nextProjectId: string | null) {
    loadedProjectId = nextProjectId;
    currentProject = null;
    currentProjectPath = '';
    projectError = '';

    if (!nextProjectId) {
      if (isProjectPage(selectedPage)) {
        selectedPage = 'global-suppliers';
      }
      return;
    }

    projectLoading = true;
    try {
      const [project, diskPath] = await Promise.all([
        getProject(nextProjectId),
        getProjectDiskPath(nextProjectId),
      ]);
      currentProject = project;
      currentProjectPath = diskPath;
      if (!isProjectPage(selectedPage)) {
        selectedPage = 'project-general';
      }
    } catch (err: any) {
      projectError = err?.message ?? String(err);
      selectedPage = 'global-suppliers';
    } finally {
      projectLoading = false;
    }
  }

  function isProjectPage(page: PageKey): boolean {
    return page === 'project-general' || page === 'project-sourcing';
  }

  const navigationGroups = $derived.by<NavigationGroup[]>(() => {
    const groups: NavigationGroup[] = [
      {
        label: 'Global',
        nodes: [
          {
            id: 'global-suppliers',
            key: 'global-suppliers',
            label: 'Suppliers',
            hint: 'Provider access, credential storage, and sourcing status',
            defaultExpanded: true,
            children: [
              {
                id: 'global-supplier-digikey',
                key: 'global-supplier-digikey',
                label: 'DigiKey',
                hint: 'OAuth credentials and locale defaults',
              },
              {
                id: 'global-supplier-mouser',
                key: 'global-supplier-mouser',
                label: 'Mouser',
                hint: 'API key and provider readiness',
              },
              {
                id: 'global-supplier-lcsc',
                key: 'global-supplier-lcsc',
                label: 'LCSC',
                hint: 'Public provider settings',
              },
            ],
          },
          {
            id: 'global-integrations',
            label: 'Integrations',
            hint: 'External tool paths and import defaults',
            children: [
              {
                id: 'global-integration-kicad',
                key: 'global-integration-kicad',
                label: 'KiCad',
                hint: 'Default project roots for importer discovery',
              },
            ],
          },
        ],
      },
    ];

    if (currentProject) {
      groups.push({
        label: 'Project',
        nodes: [
          {
            id: 'project-general',
            key: 'project-general',
            label: 'General',
            hint: 'Identity, location, and import context',
          },
          {
            id: 'project-sourcing',
            key: 'project-sourcing',
            label: 'Sourcing',
            hint: 'Project-level sourcing behavior and readiness',
          },
        ],
      });
    }

    return groups;
  });

  const selectedMeta = $derived.by(() => {
    switch (selectedPage) {
      case 'global-supplier-digikey':
        return {
          title: 'DigiKey',
          description: 'Configure DigiKey access, locale defaults, and secure client-secret handling.',
          scope: 'Global',
          path: ['Global', 'Suppliers', 'DigiKey'],
          supplierSection: 'digikey' as const,
        };
      case 'global-supplier-mouser':
        return {
          title: 'Mouser',
          description: 'Configure Mouser access and secure API-key storage.',
          scope: 'Global',
          path: ['Global', 'Suppliers', 'Mouser'],
          supplierSection: 'mouser' as const,
        };
      case 'global-supplier-lcsc':
        return {
          title: 'LCSC',
          description: 'Configure LCSC provider defaults and readiness.',
          scope: 'Global',
          path: ['Global', 'Suppliers', 'LCSC'],
          supplierSection: 'lcsc' as const,
        };
      case 'global-integration-kicad':
        return {
          title: 'KiCad',
          description: 'Set the default folders the KiCad importer scans for projects when it opens.',
          scope: 'Global',
          path: ['Global', 'Integrations', 'KiCad'],
        };
      case 'project-general':
        return {
          title: 'Project General',
          description: 'Project identity, disk context, and the current requirement set.',
          scope: currentProject ? currentProject.name : 'Project',
          path: ['Project', 'General'],
        };
      case 'project-sourcing':
        return {
          title: 'Project Sourcing',
          description: 'How this project resolves engineering parts, on-hand stock, and procurement lookup.',
          scope: currentProject ? currentProject.name : 'Project',
          path: ['Project', 'Sourcing'],
        };
      default:
        return {
          title: 'Suppliers',
          description: 'Global provider access, credential storage, and sourcing configuration.',
          scope: 'Global',
          path: ['Global', 'Suppliers'],
          supplierSection: 'overview' as const,
        };
    }
  });
</script>

<PreferencesShell
  groups={navigationGroups}
  bind:selectedPage
  pageTitle={selectedMeta.title}
  pageDescription={selectedMeta.description}
  pageScope={selectedMeta.scope}
  pagePath={selectedMeta.path}
>
  {#if selectedPage === 'global-suppliers' || selectedPage === 'global-supplier-digikey' || selectedPage === 'global-supplier-mouser' || selectedPage === 'global-supplier-lcsc'}
    <SuppliersSettingsPage section={selectedMeta.supplierSection ?? 'overview'} />
  {:else if selectedPage === 'global-integration-kicad'}
    <KiCadIntegrationsPage />
  {:else if selectedPage === 'project-general'}
    <ProjectGeneralSettingsPage
      project={currentProject}
      projectPath={currentProjectPath}
      loading={projectLoading}
      error={projectError}
    />
  {:else if selectedPage === 'project-sourcing'}
    <ProjectSourcingSettingsPage
      project={currentProject}
      loading={projectLoading}
      error={projectError}
    />
  {/if}
</PreferencesShell>

<style>
  :global(body) {
    overflow: hidden;
  }
</style>