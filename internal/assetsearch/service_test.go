package assetsearch

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/C-Ma-P/trace/internal/domain"
	"github.com/C-Ma-P/trace/internal/ingest"
)

type componentRepoStub struct {
	domain.ComponentRepository
	getComponentResult domain.Component
	getComponentErr    error
}

func (s *componentRepoStub) GetComponent(_ context.Context, _ string) (domain.Component, error) {
	if s.getComponentErr != nil {
		return domain.Component{}, s.getComponentErr
	}
	return s.getComponentResult, nil
}

type assetRepoStub struct {
	domain.ComponentAssetRepository
	createdAssets []domain.ComponentAsset
	createErr     error
}

func (s *assetRepoStub) CreateComponentAsset(_ context.Context, a domain.ComponentAsset) (domain.ComponentAsset, error) {
	if s.createErr != nil {
		return domain.ComponentAsset{}, s.createErr
	}
	s.createdAssets = append(s.createdAssets, a)
	if a.ID == "" {
		a.ID = "created-id"
	}
	return a, nil
}

func componentRef(id, mpn, mfr string) domain.Component {
	return domain.Component{ID: id, MPN: mpn, Manufacturer: mfr}
}

func TestImportSearchResult_ValidatesInputs(t *testing.T) {
	svc := NewService(NewRegistry(), &componentRepoStub{}, &assetRepoStub{})

	_, err := svc.ImportSearchResult(context.Background(), ImportRequest{})
	if err == nil || !strings.Contains(err.Error(), "component ID is required") {
		t.Fatalf("expected component ID validation error, got: %v", err)
	}

	_, err = svc.ImportSearchResult(context.Background(), ImportRequest{ComponentID: "c1"})
	if err == nil || !strings.Contains(err.Error(), "provider is required") {
		t.Fatalf("expected provider validation error, got: %v", err)
	}

	_, err = svc.ImportSearchResult(context.Background(), ImportRequest{ComponentID: "c1", Provider: "p1"})
	if err == nil || !strings.Contains(err.Error(), "external ID is required") {
		t.Fatalf("expected external ID validation error, got: %v", err)
	}
}

func TestImportSearchResult_UnknownProvider(t *testing.T) {
	svc := NewService(NewRegistry(), &componentRepoStub{}, &assetRepoStub{})
	_, err := svc.ImportSearchResult(context.Background(), ImportRequest{ComponentID: "c1", Provider: "missing", ExternalID: "e1"})
	if err == nil || !strings.Contains(err.Error(), `unknown provider "missing"`) {
		t.Fatalf("expected unknown provider error, got: %v", err)
	}
}

func TestImportSearchResult_ProviderFailureIsWrapped(t *testing.T) {
	r := NewRegistry()
	r.Register(&testProvider{
		name:        "p1",
		displayName: "Provider One",
		importFn: func(_ context.Context, _ ImportRequest) (ImportResponse, error) {
			return ImportResponse{}, errors.New("upstream down")
		},
	})

	svc := NewService(r, &componentRepoStub{}, &assetRepoStub{})
	_, err := svc.ImportSearchResult(context.Background(), ImportRequest{ComponentID: "c1", Provider: "p1", ExternalID: "e1"})
	if err == nil || !strings.Contains(err.Error(), `provider "p1" import failed`) {
		t.Fatalf("expected wrapped provider error, got: %v", err)
	}
}

func TestImportSearchResult_DirectPersistPath(t *testing.T) {
	repo := &assetRepoStub{}
	r := NewRegistry()
	r.Register(&testProvider{
		name:        "p1",
		displayName: "Provider One",
		importFn: func(_ context.Context, _ ImportRequest) (ImportResponse, error) {
			return ImportResponse{ImportedAssets: []ImportedAsset{
				{AssetType: string(domain.AssetTypeDatasheet), Label: "Datasheet", URLOrPath: "https://example.test/ds.pdf"},
				{AssetType: "unknown", Label: "Bad", URLOrPath: "bad"},
			}}, nil
		},
	})

	svc := NewService(r, &componentRepoStub{}, repo)
	resp, err := svc.ImportSearchResult(context.Background(), ImportRequest{ComponentID: "c1", Provider: "p1", ExternalID: "e1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.createdAssets) != 1 {
		t.Fatalf("expected 1 persisted asset, got %d", len(repo.createdAssets))
	}
	if len(resp.Warnings) == 0 {
		t.Fatalf("expected warning for unknown asset type")
	}
}

func TestImportSearchResult_IngestionPath(t *testing.T) {
	tempDir := t.TempDir()
	artifactPath := filepath.Join(tempDir, "part.pdf")
	if err := os.WriteFile(artifactPath, []byte("pdf"), 0o644); err != nil {
		t.Fatalf("write artifact: %v", err)
	}

	componentRepo := &componentRepoStub{getComponentResult: componentRef("c1", "MPN-1", "Acme")}
	assetRepo := &assetRepoStub{}
	ingestSvc := ingest.NewService(filepath.Join(tempDir, "assets"), componentRepo, assetRepo)

	r := NewRegistry()
	r.Register(&testProvider{
		name:        "p1",
		displayName: "Provider One",
		importFn: func(_ context.Context, _ ImportRequest) (ImportResponse, error) {
			return ImportResponse{Artifacts: []DownloadedArtifact{
				{FilePath: artifactPath, Description: "datasheet"},
				{FilePath: filepath.Join(tempDir, "missing.pdf"), Description: "missing"},
			}}, nil
		},
	})

	svc := NewService(r, componentRepo, assetRepo, ingestSvc)
	resp, err := svc.ImportSearchResult(context.Background(), ImportRequest{ComponentID: "c1", Provider: "p1", ExternalID: "e1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.ImportedAssets) != 1 {
		t.Fatalf("expected 1 imported asset from ingestion, got %d", len(resp.ImportedAssets))
	}
	if len(resp.Warnings) == 0 {
		t.Fatalf("expected warning from missing artifact ingestion failure")
	}
}
