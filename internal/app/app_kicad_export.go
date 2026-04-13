package app

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
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
