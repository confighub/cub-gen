package main

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPublishFromImportFile(t *testing.T) {
	setupAliases(t)

	importOut, importErr, err := runWithCapturedIO([]string{"gitops", "import", "--space", "platform", "--json", "helm", "render-target"})
	if err != nil {
		t.Fatalf("gitops import returned error: %v\nstderr=%s", err, importErr)
	}
	if importErr != "" {
		t.Fatalf("expected empty stderr from import, got %q", importErr)
	}

	inPath := filepath.Join(t.TempDir(), "import.json")
	if err := os.WriteFile(inPath, []byte(importOut), 0o644); err != nil {
		t.Fatalf("write import json: %v", err)
	}

	out, stderr, err := runWithCapturedIO([]string{"publish", "--in", inPath})
	if err != nil {
		t.Fatalf("publish returned error: %v\nstderr=%s", err, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr from publish, got %q", stderr)
	}

	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal publish output: %v\noutput=%s", err, out)
	}
	if got["schema_version"] != "cub.confighub.io/change-bundle/v1" {
		t.Fatalf("unexpected schema_version: %v", got["schema_version"])
	}
	if got["source"] != "cub-gen" {
		t.Fatalf("unexpected source: %v", got["source"])
	}
	if got["change_id"] == "" {
		t.Fatalf("expected non-empty change_id: %v", got["change_id"])
	}

	summary, ok := got["summary"].(map[string]any)
	if !ok {
		t.Fatalf("expected summary object, got: %T", got["summary"])
	}
	if summary["discovered_resources"] != float64(1) {
		t.Fatalf("expected discovered_resources=1, got %v", summary["discovered_resources"])
	}
}

func TestPublishRejectsInvalidJSON(t *testing.T) {
	inPath := filepath.Join(t.TempDir(), "invalid.json")
	if err := os.WriteFile(inPath, []byte("not-json"), 0o644); err != nil {
		t.Fatalf("write invalid input: %v", err)
	}

	_, _, err := runWithCapturedIO([]string{"publish", "--in", inPath})
	if err == nil {
		t.Fatal("expected publish to fail on invalid json")
	}
	if got := err.Error(); got == "" || !strings.Contains(got, "parse import flow json") {
		t.Fatalf("expected parse import flow json error, got %q", got)
	}
}

func TestPublishFromStdin(t *testing.T) {
	setupAliases(t)

	importOut, importErr, err := runWithCapturedIO([]string{"gitops", "import", "--space", "platform", "--json", "helm", "render-target"})
	if err != nil {
		t.Fatalf("gitops import returned error: %v\nstderr=%s", err, importErr)
	}
	if importErr != "" {
		t.Fatalf("expected empty stderr from import, got %q", importErr)
	}

	out, stderr, err := runWithCapturedIOAndStdin([]string{"publish", "--in", "-"}, importOut)
	if err != nil {
		t.Fatalf("publish stdin returned error: %v\nstderr=%s", err, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr from publish stdin, got %q", stderr)
	}

	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal publish stdin output: %v\noutput=%s", err, out)
	}
	if got["schema_version"] != "cub.confighub.io/change-bundle/v1" {
		t.Fatalf("unexpected schema_version: %v", got["schema_version"])
	}
}

func runWithCapturedIOAndStdin(args []string, stdin string) (stdout, stderr string, err error) {
	oldOut := os.Stdout
	oldErr := os.Stderr
	oldIn := os.Stdin

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
	inR, inW, pipeErr := os.Pipe()
	if pipeErr != nil {
		_ = outR.Close()
		_ = outW.Close()
		_ = errR.Close()
		_ = errW.Close()
		return "", "", pipeErr
	}

	if _, writeErr := inW.Write([]byte(stdin)); writeErr != nil {
		_ = outR.Close()
		_ = outW.Close()
		_ = errR.Close()
		_ = errW.Close()
		_ = inR.Close()
		_ = inW.Close()
		return "", "", writeErr
	}
	_ = inW.Close()

	os.Stdout = outW
	os.Stderr = errW
	os.Stdin = inR

	defer func() {
		os.Stdout = oldOut
		os.Stderr = oldErr
		os.Stdin = oldIn
	}()

	err = run(args)

	_ = outW.Close()
	_ = errW.Close()
	_ = inR.Close()

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
