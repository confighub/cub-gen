package main

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/confighub/cub-gen/internal/exampletruth"
)

func TestExamplesPathModeDiscoverAndImport(t *testing.T) {
	tests := exampletruth.BridgeSymmetryMatrix()
	assertBridgeSymmetryMatrixCoverage(t, tests)

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			repoPath, err := filepath.Abs(filepath.Join("..", "..", tt.RepoSuffix))
			if err != nil {
				t.Fatalf("resolve repo path: %v", err)
			}

			discoverOut, discoverErr, err := runWithCapturedIO([]string{"gitops", "discover", "--space", "platform", "--json", repoPath})
			if err != nil {
				t.Fatalf("discover returned error: %v\nstderr=%s", err, discoverErr)
			}
			if strings.TrimSpace(discoverErr) != "" {
				t.Fatalf("expected empty discover stderr, got: %q", discoverErr)
			}

			var discover map[string]any
			if err := json.Unmarshal([]byte(discoverOut), &discover); err != nil {
				t.Fatalf("unmarshal discover output: %v\noutput=%s", err, discoverOut)
			}
			assertFirstGeneratorRecord(t, discover, tt.ExpectedProfile, tt.ExpectedKind)

			importOut, importErr, err := runWithCapturedIO([]string{"gitops", "import", "--space", "platform", "--json", repoPath, repoPath})
			if err != nil {
				t.Fatalf("import returned error: %v\nstderr=%s", err, importErr)
			}
			if strings.TrimSpace(importErr) != "" {
				t.Fatalf("expected empty import stderr, got: %q", importErr)
			}

			var imp map[string]any
			if err := json.Unmarshal([]byte(importOut), &imp); err != nil {
				t.Fatalf("unmarshal import output: %v\noutput=%s", err, importOut)
			}
			assertFirstGeneratorRecord(t, imp, tt.ExpectedProfile, tt.ExpectedKind)
		})
	}
}

func assertFirstGeneratorRecord(t *testing.T, payload map[string]any, expectedProfile, expectedKind string) {
	t.Helper()

	var records []any
	if discoveredAny, ok := payload["discovered"]; ok {
		if arr, ok := discoveredAny.([]any); ok && len(arr) > 0 {
			records = arr
		}
	}
	if len(records) == 0 {
		if resourcesAny, ok := payload["resources"]; ok {
			if arr, ok := resourcesAny.([]any); ok && len(arr) > 0 {
				records = arr
			}
		}
	}
	if len(records) == 0 {
		t.Fatalf("missing discovered/resources records in payload: %+v", payload)
	}

	first, ok := records[0].(map[string]any)
	if !ok {
		t.Fatalf("expected first record object, got: %#v", records[0])
	}

	if got := first["generator_profile"]; got != expectedProfile {
		t.Fatalf("expected generator_profile=%q, got %v", expectedProfile, got)
	}
	if got := first["generator_kind"]; got != expectedKind {
		t.Fatalf("expected generator_kind=%q, got %v", expectedKind, got)
	}
}
