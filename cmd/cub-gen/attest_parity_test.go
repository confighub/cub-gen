package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAttestGoldenJSON(t *testing.T) {
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
	replaceString(got, "generated_at", "<timestamp>")
	replaceString(got, "bundle_digest", "<bundle_digest>")
	replaceString(got, "attestation_digest", "<attestation_digest>")
	replaceString(got, "change_id", "<change_id>")
	assertGoldenJSON(t, "testdata/parity/attest.json.golden.json", got)
}

func TestAttestGoldenTamperedError(t *testing.T) {
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
	normalizedErr := normalizeVerifyError(err.Error()) + "\n"
	assertGoldenText(t, "testdata/parity/attest-tampered.stderr.golden.txt", normalizedErr)
}
