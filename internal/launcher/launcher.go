package launcher

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/C-Ma-P/trace/internal/paths"
)

const maxRecentProjects = 10

type RecentProject struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Subtitle    string `json:"subtitle,omitempty"`
	OpenedAtUTC string `json:"openedAtUtc"`
	Pinned      bool   `json:"pinned,omitempty"`
}

type State struct {
	RecentProjects []RecentProject `json:"recentProjects"`
}

type Store struct {
	mu sync.Mutex
}

func NewStore() *Store {
	return &Store{}
}

func (s *Store) Load() State {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadUnsafe()
}

func (s *Store) loadUnsafe() State {
	p, err := paths.LauncherStatePath()
	if err != nil {
		return State{}
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return State{}
	}
	var st State
	if err := json.Unmarshal(data, &st); err != nil {
		return State{}
	}
	return st
}

func (s *Store) TouchProject(id, name, subtitle string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	st := s.loadUnsafe()

	var existing *RecentProject
	filtered := make([]RecentProject, 0, len(st.RecentProjects))
	for _, rp := range st.RecentProjects {
		if rp.ID == id {
			rpCopy := rp
			existing = &rpCopy
			continue
		}
		filtered = append(filtered, rp)
	}

	entry := RecentProject{
		ID:          id,
		Name:        name,
		Subtitle:    subtitle,
		OpenedAtUTC: time.Now().UTC().Format(time.RFC3339),
	}
	if existing != nil {
		entry.Pinned = existing.Pinned
	}

	pinned := make([]RecentProject, 0, len(filtered))
	unpinned := make([]RecentProject, 0, len(filtered))
	for _, rp := range filtered {
		if rp.Pinned {
			pinned = append(pinned, rp)
		} else {
			unpinned = append(unpinned, rp)
		}
	}

	if entry.Pinned {
		pinned = append(pinned, entry)
	} else {
		unpinned = append([]RecentProject{entry}, unpinned...)
	}
	if len(unpinned) > maxRecentProjects {
		unpinned = unpinned[:maxRecentProjects]
	}
	st.RecentProjects = append([]RecentProject{}, pinned...)
	st.RecentProjects = append(st.RecentProjects, unpinned...)

	return s.saveUnsafe(st)
}

func (s *Store) SetPinned(id string, pinned bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	st := s.loadUnsafe()
	for i := range st.RecentProjects {
		if st.RecentProjects[i].ID == id {
			st.RecentProjects[i].Pinned = pinned
			break
		}
	}
	st.RecentProjects = normalizeRecentProjects(st.RecentProjects)
	return s.saveUnsafe(st)
}

func (s *Store) Reorder(ids []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	st := s.loadUnsafe()
	byID := map[string]RecentProject{}
	for _, rp := range st.RecentProjects {
		byID[rp.ID] = rp
	}

	reordered := make([]RecentProject, 0, len(st.RecentProjects))
	seen := map[string]struct{}{}
	for _, id := range ids {
		if rp, ok := byID[id]; ok {
			reordered = append(reordered, rp)
			seen[id] = struct{}{}
		}
	}
	for _, rp := range st.RecentProjects {
		if _, ok := seen[rp.ID]; ok {
			continue
		}
		reordered = append(reordered, rp)
	}

	st.RecentProjects = normalizeRecentProjects(reordered)
	return s.saveUnsafe(st)
}

func normalizeRecentProjects(in []RecentProject) []RecentProject {
	pinned := make([]RecentProject, 0, len(in))
	unpinned := make([]RecentProject, 0, len(in))
	for _, rp := range in {
		if rp.Pinned {
			pinned = append(pinned, rp)
		} else {
			unpinned = append(unpinned, rp)
		}
	}
	if len(unpinned) > maxRecentProjects {
		unpinned = unpinned[:maxRecentProjects]
	}
	out := append([]RecentProject{}, pinned...)
	out = append(out, unpinned...)
	return out
}

func (s *Store) RemoveProject(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	st := s.loadUnsafe()
	filtered := make([]RecentProject, 0, len(st.RecentProjects))
	for _, rp := range st.RecentProjects {
		if rp.ID != id {
			filtered = append(filtered, rp)
		}
	}
	st.RecentProjects = filtered
	return s.saveUnsafe(st)
}

func (s *Store) saveUnsafe(st State) error {
	if _, err := paths.EnsureTraceHome(); err != nil {
		return err
	}

	p, err := paths.LauncherStatePath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal launcher state: %w", err)
	}

	dir := filepath.Dir(p)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create launcher state dir: %w", err)
	}

	return writeFileAtomic(p, data, 0o644)
}

func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	base := filepath.Base(path)

	f, err := os.CreateTemp(dir, base+".*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmp := f.Name()

	cleanup := func() {
		_ = f.Close()
		_ = os.Remove(tmp)
	}

	if _, err := f.Write(data); err != nil {
		cleanup()
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := f.Chmod(perm); err != nil {
		cleanup()
		return fmt.Errorf("chmod temp file: %w", err)
	}
	if err := f.Close(); err != nil {
		cleanup()
		return fmt.Errorf("close temp file: %w", err)
	}

	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename temp file: %w", err)
	}

	return nil
}
