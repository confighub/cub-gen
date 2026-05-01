package exampletruth

import "path/filepath"

// FamilyFixture defines the authoritative source-side example fixture for a
// first-class generator family.
type FamilyFixture struct {
	Name            string
	RepoSuffix      string
	ExpectedProfile string
	ExpectedKind    string
}

func BridgeSymmetryMatrix() []FamilyFixture {
	return []FamilyFixture{
		{
			Name:            "app-of-apps",
			RepoSuffix:      filepath.Join("testdata", "app-of-apps-standalone"),
			ExpectedProfile: "app-of-apps",
			ExpectedKind:    "app-of-apps",
		},
		{
			Name:            "applicationset",
			RepoSuffix:      filepath.Join("testdata", "applicationset-standalone"),
			ExpectedProfile: "applicationset",
			ExpectedKind:    "applicationset",
		},
		{
			Name:            "helm",
			RepoSuffix:      filepath.Join("examples", "helm-paas"),
			ExpectedProfile: "helm-paas",
			ExpectedKind:    "helm",
		},
		{
			Name:            "score",
			RepoSuffix:      filepath.Join("examples", "scoredev-paas"),
			ExpectedProfile: "scoredev-paas",
			ExpectedKind:    "score",
		},
		{
			Name:            "spring",
			RepoSuffix:      filepath.Join("examples", "springboot-paas"),
			ExpectedProfile: "springboot-paas",
			ExpectedKind:    "springboot",
		},
		{
			Name:            "backstage",
			RepoSuffix:      filepath.Join("examples", "backstage-idp"),
			ExpectedProfile: "backstage-idp",
			ExpectedKind:    "backstage",
		},
		{
			Name:            "no-config-platform",
			RepoSuffix:      filepath.Join("examples", "just-apps-no-platform-config"),
			ExpectedProfile: "no-config-platform",
			ExpectedKind:    "no-config-platform",
		},
		{
			Name:            "openchoreo",
			RepoSuffix:      filepath.Join("testdata", "openchoreo-hardgate"),
			ExpectedProfile: "openchoreo",
			ExpectedKind:    "openchoreo",
		},
		{
			Name:            "ops",
			RepoSuffix:      filepath.Join("examples", "ops-workflow"),
			ExpectedProfile: "ops-workflow",
			ExpectedKind:    "opsworkflow",
		},
		{
			Name:            "c3agent",
			RepoSuffix:      filepath.Join("examples", "c3agent"),
			ExpectedProfile: "c3agent",
			ExpectedKind:    "c3agent",
		},
		{
			Name:            "swamp",
			RepoSuffix:      filepath.Join("examples", "swamp-automation"),
			ExpectedProfile: "swamp",
			ExpectedKind:    "swamp",
		},
	}
}
