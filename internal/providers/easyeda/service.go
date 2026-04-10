package easyeda

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	easyeda "github.com/C-Ma-P/go-easyeda"

	"trace/internal/ingest"
)

// Service orchestrates EasyEDA/LCSC component asset import.
type Service struct {
	client *easyeda.Client
	ingest *ingest.Service
}

// NewService creates an EasyEDA provider service.
func NewService(ingestSvc *ingest.Service) *Service {
	return &Service{
		client: easyeda.NewClient(),
		ingest: ingestSvc,
	}
}

// ImportComponentAssets fetches EasyEDA/LCSC component data for the given LCSC
// ID, converts it to KiCad artifacts, and ingests them as managed assets for
// the given component.
func (s *Service) ImportComponentAssets(ctx context.Context, req ImportRequest) (ImportResult, error) {
	result := ImportResult{LCSCID: req.LCSCID}

	// Validate inputs.
	if req.ComponentID == "" {
		return result, fmt.Errorf("component ID is required")
	}
	if err := easyeda.ValidateLCSCID(req.LCSCID); err != nil {
		return result, fmt.Errorf("invalid LCSC ID: %w", err)
	}

	log.Printf("[easyeda] starting import for LCSC %s (component %s)", req.LCSCID, req.ComponentID)

	// Fetch the component bundle from EasyEDA.
	bundle, err := s.client.FetchComponentBundle(ctx, req.LCSCID, easyeda.FetchOptions{
		Download3DModel:   false,
		DownloadStepModel: true,
	})
	if err != nil {
		return result, fmt.Errorf("fetching EasyEDA bundle for %s: %w", req.LCSCID, err)
	}
	log.Printf("[easyeda] fetched bundle for LCSC %s (name=%q)", req.LCSCID, bundleName(bundle))

	meta := bundle.Extracted

	// Create temp working directory.
	tmpDir, err := os.MkdirTemp("", "trace-easyeda-*")
	if err != nil {
		return result, fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Determine a clean base name for generated files.
	baseName := sanitizeSymbolName(bundleName(bundle))
	if baseName == "" {
		baseName = req.LCSCID
	}

	// Convert symbol.
	symbolOK := false
	if meta != nil && len(meta.SymbolRaw) > 0 {
		symContent, err := s.convertAndWriteSymbol(meta.SymbolRaw, req.LCSCID, baseName, tmpDir)
		if err != nil {
			msg := fmt.Sprintf("symbol conversion failed: %v", err)
			log.Printf("[easyeda] %s", msg)
			result.Warnings = append(result.Warnings, msg)
		} else {
			log.Printf("[easyeda] symbol converted: %d bytes", len(symContent))
			symbolOK = true
		}
	} else {
		result.Warnings = append(result.Warnings, "no symbol data available in upstream bundle")
	}

	// Convert footprint.
	footprintOK := false
	if meta != nil && len(meta.FootprintRaw) > 0 {
		isSMD := false
		if meta.IsSMD != nil {
			isSMD = *meta.IsSMD
		}
		fpContent, err := s.convertAndWriteFootprint(meta.FootprintRaw, isSMD, baseName, tmpDir)
		if err != nil {
			msg := fmt.Sprintf("footprint conversion failed: %v", err)
			log.Printf("[easyeda] %s", msg)
			result.Warnings = append(result.Warnings, msg)
		} else {
			log.Printf("[easyeda] footprint converted: %d bytes", len(fpContent))
			footprintOK = true
		}
	} else {
		result.Warnings = append(result.Warnings, "no footprint data available in upstream bundle")
	}

	// Download and write 3D model.
	model3DOK := false
	if bundle.StepModel != nil && len(bundle.StepModel) > 0 {
		stepFile := filepath.Join(tmpDir, baseName+".step")
		if err := os.WriteFile(stepFile, bundle.StepModel, 0o644); err != nil {
			msg := fmt.Sprintf("writing STEP file: %v", err)
			log.Printf("[easyeda] %s", msg)
			result.Warnings = append(result.Warnings, msg)
		} else {
			log.Printf("[easyeda] STEP model written: %d bytes", len(bundle.StepModel))
			model3DOK = true
		}
	} else if meta != nil && meta.Model3DUUID != "" {
		// Try downloading STEP model if not already in the bundle.
		step, err := s.client.DownloadStepModel(ctx, meta.Model3DUUID)
		if err != nil {
			msg := fmt.Sprintf("3D model download failed: %v", err)
			log.Printf("[easyeda] %s", msg)
			result.Warnings = append(result.Warnings, msg)
		} else if len(step) > 0 {
			stepFile := filepath.Join(tmpDir, baseName+".step")
			if err := os.WriteFile(stepFile, step, 0o644); err != nil {
				msg := fmt.Sprintf("writing STEP file: %v", err)
				log.Printf("[easyeda] %s", msg)
				result.Warnings = append(result.Warnings, msg)
			} else {
				log.Printf("[easyeda] STEP model downloaded: %d bytes", len(step))
				model3DOK = true
			}
		}
	} else {
		result.Warnings = append(result.Warnings, "no 3D model available for this component")
	}

	// Check for hard failure: both symbol and footprint failed.
	if !symbolOK && !footprintOK {
		return result, fmt.Errorf("both symbol and footprint conversion failed for %s", req.LCSCID)
	}

	// Ingest all generated files from the temp directory.
	ingestResult, err := s.ingest.IngestFromDir(ctx, ingest.IngestRequest{
		ComponentID: req.ComponentID,
		FilePath:    tmpDir,
		SourceKind:  "easyeda",
		SourceLabel: fmt.Sprintf("EasyEDA/LCSC %s", req.LCSCID),
	})
	if err != nil {
		return result, fmt.Errorf("ingestion failed: %w", err)
	}

	// Populate result.
	for _, asset := range ingestResult.Assets {
		switch asset.AssetType {
		case "symbol":
			result.SymbolImported = true
			result.SymbolAssetID = asset.AssetID
		case "footprint":
			result.FootprintImported = true
			result.FootprintAssetID = asset.AssetID
		case "3d_model":
			result.Model3DImported = true
			result.Model3DAssetID = asset.AssetID
		}
	}
	result.Warnings = append(result.Warnings, ingestResult.Warnings...)

	assetCount := len(ingestResult.Assets)
	log.Printf("[easyeda] import complete for LCSC %s: %d assets ingested (sym=%v fp=%v 3d=%v)",
		req.LCSCID, assetCount, result.SymbolImported, result.FootprintImported, result.Model3DImported)

	// Report unexpected partial ingestion results.
	if symbolOK && !result.SymbolImported {
		result.Warnings = append(result.Warnings, "symbol was converted but not ingested")
	}
	if footprintOK && !result.FootprintImported {
		result.Warnings = append(result.Warnings, "footprint was converted but not ingested")
	}
	if model3DOK && !result.Model3DImported {
		result.Warnings = append(result.Warnings, "3D model was written but not ingested")
	}

	return result, nil
}

// convertAndWriteSymbol parses symbol data, converts to KiCad, and writes to tmpDir.
func (s *Service) convertAndWriteSymbol(symbolRaw json.RawMessage, lcscID, baseName, tmpDir string) (string, error) {
	parsed, err := parseSymbolShapes(symbolRaw)
	if err != nil {
		return "", fmt.Errorf("parsing symbol shapes: %w", err)
	}

	content, err := convertSymbol(parsed, lcscID)
	if err != nil {
		return "", err
	}

	symFile := filepath.Join(tmpDir, baseName+".kicad_sym")
	if err := os.WriteFile(symFile, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("writing symbol file: %w", err)
	}

	return content, nil
}

// convertAndWriteFootprint parses footprint data, converts to KiCad, and writes to tmpDir.
func (s *Service) convertAndWriteFootprint(footprintRaw json.RawMessage, isSMD bool, baseName, tmpDir string) (string, error) {
	parsed, err := parseFootprintShapes(footprintRaw, isSMD)
	if err != nil {
		return "", fmt.Errorf("parsing footprint shapes: %w", err)
	}

	content, err := convertFootprint(parsed)
	if err != nil {
		return "", err
	}

	fpFile := filepath.Join(tmpDir, baseName+".kicad_mod")
	if err := os.WriteFile(fpFile, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("writing footprint file: %w", err)
	}

	return content, nil
}

// bundleName returns the best available name from a bundle.
func bundleName(bundle *easyeda.ComponentBundle) string {
	if bundle.Extracted != nil {
		if bundle.Extracted.Name != "" {
			return bundle.Extracted.Name
		}
		if bundle.Extracted.Package != "" {
			return bundle.Extracted.Package
		}
	}
	return bundle.LCSCID
}
