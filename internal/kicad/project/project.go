package project

import (
	"archive/zip"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/C-Ma-P/trace/internal/kicad"
	"github.com/C-Ma-P/trace/internal/kicad/sch"
	"github.com/C-Ma-P/trace/internal/kicad/sym"
)

const (
	gridCols     = 8
	gridSpacingX = 30.0
	gridSpacingY = 30.0
	gridOriginX  = 35.0
	gridOriginY  = 40.0
	fpLibName    = "trace_fp"
	symLibName   = "trace_export"
	fpPrettyDir  = "trace_fp.pretty"
)

func Export(input kicad.ExportInput) (kicad.ExportOutput, error) {
	var warnings []string

	if input.SchematicUUID == "" {
		input.SchematicUUID = newUUID()
	}

	parts := make([]kicad.ExportedPart, len(input.Parts))
	copy(parts, input.Parts)
	for i := range parts {
		col := i % gridCols
		row := i / gridCols
		parts[i].X = gridOriginX + float64(col)*gridSpacingX
		parts[i].Y = gridOriginY + float64(row)*gridSpacingY
	}

	lib := sym.New()
	for _, p := range parts {
		if p.SymbolSrcPath == "" || lib.Has(p.SymbolLibKey) {
			continue
		}
		if err := lib.AddFromFile(p.SymbolLibKey, p.SymbolSrcPath, ""); err != nil {
			warnings = append(warnings, fmt.Sprintf("symbol load failed for %q: %v", p.SymbolLibKey, err))
		}
	}

	outDir, err := os.MkdirTemp("", "trace-kicad-export-*")
	if err != nil {
		return kicad.ExportOutput{}, fmt.Errorf("create output dir: %w", err)
	}

	safeName := sanitizeFileName(input.ProjectName)
	if safeName == "" {
		safeName = "trace_export"
	}

	proPath := filepath.Join(outDir, safeName+".kicad_pro")
	if err := writeProFile(proPath, safeName); err != nil {
		_ = os.RemoveAll(outDir)
		return kicad.ExportOutput{}, fmt.Errorf("write .kicad_pro: %w", err)
	}

	symLibPath := filepath.Join(outDir, symLibName+".kicad_sym")
	if err := writeSymLib(symLibPath, lib); err != nil {
		_ = os.RemoveAll(outDir)
		return kicad.ExportOutput{}, fmt.Errorf("write .kicad_sym: %w", err)
	}

	hasFP := false
	for _, p := range parts {
		if p.FootprintSrcPath != "" && p.FootprintModuleName != "" {
			if err := copyFootprint(outDir, p); err != nil {
				warnings = append(warnings, fmt.Sprintf("footprint copy failed for %q: %v", p.FootprintModuleName, err))
			} else {
				hasFP = true
			}
		}
	}

	if err := writeSymLibTable(filepath.Join(outDir, "sym-lib-table")); err != nil {
		_ = os.RemoveAll(outDir)
		return kicad.ExportOutput{}, fmt.Errorf("write sym-lib-table: %w", err)
	}

	if hasFP {
		if err := writeFPLibTable(filepath.Join(outDir, "fp-lib-table")); err != nil {

			warnings = append(warnings, fmt.Sprintf("fp-lib-table write failed: %v", err))
		}
	}

	schPath := filepath.Join(outDir, safeName+".kicad_sch")
	if err := writeSchematic(schPath, input.ProjectName, input.SchematicUUID, parts, lib); err != nil {
		_ = os.RemoveAll(outDir)
		return kicad.ExportOutput{}, fmt.Errorf("write .kicad_sch: %w", err)
	}

	zipPath := outDir + ".zip"
	if err := zipDir(outDir, zipPath, safeName); err != nil {
		_ = os.RemoveAll(outDir)
		return kicad.ExportOutput{}, fmt.Errorf("create zip: %w", err)
	}

	return kicad.ExportOutput{
		Dir:      outDir,
		ZipPath:  zipPath,
		Warnings: warnings,
	}, nil
}

func writeProFile(path, projectName string) error {
	pro := map[string]interface{}{
		"board": map[string]interface{}{
			"3dviewports":     []interface{}{},
			"design_settings": map[string]interface{}{},
			"layer_presets":   []interface{}{},
			"viewports":       []interface{}{},
		},
		"boards": []interface{}{},
		"cvpcb": map[string]interface{}{
			"equivalence_files": []interface{}{},
		},
		"libraries": map[string]interface{}{
			"pinned_footprint_libs": []interface{}{},
			"pinned_symbol_libs":    []interface{}{},
		},
		"meta": map[string]interface{}{
			"filename": projectName + ".kicad_pro",
			"version":  1,
		},
		"net_settings": map[string]interface{}{
			"classes": []interface{}{},
			"meta":    map[string]interface{}{"version": 3},
		},
		"schematic": map[string]interface{}{
			"annotate_start_num": 0,
			"drawing":            map[string]interface{}{},
			"legacy_lib_dir":     "",
			"legacy_lib_list":    []interface{}{},
		},
		"sheets":         []interface{}{},
		"text_variables": map[string]interface{}{},
	}
	data, err := json.MarshalIndent(pro, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func writeSymLib(path string, lib *sym.Library) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return lib.WriteTo(f)
}

func writeSymLibTable(path string) error {
	const content = `(sym_lib_table
  (lib
    (name "trace_export")
    (type "KiCad")
    (uri "${KIPRJMOD}/trace_export.kicad_sym")
    (options "")
    (descr "Trace export library")
  )
)
`
	return os.WriteFile(path, []byte(content), 0o644)
}

func writeFPLibTable(path string) error {
	content := fmt.Sprintf(`(fp_lib_table
  (lib
    (name %q)
    (type "KiCad")
    (uri "${KIPRJMOD}/%s")
    (options "")
    (descr "Trace export footprint library")
  )
)
`, fpLibName, fpPrettyDir)
	return os.WriteFile(path, []byte(content), 0o644)
}

func copyFootprint(outDir string, p kicad.ExportedPart) error {
	prettyDir := filepath.Join(outDir, fpPrettyDir)
	if err := os.MkdirAll(prettyDir, 0o755); err != nil {
		return err
	}
	dst := filepath.Join(prettyDir, sanitizeFileName(p.FootprintModuleName)+".kicad_mod")
	return copyFile(p.FootprintSrcPath, dst)
}

func writeSchematic(path, projectName, schUUID string, parts []kicad.ExportedPart, lib *sym.Library) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return sch.Write(f, projectName, schUUID, parts, lib)
}

func zipDir(srcDir, destZip, folderName string) error {
	zf, err := os.Create(destZip)
	if err != nil {
		return err
	}
	defer zf.Close()

	zw := zip.NewWriter(zf)
	defer zw.Close()

	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		zipEntry := folderName + "/" + filepath.ToSlash(rel)
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		w, err := zw.Create(zipEntry)
		if err != nil {
			return err
		}
		_, err = io.Copy(w, f)
		return err
	})
}

var nonAlnum = regexp.MustCompile(`[^A-Za-z0-9_\-]`)

func sanitizeFileName(s string) string {
	s = strings.TrimSpace(s)
	s = nonAlnum.ReplaceAllString(s, "_")
	s = strings.Trim(s, "_")
	if s == "" {
		return "export"
	}
	return s
}

func newUUID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])

	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
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
	_, err = io.Copy(out, in)
	return err
}
