package main

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"
)

var sha256Re = regexp.MustCompile(`sha256:[a-f0-9]{64}`)

func TestVerifyGoldenText(t *testing.T) {
	setupAliases(t)

	bundleJSON, err := generateBundleJSON()
	if err != nil {
		t.Fatalf("generate bundle: %v", err)
	}

	out, stderr, err := runWithCapturedIOAndStdin([]string{"verify", "--in", "-"}, bundleJSON)
	if err != nil {
		t.Fatalf("verify returned error: %v\nstderr=%s", err, stderr)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	normalized := normalizeVerifyText(out)
	assertGoldenText(t, "testdata/parity/verify.stdout.golden.txt", normalized)
}

func TestVerifyGoldenJSON(t *testing.T) {
	setupAliases(t)

	bundleJSON, err := generateBundleJSON()
	if err != nil {
		t.Fatalf("generate bundle: %v", err)
	}

	out, stderr, err := runWithCapturedIOAndStdin([]string{"verify", "--json", "--in", "-"}, bundleJSON)
	if err != nil {
		t.Fatalf("verify --json returned error: %v\nstderr=%s", err, stderr)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal verify json: %v\noutput=%s", err, out)
	}
	replaceString(got, "bundle_digest", "<bundle_digest>")
	replaceString(got, "change_id", "<change_id>")
	assertGoldenJSON(t, "testdata/parity/verify.json.golden.json", got)
}

func TestVerifyGoldenTamperedError(t *testing.T) {
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

	_, _, err = runWithCapturedIOAndStdin([]string{"verify", "--in", "-"}, string(tamperedBytes))
	if err == nil {
		t.Fatal("expected verify to fail for tampered bundle")
	}

	normalizedErr := normalizeVerifyError(err.Error()) + "\n"
	assertGoldenText(t, "testdata/parity/verify-tampered.stderr.golden.txt", normalizedErr)
}

func normalizeVerifyText(s string) string {
	return sha256Re.ReplaceAllString(s, "<bundle_digest>")
}

func normalizeVerifyError(s string) string {
	return sha256Re.ReplaceAllString(s, "<bundle_digest>")
}
