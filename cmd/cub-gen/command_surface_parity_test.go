package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestTopLevelCommandGoldenHelp(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		stdoutGolden string
		stderrGolden string
	}{
		{
			name:         "top-help",
			args:         []string{"--help"},
			stdoutGolden: filepath.Join("testdata", "parity", "top-help.stdout.golden.txt"),
		},
		{
			name:         "publish-help",
			args:         []string{"publish", "--help"},
			stderrGolden: filepath.Join("testdata", "parity", "publish-help.stderr.golden.txt"),
		},
		{
			name:         "verify-help",
			args:         []string{"verify", "--help"},
			stderrGolden: filepath.Join("testdata", "parity", "verify-help.stderr.golden.txt"),
		},
		{
			name:         "attest-help",
			args:         []string{"attest", "--help"},
			stderrGolden: filepath.Join("testdata", "parity", "attest-help.stderr.golden.txt"),
		},
		{
			name:         "verify-attestation-help",
			args:         []string{"verify-attestation", "--help"},
			stderrGolden: filepath.Join("testdata", "parity", "verify-attestation-help.stderr.golden.txt"),
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

func TestTopLevelCommandErrorModes(t *testing.T) {
	tests := []struct {
		name string
		args []string
		sub  string
	}{
		{
			name: "publish-extra-arg",
			args: []string{"publish", "one"},
			sub:  "usage: cub-gen publish",
		},
		{
			name: "verify-extra-arg",
			args: []string{"verify", "extra"},
			sub:  "usage: cub-gen verify",
		},
		{
			name: "attest-extra-arg",
			args: []string{"attest", "extra"},
			sub:  "usage: cub-gen attest",
		},
		{
			name: "verify-attestation-extra-arg",
			args: []string{"verify-attestation", "extra"},
			sub:  "usage: cub-gen verify-attestation",
		},
		{
			name: "verify-attestation-invalid-json",
			args: []string{"verify-attestation", "--in", "/dev/null"},
			sub:  "parse attestation json",
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
