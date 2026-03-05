package main

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitOpsParityGoldenDiscover(t *testing.T) {
	aliases := setupAliases(t)

	out, stderr, err := runWithCapturedIO([]string{"gitops", "discover", "--space", "platform", "--json", "helm"})
	if err != nil {
		t.Fatalf("run discover returned error: %v\nstderr=%s", err, stderr)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got: %s", stderr)
	}

	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal discover json: %v\noutput=%s", err, out)
	}
	normalizeDiscover(got)

	got["target_path_expected_suffix"] = filepath.ToSlash(filepath.Join("examples", "helm-paas"))
	got["alias_path_suffix"] = trimToSuffix(filepath.ToSlash(aliases["helm"]), filepath.ToSlash(filepath.Join("examples", "helm-paas")))

	assertGoldenJSON(t, filepath.Join("testdata", "parity", "gitops-discover.golden.json"), got)
}

func TestGitOpsParityGoldenImport(t *testing.T) {
	setupAliases(t)

	out, stderr, err := runWithCapturedIO([]string{"gitops", "import", "--space", "platform", "--json", "spring", "render-target"})
	if err != nil {
		t.Fatalf("run import returned error: %v\nstderr=%s", err, stderr)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got: %s", stderr)
	}

	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal import json: %v\noutput=%s", err, out)
	}
	normalizeImport(got)

	assertGoldenJSON(t, filepath.Join("testdata", "parity", "gitops-import.golden.json"), got)
}

func TestGitOpsParityGoldenCleanup(t *testing.T) {
	setupAliases(t)

	_, _, err := runWithCapturedIO([]string{"gitops", "discover", "--space", "platform", "--json", "score"})
	if err != nil {
		t.Fatalf("pre-cleanup discover returned error: %v", err)
	}

	out, stderr, err := runWithCapturedIO([]string{"gitops", "cleanup", "--space", "platform", "--json", "score"})
	if err != nil {
		t.Fatalf("run cleanup returned error: %v\nstderr=%s", err, stderr)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got: %s", stderr)
	}

	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal cleanup json: %v\noutput=%s", err, out)
	}
	normalizeCleanup(got)

	assertGoldenJSON(t, filepath.Join("testdata", "parity", "gitops-cleanup.golden.json"), got)
}

func TestGitOpsParityGoldenDiscoverTable(t *testing.T) {
	setupAliases(t)

	out, stderr, err := runWithCapturedIO([]string{"gitops", "discover", "--space", "platform", "helm"})
	if err != nil {
		t.Fatalf("run discover table returned error: %v\nstderr=%s", err, stderr)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got: %s", stderr)
	}
	assertGoldenText(t, filepath.Join("testdata", "parity", "gitops-discover.table.golden.txt"), out)
}

func TestGitOpsParityGoldenImportTable(t *testing.T) {
	setupAliases(t)

	out, stderr, err := runWithCapturedIO([]string{"gitops", "import", "--space", "platform", "spring", "render-target"})
	if err != nil {
		t.Fatalf("run import table returned error: %v\nstderr=%s", err, stderr)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got: %s", stderr)
	}
	assertGoldenText(t, filepath.Join("testdata", "parity", "gitops-import.table.golden.txt"), out)
}

func TestGitOpsParityGoldenHelp(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		stdoutGolden string
		stderrGolden string
	}{
		{
			name:         "gitops-help",
			args:         []string{"gitops", "--help"},
			stdoutGolden: filepath.Join("testdata", "parity", "gitops-help.stdout.golden.txt"),
		},
		{
			name:         "gitops-discover-help",
			args:         []string{"gitops", "discover", "--help"},
			stderrGolden: filepath.Join("testdata", "parity", "gitops-discover-help.stderr.golden.txt"),
		},
		{
			name:         "gitops-import-help",
			args:         []string{"gitops", "import", "--help"},
			stderrGolden: filepath.Join("testdata", "parity", "gitops-import-help.stderr.golden.txt"),
		},
		{
			name:         "gitops-cleanup-help",
			args:         []string{"gitops", "cleanup", "--help"},
			stderrGolden: filepath.Join("testdata", "parity", "gitops-cleanup-help.stderr.golden.txt"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, err := runWithCapturedIO(tt.args)
			if err != nil {
				t.Fatalf("run help returned error: %v", err)
			}

			if tt.stdoutGolden == "" {
				if strings.TrimSpace(stdout) != "" {
					t.Fatalf("expected empty stdout, got: %q", stdout)
				}
			} else {
				assertGoldenText(t, tt.stdoutGolden, stdout)
			}

			if tt.stderrGolden == "" {
				if strings.TrimSpace(stderr) != "" {
					t.Fatalf("expected empty stderr, got: %q", stderr)
				}
			} else {
				assertGoldenText(t, tt.stderrGolden, stderr)
			}
		})
	}
}

func TestGitOpsParityErrorModes(t *testing.T) {
	setupAliases(t)

	tests := []struct {
		name string
		args []string
		sub  string
	}{
		{
			name: "missing-subcommand",
			args: []string{"gitops"},
			sub:  "gitops subcommand required",
		},
		{
			name: "discover-missing-target",
			args: []string{"gitops", "discover"},
			sub:  "usage: cub-gen gitops discover",
		},
		{
			name: "import-missing-render-target",
			args: []string{"gitops", "import", "helm"},
			sub:  "usage: cub-gen gitops import",
		},
		{
			name: "unsupported-where-resource",
			args: []string{"gitops", "discover", "--where-resource", "metadata.namespace = 'argocd'", "helm"},
			sub:  "unsupported where-resource clause",
		},
		{
			name: "unknown-target",
			args: []string{"gitops", "discover", "does-not-exist"},
			sub:  "not a directory and not found in aliases",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := runWithCapturedIO(tt.args)
			if err == nil {
				t.Fatalf("expected error for args %v", tt.args)
			}
			if !strings.Contains(err.Error(), tt.sub) {
				t.Fatalf("expected error containing %q, got %q", tt.sub, err.Error())
			}
		})
	}
}

func TestGitOpsParityErrorMissingRenderProvider(t *testing.T) {
	helmAbs, err := filepath.Abs(filepath.Join("..", "..", "examples", "scoredev-paas"))
	if err != nil {
		t.Fatalf("resolve score path: %v", err)
	}
	cfgDir := t.TempDir()
	cfgPath := filepath.Join(cfgDir, "targets.json")
	cfg := map[string]any{
		"targets": map[string]any{
			"score": helmAbs,
			"render-flux-only": map[string]any{
				"toolchain": "kubernetes/yaml",
				"providers": []string{"fluxrenderer"},
			},
		},
	}
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatalf("marshal cfg: %v", err)
	}
	if err := os.WriteFile(cfgPath, b, 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}
	t.Setenv("CUB_GEN_TARGETS_FILE", cfgPath)

	_, _, runErr := runWithCapturedIO([]string{"gitops", "import", "score", "render-flux-only"})
	if runErr == nil {
		t.Fatal("expected missing provider error")
	}
	if !strings.Contains(runErr.Error(), "missing providers") {
		t.Fatalf("expected missing providers error, got %q", runErr.Error())
	}
}

func setupAliases(t *testing.T) map[string]string {
	t.Helper()

	helmAbs, err := filepath.Abs(filepath.Join("..", "..", "examples", "helm-paas"))
	if err != nil {
		t.Fatalf("resolve helm path: %v", err)
	}
	scoreAbs, err := filepath.Abs(filepath.Join("..", "..", "examples", "scoredev-paas"))
	if err != nil {
		t.Fatalf("resolve score path: %v", err)
	}
	springAbs, err := filepath.Abs(filepath.Join("..", "..", "examples", "springboot-paas"))
	if err != nil {
		t.Fatalf("resolve spring path: %v", err)
	}

	cfgDir := t.TempDir()
	cfgPath := filepath.Join(cfgDir, "targets.json")
	// Render target metadata (no repo path required in this prototype).
	cfgAny := map[string]any{
		"targets": map[string]any{
			"helm":   helmAbs,
			"score":  scoreAbs,
			"spring": springAbs,
			"render-target": map[string]any{
				"toolchain": "kubernetes/yaml",
				"providers": []string{"fluxrenderer", "argocdrenderer"},
			},
		},
	}
	b, err := json.MarshalIndent(cfgAny, "", "  ")
	if err != nil {
		t.Fatalf("marshal targets config: %v", err)
	}
	if err := os.WriteFile(cfgPath, b, 0o644); err != nil {
		t.Fatalf("write targets config: %v", err)
	}
	t.Setenv("CUB_GEN_TARGETS_FILE", cfgPath)

	return map[string]string{
		"helm":   helmAbs,
		"score":  scoreAbs,
		"spring": springAbs,
	}
}

func runWithCapturedIO(args []string) (stdout, stderr string, err error) {
	oldOut := os.Stdout
	oldErr := os.Stderr

	outR, outW, pipeErr := os.Pipe()
	if pipeErr != nil {
		return "", "", pipeErr
	}
	errR, errW, pipeErr := os.Pipe()
	if pipeErr != nil {
		_ = outR.Close()
		_ = outW.Close()
		return "", "", pipeErr
	}

	os.Stdout = outW
	os.Stderr = errW

	defer func() {
		os.Stdout = oldOut
		os.Stderr = oldErr
	}()

	err = run(args)

	_ = outW.Close()
	_ = errW.Close()

	outBytes, outReadErr := io.ReadAll(outR)
	errBytes, errReadErr := io.ReadAll(errR)
	_ = outR.Close()
	_ = errR.Close()
	if outReadErr != nil {
		return "", "", outReadErr
	}
	if errReadErr != nil {
		return "", "", errReadErr
	}

	return string(outBytes), string(errBytes), err
}

func normalizeDiscover(m map[string]any) {
	replaceString(m, "target_path", "<target_path>")
	replaceString(m, "discover_unit_slug", "<discover_unit_slug>")
	replaceString(m, "discover_file", "<discover_file>")
	replaceString(m, "discovered_at", "<timestamp>")
}

func normalizeImport(m map[string]any) {
	replaceString(m, "target_path", "<target_path>")
	replaceString(m, "discover_unit_slug", "<discover_unit_slug>")
	replaceString(m, "imported_at", "<timestamp>")

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

func normalizeCleanup(m map[string]any) {
	replaceString(m, "discover_file", "<discover_file>")
}

func asSlice(v any) []map[string]any {
	raw, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		if m, ok := item.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

func replaceString(m map[string]any, key, value string) {
	if _, ok := m[key]; ok {
		m[key] = value
	}
}

func assertGoldenJSON(t *testing.T, path string, got any) {
	t.Helper()

	b, err := json.MarshalIndent(got, "", "  ")
	if err != nil {
		t.Fatalf("marshal golden payload: %v", err)
	}
	b = append(b, '\n')

	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir golden dir: %v", err)
		}
		if err := os.WriteFile(path, b, 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden file %s: %v (set UPDATE_GOLDEN=1 to create)", path, err)
	}

	if string(want) != string(b) {
		t.Fatalf("golden mismatch for %s\n--- want ---\n%s\n--- got ---\n%s", path, string(want), string(b))
	}
}

func assertGoldenText(t *testing.T, path string, got string) {
	t.Helper()

	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir golden dir: %v", err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden file %s: %v (set UPDATE_GOLDEN=1 to create)", path, err)
	}
	if string(want) != got {
		t.Fatalf("golden mismatch for %s\n--- want ---\n%s\n--- got ---\n%s", path, string(want), got)
	}
}

func trimToSuffix(value, suffix string) string {
	idx := strings.LastIndex(value, suffix)
	if idx < 0 {
		return value
	}
	return value[idx:]
}
