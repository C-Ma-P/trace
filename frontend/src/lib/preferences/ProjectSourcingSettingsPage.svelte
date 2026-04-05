<script lang="ts">
	import type { Project } from '../backend';

	let {
		project,
		loading,
		error,
	}: {
		project: Project | null;
		loading: boolean;
		error: string;
	} = $props();

	const summary = $derived.by(() => {
		const requirements = project?.requirements ?? [];
		const resolved = requirements.filter((req) => req.resolution || req.selectedComponentId).length;
		const unresolved = requirements.length - resolved;
		const categories = new Set(requirements.map((req) => req.category));
		return {
			resolved,
			unresolved,
			categories: categories.size,
		};
	});
</script>

{#if loading}
	<div class="page-empty">Loading project sourcing settings…</div>
{:else if error}
	<div class="page-error">{error}</div>
{:else if !project}
	<div class="page-empty">Open a project window to see project sourcing preferences.</div>
{:else}
	<div class="settings-page">
		<section class="settings-section">
			<div class="section-header">
				<h2>Sourcing model</h2>
				<p>
					This project resolves each requirement to an intended part definition first. On-hand stock and supplier procurement options stay separate layers so planning does not treat “known part”, “owned stock”, and “where to buy it” as the same decision.
				</p>
			</div>

			<div class="policy-list">
				<div class="policy-item">
					<span class="policy-label">Engineering decision</span>
					<p>Choosing a component definition resolves the requirement to the intended part.</p>
				</div>
				<div class="policy-item">
					<span class="policy-label">Stock check</span>
					<p>On-hand quantity is evaluated separately from the chosen part identity.</p>
				</div>
				<div class="policy-item">
					<span class="policy-label">Procurement lookup</span>
					<p>Supplier offers remain procurement-facing and are not stored as the selected engineering part.</p>
				</div>
			</div>
		</section>

		<section class="settings-section">
			<div class="section-header">
				<h2>Current project readiness</h2>
				<p>Lightweight project-level sourcing context for this pass.</p>
			</div>

			<div class="stats-row">
				<div class="stat-item">
					<span class="stat-label">Resolved requirements</span>
					<strong>{summary.resolved}</strong>
				</div>
				<div class="stat-item">
					<span class="stat-label">Unresolved requirements</span>
					<strong>{summary.unresolved}</strong>
				</div>
				<div class="stat-item">
					<span class="stat-label">Requirement categories</span>
					<strong>{summary.categories}</strong>
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

	.policy-list {
		display: flex;
		flex-direction: column;
	}

	.policy-item {
		padding: 10px 0;
		border-top: 1px solid var(--color-border);
	}

	.policy-item:first-child {
		padding-top: 0;
		border-top: none;
	}

	.policy-label,
	.stat-label {
		display: inline-flex;
		margin-bottom: 8px;
		color: var(--color-text-muted);
		text-transform: uppercase;
		letter-spacing: 0.08em;
		font-size: 10px;
	}

	.policy-item p {
		color: var(--color-text-primary);
		line-height: 1.5;
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
		display: block;
		font-size: 17px;
		font-weight: 600;
	}

	@media (max-width: 720px) {
		.stats-row {
			grid-template-columns: 1fr;
		}
	}
</style>