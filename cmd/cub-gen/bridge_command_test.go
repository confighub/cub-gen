package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	bridgeflow "github.com/confighub/cub-gen/internal/bridge"
	"github.com/confighub/cub-gen/internal/publish"
)

func TestBridgeIngestCommand(t *testing.T) {
	setupAliases(t)

	publishOut, publishErr, err := runWithCapturedIO([]string{"publish", "--space", "platform", "helm", "render-target"})
	if err != nil {
		t.Fatalf("publish returned error: %v\nstderr=%s", err, publishErr)
	}
	if strings.TrimSpace(publishErr) != "" {
		t.Fatalf("expected empty publish stderr, got %q", publishErr)
	}

	var bundle publish.ChangeBundle
	if err := json.Unmarshal([]byte(publishOut), &bundle); err != nil {
		t.Fatalf("unmarshal bundle: %v\noutput=%s", err, publishOut)
	}

	var receivedPath string
	var receivedKey string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		receivedKey = r.Header.Get("Idempotency-Key")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"artifact_id":"wet_art_123","status":"created"}`))
	}))
	defer srv.Close()

	out, stderr, err := runWithCapturedIOAndStdin([]string{"bridge", "ingest", "--in", "-", "--base-url", srv.URL}, publishOut)
	if err != nil {
		t.Fatalf("bridge ingest returned error: %v\nstderr=%s", err, stderr)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if receivedPath != "/api/v1/governed-wet-artifacts:ingest" {
		t.Fatalf("expected ingest path, got %q", receivedPath)
	}

	var got bridgeflow.IngestResult
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal bridge ingest output: %v\noutput=%s", err, out)
	}
	if got.ChangeID != bundle.ChangeID {
		t.Fatalf("expected change_id %q, got %q", bundle.ChangeID, got.ChangeID)
	}
	if got.BundleDigest != bundle.BundleDigest {
		t.Fatalf("expected bundle_digest %q, got %q", bundle.BundleDigest, got.BundleDigest)
	}
	expectedKey := bundle.ChangeID + ":" + bundle.BundleDigest
	if receivedKey != expectedKey {
		t.Fatalf("expected idempotency key %q, got %q", expectedKey, receivedKey)
	}
}

func TestBridgeDecisionLifecycleCommands(t *testing.T) {
	setupAliases(t)

	publishOut, publishErr, err := runWithCapturedIO([]string{"publish", "--space", "platform", "score", "render-target"})
	if err != nil {
		t.Fatalf("publish returned error: %v\nstderr=%s", err, publishErr)
	}
	if strings.TrimSpace(publishErr) != "" {
		t.Fatalf("expected empty publish stderr, got %q", publishErr)
	}

	var bundle publish.ChangeBundle
	if err := json.Unmarshal([]byte(publishOut), &bundle); err != nil {
		t.Fatalf("unmarshal bundle: %v\noutput=%s", err, publishOut)
	}

	attOut, attErr, err := runWithCapturedIOAndStdin([]string{"attest", "--in", "-", "--verifier", "ci-bot"}, publishOut)
	if err != nil {
		t.Fatalf("attest returned error: %v\nstderr=%s", err, attErr)
	}
	if strings.TrimSpace(attErr) != "" {
		t.Fatalf("expected empty attest stderr, got %q", attErr)
	}

	attPath := filepath.Join(t.TempDir(), "attestation.json")
	if writeErr := os.WriteFile(attPath, []byte(attOut), 0o644); writeErr != nil {
		t.Fatalf("write attestation file: %v", writeErr)
	}

	ingestBytes, err := json.Marshal(bridgeflow.IngestResult{
		StatusCode:     http.StatusCreated,
		ArtifactID:     "wet_art_123",
		Status:         "created",
		ChangeID:       bundle.ChangeID,
		BundleDigest:   bundle.BundleDigest,
		IdempotencyKey: bundle.ChangeID + ":" + bundle.BundleDigest,
	})
	if err != nil {
		t.Fatalf("marshal ingest result: %v", err)
	}

	createOut, createErr, err := runWithCapturedIOAndStdin([]string{"bridge", "decision", "create", "--ingest", "-"}, string(ingestBytes))
	if err != nil {
		t.Fatalf("bridge decision create returned error: %v\nstderr=%s", err, createErr)
	}
	if strings.TrimSpace(createErr) != "" {
		t.Fatalf("expected empty decision create stderr, got %q", createErr)
	}

	attachOut, attachErr, err := runWithCapturedIOAndStdin([]string{"bridge", "decision", "attach", "--decision", "-", "--attestation", attPath}, createOut)
	if err != nil {
		t.Fatalf("bridge decision attach returned error: %v\nstderr=%s", err, attachErr)
	}
	if strings.TrimSpace(attachErr) != "" {
		t.Fatalf("expected empty decision attach stderr, got %q", attachErr)
	}

	applyOut, applyErr, err := runWithCapturedIOAndStdin([]string{
		"bridge", "decision", "apply",
		"--decision", "-",
		"--state", "ALLOW",
		"--approved-by", "platform-owner",
		"--reason", "policy checks passed",
	}, attachOut)
	if err != nil {
		t.Fatalf("bridge decision apply returned error: %v\nstderr=%s", err, applyErr)
	}
	if strings.TrimSpace(applyErr) != "" {
		t.Fatalf("expected empty decision apply stderr, got %q", applyErr)
	}

	var rec bridgeflow.DecisionRecord
	if err := json.Unmarshal([]byte(applyOut), &rec); err != nil {
		t.Fatalf("unmarshal applied decision: %v\noutput=%s", err, applyOut)
	}
	if rec.State != bridgeflow.DecisionStateAllow {
		t.Fatalf("expected decision state %q, got %q", bridgeflow.DecisionStateAllow, rec.State)
	}
	if rec.ApprovedBy != "platform-owner" {
		t.Fatalf("expected approved_by platform-owner, got %q", rec.ApprovedBy)
	}
	if rec.DecisionReason != "policy checks passed" {
		t.Fatalf("unexpected decision reason %q", rec.DecisionReason)
	}
}

func TestBridgePromoteLifecycleCommands(t *testing.T) {
	initOut, initErr, err := runWithCapturedIO([]string{
		"bridge", "promote", "init",
		"--change-id", "chg_123",
		"--app-pr-repo", "github.com/confighub/apps",
		"--app-pr-number", "42",
		"--app-pr-url", "https://github.com/confighub/apps/pull/42",
		"--mr-id", "mr_123",
		"--mr-url", "https://confighub.example/mr/123",
	})
	if err != nil {
		t.Fatalf("bridge promote init returned error: %v\nstderr=%s", err, initErr)
	}
	if strings.TrimSpace(initErr) != "" {
		t.Fatalf("expected empty promote init stderr, got %q", initErr)
	}

	governOut, governErr, err := runWithCapturedIOAndStdin([]string{
		"bridge", "promote", "govern",
		"--flow", "-",
		"--state", "ALLOW",
		"--decision-ref", "decision_123",
	}, initOut)
	if err != nil {
		t.Fatalf("bridge promote govern returned error: %v\nstderr=%s", err, governErr)
	}
	if strings.TrimSpace(governErr) != "" {
		t.Fatalf("expected empty promote govern stderr, got %q", governErr)
	}

	verifyOut, verifyErr, err := runWithCapturedIOAndStdin([]string{"bridge", "promote", "verify", "--flow", "-"}, governOut)
	if err != nil {
		t.Fatalf("bridge promote verify returned error: %v\nstderr=%s", err, verifyErr)
	}
	if strings.TrimSpace(verifyErr) != "" {
		t.Fatalf("expected empty promote verify stderr, got %q", verifyErr)
	}

	openOut, openErr, err := runWithCapturedIOAndStdin([]string{
		"bridge", "promote", "open",
		"--flow", "-",
		"--repo", "github.com/confighub/platform-dry",
		"--number", "7",
		"--url", "https://github.com/confighub/platform-dry/pull/7",
	}, verifyOut)
	if err != nil {
		t.Fatalf("bridge promote open returned error: %v\nstderr=%s", err, openErr)
	}
	if strings.TrimSpace(openErr) != "" {
		t.Fatalf("expected empty promote open stderr, got %q", openErr)
	}

	approveOut, approveErr, err := runWithCapturedIOAndStdin([]string{
		"bridge", "promote", "approve",
		"--flow", "-",
		"--by", "platform-owner",
	}, openOut)
	if err != nil {
		t.Fatalf("bridge promote approve returned error: %v\nstderr=%s", err, approveErr)
	}
	if strings.TrimSpace(approveErr) != "" {
		t.Fatalf("expected empty promote approve stderr, got %q", approveErr)
	}

	mergeOut, mergeErr, err := runWithCapturedIOAndStdin([]string{
		"bridge", "promote", "merge",
		"--flow", "-",
		"--by", "platform-owner",
	}, approveOut)
	if err != nil {
		t.Fatalf("bridge promote merge returned error: %v\nstderr=%s", err, mergeErr)
	}
	if strings.TrimSpace(mergeErr) != "" {
		t.Fatalf("expected empty promote merge stderr, got %q", mergeErr)
	}

	var flow bridgeflow.PromotionFlow
	if err := json.Unmarshal([]byte(mergeOut), &flow); err != nil {
		t.Fatalf("unmarshal merge output: %v\noutput=%s", err, mergeOut)
	}
	if flow.State != bridgeflow.PromotionStatePromoted {
		t.Fatalf("expected promote state %q, got %q", bridgeflow.PromotionStatePromoted, flow.State)
	}
	if !flow.PlatformReviewApproved || !flow.PromotionMerged {
		t.Fatalf("expected review + merge gates to be true, got flow=%+v", flow)
	}
}

func TestBridgeDecisionQueryCommand(t *testing.T) {
	record := bridgeflow.DecisionRecord{
		SchemaVersion:     "cub.confighub.io/governed-decision-state/v1",
		Source:            "cub-gen",
		ChangeID:          "chg_123",
		BundleDigest:      "sha256:bundle",
		State:             bridgeflow.DecisionStateIngested,
		AttestationDigest: "sha256:attestation",
		UpdatedAt:         time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC).Format(time.RFC3339),
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/governed-wet-decisions/chg_123" {
			t.Fatalf("unexpected query path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(record)
	}))
	defer srv.Close()

	out, stderr, err := runWithCapturedIO([]string{
		"bridge", "decision", "query",
		"--base-url", srv.URL,
		"--change-id", "chg_123",
	})
	if err != nil {
		t.Fatalf("bridge decision query returned error: %v\nstderr=%s", err, stderr)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var got bridgeflow.DecisionRecord
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal query output: %v\noutput=%s", err, out)
	}
	if got.ChangeID != "chg_123" {
		t.Fatalf("expected change_id chg_123, got %q", got.ChangeID)
	}
}
