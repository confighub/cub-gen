package bridge

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/confighub/cub-gen/internal/attest"
)

func TestNewDecisionRecordFromIngest(t *testing.T) {
	bundle := sampleBundle("chg_1")
	ingest := IngestResult{
		StatusCode:     http.StatusCreated,
		ArtifactID:     "wet_art_123",
		Status:         "created",
		ChangeID:       bundle.ChangeID,
		BundleDigest:   bundle.BundleDigest,
		IdempotencyKey: bundle.ChangeID + ":" + bundle.BundleDigest,
	}
	ts := time.Date(2026, 3, 6, 10, 0, 0, 0, time.UTC)

	rec, err := NewDecisionRecord(ingest, ts)
	if err != nil {
		t.Fatalf("NewDecisionRecord returned error: %v", err)
	}
	if rec.SchemaVersion != decisionSchemaVersion {
		t.Fatalf("expected schema %q, got %q", decisionSchemaVersion, rec.SchemaVersion)
	}
	if rec.Source != decisionSource {
		t.Fatalf("expected source %q, got %q", decisionSource, rec.Source)
	}
	if rec.State != DecisionStateIngested {
		t.Fatalf("expected state %q, got %q", DecisionStateIngested, rec.State)
	}
	if rec.ChangeID != bundle.ChangeID || rec.BundleDigest != bundle.BundleDigest {
		t.Fatalf("unexpected decision identity linkage: %+v", rec)
	}
	if rec.UpdatedAt != ts.Format(time.RFC3339) {
		t.Fatalf("expected updated_at %q, got %q", ts.Format(time.RFC3339), rec.UpdatedAt)
	}
}

func TestAttachAttestationAndAllowDecision(t *testing.T) {
	bundle := sampleBundle("chg_1")
	att, err := attest.BuildAt(bundle, time.Date(2026, 3, 6, 10, 1, 0, 0, time.UTC), "ci-bot")
	if err != nil {
		t.Fatalf("build attestation: %v", err)
	}
	rec, err := NewDecisionRecord(IngestResult{
		ChangeID:       bundle.ChangeID,
		BundleDigest:   bundle.BundleDigest,
		ArtifactID:     "wet_art_123",
		IdempotencyKey: bundle.ChangeID + ":" + bundle.BundleDigest,
	}, time.Date(2026, 3, 6, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("new decision: %v", err)
	}

	attested, err := AttachAttestation(rec, att, time.Date(2026, 3, 6, 10, 2, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("AttachAttestation returned error: %v", err)
	}
	if attested.State != DecisionStateAttested {
		t.Fatalf("expected state %q, got %q", DecisionStateAttested, attested.State)
	}
	if attested.AttestationDigest != att.AttestationDigest {
		t.Fatalf("expected attestation digest %q, got %q", att.AttestationDigest, attested.AttestationDigest)
	}

	decided, err := ApplyDecision(attested, DecisionRequest{
		State:          DecisionStateAllow,
		ApprovedBy:     "platform-approver",
		DecisionReason: "policy checks passed",
	}, time.Date(2026, 3, 6, 10, 3, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("ApplyDecision returned error: %v", err)
	}
	if decided.State != DecisionStateAllow {
		t.Fatalf("expected state %q, got %q", DecisionStateAllow, decided.State)
	}
	if decided.DecidedAt == "" || decided.DecisionReason == "" {
		t.Fatalf("expected terminal decision fields to be set: %+v", decided)
	}
}

func TestApplyDecisionRequiresExplicitAuthority(t *testing.T) {
	bundle := sampleBundle("chg_1")
	att, err := attest.BuildAt(bundle, time.Date(2026, 3, 6, 10, 1, 0, 0, time.UTC), "ci-bot")
	if err != nil {
		t.Fatalf("build attestation: %v", err)
	}
	rec, err := NewDecisionRecord(IngestResult{
		ChangeID:       bundle.ChangeID,
		BundleDigest:   bundle.BundleDigest,
		IdempotencyKey: bundle.ChangeID + ":" + bundle.BundleDigest,
	}, time.Date(2026, 3, 6, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("new decision: %v", err)
	}
	rec, err = AttachAttestation(rec, att, time.Date(2026, 3, 6, 10, 2, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("attach attestation: %v", err)
	}

	_, err = ApplyDecision(rec, DecisionRequest{
		State:          DecisionStateAllow,
		DecisionReason: "missing authority",
	}, time.Date(2026, 3, 6, 10, 3, 0, 0, time.UTC))
	if err == nil {
		t.Fatal("expected explicit authority error, got nil")
	}
	expected := "explicit decision authority required: set approved_by or policy_decision_ref"
	if err.Error() != expected {
		t.Fatalf("expected %q, got %q", expected, err.Error())
	}
}

func TestApplyDecisionRequiresAttestedState(t *testing.T) {
	bundle := sampleBundle("chg_1")
	rec, err := NewDecisionRecord(IngestResult{
		ChangeID:       bundle.ChangeID,
		BundleDigest:   bundle.BundleDigest,
		IdempotencyKey: bundle.ChangeID + ":" + bundle.BundleDigest,
	}, time.Date(2026, 3, 6, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("new decision: %v", err)
	}

	_, err = ApplyDecision(rec, DecisionRequest{
		State:          DecisionStateAllow,
		ApprovedBy:     "platform-approver",
		DecisionReason: "policy checks passed",
	}, time.Date(2026, 3, 6, 10, 3, 0, 0, time.UTC))
	if err == nil {
		t.Fatal("expected attested-state gate error, got nil")
	}
	expected := `decision transition requires attested state, got "INGESTED"`
	if err.Error() != expected {
		t.Fatalf("expected %q, got %q", expected, err.Error())
	}
}

func TestQueryDecisionByChangeID(t *testing.T) {
	bundle := sampleBundle("chg_1")
	att, err := attest.BuildAt(bundle, time.Date(2026, 3, 6, 10, 1, 0, 0, time.UTC), "ci-bot")
	if err != nil {
		t.Fatalf("build attestation: %v", err)
	}
	ingested, err := NewDecisionRecord(IngestResult{
		ChangeID:       bundle.ChangeID,
		BundleDigest:   bundle.BundleDigest,
		IdempotencyKey: bundle.ChangeID + ":" + bundle.BundleDigest,
	}, time.Date(2026, 3, 6, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("new decision: %v", err)
	}
	attested, err := AttachAttestation(ingested, att, time.Date(2026, 3, 6, 10, 2, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("attach attestation: %v", err)
	}
	record, err := ApplyDecision(attested, DecisionRequest{
		State:             DecisionStateEscalate,
		PolicyDecisionRef: "policy:123",
		DecisionReason:    "requires SRE approval",
	}, time.Date(2026, 3, 6, 10, 3, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("apply decision: %v", err)
	}

	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != defaultDecisionPath+"/chg_1" {
			t.Fatalf("expected path %s, got %s", defaultDecisionPath+"/chg_1", r.URL.Path)
		}
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(record)
	}))
	defer srv.Close()

	out, err := QueryDecisionByChangeID(context.Background(), DecisionClient{
		BaseURL:     srv.URL,
		BearerToken: "token-123",
	}, "chg_1")
	if err != nil {
		t.Fatalf("QueryDecisionByChangeID returned error: %v", err)
	}
	if gotAuth != "Bearer token-123" {
		t.Fatalf("expected bearer auth header, got %q", gotAuth)
	}
	if out.ChangeID != "chg_1" || out.State != DecisionStateEscalate {
		t.Fatalf("unexpected decision query response: %+v", out)
	}
}

func TestQueryDecisionByChangeIDNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("not found"))
	}))
	defer srv.Close()

	_, err := QueryDecisionByChangeID(context.Background(), DecisionClient{BaseURL: srv.URL}, "chg_404")
	if err == nil {
		t.Fatal("expected not found error, got nil")
	}
	if !errors.Is(err, ErrDecisionNotFound) {
		t.Fatalf("expected ErrDecisionNotFound, got %v", err)
	}
}

func TestQueryDecisionByChangeIDRejectsMismatchedResponse(t *testing.T) {
	record := DecisionRecord{
		SchemaVersion:     decisionSchemaVersion,
		Source:            decisionSource,
		ChangeID:          "wrong",
		BundleDigest:      "sha256:digest",
		State:             DecisionStateIngested,
		AttestationDigest: "sha256:att",
		UpdatedAt:         "2026-03-06T10:00:00Z",
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(record)
	}))
	defer srv.Close()

	_, err := QueryDecisionByChangeID(context.Background(), DecisionClient{BaseURL: srv.URL}, "chg_1")
	if err == nil {
		t.Fatal("expected mismatch error, got nil")
	}
	if !strings.Contains(err.Error(), "decision response change_id mismatch") {
		t.Fatalf("expected mismatch error, got %v", err)
	}
}
