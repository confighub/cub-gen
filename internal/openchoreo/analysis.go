package openchoreo

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const SupportedAPIVersion = "openchoreo.dev/v1alpha1"

type Artifact struct {
	Path       string `json:"path"`
	APIVersion string `json:"api_version"`
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	Role       string `json:"role"`
}

type Diagnostic struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Path    string `json:"path,omitempty"`
}

type Analysis struct {
	Artifacts   []Artifact   `json:"artifacts"`
	Inputs      []string     `json:"inputs"`
	Variants    []string     `json:"variants,omitempty"`
	Workload    string       `json:"workload,omitempty"`
	Diagnostics []Diagnostic `json:"diagnostics,omitempty"`

	workloadSecretRefs []workloadSecretRef
}

type workloadSecretRef struct {
	Path string
	Name string
}

type yamlDoc struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name            string `yaml:"name"`
		OwnerReferences []struct {
			APIVersion string `yaml:"apiVersion"`
			Kind       string `yaml:"kind"`
			Name       string `yaml:"name"`
		} `yaml:"ownerReferences"`
	} `yaml:"metadata"`
	Spec map[string]any `yaml:"spec"`
}

func AnalyzeRepo(root string) (Analysis, error) {
	var analysis Analysis
	if strings.TrimSpace(root) == "" {
		return analysis, errors.New("openchoreo repo path is required")
	}

	if err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if shouldSkipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(d.Name()))
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !isCandidateOpenChoreoPath(rel, raw) {
			return nil
		}
		docs, err := readYAMLDocs(raw)
		if err != nil {
			if !bytes.Contains(bytes.ToLower(raw), []byte("openchoreo")) {
				return nil
			}
			return fmt.Errorf("parse %s: %w", rel, err)
		}
		for _, doc := range docs {
			analysis.addDoc(rel, doc)
		}
		return nil
	}); err != nil {
		return Analysis{}, err
	}

	analysis.finalize()
	return analysis, nil
}

func (a Analysis) HasOpenChoreo() bool {
	return len(a.Artifacts) > 0 || len(a.Diagnostics) > 0
}

func (a Analysis) HasWorkload() bool {
	for _, artifact := range a.Artifacts {
		if artifact.Kind == "Workload" {
			return true
		}
	}
	return false
}

func (a Analysis) BlockingDiagnostics() []Diagnostic {
	var out []Diagnostic
	for _, diagnostic := range a.Diagnostics {
		switch diagnostic.Code {
		case "unsupported_openchoreo_api_version", "unsupported_openchoreo_kind", "missing_component_type", "missing_release_binding", "missing_rendered_release", "unresolved_secret_reference":
			out = append(out, diagnostic)
		case "unknown_rendered_owner":
			if a.HasWorkload() {
				out = append(out, diagnostic)
			}
		}
	}
	return out
}

func (a *Analysis) addDoc(rel string, doc yamlDoc) {
	apiVersion := strings.TrimSpace(doc.APIVersion)
	kind := canonicalKind(doc.Kind)
	if !strings.Contains(strings.ToLower(apiVersion), "openchoreo") {
		if strings.HasPrefix(rel, "rendered/") && isRenderedKubernetesKind(kind) {
			if !hasRenderedReleaseOwnerReference(doc) {
				a.Diagnostics = append(a.Diagnostics, Diagnostic{
					Code:    "unknown_rendered_owner",
					Message: "rendered Kubernetes resource lacks an OpenChoreo RenderedRelease ownerReference; refusing to guess generated-resource ownership",
					Path:    rel,
				})
			}
			a.addArtifact(Artifact{
				Path:       rel,
				APIVersion: apiVersion,
				Kind:       kind,
				Name:       strings.TrimSpace(doc.Metadata.Name),
				Role:       "rendered-manifest",
			})
		}
		return
	}
	if apiVersion != SupportedAPIVersion {
		a.Diagnostics = append(a.Diagnostics, Diagnostic{
			Code:    "unsupported_openchoreo_api_version",
			Message: fmt.Sprintf("unsupported OpenChoreo apiVersion %q; supported version is %s", apiVersion, SupportedAPIVersion),
			Path:    rel,
		})
		return
	}
	role := roleForKind(kind)
	if role == "" {
		a.Diagnostics = append(a.Diagnostics, Diagnostic{
			Code:    "unsupported_openchoreo_kind",
			Message: fmt.Sprintf("unsupported OpenChoreo kind %q; refusing to guess lineage", kind),
			Path:    rel,
		})
		return
	}
	a.addArtifact(Artifact{
		Path:       rel,
		APIVersion: apiVersion,
		Kind:       kind,
		Name:       strings.TrimSpace(doc.Metadata.Name),
		Role:       role,
	})
	if kind == "Workload" && a.Workload == "" {
		a.Workload = strings.TrimSpace(doc.Metadata.Name)
	}
	if variant := variantNameFromPath(rel); variant != "" {
		a.Variants = append(a.Variants, variant)
	}
	for _, secretRef := range secretRefsInSpec(doc.Spec) {
		if secretRef == "" {
			continue
		}
		a.workloadSecretRefs = append(a.workloadSecretRefs, workloadSecretRef{Path: rel, Name: secretRef})
	}
}

func (a *Analysis) addArtifact(artifact Artifact) {
	if artifact.Path == "" || artifact.Kind == "" {
		return
	}
	a.Artifacts = append(a.Artifacts, artifact)
	a.Inputs = append(a.Inputs, artifact.Path)
}

func (a *Analysis) finalize() {
	a.Inputs = uniqueSorted(a.Inputs)
	a.Variants = uniqueSorted(a.Variants)
	sort.Slice(a.Artifacts, func(i, j int) bool {
		if a.Artifacts[i].Role != a.Artifacts[j].Role {
			return a.Artifacts[i].Role < a.Artifacts[j].Role
		}
		return a.Artifacts[i].Path < a.Artifacts[j].Path
	})
	if !a.HasOpenChoreo() {
		return
	}
	if a.HasWorkload() && !a.hasRole("component-type") {
		a.Diagnostics = append(a.Diagnostics, Diagnostic{
			Code:    "missing_component_type",
			Message: "OpenChoreo Workload found but no ComponentType was found; refusing to infer platform template lineage",
		})
	}
	if a.HasWorkload() && !a.hasRole("release-binding") {
		a.Diagnostics = append(a.Diagnostics, Diagnostic{
			Code:    "missing_release_binding",
			Message: "OpenChoreo Workload found but no ReleaseBinding was found; deployable variants are unknown",
		})
	}
	if a.HasWorkload() && !a.hasRole("rendered-release") {
		a.Diagnostics = append(a.Diagnostics, Diagnostic{
			Code:    "missing_rendered_release",
			Message: "OpenChoreo Workload found but no RenderedRelease was found; rendered ownership cannot be proven",
		})
	}
	for _, ref := range a.workloadSecretRefs {
		if !a.hasSecretReference(ref.Name) {
			a.Diagnostics = append(a.Diagnostics, Diagnostic{
				Code:    "unresolved_secret_reference",
				Message: fmt.Sprintf("Workload references SecretReference %q but no matching SecretReference artifact was found", ref.Name),
				Path:    ref.Path,
			})
		}
	}
	sort.Slice(a.Diagnostics, func(i, j int) bool {
		if a.Diagnostics[i].Code != a.Diagnostics[j].Code {
			return a.Diagnostics[i].Code < a.Diagnostics[j].Code
		}
		return a.Diagnostics[i].Path < a.Diagnostics[j].Path
	})
}

func (a Analysis) hasRole(role string) bool {
	for _, artifact := range a.Artifacts {
		if artifact.Role == role {
			return true
		}
	}
	return false
}

func (a Analysis) hasSecretReference(name string) bool {
	for _, artifact := range a.Artifacts {
		if artifact.Kind == "SecretReference" && artifact.Name == name {
			return true
		}
	}
	return false
}

func readYAMLDocs(raw []byte) ([]yamlDoc, error) {
	dec := yaml.NewDecoder(bytes.NewReader(raw))
	var docs []yamlDoc
	for {
		var doc yamlDoc
		err := dec.Decode(&doc)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(doc.Kind) == "" {
			continue
		}
		docs = append(docs, doc)
	}
	return docs, nil
}

func isCandidateOpenChoreoPath(rel string, raw []byte) bool {
	normalized := filepath.ToSlash(rel)
	if bytes.Contains(bytes.ToLower(raw), []byte("openchoreo")) {
		return true
	}
	return strings.HasPrefix(normalized, "rendered/")
}

func roleForKind(kind string) string {
	switch kind {
	case "Component":
		return "component"
	case "Workload":
		return "workload"
	case "ComponentType", "ClusterComponentType":
		return "component-type"
	case "ReleaseBinding":
		return "release-binding"
	case "SecretReference":
		return "secret-reference"
	case "RenderedRelease":
		return "rendered-release"
	default:
		return ""
	}
}

func canonicalKind(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "component":
		return "Component"
	case "workload":
		return "Workload"
	case "componenttype":
		return "ComponentType"
	case "clustercomponenttype":
		return "ClusterComponentType"
	case "releasebinding":
		return "ReleaseBinding"
	case "secretreference":
		return "SecretReference"
	case "renderedrelease":
		return "RenderedRelease"
	case "deployment":
		return "Deployment"
	case "service":
		return "Service"
	case "configmap":
		return "ConfigMap"
	case "secret":
		return "Secret"
	default:
		return strings.TrimSpace(kind)
	}
}

func isRenderedKubernetesKind(kind string) bool {
	switch kind {
	case "Deployment", "Service", "ConfigMap", "Secret":
		return true
	default:
		return false
	}
}

func hasRenderedReleaseOwnerReference(doc yamlDoc) bool {
	for _, owner := range doc.Metadata.OwnerReferences {
		if strings.TrimSpace(owner.APIVersion) == SupportedAPIVersion && canonicalKind(owner.Kind) == "RenderedRelease" && strings.TrimSpace(owner.Name) != "" {
			return true
		}
	}
	return false
}

func secretRefsInSpec(spec map[string]any) []string {
	var out []string
	var walk func(any)
	walk = func(v any) {
		switch typed := v.(type) {
		case map[string]any:
			for key, value := range typed {
				lowerKey := strings.ToLower(key)
				if lowerKey == "secretref" || lowerKey == "secretreference" || lowerKey == "secretreferenceref" {
					out = append(out, secretRefName(value))
				}
				walk(value)
			}
		case []any:
			for _, item := range typed {
				walk(item)
			}
		}
	}
	walk(spec)
	return uniqueSorted(out)
}

func secretRefName(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case map[string]any:
		for _, key := range []string{"name", "secretName", "ref"} {
			if raw, ok := typed[key].(string); ok {
				return strings.TrimSpace(raw)
			}
		}
	}
	return ""
}

func variantNameFromPath(path string) string {
	parts := strings.Split(filepath.ToSlash(path), "/")
	for i, part := range parts {
		switch part {
		case "envs", "environments", "rendered":
			if i+1 < len(parts) && parts[i+1] != "" {
				return parts[i+1]
			}
		}
	}
	base := strings.ToLower(filepath.Base(path))
	for _, suffix := range []string{".yaml", ".yml"} {
		base = strings.TrimSuffix(base, suffix)
	}
	for _, prefix := range []string{"release-binding-", "rendered-release-"} {
		if strings.HasPrefix(base, prefix) {
			return strings.TrimPrefix(base, prefix)
		}
	}
	return ""
}

func shouldSkipDir(name string) bool {
	switch name {
	case ".git", ".cub-gen", "node_modules", "vendor", ".idea", ".vscode":
		return true
	default:
		return false
	}
}

func uniqueSorted(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, raw := range in {
		value := strings.TrimSpace(raw)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
