package service

import (
	"bufio"
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/C-Ma-P/trace/internal/domain"
	"github.com/C-Ma-P/trace/internal/kicad"
	"github.com/C-Ma-P/trace/internal/kicad/project"
)

func (s *Service) ExportProjectKiCad(ctx context.Context, projectID string) (kicad.ExportOutput, error) {

	plan, err := s.PlanProject(ctx, projectID)
	if err != nil {
		return kicad.ExportOutput{}, fmt.Errorf("load project plan: %w", err)
	}

	var parts []kicad.ExportedPart
	var warnings []string
	refCounters := make(map[string]int)

	schematicUUID := newUUID()

	for _, rp := range plan.Requirements {
		if rp.Readiness.Status != domain.ReadinessReady {
			warnings = append(warnings, fmt.Sprintf("skipping %q: %s (%s)",
				rp.Requirement.Name,
				rp.Readiness.Status,
				strings.Join(rp.Readiness.Blockers, "; ")))
			continue
		}

		preferred := preferredCandidate(rp.Candidates)
		if preferred == nil || preferred.ComponentID == nil {
			warnings = append(warnings, fmt.Sprintf("skipping %q: no preferred component ID", rp.Requirement.Name))
			continue
		}

		detail, err := s.assets.GetComponentWithAssets(ctx, *preferred.ComponentID)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("skipping %q: cannot load assets: %v", rp.Requirement.Name, err))
			continue
		}
		if detail.SelectedSymbolAsset == nil {
			warnings = append(warnings, fmt.Sprintf("skipping %q: no selected symbol asset", rp.Requirement.Name))
			continue
		}

		comp := detail.Component
		symAsset := detail.SelectedSymbolAsset

		prefix := refPrefix(comp.Category)
		refCounters[prefix]++
		ref := fmt.Sprintf("%s%d", prefix, refCounters[prefix])

		symKey := sanitizeSymKey(comp.ID)
		symValue := comp.MPN
		if symValue == "" {
			symValue = comp.Description
		}
		if symValue == "" {
			symValue = comp.ID[:8]
		}

		fpRef := ""
		fpSrcPath := ""
		fpModName := ""
		if detail.SelectedFootprintAsset != nil {
			fpSrcPath = detail.SelectedFootprintAsset.URLOrPath
			fpModName, fpRef = footprintRef(detail.SelectedFootprintAsset)
		}

		datasheet := ""
		if detail.SelectedDatasheetAsset != nil {
			datasheet = detail.SelectedDatasheetAsset.URLOrPath
			if strings.HasPrefix(datasheet, "http://") || strings.HasPrefix(datasheet, "https://") {

			} else {
				datasheet = ""
			}
		}

		part := kicad.ExportedPart{
			UUID:                newUUID(),
			Reference:           ref,
			Value:               symValue,
			DisplayName:         componentDefinitionLabel(comp),
			Category:            string(comp.Category),
			SymbolLibKey:        symKey,
			SymbolSrcPath:       symAsset.URLOrPath,
			FootprintRef:        fpRef,
			FootprintSrcPath:    fpSrcPath,
			FootprintModuleName: fpModName,
			Manufacturer:        comp.Manufacturer,
			MPN:                 comp.MPN,
			Package:             comp.Package,
			Datasheet:           datasheet,
			InBOM:               true,
			OnBoard:             true,
		}
		parts = append(parts, part)
	}

	if len(parts) == 0 {
		return kicad.ExportOutput{}, fmt.Errorf("no ready parts to export (see warnings: %s)",
			strings.Join(warnings, "; "))
	}

	projectName := plan.Project.Name
	if projectName == "" {
		projectName = "trace_export"
	}

	output, err := project.Export(kicad.ExportInput{
		ProjectName:   projectName,
		SchematicUUID: schematicUUID,
		Parts:         parts,
	})
	if err != nil {
		return kicad.ExportOutput{}, fmt.Errorf("generate KiCad files: %w", err)
	}

	output.Warnings = append(warnings, output.Warnings...)
	return output, nil
}

func preferredCandidate(candidates []domain.ProjectPartCandidate) *domain.ProjectPartCandidate {
	for i := range candidates {
		if candidates[i].Preferred {
			return &candidates[i]
		}
	}
	return nil
}

func refPrefix(cat domain.Category) string {
	switch cat {
	case domain.CategoryResistor:
		return "R"
	case domain.CategoryCapacitor:
		return "C"
	case domain.CategoryInductor:
		return "L"
	case domain.CategoryFerriteBead:
		return "FB"
	case domain.CategoryDiode:
		return "D"
	case domain.CategoryLED:
		return "D"
	case domain.CategoryTransistorBJT:
		return "Q"
	case domain.CategoryTransistorMOSFET:
		return "Q"
	case domain.CategoryRegulatorLinear, domain.CategoryRegulatorSwitching:
		return "U"
	case domain.CategoryIntegratedCircuit:
		return "U"
	case domain.CategoryConnector:
		return "J"
	case domain.CategorySwitch:
		return "SW"
	case domain.CategoryCrystalOscillator:
		return "Y"
	case domain.CategoryFuse:
		return "F"
	case domain.CategoryBattery:
		return "BT"
	case domain.CategorySensor:
		return "U"
	case domain.CategoryModule:
		return "U"
	default:
		return "U"
	}
}

var nonAlnumSym = regexp.MustCompile(`[^A-Za-z0-9_]`)

func sanitizeSymKey(id string) string {
	s := nonAlnumSym.ReplaceAllString(id, "_")
	s = strings.Trim(s, "_")
	if len(s) > 64 {
		s = s[:64]
	}
	if s == "" {
		return "COMP"
	}
	return s
}

func footprintRef(asset *domain.ComponentAsset) (moduleName, footprintRef string) {

	if mod, ok := readModuleName(asset.URLOrPath); ok {

		parts := strings.SplitN(mod, ":", 2)
		if len(parts) == 2 {

			return sanitizeFileName(parts[1]), mod
		}

		return sanitizeFileName(mod), "trace_fp:" + mod
	}

	label := strings.TrimSpace(asset.Label)
	if label != "" {
		converted := strings.Replace(label, "/", ":", 1)
		if strings.Contains(converted, ":") {
			parts := strings.SplitN(converted, ":", 2)
			return sanitizeFileName(parts[1]), converted
		}
		return sanitizeFileName(label), "trace_fp:" + label
	}

	stem := strings.TrimSuffix(filepath.Base(asset.URLOrPath), ".kicad_mod")
	stem = strings.TrimSuffix(stem, filepath.Ext(stem))
	return sanitizeFileName(stem), "trace_fp:" + stem
}

func readModuleName(path string) (string, bool) {
	if path == "" {
		return "", false
	}
	f, err := os.Open(path)
	if err != nil {
		return "", false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var name string
		for _, kw := range []string{"module ", "footprint "} {
			if idx := strings.Index(line, "("+kw); idx >= 0 {
				rest := line[idx+1+len(kw):]
				name = extractFirstToken(rest)
				break
			}
		}
		if name != "" {
			return name, true
		}

		break
	}
	return "", false
}

func extractFirstToken(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, `"`) {

		end := strings.Index(s[1:], `"`)
		if end < 0 {
			return ""
		}
		return s[1 : end+1]
	}

	for i, c := range s {
		if c == ' ' || c == '\t' || c == '(' || c == ')' || c == '\n' {
			return s[:i]
		}
	}
	return s
}

var nonAlnumFile = regexp.MustCompile(`[^A-Za-z0-9_\-\.]`)

func sanitizeFileName(s string) string {
	s = strings.TrimSpace(s)
	s = nonAlnumFile.ReplaceAllString(s, "_")

	return strings.Trim(s, "_")
}

func newUUID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
