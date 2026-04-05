package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/wailsapp/wails/v3/pkg/application"

	"componentmanager/internal/app"
	"componentmanager/internal/assetsearch"
	"componentmanager/internal/assetsearch/providers"
	"componentmanager/internal/kicad"
	"componentmanager/internal/kicadconfig"
	"componentmanager/internal/secretstore"
	"componentmanager/internal/service"
	"componentmanager/internal/store/postgres"
	"componentmanager/internal/supplierconfig"
	"componentmanager/internal/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	dsn := "postgres://meet:changeme@localhost:5432/componentmanager?sslmode=disable"
	if d := os.Getenv("DATABASE_URL"); d != "" {
		dsn = d
	}

	var backendApp *app.App
	svc, assetSearchSvc, db, initErr := initService(dsn)
	if initErr != nil {
		backendApp = app.NewFailed(initErr.Error())
	} else {
		defer db.Close()
		backendApp = app.New(svc, assetSearchSvc)
	}

	appSvc := &AppService{App: backendApp}
	appInstance := application.New(application.Options{
		Name: "Trace",
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Services: []application.Service{
			application.NewService(appSvc),
		},
	})

	controller := windows.NewController(appInstance, backendApp)
	appInstance.RegisterService(application.NewService(&WindowService{controller: controller}))

	controller.EnsureLauncherWindow()

	if err := appInstance.Run(); err != nil {
		log.Fatal(err)
	}
}

type AppService struct {
	*app.App
}

type WindowService struct {
	controller *windows.Controller
}

func (w *WindowService) OpenProjectWindow(projectID string) error {
	if w.controller == nil {
		return fmt.Errorf("window controller not available")
	}
	return w.controller.OpenProjectWindow(projectID, true)
}

func (w *WindowService) OpenProjectWindowKeepLauncher(projectID string) error {
	if w.controller == nil {
		return fmt.Errorf("window controller not available")
	}
	return w.controller.OpenProjectWindow(projectID, false)
}

func (w *WindowService) ListOpenProjectIDs() []string {
	if w.controller == nil {
		return []string{}
	}
	return w.controller.ListOpenProjectIDs()
}

func (w *WindowService) OpenPreferencesWindow(projectID string) {
	if w.controller == nil {
		return
	}
	w.controller.OpenPreferencesWindow(projectID)
}

func (w *WindowService) PickDirectory(startDir string) (string, error) {
	if w.controller == nil {
		return "", fmt.Errorf("window controller not available")
	}
	return w.controller.PickDirectory(startDir)
}

func (w *WindowService) SetLauncherView(view string) error {
	if w.controller == nil {
		return fmt.Errorf("window controller not available")
	}
	return w.controller.SetLauncherView(view)
}

func initService(dsn string) (*service.Service, *assetsearch.Service, *sqlx.DB, error) {
	db, err := sqlx.Connect("pgx", dsn)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("cannot connect to PostgreSQL (%s): %w", dsn, err)
	}

	store := postgres.New(db)
	if err := store.Migrate(context.Background()); err != nil {
		db.Close()
		return nil, nil, nil, fmt.Errorf("database migration failed: %w", err)
	}

	compRepo := postgres.NewComponentRepository(store)
	projRepo := postgres.NewProjectRepository(store)
	assetRepo := postgres.NewComponentAssetRepository(store)
	prefRepo := postgres.NewPreferenceRepository(store)
	kicadSvc := kicad.New(nil)
	kicadCfg := kicadconfig.NewManager(prefRepo)
	secretSvc := secretstore.NewKeyringStore("trace")
	supplierCfg := supplierconfig.NewManager(prefRepo, secretSvc, os.Getenv)
	svc := service.New(compRepo, projRepo, assetRepo, kicadSvc).
		SetKiCadConfig(kicadCfg).
		SetSupplierConfig(supplierCfg)

	if err := svc.SyncCanonicalAttributeDefinitions(context.Background()); err != nil {
		db.Close()
		return nil, nil, nil, fmt.Errorf("attribute definition sync failed: %w", err)
	}

	reg := assetsearch.NewRegistry()
	reg.Register(&providers.SnapEDA{})
	reg.Register(&providers.UltraLibrarian{})
	assetSearchSvc := assetsearch.NewService(reg, compRepo, assetRepo)

	return svc, assetSearchSvc, db, nil
}
