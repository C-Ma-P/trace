package activity_test

import (
	"bytes"
	"log"
	"strings"
	"sync"
	"testing"

	"trace/internal/activity"
)

// captureEmitter records every emitted event.
type captureEmitter struct {
	mu     sync.Mutex
	events []activity.Event
}

func (c *captureEmitter) Emit(e activity.Event) {
	c.mu.Lock()
	c.events = append(c.events, e)
	c.mu.Unlock()
}

func (c *captureEmitter) take() []activity.Event {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := c.events
	c.events = nil
	return out
}

// newTestLogger returns a logger that writes to a *bytes.Buffer.
func newTestLogger() (*log.Logger, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	return log.New(buf, "", 0), buf
}

func TestReporterNilSafe(t *testing.T) {
	// A nil *Reporter must not panic on any method call.
	var r *activity.Reporter
	r.Emit(activity.NewPhoneEvent(activity.SeverityInfo, "k", "m", nil))
	r.Report(activity.NewPhoneEvent(activity.SeverityInfo, "k", "m", nil), "[prefix]")
	r.Phone(activity.Phone{
		Severity: activity.SeverityInfo,
		Kind:     "k",
		Message:  "m",
	})
	r.Sourcing(activity.Sourcing{
		Severity: activity.SeverityInfo,
		Kind:     "k",
		Message:  "m",
	})
	r.AssetProbe(activity.AssetProbe{
		Severity: activity.SeverityInfo,
		Kind:     "k",
		Message:  "m",
	})
}

func TestNewReporterNilEmitterDefaultsToNop(t *testing.T) {
	// Passing nil emitter must not panic and must not emit anything observable.
	logger, _ := newTestLogger()
	r := activity.NewReporter(logger, nil)
	// Should not panic:
	r.Phone(activity.Phone{
		Severity: activity.SeverityInfo,
		Kind:     "test-kind",
		Message:  "test message",
	})
}

func TestNewReporterNilLoggerUsesDefault(t *testing.T) {
	// Passing nil logger should not panic; it uses log.Default().
	emitter := &captureEmitter{}
	r := activity.NewReporter(nil, emitter)
	r.Phone(activity.Phone{
		Severity: activity.SeverityInfo,
		Kind:     "k",
		Message:  "m",
	})
	events := emitter.take()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
}

func TestReporterEmitDoesNotLog(t *testing.T) {
	emitter := &captureEmitter{}
	logger, buf := newTestLogger()
	r := activity.NewReporter(logger, emitter)

	r.Emit(activity.NewPhoneEvent(activity.SeverityInfo, "silent-kind", "silent message", nil))

	events := emitter.take()
	if len(events) != 1 {
		t.Fatalf("expected 1 emitted event, got %d", len(events))
	}
	if buf.Len() != 0 {
		t.Fatalf("expected no log output, got: %q", buf.String())
	}
}

func TestReporterReportEmitsAndLogs(t *testing.T) {
	emitter := &captureEmitter{}
	logger, buf := newTestLogger()
	r := activity.NewReporter(logger, emitter)

	event := activity.NewSourcingEvent(activity.SeverityInfo, "cache-hit", "Sourcing cache hit", nil)
	r.Report(event, "[sourcing]")

	events := emitter.take()
	if len(events) != 1 {
		t.Fatalf("expected 1 emitted event, got %d", len(events))
	}
	if events[0].Kind != "cache-hit" {
		t.Errorf("unexpected kind: %q", events[0].Kind)
	}
	logLine := buf.String()
	if !strings.Contains(logLine, "[sourcing]") {
		t.Errorf("log line missing prefix: %q", logLine)
	}
	if !strings.Contains(logLine, "Sourcing cache hit") {
		t.Errorf("log line missing message: %q", logLine)
	}
}

func TestReporterReportIncludesMetadataInLog(t *testing.T) {
	emitter := &captureEmitter{}
	logger, buf := newTestLogger()
	r := activity.NewReporter(logger, emitter)

	meta := map[string]any{"mpn": "STM32F103", "vendor": "digikey"}
	event := activity.NewSourcingEvent(activity.SeverityInfo, "cache-miss", "Sourcing cache miss", meta)
	r.Report(event, "[sourcing]")

	logLine := buf.String()
	if !strings.Contains(logLine, "mpn=STM32F103") {
		t.Errorf("log line missing mpn metadata: %q", logLine)
	}
	if !strings.Contains(logLine, "vendor=digikey") {
		t.Errorf("log line missing vendor metadata: %q", logLine)
	}
}

func TestReporterPhoneEmitsPhoneDomain(t *testing.T) {
	emitter := &captureEmitter{}
	logger, buf := newTestLogger()
	r := activity.NewReporter(logger, emitter)

	r.Phone(activity.Phone{
		Severity: activity.SeverityWarning,
		Kind:     "lookup-failed",
		Message:  "Vendor lookup failed",
		Metadata: map[string]any{"vendor": "lcsc", "partId": "C123"},
	})

	events := emitter.take()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Domain != activity.DomainPhone {
		t.Errorf("expected domain %q, got %q", activity.DomainPhone, events[0].Domain)
	}
	if events[0].Severity != activity.SeverityWarning {
		t.Errorf("unexpected severity: %q", events[0].Severity)
	}
	logLine := buf.String()
	if !strings.Contains(logLine, "[phone-intake]") {
		t.Errorf("log line missing [phone-intake] prefix: %q", logLine)
	}
}

func TestReporterSourcingEmitsSourcingDomain(t *testing.T) {
	emitter := &captureEmitter{}
	logger, buf := newTestLogger()
	r := activity.NewReporter(logger, emitter)

	r.Sourcing(activity.Sourcing{
		Severity: activity.SeverityError,
		Kind:     "request-failed",
		Message:  "Sourcing request failed",
		Metadata: map[string]any{"error": "timeout"},
	})

	events := emitter.take()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Domain != activity.DomainSourcing {
		t.Errorf("expected domain %q, got %q", activity.DomainSourcing, events[0].Domain)
	}
	logLine := buf.String()
	if !strings.Contains(logLine, "[sourcing]") {
		t.Errorf("log line missing [sourcing] prefix: %q", logLine)
	}
}

func TestReporterAssetProbeEmitsAssetProbeDomain(t *testing.T) {
	emitter := &captureEmitter{}
	logger, buf := newTestLogger()
	r := activity.NewReporter(logger, emitter)

	r.AssetProbe(activity.AssetProbe{
		Severity: activity.SeverityInfo,
		Kind:     "cache-hit",
		Message:  "Asset probe cache hit",
		Metadata: map[string]any{"mpn": "LM741"},
	})

	events := emitter.take()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Domain != activity.DomainAssetProbe {
		t.Errorf("expected domain %q, got %q", activity.DomainAssetProbe, events[0].Domain)
	}
	logLine := buf.String()
	if !strings.Contains(logLine, "[sourcing]") {
		t.Errorf("asset probe log line missing [sourcing] prefix: %q", logLine)
	}
}

func TestReporterReportEmitsExactlyOnce(t *testing.T) {
	emitter := &captureEmitter{}
	logger, _ := newTestLogger()
	r := activity.NewReporter(logger, emitter)

	r.Report(activity.NewPhoneEvent(activity.SeverityInfo, "k", "m", nil), "[phone-intake]")
	r.Report(activity.NewPhoneEvent(activity.SeverityInfo, "k", "m", nil), "[phone-intake]")

	events := emitter.take()
	if len(events) != 2 {
		t.Fatalf("expected 2 events (one per Report call), got %d", len(events))
	}
}

func TestReporterMetadataCopiedNotMutated(t *testing.T) {
	emitter := &captureEmitter{}
	logger, _ := newTestLogger()
	r := activity.NewReporter(logger, emitter)

	meta := map[string]any{"key": "original"}
	r.Phone(activity.Phone{
		Severity: activity.SeverityInfo,
		Kind:     "k",
		Message:  "m",
		Metadata: meta,
	})

	// Mutate original after emit.
	meta["key"] = "mutated"

	events := emitter.take()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Metadata["key"] != "original" {
		t.Errorf("event metadata was mutated: got %v", events[0].Metadata["key"])
	}
}

func TestReporterLogMetadataSortedStably(t *testing.T) {
	emitter := &captureEmitter{}
	logger, buf := newTestLogger()
	r := activity.NewReporter(logger, emitter)

	// Run multiple times to catch any non-determinism from map iteration.
	for i := 0; i < 20; i++ {
		buf.Reset()
		r.Sourcing(activity.Sourcing{
			Severity: activity.SeverityInfo,
			Kind:     "k",
			Message:  "msg",
			Metadata: map[string]any{"z": 1, "a": 2, "m": 3},
		})
		logLine := buf.String()
		idx := func(s string) int { return strings.Index(logLine, s) }
		if idx("a=2") > idx("m=3") || idx("m=3") > idx("z=1") {
			t.Errorf("metadata not sorted: %q", logLine)
		}
	}
}
