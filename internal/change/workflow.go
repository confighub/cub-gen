package change

import (
	"archive/tar"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/confighub/cub-gen/internal/attest"
	gitopsflow "github.com/confighub/cub-gen/internal/gitops"
	"github.com/confighub/cub-gen/internal/importer"
	"github.com/confighub/cub-gen/internal/model"
	"github.com/confighub/cub-gen/internal/publish"
)

type PreviewInput struct {
	TargetSlug       string                  `json:"target_slug"`
	TargetPath       string                  `json:"target_path,omitempty"`
	RenderTargetSlug string                  `json:"render_target_slug"`
	RenderTargetPath string                  `json:"render_target_path,omitempty"`
	Space            string                  `json:"space"`
	Ref              string                  `json:"ref"`
	WhereResource    string                  `json:"where_resource,omitempty"`
	HelmCLIOverrides []model.HelmCLIOverride `json:"helm_cli_overrides,omitempty"`
}

type PreviewSummary struct {
	ChangeID          string `json:"change_id"`
	TraceID           string `json:"trace_id"`
	BundleDigest      string `json:"bundle_digest"`
	AttestationDigest string `json:"attestation_digest"`
}

type PreviewCounts struct {
	DiscoveredResources int `json:"discovered_resources"`
	DryInputs           int `json:"dry_inputs"`
	WetTargets          int `json:"wet_targets"`
	InversePatches      int `json:"inverse_patches"`
}

type PreviewVerification struct {
	BundleValid      bool   `json:"bundle_valid"`
	AttestationValid bool   `json:"attestation_valid"`
	Verifier         string `json:"verifier"`
}

type PreviewResult struct {
	Input              PreviewInput             `json:"input"`
	Change             PreviewSummary           `json:"change"`
	DiscoveredProfiles []string                 `json:"discovered_profiles"`
	Counts             PreviewCounts            `json:"counts"`
	EditRecommendation model.InverseEditPointer `json:"edit_recommendation"`
	Verification       PreviewVerification      `json:"verification"`
}

type DiffInput struct {
	TargetSlug       string                  `json:"target_slug"`
	TargetPath       string                  `json:"target_path,omitempty"`
	RenderTargetSlug string                  `json:"render_target_slug"`
	RenderTargetPath string                  `json:"render_target_path,omitempty"`
	Space            string                  `json:"space"`
	BeforeRef        string                  `json:"before_ref"`
	AfterRef         string                  `json:"after_ref"`
	WhereResource    string                  `json:"where_resource,omitempty"`
	HelmCLIOverrides []model.HelmCLIOverride `json:"helm_cli_overrides,omitempty"`
}

type DiffQuery struct {
	BeforeRef     string `json:"before_ref"`
	AfterRef      string `json:"after_ref"`
	DryPathFilter string `json:"dry_path_filter,omitempty"`
	WetPathFilter string `json:"wet_path_filter,omitempty"`
	OwnerFilter   string `json:"owner_filter,omitempty"`
	MatchCount    int    `json:"match_count"`
}

type DiffSnapshot struct {
	Change             PreviewSummary `json:"change"`
	GeneratorProfiles  []string       `json:"generator_profiles"`
	DiscoveredResource int            `json:"discovered_resources"`
}

type DiffResult struct {
	Input  DiffInput    `json:"input"`
	Query  DiffQuery    `json:"query"`
	Before DiffSnapshot `json:"before"`
	After  DiffSnapshot `json:"after"`
	Diffs  []DiffEntry  `json:"diffs"`
}

type RevisionDiffInput struct {
	TargetSlug       string                  `json:"target_slug"`
	TargetPath       string                  `json:"target_path,omitempty"`
	RenderTargetSlug string                  `json:"render_target_slug"`
	RenderTargetPath string                  `json:"render_target_path,omitempty"`
	Space            string                  `json:"space"`
	FromRef          string                  `json:"from_ref"`
	ToRef            string                  `json:"to_ref"`
	WhereResource    string                  `json:"where_resource,omitempty"`
	HelmCLIOverrides []model.HelmCLIOverride `json:"helm_cli_overrides,omitempty"`
}

type RevisionDiffQuery struct {
	FromRef       string `json:"from_ref"`
	ToRef         string `json:"to_ref"`
	DryPathFilter string `json:"dry_path_filter,omitempty"`
	WetPathFilter string `json:"wet_path_filter,omitempty"`
	OwnerFilter   string `json:"owner_filter,omitempty"`
	MatchCount    int    `json:"match_count"`
}

type RevisionDiffEntry struct {
	Cause  RevisionDiffCause `json:"cause"`
	Effect DiffEntry         `json:"effect"`
}

type RevisionDiffResult struct {
	Input   RevisionDiffInput   `json:"input"`
	Query   RevisionDiffQuery   `json:"query"`
	Before  DiffSnapshot        `json:"before"`
	After   DiffSnapshot        `json:"after"`
	Changes []RevisionDiffEntry `json:"changes"`
}

type PreviewOptions struct {
	Space            string
	Ref              string
	WhereResource    string
	Verifier         string
	HelmCLIOverrides []model.HelmCLIOverride
}

type DiffOptions struct {
	Space            string
	BeforeRef        string
	AfterRef         string
	WhereResource    string
	HelmCLIOverrides []model.HelmCLIOverride
	DryPathFilter    string
	WetPathFilter    string
	OwnerFilter      string
}

type RevisionDiffOptions struct {
	Space            string
	FromRef          string
	ToRef            string
	WhereResource    string
	HelmCLIOverrides []model.HelmCLIOverride
	DryPathFilter    string
	WetPathFilter    string
	OwnerFilter      string
}

func BuildPreviewResult(
	targetSlug, renderTargetSlug string,
	opts PreviewOptions,
) (PreviewResult, publish.ChangeBundle, gitopsflow.ImportFlowResult, error) {
	imported, err := gitopsflow.ImportWithOptions(targetSlug, renderTargetSlug, opts.Ref, opts.Space, opts.WhereResource, importer.ImportOptions{
		HelmCLIOverrides: opts.HelmCLIOverrides,
	})
	if err != nil {
		return PreviewResult{}, publish.ChangeBundle{}, gitopsflow.ImportFlowResult{}, err
	}

	bundle := publish.BuildBundle(imported)
	if err := publish.VerifyBundle(bundle); err != nil {
		return PreviewResult{}, publish.ChangeBundle{}, gitopsflow.ImportFlowResult{}, fmt.Errorf("verify generated bundle: %w", err)
	}

	attestationRecord, err := attest.Build(bundle, opts.Verifier)
	if err != nil {
		return PreviewResult{}, publish.ChangeBundle{}, gitopsflow.ImportFlowResult{}, fmt.Errorf("build attestation: %w", err)
	}
	if err := attest.VerifyRecordAgainstBundle(attestationRecord, bundle); err != nil {
		return PreviewResult{}, publish.ChangeBundle{}, gitopsflow.ImportFlowResult{}, fmt.Errorf("verify generated attestation: %w", err)
	}

	topEdit, ok := BestInverseEditPointer(imported.Provenance)
	if !ok {
		topEdit = model.InverseEditPointer{
			Owner:    "unknown",
			EditHint: "No inverse edit hint produced.",
		}
	}

	result := PreviewResult{
		Input: PreviewInput{
			TargetSlug:       imported.TargetSlug,
			TargetPath:       imported.TargetPath,
			RenderTargetSlug: imported.RenderTargetSlug,
			RenderTargetPath: imported.RenderTargetPath,
			Space:            imported.Space,
			Ref:              imported.Ref,
			WhereResource:    strings.TrimSpace(opts.WhereResource),
			HelmCLIOverrides: append([]model.HelmCLIOverride(nil), imported.HelmCLIOverrides...),
		},
		Change: PreviewSummary{
			ChangeID:          bundle.ChangeID,
			TraceID:           bundle.TraceID,
			BundleDigest:      bundle.BundleDigest,
			AttestationDigest: attestationRecord.AttestationDigest,
		},
		DiscoveredProfiles: discoveredProfiles(imported.Discovered),
		Counts: PreviewCounts{
			DiscoveredResources: len(imported.Discovered),
			DryInputs:           len(imported.DryInputs),
			WetTargets:          len(imported.WetManifestTargets),
			InversePatches:      countInversePatches(imported.InversePlans),
		},
		EditRecommendation: topEdit,
		Verification: PreviewVerification{
			BundleValid:      true,
			AttestationValid: true,
			Verifier:         opts.Verifier,
		},
	}

	return result, bundle, imported, nil
}

func BuildDiffResult(targetSlug, renderTargetSlug string, opts DiffOptions) (DiffResult, error) {
	beforeMaterialized, err := materializeTargetPairAtRef(targetSlug, renderTargetSlug, opts.BeforeRef)
	if err != nil {
		return DiffResult{}, err
	}
	defer func() {
		_ = beforeMaterialized.cleanup()
	}()

	afterMaterialized, err := materializeTargetPairAtRef(targetSlug, renderTargetSlug, opts.AfterRef)
	if err != nil {
		return DiffResult{}, err
	}
	defer func() {
		_ = afterMaterialized.cleanup()
	}()

	beforePreview, beforeBundle, beforeImported, err := BuildPreviewResult(
		beforeMaterialized.TargetPath,
		beforeMaterialized.RenderTargetPath,
		PreviewOptions{
			Space:            opts.Space,
			Ref:              opts.BeforeRef,
			WhereResource:    opts.WhereResource,
			Verifier:         "cub-gen",
			HelmCLIOverrides: opts.HelmCLIOverrides,
		},
	)
	if err != nil {
		return DiffResult{}, err
	}
	afterPreview, afterBundle, afterImported, err := BuildPreviewResult(
		afterMaterialized.TargetPath,
		afterMaterialized.RenderTargetPath,
		PreviewOptions{
			Space:            opts.Space,
			Ref:              opts.AfterRef,
			WhereResource:    opts.WhereResource,
			Verifier:         "cub-gen",
			HelmCLIOverrides: opts.HelmCLIOverrides,
		},
	)
	if err != nil {
		return DiffResult{}, err
	}
	if err := ensureHelmDiffCompatible(beforeImported); err != nil {
		return DiffResult{}, err
	}
	if err := ensureHelmDiffCompatible(afterImported); err != nil {
		return DiffResult{}, err
	}

	beforeFields, err := renderHelmFieldSnapshot(beforeImported, beforeMaterialized.TargetPath)
	if err != nil {
		return DiffResult{}, err
	}
	afterFields, err := renderHelmFieldSnapshot(afterImported, afterMaterialized.TargetPath)
	if err != nil {
		return DiffResult{}, err
	}

	diffs := CollectDiffEntries(beforeFields, afterFields, beforeImported.Provenance, afterImported.Provenance, opts.DryPathFilter, opts.WetPathFilter, opts.OwnerFilter)
	if len(diffs) == 0 {
		return DiffResult{}, fmt.Errorf("no change diff matched filters (dry_path=%q wet_path=%q owner=%q)", opts.DryPathFilter, opts.WetPathFilter, opts.OwnerFilter)
	}

	targetPathDisplay, _ := filepath.Abs(targetSlug)
	renderTargetPathDisplay, _ := filepath.Abs(renderTargetSlug)

	return DiffResult{
		Input: DiffInput{
			TargetSlug:       targetSlug,
			TargetPath:       targetPathDisplay,
			RenderTargetSlug: renderTargetSlug,
			RenderTargetPath: renderTargetPathDisplay,
			Space:            opts.Space,
			BeforeRef:        opts.BeforeRef,
			AfterRef:         opts.AfterRef,
			WhereResource:    opts.WhereResource,
			HelmCLIOverrides: append([]model.HelmCLIOverride(nil), opts.HelmCLIOverrides...),
		},
		Query: DiffQuery{
			BeforeRef:     opts.BeforeRef,
			AfterRef:      opts.AfterRef,
			DryPathFilter: opts.DryPathFilter,
			WetPathFilter: opts.WetPathFilter,
			OwnerFilter:   opts.OwnerFilter,
			MatchCount:    len(diffs),
		},
		Before: DiffSnapshot{
			Change:             PreviewSummary{ChangeID: beforeBundle.ChangeID, TraceID: beforeBundle.TraceID, BundleDigest: beforeBundle.BundleDigest, AttestationDigest: beforePreview.Change.AttestationDigest},
			GeneratorProfiles:  discoveredProfiles(beforeImported.Discovered),
			DiscoveredResource: len(beforeImported.Discovered),
		},
		After: DiffSnapshot{
			Change:             PreviewSummary{ChangeID: afterBundle.ChangeID, TraceID: afterBundle.TraceID, BundleDigest: afterBundle.BundleDigest, AttestationDigest: afterPreview.Change.AttestationDigest},
			GeneratorProfiles:  discoveredProfiles(afterImported.Discovered),
			DiscoveredResource: len(afterImported.Discovered),
		},
		Diffs: diffs,
	}, nil
}

func BuildRevisionDiffResult(targetSlug, renderTargetSlug string, opts RevisionDiffOptions) (RevisionDiffResult, error) {
	diff, err := BuildDiffResult(targetSlug, renderTargetSlug, DiffOptions{
		Space:            opts.Space,
		BeforeRef:        opts.FromRef,
		AfterRef:         opts.ToRef,
		WhereResource:    opts.WhereResource,
		HelmCLIOverrides: opts.HelmCLIOverrides,
		DryPathFilter:    opts.DryPathFilter,
		WetPathFilter:    opts.WetPathFilter,
		OwnerFilter:      opts.OwnerFilter,
	})
	if err != nil {
		return RevisionDiffResult{}, err
	}

	changes := make([]RevisionDiffEntry, 0, len(diff.Diffs))
	for _, entry := range diff.Diffs {
		changes = append(changes, RevisionDiffEntry{
			Cause:  BuildRevisionDiffCause(entry),
			Effect: entry,
		})
	}

	return RevisionDiffResult{
		Input: RevisionDiffInput{
			TargetSlug:       diff.Input.TargetSlug,
			TargetPath:       diff.Input.TargetPath,
			RenderTargetSlug: diff.Input.RenderTargetSlug,
			RenderTargetPath: diff.Input.RenderTargetPath,
			Space:            diff.Input.Space,
			FromRef:          opts.FromRef,
			ToRef:            opts.ToRef,
			WhereResource:    diff.Input.WhereResource,
			HelmCLIOverrides: append([]model.HelmCLIOverride(nil), diff.Input.HelmCLIOverrides...),
		},
		Query: RevisionDiffQuery{
			FromRef:       opts.FromRef,
			ToRef:         opts.ToRef,
			DryPathFilter: diff.Query.DryPathFilter,
			WetPathFilter: diff.Query.WetPathFilter,
			OwnerFilter:   diff.Query.OwnerFilter,
			MatchCount:    len(changes),
		},
		Before:  diff.Before,
		After:   diff.After,
		Changes: changes,
	}, nil
}

type gitRefMaterialization struct {
	TargetPath       string
	RenderTargetPath string
	cleanup          func() error
}

func ensureHelmDiffCompatible(imported gitopsflow.ImportFlowResult) error {
	if len(imported.Discovered) == 0 {
		return errors.New("change diff requires a detected generator")
	}
	if len(imported.Discovered) != 1 || imported.Discovered[0].GeneratorKind != string(model.GeneratorHelm) {
		return errors.New("change diff currently supports a single Helm generator root")
	}
	return nil
}

func materializeTargetPairAtRef(targetPath, renderTargetPath, ref string) (gitRefMaterialization, error) {
	targetAbs, err := canonicalPath(targetPath)
	if err != nil {
		return gitRefMaterialization{}, fmt.Errorf("resolve target path: %w", err)
	}
	renderAbs, err := canonicalPath(renderTargetPath)
	if err != nil {
		return gitRefMaterialization{}, fmt.Errorf("resolve render target path: %w", err)
	}
	repoRoot, err := gitRepoRoot(targetAbs)
	if err != nil {
		return gitRefMaterialization{}, err
	}
	renderRepoRoot, err := gitRepoRoot(renderAbs)
	if err != nil {
		return gitRefMaterialization{}, err
	}
	if repoRoot != renderRepoRoot {
		return gitRefMaterialization{}, errors.New("change diff requires target and render target to be in the same git repository")
	}
	targetRel, err := filepath.Rel(repoRoot, targetAbs)
	if err != nil {
		return gitRefMaterialization{}, fmt.Errorf("resolve target relative path: %w", err)
	}
	renderRel, err := filepath.Rel(repoRoot, renderAbs)
	if err != nil {
		return gitRefMaterialization{}, fmt.Errorf("resolve render target relative path: %w", err)
	}

	worktreeDir, err := os.MkdirTemp("", "cub-gen-change-diff-*")
	if err != nil {
		return gitRefMaterialization{}, fmt.Errorf("create temp ref dir: %w", err)
	}
	if err := extractGitRef(repoRoot, ref, worktreeDir); err != nil {
		_ = os.RemoveAll(worktreeDir)
		return gitRefMaterialization{}, fmt.Errorf("materialize ref %s: %w", ref, err)
	}
	cleanup := func() error {
		return os.RemoveAll(worktreeDir)
	}

	return gitRefMaterialization{
		TargetPath:       joinWorktreePath(worktreeDir, targetRel),
		RenderTargetPath: joinWorktreePath(worktreeDir, renderRel),
		cleanup:          cleanup,
	}, nil
}

func extractGitRef(repoRoot, ref, dest string) error {
	cmd := exec.Command("git", "-C", repoRoot, "archive", "--format=tar", ref)
	archiveBytes, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("git archive %s: %s", ref, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return err
	}

	reader := tar.NewReader(bytes.NewReader(archiveBytes))
	for {
		header, err := reader.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return err
		}
		target := filepath.Join(dest, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, reader); err != nil {
				_ = f.Close()
				return err
			}
			if err := f.Close(); err != nil {
				return err
			}
		}
	}
	return nil
}

func gitRepoRoot(path string) (string, error) {
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("stat path %s: %w", path, err)
	}
	out, err := runCommand("", "git", "-C", path, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("resolve git root for %s: %w", path, err)
	}
	return canonicalPath(strings.TrimSpace(out))
}

func joinWorktreePath(worktreeDir, rel string) string {
	if rel == "." || rel == "" {
		return worktreeDir
	}
	return filepath.Join(worktreeDir, rel)
}

func canonicalPath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	evaluated, err := filepath.EvalSymlinks(abs)
	if err == nil {
		return evaluated, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return abs, nil
	}
	return "", err
}

func runCommand(dir, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	if strings.TrimSpace(dir) != "" {
		cmd.Dir = dir
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

func renderHelmFieldSnapshot(imported gitopsflow.ImportFlowResult, repoPath string) (map[DiffFieldKey]any, error) {
	chartDir, valueFiles, err := helmRenderPlan(imported, repoPath)
	if err != nil {
		return nil, err
	}
	args := []string{"template", "cub-gen-diff", chartDir, "--include-crds"}
	for _, valueFile := range valueFiles {
		args = append(args, "-f", valueFile)
	}
	for _, override := range imported.HelmCLIOverrides {
		flagName := strings.TrimSpace(override.Flag)
		switch flagName {
		case "--set", "--set-string":
			args = append(args, flagName, fmt.Sprintf("%s=%s", override.Key, override.Value))
		case "--set-file":
			args = append(args, flagName, fmt.Sprintf("%s=%s", override.Key, override.FilePath))
		}
	}
	rendered, err := runCommand(chartDir, "helm", args...)
	if err != nil {
		return nil, err
	}
	return FlattenRenderedManifestFields(rendered)
}

func helmRenderPlan(imported gitopsflow.ImportFlowResult, repoPath string) (string, []string, error) {
	for _, record := range imported.Provenance {
		if strings.TrimSpace(record.ChartPath) == "" {
			continue
		}
		chartDir := filepath.Join(repoPath, filepath.Dir(filepath.FromSlash(record.ChartPath)))
		selected := record.ValuesPaths
		if record.HelmLayeredAnalysis != nil && len(record.HelmLayeredAnalysis.SelectedValueFiles) > 0 {
			selected = record.HelmLayeredAnalysis.SelectedValueFiles
		}
		valueFiles := make([]string, 0, len(selected))
		for _, rel := range selected {
			valueFiles = append(valueFiles, filepath.Join(repoPath, filepath.FromSlash(rel)))
		}
		return chartDir, valueFiles, nil
	}
	return "", nil, errors.New("change diff could not resolve a Helm chart render plan")
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
