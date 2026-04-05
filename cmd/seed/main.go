// cmd/seed populates the database with realistic hobbyist components for UI development.
// Run with: go run cmd/seed/main.go
// Set DATABASE_URL env var if your DSN differs from the default.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"

	"componentmanager/internal/domain"
	"componentmanager/internal/domain/registry"
	"componentmanager/internal/launcher"
	"componentmanager/internal/paths"
	"componentmanager/internal/service"
	"componentmanager/internal/store/postgres"
)

func main() {
	wipe := flag.Bool("wipe", false, "wipe DB + local Trace project state before seeding")
	flag.Parse()

	dsn := "postgres://meet:changeme@localhost:5432/componentmanager?sslmode=disable"
	if d := os.Getenv("DATABASE_URL"); d != "" {
		dsn = d
	}

	db, err := sqlx.Connect("pgx", dsn)
	if err != nil {
		log.Fatalf("cannot connect to database: %v", err)
	}
	defer db.Close()

	store := postgres.New(db)
	ctx := context.Background()

	if err := store.Migrate(ctx); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	if *wipe {
		if err := wipeDatabase(ctx, db); err != nil {
			log.Fatalf("wipe database: %v", err)
		}
		if err := wipeTraceProjectState(); err != nil {
			log.Fatalf("wipe local project state: %v", err)
		}
		log.Printf("wiped DB + local Trace project state")
	}

	compRepo := postgres.NewComponentRepository(store)
	projRepo := postgres.NewProjectRepository(store)
	assetRepo := postgres.NewComponentAssetRepository(store)
	svc := service.New(compRepo, projRepo, assetRepo)

	if err := svc.SyncCanonicalAttributeDefinitions(ctx); err != nil {
		log.Fatalf("sync attribute definitions: %v", err)
	}

	compCreated := 0
	for _, c := range seedComponents() {
		if _, err := svc.CreateComponent(ctx, c); err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
				log.Printf("  ~ duplicate   %-22s  %s  %s (skipped)", c.Category, c.Manufacturer, c.MPN)
				continue
			}
			log.Printf("SKIP component %s %s: %v", c.Manufacturer, c.MPN, err)
			continue
		}
		log.Printf("  + component  %-22s  %s  %s", c.Category, c.Manufacturer, c.MPN)
		compCreated++
	}
	log.Printf("created %d components", compCreated)

	launcherStore := launcher.NewStore()

	projCreated := 0
	for _, p := range seedProjects() {
		created, err := svc.CreateProject(ctx, p)
		if err != nil {
			log.Printf("SKIP project %q: %v", p.Name, err)
			continue
		}
		if err := ensureProjectOnDisk(created); err != nil {
			log.Fatalf("write project to disk %q: %v", created.Name, err)
		}
		_ = launcherStore.TouchProject(created.ID, created.Name, created.Description)
		log.Printf("  * project  %q  (%d requirements)", created.Name, len(created.Requirements))
		projCreated++
	}
	log.Printf("created %d projects", projCreated)
}

func wipeDatabase(ctx context.Context, db *sqlx.DB) error {
	// TRUNCATE does not support IF EXISTS, so we defensively ignore undefined_table
	// to keep this compatible across schema versions/migrations.
	_, err := db.ExecContext(ctx, `
		do $$
		begin
			begin execute 'truncate table requirement_constraints cascade'; exception when undefined_table then end;
			begin execute 'truncate table project_requirements cascade'; exception when undefined_table then end;
			begin execute 'truncate table projects cascade'; exception when undefined_table then end;
			begin execute 'truncate table component_assets cascade'; exception when undefined_table then end;
			begin execute 'truncate table component_attributes cascade'; exception when undefined_table then end;
			begin execute 'truncate table components cascade'; exception when undefined_table then end;
			begin execute 'truncate table attribute_definitions cascade'; exception when undefined_table then end;
			begin execute 'truncate table inventory_lots cascade'; exception when undefined_table then end;
		end $$;
	`)
	return err
}

func wipeTraceProjectState() error {
	traceHome, err := paths.TraceHomeDir()
	if err != nil {
		return err
	}
	traceHome = filepath.Clean(traceHome)
	if traceHome == "." || traceHome == string(filepath.Separator) {
		return fmt.Errorf("refusing to wipe trace home %q", traceHome)
	}

	projectsDir, err := paths.ProjectsDir()
	if err != nil {
		return err
	}
	projectsDir = filepath.Clean(projectsDir)
	if !strings.HasPrefix(projectsDir+string(filepath.Separator), traceHome+string(filepath.Separator)) {
		return fmt.Errorf("refusing to wipe projects dir outside trace home: %q", projectsDir)
	}
	if err := os.RemoveAll(projectsDir); err != nil {
		return fmt.Errorf("remove projects dir: %w", err)
	}

	launcherPath, err := paths.LauncherStatePath()
	if err != nil {
		return err
	}
	launcherPath = filepath.Clean(launcherPath)
	if !strings.HasPrefix(launcherPath, traceHome+string(filepath.Separator)) {
		return fmt.Errorf("refusing to wipe launcher state outside trace home: %q", launcherPath)
	}
	if err := os.Remove(launcherPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove launcher state: %w", err)
	}

	return nil
}

func ensureProjectOnDisk(project domain.Project) error {
	projectsDir, err := paths.EnsureProjectsDir()
	if err != nil {
		return err
	}
	projectDir := filepath.Join(projectsDir, project.ID)
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		return fmt.Errorf("create project dir: %w", err)
	}
	metadataPath := filepath.Join(projectDir, "project.json")
	metadataBytes, err := json.Marshal(struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		CreatedAt   string `json:"createdAt"`
	}{
		ID:          project.ID,
		Name:        project.Name,
		Description: project.Description,
		CreatedAt:   project.CreatedAt.UTC().Format(time.RFC3339),
	})
	if err != nil {
		return err
	}
	if err := os.WriteFile(metadataPath, metadataBytes, 0o644); err != nil {
		return fmt.Errorf("write project metadata: %w", err)
	}
	return nil
}

// helpers for attribute value construction

func numAttr(key string, v float64, unit string) domain.AttributeValue {
	return domain.AttributeValue{Key: key, ValueType: domain.ValueTypeNumber, Number: &v, Unit: unit}
}

func textAttr(key, v string) domain.AttributeValue {
	return domain.AttributeValue{Key: key, ValueType: domain.ValueTypeText, Text: &v}
}

func qty(n int, mode domain.QuantityMode) (*int, domain.QuantityMode) {
	return &n, mode
}

func resistor(mpn, manufacturer, pkg, rtype string, ohms, tolerance, powerW float64, q int, location string) domain.Component {
	n, mode := qty(q, domain.QuantityModeExact)
	return domain.Component{
		Category:     domain.CategoryResistor,
		MPN:          mpn,
		Manufacturer: manufacturer,
		Package:      pkg,
		Description:  rtype + " resistor",
		Quantity:     n,
		QuantityMode: mode,
		Location:     location,
		Attributes: []domain.AttributeValue{
			numAttr(registry.AttrResistanceOhms, ohms, "ohm"),
			numAttr(registry.AttrTolerancePercent, tolerance, "percent"),
			numAttr(registry.AttrPowerW, powerW, "W"),
			textAttr(registry.AttrPackage, pkg),
			textAttr(registry.AttrResistorType, rtype),
		},
	}
}

func capacitor(mpn, manufacturer, pkg, ctype, dielectric string, farads, tolerance, voltage float64, q int, location string) domain.Component {
	n, mode := qty(q, domain.QuantityModeExact)
	return domain.Component{
		Category:     domain.CategoryCapacitor,
		MPN:          mpn,
		Manufacturer: manufacturer,
		Package:      pkg,
		Description:  ctype + " capacitor",
		Quantity:     n,
		QuantityMode: mode,
		Location:     location,
		Attributes: []domain.AttributeValue{
			numAttr(registry.AttrCapacitanceF, farads, "F"),
			numAttr(registry.AttrTolerancePercent, tolerance, "percent"),
			numAttr(registry.AttrVoltageV, voltage, "V"),
			textAttr(registry.AttrPackage, pkg),
			textAttr(registry.AttrDielectric, dielectric),
			textAttr(registry.AttrCapacitorType, ctype),
		},
	}
}

func inductor(mpn, manufacturer, pkg, itype string, henries, tolerancePct, currentA, dcrOhm float64, q int, location string) domain.Component {
	n, mode := qty(q, domain.QuantityModeExact)
	return domain.Component{
		Category:     domain.CategoryInductor,
		MPN:          mpn,
		Manufacturer: manufacturer,
		Package:      pkg,
		Description:  itype + " inductor",
		Quantity:     n,
		QuantityMode: mode,
		Location:     location,
		Attributes: []domain.AttributeValue{
			numAttr(registry.AttrInductanceH, henries, "H"),
			numAttr(registry.AttrTolerancePercent, tolerancePct, "percent"),
			numAttr(registry.AttrCurrentA, currentA, "A"),
			numAttr(registry.AttrDCROhms, dcrOhm, "ohm"),
			textAttr(registry.AttrPackage, pkg),
			textAttr(registry.AttrInductorType, itype),
		},
	}
}

func ic(mpn, manufacturer, pkg, description string, q int, location string) domain.Component {
	n, mode := qty(q, domain.QuantityModeExact)
	return domain.Component{
		Category:     domain.CategoryIntegratedCircuit,
		MPN:          mpn,
		Manufacturer: manufacturer,
		Package:      pkg,
		Description:  description,
		Quantity:     n,
		QuantityMode: mode,
		Location:     location,
	}
}

// ── Project constraint helpers ────────────────────────────────────────────────

func numReq(key string, op domain.Operator, v float64, unit string) domain.RequirementConstraint {
	return domain.RequirementConstraint{Key: key, ValueType: domain.ValueTypeNumber, Operator: op, Number: &v, Unit: unit}
}

func textReq(key, v string) domain.RequirementConstraint {
	return domain.RequirementConstraint{Key: key, ValueType: domain.ValueTypeText, Operator: domain.OperatorEqual, Text: &v}
}

func req(name string, cat domain.Category, qty int, constraints ...domain.RequirementConstraint) domain.ProjectRequirement {
	return domain.ProjectRequirement{Name: name, Category: cat, Quantity: qty, Constraints: constraints}
}

func project(name, description string, reqs ...domain.ProjectRequirement) domain.Project {
	return domain.Project{Name: name, Description: description, Requirements: reqs}
}

// ── Seed projects ─────────────────────────────────────────────────────────────

func seedProjects() []domain.Project {
	R := domain.CategoryResistor
	C := domain.CategoryCapacitor
	L := domain.CategoryInductor
	IC := domain.CategoryIntegratedCircuit
	eq := domain.OperatorEqual
	gte := domain.OperatorGTE
	lte := domain.OperatorLTE //nolint:unused

	_ = lte // used below

	return []domain.Project{

		// ── 1. NE555 Astable LED Blinker (~1 Hz) ─────────────────────────────
		// Classic astable: f ≈ 1/(0.693 · (R_A + 2·R_B) · C)
		// R_A=47k, R_B=4.7k, C=10µF → ~1 Hz
		project(
			"NE555 Astable LED Blinker",
			"1 Hz blink rate using NE555 in astable mode. R_A=47kΩ, R_B=4.7kΩ, C=10µF.",
			req("Timer IC", IC, 1),
			req("Timing R_A", R, 1, numReq(registry.AttrResistanceOhms, eq, 47_000, "ohm"), textReq(registry.AttrPackage, "0402")),
			req("Timing R_B", R, 1, numReq(registry.AttrResistanceOhms, eq, 4_700, "ohm"), textReq(registry.AttrPackage, "0402")),
			req("Timing capacitor", C, 1, numReq(registry.AttrCapacitanceF, gte, 10e-6, "F")),
			req("Power decoupling", C, 1, numReq(registry.AttrCapacitanceF, eq, 100e-9, "F"), textReq(registry.AttrCapacitorType, "MLCC")),
			req("LED current limiter", R, 1, numReq(registry.AttrResistanceOhms, eq, 470, "ohm")),
		),

		// ── 2. LM7805 5V Linear Regulator Module ─────────────────────────────
		// Datasheet-recommended decoupling: 0.33µF input, 0.1µF output.
		// Using larger electrolytic bulk caps for better transient response.
		project(
			"LM7805 5V Linear Regulator Module",
			"5V regulated supply from 9V input. Includes bulk electrolytics and ceramic decoupling per LM7805 datasheet.",
			req("Voltage regulator", IC, 1),
			req("Input bulk cap", C, 1,
				numReq(registry.AttrCapacitanceF, gte, 100e-6, "F"),
				numReq(registry.AttrVoltageV, gte, 50, "V"),
				textReq(registry.AttrCapacitorType, "electrolytic"),
			),
			req("Output bulk cap", C, 1,
				numReq(registry.AttrCapacitanceF, gte, 10e-6, "F"),
				textReq(registry.AttrCapacitorType, "electrolytic"),
			),
			req("Input decoupling", C, 1,
				numReq(registry.AttrCapacitanceF, gte, 100e-9, "F"),
				textReq(registry.AttrCapacitorType, "MLCC"),
			),
			req("Output decoupling", C, 1,
				numReq(registry.AttrCapacitanceF, gte, 100e-9, "F"),
				textReq(registry.AttrCapacitorType, "MLCC"),
			),
		),

		// ── 3. ATmega328P Minimal Breadboard Circuit ──────────────────────────
		// Follows Atmel AVR hardware design guidelines:
		// 100nF on each VCC/AVCC pin, 1µF bulk, 10k reset pull-up, 4.7k I2C pull-ups.
		project(
			"ATmega328P Minimal Breadboard Circuit",
			"Bare-minimum circuit to run an ATmega328P: decoupling caps, reset pull-up, and I2C pull-ups.",
			req("MCU", IC, 1),
			req("VCC decoupling ×4", C, 4,
				numReq(registry.AttrCapacitanceF, eq, 100e-9, "F"),
				textReq(registry.AttrPackage, "0402"),
			),
			req("AVCC bulk cap", C, 1,
				numReq(registry.AttrCapacitanceF, gte, 1e-6, "F"),
				textReq(registry.AttrPackage, "0402"),
			),
			req("Reset pull-up", R, 1,
				numReq(registry.AttrResistanceOhms, eq, 10_000, "ohm"),
				textReq(registry.AttrPackage, "0402"),
			),
			req("I2C SDA/SCL pull-ups", R, 2,
				numReq(registry.AttrResistanceOhms, eq, 4_700, "ohm"),
				textReq(registry.AttrPackage, "0402"),
			),
		),

		// ── 4. Passive RC Low-Pass Filter (1.6 kHz −3 dB) ────────────────────
		// f_c = 1 / (2π·R·C) = 1 / (2π·10kΩ·10nF) ≈ 1591 Hz
		project(
			"Passive RC Low-Pass Filter (1.6 kHz)",
			"First-order single-pole RC filter, −3 dB at ~1.6 kHz. Use as an anti-aliasing or audio treble-roll-off filter.",
			req("Series resistor", R, 1,
				numReq(registry.AttrResistanceOhms, eq, 10_000, "ohm"),
				textReq(registry.AttrPackage, "0402"),
			),
			req("Shunt capacitor", C, 1,
				numReq(registry.AttrCapacitanceF, eq, 10e-9, "F"),
				textReq(registry.AttrPackage, "0402"),
			),
		),

		// ── 5. TL072 Non-Inverting Amplifier (Gain = 11) ─────────────────────
		// V_out = V_in × (1 + R_f/R_g) = V_in × (1 + 10k/1k) = 11×
		project(
			"TL072 Non-Inverting Amplifier (Gain ×11)",
			"Single-supply non-inverting audio amplifier stage. Gain = 1 + R_f/R_g = 11. Includes supply bypass.",
			req("Op-amp", IC, 1),
			req("Feedback resistor R_f", R, 1,
				numReq(registry.AttrResistanceOhms, eq, 10_000, "ohm"),
			),
			req("Gain-set resistor R_g", R, 1,
				numReq(registry.AttrResistanceOhms, eq, 1_000, "ohm"),
			),
			req("Supply bypass ×2", C, 2,
				numReq(registry.AttrCapacitanceF, eq, 100e-9, "F"),
				textReq(registry.AttrCapacitorType, "MLCC"),
			),
			req("Input coupling cap", C, 1,
				numReq(registry.AttrCapacitanceF, gte, 1e-6, "F"),
			),
		),

		// ── 6. Boost Converter Output LC Filter ───────────────────────────────
		// Typical 2nd-order LC output filter for a switching converter.
		// Inductor 4.7µH ≥2A, output caps ≥47µF/35V for low ripple.
		project(
			"Boost Converter Output LC Filter",
			"Output-stage LC filter for a boost converter. Reduces switching ripple. Inductor rated ≥2 A.",
			req("Output inductor", L, 1,
				numReq(registry.AttrInductanceH, gte, 4.7e-6, "H"),
				numReq(registry.AttrCurrentA, gte, 2.0, "A"),
			),
			req("Output filter caps ×2", C, 2,
				numReq(registry.AttrCapacitanceF, gte, 47e-6, "F"),
				numReq(registry.AttrVoltageV, gte, 35, "V"),
			),
			req("Input bulk cap", C, 1,
				numReq(registry.AttrCapacitanceF, gte, 10e-6, "F"),
				numReq(registry.AttrVoltageV, gte, 50, "V"),
			),
			req("Feedback high-side R", R, 1,
				numReq(registry.AttrResistanceOhms, gte, 47_000, "ohm"),
			),
			req("Feedback low-side R", R, 1,
				numReq(registry.AttrResistanceOhms, eq, 10_000, "ohm"),
			),
		),

		// ── 7. LM358 Window Comparator ────────────────────────────────────────
		// Two comparator channels set upper/lower voltage thresholds.
		// Voltage divider from rail sets each threshold.
		project(
			"LM358 Window Comparator",
			"Dual-threshold window comparator using LM358. Triggers output when input is between V_lo and V_hi.",
			req("Dual op-amp / comparator", IC, 1),
			req("Upper threshold divider R1", R, 1,
				numReq(registry.AttrResistanceOhms, eq, 47_000, "ohm"),
			),
			req("Mid divider R2", R, 1,
				numReq(registry.AttrResistanceOhms, eq, 22_000, "ohm"),
			),
			req("Lower threshold divider R3", R, 1,
				numReq(registry.AttrResistanceOhms, eq, 10_000, "ohm"),
			),
			req("Output pull-up resistor", R, 1,
				numReq(registry.AttrResistanceOhms, eq, 4_700, "ohm"),
			),
			req("Supply bypass", C, 1,
				numReq(registry.AttrCapacitanceF, gte, 100e-9, "F"),
				textReq(registry.AttrCapacitorType, "MLCC"),
			),
		),

		// ── 8. AMS1117-3.3 LDO Regulator for 3.3V Logic ──────────────────────
		// Typical application: USB 5V → 3.3V for microcontroller/sensor boards.
		project(
			"AMS1117-3.3 LDO Regulator (5V → 3.3V)",
			"3.3V LDO supply for a microcontroller board. Converts USB 5V. Output rated 1A.",
			req("LDO regulator", IC, 1),
			req("Input filter cap", C, 1,
				numReq(registry.AttrCapacitanceF, gte, 10e-6, "F"),
				numReq(registry.AttrVoltageV, gte, 10, "V"),
				textReq(registry.AttrCapacitorType, "electrolytic"),
			),
			req("Output cap", C, 1,
				numReq(registry.AttrCapacitanceF, gte, 10e-6, "F"),
				textReq(registry.AttrCapacitorType, "electrolytic"),
			),
			req("Ceramic bypass", C, 2,
				numReq(registry.AttrCapacitanceF, eq, 100e-9, "F"),
				textReq(registry.AttrCapacitorType, "MLCC"),
			),
			req("LED status resistor", R, 1,
				numReq(registry.AttrResistanceOhms, eq, 470, "ohm"),
			),
		),

		// ── 9. 555 Monostable Delay Timer ────────────────────────────────────
		project(
			"555 Monostable Delay Timer",
			"Single-shot timer using NE555 and a timing resistor/capacitor network.",
			req("Timer IC", IC, 1),
			req("Timing resistor", R, 1,
				numReq(registry.AttrResistanceOhms, eq, 47_000, "ohm"),
				textReq(registry.AttrPackage, "0402"),
			),
			req("Timing capacitor", C, 1,
				numReq(registry.AttrCapacitanceF, eq, 100e-9, "F"),
			),
			req("Reset capacitor", C, 1,
				numReq(registry.AttrCapacitanceF, eq, 100e-9, "F"),
			),
		),

		// ── 10. 74HC595 LED Driver Board ────────────────────────────────────
		project(
			"74HC595 LED Driver Board",
			"Serial-in, parallel-out LED driver using a shift register and current-limiting resistors.",
			req("Shift register IC", IC, 1),
			req("LED current limit", R, 8,
				numReq(registry.AttrResistanceOhms, eq, 220, "ohm"),
			),
			req("Bypass capacitor", C, 1,
				numReq(registry.AttrCapacitanceF, eq, 100e-9, "F"),
				textReq(registry.AttrCapacitorType, "MLCC"),
			),
		),

		// ── 11. LM317 Adjustable 5V Supply ───────────────────────────────────
		project(
			"LM317 Adjustable 5V Supply",
			"Adjustable regulator with output set resistors and decoupling capacitors.",
			req("Adjustable regulator", IC, 1),
			req("Output set resistor", R, 1,
				numReq(registry.AttrResistanceOhms, eq, 240, "ohm"),
			),
			req("Adjust resistor", R, 1,
				numReq(registry.AttrResistanceOhms, eq, 4_700, "ohm"),
			),
			req("Input bypass", C, 1,
				numReq(registry.AttrCapacitanceF, eq, 100e-9, "F"),
			),
			req("Output smoothing", C, 1,
				numReq(registry.AttrCapacitanceF, gte, 10e-6, "F"),
			),
		),

		// ── 12. 78L05 Fixed 5V Regulator Module ──────────────────────────────
		project(
			"78L05 Fixed 5V Regulator Module",
			"Simple 5V linear regulator with input and output bypass capacitors.",
			req("Fixed regulator", IC, 1),
			req("Input capacitor", C, 1,
				numReq(registry.AttrCapacitanceF, gte, 10e-6, "F"),
				textReq(registry.AttrCapacitorType, "electrolytic"),
			),
			req("Output capacitor", C, 1,
				numReq(registry.AttrCapacitanceF, eq, 100e-9, "F"),
				textReq(registry.AttrCapacitorType, "MLCC"),
			),
		),

		// ── 13. Inverting Audio Op-Amp Stage ────────────────────────────────
		project(
			"Inverting Audio Op-Amp Stage",
			"LM741-based inverting amplifier for small audio signals.",
			req("Op-amp", IC, 1),
			req("Input resistor", R, 1,
				numReq(registry.AttrResistanceOhms, eq, 10_000, "ohm"),
			),
			req("Feedback resistor", R, 1,
				numReq(registry.AttrResistanceOhms, eq, 100_000, "ohm"),
			),
			req("Power bypass", C, 2,
				numReq(registry.AttrCapacitanceF, eq, 100e-9, "F"),
				textReq(registry.AttrCapacitorType, "MLCC"),
			),
		),

		// ── 14. Comparator Threshold Alarm ──────────────────────────────────
		project(
			"Comparator Threshold Alarm",
			"Dual comparator threshold sensor with hysteresis and pull-up resistors.",
			req("Comparator IC", IC, 1),
			req("Upper threshold resistor", R, 1,
				numReq(registry.AttrResistanceOhms, eq, 47_000, "ohm"),
			),
			req("Lower threshold resistor", R, 1,
				numReq(registry.AttrResistanceOhms, eq, 10_000, "ohm"),
			),
			req("Hysteresis capacitor", C, 1,
				numReq(registry.AttrCapacitanceF, eq, 100e-9, "F"),
			),
		),

		// ── 15. Audio High-Pass Filter ─────────────────────────────────────
		project(
			"Audio High-Pass Filter",
			"First-order RC high-pass filter for audio signals.",
			req("Series resistor", R, 1,
				numReq(registry.AttrResistanceOhms, eq, 10_000, "ohm"),
			),
			req("Coupling capacitor", C, 1,
				numReq(registry.AttrCapacitanceF, eq, 100e-9, "F"),
				textReq(registry.AttrCapacitorType, "MLCC"),
			),
		),

		// ── 16. RC Debounce Circuit ────────────────────────────────────────
		project(
			"RC Debounce Circuit",
			"Button debounce using an RC network and pull-up resistor.",
			req("Pull-up resistor", R, 1,
				numReq(registry.AttrResistanceOhms, eq, 10_000, "ohm"),
			),
			req("Debounce capacitor", C, 1,
				numReq(registry.AttrCapacitanceF, eq, 100e-9, "F"),
			),
		),

		// ── 17. Microcontroller Decoupling Kit ─────────────────────────────
		project(
			"Microcontroller Decoupling Kit",
			"Power decoupling for a microcontroller board with multiple 100nF ceramics.",
			req("Decoupling capacitor", C, 4,
				numReq(registry.AttrCapacitanceF, eq, 100e-9, "F"),
				textReq(registry.AttrCapacitorType, "MLCC"),
			),
			req("Microcontroller", IC, 1),
		),

		// ── 18. Low-Cost Power Filter ───────────────────────────────────────
		project(
			"Low-Cost Power Filter",
			"Simple LC filter using an inductor and a decoupling capacitor.",
			req("Filter inductor", L, 1,
				numReq(registry.AttrInductanceH, eq, 100e-6, "H"),
			),
			req("Filter capacitor", C, 1,
				numReq(registry.AttrCapacitanceF, eq, 100e-9, "F"),
				textReq(registry.AttrCapacitorType, "MLCC"),
			),
		),

		// ── 19. Breadboard Power Bus ───────────────────────────────────────
		project(
			"Breadboard Power Bus",
			"Regulated power bus with bulk and bypass capacitors for prototyping.",
			req("Bulk capacitor", C, 1,
				numReq(registry.AttrCapacitanceF, gte, 10e-6, "F"),
				textReq(registry.AttrCapacitorType, "electrolytic"),
			),
			req("Bypass capacitor", C, 2,
				numReq(registry.AttrCapacitanceF, eq, 100e-9, "F"),
				textReq(registry.AttrCapacitorType, "MLCC"),
			),
			req("Regulator IC", IC, 1),
		),

		// ── 20. Digital Logic Demo Board ───────────────────────────────────
		project(
			"Digital Logic Demo Board",
			"Simple logic board with a shift register and support passives.",
			req("Logic IC", IC, 1),
			req("Logic pull-up resistor", R, 2,
				numReq(registry.AttrResistanceOhms, eq, 4_700, "ohm"),
			),
			req("Decoupling capacitor", C, 1,
				numReq(registry.AttrCapacitanceF, eq, 100e-9, "F"),
				textReq(registry.AttrCapacitorType, "MLCC"),
			),
		),
	}
}

func seedComponents() []domain.Component {
	return []domain.Component{
		// ── Resistors ────────────────────────────────────────────────────────────
		resistor("RC0402JR-0710KL", "Yageo", "0402", "thick-film", 10_000, 5, 0.063, 500, "bin-R1"),
		resistor("RC0402JR-07100KL", "Yageo", "0402", "thick-film", 100_000, 5, 0.063, 320, "bin-R1"),
		resistor("RC0402JR-071KL", "Yageo", "0402", "thick-film", 1_000, 5, 0.063, 450, "bin-R1"),
		resistor("RC0402JR-07100RL", "Yageo", "0402", "thick-film", 100, 5, 0.063, 200, "bin-R1"),
		resistor("RC0402JR-0747KL", "Yageo", "0402", "thick-film", 47_000, 5, 0.063, 180, "bin-R1"),
		resistor("RC0402JR-074K7L", "Yageo", "0402", "thick-film", 4_700, 5, 0.063, 250, "bin-R1"),
		resistor("RC0402JR-0722KL", "Yageo", "0402", "thick-film", 22_000, 5, 0.063, 100, "bin-R1"),
		resistor("RC0402JR-071ML", "Yageo", "0402", "thick-film", 1_000_000, 5, 0.063, 75, "bin-R2"),
		resistor("RC0402JR-07470RL", "Yageo", "0402", "thick-film", 470, 5, 0.063, 300, "bin-R1"),
		resistor("RC0402JR-07220RL", "Yageo", "0402", "thick-film", 220, 5, 0.063, 250, "bin-R1"),
		resistor("RC0402JR-0733RL", "Yageo", "0402", "thick-film", 33, 5, 0.063, 150, "bin-R1"),
		resistor("RC0603JR-0710KL", "Yageo", "0603", "thick-film", 10_000, 5, 0.1, 300, "bin-R2"),
		resistor("RC0603JR-071KL", "Yageo", "0603", "thick-film", 1_000, 5, 0.1, 280, "bin-R2"),
		resistor("RC0603JR-07100KL", "Yageo", "0603", "thick-film", 100_000, 5, 0.1, 150, "bin-R2"),
		resistor("RC0603JR-0733KL", "Yageo", "0603", "thick-film", 33_000, 5, 0.1, 120, "bin-R2"),
		resistor("RC0603JR-0710RL", "Yageo", "0603", "thick-film", 10, 5, 0.1, 90, "bin-R2"),
		resistor("RC0805JR-0710KL", "Yageo", "0805", "thick-film", 10_000, 5, 0.125, 200, "bin-R3"),
		resistor("RC0805JR-071KL", "Yageo", "0805", "thick-film", 1_000, 5, 0.125, 160, "bin-R3"),
		resistor("CRCW120610K0FKEA", "Vishay", "1206", "thick-film", 10_000, 1, 0.25, 50, "bin-R3"),
		resistor("CRCW120610K0JNEA", "Vishay", "1206", "thick-film", 10_000, 5, 0.25, 80, "bin-R3"),
		resistor("MFR-25FBF52-10K", "Yageo", "through-hole", "metal-film", 10_000, 1, 0.25, 40, "tray-TH"),
		resistor("MFR-25FBF52-1K", "Yageo", "through-hole", "metal-film", 1_000, 1, 0.25, 60, "tray-TH"),
		resistor("MFR-25FBF52-100K", "Yageo", "through-hole", "metal-film", 100_000, 1, 0.25, 30, "tray-TH"),
		resistor("MFR-25FBF52-470R", "Yageo", "through-hole", "metal-film", 470, 1, 0.25, 80, "tray-TH"),
		resistor("MFR-25FBF52-220R", "Yageo", "through-hole", "metal-film", 220, 1, 0.25, 70, "tray-TH"),

		// ── Capacitors ───────────────────────────────────────────────────────────
		capacitor("CL05B104KO5NNNC", "Samsung", "0402", "MLCC", "X5R", 100e-9, 10, 10, 500, "bin-C1"),
		capacitor("CL05A104KA5NNNC", "Samsung", "0402", "MLCC", "X7R", 100e-9, 10, 50, 400, "bin-C1"),
		capacitor("CL05B103KB5NNNC", "Samsung", "0402", "MLCC", "X7R", 10e-9, 10, 50, 300, "bin-C1"),
		capacitor("CL05B223KB5NNNC", "Samsung", "0402", "MLCC", "X7R", 22e-9, 10, 50, 200, "bin-C1"),
		capacitor("CL10B104KA8NNNC", "Samsung", "0603", "MLCC", "X5R", 100e-9, 10, 50, 350, "bin-C2"),
		capacitor("CL10A106KQ8NNNC", "Samsung", "0603", "MLCC", "X5R", 10e-6, 10, 10, 200, "bin-C2"),
		capacitor("CL10B105KA8NNNC", "Samsung", "0603", "MLCC", "X5R", 1e-6, 10, 50, 250, "bin-C2"),
		capacitor("GRM188R71H104KA93D", "Murata", "0603", "MLCC", "X7R", 100e-9, 10, 50, 180, "bin-C2"),
		capacitor("CL21B105KAFNNNE", "Samsung", "0805", "MLCC", "X5R", 1e-6, 10, 50, 150, "bin-C3"),
		capacitor("CL21A476MQYNNNE", "Samsung", "0805", "MLCC", "X5R", 47e-6, 20, 10, 80, "bin-C3"),
		capacitor("UVR1H100MDD", "Nichicon", "through-hole", "electrolytic", "Al-electrolytic", 10e-6, 20, 50, 60, "tray-TH"),
		capacitor("UVR1H470MDD", "Nichicon", "through-hole", "electrolytic", "Al-electrolytic", 47e-6, 20, 50, 40, "tray-TH"),
		capacitor("UVR1H101MDD", "Nichicon", "through-hole", "electrolytic", "Al-electrolytic", 100e-6, 20, 50, 35, "tray-TH"),
		capacitor("UVR1A102MDD", "Nichicon", "through-hole", "electrolytic", "Al-electrolytic", 1000e-6, 20, 10, 20, "tray-TH"),
		capacitor("EEE-FK1A100R", "Panasonic", "SMD-radial", "electrolytic", "Al-electrolytic", 10e-6, 20, 10, 50, "bin-C3"),
		capacitor("UVR1V470MDD", "Nichicon", "through-hole", "electrolytic", "Al-electrolytic", 47e-6, 20, 35, 30, "tray-TH"),
		capacitor("UVZ1E101MPD", "Nichicon", "through-hole", "electrolytic", "Al-electrolytic", 100e-6, 20, 25, 25, "tray-TH"),

		// ── Inductors ────────────────────────────────────────────────────────────
		inductor("LQH32CN100K33L", "Murata", "1210", "multilayer", 100e-6, 10, 0.21, 1.6, 40, "bin-L1"),
		inductor("LQH32CN4R7M33L", "Murata", "1210", "multilayer", 4.7e-6, 20, 0.7, 0.18, 60, "bin-L1"),
		inductor("SRR1260-1R0Y", "Bourns", "1210", "shielded-power", 1e-6, 30, 5.0, 0.024, 30, "bin-L1"),
		inductor("SRR1260-100Y", "Bourns", "1210", "shielded-power", 100e-6, 20, 1.4, 0.18, 20, "bin-L1"),
		inductor("SRR1260-4R7Y", "Bourns", "1210", "shielded-power", 4.7e-6, 20, 2.8, 0.042, 25, "bin-L1"),
		inductor("AISC-0805-R10J-T", "Bourns", "0805", "wirewound", 100e-9, 5, 0.3, 0.35, 80, "bin-L1"),
		inductor("AISC-0805-R47J-T", "Bourns", "0805", "wirewound", 470e-9, 5, 0.25, 0.6, 60, "bin-L2"),
		inductor("AISC-1008-R10J-T", "Bourns", "1008", "wirewound", 100e-9, 5, 0.5, 0.22, 45, "bin-L2"),

		// ── Integrated Circuits ──────────────────────────────────────────────────
		ic("NE555P", "Texas Instruments", "DIP-8", "NE555 timer", 15, "tray-IC1"),
		ic("LM741CN", "Texas Instruments", "DIP-8", "General-purpose op-amp", 10, "tray-IC1"),
		ic("LM358N", "Texas Instruments", "DIP-8", "Dual op-amp", 12, "tray-IC1"),
		ic("LM7805CT", "Texas Instruments", "TO-220", "5V linear voltage regulator", 8, "tray-IC2"),
		ic("LM7812CT", "Texas Instruments", "TO-220", "12V linear voltage regulator", 5, "tray-IC2"),
		ic("LM317T", "Texas Instruments", "TO-220", "Adjustable positive linear regulator", 6, "tray-IC2"),
		ic("TL072CP", "Texas Instruments", "DIP-8", "Low-noise JFET-input dual op-amp", 8, "tray-IC1"),
		ic("LM393N", "Texas Instruments", "DIP-8", "Dual comparator", 6, "tray-IC1"),
		ic("74HC595N", "Texas Instruments", "DIP-16", "8-bit serial-in parallel-out shift register", 5, "tray-IC1"),
		ic("CD4017BE", "Texas Instruments", "DIP-16", "Decade counter / divider", 4, "tray-IC1"),
		ic("ATmega328P-PU", "Microchip", "DIP-28", "8-bit AVR microcontroller, 32KB flash", 4, "tray-IC3"),
		ic("ATtiny85-20PU", "Microchip", "DIP-8", "8-bit AVR microcontroller, 8KB flash", 7, "tray-IC3"),
		ic("STM32F103C8T6", "STMicroelectronics", "LQFP-48", "32-bit ARM Cortex-M3 microcontroller", 3, "tray-IC3"),
		ic("MCP2221A-I/P", "Microchip", "DIP-14", "USB 2.0 to I2C/UART protocol converter", 3, "tray-IC3"),
		ic("AMS1117-3.3", "Advanced Monolithic", "SOT-223", "3.3V LDO linear regulator, 1A", 6, "tray-IC2"),
	}
}
