package app

import "github.com/C-Ma-P/trace/internal/activity"

func (a *App) SetActivityHub(hub *activity.Hub) {
	a.activityHub = hub
}

func (a *App) GetActivityEvents() []activity.Event {
	if a.activityHub == nil {
		return nil
	}
	return a.activityHub.RecentEvents()
}
