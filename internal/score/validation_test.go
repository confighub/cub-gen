package score

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateWorkloadAllowExample(t *testing.T) {
	result, err := ValidateWorkload(ValidateWorkloadOptions{
		ScorePath:    filepath.Join("..", "..", "examples", "scoredev-paas", "score.yaml"),
		ContractPath: filepath.Join("..", "..", "examples", "scoredev-paas", "platform", "contracts", "workload-class.yaml"),
	})
	if err != nil {
		t.Fatalf("ValidateWorkload() error = %v", err)
	}
	if !result.Allowed || result.State != "ALLOW" {
		t.Fatalf("expected ALLOW result, got %+v", result)
	}
	if got, want := strings.Join(result.ResourceTypes, ","), "dns,postgres,redis"; got != want {
		t.Fatalf("resource types = %q, want %q", got, want)
	}
	if len(result.UnapprovedResourceTypes) != 0 {
		t.Fatalf("expected no unapproved resource types, got %+v", result.UnapprovedResourceTypes)
	}
}

func TestValidateWorkloadEscalateForUnapprovedResourceType(t *testing.T) {
	dir := t.TempDir()
	scorePath := filepath.Join(dir, "score.yaml")
	contractPath := filepath.Join(dir, "workload-class.yaml")

	scoreYAML := `apiVersion: score.dev/v1b1
kind: Workload
metadata:
  name: checkout-api
resources:
  db:
    type: postgres
  cache:
    type: redis
  ml-gpu:
    type: gpu-pool
`
	contractYAML := `apiVersion: platform.confighub.io/v1alpha1
kind: WorkloadClass
metadata:
  name: web-api-standard
spec:
  approvedResourceTypes:
    - postgres
    - redis
    - dns
`
	if err := os.WriteFile(scorePath, []byte(scoreYAML), 0o644); err != nil {
		t.Fatalf("write score file: %v", err)
	}
	if err := os.WriteFile(contractPath, []byte(contractYAML), 0o644); err != nil {
		t.Fatalf("write contract file: %v", err)
	}

	result, err := ValidateWorkload(ValidateWorkloadOptions{
		ScorePath:    scorePath,
		ContractPath: contractPath,
	})
	if err != nil {
		t.Fatalf("ValidateWorkload() error = %v", err)
	}
	if result.Allowed || result.State != "ESCALATE" {
		t.Fatalf("expected ESCALATE result, got %+v", result)
	}
	if got, want := strings.Join(result.UnapprovedResourceTypes, ","), "gpu-pool"; got != want {
		t.Fatalf("unapproved resource types = %q, want %q", got, want)
	}
}
