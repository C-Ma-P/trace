<script lang="ts">
  import {
    planProject,
    setPreferredCandidate,
    removePartCandidate,
    demotePreferredCandidate,
    importProviderCandidate,
    categoryDisplayName,
    type Project,
    type ProjectPlan,
    type RequirementPlan,
    type CategoryInfo,
    type PartCandidate,
    type SavedSupplierOffer,
    type ExportReadinessStatus,
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
  let expandedReqs = $state<Set<string>>(new Set());
  let actionInProgress = $state<Record<string, boolean>>({});

  $effect(() => {
    if (project) {
      loadPlan();
    }
  });

  async function loadPlan() {
    loading = true;
    error = '';
    try {
      plan = await planProject(project.id);
      // Auto-expand requirements that need attention
      if (plan) {
        const next = new Set<string>();
        for (const rp of plan.requirements) {
          const status = reqStatus(rp);
          if (status.level !== 'ok') {
            next.add(rp.requirement.id);
          }
        }
        // On first load, expand those needing attention; after that, keep user's choice
        if (expandedReqs.size === 0 && next.size > 0) {
          expandedReqs = next;
        }
      }
    } catch (e: any) {
      error = e?.message ?? String(e);
    } finally {
      loading = false;
    }
  }

  function toggleExpand(reqId: string) {
    const next = new Set(expandedReqs);
    if (next.has(reqId)) {
      next.delete(reqId);
    } else {
      next.add(reqId);
    }
    expandedReqs = next;
  }

  // ---- Readiness summary ----

  interface ReadinessSummary {
    total: number;
    ready: number;
    missingPreferred: number;
    providerNotImported: number;
    missingSymbol: number;
    missingFootprint: number;
  }

  function computeReadiness(rps: RequirementPlan[]): ReadinessSummary {
    let total = 0, ready = 0, missingPreferred = 0, providerNotImported = 0, missingSymbol = 0, missingFootprint = 0;
    for (const rp of rps) {
      total++;
      switch (rp.readiness.status as ExportReadinessStatus) {
        case 'ready':
          ready++;
          break;
        case 'missing_preferred':
          missingPreferred++;
          break;
        case 'provider_not_imported':
          providerNotImported++;
          break;
        case 'missing_symbol':
          missingSymbol++;
          break;
        case 'missing_footprint':
          missingFootprint++;
          break;
      }
    }
    return { total, ready, missingPreferred, providerNotImported, missingSymbol, missingFootprint };
  }

  // ---- Per-requirement status ----

  interface ReqStatus {
    level: 'ok' | 'warn' | 'danger';
    warnings: string[];
  }

  function reqStatus(rp: RequirementPlan): ReqStatus {
    const readiness = rp.readiness;
    if (readiness.status === 'ready') {
      return { level: 'ok', warnings: [] };
    }
    const level = readiness.status === 'missing_preferred' ? 'danger' : 'warn';
    return { level, warnings: readiness.blockers ?? [] };
  }

  // ---- Actions ----

  async function handlePromoteCandidate(requirementId: string, candidateId: string) {
    const key = `promote-${candidateId}`;
    if (actionInProgress[key]) return;
    actionInProgress = { ...actionInProgress, [key]: true };
    error = '';
    try {
      await setPreferredCandidate(requirementId, candidateId);
      await loadPlan();
      onupdated?.();
    } catch (e: any) {
      error = e?.message ?? String(e);
    } finally {
      actionInProgress = { ...actionInProgress, [key]: false };
    }
  }

  async function handleDemoteCandidate(requirementId: string, candidateId: string) {
    const key = `demote-${candidateId}`;
    if (actionInProgress[key]) return;
    actionInProgress = { ...actionInProgress, [key]: true };
    error = '';
    try {
      await demotePreferredCandidate(requirementId, candidateId);
      await loadPlan();
      onupdated?.();
    } catch (e: any) {
      error = e?.message ?? String(e);
    } finally {
      actionInProgress = { ...actionInProgress, [key]: false };
    }
  }

  async function handleRemoveCandidate(candidateId: string, isPreferred: boolean) {
    if (isPreferred) {
      const confirmed = confirm('This is the preferred candidate. Removing it will also clear the current engineering choice for this requirement. Continue?');
      if (!confirmed) return;
    }
    const key = `remove-cand-${candidateId}`;
    if (actionInProgress[key]) return;
    actionInProgress = { ...actionInProgress, [key]: true };
    error = '';
    try {
      await removePartCandidate(candidateId);
      await loadPlan();
      onupdated?.();
    } catch (e: any) {
      error = e?.message ?? String(e);
    } finally {
      actionInProgress = { ...actionInProgress, [key]: false };
    }
  }

  async function handleImportCandidate(candidateId: string) {
    const key = `import-${candidateId}`;
    if (actionInProgress[key]) return;
    actionInProgress = { ...actionInProgress, [key]: true };
    error = '';
    try {
      await importProviderCandidate(candidateId);
      await loadPlan();
      onupdated?.();
    } catch (e: any) {
      error = e?.message ?? String(e);
    } finally {
      actionInProgress = { ...actionInProgress, [key]: false };
    }
  }

  function openUrl(url: string) {
    if (url) {
      Browser.OpenURL(url);
    }
  }

  function openComponent(componentId: string | null) {
    if (!componentId) return;
    window.location.href = `${window.location.pathname}?mode=components&componentId=${encodeURIComponent(
      componentId,
    )}`;
  }

  // ---- Helpers ----

  function formatPrice(offer: SavedSupplierOffer): string {
    if (offer.unitPrice === null) return '—';
    return offer.unitPrice < 1 ? offer.unitPrice.toFixed(4) : offer.unitPrice.toFixed(2);
  }

  function formatCount(value: number | null): string {
    if (value === null) return '—';
    return value.toLocaleString();
  }

  function candidateLabel(c: PartCandidate): string {
    if (c.component?.mpn) return c.component.mpn;
    if (c.sourceOffer?.mpn) return c.sourceOffer.mpn;
    if (c.component?.manufacturer) return c.component.manufacturer;
    if (c.sourceOffer?.manufacturer) return c.sourceOffer.manufacturer;
    if (c.componentId) return c.componentId.slice(0, 8);
    return c.id.slice(0, 8);
  }

  function candidateDisplayMpn(c: PartCandidate): string {
    return c.component?.mpn || c.sourceOffer?.mpn || '—';
  }

  function candidateDisplayManufacturer(c: PartCandidate): string {
    return c.component?.manufacturer || c.sourceOffer?.manufacturer || '—';
  }

  function candidateDisplayPackage(c: PartCandidate): string {
    return c.component?.package || c.sourceOffer?.package || '—';
  }

  function originLabel(origin: string): string {
    if (origin === 'provider') return 'Provider';
    if (origin === 'imported_from_supplier') return 'Imported';
    return 'Local';
  }

  function providerOrOriginLabel(c: PartCandidate): string {
    if (c.origin === 'provider' && c.sourceOffer?.provider) {
      return c.sourceOffer.provider;
    }
    return originLabel(c.origin);
  }

  function preferredCandidate(rp: RequirementPlan): PartCandidate | undefined {
    return rp.candidates.find(c => c.preferred);
  }

  function alternateCandidates(rp: RequirementPlan): PartCandidate[] {
    return rp.candidates.filter(c => !c.preferred);
  }

  function statusIcon(level: 'ok' | 'warn' | 'danger'): string {
    if (level === 'ok') return '✓';
    if (level === 'warn') return '⚠';
    return '✗';
  }
</script>

<div class="finalize-tab">
  <div class="section-header">
    <h3 class="section-title">Finalize</h3>
    <button class="btn btn-secondary btn-sm" onclick={loadPlan} disabled={loading}>
      {loading ? 'Loading…' : 'Refresh'}
    </button>
  </div>

  {#if error}
    <div class="error-text" style="margin-bottom: 12px;">{error}</div>
  {/if}

  {#if loading && !plan}
    <div class="empty-msg">Loading project data…</div>
  {:else if plan && plan.requirements.length === 0}
    <div class="empty-msg">No requirements defined. Add requirements in the Requirements tab first.</div>
  {:else if plan}
    <!-- Project Readiness Summary -->
    {@const readiness = computeReadiness(plan.requirements)}
    <div class="readiness-summary">
      <div class="readiness-row">
        <div class="readiness-stat">
          <span class="readiness-value">{readiness.total}</span>
          <span class="readiness-label">Requirements</span>
        </div>
        <div class="readiness-stat">
          <span class="readiness-value readiness-ok">{readiness.ready}</span>
          <span class="readiness-label">Export ready</span>
        </div>
        <div class="readiness-stat">
          <span class="readiness-value {readiness.missingPreferred > 0 ? 'readiness-danger' : 'readiness-ok'}">{readiness.missingPreferred}</span>
          <span class="readiness-label">Missing preferred</span>
        </div>
        <div class="readiness-stat">
          <span class="readiness-value {readiness.providerNotImported > 0 ? 'readiness-warn' : 'readiness-ok'}">{readiness.providerNotImported}</span>
          <span class="readiness-label">Provider (not imported)</span>
        </div>
        <div class="readiness-stat">
          <span class="readiness-value {readiness.missingSymbol > 0 ? 'readiness-warn' : 'readiness-ok'}">{readiness.missingSymbol}</span>
          <span class="readiness-label">Missing symbol</span>
        </div>
        <div class="readiness-stat">
          <span class="readiness-value {readiness.missingFootprint > 0 ? 'readiness-warn' : 'readiness-ok'}">{readiness.missingFootprint}</span>
          <span class="readiness-label">Missing footprint</span>
        </div>
      </div>

      <!-- Future export actions -->
      <div class="future-actions">
        <button class="btn btn-secondary btn-sm" disabled title="Coming soon: Export BOM from finalized requirements">Export BOM</button>
        <button class="btn btn-secondary btn-sm" disabled title="Coming soon: Generate KiCad project">KiCad Project</button>
      </div>
    </div>

    <!-- Per-requirement list -->
    <div class="req-list">
      {#each plan.requirements as rp}
        {@const status = reqStatus(rp)}
        {@const preferred = preferredCandidate(rp)}
        {@const alternates = alternateCandidates(rp)}

        <div class="req-card" class:req-card-ok={status.level === 'ok'} class:req-card-warn={status.level === 'warn'} class:req-card-danger={status.level === 'danger'}>
          <!-- Header row: always visible -->
          <button class="req-card-header" onclick={() => toggleExpand(rp.requirement.id)}>
            <div class="req-header-left">
              <span class="status-dot status-dot-{status.level}" title={status.warnings.join('; ') || 'OK'}>{statusIcon(status.level)}</span>
              <span class="req-name">{rp.requirement.name || 'Unnamed'}</span>
              <span class="badge">{categoryDisplayName(categories, rp.requirement.category)}</span>
              <span class="badge badge-qty">×{rp.requirement.quantity}</span>
            </div>
            <div class="req-header-right">
              {#if preferred}
                <span class="preferred-summary">
                  <span class="preferred-summary-label">Preferred:</span>
                  <strong>{candidateLabel(preferred)}</strong>
                </span>
              {:else}
                <span class="no-preferred-summary">No preferred part</span>
              {/if}
              <span class="expand-icon">{expandedReqs.has(rp.requirement.id) ? '▾' : '▸'}</span>
            </div>
          </button>

          <!-- Warnings strip -->
          {#if status.warnings.length > 0}
            <div class="warnings-strip">
              {#each status.warnings as w}
                <span class="warning-item">{w}</span>
              {/each}
            </div>
          {/if}

          {#if expandedReqs.has(rp.requirement.id)}
            <div class="req-expanded">
              <!-- Preferred Candidate Section -->
              <section class="finalize-section">
                <div class="subsection-header">
                  <h4>Preferred Part</h4>
                </div>
                {#if preferred}
                  <div class="preferred-card">
                    {#if preferred.sourceOffer?.imageUrl}
                      <img src={preferred.sourceOffer.imageUrl} alt="" class="offer-thumb-lg" />
                    {/if}
                    <div class="preferred-grid">
                      <div class="preferred-field">
                        <span class="field-label">MPN</span>
                        <span class="field-value mpn-cell">{candidateDisplayMpn(preferred)}</span>
                      </div>
                      <div class="preferred-field">
                        <span class="field-label">Manufacturer</span>
                        <span class="field-value">{candidateDisplayManufacturer(preferred)}</span>
                      </div>
                      <div class="preferred-field">
                        <span class="field-label">Package</span>
                        <span class="field-value">{candidateDisplayPackage(preferred)}</span>
                      </div>
                      <div class="preferred-field">
                        <span class="field-label">Origin</span>
                        <span class="field-value">
                          <span class="origin-badge origin-{preferred.origin}">{providerOrOriginLabel(preferred)}</span>
                        </span>
                      </div>
                      {#if preferred.sourceOffer}
                        <div class="preferred-field">
                          <span class="field-label">Supplier</span>
                          <span class="field-value"><span class="provider-badge">{preferred.sourceOffer.provider}</span></span>
                        </div>
                        <div class="preferred-field">
                          <span class="field-label">Available Assets</span>
                          <span class="field-value">
                            <div class="asset-badges">
                              {#if preferred.sourceOffer.hasSymbol}<span class="asset-badge">Symbol</span>{/if}
                              {#if preferred.sourceOffer.hasFootprint}<span class="asset-badge">Footprint</span>{/if}
                              {#if preferred.sourceOffer.hasDatasheet}<span class="asset-badge">Datasheet</span>{/if}
                              {#if !preferred.sourceOffer.hasSymbol && !preferred.sourceOffer.hasFootprint && !preferred.sourceOffer.hasDatasheet}
                                <span class="muted-cell">None</span>
                              {/if}
                            </div>
                          </span>
                        </div>
                        {#if preferred.sourceOffer.assetProbeState && preferred.sourceOffer.assetProbeState !== 'probed'}
                          <div class="preferred-field">
                            <span class="field-label">Probe status</span>
                            <span class="field-value">{preferred.sourceOffer.assetProbeState}</span>
                          </div>
                        {/if}
                        {#if preferred.sourceOffer.assetProbeError}
                          <div class="preferred-field error-text">Probe error: {preferred.sourceOffer.assetProbeError}</div>
                        {/if}
                        {#if preferred.sourceOffer.unitPrice !== null}
                          <div class="preferred-field">
                            <span class="field-label">Price</span>
                            <span class="field-value">
                              {formatPrice(preferred.sourceOffer)}
                              {#if preferred.sourceOffer.currency}
                                <span class="currency-label">{preferred.sourceOffer.currency}</span>
                              {/if}
                            </span>
                          </div>
                        {/if}
                        {#if preferred.sourceOffer.stock !== null}
                          <div class="preferred-field">
                            <span class="field-label">Stock</span>
                            <span class="field-value">
                              {formatCount(preferred.sourceOffer.stock)}
                              {#if preferred.sourceOffer.stock === 0}
                                <span class="badge badge-danger" style="margin-left:4px;">OOS</span>
                              {/if}
                            </span>
                          </div>
                        {/if}
                      {/if}
                      {#if rp.selectedPart}
                        <div class="preferred-field">
                          <span class="field-label">On Hand</span>
                          <span class="field-value">{rp.selectedPart.onHandQuantity} / {rp.requirement.quantity} required</span>
                        </div>
                        {#if rp.selectedPart.shortfallQuantity > 0}
                          <div class="preferred-field">
                            <span class="field-label">Shortfall</span>
                            <span class="field-value shortfall-value">{rp.selectedPart.shortfallQuantity}</span>
                          </div>
                        {/if}
                      {/if}
                    </div>
                    <div class="preferred-actions">
                      {#if preferred.origin === 'provider'}
                        <button
                          class="btn btn-primary btn-sm"
                          onclick={() => handleImportCandidate(preferred.id)}
                          disabled={!!actionInProgress[`import-${preferred.id}`]}
                        >
                          {actionInProgress[`import-${preferred.id}`] ? 'Importing…' : 'Import to Catalog'}
                        </button>
                      {/if}
                      {#if preferred.sourceOffer?.productUrl}
                        <button
                          class="btn btn-ghost btn-sm"
                          onclick={() => openUrl(preferred.sourceOffer!.productUrl)}
                        >
                          Open URL
                        </button>
                      {/if}
                      {#if preferred.componentId}
                        <button
                          class="btn btn-ghost btn-sm"
                          onclick={() => openComponent(preferred.componentId)}
                        >
                          Open Component
                        </button>
                      {/if}
                      <button
                        class="btn btn-ghost btn-sm"
                        onclick={() => handleDemoteCandidate(rp.requirement.id, preferred.id)}
                        disabled={!!actionInProgress[`demote-${preferred.id}`]}
                      >
                        Demote to Alternate
                      </button>
                    </div>
                  </div>
                {:else}
                  <div class="empty-msg">No preferred candidate selected. Promote an alternate below or return to Plan to add candidates.</div>
                {/if}
              </section>

              <!-- Alternate Candidates Section -->
              <section class="finalize-section">
                <div class="subsection-header">
                  <h4>Alternate Candidates</h4>
                  <span class="subsection-count">{alternates.length}</span>
                </div>
                {#if alternates.length === 0}
                  <div class="empty-msg">No alternate candidates. Use the Plan tab to add more options.</div>
                {:else}
                  <table class="match-table">
                    <thead>
                      <tr>
                        <th>MPN</th>
                        <th>Manufacturer</th>
                        <th>Package</th>
                        <th>Provider</th>
                        <th>Assets</th>
                        <th>Origin</th>
                        <th></th>
                      </tr>
                    </thead>
                    <tbody>
                      {#each alternates as alt}
                        <tr>
                          <td class="mpn-cell">{candidateDisplayMpn(alt)}</td>
                          <td>{candidateDisplayManufacturer(alt)}</td>
                          <td>{candidateDisplayPackage(alt)}</td>
                          <td>
                            {#if alt.sourceOffer?.provider}
                              <span class="provider-badge">{alt.sourceOffer.provider}</span>
                            {:else}
                              —
                            {/if}
                          </td>
                          <td>
                            {#if alt.sourceOffer}
                              <div class="asset-badges">
                                {#if alt.sourceOffer.hasSymbol}<span class="asset-badge">Symbol</span>{/if}
                                {#if alt.sourceOffer.hasFootprint}<span class="asset-badge">Footprint</span>{/if}
                                {#if alt.sourceOffer.hasDatasheet}<span class="asset-badge">Datasheet</span>{/if}
                                {#if !alt.sourceOffer.hasSymbol && !alt.sourceOffer.hasFootprint && !alt.sourceOffer.hasDatasheet}
                                  <span class="muted-cell">None</span>
                                {/if}
                              </div>
                            {:else}
                              —
                            {/if}
                          </td>
                          <td><span class="origin-badge origin-{alt.origin}">{providerOrOriginLabel(alt)}</span></td>
                          <td class="action-cell">
                            {#if alt.origin === 'provider'}
                              <button
                                class="btn btn-secondary btn-sm"
                                onclick={() => handleImportCandidate(alt.id)}
                                disabled={!!actionInProgress[`import-${alt.id}`]}
                              >
                                {actionInProgress[`import-${alt.id}`] ? 'Importing…' : 'Import'}
                              </button>
                            {/if}
                            <button
                              class="btn btn-primary btn-sm"
                              onclick={() => handlePromoteCandidate(rp.requirement.id, alt.id)}
                              disabled={!!actionInProgress[`promote-${alt.id}`]}
                            >
                              Promote to Preferred
                            </button>
                            {#if alt.sourceOffer?.productUrl}
                              <button
                                class="btn btn-ghost btn-sm"
                                onclick={() => openUrl(alt.sourceOffer!.productUrl)}
                              >
                                Open URL
                              </button>
                            {/if}
                            {#if alt.componentId}
                              <button
                                class="btn btn-ghost btn-sm"
                                onclick={() => openComponent(alt.componentId)}
                              >
                                Open Component
                              </button>
                            {/if}
                            <button
                              class="btn btn-ghost btn-sm"
                              onclick={() => handleRemoveCandidate(alt.id, false)}
                              disabled={!!actionInProgress[`remove-cand-${alt.id}`]}
                            >
                              Remove
                            </button>
                          </td>
                        </tr>
                      {/each}
                    </tbody>
                  </table>
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
  .finalize-tab {
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
  .error-text {
    color: var(--color-danger-text);
    font-size: 13px;
  }

  /* ---- Readiness Summary ---- */
  .readiness-summary {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    margin-bottom: 20px;
    padding: 14px 18px;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    background: var(--color-bg-surface);
  }
  .readiness-row {
    display: flex;
    gap: 24px;
  }
  .readiness-stat {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 2px;
  }
  .readiness-value {
    font-size: 20px;
    font-weight: 700;
    font-variant-numeric: tabular-nums;
    color: var(--color-text-primary);
  }
  .readiness-label {
    font-size: 10px;
    letter-spacing: 0.06em;
    text-transform: uppercase;
    color: var(--color-text-muted);
    white-space: nowrap;
  }
  .readiness-ok {
    color: var(--color-success-text);
  }
  .readiness-warn {
    color: var(--color-warning-text);
  }
  .readiness-danger {
    color: var(--color-danger-text);
  }
  .future-actions {
    display: flex;
    gap: 8px;
    flex-shrink: 0;
  }

  /* ---- Requirement List ---- */
  .req-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  .req-card {
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    background: var(--color-bg-surface);
    overflow: hidden;
  }
  .req-card-warn {
    border-color: var(--color-warning-border);
  }
  .req-card-danger {
    border-color: var(--color-danger-border);
  }
  .req-card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    width: 100%;
    padding: 12px 16px;
    text-align: left;
    transition: background 0.1s;
  }
  .req-card-header:hover {
    background: var(--color-bg-hover);
  }
  .req-header-left {
    display: flex;
    align-items: center;
    gap: 8px;
  }
  .req-header-right {
    display: flex;
    align-items: center;
    gap: 12px;
    font-size: 12px;
  }
  .req-name {
    font-weight: 600;
    font-size: 13px;
  }
  .status-dot {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 18px;
    height: 18px;
    border-radius: 999px;
    font-size: 10px;
    font-weight: 700;
    flex-shrink: 0;
  }
  .status-dot-ok {
    background: var(--color-success-soft);
    color: var(--color-success-text);
  }
  .status-dot-warn {
    background: var(--color-warning-soft);
    color: var(--color-warning-text);
  }
  .status-dot-danger {
    background: var(--color-danger-soft);
    color: var(--color-danger-text);
  }
  .preferred-summary {
    color: var(--color-text-secondary);
    font-size: 12px;
  }
  .preferred-summary-label {
    color: var(--color-text-muted);
    margin-right: 4px;
  }
  .no-preferred-summary {
    color: var(--color-text-muted);
    font-size: 12px;
    font-style: italic;
  }
  .expand-icon {
    color: var(--color-text-muted);
    font-size: 11px;
  }
  .badge-qty {
    font-variant-numeric: tabular-nums;
    font-weight: 600;
  }

  /* ---- Warnings ---- */
  .warnings-strip {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
    padding: 6px 16px 8px;
    background: var(--color-bg-muted);
    border-top: 1px solid var(--color-border);
  }
  .warning-item {
    display: inline-flex;
    align-items: center;
    padding: 2px 8px;
    border-radius: 999px;
    font-size: 11px;
    color: var(--color-warning-text);
    background: var(--color-warning-soft);
    border: 1px solid var(--color-warning-border);
  }

  /* ---- Expanded content ---- */
  .req-expanded {
    padding: 14px 16px;
    border-top: 1px solid var(--color-border);
    background: var(--color-bg-muted);
    display: flex;
    flex-direction: column;
    gap: 18px;
  }
  .finalize-section {
    display: flex;
    flex-direction: column;
    gap: 10px;
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
    display: block;
  }
  .subsection-count {
    font-size: 11px;
    color: var(--color-text-muted);
    margin-left: 4px;
  }

  /* ---- Preferred card ---- */
  .preferred-card {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 16px;
    padding: 12px 14px;
    border: 1px solid var(--color-success-border);
    border-radius: var(--radius-md);
    background: var(--color-success-soft);
  }
  .offer-thumb-lg {
    width: 48px;
    height: 48px;
    object-fit: contain;
    border-radius: 4px;
    flex-shrink: 0;
  }
  .preferred-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
    gap: 10px 20px;
  }
  .preferred-field {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }
  .field-label {
    font-size: 10px;
    letter-spacing: 0.06em;
    text-transform: uppercase;
    color: var(--color-text-muted);
  }
  .field-value {
    font-size: 12px;
    color: var(--color-text-primary);
  }
  .preferred-actions {
    flex-shrink: 0;
  }
  .shortfall-value {
    color: var(--color-danger-text);
    font-weight: 600;
  }

  /* ---- Tables ---- */
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
  .match-table th:last-child {
    width: 220px;
  }
  .match-table td {
    padding: 6px 10px;
    border-bottom: 1px solid var(--color-border);
  }
  .match-table tbody tr:hover {
    background: var(--color-bg-surface);
  }
  .mpn-cell {
    font-weight: 600;
  }
  .qty-cell {
    font-variant-numeric: tabular-nums;
    font-weight: 500;
  }
  .action-cell {
    display: flex;
    gap: 6px;
    align-items: center;
    justify-content: flex-end;
    flex-wrap: wrap;
    min-width: 180px;
    text-align: right;
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
    border: 1px solid var(--color-warning-border);
  }
  .origin-imported_from_supplier {
    background: var(--color-success-soft);
    color: var(--color-success-text);
    border: 1px solid var(--color-success-border);
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
  .asset-badges {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
  }
  .asset-badge {
    font-size: 10px;
    padding: 2px 6px;
    border-radius: 999px;
    color: var(--color-text-secondary);
    background: var(--color-bg-muted);
    font-weight: 600;
  }
  .currency-label {
    font-size: 10px;
    color: var(--color-text-muted);
    margin-left: 2px;
  }

  @media (max-width: 720px) {
    .readiness-summary {
      flex-direction: column;
      align-items: stretch;
    }
    .readiness-row {
      flex-wrap: wrap;
    }
    .preferred-card {
      flex-direction: column;
    }
    .preferred-grid {
      grid-template-columns: 1fr 1fr;
    }
  }
</style>
