package secretstore

import (
	"errors"
	"testing"

	keyring "github.com/zalando/go-keyring"
)

func TestNewKeyringStore_DefaultsAndTrim(t *testing.T) {
	store := NewKeyringStore("  custom  ")
	if store.serviceName != "custom" {
		t.Fatalf("expected trimmed service name, got %q", store.serviceName)
	}
	if store.probeKey == "" {
		t.Fatalf("expected probe key to be set")
	}

	defaultStore := NewKeyringStore("   ")
	if defaultStore.serviceName != "trace" {
		t.Fatalf("expected default service name trace, got %q", defaultStore.serviceName)
	}
}

func TestMapError(t *testing.T) {
	if err := mapError(nil); err != nil {
		t.Fatalf("expected nil error mapping")
	}

	notFound := mapError(keyring.ErrNotFound)
	if !errors.Is(notFound, ErrNotFound) {
		t.Fatalf("expected ErrNotFound mapping, got %v", notFound)
	}

	unsupported := mapError(keyring.ErrUnsupportedPlatform)
	if !errors.Is(unsupported, ErrUnavailable) {
		t.Fatalf("expected ErrUnavailable mapping, got %v", unsupported)
	}

	generic := mapError(errors.New("boom"))
	if !errors.Is(generic, ErrUnavailable) {
		t.Fatalf("expected generic errors to wrap ErrUnavailable, got %v", generic)
	}
}
