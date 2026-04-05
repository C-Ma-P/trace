package kicadconfig

import (
	"context"
	"strings"
	"testing"
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

func TestSavePreferences_NormalizesRoots(t *testing.T) {
	repo := newFakePreferenceRepo()
	mgr := NewManager(repo)

	prefs, err := mgr.SavePreferences(context.Background(), UpdateInput{
		ProjectRoots: []string{"  /tmp/projects  ", "/tmp/projects", "/tmp/kicad/../boards"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got, want := len(prefs.ProjectRoots), 2; got != want {
		t.Fatalf("expected %d roots, got %d", want, got)
	}
	if prefs.ProjectRoots[0] != "/tmp/projects" {
		t.Fatalf("expected first normalized root, got %q", prefs.ProjectRoots[0])
	}
	if prefs.ProjectRoots[1] != "/tmp/boards" {
		t.Fatalf("expected cleaned second root, got %q", prefs.ProjectRoots[1])
	}
	if raw := repo.values[prefProjectRoots]; raw != `["/tmp/projects","/tmp/boards"]` {
		t.Fatalf("unexpected persisted payload: %q", raw)
	}
}

func TestGetPreferences_EmptyWhenUnset(t *testing.T) {
	repo := newFakePreferenceRepo()
	mgr := NewManager(repo)

	prefs, err := mgr.GetPreferences(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(prefs.ProjectRoots) != 0 {
		t.Fatalf("expected no stored roots, got %#v", prefs.ProjectRoots)
	}
}
