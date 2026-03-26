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
	if got, want := matrix.Summary.FeaturedExamples, 12; got != want {
		t.Fatalf("featured examples = %d, want %d", got, want)
	}
	if got, want := matrix.Summary.GeneratorFixtures, 8; got != want {
		t.Fatalf("generator fixtures = %d, want %d", got, want)
	}
	if got, want := matrix.Summary.SourceChainVerified, 8; got != want {
		t.Fatalf("source-chain verified = %d, want %d", got, want)
	}
	if got, want := matrix.Summary.ConnectedReleaseGated, 12; got != want {
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

	live := rows["live-reconcile"]
	if live.RealLiveProof != RealLiveStandalone {
		t.Fatalf("live-reconcile real_live_proof = %q, want %q", live.RealLiveProof, RealLiveStandalone)
	}
	if live.SourceChainVerified {
		t.Fatal("live-reconcile must not be marked source-chain verified")
	}

	c3agent := rows["c3agent"]
	if c3agent.AIFirstSurface != AIFirstExplicit {
		t.Fatalf("c3agent ai_first_surface = %q, want %q", c3agent.AIFirstSurface, AIFirstExplicit)
	}

	ops := rows["ops-workflow"]
	if ops.AIFirstSurface != AIFirstPartial {
		t.Fatalf("ops-workflow ai_first_surface = %q, want %q", ops.AIFirstSurface, AIFirstPartial)
	}
}
