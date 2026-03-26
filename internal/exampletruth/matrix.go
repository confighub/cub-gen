package exampletruth

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	RealLiveNone          = "none"
	RealLivePairedHarness = "paired-harness"
	RealLiveStandalone    = "standalone"

	AIFirstNone     = "none"
	AIFirstPartial  = "partial"
	AIFirstExplicit = "explicit"
)

const (
	schemaVersion       = "cub.confighub.io/example-truth-matrix/v1"
	sourceChainCommand  = "go test ./cmd/cub-gen -run '^(TestExamplesPathModeDiscoverAndImport|TestExamplesPathModeBridgeFlow)$' -count=1 -v"
	connectedCITarget   = "make ci-connected"
	lifecycleGateScript = "./examples/demo/run-all-connected-lifecycles.sh"
)

var exampleLinkPattern = regexp.MustCompile(`\]\((?:\./|\.\./)?([a-z0-9-]+)/\)`)

type ProofRefs struct {
	SourceChain          []string `json:"source_chain,omitempty"`
	ConnectedMode        []string `json:"connected_mode,omitempty"`
	ConnectedReleaseGate []string `json:"connected_release_gate,omitempty"`
	RealLive             []string `json:"real_live,omitempty"`
	AIFirst              []string `json:"ai_first,omitempty"`
}

type ExampleRow struct {
	Example               string    `json:"example"`
	GeneratorFixture      bool      `json:"generator_fixture"`
	GeneratorKind         string    `json:"generator_kind,omitempty"`
	SourceChainVerified   bool      `json:"source_chain_verified"`
	ConnectedModePresent  bool      `json:"connected_mode_present"`
	ConnectedReleaseGated bool      `json:"connected_release_gated"`
	RealLiveProof         string    `json:"real_live_proof"`
	AIFirstSurface        string    `json:"ai_first_surface"`
	ProofCommands         []string  `json:"proof_commands"`
	ProofRefs             ProofRefs `json:"proof_refs"`
	TrackingIssues        []string  `json:"tracking_issues"`
	Notes                 []string  `json:"notes,omitempty"`
}

type Summary struct {
	FeaturedExamples      int            `json:"featured_examples"`
	GeneratorFixtures     int            `json:"generator_fixtures"`
	SourceChainVerified   int            `json:"source_chain_verified"`
	ConnectedModePresent  int            `json:"connected_mode_present"`
	ConnectedReleaseGated int            `json:"connected_release_gated"`
	RealLiveProof         map[string]int `json:"real_live_proof"`
	AIFirstSurface        map[string]int `json:"ai_first_surface"`
}

type Matrix struct {
	SchemaVersion string       `json:"schema_version"`
	Rows          []ExampleRow `json:"rows"`
	Summary       Summary      `json:"summary"`
}

func Collect(root string) (Matrix, error) {
	featured, err := featuredExampleSlugs(root)
	if err != nil {
		return Matrix{}, err
	}

	fixtures := map[string]FamilyFixture{}
	for _, fixture := range BridgeSymmetryMatrix() {
		fixtures[filepath.Base(fixture.RepoSuffix)] = fixture
	}

	lifecycleReleaseGate, err := parseConnectedLifecycleExamples(filepath.Join(root, "examples", "demo", "run-all-connected-lifecycles.sh"))
	if err != nil {
		return Matrix{}, err
	}

	ciConnectedDeps, err := makeTargetDependencies(filepath.Join(root, "Makefile"), "ci-connected")
	if err != nil {
		return Matrix{}, err
	}

	aiExplicit, err := parseReadmeSectionExamples(
		filepath.Join(root, "examples", "README.md"),
		"## AI + automation patterns",
		"## ",
	)
	if err != nil {
		return Matrix{}, err
	}

	aiTrack, err := parseReadmeSectionExamples(
		filepath.Join(root, "examples", "demo", "README.md"),
		"### AI work platform track",
		"### ",
		"## ",
	)
	if err != nil {
		return Matrix{}, err
	}

	rows := make([]ExampleRow, 0, len(featured))
	for _, slug := range featured {
		row := ExampleRow{
			Example:        slug,
			RealLiveProof:  RealLiveNone,
			AIFirstSurface: AIFirstNone,
		}

		if fixture, ok := fixtures[slug]; ok {
			row.GeneratorFixture = true
			row.GeneratorKind = fixture.ExpectedKind
			row.SourceChainVerified = true
			row.ProofRefs.SourceChain = []string{sourceChainCommand}
		}

		if fileExists(filepath.Join(root, "examples", slug, "demo-connected.sh")) {
			row.ConnectedModePresent = true
			row.ProofRefs.ConnectedMode = []string{fmt.Sprintf("./examples/%s/demo-connected.sh", slug)}
		}

		if _, ok := lifecycleReleaseGate[slug]; ok {
			row.ConnectedReleaseGated = true
			row.ProofRefs.ConnectedReleaseGate = []string{connectedCITarget, lifecycleGateScript}
		}
		if slug == "live-reconcile" && hasAll(ciConnectedDeps, "test-live-reconcile-flux", "test-live-reconcile-argo") {
			row.ConnectedReleaseGated = true
			row.ProofRefs.ConnectedReleaseGate = []string{
				connectedCITarget,
				"./examples/demo/e2e-live-reconcile-flux.sh",
				"./examples/demo/e2e-live-reconcile-argo.sh",
				"./test/checks/check-story-evidence.sh",
			}
		}

		switch slug {
		case "helm-paas":
			row.RealLiveProof = RealLivePairedHarness
			row.ProofRefs.RealLive = []string{
				"./examples/demo/e2e-connected-governed-reconcile-helm.sh",
				"./examples/live-reconcile/demo-local.sh",
			}
			row.Notes = append(row.Notes, "Real LIVE proof is paired through the live-reconcile harness, not standalone in helm-paas.")
		case "live-reconcile":
			row.RealLiveProof = RealLiveStandalone
			row.ProofRefs.RealLive = []string{
				"./examples/demo/e2e-live-reconcile-flux.sh",
				"./examples/demo/e2e-live-reconcile-argo.sh",
				"./examples/demo/e2e-connected-governed-reconcile-helm.sh",
			}
			row.Notes = append(row.Notes, "Runtime harness for WET->LIVE proof; source-side generator proof lives in paired examples.")
		}

		if _, ok := aiExplicit[slug]; ok {
			row.AIFirstSurface = AIFirstExplicit
			row.ProofRefs.AIFirst = []string{"examples/README.md#ai--automation-patterns"}
		} else if _, ok := aiTrack[slug]; ok {
			row.AIFirstSurface = AIFirstPartial
			row.ProofRefs.AIFirst = []string{"examples/demo/README.md#ai-work-platform-track"}
		}

		row.ProofCommands = commandRefs(row.ProofRefs)
		row.TrackingIssues = trackingIssuesForExample(slug, row.AIFirstSurface)
		rows = append(rows, row)
	}

	sort.Slice(rows, func(i, j int) bool {
		return rows[i].Example < rows[j].Example
	})

	return Matrix{
		SchemaVersion: schemaVersion,
		Rows:          rows,
		Summary:       summarize(rows),
	}, nil
}

func featuredExampleSlugs(root string) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(root, "examples"))
	if err != nil {
		return nil, fmt.Errorf("read examples directory: %w", err)
	}

	var slugs []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		slug := entry.Name()
		if slug == "demo" || slug == "incubator" {
			continue
		}
		exampleDir := filepath.Join(root, "examples", slug)
		if fileExists(filepath.Join(exampleDir, "demo-local.sh")) && fileExists(filepath.Join(exampleDir, "demo-connected.sh")) {
			slugs = append(slugs, slug)
		}
	}
	sort.Strings(slugs)
	return slugs, nil
}

func parseConnectedLifecycleExamples(path string) (map[string]struct{}, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open connected lifecycle runner: %w", err)
	}
	defer file.Close()

	inArray := false
	out := map[string]struct{}{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		switch {
		case line == "examples=(":
			inArray = true
		case inArray && line == ")":
			return out, nil
		case inArray:
			line = strings.Trim(line, "\"")
			if line != "" {
				out[line] = struct{}{}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan connected lifecycle runner: %w", err)
	}
	return nil, fmt.Errorf("examples array not found in %s", path)
}

func makeTargetDependencies(path, target string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open makefile: %w", err)
	}
	defer file.Close()

	prefix := target + ":"
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		fields := strings.Fields(strings.TrimSpace(strings.TrimPrefix(line, prefix)))
		return fields, nil
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan makefile: %w", err)
	}
	return nil, fmt.Errorf("target not found in makefile: %s", target)
}

func parseReadmeSectionExamples(path, heading string, stopPrefixes ...string) (map[string]struct{}, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open README section source %s: %w", path, err)
	}
	defer file.Close()

	inSection := false
	out := map[string]struct{}{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if !inSection {
			if trimmed == heading {
				inSection = true
			}
			continue
		}

		for _, prefix := range stopPrefixes {
			if strings.HasPrefix(trimmed, prefix) {
				return out, nil
			}
		}

		for _, match := range exampleLinkPattern.FindAllStringSubmatch(line, -1) {
			slug := match[1]
			if slug == "demo" || slug == "incubator" {
				continue
			}
			out[slug] = struct{}{}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan README section source %s: %w", path, err)
	}
	return nil, fmt.Errorf("heading not found or section did not terminate in %s: %s", path, heading)
}

func summarize(rows []ExampleRow) Summary {
	summary := Summary{
		FeaturedExamples: len(rows),
		RealLiveProof: map[string]int{
			RealLiveNone:          0,
			RealLivePairedHarness: 0,
			RealLiveStandalone:    0,
		},
		AIFirstSurface: map[string]int{
			AIFirstNone:     0,
			AIFirstPartial:  0,
			AIFirstExplicit: 0,
		},
	}
	for _, row := range rows {
		if row.GeneratorFixture {
			summary.GeneratorFixtures++
		}
		if row.SourceChainVerified {
			summary.SourceChainVerified++
		}
		if row.ConnectedModePresent {
			summary.ConnectedModePresent++
		}
		if row.ConnectedReleaseGated {
			summary.ConnectedReleaseGated++
		}
		summary.RealLiveProof[row.RealLiveProof]++
		summary.AIFirstSurface[row.AIFirstSurface]++
	}
	return summary
}

func commandRefs(refs ProofRefs) []string {
	out := uniqueStrings(append([]string{},
		append([]string{}, refs.SourceChain...)...,
	))
	out = uniqueStrings(append(out, refs.ConnectedMode...))
	out = uniqueStrings(append(out, refs.ConnectedReleaseGate...))
	out = uniqueStrings(append(out, refs.RealLive...))
	return out
}

func trackingIssuesForExample(slug, aiSurface string) []string {
	issues := []string{"#173", "#183"}
	switch slug {
	case "helm-paas":
		issues = append(issues, "#177", "#187")
	case "scoredev-paas":
		issues = append(issues, "#178")
	case "springboot-paas":
		issues = append(issues, "#179")
	case "ops-workflow", "swamp-automation":
		issues = append(issues, "#180")
	case "backstage-idp", "just-apps-no-platform-config", "confighub-actions", "c3agent", "ai-ops-paas", "live-reconcile":
		issues = append(issues, "#181")
	case "swamp-project":
		issues = append(issues, "#180")
	}
	if aiSurface != AIFirstNone {
		issues = append(issues, "#202")
	}
	if slug == "c3agent" {
		issues = append(issues, "#216")
	}
	return uniqueStrings(issues)
}

func hasAll(values []string, wanted ...string) bool {
	set := map[string]struct{}{}
	for _, value := range values {
		set[value] = struct{}{}
	}
	for _, value := range wanted {
		if _, ok := set[value]; !ok {
			return false
		}
	}
	return true
}

func uniqueStrings(values []string) []string {
	set := map[string]struct{}{}
	for _, value := range values {
		if value == "" {
			continue
		}
		set[value] = struct{}{}
	}
	out := make([]string, 0, len(set))
	for value := range set {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
