package detect

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/confighub/cub-gen/internal/applicationset"
	"github.com/confighub/cub-gen/internal/model"
	"github.com/confighub/cub-gen/internal/registry"
	"gopkg.in/yaml.v3"
)

// ScanRepo scans a local repository path and returns deterministic generator detections.
func ScanRepo(repoPath, ref string) (model.DetectionResult, error) {
	if repoPath == "" {
		return model.DetectionResult{}, errors.New("repo path is required")
	}

	absRepo, err := filepath.Abs(repoPath)
	if err != nil {
		return model.DetectionResult{}, fmt.Errorf("resolve repo path: %w", err)
	}

	st, err := os.Stat(absRepo)
	if err != nil {
		return model.DetectionResult{}, fmt.Errorf("stat repo: %w", err)
	}
	if !st.IsDir() {
		return model.DetectionResult{}, fmt.Errorf("repo path is not a directory: %s", absRepo)
	}

	helmDetections, err := detectHelm(absRepo)
	if err != nil {
		return model.DetectionResult{}, err
	}
	scoreDetections, err := detectScore(absRepo)
	if err != nil {
		return model.DetectionResult{}, err
	}
	springDetections, err := detectSpringBoot(absRepo)
	if err != nil {
		return model.DetectionResult{}, err
	}
	backstageDetections, err := detectBackstage(absRepo)
	if err != nil {
		return model.DetectionResult{}, err
	}
	noConfigPlatformDetections, err := detectNoConfigPlatform(absRepo)
	if err != nil {
		return model.DetectionResult{}, err
	}
	opsDetections, err := detectOpsWorkflow(absRepo)
	if err != nil {
		return model.DetectionResult{}, err
	}
	c3agentDetections, err := detectC3Agent(absRepo)
	if err != nil {
		return model.DetectionResult{}, err
	}
	swampDetections, err := detectSwamp(absRepo)
	if err != nil {
		return model.DetectionResult{}, err
	}

	all := append(helmDetections, scoreDetections...)
	all = append(all, springDetections...)
	all = append(all, backstageDetections...)
	all = append(all, noConfigPlatformDetections...)
	all = append(all, opsDetections...)
	all = append(all, c3agentDetections...)
	all = append(all, swampDetections...)
	if len(all) == 0 {
		applicationSetDetections, err := detectApplicationSet(absRepo)
		if err != nil {
			return model.DetectionResult{}, err
		}
		all = append(all, applicationSetDetections...)
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].Kind != all[j].Kind {
			return all[i].Kind < all[j].Kind
		}
		if all[i].Root != all[j].Root {
			return all[i].Root < all[j].Root
		}
		return all[i].Name < all[j].Name
	})

	chains, err := loadGeneratorChains(absRepo, all)
	if err != nil {
		return model.DetectionResult{}, err
	}

	return model.DetectionResult{
		Repo:       absRepo,
		Ref:        ref,
		DetectedAt: time.Now().UTC().Format(time.RFC3339),
		Generators: all,
		Chains:     chains,
	}, nil
}

type rawGeneratorChainFile struct {
	Chains []rawGeneratorChain `yaml:"chains"`
}

type rawGeneratorChain struct {
	ID       string                     `yaml:"id"`
	Name     string                     `yaml:"name"`
	Stages   []rawGeneratorChainStage   `yaml:"stages"`
	Mappings []rawGeneratorChainMapping `yaml:"mappings"`
}

type rawGeneratorChainStage struct {
	Kind string `yaml:"kind"`
	Root string `yaml:"root"`
}

type rawGeneratorChainMapping struct {
	DownstreamWetPath  string  `yaml:"downstream_wet_path"`
	DownstreamDryPath  string  `yaml:"downstream_dry_path"`
	UpstreamKind       string  `yaml:"upstream_kind"`
	UpstreamRoot       string  `yaml:"upstream_root"`
	UpstreamDryPath    string  `yaml:"upstream_dry_path"`
	UpstreamSourcePath string  `yaml:"upstream_source_path"`
	UpstreamOwner      string  `yaml:"upstream_owner"`
	UpstreamTransform  string  `yaml:"upstream_transform"`
	UpstreamConfidence float64 `yaml:"upstream_confidence"`
}

func loadGeneratorChains(repo string, detections []model.GeneratorDetection) ([]model.GeneratorChain, error) {
	path := filepath.Join(repo, ".cub-gen-chain.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read generator chains: %w", err)
	}

	var raw rawGeneratorChainFile
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse generator chains: %w", err)
	}
	if len(raw.Chains) == 0 {
		return nil, nil
	}

	out := make([]model.GeneratorChain, 0, len(raw.Chains))
	for idx, chain := range raw.Chains {
		if len(chain.Stages) < 2 {
			return nil, fmt.Errorf("generator chain %q must declare at least two stages", rawChainID(chain, idx))
		}

		stages := make([]model.GeneratorChainStage, 0, len(chain.Stages))
		for stageIdx, stage := range chain.Stages {
			kind, err := parseGeneratorKind(stage.Kind)
			if err != nil {
				return nil, fmt.Errorf("generator chain %q stage %d: %w", rawChainID(chain, idx), stageIdx+1, err)
			}
			root := normalizeChainRoot(stage.Root)
			detection, ok := matchGeneratorDetection(detections, kind, root)
			if !ok {
				return nil, fmt.Errorf("generator chain %q stage %d: no detected generator matched kind=%q root=%q", rawChainID(chain, idx), stageIdx+1, kind, root)
			}
			stages = append(stages, model.GeneratorChainStage{
				Kind:        detection.Kind,
				Profile:     detection.Profile,
				Name:        detection.Name,
				Root:        detection.Root,
				DetectionID: detection.ID,
				Confidence:  detection.Confidence,
			})
		}

		mappings := make([]model.GeneratorChainMapping, 0, len(chain.Mappings))
		for mappingIdx, mapping := range chain.Mappings {
			if strings.TrimSpace(mapping.DownstreamWetPath) == "" {
				return nil, fmt.Errorf("generator chain %q mapping %d must declare downstream_wet_path", rawChainID(chain, idx), mappingIdx+1)
			}
			if strings.TrimSpace(mapping.UpstreamDryPath) == "" {
				return nil, fmt.Errorf("generator chain %q mapping %d must declare upstream_dry_path", rawChainID(chain, idx), mappingIdx+1)
			}
			if strings.TrimSpace(mapping.UpstreamSourcePath) == "" {
				return nil, fmt.Errorf("generator chain %q mapping %d must declare upstream_source_path", rawChainID(chain, idx), mappingIdx+1)
			}
			if mapping.UpstreamConfidence < 0 || mapping.UpstreamConfidence > 1 {
				return nil, fmt.Errorf("generator chain %q mapping %d upstream_confidence must be between 0 and 1", rawChainID(chain, idx), mappingIdx+1)
			}

			upstreamStage := stages[0]
			if strings.TrimSpace(mapping.UpstreamKind) != "" || strings.TrimSpace(mapping.UpstreamRoot) != "" {
				kind, err := parseGeneratorKind(mapping.UpstreamKind)
				if err != nil {
					return nil, fmt.Errorf("generator chain %q mapping %d: %w", rawChainID(chain, idx), mappingIdx+1, err)
				}
				root := normalizeChainRoot(mapping.UpstreamRoot)
				stage, ok := matchChainStage(stages, kind, root)
				if !ok {
					return nil, fmt.Errorf("generator chain %q mapping %d: no upstream stage matched kind=%q root=%q", rawChainID(chain, idx), mappingIdx+1, kind, root)
				}
				upstreamStage = stage
			}

			confidence := mapping.UpstreamConfidence
			if confidence == 0 {
				confidence = upstreamStage.Confidence
			}
			mappings = append(mappings, model.GeneratorChainMapping{
				DownstreamWetPath:  strings.TrimSpace(mapping.DownstreamWetPath),
				DownstreamDryPath:  strings.TrimSpace(mapping.DownstreamDryPath),
				UpstreamKind:       upstreamStage.Kind,
				UpstreamProfile:    upstreamStage.Profile,
				UpstreamRoot:       upstreamStage.Root,
				UpstreamDryPath:    strings.TrimSpace(mapping.UpstreamDryPath),
				UpstreamSourcePath: filepath.ToSlash(strings.TrimSpace(mapping.UpstreamSourcePath)),
				UpstreamOwner:      strings.TrimSpace(mapping.UpstreamOwner),
				UpstreamTransform:  strings.TrimSpace(mapping.UpstreamTransform),
				UpstreamConfidence: confidence,
			})
		}

		out = append(out, model.GeneratorChain{
			ID:       rawChainID(chain, idx),
			Name:     strings.TrimSpace(chain.Name),
			Stages:   stages,
			Mappings: mappings,
		})
	}

	return out, nil
}

func rawChainID(chain rawGeneratorChain, idx int) string {
	if strings.TrimSpace(chain.ID) != "" {
		return strings.TrimSpace(chain.ID)
	}
	if strings.TrimSpace(chain.Name) != "" {
		return strings.TrimSpace(chain.Name)
	}
	return fmt.Sprintf("chain-%d", idx+1)
}

func parseGeneratorKind(raw string) (model.GeneratorKind, error) {
	kind := model.GeneratorKind(strings.TrimSpace(raw))
	for _, supported := range registry.Kinds() {
		if supported == kind {
			return kind, nil
		}
	}
	return "", fmt.Errorf("unknown generator kind %q", raw)
}

func normalizeChainRoot(root string) string {
	root = filepath.ToSlash(strings.TrimSpace(root))
	if root == "." {
		return ""
	}
	return root
}

func matchGeneratorDetection(detections []model.GeneratorDetection, kind model.GeneratorKind, root string) (model.GeneratorDetection, bool) {
	for _, detection := range detections {
		if detection.Kind == kind && normalizeChainRoot(detection.Root) == root {
			return detection, true
		}
	}
	return model.GeneratorDetection{}, false
}

func matchChainStage(stages []model.GeneratorChainStage, kind model.GeneratorKind, root string) (model.GeneratorChainStage, bool) {
	for _, stage := range stages {
		if stage.Kind == kind && normalizeChainRoot(stage.Root) == root {
			return stage, true
		}
	}
	return model.GeneratorChainStage{}, false
}

func detectHelm(repo string) ([]model.GeneratorDetection, error) {
	detected := make(map[string]model.GeneratorDetection)
	err := filepath.WalkDir(repo, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() && shouldSkipDir(d.Name()) {
			return filepath.SkipDir
		}
		if d.IsDir() || d.Name() != "Chart.yaml" {
			return nil
		}

		root := filepath.Dir(path)
		relRoot, err := filepath.Rel(repo, root)
		if err != nil {
			return err
		}
		if relRoot == "." {
			relRoot = ""
		}
		if isNestedHelmDependencyChart(relRoot) {
			return nil
		}
		inputs := []string{filepath.ToSlash(relJoin(relRoot, "Chart.yaml"))}
		matches, _ := filepath.Glob(filepath.Join(root, "values*.yaml"))
		for _, match := range matches {
			rel, relErr := filepath.Rel(repo, match)
			if relErr != nil {
				continue
			}
			inputs = append(inputs, filepath.ToSlash(rel))
		}
		inputs = append(inputs, existingHelmAuxiliaryInputs(repo, root)...)
		sort.Strings(inputs)

		name := filepath.Base(root)
		id := shortID("helm:" + filepath.ToSlash(relRoot))
		key := "helm:" + relRoot
		detected[key] = model.GeneratorDetection{
			ID:         "gen_" + id,
			Kind:       model.GeneratorHelm,
			Profile:    profileForKind(model.GeneratorHelm),
			Name:       name,
			Root:       filepath.ToSlash(relRoot),
			Inputs:     inputs,
			Confidence: 0.98,
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("detect helm: %w", err)
	}
	return mapValuesSorted(detected), nil
}

func existingHelmAuxiliaryInputs(repo, root string) []string {
	candidates := make([]string, 0, 24)
	globs := []string{
		filepath.Join(root, "values.schema.json"),
		filepath.Join(root, "gitops", "argo", "applicationset.yaml"),
		filepath.Join(root, "gitops", "argo", "applicationset.yml"),
		filepath.Join(root, "platform", "clusters", "*.yaml"),
		filepath.Join(root, "platform", "clusters", "*.yml"),
		filepath.Join(root, "platform", "clusters", "*.json"),
		filepath.Join(root, "platform", "catalogs", "managed-service-catalog", "*.yaml"),
		filepath.Join(root, "platform", "catalogs", "managed-service-catalog", "*.yml"),
		filepath.Join(root, "platform", "catalogs", "managed-service-catalog", "*.json"),
		filepath.Join(root, "platform", "catalogs", "customer-service-catalog", "*.yaml"),
		filepath.Join(root, "platform", "catalogs", "customer-service-catalog", "*.yml"),
		filepath.Join(root, "platform", "catalogs", "customer-service-catalog", "*.json"),
	}
	for _, pattern := range globs {
		matches, _ := filepath.Glob(pattern)
		for _, match := range matches {
			rel, err := filepath.Rel(repo, match)
			if err != nil {
				continue
			}
			candidates = append(candidates, filepath.ToSlash(rel))
		}
	}
	_ = filepath.WalkDir(filepath.Join(root, "crds"), func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(d.Name()))
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}
		rel, err := filepath.Rel(repo, path)
		if err != nil {
			return nil
		}
		candidates = append(candidates, filepath.ToSlash(rel))
		return nil
	})
	_ = filepath.WalkDir(filepath.Join(root, "charts"), func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		name := strings.ToLower(d.Name())
		if name != "chart.yaml" && !strings.HasPrefix(name, "values") {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(name))
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}
		rel, err := filepath.Rel(repo, path)
		if err != nil {
			return nil
		}
		candidates = append(candidates, filepath.ToSlash(rel))
		return nil
	})
	return uniqueSortedStrings(candidates)
}

func isNestedHelmDependencyChart(relRoot string) bool {
	normalized := filepath.ToSlash(strings.TrimSpace(relRoot))
	return normalized == "charts" || strings.HasPrefix(normalized, "charts/")
}

func detectApplicationSet(repo string) ([]model.GeneratorDetection, error) {
	detected := make(map[string]model.GeneratorDetection)
	err := filepath.WalkDir(repo, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() && shouldSkipDir(d.Name()) {
			return filepath.SkipDir
		}
		if d.IsDir() {
			return nil
		}
		name := strings.ToLower(d.Name())
		if name != "applicationset.yaml" && name != "applicationset.yml" {
			return nil
		}

		relPath, err := filepath.Rel(repo, path)
		if err != nil {
			return err
		}
		relPath = filepath.ToSlash(relPath)
		if !applicationset.IsApplicationSetFile(repo, relPath) {
			return nil
		}

		doc, err := applicationset.ParseFile(repo, relPath)
		if err != nil {
			return nil
		}
		root := filepath.ToSlash(filepath.Dir(relPath))
		if root == "." {
			root = ""
		}

		inputs := []string{doc.Path}
		inputs = append(inputs, existingApplicationSetAuxiliaryInputs(repo)...)
		inputs = uniqueSortedStrings(inputs)

		id := shortID("applicationset:" + relPath)
		key := "applicationset:" + relPath
		detected[key] = model.GeneratorDetection{
			ID:         "gen_" + id,
			Kind:       model.GeneratorApplicationSet,
			Profile:    profileForKind(model.GeneratorApplicationSet),
			Name:       doc.Name,
			Root:       root,
			Inputs:     inputs,
			Confidence: 0.95,
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("detect applicationset: %w", err)
	}
	return mapValuesSorted(detected), nil
}

func existingApplicationSetAuxiliaryInputs(repo string) []string {
	candidates := make([]string, 0, 8)
	globs := []string{
		filepath.Join(repo, "clusters", "*.yaml"),
		filepath.Join(repo, "clusters", "*.yml"),
		filepath.Join(repo, "clusters", "*.json"),
		filepath.Join(repo, "inventory", "clusters", "*.yaml"),
		filepath.Join(repo, "inventory", "clusters", "*.yml"),
		filepath.Join(repo, "inventory", "clusters", "*.json"),
		filepath.Join(repo, "platform", "clusters", "*.yaml"),
		filepath.Join(repo, "platform", "clusters", "*.yml"),
		filepath.Join(repo, "platform", "clusters", "*.json"),
	}
	for _, pattern := range globs {
		matches, _ := filepath.Glob(pattern)
		for _, match := range matches {
			rel, err := filepath.Rel(repo, match)
			if err != nil {
				continue
			}
			candidates = append(candidates, filepath.ToSlash(rel))
		}
	}
	return uniqueSortedStrings(candidates)
}

func detectScore(repo string) ([]model.GeneratorDetection, error) {
	detected := make(map[string]model.GeneratorDetection)
	err := filepath.WalkDir(repo, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() && shouldSkipDir(d.Name()) {
			return filepath.SkipDir
		}
		if d.IsDir() {
			return nil
		}
		if d.Name() != "score.yaml" && d.Name() != "score.yml" {
			return nil
		}

		b, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		content := strings.ToLower(string(b))
		if !strings.Contains(content, "score.dev/") && !strings.Contains(content, "kind: workload") {
			return nil
		}

		root := filepath.Dir(path)
		relRoot, err := filepath.Rel(repo, root)
		if err != nil {
			return err
		}
		if relRoot == "." {
			relRoot = ""
		}
		relFile, err := filepath.Rel(repo, path)
		if err != nil {
			return err
		}
		inputs := []string{filepath.ToSlash(relFile)}
		name := filepath.Base(root)
		id := shortID("score:" + filepath.ToSlash(relRoot))
		key := "score:" + relRoot
		detected[key] = model.GeneratorDetection{
			ID:         "gen_" + id,
			Kind:       model.GeneratorScore,
			Profile:    profileForKind(model.GeneratorScore),
			Name:       name,
			Root:       filepath.ToSlash(relRoot),
			Inputs:     inputs,
			Confidence: 0.96,
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("detect score: %w", err)
	}
	return mapValuesSorted(detected), nil
}

func detectSpringBoot(repo string) ([]model.GeneratorDetection, error) {
	buildRoots := map[string]string{}

	err := filepath.WalkDir(repo, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() && shouldSkipDir(d.Name()) {
			return filepath.SkipDir
		}
		if d.IsDir() {
			return nil
		}

		switch d.Name() {
		case "pom.xml", "build.gradle", "build.gradle.kts":
			root := filepath.Dir(path)
			buildRoots[root] = d.Name()
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan spring build files: %w", err)
	}

	detected := make([]model.GeneratorDetection, 0, len(buildRoots))
	for root, buildFile := range buildRoots {
		appYAML := filepath.Join(root, "src", "main", "resources", "application.yaml")
		appYML := filepath.Join(root, "src", "main", "resources", "application.yml")
		cfg := ""
		if _, err := os.Stat(appYAML); err == nil {
			cfg = appYAML
		} else if _, err := os.Stat(appYML); err == nil {
			cfg = appYML
		}
		if cfg == "" {
			continue
		}

		relRoot, err := filepath.Rel(repo, root)
		if err != nil {
			return nil, err
		}
		if relRoot == "." {
			relRoot = ""
		}
		inputs := make([]string, 0, 4)
		inputs = append(inputs, filepath.ToSlash(relJoin(relRoot, buildFile)))
		relCfg, err := filepath.Rel(repo, cfg)
		if err == nil {
			inputs = append(inputs, filepath.ToSlash(relCfg))
		}
		cfgDir := filepath.Dir(cfg)
		profiles, _ := filepath.Glob(filepath.Join(cfgDir, "application-*.yaml"))
		profilesYml, _ := filepath.Glob(filepath.Join(cfgDir, "application-*.yml"))
		profiles = append(profiles, profilesYml...)
		for _, profile := range profiles {
			rel, relErr := filepath.Rel(repo, profile)
			if relErr != nil {
				continue
			}
			inputs = append(inputs, filepath.ToSlash(rel))
		}
		sort.Strings(inputs)

		name := filepath.Base(root)
		detected = append(detected, model.GeneratorDetection{
			ID:         "gen_" + shortID("springboot:"+filepath.ToSlash(relRoot)),
			Kind:       model.GeneratorSpringBoot,
			Profile:    profileForKind(model.GeneratorSpringBoot),
			Name:       name,
			Root:       filepath.ToSlash(relRoot),
			Inputs:     unique(inputs),
			Confidence: 0.93,
		})
	}

	sort.Slice(detected, func(i, j int) bool {
		if detected[i].Root == detected[j].Root {
			return detected[i].Name < detected[j].Name
		}
		return detected[i].Root < detected[j].Root
	})
	return detected, nil
}

func detectBackstage(repo string) ([]model.GeneratorDetection, error) {
	detected := make(map[string]model.GeneratorDetection)
	err := filepath.WalkDir(repo, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() && shouldSkipDir(d.Name()) {
			return filepath.SkipDir
		}
		if d.IsDir() {
			return nil
		}
		if d.Name() != "catalog-info.yaml" && d.Name() != "catalog-info.yml" {
			return nil
		}

		b, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		content := strings.ToLower(string(b))
		if !strings.Contains(content, "backstage.io/") || !strings.Contains(content, "kind: component") {
			return nil
		}

		root := filepath.Dir(path)
		relRoot, err := filepath.Rel(repo, root)
		if err != nil {
			return err
		}
		if relRoot == "." {
			relRoot = ""
		}

		relCatalog, err := filepath.Rel(repo, path)
		if err != nil {
			return err
		}
		inputs := []string{filepath.ToSlash(relCatalog)}

		for _, candidate := range []string{"app-config.yaml", "app-config.yml"} {
			candidatePath := filepath.Join(root, candidate)
			if _, statErr := os.Stat(candidatePath); statErr == nil {
				rel, relErr := filepath.Rel(repo, candidatePath)
				if relErr == nil {
					inputs = append(inputs, filepath.ToSlash(rel))
				}
			}
		}
		sort.Strings(inputs)

		name := filepath.Base(root)
		id := shortID("backstage:" + filepath.ToSlash(relRoot))
		key := "backstage:" + relRoot
		detected[key] = model.GeneratorDetection{
			ID:         "gen_" + id,
			Kind:       model.GeneratorBackstage,
			Profile:    profileForKind(model.GeneratorBackstage),
			Name:       name,
			Root:       filepath.ToSlash(relRoot),
			Inputs:     inputs,
			Confidence: 0.91,
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("detect backstage: %w", err)
	}
	return mapValuesSorted(detected), nil
}

func detectNoConfigPlatform(repo string) ([]model.GeneratorDetection, error) {
	detected := make(map[string]model.GeneratorDetection)
	err := filepath.WalkDir(repo, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() && shouldSkipDir(d.Name()) {
			return filepath.SkipDir
		}
		if d.IsDir() {
			return nil
		}
		name := strings.ToLower(d.Name())
		if name != "no-config-platform.yaml" && name != "no-config-platform.yml" && name != "no-config-platform.json" {
			return nil
		}

		b, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		content := strings.ToLower(string(b))
		if !strings.Contains(content, "service: no-config-platform") {
			return nil
		}

		root := filepath.Dir(path)
		relRoot, err := filepath.Rel(repo, root)
		if err != nil {
			return err
		}
		if relRoot == "." {
			relRoot = ""
		}

		inputs := []string{}
		patterns := []string{
			"no-config-platform*.yaml",
			"no-config-platform*.yml",
			"no-config-platform*.json",
		}
		for _, pattern := range patterns {
			matches, _ := filepath.Glob(filepath.Join(root, pattern))
			for _, match := range matches {
				rel, relErr := filepath.Rel(repo, match)
				if relErr != nil {
					continue
				}
				inputs = append(inputs, filepath.ToSlash(rel))
			}
		}
		inputs = unique(inputs)
		sort.Strings(inputs)
		if len(inputs) == 0 {
			relFile, relErr := filepath.Rel(repo, path)
			if relErr == nil {
				inputs = append(inputs, filepath.ToSlash(relFile))
			}
		}

		detectKey := string(model.GeneratorNoConfigPlatform) + ":" + relRoot
		detected[detectKey] = model.GeneratorDetection{
			ID:         "gen_" + shortID(detectKey),
			Kind:       model.GeneratorNoConfigPlatform,
			Profile:    profileForKind(model.GeneratorNoConfigPlatform),
			Name:       filepath.Base(root),
			Root:       filepath.ToSlash(relRoot),
			Inputs:     inputs,
			Confidence: 0.90,
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("detect no-config-platform: %w", err)
	}
	return mapValuesSorted(detected), nil
}

func detectOpsWorkflow(repo string) ([]model.GeneratorDetection, error) {
	detected := make(map[string]model.GeneratorDetection)
	err := filepath.WalkDir(repo, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() && shouldSkipDir(d.Name()) {
			return filepath.SkipDir
		}
		if d.IsDir() {
			return nil
		}

		name := strings.ToLower(d.Name())
		if name != "operations.yaml" && name != "operations.yml" && name != "workflow.yaml" && name != "workflow.yml" {
			return nil
		}

		b, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		content := strings.ToLower(string(b))
		if !strings.Contains(content, "actions:") && !strings.Contains(content, "workflow:") {
			return nil
		}

		root := filepath.Dir(path)
		relRoot, err := filepath.Rel(repo, root)
		if err != nil {
			return err
		}
		if relRoot == "." {
			relRoot = ""
		}

		inputs := []string{}
		patterns := []string{"operations*.yaml", "operations*.yml", "workflow*.yaml", "workflow*.yml", "actions*.yaml", "actions*.yml"}
		for _, pattern := range patterns {
			matches, _ := filepath.Glob(filepath.Join(root, pattern))
			for _, match := range matches {
				rel, relErr := filepath.Rel(repo, match)
				if relErr != nil {
					continue
				}
				inputs = append(inputs, filepath.ToSlash(rel))
			}
		}
		inputs = unique(inputs)
		sort.Strings(inputs)
		if len(inputs) == 0 {
			relFile, relErr := filepath.Rel(repo, path)
			if relErr == nil {
				inputs = append(inputs, filepath.ToSlash(relFile))
			}
		}

		detected["opsworkflow:"+relRoot] = model.GeneratorDetection{
			ID:         "gen_" + shortID("opsworkflow:"+filepath.ToSlash(relRoot)),
			Kind:       model.GeneratorOpsFlow,
			Profile:    profileForKind(model.GeneratorOpsFlow),
			Name:       filepath.Base(root),
			Root:       filepath.ToSlash(relRoot),
			Inputs:     inputs,
			Confidence: 0.89,
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("detect ops workflow: %w", err)
	}
	return mapValuesSorted(detected), nil
}

func detectC3Agent(repo string) ([]model.GeneratorDetection, error) {
	detected := make(map[string]model.GeneratorDetection)
	err := filepath.WalkDir(repo, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() && shouldSkipDir(d.Name()) {
			return filepath.SkipDir
		}
		if d.IsDir() {
			return nil
		}
		name := strings.ToLower(d.Name())
		if name != "c3agent.yaml" && name != "c3agent.yml" && name != "c3agent.json" {
			return nil
		}

		b, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		content := string(b)
		if !hasC3AgentServiceSignature(content) {
			return nil
		}
		confidence := 0.90
		if hasC3AgentFleetSignal(content) {
			confidence = 0.92
		}

		root := filepath.Dir(path)
		relRoot, err := filepath.Rel(repo, root)
		if err != nil {
			return err
		}
		if relRoot == "." {
			relRoot = ""
		}

		inputs := []string{}
		patterns := []string{"c3agent*.yaml", "c3agent*.yml", "c3agent*.json"}
		for _, pattern := range patterns {
			matches, _ := filepath.Glob(filepath.Join(root, pattern))
			for _, match := range matches {
				rel, relErr := filepath.Rel(repo, match)
				if relErr != nil {
					continue
				}
				inputs = append(inputs, filepath.ToSlash(rel))
			}
		}
		inputs = unique(inputs)
		sort.Strings(inputs)
		if len(inputs) == 0 {
			relFile, relErr := filepath.Rel(repo, path)
			if relErr == nil {
				inputs = append(inputs, filepath.ToSlash(relFile))
			}
		}

		detected["c3agent:"+relRoot] = model.GeneratorDetection{
			ID:         "gen_" + shortID("c3agent:"+filepath.ToSlash(relRoot)),
			Kind:       model.GeneratorC3Agent,
			Profile:    profileForKind(model.GeneratorC3Agent),
			Name:       filepath.Base(root),
			Root:       filepath.ToSlash(relRoot),
			Inputs:     inputs,
			Confidence: confidence,
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("detect c3agent: %w", err)
	}
	return mapValuesSorted(detected), nil
}

func hasC3AgentServiceSignature(content string) bool {
	if jsonObject := topLevelJSONMap(content); jsonObject != nil {
		if service, ok := jsonObject["service"].(string); ok && strings.EqualFold(strings.TrimSpace(service), "c3agent") {
			return true
		}
	}
	service, ok := topLevelYAMLValue(content, "service")
	return ok && strings.EqualFold(service, "c3agent")
}

func hasC3AgentFleetSignal(content string) bool {
	if jsonObject := topLevelJSONMap(content); jsonObject != nil {
		if _, ok := jsonObject["fleet"]; ok {
			return true
		}
	}
	_, ok := topLevelYAMLValue(content, "fleet")
	return ok
}

func topLevelJSONMap(content string) map[string]any {
	var obj map[string]any
	if err := json.Unmarshal([]byte(content), &obj); err != nil {
		return nil
	}
	return obj
}

func topLevelYAMLValue(content, key string) (string, bool) {
	for _, rawLine := range strings.Split(content, "\n") {
		line := strings.TrimRight(rawLine, "\r")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || trimmed == "---" || trimmed == "..." {
			continue
		}
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			continue
		}

		sep := strings.Index(line, ":")
		if sep <= 0 {
			continue
		}
		candidate := strings.TrimSpace(line[:sep])
		candidate = strings.Trim(candidate, `"'`)
		if !strings.EqualFold(candidate, key) {
			continue
		}

		value := strings.TrimSpace(stripYAMLInlineComment(line[sep+1:]))
		value = strings.Trim(value, `"'`)
		return value, true
	}
	return "", false
}

func stripYAMLInlineComment(value string) string {
	inSingle := false
	inDouble := false
	for i, r := range value {
		switch r {
		case '\'':
			if !inDouble {
				inSingle = !inSingle
			}
		case '"':
			if !inSingle {
				inDouble = !inDouble
			}
		case '#':
			if inSingle || inDouble {
				continue
			}
			if i == 0 {
				return ""
			}
			prev := rune(value[i-1])
			if prev == ' ' || prev == '\t' {
				return value[:i]
			}
		}
	}
	return value
}

func detectSwamp(repo string) ([]model.GeneratorDetection, error) {
	detected := make(map[string]model.GeneratorDetection)
	err := filepath.WalkDir(repo, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() && shouldSkipDir(d.Name()) {
			return filepath.SkipDir
		}
		if d.IsDir() {
			return nil
		}
		name := strings.ToLower(d.Name())
		if name != ".swamp.yaml" && name != ".swamp.yml" {
			return nil
		}

		b, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		content := strings.ToLower(string(b))
		if !strings.Contains(content, "swamp") {
			return nil
		}

		root := filepath.Dir(path)
		relRoot, err := filepath.Rel(repo, root)
		if err != nil {
			return err
		}
		if relRoot == "." {
			relRoot = ""
		}

		inputs := []string{}
		relFile, relErr := filepath.Rel(repo, path)
		if relErr == nil {
			inputs = append(inputs, filepath.ToSlash(relFile))
		}
		// Collect workflow files from sibling and child directories.
		err = filepath.WalkDir(root, func(candidate string, entry fs.DirEntry, candidateErr error) error {
			if candidateErr != nil {
				return candidateErr
			}
			if entry.IsDir() {
				if candidate != root && shouldSkipDir(entry.Name()) {
					return filepath.SkipDir
				}
				return nil
			}
			base := strings.ToLower(entry.Name())
			if !strings.HasPrefix(base, "workflow-") || !(strings.HasSuffix(base, ".yaml") || strings.HasSuffix(base, ".yml")) {
				return nil
			}
			rel, rErr := filepath.Rel(repo, candidate)
			if rErr != nil {
				return nil
			}
			inputs = append(inputs, filepath.ToSlash(rel))
			return nil
		})
		if err != nil {
			return err
		}
		inputs = unique(inputs)
		sort.Strings(inputs)

		detected["swamp:"+relRoot] = model.GeneratorDetection{
			ID:         "gen_" + shortID("swamp:"+filepath.ToSlash(relRoot)),
			Kind:       model.GeneratorSwamp,
			Profile:    profileForKind(model.GeneratorSwamp),
			Name:       filepath.Base(root),
			Root:       filepath.ToSlash(relRoot),
			Inputs:     inputs,
			Confidence: 0.89,
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("detect swamp: %w", err)
	}
	return mapValuesSorted(detected), nil
}

func mapValuesSorted(m map[string]model.GeneratorDetection) []model.GeneratorDetection {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]model.GeneratorDetection, 0, len(keys))
	for _, k := range keys {
		out = append(out, m[k])
	}
	return out
}

func shouldSkipDir(name string) bool {
	switch name {
	// TODO: lift-upstream should be handled with per-directory .cubignore
	// rather than a global skip. This is a temporary workaround for the
	// springboot-paas example structure. See PR #228 discussion.
	case ".git", "node_modules", "vendor", ".idea", ".vscode", "lift-upstream":
		return true
	default:
		return false
	}
}

func shortID(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])[:16]
}

func relJoin(root, base string) string {
	if root == "" {
		return base
	}
	return filepath.Join(root, base)
}

func unique(v []string) []string {
	seen := make(map[string]struct{}, len(v))
	out := make([]string, 0, len(v))
	for _, item := range v {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

func profileForKind(kind model.GeneratorKind) string {
	return registry.Profile(kind)
}

func uniqueSortedStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
