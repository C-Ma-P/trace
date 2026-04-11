package activity

import (
	"fmt"
	"log"
	"sort"
	"strings"
)

type Reporter struct {
	logger  *log.Logger
	emitter Emitter
}

func NewReporter(logger *log.Logger, emitter Emitter) *Reporter {
	if logger == nil {
		logger = log.Default()
	}
	if emitter == nil {
		emitter = NopEmitter
	}
	return &Reporter{logger: logger, emitter: emitter}
}

func (r *Reporter) Emit(event Event) {
	if r == nil {
		return
	}
	r.emitter.Emit(event)
}

func (r *Reporter) Report(event Event, logPrefix string) {
	if r == nil {
		return
	}
	r.emitter.Emit(event)
	if len(event.Metadata) > 0 {
		r.logger.Printf("%s %s %s", logPrefix, event.Message, formatMeta(event.Metadata))
	} else {
		r.logger.Printf("%s %s", logPrefix, event.Message)
	}
}

type Phone struct {
	Severity Severity
	Kind     string
	Message  string
	Metadata map[string]any
}

type Sourcing struct {
	Severity Severity
	Kind     string
	Message  string
	Metadata map[string]any
}

type AssetProbe struct {
	Severity Severity
	Kind     string
	Message  string
	Metadata map[string]any
}

func (r *Reporter) Phone(p Phone) {
	r.Report(NewPhoneEvent(p.Severity, p.Kind, p.Message, p.Metadata), "[phone-intake]")
}

func (r *Reporter) Sourcing(s Sourcing) {
	r.Report(NewSourcingEvent(s.Severity, s.Kind, s.Message, s.Metadata), "[sourcing]")
}

func (r *Reporter) AssetProbe(a AssetProbe) {
	r.Report(NewAssetProbeEvent(a.Severity, a.Kind, a.Message, a.Metadata), "[sourcing]")
}

// formatMeta renders metadata as sorted "key=value" pairs joined by spaces.
func formatMeta(m map[string]any) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%v", k, m[k]))
	}
	return strings.Join(parts, " ")
}
