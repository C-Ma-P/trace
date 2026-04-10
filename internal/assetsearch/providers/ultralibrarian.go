package providers

import (
	"context"
	"fmt"

	"trace/internal/assetsearch"
)

// UltraLibrarian is a stub provider for ultralibrarian.com.
type UltraLibrarian struct{}

func (p *UltraLibrarian) Name() string        { return "ultralibrarian" }
func (p *UltraLibrarian) DisplayName() string { return "Ultra Librarian" }

func (p *UltraLibrarian) Search(_ context.Context, _ assetsearch.SearchRequest) ([]assetsearch.SearchCandidate, error) {
	return nil, fmt.Errorf("Ultra Librarian provider not implemented")
}

func (p *UltraLibrarian) Import(_ context.Context, _ assetsearch.ImportRequest) (assetsearch.ImportResponse, error) {
	return assetsearch.ImportResponse{}, fmt.Errorf("Ultra Librarian provider not implemented")
}
