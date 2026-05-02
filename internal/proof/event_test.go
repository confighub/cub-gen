package proof

import (
	"strings"
	"testing"
	"time"
)

func TestNewEventIsTraceableAndDeterministic(t *testing.T) {
	at := time.Date(2026, 3, 6, 10, 0, 0, 0, time.UTC)
	event := NewEvent(Input{
		EventType:      EventTypeChangeBundlePublished,
		EventTime:      at,
		Source:         "cub-gen",
		ChangeID:       "chg_123",
		Space:          "platform",
		TargetSlug:     "helm-paas",
		TargetPath:     "/repo",
		ArtifactKind:   ArtifactKindChangeBundle,
		ArtifactDigest: "sha256:abc",
		SummaryCounts: map[string]int{
			"provenance_records": 1,
		},
		GeneratorProfiles: []string{"helm-paas"},
	})

	if event.SchemaVersion != SchemaVersion {
		t.Fatalf("unexpected schema_version: %q", event.SchemaVersion)
	}
	if event.TraceID != "chg_123" {
		t.Fatalf("expected trace_id from change_id, got %q", event.TraceID)
	}
	if !strings.HasPrefix(event.EventID, "evt_") {
		t.Fatalf("expected evt_ event id, got %q", event.EventID)
	}

	again := NewEvent(Input{
		EventType:      EventTypeChangeBundlePublished,
		EventTime:      at,
		Source:         "cub-gen",
		ChangeID:       "chg_123",
		Space:          "platform",
		TargetSlug:     "helm-paas",
		TargetPath:     "/repo",
		ArtifactKind:   ArtifactKindChangeBundle,
		ArtifactDigest: "sha256:def",
		SummaryCounts: map[string]int{
			"provenance_records": 1,
		},
		GeneratorProfiles: []string{"helm-paas"},
	})
	if event.EventID != again.EventID {
		t.Fatalf("expected event id to ignore artifact digest, got %q and %q", event.EventID, again.EventID)
	}
}

func TestTraceIDFallback(t *testing.T) {
	got := TraceID("", "platform", "target", "render", "HEAD")
	again := TraceID("", "platform", "target", "render", "HEAD")
	if got == "" || !strings.HasPrefix(got, "trace_") {
		t.Fatalf("expected fallback trace id, got %q", got)
	}
	if got != again {
		t.Fatalf("expected deterministic fallback trace id, got %q and %q", got, again)
	}
}

func TestValidateArtifactEvents(t *testing.T) {
	at := time.Date(2026, 3, 6, 10, 0, 0, 0, time.UTC)
	event := NewEvent(Input{
		EventType:      EventTypeAttestationVerified,
		EventTime:      at,
		Source:         "cub-gen",
		ChangeID:       "chg_123",
		Space:          "platform",
		ArtifactKind:   ArtifactKindAttestation,
		ArtifactDigest: "sha256:abc",
	})

	err := ValidateArtifactEvents([]Event{event}, Expected{
		EventType:      EventTypeAttestationVerified,
		EventTime:      "2026-03-06T10:00:00Z",
		Source:         "cub-gen",
		TraceID:        "chg_123",
		ChangeID:       "chg_123",
		Space:          "platform",
		ArtifactKind:   ArtifactKindAttestation,
		ArtifactDigest: "sha256:abc",
	})
	if err != nil {
		t.Fatalf("ValidateArtifactEvents returned error: %v", err)
	}

	tampered := event
	tampered.TraceID = "different"
	if err := ValidateArtifactEvents([]Event{tampered}, Expected{
		EventType:      EventTypeAttestationVerified,
		EventTime:      "2026-03-06T10:00:00Z",
		Source:         "cub-gen",
		TraceID:        "chg_123",
		ChangeID:       "chg_123",
		Space:          "platform",
		ArtifactKind:   ArtifactKindAttestation,
		ArtifactDigest: "sha256:abc",
	}); err == nil || !strings.Contains(err.Error(), "event_id mismatch") {
		t.Fatalf("expected event id mismatch, got %v", err)
	}
}
