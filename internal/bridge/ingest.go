package bridge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/confighub/cub-gen/internal/model"
	"github.com/confighub/cub-gen/internal/publish"
)

const (
	ingestSchemaVersion = "cub.confighub.io/governed-wet-ingest/v1"
	defaultEndpointPath = "/api/v1/governed-wet-artifacts:ingest"
	defaultHTTPTimeout  = 30 * time.Second
)

var defaultHTTPClient = &http.Client{Timeout: defaultHTTPTimeout}

// Client defines the endpoint and auth for bridge ingest.
type Client struct {
	BaseURL      string
	BearerToken  string
	EndpointPath string
	HTTPClient   *http.Client
}

// IngestPayload is the bridge request payload for governed WET artifact ingest.
type IngestPayload struct {
	SchemaVersion      string                       `json:"schema_version"`
	Source             string                       `json:"source"`
	Space              string                       `json:"space"`
	TargetSlug         string                       `json:"target_slug"`
	TargetPath         string                       `json:"target_path"`
	RenderTargetSlug   string                       `json:"render_target_slug"`
	Ref                string                       `json:"ref"`
	ChangeID           string                       `json:"change_id"`
	DigestAlgorithm    string                       `json:"digest_algorithm"`
	BundleDigest       string                       `json:"bundle_digest"`
	IdempotencyKey     string                       `json:"idempotency_key"`
	Summary            publish.Summary              `json:"summary"`
	Contracts          []model.GeneratorContract    `json:"contracts"`
	Provenance         []model.ProvenanceRecord     `json:"provenance"`
	InversePlans       []model.InverseTransformPlan `json:"inverse_transform_plans"`
	DryInputs          []model.DryInputRef          `json:"dry_inputs"`
	WetManifestTargets []model.WetManifestTarget    `json:"wet_manifest_targets"`
}

// IngestResult captures bridge ingest response metadata.
type IngestResult struct {
	StatusCode     int    `json:"status_code"`
	ArtifactID     string `json:"artifact_id,omitempty"`
	Status         string `json:"status,omitempty"`
	Idempotent     bool   `json:"idempotent"`
	ChangeID       string `json:"change_id"`
	BundleDigest   string `json:"bundle_digest"`
	IdempotencyKey string `json:"idempotency_key"`
}

type ingestAPIResponse struct {
	ArtifactID string `json:"artifact_id,omitempty"`
	Status     string `json:"status,omitempty"`
	Message    string `json:"message,omitempty"`
}

// BuildIngestPayload validates and maps a change-bundle to ingest payload.
func BuildIngestPayload(bundle publish.ChangeBundle) (IngestPayload, error) {
	if err := publish.VerifyBundle(bundle); err != nil {
		return IngestPayload{}, fmt.Errorf("verify bundle before ingest: %w", err)
	}
	if strings.TrimSpace(bundle.ChangeID) == "" {
		return IngestPayload{}, fmt.Errorf("bridge ingest requires non-empty change_id")
	}

	key := bundle.ChangeID + ":" + bundle.BundleDigest

	return IngestPayload{
		SchemaVersion:      ingestSchemaVersion,
		Source:             bundle.Source,
		Space:              bundle.Space,
		TargetSlug:         bundle.TargetSlug,
		TargetPath:         bundle.TargetPath,
		RenderTargetSlug:   bundle.RenderTargetSlug,
		Ref:                bundle.Ref,
		ChangeID:           bundle.ChangeID,
		DigestAlgorithm:    bundle.DigestAlgorithm,
		BundleDigest:       bundle.BundleDigest,
		IdempotencyKey:     key,
		Summary:            bundle.Summary,
		Contracts:          bundle.Contracts,
		Provenance:         bundle.Provenance,
		InversePlans:       bundle.InversePlans,
		DryInputs:          bundle.DryInputs,
		WetManifestTargets: bundle.WetManifestTargets,
	}, nil
}

// IngestBundle posts a validated bridge payload to ConfigHub ingest endpoint.
// HTTP 409 is treated as idempotent success.
func IngestBundle(ctx context.Context, client Client, bundle publish.ChangeBundle) (IngestResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	payload, err := BuildIngestPayload(bundle)
	if err != nil {
		return IngestResult{}, err
	}

	endpoint, err := resolveEndpoint(client)
	if err != nil {
		return IngestResult{}, err
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return IngestResult{}, fmt.Errorf("marshal ingest payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return IngestResult{}, fmt.Errorf("build ingest request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", payload.IdempotencyKey)
	if token := strings.TrimSpace(client.BearerToken); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	httpClient := client.HTTPClient
	if httpClient == nil {
		httpClient = defaultHTTPClient
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return IngestResult{}, fmt.Errorf("send ingest request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	var decoded ingestAPIResponse
	_ = json.Unmarshal(respBody, &decoded)

	result := IngestResult{
		StatusCode:     resp.StatusCode,
		ArtifactID:     strings.TrimSpace(decoded.ArtifactID),
		Status:         strings.TrimSpace(decoded.Status),
		Idempotent:     resp.StatusCode == http.StatusConflict,
		ChangeID:       payload.ChangeID,
		BundleDigest:   payload.BundleDigest,
		IdempotencyKey: payload.IdempotencyKey,
	}

	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated, http.StatusAccepted:
		if result.Status == "" {
			result.Status = "ingested"
		}
		return result, nil
	case http.StatusConflict:
		if result.Status == "" {
			result.Status = "exists"
		}
		return result, nil
	default:
		msg := strings.TrimSpace(string(respBody))
		if msg == "" {
			msg = "<empty>"
		}
		return IngestResult{}, fmt.Errorf("bridge ingest failed: status=%d body=%s", resp.StatusCode, msg)
	}
}

func resolveEndpoint(client Client) (string, error) {
	base := strings.TrimSpace(client.BaseURL)
	if base == "" {
		return "", fmt.Errorf("bridge client base URL is required")
	}
	parsed, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("parse bridge base URL: %w", err)
	}
	path := strings.TrimSpace(client.EndpointPath)
	if path == "" {
		path = defaultEndpointPath
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	parsed.Path = strings.TrimSuffix(parsed.Path, "/") + path
	return parsed.String(), nil
}
