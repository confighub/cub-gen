package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/confighub/cub-gen/internal/mutationgate"
)

func TestGateMutationSpringRoutesJSON(t *testing.T) {
	setupAliases(t)
	routesPath := filepath.Join("..", "..", "examples", "springboot-paas", "operational", "field-routes.yaml")

	out, stderr, err := runWithCapturedIO([]string{
		"gate", "mutation",
		"--routes", routesPath,
		"--json",
		"--at", "2026-05-02T12:00:00Z",
		"feature.inventory.reservationMode",
	})
	if err != nil {
		t.Fatalf("gate mutation returned error: %v\nstderr=%s", err, stderr)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	var allow mutationgate.Decision
	if err := json.Unmarshal([]byte(out), &allow); err != nil {
		t.Fatalf("unmarshal allow decision: %v\noutput=%s", err, out)
	}
	if allow.Route.Kind != mutationgate.RouteApplyHere || allow.Decision.State != mutationgate.DecisionAllow {
		t.Fatalf("expected apply-here/ALLOW, got route=%s state=%s", allow.Route.Kind, allow.Decision.State)
	}
	if len(allow.ProofEvents) != 1 || allow.ProofEvents[0].EventType != "mutation_apply_gate.evaluated" {
		t.Fatalf("unexpected proof events: %+v", allow.ProofEvents)
	}

	out, stderr, err = runWithCapturedIO([]string{
		"gate", "mutation",
		"--routes", routesPath,
		"--json",
		"--at", "2026-05-02T12:00:00Z",
		"spring.datasource.url",
	})
	if err != nil {
		t.Fatalf("gate mutation block returned error without --enforce: %v\nstderr=%s", err, stderr)
	}
	var block mutationgate.Decision
	if err := json.Unmarshal([]byte(out), &block); err != nil {
		t.Fatalf("unmarshal block decision: %v\noutput=%s", err, out)
	}
	if block.Route.Kind != mutationgate.RouteBlockEscalate || block.Decision.State != mutationgate.DecisionBlock {
		t.Fatalf("expected block/escalate/BLOCK, got route=%s state=%s", block.Route.Kind, block.Decision.State)
	}
}

func TestGateMutationSpringLiftGolden(t *testing.T) {
	routesPath := filepath.Join("..", "..", "examples", "springboot-paas", "operational", "field-routes.yaml")
	out, stderr, err := runWithCapturedIO([]string{
		"gate", "mutation",
		"--routes", routesPath,
		"--json",
		"--at", "2026-05-02T12:00:00Z",
		"spring.cache.type",
	})
	if err != nil {
		t.Fatalf("gate mutation returned error: %v\nstderr=%s", err, stderr)
	}
	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal gate decision: %v\noutput=%s", err, out)
	}
	assertGoldenJSON(t, filepath.Join("testdata", "parity", "gate-mutation-spring-lift.golden.json"), got)
}

func TestGateMutationEnforceRejectsNonAllow(t *testing.T) {
	routesPath := filepath.Join("..", "..", "examples", "springboot-paas", "operational", "field-routes.yaml")
	out, stderr, err := runWithCapturedIO([]string{
		"gate", "mutation",
		"--routes", routesPath,
		"--json",
		"--enforce",
		"spring.cache.type",
	})
	if err == nil {
		t.Fatal("expected enforce mode to reject ESCALATE decision")
	}
	if strings.TrimSpace(out) == "" {
		t.Fatal("expected decision JSON before enforce error")
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(err.Error(), "ESCALATE") {
		t.Fatalf("expected ESCALATE error, got %v", err)
	}
}

func TestGateMutationBundleOpenChoreoJSON(t *testing.T) {
	setupAliases(t)
	ocPath := filepath.Join("..", "..", "testdata", "openchoreo-hardgate")
	publishOut, publishErr, err := runWithCapturedIO([]string{"publish", "--space", "platform", ocPath})
	if err != nil {
		t.Fatalf("publish openchoreo returned error: %v\nstderr=%s", err, publishErr)
	}
	bundlePath := filepath.Join(t.TempDir(), "bundle.json")
	if err := os.WriteFile(bundlePath, []byte(publishOut), 0o644); err != nil {
		t.Fatalf("write bundle: %v", err)
	}

	out, stderr, err := runWithCapturedIO([]string{
		"gate", "mutation",
		"--bundle", bundlePath,
		"--json",
		"--github-pr-repo", "github.com/example/payments-api",
		"Deployment/spec/template/spec/containers[name=main]/image",
	})
	if err != nil {
		t.Fatalf("gate mutation openchoreo returned error: %v\nstderr=%s", err, stderr)
	}
	var got mutationgate.Decision
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal openchoreo decision: %v\noutput=%s", err, out)
	}
	if got.Route.Kind != mutationgate.RouteLiftUpstream || got.Decision.State != mutationgate.DecisionEscalate {
		t.Fatalf("expected lift-upstream/ESCALATE, got route=%s state=%s", got.Route.Kind, got.Decision.State)
	}
	if got.Proof.Generator != "openchoreo" {
		t.Fatalf("expected openchoreo proof, got %+v", got.Proof)
	}
	if len(got.NextActions) != 1 || got.NextActions[0].Repo != "github.com/example/payments-api" {
		t.Fatalf("unexpected next actions: %+v", got.NextActions)
	}
}

func TestGateMutationMissingRouteReviewRequired(t *testing.T) {
	policyPath := filepath.Join(t.TempDir(), "policy.json")
	if err := os.WriteFile(policyPath, []byte(`{
  "schema_version": "cub.confighub.io/generator-route-policy/v1",
  "routes": [
    {
      "wet_path": "Deployment/spec/replicas",
      "route": "apply-here",
      "owner": "app-team"
    }
  ]
}`), 0o644); err != nil {
		t.Fatalf("write policy: %v", err)
	}
	out, stderr, err := runWithCapturedIO([]string{
		"gate", "mutation",
		"--policy", policyPath,
		"--json",
		"Deployment/spec/template/spec/securityContext/runAsUser",
	})
	if err != nil {
		t.Fatalf("gate mutation missing route returned error: %v\nstderr=%s", err, stderr)
	}
	var got mutationgate.Decision
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal review-required decision: %v\noutput=%s", err, out)
	}
	if got.Route.Kind != mutationgate.RouteReview || got.Decision.State != mutationgate.DecisionEscalate {
		t.Fatalf("expected review-required/ESCALATE, got route=%s state=%s", got.Route.Kind, got.Decision.State)
	}
}
