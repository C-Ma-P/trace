package paths

import (
	"fmt"
	"os"
	"path/filepath"
)

const traceHomeDirName = ".trace"

func TraceHomeDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home dir: %w", err)
	}
	return filepath.Join(home, traceHomeDirName), nil
}

func ProjectsDir() (string, error) {
	home, err := TraceHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "projects"), nil
}

func LauncherStatePath() (string, error) {
	home, err := TraceHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "launcher.json"), nil
}

func AssetsDir() (string, error) {
	home, err := TraceHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "assets"), nil
}

func EnsureAssetsDir() (string, error) {
	dir, err := AssetsDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create assets dir: %w", err)
	}
	return dir, nil
}

func EnsureTraceHome() (string, error) {
	home, err := TraceHomeDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(home, 0o755); err != nil {
		return "", fmt.Errorf("create trace home dir: %w", err)
	}
	return home, nil
}

func EnsurePhoneIntakePKIDir() (string, error) {
	home, err := TraceHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, "phoneintake-pki")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("create phoneintake pki dir: %w", err)
	}
	return dir, nil
}

func EnsureProjectsDir() (string, error) {
	projects, err := ProjectsDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(projects, 0o755); err != nil {
		return "", fmt.Errorf("create projects dir: %w", err)
	}
	return projects, nil
}
