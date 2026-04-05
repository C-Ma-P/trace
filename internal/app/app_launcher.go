package app

import "fmt"

func (a *App) ListRecentProjects() []RecentProjectResponse {
	if a.launcher == nil {
		return []RecentProjectResponse{}
	}
	st := a.launcher.Load()
	out := make([]RecentProjectResponse, len(st.RecentProjects))
	for i, rp := range st.RecentProjects {
		out[i] = RecentProjectResponse{
			ID:          rp.ID,
			Name:        rp.Name,
			Subtitle:    rp.Subtitle,
			OpenedAtUTC: rp.OpenedAtUTC,
			Pinned:      rp.Pinned,
		}
	}
	return out
}

func (a *App) SetRecentProjectPinned(projectID string, pinned bool) error {
	if a.launcher == nil {
		return fmt.Errorf("launcher store not available")
	}
	return a.launcher.SetPinned(projectID, pinned)
}

func (a *App) ReorderRecentProjects(projectIDs []string) error {
	if a.launcher == nil {
		return fmt.Errorf("launcher store not available")
	}
	return a.launcher.Reorder(projectIDs)
}
