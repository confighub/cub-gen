package main

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeneratorsGoldenJSON(t *testing.T) {
	out, stderr, err := runWithCapturedIO([]string{"generators", "--json"})
	if err != nil {
		t.Fatalf("run generators --json returned error: %v\nstderr=%s", err, stderr)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got: %q", stderr)
	}

	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal generators json: %v\noutput=%s", err, out)
	}
	assertGoldenJSON(t, filepath.Join("testdata", "parity", "generators.golden.json"), got)
}

func TestGeneratorsGoldenTable(t *testing.T) {
	out, stderr, err := runWithCapturedIO([]string{"generators"})
	if err != nil {
		t.Fatalf("run generators returned error: %v\nstderr=%s", err, stderr)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got: %q", stderr)
	}
	assertGoldenText(t, filepath.Join("testdata", "parity", "generators.table.golden.txt"), out)
}

func TestGeneratorsGoldenHelp(t *testing.T) {
	stdout, stderr, err := runWithCapturedIO([]string{"generators", "--help"})
	if err != nil {
		t.Fatalf("run generators --help returned error: %v", err)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Fatalf("expected empty stdout, got: %q", stdout)
	}
	assertGoldenText(t, filepath.Join("testdata", "parity", "generators-help.stderr.golden.txt"), stderr)
}
