package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVerifyAttestationGoldenJSON(t *testing.T) {
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
	replaceString(got, "attestation_digest", "<attestation_digest>")
	replaceString(got, "bundle_digest", "<bundle_digest>")
	replaceString(got, "change_id", "<change_id>")
	assertGoldenJSON(t, "testdata/parity/verify-attestation.json.golden.json", got)
}

func TestVerifyAttestationGoldenTamperedError(t *testing.T) {
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
		t.Fatal("expected verify-attestation to fail")
	}
	assertGoldenText(t, "testdata/parity/verify-attestation-tampered.stderr.golden.txt", err.Error()+"\n")
}

func TestVerifyAttestationGoldenLinkedScoreJSON(t *testing.T) {
	assertVerifyAttestationLinkedGolden(t, "score", "testdata/parity/verify-attestation-linked-score.json.golden.json")
}

func TestVerifyAttestationGoldenLinkedSpringJSON(t *testing.T) {
	assertVerifyAttestationLinkedGolden(t, "spring", "testdata/parity/verify-attestation-linked-spring.json.golden.json")
}

func assertVerifyAttestationLinkedGolden(t *testing.T, target, goldenPath string) {
	t.Helper()
	setupAliases(t)

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
		t.Fatalf("verify-attestation linked --json returned error: %v\nstderr=%s", err, stderr)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal verify-attestation linked json: %v\noutput=%s", err, out)
	}
	replaceString(got, "attestation_digest", "<attestation_digest>")
	replaceString(got, "bundle_digest", "<bundle_digest>")
	replaceString(got, "change_id", "<change_id>")
	assertGoldenJSON(t, goldenPath, got)
}
