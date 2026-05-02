package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProofEventsFromBundleJSON(t *testing.T) {
	setupAliases(t)

	bundleJSON, err := generateBundleJSON()
	if err != nil {
		t.Fatalf("generate bundle: %v", err)
	}

	out, stderr, err := runWithCapturedIOAndStdin([]string{"proof", "events", "--in", "-"}, bundleJSON)
	if err != nil {
		t.Fatalf("proof events returned error: %v\nstderr=%s", err, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal proof events output: %v\noutput=%s", err, out)
	}
	if got["schema_version"] != "cub.confighub.io/proof-log/v1" {
		t.Fatalf("unexpected schema_version: %v", got["schema_version"])
	}
	if got["source_artifact_kind"] != "change_bundle" {
		t.Fatalf("unexpected source artifact kind: %v", got["source_artifact_kind"])
	}
	if got["event_count"] != float64(1) {
		t.Fatalf("expected one proof event, got %v", got["event_count"])
	}
	events, ok := got["events"].([]any)
	if !ok || len(events) != 1 {
		t.Fatalf("expected one event, got %v", got["events"])
	}
	event, ok := events[0].(map[string]any)
	if !ok {
		t.Fatalf("expected event object, got %v", events[0])
	}
	if event["event_type"] != "change_bundle.published" {
		t.Fatalf("unexpected event_type: %v", event["event_type"])
	}
	if event["trace_id"] != got["trace_id"] {
		t.Fatalf("expected event trace_id to match log trace_id")
	}
}

func TestProofEventsNDJSONFromAttestationLinked(t *testing.T) {
	setupAliases(t)

	attJSON, bundleJSON, err := generateAttestationJSON("ci-bot")
	if err != nil {
		t.Fatalf("generate attestation: %v", err)
	}

	dir := t.TempDir()
	attPath := filepath.Join(dir, "attestation.json")
	bundlePath := filepath.Join(dir, "bundle.json")
	if err := os.WriteFile(attPath, []byte(attJSON), 0o644); err != nil {
		t.Fatalf("write attestation: %v", err)
	}
	if err := os.WriteFile(bundlePath, []byte(bundleJSON), 0o644); err != nil {
		t.Fatalf("write bundle: %v", err)
	}

	out, stderr, err := runWithCapturedIO([]string{"proof", "events", "--in", attPath, "--bundle", bundlePath, "--ndjson"})
	if err != nil {
		t.Fatalf("proof events ndjson returned error: %v\nstderr=%s", err, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected one NDJSON line, got %d: %q", len(lines), out)
	}
	var event map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &event); err != nil {
		t.Fatalf("unmarshal NDJSON event: %v\nline=%s", err, lines[0])
	}
	if event["event_type"] != "attestation.verified" {
		t.Fatalf("unexpected event_type: %v", event["event_type"])
	}
	if event["parent_artifact_kind"] != "change_bundle" {
		t.Fatalf("unexpected parent_artifact_kind: %v", event["parent_artifact_kind"])
	}
}

func TestProofEventsRejectsTamperedBundle(t *testing.T) {
	setupAliases(t)

	bundleJSON, err := generateBundleJSON()
	if err != nil {
		t.Fatalf("generate bundle: %v", err)
	}
	var bundle map[string]any
	if err := json.Unmarshal([]byte(bundleJSON), &bundle); err != nil {
		t.Fatalf("unmarshal bundle: %v", err)
	}
	bundle["space"] = "tampered"
	tamperedBytes, err := json.Marshal(bundle)
	if err != nil {
		t.Fatalf("marshal tampered bundle: %v", err)
	}

	_, _, err = runWithCapturedIOAndStdin([]string{"proof", "events", "--in", "-"}, string(tamperedBytes))
	if err == nil {
		t.Fatal("expected proof events to reject tampered bundle")
	}
	if !strings.Contains(err.Error(), "bundle digest mismatch") {
		t.Fatalf("expected bundle digest mismatch, got %v", err)
	}
}
