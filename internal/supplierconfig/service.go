package supplierconfig

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/C-Ma-P/trace/internal/activity"
	"github.com/C-Ma-P/trace/internal/domain"
	"github.com/C-Ma-P/trace/internal/secretstore"
	"github.com/C-Ma-P/trace/internal/sourcing"
)

const (
	preferencePrefix = "suppliers."

	prefDigiKeyEnabled    = "suppliers.digikey.enabled"
	prefDigiKeyClientID   = "suppliers.digikey.client_id"
	prefDigiKeyCustomerID = "suppliers.digikey.customer_id"
	prefDigiKeySite       = "suppliers.digikey.site"
	prefDigiKeyLanguage   = "suppliers.digikey.language"
	prefDigiKeyCurrency   = "suppliers.digikey.currency"

	prefMouserEnabled = "suppliers.mouser.enabled"

	prefLCSCEnabled  = "suppliers.lcsc.enabled"
	prefLCSCCurrency = "suppliers.lcsc.currency"

	secretDigiKeyClientID     = "trace.suppliers.digikey.client_id"
	secretDigiKeyClientSecret = "trace.suppliers.digikey.client_secret"
	secretMouserAPIKey        = "trace.suppliers.mouser.api_key"
)

type EnvLookup func(string) string

type Manager struct {
	prefs           domain.PreferenceRepository
	secrets         secretstore.Store
	env             EnvLookup
	activityEmitter activity.Emitter
	mu              sync.Mutex
	coordinator     *sourcing.Coordinator
}

type Preferences struct {
	SecureStorageAvailable bool            `json:"secureStorageAvailable"`
	SecureStorageMessage   string          `json:"secureStorageMessage"`
	DigiKey                DigiKeySettings `json:"digikey"`
	Mouser                 MouserSettings  `json:"mouser"`
	LCSC                   LCSCSettings    `json:"lcsc"`
}

type DigiKeySettings struct {
	Enabled            bool           `json:"enabled"`
	ClientID           string         `json:"clientId"`
	CustomerID         string         `json:"customerId"`
	Site               string         `json:"site"`
	Language           string         `json:"language"`
	Currency           string         `json:"currency"`
	ClientSecretStored bool           `json:"clientSecretStored"`
	Status             ProviderStatus `json:"status"`
}

type MouserSettings struct {
	Enabled      bool           `json:"enabled"`
	APIKeyStored bool           `json:"apiKeyStored"`
	Status       ProviderStatus `json:"status"`
}

type LCSCSettings struct {
	Enabled  bool           `json:"enabled"`
	Currency string         `json:"currency"`
	Status   ProviderStatus `json:"status"`
}

type ProviderStatus struct {
	Provider     string `json:"provider"`
	Enabled      bool   `json:"enabled"`
	Complete     bool   `json:"complete"`
	State        string `json:"state"`
	StorageMode  string `json:"storageMode"`
	Source       string `json:"source"`
	Message      string `json:"message"`
	HasSecret    bool   `json:"hasSecret"`
	SecretStored bool   `json:"secretStored"`
}

type UpdateInput struct {
	DigiKey DigiKeyInput `json:"digikey"`
	Mouser  MouserInput  `json:"mouser"`
	LCSC    LCSCInput    `json:"lcsc"`
}

type DigiKeyInput struct {
	Enabled             bool    `json:"enabled"`
	ClientID            string  `json:"clientId"`
	CustomerID          string  `json:"customerId"`
	Site                string  `json:"site"`
	Language            string  `json:"language"`
	Currency            string  `json:"currency"`
	ReplaceClientSecret *string `json:"replaceClientSecret"`
}

type MouserInput struct {
	Enabled       bool    `json:"enabled"`
	ReplaceAPIKey *string `json:"replaceApiKey"`
}

type LCSCInput struct {
	Enabled  bool   `json:"enabled"`
	Currency string `json:"currency"`
}

type ResolvedConfig struct {
	Config       sourcing.Config `json:"-"`
	Preferences  Preferences     `json:"preferences"`
	FromAppPrefs bool            `json:"-"`
}

type storedPreferences struct {
	DigiKeyEnabled    bool
	DigiKeyClientID   string
	DigiKeyCustomerID string
	DigiKeySite       string
	DigiKeyLanguage   string
	DigiKeyCurrency   string
	MouserEnabled     bool
	LCSCEnabled       bool
	LCSCCurrency      string
}

func NewManager(prefs domain.PreferenceRepository, secrets secretstore.Store, env EnvLookup, emitter activity.Emitter) *Manager {
	if env == nil {
		env = os.Getenv
	}
	if emitter == nil {
		emitter = activity.NopEmitter
	}
	return &Manager{prefs: prefs, secrets: secrets, env: env, activityEmitter: emitter}
}

func (s *Manager) GetPreferences(ctx context.Context) (Preferences, error) {
	resolved, err := s.Resolve(ctx)
	if err != nil {
		return Preferences{}, err
	}
	return resolved.Preferences, nil
}

func (s *Manager) SavePreferences(ctx context.Context, input UpdateInput) (Preferences, error) {
	if err := s.validateSecretWrites(input); err != nil {
		return Preferences{}, err
	}

	values := map[string]string{
		prefDigiKeyEnabled:    strconv.FormatBool(input.DigiKey.Enabled),
		prefDigiKeyClientID:   strings.TrimSpace(input.DigiKey.ClientID),
		prefDigiKeyCustomerID: strings.TrimSpace(input.DigiKey.CustomerID),
		prefDigiKeySite:       strings.TrimSpace(input.DigiKey.Site),
		prefDigiKeyLanguage:   strings.TrimSpace(input.DigiKey.Language),
		prefDigiKeyCurrency:   strings.TrimSpace(input.DigiKey.Currency),
		prefMouserEnabled:     strconv.FormatBool(input.Mouser.Enabled),
		prefLCSCEnabled:       strconv.FormatBool(input.LCSC.Enabled),
		prefLCSCCurrency:      strings.TrimSpace(input.LCSC.Currency),
	}
	if err := s.prefs.SetMany(ctx, values); err != nil {
		return Preferences{}, err
	}

	if s.secrets.Available() {
		if clientID := strings.TrimSpace(input.DigiKey.ClientID); clientID != "" {
			if err := s.secrets.Set(secretDigiKeyClientID, clientID); err != nil {
				return Preferences{}, err
			}
		} else {
			if err := s.secrets.Delete(secretDigiKeyClientID); err != nil && !errors.Is(err, secretstore.ErrNotFound) {
				return Preferences{}, err
			}
		}
	}

	if input.DigiKey.ReplaceClientSecret != nil {
		if err := s.secrets.Set(secretDigiKeyClientSecret, strings.TrimSpace(*input.DigiKey.ReplaceClientSecret)); err != nil {
			return Preferences{}, err
		}
	}
	if input.Mouser.ReplaceAPIKey != nil {
		if err := s.secrets.Set(secretMouserAPIKey, strings.TrimSpace(*input.Mouser.ReplaceAPIKey)); err != nil {
			return Preferences{}, err
		}
	}

	return s.GetPreferences(ctx)
}

func (s *Manager) ClearSecret(ctx context.Context, provider, secret string) (Preferences, error) {
	if strings.ToLower(strings.TrimSpace(provider)) == "digikey" && secret == "client_secret" {
		if err := s.secrets.Delete(secretDigiKeyClientSecret); err != nil {
			return Preferences{}, err
		}
		if err := s.secrets.Delete(secretDigiKeyClientID); err != nil && !errors.Is(err, secretstore.ErrNotFound) {
			return Preferences{}, err
		}
		return s.GetPreferences(ctx)
	}

	secretKey, err := secretKeyFor(provider, secret)
	if err != nil {
		return Preferences{}, err
	}
	if err := s.secrets.Delete(secretKey); err != nil {
		return Preferences{}, err
	}
	return s.GetPreferences(ctx)
}

func (s *Manager) BuildSourcingService(ctx context.Context) (*sourcing.Service, error) {
	coord, err := s.resolveSourcingCoordinator(ctx)
	if err != nil {
		return nil, err
	}
	return coord.Service(), nil
}

func (s *Manager) GetSourcingCoordinator(ctx context.Context) (*sourcing.Coordinator, error) {
	return s.resolveSourcingCoordinator(ctx)
}

func (s *Manager) resolveSourcingCoordinator(ctx context.Context) (*sourcing.Coordinator, error) {
	resolved, err := s.Resolve(ctx)
	if err != nil {
		return nil, err
	}
	fingerprint := sourcing.NormalizedConfigFingerprint(resolved.Config)

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.coordinator != nil && s.coordinator.ConfigFingerprint() == fingerprint {
		return s.coordinator, nil
	}
	s.coordinator = sourcing.NewCoordinatorWithEmitter(resolved.Config, s.activityEmitter)
	return s.coordinator, nil
}

func (s *Manager) Resolve(ctx context.Context) (ResolvedConfig, error) {
	stored, err := s.loadStored(ctx)
	if err != nil {
		return ResolvedConfig{}, err
	}

	secureAvailable := s.secrets != nil && s.secrets.Available()
	preferences := Preferences{SecureStorageAvailable: secureAvailable}
	if secureAvailable {
		preferences.SecureStorageMessage = "Supplier secrets are stored in the system credential store when you save them here."
	} else {
		preferences.SecureStorageMessage = "Secure storage is unavailable. Trace will not save supplier secrets locally; use environment variables instead."
	}

	digiKeySecret, digiKeySecretSource, digiKeySecretStored, err := s.resolveSecret(secretDigiKeyClientSecret, "DIGIKEY_CLIENT_SECRET")
	if err != nil {
		return ResolvedConfig{}, err
	}
	mouserAPIKey, mouserSecretSource, mouserSecretStored, err := s.resolveSecret(secretMouserAPIKey, "MOUSER_API_KEY")
	if err != nil {
		return ResolvedConfig{}, err
	}

	digiKeyClientID, digiKeyClientIDSource, _, err := s.resolvePreferenceValue(stored.DigiKeyClientID, secretDigiKeyClientID, "DIGIKEY_CLIENT_ID")
	if err != nil {
		return ResolvedConfig{}, err
	}
	digiKeyCustomerID, _ := pickValue(stored.DigiKeyCustomerID, strings.TrimSpace(s.env("DIGIKEY_CUSTOMER_ID")))
	digiKeySite, _ := pickValue(stored.DigiKeySite, strings.TrimSpace(s.env("DIGIKEY_SITE")))
	digiKeyLanguage, _ := pickValue(stored.DigiKeyLanguage, strings.TrimSpace(s.env("DIGIKEY_LANGUAGE")))
	digiKeyCurrency, _ := pickValue(stored.DigiKeyCurrency, strings.TrimSpace(s.env("DIGIKEY_CURRENCY")))
	lcscCurrency, lcscCurrencySource := pickValue(stored.LCSCCurrency, strings.TrimSpace(s.env("LCSC_CURRENCY")))

	digiKeyMissing := missingFields(
		requiredField("client ID", digiKeyClientID),
		requiredField("client secret", digiKeySecret),
	)
	mouserMissing := missingFields(requiredField("API key", mouserAPIKey))

	digiKeySource := mergeSources(digiKeyClientIDSource, digiKeySecretSource)
	mouserSource := mergeSources("missing", mouserSecretSource)
	lcscSource := lcscCurrencySource
	if lcscSource == "missing" {
		lcscSource = "preferences"
	}

	preferences.DigiKey = DigiKeySettings{
		Enabled:            stored.DigiKeyEnabled,
		ClientID:           digiKeyClientID,
		CustomerID:         digiKeyCustomerID,
		Site:               digiKeySite,
		Language:           digiKeyLanguage,
		Currency:           digiKeyCurrency,
		ClientSecretStored: digiKeySecretStored,
		Status: providerStatus(providerStatusInput{
			Provider:        sourcing.ProviderDigiKey,
			Enabled:         stored.DigiKeyEnabled,
			Missing:         digiKeyMissing,
			SecretSource:    digiKeySecretSource,
			Source:          digiKeySource,
			SecureAvailable: secureAvailable,
			SecretStored:    digiKeySecretStored,
			HasSecret:       digiKeySecret != "",
		}),
	}
	preferences.Mouser = MouserSettings{
		Enabled:      stored.MouserEnabled,
		APIKeyStored: mouserSecretStored,
		Status: providerStatus(providerStatusInput{
			Provider:        sourcing.ProviderMouser,
			Enabled:         stored.MouserEnabled,
			Missing:         mouserMissing,
			SecretSource:    mouserSecretSource,
			Source:          mouserSource,
			SecureAvailable: secureAvailable,
			SecretStored:    mouserSecretStored,
			HasSecret:       mouserAPIKey != "",
		}),
	}
	preferences.LCSC = LCSCSettings{
		Enabled:  stored.LCSCEnabled,
		Currency: lcscCurrency,
		Status: ProviderStatus{
			Provider:     sourcing.ProviderLCSC,
			Enabled:      stored.LCSCEnabled,
			Complete:     stored.LCSCEnabled,
			State:        ternary(stored.LCSCEnabled, "configured", "disabled"),
			StorageMode:  "none",
			Source:       lcscSource,
			Message:      ternary(stored.LCSCEnabled, "Configured. LCSC currently does not require a saved secret.", "Disabled in Preferences."),
			HasSecret:    false,
			SecretStored: false,
		},
	}

	config := sourcing.Config{
		DigiKey: sourcing.DigiKeyConfig{
			Enabled:      stored.DigiKeyEnabled,
			ClientID:     digiKeyClientID,
			ClientSecret: digiKeySecret,
			CustomerID:   digiKeyCustomerID,
			Site:         digiKeySite,
			Language:     digiKeyLanguage,
			Currency:     digiKeyCurrency,
		},
		Mouser: sourcing.MouserConfig{
			Enabled: stored.MouserEnabled,
			APIKey:  mouserAPIKey,
		},
		LCSC: sourcing.LCSCConfig{
			Enabled:  stored.LCSCEnabled,
			Currency: lcscCurrency,
		},
	}

	return ResolvedConfig{Config: config, Preferences: preferences, FromAppPrefs: hasAnyStoredValue(stored)}, nil
}

func (s *Manager) validateSecretWrites(input UpdateInput) error {
	if s.secrets == nil {
		return fmt.Errorf("secure storage not configured")
	}
	if input.DigiKey.ReplaceClientSecret != nil && strings.TrimSpace(*input.DigiKey.ReplaceClientSecret) == "" {
		return fmt.Errorf("digikey client secret cannot be empty when replacing")
	}
	if input.Mouser.ReplaceAPIKey != nil && strings.TrimSpace(*input.Mouser.ReplaceAPIKey) == "" {
		return fmt.Errorf("mouser API key cannot be empty when replacing")
	}
	if (input.DigiKey.ReplaceClientSecret != nil || input.Mouser.ReplaceAPIKey != nil) && !s.secrets.Available() {
		return fmt.Errorf("secure storage unavailable; Trace will not save supplier secrets outside the system credential store")
	}
	return nil
}

func (s *Manager) loadStored(ctx context.Context) (storedPreferences, error) {
	values, err := s.prefs.List(ctx, preferencePrefix)
	if err != nil {
		return storedPreferences{}, err
	}

	lcscEnabled := true
	if raw := strings.TrimSpace(s.env("LCSC_ENABLED")); raw != "" {
		if parsed, err := strconv.ParseBool(raw); err == nil {
			lcscEnabled = parsed
		}
	}

	return storedPreferences{
		DigiKeyEnabled:    boolValue(values, prefDigiKeyEnabled, true),
		DigiKeyClientID:   strings.TrimSpace(values[prefDigiKeyClientID]),
		DigiKeyCustomerID: strings.TrimSpace(values[prefDigiKeyCustomerID]),
		DigiKeySite:       strings.TrimSpace(values[prefDigiKeySite]),
		DigiKeyLanguage:   strings.TrimSpace(values[prefDigiKeyLanguage]),
		DigiKeyCurrency:   strings.TrimSpace(values[prefDigiKeyCurrency]),
		MouserEnabled:     boolValue(values, prefMouserEnabled, true),
		LCSCEnabled:       boolValue(values, prefLCSCEnabled, lcscEnabled),
		LCSCCurrency:      strings.TrimSpace(values[prefLCSCCurrency]),
	}, nil
}

func (s *Manager) resolveSecret(secretKey, envKey string) (string, string, bool, error) {
	if s.secrets != nil && s.secrets.Available() {
		value, err := s.secrets.Get(secretKey)
		switch {
		case err == nil && strings.TrimSpace(value) != "":
			return strings.TrimSpace(value), "preferences", true, nil
		case err == nil:
			return strings.TrimSpace(s.env(envKey)), envOrMissing(s.env(envKey)), false, nil
		case errors.Is(err, secretstore.ErrNotFound):
			return strings.TrimSpace(s.env(envKey)), envOrMissing(s.env(envKey)), false, nil
		case errors.Is(err, secretstore.ErrUnavailable):
			fallthrough
		default:
			if envValue := strings.TrimSpace(s.env(envKey)); envValue != "" {
				return envValue, "environment", false, nil
			}
			return "", "unavailable", false, nil
		}
	}

	if envValue := strings.TrimSpace(s.env(envKey)); envValue != "" {
		return envValue, "environment", false, nil
	}
	return "", "unavailable", false, nil
}

func (s *Manager) resolvePreferenceValue(prefValue, secretKey, envKey string) (string, string, bool, error) {
	value, source, stored, err := s.resolveSecret(secretKey, envKey)
	if err != nil {
		return "", "", false, err
	}
	if source == "preferences" {
		return value, source, stored, nil
	}
	if trimmed := strings.TrimSpace(prefValue); trimmed != "" {
		return trimmed, "preferences", false, nil
	}
	return value, source, stored, nil
}

func secretKeyFor(provider, secret string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "digikey":
		if secret == "client_secret" {
			return secretDigiKeyClientSecret, nil
		}
		if secret == "client_id" {
			return secretDigiKeyClientID, nil
		}
	case "mouser":
		if secret == "api_key" {
			return secretMouserAPIKey, nil
		}
	}
	return "", fmt.Errorf("unknown supplier secret %q for provider %q", secret, provider)
}

type providerStatusInput struct {
	Provider        string
	Enabled         bool
	Missing         []string
	SecretSource    string
	Source          string
	SecureAvailable bool
	SecretStored    bool
	HasSecret       bool
}

func providerStatus(input providerStatusInput) ProviderStatus {
	status := ProviderStatus{
		Provider:     input.Provider,
		Enabled:      input.Enabled,
		Complete:     input.Enabled && len(input.Missing) == 0,
		Source:       input.Source,
		HasSecret:    input.HasSecret,
		SecretStored: input.SecretStored,
	}

	if !input.Enabled {
		status.State = "disabled"
		status.StorageMode = storageMode(input.SecretSource, input.SecureAvailable)
		status.Message = "Disabled in Preferences."
		return status
	}

	status.StorageMode = storageMode(input.SecretSource, input.SecureAvailable)
	if len(input.Missing) == 0 {
		status.State = "configured"
		switch input.SecretSource {
		case "preferences":
			status.Message = fmt.Sprintf("Configured. %s credentials are stored in the system credential store.", input.Provider)
		case "environment":
			status.Message = fmt.Sprintf("Configured using environment variables for %s credentials.", input.Provider)
		default:
			status.Message = "Configured."
		}
		return status
	}

	status.State = "incomplete"
	if !input.SecureAvailable && input.SecretSource == "unavailable" {
		status.Message = fmt.Sprintf("Secure storage is unavailable. Missing %s. Use environment variables until a system credential store is available.", strings.Join(input.Missing, ", "))
		return status
	}
	status.Message = fmt.Sprintf("Missing %s.", strings.Join(input.Missing, ", "))
	return status
}

func storageMode(secretSource string, secureAvailable bool) string {
	switch secretSource {
	case "preferences":
		return "keychain"
	case "environment":
		return "environment"
	case "unavailable":
		if !secureAvailable {
			return "unavailable"
		}
	}
	return "missing"
}

func missingFields(values ...string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" {
			result = append(result, value)
		}
	}
	return result
}

func requiredField(label, value string) string {
	if strings.TrimSpace(value) == "" {
		return label
	}
	return ""
}

func pickValue(preferenceValue, envValue string) (string, string) {
	if trimmed := strings.TrimSpace(preferenceValue); trimmed != "" {
		return trimmed, "preferences"
	}
	if trimmed := strings.TrimSpace(envValue); trimmed != "" {
		return trimmed, "environment"
	}
	return "", "missing"
}

func mergeSources(primary, secondary string) string {
	if primary == secondary {
		return primary
	}
	if primary == "missing" {
		return secondary
	}
	if secondary == "missing" {
		return primary
	}
	if primary == "preferences" && secondary == "preferences" {
		return "preferences"
	}
	if primary == "environment" && secondary == "environment" {
		return "environment"
	}
	return "mixed"
}

func envOrMissing(value string) string {
	if strings.TrimSpace(value) != "" {
		return "environment"
	}
	return "missing"
}

func boolValue(values map[string]string, key string, fallback bool) bool {
	raw, ok := values[key]
	if !ok {
		return fallback
	}
	parsed, err := strconv.ParseBool(strings.TrimSpace(raw))
	if err != nil {
		return fallback
	}
	return parsed
}

func hasAnyStoredValue(stored storedPreferences) bool {
	return stored.DigiKeyClientID != "" || stored.DigiKeyCustomerID != "" || stored.DigiKeySite != "" || stored.DigiKeyLanguage != "" || stored.DigiKeyCurrency != "" || stored.LCSCCurrency != ""
}

func ternary[T any](condition bool, whenTrue, whenFalse T) T {
	if condition {
		return whenTrue
	}
	return whenFalse
}
