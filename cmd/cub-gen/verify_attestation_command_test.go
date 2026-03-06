package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVerifyAttestationFromStdin(t *testing.T) {
	setupAliases(t)

	attJSON, _, err := generateAttestationJSON("ci-bot")
	if err != nil {
		t.Fatalf("generate attestation: %v", err)
	}

	out, stderr, err := runWithCapturedIOAndStdin([]string{"verify-attestation", "--in", "-"}, attJSON)
	if err != nil {
		t.Fatalf("verify-attestation returned error: %v\nstderr=%s", err, stderr)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(out, "Attestation verification OK:") {
		t.Fatalf("expected success output, got %q", out)
	}
}

func TestVerifyAttestationAgainstBundleFile(t *testing.T) {
	setupAliases(t)

	attJSON, bundleJSON, err := generateAttestationJSON("ci-bot")
	if err != nil {
		t.Fatalf("generate attestation: %v", err)
	}

	attPath := filepath.Join(t.TempDir(), "attestation.json")
	bundlePath := filepath.Join(t.TempDir(), "bundle.json")
	if err := os.WriteFile(attPath, []byte(attJSON), 0o644); err != nil {
		t.Fatalf("write attestation file: %v", err)
	}
	if err := os.WriteFile(bundlePath, []byte(bundleJSON), 0o644); err != nil {
		t.Fatalf("write bundle file: %v", err)
	}

	out, stderr, err := runWithCapturedIO([]string{"verify-attestation", "--in", attPath, "--bundle", bundlePath})
	if err != nil {
		t.Fatalf("verify-attestation linked returned error: %v\nstderr=%s", err, stderr)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(out, "Attestation verification OK (linked):") {
		t.Fatalf("expected linked success output, got %q", out)
	}
}

func TestVerifyAttestationJSONOutput(t *testing.T) {
	setupAliases(t)

	attJSON, _, err := generateAttestationJSON("ci-bot")
	if err != nil {
		t.Fatalf("generate attestation: %v", err)
	}

	out, stderr, err := runWithCapturedIOAndStdin([]string{"verify-attestation", "--json", "--in", "-"}, attJSON)
	if err != nil {
		t.Fatalf("verify-attestation --json returned error: %v\nstderr=%s", err, stderr)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal verify-attestation json: %v\noutput=%s", err, out)
	}
	if valid, ok := got["valid"].(bool); !ok || !valid {
		t.Fatalf("expected valid=true, got %v", got["valid"])
	}
	if linked, ok := got["linked_bundle_check"].(bool); !ok || linked {
		t.Fatalf("expected linked_bundle_check=false, got %v", got["linked_bundle_check"])
	}
}

func TestVerifyAttestationJSONOutputFirstThreeTargets(t *testing.T) {
	setupAliases(t)

	targets := []string{"helm", "score", "spring"}
	for _, target := range targets {
		t.Run(target, func(t *testing.T) {
			attJSON, _, err := generateAttestationJSONForTarget("ci-bot", target)
			if err != nil {
				t.Fatalf("generate attestation for target %q: %v", target, err)
			}

			out, stderr, err := runWithCapturedIOAndStdin([]string{"verify-attestation", "--json", "--in", "-"}, attJSON)
			if err != nil {
				t.Fatalf("verify-attestation --json returned error: %v\nstderr=%s", err, stderr)
			}
			if strings.TrimSpace(stderr) != "" {
				t.Fatalf("expected empty stderr, got %q", stderr)
			}

			var got map[string]any
			if err := json.Unmarshal([]byte(out), &got); err != nil {
				t.Fatalf("unmarshal verify-attestation json: %v\noutput=%s", err, out)
			}
			if valid, ok := got["valid"].(bool); !ok || !valid {
				t.Fatalf("expected valid=true for target %q, got %v", target, got["valid"])
			}
			if linked, ok := got["linked_bundle_check"].(bool); !ok || linked {
				t.Fatalf("expected linked_bundle_check=false for target %q, got %v", target, got["linked_bundle_check"])
			}
		})
	}
}

func TestVerifyAttestationLinkedJSONOutputFirstThreeTargets(t *testing.T) {
	setupAliases(t)

	targets := []string{"helm", "score", "spring"}
	for _, target := range targets {
		t.Run(target, func(t *testing.T) {
			attJSON, bundleJSON, err := generateAttestationJSONForTarget("ci-bot", target)
			if err != nil {
				t.Fatalf("generate attestation for target %q: %v", target, err)
			}

			attPath := filepath.Join(t.TempDir(), "attestation.json")
			bundlePath := filepath.Join(t.TempDir(), "bundle.json")
			if err := os.WriteFile(attPath, []byte(attJSON), 0o644); err != nil {
				t.Fatalf("write attestation file: %v", err)
			}
			if err := os.WriteFile(bundlePath, []byte(bundleJSON), 0o644); err != nil {
				t.Fatalf("write bundle file: %v", err)
			}

			out, stderr, err := runWithCapturedIO([]string{"verify-attestation", "--json", "--in", attPath, "--bundle", bundlePath})
			if err != nil {
				t.Fatalf("verify-attestation --json linked returned error: %v\nstderr=%s", err, stderr)
			}
			if strings.TrimSpace(stderr) != "" {
				t.Fatalf("expected empty stderr, got %q", stderr)
			}

			var got map[string]any
			if err := json.Unmarshal([]byte(out), &got); err != nil {
				t.Fatalf("unmarshal verify-attestation linked json: %v\noutput=%s", err, out)
			}
			if valid, ok := got["valid"].(bool); !ok || !valid {
				t.Fatalf("expected valid=true for target %q, got %v", target, got["valid"])
			}
			if linked, ok := got["linked_bundle_check"].(bool); !ok || !linked {
				t.Fatalf("expected linked_bundle_check=true for target %q, got %v", target, got["linked_bundle_check"])
			}
		})
	}
}

func TestVerifyAttestationDetectsTamper(t *testing.T) {
	setupAliases(t)

	attJSON, _, err := generateAttestationJSON("ci-bot")
	if err != nil {
		t.Fatalf("generate attestation: %v", err)
	}

	var rec map[string]any
	if err := json.Unmarshal([]byte(attJSON), &rec); err != nil {
		t.Fatalf("unmarshal attestation: %v", err)
	}
	rec["status"] = "tampered"
	tamperedBytes, err := json.Marshal(rec)
	if err != nil {
		t.Fatalf("marshal tampered attestation: %v", err)
	}

	_, _, err = runWithCapturedIOAndStdin([]string{"verify-attestation", "--in", "-"}, string(tamperedBytes))
	if err == nil {
		t.Fatal("expected verify-attestation to fail for tampered attestation")
	}
	if !strings.Contains(err.Error(), "unsupported status") {
		t.Fatalf("expected unsupported status error, got %q", err.Error())
	}
}

func TestVerifyAttestationDetectsBundleLinkMismatch(t *testing.T) {
	setupAliases(t)

	attJSON, bundleJSON, err := generateAttestationJSON("ci-bot")
	if err != nil {
		t.Fatalf("generate attestation: %v", err)
	}

	var bundle map[string]any
	if err := json.Unmarshal([]byte(bundleJSON), &bundle); err != nil {
		t.Fatalf("unmarshal bundle: %v", err)
	}
	bundle["space"] = "tampered"
	tamperedBundleBytes, err := json.Marshal(bundle)
	if err != nil {
		t.Fatalf("marshal tampered bundle: %v", err)
	}

	attPath := filepath.Join(t.TempDir(), "attestation.json")
	bundlePath := filepath.Join(t.TempDir(), "bundle.json")
	if err := os.WriteFile(attPath, []byte(attJSON), 0o644); err != nil {
		t.Fatalf("write attestation file: %v", err)
	}
	if err := os.WriteFile(bundlePath, tamperedBundleBytes, 0o644); err != nil {
		t.Fatalf("write tampered bundle file: %v", err)
	}

	_, _, err = runWithCapturedIO([]string{"verify-attestation", "--in", attPath, "--bundle", bundlePath})
	if err == nil {
		t.Fatal("expected verify-attestation to fail for bundle mismatch")
	}
	if !strings.Contains(err.Error(), "bundle digest mismatch") {
		t.Fatalf("expected bundle digest mismatch error, got %q", err.Error())
	}
}

func generateAttestationJSON(verifier string) (string, string, error) {
	return generateAttestationJSONForTarget(verifier, "helm")
}

func generateAttestationJSONForTarget(verifier, target string) (string, string, error) {
	bundleJSON, err := generateBundleJSONForTarget(target)
	if err != nil {
		return "", "", err
	}
	out, stderr, err := runWithCapturedIOAndStdin([]string{"attest", "--in", "-", "--verifier", verifier}, bundleJSON)
	if err != nil {
		return "", "", err
	}
	if strings.TrimSpace(stderr) != "" {
		return "", "", fmt.Errorf("unexpected stderr from attest: %s", stderr)
	}
	return out, bundleJSON, nil
}
