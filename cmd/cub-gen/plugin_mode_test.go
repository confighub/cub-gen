package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestPluginModeTopLevelHelpRendersCubGen(t *testing.T) {
	t.Setenv("CUB_PLUGIN", "1")

	stdout, stderr, err := runWithCapturedIO([]string{"--help"})
	if err != nil {
		t.Fatalf("plugin help failed: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, "cub gen: trace repo config to rendered output so you know what to edit") {
		t.Fatalf("plugin help did not render cub gen title:\n%s", stdout)
	}
	if !strings.Contains(stdout, "  cub gen gitops import --space my-space ./examples/helm-paas") {
		t.Fatalf("plugin help did not render cub gen examples:\n%s", stdout)
	}
	if strings.Contains(stdout, "cub-gen gitops import") {
		t.Fatalf("plugin help leaked standalone invocation:\n%s", stdout)
	}
}

func TestPluginModeAliasesRouteToExistingCommands(t *testing.T) {
	t.Setenv("CUB_PLUGIN", "1")

	fanoutManifest, err := filepath.Abs(filepath.Join("..", "..", "testdata", "variant-fanout", "platform.yaml"))
	if err != nil {
		t.Fatalf("resolve variant fanout fixture: %v", err)
	}
	stdout, stderr, err := runWithCapturedIO([]string{"fanout", "--variant", "dev", "--json", fanoutManifest})
	if err != nil {
		t.Fatalf("cub gen fanout alias failed: %v\nstderr=%s", err, stderr)
	}
	var fanout map[string]any
	if err := json.Unmarshal([]byte(stdout), &fanout); err != nil {
		t.Fatalf("parse fanout alias output: %v\n%s", err, stdout)
	}
	summary, ok := fanout["summary"].(map[string]any)
	if !ok {
		t.Fatalf("fanout output missing summary: %+v", fanout)
	}
	if got := summary["variant_count"]; got != float64(3) {
		t.Fatalf("variant_count = %v, want 3", got)
	}

	adaptManifest, err := filepath.Abs(filepath.Join("..", "..", "testdata", "deployment-adaptation", "platform.yaml"))
	if err != nil {
		t.Fatalf("resolve adaptation fixture: %v", err)
	}
	stdout, stderr, err = runWithCapturedIO([]string{"adapt", "--json", adaptManifest})
	if err != nil {
		t.Fatalf("cub gen adapt alias failed: %v\nstderr=%s", err, stderr)
	}
	var adapt map[string]any
	if err := json.Unmarshal([]byte(stdout), &adapt); err != nil {
		t.Fatalf("parse adapt alias output: %v\n%s", err, stdout)
	}
	summary, ok = adapt["summary"].(map[string]any)
	if !ok {
		t.Fatalf("adapt output missing summary: %+v", adapt)
	}
	if got := summary["placeholder_count"]; got != float64(3) {
		t.Fatalf("placeholder_count = %v, want 3", got)
	}
}

func TestPluginModeBundleHelpRoutesToCommandHelp(t *testing.T) {
	t.Setenv("CUB_PLUGIN", "1")

	tests := []struct {
		name      string
		args      []string
		wantTitle string
	}{
		{
			name:      "bundle help",
			args:      []string{"bundle", "help"},
			wantTitle: "cub gen publish: build verifiable evidence",
		},
		{
			name:      "bundle publish help",
			args:      []string{"bundle", "publish", "help"},
			wantTitle: "cub gen publish: build verifiable evidence",
		},
		{
			name:      "bundle verify help",
			args:      []string{"bundle", "verify", "help"},
			wantTitle: "Usage of verify:",
		},
		{
			name:      "bundle events help",
			args:      []string{"bundle", "events", "help"},
			wantTitle: "Usage of proof events:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, err := runWithCapturedIO(tt.args)
			if err != nil {
				t.Fatalf("%s failed: %v\nstdout=%s\nstderr=%s", tt.name, err, stdout, stderr)
			}
			if stdout != "" {
				t.Fatalf("expected empty stdout, got %q", stdout)
			}
			if !strings.Contains(stderr, tt.wantTitle) {
				t.Fatalf("help output missing %q:\n%s", tt.wantTitle, stderr)
			}
			if strings.Contains(stderr, "cub-gen ") {
				t.Fatalf("plugin help leaked standalone invocation:\n%s", stderr)
			}
		})
	}
}

func TestCubHostDispatchesGenPluginWhenAvailable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("cub plugin exec uses syscall.Exec")
	}
	cub, err := exec.LookPath("cub")
	if err != nil {
		t.Skip("cub host not on PATH")
	}

	tmpDir := t.TempDir()
	pluginBin := filepath.Join(tmpDir, "plugins", "gen", "main")
	if err := os.MkdirAll(filepath.Dir(pluginBin), 0o755); err != nil {
		t.Fatalf("mkdir plugin dir: %v", err)
	}

	buildCmd := exec.Command("go", "build", "-o", pluginBin, ".")
	buildCmd.Stderr = new(bytes.Buffer)
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("go build plugin binary failed: %v\n%s", err, buildCmd.Stderr.(*bytes.Buffer).String())
	}

	cmd := exec.Command(cub, "gen", "--help")
	cmd.Env = append(os.Environ(),
		"CUB_CONFIG="+filepath.Join(tmpDir, "config.yaml"),
		"CONFIGHUB_AGENT=1",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("cub gen --help failed: %v\n%s", err, string(out))
	}
	if !strings.Contains(string(out), "cub gen: trace repo config to rendered output so you know what to edit") {
		t.Fatalf("cub host did not dispatch gen plugin help:\n%s", string(out))
	}
}
