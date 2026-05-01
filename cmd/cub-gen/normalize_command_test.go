package main

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	normalizeflow "github.com/confighub/cub-gen/internal/normalize"
)

func TestNormalizePreviewSpringJSON(t *testing.T) {
	setupAliases(t)

	stdout, stderr, err := runWithCapturedIO([]string{"normalize", "preview", "--space", "platform", "--json", "spring"})
	if err != nil {
		t.Fatalf("normalize preview failed: %v\nstderr=%s", err, stderr)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got: %s", stderr)
	}
	var plan normalizeflow.Plan
	if err := json.Unmarshal([]byte(stdout), &plan); err != nil {
		t.Fatalf("parse normalize json: %v\n%s", err, stdout)
	}
	if plan.Summary.ProposalCount != 5 || plan.Summary.TransformCount != 5 {
		t.Fatalf("unexpected normalize summary: %+v", plan.Summary)
	}
	if got := plan.PatchSet.Proposals[0].Transform; got != normalizeflow.TransformRoutePolicy {
		t.Fatalf("expected route policy first, got %q", got)
	}
}

func TestNormalizePreviewSpringPatchGolden(t *testing.T) {
	setupAliases(t)

	stdout, stderr, err := runWithCapturedIO([]string{"normalize", "preview", "--space", "platform", "--patch", "spring"})
	if err != nil {
		t.Fatalf("normalize preview --patch failed: %v\nstderr=%s", err, stderr)
	}
	assertGoldenText(t, filepath.Join("testdata", "parity", "normalize-preview-spring.patch.golden.diff"), stdout)
}

func TestNormalizePreviewUnknownRepoNoop(t *testing.T) {
	target := t.TempDir()
	stdout, stderr, err := runWithCapturedIO([]string{"normalize", "preview", "--space", "platform", "--json", target})
	if err != nil {
		t.Fatalf("normalize preview unknown failed: %v\nstderr=%s", err, stderr)
	}
	var plan normalizeflow.Plan
	if err := json.Unmarshal([]byte(stdout), &plan); err != nil {
		t.Fatalf("parse normalize noop json: %v\n%s", err, stdout)
	}
	if plan.Summary.ProposalCount != 0 {
		t.Fatalf("expected no proposals, got %+v", plan.Summary)
	}
	if len(plan.Diagnostics) != 1 || plan.Diagnostics[0].Code != "NO_KNOWN_PATTERNS" {
		t.Fatalf("expected no-op diagnostic, got %+v", plan.Diagnostics)
	}
}
