package activity

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

type Domain string

type Severity string

const (
	DomainActivity   Domain = "activity"
	DomainSourcing   Domain = "sourcing"
	DomainPhone      Domain = "phone"
	DomainImport     Domain = "import"
	DomainAssetProbe Domain = "asset-probe"

	SeverityInfo    Severity = "info"
	SeveritySuccess Severity = "success"
	SeverityWarning Severity = "warning"
	SeverityError   Severity = "error"
)

type Event struct {
	ID        string         `json:"id"`
	Timestamp time.Time      `json:"timestamp"`
	Domain    Domain         `json:"domain"`
	Severity  Severity       `json:"severity"`
	Kind      string         `json:"kind,omitempty"`
	Message   string         `json:"message"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

type Emitter interface {
	Emit(Event)
}

type nopEmitter struct{}

func (nopEmitter) Emit(Event) {}

var NopEmitter Emitter = nopEmitter{}

const defaultHubCapacity = 100

func NewHub(maxEvents int) *Hub {
	if maxEvents <= 0 {
		maxEvents = defaultHubCapacity
	}
	return &Hub{
		maxEvents: maxEvents,
		subs:      make(map[chan Event]struct{}),
	}
}

type Hub struct {
	mu        sync.Mutex
	maxEvents int
	events    []Event
	subs      map[chan Event]struct{}
}

func (h *Hub) Emit(event Event) {
	if h == nil {
		return
	}

	if event.ID == "" {
		event.ID = generateID(event)
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	h.events = append([]Event{event}, h.events...)
	if len(h.events) > h.maxEvents {
		h.events = h.events[:h.maxEvents]
	}

	for ch := range h.subs {
		select {
		case ch <- event:
		default:
		}
	}
}

func (h *Hub) RecentEvents() []Event {
	if h == nil {
		return nil
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]Event, len(h.events))
	copy(out, h.events)
	return out
}

func (h *Hub) Subscribe() chan Event {
	if h == nil {
		return make(chan Event, 1)
	}
	ch := make(chan Event, 32)
	h.mu.Lock()
	h.subs[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

func (h *Hub) Unsubscribe(ch chan Event) {
	if h == nil || ch == nil {
		return
	}
	h.mu.Lock()
	if _, ok := h.subs[ch]; ok {
		delete(h.subs, ch)
		close(ch)
	}
	h.mu.Unlock()
}

func NewEvent(domain Domain, severity Severity, kind, message string, metadata map[string]any) Event {
	return Event{
		Domain:   domain,
		Severity: severity,
		Kind:     kind,
		Message:  message,
		Metadata: copyMetadata(metadata),
	}
}

func NewActivityEvent(severity Severity, kind, message string, metadata map[string]any) Event {
	return NewEvent(DomainActivity, severity, kind, message, metadata)
}

func NewSourcingEvent(severity Severity, kind, message string, metadata map[string]any) Event {
	return NewEvent(DomainSourcing, severity, kind, message, metadata)
}

func NewPhoneEvent(severity Severity, kind, message string, metadata map[string]any) Event {
	return NewEvent(DomainPhone, severity, kind, message, metadata)
}

func NewImportEvent(severity Severity, kind, message string, metadata map[string]any) Event {
	return NewEvent(DomainImport, severity, kind, message, metadata)
}

func NewAssetProbeEvent(severity Severity, kind, message string, metadata map[string]any) Event {
	return NewEvent(DomainAssetProbe, severity, kind, message, metadata)
}

func copyMetadata(input map[string]any) map[string]any {
	if input == nil {
		return nil
	}
	out := make(map[string]any, len(input))
	for k, v := range input {
		out[k] = v
	}
	return out
}

func generateID(event Event) string {
	return fmt.Sprintf("%s-%d-%x", event.Domain, time.Now().UnixNano(), rand.Int63())
}
