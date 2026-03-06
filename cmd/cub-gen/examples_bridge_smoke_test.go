package main

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestExamplesPathModeBridgeFlow(t *testing.T) {
	tests := []struct {
		name            string
		repoSuffix      string
		expectedProfile string
	}{
		{
			name:            "helm",
			repoSuffix:      filepath.Join("examples", "helm-paas"),
			expectedProfile: "helm-paas",
		},
		{
			name:            "score",
			repoSuffix:      filepath.Join("examples", "scoredev-paas"),
			expectedProfile: "scoredev-paas",
		},
		{
			name:            "spring",
			repoSuffix:      filepath.Join("examples", "springboot-paas"),
			expectedProfile: "springboot-paas",
		},
		{
			name:            "backstage",
			repoSuffix:      filepath.Join("examples", "backstage-idp"),
			expectedProfile: "backstage-idp",
		},
		{
			name:            "ably",
			repoSuffix:      filepath.Join("examples", "ably-config"),
			expectedProfile: "ably-config",
		},
		{
			name:            "ops",
			repoSuffix:      filepath.Join("examples", "ops-workflow"),
			expectedProfile: "ops-workflow",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoPath, err := filepath.Abs(filepath.Join("..", "..", tt.repoSuffix))
			if err != nil {
				t.Fatalf("resolve repo path: %v", err)
			}

			publishOut, publishErr, err := runWithCapturedIO([]string{"publish", "--space", "platform", repoPath, repoPath})
			if err != nil {
				t.Fatalf("publish returned error: %v\nstderr=%s", err, publishErr)
			}
			if strings.TrimSpace(publishErr) != "" {
				t.Fatalf("expected empty publish stderr, got %q", publishErr)
			}

			var bundle map[string]any
			if err := json.Unmarshal([]byte(publishOut), &bundle); err != nil {
				t.Fatalf("unmarshal publish output: %v\noutput=%s", err, publishOut)
			}
			if bundle["schema_version"] != "cub.confighub.io/change-bundle/v1" {
				t.Fatalf("unexpected bundle schema_version: %v", bundle["schema_version"])
			}
			assertBundleProfile(t, bundle, tt.expectedProfile)

			verifyOut, verifyErr, err := runWithCapturedIOAndStdin([]string{"verify", "--json", "--in", "-"}, publishOut)
			if err != nil {
				t.Fatalf("verify returned error: %v\nstderr=%s", err, verifyErr)
			}
			if strings.TrimSpace(verifyErr) != "" {
				t.Fatalf("expected empty verify stderr, got %q", verifyErr)
			}
			var verify map[string]any
			if err := json.Unmarshal([]byte(verifyOut), &verify); err != nil {
				t.Fatalf("unmarshal verify output: %v\noutput=%s", err, verifyOut)
			}
			if valid, ok := verify["valid"].(bool); !ok || !valid {
				t.Fatalf("expected valid=true, got %v", verify["valid"])
			}

			attestOut, attestErr, err := runWithCapturedIOAndStdin([]string{"attest", "--in", "-", "--verifier", "ci-bot"}, publishOut)
			if err != nil {
				t.Fatalf("attest returned error: %v\nstderr=%s", err, attestErr)
			}
			if strings.TrimSpace(attestErr) != "" {
				t.Fatalf("expected empty attest stderr, got %q", attestErr)
			}
			var attest map[string]any
			if err := json.Unmarshal([]byte(attestOut), &attest); err != nil {
				t.Fatalf("unmarshal attest output: %v\noutput=%s", err, attestOut)
			}
			if attest["status"] != "verified" {
				t.Fatalf("expected attestation status verified, got %v", attest["status"])
			}

			verifyAttOut, verifyAttErr, err := runWithCapturedIOAndStdin([]string{"verify-attestation", "--json", "--in", "-"}, attestOut)
			if err != nil {
				t.Fatalf("verify-attestation returned error: %v\nstderr=%s", err, verifyAttErr)
			}
			if strings.TrimSpace(verifyAttErr) != "" {
				t.Fatalf("expected empty verify-attestation stderr, got %q", verifyAttErr)
			}
			var verifyAtt map[string]any
			if err := json.Unmarshal([]byte(verifyAttOut), &verifyAtt); err != nil {
				t.Fatalf("unmarshal verify-attestation output: %v\noutput=%s", err, verifyAttOut)
			}
			if valid, ok := verifyAtt["valid"].(bool); !ok || !valid {
				t.Fatalf("expected attestation valid=true, got %v", verifyAtt["valid"])
			}
		})
	}
}

func assertBundleProfile(t *testing.T, bundle map[string]any, expectedProfile string) {
	t.Helper()

	summary, ok := bundle["summary"].(map[string]any)
	if !ok {
		t.Fatalf("expected summary object, got %T", bundle["summary"])
	}

	profiles, ok := summary["generator_profiles"].([]any)
	if !ok || len(profiles) != 1 {
		t.Fatalf("expected one generator profile, got %#v", summary["generator_profiles"])
	}

	if got := profiles[0]; got != expectedProfile {
		t.Fatalf("expected profile %q, got %#v", expectedProfile, got)
	}
}
