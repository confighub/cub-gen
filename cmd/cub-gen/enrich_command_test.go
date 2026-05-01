package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	enrichflow "github.com/confighub/cub-gen/internal/enrich"
)

func TestEnrichPreviewAppOfAppsGolden(t *testing.T) {
	setupAliases(t)

	stdout, stderr, err := runWithCapturedIO([]string{"enrich", "preview", "--space", "platform", "--json", "app-of-apps", "render-target"})
	if err != nil {
		t.Fatalf("enrich preview failed: %v\nstderr=%s", err, stderr)
	}
	var plan enrichflow.Plan
	if err := json.Unmarshal([]byte(stdout), &plan); err != nil {
		t.Fatalf("parse preview json: %v\n%s", err, stdout)
	}
	plan = normalizeEnrichPlan(plan)
	assertGoldenJSON(t, filepath.Join("testdata", "parity", "enrich-preview-app-of-apps.golden.json"), plan)
}

func TestEnrichPreviewPatchGolden(t *testing.T) {
	setupAliases(t)

	stdout, stderr, err := runWithCapturedIO([]string{"enrich", "preview", "--space", "platform", "--patch", "app-of-apps", "render-target"})
	if err != nil {
		t.Fatalf("enrich preview --patch failed: %v\nstderr=%s", err, stderr)
	}
	assertGoldenText(t, filepath.Join("testdata", "parity", "enrich-preview-app-of-apps.patch.golden.diff"), normalizeEnrichPatch(stdout))
}

func TestEnrichWriteCreatesSidecarAndRefusesOverwrite(t *testing.T) {
	source, err := filepath.Abs(filepath.Join("..", "..", "testdata", "app-of-apps-standalone"))
	if err != nil {
		t.Fatalf("resolve fixture path: %v", err)
	}
	target := t.TempDir()
	copyTestDir(t, source, target)

	stdout, stderr, err := runWithCapturedIO([]string{"enrich", "write", "--space", "platform", "--json", target})
	if err != nil {
		t.Fatalf("enrich write failed: %v\nstderr=%s", err, stderr)
	}
	var result enrichflow.WriteResult
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("parse write result: %v\n%s", err, stdout)
	}
	if len(result.Written) != 1 {
		t.Fatalf("expected one sidecar write, got %+v", result)
	}
	if _, err := os.Stat(filepath.Join(target, filepath.FromSlash(result.Written[0]))); err != nil {
		t.Fatalf("expected written sidecar: %v", err)
	}

	_, _, err = runWithCapturedIO([]string{"enrich", "write", "--space", "platform", "--json", target})
	if err == nil {
		t.Fatal("expected second write to require review")
	}
	if !strings.Contains(err.Error(), "requires review") {
		t.Fatalf("expected review-required error, got %q", err.Error())
	}
}

func normalizeEnrichPatch(text string) string {
	replacements := []struct {
		re   *regexp.Regexp
		with string
	}{
		{regexp.MustCompile(`"change_id": "[^"]+"`), `"change_id": "<change_id>"`},
		{regexp.MustCompile(`"input_digest": "[^"]+"`), `"input_digest": "<input_digest>"`},
		{regexp.MustCompile(`"uri": "[^"]+"`), `"uri": "<source_uri>"`},
		{regexp.MustCompile(`"value": "chg_[^"]+"`), `"value": "<change_id>"`},
	}
	for _, replacement := range replacements {
		text = replacement.re.ReplaceAllString(text, replacement.with)
	}
	return text
}

func normalizeEnrichPlan(plan enrichflow.Plan) enrichflow.Plan {
	plan.TargetPath = "<target_path>"
	plan.RenderTargetPath = "<render_target_path>"
	plan.GeneratedAt = "<timestamp>"
	for i := range plan.Artifacts {
		body := &plan.Artifacts[i].Body
		body.Generator.InputDigest = "<input_digest>"
		body.Generator.ChangeID = "<change_id>"
		for j := range body.PRMRLinks {
			body.PRMRLinks[j].ChangeID = "<change_id>"
		}
		for j := range body.SourceLinks {
			if body.SourceLinks[j].URI != "" {
				body.SourceLinks[j].URI = "<source_uri>"
			}
		}
		for j := range body.ProposedAnnotations {
			if body.ProposedAnnotations[j].Key == "cub.confighub.io/change-id" {
				body.ProposedAnnotations[j].Value = "<change_id>"
			}
		}
	}
	return plan
}

func copyTestDir(t *testing.T, source, target string) {
	t.Helper()
	if err := filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		dst := filepath.Join(target, rel)
		if entry.IsDir() {
			return os.MkdirAll(dst, 0o755)
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dst, raw, 0o644)
	}); err != nil {
		t.Fatalf("copy fixture: %v", err)
	}
}
