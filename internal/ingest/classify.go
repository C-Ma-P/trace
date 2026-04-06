package ingest

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"componentmanager/internal/domain"
)

// classifyFile determines the asset type for a file based on its extension.
// Returns empty string if the file type is not supported.
func classifyFile(name string) domain.AssetType {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".kicad_sym":
		return domain.AssetTypeSymbol
	case ".kicad_mod":
		return domain.AssetTypeFootprint
	case ".step", ".stp", ".wrl":
		return domain.AssetType3DModel
	case ".pdf":
		return domain.AssetTypeDatasheet
	default:
		return ""
	}
}

// isPrettyDir checks whether a directory name ends with .pretty,
// indicating a KiCad footprint library directory.
func isPrettyDir(name string) bool {
	return strings.HasSuffix(strings.ToLower(name), ".pretty")
}

// isZipFile checks whether a filename has a .zip extension.
func isZipFile(name string) bool {
	return strings.ToLower(filepath.Ext(name)) == ".zip"
}

// supportedExtensions lists all file extensions recognised during ingestion.
var supportedExtensions = map[string]bool{
	".kicad_sym": true,
	".kicad_mod": true,
	".step":      true,
	".stp":       true,
	".wrl":       true,
	".pdf":       true,
}

func isSupportedExtension(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return supportedExtensions[ext]
}

// PathKind describes what kind of input a path is.
type PathKind string

const (
	PathKindFile    PathKind = "file"
	PathKindDir     PathKind = "dir"
	PathKindZip     PathKind = "zip"
	PathKindMissing PathKind = "missing"
)

// PathValidation is the result of ValidatePath.
type PathValidation struct {
	Valid    bool     `json:"valid"`
	Reason   string   `json:"reason"`   // human-readable problem, empty when valid
	PathKind PathKind `json:"pathKind"` // "file", "dir", "zip", "missing"
}

// ValidatePath checks whether a path is usable for ingestion and returns a
// structured result. It does not ingest anything — callers use this for
// pre-flight UI validation.
func ValidatePath(path string) PathValidation {
	if path == "" {
		return PathValidation{Valid: false, Reason: "", PathKind: PathKindMissing}
	}

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return PathValidation{Valid: false, Reason: "Path not found", PathKind: PathKindMissing}
		}
		return PathValidation{Valid: false, Reason: fmt.Sprintf("Cannot access path: %v", err), PathKind: PathKindMissing}
	}

	if info.IsDir() {
		if hasAnySupportedFile(path) {
			return PathValidation{Valid: true, PathKind: PathKindDir}
		}
		return PathValidation{
			Valid:    false,
			Reason:   "Directory contains no supported asset files (.kicad_sym, .kicad_mod, .step, .stp, .wrl, .pdf)",
			PathKind: PathKindDir,
		}
	}

	if isZipFile(path) {
		return PathValidation{Valid: true, PathKind: PathKindZip}
	}

	if isSupportedExtension(path) {
		return PathValidation{Valid: true, PathKind: PathKindFile}
	}

	ext := strings.ToLower(filepath.Ext(path))
	if ext == "" {
		return PathValidation{Valid: false, Reason: "File has no extension — supported: .kicad_sym, .kicad_mod, .step, .stp, .wrl, .pdf, .zip", PathKind: PathKindFile}
	}
	return PathValidation{
		Valid:    false,
		Reason:   fmt.Sprintf("Unsupported file type %q — supported: .kicad_sym, .kicad_mod, .step, .stp, .wrl, .pdf, .zip", ext),
		PathKind: PathKindFile,
	}
}

// hasAnySupportedFile reports whether dir contains at least one supported file,
// searching recursively through subdirectories. .pretty directories count as
// supported (they produce footprint assets).
func hasAnySupportedFile(dir string) bool {
	found := false
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if found {
			return fs.SkipAll
		}
		if d.IsDir() {
			if isPrettyDir(d.Name()) {
				found = true
				return fs.SkipDir
			}
			// Skip hidden/metadata dirs but keep walking.
			if d.Name() != "." && (strings.HasPrefix(d.Name(), ".") || strings.HasPrefix(d.Name(), "__")) {
				return fs.SkipDir
			}
			return nil
		}
		if isSupportedExtension(d.Name()) || isZipFile(d.Name()) {
			found = true
			return fs.SkipAll
		}
		return nil
	})
	return found
}
