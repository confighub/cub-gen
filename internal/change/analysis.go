package change

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/confighub/cub-gen/internal/model"
	"gopkg.in/yaml.v3"
)

const (
	helmCLIOverrideTransform      = "helm-cli-override"
	helmExternalTransform         = "helm-external"
	helmBuiltinTransform          = "helm-builtin"
	helmHelperTransform           = "helm-helper"
	helmNonDeterministicTransform = "helm-nondeterministic"
	helmDefaultTransform          = "helm-default"
)

type ProvenanceHop struct {
	GeneratorKind    string  `json:"generator_kind,omitempty"`
	GeneratorProfile string  `json:"generator_profile,omitempty"`
	DryPath          string  `json:"dry_path,omitempty"`
	SourcePath       string  `json:"source_path,omitempty"`
	SourceTransform  string  `json:"source_transform,omitempty"`
	Confidence       float64 `json:"confidence,omitempty"`
}

type ImpactEntry struct {
	Owner            string          `json:"owner,omitempty"`
	WetPath          string          `json:"wet_path"`
	DryPath          string          `json:"dry_path"`
	EditHint         string          `json:"edit_hint,omitempty"`
	Confidence       float64         `json:"confidence"`
	SourcePath       string          `json:"source_path,omitempty"`
	SourceTransform  string          `json:"source_transform,omitempty"`
	OriginType       string          `json:"origin_type,omitempty"`
	GeneratorName    string          `json:"generator_name,omitempty"`
	GeneratorProfile string          `json:"generator_profile,omitempty"`
	Hops             []ProvenanceHop `json:"hops,omitempty"`
	Warning          string          `json:"warning,omitempty"`
}

type ExplainSuggestion struct {
	Owner            string          `json:"owner"`
	WetPath          string          `json:"wet_path"`
	DryPath          string          `json:"dry_path"`
	EditHint         string          `json:"edit_hint"`
	Confidence       float64         `json:"confidence"`
	SourcePath       string          `json:"source_path,omitempty"`
	SourceTransform  string          `json:"source_transform,omitempty"`
	OriginType       string          `json:"origin_type,omitempty"`
	GeneratorName    string          `json:"generator_name,omitempty"`
	GeneratorProfile string          `json:"generator_profile,omitempty"`
	Hops             []ProvenanceHop `json:"hops,omitempty"`
	Warning          string          `json:"warning,omitempty"`
}

type DiffFieldState struct {
	Exists           bool            `json:"exists"`
	Value            any             `json:"value,omitempty"`
	Owner            string          `json:"owner,omitempty"`
	WetPath          string          `json:"wet_path,omitempty"`
	DryPath          string          `json:"dry_path,omitempty"`
	EditHint         string          `json:"edit_hint,omitempty"`
	Confidence       float64         `json:"confidence,omitempty"`
	SourcePath       string          `json:"source_path,omitempty"`
	SourceTransform  string          `json:"source_transform,omitempty"`
	OriginType       string          `json:"origin_type,omitempty"`
	GeneratorName    string          `json:"generator_name,omitempty"`
	GeneratorProfile string          `json:"generator_profile,omitempty"`
	Hops             []ProvenanceHop `json:"hops,omitempty"`
	Warning          string          `json:"warning,omitempty"`
}

type ManifestRef struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
}

type DiffFieldKey struct {
	Manifest ManifestRef
	Field    string
}

type DiffEntry struct {
	Manifest ManifestRef    `json:"manifest"`
	WetPath  string         `json:"wet_path"`
	Before   DiffFieldState `json:"before"`
	After    DiffFieldState `json:"after"`
}

type RevisionDiffCause struct {
	Owner                 string          `json:"owner,omitempty"`
	EditHint              string          `json:"edit_hint,omitempty"`
	Confidence            float64         `json:"confidence,omitempty"`
	BeforeDryPath         string          `json:"before_dry_path,omitempty"`
	AfterDryPath          string          `json:"after_dry_path,omitempty"`
	BeforeSourcePath      string          `json:"before_source_path,omitempty"`
	AfterSourcePath       string          `json:"after_source_path,omitempty"`
	BeforeSourceTransform string          `json:"before_source_transform,omitempty"`
	AfterSourceTransform  string          `json:"after_source_transform,omitempty"`
	BeforeOriginType      string          `json:"before_origin_type,omitempty"`
	AfterOriginType       string          `json:"after_origin_type,omitempty"`
	Hops                  []ProvenanceHop `json:"hops,omitempty"`
	Warning               string          `json:"warning,omitempty"`
}

func DiffStateFromProvenance(provenance []model.ProvenanceRecord, wetPath string, value any) DiffFieldState {
	best := DiffFieldState{Exists: true, Value: value, WetPath: wetPath}
	bestConfidence := -1.0
	for _, record := range provenance {
		origin, ok := BestFieldOrigin(record.FieldOriginMap, wetPath, "")
		if !ok {
			continue
		}
		candidate := DiffFieldState{
			Exists:           true,
			Value:            value,
			Owner:            "",
			WetPath:          wetPath,
			DryPath:          origin.DryPath,
			EditHint:         "",
			Confidence:       origin.Confidence,
			SourcePath:       origin.SourcePath,
			SourceTransform:  origin.Transform,
			OriginType:       explainOriginType(origin.SourcePath, origin.Transform),
			GeneratorName:    record.GeneratorName,
			GeneratorProfile: record.GeneratorProfile,
			Hops:             provenanceHops(origin),
		}
		if pointer, ok := bestInversePointer(record.InverseEditPointers, wetPath, origin.DryPath); ok {
			pointer = applyFieldOriginToPointer(pointer, origin)
			candidate.Owner = pointer.Owner
			candidate.EditHint = pointer.EditHint
			if pointer.Confidence > candidate.Confidence {
				candidate.Confidence = pointer.Confidence
			}
		}
		switch origin.Transform {
		case helmCLIOverrideTransform:
			candidate.Warning = overrideAwareWarning()
		case helmExternalTransform:
			candidate.Warning = externalAwareWarning(origin.SourcePath)
		case helmBuiltinTransform:
			candidate.Warning = builtinAwareWarning(origin.SourcePath)
		case helmHelperTransform:
			candidate.Warning = helperAwareWarning()
		case helmNonDeterministicTransform:
			candidate.Warning = nonDeterministicAwareWarning(origin.SourcePath)
		case helmDefaultTransform:
			candidate.Warning = defaultAwareWarning()
		}
		if candidate.Confidence > bestConfidence {
			best = candidate
			bestConfidence = candidate.Confidence
		}
	}
	return best
}

func FlattenRenderedManifestFields(rendered string) (map[DiffFieldKey]any, error) {
	out := map[DiffFieldKey]any{}
	dec := yaml.NewDecoder(strings.NewReader(rendered))
	docIndex := 0
	for {
		var raw any
		if err := dec.Decode(&raw); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("decode rendered yaml: %w", err)
		}
		docIndex++
		converted := yamlToJSONCompatible(raw)
		if converted == nil {
			continue
		}
		obj, ok := converted.(map[string]any)
		if !ok || len(obj) == 0 {
			continue
		}
		ref := renderedManifestRef(obj, docIndex)
		flattenManifestValue(out, ref, "", obj)
	}
	return out, nil
}

func CollectDiffEntries(
	beforeFields, afterFields map[DiffFieldKey]any,
	beforeProvenance, afterProvenance []model.ProvenanceRecord,
	dryFilter, wetFilter, ownerFilter string,
) []DiffEntry {
	keys := make([]DiffFieldKey, 0, len(beforeFields)+len(afterFields))
	seen := map[DiffFieldKey]struct{}{}
	for key := range beforeFields {
		seen[key] = struct{}{}
		keys = append(keys, key)
	}
	for key := range afterFields {
		if _, ok := seen[key]; ok {
			continue
		}
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		left, right := keys[i], keys[j]
		if left.Manifest.Kind != right.Manifest.Kind {
			return left.Manifest.Kind < right.Manifest.Kind
		}
		if left.Manifest.Name != right.Manifest.Name {
			return left.Manifest.Name < right.Manifest.Name
		}
		if left.Manifest.Namespace != right.Manifest.Namespace {
			return left.Manifest.Namespace < right.Manifest.Namespace
		}
		return left.Field < right.Field
	})

	out := make([]DiffEntry, 0)
	for _, key := range keys {
		beforeValue, beforeExists := beforeFields[key]
		afterValue, afterExists := afterFields[key]
		if beforeExists && afterExists && valuesEqual(beforeValue, afterValue) {
			continue
		}

		entry := DiffEntry{
			Manifest: key.Manifest,
			WetPath:  key.Manifest.Kind + "/" + key.Field,
			Before:   DiffFieldState{Exists: beforeExists, Value: beforeValue},
			After:    DiffFieldState{Exists: afterExists, Value: afterValue},
		}
		if beforeExists {
			entry.Before = DiffStateFromProvenance(beforeProvenance, entry.WetPath, beforeValue)
		}
		if afterExists {
			entry.After = DiffStateFromProvenance(afterProvenance, entry.WetPath, afterValue)
		}
		if !diffMatchesFilters(entry, dryFilter, wetFilter, ownerFilter) {
			continue
		}
		out = append(out, entry)
	}
	return out
}

func BuildRevisionDiffCause(entry DiffEntry) RevisionDiffCause {
	owner := strings.TrimSpace(entry.After.Owner)
	if owner == "" {
		owner = strings.TrimSpace(entry.Before.Owner)
	}
	editHint := strings.TrimSpace(entry.After.EditHint)
	if editHint == "" {
		editHint = strings.TrimSpace(entry.Before.EditHint)
	}
	confidence := entry.After.Confidence
	if confidence == 0 {
		confidence = entry.Before.Confidence
	}
	hops := entry.After.Hops
	if len(hops) == 0 {
		hops = entry.Before.Hops
	}
	warning := strings.TrimSpace(entry.After.Warning)
	if warning == "" {
		warning = strings.TrimSpace(entry.Before.Warning)
	}

	return RevisionDiffCause{
		Owner:                 owner,
		EditHint:              editHint,
		Confidence:            confidence,
		BeforeDryPath:         entry.Before.DryPath,
		AfterDryPath:          entry.After.DryPath,
		BeforeSourcePath:      entry.Before.SourcePath,
		AfterSourcePath:       entry.After.SourcePath,
		BeforeSourceTransform: entry.Before.SourceTransform,
		AfterSourceTransform:  entry.After.SourceTransform,
		BeforeOriginType:      entry.Before.OriginType,
		AfterOriginType:       entry.After.OriginType,
		Hops:                  hops,
		Warning:               warning,
	}
}

func BestInverseEditPointer(provenance []model.ProvenanceRecord) (model.InverseEditPointer, bool) {
	best := model.InverseEditPointer{}
	bestConfidence := -1.0
	for _, record := range provenance {
		for _, pointer := range record.InverseEditPointers {
			candidate := pointer
			if source, ok := BestFieldOrigin(record.FieldOriginMap, pointer.WetPath, pointer.DryPath); ok {
				candidate = applyFieldOriginToPointer(candidate, source)
			}
			if candidate.Confidence > bestConfidence {
				best = candidate
				bestConfidence = candidate.Confidence
			}
		}
	}
	if bestConfidence < 0 {
		return model.InverseEditPointer{}, false
	}
	return best, true
}

func PickInverseSuggestion(
	provenance []model.ProvenanceRecord,
	wetFilter, dryFilter, ownerFilter string,
) (ExplainSuggestion, int, bool) {
	matchCount := 0
	best := ExplainSuggestion{}
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

			var origin model.FieldOrigin
			sourcePath := ""
			sourceTransform := ""
			sourceConfidence := 0.0
			if source, ok := BestFieldOrigin(record.FieldOriginMap, pointer.WetPath, pointer.DryPath); ok {
				origin = source
				sourcePath = source.SourcePath
				sourceTransform = source.Transform
				sourceConfidence = source.Confidence
			}

			candidate := ExplainSuggestion{
				Owner:            pointer.Owner,
				WetPath:          pointer.WetPath,
				DryPath:          pointer.DryPath,
				EditHint:         pointer.EditHint,
				Confidence:       pointer.Confidence,
				SourcePath:       sourcePath,
				SourceTransform:  sourceTransform,
				OriginType:       explainOriginType(sourcePath, sourceTransform),
				GeneratorName:    record.GeneratorName,
				GeneratorProfile: record.GeneratorProfile,
				Hops:             provenanceHops(origin),
			}
			if sourceTransform == helmCLIOverrideTransform {
				candidate.Owner = "release-automation"
				candidate.EditHint = overrideAwareEditHint(sourcePath, pointer.EditHint)
				candidate.Warning = overrideAwareWarning()
				if sourceConfidence > candidate.Confidence {
					candidate.Confidence = sourceConfidence
				}
			}
			if sourceTransform == helmExternalTransform {
				candidate.EditHint = externalAwareEditHint(sourcePath, pointer.EditHint)
				candidate.Warning = externalAwareWarning(sourcePath)
				if sourceConfidence > candidate.Confidence {
					candidate.Confidence = sourceConfidence
				}
			}
			if sourceTransform == helmBuiltinTransform {
				candidate.Warning = builtinAwareWarning(sourcePath)
				if sourceConfidence > candidate.Confidence {
					candidate.Confidence = sourceConfidence
				}
			}
			if sourceTransform == helmHelperTransform {
				candidate.Warning = helperAwareWarning()
				if sourceConfidence > candidate.Confidence {
					candidate.Confidence = sourceConfidence
				}
			}
			if sourceTransform == helmNonDeterministicTransform {
				candidate.Warning = nonDeterministicAwareWarning(sourcePath)
				if sourceConfidence > candidate.Confidence {
					candidate.Confidence = sourceConfidence
				}
			}
			if sourceTransform == helmDefaultTransform {
				candidate.Warning = defaultAwareWarning()
				if sourceConfidence > candidate.Confidence {
					candidate.Confidence = sourceConfidence
				}
			}
			if candidate.Confidence > bestConfidence {
				best = candidate
				bestConfidence = candidate.Confidence
			}
		}
	}

	if bestConfidence < 0 {
		return ExplainSuggestion{}, 0, false
	}
	return best, matchCount, true
}

func CollectImpactSuggestions(
	provenance []model.ProvenanceRecord,
	dryFilter, wetFilter, ownerFilter string,
) ([]ImpactEntry, int, bool) {
	impacts := make([]ImpactEntry, 0)

	for _, record := range provenance {
		for _, origin := range record.FieldOriginMap {
			if dryFilter != "" && origin.DryPath != dryFilter {
				continue
			}
			if wetFilter != "" && origin.WetPath != wetFilter {
				continue
			}

			pointer, ok := bestInversePointer(record.InverseEditPointers, origin.WetPath, origin.DryPath)
			entry := ImpactEntry{
				Owner:            "",
				WetPath:          origin.WetPath,
				DryPath:          origin.DryPath,
				EditHint:         "",
				Confidence:       origin.Confidence,
				SourcePath:       origin.SourcePath,
				SourceTransform:  origin.Transform,
				OriginType:       explainOriginType(origin.SourcePath, origin.Transform),
				GeneratorName:    record.GeneratorName,
				GeneratorProfile: record.GeneratorProfile,
				Hops:             provenanceHops(origin),
			}
			if ok {
				entry.Owner = pointer.Owner
				entry.EditHint = pointer.EditHint
			}
			switch origin.Transform {
			case helmCLIOverrideTransform:
				entry.Owner = "release-automation"
				entry.EditHint = overrideAwareEditHint(origin.SourcePath, entry.EditHint)
				entry.Warning = overrideAwareWarning()
			case helmExternalTransform:
				entry.EditHint = externalAwareEditHint(origin.SourcePath, entry.EditHint)
				entry.Warning = externalAwareWarning(origin.SourcePath)
			case helmBuiltinTransform:
				entry.Warning = builtinAwareWarning(origin.SourcePath)
			case helmHelperTransform:
				entry.Warning = helperAwareWarning()
			case helmNonDeterministicTransform:
				entry.Warning = nonDeterministicAwareWarning(origin.SourcePath)
			case helmDefaultTransform:
				entry.Warning = defaultAwareWarning()
			}
			if ownerFilter != "" && entry.Owner != ownerFilter {
				continue
			}
			impacts = append(impacts, entry)
		}
	}

	sortImpactEntries(impacts)
	if len(impacts) == 0 {
		return nil, 0, false
	}
	return impacts, len(impacts), true
}

func BestFieldOrigin(origins []model.FieldOrigin, wetPath, dryPath string) (model.FieldOrigin, bool) {
	if wetPath != "" && dryPath != "" {
		return bestMatchingFieldOrigin(origins, func(origin model.FieldOrigin) bool {
			return origin.WetPath == wetPath && origin.DryPath == dryPath
		})
	}
	if dryPath != "" {
		return bestMatchingFieldOrigin(origins, func(origin model.FieldOrigin) bool {
			return origin.DryPath == dryPath
		})
	}
	if wetPath != "" {
		return bestMatchingFieldOrigin(origins, func(origin model.FieldOrigin) bool {
			return origin.WetPath == wetPath
		})
	}
	return bestMatchingFieldOrigin(origins, func(model.FieldOrigin) bool {
		return true
	})
}

func bestMatchingFieldOrigin(origins []model.FieldOrigin, match func(model.FieldOrigin) bool) (model.FieldOrigin, bool) {
	best := model.FieldOrigin{}
	bestConfidence := -1.0
	for _, origin := range origins {
		if !match(origin) {
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

func bestInversePointer(pointers []model.InverseEditPointer, wetPath, dryPath string) (model.InverseEditPointer, bool) {
	best := model.InverseEditPointer{}
	bestConfidence := -1.0

	for _, pointer := range pointers {
		if wetPath != "" && pointer.WetPath != wetPath {
			continue
		}
		if dryPath != "" && pointer.DryPath != dryPath {
			continue
		}
		if pointer.Confidence > bestConfidence {
			best = pointer
			bestConfidence = pointer.Confidence
		}
	}
	if bestConfidence >= 0 {
		return best, true
	}

	for _, pointer := range pointers {
		if dryPath != "" && pointer.DryPath != dryPath {
			continue
		}
		if pointer.Confidence > bestConfidence {
			best = pointer
			bestConfidence = pointer.Confidence
		}
	}
	if bestConfidence >= 0 {
		return best, true
	}

	for _, pointer := range pointers {
		if wetPath != "" && pointer.WetPath != wetPath {
			continue
		}
		if pointer.Confidence > bestConfidence {
			best = pointer
			bestConfidence = pointer.Confidence
		}
	}
	if bestConfidence < 0 {
		return model.InverseEditPointer{}, false
	}
	return best, true
}

func provenanceHops(origin model.FieldOrigin) []ProvenanceHop {
	if len(origin.Hops) == 0 {
		return nil
	}
	hops := make([]ProvenanceHop, 0, len(origin.Hops))
	for _, hop := range origin.Hops {
		hops = append(hops, ProvenanceHop{
			GeneratorKind:    hop.GeneratorKind,
			GeneratorProfile: hop.GeneratorProfile,
			DryPath:          hop.DryPath,
			SourcePath:       hop.SourcePath,
			SourceTransform:  hop.Transform,
			Confidence:       hop.Confidence,
		})
	}
	return hops
}

func applyFieldOriginToPointer(pointer model.InverseEditPointer, origin model.FieldOrigin) model.InverseEditPointer {
	switch origin.Transform {
	case helmCLIOverrideTransform:
		pointer.Owner = "release-automation"
		pointer.EditHint = overrideAwareEditHint(origin.SourcePath, pointer.EditHint)
	case helmExternalTransform:
		pointer.EditHint = externalAwareEditHint(origin.SourcePath, pointer.EditHint)
	case helmBuiltinTransform, helmHelperTransform, helmNonDeterministicTransform, helmDefaultTransform:
	default:
		return pointer
	}
	if origin.Confidence > pointer.Confidence {
		pointer.Confidence = origin.Confidence
	}
	return pointer
}

func overrideAwareEditHint(sourcePath, fallback string) string {
	if strings.TrimSpace(sourcePath) == "" {
		return fallback
	}
	return fmt.Sprintf("This render used %s. Edit that Helm CLI override or the CI/pipeline step that passes it before changing values files.", sourcePath)
}

func overrideAwareWarning() string {
	return "This field was overridden by the Helm CLI invocation for this run, so editing values files alone will not change the rendered output."
}

func externalAwareEditHint(sourcePath, fallback string) string {
	if strings.TrimSpace(sourcePath) == "" {
		return fallback
	}
	return fmt.Sprintf("This field is declared as an external reference in %s. Update that external source or change the reference in %s before editing local values.", sourcePath, sourcePath)
}

func externalAwareWarning(sourcePath string) string {
	if strings.TrimSpace(sourcePath) == "" {
		return "This field is sourced externally, so provenance ends at that external system instead of a repo-owned value."
	}
	return fmt.Sprintf("This field is sourced externally via %s, so provenance ends at that external system instead of a repo-owned value.", sourcePath)
}

func builtinAwareWarning(sourcePath string) string {
	if sourcePath == "<helm-builtin>:.Chart.AppVersion" {
		return "This field currently comes from the Helm built-in .Chart.AppVersion path, not from an observed values file."
	}
	if strings.HasPrefix(sourcePath, "<helm-builtin>:.Files.Get(") {
		return "This field currently comes from Helm built-in .Files.Get chart-file input, not from an observed values file."
	}
	return "This field currently comes from a Helm built-in source, not from an observed values file."
}

func helperAwareWarning() string {
	return "This field is routed through a Helm helper chain before it reaches the rendered manifest, so cub-gen traced the helper back to the underlying values source."
}

func nonDeterministicAwareWarning(sourcePath string) string {
	fn := strings.TrimPrefix(sourcePath, "<helm-nondeterministic>:")
	if strings.TrimSpace(fn) == "" || fn == sourcePath {
		return "This field currently comes from Helm render-time logic, so provenance ends at render time instead of a stable DRY file."
	}
	return fmt.Sprintf("This field currently comes from Helm render-time %s logic, so provenance ends at render time instead of a stable DRY file.", fn)
}

func defaultAwareWarning() string {
	return "This field is not set in the observed values files, so chart defaults or helper logic are likely supplying it."
}

func explainOriginType(sourcePath, sourceTransform string) string {
	switch {
	case sourceTransform == helmCLIOverrideTransform:
		return "external"
	case sourceTransform == helmExternalTransform:
		return "external"
	case sourceTransform == helmBuiltinTransform:
		return "builtin"
	case sourceTransform == helmHelperTransform:
		return "helper"
	case sourceTransform == helmNonDeterministicTransform:
		return "non-deterministic"
	case sourceTransform == helmDefaultTransform:
		return "default"
	case strings.TrimSpace(sourcePath) != "":
		return "dry-file"
	default:
		return ""
	}
}

func sortImpactEntries(impacts []ImpactEntry) {
	sort.Slice(impacts, func(i, j int) bool {
		if impacts[i].DryPath != impacts[j].DryPath {
			return impacts[i].DryPath < impacts[j].DryPath
		}
		if impacts[i].WetPath != impacts[j].WetPath {
			return impacts[i].WetPath < impacts[j].WetPath
		}
		if impacts[i].Confidence != impacts[j].Confidence {
			return impacts[i].Confidence > impacts[j].Confidence
		}
		if impacts[i].SourcePath != impacts[j].SourcePath {
			return impacts[i].SourcePath < impacts[j].SourcePath
		}
		return impacts[i].GeneratorName < impacts[j].GeneratorName
	})
}

func diffMatchesFilters(entry DiffEntry, dryFilter, wetFilter, ownerFilter string) bool {
	if wetFilter != "" && entry.WetPath != wetFilter {
		return false
	}
	if dryFilter != "" {
		match := (entry.Before.Exists && entry.Before.DryPath == dryFilter) || (entry.After.Exists && entry.After.DryPath == dryFilter)
		if !match {
			return false
		}
	}
	if ownerFilter != "" {
		match := (entry.Before.Exists && entry.Before.Owner == ownerFilter) || (entry.After.Exists && entry.After.Owner == ownerFilter)
		if !match {
			return false
		}
	}
	return true
}

func renderedManifestRef(obj map[string]any, idx int) ManifestRef {
	ref := ManifestRef{
		Kind: stringValue(obj["kind"]),
	}
	if ref.Kind == "" {
		ref.Kind = fmt.Sprintf("Doc%d", idx)
	}
	if metadata, ok := obj["metadata"].(map[string]any); ok {
		ref.Name = stringValue(metadata["name"])
		ref.Namespace = stringValue(metadata["namespace"])
	}
	if ref.Name == "" {
		ref.Name = fmt.Sprintf("doc-%d", idx)
	}
	return ref
}

func flattenManifestValue(out map[DiffFieldKey]any, ref ManifestRef, prefix string, value any) {
	switch typed := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			next := key
			if prefix != "" {
				next = prefix + "/" + key
			}
			flattenManifestValue(out, ref, next, typed[key])
		}
	case []any:
		for i, item := range typed {
			next := fmt.Sprintf("%s[%d]", prefix, i)
			flattenManifestValue(out, ref, next, item)
		}
	default:
		if prefix == "" {
			return
		}
		out[DiffFieldKey{Manifest: ref, Field: prefix}] = typed
	}
}

func yamlToJSONCompatible(v any) any {
	switch typed := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, value := range typed {
			out[key] = yamlToJSONCompatible(value)
		}
		return out
	case map[any]any:
		out := make(map[string]any, len(typed))
		for key, value := range typed {
			out[fmt.Sprint(key)] = yamlToJSONCompatible(value)
		}
		return out
	case []any:
		out := make([]any, 0, len(typed))
		for _, value := range typed {
			out = append(out, yamlToJSONCompatible(value))
		}
		return out
	default:
		return typed
	}
}

func stringValue(v any) string {
	s, _ := v.(string)
	return strings.TrimSpace(s)
}

func valuesEqual(left, right any) bool {
	return fmt.Sprintf("%#v", left) == fmt.Sprintf("%#v", right)
}
