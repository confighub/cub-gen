package normalize

import (
	"path/filepath"
	"strings"
	"testing"

	gitopsflow "github.com/confighub/cub-gen/internal/gitops"
	"github.com/confighub/cub-gen/internal/model"
)

func TestBuildPlanSpringBootGovernedPatchSet(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "examples", "springboot-paas"))
	if err != nil {
		t.Fatalf("resolve spring example: %v", err)
	}
	plan, err := BuildPlan(exampleImportResult(root))
	if err != nil {
		t.Fatalf("build normalize plan: %v", err)
	}

	if plan.Summary.ProposalCount != 5 {
		t.Fatalf("expected 5 proposals, got %+v", plan.Summary)
	}
	if plan.Summary.TransformCount != 5 {
		t.Fatalf("expected 5 transforms, got %+v", plan.Summary)
	}
	if plan.Summary.RiskCounts[RiskHigh] != 1 || plan.Summary.RiskCounts[RiskMedium] != 2 || plan.Summary.RiskCounts[RiskLow] != 2 {
		t.Fatalf("unexpected risk counts: %+v", plan.Summary.RiskCounts)
	}

	byID := proposalsByID(plan)
	routePolicy := byID["01-route-policy-annotation"]
	if routePolicy.Transform != TransformRoutePolicy {
		t.Fatalf("missing route policy proposal: %+v", routePolicy)
	}
	if routePolicy.SourcePath != "operational/field-routes.yaml" {
		t.Fatalf("unexpected route policy source: %q", routePolicy.SourcePath)
	}
	if !strings.Contains(routePolicy.Patch.Content, routePolicyAnnotation) {
		t.Fatalf("route policy content missing annotation key:\n%s", routePolicy.Patch.Content)
	}
	if !strings.Contains(routePolicy.Patch.Content, `"route": "generator-owned"`) {
		t.Fatalf("route policy content missing generator-owned route:\n%s", routePolicy.Patch.Content)
	}

	secretRefs := byID["05-secret-references"]
	if secretRefs.Transform != TransformSecretRefs {
		t.Fatalf("missing secret reference proposal: %+v", secretRefs)
	}
	if !strings.Contains(secretRefs.Patch.Content, "SPRING_DATASOURCE_URL") {
		t.Fatalf("secret proposal does not name datasource env:\n%s", secretRefs.Patch.Content)
	}
	if count := strings.Count(secretRefs.Patch.Content, "SPRING_DATASOURCE_URL"); count != 1 {
		t.Fatalf("expected deduped datasource env once, got %d:\n%s", count, secretRefs.Patch.Content)
	}
}

func TestBuildPlanUnknownRepoNoops(t *testing.T) {
	root := t.TempDir()
	plan, err := BuildPlan(gitopsflow.ImportFlowResult{
		Space:            "platform",
		TargetSlug:       "unknown",
		TargetPath:       root,
		RenderTargetSlug: "unknown",
		RenderTargetPath: root,
		Ref:              "HEAD",
		ImportedAt:       "2026-04-30T00:00:00Z",
	})
	if err != nil {
		t.Fatalf("build normalize plan: %v", err)
	}
	if plan.Summary.ProposalCount != 0 {
		t.Fatalf("expected no proposals, got %+v", plan.Summary)
	}
	if len(plan.Diagnostics) != 1 || plan.Diagnostics[0].Code != "NO_KNOWN_PATTERNS" {
		t.Fatalf("expected no-op diagnostic, got %+v", plan.Diagnostics)
	}
}

func TestRenderPatchIncludesAllCreateProposals(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "examples", "springboot-paas"))
	if err != nil {
		t.Fatalf("resolve spring example: %v", err)
	}
	plan, err := BuildPlan(exampleImportResult(root))
	if err != nil {
		t.Fatalf("build normalize plan: %v", err)
	}
	patch, err := RenderPatch(plan)
	if err != nil {
		t.Fatalf("render patch: %v", err)
	}
	for _, expected := range []string{
		".cub-gen/normalize/confighub-route-policy.annotation.json",
		".cub-gen/normalize/lift-upstream.proposals.json",
		".cub-gen/normalize/deployable-variants.yaml",
		".cub-gen/normalize/owner-annotations.yaml",
		".cub-gen/normalize/secret-references.yaml",
	} {
		if !strings.Contains(patch, "+++ b/"+expected) {
			t.Fatalf("patch missing %s:\n%s", expected, patch)
		}
	}
}

func proposalsByID(plan Plan) map[string]Proposal {
	out := map[string]Proposal{}
	for _, proposal := range plan.PatchSet.Proposals {
		out[proposal.ID] = proposal
	}
	return out
}

func exampleImportResult(root string) gitopsflow.ImportFlowResult {
	return gitopsflow.ImportFlowResult{
		Space:            "platform",
		TargetSlug:       "springboot-paas",
		TargetPath:       root,
		RenderTargetSlug: "springboot-paas",
		RenderTargetPath: root,
		Ref:              "HEAD",
		ImportedAt:       "2026-04-30T00:00:00Z",
		DryInputs: []model.DryInputRef{
			{GeneratorID: "gen_spring", Profile: "springboot-paas", Role: "build-config", Owner: "platform-engineer", Path: "pom.xml", Required: true},
			{GeneratorID: "gen_spring", Profile: "springboot-paas", Role: "app-config-base", Owner: "app-team", Path: "src/main/resources/application.yaml", Required: true},
			{GeneratorID: "gen_spring", Profile: "springboot-paas", Role: "app-config-profile", Owner: "app-team", Path: "src/main/resources/application-dev.yaml", Required: true},
			{GeneratorID: "gen_spring", Profile: "springboot-paas", Role: "app-config-profile", Owner: "app-team", Path: "src/main/resources/application-prod.yaml", Required: true},
			{GeneratorID: "gen_spring", Profile: "springboot-paas", Role: "app-config-profile", Owner: "app-team", Path: "src/main/resources/application-stage.yaml", Required: true},
		},
		WetManifestTargets: []model.WetManifestTarget{
			{GeneratorID: "gen_spring", Kind: "Kustomization", Name: "springboot-paas", Namespace: "apps", Owner: "platform-runtime"},
			{GeneratorID: "gen_spring", Kind: "Deployment", Name: "springboot-paas", Namespace: "apps", Owner: "platform-runtime"},
			{GeneratorID: "gen_spring", Kind: "ConfigMap", Name: "springboot-paas-config", Namespace: "apps", Owner: "platform-runtime"},
		},
	}
}
