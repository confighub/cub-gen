package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/confighub/cub-gen/internal/model"
)

func TestChangePreviewJSON(t *testing.T) {
	setupAliases(t)

	out, stderr, err := runWithCapturedIO([]string{
		"change", "preview",
		"--space", "platform",
		"score",
		"render-target",
	})
	if err != nil {
		t.Fatalf("change preview returned error: %v\nstderr=%s", err, stderr)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal change preview output: %v\noutput=%s", err, out)
	}

	change, ok := got["change"].(map[string]any)
	if !ok {
		t.Fatalf("expected change object, got %T", got["change"])
	}
	if changeID, ok := change["change_id"].(string); !ok || !strings.HasPrefix(changeID, "chg_") {
		t.Fatalf("unexpected change_id: %v", change["change_id"])
	}
	if digest, ok := change["bundle_digest"].(string); !ok || !strings.HasPrefix(digest, "sha256:") {
		t.Fatalf("unexpected bundle_digest: %v", change["bundle_digest"])
	}

	verification, ok := got["verification"].(map[string]any)
	if !ok {
		t.Fatalf("expected verification object, got %T", got["verification"])
	}
	if valid, ok := verification["bundle_valid"].(bool); !ok || !valid {
		t.Fatalf("expected bundle_valid=true, got %v", verification["bundle_valid"])
	}
	if valid, ok := verification["attestation_valid"].(bool); !ok || !valid {
		t.Fatalf("expected attestation_valid=true, got %v", verification["attestation_valid"])
	}

	recommendation, ok := got["edit_recommendation"].(map[string]any)
	if !ok {
		t.Fatalf("expected edit_recommendation object, got %T", got["edit_recommendation"])
	}
	if owner, ok := recommendation["owner"].(string); !ok || strings.TrimSpace(owner) == "" {
		t.Fatalf("expected non-empty owner, got %v", recommendation["owner"])
	}
}

func TestChangePreviewDefaultsRenderTargetToTarget(t *testing.T) {
	repoPath, err := filepath.Abs(filepath.Join("..", "..", "examples", "scoredev-paas"))
	if err != nil {
		t.Fatalf("resolve score path: %v", err)
	}

	out, stderr, err := runWithCapturedIO([]string{
		"change", "preview",
		"--space", "platform",
		repoPath,
	})
	if err != nil {
		t.Fatalf("change preview shorthand returned error: %v\nstderr=%s", err, stderr)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal change preview shorthand output: %v\noutput=%s", err, out)
	}

	input, ok := got["input"].(map[string]any)
	if !ok {
		t.Fatalf("expected input object, got %T", got["input"])
	}
	if input["target_slug"] != "scoredev-paas" {
		t.Fatalf("expected target_slug=scoredev-paas, got %v", input["target_slug"])
	}
	if input["render_target_slug"] != "scoredev-paas" {
		t.Fatalf("expected render_target_slug=scoredev-paas, got %v", input["render_target_slug"])
	}
	if input["target_path"] != repoPath {
		t.Fatalf("expected target_path=%s, got %v", repoPath, input["target_path"])
	}
	if input["render_target_path"] != repoPath {
		t.Fatalf("expected render_target_path=%s, got %v", repoPath, input["render_target_path"])
	}
}

func TestChangeRunLocalJSON(t *testing.T) {
	setupAliases(t)

	out, stderr, err := runWithCapturedIO([]string{
		"change", "run",
		"--mode", "local",
		"--space", "platform",
		"score",
		"render-target",
	})
	if err != nil {
		t.Fatalf("change run returned error: %v\nstderr=%s", err, stderr)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal change run output: %v\noutput=%s", err, out)
	}
	if mode, ok := got["mode"].(string); !ok || mode != "local" {
		t.Fatalf("expected mode=local, got %v", got["mode"])
	}
	decision, ok := got["decision"].(map[string]any)
	if !ok {
		t.Fatalf("expected decision object, got %T", got["decision"])
	}
	if state, ok := decision["state"].(string); !ok || state != "ALLOW" {
		t.Fatalf("expected decision state ALLOW, got %v", decision["state"])
	}
	if source, ok := decision["source"].(string); !ok || source != "local-preview" {
		t.Fatalf("expected decision source local-preview, got %v", decision["source"])
	}
}

func TestChangeRunConnectedMissingBaseURL(t *testing.T) {
	setupAliases(t)
	t.Setenv("CONFIGHUB_BASE_URL", "")

	_, _, err := runWithCapturedIO([]string{
		"change", "run",
		"--mode", "connected",
		"--space", "platform",
		"score",
		"render-target",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "requires --base-url or CONFIGHUB_BASE_URL") {
		t.Fatalf("unexpected error: %q", err.Error())
	}
}

func TestChangeExplainJSON(t *testing.T) {
	setupAliases(t)

	out, stderr, err := runWithCapturedIO([]string{
		"change", "explain",
		"--space", "platform",
		"score",
		"render-target",
	})
	if err != nil {
		t.Fatalf("change explain returned error: %v\nstderr=%s", err, stderr)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal change explain output: %v\noutput=%s", err, out)
	}

	explanation, ok := got["explanation"].(map[string]any)
	if !ok {
		t.Fatalf("expected explanation object, got %T", got["explanation"])
	}
	if owner, ok := explanation["owner"].(string); !ok || strings.TrimSpace(owner) == "" {
		t.Fatalf("expected non-empty owner, got %v", explanation["owner"])
	}
	if wetPath, ok := explanation["wet_path"].(string); !ok || strings.TrimSpace(wetPath) == "" {
		t.Fatalf("expected non-empty wet_path, got %v", explanation["wet_path"])
	}
	if dryPath, ok := explanation["dry_path"].(string); !ok || strings.TrimSpace(dryPath) == "" {
		t.Fatalf("expected non-empty dry_path, got %v", explanation["dry_path"])
	}
}

func TestPickInverseSuggestionIncludesGeneratorChainHops(t *testing.T) {
	provenance := []model.ProvenanceRecord{{
		GeneratorName:    "checkout-api",
		GeneratorProfile: "helm-paas",
		FieldOriginMap: []model.FieldOrigin{{
			DryPath:    "containers.api.image",
			WetPath:    "Deployment/spec/template/spec/containers[0]/image",
			SourcePath: "score.yaml",
			Transform:  "score-to-helm",
			Confidence: 0.80,
			Hops: []model.FieldOriginHop{
				{
					GeneratorKind:    "score",
					GeneratorProfile: "scoredev-paas",
					DryPath:          "containers.api.image",
					SourcePath:       "score.yaml",
					Transform:        "score-to-helm",
					Confidence:       0.94,
				},
				{
					GeneratorKind:    "helm",
					GeneratorProfile: "helm-paas",
					DryPath:          "values.image.tag",
					SourcePath:       "chart/values.yaml",
					Transform:        "helm-values",
					Confidence:       0.86,
				},
			},
		}},
		InverseEditPointers: []model.InverseEditPointer{{
			WetPath:    "Deployment/spec/template/spec/containers[0]/image",
			DryPath:    "containers.api.image",
			Owner:      "app-team",
			EditHint:   "Edit containers.api.image in score.yaml.",
			Confidence: 0.80,
		}},
	}}

	suggestion, matchCount, ok := pickInverseSuggestion(provenance, "Deployment/spec/template/spec/containers[0]/image", "", "")
	if !ok {
		t.Fatal("expected inverse suggestion")
	}
	if matchCount != 1 {
		t.Fatalf("expected 1 match, got %d", matchCount)
	}
	if len(suggestion.Hops) != 2 {
		t.Fatalf("expected 2 provenance hops, got %+v", suggestion.Hops)
	}
	if suggestion.Hops[0].GeneratorKind != "score" || suggestion.Hops[1].GeneratorKind != "helm" {
		t.Fatalf("unexpected hop order: %+v", suggestion.Hops)
	}
}

func TestChangeImpactJSON(t *testing.T) {
	setupAliases(t)

	out, stderr, err := runWithCapturedIO([]string{
		"change", "impact",
		"--space", "platform",
		"--dry-path", "service.ports.web.port",
		"score",
		"render-target",
	})
	if err != nil {
		t.Fatalf("change impact returned error: %v\nstderr=%s", err, stderr)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal change impact output: %v\noutput=%s", err, out)
	}

	query, ok := got["query"].(map[string]any)
	if !ok {
		t.Fatalf("expected query object, got %T", got["query"])
	}
	if gotDryPath, ok := query["dry_path_filter"].(string); !ok || gotDryPath != "service.ports.web.port" {
		t.Fatalf("expected dry_path_filter=service.ports.web.port, got %v", query["dry_path_filter"])
	}

	impacts, ok := got["impacts"].([]any)
	if !ok || len(impacts) == 0 {
		t.Fatalf("expected non-empty impacts, got %T %#v", got["impacts"], got["impacts"])
	}
	first, ok := impacts[0].(map[string]any)
	if !ok {
		t.Fatalf("expected impact entry object, got %T", impacts[0])
	}
	if wetPath, ok := first["wet_path"].(string); !ok || wetPath != "Service/spec/ports[name=web]/port" {
		t.Fatalf("expected wet_path=Service/spec/ports[name=web]/port, got %v", first["wet_path"])
	}
	if owner, ok := first["owner"].(string); !ok || owner != "app-team" {
		t.Fatalf("expected owner=app-team, got %v", first["owner"])
	}
}

func TestChangeImpactHelmShowsOverlayAndBaseSources(t *testing.T) {
	setupAliases(t)

	out, stderr, err := runWithCapturedIO([]string{
		"change", "impact",
		"--space", "platform",
		"--dry-path", "values.image.tag",
		"helm",
		"render-target",
	})
	if err != nil {
		t.Fatalf("change impact returned error: %v\nstderr=%s", err, stderr)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal change impact output: %v\noutput=%s", err, out)
	}

	impacts, ok := got["impacts"].([]any)
	if !ok || len(impacts) != 2 {
		t.Fatalf("expected 2 impacts for Helm image tag, got %T %#v", got["impacts"], got["impacts"])
	}
	first, ok := impacts[0].(map[string]any)
	if !ok {
		t.Fatalf("expected first impact entry object, got %T", impacts[0])
	}
	second, ok := impacts[1].(map[string]any)
	if !ok {
		t.Fatalf("expected second impact entry object, got %T", impacts[1])
	}
	if first["source_path"] != "values-prod.yaml" || second["source_path"] != "values.yaml" {
		t.Fatalf("expected overlay-first sources, got %v then %v", first["source_path"], second["source_path"])
	}
}

func TestChangeHelmOverlayRecommendation(t *testing.T) {
	setupAliases(t)

	previewOut, stderr, err := runWithCapturedIO([]string{
		"change", "preview",
		"--space", "platform",
		"helm",
		"render-target",
	})
	if err != nil {
		t.Fatalf("change preview returned error: %v\nstderr=%s", err, stderr)
	}

	var preview map[string]any
	if err := json.Unmarshal([]byte(previewOut), &preview); err != nil {
		t.Fatalf("unmarshal change preview output: %v\noutput=%s", err, previewOut)
	}
	recommendation, ok := preview["edit_recommendation"].(map[string]any)
	if !ok {
		t.Fatalf("expected edit_recommendation object, got %T", preview["edit_recommendation"])
	}
	editHint, ok := recommendation["edit_hint"].(string)
	if !ok {
		t.Fatalf("expected edit hint string, got %T", recommendation["edit_hint"])
	}
	if !strings.Contains(editHint, "values-prod.yaml") || !strings.Contains(editHint, "values.yaml") {
		t.Fatalf("expected Helm edit hint to mention overlay and base values files, got %q", editHint)
	}

	explainOut, explainErr, err := runWithCapturedIO([]string{
		"change", "explain",
		"--space", "platform",
		"helm",
		"render-target",
	})
	if err != nil {
		t.Fatalf("change explain returned error: %v\nstderr=%s", err, explainErr)
	}

	var explain map[string]any
	if err := json.Unmarshal([]byte(explainOut), &explain); err != nil {
		t.Fatalf("unmarshal change explain output: %v\noutput=%s", err, explainOut)
	}
	explanation, ok := explain["explanation"].(map[string]any)
	if !ok {
		t.Fatalf("expected explanation object, got %T", explain["explanation"])
	}
	if got, ok := explanation["source_path"].(string); !ok || got != "values-prod.yaml" {
		t.Fatalf("expected source_path=values-prod.yaml, got %v", explanation["source_path"])
	}
	if got, ok := explanation["source_transform"].(string); !ok || got != "helm-values-overlay" {
		t.Fatalf("expected source_transform=helm-values-overlay, got %v", explanation["source_transform"])
	}
}

func TestChangeExplainWetPathFilter(t *testing.T) {
	setupAliases(t)

	previewOut, _, err := runWithCapturedIO([]string{
		"change", "preview",
		"--space", "platform",
		"score",
		"render-target",
	})
	if err != nil {
		t.Fatalf("change preview returned error: %v", err)
	}
	var preview map[string]any
	if err := json.Unmarshal([]byte(previewOut), &preview); err != nil {
		t.Fatalf("unmarshal change preview output: %v", err)
	}
	recommendation, ok := preview["edit_recommendation"].(map[string]any)
	if !ok {
		t.Fatalf("expected edit_recommendation object, got %T", preview["edit_recommendation"])
	}
	wetPath, ok := recommendation["wet_path"].(string)
	if !ok || strings.TrimSpace(wetPath) == "" {
		t.Fatalf("expected wet_path recommendation, got %v", recommendation["wet_path"])
	}

	explainOut, _, err := runWithCapturedIO([]string{
		"change", "explain",
		"--space", "platform",
		"--wet-path", wetPath,
		"score",
		"render-target",
	})
	if err != nil {
		t.Fatalf("change explain returned error: %v", err)
	}
	var explain map[string]any
	if err := json.Unmarshal([]byte(explainOut), &explain); err != nil {
		t.Fatalf("unmarshal change explain output: %v", err)
	}
	query, ok := explain["query"].(map[string]any)
	if !ok {
		t.Fatalf("expected query object, got %T", explain["query"])
	}
	if got, ok := query["wet_path_filter"].(string); !ok || got != wetPath {
		t.Fatalf("expected wet_path_filter=%q, got %v", wetPath, query["wet_path_filter"])
	}
}

func TestChangeImpactByChangeIDFromBundle(t *testing.T) {
	setupAliases(t)

	publishOut, stderr, err := runWithCapturedIO([]string{
		"publish",
		"--space", "platform",
		"score",
		"render-target",
	})
	if err != nil {
		t.Fatalf("publish returned error: %v\nstderr=%s", err, stderr)
	}

	var bundle map[string]any
	if err := json.Unmarshal([]byte(publishOut), &bundle); err != nil {
		t.Fatalf("unmarshal publish output: %v", err)
	}
	changeID, ok := bundle["change_id"].(string)
	if !ok || strings.TrimSpace(changeID) == "" {
		t.Fatalf("missing change_id in bundle: %v", bundle["change_id"])
	}

	bundlePath := filepath.Join(t.TempDir(), "bundle.json")
	if err := os.WriteFile(bundlePath, []byte(publishOut), 0o600); err != nil {
		t.Fatalf("write bundle file: %v", err)
	}

	out, impactErr, err := runWithCapturedIO([]string{
		"change", "impact",
		"--change-id", changeID,
		"--bundle", bundlePath,
		"--dry-path", "containers.main.image",
	})
	if err != nil {
		t.Fatalf("change impact by change-id returned error: %v\nstderr=%s", err, impactErr)
	}

	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal change impact output: %v\noutput=%s", err, out)
	}
	change, ok := got["change"].(map[string]any)
	if !ok {
		t.Fatalf("expected change object, got %T", got["change"])
	}
	if gotID, ok := change["change_id"].(string); !ok || gotID != changeID {
		t.Fatalf("expected change_id=%q, got %v", changeID, change["change_id"])
	}
}

func TestChangeExplainByChangeIDFromBundle(t *testing.T) {
	setupAliases(t)

	publishOut, stderr, err := runWithCapturedIO([]string{
		"publish",
		"--space", "platform",
		"score",
		"render-target",
	})
	if err != nil {
		t.Fatalf("publish returned error: %v\nstderr=%s", err, stderr)
	}

	var bundle map[string]any
	if err := json.Unmarshal([]byte(publishOut), &bundle); err != nil {
		t.Fatalf("unmarshal publish output: %v", err)
	}
	changeID, ok := bundle["change_id"].(string)
	if !ok || strings.TrimSpace(changeID) == "" {
		t.Fatalf("missing change_id in bundle: %v", bundle["change_id"])
	}

	bundlePath := filepath.Join(t.TempDir(), "bundle.json")
	if err := os.WriteFile(bundlePath, []byte(publishOut), 0o600); err != nil {
		t.Fatalf("write bundle file: %v", err)
	}

	out, explainErr, err := runWithCapturedIO([]string{
		"change", "explain",
		"--change-id", changeID,
		"--bundle", bundlePath,
		"--owner", "app-team",
	})
	if err != nil {
		t.Fatalf("change explain by change-id returned error: %v\nstderr=%s", err, explainErr)
	}

	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal change explain output: %v\noutput=%s", err, out)
	}
	change, ok := got["change"].(map[string]any)
	if !ok {
		t.Fatalf("expected change object, got %T", got["change"])
	}
	if gotID, ok := change["change_id"].(string); !ok || gotID != changeID {
		t.Fatalf("expected change_id=%q, got %v", changeID, change["change_id"])
	}
}

func TestChangeExplainByChangeIDMismatch(t *testing.T) {
	setupAliases(t)

	publishOut, stderr, err := runWithCapturedIO([]string{
		"publish",
		"--space", "platform",
		"score",
		"render-target",
	})
	if err != nil {
		t.Fatalf("publish returned error: %v\nstderr=%s", err, stderr)
	}
	bundlePath := filepath.Join(t.TempDir(), "bundle.json")
	if err := os.WriteFile(bundlePath, []byte(publishOut), 0o600); err != nil {
		t.Fatalf("write bundle file: %v", err)
	}

	_, _, err = runWithCapturedIO([]string{
		"change", "explain",
		"--change-id", "chg_mismatch",
		"--bundle", bundlePath,
	})
	if err == nil {
		t.Fatal("expected mismatch error")
	}
	if !strings.Contains(err.Error(), "bundle change_id mismatch") {
		t.Fatalf("unexpected error: %q", err.Error())
	}
}

func TestChangeDiffJSON(t *testing.T) {
	repo := t.TempDir()
	firstRef := seedGitHelmRepo(t, repo, "v1.0.0")
	secondRef := updateGitHelmRepoValue(t, repo, "v1.0.1")

	result, err := buildChangeDiffResult(repo, repo, "platform", firstRef, secondRef, "", nil, "", "", "")
	if err != nil {
		t.Fatalf("buildChangeDiffResult returned error: %v", err)
	}

	if result.Query.BeforeRef != firstRef {
		t.Fatalf("expected before_ref=%s, got %v", firstRef, result.Query.BeforeRef)
	}
	if result.Query.AfterRef != secondRef {
		t.Fatalf("expected after_ref=%s, got %v", secondRef, result.Query.AfterRef)
	}

	if len(result.Diffs) == 0 {
		t.Fatalf("expected non-empty diffs")
	}

	var imageDiff *changeDiffEntry
	for i := range result.Diffs {
		entry := &result.Diffs[i]
		if entry.WetPath == "Deployment/spec/template/spec/containers[0]/image" {
			imageDiff = entry
			break
		}
	}
	if imageDiff == nil {
		t.Fatalf("expected image diff entry, got %#v", result.Diffs)
	}
	if imageDiff.Before.Value != "ghcr.io/example/diff-demo:v1.0.0" {
		t.Fatalf("unexpected before value: %v", imageDiff.Before.Value)
	}
	if imageDiff.After.Value != "ghcr.io/example/diff-demo:v1.0.1" {
		t.Fatalf("unexpected after value: %v", imageDiff.After.Value)
	}
	if imageDiff.Before.SourcePath != "values.yaml" || imageDiff.After.SourcePath != "values.yaml" {
		t.Fatalf("expected before/after source_path values.yaml, got before=%v after=%v", imageDiff.Before.SourcePath, imageDiff.After.SourcePath)
	}
}

func TestChangeCommandErrorModes(t *testing.T) {
	tests := []struct {
		name string
		args []string
		sub  string
	}{
		{
			name: "missing-subcommand",
			args: []string{"change"},
			sub:  "change subcommand required",
		},
		{
			name: "unknown-subcommand",
			args: []string{"change", "unknown"},
			sub:  "unknown change subcommand",
		},
		{
			name: "preview-missing-targets",
			args: []string{"change", "preview"},
			sub:  "usage: cub-gen change preview",
		},
		{
			name: "run-missing-targets",
			args: []string{"change", "run"},
			sub:  "usage: cub-gen change run",
		},
		{
			name: "diff-missing-targets",
			args: []string{"change", "diff", "--before-ref", "main", "--after-ref", "HEAD"},
			sub:  "usage: cub-gen change diff",
		},
		{
			name: "explain-missing-targets",
			args: []string{"change", "explain"},
			sub:  "usage: cub-gen change explain",
		},
		{
			name: "impact-missing-targets",
			args: []string{"change", "impact"},
			sub:  "usage: cub-gen change impact",
		},
		{
			name: "explain-change-id-missing-bundle",
			args: []string{"change", "explain", "--change-id", "chg_123"},
			sub:  "requires --bundle FILE",
		},
		{
			name: "impact-change-id-missing-bundle",
			args: []string{"change", "impact", "--change-id", "chg_123"},
			sub:  "requires --bundle FILE",
		},
		{
			name: "api-missing-subcommand",
			args: []string{"change", "api"},
			sub:  "change api subcommand required",
		},
		{
			name: "api-unknown-subcommand",
			args: []string{"change", "api", "unknown"},
			sub:  "unknown change api subcommand",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := runWithCapturedIO(tc.args)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tc.sub) {
				t.Fatalf("expected error containing %q, got %q", tc.sub, err.Error())
			}
		})
	}
}

func seedGitHelmRepo(t *testing.T, repo, imageTag string) string {
	t.Helper()
	mustRun(t, repo, "git", "init")
	mustRun(t, repo, "git", "config", "user.name", "Codex")
	mustRun(t, repo, "git", "config", "user.email", "codex@example.com")
	mustWriteGitFile(t, filepath.Join(repo, "Chart.yaml"), "apiVersion: v2\nname: diff-demo\nversion: 0.1.0\n")
	mustWriteGitFile(t, filepath.Join(repo, "values.yaml"), "image:\n  repository: ghcr.io/example/diff-demo\n  tag: "+imageTag+"\n")
	mustWriteGitFile(t, filepath.Join(repo, "templates", "deployment.yaml"), `apiVersion: apps/v1
kind: Deployment
metadata:
  name: diff-demo
spec:
  template:
    spec:
      containers:
        - name: diff-demo
          image: "{{ .Values.image.repository }}:{{ .Values.image.tag }}"
`)
	mustRun(t, repo, "git", "add", ".")
	mustRun(t, repo, "git", "commit", "-m", "initial")
	return strings.TrimSpace(mustRun(t, repo, "git", "rev-parse", "HEAD"))
}

func updateGitHelmRepoValue(t *testing.T, repo, imageTag string) string {
	t.Helper()
	mustWriteGitFile(t, filepath.Join(repo, "values.yaml"), "image:\n  repository: ghcr.io/example/diff-demo\n  tag: "+imageTag+"\n")
	mustRun(t, repo, "git", "add", "values.yaml")
	mustRun(t, repo, "git", "commit", "-m", "update tag")
	return strings.TrimSpace(mustRun(t, repo, "git", "rev-parse", "HEAD"))
}

func mustWriteGitFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func mustRun(t *testing.T, dir, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s failed: %v\n%s", name, strings.Join(args, " "), err, string(out))
	}
	return string(out)
}
