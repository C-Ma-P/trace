package assetsearch

import "context"

// Provider is the interface each asset source must implement.
type Provider interface {
	// Name returns a stable identifier for this provider (e.g. "snapeda").
	Name() string

	// DisplayName returns a human-readable label (e.g. "SnapEDA").
	DisplayName() string

	// Search returns normalized candidates matching the request.
	// Implementations should return an empty slice (not an error) when
	// the provider simply has no results.
	Search(ctx context.Context, req SearchRequest) ([]SearchCandidate, error)

	// Import fetches and normalizes assets for a single candidate.
	// The caller is responsible for persisting the returned assets.
	Import(ctx context.Context, req ImportRequest) (ImportResponse, error)
}

// Registry holds the set of available providers.
type Registry struct {
	providers map[string]Provider
	order     []string // insertion order for deterministic iteration
}

// NewRegistry creates an empty provider registry.
func NewRegistry() *Registry {
	return &Registry{providers: make(map[string]Provider)}
}

// Register adds a provider. Overwrites any previous provider with the same name.
func (r *Registry) Register(p Provider) {
	name := p.Name()
	if _, exists := r.providers[name]; !exists {
		r.order = append(r.order, name)
	}
	r.providers[name] = p
}

// Get returns a provider by name, or nil if not found.
func (r *Registry) Get(name string) Provider {
	return r.providers[name]
}

// All returns every registered provider in registration order.
func (r *Registry) All() []Provider {
	out := make([]Provider, 0, len(r.order))
	for _, name := range r.order {
		if p, ok := r.providers[name]; ok {
			out = append(out, p)
		}
	}
	return out
}
