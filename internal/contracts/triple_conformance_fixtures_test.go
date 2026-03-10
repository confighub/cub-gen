package contracts_test

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/confighub/cub-gen/internal/contracts"
	"github.com/confighub/cub-gen/internal/importer"
	"github.com/confighub/cub-gen/internal/model"
	"github.com/confighub/cub-gen/internal/registry"
)

type familyFixture struct {
	Name    string
	RepoDir string
	Kind    model.GeneratorKind
	Profile string
}

func TestContractTripleConformanceFixtures(t *testing.T) {
	fixtures := allFamilyFixtures()
	assertFixtureCoverageMatchesRegistryKinds(t, fixtures)

	for _, fixture := range fixtures {
		fixture := fixture
		t.Run(fixture.Name, func(t *testing.T) {
			repo := filepath.Join("..", "..", "examples", fixture.RepoDir)

			first, err := importer.ImportRepo(repo, "main", "platform")
			if err != nil {
				t.Fatalf("ImportRepo first run returned error: %v", err)
			}
			second, err := importer.ImportRepo(repo, "main", "platform")
			if err != nil {
				t.Fatalf("ImportRepo second run returned error: %v", err)
			}

			if len(first.Detection.Generators) != 1 {
				t.Fatalf("expected one detected generator, got %d", len(first.Detection.Generators))
			}
			if first.Detection.Generators[0].Kind != fixture.Kind {
				t.Fatalf("expected detected kind %q, got %q", fixture.Kind, first.Detection.Generators[0].Kind)
			}
			if first.Detection.Generators[0].Profile != fixture.Profile {
				t.Fatalf("expected detected profile %q, got %q", fixture.Profile, first.Detection.Generators[0].Profile)
			}

			if err := contracts.ValidateGovernedImportTriples(
				len(first.Detection.Generators),
				first.GeneratorContracts,
				first.Provenance,
				first.InversePlans,
			); err != nil {
				t.Fatalf("expected governed triple validation to pass, got: %v", err)
			}

			if len(first.GeneratorContracts) != len(first.Detection.Generators) {
				t.Fatalf("expected contract count %d, got %d", len(first.Detection.Generators), len(first.GeneratorContracts))
			}
			if len(first.Provenance) != len(first.Detection.Generators) {
				t.Fatalf("expected provenance count %d, got %d", len(first.Detection.Generators), len(first.Provenance))
			}
			if len(first.InversePlans) != len(first.Detection.Generators) {
				t.Fatalf("expected inverse plan count %d, got %d", len(first.Detection.Generators), len(first.InversePlans))
			}

			assertTripleSchemaVersions(t, first)
			assertTripleRequiredFields(t, first)

			normalizeTripleTimestamps(&first)
			normalizeTripleTimestamps(&second)

			if !reflect.DeepEqual(first.GeneratorContracts, second.GeneratorContracts) {
				t.Fatalf("expected deterministic generator contracts across runs")
			}
			if !reflect.DeepEqual(first.Provenance, second.Provenance) {
				t.Fatalf("expected deterministic provenance ordering/content across runs")
			}
			if !reflect.DeepEqual(first.InversePlans, second.InversePlans) {
				t.Fatalf("expected deterministic inverse plan ordering/content across runs")
			}
		})
	}
}

func allFamilyFixtures() []familyFixture {
	return []familyFixture{
		{Name: "helm", RepoDir: "helm-paas", Kind: model.GeneratorHelm, Profile: "helm-paas"},
		{Name: "score", RepoDir: "scoredev-paas", Kind: model.GeneratorScore, Profile: "scoredev-paas"},
		{Name: "spring", RepoDir: "springboot-paas", Kind: model.GeneratorSpringBoot, Profile: "springboot-paas"},
		{Name: "backstage", RepoDir: "backstage-idp", Kind: model.GeneratorBackstage, Profile: "backstage-idp"},
		{Name: "ably", RepoDir: "just-apps-no-platform-config", Kind: model.GeneratorAbly, Profile: "ably-config"},
		{Name: "ops", RepoDir: "ops-workflow", Kind: model.GeneratorOpsFlow, Profile: "ops-workflow"},
		{Name: "c3agent", RepoDir: "c3agent", Kind: model.GeneratorC3Agent, Profile: "c3agent"},
		{Name: "swamp", RepoDir: "swamp-automation", Kind: model.GeneratorSwamp, Profile: "swamp"},
	}
}

func assertFixtureCoverageMatchesRegistryKinds(t *testing.T, fixtures []familyFixture) {
	t.Helper()
	covered := map[model.GeneratorKind]struct{}{}
	for _, f := range fixtures {
		covered[f.Kind] = struct{}{}
	}
	for _, kind := range registry.Kinds() {
		if _, ok := covered[kind]; !ok {
			t.Fatalf("fixture coverage missing generator kind %q", kind)
		}
	}
	if len(covered) != len(registry.Kinds()) {
		t.Fatalf("fixture coverage includes unknown kinds: covered=%d registry=%d", len(covered), len(registry.Kinds()))
	}
}

func assertTripleSchemaVersions(t *testing.T, result model.ImportResult) {
	t.Helper()
	for i, contract := range result.GeneratorContracts {
		if contract.SchemaVersion != "cub.confighub.io/generator-contract/v1" {
			t.Fatalf("contract[%d] schema_version mismatch: %q", i, contract.SchemaVersion)
		}
	}
	for i, provenance := range result.Provenance {
		if provenance.SchemaVersion != "cub.confighub.io/provenance/v1" {
			t.Fatalf("provenance[%d] schema_version mismatch: %q", i, provenance.SchemaVersion)
		}
	}
	for i, plan := range result.InversePlans {
		if plan.SchemaVersion != "cub.confighub.io/inverse-transform-plan/v1" {
			t.Fatalf("inverse_plan[%d] schema_version mismatch: %q", i, plan.SchemaVersion)
		}
	}
}

func assertTripleRequiredFields(t *testing.T, result model.ImportResult) {
	t.Helper()
	for i, contract := range result.GeneratorContracts {
		if contract.GeneratorID == "" || contract.Name == "" || contract.Kind == "" || contract.Profile == "" {
			t.Fatalf("contract[%d] missing required fields: %+v", i, contract)
		}
		if len(contract.Inputs) == 0 || len(contract.Capabilities) == 0 {
			t.Fatalf("contract[%d] missing required array fields: %+v", i, contract)
		}
	}
	for i, provenance := range result.Provenance {
		if provenance.ProvenanceID == "" || provenance.ChangeID == "" || provenance.GeneratorID == "" || provenance.GeneratorName == "" || provenance.GeneratorProfile == "" {
			t.Fatalf("provenance[%d] missing required identity fields: %+v", i, provenance)
		}
		if len(provenance.Sources) == 0 || len(provenance.Outputs) == 0 || len(provenance.FieldOriginMap) == 0 || len(provenance.InverseEditPointers) == 0 {
			t.Fatalf("provenance[%d] missing required array fields: %+v", i, provenance)
		}
	}
	for i, plan := range result.InversePlans {
		if plan.PlanID == "" || plan.ChangeID == "" || plan.SourceKind == "" || plan.TargetUnitID == "" || plan.Status == "" {
			t.Fatalf("inverse_plan[%d] missing required fields: %+v", i, plan)
		}
		if len(plan.Patches) == 0 {
			t.Fatalf("inverse_plan[%d] missing required patches: %+v", i, plan)
		}
	}
}

func normalizeTripleTimestamps(result *model.ImportResult) {
	for i := range result.Provenance {
		result.Provenance[i].RenderedAt = "<rendered_at>"
	}
	for i := range result.InversePlans {
		result.InversePlans[i].CreatedAt = "<created_at>"
	}
}
