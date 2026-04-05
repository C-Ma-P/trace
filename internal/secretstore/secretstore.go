package secretstore

import (
	"errors"
	"fmt"
	"strings"

	keyring "github.com/zalando/go-keyring"
)

var ErrNotFound = errors.New("secret not found")
var ErrUnavailable = errors.New("secret store unavailable")

type Store interface {
	Set(key string, value string) error
	Get(key string) (string, error)
	Delete(key string) error
	Available() bool
}

type KeyringStore struct {
	serviceName string
	probeKey    string
}

func NewKeyringStore(serviceName string) *KeyringStore {
	serviceName = strings.TrimSpace(serviceName)
	if serviceName == "" {
		serviceName = "trace"
	}
	return &KeyringStore{serviceName: serviceName, probeKey: "__trace_probe__"}
}

func (s *KeyringStore) Available() bool {
	_, err := keyring.Get(s.serviceName, s.probeKey)
	return err == nil || errors.Is(err, keyring.ErrNotFound)
}

func (s *KeyringStore) Set(key string, value string) error {
	if !s.Available() {
		return ErrUnavailable
	}
	if err := keyring.Set(s.serviceName, key, value); err != nil {
		return mapError(err)
	}
	return nil
}

func (s *KeyringStore) Get(key string) (string, error) {
	if !s.Available() {
		return "", ErrUnavailable
	}
	value, err := keyring.Get(s.serviceName, key)
	if err != nil {
		return "", mapError(err)
	}
	return value, nil
}

func (s *KeyringStore) Delete(key string) error {
	if !s.Available() {
		return ErrUnavailable
	}
	if err := keyring.Delete(s.serviceName, key); err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return nil
		}
		return mapError(err)
	}
	return nil
}

func mapError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, keyring.ErrNotFound) {
		return ErrNotFound
	}
	if errors.Is(err, keyring.ErrUnsupportedPlatform) {
		return ErrUnavailable
	}
	return fmt.Errorf("%w: %v", ErrUnavailable, err)
}

var _ Store = (*KeyringStore)(nil)
