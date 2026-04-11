package kicadconfig

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"

	"github.com/C-Ma-P/trace/internal/domain"
)

const (
	preferencePrefix = "integrations.kicad."
	prefProjectRoots = "integrations.kicad.project_roots"
)

type Manager struct {
	prefs domain.PreferenceRepository
}

type Preferences struct {
	ProjectRoots []string `json:"projectRoots"`
}

type UpdateInput struct {
	ProjectRoots []string `json:"projectRoots"`
}

func NewManager(prefs domain.PreferenceRepository) *Manager {
	return &Manager{prefs: prefs}
}

func (m *Manager) GetPreferences(ctx context.Context) (Preferences, error) {
	stored, err := m.loadStored(ctx)
	if err != nil {
		return Preferences{}, err
	}
	return Preferences{ProjectRoots: stored}, nil
}

func (m *Manager) SavePreferences(ctx context.Context, input UpdateInput) (Preferences, error) {
	roots := normalizeRoots(input.ProjectRoots)
	payload, err := json.Marshal(roots)
	if err != nil {
		return Preferences{}, err
	}
	if err := m.prefs.SetMany(ctx, map[string]string{prefProjectRoots: string(payload)}); err != nil {
		return Preferences{}, err
	}
	return Preferences{ProjectRoots: roots}, nil
}

func (m *Manager) loadStored(ctx context.Context) ([]string, error) {
	values, err := m.prefs.List(ctx, preferencePrefix)
	if err != nil {
		return nil, err
	}
	raw := strings.TrimSpace(values[prefProjectRoots])
	if raw == "" {
		return []string{}, nil
	}
	var roots []string
	if err := json.Unmarshal([]byte(raw), &roots); err != nil {
		return nil, err
	}
	return normalizeRoots(roots), nil
}

func normalizeRoots(roots []string) []string {
	seen := make(map[string]struct{}, len(roots))
	normalized := make([]string, 0, len(roots))
	for _, root := range roots {
		trimmed := strings.TrimSpace(root)
		if trimmed == "" {
			continue
		}
		cleaned := filepath.Clean(trimmed)
		if _, ok := seen[cleaned]; ok {
			continue
		}
		seen[cleaned] = struct{}{}
		normalized = append(normalized, cleaned)
	}
	return normalized
}
