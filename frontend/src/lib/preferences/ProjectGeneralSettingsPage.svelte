<script lang="ts">
	import type { Project } from '../backend';

	let {
		project,
		projectPath,
		loading,
		error,
	}: {
		project: Project | null;
		projectPath: string;
		loading: boolean;
		error: string;
	} = $props();

	function importSource(project: Project | null): string {
		if (!project?.importSourceType) {
			return 'No import metadata';
		}
		if (!project.importSourcePath) {
			return project.importSourceType;
		}
		return `${project.importSourceType} · ${project.importSourcePath}`;
	}
</script>

{#if loading}
	<div class="page-empty">Loading project preferences…</div>
{:else if error}
	<div class="page-error">{error}</div>
{:else if !project}
	<div class="page-empty">Open a project window to see project settings.</div>
{:else}
	<div class="settings-page">
		<section class="settings-section">
			<div class="section-header">
				<h2>Project identity</h2>
				<p>The preferences window is attached to the current project context, not the launcher.</p>
			</div>

			<div class="property-list">
				<div class="info-row">
					<span class="info-label">Name</span>
					<span>{project.name}</span>
				</div>
				<div class="info-row">
					<span class="info-label">Project ID</span>
					<span class="mono">{project.id}</span>
				</div>
				<div class="info-row">
					<span class="info-label">Path</span>
					<span class="mono">{projectPath || 'Unavailable'}</span>
				</div>
				<div class="info-row">
					<span class="info-label">Import source</span>
					<span>{importSource(project)}</span>
				</div>
			</div>
		</section>

		<section class="settings-section">
			<div class="section-header">
				<h2>Project summary</h2>
				<p>Current project content that affects planning and sourcing behavior.</p>
			</div>

			<div class="stats-row">
				<div class="stat-item">
					<span class="stat-label">Requirements</span>
					<strong>{project.requirements.length}</strong>
				</div>
				<div class="stat-item">
					<span class="stat-label">Resolved parts</span>
					<strong>{project.requirements.filter((req) => req.resolution || req.selectedComponentId).length}</strong>
				</div>
				<div class="stat-item">
					<span class="stat-label">Created</span>
					<strong>{new Date(project.createdAt).toLocaleDateString()}</strong>
				</div>
			</div>
		</section>
	</div>
{/if}

<style>
	.settings-page {
		display: flex;
		flex-direction: column;
		gap: 12px;
	}

	.page-empty,
	.page-error {
		padding: 28px 0;
	}

	.page-error {
		color: var(--color-danger-text);
	}

	.settings-section {
		border: 1px solid var(--color-border);
		border-radius: var(--radius-lg);
		background: var(--color-bg-surface);
		padding: 14px 16px;
		display: flex;
		flex-direction: column;
		gap: 14px;
	}

	.section-header h2 {
		font-size: 14px;
		font-weight: 600;
		margin-bottom: 4px;
	}

	.section-header p {
		color: var(--color-text-secondary);
		line-height: 1.45;
	}

	.property-list {
		display: flex;
		flex-direction: column;
	}

	.info-row {
		display: grid;
		grid-template-columns: 140px minmax(0, 1fr);
		gap: 16px;
		align-items: start;
		padding: 10px 0;
		border-top: 1px solid var(--color-border);
	}

	.info-row:first-child {
		padding-top: 0;
		border-top: none;
	}

	.info-label,
	.stat-label {
		color: var(--color-text-muted);
		text-transform: uppercase;
		letter-spacing: 0.08em;
		font-size: 10px;
	}

	.mono {
		font-family: var(--font-mono);
		word-break: break-all;
	}

	.stats-row {
		display: grid;
		grid-template-columns: repeat(3, minmax(0, 1fr));
		gap: 12px;
	}

	.stat-item {
		padding: 12px;
		background: rgba(255, 255, 255, 0.015);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-lg);
		display: flex;
		flex-direction: column;
		gap: 6px;
	}

	.stat-item strong {
		font-size: 17px;
		font-weight: 600;
	}

	@media (max-width: 720px) {
		.info-row,
		.stats-row {
			grid-template-columns: 1fr;
		}
	}
</style>