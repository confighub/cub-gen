package normalize

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	gitopsflow "github.com/confighub/cub-gen/internal/gitops"
	"github.com/confighub/cub-gen/internal/springboot"
	"gopkg.in/yaml.v3"
)

const (
	PlanSchemaVersion = "cub.confighub.io/normalize-plan/v1"

	TransformRoutePolicy    = "add-route-policy-annotation"
	TransformLiftUpstream   = "lift-generated-patch-to-source"
	TransformVariantSplit   = "split-env-values-into-variants"
	TransformOwnerLabels    = "add-missing-owners"
	TransformSecretRefs     = "replace-implicit-secret-wiring"
	ActionCreate            = "create"
	ActionReviewPatch       = "review-patch"
	RiskLow                 = "low"
	RiskMedium              = "medium"
	RiskHigh                = "high"
	routePolicyAnnotation   = "confighub.io/generator-route-policy"
	routePolicyRequiredAnno = "confighub.io/generator-route-policy-required"
)

// Plan is a read-only set of governed rewrite proposals. It never means cub-gen
// has mutated app, platform, or rendered manifests.
type Plan struct {
	SchemaVersion    string       `json:"schema_version"`
	Space            string       `json:"space"`
	TargetSlug       string       `json:"target_slug"`
	TargetPath       string       `json:"target_path,omitempty"`
	RenderTargetSlug string       `json:"render_target_slug"`
	RenderTargetPath string       `json:"render_target_path,omitempty"`
	Ref              string       `json:"ref"`
	GeneratedAt      string       `json:"generated_at"`
	Summary          Summary      `json:"summary"`
	PatchSet         PatchSet     `json:"patch_set"`
	Diagnostics      []Diagnostic `json:"diagnostics,omitempty"`
}

type Summary struct {
	ProposalCount   int            `json:"proposal_count"`
	PatchCount      int            `json:"patch_count"`
	TransformCount  int            `json:"transform_count"`
	RiskCounts      map[string]int `json:"risk_counts,omitempty"`
	DiagnosticCount int            `json:"diagnostic_count"`
}

type PatchSet struct {
	ID         string     `json:"id"`
	Title      string     `json:"title"`
	ReviewMode string     `json:"review_mode"`
	Proposals  []Proposal `json:"proposals"`
}

type Proposal struct {
	ID             string           `json:"id"`
	Transform      string           `json:"transform"`
	Title          string           `json:"title"`
	SourcePath     string           `json:"source_path"`
	Owner          string           `json:"owner"`
	Risk           string           `json:"risk"`
	Review         string           `json:"review"`
	Why            string           `json:"why"`
	RenderedImpact []RenderedImpact `json:"rendered_impact"`
	Patch          Patch            `json:"patch"`
}

type RenderedImpact struct {
	ResourceType string `json:"resource_type"`
	ResourceName string `json:"resource_name,omitempty"`
	Path         string `json:"path"`
	Action       string `json:"action"`
	Explanation  string `json:"explanation"`
}

type Patch struct {
	Path        string `json:"path"`
	Action      string `json:"action"`
	ContentType string `json:"content_type"`
	Content     string `json:"content"`
}

type Diagnostic struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Path    string `json:"path,omitempty"`
}

type manifestDoc struct {
	RelPath    string            `yaml:"-"`
	APIVersion string            `yaml:"apiVersion"`
	Kind       string            `yaml:"kind"`
	Metadata   manifestMetadata  `yaml:"metadata"`
	Data       map[string]string `yaml:"data"`
	Spec       deploymentSpec    `yaml:"spec"`
}

type manifestMetadata struct {
	Name        string            `yaml:"name"`
	Namespace   string            `yaml:"namespace"`
	Annotations map[string]string `yaml:"annotations"`
	Labels      map[string]string `yaml:"labels"`
}

type deploymentSpec struct {
	Replicas *int             `yaml:"replicas"`
	Template deploymentPodTpl `yaml:"template"`
}

type deploymentPodTpl struct {
	Spec deploymentPodSpec `yaml:"spec"`
}

type deploymentPodSpec struct {
	Containers []containerSpec `yaml:"containers"`
}

type containerSpec struct {
	Name string    `yaml:"name"`
	Env  []envSpec `yaml:"env"`
}

type envSpec struct {
	Name      string         `yaml:"name"`
	Value     string         `yaml:"value"`
	ValueFrom map[string]any `yaml:"valueFrom"`
}

type routePolicyProposal struct {
	SchemaVersion string            `json:"schema_version"`
	AppliesTo     annotationTarget  `json:"applies_to"`
	Annotations   map[string]string `json:"annotations"`
	Policy        routePolicy       `json:"policy"`
}

type annotationTarget struct {
	Space      string `json:"space"`
	TargetSlug string `json:"target_slug"`
}

type routePolicy struct {
	SchemaVersion string            `json:"schema_version"`
	Routes        []routePolicyRule `json:"routes"`
}

type routePolicyRule struct {
	ResourceType string `json:"resource_type,omitempty"`
	ResourceName string `json:"resource_name,omitempty"`
	Path         string `json:"path,omitempty"`
	WetPath      string `json:"wet_path,omitempty"`
	Route        string `json:"route"`
	Owner        string `json:"owner"`
	SourcePath   string `json:"source_path"`
	ProposalHint string `json:"proposal_hint"`
}

type liftUpstreamProposal struct {
	SchemaVersion string            `json:"schema_version"`
	Routes        []liftUpstreamRow `json:"routes"`
}

type liftUpstreamRow struct {
	FieldPattern     string   `json:"field_pattern"`
	Owner            string   `json:"owner"`
	Reason           string   `json:"reason"`
	SourceCandidates []string `json:"source_candidates"`
	RenderedPaths    []string `json:"rendered_paths"`
}

type variantPlan struct {
	SchemaVersion string       `yaml:"schema_version"`
	Component     string       `yaml:"component"`
	Variants      []variantRow `yaml:"variants"`
}

type variantRow struct {
	Name          string   `yaml:"name"`
	SourceInputs  []string `yaml:"source_inputs"`
	RenderedProof string   `yaml:"rendered_proof,omitempty"`
}

type ownerAnnotationPlan struct {
	SchemaVersion string            `yaml:"schema_version"`
	Annotations   []ownerAnnotation `yaml:"annotations"`
}

type ownerAnnotation struct {
	Scope      string `yaml:"scope"`
	Path       string `yaml:"path,omitempty"`
	Kind       string `yaml:"kind,omitempty"`
	Name       string `yaml:"name,omitempty"`
	Namespace  string `yaml:"namespace,omitempty"`
	Owner      string `yaml:"owner"`
	Annotation string `yaml:"annotation"`
}

type secretReferencePlan struct {
	SchemaVersion string            `yaml:"schema_version"`
	References    []secretReference `yaml:"references"`
}

type secretReference struct {
	Resource    string            `yaml:"resource"`
	Namespace   string            `yaml:"namespace,omitempty"`
	Container   string            `yaml:"container,omitempty"`
	Env         string            `yaml:"env"`
	Owner       string            `yaml:"owner"`
	Current     string            `yaml:"current"`
	ProposedRef map[string]string `yaml:"proposed_ref"`
	Review      string            `yaml:"review"`
}

// BuildPlan creates a deterministic preview-only normalize plan from an import
// result and local repo files.
func BuildPlan(imported gitopsflow.ImportFlowResult) (Plan, error) {
	targetPath := strings.TrimSpace(imported.TargetPath)
	if targetPath == "" {
		targetPath = strings.TrimSpace(imported.TargetSlug)
	}
	if targetPath == "" {
		return Plan{}, errors.New("target path is required")
	}
	root, err := filepath.Abs(targetPath)
	if err != nil {
		return Plan{}, fmt.Errorf("resolve target path: %w", err)
	}

	plan := Plan{
		SchemaVersion:    PlanSchemaVersion,
		Space:            imported.Space,
		TargetSlug:       imported.TargetSlug,
		TargetPath:       imported.TargetPath,
		RenderTargetSlug: imported.RenderTargetSlug,
		RenderTargetPath: imported.RenderTargetPath,
		Ref:              imported.Ref,
		GeneratedAt:      imported.ImportedAt,
		PatchSet: PatchSet{
			ID:         "normalize-" + sanitizeID(imported.TargetSlug),
			Title:      "Governed config rewrite preview",
			ReviewMode: ActionReviewPatch,
		},
	}
	if plan.PatchSet.ID == "normalize-" {
		plan.PatchSet.ID = "normalize-" + sanitizeID(filepath.Base(root))
	}

	docs, err := readManifestDocs(root)
	if err != nil {
		return Plan{}, err
	}

	routes, routesPath, err := readSpringRoutes(root)
	if err != nil {
		return Plan{}, err
	}
	if routes != nil && len(routes.Routes) > 0 {
		if proposal, ok, err := buildRoutePolicyProposal(plan, *routes, routesPath, docs); err != nil {
			return Plan{}, err
		} else if ok {
			plan.PatchSet.Proposals = append(plan.PatchSet.Proposals, proposal)
		}
		if proposal, ok, err := buildLiftUpstreamProposal(*routes, routesPath, root, docs); err != nil {
			return Plan{}, err
		} else if ok {
			plan.PatchSet.Proposals = append(plan.PatchSet.Proposals, proposal)
		}
	}

	if proposal, ok, err := buildVariantProposal(root); err != nil {
		return Plan{}, err
	} else if ok {
		plan.PatchSet.Proposals = append(plan.PatchSet.Proposals, proposal)
	}

	if proposal, ok, err := buildOwnerAnnotationProposal(imported, docs); err != nil {
		return Plan{}, err
	} else if ok {
		plan.PatchSet.Proposals = append(plan.PatchSet.Proposals, proposal)
	}

	if routes != nil {
		if proposal, ok, err := buildSecretReferenceProposal(*routes, docs); err != nil {
			return Plan{}, err
		} else if ok {
			plan.PatchSet.Proposals = append(plan.PatchSet.Proposals, proposal)
		}
	}

	sort.Slice(plan.PatchSet.Proposals, func(i, j int) bool {
		return plan.PatchSet.Proposals[i].ID < plan.PatchSet.Proposals[j].ID
	})
	if len(plan.PatchSet.Proposals) == 0 {
		plan.Diagnostics = append(plan.Diagnostics, Diagnostic{
			Code:    "NO_KNOWN_PATTERNS",
			Message: "no supported normalize proposals found; preview is a no-op",
			Path:    filepath.ToSlash(targetPath),
		})
	}
	RefreshSummary(&plan)
	return plan, nil
}

func RefreshSummary(plan *Plan) {
	if plan == nil {
		return
	}
	risks := map[string]int{}
	transforms := map[string]bool{}
	patches := 0
	for _, proposal := range plan.PatchSet.Proposals {
		transforms[proposal.Transform] = true
		if proposal.Risk != "" {
			risks[proposal.Risk]++
		}
		if proposal.Patch.Path != "" {
			patches++
		}
	}
	plan.Summary = Summary{
		ProposalCount:   len(plan.PatchSet.Proposals),
		PatchCount:      patches,
		TransformCount:  len(transforms),
		RiskCounts:      risks,
		DiagnosticCount: len(plan.Diagnostics),
	}
	if len(plan.Summary.RiskCounts) == 0 {
		plan.Summary.RiskCounts = nil
	}
}

func RenderPatch(plan Plan) (string, error) {
	var out strings.Builder
	for _, proposal := range plan.PatchSet.Proposals {
		patch := proposal.Patch
		if patch.Action != ActionCreate || strings.TrimSpace(patch.Path) == "" {
			continue
		}
		text := strings.TrimSuffix(patch.Content, "\n")
		lines := []string{}
		if text != "" {
			lines = strings.Split(text, "\n")
		}
		fmt.Fprintf(&out, "diff --git a/%s b/%s\n", patch.Path, patch.Path)
		fmt.Fprintln(&out, "new file mode 100644")
		fmt.Fprintln(&out, "index 0000000..0000000")
		fmt.Fprintln(&out, "--- /dev/null")
		fmt.Fprintf(&out, "+++ b/%s\n", patch.Path)
		fmt.Fprintf(&out, "@@ -0,0 +1,%d @@\n", len(lines))
		for _, line := range lines {
			fmt.Fprintf(&out, "+%s\n", line)
		}
	}
	return out.String(), nil
}

func buildRoutePolicyProposal(plan Plan, routes springboot.FieldRoutes, sourcePath string, docs []manifestDoc) (Proposal, bool, error) {
	configMapName := firstResourceName(docs, "ConfigMap")
	deploymentName := firstResourceName(docs, "Deployment")
	rules := make([]routePolicyRule, 0, len(routes.Routes)*2)
	for _, route := range routes.Routes {
		match := strings.TrimSpace(route.Match)
		owner := strings.TrimSpace(route.Owner)
		if match == "" || owner == "" {
			continue
		}
		policyRoute := policyRouteFor(route.DefaultAction)
		if policyRoute == "" {
			continue
		}
		hint := proposalHintFor(route)
		if strings.HasPrefix(match, "securityContext.") {
			rules = append(rules, routePolicyRule{
				ResourceType: "Deployment",
				ResourceName: deploymentName,
				Path:         "spec.template.spec." + match,
				WetPath:      "Deployment/spec/template/spec/" + strings.ReplaceAll(match, ".", "/"),
				Route:        policyRoute,
				Owner:        owner,
				SourcePath:   sourcePath,
				ProposalHint: hint,
			})
			continue
		}
		rules = append(rules, routePolicyRule{
			ResourceType: "ConfigMap",
			ResourceName: configMapName,
			Path:         "data.application*.yaml:" + match,
			WetPath:      "ConfigMap/data/application*.yaml:" + match,
			Route:        policyRoute,
			Owner:        owner,
			SourcePath:   sourcePath,
			ProposalHint: hint,
		})
		if envPattern := envPatternForFieldPattern(match); envPattern != "" && deploymentName != "" {
			rules = append(rules, routePolicyRule{
				ResourceType: "Deployment",
				ResourceName: deploymentName,
				Path:         "spec.template.spec.containers[*].env[" + envPattern + "].value",
				WetPath:      "Deployment/spec/template/spec/containers[*]/env[" + envPattern + "]/value",
				Route:        policyRoute,
				Owner:        owner,
				SourcePath:   sourcePath,
				ProposalHint: hint,
			})
		}
	}
	if len(rules) == 0 {
		return Proposal{}, false, nil
	}
	sort.Slice(rules, func(i, j int) bool {
		if rules[i].Route != rules[j].Route {
			return rules[i].Route < rules[j].Route
		}
		if rules[i].Owner != rules[j].Owner {
			return rules[i].Owner < rules[j].Owner
		}
		if rules[i].ResourceType != rules[j].ResourceType {
			return rules[i].ResourceType < rules[j].ResourceType
		}
		if rules[i].ResourceName != rules[j].ResourceName {
			return rules[i].ResourceName < rules[j].ResourceName
		}
		return rules[i].Path < rules[j].Path
	})
	policy := routePolicy{
		SchemaVersion: "cub.confighub.io/generator-route-policy/v1",
		Routes:        rules,
	}
	compactPolicy, err := json.Marshal(policy)
	if err != nil {
		return Proposal{}, false, fmt.Errorf("marshal route policy: %w", err)
	}
	body := routePolicyProposal{
		SchemaVersion: "cub.confighub.io/unit-annotation-proposal/v1",
		AppliesTo: annotationTarget{
			Space:      plan.Space,
			TargetSlug: plan.TargetSlug,
		},
		Annotations: map[string]string{
			routePolicyRequiredAnno: "true",
			routePolicyAnnotation:   string(compactPolicy),
		},
		Policy: policy,
	}
	content, err := marshalJSON(body)
	if err != nil {
		return Proposal{}, false, err
	}
	return Proposal{
		ID:         "01-route-policy-annotation",
		Transform:  TransformRoutePolicy,
		Title:      "Add ConfigHub route-policy annotations",
		SourcePath: sourcePath,
		Owner:      joinSorted(uniqueRouteOwners(routes.Routes)),
		Risk:       RiskMedium,
		Review:     "Apply these annotations to the ConfigHub Unit after review; do not mutate rendered Kubernetes YAML directly.",
		Why:        "field-routes.yaml already says which app config paths are apply-here, lift-upstream, or generator-owned; this proposal makes that boundary enforceable by ConfigHub.",
		RenderedImpact: []RenderedImpact{{
			ResourceType: "ConfigHub Unit",
			ResourceName: plan.TargetSlug,
			Path:         "metadata.annotations[" + routePolicyAnnotation + "]",
			Action:       "gate-future-mutations",
			Explanation:  "future rendered-config edits can be routed to apply-here, lift-upstream, or generator-owned decisions before a Unit revision is accepted",
		}},
		Patch: Patch{
			Path:        ".cub-gen/normalize/confighub-route-policy.annotation.json",
			Action:      ActionCreate,
			ContentType: "application/json",
			Content:     content,
		},
	}, true, nil
}

func buildLiftUpstreamProposal(routes springboot.FieldRoutes, sourcePath, root string, docs []manifestDoc) (Proposal, bool, error) {
	candidates := springSourceCandidates(root)
	rows := []liftUpstreamRow{}
	for _, route := range routes.Routes {
		if route.DefaultAction != springboot.ActionLiftUpstream {
			continue
		}
		rows = append(rows, liftUpstreamRow{
			FieldPattern:     route.Match,
			Owner:            route.Owner,
			Reason:           route.Reason,
			SourceCandidates: candidates,
			RenderedPaths:    renderedPathsForRoute(route, docs),
		})
	}
	if len(rows) == 0 {
		return Proposal{}, false, nil
	}
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].FieldPattern < rows[j].FieldPattern
	})
	content, err := marshalJSON(liftUpstreamProposal{
		SchemaVersion: "cub.confighub.io/lift-upstream-proposal/v1",
		Routes:        rows,
	})
	if err != nil {
		return Proposal{}, false, err
	}
	return Proposal{
		ID:         "02-lift-upstream-routes",
		Transform:  TransformLiftUpstream,
		Title:      "Route generated patches back to source",
		SourcePath: sourcePath,
		Owner:      joinSorted(uniqueLiftOwners(routes.Routes)),
		Risk:       RiskMedium,
		Review:     "Use this as a PR checklist when a generated ConfigMap or Deployment edit should become an upstream app-source change.",
		Why:        "lift-upstream fields are not durable as one-off rendered edits; the source repo must carry the intent.",
		RenderedImpact: []RenderedImpact{{
			ResourceType: "ConfigMap/Deployment",
			Path:         "fields matching lift-upstream routes",
			Action:       "convert-rendered-edit-to-source-pr",
			Explanation:  "reviewers get the source candidate files before accepting a rendered change that would otherwise be overwritten",
		}},
		Patch: Patch{
			Path:        ".cub-gen/normalize/lift-upstream.proposals.json",
			Action:      ActionCreate,
			ContentType: "application/json",
			Content:     content,
		},
	}, true, nil
}

func buildVariantProposal(root string) (Proposal, bool, error) {
	profilePaths, err := filepath.Glob(filepath.Join(root, "src", "main", "resources", "application-*.yaml"))
	if err != nil {
		return Proposal{}, false, fmt.Errorf("scan spring profiles: %w", err)
	}
	if len(profilePaths) == 0 {
		return Proposal{}, false, nil
	}
	sort.Strings(profilePaths)
	component := componentName(root)
	rows := []variantRow{}
	for _, profilePath := range profilePaths {
		env := strings.TrimSuffix(strings.TrimPrefix(filepath.Base(profilePath), "application-"), ".yaml")
		if env == "" {
			continue
		}
		relProfile := mustRel(root, profilePath)
		rendered := findRenderedProofForEnv(root, env)
		row := variantRow{
			Name:          env,
			SourceInputs:  []string{relProfile},
			RenderedProof: rendered,
		}
		rows = append(rows, row)
	}
	if len(rows) < 2 {
		return Proposal{}, false, nil
	}
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].Name < rows[j].Name
	})
	content, err := marshalYAML(variantPlan{
		SchemaVersion: "cub.confighub.io/deployable-variant-plan/v1",
		Component:     component,
		Variants:      rows,
	})
	if err != nil {
		return Proposal{}, false, err
	}
	return Proposal{
		ID:         "03-deployable-variants",
		Transform:  TransformVariantSplit,
		Title:      "Make environment variants explicit",
		SourcePath: "src/main/resources/application-*.yaml",
		Owner:      "app-team",
		Risk:       RiskLow,
		Review:     "Review the generated variant inventory before wiring it into platform import or fanout.",
		Why:        "profile files already encode Deployment Variants; making the variants explicit lets ConfigHub reason about dev, stage, and prod separately.",
		RenderedImpact: []RenderedImpact{{
			ResourceType: "Deployment Variant",
			ResourceName: component,
			Path:         "variants[*]",
			Action:       "make-variant-boundaries-visible",
			Explanation:  "each profile gets a concrete Deployment Variant with its source inputs and rendered proof path",
		}},
		Patch: Patch{
			Path:        ".cub-gen/normalize/deployable-variants.yaml",
			Action:      ActionCreate,
			ContentType: "application/yaml",
			Content:     content,
		},
	}, true, nil
}

func buildOwnerAnnotationProposal(imported gitopsflow.ImportFlowResult, docs []manifestDoc) (Proposal, bool, error) {
	annotations := []ownerAnnotation{}
	seen := map[string]bool{}
	for _, input := range imported.DryInputs {
		owner := strings.TrimSpace(input.Owner)
		p := strings.TrimSpace(input.Path)
		if owner == "" || p == "" {
			continue
		}
		key := "source|" + p + "|" + owner
		if seen[key] {
			continue
		}
		seen[key] = true
		annotations = append(annotations, ownerAnnotation{
			Scope:      "source",
			Path:       p,
			Owner:      owner,
			Annotation: "cub.confighub.io/owner=" + owner,
		})
	}
	renderedOwnersByKind, defaultRenderedOwner := renderedOwnerHints(imported)
	if len(docs) > 0 {
		for _, doc := range docs {
			if strings.TrimSpace(doc.Kind) == "" || strings.TrimSpace(doc.Metadata.Name) == "" {
				continue
			}
			owner := renderedOwnersByKind[doc.Kind]
			if owner == "" {
				owner = defaultRenderedOwner
			}
			if owner == "" {
				continue
			}
			key := "rendered|" + doc.Kind + "|" + doc.Metadata.Namespace + "|" + doc.Metadata.Name + "|" + owner
			if seen[key] {
				continue
			}
			seen[key] = true
			annotations = append(annotations, ownerAnnotation{
				Scope:      "rendered",
				Kind:       doc.Kind,
				Name:       doc.Metadata.Name,
				Namespace:  doc.Metadata.Namespace,
				Owner:      owner,
				Annotation: "cub.confighub.io/owner=" + owner,
			})
		}
	}
	for _, target := range imported.WetManifestTargets {
		owner := strings.TrimSpace(target.Owner)
		if owner == "" || strings.TrimSpace(target.Kind) == "" || strings.TrimSpace(target.Name) == "" {
			continue
		}
		key := "rendered|" + target.Kind + "|" + target.Namespace + "|" + target.Name + "|" + owner
		if seen[key] {
			continue
		}
		if len(docs) > 0 && hasRenderedKind(docs, target.Kind) {
			continue
		}
		seen[key] = true
		annotations = append(annotations, ownerAnnotation{
			Scope:      "rendered",
			Kind:       target.Kind,
			Name:       target.Name,
			Namespace:  target.Namespace,
			Owner:      owner,
			Annotation: "cub.confighub.io/owner=" + owner,
		})
	}
	if len(annotations) == 0 {
		return Proposal{}, false, nil
	}
	sort.Slice(annotations, func(i, j int) bool {
		if annotations[i].Scope != annotations[j].Scope {
			return annotations[i].Scope < annotations[j].Scope
		}
		if annotations[i].Owner != annotations[j].Owner {
			return annotations[i].Owner < annotations[j].Owner
		}
		if annotations[i].Path != annotations[j].Path {
			return annotations[i].Path < annotations[j].Path
		}
		if annotations[i].Kind != annotations[j].Kind {
			return annotations[i].Kind < annotations[j].Kind
		}
		return annotations[i].Name < annotations[j].Name
	})
	content, err := marshalYAML(ownerAnnotationPlan{
		SchemaVersion: "cub.confighub.io/owner-annotation-plan/v1",
		Annotations:   annotations,
	})
	if err != nil {
		return Proposal{}, false, err
	}
	return Proposal{
		ID:         "04-owner-annotations",
		Transform:  TransformOwnerLabels,
		Title:      "Add missing owner annotations",
		SourcePath: "imported provenance",
		Owner:      joinSorted(uniqueAnnotationOwners(annotations)),
		Risk:       RiskLow,
		Review:     "Review owners before applying labels or annotations to sidecars, Units, or rendered artifacts.",
		Why:        "ownership exists in the generator proof but is easy to miss in normal YAML review.",
		RenderedImpact: []RenderedImpact{{
			ResourceType: "source/rendered artifacts",
			Path:         "metadata.annotations[cub.confighub.io/owner]",
			Action:       "make-review-owner-visible",
			Explanation:  "reviewers can see who owns each source input and rendered target without reading generator internals",
		}},
		Patch: Patch{
			Path:        ".cub-gen/normalize/owner-annotations.yaml",
			Action:      ActionCreate,
			ContentType: "application/yaml",
			Content:     content,
		},
	}, true, nil
}

func buildSecretReferenceProposal(routes springboot.FieldRoutes, docs []manifestDoc) (Proposal, bool, error) {
	refs := []secretReference{}
	seen := map[string]bool{}
	for _, doc := range docs {
		if doc.Kind != "Deployment" {
			continue
		}
		for _, container := range doc.Spec.Template.Spec.Containers {
			for _, env := range container.Env {
				if strings.TrimSpace(env.Value) == "" || len(env.ValueFrom) > 0 || !looksSecretLike(env.Name, env.Value) {
					continue
				}
				owner := routeOwnerForEnv(routes, env.Name)
				if owner == "" {
					owner = "platform-engineering"
				}
				key := doc.Kind + "/" + doc.Metadata.Namespace + "/" + doc.Metadata.Name + "/" + container.Name + "/" + env.Name
				if seen[key] {
					continue
				}
				seen[key] = true
				refs = append(refs, secretReference{
					Resource:  doc.Kind + "/" + doc.Metadata.Name,
					Namespace: doc.Metadata.Namespace,
					Container: container.Name,
					Env:       env.Name,
					Owner:     owner,
					Current:   "literal-value",
					ProposedRef: map[string]string{
						"secret_name": secretNameFor(doc.Metadata.Name, env.Name),
						"key":         secretKeyFor(env.Name),
					},
					Review: "replace env.value with env.valueFrom.secretKeyRef only after the secret lifecycle is owned and audited",
				})
			}
		}
	}
	if len(refs) == 0 {
		return Proposal{}, false, nil
	}
	sort.Slice(refs, func(i, j int) bool {
		if refs[i].Resource != refs[j].Resource {
			return refs[i].Resource < refs[j].Resource
		}
		return refs[i].Env < refs[j].Env
	})
	content, err := marshalYAML(secretReferencePlan{
		SchemaVersion: "cub.confighub.io/secret-reference-proposal/v1",
		References:    refs,
	})
	if err != nil {
		return Proposal{}, false, err
	}
	return Proposal{
		ID:         "05-secret-references",
		Transform:  TransformSecretRefs,
		Title:      "Replace implicit secret wiring with explicit references",
		SourcePath: "rendered Deployment env",
		Owner:      joinSorted(uniqueSecretOwners(refs)),
		Risk:       RiskHigh,
		Review:     "Do not apply automatically; confirm the secret owner, key name, and rotation path first.",
		Why:        "literal datasource, token, password, or secret-shaped env values should have an explicit SecretReference contract before platform automation owns them.",
		RenderedImpact: []RenderedImpact{{
			ResourceType: "Deployment",
			Path:         "spec.template.spec.containers[*].env[*].valueFrom.secretKeyRef",
			Action:       "replace-literal-with-explicit-reference",
			Explanation:  "generated Deployments can use auditable secret references instead of implicit literal wiring",
		}},
		Patch: Patch{
			Path:        ".cub-gen/normalize/secret-references.yaml",
			Action:      ActionCreate,
			ContentType: "application/yaml",
			Content:     content,
		},
	}, true, nil
}

func readSpringRoutes(root string) (*springboot.FieldRoutes, string, error) {
	rel := filepath.ToSlash(filepath.Join("operational", "field-routes.yaml"))
	p := filepath.Join(root, filepath.FromSlash(rel))
	raw, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, "", nil
		}
		return nil, "", fmt.Errorf("read field routes: %w", err)
	}
	var routes springboot.FieldRoutes
	if err := yaml.Unmarshal(raw, &routes); err != nil {
		return nil, "", fmt.Errorf("parse field routes: %w", err)
	}
	return &routes, rel, nil
}

func readManifestDocs(root string) ([]manifestDoc, error) {
	patterns := []string{
		filepath.Join(root, "operational", "*.yaml"),
		filepath.Join(root, "confighub", "*.yaml"),
	}
	paths := []string{}
	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("scan manifests: %w", err)
		}
		paths = append(paths, matches...)
	}
	sort.Strings(paths)
	docs := []manifestDoc{}
	for _, p := range paths {
		raw, err := os.ReadFile(p)
		if err != nil {
			return nil, fmt.Errorf("read manifest %s: %w", p, err)
		}
		rel := mustRel(root, p)
		dec := yaml.NewDecoder(bytes.NewReader(raw))
		for {
			var doc manifestDoc
			err := dec.Decode(&doc)
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				return nil, fmt.Errorf("parse manifest %s: %w", rel, err)
			}
			if strings.TrimSpace(doc.Kind) == "" {
				continue
			}
			doc.RelPath = rel
			docs = append(docs, doc)
		}
	}
	return docs, nil
}

func policyRouteFor(action springboot.RouteAction) string {
	switch action {
	case springboot.ActionMutableInCH:
		return "apply-here"
	case springboot.ActionLiftUpstream:
		return "lift-upstream"
	case springboot.ActionGeneratorOwned:
		return "generator-owned"
	default:
		return ""
	}
}

func proposalHintFor(route springboot.FieldRoute) string {
	switch route.DefaultAction {
	case springboot.ActionMutableInCH:
		return "allow local ConfigHub mutation and preserve it across generator refresh"
	case springboot.ActionLiftUpstream:
		return "create a source PR before accepting this as a durable change"
	case springboot.ActionGeneratorOwned:
		return "block direct mutation and escalate to the platform owner"
	default:
		return strings.TrimSpace(route.Reason)
	}
}

func renderedPathsForRoute(route springboot.FieldRoute, docs []manifestDoc) []string {
	match := strings.TrimSpace(route.Match)
	if match == "" {
		return nil
	}
	out := []string{"ConfigMap/data/application*.yaml:" + match}
	if envPattern := envPatternForFieldPattern(match); envPattern != "" && firstResourceName(docs, "Deployment") != "" {
		out = append(out, "Deployment/spec/template/spec/containers[*]/env["+envPattern+"]/value")
	}
	sort.Strings(out)
	return out
}

func firstResourceName(docs []manifestDoc, kind string) string {
	for _, doc := range docs {
		if doc.Kind == kind && strings.TrimSpace(doc.Metadata.Name) != "" {
			return doc.Metadata.Name
		}
	}
	return ""
}

func hasRenderedKind(docs []manifestDoc, kind string) bool {
	for _, doc := range docs {
		if doc.Kind == kind {
			return true
		}
	}
	return false
}

func renderedOwnerHints(imported gitopsflow.ImportFlowResult) (map[string]string, string) {
	ownersByKind := map[string]string{}
	counts := map[string]int{}
	for _, target := range imported.WetManifestTargets {
		kind := strings.TrimSpace(target.Kind)
		owner := strings.TrimSpace(target.Owner)
		if kind == "" || owner == "" {
			continue
		}
		if ownersByKind[kind] == "" {
			ownersByKind[kind] = owner
		}
		counts[owner]++
	}
	defaultOwner := ""
	defaultCount := 0
	for owner, count := range counts {
		if count > defaultCount || (count == defaultCount && owner < defaultOwner) {
			defaultOwner = owner
			defaultCount = count
		}
	}
	return ownersByKind, defaultOwner
}

func springSourceCandidates(root string) []string {
	patterns := []string{
		filepath.Join(root, "src", "main", "resources", "application.yaml"),
		filepath.Join(root, "src", "main", "resources", "application-*.yaml"),
	}
	var out []string
	for _, pattern := range patterns {
		matches, _ := filepath.Glob(pattern)
		sort.Strings(matches)
		for _, match := range matches {
			out = append(out, mustRel(root, match))
		}
	}
	return uniqueStrings(out)
}

func componentName(root string) string {
	appPath := filepath.Join(root, "src", "main", "resources", "application.yaml")
	raw, err := os.ReadFile(appPath)
	if err == nil {
		var doc map[string]any
		if yaml.Unmarshal(raw, &doc) == nil {
			if name := nestedString(doc, "spring", "application", "name"); name != "" {
				return name
			}
		}
	}
	return filepath.Base(root)
}

func nestedString(m map[string]any, keys ...string) string {
	var cur any = m
	for _, key := range keys {
		next, ok := cur.(map[string]any)
		if !ok {
			return ""
		}
		cur, ok = next[key]
		if !ok {
			return ""
		}
	}
	if s, ok := cur.(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}

func findRenderedProofForEnv(root, env string) string {
	matches, _ := filepath.Glob(filepath.Join(root, "confighub", "*-"+env+".yaml"))
	sort.Strings(matches)
	if len(matches) == 0 {
		return ""
	}
	return mustRel(root, matches[0])
}

func envPatternForFieldPattern(pattern string) string {
	p := strings.TrimSpace(pattern)
	if p == "" {
		return ""
	}
	p = strings.TrimSuffix(p, ".*")
	p = strings.ReplaceAll(p, ".", "_")
	p = strings.ToUpper(p)
	if strings.HasSuffix(pattern, ".*") {
		return p + "_*"
	}
	return p
}

func routeOwnerForEnv(routes springboot.FieldRoutes, envName string) string {
	field := strings.ToLower(strings.ReplaceAll(envName, "_", "."))
	for _, route := range routes.Routes {
		ok, err := path.Match(strings.TrimSpace(route.Match), field)
		if err == nil && ok {
			return strings.TrimSpace(route.Owner)
		}
	}
	return ""
}

func looksSecretLike(name, value string) bool {
	upperName := strings.ToUpper(strings.TrimSpace(name))
	lowerValue := strings.ToLower(strings.TrimSpace(value))
	if strings.Contains(upperName, "PASSWORD") || strings.Contains(upperName, "TOKEN") || strings.Contains(upperName, "SECRET") {
		return true
	}
	if strings.Contains(upperName, "DATASOURCE") && strings.Contains(lowerValue, "jdbc:") {
		return true
	}
	return false
}

func secretNameFor(resourceName, envName string) string {
	base := strings.ToLower(resourceName)
	base = strings.Trim(base, "-_ .")
	env := strings.ToLower(envName)
	switch {
	case strings.Contains(env, "datasource"):
		return base + "-datasource"
	case strings.Contains(env, "token"):
		return base + "-token"
	case strings.Contains(env, "password"):
		return base + "-password"
	default:
		return base + "-secret"
	}
}

func secretKeyFor(envName string) string {
	env := strings.ToLower(envName)
	parts := strings.Split(env, "_")
	if len(parts) > 0 {
		last := parts[len(parts)-1]
		if last != "" {
			return strings.ReplaceAll(last, "-", "_")
		}
	}
	return "value"
}

func uniqueRouteOwners(routes []springboot.FieldRoute) []string {
	out := []string{}
	for _, route := range routes {
		if owner := strings.TrimSpace(route.Owner); owner != "" {
			out = append(out, owner)
		}
	}
	return uniqueStrings(out)
}

func uniqueLiftOwners(routes []springboot.FieldRoute) []string {
	out := []string{}
	for _, route := range routes {
		if route.DefaultAction == springboot.ActionLiftUpstream && strings.TrimSpace(route.Owner) != "" {
			out = append(out, route.Owner)
		}
	}
	return uniqueStrings(out)
}

func uniqueAnnotationOwners(annotations []ownerAnnotation) []string {
	out := []string{}
	for _, annotation := range annotations {
		if annotation.Owner != "" {
			out = append(out, annotation.Owner)
		}
	}
	return uniqueStrings(out)
}

func uniqueSecretOwners(refs []secretReference) []string {
	out := []string{}
	for _, ref := range refs {
		if ref.Owner != "" {
			out = append(out, ref.Owner)
		}
	}
	return uniqueStrings(out)
}

func uniqueStrings(in []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, raw := range in {
		s := strings.TrimSpace(raw)
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

func joinSorted(values []string) string {
	values = uniqueStrings(values)
	return strings.Join(values, ",")
}

func sanitizeID(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "\\", "/")
	value = strings.Trim(value, "/")
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		ok := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if ok {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func mustRel(root, p string) string {
	rel, err := filepath.Rel(root, p)
	if err != nil {
		return filepath.ToSlash(p)
	}
	return filepath.ToSlash(rel)
}

func marshalJSON(v any) (string, error) {
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal normalize proposal: %w", err)
	}
	return string(append(raw, '\n')), nil
}

func marshalYAML(v any) (string, error) {
	raw, err := yaml.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("marshal normalize proposal: %w", err)
	}
	return string(raw), nil
}
