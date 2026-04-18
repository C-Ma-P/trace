package providers

import (
	"context"
	"strings"
	"testing"

	"github.com/C-Ma-P/trace/internal/assetsearch"
)

func TestSnapEDAStub(t *testing.T) {
	p := &SnapEDA{}
	if p.Name() != "snapeda" {
		t.Fatalf("unexpected name: %q", p.Name())
	}
	if p.DisplayName() != "SnapEDA" {
		t.Fatalf("unexpected display name: %q", p.DisplayName())
	}

	_, err := p.Search(context.Background(), assetsearch.SearchRequest{})
	if err == nil || !strings.Contains(err.Error(), "not implemented") {
		t.Fatalf("expected not implemented error, got: %v", err)
	}

	_, err = p.Import(context.Background(), assetsearch.ImportRequest{})
	if err == nil || !strings.Contains(err.Error(), "not implemented") {
		t.Fatalf("expected not implemented error, got: %v", err)
	}
}

func TestUltraLibrarianStub(t *testing.T) {
	p := &UltraLibrarian{}
	if p.Name() != "ultralibrarian" {
		t.Fatalf("unexpected name: %q", p.Name())
	}
	if p.DisplayName() != "Ultra Librarian" {
		t.Fatalf("unexpected display name: %q", p.DisplayName())
	}

	_, err := p.Search(context.Background(), assetsearch.SearchRequest{})
	if err == nil || !strings.Contains(err.Error(), "not implemented") {
		t.Fatalf("expected not implemented error, got: %v", err)
	}

	_, err = p.Import(context.Background(), assetsearch.ImportRequest{})
	if err == nil || !strings.Contains(err.Error(), "not implemented") {
		t.Fatalf("expected not implemented error, got: %v", err)
	}
}
