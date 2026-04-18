package launcher

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestStoreTouchPinReorderRemove(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	store := NewStore()
	if err := store.TouchProject("p1", "Project 1", "first"); err != nil {
		t.Fatalf("touch p1: %v", err)
	}
	if err := store.TouchProject("p2", "Project 2", "second"); err != nil {
		t.Fatalf("touch p2: %v", err)
	}

	if err := store.SetPinned("p1", true); err != nil {
		t.Fatalf("pin p1: %v", err)
	}
	if err := store.Reorder([]string{"p2", "p1"}); err != nil {
		t.Fatalf("reorder: %v", err)
	}

	st := store.Load()
	if len(st.RecentProjects) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(st.RecentProjects))
	}
	if st.RecentProjects[0].ID != "p1" || !st.RecentProjects[0].Pinned {
		t.Fatalf("expected pinned project p1 first, got %+v", st.RecentProjects[0])
	}
	if st.RecentProjects[1].ID != "p2" {
		t.Fatalf("expected p2 second, got %+v", st.RecentProjects[1])
	}

	if err := store.RemoveProject("p1"); err != nil {
		t.Fatalf("remove p1: %v", err)
	}
	st = store.Load()
	if len(st.RecentProjects) != 1 || st.RecentProjects[0].ID != "p2" {
		t.Fatalf("expected only p2 after removal, got %+v", st.RecentProjects)
	}
}

func TestStoreTouchProjectCapsUnpinnedProjects(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	store := NewStore()
	for i := 0; i < maxRecentProjects+3; i++ {
		id := fmt.Sprintf("project-%d", i)
		if err := store.TouchProject(id, "Project", ""); err != nil {
			t.Fatalf("touch %s: %v", id, err)
		}
	}

	st := store.Load()
	if len(st.RecentProjects) != maxRecentProjects {
		t.Fatalf("expected %d projects capped, got %d", maxRecentProjects, len(st.RecentProjects))
	}
}

func TestWriteFileAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	want := []byte("hello")

	if err := writeFileAtomic(path, want, 0o644); err != nil {
		t.Fatalf("writeFileAtomic: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read written file: %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("unexpected content: %q", string(got))
	}
}
