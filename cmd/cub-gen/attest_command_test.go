package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAttestFromStdin(t *testing.T) {
	setupAliases(t)

	bundleJSON, err := generateBundleJSON()
	if err != nil {
		t.Fatalf("generate bundle: %v", err)
	}

	out, stderr, err := runWithCapturedIOAndStdin([]string{"attest", "--in", "-", "--verifier", "ci-bot"}, bundleJSON)
	if err != nil {
		t.Fatalf("attest returned error: %v\nstderr=%s", err, stderr)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal attest output: %v\noutput=%s", err, out)
	}
	if got["schema_version"] != "cub.confighub.io/attestation/v1" {
		t.Fatalf("unexpected schema_version: %v", got["schema_version"])
	}
	if got["source"] != "cub-gen" {
		t.Fatalf("unexpected source: %v", got["source"])
	}
	if got["status"] != "verified" {
		t.Fatalf("unexpected status: %v", got["status"])
	}
	if got["verifier"] != "ci-bot" {
		t.Fatalf("unexpected verifier: %v", got["verifier"])
	}
	if got["digest_algorithm"] != "sha256" {
		t.Fatalf("unexpected digest_algorithm: %v", got["digest_algorithm"])
	}
	if v, ok := got["attestation_digest"].(string); !ok || !strings.HasPrefix(v, "sha256:") {
		t.Fatalf("unexpected attestation_digest: %v", got["attestation_digest"])
	}
}

func TestAttestFromStdinSupportedTargets(t *testing.T) {
	setupAliases(t)

	targets := []string{"helm", "score", "spring", "backstage", "no-config-platform", "ops"}
	for _, target := range targets {
		t.Run(target, func(t *testing.T) {
			bundleJSON, err := generateBundleJSONForTarget(target)
			if err != nil {
				t.Fatalf("generate bundle for target %q: %v", target, err)
			}

			out, stderr, err := runWithCapturedIOAndStdin([]string{"attest", "--in", "-", "--verifier", "ci-bot"}, bundleJSON)
			if err != nil {
				t.Fatalf("attest returned error: %v\nstderr=%s", err, stderr)
			}
			if strings.TrimSpace(stderr) != "" {
				t.Fatalf("expected empty stderr, got %q", stderr)
			}

			var got map[string]any
			if err := json.Unmarshal([]byte(out), &got); err != nil {
				t.Fatalf("unmarshal attest output: %v\noutput=%s", err, out)
			}
			if got["status"] != "verified" {
				t.Fatalf("unexpected status for target %q: %v", target, got["status"])
			}
			if got["verifier"] != "ci-bot" {
				t.Fatalf("unexpected verifier for target %q: %v", target, got["verifier"])
			}
			if v, ok := got["attestation_digest"].(string); !ok || !strings.HasPrefix(v, "sha256:") {
				t.Fatalf("unexpected attestation_digest for target %q: %v", target, got["attestation_digest"])
			}
		})
	}
}

func TestAttestFromFileToFile(t *testing.T) {
	setupAliases(t)

	bundleJSON, err := generateBundleJSON()
	if err != nil {
		t.Fatalf("generate bundle: %v", err)
	}
	inPath := filepath.Join(t.TempDir(), "bundle.json")
	if err := os.WriteFile(inPath, []byte(bundleJSON), 0o644); err != nil {
		t.Fatalf("write input bundle: %v", err)
	}
	outPath := filepath.Join(t.TempDir(), "attestation.json")

	_, stderr, err := runWithCapturedIO([]string{"attest", "--in", inPath, "--out", outPath})
	if err != nil {
		t.Fatalf("attest file->file returned error: %v\nstderr=%s", err, stderr)
	}

	b, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read attestation output: %v", err)
	}
	if !strings.Contains(string(b), "\"schema_version\": \"cub.confighub.io/attestation/v1\"") {
		t.Fatalf("unexpected attestation payload: %s", string(b))
	}
}

func TestAttestDetectsTamperedBundle(t *testing.T) {
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

	_, _, err = runWithCapturedIOAndStdin([]string{"attest", "--in", "-"}, string(tamperedBytes))
	if err == nil {
		t.Fatal("expected attest to fail for tampered bundle")
	}
	if !strings.Contains(err.Error(), "bundle digest mismatch") {
		t.Fatalf("expected digest mismatch error, got %q", err.Error())
	}
}
