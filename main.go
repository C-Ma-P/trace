package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/wailsapp/wails/v3/pkg/application"

	"trace/internal/activity"
	"trace/internal/app"
	"trace/internal/assetsearch"
	"trace/internal/assetsearch/providers"
	"trace/internal/domain/registry"
	"trace/internal/ingest"
	"trace/internal/kicad"
	"trace/internal/kicadconfig"
	"trace/internal/paths"
	"trace/internal/phoneintake"
	easyedaprovider "trace/internal/providers/easyeda"
	"trace/internal/secretstore"
	"trace/internal/service"
	"trace/internal/store/postgres"
	"trace/internal/supplierconfig"
	"trace/internal/windows"
)

var startupTime = time.Now()

func startupLog(msg string) {
	log.Printf("[startup +%dms] %s", time.Since(startupTime).Milliseconds(), msg)
}

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var appIcon []byte

func main() {
	startupLog("startup")

	dsn := "postgres://meet:changeme@localhost:5432/trace?sslmode=disable"
	if d := os.Getenv("DATABASE_URL"); d != "" {
		dsn = d
	}

	var backendApp *app.App
	activityHub := activity.NewHub(100)
	svc, assetSearchSvc, ingestSvc, easyedaSvc, db, initErr := initService(dsn, activityHub)
	if initErr != nil {
		log.Printf("[startup] init failed: %v", initErr)
		backendApp = app.NewFailed(initErr.Error())
	} else {
		defer db.Close()
		backendApp = app.New(svc, assetSearchSvc, ingestSvc, easyedaSvc)
		backendApp.SetActivityHub(activityHub)

		store := postgres.New(db)
		compRepo := postgres.NewComponentRepository(store)
		bagRepo := postgres.NewBagRepository(store)
		intakePort := 8741
		if p := os.Getenv("TRACE_INTAKE_PORT"); p != "" {
			if parsed, err := strconv.Atoi(p); err == nil {
				intakePort = parsed
			}
		}
		pkiDir, err := paths.EnsurePhoneIntakePKIDir()
		if err != nil {
			log.Printf("[startup] phone-intake PKI dir: %v — phone intake disabled", err)
			pkiDir = ""
		}
		intakeServer := phoneintake.NewServer(svc, compRepo, bagRepo, intakePort, activityHub, pkiDir)
		backendApp.SetIntakeServer(intakeServer)
		backendApp.SetBagRepo(bagRepo)
		defer intakeServer.Stop()
	}
	startupLog("app constructed")

	appSvc := &AppService{App: backendApp}
	appInstance := application.New(application.Options{
		Name: "Trace",
		Icon: appIcon,
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Services: []application.Service{
			application.NewService(appSvc),
		},
	})

	controller := windows.NewController(appInstance, backendApp)
	appInstance.RegisterService(application.NewService(&WindowService{controller: controller}))

	startupLog("before first window creation")
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

func (w *WindowService) PickAssetFile() (string, error) {
	if w.controller == nil {
		return "", fmt.Errorf("window controller not available")
	}
	return w.controller.PickFile("Import Component Asset",
		application.FileFilter{
			DisplayName: "KiCad Assets (*.kicad_sym, *.kicad_mod, *.zip)",
			Pattern:     "*.kicad_sym;*.kicad_mod;*.zip",
		},
		application.FileFilter{
			DisplayName: "3D Models (*.step, *.stp, *.wrl)",
			Pattern:     "*.step;*.stp;*.wrl",
		},
		application.FileFilter{
			DisplayName: "Datasheets (*.pdf)",
			Pattern:     "*.pdf",
		},
		application.FileFilter{
			DisplayName: "All Files (*)",
			Pattern:     "*",
		},
	)
}

func (w *WindowService) PickAssetDir() (string, error) {
	if w.controller == nil {
		return "", fmt.Errorf("window controller not available")
	}
	return w.controller.PickDirectory("")
}

func (w *WindowService) SetLauncherView(view string) error {
	if w.controller == nil {
		return fmt.Errorf("window controller not available")
	}
	return w.controller.SetLauncherView(view)
}

func initService(dsn string, emitter activity.Emitter) (*service.Service, *assetsearch.Service, *ingest.Service, *easyedaprovider.Service, *sqlx.DB, error) {
	ctx := context.Background()

	startupLog("before DB connect")
	db, err := sqlx.Connect("pgx", dsn)
	if err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("cannot connect to PostgreSQL (%s): %w", dsn, err)
	}
	startupLog("after DB connect")

	store := postgres.New(db)
	startupLog("before migrations")
	if err := store.Migrate(ctx); err != nil {
		db.Close()
		return nil, nil, nil, nil, nil, fmt.Errorf("database migration failed: %w", err)
	}
	startupLog("after migrations")

	compRepo := postgres.NewComponentRepository(store)
	projRepo := postgres.NewProjectRepository(store)
	assetRepo := postgres.NewComponentAssetRepository(store)
	prefRepo := postgres.NewPreferenceRepository(store)
	kicadSvc := kicad.New(nil)
	kicadCfg := kicadconfig.NewManager(prefRepo)
	secretSvc := secretstore.NewKeyringStore("trace")
	supplierCfg := supplierconfig.NewManager(prefRepo, secretSvc, os.Getenv, emitter)
	svc := service.New(compRepo, projRepo, assetRepo, kicadSvc).
		SetKiCadConfig(kicadCfg).
		SetSupplierConfig(supplierCfg)

	// Versioned canonical attribute sync: only re-sync when the registry
	// version constant changes, saving the cost on every normal startup.
	wantVersion := strconv.Itoa(registry.CanonicalRegistryVersion)
	startupLog("before canonical attribute version check (want v" + wantVersion + ")")
	prefs, err := prefRepo.List(ctx, "system.canonical_registry_version")
	if err != nil {
		db.Close()
		return nil, nil, nil, nil, nil, fmt.Errorf("reading canonical registry version failed: %w", err)
	}
	storedVersion := prefs["system.canonical_registry_version"]

	if storedVersion == wantVersion {
		startupLog("canonical attribute sync skipped (version matched: v" + wantVersion + ")")
	} else {
		if storedVersion == "" {
			startupLog("canonical attribute sync: no stored version, running sync")
		} else {
			startupLog("canonical attribute sync: version changed v" + storedVersion + " → v" + wantVersion + ", running sync")
		}
		if err := svc.SyncCanonicalAttributeDefinitions(ctx); err != nil {
			db.Close()
			return nil, nil, nil, nil, nil, fmt.Errorf("attribute definition sync failed: %w", err)
		}
		if err := prefRepo.SetMany(ctx, map[string]string{
			"system.canonical_registry_version": wantVersion,
		}); err != nil {
			db.Close()
			return nil, nil, nil, nil, nil, fmt.Errorf("storing canonical registry version failed: %w", err)
		}
		startupLog("canonical attribute sync complete (stored v" + wantVersion + ")")
	}

	reg := assetsearch.NewRegistry()
	reg.Register(&providers.SnapEDA{})
	reg.Register(&providers.UltraLibrarian{})

	assetsDir, err := paths.EnsureAssetsDir()
	if err != nil {
		db.Close()
		return nil, nil, nil, nil, nil, fmt.Errorf("ensure assets directory: %w", err)
	}
	ingestSvc := ingest.NewService(assetsDir, compRepo, assetRepo)

	assetSearchSvc := assetsearch.NewService(reg, compRepo, assetRepo, ingestSvc)

	easyedaSvc := easyedaprovider.NewService(ingestSvc)

	startupLog("service construction complete")
	return svc, assetSearchSvc, ingestSvc, easyedaSvc, db, nil
}
