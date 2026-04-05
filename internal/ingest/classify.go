package ingest

import (
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
