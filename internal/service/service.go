package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sync"

	"fmt"

	"github.com/C-Ma-P/trace/internal/domain"
	"github.com/C-Ma-P/trace/internal/domain/registry"
	"github.com/C-Ma-P/trace/internal/kicad"
	"github.com/C-Ma-P/trace/internal/kicadconfig"
	"github.com/C-Ma-P/trace/internal/sourcing"
	"github.com/C-Ma-P/trace/internal/supplierconfig"
)

type Service struct {
	components            domain.ComponentRepository
	projects              domain.ProjectRepository
	assets                domain.ComponentAssetRepository
	kicad                 *kicad.Service
	kicadConfig           *kicadconfig.Manager
	sourcing              *sourcing.Service
	sourcingCoordinator   *sourcing.Coordinator
	sourcingCoordinatorMu sync.Mutex
	supplierConfig        *supplierconfig.Manager
}

func New(components domain.ComponentRepository, projects domain.ProjectRepository, assets domain.ComponentAssetRepository, kicadServices ...*kicad.Service) *Service {
	var importer *kicad.Service
	if len(kicadServices) > 0 {
		importer = kicadServices[0]
	}
	return &Service{components: components, projects: projects, assets: assets, kicad: importer}
}

func (s *Service) SetSourcing(sourcingSvc *sourcing.Service) *Service {
	s.sourcingCoordinatorMu.Lock()
	s.sourcing = sourcingSvc
	s.sourcingCoordinator = nil
	s.sourcingCoordinatorMu.Unlock()
	return s
}

func (s *Service) SetKiCadConfig(configSvc *kicadconfig.Manager) *Service {
	s.kicadConfig = configSvc
	return s
}

func (s *Service) SetSupplierConfig(configSvc *supplierconfig.Manager) *Service {
	s.supplierConfig = configSvc
	return s
}

func (s *Service) GetKiCadPreferences(ctx context.Context) (kicadconfig.Preferences, error) {
	if s.kicadConfig == nil {
		return kicadconfig.Preferences{}, fmt.Errorf("KiCad preferences not configured")
	}
	return s.kicadConfig.GetPreferences(ctx)
}

func (s *Service) SaveKiCadPreferences(ctx context.Context, input kicadconfig.UpdateInput) (kicadconfig.Preferences, error) {
	if s.kicadConfig == nil {
		return kicadconfig.Preferences{}, fmt.Errorf("KiCad preferences not configured")
	}
	return s.kicadConfig.SavePreferences(ctx, input)
}

func (s *Service) GetSupplierPreferences(ctx context.Context) (supplierconfig.Preferences, error) {
	if s.supplierConfig == nil {
		return supplierconfig.Preferences{}, fmt.Errorf("supplier preferences not configured")
	}
	return s.supplierConfig.GetPreferences(ctx)
}

func (s *Service) SaveSupplierPreferences(ctx context.Context, input supplierconfig.UpdateInput) (supplierconfig.Preferences, error) {
	if s.supplierConfig == nil {
		return supplierconfig.Preferences{}, fmt.Errorf("supplier preferences not configured")
	}
	return s.supplierConfig.SavePreferences(ctx, input)
}

func (s *Service) ClearSupplierSecret(ctx context.Context, provider, secret string) (supplierconfig.Preferences, error) {
	if s.supplierConfig == nil {
		return supplierconfig.Preferences{}, fmt.Errorf("supplier preferences not configured")
	}
	return s.supplierConfig.ClearSecret(ctx, provider, secret)
}

func (s *Service) UpsertAttributeDefinition(ctx context.Context, def domain.AttributeDefinition) error {
	return s.components.UpsertAttributeDefinition(ctx, def)
}

func (s *Service) SyncCanonicalAttributeDefinitions(ctx context.Context) error {
	for _, category := range registry.Categories() {
		for _, def := range registry.DefinitionsForCategory(category) {
			if err := s.components.UpsertAttributeDefinition(ctx, def); err != nil {
				return err
			}
		}
	}
	return nil
}

func componentDefinitionLabel(component domain.Component) string {
	parts := []string{component.Manufacturer, component.MPN}
	label := ""
	for _, part := range parts {
		part = fmt.Sprintf("%s", part)
		if part == "" {
			continue
		}
		if label == "" {
			label = part
			continue
		}
		label += " " + part
	}
	if label != "" {
		return label
	}
	if component.Description != "" {
		return component.Description
	}
	return component.ID
}

func newID() string {
	buf := make([]byte, 16)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)
}
