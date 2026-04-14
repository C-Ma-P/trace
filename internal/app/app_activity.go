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

func (a *App) emitActivityError(kind, message string) {
	if a.activityHub != nil {
		a.activityHub.Emit(activity.Event{
			Domain:   activity.DomainActivity,
			Severity: activity.SeverityError,
			Kind:     kind,
			Message:  message,
		})
	}
}
