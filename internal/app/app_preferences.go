package app

import (
	"context"

	"trace/internal/kicadconfig"
	"trace/internal/supplierconfig"
)

func (a *App) GetKiCadPreferences() (KiCadPreferencesResponse, error) {
	if err := a.checkReady(); err != nil {
		return KiCadPreferencesResponse{}, err
	}
	prefs, err := a.svc.GetKiCadPreferences(context.Background())
	if err != nil {
		return KiCadPreferencesResponse{}, err
	}
	return kiCadPreferencesToResponse(prefs), nil
}

func (a *App) SaveKiCadPreferences(input SaveKiCadPreferencesInput) (KiCadPreferencesResponse, error) {
	if err := a.checkReady(); err != nil {
		return KiCadPreferencesResponse{}, err
	}
	prefs, err := a.svc.SaveKiCadPreferences(context.Background(), kicadconfig.UpdateInput{
		ProjectRoots: input.ProjectRoots,
	})
	if err != nil {
		return KiCadPreferencesResponse{}, err
	}
	return kiCadPreferencesToResponse(prefs), nil
}

func (a *App) GetSupplierPreferences() (SupplierPreferencesResponse, error) {
	if err := a.checkReady(); err != nil {
		return SupplierPreferencesResponse{}, err
	}
	prefs, err := a.svc.GetSupplierPreferences(context.Background())
	if err != nil {
		return SupplierPreferencesResponse{}, err
	}
	return supplierPreferencesToResponse(prefs), nil
}

func (a *App) SaveSupplierPreferences(input SaveSupplierPreferencesInput) (SupplierPreferencesResponse, error) {
	if err := a.checkReady(); err != nil {
		return SupplierPreferencesResponse{}, err
	}
	prefs, err := a.svc.SaveSupplierPreferences(context.Background(), supplierconfig.UpdateInput{
		DigiKey: supplierconfig.DigiKeyInput{
			Enabled:             input.DigiKey.Enabled,
			ClientID:            input.DigiKey.ClientID,
			CustomerID:          input.DigiKey.CustomerID,
			Site:                input.DigiKey.Site,
			Language:            input.DigiKey.Language,
			Currency:            input.DigiKey.Currency,
			ReplaceClientSecret: input.DigiKey.ReplaceClientSecret,
		},
		Mouser: supplierconfig.MouserInput{
			Enabled:       input.Mouser.Enabled,
			ReplaceAPIKey: input.Mouser.ReplaceAPIKey,
		},
		LCSC: supplierconfig.LCSCInput{
			Enabled:  input.LCSC.Enabled,
			Currency: input.LCSC.Currency,
		},
	})
	if err != nil {
		return SupplierPreferencesResponse{}, err
	}
	return supplierPreferencesToResponse(prefs), nil
}

func (a *App) ClearSupplierSecret(provider, secret string) (SupplierPreferencesResponse, error) {
	if err := a.checkReady(); err != nil {
		return SupplierPreferencesResponse{}, err
	}
	prefs, err := a.svc.ClearSupplierSecret(context.Background(), provider, secret)
	if err != nil {
		return SupplierPreferencesResponse{}, err
	}
	return supplierPreferencesToResponse(prefs), nil
}

func kiCadPreferencesToResponse(prefs kicadconfig.Preferences) KiCadPreferencesResponse {
	return KiCadPreferencesResponse{
		ProjectRoots: append([]string{}, prefs.ProjectRoots...),
	}
}

func supplierPreferencesToResponse(prefs supplierconfig.Preferences) SupplierPreferencesResponse {
	return SupplierPreferencesResponse{
		SecureStorageAvailable: prefs.SecureStorageAvailable,
		SecureStorageMessage:   prefs.SecureStorageMessage,
		DigiKey: SupplierDigiKeyResponse{
			Enabled:            prefs.DigiKey.Enabled,
			ClientID:           prefs.DigiKey.ClientID,
			CustomerID:         prefs.DigiKey.CustomerID,
			Site:               prefs.DigiKey.Site,
			Language:           prefs.DigiKey.Language,
			Currency:           prefs.DigiKey.Currency,
			ClientSecretStored: prefs.DigiKey.ClientSecretStored,
			Status:             supplierProviderStatusToResponse(prefs.DigiKey.Status),
		},
		Mouser: SupplierMouserResponse{
			Enabled:      prefs.Mouser.Enabled,
			APIKeyStored: prefs.Mouser.APIKeyStored,
			Status:       supplierProviderStatusToResponse(prefs.Mouser.Status),
		},
		LCSC: SupplierLCSCResponse{
			Enabled:  prefs.LCSC.Enabled,
			Currency: prefs.LCSC.Currency,
			Status:   supplierProviderStatusToResponse(prefs.LCSC.Status),
		},
	}
}

func supplierProviderStatusToResponse(status supplierconfig.ProviderStatus) SupplierProviderConfigResponse {
	return SupplierProviderConfigResponse{
		Provider:     status.Provider,
		Enabled:      status.Enabled,
		Complete:     status.Complete,
		State:        status.State,
		StorageMode:  status.StorageMode,
		Source:       status.Source,
		Message:      status.Message,
		HasSecret:    status.HasSecret,
		SecretStored: status.SecretStored,
	}
}
