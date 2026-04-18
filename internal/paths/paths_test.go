package paths

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPathBuildersUseTraceHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	traceHome, err := TraceHomeDir()
	if err != nil {
		t.Fatalf("TraceHomeDir error: %v", err)
	}
	expectedHome := filepath.Join(home, ".trace")
	if traceHome != expectedHome {
		t.Fatalf("expected %q, got %q", expectedHome, traceHome)
	}

	projects, err := ProjectsDir()
	if err != nil {
		t.Fatalf("ProjectsDir error: %v", err)
	}
	if projects != filepath.Join(expectedHome, "projects") {
		t.Fatalf("unexpected projects path: %q", projects)
	}

	launcher, err := LauncherStatePath()
	if err != nil {
		t.Fatalf("LauncherStatePath error: %v", err)
	}
	if launcher != filepath.Join(expectedHome, "launcher.json") {
		t.Fatalf("unexpected launcher state path: %q", launcher)
	}

	assets, err := AssetsDir()
	if err != nil {
		t.Fatalf("AssetsDir error: %v", err)
	}
	if assets != filepath.Join(expectedHome, "assets") {
		t.Fatalf("unexpected assets path: %q", assets)
	}
}

func TestEnsureDirsCreateExpectedPaths(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	traceHome, err := EnsureTraceHome()
	if err != nil {
		t.Fatalf("EnsureTraceHome error: %v", err)
	}
	if info, err := os.Stat(traceHome); err != nil || !info.IsDir() {
		t.Fatalf("expected trace home directory to exist")
	}

	assets, err := EnsureAssetsDir()
	if err != nil {
		t.Fatalf("EnsureAssetsDir error: %v", err)
	}
	if info, err := os.Stat(assets); err != nil || !info.IsDir() {
		t.Fatalf("expected assets directory to exist")
	}

	projects, err := EnsureProjectsDir()
	if err != nil {
		t.Fatalf("EnsureProjectsDir error: %v", err)
	}
	if info, err := os.Stat(projects); err != nil || !info.IsDir() {
		t.Fatalf("expected projects directory to exist")
	}

	pki, err := EnsurePhoneIntakePKIDir()
	if err != nil {
		t.Fatalf("EnsurePhoneIntakePKIDir error: %v", err)
	}
	if info, err := os.Stat(pki); err != nil || !info.IsDir() {
		t.Fatalf("expected pki directory to exist")
	}
}
