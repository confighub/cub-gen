package main

import (
	"sort"
	"strings"
	"testing"

	"github.com/confighub/cub-gen/internal/exampletruth"
	"github.com/confighub/cub-gen/internal/registry"
)

func TestBridgeSymmetryMatrix(t *testing.T) {
	matrix := exampletruth.BridgeSymmetryMatrix()
	if len(matrix) == 0 {
		t.Fatal("bridge symmetry matrix must not be empty")
	}
	assertBridgeSymmetryMatrixCoverage(t, matrix)
}

func assertBridgeSymmetryMatrixCoverage(t *testing.T, matrix []exampletruth.FamilyFixture) {
	t.Helper()

	byKind := map[string]exampletruth.FamilyFixture{}
	for _, fixture := range matrix {
		if fixture.ExpectedKind == "" {
			t.Fatalf("matrix fixture %q has empty expectedKind", fixture.Name)
		}
		if fixture.ExpectedProfile == "" {
			t.Fatalf("matrix fixture %q has empty expectedProfile", fixture.Name)
		}
		if fixture.RepoSuffix == "" {
			t.Fatalf("matrix fixture %q has empty repoSuffix", fixture.Name)
		}
		if prev, exists := byKind[fixture.ExpectedKind]; exists {
			t.Fatalf("matrix has duplicate kind %q in fixtures %q and %q", fixture.ExpectedKind, prev.Name, fixture.Name)
		}
		byKind[fixture.ExpectedKind] = fixture
	}

	missing := make([]string, 0)
	for _, kind := range registry.Kinds() {
		if _, ok := byKind[string(kind)]; !ok {
			missing = append(missing, string(kind))
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		t.Fatalf("bridge symmetry matrix missing registry kinds: %s", joinCSV(missing))
	}

	extras := make([]string, 0)
	registryKinds := map[string]struct{}{}
	for _, kind := range registry.Kinds() {
		registryKinds[string(kind)] = struct{}{}
	}
	for kind := range byKind {
		if _, ok := registryKinds[kind]; !ok {
			extras = append(extras, kind)
		}
	}
	if len(extras) > 0 {
		sort.Strings(extras)
		t.Fatalf("bridge symmetry matrix includes unknown kinds: %s", joinCSV(extras))
	}
}

func joinCSV(values []string) string {
	clone := append([]string(nil), values...)
	sort.Strings(clone)
	return strings.Join(clone, ", ")
}
