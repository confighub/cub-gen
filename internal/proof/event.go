package proof

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	SchemaVersion = "cub.confighub.io/proof-event/v1"

	EventTypeChangeBundlePublished = "change_bundle.published"
	EventTypeAttestationVerified   = "attestation.verified"

	ArtifactKindChangeBundle = "change_bundle"
	ArtifactKindAttestation  = "attestation"

	digestAlgorithm = "sha256"
)

// Event is a log-safe proof statement. It can be copied into Pilot logs,
// validation output, or attestation stores without losing the change trace.
type Event struct {
	SchemaVersion        string         `json:"schema_version"`
	EventID              string         `json:"event_id"`
	EventType            string         `json:"event_type"`
	EventTime            string         `json:"event_time"`
	Source               string         `json:"source"`
	TraceID              string         `json:"trace_id"`
	ChangeID             string         `json:"change_id,omitempty"`
	Space                string         `json:"space,omitempty"`
	TargetSlug           string         `json:"target_slug,omitempty"`
	TargetPath           string         `json:"target_path,omitempty"`
	RenderTargetSlug     string         `json:"render_target_slug,omitempty"`
	RenderTargetPath     string         `json:"render_target_path,omitempty"`
	ArtifactKind         string         `json:"artifact_kind"`
	ArtifactDigest       string         `json:"artifact_digest,omitempty"`
	ParentEventID        string         `json:"parent_event_id,omitempty"`
	ParentArtifactKind   string         `json:"parent_artifact_kind,omitempty"`
	ParentArtifactDigest string         `json:"parent_artifact_digest,omitempty"`
	SummaryCounts        map[string]int `json:"summary_counts,omitempty"`
	GeneratorProfiles    []string       `json:"generator_profiles,omitempty"`
}

// Input contains the stable fields used to create a proof event.
type Input struct {
	EventType            string
	EventTime            time.Time
	Source               string
	ChangeID             string
	Space                string
	TargetSlug           string
	TargetPath           string
	RenderTargetSlug     string
	RenderTargetPath     string
	Ref                  string
	ArtifactKind         string
	ArtifactDigest       string
	ParentEventID        string
	ParentArtifactKind   string
	ParentArtifactDigest string
	SummaryCounts        map[string]int
	GeneratorProfiles    []string
}

// NewEvent returns a deterministic proof event for a generated artifact.
func NewEvent(input Input) Event {
	profiles := append([]string(nil), input.GeneratorProfiles...)
	counts := copyCounts(input.SummaryCounts)
	event := Event{
		SchemaVersion:        SchemaVersion,
		EventType:            strings.TrimSpace(input.EventType),
		EventTime:            input.EventTime.UTC().Format(time.RFC3339),
		Source:               strings.TrimSpace(input.Source),
		TraceID:              TraceID(input.ChangeID, input.Space, input.TargetSlug, input.RenderTargetSlug, input.Ref),
		ChangeID:             strings.TrimSpace(input.ChangeID),
		Space:                strings.TrimSpace(input.Space),
		TargetSlug:           strings.TrimSpace(input.TargetSlug),
		TargetPath:           strings.TrimSpace(input.TargetPath),
		RenderTargetSlug:     strings.TrimSpace(input.RenderTargetSlug),
		RenderTargetPath:     strings.TrimSpace(input.RenderTargetPath),
		ArtifactKind:         strings.TrimSpace(input.ArtifactKind),
		ArtifactDigest:       strings.TrimSpace(input.ArtifactDigest),
		ParentEventID:        strings.TrimSpace(input.ParentEventID),
		ParentArtifactKind:   strings.TrimSpace(input.ParentArtifactKind),
		ParentArtifactDigest: strings.TrimSpace(input.ParentArtifactDigest),
		SummaryCounts:        counts,
		GeneratorProfiles:    profiles,
	}
	event.EventID = EventID(event)
	return event
}

// TraceID returns the cross-artifact trace key. When a change_id exists, it is
// the trace key. Otherwise a stable fallback is derived from the local context.
func TraceID(changeID, space, targetSlug, renderTargetSlug, ref string) string {
	if changeID = strings.TrimSpace(changeID); changeID != "" {
		return changeID
	}
	parts := []string{
		strings.TrimSpace(space),
		strings.TrimSpace(targetSlug),
		strings.TrimSpace(renderTargetSlug),
		strings.TrimSpace(ref),
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return "trace_" + hex.EncodeToString(sum[:])[:16]
}

// EventID returns the deterministic event id. ArtifactDigest is deliberately
// excluded so an event can identify the artifact digest it is embedded inside.
func EventID(event Event) string {
	input := event
	input.EventID = ""
	input.ArtifactDigest = ""
	b, err := json.Marshal(input)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(b)
	return "evt_" + hex.EncodeToString(sum[:])[:24]
}

// BlankArtifactDigests returns a deep copy with event artifact digests removed
// before envelope digest calculation.
func BlankArtifactDigests(events []Event) []Event {
	copied := append([]Event(nil), events...)
	for i := range copied {
		copied[i].ArtifactDigest = ""
		copied[i].SummaryCounts = copyCounts(copied[i].SummaryCounts)
		copied[i].GeneratorProfiles = append([]string(nil), copied[i].GeneratorProfiles...)
	}
	return copied
}

// SetArtifactDigest returns a deep copy with the artifact digest set on every
// event that belongs to the artifact kind.
func SetArtifactDigest(events []Event, artifactKind, digest string) []Event {
	copied := append([]Event(nil), events...)
	for i := range copied {
		copied[i].SummaryCounts = copyCounts(copied[i].SummaryCounts)
		copied[i].GeneratorProfiles = append([]string(nil), copied[i].GeneratorProfiles...)
		if copied[i].ArtifactKind == artifactKind {
			copied[i].ArtifactDigest = strings.TrimSpace(digest)
		}
	}
	return copied
}

// FindEventID returns the first event id for eventType.
func FindEventID(events []Event, eventType string) string {
	for _, event := range events {
		if event.EventType == eventType {
			return event.EventID
		}
	}
	return ""
}

// Expected captures the envelope fields a proof event must link to.
type Expected struct {
	EventType            string
	EventTime            string
	Source               string
	TraceID              string
	ChangeID             string
	Space                string
	TargetSlug           string
	TargetPath           string
	RenderTargetSlug     string
	RenderTargetPath     string
	ArtifactKind         string
	ArtifactDigest       string
	ParentEventID        string
	ParentArtifactKind   string
	ParentArtifactDigest string
}

// ValidateArtifactEvents checks that an artifact carries at least one proof
// event that can be logged and joined back to the artifact.
func ValidateArtifactEvents(events []Event, expected Expected) error {
	if len(events) == 0 {
		return fmt.Errorf("missing proof_events")
	}
	found := false
	for i, event := range events {
		if err := ValidateEvent(event); err != nil {
			return fmt.Errorf("proof_events[%d]: %w", i, err)
		}
		if event.EventType != expected.EventType {
			continue
		}
		found = true
		if err := validateExpected(event, expected); err != nil {
			return fmt.Errorf("proof_events[%d]: %w", i, err)
		}
	}
	if !found {
		return fmt.Errorf("missing proof event_type %q", expected.EventType)
	}
	return nil
}

// ValidateEvent checks log-safety and deterministic identity for one event.
func ValidateEvent(event Event) error {
	if event.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported schema_version %q", event.SchemaVersion)
	}
	if strings.TrimSpace(event.EventID) == "" {
		return fmt.Errorf("missing event_id")
	}
	if expected := EventID(event); event.EventID != expected {
		return fmt.Errorf("event_id mismatch: expected %s, got %s", expected, event.EventID)
	}
	if strings.TrimSpace(event.EventType) == "" {
		return fmt.Errorf("missing event_type")
	}
	if strings.TrimSpace(event.EventTime) == "" {
		return fmt.Errorf("missing event_time")
	}
	if _, err := time.Parse(time.RFC3339, event.EventTime); err != nil {
		return fmt.Errorf("invalid event_time %q", event.EventTime)
	}
	if strings.TrimSpace(event.Source) == "" {
		return fmt.Errorf("missing source")
	}
	if strings.TrimSpace(event.TraceID) == "" {
		return fmt.Errorf("missing trace_id")
	}
	if strings.TrimSpace(event.ArtifactKind) == "" {
		return fmt.Errorf("missing artifact_kind")
	}
	if digest := strings.TrimSpace(event.ArtifactDigest); digest != "" && !strings.HasPrefix(digest, digestAlgorithm+":") {
		return fmt.Errorf("artifact_digest must use %s prefix", digestAlgorithm)
	}
	if digest := strings.TrimSpace(event.ParentArtifactDigest); digest != "" && !strings.HasPrefix(digest, digestAlgorithm+":") {
		return fmt.Errorf("parent_artifact_digest must use %s prefix", digestAlgorithm)
	}
	return nil
}

func validateExpected(event Event, expected Expected) error {
	checks := []struct {
		name string
		got  string
		want string
	}{
		{"event_time", event.EventTime, expected.EventTime},
		{"source", event.Source, expected.Source},
		{"trace_id", event.TraceID, expected.TraceID},
		{"change_id", event.ChangeID, expected.ChangeID},
		{"space", event.Space, expected.Space},
		{"target_slug", event.TargetSlug, expected.TargetSlug},
		{"target_path", event.TargetPath, expected.TargetPath},
		{"render_target_slug", event.RenderTargetSlug, expected.RenderTargetSlug},
		{"render_target_path", event.RenderTargetPath, expected.RenderTargetPath},
		{"artifact_kind", event.ArtifactKind, expected.ArtifactKind},
		{"artifact_digest", event.ArtifactDigest, expected.ArtifactDigest},
	}
	for _, check := range checks {
		if strings.TrimSpace(check.got) != strings.TrimSpace(check.want) {
			return fmt.Errorf("%s mismatch: expected %q, got %q", check.name, check.want, check.got)
		}
	}
	optionalChecks := []struct {
		name string
		got  string
		want string
	}{
		{"parent_event_id", event.ParentEventID, expected.ParentEventID},
		{"parent_artifact_kind", event.ParentArtifactKind, expected.ParentArtifactKind},
		{"parent_artifact_digest", event.ParentArtifactDigest, expected.ParentArtifactDigest},
	}
	for _, check := range optionalChecks {
		if strings.TrimSpace(check.want) == "" {
			continue
		}
		if strings.TrimSpace(check.got) != strings.TrimSpace(check.want) {
			return fmt.Errorf("%s mismatch: expected %q, got %q", check.name, check.want, check.got)
		}
	}
	return nil
}

func copyCounts(in map[string]int) map[string]int {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]int, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
