package exampletruth

import (
	"path/filepath"
	"testing"
)

func TestCollect(t *testing.T) {
	root := filepath.Join("..", "..")

	matrix, err := Collect(root)
	if err != nil {
		t.Fatalf("collect example truth matrix: %v", err)
	}

	if matrix.SchemaVersion != schemaVersion {
		t.Fatalf("unexpected schema version: %s", matrix.SchemaVersion)
	}
	if got, want := matrix.Summary.FeaturedExamples, 14; got != want {
		t.Fatalf("featured examples = %d, want %d", got, want)
	}
	if got, want := matrix.Summary.GeneratorFixtures, 9; got != want {
		t.Fatalf("generator fixtures = %d, want %d", got, want)
	}
	if got, want := matrix.Summary.SourceChainVerified, 9; got != want {
		t.Fatalf("source-chain verified = %d, want %d", got, want)
	}
	if got, want := matrix.Summary.ConnectedReleaseGated, 2; got != want {
		t.Fatalf("connected release gated = %d, want %d", got, want)
	}

	rows := map[string]ExampleRow{}
	for _, row := range matrix.Rows {
		rows[row.Example] = row
	}

	helm := rows["helm-paas"]
	if helm.RealLiveProof != RealLivePairedHarness {
		t.Fatalf("helm-paas real_live_proof = %q, want %q", helm.RealLiveProof, RealLivePairedHarness)
	}
	if !helm.SourceChainVerified {
		t.Fatal("helm-paas should be source-chain verified")
	}
	if !helm.ConnectedReleaseGated {
		t.Fatal("helm-paas should be in the connected smoke lane")
	}

	spring := rows["springboot-paas"]
	if !spring.ConnectedReleaseGated {
		t.Fatal("springboot-paas should be in the connected smoke lane")
	}
	if spring.RealLiveProof != RealLiveStandalone {
		t.Fatalf("springboot-paas real_live_proof = %q, want %q", spring.RealLiveProof, RealLiveStandalone)
	}

	live := rows["live-reconcile"]
	if live.RealLiveProof != RealLiveStandalone {
		t.Fatalf("live-reconcile real_live_proof = %q, want %q", live.RealLiveProof, RealLiveStandalone)
	}
	if live.SourceChainVerified {
		t.Fatal("live-reconcile must not be marked source-chain verified")
	}
	if live.ConnectedReleaseGated {
		t.Fatal("live-reconcile should not be in the connected smoke lane")
	}

	c3agent := rows["c3agent"]
	if c3agent.AIFirstSurface != AIFirstExplicit {
		t.Fatalf("c3agent ai_first_surface = %q, want %q", c3agent.AIFirstSurface, AIFirstExplicit)
	}

	ops := rows["ops-workflow"]
	if ops.AIFirstSurface != AIFirstPartial {
		t.Fatalf("ops-workflow ai_first_surface = %q, want %q", ops.AIFirstSurface, AIFirstPartial)
	}

	openChoreo := rows["openchoreo"]
	if !openChoreo.SourceChainVerified || openChoreo.GeneratorKind != "openchoreo" {
		t.Fatalf("openchoreo should be source-chain verified via the hardgate fixture: %+v", openChoreo)
	}

	kubara := rows["kubara"]
	if kubara.GeneratorFixture {
		t.Fatal("kubara is a pattern wrapper, not a private Kubara conformance fixture")
	}
}
