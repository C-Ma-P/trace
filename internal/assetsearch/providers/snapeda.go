package providers

import (
	"context"
	"fmt"

	"trace/internal/assetsearch"
)

// SnapEDA is a stub provider for snapeda.com.
type SnapEDA struct{}

func (p *SnapEDA) Name() string        { return "snapeda" }
func (p *SnapEDA) DisplayName() string { return "SnapEDA" }

func (p *SnapEDA) Search(_ context.Context, _ assetsearch.SearchRequest) ([]assetsearch.SearchCandidate, error) {
	return nil, fmt.Errorf("SnapEDA provider not implemented")
}

func (p *SnapEDA) Import(_ context.Context, _ assetsearch.ImportRequest) (assetsearch.ImportResponse, error) {
	return assetsearch.ImportResponse{}, fmt.Errorf("SnapEDA provider not implemented")
}
