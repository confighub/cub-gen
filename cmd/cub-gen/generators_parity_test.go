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

func TestGeneratorsGoldenJSONKindFilter(t *testing.T) {
	out, stderr, err := runWithCapturedIO([]string{"generators", "--json", "--kind", "helm"})
	if err != nil {
		t.Fatalf("run generators --json --kind returned error: %v\nstderr=%s", err, stderr)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got: %q", stderr)
	}

	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal generators kind json: %v\noutput=%s", err, out)
	}
	assertGoldenJSON(t, filepath.Join("testdata", "parity", "generators-kind-helm.golden.json"), got)
}

func TestGeneratorsGoldenJSONCapabilityFilter(t *testing.T) {
	out, stderr, err := runWithCapturedIO([]string{"generators", "--json", "--capability", "inverse-workflow-patch"})
	if err != nil {
		t.Fatalf("run generators --json --capability returned error: %v\nstderr=%s", err, stderr)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got: %q", stderr)
	}

	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal generators capability json: %v\noutput=%s", err, out)
	}
	assertGoldenJSON(t, filepath.Join("testdata", "parity", "generators-capability-ops.golden.json"), got)
}

func TestGeneratorsGoldenJSONProfileFilter(t *testing.T) {
	out, stderr, err := runWithCapturedIO([]string{"generators", "--json", "--profile", "springboot-paas"})
	if err != nil {
		t.Fatalf("run generators --json --profile returned error: %v\nstderr=%s", err, stderr)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got: %q", stderr)
	}

	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal generators profile json: %v\noutput=%s", err, out)
	}
	assertGoldenJSON(t, filepath.Join("testdata", "parity", "generators-profile-spring.golden.json"), got)
}

func TestGeneratorsGoldenJSONCombinedFilters(t *testing.T) {
	out, stderr, err := runWithCapturedIO([]string{"generators", "--json", "--kind", "score", "--profile", "scoredev-paas", "--capability", "workload-spec"})
	if err != nil {
		t.Fatalf("run generators --json combined filters returned error: %v\nstderr=%s", err, stderr)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got: %q", stderr)
	}

	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal generators combined filter json: %v\noutput=%s", err, out)
	}
	assertGoldenJSON(t, filepath.Join("testdata", "parity", "generators-combined-score.golden.json"), got)
}

func TestGeneratorsGoldenJSONNoMatches(t *testing.T) {
	out, stderr, err := runWithCapturedIO([]string{"generators", "--json", "--profile", "non-existent-profile"})
	if err != nil {
		t.Fatalf("run generators --json no matches returned error: %v\nstderr=%s", err, stderr)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got: %q", stderr)
	}

	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal generators no matches json: %v\noutput=%s", err, out)
	}
	assertGoldenJSON(t, filepath.Join("testdata", "parity", "generators-empty.golden.json"), got)
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

func TestGeneratorsGoldenTableKindFilter(t *testing.T) {
	out, stderr, err := runWithCapturedIO([]string{"generators", "--kind", "helm"})
	if err != nil {
		t.Fatalf("run generators --kind returned error: %v\nstderr=%s", err, stderr)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got: %q", stderr)
	}
	assertGoldenText(t, filepath.Join("testdata", "parity", "generators-kind-helm.table.golden.txt"), out)
}

func TestGeneratorsGoldenTableNoMatches(t *testing.T) {
	out, stderr, err := runWithCapturedIO([]string{"generators", "--profile", "non-existent-profile"})
	if err != nil {
		t.Fatalf("run generators --profile no matches returned error: %v\nstderr=%s", err, stderr)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got: %q", stderr)
	}
	assertGoldenText(t, filepath.Join("testdata", "parity", "generators-empty.table.golden.txt"), out)
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
