package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"componentmanager/internal/domain"
)

type ProjectRepository struct {
	store *Store
}

type projectRequirementRow struct {
	ID                    string          `db:"id"`
	ProjectID             string          `db:"project_id"`
	Name                  string          `db:"name"`
	Category              domain.Category `db:"category"`
	Quantity              int             `db:"quantity"`
	SelectedComponentID   *string         `db:"selected_component_id"`
	ResolutionKind        *string         `db:"resolution_kind"`
	ResolutionComponentID *string         `db:"resolution_component_id"`
}

func requirementRowFromDomain(req domain.ProjectRequirement) projectRequirementRow {
	req.NormalizeResolution()
	row := projectRequirementRow{
		ID:                  req.ID,
		ProjectID:           req.ProjectID,
		Name:                req.Name,
		Category:            req.Category,
		Quantity:            req.Quantity,
		SelectedComponentID: req.SelectedComponentID,
	}
	if req.Resolution != nil {
		kind := string(req.Resolution.Kind)
		row.ResolutionKind = &kind
		row.ResolutionComponentID = req.Resolution.ComponentID
	}
	return row
}

func (r projectRequirementRow) toDomain() domain.ProjectRequirement {
	req := domain.ProjectRequirement{
		ID:                  r.ID,
		ProjectID:           r.ProjectID,
		Name:                r.Name,
		Category:            r.Category,
		Quantity:            r.Quantity,
		SelectedComponentID: r.SelectedComponentID,
	}
	if r.ResolutionKind != nil {
		resolution := &domain.RequirementResolution{
			Kind:        domain.RequirementResolutionKind(*r.ResolutionKind),
			ComponentID: r.ResolutionComponentID,
		}
		req.Resolution = resolution
	}
	req.NormalizeResolution()
	return req
}

func NewProjectRepository(store *Store) *ProjectRepository {
	return &ProjectRepository{store: store}
}

func (r *ProjectRepository) CreateProject(ctx context.Context, project domain.Project) (domain.Project, error) {
	tx, err := r.store.db.BeginTxx(ctx, nil)
	if err != nil {
		return domain.Project{}, err
	}
	defer tx.Rollback()

	if err := tx.QueryRowxContext(ctx, `
		insert into projects(id, name, description, import_source_type, import_source_path, imported_at)
		values ($1, $2, $3, $4, $5, $6)
		returning created_at, updated_at, import_source_type, import_source_path, imported_at
	`, project.ID, project.Name, project.Description, project.ImportSourceType, project.ImportSourcePath, project.ImportedAt,
	).Scan(&project.CreatedAt, &project.UpdatedAt, &project.ImportSourceType, &project.ImportSourcePath, &project.ImportedAt); err != nil {
		return domain.Project{}, err
	}

	for _, requirement := range project.Requirements {
		row := requirementRowFromDomain(requirement)
		if _, err := tx.ExecContext(ctx, `
			insert into project_requirements(id, project_id, name, category, quantity, selected_component_id, resolution_kind, resolution_component_id)
			values ($1, $2, $3, $4, $5, $6, $7, $8)
		`, row.ID, row.ProjectID, row.Name, row.Category, row.Quantity, row.SelectedComponentID, row.ResolutionKind, row.ResolutionComponentID); err != nil {
			return domain.Project{}, err
		}

		for _, constraint := range requirement.Constraints {
			if err := validateConstraint(constraint); err != nil {
				return domain.Project{}, err
			}

			if _, err := tx.ExecContext(ctx, `
				insert into requirement_constraints(requirement_id, key, value_type, operator, text_value, number_value, bool_value, unit)
				values ($1, $2, $3, $4, $5, $6, $7, $8)
			`, requirement.ID, constraint.Key, constraint.ValueType, constraint.Operator, constraint.Text, constraint.Number, constraint.Bool, constraint.Unit); err != nil {
				return domain.Project{}, err
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return domain.Project{}, err
	}

	return project, nil
}

func (r *ProjectRepository) GetProject(ctx context.Context, id string) (domain.Project, error) {
	var project domain.Project
	if err := r.store.db.GetContext(ctx, &project, `
		select id, name, description, import_source_type, import_source_path, imported_at, created_at, updated_at
		from projects
		where id = $1
	`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Project{}, domain.ErrNotFound{ID: id}
		}
		return domain.Project{}, err
	}

	projects, err := r.hydrateProjects(ctx, []domain.Project{project})
	if err != nil {
		return domain.Project{}, err
	}
	return projects[0], nil
}

func (r *ProjectRepository) getConstraintsByRequirementIDs(ctx context.Context, requirementIDs []string) (map[string][]domain.RequirementConstraint, error) {
	result := make(map[string][]domain.RequirementConstraint)
	if len(requirementIDs) == 0 {
		return result, nil
	}

	query, args, err := sqlx.In(`
		select requirement_id, key, value_type, operator, text_value, number_value, bool_value, unit
		from requirement_constraints
		where requirement_id in (?)
		order by key, operator
	`, requirementIDs)
	if err != nil {
		return nil, err
	}

	var rows []constraintRow
	if err := r.store.db.SelectContext(ctx, &rows, r.store.db.Rebind(query), args...); err != nil {
		return nil, err
	}

	for _, row := range rows {
		result[row.RequirementID] = append(result[row.RequirementID], row.toRequirementConstraint())
	}
	return result, nil
}

func (r *ProjectRepository) ListProjects(ctx context.Context) ([]domain.Project, error) {
	var projects []domain.Project
	if err := r.store.db.SelectContext(ctx, &projects, `
		select id, name, description, import_source_type, import_source_path, imported_at, created_at, updated_at
		from projects
		order by name
	`); err != nil {
		return nil, err
	}
	return r.hydrateProjects(ctx, projects)
}

func (r *ProjectRepository) hydrateProjects(ctx context.Context, projects []domain.Project) ([]domain.Project, error) {
	if len(projects) == 0 {
		return projects, nil
	}

	projectIDs := make([]string, len(projects))
	for i, p := range projects {
		projectIDs[i] = p.ID
	}

	reqQuery, reqArgs, err := sqlx.In(`
		select id, project_id, name, category, quantity, selected_component_id, resolution_kind, resolution_component_id
		from project_requirements
		where project_id in (?)
		order by project_id, name
	`, projectIDs)
	if err != nil {
		return nil, err
	}
	var requirementRows []projectRequirementRow
	if err := r.store.db.SelectContext(ctx, &requirementRows, r.store.db.Rebind(reqQuery), reqArgs...); err != nil {
		return nil, err
	}
	requirements := make([]domain.ProjectRequirement, len(requirementRows))
	for i := range requirementRows {
		requirements[i] = requirementRows[i].toDomain()
	}

	if len(requirements) > 0 {
		requirementIDs := make([]string, len(requirements))
		for i, req := range requirements {
			requirementIDs[i] = req.ID
		}
		constraintsByReq, err := r.getConstraintsByRequirementIDs(ctx, requirementIDs)
		if err != nil {
			return nil, err
		}
		for i := range requirements {
			requirements[i].Constraints = constraintsByReq[requirements[i].ID]
		}
	}

	reqsByProject := make(map[string][]domain.ProjectRequirement, len(projects))
	for _, req := range requirements {
		reqsByProject[req.ProjectID] = append(reqsByProject[req.ProjectID], req)
	}
	for i := range projects {
		reqs := reqsByProject[projects[i].ID]
		if reqs == nil {
			reqs = []domain.ProjectRequirement{}
		}
		projects[i].Requirements = reqs
	}

	return projects, nil
}

func (r *ProjectRepository) UpdateProject(ctx context.Context, project domain.Project) (domain.Project, error) {
	if err := r.store.db.QueryRowxContext(ctx, `
		update projects
		set name = $1, description = $2, updated_at = now()
		where id = $3
		returning import_source_type, import_source_path, imported_at, created_at, updated_at
	`, project.Name, project.Description, project.ID,
	).Scan(&project.ImportSourceType, &project.ImportSourcePath, &project.ImportedAt, &project.CreatedAt, &project.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Project{}, domain.ErrNotFound{ID: project.ID}
		}
		return domain.Project{}, err
	}
	return project, nil
}

func (r *ProjectRepository) ReplaceProjectRequirements(ctx context.Context, projectID string, requirements []domain.ProjectRequirement) error {
	tx, err := r.store.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// delete constraints for all of this project's existing requirements
	if _, err := tx.ExecContext(ctx, `
		delete from requirement_constraints
		where requirement_id in (
			select id from project_requirements where project_id = $1
		)
	`, projectID); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `
		delete from project_requirements where project_id = $1
	`, projectID); err != nil {
		return err
	}

	for _, req := range requirements {
		row := requirementRowFromDomain(req)
		if _, err := tx.ExecContext(ctx, `
			insert into project_requirements(id, project_id, name, category, quantity, selected_component_id, resolution_kind, resolution_component_id)
			values ($1, $2, $3, $4, $5, $6, $7, $8)
		`, row.ID, row.ProjectID, row.Name, row.Category, row.Quantity, row.SelectedComponentID, row.ResolutionKind, row.ResolutionComponentID); err != nil {
			return err
		}

		for _, constraint := range req.Constraints {
			if err := validateConstraint(constraint); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `
				insert into requirement_constraints(requirement_id, key, value_type, operator, text_value, number_value, bool_value, unit)
				values ($1, $2, $3, $4, $5, $6, $7, $8)
			`, req.ID, constraint.Key, constraint.ValueType, constraint.Operator, constraint.Text, constraint.Number, constraint.Bool, constraint.Unit); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func (r *ProjectRepository) AddProjectRequirements(ctx context.Context, projectID string, requirements []domain.ProjectRequirement) error {
	tx, err := r.store.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, req := range requirements {
		row := requirementRowFromDomain(req)
		if _, err := tx.ExecContext(ctx, `
			insert into project_requirements(id, project_id, name, category, quantity, selected_component_id, resolution_kind, resolution_component_id)
			values ($1, $2, $3, $4, $5, $6, $7, $8)
		`, row.ID, projectID, row.Name, row.Category, row.Quantity, row.SelectedComponentID, row.ResolutionKind, row.ResolutionComponentID); err != nil {
			return err
		}

		for _, constraint := range req.Constraints {
			if err := validateConstraint(constraint); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `
				insert into requirement_constraints(requirement_id, key, value_type, operator, text_value, number_value, bool_value, unit)
				values ($1, $2, $3, $4, $5, $6, $7, $8)
			`, req.ID, constraint.Key, constraint.ValueType, constraint.Operator, constraint.Text, constraint.Number, constraint.Bool, constraint.Unit); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func (r *ProjectRepository) SetProjectImportMetadata(ctx context.Context, projectID string, sourceType, sourcePath *string, importedAt *time.Time) error {
	res, err := r.store.db.ExecContext(ctx, `
		update projects
		set import_source_type = $1,
		    import_source_path = $2,
		    imported_at = $3,
		    updated_at = now()
		where id = $4
	`, sourceType, sourcePath, importedAt, projectID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrNotFound{ID: projectID}
	}
	return nil
}

func (r *ProjectRepository) GetRequirement(ctx context.Context, requirementID string) (domain.ProjectRequirement, error) {
	var row projectRequirementRow
	if err := r.store.db.GetContext(ctx, &row, `
		select id, project_id, name, category, quantity, selected_component_id, resolution_kind, resolution_component_id
		from project_requirements
		where id = $1
	`, requirementID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ProjectRequirement{}, domain.ErrNotFound{ID: requirementID}
		}
		return domain.ProjectRequirement{}, err
	}
	req := row.toDomain()

	constraints, err := r.getConstraintsByRequirementIDs(ctx, []string{requirementID})
	if err != nil {
		return domain.ProjectRequirement{}, err
	}
	req.Constraints = constraints[requirementID]
	return req, nil
}

func (r *ProjectRepository) SetRequirementResolution(ctx context.Context, requirementID string, resolution *domain.RequirementResolution) error {
	selectedComponentID := (*string)(nil)
	resolutionKind := (*string)(nil)
	resolutionComponentID := (*string)(nil)
	if resolution != nil {
		copyResolution := *resolution
		copyResolution.Normalize()
		if !copyResolution.IsZero() {
			kind := string(copyResolution.Kind)
			resolutionKind = &kind
			resolutionComponentID = copyResolution.ComponentID
			if copyResolution.Kind == domain.RequirementResolutionKindInternalComponent {
				selectedComponentID = copyResolution.ComponentID
			}
		}
	}
	res, err := r.store.db.ExecContext(ctx, `
		update project_requirements
		set selected_component_id = $1,
		    resolution_kind = $2,
		    resolution_component_id = $3
		where id = $4
	`, selectedComponentID, resolutionKind, resolutionComponentID, requirementID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrNotFound{ID: requirementID}
	}
	return nil
}

func (r *ProjectRepository) DeleteProject(ctx context.Context, id string) error {
	res, err := r.store.db.ExecContext(ctx, `delete from projects where id = $1`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrNotFound{ID: id}
	}
	return nil
}

// --- Part Candidates ---

func (r *ProjectRepository) AddPartCandidate(ctx context.Context, candidate domain.ProjectPartCandidate) (domain.ProjectPartCandidate, error) {
	if err := r.store.db.QueryRowxContext(ctx, `
		insert into project_part_candidates(id, project_id, requirement_id, component_id, preferred, origin)
		values ($1, $2, $3, $4, $5, $6)
		returning created_at, updated_at
	`, candidate.ID, candidate.ProjectID, candidate.RequirementID, candidate.ComponentID, candidate.Preferred, candidate.Origin,
	).Scan(&candidate.CreatedAt, &candidate.UpdatedAt); err != nil {
		return domain.ProjectPartCandidate{}, err
	}
	return candidate, nil
}

func (r *ProjectRepository) SetPreferredCandidate(ctx context.Context, requirementID, candidateID string) error {
	tx, err := r.store.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		update project_part_candidates
		set preferred = false, updated_at = now()
		where requirement_id = $1 and preferred = true
	`, requirementID); err != nil {
		return err
	}

	res, err := tx.ExecContext(ctx, `
		update project_part_candidates
		set preferred = true, updated_at = now()
		where id = $1 and requirement_id = $2
	`, candidateID, requirementID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrNotFound{ID: candidateID}
	}

	return tx.Commit()
}

func (r *ProjectRepository) RemovePartCandidate(ctx context.Context, candidateID string) error {
	res, err := r.store.db.ExecContext(ctx, `delete from project_part_candidates where id = $1`, candidateID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrNotFound{ID: candidateID}
	}
	return nil
}

func (r *ProjectRepository) ListPartCandidates(ctx context.Context, requirementID string) ([]domain.ProjectPartCandidate, error) {
	var candidates []domain.ProjectPartCandidate
	if err := r.store.db.SelectContext(ctx, &candidates, `
		select id, project_id, requirement_id, component_id, preferred, origin, created_at, updated_at
		from project_part_candidates
		where requirement_id = $1
		order by preferred desc, created_at
	`, requirementID); err != nil {
		return nil, err
	}
	return candidates, nil
}

func (r *ProjectRepository) ListPartCandidatesByProject(ctx context.Context, projectID string) ([]domain.ProjectPartCandidate, error) {
	var candidates []domain.ProjectPartCandidate
	if err := r.store.db.SelectContext(ctx, &candidates, `
		select id, project_id, requirement_id, component_id, preferred, origin, created_at, updated_at
		from project_part_candidates
		where project_id = $1
		order by requirement_id, preferred desc, created_at
	`, projectID); err != nil {
		return nil, err
	}
	return candidates, nil
}

// --- Saved Supplier Offers ---

func (r *ProjectRepository) SaveSupplierOffer(ctx context.Context, offer domain.SavedSupplierOffer) (domain.SavedSupplierOffer, error) {
	if err := r.store.db.QueryRowxContext(ctx, `
		insert into saved_supplier_offers(id, project_id, requirement_id, provider, provider_part_id, product_url,
			manufacturer, mpn, description, package, stock, moq, unit_price, currency, linked_component_id, captured_at)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		returning created_at
	`, offer.ID, offer.ProjectID, offer.RequirementID, offer.Provider, offer.ProviderPartID, offer.ProductURL,
		offer.Manufacturer, offer.MPN, offer.Description, offer.Package, offer.Stock, offer.MOQ,
		offer.UnitPrice, offer.Currency, offer.LinkedComponentID, offer.CapturedAt,
	).Scan(&offer.CreatedAt); err != nil {
		return domain.SavedSupplierOffer{}, err
	}
	return offer, nil
}

func (r *ProjectRepository) RemoveSavedSupplierOffer(ctx context.Context, offerID string) error {
	res, err := r.store.db.ExecContext(ctx, `delete from saved_supplier_offers where id = $1`, offerID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrNotFound{ID: offerID}
	}
	return nil
}

func (r *ProjectRepository) ListSavedSupplierOffers(ctx context.Context, requirementID string) ([]domain.SavedSupplierOffer, error) {
	var offers []domain.SavedSupplierOffer
	if err := r.store.db.SelectContext(ctx, &offers, `
		select id, project_id, requirement_id, provider, provider_part_id, product_url,
			manufacturer, mpn, description, package, stock, moq, unit_price, currency,
			linked_component_id, captured_at, created_at
		from saved_supplier_offers
		where requirement_id = $1
		order by created_at desc
	`, requirementID); err != nil {
		return nil, err
	}
	return offers, nil
}

func (r *ProjectRepository) ListSavedSupplierOffersByProject(ctx context.Context, projectID string) ([]domain.SavedSupplierOffer, error) {
	var offers []domain.SavedSupplierOffer
	if err := r.store.db.SelectContext(ctx, &offers, `
		select id, project_id, requirement_id, provider, provider_part_id, product_url,
			manufacturer, mpn, description, package, stock, moq, unit_price, currency,
			linked_component_id, captured_at, created_at
		from saved_supplier_offers
		where project_id = $1
		order by requirement_id, created_at desc
	`, projectID); err != nil {
		return nil, err
	}
	return offers, nil
}

func (r *ProjectRepository) LinkSupplierOfferToComponent(ctx context.Context, offerID, componentID string) error {
	res, err := r.store.db.ExecContext(ctx, `
		update saved_supplier_offers
		set linked_component_id = $1
		where id = $2
	`, componentID, offerID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrNotFound{ID: offerID}
	}
	return nil
}

func validateConstraint(constraint domain.RequirementConstraint) error {
	switch constraint.ValueType {
	case domain.ValueTypeText:
		if constraint.Operator != domain.OperatorEqual {
			return fmt.Errorf("constraint %q only supports eq for text", constraint.Key)
		}
		if constraint.Text == nil {
			return fmt.Errorf("constraint %q requires text value", constraint.Key)
		}
	case domain.ValueTypeBool:
		if constraint.Operator != domain.OperatorEqual {
			return fmt.Errorf("constraint %q only supports eq for bool", constraint.Key)
		}
		if constraint.Bool == nil {
			return fmt.Errorf("constraint %q requires bool value", constraint.Key)
		}
	case domain.ValueTypeNumber:
		if constraint.Number == nil {
			return fmt.Errorf("constraint %q requires number value", constraint.Key)
		}
		switch constraint.Operator {
		case domain.OperatorEqual, domain.OperatorGTE, domain.OperatorLTE:
		default:
			return fmt.Errorf("constraint %q has unsupported operator %q for number", constraint.Key, constraint.Operator)
		}
	default:
		return fmt.Errorf("constraint %q has unsupported value type %q", constraint.Key, constraint.ValueType)
	}

	return nil
}

var _ domain.ProjectRepository = (*ProjectRepository)(nil)
