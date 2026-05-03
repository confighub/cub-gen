package mutationgate

import (
	"path/filepath"
	"testing"
	"time"

	gitopsflow "github.com/confighub/cub-gen/internal/gitops"
	"github.com/confighub/cub-gen/internal/publish"
)

func TestEvaluateSpringRoutes(t *testing.T) {
	policy, source, err := LoadSpringRoutesFile(filepath.Join("..", "..", "examples", "springboot-paas", "operational", "field-routes.yaml"))
	if err != nil {
		t.Fatalf("load spring routes: %v", err)
	}

	allow, err := Evaluate(policy, Request{
		ChangeID:     "chg_spring",
		PolicySource: source,
		Mutation: Mutation{
			RenderedField: "feature.inventory.reservationMode",
		},
		EvaluatedAt: time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("evaluate allow: %v", err)
	}
	if allow.Route.Kind != RouteApplyHere || allow.Decision.State != DecisionAllow {
		t.Fatalf("expected apply-here/ALLOW, got route=%s state=%s", allow.Route.Kind, allow.Decision.State)
	}
	if len(allow.ProofEvents) != 1 || allow.ProofEvents[0].RouteKind != RouteApplyHere || allow.ProofEvents[0].DecisionState != DecisionAllow {
		t.Fatalf("unexpected proof events: %+v", allow.ProofEvents)
	}
	if err := ValidateDecisionRecord(allow); err != nil {
		t.Fatalf("allow decision should be valid proof: %v", err)
	}

	lift, err := Evaluate(policy, Request{
		ChangeID:     "chg_spring",
		PolicySource: source,
		Mutation: Mutation{
			RenderedField: "spring.cache.type",
		},
		GitHubPRRepo: "github.com/example/inventory-api",
		EvaluatedAt:  time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("evaluate lift-upstream: %v", err)
	}
	if lift.Route.Kind != RouteLiftUpstream || lift.Decision.State != DecisionEscalate {
		t.Fatalf("expected lift-upstream/ESCALATE, got route=%s state=%s", lift.Route.Kind, lift.Decision.State)
	}
	if len(lift.NextActions) != 1 || lift.NextActions[0].Kind != "create-or-link-github-pr" {
		t.Fatalf("unexpected lift-upstream next actions: %+v", lift.NextActions)
	}
	if !sameStrings(lift.NextActions[0].Files, []string{"pom.xml", "src/main/resources/application.yaml"}) {
		t.Fatalf("lift-upstream should point at source proposal files, got %+v", lift.NextActions[0].Files)
	}
	if err := ValidateDecisionRecord(lift); err != nil {
		t.Fatalf("lift decision should be valid proof: %v", err)
	}

	block, err := Evaluate(policy, Request{
		ChangeID:     "chg_spring",
		PolicySource: source,
		Mutation: Mutation{
			RenderedField: "spring.datasource.url",
		},
		EvaluatedAt: time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("evaluate block: %v", err)
	}
	if block.Route.Kind != RouteBlockEscalate || block.Decision.State != DecisionBlock {
		t.Fatalf("expected block/escalate/BLOCK, got route=%s state=%s", block.Route.Kind, block.Decision.State)
	}
	if err := ValidateDecisionRecord(block); err != nil {
		t.Fatalf("block decision should be valid proof: %v", err)
	}
}

func TestEvaluateMissingRouteRequiresReview(t *testing.T) {
	decision, err := Evaluate(Policy{
		SchemaVersion: "cub.confighub.io/generator-route-policy/v1",
		Routes: []Rule{{
			WetPath: "Deployment/spec/replicas",
			Route:   RouteApplyHere,
			Owner:   "app-team",
		}},
	}, Request{
		ChangeID: "chg_missing",
		Mutation: Mutation{
			RenderedField: "Deployment/spec/template/spec/securityContext/runAsUser",
		},
		EvaluatedAt: time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("evaluate missing route: %v", err)
	}
	if decision.Route.Kind != RouteReview || decision.Decision.State != DecisionEscalate {
		t.Fatalf("expected review-required/ESCALATE, got route=%s state=%s", decision.Route.Kind, decision.Decision.State)
	}
}

func sameStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func TestPolicyFromBundleOpenChoreoRoute(t *testing.T) {
	imported, err := gitopsflow.Import(
		filepath.Join("..", "..", "testdata", "openchoreo-hardgate"),
		filepath.Join("..", "..", "testdata", "openchoreo-hardgate"),
		"HEAD",
		"platform",
		"",
	)
	if err != nil {
		t.Fatalf("import openchoreo: %v", err)
	}
	bundle := publish.BuildBundleAt(imported, time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC))
	policy, err := PolicyFromBundle(bundle)
	if err != nil {
		t.Fatalf("policy from bundle: %v", err)
	}
	decision, err := Evaluate(policy, Request{
		ChangeID:     bundle.ChangeID,
		PolicySource: "bundle.json",
		Mutation: Mutation{
			RenderedField: "Deployment/spec/template/spec/containers[name=main]/image",
		},
		EvaluatedAt: time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("evaluate openchoreo: %v", err)
	}
	if decision.Route.Kind != RouteLiftUpstream || decision.Decision.State != DecisionEscalate {
		t.Fatalf("expected lift-upstream/ESCALATE, got route=%s state=%s", decision.Route.Kind, decision.Decision.State)
	}
	if decision.Proof.Generator != "openchoreo" || decision.Proof.SourceFile == "" {
		t.Fatalf("unexpected proof: %+v", decision.Proof)
	}
}
