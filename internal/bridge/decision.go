package bridge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/confighub/cub-gen/internal/attest"
)

const (
	decisionSchemaVersion  = "cub.confighub.io/governed-decision-state/v1"
	decisionSource         = "cub-gen"
	defaultDecisionPath    = "/api/v1/governed-wet-decisions"
	digestPrefix           = "sha256:"
	decisionStatusIngested = DecisionState("INGESTED")
	decisionStatusAttested = DecisionState("ATTESTED")
	decisionStatusAllow    = DecisionState("ALLOW")
	decisionStatusEscalate = DecisionState("ESCALATE")
	decisionStatusBlock    = DecisionState("BLOCK")
)

var ErrDecisionNotFound = errors.New("decision not found")

type DecisionState string

const (
	DecisionStateIngested DecisionState = decisionStatusIngested
	DecisionStateAttested DecisionState = decisionStatusAttested
	DecisionStateAllow    DecisionState = decisionStatusAllow
	DecisionStateEscalate DecisionState = decisionStatusEscalate
	DecisionStateBlock    DecisionState = decisionStatusBlock
)

// DecisionRecord captures governed decision state for a bridge change_id.
type DecisionRecord struct {
	SchemaVersion       string        `json:"schema_version"`
	Source              string        `json:"source"`
	ChangeID            string        `json:"change_id"`
	BundleDigest        string        `json:"bundle_digest"`
	ArtifactID          string        `json:"artifact_id,omitempty"`
	IdempotencyKey      string        `json:"idempotency_key,omitempty"`
	State               DecisionState `json:"state"`
	AttestationDigest   string        `json:"attestation_digest,omitempty"`
	AttestationVerifier string        `json:"attestation_verifier,omitempty"`
	AttestedAt          string        `json:"attested_at,omitempty"`
	PolicyDecisionRef   string        `json:"policy_decision_ref,omitempty"`
	ApprovedBy          string        `json:"approved_by,omitempty"`
	DecisionReason      string        `json:"decision_reason,omitempty"`
	DecidedAt           string        `json:"decided_at,omitempty"`
	UpdatedAt           string        `json:"updated_at"`
}

// DecisionRequest applies a terminal governed decision.
type DecisionRequest struct {
	State             DecisionState `json:"state"`
	PolicyDecisionRef string        `json:"policy_decision_ref,omitempty"`
	ApprovedBy        string        `json:"approved_by,omitempty"`
	DecisionReason    string        `json:"decision_reason"`
}

// DecisionClient holds bridge API connection settings for decision queries.
type DecisionClient struct {
	BaseURL      string
	BearerToken  string
	EndpointPath string
	HTTPClient   *http.Client
}

// NewDecisionRecord initializes decision state from a successful ingest result.
func NewDecisionRecord(ingest IngestResult, at time.Time) (DecisionRecord, error) {
	changeID := strings.TrimSpace(ingest.ChangeID)
	if changeID == "" {
		return DecisionRecord{}, fmt.Errorf("decision record requires non-empty change_id")
	}
	bundleDigest := strings.TrimSpace(ingest.BundleDigest)
	if bundleDigest == "" {
		return DecisionRecord{}, fmt.Errorf("decision record requires non-empty bundle_digest")
	}
	if !strings.HasPrefix(bundleDigest, digestPrefix) {
		return DecisionRecord{}, fmt.Errorf("decision record bundle_digest must use %s prefix", digestPrefix)
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}

	rec := DecisionRecord{
		SchemaVersion:  decisionSchemaVersion,
		Source:         decisionSource,
		ChangeID:       changeID,
		BundleDigest:   bundleDigest,
		ArtifactID:     strings.TrimSpace(ingest.ArtifactID),
		IdempotencyKey: strings.TrimSpace(ingest.IdempotencyKey),
		State:          DecisionStateIngested,
		UpdatedAt:      at.UTC().Format(time.RFC3339),
	}
	return rec, ValidateDecisionRecord(rec)
}

// AttachAttestation links verified attestation evidence to a decision record.
func AttachAttestation(rec DecisionRecord, attestation attest.Record, at time.Time) (DecisionRecord, error) {
	if err := ValidateDecisionRecord(rec); err != nil {
		return DecisionRecord{}, err
	}
	if rec.State != DecisionStateIngested && rec.State != DecisionStateAttested {
		return DecisionRecord{}, fmt.Errorf("cannot attach attestation after terminal decision state %q", rec.State)
	}
	if err := attest.VerifyRecord(attestation); err != nil {
		return DecisionRecord{}, fmt.Errorf("verify attestation before decision link: %w", err)
	}
	if strings.TrimSpace(attestation.BundleDigest) != rec.BundleDigest {
		return DecisionRecord{}, fmt.Errorf("attestation bundle digest mismatch: decision=%s attestation=%s", rec.BundleDigest, strings.TrimSpace(attestation.BundleDigest))
	}
	if attChangeID := strings.TrimSpace(attestation.ChangeID); attChangeID != "" && attChangeID != rec.ChangeID {
		return DecisionRecord{}, fmt.Errorf("attestation change_id mismatch: decision=%s attestation=%s", rec.ChangeID, attChangeID)
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}

	rec.State = DecisionStateAttested
	rec.AttestationDigest = strings.TrimSpace(attestation.AttestationDigest)
	rec.AttestationVerifier = strings.TrimSpace(attestation.Verifier)
	rec.AttestedAt = at.UTC().Format(time.RFC3339)
	rec.UpdatedAt = rec.AttestedAt
	return rec, ValidateDecisionRecord(rec)
}

// ApplyDecision enforces explicit authority semantics and terminal decisions.
func ApplyDecision(rec DecisionRecord, req DecisionRequest, at time.Time) (DecisionRecord, error) {
	if err := ValidateDecisionRecord(rec); err != nil {
		return DecisionRecord{}, err
	}
	if rec.State != DecisionStateAttested {
		return DecisionRecord{}, fmt.Errorf("decision transition requires attested state, got %q", rec.State)
	}
	if !isTerminalDecision(req.State) {
		return DecisionRecord{}, fmt.Errorf("unsupported decision state %q", req.State)
	}
	approvedBy := strings.TrimSpace(req.ApprovedBy)
	policyRef := strings.TrimSpace(req.PolicyDecisionRef)
	if approvedBy == "" && policyRef == "" {
		return DecisionRecord{}, fmt.Errorf("explicit decision authority required: set approved_by or policy_decision_ref")
	}
	reason := strings.TrimSpace(req.DecisionReason)
	if reason == "" {
		return DecisionRecord{}, fmt.Errorf("decision_reason is required")
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}
	rec.State = req.State
	rec.ApprovedBy = approvedBy
	rec.PolicyDecisionRef = policyRef
	rec.DecisionReason = reason
	rec.DecidedAt = at.UTC().Format(time.RFC3339)
	rec.UpdatedAt = rec.DecidedAt
	return rec, ValidateDecisionRecord(rec)
}

// ValidateDecisionRecord enforces decision contract invariants.
func ValidateDecisionRecord(rec DecisionRecord) error {
	if rec.SchemaVersion != decisionSchemaVersion {
		return fmt.Errorf("unsupported schema_version %q", rec.SchemaVersion)
	}
	if rec.Source != decisionSource {
		return fmt.Errorf("unsupported source %q", rec.Source)
	}
	if strings.TrimSpace(rec.ChangeID) == "" {
		return fmt.Errorf("missing change_id")
	}
	if strings.TrimSpace(rec.BundleDigest) == "" {
		return fmt.Errorf("missing bundle_digest")
	}
	if !strings.HasPrefix(rec.BundleDigest, digestPrefix) {
		return fmt.Errorf("bundle_digest must use %s prefix", digestPrefix)
	}
	switch rec.State {
	case DecisionStateIngested, DecisionStateAttested, DecisionStateAllow, DecisionStateEscalate, DecisionStateBlock:
	default:
		return fmt.Errorf("unsupported state %q", rec.State)
	}
	if rec.State != DecisionStateIngested {
		if strings.TrimSpace(rec.AttestationDigest) == "" {
			return fmt.Errorf("state %q requires attestation_digest linkage", rec.State)
		}
		if !strings.HasPrefix(rec.AttestationDigest, digestPrefix) {
			return fmt.Errorf("attestation_digest must use %s prefix", digestPrefix)
		}
	}
	if isTerminalDecision(rec.State) {
		if strings.TrimSpace(rec.DecidedAt) == "" {
			return fmt.Errorf("state %q requires decided_at", rec.State)
		}
		if strings.TrimSpace(rec.DecisionReason) == "" {
			return fmt.Errorf("state %q requires decision_reason", rec.State)
		}
		if strings.TrimSpace(rec.ApprovedBy) == "" && strings.TrimSpace(rec.PolicyDecisionRef) == "" {
			return fmt.Errorf("state %q requires approved_by or policy_decision_ref", rec.State)
		}
	}
	if strings.TrimSpace(rec.UpdatedAt) == "" {
		return fmt.Errorf("missing updated_at")
	}
	return nil
}

// QueryDecisionByChangeID resolves governed decision state by change_id.
func QueryDecisionByChangeID(ctx context.Context, client DecisionClient, changeID string) (DecisionRecord, error) {
	id := strings.TrimSpace(changeID)
	if id == "" {
		return DecisionRecord{}, fmt.Errorf("query decision requires non-empty change_id")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	endpoint, err := resolveDecisionEndpoint(client, id)
	if err != nil {
		return DecisionRecord{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return DecisionRecord{}, fmt.Errorf("build decision query request: %w", err)
	}
	if token := strings.TrimSpace(client.BearerToken); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	httpClient := client.HTTPClient
	if httpClient == nil {
		httpClient = defaultHTTPClient
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return DecisionRecord{}, fmt.Errorf("send decision query request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	switch resp.StatusCode {
	case http.StatusOK:
		// continue
	case http.StatusNotFound:
		return DecisionRecord{}, fmt.Errorf("%w: change_id=%s", ErrDecisionNotFound, id)
	default:
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			msg = "<empty>"
		}
		return DecisionRecord{}, fmt.Errorf("decision query failed: status=%d body=%s", resp.StatusCode, msg)
	}

	var rec DecisionRecord
	if err := json.Unmarshal(body, &rec); err != nil {
		return DecisionRecord{}, fmt.Errorf("parse decision query response: %w", err)
	}
	if err := ValidateDecisionRecord(rec); err != nil {
		return DecisionRecord{}, fmt.Errorf("invalid decision query response: %w", err)
	}
	if rec.ChangeID != id {
		return DecisionRecord{}, fmt.Errorf("decision response change_id mismatch: expected %s, got %s", id, rec.ChangeID)
	}
	return rec, nil
}

func resolveDecisionEndpoint(client DecisionClient, changeID string) (string, error) {
	base := strings.TrimSpace(client.BaseURL)
	if base == "" {
		return "", fmt.Errorf("decision client base URL is required")
	}
	parsed, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("parse decision base URL: %w", err)
	}
	basePath := strings.TrimSpace(client.EndpointPath)
	if basePath == "" {
		basePath = defaultDecisionPath
	}
	if !strings.HasPrefix(basePath, "/") {
		basePath = "/" + basePath
	}
	parsed.Path = path.Clean(strings.TrimSuffix(parsed.Path, "/") + basePath + "/" + url.PathEscape(changeID))
	return parsed.String(), nil
}

func isTerminalDecision(state DecisionState) bool {
	return state == DecisionStateAllow || state == DecisionStateEscalate || state == DecisionStateBlock
}
