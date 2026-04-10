# Component Manager — Cleanup & Consolidation Plan

---

# 1. Repo Health Summary

## Overall Code Health
The codebase is in moderate health. The architecture is sound — clean domain/service/repository layering with idiomatic Go interfaces. The frontend uses modern Svelte 5 runes correctly. The main issues are organic growth: bloated files, duplicated patterns, overlapping abstractions around the sourcing/candidate workflow, and a few legacy paths that coexist with newer ones.

## Main Cleanup Themes
1. **Duplicated project-creation disk logic** — same ~30 lines inlined 3x instead of using the existing `createProjectDiskState()` helper
2. **Bloated service.go** — 1,299 lines, 65 methods, mixing component/project/sourcing/asset/preference concerns
3. **Overlapping candidate/offer/resolution abstractions** — 6+ methods doing variations of "connect an offer or component to a requirement"
4. **Frontend pattern duplication** — edit-mode boilerplate, `typeLabels` map, and list/filter patterns copied across 5+ components
5. **Dead/stub code** — two unimplemented asset search providers, empty `internal/startup/` directory, duplicate stubs in frontend
6. **Tracked build artifacts** — compiled binaries and generated wailsjs files in git history despite .gitignore rules

## Top Opportunities
- **Delete dead code** — provider stubs, empty startup dir, duplicate frontend stubs (~5 min, zero risk)
- **Extract shared disk-state helper** — eliminate 60+ duplicated lines across 2 functions (~15 min, low risk)
- **Extract frontend constants/composables** — eliminate ~200 lines of repeated patterns (~30 min, low risk)
- **Split service.go** — move sourcing and asset methods to separate files (~45 min, medium risk)

## Stable vs. Messy
- **Stable**: domain types, repository interfaces, store/postgres layer, registry, secretstore, paths, launcher, kicadconfig, ingest, TLS generation
- **Messy**: `service/service.go` (too many concerns), `app/app_projects.go` (duplicated disk logic + 27 methods), frontend `AssetsTab`/`AssetSection`/`DetailsTab`/`SpecificationsSection` overlap, `phoneintake/page.go` (800-line HTML string), `sourcing/types.go` (magic scoring numbers)

---

# 2. Cleanup Findings Ledger

---

## F01: Project disk-state creation duplicated 3x

- **Severity**: high
- **Confidence**: high
- **Area**: backend
- **Files**:
  - `internal/app/app_projects.go` (lines 68–118: `CreateBlankProject`)
  - `internal/app/app_projects.go` (lines 120–177: `CreateProjectWithDisk`)
  - `internal/app/app_kicad.go` (lines 79–111: `createProjectDiskState`)
- **Problem**:
  - `createProjectDiskState()` already exists in `app_kicad.go` and does exactly what `CreateBlankProject` and `CreateProjectWithDisk` do inline: create dir, write `project.json`, clean up on error. The two `app_projects.go` functions duplicate this logic verbatim instead of calling it.
- **Evidence**:
  - `CreateBlankProject` lines 78–103 and `CreateProjectWithDisk` lines 132–163 are character-for-character identical to `createProjectDiskState`. The only differences are the variable names for project name/description.
- **Recommended cleanup**:
  - Move `createProjectDiskState` to `app_projects.go` (or a shared `app_helpers.go`). Rewrite `CreateBlankProject` and `CreateProjectWithDisk` to call it. Delete the inline copies.
- **Risk**: Low — pure refactor, no behavior change.
- **Batch**: B1-dead-and-duplicate

---

## F02: Compiled binaries and generated files tracked in git

- **Severity**: medium
- **Confidence**: high
- **Area**: cross-cutting
- **Files**:
  - `bin/trace` (33 MB ELF binary)
  - `build/bin/component-manager` (14 MB binary)
  - `frontend/wailsjs/go/app/App.d.ts`
  - `frontend/wailsjs/go/app/App.js`
  - `frontend/wailsjs/go/models.ts`
  - `frontend/wailsjs/runtime/package.json`
  - `frontend/wailsjs/runtime/runtime.d.ts`
  - `frontend/wailsjs/runtime/runtime.js`
- **Problem**:
  - These files are gitignored but were added to tracking before the gitignore rules existed. They inflate the repo by ~47 MB and create spurious diffs.
- **Evidence**:
  - `git ls-files bin/ build/bin/ frontend/wailsjs/` shows all files tracked. `.gitignore` contains `/build/`, `/bin/`, `frontend/wailsjs/`.
- **Recommended cleanup**:
  - `git rm --cached` these files and commit. The .gitignore rules will keep them out going forward.
- **Risk**: Medium — requires a coordinated commit and any CI that depends on committed wailsjs stubs may break. Need to verify Wails generates these on build.
- **Batch**: B1-dead-and-duplicate

---

## F03: Asset search provider stubs — keep, ensure consistent pattern

- **Severity**: low
- **Confidence**: high
- **Area**: backend
- **Files**:
  - `internal/assetsearch/providers/snapeda.go`
  - `internal/assetsearch/providers/ultralibrarian.go`
- **Problem**:
  - Both are stub implementations that return "not implemented" errors. They are intentionally kept as placeholders for future implementation.
- **Evidence**:
  - Both files: `return nil, fmt.Errorf("... provider not implemented")` for both `Search` and `Import`.
- **Recommended cleanup**:
  - **Keep both files.** Verify they follow the same interface pattern as the EasyEDA provider so future implementation is straightforward. No deletion.
- **Risk**: Zero.
- **Batch**: none (no action needed)

---

## F04: Empty `internal/startup/` directory

- **Severity**: low
- **Confidence**: high
- **Area**: backend
- **Files**:
  - `internal/startup/` (empty directory)
- **Problem**:
  - Directory contains no files. Likely a remnant of a planned module that was never implemented or was moved elsewhere.
- **Evidence**:
  - `ls -la internal/startup/` shows only `.` and `..`.
- **Recommended cleanup**:
  - Delete the directory.
- **Risk**: Zero.
- **Batch**: B1-dead-and-duplicate

---

## F05: Duplicate frontend stub files

- **Severity**: low
- **Confidence**: high
- **Area**: frontend
- **Files**:
  - `frontend/src/stubs/app-environment.ts`
  - `frontend/src/stubs/$app/environment.js`
- **Problem**:
  - Identical content: both export `browser = true, building = false, dev = false, version = ''`. One is TypeScript, one is JavaScript — only one should be needed.
- **Evidence**:
  - File contents are character-identical.
- **Recommended cleanup**:
  - Determine which is actually imported (check `tsconfig.json` paths or Vite aliases). Delete the unused one.
- **Risk**: Low — just need to verify which alias is active.
- **Batch**: B1-dead-and-duplicate

---

## F06: `typeLabels` map duplicated 3x in frontend

- **Severity**: medium
- **Confidence**: high
- **Area**: frontend
- **Files**:
  - `frontend/src/lib/components/AssetsTab.svelte` (line 42)
  - `frontend/src/lib/components/AddFromFileModal.svelte` (line 96)
  - `frontend/src/lib/components/AssetSection.svelte` (line 36)
- **Problem**:
  - The same `{ symbol: 'Symbol', footprint: 'Footprint', '3d_model': '3D Model', datasheet: 'Datasheet' }` map is defined inline in 3 separate components.
- **Evidence**:
  - `grep -rn "typeLabels" frontend/src/lib/ --include="*.svelte"` shows 3 definition sites.
- **Recommended cleanup**:
  - Extract to `frontend/src/lib/constants.ts` as `export const ASSET_TYPE_LABELS`. Import in all 3 files.
- **Risk**: Zero — pure extraction.
- **Batch**: B2-frontend-consolidation

---

## F07: Edit-mode boilerplate duplicated across 5+ components

- **Severity**: medium
- **Confidence**: high
- **Area**: frontend
- **Files**:
  - `frontend/src/lib/components/DetailsTab.svelte`
  - `frontend/src/lib/components/InventoryCard.svelte`
  - `frontend/src/lib/components/SpecificationsSection.svelte`
  - `frontend/src/lib/components/InventoryTab.svelte`
  - `frontend/src/lib/preferences/KiCadIntegrationsPage.svelte`
- **Problem**:
  - Each component independently declares `editing`, `saving`, `error` state variables and `startEdit`/`cancelEdit`/`save` handlers with identical try/catch/finally patterns.
- **Evidence**:
  - Pattern: `let editing = $state(false); let saving = $state(false); let error = $state('');` followed by identical async save wrapper.
- **Recommended cleanup**:
  - Extract a `createEditMode()` helper function to `frontend/src/lib/editMode.ts` that returns `{ editing, saving, error, startEdit, cancelEdit, wrapSave }`. Each component calls `wrapSave(async () => { /* actual save logic */ })`.
- **Risk**: Low — each component's save logic is unique, only the surrounding boilerplate is shared.
- **Batch**: B2-frontend-consolidation

---

## F08: `DetailsTab.svelte` and `SpecificationsSection.svelte` are near-duplicates

- **Severity**: medium
- **Confidence**: medium
- **Area**: frontend
- **Files**:
  - `frontend/src/lib/components/DetailsTab.svelte` (248 lines)
  - `frontend/src/lib/components/SpecificationsSection.svelte` (262 lines)
- **Problem**:
  - Both render the same MPN/Manufacturer/Package/Description metadata fields with the same edit flow and the same `buildEditAttributes` pattern. One is used as a tab, the other as a section — but the core rendering is ~70% identical.
- **Evidence**:
  - Both contain: metadata fields, attribute definitions loading, attribute editor grid, identical save/cancel handlers.
- **Recommended cleanup**:
  - Determine which is the canonical version. If both are used simultaneously, extract shared logic into a `ComponentMetadataEditor.svelte` component that both consume.
- **Risk**: Medium — need to verify where each is mounted to ensure the right one is consolidated.
- **Batch**: B2-frontend-consolidation

---

## F09: Legacy svelte/store (`writable`/`derived`) in workspace stores

- **Severity**: low
- **Confidence**: high
- **Area**: frontend
- **Files**:
  - `frontend/src/lib/components/componentsWorkspaceStore.ts`
  - `frontend/src/lib/projects/projectsWorkspaceStore.ts`
  - `frontend/src/lib/launcher/launcherWorkspaceStore.ts`
- **Problem**:
  - These stores use `writable`/`derived` from `svelte/store` while the rest of the codebase uses Svelte 5 runes (`$state`, `$derived`). This creates two incompatible reactive systems.
- **Evidence**:
  - `grep -rn "writable\|derived" frontend/src/lib/ --include="*.ts"` shows all 3 stores import from `svelte/store`.
- **Recommended cleanup**:
  - Migrate to runes-based stores using `$state` in a module-level `<script module>` or plain `.svelte.ts` files. This is a Svelte 5 best practice.
- **Risk**: Medium — requires changing how stores are subscribed to in consuming components (from `$store` syntax to direct property access or `.subscribe()`).
- **Batch**: B4-frontend-modernization

---

## F10: `service.go` is 1,299 lines with 65 methods

- **Severity**: high
- **Confidence**: high
- **Area**: backend
- **Files**:
  - `internal/service/service.go`
- **Problem**:
  - Single file mixing component CRUD, project CRUD, requirement management, part candidate/offer workflow (6+ closely related methods), sourcing orchestration, asset management, and preference management. The `Service` struct has too many responsibilities.
- **Evidence**:
  - `grep -c "^func (s \*Service)" internal/service/service.go` → 65 methods.
  - Methods span: component lifecycle (~10), inventory (~3), preferences (~5), projects (~5), requirements (~3), candidates/offers (~8), sourcing (~5), assets (~8), planning (~3), KiCad import (~3), utilities (~12).
- **Recommended cleanup**:
  - **Phase 1** (low risk): Split into multiple files within the same package: `service_components.go`, `service_projects.go`, `service_sourcing.go`, `service_assets.go`, `service_preferences.go`. Keep the `Service` struct — just split the methods across files.
  - **Phase 2** (higher risk, later): Consider splitting into separate service types if the file split reveals clean boundaries.
- **Risk**: Phase 1 is zero-risk (file splits don't change behavior). Phase 2 requires interface changes.
- **Batch**: B3-backend-structure

---

## F11: Overlapping candidate/offer resolution abstractions

- **Severity**: medium
- **Confidence**: high
- **Area**: backend
- **Files**:
  - `internal/service/service.go` (methods at lines 405–811)
- **Problem**:
  - Six overlapping methods for connecting components/offers to requirements:
    1. `SelectComponentForRequirement` — legacy, sets resolution directly
    2. `AddPartCandidate` — adds local candidate without setting preferred
    3. `SetPreferredCandidate` — sets existing candidate as preferred
    4. `AddLocalComponentAsCandidateAndSetPreferred` — combines #2 + #3
    5. `ImportSupplierOffer` — creates component from offer + candidate + optionally preferred
    6. `AddProviderCandidate` — saves offer + creates candidate + optionally preferred
    7. `ImportProviderCandidate` — imports provider-backed candidate into catalog
  - The "preferred candidate ↔ requirement resolution" invariant is enforced across 5 separate functions with no shared enforcement point.
- **Evidence**:
  - `SetPreferredCandidate`, `DemotePreferredCandidate`, `RemovePartCandidate`, `ClearSelectedComponentForRequirement`, `ImportProviderCandidate` all independently clear/set resolution state.
- **Recommended cleanup**:
  - Extract a `setResolutionFromCandidate(ctx, requirementID, candidateID)` and `clearResolution(ctx, requirementID)` pair that all methods call. This centralizes the invariant.
  - Consider deprecating `SelectComponentForRequirement` if it's the legacy path.
- **Risk**: Medium — must preserve existing behavior for all callers. Good test coverage exists.
- **Batch**: B5-backend-consolidation

---

## F12: `app/types.go` contains ~588 lines of response/input types

- **Severity**: low
- **Confidence**: high
- **Area**: backend
- **Files**:
  - `internal/app/types.go`
- **Problem**:
  - All request/response DTOs for the Wails API are in a single file. While not broken, it's harder to navigate than splitting by domain.
- **Evidence**:
  - 588 lines of struct definitions with no methods.
- **Recommended cleanup**:
  - No immediate action needed — this is low-priority. If `app_projects.go` gets split, the types can move with it.
- **Risk**: Zero.
- **Batch**: B6-low-priority

---

## F13: `phoneintake/page.go` — 625-line HTML/JS string literal

- **Severity**: medium
- **Confidence**: high
- **Area**: backend
- **Files**:
  - `internal/phoneintake/page.go`
- **Problem**:
  - Entire phone intake barcode scanner UI is embedded as a Go string literal. No syntax highlighting, no separate development/testing path, difficult to maintain.
- **Evidence**:
  - 625 lines, ~90% is a raw HTML template string containing unminified JavaScript.
- **Recommended cleanup**:
  - Move to an embedded `//go:embed` file (`page.html`) so the HTML has syntax highlighting and can be developed/tested independently.
- **Risk**: Low — pure file extraction with `embed` directive.
- **Batch**: B3-backend-structure

---

## F14: `sourceTypes.go` magic scoring numbers

- **Severity**: low
- **Confidence**: high
- **Area**: backend
- **Files**:
  - `internal/sourcing/types.go`
- **Problem**:
  - `scoreOffer()` and `RankOffers()` use hardcoded magic numbers for scoring: 120, 55, -35, 40, 18, 26, 14, -8, 10, 12, 8, +1000. No named constants, no documentation of why these values were chosen.
- **Evidence**:
  - `types.go` lines with direct integer literals in scoring logic.
- **Recommended cleanup**:
  - Extract named constants: `const scoreMPNExact = 120`, `const scoreMPNPartial = 55`, etc. Add a brief comment block explaining the scoring strategy.
- **Risk**: Zero — no behavior change.
- **Batch**: B6-low-priority

---

## F15: Unit conversion functions duplicated in EasyEDA provider

- **Severity**: low
- **Confidence**: high
- **Area**: backend
- **Files**:
  - `internal/providers/easyeda/convert_symbol.go` (line 10: `pxToMM`)
  - `internal/providers/easyeda/convert_footprint.go` (line 10: `eeToMM`)
- **Problem**:
  - Both functions compute `dim * 10 * 0.0254` (identical formula). Different names (`pxToMM` vs `eeToMM`) suggest they might be different, but they're not.
- **Evidence**:
  - `pxToMM(dim) = 10.0 * dim * 0.0254` and `eeToMM(dim) = dim * 10 * 0.0254` — algebraically identical.
- **Recommended cleanup**:
  - Consolidate into a single `eeToMM` in a shared location (e.g., `convert_common.go`). Rename all usages.
- **Risk**: Zero — same formula.
- **Batch**: B6-low-priority

---

## F16: `ProjectRepository` interface has 22 methods

- **Severity**: medium
- **Confidence**: medium
- **Area**: backend
- **Files**:
  - `internal/domain/repository.go` (lines 36–68)
  - `internal/store/postgres/project_repository.go` (681 lines)
- **Problem**:
  - The `ProjectRepository` interface mixes project CRUD (5), requirement ops (3), part candidate ops (7), and supplier offer ops (5). This makes it hard to implement test stubs and blurs boundaries.
- **Recommended cleanup**:
  - **Phase 1**: No interface split yet — just document the groupings with comments.
  - **Phase 2** (future): Consider splitting into `ProjectRepository`, `CandidateRepository`, `OfferRepository` if the service split (F10) creates natural boundaries.
- **Risk**: High for Phase 2 — requires updating all callers. Low for Phase 1.
- **Batch**: B6-low-priority

---

## F17: `app_projects.go` has 3 project creation methods with overlapping semantics

- **Severity**: medium
- **Confidence**: high
- **Area**: backend
- **Files**:
  - `internal/app/app_projects.go`
- **Problem**:
  - Three project creation endpoints:
    1. `CreateProject` — no disk state, just DB
    2. `CreateBlankProject` — creates disk state + DB with defaults
    3. `CreateProjectWithDisk` — creates disk state + DB with user-provided name
  - `CreateProject` appears to be the legacy path that doesn't create a disk folder. The newer paths both create disk state.
- **Evidence**:
  - `CreateProject` at line 49 doesn't call `paths.EnsureProjectsDir()`. The other two do.
- **Recommended cleanup**:
  - After fixing F01, determine if `CreateProject` is still called. If not, remove it. If yes, consider whether all projects should have disk state.
- **Risk**: Medium — need to verify frontend callers.
- **Batch**: B5-backend-consolidation

---

## F18: `InventoryBagRepository` interface may be dead

- **Severity**: low
- **Confidence**: medium
- **Area**: backend
- **Files**:
  - `internal/domain/repository.go` (lines 70+)
  - `internal/domain/bag.go`
  - `internal/store/postgres/bag_repository.go`
- **Problem**:
  - The `InventoryBagRepository` is defined and implemented, but it's wired through `app.SetBagRepo()` — an optional setter — not through the service layer. The domain `InventoryBag` type and the repo implementation may be used only from the app layer directly, bypassing the service.
- **Evidence**:
  - `app_intake.go` calls `a.bagRepo` methods directly without going through `a.svc`.
  - `app_components.go` also references `a.bagRepo` for image URL lookup.
- **Recommended cleanup**:
  - This is a known pattern (optional feature wiring). Leave as-is unless the feature is being removed.
- **Risk**: N/A — just documenting.
- **Batch**: none

---

## F19: `SelectComponentForRequirement` / `ClearSelectedComponentForRequirement` may be legacy

- **Severity**: medium
- **Confidence**: medium
- **Area**: backend
- **Files**:
  - `internal/service/service.go` (lines 405–416)
  - `internal/app/app_projects.go` (lines 333–347)
- **Problem**:
  - These methods directly set `requirement.Resolution` without going through the candidate workflow. They appear to be a legacy path from before the candidate/offer model was added.
- **Evidence**:
  - `SelectComponentForRequirement` just calls `SetRequirementResolution`. No candidate is created. The newer workflow always creates a `ProjectPartCandidate` first.
- **Recommended cleanup**:
  - Check if the frontend still calls these. If not, remove. If yes, consider whether they should be redirected through `AddLocalComponentAsCandidateAndSetPreferred`.
- **Risk**: Medium — need to verify all callers.
- **Batch**: B5-backend-consolidation

---

## F20: `app.go` `GetRequirementDefinitions` and `GetCategoryDefinitions` overlap

- **Severity**: low
- **Confidence**: medium
- **Area**: backend
- **Files**:
  - `internal/app/app.go`
- **Problem**:
  - `GetCategoryDefinitions` returns attribute definitions per category. `GetRequirementDefinitions` returns constraint definitions per category. Both iterate the same `registry.Categories()` list with the same pattern but call different registry functions.
- **Recommended cleanup**:
  - This is intentional design (attributes vs constraints are different things). No action needed.
- **Risk**: N/A.
- **Batch**: none

---

## F21: Frontend Splitpanes layout pattern repeated

- **Severity**: low
- **Confidence**: medium
- **Area**: frontend
- **Files**:
  - `frontend/src/lib/components/ComponentsWorkspace.svelte`
  - `frontend/src/lib/projects/ProjectsWorkspace.svelte`
- **Problem**:
  - Both workspaces use the exact same Splitpanes container pattern with list-on-left, detail-on-right. The boilerplate is small (~15 lines each) so not worth abstracting.
- **Recommended cleanup**:
  - No action needed — the duplicated boilerplate is minimal and the components have different enough contents.
- **Risk**: N/A.
- **Batch**: none

---

## F22: `service_test.go` has all test stubs in one file at 1,192 lines

- **Severity**: medium
- **Confidence**: high
- **Area**: backend
- **Files**:
  - `internal/service/service_test.go`
- **Problem**:
  - Three full repository stubs (`stubComponentRepo`, `stubProjectRepo`, `stubAssetRepo`) plus all test cases in a single file. Hard to navigate.
- **Evidence**:
  - 1,192 lines, ~40 test functions.
- **Recommended cleanup**:
  - Split stubs into `service_test_stubs_test.go`. Split test cases by domain: `service_component_test.go`, `service_project_test.go`, etc.
- **Risk**: Zero — test file organization.
- **Batch**: B3-backend-structure

---

## F23: Dual-mode `Resolution` / `SelectedComponentID` in `ProjectRequirement`

- **Severity**: medium
- **Confidence**: high
- **Area**: backend
- **Files**:
  - `internal/domain/project.go` (lines 38–80)
- **Problem**:
  - `ProjectRequirement` has both a legacy `SelectedComponentID *string` and a newer `Resolution *RequirementResolution`. `NormalizeResolution()` tries to keep them in sync but this dual representation is fragile and adds complexity throughout the codebase.
- **Evidence**:
  - `NormalizeResolution()` contains bidirectional sync logic. Service methods must remember to normalize after any mutation.
- **Recommended cleanup**:
  - This needs a data migration to move all requirements to `Resolution` exclusively and drop `SelectedComponentID`. High-risk, defer to a dedicated migration batch.
- **Risk**: High — database migration + all callers.
- **Batch**: B7-structural-refactors

---

## F24: `vite.config.ts.timestamp-*` file in workspace root

- **Severity**: low
- **Confidence**: high
- **Area**: cross-cutting
- **Files**:
  - `frontend/vite.config.ts.timestamp-1775378745383-ceef56153495a.mjs`
- **Problem**:
  - Vite temp file present in the workspace. `.gitignore` has `*.timestamp*.mjs` so it's not tracked, but it clutters the file tree.
- **Recommended cleanup**:
  - Delete the file. It's regenerated by Vite on every build.
- **Risk**: Zero.
- **Batch**: B1-dead-and-duplicate

---

## F25: Frontend `PlanTab.svelte` and `FinalizeTab.svelte` are both 800-1000+ lines

- **Severity**: medium
- **Confidence**: high
- **Area**: frontend
- **Files**:
  - `frontend/src/lib/projects/PlanTab.svelte` (1,089 lines)
  - `frontend/src/lib/projects/FinalizeTab.svelte` (877 lines)
- **Problem**:
  - Both tabs handle requirement-level expansion, sourcing results display, candidate management, and offer display. They share overlapping state patterns (`expandedReqs`, `actionInProgress`, sourcing-by-requirement maps) but implement them independently.
- **Recommended cleanup**:
  - Extract shared requirement-level expansion and sourcing state into a composable or shared component. Extract requirement row rendering into a `RequirementPlanRow.svelte` component.
- **Risk**: Medium — these are the most complex UI components and require careful testing.
- **Batch**: B4-frontend-modernization

---

## F26: `SuppliersSettingsPage.svelte` uses 9 separate state variables for 3 suppliers

- **Severity**: low
- **Confidence**: high
- **Area**: frontend
- **Files**:
  - `frontend/src/lib/preferences/SuppliersSettingsPage.svelte` (535 lines)
- **Problem**:
  - State for DigiKey, Mouser, and LCSC settings is managed with 3 separate groups of variables instead of a single `suppliers: Record<string, SupplierState>` object.
- **Recommended cleanup**:
  - Refactor state into a keyed object. This also simplifies the render logic to use a loop over suppliers.
- **Risk**: Low — internal refactor.
- **Batch**: B4-frontend-modernization

---

## F27: `hydrateCandidates()` silently skips errors

- **Severity**: medium
- **Confidence**: high
- **Area**: backend
- **Files**:
  - `internal/service/service.go`
- **Problem**:
  - When hydrating candidates with component data, if `GetComponent` fails, the candidate is silently skipped with `continue`. This masks data integrity issues.
- **Recommended cleanup**:
  - Log a warning when a candidate's component can't be loaded. Consider returning the candidate with a nil Component field and a flag indicating load failure, rather than silently dropping it.
- **Risk**: Low — adding logging doesn't change behavior.
- **Batch**: B5-backend-consolidation

---

## F28: `findOrCreateComponentFromOffer` and `ResolveComponentFromOffer` overlap

- **Severity**: medium
- **Confidence**: high
- **Area**: backend
- **Files**:
  - `internal/service/service.go`
- **Problem**:
  - Both methods find-or-create a component from supplier offer data using manufacturer+MPN deduplication. `findOrCreateComponentFromOffer` is private and used by `ImportSupplierOffer`. `ResolveComponentFromOffer` is public and used by `phoneintake`. They have similar but not identical logic.
- **Recommended cleanup**:
  - Unify into one method. If the public API needs it, expose `findOrCreateComponentFromOffer` (renamed) as the single implementation.
- **Risk**: Medium — need to compare exact behavior differences.
- **Batch**: B5-backend-consolidation

---

---

# 3. Execution Batches

---

## Batch B1: Dead Code and Obvious Duplicates

**Goal**: Remove dead files, untrack artifacts, eliminate clear-cut duplication.

**Why together**: All zero/low-risk deletions and extractions with no behavioral change.

**Files/modules in scope**:
- `internal/startup/` — delete directory
- `frontend/src/stubs/app-environment.ts` OR `frontend/src/stubs/$app/environment.js` — delete the unused one
- `frontend/vite.config.ts.timestamp-*.mjs` — delete
- `bin/trace`, `build/bin/component-manager`, `frontend/wailsjs/` — `git rm --cached`
- `internal/app/app_projects.go` — refactor `CreateBlankProject`/`CreateProjectWithDisk` to call `createProjectDiskState`
- `internal/app/app_kicad.go` — move `createProjectDiskState` to shared location

**Exact changes**:
1. `git rm --cached bin/trace build/bin/component-manager frontend/wailsjs/go/app/App.d.ts frontend/wailsjs/go/app/App.js frontend/wailsjs/go/models.ts frontend/wailsjs/runtime/package.json frontend/wailsjs/runtime/runtime.d.ts frontend/wailsjs/runtime/runtime.js`
2. Delete `internal/assetsearch/providers/snapeda.go`, `internal/assetsearch/providers/ultralibrarian.go`
3. Delete `internal/startup/` directory
4. Delete `frontend/vite.config.ts.timestamp-*.mjs`
5. Check which stub is imported (via tsconfig paths), delete the other
6. Move `createProjectDiskState` from `app_kicad.go` to `app_projects.go`. Rewrite `CreateBlankProject` and `CreateProjectWithDisk` to call it.

**What NOT to touch**: Any behavioral code, service layer, domain types.

**Risk level**: Low

**Validation steps**:
- `go build ./...` passes
- `go test ./...` passes
- `npm run build` in `frontend/` passes

**Acceptance criteria**:
- All deleted files gone
- Binary files untracked
- `CreateBlankProject` and `CreateProjectWithDisk` both call `createProjectDiskState`
- All tests pass

---

## Batch B2: Frontend Constants and Pattern Extraction

**Goal**: Extract duplicated frontend constants and the edit-mode boilerplate.

**Why together**: All are simple extractions from frontend components with no behavioral change.

**Files/modules in scope**:
- Create `frontend/src/lib/constants.ts` with `ASSET_TYPE_LABELS`
- Update `AssetsTab.svelte`, `AddFromFileModal.svelte`, `AssetSection.svelte` to import from constants
- Create `frontend/src/lib/editMode.ts` with `createEditMode()` helper (optional — only if the pattern is identical enough)
- Determine relationship between `DetailsTab.svelte` and `SpecificationsSection.svelte` — consolidate if one is unused

**Exact changes**:
1. Create `frontend/src/lib/constants.ts`:
   ```ts
   export const ASSET_TYPE_LABELS: Record<string, string> = {
     symbol: 'Symbol',
     footprint: 'Footprint',
     '3d_model': '3D Model',
     datasheet: 'Datasheet',
   };
   ```
2. In `AssetsTab.svelte`, `AddFromFileModal.svelte`, `AssetSection.svelte`: replace inline `typeLabels` with import
3. Investigate `DetailsTab.svelte` vs `SpecificationsSection.svelte` usage — grep for imports to determine which is used where

**What NOT to touch**: `PlanTab.svelte`, `FinalizeTab.svelte`, any state management stores, backend code.

**Risk level**: Low

**Validation steps**:
- `npm run build` passes
- `npx svelte-check` passes
- Manual: open component detail view, verify asset type labels render correctly

**Acceptance criteria**:
- Single source of truth for asset type labels
- No duplicate `typeLabels` definitions
- All frontend builds pass

---

## Batch B3: Backend File Organization

**Goal**: Split overgrown backend files for navigability without changing behavior.

**Why together**: All are file-level splits within the same package — no API or behavior change.

**Files/modules in scope**:
- `internal/service/service.go` → split into:
  - `service.go` (struct, constructor, preferences, utilities)
  - `service_components.go` (component CRUD, inventory, attributes)
  - `service_projects.go` (project CRUD, requirements, planning)
  - `service_candidates.go` (part candidates, offers, resolution)
  - `service_sourcing.go` (sourcing orchestration)
  - `service_assets.go` (asset management)
- `internal/service/service_test.go` → split into:
  - `stubs_test.go` (test stubs)
  - `service_components_test.go`
  - `service_projects_test.go`
  - `service_candidates_test.go`
- `internal/phoneintake/page.go` — extract HTML template to `page.html` with `//go:embed`

**Exact changes**:
1. For `service.go`: create new files and move method blocks. Keep all imports. Do NOT rename anything.
2. For `service_test.go`: move stub structs to `stubs_test.go`, split test functions by domain.
3. For `page.go`: create `page.html`, embed it with `//go:embed page.html`, remove the string literal.

**What NOT to touch**: Method signatures, types, domain package, app package, repository layer.

**Risk level**: Zero (file splits in same package are invisible to Go)

**Validation steps**:
- `go build ./...`
- `go test ./...`
- Manual: verify phone intake page loads correctly (start intake server, visit URL)

**Acceptance criteria**:
- `service.go` is under 200 lines (struct + constructor + preferences)
- Each split file is self-contained and under 400 lines
- All tests pass

---

## Batch B4: Frontend Modernization

**Goal**: Migrate legacy stores to runes, reduce state fragmentation in large components.

**Why together**: All are frontend-only changes that modernize reactive patterns.

**Files/modules in scope**:
- `frontend/src/lib/components/componentsWorkspaceStore.ts` — migrate to `.svelte.ts` runes
- `frontend/src/lib/projects/projectsWorkspaceStore.ts` — migrate to `.svelte.ts` runes
- `frontend/src/lib/launcher/launcherWorkspaceStore.ts` — migrate to `.svelte.ts` runes
- `frontend/src/lib/preferences/SuppliersSettingsPage.svelte` — consolidate supplier state

**Exact changes**:
1. For each store file:
   - Rename to `.svelte.ts`
   - Replace `writable()` with `$state()`
   - Replace `derived()` with `$derived()`
   - Remove `get()` calls — use direct property access
   - Update consuming components to read properties directly instead of `$store` syntax
2. For SuppliersSettingsPage: refactor 9 vars into keyed supplier object

**What NOT to touch**: `PlanTab.svelte`, `FinalizeTab.svelte` (complex, separate batch). Backend code.

**Risk level**: Medium — runes migration changes subscription semantics

**Validation steps**:
- `npm run build`
- `npx svelte-check`
- Manual: navigate to Components, Projects, Launcher, Preferences → Suppliers. Verify all state updates correctly.

**Acceptance criteria**:
- No imports from `svelte/store` in any `.ts` file
- All store consumers use direct property access
- No regressions in component/project/launcher navigation

---

## Batch B5: Backend Logic Consolidation

**Goal**: Reduce overlapping service abstractions and strengthen invariant enforcement.

**Why together**: All require careful behavioral analysis with existing test coverage.

**Files/modules in scope**:
- `internal/service/service.go` (or the split files from B3):
  - Extract `setResolutionFromCandidate()` / `clearResolution()` helpers
  - Unify `findOrCreateComponentFromOffer` and `ResolveComponentFromOffer`
  - Add logging to `hydrateCandidates()` on error
  - Evaluate whether `SelectComponentForRequirement` is still needed
- `internal/app/app_projects.go`:
  - Evaluate whether `CreateProject` (without disk) is still called
  - If not, remove it

**Exact changes**:
1. Extract resolution invariant into two helpers:
   ```go
   func (s *Service) setResolutionForCandidate(ctx, requirementID, candidate) error
   func (s *Service) clearResolutionForRequirement(ctx, requirementID) error
   ```
2. Replace inline resolution logic in `SetPreferredCandidate`, `DemotePreferredCandidate`, `RemovePartCandidate`, `ImportProviderCandidate`, `AddProviderCandidate`
3. Make `ResolveComponentFromOffer` call `findOrCreateComponentFromOffer` internally
4. Add `log.Printf("warning: ...")` in `hydrateCandidates` when component load fails

**What NOT to touch**: Repository interfaces, domain types, frontend code.

**Risk level**: Medium — behavioral changes require thorough test verification

**Validation steps**:
- `go test ./internal/service/ -v -count=1`
- `go test ./... -count=1`
- Manual: test full plan/finalize workflow — add candidate, set preferred, demote, remove, import offer

**Acceptance criteria**:
- Resolution invariant enforced from exactly 2 helper functions
- `hydrateCandidates` logs warnings instead of silent `continue`
- All existing tests pass

---

## Batch B6: Low-Priority Polish

**Goal**: Minor improvements that reduce cognitive load but don't affect behavior.

**Files in scope**:
- `internal/sourcing/types.go` — extract scoring constants
- `internal/providers/easyeda/convert_symbol.go` + `convert_footprint.go` — unify `pxToMM`/`eeToMM`
- `internal/app/types.go` — add section comments (optional)

**Risk level**: Zero

---

## Batch B7: Structural Refactors (Defer)

**Goal**: Larger structural changes that require data migrations or significant API changes.

**Files in scope**:
- `internal/domain/project.go` — migrate `SelectedComponentID` → `Resolution` exclusively
- `internal/domain/repository.go` — split `ProjectRepository` interface

**Risk level**: High — database migration + all callers

**NOT recommended for this cleanup pass. Document and defer.**

---

# 4. Recommended Execution Order

1. **B1: Dead Code and Obvious Duplicates** — Safest, highest ROI. Zero risk of breakage. Removes noise from codebase. Should be done first to establish a clean baseline.

2. **B2: Frontend Constants and Pattern Extraction** — Low risk, reduces duplication in the most-edited UI components. Quick wins that make subsequent UI work easier.

3. **B3: Backend File Organization** — Zero risk (same-package file splits). Makes the service layer navigable, which is prerequisite for B5.

4. **B4: Frontend Modernization** — Medium risk but important. Runes migration eliminates tech debt from having two reactive systems. Better to do before adding new features.

5. **B5: Backend Logic Consolidation** — Medium risk. Depends on B3 for navigability. Reduces the most dangerous duplication (invariant enforcement). Well-covered by existing tests.

6. **B6: Low-Priority Polish** — Zero risk but low ROI. Do whenever convenient.

7. **B7: Structural Refactors** — High risk, requires careful planning. Defer to a dedicated session.

---

# 5. Coding-Agent Handoff Plan

---

## Agent Instructions for B1: Dead Code and Obvious Duplicates

**Objective**: Remove dead files, untrack build artifacts, consolidate project creation helper.

**Files to inspect first**:
- `internal/startup/` — confirm empty
- `frontend/src/stubs/` — determine which stub is imported via `tsconfig.json`/`vite.config.ts`
- `internal/app/app_kicad.go` lines 79–111 — read `createProjectDiskState`
- `internal/app/app_projects.go` lines 68–177 — read `CreateBlankProject` and `CreateProjectWithDisk`

**Required changes**:
1. Delete `internal/startup/` directory
2. Delete `frontend/vite.config.ts.timestamp-*.mjs`
3. Determine which frontend stub is unused and delete it
4. Move `createProjectDiskState` function from `internal/app/app_kicad.go` to `internal/app/app_projects.go`
5. Rewrite `CreateBlankProject` body to:
   ```go
   projectID := newID()
   projectDir, err := createProjectDiskState(projectID, "Untitled Project", "")
   if err != nil {
       return ProjectResponse{}, err
   }
   p, err := a.svc.CreateProject(context.Background(), domain.Project{
       ID: projectID, Name: "Untitled Project", Description: "",
   })
   if err != nil {
       _ = os.RemoveAll(projectDir)
       return ProjectResponse{}, err
   }
   if a.launcher != nil {
       _ = a.launcher.TouchProject(p.ID, p.Name, p.Description)
   }
   return projectToResponse(p), nil
   ```
6. Rewrite `CreateProjectWithDisk` similarly, using `req.Name` and `req.Description`
7. Run `git rm --cached bin/trace build/bin/component-manager frontend/wailsjs/go/app/App.d.ts frontend/wailsjs/go/app/App.js frontend/wailsjs/go/models.ts frontend/wailsjs/runtime/package.json frontend/wailsjs/runtime/runtime.d.ts frontend/wailsjs/runtime/runtime.js` (requires user confirmation before executing)

**Safety checks**:
- `go build ./...`
- `go test ./...`
- `npm run build` in `frontend/`

**Tests to run**:
- `go test ./internal/app/... -v`

**Manual UI flows**: Create a new blank project, create a named project, import a KiCad project — all should create disk state correctly.

---

## Agent Instructions for B2: Frontend Constants Extraction

**Objective**: Extract duplicated `typeLabels` into shared constants.

**Files to inspect first**:
- `frontend/src/lib/components/AssetsTab.svelte` (line 42)
- `frontend/src/lib/components/AddFromFileModal.svelte` (line 96)
- `frontend/src/lib/components/AssetSection.svelte` (line 36)

**Required changes**:
1. Create `frontend/src/lib/constants.ts`:
   ```ts
   export const ASSET_TYPE_LABELS: Record<string, string> = {
     symbol: 'Symbol',
     footprint: 'Footprint',
     '3d_model': '3D Model',
     datasheet: 'Datasheet',
   };
   ```
2. In each of the 3 files: replace the local `typeLabels` definition with `import { ASSET_TYPE_LABELS } from '../constants';` (adjust path as needed) and rename usage from `typeLabels` to `ASSET_TYPE_LABELS`.
3. Also check `DetailsTab.svelte` and `SpecificationsSection.svelte` to determine if they're both used or one is dead:
   - `grep -rn "DetailsTab\|SpecificationsSection" frontend/src/ --include="*.svelte"`
   - If only one is imported anywhere, the other is dead code — delete it.

**Safety checks**:
- `npm run build`
- `npx svelte-check`

**Tests to run**: None (no unit tests for these components).

**Manual UI flows**: Open component detail, switch to assets tab, verify labels show "Symbol", "Footprint", "3D Model", "Datasheet" correctly. Open add-from-file modal, verify same labels.

---

## Agent Instructions for B3: Backend File Split

**Objective**: Split `service.go` (1,299 lines) into domain-specific files.

**Files to inspect first**:
- `internal/service/service.go` — read entirely to understand method groupings
- `internal/service/service_test.go` — identify test groupings

**Required changes**:
1. Create these files in `internal/service/`:
   - `service_components.go` — move methods: `CreateComponent`, `GetComponent`, `UpdateComponentMetadata`, `ReplaceComponentAttributes`, `FindComponents`, `DeleteComponent`, `UpsertAttributeDefinition`, `SyncCanonicalAttributeDefinitions`, `UpdateComponentInventory`, `AdjustComponentQuantity`, `StampInventory`
   - `service_projects.go` — move methods: `CreateProject`, `GetProject`, `ListProjects`, `DeleteProject`, `UpdateProject`, `ReplaceProjectRequirements`, `AddProjectRequirements`, `SetProjectImportMetadata`, `PlanProject`, matching helpers (`prepareProjectRequirements`, `matchesRequirement`, `matchesMetadataConstraint`, `valueMatches`, `totalQuantity`, `buildSelectedPart`, `computeExportReadiness`)
   - `service_candidates.go` — move methods: `AddPartCandidate`, `SetPreferredCandidate`, `DemotePreferredCandidate`, `RemovePartCandidate`, `AddLocalComponentAsCandidateAndSetPreferred`, `SaveSupplierOfferForRequirement`, `ImportSupplierOffer`, `AddProviderCandidate`, `ImportProviderCandidate`, `RemoveSavedSupplierOffer`, `ListPartCandidates`, `ListSavedSupplierOffers`, matching helpers (`hydrateCandidates`, `findOrCreateComponentFromOffer`)
   - `service_sourcing.go` — move methods: `resolveSourcingService`, `SourceRequirement`, `SourceRequirementFromProvider`, `SourcingProviders`, `LookupVendorPartID`, `ResolveComponentFromOffer`, `categoryFromCatalog`
   - `service_assets.go` — move methods: all asset-related methods (`CreateComponentAsset`, `ListComponentAssets`, etc.)
   - Keep in `service.go`: `Service` struct, `New()`, `Set*()` builders, `SelectComponentForRequirement`, `ClearSelectedComponentForRequirement`, `newID`, `componentDefinitionLabel`

2. For each new file, add the correct `package service` declaration and required imports.

3. Split `service_test.go`:
   - `stubs_test.go` — all `stub*Repo` structs
   - Keep tests in the original file or split by domain (optional)

4. For `phoneintake/page.go`:
   - Create `internal/phoneintake/page.html` with the HTML content
   - Replace string literal with `//go:embed page.html` + `var scannerPage string`

**What NOT to do**: Do not rename any functions, types, or variables. Do not change any method signatures. Do not refactor logic.

**Safety checks**:
- `go build ./...`
- `go test ./...`
- `go vet ./...`

**Tests to run**:
- `go test ./internal/service/ -v -count=1`
- `go test ./internal/phoneintake/ -v -count=1` (if tests exist)

---

## Agent Instructions for B4: Frontend Store Migration

**Objective**: Migrate legacy `svelte/store` to Svelte 5 runes.

**Files to inspect first**:
- `frontend/src/lib/components/componentsWorkspaceStore.ts` — read fully
- `frontend/src/lib/projects/projectsWorkspaceStore.ts` — read fully
- `frontend/src/lib/launcher/launcherWorkspaceStore.ts` — read fully
- All `.svelte` files that import from these stores (grep for the import paths)

**Required changes**:
1. Rename each store file from `.ts` to `.svelte.ts`
2. Rewrite using runes:
   ```ts
   // Before (legacy)
   import { writable, derived, get } from 'svelte/store';
   const items = writable<Item[]>([]);
   const selected = writable<string | null>(null);
   
   // After (runes)
   let items = $state<Item[]>([]);
   let selected = $state<string | null>(null);
   let filtered = $derived(items.filter(...));
   ```
3. Update all consuming `.svelte` components:
   - Replace `$storeName` with `store.propertyName`
   - Replace `storeName.set(value)` with `store.propertyName = value`
   - Replace `get(storeName)` with `store.propertyName`
4. Update import paths in consuming files to use `.svelte.ts` extension

**Safety checks**:
- `npm run build`
- `npx svelte-check`

**Manual UI flows**: Full navigation test — launch app, switch between components/projects, select items, filter, create new items.

---

## Agent Instructions for B5: Backend Logic Consolidation

**Objective**: Centralize resolution invariant, unify offer-to-component deduplication.

**Files to inspect first**:
- The candidate/offer methods in `service_candidates.go` (or `service.go` if B3 not yet done)
- All callers of `SetRequirementResolution`
- `findOrCreateComponentFromOffer` and `ResolveComponentFromOffer` — compare side by side
- `hydrateCandidates` — find the silent `continue`

**Required changes**:
1. Create two private helpers in the candidates file:
   ```go
   func (s *Service) applyPreferredResolution(ctx context.Context, requirementID, componentID string) error {
       return s.projectRepo.SetRequirementResolution(ctx, requirementID,
           &domain.RequirementResolution{Kind: domain.ResolutionInternalComponent, ComponentID: &componentID})
   }
   
   func (s *Service) clearPreferredResolution(ctx context.Context, requirementID string) error {
       return s.projectRepo.SetRequirementResolution(ctx, requirementID, nil)
   }
   ```
2. Replace all inline `SetRequirementResolution` calls in candidate methods with these helpers
3. Make `ResolveComponentFromOffer` call `findOrCreateComponentFromOffer` internally (or vice versa)
4. In `hydrateCandidates`, add logging: `log.Printf("warning: cannot load component %s for candidate %s: %v", c.ComponentID, c.ID, err)`
5. Check if `SelectComponentForRequirement` is called from frontend:
   - `grep -rn "selectComponentForRequirement\|SelectComponentForRequirement" frontend/src/`
   - If not called, mark as deprecated with a TODO comment

**Safety checks**:
- `go test ./internal/service/ -v -count=1`
- `go test ./... -count=1`

**Manual UI flows**: Complete plan/finalize workflow test — add local candidate, set preferred, demote, remove. Import supplier offer. Verify all resolution states update correctly.

---

# 6. Quick Wins

These can be done immediately with near-zero risk:

1. **Delete `internal/startup/` empty directory**
2. **Delete `frontend/vite.config.ts.timestamp-*.mjs`** — Vite temp file
3. **Extract `ASSET_TYPE_LABELS` constant** — 3 files with identical map
4. **Move `createProjectDiskState`** — already exists in `app_kicad.go`, duplicated inline in `app_projects.go`
5. **Unify `pxToMM`/`eeToMM`** — identical formula, different names

---

# 7. Watch-Outs

1. **`git rm --cached` of wailsjs files** — Wails build may depend on these being committed. Test that `wails build` regenerates them from scratch before removing from tracking. If CI caches the git checkout, the build may break.

2. **Runes store migration (B4)** — The `$store` subscription syntax is fundamentally different from runes. Every consumer of legacy stores MUST be updated simultaneously. Partial migration will cause runtime errors. Plan to grep for ALL imports and update them in one pass.

3. **`DetailsTab.svelte` vs `SpecificationsSection.svelte`** — Before assuming one is dead, verify both aren't used in different contexts (e.g., one in a modal, one in the main view). Deleting the wrong one breaks the UI.

4. **Service file split (B3)** — While moving methods between files is safe in Go, be careful with unexported helpers that are used by methods in different logical groups. Map the call graph before splitting.

5. **Resolution invariant refactor (B5)** — The dual `SelectedComponentID`/`Resolution` system means the new helpers must maintain backward compatibility with both fields. `NormalizeResolution()` must still be called at all the right points.

6. **`SelectComponentForRequirement` deprecation** — If the frontend still uses this, removing it breaks the app. Verify with a frontend grep before any changes.

7. **`phoneintake/page.go` → `page.html` extraction** — The HTML template uses Go template syntax (`{{.Variables}}`). Verify the embed works with `template.New().Parse()` at runtime, not just string substitution.

---

# 8. Final Prioritized Checklist

From highest value / lowest risk → lowest value / highest risk:

| # | Item | Risk | Value | Batch |
|---|------|------|-------|-------|
| 1 | Delete empty `internal/startup/` directory | Zero | Low | B1 |
| 2 | Delete Vite timestamp temp file | Zero | Low | B1 |
| 3 | Consolidate `createProjectDiskState` usage | Low | High | B1 |
| 4 | Extract `ASSET_TYPE_LABELS` to shared constant | Zero | Medium | B2 |
| 5 | Unify `pxToMM`/`eeToMM` in EasyEDA provider | Zero | Low | B6 |
| 6 | Extract sourcing score constants | Zero | Low | B6 |
| 7 | Split `service.go` into domain files | Zero | High | B3 |
| 8 | Split `service_test.go` stubs | Zero | Medium | B3 |
| 9 | Extract `phoneintake/page.html` via embed | Low | Medium | B3 |
| 10 | Determine & consolidate DetailsTab/SpecificationsSection | Low | Medium | B2 |
| 11 | Migrate workspace stores to runes | Medium | Medium | B4 |
| 12 | Consolidate SuppliersSettingsPage state | Low | Low | B4 |
| 13 | Centralize resolution invariant helpers | Medium | High | B5 |
| 14 | Unify `findOrCreateComponentFromOffer`/`ResolveComponentFromOffer` | Medium | Medium | B5 |
| 15 | Add logging to `hydrateCandidates` | Low | Medium | B5 |
| 16 | Evaluate `SelectComponentForRequirement` deprecation | Medium | Medium | B5 |
| 17 | Evaluate `CreateProject` (no disk) removal | Medium | Low | B5 |
| 18 | `git rm --cached` build artifacts | Medium | Medium | B1 |
| 19 | Delete unused frontend stub | Low | Low | B1 |
| 20 | Split `PlanTab`/`FinalizeTab` shared patterns | Medium | Medium | B4 |
| 21 | Split `ProjectRepository` interface | High | Medium | B7 |
| 22 | Migrate `SelectedComponentID` → `Resolution` exclusively | High | High | B7 |

---

# Implementation Punch List

## Task T01
- **Objective**: Delete empty `internal/startup/` directory
- **Files to edit**: none
- **Files to delete**: `internal/startup/` (directory)
- **Tests to run**: `go build ./...`
- **Manual verification**: none
- **Dependencies**: none

## Task T02
- **Objective**: Delete Vite timestamp temp file from workspace
- **Files to edit**: none
- **Files to delete**: `frontend/vite.config.ts.timestamp-1775378745383-ceef56153495a.mjs`
- **Tests to run**: `npm run build` in `frontend/`
- **Manual verification**: none
- **Dependencies**: none

## Task T03
- **Objective**: Consolidate project disk-state creation into shared helper
- **Files to edit**: `internal/app/app_projects.go`, `internal/app/app_kicad.go`
- **Files to delete**: none
- **Tests to run**: `go test ./internal/app/... -v`, `go build ./...`
- **Manual verification**: Create blank project, create named project, import KiCad project — all should create `~/.trace/projects/<id>/project.json`
- **Dependencies**: none

## Task T04
- **Objective**: Determine and delete unused frontend stub file
- **Files to edit**: none
- **Files to delete**: One of: `frontend/src/stubs/app-environment.ts` or `frontend/src/stubs/$app/environment.js`
- **Tests to run**: `npm run build`, `npx svelte-check`
- **Manual verification**: Check `tsconfig.json` and `vite.config.ts` for path aliases pointing to stubs
- **Dependencies**: none

## Task T05
- **Objective**: Untrack committed binary and generated files from git
- **Files to edit**: none (git index only)
- **Files to delete**: none (files remain on disk, just untracked)
- **Tests to run**: `go build ./...`, `npm run build`
- **Manual verification**: `git ls-files bin/ build/bin/ frontend/wailsjs/` should return empty after commit. Verify `wails build` regenerates wailsjs files.
- **Dependencies**: none (but requires user confirmation before git operations)

## Task T06
- **Objective**: Extract `ASSET_TYPE_LABELS` constant to shared frontend module
- **Files to edit**: Create `frontend/src/lib/constants.ts`; edit `frontend/src/lib/components/AssetsTab.svelte`, `frontend/src/lib/components/AddFromFileModal.svelte`, `frontend/src/lib/components/AssetSection.svelte`
- **Files to delete**: none
- **Tests to run**: `npm run build`, `npx svelte-check`
- **Manual verification**: Open component detail view → Assets tab. Verify "Symbol", "Footprint", "3D Model", "Datasheet" labels render.
- **Dependencies**: none

## Task T07
- **Objective**: Determine if `DetailsTab.svelte` or `SpecificationsSection.svelte` is dead code
- **Files to edit**: depends on finding
- **Files to delete**: the unused one (if any)
- **Tests to run**: `npm run build`, `npx svelte-check`
- **Manual verification**: `grep -rn "DetailsTab\|SpecificationsSection" frontend/src/ --include="*.svelte"` to find importers. Navigate to component detail and verify the surviving component renders correctly.
- **Dependencies**: none

## Task T08
- **Objective**: Split `service.go` into domain-specific files
- **Files to edit**: `internal/service/service.go` (remove moved methods); create `internal/service/service_components.go`, `service_projects.go`, `service_candidates.go`, `service_sourcing.go`, `service_assets.go`
- **Files to delete**: none
- **Tests to run**: `go build ./...`, `go test ./internal/service/ -v -count=1`, `go vet ./...`
- **Manual verification**: none (pure file reorganization)
- **Dependencies**: none

## Task T09
- **Objective**: Split `service_test.go` — extract stubs to separate file
- **Files to edit**: `internal/service/service_test.go` (remove stubs); create `internal/service/stubs_test.go`
- **Files to delete**: none
- **Tests to run**: `go test ./internal/service/ -v -count=1`
- **Manual verification**: none
- **Dependencies**: T08 (easier to split tests after splitting source)

## Task T10
- **Objective**: Extract phone intake HTML to embedded file
- **Files to edit**: `internal/phoneintake/page.go`; create `internal/phoneintake/page.html`
- **Files to delete**: none
- **Tests to run**: `go build ./...`
- **Manual verification**: Start phone intake server, navigate to the intake URL in browser, verify QR scanner page loads and functions correctly.
- **Dependencies**: none

## Task T11
- **Objective**: Unify `pxToMM` and `eeToMM` into single conversion function
- **Files to edit**: `internal/providers/easyeda/convert_symbol.go`, `internal/providers/easyeda/convert_footprint.go`; optionally create `internal/providers/easyeda/units.go`
- **Files to delete**: none
- **Tests to run**: `go build ./...`, `go test ./internal/providers/easyeda/...` (if tests exist)
- **Manual verification**: Import an EasyEDA component, verify symbol and footprint dimensions are correct.
- **Dependencies**: none

## Task T12
- **Objective**: Extract sourcing score magic numbers into named constants
- **Files to edit**: `internal/sourcing/types.go`
- **Files to delete**: none
- **Tests to run**: `go test ./internal/sourcing/ -v`
- **Manual verification**: none (behavior unchanged)
- **Dependencies**: none

## Task T13
- **Objective**: Migrate `componentsWorkspaceStore` to Svelte 5 runes
- **Files to edit**: Rename `frontend/src/lib/components/componentsWorkspaceStore.ts` → `.svelte.ts`; rewrite internals; update all consuming `.svelte` files
- **Files to delete**: `frontend/src/lib/components/componentsWorkspaceStore.ts` (replaced by `.svelte.ts`)
- **Tests to run**: `npm run build`, `npx svelte-check`
- **Manual verification**: Navigate to Components workspace, filter components, select a component, verify detail panel loads, create/edit/delete component.
- **Dependencies**: none

## Task T14
- **Objective**: Migrate `projectsWorkspaceStore` to Svelte 5 runes
- **Files to edit**: Rename `frontend/src/lib/projects/projectsWorkspaceStore.ts` → `.svelte.ts`; rewrite internals; update consumers
- **Files to delete**: `frontend/src/lib/projects/projectsWorkspaceStore.ts`
- **Tests to run**: `npm run build`, `npx svelte-check`
- **Manual verification**: Navigate to Projects workspace, select projects, verify detail loads.
- **Dependencies**: none

## Task T15
- **Objective**: Migrate `launcherWorkspaceStore` to Svelte 5 runes
- **Files to edit**: Rename `frontend/src/lib/launcher/launcherWorkspaceStore.ts` → `.svelte.ts`; rewrite; update consumers
- **Files to delete**: `frontend/src/lib/launcher/launcherWorkspaceStore.ts`
- **Tests to run**: `npm run build`, `npx svelte-check`
- **Manual verification**: Open launcher, filter projects, verify pinned/unpinned ordering, open project.
- **Dependencies**: none

## Task T16
- **Objective**: Centralize resolution invariant into two service helpers
- **Files to edit**: `internal/service/service_candidates.go` (or `service.go` if B3 not done)
- **Files to delete**: none
- **Tests to run**: `go test ./internal/service/ -v -count=1`
- **Manual verification**: Full candidate workflow: add → set preferred → demote → remove. Import offer → set preferred. Verify requirement resolution updates correctly at each step.
- **Dependencies**: T08 (service file split makes this easier to locate)

## Task T17
- **Objective**: Unify component-from-offer deduplication logic
- **Files to edit**: `internal/service/service_candidates.go` and `service_sourcing.go` (or `service.go`)
- **Files to delete**: none
- **Tests to run**: `go test ./internal/service/ -v -count=1`
- **Manual verification**: Import supplier offer for existing component (should reuse). Import for new component (should create). Phone intake scan for existing component (should reuse).
- **Dependencies**: T08, T16

## Task T18
- **Objective**: Add warning logging to `hydrateCandidates` on component load failure
- **Files to edit**: `internal/service/service_candidates.go` (or `service.go`)
- **Files to delete**: none
- **Tests to run**: `go test ./internal/service/ -v -count=1`
- **Manual verification**: none (logging only)
- **Dependencies**: T08

## Task T19
- **Objective**: Evaluate and document `SelectComponentForRequirement` deprecation status
- **Files to edit**: `internal/service/service.go` or split file (add TODO/deprecation comment)
- **Files to delete**: none
- **Tests to run**: none
- **Manual verification**: `grep -rn "selectComponentForRequirement\|SelectComponentForRequirement" frontend/src/` — if no results, comment as deprecated
- **Dependencies**: T08

## Task T20
- **Objective**: Consolidate `SuppliersSettingsPage` state into keyed supplier object
- **Files to edit**: `frontend/src/lib/preferences/SuppliersSettingsPage.svelte`
- **Files to delete**: none
- **Tests to run**: `npm run build`, `npx svelte-check`
- **Manual verification**: Open Preferences → Suppliers. Toggle each supplier enable/disable. Edit API keys. Save. Verify all persist correctly.
- **Dependencies**: none
