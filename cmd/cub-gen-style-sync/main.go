package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/confighub/cub-gen/internal/registry"
)

func main() {
	fs := flag.NewFlagSet("cub-gen-style-sync", flag.ExitOnError)
	outDir := fs.String("out", filepath.Join("docs", "triple-styles"), "Output directory")
	_ = fs.Parse(os.Args[1:])

	if err := syncStyles(*outDir); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	fmt.Printf("synced triple styles to %s\n", *outDir)
}

type styleModel struct {
	Kind         string
	Profile      string
	ResourceKind string
	ResourceType string
	Capabilities []string

	Contract contractModel

	Provenance provenanceModel

	Inverse inverseModel

	Hints map[string]string
}

type contractModel struct {
	InputRoleRules   []registry.InputRoleRule
	DefaultInputRole string
	RoleOwners       map[string]string
	DefaultOwner     string
	RoleSchemaRefs   map[string]string
	WetTargets       []registry.WetTargetTemplate
}

type provenanceModel struct {
	FieldOriginTransform        string
	FieldOriginOverlayTransform string
	FieldOriginConfidences      map[string]float64
	RenderedLineageTemplates    []registry.RenderedLineageTemplate
}

type inverseModel struct {
	InversePatchTemplates   map[string]registry.InversePatchTemplate
	InversePointerTemplates map[string]registry.InversePointerTemplate
	InversePatchReasons     map[string]string
	InverseEditHints        map[string]string
}

func syncStyles(root string) error {
	styleA := filepath.Join(root, "style-a-yaml")
	styleB := filepath.Join(root, "style-b-markdown")
	styleC := filepath.Join(root, "style-c-yaml-plus-docs")

	for _, path := range []string{styleA, styleB, styleC} {
		if err := os.RemoveAll(path); err != nil {
			return err
		}
		if err := os.MkdirAll(path, 0o755); err != nil {
			return err
		}
	}

	kinds := registry.Kinds()
	entries := make([]styleModel, 0, len(kinds))
	for _, kind := range kinds {
		spec, ok := registry.Spec(kind)
		if !ok {
			continue
		}
		entries = append(entries, styleModel{
			Kind:         string(spec.Kind),
			Profile:      spec.Profile,
			ResourceKind: spec.ResourceKind,
			ResourceType: spec.ResourceType,
			Capabilities: append([]string(nil), spec.Capabilities...),
			Contract: contractModel{
				InputRoleRules:   append([]registry.InputRoleRule(nil), spec.InputRoleRules...),
				DefaultInputRole: spec.DefaultInputRole,
				RoleOwners:       copyMap(spec.RoleOwners),
				DefaultOwner:     spec.DefaultOwner,
				RoleSchemaRefs:   copyMap(spec.RoleSchemaRefs),
				WetTargets:       append([]registry.WetTargetTemplate(nil), spec.WetTargets...),
			},
			Provenance: provenanceModel{
				FieldOriginTransform:        spec.FieldOriginTransform,
				FieldOriginOverlayTransform: spec.FieldOriginOverlayTransform,
				FieldOriginConfidences:      copyMapFloat(spec.FieldOriginConfidences),
				RenderedLineageTemplates:    append([]registry.RenderedLineageTemplate(nil), spec.RenderedLineageTemplates...),
			},
			Inverse: inverseModel{
				InversePatchTemplates:   copyMapPatch(spec.InversePatchTemplates),
				InversePointerTemplates: copyMapPointer(spec.InversePointerTemplates),
				InversePatchReasons:     copyMap(spec.InversePatchReasons),
				InverseEditHints:        copyMap(spec.InverseEditHints),
			},
			Hints: copyMap(spec.HintDefaults),
		})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Kind < entries[j].Kind })

	for _, entry := range entries {
		yamlOut := renderYAML(entry)
		markdownOut := renderMarkdown(entry)

		styleAPath := filepath.Join(styleA, entry.Kind+".yaml")
		if err := os.WriteFile(styleAPath, []byte(yamlOut), 0o644); err != nil {
			return err
		}

		styleBPath := filepath.Join(styleB, entry.Kind+".md")
		if err := os.WriteFile(styleBPath, []byte(markdownOut), 0o644); err != nil {
			return err
		}

		styleCKind := filepath.Join(styleC, entry.Kind)
		if err := os.MkdirAll(styleCKind, 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(styleCKind, "triple.yaml"), []byte(yamlOut), 0o644); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(styleCKind, "triple.md"), []byte(markdownOut), 0o644); err != nil {
			return err
		}
	}

	index := renderIndex(entries)
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte(index), 0o644); err != nil {
		return err
	}
	return nil
}

func renderIndex(entries []styleModel) string {
	var b strings.Builder
	b.WriteString("# Triple Style Comparison\n\n")
	b.WriteString("This folder provides three fully populated representations of the generator triple model for all supported generator kinds.\n\n")
	b.WriteString("1. Style A (`style-a-yaml`): YAML-first representation.\n")
	b.WriteString("2. Style B (`style-b-markdown`): human-readable markdown + tables.\n")
	b.WriteString("3. Style C (`style-c-yaml-plus-docs`): YAML + markdown pair per kind.\n\n")
	b.WriteString("Canonical runtime source remains Go registry specs in `internal/registry/registry.go`.\n\n")
	b.WriteString("| Kind | Style A | Style B | Style C |\n")
	b.WriteString("| --- | --- | --- | --- |\n")
	for _, entry := range entries {
		fmt.Fprintf(&b, "| `%s` | [yaml](style-a-yaml/%s.yaml) | [markdown](style-b-markdown/%s.md) | [pair](style-c-yaml-plus-docs/%s/) |\n",
			entry.Kind, entry.Kind, entry.Kind, entry.Kind)
	}
	b.WriteString("\n## Regenerate\n\n")
	b.WriteString("```bash\n")
	b.WriteString("go run ./cmd/cub-gen-style-sync\n")
	b.WriteString("```\n")
	return b.String()
}

func renderYAML(entry styleModel) string {
	var b strings.Builder
	w := yamlWriter{b: &b}

	w.line(0, "kind: %s", yamlQuote(entry.Kind))
	w.line(0, "profile: %s", yamlQuote(entry.Profile))
	w.line(0, "resource_kind: %s", yamlQuote(entry.ResourceKind))
	w.line(0, "resource_type: %s", yamlQuote(entry.ResourceType))

	w.line(0, "capabilities:")
	for _, c := range entry.Capabilities {
		w.line(1, "- %s", yamlQuote(c))
	}

	w.line(0, "contract:")
	w.line(1, "default_input_role: %s", yamlQuote(entry.Contract.DefaultInputRole))
	w.line(1, "default_owner: %s", yamlQuote(entry.Contract.DefaultOwner))

	w.line(1, "input_role_rules:")
	for _, rule := range entry.Contract.InputRoleRules {
		w.line(2, "- role: %s", yamlQuote(rule.Role))
		w.stringSlice(3, "exact_basenames", rule.ExactBasenames)
		w.stringSlice(3, "prefixes", rule.Prefixes)
		w.stringSlice(3, "extensions", rule.Extensions)
	}

	w.line(1, "role_owners:")
	writeMapStringString(&w, 2, entry.Contract.RoleOwners)

	w.line(1, "role_schema_refs:")
	writeMapStringString(&w, 2, entry.Contract.RoleSchemaRefs)

	w.line(1, "wet_targets:")
	for _, target := range entry.Contract.WetTargets {
		w.line(2, "- kind: %s", yamlQuote(target.Kind))
		w.line(3, "name_template: %s", yamlQuote(target.NameTemplate))
		w.line(3, "owner: %s", yamlQuote(target.Owner))
		w.line(3, "namespace: %s", yamlQuote(target.Namespace))
		w.line(3, "source_dry_path_template: %s", yamlQuote(target.SourceDryPathTemplate))
	}

	w.line(0, "provenance:")
	w.line(1, "field_origin_transform: %s", yamlQuote(entry.Provenance.FieldOriginTransform))
	w.line(1, "field_origin_overlay_transform: %s", yamlQuote(entry.Provenance.FieldOriginOverlayTransform))
	w.line(1, "field_origin_confidences:")
	writeMapStringFloat(&w, 2, entry.Provenance.FieldOriginConfidences)

	w.line(1, "rendered_lineage_templates:")
	for _, tpl := range entry.Provenance.RenderedLineageTemplates {
		w.line(2, "- kind: %s", yamlQuote(tpl.Kind))
		w.line(3, "name_template: %s", yamlQuote(tpl.NameTemplate))
		w.line(3, "namespace: %s", yamlQuote(tpl.Namespace))
		w.line(3, "source_path_hint: %s", yamlQuote(tpl.SourcePathHint))
		w.line(3, "source_path_hint_fallback: %s", yamlQuote(tpl.SourcePathHintFallback))
		w.line(3, "source_path_hint_multi: %t", tpl.SourcePathHintMulti)
		w.line(3, "source_dry_path_template: %s", yamlQuote(tpl.SourceDryPathTemplate))
		w.line(3, "optional: %t", tpl.Optional)
	}

	w.line(0, "inverse:")
	w.line(1, "inverse_patch_templates:")
	for _, key := range sortedKeysPatch(entry.Inverse.InversePatchTemplates) {
		tpl := entry.Inverse.InversePatchTemplates[key]
		w.line(2, "%s:", key)
		w.line(3, "editable_by: %s", yamlQuote(tpl.EditableBy))
		w.line(3, "confidence: %.2f", tpl.Confidence)
		w.line(3, "requires_review: %t", tpl.RequiresReview)
	}

	w.line(1, "inverse_pointer_templates:")
	for _, key := range sortedKeysPointer(entry.Inverse.InversePointerTemplates) {
		tpl := entry.Inverse.InversePointerTemplates[key]
		w.line(2, "%s:", key)
		w.line(3, "owner: %s", yamlQuote(tpl.Owner))
		w.line(3, "confidence: %.2f", tpl.Confidence)
	}

	w.line(1, "inverse_patch_reasons:")
	writeMapStringString(&w, 2, entry.Inverse.InversePatchReasons)

	w.line(1, "inverse_edit_hints:")
	writeMapStringString(&w, 2, entry.Inverse.InverseEditHints)

	w.line(0, "hints:")
	w.line(1, "defaults:")
	writeMapStringString(&w, 2, entry.Hints)

	return b.String()
}

func renderMarkdown(entry styleModel) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s Triple\n\n", entry.Kind)
	fmt.Fprintf(&b, "- Profile: `%s`\n", entry.Profile)
	fmt.Fprintf(&b, "- Resource: `%s` (`%s`)\n", entry.ResourceKind, entry.ResourceType)
	fmt.Fprintf(&b, "- Capabilities: %s\n\n", strings.Join(entry.Capabilities, ", "))

	b.WriteString("```mermaid\n")
	b.WriteString("flowchart LR\n")
	b.WriteString("  dry[\"DRY Inputs\"] --> gen[\"Generator\"] --> wet[\"WET Targets\"]\n")
	b.WriteString("```\n\n")

	b.WriteString("## Contract\n\n")
	fmt.Fprintf(&b, "- Default input role: `%s`\n", entry.Contract.DefaultInputRole)
	fmt.Fprintf(&b, "- Default owner: `%s`\n\n", entry.Contract.DefaultOwner)
	b.WriteString("### Input role rules\n\n")
	b.WriteString("| Role | Exact basenames | Prefixes | Extensions |\n")
	b.WriteString("| --- | --- | --- | --- |\n")
	for _, rule := range entry.Contract.InputRoleRules {
		fmt.Fprintf(&b, "| `%s` | %s | %s | %s |\n",
			rule.Role,
			mdCell(strings.Join(rule.ExactBasenames, ", ")),
			mdCell(strings.Join(rule.Prefixes, ", ")),
			mdCell(strings.Join(rule.Extensions, ", ")),
		)
	}
	b.WriteString("\n### Role owners\n\n")
	b.WriteString("| Role | Owner |\n")
	b.WriteString("| --- | --- |\n")
	for _, key := range sortedKeysString(entry.Contract.RoleOwners) {
		fmt.Fprintf(&b, "| `%s` | `%s` |\n", key, entry.Contract.RoleOwners[key])
	}

	b.WriteString("\n### Role schema refs\n\n")
	b.WriteString("| Role | Schema ref |\n")
	b.WriteString("| --- | --- |\n")
	for _, key := range sortedKeysString(entry.Contract.RoleSchemaRefs) {
		fmt.Fprintf(&b, "| `%s` | `%s` |\n", key, entry.Contract.RoleSchemaRefs[key])
	}

	b.WriteString("\n### WET targets\n\n")
	b.WriteString("| Kind | Name template | Owner | Namespace | Source DRY path template |\n")
	b.WriteString("| --- | --- | --- | --- | --- |\n")
	for _, target := range entry.Contract.WetTargets {
		fmt.Fprintf(&b, "| `%s` | `%s` | `%s` | `%s` | `%s` |\n",
			target.Kind,
			target.NameTemplate,
			target.Owner,
			target.Namespace,
			target.SourceDryPathTemplate,
		)
	}

	b.WriteString("\n## Provenance\n\n")
	fmt.Fprintf(&b, "- Field-origin transform: `%s`\n", entry.Provenance.FieldOriginTransform)
	fmt.Fprintf(&b, "- Field-origin overlay transform: `%s`\n\n", entry.Provenance.FieldOriginOverlayTransform)
	b.WriteString("### Field-origin confidences\n\n")
	b.WriteString("| Key | Confidence |\n")
	b.WriteString("| --- | --- |\n")
	for _, key := range sortedKeysFloat(entry.Provenance.FieldOriginConfidences) {
		fmt.Fprintf(&b, "| `%s` | %.2f |\n", key, entry.Provenance.FieldOriginConfidences[key])
	}

	b.WriteString("\n### Rendered lineage templates\n\n")
	b.WriteString("| Kind | Name template | Namespace | Source path hint | Hint fallback | Multi hint | Source DRY path template | Optional |\n")
	b.WriteString("| --- | --- | --- | --- | --- | --- | --- | --- |\n")
	for _, tpl := range entry.Provenance.RenderedLineageTemplates {
		fmt.Fprintf(&b, "| `%s` | `%s` | `%s` | `%s` | `%s` | `%t` | `%s` | `%t` |\n",
			tpl.Kind,
			tpl.NameTemplate,
			tpl.Namespace,
			tpl.SourcePathHint,
			tpl.SourcePathHintFallback,
			tpl.SourcePathHintMulti,
			tpl.SourceDryPathTemplate,
			tpl.Optional,
		)
	}

	b.WriteString("\n## Inverse\n\n")
	b.WriteString("### Inverse patch templates\n\n")
	b.WriteString("| Key | Editable by | Confidence | Requires review |\n")
	b.WriteString("| --- | --- | --- | --- |\n")
	for _, key := range sortedKeysPatch(entry.Inverse.InversePatchTemplates) {
		tpl := entry.Inverse.InversePatchTemplates[key]
		fmt.Fprintf(&b, "| `%s` | `%s` | %.2f | `%t` |\n", key, tpl.EditableBy, tpl.Confidence, tpl.RequiresReview)
	}

	b.WriteString("\n### Inverse pointer templates\n\n")
	b.WriteString("| Key | Owner | Confidence |\n")
	b.WriteString("| --- | --- | --- |\n")
	for _, key := range sortedKeysPointer(entry.Inverse.InversePointerTemplates) {
		tpl := entry.Inverse.InversePointerTemplates[key]
		fmt.Fprintf(&b, "| `%s` | `%s` | %.2f |\n", key, tpl.Owner, tpl.Confidence)
	}

	b.WriteString("\n### Inverse patch reasons\n\n")
	b.WriteString("| Key | Reason |\n")
	b.WriteString("| --- | --- |\n")
	for _, key := range sortedKeysString(entry.Inverse.InversePatchReasons) {
		fmt.Fprintf(&b, "| `%s` | %s |\n", key, mdCell(entry.Inverse.InversePatchReasons[key]))
	}

	b.WriteString("\n### Inverse edit hints\n\n")
	b.WriteString("| Key | Hint |\n")
	b.WriteString("| --- | --- |\n")
	for _, key := range sortedKeysString(entry.Inverse.InverseEditHints) {
		fmt.Fprintf(&b, "| `%s` | %s |\n", key, mdCell(entry.Inverse.InverseEditHints[key]))
	}

	b.WriteString("\n### Hint defaults\n\n")
	b.WriteString("| Key | Value |\n")
	b.WriteString("| --- | --- |\n")
	for _, key := range sortedKeysString(entry.Hints) {
		fmt.Fprintf(&b, "| `%s` | `%s` |\n", key, entry.Hints[key])
	}

	return b.String()
}

type yamlWriter struct {
	b *strings.Builder
}

func (w yamlWriter) line(indent int, format string, args ...any) {
	w.b.WriteString(strings.Repeat("  ", indent))
	w.b.WriteString(fmt.Sprintf(format, args...))
	w.b.WriteByte('\n')
}

func (w yamlWriter) stringSlice(indent int, key string, values []string) {
	w.line(indent, "%s:", key)
	if len(values) == 0 {
		w.line(indent+1, "[]")
		return
	}
	for _, value := range values {
		w.line(indent+1, "- %s", yamlQuote(value))
	}
}

func writeMapStringString(w *yamlWriter, indent int, values map[string]string) {
	keys := sortedKeysString(values)
	if len(keys) == 0 {
		w.line(indent, "{}")
		return
	}
	for _, key := range keys {
		w.line(indent, "%s: %s", key, yamlQuote(values[key]))
	}
}

func writeMapStringFloat(w *yamlWriter, indent int, values map[string]float64) {
	keys := sortedKeysFloat(values)
	if len(keys) == 0 {
		w.line(indent, "{}")
		return
	}
	for _, key := range keys {
		w.line(indent, "%s: %.2f", key, values[key])
	}
}

func yamlQuote(value string) string {
	return strconv.Quote(value)
}

func mdCell(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	value = strings.ReplaceAll(value, "|", "\\|")
	value = strings.ReplaceAll(value, "\n", "<br/>")
	return value
}

func copyMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func copyMapFloat(in map[string]float64) map[string]float64 {
	out := make(map[string]float64, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func copyMapPatch(in map[string]registry.InversePatchTemplate) map[string]registry.InversePatchTemplate {
	out := make(map[string]registry.InversePatchTemplate, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func copyMapPointer(in map[string]registry.InversePointerTemplate) map[string]registry.InversePointerTemplate {
	out := make(map[string]registry.InversePointerTemplate, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func sortedKeysString(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedKeysFloat(values map[string]float64) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedKeysPatch(values map[string]registry.InversePatchTemplate) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedKeysPointer(values map[string]registry.InversePointerTemplate) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
