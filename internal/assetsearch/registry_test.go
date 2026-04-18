package assetsearch

import (
	"context"
	"errors"
	"testing"
)

type testProvider struct {
	name        string
	displayName string
	searchFn    func(context.Context, SearchRequest) ([]SearchCandidate, error)
	importFn    func(context.Context, ImportRequest) (ImportResponse, error)
}

func (p *testProvider) Name() string        { return p.name }
func (p *testProvider) DisplayName() string { return p.displayName }
func (p *testProvider) Search(ctx context.Context, req SearchRequest) ([]SearchCandidate, error) {
	if p.searchFn == nil {
		return nil, nil
	}
	return p.searchFn(ctx, req)
}
func (p *testProvider) Import(ctx context.Context, req ImportRequest) (ImportResponse, error) {
	if p.importFn == nil {
		return ImportResponse{}, nil
	}
	return p.importFn(ctx, req)
}

func TestRegistryRegisterGetAll(t *testing.T) {
	r := NewRegistry()
	p1 := &testProvider{name: "p1", displayName: "Provider One"}
	p2 := &testProvider{name: "p2", displayName: "Provider Two"}

	r.Register(p1)
	r.Register(p2)

	if got := r.Get("p1"); got != p1 {
		t.Fatalf("expected provider p1")
	}
	if got := r.Get("missing"); got != nil {
		t.Fatalf("expected missing provider to be nil")
	}

	all := r.All()
	if len(all) != 2 {
		t.Fatalf("expected 2 providers, got %d", len(all))
	}
	if all[0].Name() != "p1" || all[1].Name() != "p2" {
		t.Fatalf("unexpected provider order: %q, %q", all[0].Name(), all[1].Name())
	}
}

func TestRegistryRegisterOverwriteKeepsOrder(t *testing.T) {
	r := NewRegistry()
	first := &testProvider{name: "p1", displayName: "Provider One"}
	replacement := &testProvider{name: "p1", displayName: "Provider One v2"}

	r.Register(first)
	r.Register(replacement)

	all := r.All()
	if len(all) != 1 {
		t.Fatalf("expected 1 provider after overwrite, got %d", len(all))
	}
	if all[0] != replacement {
		t.Fatalf("expected overwritten provider to be returned")
	}
}

func TestSearchForComponent_RequiresComponentID(t *testing.T) {
	svc := NewService(NewRegistry(), &componentRepoStub{}, &assetRepoStub{})
	_, err := svc.SearchForComponent(context.Background(), SearchRequest{})
	if err == nil {
		t.Fatal("expected error for missing component ID")
	}
}

func TestSearchForComponent_LooksUpComponentWhenRequestMissingIdentity(t *testing.T) {
	provider := &testProvider{
		name:        "p1",
		displayName: "Provider One",
		searchFn: func(_ context.Context, req SearchRequest) ([]SearchCandidate, error) {
			if req.MPN != "MPN-123" {
				t.Fatalf("expected looked-up MPN, got %q", req.MPN)
			}
			if req.Manufacturer != "Acme" {
				t.Fatalf("expected looked-up manufacturer, got %q", req.Manufacturer)
			}
			return []SearchCandidate{{ExternalID: "ext-1", Title: "Candidate"}}, nil
		},
	}

	r := NewRegistry()
	r.Register(provider)

	svc := NewService(
		r,
		&componentRepoStub{getComponentResult: componentRef("c1", "MPN-123", "Acme")},
		&assetRepoStub{},
	)

	resp, err := svc.SearchForComponent(context.Background(), SearchRequest{ComponentID: "c1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.ProviderResults) != 1 {
		t.Fatalf("expected 1 provider result, got %d", len(resp.ProviderResults))
	}
	if len(resp.ProviderResults[0].Candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(resp.ProviderResults[0].Candidates))
	}
}

func TestSearchForComponent_ProviderPartialFailure(t *testing.T) {
	r := NewRegistry()
	r.Register(&testProvider{
		name:        "ok",
		displayName: "OK",
		searchFn: func(_ context.Context, _ SearchRequest) ([]SearchCandidate, error) {
			return []SearchCandidate{{ExternalID: "ok-1"}}, nil
		},
	})
	r.Register(&testProvider{
		name:        "bad",
		displayName: "Bad",
		searchFn: func(_ context.Context, _ SearchRequest) ([]SearchCandidate, error) {
			return nil, errors.New("provider unavailable")
		},
	})

	svc := NewService(r, &componentRepoStub{}, &assetRepoStub{})
	resp, err := svc.SearchForComponent(context.Background(), SearchRequest{
		ComponentID: "c1",
		MPN:         "MPN-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.ProviderResults) != 2 {
		t.Fatalf("expected 2 provider results, got %d", len(resp.ProviderResults))
	}
	if resp.ProviderResults[0].Error != "" {
		t.Fatalf("expected first provider to succeed")
	}
	if resp.ProviderResults[1].Error == "" {
		t.Fatalf("expected second provider error to be captured")
	}
}
