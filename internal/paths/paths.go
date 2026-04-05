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
