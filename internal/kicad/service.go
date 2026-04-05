package kicad

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const (
	defaultBinary = "kicad-cli"
	fieldList     = "Reference,Value,Footprint,Description,Manufacturer,MPN,Manufacturer Part Number,Part Number,LCSC,Datasheet,${QUANTITY},${DNP}"
	labelList     = "Refs,Value,Footprint,Description,Manufacturer,MPN,ManufacturerPartNumber,PartNumber,LCSC,Datasheet,Qty,DNP"
	groupByList   = "Value,Footprint,Description,Manufacturer,MPN,Manufacturer Part Number,Part Number,LCSC,Datasheet,${DNP}"
)

type BOMExporter interface {
	ExportBOM(context.Context, string) ([]byte, error)
}

type CLIExporter struct {
	Binary string
}

type Service struct {
	exporter BOMExporter
}

type bomRow struct {
	Refs             string
	Value            string
	Footprint        string
	Description      string
	Manufacturer     string
	MPN              string
	ManufacturerPart string
	PartNumber       string
	LCSC             string
	Datasheet        string
	Quantity         int
	DNP              string
	OtherFields      map[string]string
}

func New(exporter BOMExporter) *Service {
	if exporter == nil {
		exporter = CLIExporter{Binary: defaultBinary}
	}
	return &Service{exporter: exporter}
}

func (s *Service) ListProjects(ctx context.Context, roots []string, query string) ([]ProjectCandidate, error) {
	_ = ctx

	seen := make(map[string]struct{})
	candidates := make([]ProjectCandidate, 0)
	trimmedQuery := strings.TrimSpace(strings.ToLower(query))

	for _, root := range roots {
		cleanRoot := strings.TrimSpace(root)
		if cleanRoot == "" {
			continue
		}
		cleanRoot = filepath.Clean(cleanRoot)
		info, err := os.Stat(cleanRoot)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, err
		}
		if !info.IsDir() {
			continue
		}

		err = filepath.WalkDir(cleanRoot, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.Type()&os.ModeSymlink != 0 && entry.IsDir() {
				return filepath.SkipDir
			}
			if entry.IsDir() {
				return nil
			}
			if filepath.Ext(entry.Name()) != ".kicad_pro" {
				return nil
			}

			candidatePath := filepath.Clean(path)
			if _, ok := seen[candidatePath]; ok {
				return nil
			}
			candidate := ProjectCandidate{
				Name:        strings.TrimSuffix(entry.Name(), ".kicad_pro"),
				ProjectPath: candidatePath,
				ProjectDir:  filepath.Dir(candidatePath),
			}
			if !matchesQuery(trimmedQuery, candidate) {
				return nil
			}
			seen[candidatePath] = struct{}{}
			candidates = append(candidates, candidate)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		leftName := strings.ToLower(candidates[i].Name)
		rightName := strings.ToLower(candidates[j].Name)
		if leftName != rightName {
			return leftName < rightName
		}
		return candidates[i].ProjectPath < candidates[j].ProjectPath
	})

	return candidates, nil
}

func (s *Service) PreviewImport(ctx context.Context, projectPath string) (ImportPreviewResponse, error) {
	candidate, schematicPath, err := resolveProject(projectPath)
	if err != nil {
		return ImportPreviewResponse{}, err
	}

	csvBytes, err := s.exporter.ExportBOM(ctx, schematicPath)
	if err != nil {
		return ImportPreviewResponse{}, err
	}

	rows, err := parseBOM(csvBytes)
	if err != nil {
		return ImportPreviewResponse{}, err
	}

	previewRows := mapBOMRows(rows)
	summary := ImportPreviewSummary{TotalRows: len(previewRows)}
	for _, row := range previewRows {
		if row.Included {
			summary.IncludedRows++
		}
		if row.HasWarning {
			summary.WarningRows++
		}
	}

	return ImportPreviewResponse{
		SelectedProject: candidate,
		Rows:            previewRows,
		Summary:         summary,
	}, nil
}

func (c CLIExporter) ExportBOM(ctx context.Context, schematicPath string) ([]byte, error) {
	binary := strings.TrimSpace(c.Binary)
	if binary == "" {
		binary = defaultBinary
	}
	if _, err := exec.LookPath(binary); err != nil {
		return nil, fmt.Errorf("kicad-cli not found in PATH")
	}

	tmpFile, err := os.CreateTemp("", "component-manager-kicad-bom-*.csv")
	if err != nil {
		return nil, err
	}
	outputPath := tmpFile.Name()
	if closeErr := tmpFile.Close(); closeErr != nil {
		_ = os.Remove(outputPath)
		return nil, closeErr
	}
	defer os.Remove(outputPath)

	args := []string{
		"sch", "export", "bom",
		"--output", outputPath,
		"--format-preset", "CSV",
		"--fields", fieldList,
		"--labels", labelList,
		"--group-by", groupByList,
		"--include-excluded-from-bom",
		schematicPath,
	}
	cmd := exec.CommandContext(ctx, binary, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		return nil, fmt.Errorf("kicad-cli BOM export failed: %s", message)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func resolveProject(projectPath string) (ProjectCandidate, string, error) {
	cleanPath := filepath.Clean(strings.TrimSpace(projectPath))
	if cleanPath == "" || filepath.Ext(cleanPath) != ".kicad_pro" {
		return ProjectCandidate{}, "", fmt.Errorf("expected a .kicad_pro project path")
	}
	candidate := ProjectCandidate{
		Name:        strings.TrimSuffix(filepath.Base(cleanPath), ".kicad_pro"),
		ProjectPath: cleanPath,
		ProjectDir:  filepath.Dir(cleanPath),
	}

	preferred := filepath.Join(candidate.ProjectDir, candidate.Name+".kicad_sch")
	if info, err := os.Stat(preferred); err == nil && !info.IsDir() {
		return candidate, preferred, nil
	}

	matches, err := filepath.Glob(filepath.Join(candidate.ProjectDir, "*.kicad_sch"))
	if err != nil {
		return ProjectCandidate{}, "", err
	}
	if len(matches) == 1 {
		return candidate, matches[0], nil
	}
	if len(matches) == 0 {
		return ProjectCandidate{}, "", fmt.Errorf("could not find a .kicad_sch file next to %s", cleanPath)
	}
	return ProjectCandidate{}, "", fmt.Errorf("could not determine the main schematic for %s", cleanPath)
}

func parseBOM(data []byte) ([]bomRow, error) {
	reader := csv.NewReader(bytes.NewReader(data))
	reader.FieldsPerRecord = -1
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return []bomRow{}, nil
	}

	headers := records[0]
	rows := make([]bomRow, 0, len(records)-1)
	for _, record := range records[1:] {
		if isBlankRecord(record) {
			continue
		}
		values := make(map[string]string, len(headers))
		for i, header := range headers {
			if i < len(record) {
				values[normalizeHeader(header)] = strings.TrimSpace(record[i])
				continue
			}
			values[normalizeHeader(header)] = ""
		}

		qty, err := parseQuantity(values[normalizeHeader("Qty")])
		if err != nil {
			qty = 0
		}

		row := bomRow{
			Refs:             values[normalizeHeader("Refs")],
			Value:            values[normalizeHeader("Value")],
			Footprint:        values[normalizeHeader("Footprint")],
			Description:      values[normalizeHeader("Description")],
			Manufacturer:     values[normalizeHeader("Manufacturer")],
			MPN:              values[normalizeHeader("MPN")],
			ManufacturerPart: values[normalizeHeader("ManufacturerPartNumber")],
			PartNumber:       values[normalizeHeader("PartNumber")],
			LCSC:             values[normalizeHeader("LCSC")],
			Datasheet:        values[normalizeHeader("Datasheet")],
			Quantity:         qty,
			DNP:              values[normalizeHeader("DNP")],
			OtherFields:      make(map[string]string),
		}
		if row.MPN == "" {
			row.MPN = firstNonEmpty(row.ManufacturerPart, row.PartNumber)
		}
		for key, value := range values {
			if value == "" {
				continue
			}
			switch key {
			case normalizeHeader("Refs"), normalizeHeader("Value"), normalizeHeader("Footprint"), normalizeHeader("Description"), normalizeHeader("Manufacturer"), normalizeHeader("MPN"), normalizeHeader("ManufacturerPartNumber"), normalizeHeader("PartNumber"), normalizeHeader("LCSC"), normalizeHeader("Datasheet"), normalizeHeader("Qty"), normalizeHeader("DNP"):
			default:
				row.OtherFields[key] = value
			}
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func normalizeHeader(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	replacer := strings.NewReplacer(" ", "", "-", "", "_", "", "/", "")
	return replacer.Replace(value)
}

func parseQuantity(raw string) (int, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0, fmt.Errorf("quantity missing")
	}
	if qty, err := strconv.Atoi(trimmed); err == nil {
		return qty, nil
	}
	asFloat, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return 0, err
	}
	return int(asFloat), nil
}

func matchesQuery(query string, candidate ProjectCandidate) bool {
	if query == "" {
		return true
	}
	haystack := strings.ToLower(candidate.Name + " " + candidate.ProjectPath)
	for _, token := range strings.Fields(query) {
		if !strings.Contains(haystack, token) {
			return false
		}
	}
	return true
}

func isBlankRecord(record []string) bool {
	for _, value := range record {
		if strings.TrimSpace(value) != "" {
			return false
		}
	}
	return true
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
