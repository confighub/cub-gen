package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVerifyFromFile(t *testing.T) {
	setupAliases(t)

	bundleJSON, err := generateBundleJSON()
	if err != nil {
		t.Fatalf("generate bundle: %v", err)
	}

	inPath := filepath.Join(t.TempDir(), "bundle.json")
	if err := os.WriteFile(inPath, []byte(bundleJSON), 0o644); err != nil {
		t.Fatalf("write bundle json: %v", err)
	}

	out, stderr, err := runWithCapturedIO([]string{"verify", "--in", inPath})
	if err != nil {
		t.Fatalf("verify returned error: %v\nstderr=%s", err, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(out, "Bundle verification OK:") {
		t.Fatalf("expected verification OK output, got %q", out)
	}
}

func TestVerifyFromStdinJSONOutput(t *testing.T) {
	setupAliases(t)

	bundleJSON, err := generateBundleJSON()
	if err != nil {
		t.Fatalf("generate bundle: %v", err)
	}

	out, stderr, err := runWithCapturedIOAndStdin([]string{"verify", "--json", "--in", "-"}, bundleJSON)
	if err != nil {
		t.Fatalf("verify stdin returned error: %v\nstderr=%s", err, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal verify output: %v\noutput=%s", err, out)
	}
	if valid, ok := got["valid"].(bool); !ok || !valid {
		t.Fatalf("expected valid=true, got %v", got["valid"])
	}
	if got["digest_algorithm"] != "sha256" {
		t.Fatalf("unexpected digest_algorithm: %v", got["digest_algorithm"])
	}
}

func TestVerifyFromStdinJSONOutputSupportedTargets(t *testing.T) {
	setupAliases(t)

	targets := []string{"helm", "score", "spring", "backstage", "ably", "ops"}
	for _, target := range targets {
		t.Run(target, func(t *testing.T) {
			bundleJSON, err := generateBundleJSONForTarget(target)
			if err != nil {
				t.Fatalf("generate bundle for target %q: %v", target, err)
			}

			out, stderr, err := runWithCapturedIOAndStdin([]string{"verify", "--json", "--in", "-"}, bundleJSON)
			if err != nil {
				t.Fatalf("verify stdin returned error: %v\nstderr=%s", err, stderr)
			}
			if stderr != "" {
				t.Fatalf("expected empty stderr, got %q", stderr)
			}

			var got map[string]any
			if err := json.Unmarshal([]byte(out), &got); err != nil {
				t.Fatalf("unmarshal verify output: %v\noutput=%s", err, out)
			}
			if valid, ok := got["valid"].(bool); !ok || !valid {
				t.Fatalf("expected valid=true for target %q, got %v", target, got["valid"])
			}
			if got["digest_algorithm"] != "sha256" {
				t.Fatalf("unexpected digest_algorithm for target %q: %v", target, got["digest_algorithm"])
			}
		})
	}
}

func TestVerifyDetectsTamper(t *testing.T) {
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

	inPath := filepath.Join(t.TempDir(), "tampered.json")
	if err := os.WriteFile(inPath, tamperedBytes, 0o644); err != nil {
		t.Fatalf("write tampered bundle: %v", err)
	}

	_, _, err = runWithCapturedIO([]string{"verify", "--in", inPath})
	if err == nil {
		t.Fatal("expected verify to fail for tampered bundle")
	}
	if !strings.Contains(err.Error(), "bundle digest mismatch") {
		t.Fatalf("expected digest mismatch error, got %q", err.Error())
	}
}

func generateBundleJSON() (string, error) {
	return generateBundleJSONForTarget("helm")
}

func generateBundleJSONForTarget(target string) (string, error) {
	importOut, stderr, err := runWithCapturedIO([]string{"gitops", "import", "--space", "platform", "--json", target, "render-target"})
	if err != nil {
		return "", err
	}
	if stderr != "" {
		return "", fmt.Errorf("unexpected stderr from import: %s", stderr)
	}

	out, pubStderr, err := runWithCapturedIOAndStdin([]string{"publish", "--in", "-"}, importOut)
	if err != nil {
		return "", err
	}
	if pubStderr != "" {
		return "", fmt.Errorf("unexpected stderr from publish: %s", pubStderr)
	}
	return out, nil
}
