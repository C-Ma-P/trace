<script lang="ts">
  import {
    planProject,
    sourceRequirement,
    clearSelectedComponentForRequirement,
    addPartCandidate,
    setPreferredCandidate,
    setPreferredLocalComponent,
    removePartCandidate,
    addProviderCandidate,
    categoryDisplayName,
    type Project,
    type ProjectPlan,
    type RequirementPlan,
    type RequirementSelectedPart,
    type SourceRequirementResult,
    type SupplierOffer as SupplierOfferType,
    type SupplierProviderStatus,
    type CategoryInfo,
    type PartCandidate,
  } from '../backend';
  import { Browser } from '@wailsio/runtime';

  let { project, categories = [], onupdated }: {
    project: Project;
    categories?: CategoryInfo[];
    onupdated?: () => void;
  } = $props();

  let plan: ProjectPlan | null = $state(null);
  let loading = $state(false);
  let error = $state('');
  let expandedReq: string | null = $state(null);
  let supplierResultsByRequirementId = $state<Record<string, SourceRequirementResult | undefined>>({});
  let supplierLoadingByRequirementId = $state<Record<string, boolean>>({});
  let supplierErrorByRequirementId = $state<Record<string, string>>({});
  let actionInProgress = $state<Record<string, boolean>>({});

  $effect(() => {
    if (project) {
      runPlan();
    }
  });

  async function runPlan() {
    loading = true;
    error = '';
    try {
      plan = await planProject(project.id);
    } catch (e: any) {
      error = e?.message ?? String(e);
    } finally {
      loading = false;
    }
  }

  function toggleExpand(reqId: string) {
    expandedReq = expandedReq === reqId ? null : reqId;
  }

  async function loadSupplierOptions(requirementId: string, force = false) {
    if (!force && supplierResultsByRequirementId[requirementId]) {
      return;
    }

    supplierLoadingByRequirementId = {
      ...supplierLoadingByRequirementId,
      [requirementId]: true,
    };
    supplierErrorByRequirementId = {
      ...supplierErrorByRequirementId,
      [requirementId]: '',
    };

    try {
      const result = await sourceRequirement(requirementId);
      supplierResultsByRequirementId = {
        ...supplierResultsByRequirementId,
        [requirementId]: result,
      };
    } catch (e: any) {
      supplierErrorByRequirementId = {
        ...supplierErrorByRequirementId,
        [requirementId]: e?.message ?? String(e),
      };
    } finally {
      supplierLoadingByRequirementId = {
        ...supplierLoadingByRequirementId,
        [requirementId]: false,
      };
    }
  }

  async function handleAddCandidate(requirementId: string, componentId: string) {
    error = '';
    try {
      await addPartCandidate(requirementId, componentId);
      await runPlan();
    } catch (e: any) {
      error = e?.message ?? String(e);
    }
  }

  async function handleSetPreferred(requirementId: string, componentId: string) {
    error = '';
    try {
      await setPreferredLocalComponent(requirementId, componentId);
      await runPlan();
      onupdated?.();
    } catch (e: any) {
      error = e?.message ?? String(e);
    }
  }

  async function handleSetCandidatePreferred(requirementId: string, candidateId: string) {
    error = '';
    try {
      await setPreferredCandidate(requirementId, candidateId);
      await runPlan();
      onupdated?.();
    } catch (e: any) {
      error = e?.message ?? String(e);
    }
  }

  async function handleRemoveCandidate(candidateId: string, isPreferred: boolean) {
    if (isPreferred) {
      const confirmed = confirm('This is the preferred candidate. Removing it will also clear the current engineering choice for this requirement. Continue?');
      if (!confirmed) return;
    }
    error = '';
    try {
      await removePartCandidate(candidateId);
      await runPlan();
      if (isPreferred) onupdated?.();
    } catch (e: any) {
      error = e?.message ?? String(e);
    }
  }

  async function handleClearSelection(requirementId: string) {
    error = '';
    try {
      await clearSelectedComponentForRequirement(requirementId);
      await runPlan();
      onupdated?.();
    } catch (e: any) {
      error = e?.message ?? String(e);
    }
  }

  function openUrl(url: string) {
    if (url) {
      Browser.OpenURL(url);
    }
  }

  async function handleAddProviderCandidate(requirementId: string, offer: SupplierOfferType, currency: string, setPreferred: boolean) {
    const key = `provider-${requirementId}-${offer.supplierPartNumber}`;
    if (actionInProgress[key]) return;
    actionInProgress = { ...actionInProgress, [key]: true };
    error = '';
    try {
      await addProviderCandidate({
        requirementId,
        provider: offer.provider,
        providerPartId: offer.supplierPartNumber,
        productUrl: offer.productUrl,
        imageUrl: offer.imageUrl,
        manufacturer: offer.manufacturer,
        mpn: offer.mpn,
        description: offer.description,
        package: offer.package,
        stock: offer.stock,
        moq: offer.moq,
        unitPrice: offer.unitPrice,
        currency,
        setPreferred,
      });
      await runPlan();
      if (setPreferred) onupdated?.();
    } catch (e: any) {
      error = e?.message ?? String(e);
    } finally {
      actionInProgress = { ...actionInProgress, [key]: false };
    }
  }

  function statusBadge(rp: RequirementPlan): { class: string; text: string } {
    const preferred = rp.candidates.find(c => c.preferred);
    if (rp.selectedPart) {
      if (rp.selectedPart.shortfallQuantity > 0) {
        return { class: 'badge-warning', text: `Shortfall ${rp.selectedPart.shortfallQuantity}` };
      }
      return { class: 'badge-success', text: 'Resolved' };
    }
    if (preferred) {
      if (preferred.origin === 'provider') {
        return { class: 'badge-warning', text: 'Provider (not imported)' };
      }
      return { class: 'badge-success', text: 'Resolved' };
    }
    if (rp.shortfallQuantity > 0) {
      return { class: 'badge-danger', text: 'Needs part' };
    }
    return { class: 'badge-success', text: 'On hand matches' };
  }

  function supplierQualityBadge(offer: SupplierOfferType): { class: string; text: string } {
    if (offer.matchScore >= 120) {
      return { class: 'badge-success', text: 'Strong' };
    }
    if (offer.matchScore >= 50) {
      return { class: 'badge-warning', text: 'Possible' };
    }
    return { class: 'badge', text: 'Weak' };
  }

  function providerStatusClass(status: SupplierProviderStatus): string {
    if (status.status === 'success') return 'provider-status-success';
    if (status.status === 'disabled') return 'provider-status-disabled';
    return 'provider-status-error';
  }

  function formatSupplierCount(value: number | null): string {
    if (value === null) return '—';
    return value.toLocaleString();
  }

  function formatSupplierPrice(offer: SupplierOfferType): string {
    if (offer.unitPrice === null) return '—';
    return offer.unitPrice < 1 ? offer.unitPrice.toFixed(4) : offer.unitPrice.toFixed(2);
  }

  function supplierResult(requirementId: string): SourceRequirementResult | null {
    return supplierResultsByRequirementId[requirementId] ?? null;
  }

  function offerMatchKey(provider: string, supplierPartNumber: string, mpn: string): string {
    // Use the supplier SKU when non-empty; fall back to provider+MPN.
    const sku = supplierPartNumber?.trim();
    return sku ? `${provider}::sku::${sku}` : `${provider}::mpn::${mpn}`;
  }

  function isAlreadyCandidate(rp: RequirementPlan, offer: SupplierOfferType): boolean {
    const key = offerMatchKey(offer.provider, offer.supplierPartNumber, offer.mpn);
    return rp.candidates.some(c =>
      c.sourceOffer &&
      offerMatchKey(c.sourceOffer.provider, c.sourceOffer.providerPartId, c.sourceOffer.mpn) === key
    );
  }

  function originLabel(origin: string): string {
    if (origin === 'provider') return 'Provider';
    if (origin === 'imported_from_supplier') return 'Imported';
    return 'Local';
  }

  function candidateDisplayMpn(c: PartCandidate): string {
    if (c.component?.mpn) return c.component.mpn;
    if (c.sourceOffer?.mpn) return c.sourceOffer.mpn;
    return '—';
  }

  function candidateDisplayManufacturer(c: PartCandidate): string {
    if (c.component?.manufacturer) return c.component.manufacturer;
    if (c.sourceOffer?.manufacturer) return c.sourceOffer.manufacturer;
    return '—';
  }

  function candidateDisplayPackage(c: PartCandidate): string {
    if (c.component?.package) return c.component.package;
    if (c.sourceOffer?.package) return c.sourceOffer.package;
    return '—';
  }

  function resolvedPartLabel(selectedPart: RequirementSelectedPart | null): string {
    if (!selectedPart) {
      return 'No resolved part definition';
    }
    return selectedPart.displayName || selectedPart.component?.id || 'Resolved part';
  }
</script>

<div class="plan-tab">
  <div class="section-header">
    <h3 class="section-title">Project Plan</h3>
    <button class="btn btn-secondary btn-sm" onclick={runPlan} disabled={loading}>
      {loading ? 'Planning…' : 'Refresh'}
    </button>
  </div>

  {#if error}
    <div class="error-text" style="margin-bottom: 12px;">{error}</div>
  {/if}

  {#if loading && !plan}
    <div class="empty-msg">Running plan…</div>
  {:else if plan && plan.requirements.length === 0}
    <div class="empty-msg">No requirements to plan</div>
  {:else if plan}
    <div class="plan-list">
      {#each plan.requirements as rp}
        {@const status = statusBadge(rp)}
        <div class="plan-card">
          <button class="plan-card-header" onclick={() => toggleExpand(rp.requirement.id)}>
            <div class="plan-header-left">
              <span class="plan-name">{rp.requirement.name || 'Unnamed'}</span>
              <span class="badge">{categoryDisplayName(categories, rp.requirement.category)}</span>
              <span class="badge {status.class}">{status.text}</span>
              {#if rp.candidates.length > 0}
                <span class="badge">{rp.candidates.length} candidate{rp.candidates.length === 1 ? '' : 's'}</span>
              {/if}
            </div>
            <div class="plan-header-right">
              <span class="plan-qty">
                {rp.matchingOnHandQuantity} on hand across matches
              </span>
              <span class="expand-icon">{expandedReq === rp.requirement.id ? '▾' : '▸'}</span>
            </div>
          </button>

          {#if rp.selectedPart}
            <div class="selected-banner">
              <div class="selected-banner-copy">
                <span class="selected-banner-label">Preferred part</span>
                <strong>{resolvedPartLabel(rp.selectedPart)}</strong>
                <span class="selected-banner-meta">
                  On hand {rp.selectedPart.onHandQuantity} / required {rp.requirement.quantity}
                </span>
              </div>
              <button
                class="btn btn-ghost btn-sm"
                onclick={() => handleClearSelection(rp.requirement.id)}
              >
                Clear Preferred
              </button>
            </div>
          {:else if rp.candidates.some(c => c.preferred)}
            {@const preferred = rp.candidates.find(c => c.preferred)}
            {#if preferred}
              <div class="selected-banner">
                <div class="selected-banner-copy">
                  <span class="selected-banner-label">Preferred part <span class="origin-badge origin-provider">Provider</span></span>
                  <strong>{candidateDisplayMpn(preferred)} — {candidateDisplayManufacturer(preferred)}</strong>
                  <span class="selected-banner-meta">Not yet imported into catalog</span>
                </div>
                <button
                  class="btn btn-ghost btn-sm"
                  onclick={() => handleClearSelection(rp.requirement.id)}
                >
                  Clear Preferred
                </button>
              </div>
            {/if}
          {/if}

          {#if expandedReq === rp.requirement.id}
            <div class="expanded-sections">
              <!-- Candidates Section -->
              {#if rp.candidates.length > 0}
                <section class="plan-section">
                  <div class="subsection-header">
                    <h4>Part Candidates</h4>
                  </div>
                  <table class="match-table">
                    <thead>
                      <tr>
                        <th>MPN</th>
                        <th>Manufacturer</th>
                        <th>Package</th>
                        <th>Origin</th>
                        <th>Status</th>
                        <th></th>
                      </tr>
                    </thead>
                    <tbody>
                      {#each rp.candidates as candidate}
                        <tr class:selected-match={candidate.preferred}>
                          <td class="mpn-cell">{candidateDisplayMpn(candidate)}</td>
                          <td>{candidateDisplayManufacturer(candidate)}</td>
                          <td>{candidateDisplayPackage(candidate)}</td>
                          <td><span class="origin-badge origin-{candidate.origin}">{originLabel(candidate.origin)}</span></td>
                          <td>
                            {#if candidate.preferred}
                              <span class="badge badge-success">Preferred</span>
                            {:else}
                              <span class="badge">Candidate</span>
                            {/if}
                          </td>
                          <td class="action-cell">
                            {#if !candidate.preferred}
                              <button
                                class="btn btn-secondary btn-sm"
                                onclick={() => handleSetCandidatePreferred(rp.requirement.id, candidate.id)}
                              >
                                Set Preferred
                              </button>
                            {/if}
                            <button
                              class="btn btn-ghost btn-sm"
                              onclick={() => handleRemoveCandidate(candidate.id, candidate.preferred)}
                            >
                              Remove
                            </button>
                          </td>
                        </tr>
                      {/each}
                    </tbody>
                  </table>
                </section>
              {/if}

              <!-- Local Matches Section -->
              <section class="plan-section">
                <div class="subsection-header">
                  <h4>Local Matches</h4>
                  <span class="subsection-note">Components in catalog matching this requirement</span>
                </div>

                {#if rp.matches.length === 0}
                  <div class="empty-msg">No matching components</div>
                {:else}
                  <table class="match-table">
                    <thead>
                      <tr>
                        <th>MPN</th>
                        <th>Manufacturer</th>
                        <th>Package</th>
                        <th>On Hand</th>
                        <th>Score</th>
                        <th></th>
                      </tr>
                    </thead>
                    <tbody>
                      {#each rp.matches as match}
                        {@const isCandidate = rp.candidates.some(c => c.componentId === match.component.id)}
                        {@const isPreferred = rp.candidates.some(c => c.componentId === match.component.id && c.preferred)}
                        <tr class:selected-match={isPreferred}>
                          <td class="mpn-cell">{match.component.mpn || '—'}</td>
                          <td>{match.component.manufacturer || '—'}</td>
                          <td>{match.component.package || '—'}</td>
                          <td class="qty-cell">{match.onHandQuantity}</td>
                          <td class="score-cell">{match.score}</td>
                          <td class="action-cell">
                            {#if isPreferred}
                              <span class="badge badge-success">Preferred</span>
                            {:else}
                              <button
                                class="btn btn-primary btn-sm"
                                onclick={() => handleSetPreferred(rp.requirement.id, match.component.id)}
                              >
                                Set Preferred
                              </button>
                              {#if !isCandidate}
                                <button
                                  class="btn btn-secondary btn-sm"
                                  onclick={() => handleAddCandidate(rp.requirement.id, match.component.id)}
                                >
                                  Add Candidate
                                </button>
                              {:else}
                                <span class="badge">Candidate</span>
                              {/if}
                            {/if}
                          </td>
                        </tr>
                      {/each}
                    </tbody>
                  </table>
                {/if}
              </section>

              <!-- Supplier Options Section -->
              <section class="plan-section supplier-section">
                <div class="subsection-header">
                  <div>
                    <h4>Supplier Options</h4>
                    <span class="subsection-note">Live distributor search results</span>
                  </div>
                  <div class="supplier-actions">
                    <button
                      class="btn btn-secondary btn-sm"
                      onclick={() => loadSupplierOptions(rp.requirement.id, !!supplierResult(rp.requirement.id))}
                      disabled={supplierLoadingByRequirementId[rp.requirement.id]}
                    >
                      {#if supplierLoadingByRequirementId[rp.requirement.id]}
                        Searching…
                      {:else if supplierResult(rp.requirement.id)}
                        Refresh Supplier Options
                      {:else}
                        Find Supplier Options
                      {/if}
                    </button>
                  </div>
                </div>

                {#if supplierLoadingByRequirementId[rp.requirement.id]}
                  <div class="empty-msg">Finding supplier options…</div>
                {:else if supplierErrorByRequirementId[rp.requirement.id]}
                  <div class="error-text">{supplierErrorByRequirementId[rp.requirement.id]}</div>
                {:else if supplierResult(rp.requirement.id)}
                  {@const sourcingResult = supplierResult(rp.requirement.id)}

                  {#if sourcingResult && sourcingResult.providers.length > 0}
                    <div class="provider-status-list">
                      {#each sourcingResult.providers as status}
                        <div class={`provider-status ${providerStatusClass(status)}`}>
                          <span class="provider-badge">{status.provider}</span>
                          {#if status.status === 'success'}
                            <span>{status.offerCount} offer{status.offerCount === 1 ? '' : 's'}</span>
                          {:else}
                            <span>{status.error}</span>
                          {/if}
                        </div>
                      {/each}
                    </div>
                  {/if}

                  {#if sourcingResult && sourcingResult.offers.length === 0}
                    <div class="empty-msg">No supplier options found</div>
                  {:else if sourcingResult}
                    <table class="match-table supplier-table">
                      <thead>
                        <tr>
                          <th></th>
                          <th>Provider</th>
                          <th>MPN</th>
                          <th>Manufacturer</th>
                          <th>Supplier SKU</th>
                          <th>Package</th>
                          <th>Stock</th>
                          <th>MOQ</th>
                          <th>Unit Price ({sourcingResult.currency || 'USD'})</th>
                          <th>Match</th>
                          <th></th>
                        </tr>
                      </thead>
                      <tbody>
                        {#each sourcingResult.offers as offer}
                          {@const quality = supplierQualityBadge(offer)}
                          {@const alreadyCandidate = isAlreadyCandidate(rp, offer)}
                          <tr>
                            <td class="img-cell">{#if offer.imageUrl}<img src={offer.imageUrl} alt="" class="offer-thumb" />{/if}</td>
                            <td><span class="provider-badge">{offer.provider}</span></td>
                            <td class="mpn-cell">{offer.mpn || '—'}</td>
                            <td>{offer.manufacturer || '—'}</td>
                            <td>{offer.supplierPartNumber || '—'}</td>
                            <td>{offer.package || '—'}</td>
                            <td class="qty-cell">{formatSupplierCount(offer.stock)}</td>
                            <td class="qty-cell">{formatSupplierCount(offer.moq)}</td>
                            <td>{formatSupplierPrice(offer)}</td>
                            <td>
                              <div class="supplier-match-cell">
                                <span
                                  class={`badge ${quality.class}`}
                                  title={offer.matchReasons.length > 0 ? offer.matchReasons.join(' • ') : undefined}
                                >
                                  {quality.text}
                                </span>
                              </div>
                            </td>
                            <td class="action-cell">
                              {#if alreadyCandidate}
                                <span class="badge">Candidate</span>
                              {:else}
                                <button
                                  class="btn btn-primary btn-sm"
                                  onclick={() => handleAddProviderCandidate(rp.requirement.id, offer, sourcingResult.currency, true)}
                                >
                                  Add + Set Preferred
                                </button>
                                <button
                                  class="btn btn-secondary btn-sm"
                                  onclick={() => handleAddProviderCandidate(rp.requirement.id, offer, sourcingResult.currency, false)}
                                >
                                  Add as Candidate
                                </button>
                              {/if}
                            </td>
                          </tr>
                        {/each}
                      </tbody>
                    </table>
                  {/if}
                {:else}
                  <div class="empty-msg">Run supplier sourcing to fetch live distributor options for this requirement.</div>
                {/if}
              </section>
            </div>
          {/if}
        </div>
      {/each}
    </div>
  {/if}
</div>

<style>
  .plan-tab {
    padding: 20px;
  }
  .section-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 16px;
  }
  .section-title {
    font-size: 14px;
    font-weight: 600;
  }
  .empty-msg {
    color: var(--color-text-muted);
    font-size: 13px;
    padding: 12px 0;
  }
  .plan-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  .plan-card {
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    background: var(--color-bg-surface);
    overflow: hidden;
  }
  .plan-card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    width: 100%;
    padding: 12px 16px;
    text-align: left;
    transition: background 0.1s;
  }
  .plan-card-header:hover {
    background: var(--color-bg-hover);
  }
  .plan-header-left {
    display: flex;
    align-items: center;
    gap: 8px;
  }
  .plan-name {
    font-weight: 600;
    font-size: 13px;
  }
  .plan-header-right {
    display: flex;
    align-items: center;
    gap: 12px;
    font-size: 12px;
    color: var(--color-text-secondary);
  }
  .plan-qty {
    font-variant-numeric: tabular-nums;
  }
  .expand-icon {
    color: var(--color-text-muted);
    font-size: 11px;
  }
  .selected-banner {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    padding: 10px 16px;
    background: var(--color-accent-soft);
    border-top: 1px solid rgba(255, 255, 255, 0.04);
    font-size: 12px;
  }
  .selected-banner-copy {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }
  .selected-banner-label {
    font-size: 10px;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    color: var(--color-text-muted);
  }
  .selected-banner-meta {
    color: var(--color-text-secondary);
    font-size: 12px;
  }
  .expanded-sections {
    padding: 12px 16px;
    border-top: 1px solid var(--color-border);
    background: var(--color-bg-muted);
    display: flex;
    flex-direction: column;
    gap: 16px;
  }
  .plan-section {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }
  .resolution-card {
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    background: var(--color-bg-surface);
    padding: 14px;
  }
  .resolution-grid {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 12px;
  }
  .resolution-label {
    display: block;
    margin-bottom: 6px;
    font-size: 10px;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    color: var(--color-text-muted);
  }
  .subsection-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
  }
  .subsection-header h4 {
    margin: 0;
    font-size: 12px;
    font-weight: 600;
  }
  .subsection-note {
    font-size: 11px;
    color: var(--color-text-muted);
  }
  .supplier-actions {
    display: flex;
    gap: 8px;
    align-items: center;
    flex-wrap: wrap;
  }
  .provider-status-list {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
  }
  .provider-status {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    padding: 6px 10px;
    border-radius: var(--radius-sm);
    font-size: 11px;
    border: 1px solid var(--color-border);
    background: var(--color-bg-surface);
  }
  .provider-status-success {
    border-color: var(--color-success-border);
    color: var(--color-success-text);
    background: var(--color-success-soft);
  }
  .provider-status-disabled {
    color: var(--color-text-muted);
  }
  .provider-status-error {
    border-color: var(--color-danger-border);
    color: var(--color-danger-text);
    background: var(--color-danger-soft);
  }
  .provider-badge {
    display: inline-flex;
    align-items: center;
    padding: 2px 6px;
    border-radius: 999px;
    background: var(--color-bg-muted);
    font-size: 10px;
    font-weight: 600;
    letter-spacing: 0.04em;
    text-transform: uppercase;
  }
  .offer-thumb {
    width: 28px;
    height: 28px;
    object-fit: contain;
    border-radius: 3px;
    vertical-align: middle;
    transition: transform 160ms ease, box-shadow 160ms ease;
    cursor: pointer;
    transform-origin: left center;
  }
  .offer-thumb:hover {
    transform: scale(4);
    box-shadow: 0 10px 24px rgba(0, 0, 0, 0.2);
    position: relative;
    z-index: 999;
  }
  .img-cell {
    width: 32px;
    padding: 4px !important;
    text-align: center;
    overflow: visible;
    position: relative;
  }
  .match-table {
    width: 100%;
    border-collapse: collapse;
    font-size: 12px;
  }
  .match-table th {
    text-align: left;
    padding: 6px 10px;
    font-weight: 500;
    color: var(--color-text-secondary);
    font-size: 11px;
    border-bottom: 1px solid var(--color-border);
  }
  .match-table td {
    padding: 6px 10px;
    border-bottom: 1px solid var(--color-border);
  }
  .match-table tbody tr:hover {
    background: var(--color-bg-surface);
  }
  .selected-match {
    background: var(--color-success-soft) !important;
  }
  .mpn-cell {
    font-weight: 600;
  }
  .qty-cell {
    font-variant-numeric: tabular-nums;
    font-weight: 500;
  }
  .score-cell {
    color: var(--color-text-muted);
    font-variant-numeric: tabular-nums;
  }
  .supplier-table td {
    vertical-align: top;
  }
  .supplier-match-cell {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  .supplier-reasons {
    max-width: 280px;
    color: var(--color-text-muted);
    font-size: 11px;
    line-height: 1.35;
  }
  .supplier-link {
    color: var(--color-accent-text);
    text-decoration: none;
    font-weight: 500;
  }
  .supplier-link:hover {
    text-decoration: underline;
  }
  .muted-cell {
    color: var(--color-text-muted);
  }
  .action-cell {
    display: flex;
    gap: 6px;
    align-items: center;
    flex-wrap: wrap;
  }
  .origin-badge {
    display: inline-flex;
    align-items: center;
    padding: 2px 6px;
    border-radius: 999px;
    background: var(--color-bg-muted);
    font-size: 10px;
    font-weight: 500;
    letter-spacing: 0.04em;
  }
  .origin-provider {
    background: var(--color-warning-soft);
    color: var(--color-warning-text);
  }
  .origin-imported_from_supplier {
    background: var(--color-success-soft);
    color: var(--color-success-text);
  }
  @media (max-width: 980px) {
    .resolution-grid {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }
  }
  @media (max-width: 720px) {
    .selected-banner,
    .subsection-header {
      flex-direction: column;
      align-items: stretch;
    }
    .resolution-grid {
      grid-template-columns: 1fr;
    }
  }
</style>
