package activity_test

import (
	"sync"
	"testing"
	"time"

	"trace/internal/activity"
)

func TestHubMaintainsBoundedRecentEvents(t *testing.T) {
	hub := activity.NewHub(3)

	for i := 0; i < 5; i++ {
		hub.Emit(activity.NewActivityEvent(activity.SeverityInfo, "test", "event", map[string]any{"index": i}))
	}

	events := hub.RecentEvents()
	if len(events) != 3 {
		t.Fatalf("expected 3 recent events, got %d", len(events))
	}
	if events[0].Metadata["index"] != 4 || events[2].Metadata["index"] != 2 {
		t.Fatalf("unexpected event order: %#v", events)
	}
}

func TestHubEmitDoesNotBlockSlowSubscriber(t *testing.T) {
	hub := activity.NewHub(10)
	ch := hub.Subscribe()
	defer hub.Unsubscribe(ch)

	received := make([]activity.Event, 0, 2)
	var mu sync.Mutex

	go func() {
		time.Sleep(100 * time.Millisecond)
		mu.Lock()
		for len(ch) > 0 {
			received = append(received, <-ch)
		}
		mu.Unlock()
	}()

	hub.Emit(activity.NewActivityEvent(activity.SeverityInfo, "slow-subscriber", "first", nil))
	hub.Emit(activity.NewActivityEvent(activity.SeverityInfo, "slow-subscriber", "second", nil))
}
