package app

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/C-Ma-P/trace/internal/activity"
)

func (a *App) ExportProjectKiCad(projectID string) (KiCadExportResponse, error) {
	if err := a.checkReady(); err != nil {
		return KiCadExportResponse{}, err
	}

	output, err := a.svc.ExportProjectKiCad(context.Background(), projectID)
	if err != nil {
		return KiCadExportResponse{}, err
	}

	defer os.RemoveAll(output.Dir)
	defer os.Remove(output.ZipPath)

	zipBytes, err := os.ReadFile(output.ZipPath)
	if err != nil {
		return KiCadExportResponse{}, err
	}

	filename := filepath.Base(output.ZipPath)
	if !strings.HasSuffix(filename, ".zip") {
		filename = "kicad_export.zip"
	}

	return KiCadExportResponse{
		ZipBase64: base64.StdEncoding.EncodeToString(zipBytes),
		Filename:  filename,
		Warnings:  output.Warnings,
	}, nil
}

func (a *App) ExportProjectKiCadToDir(projectID, destDir string) (KiCadExportToDirResult, error) {
	if err := a.checkReady(); err != nil {
		return KiCadExportToDirResult{}, err
	}
	if destDir == "" {
		return KiCadExportToDirResult{}, fmt.Errorf("destination directory must not be empty")
	}

	emit := func(sev activity.Severity, msg string, meta map[string]any) {
		if a.activityHub != nil {
			a.activityHub.Emit(activity.Event{
				Domain:   activity.DomainExport,
				Severity: sev,
				Kind:     "kicad",
				Message:  msg,
				Metadata: meta,
			})
		}
	}

	emit(activity.SeverityInfo, fmt.Sprintf("starting KiCad export for project %s", projectID), nil)

	output, err := a.svc.ExportProjectKiCad(context.Background(), projectID)
	defer func() {
		os.RemoveAll(output.Dir)
		os.Remove(output.ZipPath)
	}()
	if err != nil {
		emit(activity.SeverityError, fmt.Sprintf("KiCad export failed: %v", err), nil)
		return KiCadExportToDirResult{}, err
	}

	filename := filepath.Base(output.ZipPath)
	if !strings.HasSuffix(filename, ".zip") {
		filename = "kicad_export.zip"
	}

	destPath := filepath.Join(destDir, filename)
	if err := copyFile(output.ZipPath, destPath); err != nil {
		emit(activity.SeverityError, fmt.Sprintf("failed to save export to %s: %v", destPath, err), nil)
		return KiCadExportToDirResult{}, fmt.Errorf("save export: %w", err)
	}

	for _, w := range output.Warnings {
		emit(activity.SeverityWarning, w, map[string]any{"savedPath": destPath})
	}

	msg := fmt.Sprintf("KiCad export saved to %s", destPath)
	if len(output.Warnings) > 0 {
		msg += fmt.Sprintf(" (%d warning(s))", len(output.Warnings))
	}
	emit(activity.SeveritySuccess, msg, map[string]any{"savedPath": destPath})

	return KiCadExportToDirResult{
		SavedPath: destPath,
		Warnings:  output.Warnings,
	}, nil
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
