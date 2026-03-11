package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/confighub/cub-gen/internal/attest"
	bridgeflow "github.com/confighub/cub-gen/internal/bridge"
	"github.com/confighub/cub-gen/internal/detect"
	gitopsflow "github.com/confighub/cub-gen/internal/gitops"
	"github.com/confighub/cub-gen/internal/importer"
	"github.com/confighub/cub-gen/internal/model"
	"github.com/confighub/cub-gen/internal/publish"
	"github.com/confighub/cub-gen/internal/registry"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		printUsage(os.Stderr)
		return errors.New("command required")
	}

	switch args[0] {
	case "help", "-h", "--help":
		printUsage(os.Stdout)
		return nil
	case "detect":
		return runDetect(args[1:])
	case "import":
		return runLegacyImport(args[1:])
	case "publish":
		return runPublish(args[1:])
	case "verify":
		return runVerify(args[1:])
	case "attest":
		return runAttest(args[1:])
	case "verify-attestation":
		return runVerifyAttestation(args[1:])
	case "change":
		return runChange(args[1:])
	case "generators":
		return runGenerators(args[1:])
	case "gitops":
		return runGitOps(args[1:])
	case "bridge":
		return runBridge(args[1:])
	default:
		printUsage(os.Stderr)
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

type generatorFamilyRecord struct {
	Kind         string                       `json:"kind"`
	Profile      string                       `json:"profile"`
	ResourceKind string                       `json:"resource_kind"`
	ResourceType string                       `json:"resource_type"`
	Capabilities []string                     `json:"capabilities"`
	Policies     *generatorFamilyPolicyRecord `json:"policies,omitempty"`
}

type generatorFamilyPolicyRecord struct {
	InversePatchTemplates       map[string]inversePatchTemplateRecord   `json:"inverse_patch_templates,omitempty"`
	InversePointerTemplates     map[string]inversePointerTemplateRecord `json:"inverse_pointer_templates,omitempty"`
	FieldOriginConfidences      map[string]float64                      `json:"field_origin_confidences,omitempty"`
	HintDefaults                map[string]string                       `json:"hint_defaults,omitempty"`
	InversePatchReasons         map[string]string                       `json:"inverse_patch_reasons,omitempty"`
	InverseEditHints            map[string]string                       `json:"inverse_edit_hints,omitempty"`
	InputRoleRules              []inputRoleRuleRecord                   `json:"input_role_rules,omitempty"`
	DefaultInputRole            string                                  `json:"default_input_role,omitempty"`
	RoleOwners                  map[string]string                       `json:"role_owners,omitempty"`
	DefaultOwner                string                                  `json:"default_owner,omitempty"`
	WetTargets                  []wetTargetTemplateRecord               `json:"wet_targets,omitempty"`
	RenderedLineageTemplates    []renderedLineageTemplateRecord         `json:"rendered_lineage_templates,omitempty"`
	FieldOriginTransform        string                                  `json:"field_origin_transform,omitempty"`
	FieldOriginOverlayTransform string                                  `json:"field_origin_overlay_transform,omitempty"`
}

type inversePatchTemplateRecord struct {
	EditableBy     string  `json:"editable_by"`
	Confidence     float64 `json:"confidence"`
	RequiresReview bool    `json:"requires_review"`
}

type inversePointerTemplateRecord struct {
	Owner      string  `json:"owner"`
	Confidence float64 `json:"confidence"`
}

type inputRoleRuleRecord struct {
	Role           string   `json:"role"`
	ExactBasenames []string `json:"exact_basenames,omitempty"`
	Prefixes       []string `json:"prefixes,omitempty"`
	Extensions     []string `json:"extensions,omitempty"`
}

type wetTargetTemplateRecord struct {
	Kind                  string `json:"kind"`
	NameTemplate          string `json:"name_template"`
	Owner                 string `json:"owner"`
	Namespace             string `json:"namespace,omitempty"`
	SourceDryPathTemplate string `json:"source_dry_path_template,omitempty"`
}

type renderedLineageTemplateRecord struct {
	Kind                   string `json:"kind"`
	NameTemplate           string `json:"name_template"`
	Namespace              string `json:"namespace,omitempty"`
	SourcePathHint         string `json:"source_path_hint,omitempty"`
	SourcePathHintFallback string `json:"source_path_hint_fallback,omitempty"`
	SourcePathHintMulti    bool   `json:"source_path_hint_multi,omitempty"`
	SourceDryPathTemplate  string `json:"source_dry_path_template,omitempty"`
	Optional               bool   `json:"optional,omitempty"`
}

func runGenerators(args []string) error {
	fs := flag.NewFlagSet("generators", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.Usage = func() {
		printGeneratorsUsage(fs.Output())
	}
	kindFilter := fs.String("kind", "", "Filter by generator kind")
	profileFilter := fs.String("profile", "", "Filter by generator profile")
	capabilityFilter := fs.String("capability", "", "Filter by capability")
	strictFilters := fs.Bool("strict-filters", false, "Fail on unknown filter values")
	jsonOut := fs.Bool("json", false, "Output JSON")
	markdownOut := fs.Bool("markdown", false, "Output Markdown")
	details := fs.Bool("details", false, "Include policy/provenance template details in JSON or Markdown output")
	pretty := fs.Bool("pretty", true, "Pretty-print JSON output")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("usage: cub-gen generators [--kind KIND] [--profile PROFILE] [--capability CAPABILITY] [--strict-filters] [--json|--markdown] [--details] [--pretty]")
	}
	if *jsonOut && *markdownOut {
		return errors.New("--markdown cannot be combined with --json")
	}
	if *details && !*jsonOut && !*markdownOut {
		return errors.New("--details requires --json or --markdown")
	}
	if *strictFilters {
		if err := validateGeneratorFilters(*kindFilter, *profileFilter, *capabilityFilter); err != nil {
			return err
		}
	}

	records := listGeneratorFamilies(*kindFilter, *profileFilter, *capabilityFilter, *details)
	if *jsonOut {
		return writeJSON(os.Stdout, map[string]any{
			"count":    len(records),
			"families": records,
		}, *pretty)
	}
	if *markdownOut {
		return writeGeneratorsMarkdown(os.Stdout, records, *details)
	}

	fmt.Println("Kind\tProfile\tResource Kind\tResource Type\tCapabilities")
	for _, record := range records {
		fmt.Printf("%s\t%s\t%s\t%s\t%s\n",
			record.Kind,
			record.Profile,
			record.ResourceKind,
			record.ResourceType,
			strings.Join(record.Capabilities, ","),
		)
	}
	return nil
}

func writeGeneratorsMarkdown(out io.Writer, records []generatorFamilyRecord, details bool) error {
	fmt.Fprintln(out, "# Generator Families")
	fmt.Fprintln(out)
	fmt.Fprintf(out, "Total: %d\n\n", len(records))
	fmt.Fprintln(out, "| Kind | Profile | Resource Kind | Resource Type | Capabilities |")
	fmt.Fprintln(out, "| --- | --- | --- | --- | --- |")
	for _, record := range records {
		fmt.Fprintf(out, "| `%s` | `%s` | `%s` | `%s` | %s |\n",
			markdownCell(record.Kind),
			markdownCell(record.Profile),
			markdownCell(record.ResourceKind),
			markdownCell(record.ResourceType),
			markdownCell(strings.Join(record.Capabilities, ", ")),
		)
	}

	if !details {
		return nil
	}

	for _, record := range records {
		if record.Policies == nil {
			continue
		}
		policies := record.Policies
		fmt.Fprintln(out)
		fmt.Fprintf(out, "## `%s`\n\n", markdownCell(record.Kind))
		fmt.Fprintf(out, "- Profile: `%s`\n", markdownCell(record.Profile))
		fmt.Fprintf(out, "- Resource: `%s` (`%s`)\n", markdownCell(record.ResourceKind), markdownCell(record.ResourceType))
		fmt.Fprintf(out, "- Capabilities: %s\n", markdownCell(strings.Join(record.Capabilities, ", ")))
		if policies.DefaultInputRole != "" {
			fmt.Fprintf(out, "- Default input role: `%s`\n", markdownCell(policies.DefaultInputRole))
		}
		if policies.DefaultOwner != "" {
			fmt.Fprintf(out, "- Default owner: `%s`\n", markdownCell(policies.DefaultOwner))
		}
		if policies.FieldOriginTransform != "" {
			fmt.Fprintf(out, "- Field-origin transform: `%s`\n", markdownCell(policies.FieldOriginTransform))
		}
		if policies.FieldOriginOverlayTransform != "" {
			fmt.Fprintf(out, "- Field-origin overlay transform: `%s`\n", markdownCell(policies.FieldOriginOverlayTransform))
		}

		if len(policies.InputRoleRules) > 0 {
			fmt.Fprintln(out)
			fmt.Fprintln(out, "### Input Role Rules")
			fmt.Fprintln(out, "| Role | Exact basenames | Prefixes | Extensions |")
			fmt.Fprintln(out, "| --- | --- | --- | --- |")
			for _, rule := range policies.InputRoleRules {
				fmt.Fprintf(out, "| `%s` | %s | %s | %s |\n",
					markdownCell(rule.Role),
					markdownCell(strings.Join(rule.ExactBasenames, ", ")),
					markdownCell(strings.Join(rule.Prefixes, ", ")),
					markdownCell(strings.Join(rule.Extensions, ", ")),
				)
			}
		}

		if len(policies.RoleOwners) > 0 {
			fmt.Fprintln(out)
			fmt.Fprintln(out, "### Role Owners")
			fmt.Fprintln(out, "| Role | Owner |")
			fmt.Fprintln(out, "| --- | --- |")
			for _, key := range sortedMapKeys(policies.RoleOwners) {
				fmt.Fprintf(out, "| `%s` | `%s` |\n", markdownCell(key), markdownCell(policies.RoleOwners[key]))
			}
		}

		if len(policies.InversePatchTemplates) > 0 {
			fmt.Fprintln(out)
			fmt.Fprintln(out, "### Inverse Patch Templates")
			fmt.Fprintln(out, "| Key | Editable by | Confidence | Requires review |")
			fmt.Fprintln(out, "| --- | --- | --- | --- |")
			for _, key := range sortedMapKeys(policies.InversePatchTemplates) {
				tpl := policies.InversePatchTemplates[key]
				fmt.Fprintf(out, "| `%s` | `%s` | %.2f | `%t` |\n",
					markdownCell(key),
					markdownCell(tpl.EditableBy),
					tpl.Confidence,
					tpl.RequiresReview,
				)
			}
		}

		if len(policies.InversePointerTemplates) > 0 {
			fmt.Fprintln(out)
			fmt.Fprintln(out, "### Inverse Pointer Templates")
			fmt.Fprintln(out, "| Key | Owner | Confidence |")
			fmt.Fprintln(out, "| --- | --- | --- |")
			for _, key := range sortedMapKeys(policies.InversePointerTemplates) {
				tpl := policies.InversePointerTemplates[key]
				fmt.Fprintf(out, "| `%s` | `%s` | %.2f |\n",
					markdownCell(key),
					markdownCell(tpl.Owner),
					tpl.Confidence,
				)
			}
		}

		if len(policies.FieldOriginConfidences) > 0 {
			fmt.Fprintln(out)
			fmt.Fprintln(out, "### Field Origin Confidences")
			fmt.Fprintln(out, "| Key | Confidence |")
			fmt.Fprintln(out, "| --- | --- |")
			for _, key := range sortedMapKeys(policies.FieldOriginConfidences) {
				fmt.Fprintf(out, "| `%s` | %.2f |\n", markdownCell(key), policies.FieldOriginConfidences[key])
			}
		}

		if len(policies.HintDefaults) > 0 {
			fmt.Fprintln(out)
			fmt.Fprintln(out, "### Hint Defaults")
			fmt.Fprintln(out, "| Key | Value |")
			fmt.Fprintln(out, "| --- | --- |")
			for _, key := range sortedMapKeys(policies.HintDefaults) {
				fmt.Fprintf(out, "| `%s` | `%s` |\n", markdownCell(key), markdownCell(policies.HintDefaults[key]))
			}
		}

		if len(policies.InversePatchReasons) > 0 {
			fmt.Fprintln(out)
			fmt.Fprintln(out, "### Inverse Patch Reasons")
			fmt.Fprintln(out, "| Key | Reason |")
			fmt.Fprintln(out, "| --- | --- |")
			for _, key := range sortedMapKeys(policies.InversePatchReasons) {
				fmt.Fprintf(out, "| `%s` | %s |\n", markdownCell(key), markdownCell(policies.InversePatchReasons[key]))
			}
		}

		if len(policies.InverseEditHints) > 0 {
			fmt.Fprintln(out)
			fmt.Fprintln(out, "### Inverse Edit Hints")
			fmt.Fprintln(out, "| Key | Hint |")
			fmt.Fprintln(out, "| --- | --- |")
			for _, key := range sortedMapKeys(policies.InverseEditHints) {
				fmt.Fprintf(out, "| `%s` | %s |\n", markdownCell(key), markdownCell(policies.InverseEditHints[key]))
			}
		}

		if len(policies.WetTargets) > 0 {
			fmt.Fprintln(out)
			fmt.Fprintln(out, "### WET Targets")
			fmt.Fprintln(out, "| Kind | Name template | Owner | Namespace | Source DRY path template |")
			fmt.Fprintln(out, "| --- | --- | --- | --- | --- |")
			for _, target := range policies.WetTargets {
				fmt.Fprintf(out, "| `%s` | `%s` | `%s` | `%s` | `%s` |\n",
					markdownCell(target.Kind),
					markdownCell(target.NameTemplate),
					markdownCell(target.Owner),
					markdownCell(target.Namespace),
					markdownCell(target.SourceDryPathTemplate),
				)
			}
		}

		if len(policies.RenderedLineageTemplates) > 0 {
			fmt.Fprintln(out)
			fmt.Fprintln(out, "### Rendered Lineage Templates")
			fmt.Fprintln(out, "| Kind | Name template | Namespace | Source path hint | Hint fallback | Multi hint | Source DRY path template | Optional |")
			fmt.Fprintln(out, "| --- | --- | --- | --- | --- | --- | --- | --- |")
			for _, tpl := range policies.RenderedLineageTemplates {
				fmt.Fprintf(out, "| `%s` | `%s` | `%s` | `%s` | `%s` | `%t` | `%s` | `%t` |\n",
					markdownCell(tpl.Kind),
					markdownCell(tpl.NameTemplate),
					markdownCell(tpl.Namespace),
					markdownCell(tpl.SourcePathHint),
					markdownCell(tpl.SourcePathHintFallback),
					tpl.SourcePathHintMulti,
					markdownCell(tpl.SourceDryPathTemplate),
					tpl.Optional,
				)
			}
		}
	}

	return nil
}

func markdownCell(value string) string {
	value = strings.ReplaceAll(value, "|", "\\|")
	value = strings.ReplaceAll(value, "\n", "<br/>")
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return value
}

func sortedMapKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func listGeneratorFamilies(kindFilter, profileFilter, capabilityFilter string, details bool) []generatorFamilyRecord {
	kindFilters := parseFilterSet(kindFilter)
	profileFilters := parseFilterSet(profileFilter)
	capabilityFilters := parseFilterSet(capabilityFilter)

	kinds := registry.Kinds()
	out := make([]generatorFamilyRecord, 0, len(kinds))
	for _, kind := range kinds {
		spec, ok := registry.Spec(kind)
		if !ok {
			continue
		}
		record := generatorFamilyRecord{
			Kind:         string(spec.Kind),
			Profile:      spec.Profile,
			ResourceKind: spec.ResourceKind,
			ResourceType: spec.ResourceType,
			Capabilities: append([]string(nil), spec.Capabilities...),
		}
		if details {
			record.Policies = generatorPolicyRecord(spec)
		}

		if len(kindFilters) > 0 {
			if _, ok := kindFilters[strings.ToLower(record.Kind)]; !ok {
				continue
			}
		}
		if len(profileFilters) > 0 {
			if _, ok := profileFilters[strings.ToLower(record.Profile)]; !ok {
				continue
			}
		}
		if len(capabilityFilters) > 0 {
			matched := false
			for _, capability := range record.Capabilities {
				if _, ok := capabilityFilters[strings.ToLower(capability)]; ok {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		out = append(out, generatorFamilyRecord{
			Kind:         record.Kind,
			Profile:      record.Profile,
			ResourceKind: record.ResourceKind,
			ResourceType: record.ResourceType,
			Capabilities: record.Capabilities,
			Policies:     record.Policies,
		})
	}
	return out
}

func generatorPolicyRecord(spec registry.FamilySpec) *generatorFamilyPolicyRecord {
	policies := &generatorFamilyPolicyRecord{
		InversePatchTemplates:   map[string]inversePatchTemplateRecord{},
		InversePointerTemplates: map[string]inversePointerTemplateRecord{},
		FieldOriginConfidences:  map[string]float64{},
		HintDefaults:            map[string]string{},
		InversePatchReasons:     map[string]string{},
		InverseEditHints:        map[string]string{},
		RoleOwners:              map[string]string{},
	}
	for key, tpl := range spec.InversePatchTemplates {
		policies.InversePatchTemplates[key] = inversePatchTemplateRecord{
			EditableBy:     tpl.EditableBy,
			Confidence:     tpl.Confidence,
			RequiresReview: tpl.RequiresReview,
		}
	}
	for key, tpl := range spec.InversePointerTemplates {
		policies.InversePointerTemplates[key] = inversePointerTemplateRecord{
			Owner:      tpl.Owner,
			Confidence: tpl.Confidence,
		}
	}
	for key, confidence := range spec.FieldOriginConfidences {
		policies.FieldOriginConfidences[key] = confidence
	}
	for key, value := range spec.HintDefaults {
		policies.HintDefaults[key] = value
	}
	for key, value := range spec.InversePatchReasons {
		policies.InversePatchReasons[key] = value
	}
	for key, value := range spec.InverseEditHints {
		policies.InverseEditHints[key] = value
	}
	for key, value := range spec.RoleOwners {
		policies.RoleOwners[key] = value
	}
	policies.DefaultInputRole = spec.DefaultInputRole
	policies.DefaultOwner = spec.DefaultOwner
	policies.FieldOriginTransform = spec.FieldOriginTransform
	policies.FieldOriginOverlayTransform = spec.FieldOriginOverlayTransform
	for _, rule := range spec.InputRoleRules {
		policies.InputRoleRules = append(policies.InputRoleRules, inputRoleRuleRecord{
			Role:           rule.Role,
			ExactBasenames: append([]string(nil), rule.ExactBasenames...),
			Prefixes:       append([]string(nil), rule.Prefixes...),
			Extensions:     append([]string(nil), rule.Extensions...),
		})
	}
	for _, wet := range spec.WetTargets {
		policies.WetTargets = append(policies.WetTargets, wetTargetTemplateRecord{
			Kind:                  wet.Kind,
			NameTemplate:          wet.NameTemplate,
			Owner:                 wet.Owner,
			Namespace:             wet.Namespace,
			SourceDryPathTemplate: wet.SourceDryPathTemplate,
		})
	}
	for _, lineage := range spec.RenderedLineageTemplates {
		policies.RenderedLineageTemplates = append(policies.RenderedLineageTemplates, renderedLineageTemplateRecord{
			Kind:                   lineage.Kind,
			NameTemplate:           lineage.NameTemplate,
			Namespace:              lineage.Namespace,
			SourcePathHint:         lineage.SourcePathHint,
			SourcePathHintFallback: lineage.SourcePathHintFallback,
			SourcePathHintMulti:    lineage.SourcePathHintMulti,
			SourceDryPathTemplate:  lineage.SourceDryPathTemplate,
			Optional:               lineage.Optional,
		})
	}
	if len(policies.InversePatchTemplates) == 0 {
		policies.InversePatchTemplates = nil
	}
	if len(policies.InversePointerTemplates) == 0 {
		policies.InversePointerTemplates = nil
	}
	if len(policies.FieldOriginConfidences) == 0 {
		policies.FieldOriginConfidences = nil
	}
	if len(policies.HintDefaults) == 0 {
		policies.HintDefaults = nil
	}
	if len(policies.InversePatchReasons) == 0 {
		policies.InversePatchReasons = nil
	}
	if len(policies.InverseEditHints) == 0 {
		policies.InverseEditHints = nil
	}
	if len(policies.InputRoleRules) == 0 {
		policies.InputRoleRules = nil
	}
	if len(policies.RoleOwners) == 0 {
		policies.RoleOwners = nil
	}
	if len(policies.WetTargets) == 0 {
		policies.WetTargets = nil
	}
	if len(policies.RenderedLineageTemplates) == 0 {
		policies.RenderedLineageTemplates = nil
	}
	return policies
}

func parseFilterSet(raw string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, part := range strings.Split(raw, ",") {
		value := strings.ToLower(strings.TrimSpace(part))
		if value == "" {
			continue
		}
		out[value] = struct{}{}
	}
	return out
}

func printGeneratorsUsage(out io.Writer) {
	fmt.Fprintln(out, "Usage:")
	fmt.Fprintln(out, "  cub-gen generators [--kind KIND] [--profile PROFILE] [--capability CAPABILITY] [--strict-filters] [--json|--markdown] [--details] [--pretty]")
	fmt.Fprintln(out, "  (KIND/PROFILE/CAPABILITY support comma-separated values)")
	fmt.Fprintln(out, "  use --strict-filters to fail on unknown filter values")
	fmt.Fprintln(out, "  use --details with --json or --markdown to include policy/provenance templates")
	fmt.Fprintln(out)
	fmt.Fprintf(out, "Supported kinds: %s\n", strings.Join(supportedGeneratorKinds(), ", "))
	fmt.Fprintf(out, "Supported profiles: %s\n", strings.Join(supportedGeneratorProfiles(), ", "))
	fmt.Fprintf(out, "Supported capabilities: %s\n", strings.Join(supportedGeneratorCapabilities(), ", "))
}

func supportedGeneratorKinds() []string {
	kinds := registry.Kinds()
	out := make([]string, 0, len(kinds))
	for _, kind := range kinds {
		out = append(out, string(kind))
	}
	sort.Strings(out)
	return out
}

func supportedGeneratorProfiles() []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(registry.Kinds()))
	for _, kind := range registry.Kinds() {
		spec, ok := registry.Spec(kind)
		if !ok || strings.TrimSpace(spec.Profile) == "" {
			continue
		}
		if _, exists := seen[spec.Profile]; exists {
			continue
		}
		seen[spec.Profile] = struct{}{}
		out = append(out, spec.Profile)
	}
	sort.Strings(out)
	return out
}

func supportedGeneratorCapabilities() []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, 16)
	for _, kind := range registry.Kinds() {
		spec, ok := registry.Spec(kind)
		if !ok {
			continue
		}
		for _, capability := range spec.Capabilities {
			if strings.TrimSpace(capability) == "" {
				continue
			}
			if _, exists := seen[capability]; exists {
				continue
			}
			seen[capability] = struct{}{}
			out = append(out, capability)
		}
	}
	sort.Strings(out)
	return out
}

func validateGeneratorFilters(kindFilter, profileFilter, capabilityFilter string) error {
	unknownKinds := unknownFilterValues(kindFilter, stringSliceToSet(supportedGeneratorKinds()))
	if len(unknownKinds) > 0 {
		return fmt.Errorf("unknown kind filter value(s): %s (supported: %s)", strings.Join(unknownKinds, ", "), strings.Join(supportedGeneratorKinds(), ", "))
	}

	unknownProfiles := unknownFilterValues(profileFilter, stringSliceToSet(supportedGeneratorProfiles()))
	if len(unknownProfiles) > 0 {
		return fmt.Errorf("unknown profile filter value(s): %s (supported: %s)", strings.Join(unknownProfiles, ", "), strings.Join(supportedGeneratorProfiles(), ", "))
	}

	unknownCapabilities := unknownFilterValues(capabilityFilter, stringSliceToSet(supportedGeneratorCapabilities()))
	if len(unknownCapabilities) > 0 {
		return fmt.Errorf("unknown capability filter value(s): %s (supported: %s)", strings.Join(unknownCapabilities, ", "), strings.Join(supportedGeneratorCapabilities(), ", "))
	}

	return nil
}

func unknownFilterValues(raw string, supported map[string]struct{}) []string {
	unknown := make([]string, 0)
	for value := range parseFilterSet(raw) {
		if _, ok := supported[value]; !ok {
			unknown = append(unknown, value)
		}
	}
	sort.Strings(unknown)
	return unknown
}

func stringSliceToSet(values []string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		key := strings.ToLower(strings.TrimSpace(value))
		if key == "" {
			continue
		}
		out[key] = struct{}{}
	}
	return out
}

func runDetect(args []string) error {
	fs := flag.NewFlagSet("detect", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	repo := fs.String("repo", ".", "Path to local repository")
	ref := fs.String("ref", "HEAD", "Git ref label to include in output")
	pretty := fs.Bool("pretty", true, "Pretty-print JSON output")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	result, err := detect.ScanRepo(*repo, *ref)
	if err != nil {
		return err
	}
	return writeJSON(os.Stdout, result, *pretty)
}

// runLegacyImport retains the original prototype import command.
func runLegacyImport(args []string) error {
	fs := flag.NewFlagSet("import", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	repo := fs.String("repo", ".", "Path to local repository")
	ref := fs.String("ref", "HEAD", "Git ref label to include in output")
	space := fs.String("space", "default", "Target ConfigHub space")
	out := fs.String("out", "-", "Output file path, or '-' for stdout")
	pretty := fs.Bool("pretty", true, "Pretty-print JSON output")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	result, err := importer.ImportRepo(*repo, *ref, *space)
	if err != nil {
		return err
	}

	if *out == "-" {
		return writeJSON(os.Stdout, result, *pretty)
	}

	f, err := os.Create(*out)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer func() {
		_ = f.Close()
	}()

	return writeJSON(f, result, *pretty)
}

func runPublish(args []string) error {
	fs := flag.NewFlagSet("publish", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	in := fs.String("in", "-", "ImportFlow JSON input path, or '-' for stdin")
	out := fs.String("out", "-", "Bundle JSON output path, or '-' for stdout")
	space := fs.String("space", "default", "ConfigHub space label (direct mode)")
	ref := fs.String("ref", "HEAD", "Git ref label to include in output (direct mode)")
	whereResource := fs.String("where-resource", "", "Additional resource filter expression (direct mode)")
	pretty := fs.Bool("pretty", true, "Pretty-print JSON output")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	var imported gitopsflow.ImportFlowResult
	switch fs.NArg() {
	case 0:
		var inputBytes []byte
		var err error
		if *in == "-" {
			inputBytes, err = io.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("read stdin: %w", err)
			}
		} else {
			inputBytes, err = os.ReadFile(*in)
			if err != nil {
				return fmt.Errorf("read input file: %w", err)
			}
		}
		if err := json.Unmarshal(inputBytes, &imported); err != nil {
			return fmt.Errorf("parse import flow json: %w", err)
		}
	case 2:
		if *in != "-" {
			return errors.New("cannot combine --in with direct target mode")
		}
		targetSlug := fs.Arg(0)
		renderTargetSlug := fs.Arg(1)
		var err error
		imported, err = gitopsflow.Import(targetSlug, renderTargetSlug, *ref, *space, *whereResource)
		if err != nil {
			return err
		}
	default:
		return errors.New("usage: cub-gen publish [flags] [<target-slug> <render-target-slug>]")
	}

	bundle := publish.BuildBundle(imported)
	if *out == "-" {
		return writeJSON(os.Stdout, bundle, *pretty)
	}

	f, err := os.Create(*out)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer func() {
		_ = f.Close()
	}()
	return writeJSON(f, bundle, *pretty)
}

func runVerify(args []string) error {
	fs := flag.NewFlagSet("verify", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	in := fs.String("in", "-", "Bundle JSON input path, or '-' for stdin")
	jsonOut := fs.Bool("json", false, "Output JSON")
	pretty := fs.Bool("pretty", true, "Pretty-print JSON output")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("usage: cub-gen verify [flags]")
	}

	var inputBytes []byte
	var err error
	if *in == "-" {
		inputBytes, err = io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("read stdin: %w", err)
		}
	} else {
		inputBytes, err = os.ReadFile(*in)
		if err != nil {
			return fmt.Errorf("read input file: %w", err)
		}
	}

	var bundle publish.ChangeBundle
	if err := json.Unmarshal(inputBytes, &bundle); err != nil {
		return fmt.Errorf("parse bundle json: %w", err)
	}
	if err := publish.VerifyBundle(bundle); err != nil {
		return err
	}

	if *jsonOut {
		return writeJSON(os.Stdout, map[string]any{
			"valid":            true,
			"digest_algorithm": bundle.DigestAlgorithm,
			"bundle_digest":    bundle.BundleDigest,
			"change_id":        bundle.ChangeID,
		}, *pretty)
	}

	fmt.Printf("Bundle verification OK: %s\n", bundle.BundleDigest)
	return nil
}

func runAttest(args []string) error {
	fs := flag.NewFlagSet("attest", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	in := fs.String("in", "-", "Bundle JSON input path, or '-' for stdin")
	out := fs.String("out", "-", "Attestation JSON output path, or '-' for stdout")
	verifier := fs.String("verifier", "cub-gen", "Verifier identity label")
	pretty := fs.Bool("pretty", true, "Pretty-print JSON output")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("usage: cub-gen attest [flags]")
	}

	var inputBytes []byte
	var err error
	if *in == "-" {
		inputBytes, err = io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("read stdin: %w", err)
		}
	} else {
		inputBytes, err = os.ReadFile(*in)
		if err != nil {
			return fmt.Errorf("read input file: %w", err)
		}
	}

	var bundle publish.ChangeBundle
	if err := json.Unmarshal(inputBytes, &bundle); err != nil {
		return fmt.Errorf("parse bundle json: %w", err)
	}
	rec, err := attest.Build(bundle, *verifier)
	if err != nil {
		return err
	}

	if *out == "-" {
		return writeJSON(os.Stdout, rec, *pretty)
	}
	f, err := os.Create(*out)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer func() {
		_ = f.Close()
	}()
	return writeJSON(f, rec, *pretty)
}

func runVerifyAttestation(args []string) error {
	fs := flag.NewFlagSet("verify-attestation", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	in := fs.String("in", "-", "Attestation JSON input path, or '-' for stdin")
	bundlePath := fs.String("bundle", "", "Optional bundle JSON input path to verify digest linkage")
	jsonOut := fs.Bool("json", false, "Output JSON")
	pretty := fs.Bool("pretty", true, "Pretty-print JSON output")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("usage: cub-gen verify-attestation [flags]")
	}

	var recBytes []byte
	var err error
	if *in == "-" {
		recBytes, err = io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("read stdin: %w", err)
		}
	} else {
		recBytes, err = os.ReadFile(*in)
		if err != nil {
			return fmt.Errorf("read input file: %w", err)
		}
	}

	var rec attest.Record
	if err := json.Unmarshal(recBytes, &rec); err != nil {
		return fmt.Errorf("parse attestation json: %w", err)
	}

	linked := false
	if strings.TrimSpace(*bundlePath) == "" {
		if err := attest.VerifyRecord(rec); err != nil {
			return err
		}
	} else {
		bundleBytes, err := os.ReadFile(*bundlePath)
		if err != nil {
			return fmt.Errorf("read bundle file: %w", err)
		}
		var bundle publish.ChangeBundle
		if err := json.Unmarshal(bundleBytes, &bundle); err != nil {
			return fmt.Errorf("parse bundle json: %w", err)
		}
		if err := attest.VerifyRecordAgainstBundle(rec, bundle); err != nil {
			return err
		}
		linked = true
	}

	if *jsonOut {
		return writeJSON(os.Stdout, map[string]any{
			"valid":               true,
			"linked_bundle_check": linked,
			"attestation_digest":  rec.AttestationDigest,
			"bundle_digest":       rec.BundleDigest,
			"change_id":           rec.ChangeID,
		}, *pretty)
	}

	if linked {
		fmt.Printf("Attestation verification OK (linked): %s\n", rec.AttestationDigest)
		return nil
	}
	fmt.Printf("Attestation verification OK: %s\n", rec.AttestationDigest)
	return nil
}

type changePreviewInput struct {
	TargetSlug       string `json:"target_slug"`
	RenderTargetSlug string `json:"render_target_slug"`
	Space            string `json:"space"`
	Ref              string `json:"ref"`
	WhereResource    string `json:"where_resource,omitempty"`
}

type changePreviewSummary struct {
	ChangeID          string `json:"change_id"`
	BundleDigest      string `json:"bundle_digest"`
	AttestationDigest string `json:"attestation_digest"`
}

type changePreviewCounts struct {
	DiscoveredResources int `json:"discovered_resources"`
	DryInputs           int `json:"dry_inputs"`
	WetTargets          int `json:"wet_targets"`
	InversePatches      int `json:"inverse_patches"`
}

type changePreviewVerification struct {
	BundleValid      bool   `json:"bundle_valid"`
	AttestationValid bool   `json:"attestation_valid"`
	Verifier         string `json:"verifier"`
}

type changePreviewResult struct {
	Input              changePreviewInput        `json:"input"`
	Change             changePreviewSummary      `json:"change"`
	DiscoveredProfiles []string                  `json:"discovered_profiles"`
	Counts             changePreviewCounts       `json:"counts"`
	EditRecommendation model.InverseEditPointer  `json:"edit_recommendation"`
	Verification       changePreviewVerification `json:"verification"`
}

type changeRunDecision struct {
	State     string `json:"state"`
	Authority string `json:"authority"`
	Source    string `json:"source"`
}

type changeRunResult struct {
	Mode           string              `json:"mode"`
	Preview        changePreviewResult `json:"preview"`
	Decision       changeRunDecision   `json:"decision"`
	PromotionReady bool                `json:"promotion_ready"`
}

type changeExplainQuery struct {
	WetPathFilter string `json:"wet_path_filter,omitempty"`
	DryPathFilter string `json:"dry_path_filter,omitempty"`
	OwnerFilter   string `json:"owner_filter,omitempty"`
	MatchCount    int    `json:"match_count"`
}

type changeExplainSuggestion struct {
	Owner            string  `json:"owner"`
	WetPath          string  `json:"wet_path"`
	DryPath          string  `json:"dry_path"`
	EditHint         string  `json:"edit_hint"`
	Confidence       float64 `json:"confidence"`
	SourcePath       string  `json:"source_path,omitempty"`
	SourceTransform  string  `json:"source_transform,omitempty"`
	GeneratorName    string  `json:"generator_name,omitempty"`
	GeneratorProfile string  `json:"generator_profile,omitempty"`
}

type changeExplainResult struct {
	Input       changePreviewInput      `json:"input"`
	Change      changePreviewSummary    `json:"change"`
	Query       changeExplainQuery      `json:"query"`
	Explanation changeExplainSuggestion `json:"explanation"`
}

func runChange(args []string) error {
	if len(args) == 0 {
		printChangeUsage(os.Stderr)
		return errors.New("change subcommand required")
	}

	switch args[0] {
	case "help", "-h", "--help":
		printChangeUsage(os.Stdout)
		return nil
	case "preview":
		return runChangePreview(args[1:])
	case "run":
		return runChangeRun(args[1:])
	case "explain":
		return runChangeExplain(args[1:])
	default:
		printChangeUsage(os.Stderr)
		return fmt.Errorf("unknown change subcommand: %s", args[0])
	}
}

func runChangePreview(args []string) error {
	fs := flag.NewFlagSet("change preview", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	space := fs.String("space", "default", "ConfigHub space label")
	ref := fs.String("ref", "HEAD", "Git ref label to include in output")
	whereResource := fs.String("where-resource", "", "Additional resource filter expression")
	out := fs.String("out", "-", "Output file path, or '-' for stdout")
	verifier := fs.String("verifier", "cub-gen", "Verifier identity label")
	jsonOut := fs.Bool("json", true, "Output JSON")
	pretty := fs.Bool("pretty", true, "Pretty-print JSON output")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	_ = jsonOut

	if fs.NArg() != 2 {
		return errors.New("usage: cub-gen change preview [flags] <target-slug> <render-target-slug>")
	}
	targetSlug := fs.Arg(0)
	renderTargetSlug := fs.Arg(1)

	result, _, _, err := buildChangePreviewResult(
		targetSlug,
		renderTargetSlug,
		*space,
		*ref,
		*whereResource,
		*verifier,
	)
	if err != nil {
		return err
	}

	if *out == "-" {
		return writeJSON(os.Stdout, result, *pretty)
	}

	f, err := os.Create(*out)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer func() {
		_ = f.Close()
	}()
	return writeJSON(f, result, *pretty)
}

func runChangeRun(args []string) error {
	fs := flag.NewFlagSet("change run", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	space := fs.String("space", "default", "ConfigHub space label")
	ref := fs.String("ref", "HEAD", "Git ref label to include in output")
	whereResource := fs.String("where-resource", "", "Additional resource filter expression")
	mode := fs.String("mode", "local", "Execution mode: local or connected")
	baseURL := fs.String("base-url", "", "ConfigHub base URL (connected mode)")
	token := fs.String("token", "", "ConfigHub token (connected mode)")
	ingestEndpoint := fs.String("ingest-endpoint", "", "Override bridge ingest endpoint path (connected mode)")
	decisionEndpoint := fs.String("decision-endpoint", "", "Override bridge decision query endpoint path (connected mode)")
	out := fs.String("out", "-", "Output file path, or '-' for stdout")
	verifier := fs.String("verifier", "cub-gen", "Verifier identity label")
	jsonOut := fs.Bool("json", true, "Output JSON")
	pretty := fs.Bool("pretty", true, "Pretty-print JSON output")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	_ = jsonOut

	if fs.NArg() != 2 {
		return errors.New("usage: cub-gen change run [flags] <target-slug> <render-target-slug>")
	}
	targetSlug := fs.Arg(0)
	renderTargetSlug := fs.Arg(1)
	runMode := strings.ToLower(strings.TrimSpace(*mode))
	if runMode != "local" && runMode != "connected" {
		return errors.New("change run --mode must be local|connected")
	}

	preview, bundle, _, err := buildChangePreviewResult(
		targetSlug,
		renderTargetSlug,
		*space,
		*ref,
		*whereResource,
		*verifier,
	)
	if err != nil {
		return err
	}

	decision := changeRunDecision{
		State:     "ALLOW",
		Authority: *verifier,
		Source:    "local-preview",
	}
	promotionReady := true

	if runMode == "connected" {
		resolvedBaseURL := strings.TrimSpace(*baseURL)
		if resolvedBaseURL == "" {
			resolvedBaseURL = strings.TrimSpace(os.Getenv("CONFIGHUB_BASE_URL"))
		}
		if resolvedBaseURL == "" {
			return errors.New("change run --mode connected requires --base-url or CONFIGHUB_BASE_URL")
		}

		resolvedToken := strings.TrimSpace(*token)
		if resolvedToken == "" {
			resolvedToken = strings.TrimSpace(os.Getenv("CONFIGHUB_TOKEN"))
		}

		ingestRes, err := bridgeflow.IngestBundle(context.Background(), bridgeflow.Client{
			BaseURL:      resolvedBaseURL,
			BearerToken:  resolvedToken,
			EndpointPath: strings.TrimSpace(*ingestEndpoint),
		}, bundle)
		if err != nil {
			return fmt.Errorf("connected ingest: %w", err)
		}

		decisionRec, err := bridgeflow.QueryDecisionByChangeID(context.Background(), bridgeflow.DecisionClient{
			BaseURL:      resolvedBaseURL,
			BearerToken:  resolvedToken,
			EndpointPath: strings.TrimSpace(*decisionEndpoint),
		}, preview.Change.ChangeID)
		if err != nil {
			return fmt.Errorf("connected decision query: %w", err)
		}

		authority := strings.TrimSpace(decisionRec.ApprovedBy)
		if authority == "" {
			authority = strings.TrimSpace(decisionRec.PolicyDecisionRef)
		}
		if authority == "" {
			authority = "confighub-policy"
		}

		decision = changeRunDecision{
			State:     string(decisionRec.State),
			Authority: authority,
			Source:    "confighub-backend",
		}
		if decision.State != "ALLOW" {
			promotionReady = false
		}
		if ingestRes.ChangeID == "" {
			promotionReady = false
		}
	}

	result := changeRunResult{
		Mode:           runMode,
		Preview:        preview,
		Decision:       decision,
		PromotionReady: promotionReady,
	}

	if *out == "-" {
		return writeJSON(os.Stdout, result, *pretty)
	}

	f, err := os.Create(*out)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer func() {
		_ = f.Close()
	}()
	return writeJSON(f, result, *pretty)
}

func runChangeExplain(args []string) error {
	fs := flag.NewFlagSet("change explain", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	space := fs.String("space", "default", "ConfigHub space label")
	ref := fs.String("ref", "HEAD", "Git ref label to include in output")
	whereResource := fs.String("where-resource", "", "Additional resource filter expression")
	wetPath := fs.String("wet-path", "", "Filter explanations to a specific WET path")
	dryPath := fs.String("dry-path", "", "Filter explanations to a specific DRY path")
	owner := fs.String("owner", "", "Filter explanations to a specific owner")
	out := fs.String("out", "-", "Output file path, or '-' for stdout")
	jsonOut := fs.Bool("json", true, "Output JSON")
	pretty := fs.Bool("pretty", true, "Pretty-print JSON output")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	_ = jsonOut

	if fs.NArg() != 2 {
		return errors.New("usage: cub-gen change explain [flags] <target-slug> <render-target-slug>")
	}
	targetSlug := fs.Arg(0)
	renderTargetSlug := fs.Arg(1)

	preview, _, imported, err := buildChangePreviewResult(
		targetSlug,
		renderTargetSlug,
		*space,
		*ref,
		*whereResource,
		"cub-gen",
	)
	if err != nil {
		return err
	}

	wetFilter := strings.TrimSpace(*wetPath)
	dryFilter := strings.TrimSpace(*dryPath)
	ownerFilter := strings.TrimSpace(*owner)

	suggestion, matchCount, ok := pickInverseSuggestion(imported.Provenance, wetFilter, dryFilter, ownerFilter)
	if !ok {
		return fmt.Errorf("no inverse edit explanation matched filters (wet_path=%q dry_path=%q owner=%q)", wetFilter, dryFilter, ownerFilter)
	}

	result := changeExplainResult{
		Input:  preview.Input,
		Change: preview.Change,
		Query: changeExplainQuery{
			WetPathFilter: wetFilter,
			DryPathFilter: dryFilter,
			OwnerFilter:   ownerFilter,
			MatchCount:    matchCount,
		},
		Explanation: suggestion,
	}

	if *out == "-" {
		return writeJSON(os.Stdout, result, *pretty)
	}

	f, err := os.Create(*out)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer func() {
		_ = f.Close()
	}()
	return writeJSON(f, result, *pretty)
}

func buildChangePreviewResult(
	targetSlug, renderTargetSlug, space, ref, whereResource, verifier string,
) (changePreviewResult, publish.ChangeBundle, gitopsflow.ImportFlowResult, error) {
	imported, err := gitopsflow.Import(targetSlug, renderTargetSlug, ref, space, whereResource)
	if err != nil {
		return changePreviewResult{}, publish.ChangeBundle{}, gitopsflow.ImportFlowResult{}, err
	}

	bundle := publish.BuildBundle(imported)
	if err := publish.VerifyBundle(bundle); err != nil {
		return changePreviewResult{}, publish.ChangeBundle{}, gitopsflow.ImportFlowResult{}, fmt.Errorf("verify generated bundle: %w", err)
	}

	attestationRecord, err := attest.Build(bundle, verifier)
	if err != nil {
		return changePreviewResult{}, publish.ChangeBundle{}, gitopsflow.ImportFlowResult{}, fmt.Errorf("build attestation: %w", err)
	}
	if err := attest.VerifyRecordAgainstBundle(attestationRecord, bundle); err != nil {
		return changePreviewResult{}, publish.ChangeBundle{}, gitopsflow.ImportFlowResult{}, fmt.Errorf("verify generated attestation: %w", err)
	}

	topEdit, ok := bestInverseEditPointer(imported.Provenance)
	if !ok {
		topEdit = model.InverseEditPointer{
			Owner:    "unknown",
			EditHint: "No inverse edit hint produced.",
		}
	}

	result := changePreviewResult{
		Input: changePreviewInput{
			TargetSlug:       targetSlug,
			RenderTargetSlug: renderTargetSlug,
			Space:            space,
			Ref:              ref,
			WhereResource:    strings.TrimSpace(whereResource),
		},
		Change: changePreviewSummary{
			ChangeID:          bundle.ChangeID,
			BundleDigest:      bundle.BundleDigest,
			AttestationDigest: attestationRecord.AttestationDigest,
		},
		DiscoveredProfiles: discoveredProfiles(imported.Discovered),
		Counts: changePreviewCounts{
			DiscoveredResources: len(imported.Discovered),
			DryInputs:           len(imported.DryInputs),
			WetTargets:          len(imported.WetManifestTargets),
			InversePatches:      countInversePatches(imported.InversePlans),
		},
		EditRecommendation: topEdit,
		Verification: changePreviewVerification{
			BundleValid:      true,
			AttestationValid: true,
			Verifier:         verifier,
		},
	}

	return result, bundle, imported, nil
}

func bestInverseEditPointer(provenance []model.ProvenanceRecord) (model.InverseEditPointer, bool) {
	best := model.InverseEditPointer{}
	found := false
	for _, record := range provenance {
		for _, pointer := range record.InverseEditPointers {
			if !found || pointer.Confidence > best.Confidence {
				best = pointer
				found = true
			}
		}
	}
	return best, found
}

func pickInverseSuggestion(
	provenance []model.ProvenanceRecord,
	wetFilter, dryFilter, ownerFilter string,
) (changeExplainSuggestion, int, bool) {
	matchCount := 0
	best := changeExplainSuggestion{}
	bestConfidence := -1.0

	for _, record := range provenance {
		for _, pointer := range record.InverseEditPointers {
			if wetFilter != "" && pointer.WetPath != wetFilter {
				continue
			}
			if dryFilter != "" && pointer.DryPath != dryFilter {
				continue
			}
			if ownerFilter != "" && pointer.Owner != ownerFilter {
				continue
			}
			matchCount++

			sourcePath := ""
			sourceTransform := ""
			if source, ok := bestFieldOrigin(record.FieldOriginMap, pointer.WetPath, pointer.DryPath); ok {
				sourcePath = source.SourcePath
				sourceTransform = source.Transform
			}

			candidate := changeExplainSuggestion{
				Owner:            pointer.Owner,
				WetPath:          pointer.WetPath,
				DryPath:          pointer.DryPath,
				EditHint:         pointer.EditHint,
				Confidence:       pointer.Confidence,
				SourcePath:       sourcePath,
				SourceTransform:  sourceTransform,
				GeneratorName:    record.GeneratorName,
				GeneratorProfile: record.GeneratorProfile,
			}
			if candidate.Confidence > bestConfidence {
				best = candidate
				bestConfidence = candidate.Confidence
			}
		}
	}

	if bestConfidence < 0 {
		return changeExplainSuggestion{}, 0, false
	}
	return best, matchCount, true
}

func bestFieldOrigin(origins []model.FieldOrigin, wetPath, dryPath string) (model.FieldOrigin, bool) {
	best := model.FieldOrigin{}
	bestConfidence := -1.0

	for _, origin := range origins {
		if wetPath != "" && origin.WetPath != wetPath {
			continue
		}
		if dryPath != "" && origin.DryPath != dryPath {
			continue
		}
		if origin.Confidence > bestConfidence {
			best = origin
			bestConfidence = origin.Confidence
		}
	}
	if bestConfidence >= 0 {
		return best, true
	}

	for _, origin := range origins {
		if dryPath != "" && origin.DryPath != dryPath {
			continue
		}
		if origin.Confidence > bestConfidence {
			best = origin
			bestConfidence = origin.Confidence
		}
	}
	if bestConfidence >= 0 {
		return best, true
	}

	for _, origin := range origins {
		if wetPath != "" && origin.WetPath != wetPath {
			continue
		}
		if origin.Confidence > bestConfidence {
			best = origin
			bestConfidence = origin.Confidence
		}
	}
	if bestConfidence < 0 {
		return model.FieldOrigin{}, false
	}
	return best, true
}

func discoveredProfiles(discovered []gitopsflow.DiscoveredResource) []string {
	set := map[string]struct{}{}
	for _, resource := range discovered {
		profile := strings.TrimSpace(resource.GeneratorProfile)
		if profile == "" {
			continue
		}
		set[profile] = struct{}{}
	}
	profiles := make([]string, 0, len(set))
	for profile := range set {
		profiles = append(profiles, profile)
	}
	sort.Strings(profiles)
	return profiles
}

func countInversePatches(plans []model.InverseTransformPlan) int {
	total := 0
	for _, plan := range plans {
		total += len(plan.Patches)
	}
	return total
}

func runGitOps(args []string) error {
	if len(args) == 0 {
		printGitOpsUsage(os.Stderr)
		return errors.New("gitops subcommand required")
	}

	switch args[0] {
	case "help", "-h", "--help":
		printGitOpsUsage(os.Stdout)
		return nil
	case "discover":
		return runGitOpsDiscover(args[1:])
	case "import":
		return runGitOpsImport(args[1:])
	case "cleanup":
		return runGitOpsCleanup(args[1:])
	default:
		printGitOpsUsage(os.Stderr)
		return fmt.Errorf("unknown gitops subcommand: %s", args[0])
	}
}

func runGitOpsDiscover(args []string) error {
	fs := flag.NewFlagSet("gitops discover", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	space := fs.String("space", "default", "ConfigHub space label")
	ref := fs.String("ref", "HEAD", "Git ref label to include in output")
	whereResource := fs.String("where-resource", "", "Additional resource filter expression")
	jsonOut := fs.Bool("json", false, "Output JSON")
	pretty := fs.Bool("pretty", true, "Pretty-print JSON output")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	if fs.NArg() != 1 {
		return errors.New("usage: cub-gen gitops discover [flags] <target-slug>")
	}
	targetSlug := fs.Arg(0)

	result, err := gitopsflow.Discover(targetSlug, *ref, *space, *whereResource)
	if err != nil {
		return err
	}

	if *jsonOut {
		return writeJSON(os.Stdout, result, *pretty)
	}

	if len(result.Resources) == 0 {
		fmt.Println("No GitOps resources were discovered for the specified target")
		return nil
	}

	printDiscoverTable(result)
	return nil
}

func runGitOpsImport(args []string) error {
	fs := flag.NewFlagSet("gitops import", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	space := fs.String("space", "default", "ConfigHub space label")
	ref := fs.String("ref", "HEAD", "Git ref label to include in output")
	whereResource := fs.String("where-resource", "", "Additional resource filter expression")
	wait := fs.Bool("wait", false, "Accepted for parity with cub gitops import")
	jsonOut := fs.Bool("json", false, "Output JSON")
	pretty := fs.Bool("pretty", true, "Pretty-print JSON output")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	_ = wait

	if fs.NArg() != 2 {
		return errors.New("usage: cub-gen gitops import [flags] <target-slug> <render-target-slug>")
	}
	targetSlug := fs.Arg(0)
	renderTargetSlug := fs.Arg(1)

	result, err := gitopsflow.Import(targetSlug, renderTargetSlug, *ref, *space, *whereResource)
	if err != nil {
		return err
	}

	if *jsonOut {
		return writeJSON(os.Stdout, result, *pretty)
	}

	if len(result.Discovered) == 0 {
		fmt.Println("No GitOps resources were discovered for the specified target")
		return nil
	}

	printImportTable(result)
	return nil
}

func runGitOpsCleanup(args []string) error {
	fs := flag.NewFlagSet("gitops cleanup", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	space := fs.String("space", "default", "ConfigHub space label")
	jsonOut := fs.Bool("json", false, "Output JSON")
	pretty := fs.Bool("pretty", true, "Pretty-print JSON output")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	if fs.NArg() != 1 {
		return errors.New("usage: cub-gen gitops cleanup [flags] <target-slug>")
	}
	targetSlug := fs.Arg(0)

	deleted, filePath, err := gitopsflow.Cleanup(targetSlug, *space)
	if err != nil {
		return err
	}

	result := map[string]any{
		"space":         *space,
		"target_slug":   targetSlug,
		"discover_file": filePath,
		"deleted":       deleted,
	}

	if *jsonOut {
		return writeJSON(os.Stdout, result, *pretty)
	}

	if deleted {
		fmt.Printf("Deleted discover unit state file: %s\n", filePath)
	} else {
		fmt.Printf("No discover unit state file found: %s\n", filePath)
	}
	return nil
}

func printDiscoverTable(result gitopsflow.DiscoverResult) {
	kindCapabilities := map[string]string{}
	for _, kind := range registry.Kinds() {
		spec, ok := registry.Spec(kind)
		if !ok {
			continue
		}
		kindCapabilities[string(kind)] = strings.Join(spec.Capabilities, ",")
	}

	confidenceByGeneratorID := map[string]float64{}
	for _, d := range result.Detections {
		confidenceByGeneratorID[d.ID] = d.Confidence
	}

	rows := make([][6]string, 0, len(result.Resources))
	for _, r := range result.Resources {
		confidence := "-"
		if v, ok := confidenceByGeneratorID[r.GeneratorID]; ok {
			confidence = fmt.Sprintf("%.2f", v)
		}
		capabilities := kindCapabilities[r.GeneratorKind]
		if strings.TrimSpace(capabilities) == "" {
			capabilities = "-"
		}
		rows = append(rows, [6]string{
			r.ResourceType,
			r.ResourceName,
			r.GeneratorKind,
			r.GeneratorProfile,
			capabilities,
			confidence,
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i][0] != rows[j][0] {
			return rows[i][0] < rows[j][0]
		}
		return rows[i][1] < rows[j][1]
	})

	fmt.Println("Resource Type\tResource Name\tGenerator Kind\tProfile\tCapabilities\tConfidence")
	for _, row := range rows {
		fmt.Printf("%s\t%s\t%s\t%s\t%s\t%s\n", row[0], row[1], row[2], row[3], row[4], row[5])
	}
}

func printImportTable(result gitopsflow.ImportFlowResult) {
	fmt.Printf("Discovered %d GitOps resources, creating renderer units...\n", len(result.Discovered))
	printImportDiscoveredTable(result)
	fmt.Printf("Created renderer units: %d\n", len(result.DryUnits))
	fmt.Println("Rendering discovered resources...")
	fmt.Printf("Created wet units: %d\n", len(result.WetUnits))
	fmt.Printf("Created links: %d\n", len(result.Links))
	fmt.Printf("Generated contracts: %d\n", len(result.Contracts))
	fmt.Printf("Generated provenance records: %d\n", len(result.Provenance))
	fmt.Printf("Generated inverse transform plans: %d\n", len(result.InversePlans))
	printImportTripleSummary(result)
	fmt.Println("GitOps import complete")
}

func printImportDiscoveredTable(result gitopsflow.ImportFlowResult) {
	kindCapabilities := map[string]string{}
	for _, kind := range registry.Kinds() {
		spec, ok := registry.Spec(kind)
		if !ok {
			continue
		}
		kindCapabilities[string(kind)] = strings.Join(spec.Capabilities, ",")
	}

	rows := make([][5]string, 0, len(result.Discovered))
	for _, r := range result.Discovered {
		capabilities := kindCapabilities[r.GeneratorKind]
		if strings.TrimSpace(capabilities) == "" {
			capabilities = "-"
		}
		rows = append(rows, [5]string{
			r.ResourceType,
			r.ResourceName,
			r.GeneratorKind,
			r.GeneratorProfile,
			capabilities,
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i][0] != rows[j][0] {
			return rows[i][0] < rows[j][0]
		}
		return rows[i][1] < rows[j][1]
	})

	fmt.Println("Resource Type\tResource Name\tGenerator Kind\tProfile\tCapabilities")
	for _, row := range rows {
		fmt.Printf("%s\t%s\t%s\t%s\t%s\n", row[0], row[1], row[2], row[3], row[4])
	}
}

func printImportTripleSummary(result gitopsflow.ImportFlowResult) {
	if len(result.Contracts) == 0 {
		return
	}

	dryInputsByGeneratorID := map[string]int{}
	for _, dry := range result.DryInputs {
		dryInputsByGeneratorID[dry.GeneratorID]++
	}

	wetTargetsByGeneratorID := map[string]int{}
	for _, wet := range result.WetManifestTargets {
		wetTargetsByGeneratorID[wet.GeneratorID]++
	}

	inversePatchesBySourceRef := map[string]int{}
	reviewRequiredBySourceRef := map[string]int{}
	editableByTotals := map[string]int{}
	for _, plan := range result.InversePlans {
		for _, patch := range plan.Patches {
			inversePatchesBySourceRef[plan.SourceRef]++
			if patch.RequiresReview {
				reviewRequiredBySourceRef[plan.SourceRef]++
			}
			if owner := strings.TrimSpace(patch.EditableBy); owner != "" {
				editableByTotals[owner]++
			}
		}
	}

	totalReviewRequired := 0
	for _, v := range reviewRequiredBySourceRef {
		totalReviewRequired += v
	}

	contracts := append([]model.GeneratorContract(nil), result.Contracts...)
	sort.Slice(contracts, func(i, j int) bool {
		if contracts[i].Kind != contracts[j].Kind {
			return contracts[i].Kind < contracts[j].Kind
		}
		return contracts[i].GeneratorID < contracts[j].GeneratorID
	})

	fmt.Println("Generator Kind\tProfile\tCapabilities\tDry Inputs\tWet Targets\tInverse Patches\tReview Required")
	for _, c := range contracts {
		fmt.Printf("%s\t%s\t%s\t%d\t%d\t%d\t%d\n",
			c.Kind,
			c.Profile,
			strings.Join(c.Capabilities, ","),
			dryInputsByGeneratorID[c.GeneratorID],
			wetTargetsByGeneratorID[c.GeneratorID],
			inversePatchesBySourceRef[c.SourcePath],
			reviewRequiredBySourceRef[c.SourcePath],
		)
	}

	if len(editableByTotals) == 0 {
		printSwampWorkflowSummary(result.Provenance)
		return
	}
	owners := make([]string, 0, len(editableByTotals))
	for owner := range editableByTotals {
		owners = append(owners, owner)
	}
	sort.Strings(owners)

	fmt.Printf("Review required patches: %d\n", totalReviewRequired)
	fmt.Println("Patch owner\tCount")
	for _, owner := range owners {
		fmt.Printf("%s\t%d\n", owner, editableByTotals[owner])
	}

	printSwampWorkflowSummary(result.Provenance)
	printOpsWorkflowSummary(result.Provenance)
}

func printSwampWorkflowSummary(provenance []model.ProvenanceRecord) {
	type swampSummaryRow struct {
		generatorName              string
		profile                    string
		workflowCount              int
		stepCount                  int
		modelRefCount              int
		missingRequiredCount       int
		unapprovedModelCount       int
		unapprovedModelMethodCount int
	}

	rows := make([]swampSummaryRow, 0, len(provenance))
	for _, prov := range provenance {
		if prov.SwampWorkflow == nil {
			continue
		}
		analysis := prov.SwampWorkflow
		rows = append(rows, swampSummaryRow{
			generatorName:              prov.GeneratorName,
			profile:                    prov.GeneratorProfile,
			workflowCount:              len(analysis.WorkflowPaths),
			stepCount:                  len(analysis.StepNames),
			modelRefCount:              len(analysis.ModelRefs),
			missingRequiredCount:       len(analysis.MissingRequiredSteps),
			unapprovedModelCount:       len(analysis.UnapprovedModels),
			unapprovedModelMethodCount: len(analysis.UnapprovedModelMethods),
		})
	}
	if len(rows) == 0 {
		return
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].profile != rows[j].profile {
			return rows[i].profile < rows[j].profile
		}
		return rows[i].generatorName < rows[j].generatorName
	})

	fmt.Println("Swamp workflow analysis")
	fmt.Println("Generator\tProfile\tWorkflows\tSteps\tModel refs\tMissing required\tUnapproved models\tUnapproved model methods")
	for _, row := range rows {
		fmt.Printf(
			"%s\t%s\t%d\t%d\t%d\t%d\t%d\t%d\n",
			row.generatorName,
			row.profile,
			row.workflowCount,
			row.stepCount,
			row.modelRefCount,
			row.missingRequiredCount,
			row.unapprovedModelCount,
			row.unapprovedModelMethodCount,
		)
	}
}

func printOpsWorkflowSummary(provenance []model.ProvenanceRecord) {
	type opsSummaryRow struct {
		generatorName         string
		profile               string
		workflowCount         int
		actionCount           int
		scheduleOverrideCount int
		approvalGateCount     int
		blockedUsedCount      int
		unapprovedActionCount int
	}

	rows := make([]opsSummaryRow, 0, len(provenance))
	for _, prov := range provenance {
		if prov.OpsWorkflow == nil {
			continue
		}
		analysis := prov.OpsWorkflow
		rows = append(rows, opsSummaryRow{
			generatorName:         prov.GeneratorName,
			profile:               prov.GeneratorProfile,
			workflowCount:         len(analysis.WorkflowPaths),
			actionCount:           len(analysis.ActionNames),
			scheduleOverrideCount: len(analysis.ScheduleOverrides),
			approvalGateCount:     len(analysis.ApprovalGates),
			blockedUsedCount:      len(analysis.BlockedActionsUsed),
			unapprovedActionCount: len(analysis.UnapprovedActions),
		})
	}
	if len(rows) == 0 {
		return
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].profile != rows[j].profile {
			return rows[i].profile < rows[j].profile
		}
		return rows[i].generatorName < rows[j].generatorName
	})

	fmt.Println("Ops workflow analysis")
	fmt.Println("Generator\tProfile\tWorkflows\tActions\tSchedule overrides\tApproval gates\tBlocked used\tUnapproved actions")
	for _, row := range rows {
		fmt.Printf(
			"%s\t%s\t%d\t%d\t%d\t%d\t%d\t%d\n",
			row.generatorName,
			row.profile,
			row.workflowCount,
			row.actionCount,
			row.scheduleOverrideCount,
			row.approvalGateCount,
			row.blockedUsedCount,
			row.unapprovedActionCount,
		)
	}
}

func writeJSON(out io.Writer, v any, pretty bool) error {
	enc := json.NewEncoder(out)
	if pretty {
		enc.SetIndent("", "  ")
	}
	return enc.Encode(v)
}

func printUsage(out io.Writer) {
	fmt.Fprintln(out, "cub-gen: prototype generator importer for agentic GitOps")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Usage:")
	fmt.Fprintln(out, "  cub-gen detect [--repo PATH] [--ref REF] [--pretty]")
	fmt.Fprintln(out, "  cub-gen import [--repo PATH] [--ref REF] [--space SPACE] [--out FILE|-] [--pretty]")
	fmt.Fprintln(out, "  cub-gen publish [--in FILE|-] [--out FILE|-] [--pretty]")
	fmt.Fprintln(out, "  cub-gen publish [--space SPACE] [--ref REF] [--where-resource EXPR] [--out FILE|-] [--pretty] <target-slug> <render-target-slug>")
	fmt.Fprintln(out, "  cub-gen verify [--in FILE|-] [--json] [--pretty]")
	fmt.Fprintln(out, "  cub-gen attest [--in FILE|-] [--out FILE|-] [--verifier NAME] [--pretty]")
	fmt.Fprintln(out, "  cub-gen verify-attestation [--in FILE|-] [--bundle FILE] [--json] [--pretty]")
	fmt.Fprintln(out, "  cub-gen change preview [--space SPACE] [--ref REF] [--where-resource EXPR] [--out FILE|-] [--verifier NAME] [--json] [--pretty] <target-slug> <render-target-slug>")
	fmt.Fprintln(out, "  cub-gen change run [--space SPACE] [--ref REF] [--where-resource EXPR] [--mode local|connected] [--base-url URL] [--token TOKEN] [--ingest-endpoint PATH] [--decision-endpoint PATH] [--out FILE|-] [--verifier NAME] [--json] [--pretty] <target-slug> <render-target-slug>")
	fmt.Fprintln(out, "  cub-gen change explain [--space SPACE] [--ref REF] [--where-resource EXPR] [--wet-path PATH] [--dry-path PATH] [--owner OWNER] [--out FILE|-] [--json] [--pretty] <target-slug> <render-target-slug>")
	fmt.Fprintln(out, "  cub-gen generators [--kind KIND] [--profile PROFILE] [--capability CAPABILITY] [--strict-filters] [--json|--markdown] [--details] [--pretty]")
	fmt.Fprintln(out, "  cub-gen gitops <discover|import|cleanup> [flags]")
	fmt.Fprintln(out, "  cub-gen bridge <ingest|decision|promote> [flags]")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "GitOps parity examples:")
	fmt.Fprintln(out, "  cub-gen gitops discover --space my-space ./examples/helm-paas")
	fmt.Fprintln(out, "  cub-gen gitops import --space my-space ./examples/helm-paas local-renderer")
	fmt.Fprintln(out, "  cub-gen gitops cleanup --space my-space ./examples/helm-paas")
	fmt.Fprintln(out, "  cub-gen gitops import --space my-space --json ./examples/helm-paas local-renderer | cub-gen publish --in -")
	fmt.Fprintln(out, "  cub-gen publish --space my-space ./examples/helm-paas ./examples/helm-paas")
	fmt.Fprintln(out, "  cub-gen publish --space my-space ./examples/scoredev-paas ./examples/scoredev-paas")
	fmt.Fprintln(out, "  cub-gen publish --space my-space ./examples/springboot-paas ./examples/springboot-paas")
	fmt.Fprintln(out, "  cub-gen publish --space my-space ./examples/backstage-idp ./examples/backstage-idp")
	fmt.Fprintln(out, "  cub-gen publish --space my-space ./examples/just-apps-no-platform-config ./examples/just-apps-no-platform-config")
	fmt.Fprintln(out, "  cub-gen publish --space my-space ./examples/ops-workflow ./examples/ops-workflow")
	fmt.Fprintln(out, "  cub-gen publish --space my-space ./examples/helm-paas ./examples/helm-paas | cub-gen verify --in -")
	fmt.Fprintln(out, "  cub-gen publish --space my-space ./examples/scoredev-paas ./examples/scoredev-paas | cub-gen verify --in -")
	fmt.Fprintln(out, "  cub-gen publish --space my-space ./examples/springboot-paas ./examples/springboot-paas | cub-gen verify --in -")
	fmt.Fprintln(out, "  cub-gen publish --space my-space ./examples/backstage-idp ./examples/backstage-idp | cub-gen verify --in -")
	fmt.Fprintln(out, "  cub-gen publish --space my-space ./examples/just-apps-no-platform-config ./examples/just-apps-no-platform-config | cub-gen verify --in -")
	fmt.Fprintln(out, "  cub-gen publish --space my-space ./examples/ops-workflow ./examples/ops-workflow | cub-gen verify --in -")
	fmt.Fprintln(out, "  cub-gen publish --space my-space ./examples/helm-paas ./examples/helm-paas | cub-gen attest --in - --verifier ci-bot")
	fmt.Fprintln(out, "  cub-gen publish --space my-space ./examples/scoredev-paas ./examples/scoredev-paas | cub-gen attest --in - --verifier ci-bot")
	fmt.Fprintln(out, "  cub-gen publish --space my-space ./examples/springboot-paas ./examples/springboot-paas | cub-gen attest --in - --verifier ci-bot")
	fmt.Fprintln(out, "  cub-gen publish --space my-space ./examples/backstage-idp ./examples/backstage-idp | cub-gen attest --in - --verifier ci-bot")
	fmt.Fprintln(out, "  cub-gen publish --space my-space ./examples/just-apps-no-platform-config ./examples/just-apps-no-platform-config | cub-gen attest --in - --verifier ci-bot")
	fmt.Fprintln(out, "  cub-gen publish --space my-space ./examples/ops-workflow ./examples/ops-workflow | cub-gen attest --in - --verifier ci-bot")
	fmt.Fprintln(out, "  cub-gen verify-attestation --in attestation.json --bundle bundle.json")
	fmt.Fprintln(out, "  cub-gen change preview --space my-space ./examples/scoredev-paas ./examples/scoredev-paas")
	fmt.Fprintln(out, "  cub-gen change run --mode local --space my-space ./examples/scoredev-paas ./examples/scoredev-paas")
	fmt.Fprintln(out, "  cub-gen change explain --space my-space --wet-path \"Deployment/spec/template/spec/containers[name=main]/image\" ./examples/scoredev-paas ./examples/scoredev-paas")
	fmt.Fprintln(out, "  cub-gen bridge ingest --in bundle.json --base-url https://confighub.example")
	fmt.Fprintln(out, "  cub-gen bridge decision query --change-id chg_123 --base-url https://confighub.example")
	fmt.Fprintln(out, "  cub-gen bridge promote init --change-id chg_123 --app-pr-repo github.com/confighub/apps --app-pr-number 42 --app-pr-url https://github.com/confighub/apps/pull/42 --mr-id mr_123 --mr-url https://confighub.example/mr/123")
	fmt.Fprintln(out, "  cub-gen generators --json")
	fmt.Fprintln(out, "  cub-gen generators --json --details")
	fmt.Fprintln(out, "  cub-gen generators --markdown --details")
	fmt.Fprintln(out, "  cub-gen generators --capability render-manifests")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Note: gitops commands are local-only prototypes that mirror cub gitops stages.")
}

func printChangeUsage(out io.Writer) {
	fmt.Fprintln(out, "cub-gen change: developer-facing change workflow commands")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Usage:")
	fmt.Fprintln(out, "  cub-gen change preview [--space SPACE] [--ref REF] [--where-resource EXPR] [--out FILE|-] [--verifier NAME] [--json] [--pretty] <target-slug> <render-target-slug>")
	fmt.Fprintln(out, "  cub-gen change run [--space SPACE] [--ref REF] [--where-resource EXPR] [--mode local|connected] [--base-url URL] [--token TOKEN] [--ingest-endpoint PATH] [--decision-endpoint PATH] [--out FILE|-] [--verifier NAME] [--json] [--pretty] <target-slug> <render-target-slug>")
	fmt.Fprintln(out, "  cub-gen change explain [--space SPACE] [--ref REF] [--where-resource EXPR] [--wet-path PATH] [--dry-path PATH] [--owner OWNER] [--out FILE|-] [--json] [--pretty] <target-slug> <render-target-slug>")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Examples:")
	fmt.Fprintln(out, "  cub-gen change preview --space my-space ./examples/helm-paas ./examples/helm-paas")
	fmt.Fprintln(out, "  cub-gen change preview --space my-space ./examples/swamp-automation ./examples/swamp-automation")
	fmt.Fprintln(out, "  cub-gen change run --mode local --space my-space ./examples/scoredev-paas ./examples/scoredev-paas")
	fmt.Fprintln(out, "  cub-gen change explain --space my-space --owner app-team ./examples/scoredev-paas ./examples/scoredev-paas")
}

func printGitOpsUsage(out io.Writer) {
	resourceKinds := registry.SupportedResourceKinds()
	kindEq := renderKindEqualsClause(resourceKinds)
	kindIn := quoteKindsWithDelimiter(resourceKinds, ",")

	fmt.Fprintln(out, "cub-gen gitops: local parity commands for cub gitops pattern")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Usage:")
	fmt.Fprintln(out, "  cub-gen gitops discover [--space SPACE] [--ref REF] [--where-resource EXPR] [--json] <target-slug>")
	fmt.Fprintln(out, "  cub-gen gitops import [--space SPACE] [--ref REF] [--where-resource EXPR] [--wait] [--json] <target-slug> <render-target-slug>")
	fmt.Fprintln(out, "  cub-gen gitops cleanup [--space SPACE] [--json] <target-slug>")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Supported where-resource clauses:")
	fmt.Fprintf(out, "  kind = %s\n", kindEq)
	fmt.Fprintf(out, "  kind IN (%s)\n", kindIn)
	fmt.Fprintln(out, "  name = 'checkout-api' | resource_name LIKE '<contains-api>' | root LIKE '<contains-prod>'")
	fmt.Fprintln(out, "  combine clauses with AND")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Examples:")
	fmt.Fprintln(out, "  cub-gen gitops discover --space my-space ./examples/scoredev-paas")
	fmt.Fprintln(out, "  cub-gen gitops discover --where-resource \"kind IN ('HelmRelease') AND resource_name LIKE '<contains-payments>'\" ./examples/helm-paas")
	fmt.Fprintln(out, "  cub-gen gitops import --space my-space ./examples/springboot-paas render-target")
	fmt.Fprintln(out, "  cub-gen gitops import --space my-space ./examples/backstage-idp render-target")
	fmt.Fprintln(out, "  cub-gen gitops import --space my-space ./examples/just-apps-no-platform-config render-target")
	fmt.Fprintln(out, "  cub-gen gitops import --space my-space ./examples/ops-workflow render-target")
	fmt.Fprintln(out, "  cub-gen gitops cleanup --space my-space ./examples/springboot-paas")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Tip: <target-slug> is a local repo path in this prototype.")
}

func quoteKindsWithDelimiter(kinds []string, delimiter string) string {
	quoted := make([]string, 0, len(kinds))
	for _, kind := range kinds {
		quoted = append(quoted, fmt.Sprintf("'%s'", kind))
	}
	return strings.Join(quoted, delimiter)
}

func renderKindEqualsClause(kinds []string) string {
	parts := make([]string, 0, len(kinds))
	for _, kind := range kinds {
		parts = append(parts, fmt.Sprintf("kind = '%s'", kind))
	}
	return strings.Join(parts, " | ")
}
