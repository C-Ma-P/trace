package ingest

import (
	"archive/zip"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"trace/internal/domain"
)

// maxExtractedFileSize is the maximum allowed size for a single file extracted
// from a zip archive. Files exceeding this limit are rejected (not silently
// truncated) to avoid persisting corrupt assets.
const maxExtractedFileSize = 256 << 20 // 256 MB

// Service handles ingestion of files into Trace-managed component asset storage.
// It is designed to be reusable by manual file import, provider downloads, and
// future batch/drag-drop import flows.
type Service struct {
	assetsDir  string // root directory for managed asset storage
	components domain.ComponentRepository
	assets     domain.ComponentAssetRepository
}

// NewService creates an ingestion service.
// assetsDir is the root managed asset storage directory (e.g. ~/.trace/assets).
func NewService(assetsDir string, components domain.ComponentRepository, assets domain.ComponentAssetRepository) *Service {
	return &Service{
		assetsDir:  assetsDir,
		components: components,
		assets:     assets,
	}
}

// IngestRequest describes what to ingest and for which component.
type IngestRequest struct {
	ComponentID string
	FilePath    string // path to a file, directory, or zip archive
	SourceKind  string // e.g. "local", "snapeda", "ultralibrarian"
	SourceLabel string // optional human-readable provenance
}

// IngestResult describes the outcome of an ingestion operation.
type IngestResult struct {
	Assets      []IngestedAsset `json:"assets"`
	Warnings    []string        `json:"warnings"`
	Unsupported []string        `json:"unsupported"`
	CountByType map[string]int  `json:"countByType"`
}

// IngestedAsset describes a single asset that was ingested and persisted.
type IngestedAsset struct {
	AssetID          string `json:"assetId"`
	AssetType        string `json:"assetType"`
	Label            string `json:"label"`
	StoredPath       string `json:"storedPath"`
	OriginalFilename string `json:"originalFilename"`
}

// Ingest inspects the input path and dispatches to the appropriate ingestion method.
func (s *Service) Ingest(ctx context.Context, req IngestRequest) (IngestResult, error) {
	if req.ComponentID == "" {
		return IngestResult{}, fmt.Errorf("component ID is required")
	}
	if req.FilePath == "" {
		return IngestResult{}, fmt.Errorf("file path is required")
	}

	// Verify component exists.
	if _, err := s.components.GetComponent(ctx, req.ComponentID); err != nil {
		return IngestResult{}, fmt.Errorf("component lookup: %w", err)
	}

	if req.SourceKind == "" {
		req.SourceKind = "local"
	}

	info, err := os.Stat(req.FilePath)
	if err != nil {
		return IngestResult{}, fmt.Errorf("stat input: %w", err)
	}

	if info.IsDir() {
		return s.ingestDirectory(ctx, req)
	}

	if isZipFile(req.FilePath) {
		return s.ingestZip(ctx, req)
	}

	return s.ingestSingleFile(ctx, req)
}

// IngestFromDir ingests all supported files from a directory. This is the seam
// for providers that extract downloads into a temp directory.
func (s *Service) IngestFromDir(ctx context.Context, req IngestRequest) (IngestResult, error) {
	if req.ComponentID == "" {
		return IngestResult{}, fmt.Errorf("component ID is required")
	}
	if req.FilePath == "" {
		return IngestResult{}, fmt.Errorf("directory path is required")
	}
	if _, err := s.components.GetComponent(ctx, req.ComponentID); err != nil {
		return IngestResult{}, fmt.Errorf("component lookup: %w", err)
	}
	if req.SourceKind == "" {
		req.SourceKind = "local"
	}
	return s.ingestDirectory(ctx, req)
}

// ingestSingleFile classifies, copies, and persists a single file.
func (s *Service) ingestSingleFile(ctx context.Context, req IngestRequest) (IngestResult, error) {
	result := newResult()

	filename := filepath.Base(req.FilePath)
	assetType := classifyFile(filename)
	if assetType == "" {
		result.Unsupported = append(result.Unsupported, filename)
		result.Warnings = append(result.Warnings, fmt.Sprintf("unsupported file type: %s", filename))
		return result, nil
	}

	ingested, err := s.copyAndPersist(ctx, req.ComponentID, req.FilePath, filename, assetType, req.SourceKind, req.SourceLabel, "")
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("failed to ingest %s: %v", filename, err))
		return result, nil
	}

	result.Assets = append(result.Assets, ingested)
	result.CountByType[string(assetType)]++
	log.Printf("[ingest] ingested %s as %s for component %s", filename, assetType, req.ComponentID)
	return result, nil
}

// ingestDirectory recursively walks a directory and ingests all supported files.
// .pretty directories are handled specially as footprint library units: their
// contained .kicad_mod files are ingested with library-qualified labels, and the
// walker does not descend into them separately. Hidden directories (prefixed
// with ".") that are not .pretty dirs are skipped.
func (s *Service) ingestDirectory(ctx context.Context, req IngestRequest) (IngestResult, error) {
	result := newResult()

	// If this directory itself is a .pretty library, ingest its .kicad_mod files.
	if isPrettyDir(req.FilePath) {
		s.ingestPrettyDir(ctx, req, filepath.Base(req.FilePath), req.FilePath, &result)
		logResult("directory", req.FilePath, req.ComponentID, result)
		return result, nil
	}

	err := filepath.WalkDir(req.FilePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("cannot access %s: %v", path, err))
			return nil
		}

		if d.IsDir() {
			// Skip the root dir itself (we're already walking it).
			if path == req.FilePath {
				return nil
			}
			// Handle .pretty directories as footprint library units.
			if isPrettyDir(d.Name()) {
				s.ingestPrettyDir(ctx, req, d.Name(), path, &result)
				return fs.SkipDir
			}
			// Skip hidden directories (e.g. __MACOSX, .git).
			if strings.HasPrefix(d.Name(), ".") || strings.HasPrefix(d.Name(), "__") {
				return fs.SkipDir
			}
			return nil
		}

		// Skip hidden files.
		if strings.HasPrefix(d.Name(), ".") {
			return nil
		}

		assetType := classifyFile(d.Name())
		if assetType == "" {
			rel, _ := filepath.Rel(req.FilePath, path)
			if rel == "" {
				rel = d.Name()
			}
			result.Unsupported = append(result.Unsupported, rel)
			return nil
		}

		ingested, err := s.copyAndPersist(ctx, req.ComponentID, path, d.Name(), assetType, req.SourceKind, req.SourceLabel, "")
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("failed to ingest %s: %v", d.Name(), err))
			return nil
		}

		result.Assets = append(result.Assets, ingested)
		result.CountByType[string(assetType)]++
		return nil
	})

	if err != nil {
		return IngestResult{}, fmt.Errorf("walk directory: %w", err)
	}

	logResult("directory", req.FilePath, req.ComponentID, result)
	return result, nil
}

// ingestPrettyDir ingests .kicad_mod files from a .pretty footprint library directory.
// Footprint assets are labelled as "LibraryName/FootprintName" for clarity.
func (s *Service) ingestPrettyDir(ctx context.Context, req IngestRequest, libName string, dirPath string, result *IngestResult) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("cannot read .pretty dir %s: %v", libName, err))
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if classifyFile(entry.Name()) != domain.AssetTypeFootprint {
			result.Unsupported = append(result.Unsupported, filepath.Join(libName, entry.Name()))
			continue
		}

		entryPath := filepath.Join(dirPath, entry.Name())
		label := fmt.Sprintf("%s/%s", strings.TrimSuffix(libName, ".pretty"), strings.TrimSuffix(entry.Name(), ".kicad_mod"))

		ingested, err := s.copyAndPersist(ctx, req.ComponentID, entryPath, entry.Name(), domain.AssetTypeFootprint, req.SourceKind, req.SourceLabel, label)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("failed to ingest %s/%s: %v", libName, entry.Name(), err))
			continue
		}
		result.Assets = append(result.Assets, ingested)
		result.CountByType["footprint"]++
	}
}

// ingestZip safely extracts a zip archive to a temp directory and ingests all
// supported files found inside. Per-entry extraction failures (e.g. oversized
// files) are captured as warnings rather than failing the entire zip.
func (s *Service) ingestZip(ctx context.Context, req IngestRequest) (IngestResult, error) {
	result := newResult()

	tmpDir, err := os.MkdirTemp("", "trace-ingest-*")
	if err != nil {
		return IngestResult{}, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	extractWarnings, err := extractZip(req.FilePath, tmpDir)
	if err != nil {
		return IngestResult{}, fmt.Errorf("extract zip: %w", err)
	}
	result.Warnings = append(result.Warnings, extractWarnings...)

	// Walk extracted contents and ingest.
	dirReq := req
	dirReq.FilePath = tmpDir
	subResult, err := s.ingestDirectory(ctx, dirReq)
	if err != nil {
		return IngestResult{}, err
	}

	// Merge sub-result.
	result.Assets = append(result.Assets, subResult.Assets...)
	result.Warnings = append(result.Warnings, subResult.Warnings...)
	result.Unsupported = append(result.Unsupported, subResult.Unsupported...)
	for k, v := range subResult.CountByType {
		result.CountByType[k] += v
	}

	logResult("zip", req.FilePath, req.ComponentID, result)
	return result, nil
}

// copyAndPersist copies a file into managed storage and creates a ComponentAsset record.
// If labelOverride is non-empty it is used as the asset label; otherwise the
// label is derived from the original filename (extension stripped).
func (s *Service) copyAndPersist(ctx context.Context, componentID, srcPath, originalFilename string, assetType domain.AssetType, sourceKind, sourceLabel, labelOverride string) (IngestedAsset, error) {
	// Determine destination path.
	destDir := filepath.Join(s.assetsDir, componentID, string(assetType))
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return IngestedAsset{}, fmt.Errorf("create asset dir: %w", err)
	}

	// Use a unique prefix to avoid collisions while keeping the original name readable.
	storedName := uniquePrefix() + "_" + sanitizeFilename(originalFilename)
	destPath := filepath.Join(destDir, storedName)

	if err := copyFile(srcPath, destPath); err != nil {
		return IngestedAsset{}, fmt.Errorf("copy file: %w", err)
	}

	// Build metadata.
	meta := map[string]string{
		"original_filename": originalFilename,
		"source_kind":       sourceKind,
	}
	if sourceLabel != "" {
		meta["source_label"] = sourceLabel
	}
	metaJSON, _ := json.Marshal(meta)

	label := labelOverride
	if label == "" {
		label = strings.TrimSuffix(originalFilename, filepath.Ext(originalFilename))
	}

	asset := domain.ComponentAsset{
		ID:           newID(),
		ComponentID:  componentID,
		AssetType:    assetType,
		Source:       sourceKind,
		Status:       domain.AssetStatusCandidate,
		Label:        label,
		URLOrPath:    destPath,
		MetadataJSON: metaJSON,
	}

	created, err := s.assets.CreateComponentAsset(ctx, asset)
	if err != nil {
		// Clean up copied file on persistence failure.
		os.Remove(destPath)
		return IngestedAsset{}, fmt.Errorf("persist asset record: %w", err)
	}

	return IngestedAsset{
		AssetID:          created.ID,
		AssetType:        string(assetType),
		Label:            label,
		StoredPath:       destPath,
		OriginalFilename: originalFilename,
	}, nil
}

// extractZip safely extracts a zip archive into destDir.
// It guards against zip-slip path traversal and enforces a per-file size limit.
// Per-entry failures (e.g. oversized files) are captured as warnings rather
// than aborting the entire extraction — this allows partial success when a zip
// contains a mix of good and bad entries.
func extractZip(zipPath, destDir string) ([]string, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("open zip: %w", err)
	}
	defer r.Close()

	destDir, err = filepath.Abs(destDir)
	if err != nil {
		return nil, err
	}

	var warnings []string

	for _, f := range r.File {
		// Guard against zip-slip.
		target := filepath.Join(destDir, f.Name)
		if !strings.HasPrefix(filepath.Clean(target), destDir+string(os.PathSeparator)) && filepath.Clean(target) != destDir {
			return nil, fmt.Errorf("zip entry %q escapes destination directory", f.Name)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return nil, err
			}
			continue
		}

		// Ensure parent directory exists.
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return nil, err
		}

		if err := extractZipFile(f, target); err != nil {
			// Per-entry failure: warn and skip this file but continue.
			warnings = append(warnings, fmt.Sprintf("skipped zip entry %q: %v", f.Name, err))
			// Remove any partially-written file.
			os.Remove(target)
			continue
		}
	}

	return warnings, nil
}

func extractZipFile(f *zip.File, target string) error {
	// Reject files whose declared uncompressed size exceeds the limit.
	// This is an early check; the actual byte count is verified below because
	// the declared size can be spoofed in a malicious archive.
	if f.UncompressedSize64 > maxExtractedFileSize {
		return fmt.Errorf("declared size %d bytes exceeds limit of %d bytes", f.UncompressedSize64, maxExtractedFileSize)
	}

	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
	if err != nil {
		return err
	}

	// Copy up to maxExtractedFileSize+1 bytes.  If we end up with more than
	// maxExtractedFileSize, the entry is oversized and we fail explicitly
	// instead of silently truncating.
	n, err := io.Copy(out, io.LimitReader(rc, maxExtractedFileSize+1))
	if closeErr := out.Close(); closeErr != nil && err == nil {
		err = closeErr
	}
	if err != nil {
		os.Remove(target)
		return err
	}
	if n > maxExtractedFileSize {
		os.Remove(target)
		return fmt.Errorf("extracted size exceeds limit of %d bytes", maxExtractedFileSize)
	}

	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

func newID() string {
	buf := make([]byte, 16)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)
}

func uniquePrefix() string {
	buf := make([]byte, 4)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)
}

func sanitizeFilename(name string) string {
	// Replace path separators and nulls to prevent traversal.
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.ReplaceAll(name, "\x00", "_")
	if name == "" {
		name = "unnamed"
	}
	return name
}

func newResult() IngestResult {
	return IngestResult{
		Assets:      []IngestedAsset{},
		Warnings:    []string{},
		Unsupported: []string{},
		CountByType: map[string]int{},
	}
}

func logResult(source, path, componentID string, r IngestResult) {
	total := len(r.Assets)
	if total == 0 {
		log.Printf("[ingest] %s %s for component %s: no supported assets found (%d unsupported files)", source, filepath.Base(path), componentID, len(r.Unsupported))
		return
	}
	parts := make([]string, 0, len(r.CountByType))
	for k, v := range r.CountByType {
		parts = append(parts, fmt.Sprintf("%d %s", v, k))
	}
	log.Printf("[ingest] %s %s for component %s: %d assets ingested (%s)", source, filepath.Base(path), componentID, total, strings.Join(parts, ", "))
}
