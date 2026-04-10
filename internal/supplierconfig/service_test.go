package supplierconfig

import (
	"context"
	"strings"
	"testing"

	"trace/internal/secretstore"
)

type fakePreferenceRepo struct {
	values map[string]string
}

func newFakePreferenceRepo() *fakePreferenceRepo {
	return &fakePreferenceRepo{values: map[string]string{}}
}

func (f *fakePreferenceRepo) List(_ context.Context, prefix string) (map[string]string, error) {
	result := map[string]string{}
	for key, value := range f.values {
		if strings.HasPrefix(key, prefix) {
			result[key] = value
		}
	}
	return result, nil
}

func (f *fakePreferenceRepo) SetMany(_ context.Context, values map[string]string) error {
	for key, value := range values {
		f.values[key] = value
	}
	return nil
}

type fakeSecretStore struct {
	available bool
	values    map[string]string
}

func newFakeSecretStore(available bool) *fakeSecretStore {
	return &fakeSecretStore{available: available, values: map[string]string{}}
}

func (f *fakeSecretStore) Set(key string, value string) error {
	if !f.available {
		return secretstore.ErrUnavailable
	}
	f.values[key] = value
	return nil
}

func (f *fakeSecretStore) Get(key string) (string, error) {
	if !f.available {
		return "", secretstore.ErrUnavailable
	}
	value, ok := f.values[key]
	if !ok {
		return "", secretstore.ErrNotFound
	}
	return value, nil
}

func (f *fakeSecretStore) Delete(key string) error {
	if !f.available {
		return secretstore.ErrUnavailable
	}
	delete(f.values, key)
	return nil
}

func (f *fakeSecretStore) Available() bool {
	return f.available
}

func TestSavePreferences_PersistsNonSecretValuesAndSecrets(t *testing.T) {
	repo := newFakePreferenceRepo()
	secrets := newFakeSecretStore(true)
	svc := NewManager(repo, secrets, nil)
	clientSecret := "top-secret"

	prefs, err := svc.SavePreferences(context.Background(), UpdateInput{
		DigiKey: DigiKeyInput{
			Enabled:             true,
			ClientID:            "trace-client",
			CustomerID:          "customer-1",
			Site:                "US",
			Language:            "en",
			Currency:            "USD",
			ReplaceClientSecret: &clientSecret,
		},
		Mouser: MouserInput{Enabled: false},
		LCSC:   LCSCInput{Enabled: true, Currency: "USD"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := repo.values[prefDigiKeyClientID]; got != "trace-client" {
		t.Fatalf("expected client ID to be persisted, got %q", got)
	}
	if got := repo.values[prefMouserEnabled]; got != "false" {
		t.Fatalf("expected mouser enabled flag to be persisted, got %q", got)
	}
	if got := secrets.values[secretDigiKeyClientSecret]; got != clientSecret {
		t.Fatalf("expected secret to be written to secret store, got %q", got)
	}
	if !prefs.DigiKey.ClientSecretStored {
		t.Fatalf("expected returned preferences to report stored client secret")
	}
	if prefs.DigiKey.Status.State != "configured" {
		t.Fatalf("expected DigiKey to be configured, got %#v", prefs.DigiKey.Status)
	}
	if prefs.Mouser.Status.State != "disabled" {
		t.Fatalf("expected Mouser to be disabled, got %#v", prefs.Mouser.Status)
	}
}

func TestResolve_PrefersSavedValuesAndFallsBackToEnvironment(t *testing.T) {
	repo := newFakePreferenceRepo()
	repo.values[prefDigiKeyEnabled] = "true"
	repo.values[prefDigiKeyClientID] = "saved-client"
	repo.values[prefMouserEnabled] = "true"
	repo.values[prefLCSCEnabled] = "true"
	secrets := newFakeSecretStore(true)
	secrets.values[secretDigiKeyClientSecret] = "saved-secret"
	env := map[string]string{
		"DIGIKEY_CLIENT_ID":     "env-client",
		"DIGIKEY_CLIENT_SECRET": "env-secret",
		"MOUSER_API_KEY":        "env-mouser",
		"LCSC_CURRENCY":         "EUR",
	}
	svc := NewManager(repo, secrets, func(key string) string { return env[key] })

	resolved, err := svc.Resolve(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resolved.Config.DigiKey.ClientID != "saved-client" {
		t.Fatalf("expected saved DigiKey client ID to win, got %q", resolved.Config.DigiKey.ClientID)
	}
	if resolved.Config.DigiKey.ClientSecret != "saved-secret" {
		t.Fatalf("expected saved DigiKey secret to win, got %q", resolved.Config.DigiKey.ClientSecret)
	}
	if resolved.Config.Mouser.APIKey != "env-mouser" {
		t.Fatalf("expected environment Mouser key fallback, got %q", resolved.Config.Mouser.APIKey)
	}
	if resolved.Config.LCSC.Currency != "EUR" {
		t.Fatalf("expected LCSC currency from environment fallback, got %q", resolved.Config.LCSC.Currency)
	}
	if resolved.Preferences.DigiKey.Status.Source != "preferences" {
		t.Fatalf("expected DigiKey source to be preferences, got %#v", resolved.Preferences.DigiKey.Status)
	}
	if resolved.Preferences.Mouser.Status.Source != "environment" {
		t.Fatalf("expected Mouser source to be environment, got %#v", resolved.Preferences.Mouser.Status)
	}
}

func TestResolve_ReportsIncompleteWhenSecureStorageUnavailable(t *testing.T) {
	repo := newFakePreferenceRepo()
	repo.values[prefDigiKeyEnabled] = "true"
	repo.values[prefDigiKeyClientID] = "saved-client"
	secrets := newFakeSecretStore(false)
	svc := NewManager(repo, secrets, nil)

	resolved, err := svc.Resolve(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resolved.Preferences.SecureStorageAvailable {
		t.Fatalf("expected secure storage to be unavailable")
	}
	if resolved.Preferences.DigiKey.Status.State != "incomplete" {
		t.Fatalf("expected DigiKey to be incomplete, got %#v", resolved.Preferences.DigiKey.Status)
	}
	if resolved.Preferences.DigiKey.Status.StorageMode != "unavailable" {
		t.Fatalf("expected storage mode unavailable, got %#v", resolved.Preferences.DigiKey.Status)
	}
	if resolved.Config.DigiKey.ClientSecret != "" {
		t.Fatalf("expected missing client secret when storage unavailable without env fallback")
	}
}

func TestClearSecret_RemovesStoredSecretAndFallsBackToEnvironment(t *testing.T) {
	repo := newFakePreferenceRepo()
	repo.values[prefMouserEnabled] = "true"
	secrets := newFakeSecretStore(true)
	secrets.values[secretMouserAPIKey] = "stored-key"
	env := map[string]string{"MOUSER_API_KEY": "env-key"}
	svc := NewManager(repo, secrets, func(key string) string { return env[key] })

	prefs, err := svc.ClearSecret(context.Background(), "mouser", "api_key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := secrets.values[secretMouserAPIKey]; ok {
		t.Fatalf("expected Mouser secret to be deleted from secret store")
	}
	if prefs.Mouser.APIKeyStored {
		t.Fatalf("expected preferences to report no stored API key after clearing")
	}
	if prefs.Mouser.Status.Source != "environment" {
		t.Fatalf("expected environment fallback after clearing secret, got %#v", prefs.Mouser.Status)
	}
}
