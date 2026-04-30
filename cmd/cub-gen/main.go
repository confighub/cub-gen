package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/confighub/cub-gen/internal/attest"
	"github.com/confighub/cub-gen/internal/detect"
	gitopsflow "github.com/confighub/cub-gen/internal/gitops"
	"github.com/confighub/cub-gen/internal/importer"
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
	case "enrich":
		return runEnrich(args[1:])
	case "generators":
		return runGenerators(args[1:])
	case "gitops":
		return runGitOps(args[1:])
	case "platform":
		return runPlatform(args[1:])
	case "bridge":
		return runBridge(args[1:])
	case "score":
		return runScore(args[1:])
	case "springboot":
		return runSpringBoot(args[1:])
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
	PathPrefixes   []string `json:"path_prefixes,omitempty"`
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
			fmt.Fprintln(out, "| Role | Exact basenames | Path prefixes | Prefixes | Extensions |")
			fmt.Fprintln(out, "| --- | --- | --- | --- | --- |")
			for _, rule := range policies.InputRoleRules {
				fmt.Fprintf(out, "| `%s` | %s | %s | %s | %s |\n",
					markdownCell(rule.Role),
					markdownCell(strings.Join(rule.ExactBasenames, ", ")),
					markdownCell(strings.Join(rule.PathPrefixes, ", ")),
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
			PathPrefixes:   append([]string(nil), rule.PathPrefixes...),
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

// runLegacyImport retains the original source-repo import command.
func runLegacyImport(args []string) error {
	fs := flag.NewFlagSet("import", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	repo := fs.String("repo", ".", "Path to local repository")
	ref := fs.String("ref", "HEAD", "Git ref label to include in output")
	space := fs.String("space", "default", "Target ConfigHub space")
	out := fs.String("out", "-", "Output file path, or '-' for stdout")
	pretty := fs.Bool("pretty", true, "Pretty-print JSON output")
	overrideFlags := addHelmCLIOverrideFlags(fs)
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	helmCLIOverrides, err := overrideFlags.parse()
	if err != nil {
		return err
	}

	result, err := importer.ImportRepoWithOptions(*repo, *ref, *space, importer.ImportOptions{
		HelmCLIOverrides: helmCLIOverrides,
	})
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
	fs.Usage = func() {
		printPublishUsage(fs.Output())
	}
	in := fs.String("in", "-", "ImportFlow JSON input path, or '-' for stdin")
	out := fs.String("out", "-", "Bundle JSON output path, or '-' for stdout")
	space := fs.String("space", "default", "ConfigHub space label (direct mode)")
	ref := fs.String("ref", "HEAD", "Git ref label to include in output (direct mode)")
	whereResource := fs.String("where-resource", "", "Additional resource filter expression (direct mode)")
	pretty := fs.Bool("pretty", true, "Pretty-print JSON output")
	overrideFlags := addHelmCLIOverrideFlags(fs)
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	helmCLIOverrides, err := overrideFlags.parse()
	if err != nil {
		return err
	}

	var imported gitopsflow.ImportFlowResult
	switch fs.NArg() {
	case 0:
		if len(helmCLIOverrides) > 0 {
			return errors.New("cannot combine Helm CLI override flags with --in pipe mode")
		}
		var inputBytes []byte
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
	case 1, 2:
		if *in != "-" {
			return errors.New("cannot combine --in with direct target mode")
		}
		targetSlug, renderTargetSlug, err := resolveTargetPairArgs(fs, "usage: cub-gen publish [flags] [<target-path> [<render-target-path>]]")
		if err != nil {
			return err
		}
		imported, err = gitopsflow.ImportWithOptions(targetSlug, renderTargetSlug, *ref, *space, *whereResource, importer.ImportOptions{
			HelmCLIOverrides: helmCLIOverrides,
		})
		if err != nil {
			return err
		}
	default:
		return errors.New("usage: cub-gen publish [flags] [<target-path> [<render-target-path>]]")
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

func writeJSON(out io.Writer, v any, pretty bool) error {
	enc := json.NewEncoder(out)
	if pretty {
		enc.SetIndent("", "  ")
	}
	return enc.Encode(v)
}

type helpSection struct {
	Title string
	Lines []string
}

func printCommandHelp(out io.Writer, title string, description []string, sections ...helpSection) {
	fmt.Fprintln(out, title)
	if len(description) > 0 {
		fmt.Fprintln(out)
		for _, line := range description {
			fmt.Fprintln(out, line)
		}
	}
	for _, section := range sections {
		if len(section.Lines) == 0 {
			continue
		}
		fmt.Fprintln(out)
		if section.Title != "" {
			fmt.Fprintf(out, "%s:\n", section.Title)
		}
		for _, line := range section.Lines {
			fmt.Fprintln(out, line)
		}
	}
}

func printUsage(out io.Writer) {
	printCommandHelp(
		out,
		"cub-gen: trace repo config to rendered output so you know what to edit",
		[]string{
			"Use cub-gen when your question starts in the repo, not the cluster:",
			"  - Which file/path/owner produced this rendered field?",
			"  - What manifests did this repo actually produce?",
			"  - What evidence bundle should I verify or send to ConfigHub?",
		},
		helpSection{
			Title: "START HERE",
			Lines: []string{
				"  gitops import     See what a repo renders and where each field came from",
				"  change explain    Find the DRY file/path to edit for a rendered field",
				"  change impact     See which rendered fields a DRY path can affect",
				"  change preview    Preview a safe repo change",
				"  enrich preview    Propose sidecar proof metadata for PR review",
				"  platform import   Read a multi-repo platform estate as a graph",
				"  detect            Detect generators in a repo",
				"  generators        List supported generators (Helm, Score, Spring Boot, ...)",
			},
		},
		helpSection{
			Title: "BUILD EVIDENCE",
			Lines: []string{
				"  publish           Build a provenance bundle from a repo or import output",
				"  verify            Verify a provenance bundle",
				"  attest            Sign a provenance bundle",
			},
		},
		helpSection{
			Title: "ADVANCED CONNECTED",
			Lines: []string{
				"  change run        Ask ConfigHub for a decision (does not deploy)",
				"  bridge            Advanced ingest/decision/promote workflows",
			},
		},
		helpSection{
			Title: "FIRST RUNS",
			Lines: []string{
				"  cub-gen gitops import --space my-space ./examples/helm-paas",
				"  cub-gen platform import --json ./testdata/platform-estate/platform.yaml",
				"  cub-gen change explain --space my-space --owner app-team ./examples/scoredev-paas",
				"  cub-gen enrich preview --space my-space ./examples/helm-paas",
				"  cub-gen publish --space my-space ./examples/helm-paas | cub-gen verify --in -",
				"  cub-gen publish --space my-space ./examples/helm-paas | cub-gen attest --in - --verifier ci-bot",
			},
		},
		helpSection{
			Title: "BOUNDARIES",
			Lines: []string{
				"  - cub-gen does not deploy and does not read live cluster state",
				"  - Use 'cub-scout' for runtime/cluster questions",
				"  - Use 'cub gitops' for cluster-side import and ConfigHub management",
				"  - If you omit <render-target-path>, cub-gen reuses <target-path>",
			},
		},
	)
}

func printChangeUsage(out io.Writer) {
	printCommandHelp(
		out,
		"cub-gen change: preview, impact, and explain safe source edits",
		[]string{
			"Use these commands after you know the repo path and want to answer:",
			"  - what will change?",
			"  - what changed between two rendered refs?",
			"  - what changed between two release revisions, and which DRY edits caused it?",
			"  - which rendered fields does this DRY path affect?",
			"  - where should I edit DRY source?",
			"  - should I ask ConfigHub for a decision?",
		},
		helpSection{
			Title: "Usage",
			Lines: []string{
				"  cub-gen change preview [--space SPACE] [--ref REF] [--where-resource EXPR] [--set KEY=VALUE] [--set-string KEY=VALUE] [--set-file KEY=PATH] [--out FILE|-] [--verifier NAME] [--json] [--pretty] <target-path> [<render-target-path>]",
				"  cub-gen change run [--space SPACE] [--ref REF] [--where-resource EXPR] [--set KEY=VALUE] [--set-string KEY=VALUE] [--set-file KEY=PATH] [--mode local|connected] [--base-url URL] [--token TOKEN] [--ingest-endpoint PATH] [--decision-endpoint PATH] [--out FILE|-] [--verifier NAME] [--json] [--pretty] <target-path> [<render-target-path>]",
				"  cub-gen change diff --before-ref REF --after-ref REF [--space SPACE] [--where-resource EXPR] [--set KEY=VALUE] [--set-string KEY=VALUE] [--set-file KEY=PATH] [--dry-path PATH] [--wet-path PATH] [--owner OWNER] [--out FILE|-] [--pretty] <target-path> [<render-target-path>]",
				"  cub-gen change revision-diff --from REF --to REF [--space SPACE] [--where-resource EXPR] [--set KEY=VALUE] [--set-string KEY=VALUE] [--set-file KEY=PATH] [--dry-path PATH] [--wet-path PATH] [--owner OWNER] [--out FILE|-] [--pretty] <target-path> [<render-target-path>]",
				"  cub-gen change impact [--space SPACE] [--ref REF] [--where-resource EXPR] [--set KEY=VALUE] [--set-string KEY=VALUE] [--set-file KEY=PATH] [--dry-path PATH] [--wet-path PATH] [--owner OWNER] [--out FILE|-] [--json] [--pretty] <target-path> [<render-target-path>]",
				"  cub-gen change impact --change-id ID --bundle FILE [--dry-path PATH] [--wet-path PATH] [--owner OWNER] [--out FILE|-] [--json] [--pretty]",
				"  cub-gen change explain [--space SPACE] [--ref REF] [--where-resource EXPR] [--set KEY=VALUE] [--set-string KEY=VALUE] [--set-file KEY=PATH] [--wet-path PATH] [--dry-path PATH] [--owner OWNER] [--out FILE|-] [--json] [--pretty] <target-path> [<render-target-path>]",
				"  cub-gen change explain --change-id ID --bundle FILE [--wet-path PATH] [--dry-path PATH] [--owner OWNER] [--out FILE|-] [--json] [--pretty]",
				"  cub-gen change api serve [--listen ADDR] [--space SPACE] [--ref REF] [--verifier NAME]",
			},
		},
		helpSection{
			Title: "Examples",
			Lines: []string{
				"  cub-gen change preview --space my-space ./examples/helm-paas",
				"  cub-gen change diff --before-ref main --after-ref HEAD ./examples/helm-paas",
				"  cub-gen change revision-diff --from v0.2-preview.1 --to v0.2-preview.2 ./examples/helm-paas",
				"  cub-gen change impact --space my-space --dry-path values.image.tag ./examples/helm-paas",
				"  cub-gen change explain --space my-space --set image.tag=v1.2.4 ./examples/helm-paas",
				"  cub-gen change run --mode local --space my-space ./examples/scoredev-paas",
				"  cub-gen change explain --space my-space --wet-path \"Deployment/spec/template/spec/containers[0]/ports[0]/containerPort\" ./examples/springboot-paas",
				"  cub-gen change explain --change-id chg_123 --bundle bundle.json --wet-path \"Deployment/spec/template/spec/containers[0]/image\"",
				"  cub-gen change api serve --listen 127.0.0.1:8787 --space my-space",
			},
		},
		helpSection{
			Title: "Tips",
			Lines: []string{
				"  - Start with 'change explain' if you already know the rendered field you care about",
				"  - Use 'change diff' when you want a field-level before/after render comparison between two git refs",
				"  - Use 'change revision-diff' when you want release-note style DRY->WET pairs between two refs",
				"  - Use 'change impact' when you know the DRY path and want the downstream blast radius",
				"  - Start with 'change preview' before 'change run'",
				"  - Helm CLI overrides win over values-prod.yaml, values.yaml, and chart defaults",
				"  - 'change run --mode connected' asks ConfigHub for a decision; it does not deploy",
				"  - If omitted, <render-target-path> defaults to <target-path>",
			},
		},
	)
}

func printGitOpsUsage(out io.Writer) {
	resourceKinds := registry.SupportedResourceKinds()
	kindEq := renderKindEqualsClause(resourceKinds)
	kindIn := quoteKindsWithDelimiter(resourceKinds, ",")

	printCommandHelp(
		out,
		"cub-gen gitops: inspect a repo and map DRY source to WET output",
		[]string{
			"Start here when your first question is: what does this repo render, and where did it come from?",
			"These commands read local repo paths. They do not import from a cluster or deploy.",
		},
		helpSection{
			Title: "Usage",
			Lines: []string{
				"  cub-gen gitops discover [--space SPACE] [--ref REF] [--where-resource EXPR] [--adoption-report] [--json] <target-path>",
				"  cub-gen gitops import [--space SPACE] [--ref REF] [--where-resource EXPR] [--set KEY=VALUE] [--set-string KEY=VALUE] [--set-file KEY=PATH] [--wait] [--json] <target-path> [<render-target-path>]",
				"  cub-gen gitops cleanup [--space SPACE] [--json] <target-path>",
			},
		},
		helpSection{
			Title: "Supported where-resource clauses",
			Lines: []string{
				fmt.Sprintf("  %s", kindEq),
				fmt.Sprintf("  kind IN (%s)", kindIn),
				"  name = 'checkout-api' | resource_name LIKE '<contains-api>' | root LIKE '<contains-prod>'",
				"  combine clauses with AND",
			},
		},
		helpSection{
			Title: "Examples",
			Lines: []string{
				"  cub-gen gitops discover --space my-space ./examples/scoredev-paas",
				"  cub-gen gitops discover --where-resource \"kind IN ('HelmRelease') AND resource_name LIKE '<contains-payments>'\" ./examples/helm-paas",
				"  cub-gen gitops import --space my-space ./examples/springboot-paas",
				"  cub-gen gitops import --space my-space --set image.tag=v1.2.4 ./examples/helm-paas",
				"  cub-gen gitops cleanup --space my-space ./examples/springboot-paas",
			},
		},
		helpSection{
			Title: "Tips",
			Lines: []string{
				"  - Start with 'gitops import' if you want provenance immediately",
				"  - Use 'gitops discover' first when you want to filter or explore a repo",
				"  - Helm CLI overrides win over values-prod.yaml, values.yaml, and chart defaults",
				"  - If omitted, <render-target-path> defaults to <target-path>",
				"  - For cluster-side import, use 'cub gitops' instead of 'cub-gen gitops'",
			},
		},
	)
}

func printPublishUsage(out io.Writer) {
	printCommandHelp(
		out,
		"cub-gen publish: build verifiable evidence from import output or repo paths",
		[]string{
			"Use pipe mode when you already have gitops import JSON, or direct mode to import + bundle in one step.",
		},
		helpSection{
			Title: "Usage",
			Lines: []string{
				"  cub-gen publish [--in FILE|-] [--out FILE|-] [--pretty]",
				"  cub-gen publish [--space SPACE] [--ref REF] [--where-resource EXPR] [--set KEY=VALUE] [--set-string KEY=VALUE] [--set-file KEY=PATH] [--out FILE|-] [--pretty] <target-path> [<render-target-path>]",
			},
		},
		helpSection{
			Title: "Examples",
			Lines: []string{
				"  cub-gen gitops import --space my-space ./examples/helm-paas | cub-gen publish --in - --out -",
				"  cub-gen publish --space my-space --set image.tag=v1.2.4 ./examples/helm-paas > bundle.json",
				"  cub-gen publish --space my-space ./examples/helm-paas > bundle.json",
			},
		},
		helpSection{
			Title: "Tips",
			Lines: []string{
				"  - If omitted, <render-target-path> defaults to <target-path>",
				"  - Helm CLI overrides win over values-prod.yaml, values.yaml, and chart defaults",
				"  - Pipe to 'cub-gen verify' and 'cub-gen attest' for release evidence",
			},
		},
	)
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
