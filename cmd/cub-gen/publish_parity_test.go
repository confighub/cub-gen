package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestPublishGoldenFromImport(t *testing.T) {
	setupAliases(t)

	importOut, stderr, err := runWithCapturedIO([]string{"gitops", "import", "--space", "platform", "--json", "helm", "render-target"})
	if err != nil {
		t.Fatalf("run import returned error: %v\nstderr=%s", err, stderr)
	}

	inPath := filepath.Join(t.TempDir(), "import.json")
	if writeErr := os.WriteFile(inPath, []byte(importOut), 0o644); writeErr != nil {
		t.Fatalf("write import input: %v", writeErr)
	}

	out, pubStderr, err := runWithCapturedIO([]string{"publish", "--in", inPath})
	if err != nil {
		t.Fatalf("run publish returned error: %v\nstderr=%s", err, pubStderr)
	}
	if pubStderr != "" {
		t.Fatalf("expected empty stderr, got %q", pubStderr)
	}

	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal publish json: %v\noutput=%s", err, out)
	}
	normalizePublish(got)

	assertGoldenJSON(t, filepath.Join("testdata", "parity", "publish-from-import.golden.json"), got)
}

func normalizePublish(m map[string]any) {
	replaceString(m, "generated_at", "<timestamp>")
	replaceString(m, "change_id", "<change_id>")
	replaceString(m, "bundle_digest", "<bundle_digest>")
	replaceString(m, "target_path", "<target_path>")

	for _, item := range asSlice(m["contracts"]) {
		replaceString(item, "source_repo", "<target_path>")
	}
	for _, item := range asSlice(m["provenance"]) {
		replaceString(item, "provenance_id", "<provenance_id>")
		replaceString(item, "change_id", "<change_id>")
		replaceString(item, "rendered_at", "<timestamp>")
		for _, source := range asSlice(item["sources"]) {
			replaceString(source, "uri", "<source_uri>")
		}
		for _, output := range asSlice(item["outputs"]) {
			replaceString(output, "digest", "<digest>")
		}
	}
	for _, item := range asSlice(m["inverse_transform_plans"]) {
		replaceString(item, "plan_id", "<plan_id>")
		replaceString(item, "change_id", "<change_id>")
		replaceString(item, "created_at", "<timestamp>")
	}
}
