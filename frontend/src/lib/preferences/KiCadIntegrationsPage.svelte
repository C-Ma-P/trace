<script lang="ts">
	import { onMount } from 'svelte';
	import {
		getKiCadPreferences,
		saveKiCadPreferences,
		type KiCadPreferences,
	} from '../backend';
	import { pickDirectory } from '../windowService';

	let loading = $state(true);
	let saving = $state(false);
	let picking = $state(false);
	let error = $state('');
	let success = $state('');
	let projectRoots = $state<string[]>([]);
	let newRoot = $state('');

	onMount(async () => {
		await loadPreferences();
	});

	function applyPreferences(next: KiCadPreferences) {
		projectRoots = [...next.projectRoots];
		newRoot = '';
	}

	async function loadPreferences() {
		loading = true;
		error = '';
		success = '';
		try {
			applyPreferences(await getKiCadPreferences());
		} catch (err: any) {
			error = err?.message ?? String(err);
		} finally {
			loading = false;
		}
	}

	function addRoot() {
		const trimmed = newRoot.trim();
		if (!trimmed) {
			error = 'Choose or enter a folder before adding it.';
			success = '';
			return;
		}
		if (projectRoots.includes(trimmed)) {
			error = 'That folder is already in the KiCad root list.';
			success = '';
			return;
		}
		projectRoots = [...projectRoots, trimmed];
		newRoot = '';
		error = '';
		success = '';
	}

	function removeRoot(root: string) {
		projectRoots = projectRoots.filter((value) => value !== root);
		success = '';
	}

	async function handlePickRoot() {
		picking = true;
		error = '';
		success = '';
		try {
			const startDir = newRoot.trim() || projectRoots.at(-1) || '';
			const selected = (await pickDirectory(startDir)).trim();
			if (!selected) {
				return;
			}
			newRoot = selected;
			error = '';
		} catch (err: any) {
			error = err?.message ?? String(err);
		} finally {
			picking = false;
		}
	}

	async function handleSave() {
		saving = true;
		error = '';
		success = '';
		try {
			const next = await saveKiCadPreferences({ projectRoots });
			applyPreferences(next);
			success = 'KiCad integration settings saved.';
		} catch (err: any) {
			error = err?.message ?? String(err);
		} finally {
			saving = false;
		}
	}
</script>

{#if loading}
	<div class="page-empty">Loading KiCad integration settings…</div>
{:else}
	<div class="settings-page">
		<section class="settings-section">
			<div class="section-header">
				<div>
					<h2>Project discovery roots</h2>
					<p>These folders are scanned automatically when the KiCad importer opens from the launcher.</p>
				</div>
			</div>

			<div class="settings-copy">
				Set one or more root directories that contain your KiCad projects. Trace will look for .kicad_pro files anywhere under these roots.
			</div>

			<div class="field-block">
				<div class="field-header">
					<label for="kicad-root-input">Staged root</label>
					<p>Browse to a folder or paste a path, then add it to the saved list.</p>
				</div>

				<div class="root-editor">
					<input
						id="kicad-root-input"
						class="form-input"
						bind:value={newRoot}
						placeholder="/path/to/kicad/projects"
						onkeydown={(event) => {
							if (event.key === 'Enter') {
								event.preventDefault();
								addRoot();
							}
						}}
					/>
					<button type="button" class="btn btn-secondary btn-sm" onclick={() => void handlePickRoot()} disabled={picking}>
						{picking ? 'Opening…' : 'Browse…'}
					</button>
					<button type="button" class="btn btn-primary btn-sm" onclick={addRoot}>Add Root</button>
				</div>
			</div>

			<div class="root-list">
				{#if projectRoots.length === 0}
					<div class="empty-block">No default KiCad roots configured yet.</div>
				{:else}
					{#each projectRoots as root}
						<div class="root-chip">
							<span>{root}</span>
							<button type="button" class="chip-remove" onclick={() => removeRoot(root)} aria-label={`Remove ${root}`}>
								×
							</button>
						</div>
					{/each}
				{/if}
			</div>
		</section>

		{#if error}
			<div class="feedback feedback-error">{error}</div>
		{/if}
		{#if success}
			<div class="feedback feedback-success">{success}</div>
		{/if}

		<div class="page-actions">
			<button type="button" class="btn btn-secondary" onclick={() => void loadPreferences()} disabled={saving}>
				Reload
			</button>
			<button type="button" class="btn btn-primary" onclick={() => void handleSave()} disabled={saving || picking}>
				{saving ? 'Saving…' : 'Save & Apply'}
			</button>
		</div>
	</div>
{/if}

<style>
	.settings-page {
		display: flex;
		flex-direction: column;
		gap: 12px;
	}

	.page-empty {
		padding: 28px 0;
		color: var(--color-text-muted);
	}

	.settings-section {
		display: flex;
		flex-direction: column;
		gap: 14px;
		border: 1px solid var(--color-border);
		border-radius: var(--radius-lg);
		background: var(--color-bg-surface);
		padding: 14px 16px;
	}

	.section-header h2 {
		font-size: 14px;
		font-weight: 600;
		margin-bottom: 4px;
	}

	.section-header p,
	.settings-copy {
		color: var(--color-text-secondary);
		line-height: 1.5;
	}

	.field-block {
		display: flex;
		flex-direction: column;
		gap: 10px;
	}

	.field-header {
		display: flex;
		flex-direction: column;
		gap: 4px;
	}

	.field-header label {
		font-size: 12px;
		font-weight: 600;
		color: var(--color-text-primary);
	}

	.field-header p {
		color: var(--color-text-secondary);
		font-size: 12px;
		line-height: 1.45;
	}

	.root-editor {
		display: flex;
		gap: 8px;
	}

	.root-editor .form-input {
		flex: 1;
	}

	.root-list {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
	}

	.root-chip {
		display: inline-flex;
		align-items: center;
		gap: 8px;
		padding: 6px 10px;
		border-radius: 999px;
		border: 1px solid var(--color-border);
		background: var(--color-bg-muted);
		max-width: 100%;
	}

	.root-chip span {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.chip-remove {
		color: var(--color-text-muted);
	}

	.empty-block {
		padding: 14px;
		border: 1px dashed var(--color-border);
		border-radius: var(--radius-md);
		color: var(--color-text-muted);
		background: var(--color-bg-app);
	}

	.feedback {
		padding: 12px 14px;
		border-radius: var(--radius-md);
		border: 1px solid var(--color-border);
	}

	.feedback-error {
		border-color: var(--color-danger-border);
		background: var(--color-danger-soft);
		color: var(--color-danger-text);
	}

	.feedback-success {
		border-color: var(--color-success-border);
		background: var(--color-success-soft);
		color: var(--color-success-text);
	}

	.page-actions {
		display: flex;
		justify-content: flex-end;
		gap: 8px;
	}

	@media (max-width: 720px) {
		.root-editor,
		.page-actions {
			flex-direction: column;
		}
	}
</style>