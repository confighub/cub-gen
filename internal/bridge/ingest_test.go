package bridge

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	gitopsflow "github.com/confighub/cub-gen/internal/gitops"
	"github.com/confighub/cub-gen/internal/model"
	"github.com/confighub/cub-gen/internal/publish"
)

func TestIngestBundleCreated(t *testing.T) {
	bundle := sampleBundle("chg_1")
	var gotPayload IngestPayload
	var gotIdempotencyKey string
	var gotAuth string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != defaultEndpointPath {
			t.Fatalf("expected endpoint %s, got %s", defaultEndpointPath, r.URL.Path)
		}
		gotIdempotencyKey = r.Header.Get("Idempotency-Key")
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotPayload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"artifact_id":"wet_art_123","status":"created"}`))
	}))
	defer srv.Close()

	res, err := IngestBundle(context.Background(), Client{
		BaseURL:     srv.URL,
		BearerToken: "token-123",
	}, bundle)
	if err != nil {
		t.Fatalf("IngestBundle returned error: %v", err)
	}

	expectedKey := bundle.ChangeID + ":" + bundle.BundleDigest
	if gotIdempotencyKey != expectedKey {
		t.Fatalf("expected idempotency key %q, got %q", expectedKey, gotIdempotencyKey)
	}
	if gotAuth != "Bearer token-123" {
		t.Fatalf("expected bearer auth header, got %q", gotAuth)
	}
	if gotPayload.SchemaVersion != ingestSchemaVersion {
		t.Fatalf("expected payload schema %q, got %q", ingestSchemaVersion, gotPayload.SchemaVersion)
	}
	if gotPayload.ChangeID != bundle.ChangeID {
		t.Fatalf("expected payload change_id %q, got %q", bundle.ChangeID, gotPayload.ChangeID)
	}
	if gotPayload.BundleDigest != bundle.BundleDigest {
		t.Fatalf("expected payload bundle_digest %q, got %q", bundle.BundleDigest, gotPayload.BundleDigest)
	}
	if len(gotPayload.Contracts) != 1 || len(gotPayload.Provenance) != 1 || len(gotPayload.InversePlans) != 1 {
		t.Fatalf("expected triple payload count of 1, got contracts=%d provenance=%d inverse=%d", len(gotPayload.Contracts), len(gotPayload.Provenance), len(gotPayload.InversePlans))
	}
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("expected status code %d, got %d", http.StatusCreated, res.StatusCode)
	}
	if res.ArtifactID != "wet_art_123" || res.Status != "created" || res.Idempotent {
		t.Fatalf("unexpected ingest result: %+v", res)
	}
}

func TestIngestBundleConflictIsIdempotent(t *testing.T) {
	bundle := sampleBundle("chg_1")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"artifact_id":"wet_art_123","status":"exists"}`))
	}))
	defer srv.Close()

	res, err := IngestBundle(context.Background(), Client{BaseURL: srv.URL}, bundle)
	if err != nil {
		t.Fatalf("IngestBundle returned error: %v", err)
	}
	if !res.Idempotent {
		t.Fatalf("expected idempotent=true on 409, got %+v", res)
	}
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("expected status code %d, got %d", http.StatusConflict, res.StatusCode)
	}
}

func TestIngestBundleRejectsInvalidBundleDigest(t *testing.T) {
	bundle := sampleBundle("chg_1")
	expectedDigest := bundle.BundleDigest
	bundle.BundleDigest = "sha256:tampered"

	_, err := IngestBundle(context.Background(), Client{BaseURL: "https://example.local"}, bundle)
	if err == nil {
		t.Fatal("expected bundle verification error, got nil")
	}
	if got := err.Error(); got != "verify bundle before ingest: bundle digest mismatch: expected "+expectedDigest+", got sha256:tampered" {
		t.Fatalf("expected deterministic bundle digest error, got %q", got)
	}
}

func TestBuildIngestPayloadRequiresChangeID(t *testing.T) {
	bundle := sampleBundle("")

	_, err := BuildIngestPayload(bundle)
	if err == nil {
		t.Fatal("expected change_id validation error, got nil")
	}
	expected := "bridge ingest requires non-empty change_id"
	if err.Error() != expected {
		t.Fatalf("expected %q, got %q", expected, err.Error())
	}
}

func sampleBundle(changeID string) publish.ChangeBundle {
	importResult := gitopsflow.ImportFlowResult{
		Space:            "platform",
		TargetSlug:       "repo",
		TargetPath:       "/tmp/repo",
		RenderTargetSlug: "render",
		Ref:              "main",
		Discovered: []gitopsflow.DiscoveredResource{
			{
				GeneratorID:      "gen_1",
				GeneratorProfile: "helm-paas",
				ResourceName:     "payments-api",
				ResourceKind:     "HelmRelease",
				ResourceType:     "helm.toolkit.fluxcd.io/v2/HelmRelease",
				GeneratorKind:    "helm",
			},
		},
		Contracts: []model.GeneratorContract{
			{
				SchemaVersion: "cub.confighub.io/generator-contract/v1",
				GeneratorID:   "gen_1",
				Name:          "payments-api",
				Kind:          "helm",
				Profile:       "helm-paas",
				Version:       "0.2.0",
				SourceRepo:    "/tmp/repo",
				SourceRef:     "main",
				SourcePath:    "",
				Inputs: []model.GeneratorInput{
					{Name: "input_01", SchemaRef: "https://json.schemastore.org/chart", Required: true},
				},
				OutputFormat:  "kubernetes/yaml",
				Transport:     "oci+git",
				Capabilities:  []string{"render-manifests"},
				Deterministic: true,
			},
		},
		Provenance: []model.ProvenanceRecord{
			{
				SchemaVersion:    "cub.confighub.io/provenance/v1",
				ProvenanceID:     "prov_1",
				ChangeID:         changeID,
				GeneratorID:      "gen_1",
				GeneratorName:    "payments-api",
				GeneratorProfile: "helm-paas",
				Version:          "0.2.0",
				InputDigest:      "sha256:input",
				Sources: []model.SourceRef{
					{Role: "generator-input", URI: "git+file:///tmp/repo#main:Chart.yaml", Revision: "main", Path: "Chart.yaml"},
				},
				Outputs: []model.OutputRef{
					{Role: "rendered-manifests", URI: "oci://example.local/platform/payments-api:latest", Digest: "sha256:output"},
				},
				FieldOriginMap: []model.FieldOrigin{
					{
						DryPath:    "values.image.tag",
						WetPath:    "Deployment/spec/template/spec/containers[0]/image",
						SourcePath: "values.yaml",
						Transform:  "helm-template",
						Confidence: 0.86,
					},
				},
				InverseEditPointers: []model.InverseEditPointer{
					{
						WetPath:    "Deployment/spec/template/spec/containers[0]/image",
						DryPath:    "values.image.tag",
						Owner:      "app-team",
						EditHint:   "Edit chart values file and keep chart template unchanged.",
						Confidence: 0.86,
					},
				},
				RenderedAt: "2026-03-06T00:00:00Z",
			},
		},
		InversePlans: []model.InverseTransformPlan{
			{
				SchemaVersion: "cub.confighub.io/inverse-transform-plan/v1",
				PlanID:        "inv_1",
				ChangeID:      changeID,
				SourceKind:    "helm",
				TargetUnitID:  "dry_1",
				Status:        "draft",
				Patches: []model.InversePatch{
					{
						Operation:      "replace",
						DryPath:        "values.image.tag",
						WetPath:        "Deployment/spec/template/spec/containers[0]/image",
						EditableBy:     "app-team",
						Confidence:     0.86,
						RequiresReview: false,
						Reason:         "Container image tag maps cleanly to helm values.",
					},
				},
				CreatedAt: "2026-03-06T00:00:00Z",
			},
		},
		DryInputs: []model.DryInputRef{
			{GeneratorID: "gen_1", Profile: "helm-paas", Role: "chart", Owner: "platform-engineer", Path: "Chart.yaml", Required: true},
		},
		WetManifestTargets: []model.WetManifestTarget{
			{GeneratorID: "gen_1", Kind: "HelmRelease", Name: "payments-api", Owner: "platform-runtime", Namespace: "apps"},
		},
	}
	return publish.BuildBundleAt(importResult, time.Date(2026, 3, 6, 0, 0, 0, 0, time.UTC))
}
