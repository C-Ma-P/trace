<div class="settings-page">
  <section class="settings-section">
    <div class="section-header">
      <div>
        <h2>Integration providers</h2>
        <p>External tooling that Trace can talk to for project discovery, import, and workflow handoff.</p>
      </div>
    </div>

    <div class="settings-copy">
      Integrations define where Trace looks for external project data and which tools it can cooperate with. Choose a provider in the navigation tree to configure its concrete paths and defaults.
    </div>

    <div class="provider-summary-list" role="list">
      <div class="provider-summary-row" role="listitem">
        <div class="provider-summary-main">
          <div class="provider-summary-title">
            <strong>KiCad</strong>
            <span class="badge">Available</span>
          </div>
          <p>Configure discovery roots used by the KiCad importer when scanning for .kicad_pro projects.</p>
        </div>
        <div class="provider-summary-meta">
          <div>
            <span class="meta-label">Purpose</span>
            <span>Project discovery</span>
          </div>
          <div>
            <span class="meta-label">Configuration</span>
            <span>Discovery roots</span>
          </div>
        </div>
      </div>
    </div>
  </section>
</div>

<style>
  .settings-page {
    display: flex;
    flex-direction: column;
    gap: 12px;
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

  .provider-summary-list {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .provider-summary-row {
    display: flex;
    justify-content: space-between;
    gap: 16px;
    padding: 12px 14px;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    background: var(--color-bg-app);
  }

  .provider-summary-main {
    display: flex;
    flex-direction: column;
    gap: 6px;
    min-width: 0;
  }

  .provider-summary-main p {
    color: var(--color-text-secondary);
    line-height: 1.45;
  }

  .provider-summary-title {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-wrap: wrap;
  }

  .provider-summary-meta {
    display: grid;
    grid-template-columns: repeat(2, minmax(120px, auto));
    gap: 12px;
    align-content: start;
    color: var(--color-text-secondary);
    font-size: 12px;
  }

  .provider-summary-meta > div {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .meta-label {
    color: var(--color-text-muted);
    text-transform: uppercase;
    letter-spacing: 0.08em;
    font-size: 10px;
  }

  .badge {
    display: inline-flex;
    align-items: center;
    padding: 2px 8px;
    border-radius: 999px;
    background: var(--color-bg-muted);
    color: var(--color-text-secondary);
    font-size: 11px;
    font-weight: 600;
  }

  @media (max-width: 720px) {
    .provider-summary-row {
      flex-direction: column;
    }

    .provider-summary-meta {
      grid-template-columns: 1fr;
    }
  }
</style>