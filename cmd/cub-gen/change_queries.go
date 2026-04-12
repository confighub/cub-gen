package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	changeflow "github.com/confighub/cub-gen/internal/change"
	"github.com/confighub/cub-gen/internal/model"
	"github.com/confighub/cub-gen/internal/publish"
)

type changeImpactQuery = changeflow.ImpactQuery
type changeProvenanceHop = changeflow.ProvenanceHop
type changeImpactEntry = changeflow.ImpactEntry
type changeImpactResult = changeflow.ImpactResult
type changeExplainQuery = changeflow.ExplainQuery
type changeExplainSuggestion = changeflow.ExplainSuggestion
type changeExplainResult = changeflow.ExplainResult

func runChangeExplain(args []string) error {
	fs := flag.NewFlagSet("change explain", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	space := fs.String("space", "default", "ConfigHub space label")
	ref := fs.String("ref", "HEAD", "Git ref label to include in output")
	whereResource := fs.String("where-resource", "", "Additional resource filter expression")
	changeID := fs.String("change-id", "", "Existing change ID to explain without creating a new lifecycle")
	bundlePath := fs.String("bundle", "", "Existing change bundle JSON file to use with --change-id")
	wetPath := fs.String("wet-path", "", "Filter explanations to a specific WET path")
	dryPath := fs.String("dry-path", "", "Filter explanations to a specific DRY path")
	owner := fs.String("owner", "", "Filter explanations to a specific owner")
	out := fs.String("out", "-", "Output file path, or '-' for stdout")
	jsonOut := fs.Bool("json", true, "Output JSON")
	pretty := fs.Bool("pretty", true, "Pretty-print JSON output")
	overrideFlags := addHelmCLIOverrideFlags(fs)
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if err := requireJSONOutput("change explain", *jsonOut); err != nil {
		return err
	}
	helmCLIOverrides, err := overrideFlags.parse()
	if err != nil {
		return err
	}

	ctx, err := loadChangeQueryContext(
		fs,
		"change explain",
		strings.TrimSpace(*changeID),
		strings.TrimSpace(*bundlePath),
		strings.TrimSpace(*space),
		strings.TrimSpace(*ref),
		strings.TrimSpace(*whereResource),
		helmCLIOverrides,
		"usage: cub-gen change explain [flags] <target-path> [<render-target-path>]",
	)
	if err != nil {
		return err
	}

	result, err := changeflow.BuildExplainResult(ctx, *wetPath, *dryPath, *owner)
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

func runChangeImpact(args []string) error {
	fs := flag.NewFlagSet("change impact", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	space := fs.String("space", "default", "ConfigHub space label")
	ref := fs.String("ref", "HEAD", "Git ref label to include in output")
	whereResource := fs.String("where-resource", "", "Additional resource filter expression")
	changeID := fs.String("change-id", "", "Existing change ID to explain without creating a new lifecycle")
	bundlePath := fs.String("bundle", "", "Existing change bundle JSON file to use with --change-id")
	dryPath := fs.String("dry-path", "", "Filter impacts to a specific DRY path")
	wetPath := fs.String("wet-path", "", "Filter impacts to a specific WET path")
	owner := fs.String("owner", "", "Filter impacts to a specific owner")
	out := fs.String("out", "-", "Output file path, or '-' for stdout")
	jsonOut := fs.Bool("json", true, "Output JSON")
	pretty := fs.Bool("pretty", true, "Pretty-print JSON output")
	overrideFlags := addHelmCLIOverrideFlags(fs)
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if err := requireJSONOutput("change impact", *jsonOut); err != nil {
		return err
	}
	helmCLIOverrides, err := overrideFlags.parse()
	if err != nil {
		return err
	}

	ctx, err := loadChangeQueryContext(
		fs,
		"change impact",
		strings.TrimSpace(*changeID),
		strings.TrimSpace(*bundlePath),
		strings.TrimSpace(*space),
		strings.TrimSpace(*ref),
		strings.TrimSpace(*whereResource),
		helmCLIOverrides,
		"usage: cub-gen change impact [flags] <target-path> [<render-target-path>]",
	)
	if err != nil {
		return err
	}

	result, err := changeflow.BuildImpactResult(ctx, *dryPath, *wetPath, *owner)
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

func loadChangeQueryContext(
	fs *flag.FlagSet,
	command string,
	changeIDFilter string,
	bundleRaw string,
	space string,
	ref string,
	whereResource string,
	helmCLIOverrides []model.HelmCLIOverride,
	targetUsage string,
) (changeflow.QueryContext, error) {
	if changeIDFilter != "" {
		if len(helmCLIOverrides) > 0 {
			return changeflow.QueryContext{}, fmt.Errorf("%s --change-id/--bundle cannot be combined with Helm CLI override flags", command)
		}
		if fs.NArg() != 0 {
			return changeflow.QueryContext{}, fmt.Errorf("usage: cub-gen %s --change-id ID --bundle FILE [flags]", command)
		}
		if bundleRaw == "" {
			return changeflow.QueryContext{}, fmt.Errorf("%s --change-id requires --bundle FILE", command)
		}
		var bundle publish.ChangeBundle
		if err := readJSONInput(bundleRaw, &bundle); err != nil {
			return changeflow.QueryContext{}, fmt.Errorf("read bundle json: %w", err)
		}
		if err := publish.VerifyBundle(bundle); err != nil {
			return changeflow.QueryContext{}, fmt.Errorf("verify bundle before %s: %w", strings.TrimPrefix(command, "change "), err)
		}
		if bundle.ChangeID == "" {
			return changeflow.QueryContext{}, errors.New("bundle does not contain change_id")
		}
		if bundle.ChangeID != changeIDFilter {
			return changeflow.QueryContext{}, fmt.Errorf("bundle change_id mismatch: expected %s, got %s", changeIDFilter, bundle.ChangeID)
		}
		return changeflow.QueryContextFromBundle(bundle), nil
	}

	if bundleRaw != "" {
		return changeflow.QueryContext{}, fmt.Errorf("%s --bundle requires --change-id", command)
	}
	targetSlug, renderTargetSlug, err := resolveTargetPairArgs(fs, targetUsage)
	if err != nil {
		return changeflow.QueryContext{}, err
	}
	preview, _, imported, err := buildChangePreviewResult(
		targetSlug,
		renderTargetSlug,
		space,
		ref,
		whereResource,
		"cub-gen",
		helmCLIOverrides,
	)
	if err != nil {
		return changeflow.QueryContext{}, err
	}
	return changeflow.NewQueryContext(preview.Input, preview.Change, imported.Provenance), nil
}
