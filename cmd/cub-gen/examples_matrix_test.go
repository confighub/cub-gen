package main

import (
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/confighub/cub-gen/internal/registry"
)

type exampleFamilyFixture struct {
	name            string
	repoSuffix      string
	expectedProfile string
	expectedKind    string
}

func bridgeSymmetryMatrix() []exampleFamilyFixture {
	return []exampleFamilyFixture{
		{
			name:            "helm",
			repoSuffix:      filepath.Join("examples", "helm-paas"),
			expectedProfile: "helm-paas",
			expectedKind:    "helm",
		},
		{
			name:            "score",
			repoSuffix:      filepath.Join("examples", "scoredev-paas"),
			expectedProfile: "scoredev-paas",
			expectedKind:    "score",
		},
		{
			name:            "spring",
			repoSuffix:      filepath.Join("examples", "springboot-paas"),
			expectedProfile: "springboot-paas",
			expectedKind:    "springboot",
		},
		{
			name:            "backstage",
			repoSuffix:      filepath.Join("examples", "backstage-idp"),
			expectedProfile: "backstage-idp",
			expectedKind:    "backstage",
		},
		{
			name:            "ably",
			repoSuffix:      filepath.Join("examples", "ably-config"),
			expectedProfile: "ably-config",
			expectedKind:    "ably",
		},
		{
			name:            "ops",
			repoSuffix:      filepath.Join("examples", "ops-workflow"),
			expectedProfile: "ops-workflow",
			expectedKind:    "opsworkflow",
		},
		{
			name:            "c3agent",
			repoSuffix:      filepath.Join("examples", "c3agent"),
			expectedProfile: "c3agent",
			expectedKind:    "c3agent",
		},
		{
			name:            "swamp",
			repoSuffix:      filepath.Join("examples", "swamp-automation"),
			expectedProfile: "swamp",
			expectedKind:    "swamp",
		},
	}
}

func TestBridgeSymmetryMatrix(t *testing.T) {
	matrix := bridgeSymmetryMatrix()
	if len(matrix) == 0 {
		t.Fatal("bridge symmetry matrix must not be empty")
	}
	assertBridgeSymmetryMatrixCoverage(t, matrix)
}

func assertBridgeSymmetryMatrixCoverage(t *testing.T, matrix []exampleFamilyFixture) {
	t.Helper()

	byKind := map[string]exampleFamilyFixture{}
	for _, fixture := range matrix {
		if fixture.expectedKind == "" {
			t.Fatalf("matrix fixture %q has empty expectedKind", fixture.name)
		}
		if fixture.expectedProfile == "" {
			t.Fatalf("matrix fixture %q has empty expectedProfile", fixture.name)
		}
		if fixture.repoSuffix == "" {
			t.Fatalf("matrix fixture %q has empty repoSuffix", fixture.name)
		}
		if prev, exists := byKind[fixture.expectedKind]; exists {
			t.Fatalf("matrix has duplicate kind %q in fixtures %q and %q", fixture.expectedKind, prev.name, fixture.name)
		}
		byKind[fixture.expectedKind] = fixture
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
