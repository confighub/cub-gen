package enrich

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	gitopsflow "github.com/confighub/cub-gen/internal/gitops"
	"github.com/confighub/cub-gen/internal/model"
)

const (
	PlanSchemaVersion     = "cub.confighub.io/enrichment-plan/v1"
	ArtifactSchemaVersion = "cub.confighub.io/enrichment-artifact/v1"

	ArtifactKindSidecarProvenance = "sidecar-provenance"

	ActionCreate         = "create"
	ActionReviewRequired = "review-required"
)

// Plan is a reviewable enrichment proposal. It never represents in-place
// mutation of app/platform manifests; artifacts are sidecars by default.
type Plan struct {
	SchemaVersion    string     `json:"schema_version"`
	Space            string     `json:"space"`
	TargetSlug       string     `json:"target_slug"`
	TargetPath       string     `json:"target_path,omitempty"`
	RenderTargetSlug string     `json:"render_target_slug"`
	RenderTargetPath string     `json:"render_target_path,omitempty"`
	Ref              string     `json:"ref"`
	GeneratedAt      string     `json:"generated_at"`
	Summary          Summary    `json:"summary"`
	Artifacts        []Artifact `json:"artifacts"`
}

type Summary struct {
	ArtifactCount       int `json:"artifact_count"`
	CreateCount         int `json:"create_count"`
	ReviewRequiredCount int `json:"review_required_count"`
	SourceLinkCount     int `json:"source_link_count"`
	OwnershipLabelCount int `json:"ownership_label_count"`
	RouteBadgeCount     int `json:"route_badge_count"`
	PRMRLinkCount       int `json:"pr_mr_link_count"`
}

type Artifact struct {
	Path         string       `json:"path"`
	Kind         string       `json:"kind"`
	Action       string       `json:"action"`
	ReviewReason string       `json:"review_reason,omitempty"`
	Explanation  []string     `json:"explanation"`
	Body         ArtifactBody `json:"body"`
}

type ArtifactBody struct {
	SchemaVersion       string               `json:"schema_version"`
	Generator           GeneratorRef         `json:"generator"`
	Target              TargetRef            `json:"target"`
	Counts              ArtifactCounts       `json:"counts"`
	ProposedAnnotations []ProposedAnnotation `json:"proposed_annotations"`
	SourceLinks         []SourceLink         `json:"source_links"`
	OwnershipLabels     []OwnershipLabel     `json:"ownership_labels"`
	RouteBadges         []RouteBadge         `json:"route_badges"`
	PRMRLinks           []PRMRLink           `json:"pr_mr_links"`
}

type GeneratorRef struct {
	ID          string `json:"id"`
	Kind        string `json:"kind"`
	Profile     string `json:"profile,omitempty"`
	Name        string `json:"name"`
	Root        string `json:"root,omitempty"`
	InputDigest string `json:"input_digest,omitempty"`
	ChangeID    string `json:"change_id,omitempty"`
}

type TargetRef struct {
	Space            string `json:"space"`
	TargetSlug       string `json:"target_slug"`
	RenderTargetSlug string `json:"render_target_slug"`
	Ref              string `json:"ref"`
}

type ArtifactCounts struct {
	DryInputs           int `json:"dry_inputs"`
	WetManifestTargets  int `json:"wet_manifest_targets"`
	FieldOrigins        int `json:"field_origins"`
	InverseEditPointers int `json:"inverse_edit_pointers"`
}

type ProposedAnnotation struct {
	Category  string `json:"category"`
	Key       string `json:"key"`
	Value     string `json:"value"`
	AppliesTo string `json:"applies_to"`
	Reason    string `json:"reason"`
}

type SourceLink struct {
	Path     string `json:"path"`
	Role     string `json:"role"`
	Owner    string `json:"owner,omitempty"`
	URI      string `json:"uri,omitempty"`
	Required bool   `json:"required"`
}

type OwnershipLabel struct {
	Owner     string `json:"owner"`
	Scope     string `json:"scope"`
	Role      string `json:"role,omitempty"`
	Path      string `json:"path,omitempty"`
	Kind      string `json:"kind,omitempty"`
	Name      string `json:"name,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Label     string `json:"label"`
}

type RouteBadge struct {
	WetPath    string  `json:"wet_path"`
	DryPath    string  `json:"dry_path"`
	Owner      string  `json:"owner"`
	Route      string  `json:"route"`
	Badge      string  `json:"badge"`
	EditHint   string  `json:"edit_hint"`
	Confidence float64 `json:"confidence"`
}

type PRMRLink struct {
	ChangeID         string `json:"change_id"`
	LinkKey          string `json:"link_key"`
	GitHubPRState    string `json:"github_pr_state"`
	ConfigHubMRState string `json:"confighub_mr_state"`
	Reason           string `json:"reason"`
}

type WriteResult struct {
	SchemaVersion string   `json:"schema_version"`
	TargetPath    string   `json:"target_path,omitempty"`
	Written       []string `json:"written"`
	Blocked       []string `json:"blocked,omitempty"`
}

type generatorIndexRecord struct {
	id      string
	kind    string
	profile string
	name    string
	root    string
}

func BuildPlan(imported gitopsflow.ImportFlowResult) Plan {
	records := generatorRecords(imported)
	provenanceByGeneratorID := map[string]model.ProvenanceRecord{}
	for _, prov := range imported.Provenance {
		provenanceByGeneratorID[prov.GeneratorID] = prov
	}
	dryByGeneratorID := map[string][]model.DryInputRef{}
	for _, dry := range imported.DryInputs {
		dryByGeneratorID[dry.GeneratorID] = append(dryByGeneratorID[dry.GeneratorID], dry)
	}
	wetByGeneratorID := map[string][]model.WetManifestTarget{}
	for _, wet := range imported.WetManifestTargets {
		wetByGeneratorID[wet.GeneratorID] = append(wetByGeneratorID[wet.GeneratorID], wet)
	}

	artifacts := make([]Artifact, 0, len(records))
	for _, record := range records {
		prov := provenanceByGeneratorID[record.id]
		body := buildArtifactBody(imported, record, prov, dryByGeneratorID[record.id], wetByGeneratorID[record.id])
		artifacts = append(artifacts, Artifact{
			Path:   sidecarPath(record),
			Kind:   ArtifactKindSidecarProvenance,
			Action: ActionCreate,
			Explanation: []string{
				"source links identify the repo files that feed this generator",
				"ownership labels identify who should review source and rendered artifacts",
				"route badges classify generated-resource edits as apply-here, lift-upstream, overlay, or block/escalate",
				"PR/MR link metadata carries the change_id so GitHub PRs and ConfigHub MRs can be correlated later",
			},
			Body: body,
		})
	}

	sort.Slice(artifacts, func(i, j int) bool {
		return artifacts[i].Path < artifacts[j].Path
	})

	plan := Plan{
		SchemaVersion:    PlanSchemaVersion,
		Space:            imported.Space,
		TargetSlug:       imported.TargetSlug,
		TargetPath:       imported.TargetPath,
		RenderTargetSlug: imported.RenderTargetSlug,
		RenderTargetPath: imported.RenderTargetPath,
		Ref:              imported.Ref,
		GeneratedAt:      imported.ImportedAt,
		Artifacts:        artifacts,
	}
	RefreshSummary(&plan)
	return plan
}

func MarkExisting(root string, plan *Plan) error {
	if plan == nil {
		return errors.New("enrichment plan is required")
	}
	for i := range plan.Artifacts {
		abs, err := artifactAbsPath(root, plan.Artifacts[i].Path)
		if err != nil {
			return err
		}
		if _, err := os.Stat(abs); err == nil {
			plan.Artifacts[i].Action = ActionReviewRequired
			plan.Artifacts[i].ReviewReason = "existing enrichment artifact must be reviewed before cub-gen will write a replacement"
			continue
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("stat enrichment artifact %s: %w", plan.Artifacts[i].Path, err)
		}
		plan.Artifacts[i].Action = ActionCreate
		plan.Artifacts[i].ReviewReason = ""
	}
	RefreshSummary(plan)
	return nil
}

func RefreshSummary(plan *Plan) {
	if plan == nil {
		return
	}
	summary := Summary{ArtifactCount: len(plan.Artifacts)}
	for _, artifact := range plan.Artifacts {
		switch artifact.Action {
		case ActionCreate:
			summary.CreateCount++
		case ActionReviewRequired:
			summary.ReviewRequiredCount++
		}
		summary.SourceLinkCount += len(artifact.Body.SourceLinks)
		summary.OwnershipLabelCount += len(artifact.Body.OwnershipLabels)
		summary.RouteBadgeCount += len(artifact.Body.RouteBadges)
		summary.PRMRLinkCount += len(artifact.Body.PRMRLinks)
	}
	plan.Summary = summary
}

func Write(root string, plan Plan) (WriteResult, error) {
	result := WriteResult{
		SchemaVersion: PlanSchemaVersion,
		TargetPath:    root,
	}
	for _, artifact := range plan.Artifacts {
		if artifact.Action == ActionReviewRequired {
			result.Blocked = append(result.Blocked, artifact.Path)
		}
	}
	if len(result.Blocked) > 0 {
		sort.Strings(result.Blocked)
		return result, fmt.Errorf("enrichment write requires review for existing artifact(s): %s", strings.Join(result.Blocked, ", "))
	}

	for _, artifact := range plan.Artifacts {
		if artifact.Action != ActionCreate {
			continue
		}
		abs, err := artifactAbsPath(root, artifact.Path)
		if err != nil {
			return result, err
		}
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			return result, fmt.Errorf("create enrichment directory: %w", err)
		}
		body, err := MarshalArtifactBody(artifact.Body, true)
		if err != nil {
			return result, err
		}
		f, err := os.OpenFile(abs, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err != nil {
			if errors.Is(err, os.ErrExist) {
				result.Blocked = append(result.Blocked, artifact.Path)
				return result, fmt.Errorf("enrichment artifact already exists and requires review: %s", artifact.Path)
			}
			return result, fmt.Errorf("create enrichment artifact %s: %w", artifact.Path, err)
		}
		if _, err := f.Write(body); err != nil {
			_ = f.Close()
			return result, fmt.Errorf("write enrichment artifact %s: %w", artifact.Path, err)
		}
		if err := f.Close(); err != nil {
			return result, fmt.Errorf("close enrichment artifact %s: %w", artifact.Path, err)
		}
		result.Written = append(result.Written, artifact.Path)
	}
	sort.Strings(result.Written)
	return result, nil
}

func RenderPatch(plan Plan) (string, error) {
	var out strings.Builder
	for _, artifact := range plan.Artifacts {
		if artifact.Action != ActionCreate {
			continue
		}
		body, err := MarshalArtifactBody(artifact.Body, true)
		if err != nil {
			return "", err
		}
		text := strings.TrimSuffix(string(body), "\n")
		lines := []string{}
		if text != "" {
			lines = strings.Split(text, "\n")
		}
		fmt.Fprintf(&out, "diff --git a/%s b/%s\n", artifact.Path, artifact.Path)
		fmt.Fprintln(&out, "new file mode 100644")
		fmt.Fprintln(&out, "index 0000000..0000000")
		fmt.Fprintln(&out, "--- /dev/null")
		fmt.Fprintf(&out, "+++ b/%s\n", artifact.Path)
		fmt.Fprintf(&out, "@@ -0,0 +1,%d @@\n", len(lines))
		for _, line := range lines {
			fmt.Fprintf(&out, "+%s\n", line)
		}
	}
	return out.String(), nil
}

func MarshalArtifactBody(body ArtifactBody, pretty bool) ([]byte, error) {
	var raw []byte
	var err error
	if pretty {
		raw, err = json.MarshalIndent(body, "", "  ")
	} else {
		raw, err = json.Marshal(body)
	}
	if err != nil {
		return nil, fmt.Errorf("marshal enrichment artifact: %w", err)
	}
	raw = append(raw, '\n')
	return raw, nil
}

func buildArtifactBody(imported gitopsflow.ImportFlowResult, record generatorIndexRecord, prov model.ProvenanceRecord, dry []model.DryInputRef, wet []model.WetManifestTarget) ArtifactBody {
	sourceLinks := buildSourceLinks(dry, prov.Sources)
	ownershipLabels := buildOwnershipLabels(dry, wet)
	routeBadges := buildRouteBadges(prov.InverseEditPointers)
	prMRLinks := buildPRMRLinks(prov.ChangeID)
	annotations := buildProposedAnnotations(record, prov, sourceLinks, ownershipLabels, routeBadges, prMRLinks)

	return ArtifactBody{
		SchemaVersion: ArtifactSchemaVersion,
		Generator: GeneratorRef{
			ID:          record.id,
			Kind:        record.kind,
			Profile:     record.profile,
			Name:        record.name,
			Root:        record.root,
			InputDigest: prov.InputDigest,
			ChangeID:    prov.ChangeID,
		},
		Target: TargetRef{
			Space:            imported.Space,
			TargetSlug:       imported.TargetSlug,
			RenderTargetSlug: imported.RenderTargetSlug,
			Ref:              imported.Ref,
		},
		Counts: ArtifactCounts{
			DryInputs:           len(dry),
			WetManifestTargets:  len(wet),
			FieldOrigins:        len(prov.FieldOriginMap),
			InverseEditPointers: len(prov.InverseEditPointers),
		},
		ProposedAnnotations: annotations,
		SourceLinks:         sourceLinks,
		OwnershipLabels:     ownershipLabels,
		RouteBadges:         routeBadges,
		PRMRLinks:           prMRLinks,
	}
}

func buildSourceLinks(dry []model.DryInputRef, sources []model.SourceRef) []SourceLink {
	uriByPath := map[string]string{}
	for _, source := range sources {
		if strings.TrimSpace(source.Path) == "" {
			continue
		}
		uriByPath[source.Path] = source.URI
	}
	out := make([]SourceLink, 0, len(dry))
	for _, input := range dry {
		out = append(out, SourceLink{
			Path:     input.Path,
			Role:     input.Role,
			Owner:    input.Owner,
			URI:      uriByPath[input.Path],
			Required: input.Required,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Owner != out[j].Owner {
			return out[i].Owner < out[j].Owner
		}
		if out[i].Role != out[j].Role {
			return out[i].Role < out[j].Role
		}
		return out[i].Path < out[j].Path
	})
	return out
}

func buildOwnershipLabels(dry []model.DryInputRef, wet []model.WetManifestTarget) []OwnershipLabel {
	out := make([]OwnershipLabel, 0, len(dry)+len(wet))
	for _, input := range dry {
		if strings.TrimSpace(input.Owner) == "" {
			continue
		}
		out = append(out, OwnershipLabel{
			Owner: input.Owner,
			Scope: "source",
			Role:  input.Role,
			Path:  input.Path,
			Label: "cub.confighub.io/owner=" + input.Owner,
		})
	}
	for _, target := range wet {
		if strings.TrimSpace(target.Owner) == "" {
			continue
		}
		out = append(out, OwnershipLabel{
			Owner:     target.Owner,
			Scope:     "rendered",
			Kind:      target.Kind,
			Name:      target.Name,
			Namespace: target.Namespace,
			Label:     "cub.confighub.io/owner=" + target.Owner,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Owner != out[j].Owner {
			return out[i].Owner < out[j].Owner
		}
		if out[i].Scope != out[j].Scope {
			return out[i].Scope < out[j].Scope
		}
		if out[i].Role != out[j].Role {
			return out[i].Role < out[j].Role
		}
		if out[i].Path != out[j].Path {
			return out[i].Path < out[j].Path
		}
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func buildRouteBadges(pointers []model.InverseEditPointer) []RouteBadge {
	out := make([]RouteBadge, 0, len(pointers))
	for _, pointer := range pointers {
		route := strings.TrimSpace(pointer.Route)
		if route == "" {
			route = "explain"
		}
		badge := route
		if strings.TrimSpace(pointer.Owner) != "" {
			badge += ":" + pointer.Owner
		}
		out = append(out, RouteBadge{
			WetPath:    pointer.WetPath,
			DryPath:    pointer.DryPath,
			Owner:      pointer.Owner,
			Route:      route,
			Badge:      badge,
			EditHint:   pointer.EditHint,
			Confidence: pointer.Confidence,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Route != out[j].Route {
			return out[i].Route < out[j].Route
		}
		if out[i].Owner != out[j].Owner {
			return out[i].Owner < out[j].Owner
		}
		if out[i].WetPath != out[j].WetPath {
			return out[i].WetPath < out[j].WetPath
		}
		return out[i].DryPath < out[j].DryPath
	})
	return out
}

func buildPRMRLinks(changeID string) []PRMRLink {
	if strings.TrimSpace(changeID) == "" {
		return nil
	}
	return []PRMRLink{{
		ChangeID:         changeID,
		LinkKey:          "cub.confighub.io/change-id",
		GitHubPRState:    "not-linked",
		ConfigHubMRState: "not-linked",
		Reason:           "carry this change_id into GitHub PR and ConfigHub MR metadata to correlate review evidence",
	}}
}

func buildProposedAnnotations(record generatorIndexRecord, prov model.ProvenanceRecord, sources []SourceLink, owners []OwnershipLabel, routes []RouteBadge, links []PRMRLink) []ProposedAnnotation {
	annotations := []ProposedAnnotation{
		{
			Category:  "source-link",
			Key:       "cub.confighub.io/generator-id",
			Value:     record.id,
			AppliesTo: "sidecar",
			Reason:    "link the sidecar to the detected generator",
		},
		{
			Category:  "source-link",
			Key:       "cub.confighub.io/generator-kind",
			Value:     record.kind,
			AppliesTo: "sidecar",
			Reason:    "make the platform generator family visible to tools and reviewers",
		},
	}
	if len(sources) > 0 {
		annotations = append(annotations, ProposedAnnotation{
			Category:  "source-link",
			Key:       "cub.confighub.io/source-paths",
			Value:     joinSourcePaths(sources),
			AppliesTo: "sidecar",
			Reason:    "show which source files feed the Deployment Variant",
		})
	}
	if len(owners) > 0 {
		annotations = append(annotations, ProposedAnnotation{
			Category:  "ownership-label",
			Key:       "cub.confighub.io/owners",
			Value:     joinOwners(owners),
			AppliesTo: "sidecar",
			Reason:    "show who should review source and rendered changes",
		})
	}
	if len(routes) > 0 {
		annotations = append(annotations, ProposedAnnotation{
			Category:  "route-badge",
			Key:       "cub.confighub.io/routes",
			Value:     joinRoutes(routes),
			AppliesTo: "sidecar",
			Reason:    "show whether generated-resource edits apply here, lift upstream, overlay, or escalate",
		})
	}
	if len(links) > 0 {
		annotations = append(annotations, ProposedAnnotation{
			Category:  "pr-mr-link",
			Key:       "cub.confighub.io/change-id",
			Value:     prov.ChangeID,
			AppliesTo: "sidecar",
			Reason:    "let GitHub PR and ConfigHub MR metadata refer to the same change",
		})
	}
	sort.Slice(annotations, func(i, j int) bool {
		if annotations[i].Category != annotations[j].Category {
			return annotations[i].Category < annotations[j].Category
		}
		return annotations[i].Key < annotations[j].Key
	})
	return annotations
}

func generatorRecords(imported gitopsflow.ImportFlowResult) []generatorIndexRecord {
	recordsByID := map[string]generatorIndexRecord{}
	for _, resource := range imported.Discovered {
		if strings.TrimSpace(resource.GeneratorID) == "" {
			continue
		}
		recordsByID[resource.GeneratorID] = generatorIndexRecord{
			id:      resource.GeneratorID,
			kind:    resource.GeneratorKind,
			profile: resource.GeneratorProfile,
			name:    resource.ResourceName,
			root:    resource.Root,
		}
	}
	for _, contract := range imported.Contracts {
		if strings.TrimSpace(contract.GeneratorID) == "" {
			continue
		}
		record := recordsByID[contract.GeneratorID]
		record.id = contract.GeneratorID
		if record.kind == "" {
			record.kind = contract.Kind
		}
		if record.profile == "" {
			record.profile = contract.Profile
		}
		if record.name == "" {
			record.name = contract.Name
		}
		if record.root == "" {
			record.root = contract.SourcePath
		}
		recordsByID[contract.GeneratorID] = record
	}
	for _, prov := range imported.Provenance {
		if strings.TrimSpace(prov.GeneratorID) == "" {
			continue
		}
		record := recordsByID[prov.GeneratorID]
		record.id = prov.GeneratorID
		if record.profile == "" {
			record.profile = prov.GeneratorProfile
		}
		if record.name == "" {
			record.name = prov.GeneratorName
		}
		recordsByID[prov.GeneratorID] = record
	}

	records := make([]generatorIndexRecord, 0, len(recordsByID))
	for _, record := range recordsByID {
		if record.name == "" {
			record.name = record.id
		}
		records = append(records, record)
	}
	sort.Slice(records, func(i, j int) bool {
		if records[i].kind != records[j].kind {
			return records[i].kind < records[j].kind
		}
		if records[i].name != records[j].name {
			return records[i].name < records[j].name
		}
		return records[i].id < records[j].id
	})
	return records
}

func sidecarPath(record generatorIndexRecord) string {
	name := sanitizePathPart(record.kind + "-" + record.name)
	if name == "" {
		name = "generator"
	}
	return ".cub-gen/enrichment/" + name + "-" + shortID(record.id) + ".provenance.json"
}

func sanitizePathPart(value string) string {
	var out strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(value) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			out.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			out.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(out.String(), "-")
}

func shortID(id string) string {
	clean := sanitizePathPart(id)
	if len(clean) <= 12 {
		return clean
	}
	return strings.Trim(clean[:12], "-")
}

func joinSourcePaths(sources []SourceLink) string {
	values := make([]string, 0, len(sources))
	seen := map[string]struct{}{}
	for _, source := range sources {
		if _, ok := seen[source.Path]; ok {
			continue
		}
		seen[source.Path] = struct{}{}
		values = append(values, source.Path)
	}
	sort.Strings(values)
	return strings.Join(values, ",")
}

func joinOwners(labels []OwnershipLabel) string {
	values := make([]string, 0, len(labels))
	seen := map[string]struct{}{}
	for _, label := range labels {
		if _, ok := seen[label.Owner]; ok {
			continue
		}
		seen[label.Owner] = struct{}{}
		values = append(values, label.Owner)
	}
	sort.Strings(values)
	return strings.Join(values, ",")
}

func joinRoutes(routes []RouteBadge) string {
	values := make([]string, 0, len(routes))
	seen := map[string]struct{}{}
	for _, route := range routes {
		if _, ok := seen[route.Badge]; ok {
			continue
		}
		seen[route.Badge] = struct{}{}
		values = append(values, route.Badge)
	}
	sort.Strings(values)
	return strings.Join(values, ",")
}

func artifactAbsPath(root, rel string) (string, error) {
	if strings.TrimSpace(root) == "" {
		return "", errors.New("target path is required")
	}
	if filepath.IsAbs(rel) {
		return "", fmt.Errorf("enrichment artifact path must be relative: %s", rel)
	}
	clean := filepath.Clean(filepath.FromSlash(rel))
	if clean == "." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || clean == ".." {
		return "", fmt.Errorf("enrichment artifact path escapes target: %s", rel)
	}
	return filepath.Join(root, clean), nil
}
