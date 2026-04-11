<script lang="ts">
	import { onMount } from 'svelte';
	import {
		clearSupplierSecret,
		getSupplierPreferences,
		saveSupplierPreferences,
		type SupplierPreferences,
		type SupplierProviderConfig,
	} from '../backend';
	import SupplierProviderBlock from './SupplierProviderBlock.svelte';

	type SupplierSection = 'overview' | 'digikey' | 'mouser' | 'lcsc';

	let { section = 'overview' }: { section?: SupplierSection } = $props();

	let loading = $state(true);
	let saving = $state(false);
	let clearingSecret = $state('');
	let error = $state('');
	let success = $state('');

	let secureStorageAvailable = $state(false);
	let secureStorageMessage = $state('');

	let digikeyEnabled = $state(true);
	let digikeyClientId = $state('');
	let digikeyCustomerId = $state('');
	let digikeySite = $state('');
	let digikeyLanguage = $state('');
	let digikeyCurrency = $state('');
	let digikeySecret = $state('');

	let mouserEnabled = $state(true);
	let mouserApiKey = $state('');

	let lcscEnabled = $state(true);
	let lcscCurrency = $state('');

	let prefs = $state<SupplierPreferences | null>(null);

	onMount(async () => {
		await loadPreferences();
	});

	async function loadPreferences() {
		loading = true;
		error = '';
		success = '';
		try {
			applyPreferences(await getSupplierPreferences());
		} catch (err: any) {
			error = err?.message ?? String(err);
		} finally {
			loading = false;
		}
	}

	function applyPreferences(next: SupplierPreferences) {
		prefs = next;
		secureStorageAvailable = next.secureStorageAvailable;
		secureStorageMessage = next.secureStorageMessage;
		digikeyEnabled = next.digikey.enabled;
		digikeyClientId = next.digikey.clientId;
		digikeyCustomerId = next.digikey.customerId;
		digikeySite = next.digikey.site;
		digikeyLanguage = next.digikey.language;
		digikeyCurrency = next.digikey.currency;
		digikeySecret = '';
		mouserEnabled = next.mouser.enabled;
		mouserApiKey = '';
		lcscEnabled = next.lcsc.enabled;
		lcscCurrency = next.lcsc.currency;
	}

	async function handleSave() {
		saving = true;
		error = '';
		success = '';
		try {
			const next = await saveSupplierPreferences({
				digikey: {
					enabled: digikeyEnabled,
					clientId: digikeyClientId,
					customerId: digikeyCustomerId,
					site: digikeySite,
					language: digikeyLanguage,
					currency: digikeyCurrency,
					replaceClientSecret: digikeySecret.trim() ? digikeySecret : null,
				},
				mouser: {
					enabled: mouserEnabled,
					replaceApiKey: mouserApiKey.trim() ? mouserApiKey : null,
				},
				lcsc: {
					enabled: lcscEnabled,
					currency: lcscCurrency,
				},
			});
			applyPreferences(next);
			success = 'Supplier settings saved and applied.';
		} catch (err: any) {
			error = err?.message ?? String(err);
		} finally {
			saving = false;
		}
	}

	async function handleClearSecret(provider: string, secret: string) {
		clearingSecret = `${provider}:${secret}`;
		error = '';
		success = '';
		try {
			const next = await clearSupplierSecret(provider, secret);
			applyPreferences(next);
			success = 'Stored secret cleared.';
		} catch (err: any) {
			error = err?.message ?? String(err);
		} finally {
			clearingSecret = '';
		}
	}

	function storageLabel(status: SupplierProviderConfig): string {
		switch (status.storageMode) {
			case 'keychain':
				return 'System keychain';
			case 'environment':
				return 'Environment fallback';
			case 'unavailable':
				return 'Unavailable';
			case 'none':
				return 'No secret required';
			default:
				return 'Missing';
		}
	}

	function sourceLabel(status: SupplierProviderConfig): string {
		switch (status.source) {
			case 'preferences':
				return 'Saved settings';
			case 'environment':
				return 'Environment';
			case 'mixed':
				return 'Mixed';
			default:
				return 'Missing';
		}
	}

	function secretLabel(status: SupplierProviderConfig, stored: boolean): string {
		if (status.storageMode === 'none') {
			return 'Not needed';
		}
		if (stored) {
			return 'Stored';
		}
		if (status.hasSecret) {
			return 'Available from environment';
		}
		return 'Missing';
	}

	const providerRows = $derived.by(() => {
		if (!prefs) {
			return [];
		}

		return [
			{
				id: 'digikey',
				name: 'DigiKey',
				detail: 'OAuth credentials, locale defaults, and secure client-secret storage',
				status: prefs.digikey.status,
				storage: storageLabel(prefs.digikey.status),
				secret: secretLabel(prefs.digikey.status, prefs.digikey.clientSecretStored),
			},
			{
				id: 'mouser',
				name: 'Mouser',
				detail: 'Single API key stored in secure storage when available',
				status: prefs.mouser.status,
				storage: storageLabel(prefs.mouser.status),
				secret: secretLabel(prefs.mouser.status, prefs.mouser.apiKeyStored),
			},
			{
				id: 'lcsc',
				name: 'LCSC',
				detail: 'Public provider settings with no secret material',
				status: prefs.lcsc.status,
				storage: storageLabel(prefs.lcsc.status),
				secret: secretLabel(prefs.lcsc.status, false),
			},
		];
	});
</script>

{#if loading}
	<div class="page-empty">Loading supplier settings…</div>
{:else}
	<div class="settings-page">
		<section class="settings-section section-banner" class:section-warning={!secureStorageAvailable}>
			<div class="section-header">
				<div>
					<h2>Credential storage</h2>
					<p>{secureStorageMessage}</p>
				</div>
				<span class={secureStorageAvailable ? 'badge badge-success' : 'badge badge-warning'}>
					{secureStorageAvailable ? 'Keychain ready' : 'Secure storage unavailable'}
				</span>
			</div>
		</section>

		{#if error}
			<div class="feedback feedback-error">{error}</div>
		{/if}
		{#if success}
			<div class="feedback feedback-success">{success}</div>
		{/if}

		{#if section === 'overview'}
			<section class="settings-section">
				<div class="section-header">
					<div>
						<h2>Supplier providers</h2>
						<p>Provider status and storage behavior. Use the navigator to edit one provider at a time.</p>
					</div>
				</div>

				<div class="provider-summary-list" role="list">
					{#each providerRows as provider (provider.id)}
						<div class="provider-summary-row" role="listitem">
							<div class="provider-summary-main">
								<div class="provider-summary-title">
									<strong>{provider.name}</strong>
									<span class={provider.status.state === 'configured' ? 'badge badge-success' : provider.status.state === 'incomplete' ? 'badge badge-warning' : 'badge'}>
										{provider.status.state === 'configured' ? 'Configured' : provider.status.state === 'incomplete' ? 'Incomplete' : 'Disabled'}
									</span>
								</div>
								<p>{provider.detail}</p>
							</div>
							<div class="provider-summary-meta">
								<div>
									<span class="meta-label">Storage</span>
									<span>{provider.storage}</span>
								</div>
								<div>
									<span class="meta-label">Secret</span>
									<span>{provider.secret}</span>
								</div>
							</div>
						</div>
					{/each}
				</div>
			</section>
		{:else if section === 'digikey'}
			<SupplierProviderBlock
				title="DigiKey"
				description="Procurement access for DigiKey searches. Client ID stays in the app database; the client secret stays in secure storage when available."
				status={prefs?.digikey.status ?? null}
				bind:enabled={digikeyEnabled}
				storageText={prefs ? storageLabel(prefs.digikey.status) : 'Loading'}
				sourceText={prefs ? sourceLabel(prefs.digikey.status) : 'Loading'}
				secretText={prefs ? secretLabel(prefs.digikey.status, prefs.digikey.clientSecretStored) : 'Loading'}
				message={prefs?.digikey.status.message ?? ''}
			>
				<div class="settings-grid two-column">
					<div class="form-group">
						<label for="digikey-client-id">Client ID</label>
						<input id="digikey-client-id" class="form-input" type="text" bind:value={digikeyClientId} placeholder="Client ID" />
					</div>
					<div class="form-group">
						<label for="digikey-customer-id">Customer ID</label>
						<input id="digikey-customer-id" class="form-input" type="text" bind:value={digikeyCustomerId} placeholder="Optional customer ID" />
					</div>
					<div class="form-group">
						<label for="digikey-site">Site</label>
						<input id="digikey-site" class="form-input" type="text" bind:value={digikeySite} placeholder="Optional site override" />
					</div>
					<div class="form-group">
						<label for="digikey-language">Language</label>
						<input id="digikey-language" class="form-input" type="text" bind:value={digikeyLanguage} placeholder="Optional locale" />
					</div>
					<div class="form-group">
						<label for="digikey-currency">Currency</label>
						<input id="digikey-currency" class="form-input" type="text" bind:value={digikeyCurrency} placeholder="Optional currency" />
					</div>
					<div class="form-group">
						<label for="digikey-client-secret">Client Secret</label>
						<input
							id="digikey-client-secret"
							class="form-input"
							type="password"
							bind:value={digikeySecret}
							placeholder={secureStorageAvailable ? 'Replace stored secret' : 'Requires system credential storage'}
							disabled={!secureStorageAvailable}
						/>
					</div>
				</div>

				<div class="inline-actions">
					<button
						type="button"
						class="btn btn-secondary btn-sm"
						disabled={!secureStorageAvailable || !prefs?.digikey.clientSecretStored || clearingSecret === 'digikey:client_secret'}
						onclick={() => void handleClearSecret('digikey', 'client_secret')}
					>
						{clearingSecret === 'digikey:client_secret' ? 'Clearing…' : 'Clear Stored Credentials'}
					</button>
				</div>
			</SupplierProviderBlock>
		{:else if section === 'mouser'}
			<SupplierProviderBlock
				title="Mouser"
				description="Mouser uses a single API key. Trace stores the key in secure storage and never returns it to the UI."
				status={prefs?.mouser.status ?? null}
				bind:enabled={mouserEnabled}
				storageText={prefs ? storageLabel(prefs.mouser.status) : 'Loading'}
				sourceText={prefs ? sourceLabel(prefs.mouser.status) : 'Loading'}
				secretText={prefs ? secretLabel(prefs.mouser.status, prefs.mouser.apiKeyStored) : 'Loading'}
				message={prefs?.mouser.status.message ?? ''}
			>
				<div class="settings-grid">
					<div class="form-group">
						<label for="mouser-api-key">API Key</label>
						<input
							id="mouser-api-key"
							class="form-input"
							type="password"
							bind:value={mouserApiKey}
							placeholder={secureStorageAvailable ? 'Replace stored API key' : 'Requires system credential storage'}
							disabled={!secureStorageAvailable}
						/>
					</div>
				</div>

				<div class="inline-actions">
					<button
						type="button"
						class="btn btn-secondary btn-sm"
						disabled={!secureStorageAvailable || !prefs?.mouser.apiKeyStored || clearingSecret === 'mouser:api_key'}
						onclick={() => void handleClearSecret('mouser', 'api_key')}
					>
						{clearingSecret === 'mouser:api_key' ? 'Clearing…' : 'Clear Stored Key'}
					</button>
				</div>
			</SupplierProviderBlock>
		{:else if section === 'lcsc'}
			<SupplierProviderBlock
				title="LCSC"
				description="LCSC currently uses only public settings. This block establishes the same settings surface without introducing secret handling that the provider does not need."
				status={prefs?.lcsc.status ?? null}
				bind:enabled={lcscEnabled}
				storageText={prefs ? storageLabel(prefs.lcsc.status) : 'Loading'}
				sourceText={prefs ? sourceLabel(prefs.lcsc.status) : 'Loading'}
				secretText={prefs ? secretLabel(prefs.lcsc.status, false) : 'Loading'}
				message={prefs?.lcsc.status.message ?? ''}
			>
				<div class="settings-grid">
					<div class="form-group">
						<label for="lcsc-currency">Currency</label>
						<input id="lcsc-currency" class="form-input" type="text" bind:value={lcscCurrency} placeholder="Optional currency override" />
					</div>
				</div>
			</SupplierProviderBlock>
		{/if}

		<div class="page-actions">
			<button type="button" class="btn btn-secondary" onclick={() => void loadPreferences()} disabled={saving}>
				Reload
			</button>
			<button type="button" class="btn btn-primary" onclick={() => void handleSave()} disabled={saving}>
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

	.section-banner {
		padding-bottom: 12px;
	}

	.section-warning {
		border-color: var(--color-warning-border);
	}

	.section-header {
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
		gap: 16px;
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

	.feedback {
		padding: 10px 12px;
		border-radius: var(--radius-lg);
		border: 1px solid transparent;
	}

	.feedback-error {
		color: var(--color-danger-text);
		background: var(--color-danger-soft);
		border-color: var(--color-danger-border);
	}

	.feedback-success {
		color: var(--color-success-text);
		background: var(--color-success-soft);
		border-color: var(--color-success-border);
	}

	.provider-summary-list {
		display: flex;
		flex-direction: column;
	}

	.provider-summary-row {
		display: grid;
		grid-template-columns: minmax(0, 1fr) 220px;
		gap: 18px;
		padding: 12px 0;
		border-top: 1px solid var(--color-border);
	}

	.provider-summary-row:first-child {
		border-top: none;
		padding-top: 0;
	}

	.provider-summary-main,
	.provider-summary-meta {
		display: flex;
		flex-direction: column;
		gap: 6px;
	}

	.provider-summary-title {
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.provider-summary-main p {
		color: var(--color-text-secondary);
		line-height: 1.45;
	}

	.provider-summary-meta {
		gap: 10px;
	}

	.provider-summary-meta div {
		display: flex;
		flex-direction: column;
		gap: 4px;
	}

	.settings-grid {
		display: grid;
		grid-template-columns: minmax(0, 1fr);
		gap: 12px;
	}

	.settings-grid.two-column {
		grid-template-columns: repeat(2, minmax(0, 1fr));
	}

	.inline-actions {
		display: flex;
		justify-content: flex-start;
	}

	.page-actions {
		display: flex;
		justify-content: flex-end;
		gap: 10px;
		padding-top: 2px;
	}

	.meta-label {
		font-size: 10px;
		letter-spacing: 0.12em;
		text-transform: uppercase;
		color: var(--color-text-muted);
	}

	@media (max-width: 720px) {
		.section-header,
		.provider-summary-row,
		.page-actions {
			flex-direction: column;
			align-items: stretch;
		}

		.provider-summary-row {
			display: flex;
		}

		.settings-grid.two-column {
			grid-template-columns: 1fr;
		}
	}
</style>