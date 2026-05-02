package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	gitopsflow "github.com/confighub/cub-gen/internal/gitops"
	"github.com/confighub/cub-gen/internal/importer"
	"github.com/confighub/cub-gen/internal/model"
	normalizeflow "github.com/confighub/cub-gen/internal/normalize"
)

func runNormalize(args []string) error {
	if len(args) == 0 {
		printNormalizeUsage(os.Stderr)
		return errors.New("normalize subcommand required")
	}

	switch args[0] {
	case "help", "-h", "--help":
		printNormalizeUsage(os.Stdout)
		return nil
	case "preview":
		return runNormalizePreview(args[1:])
	default:
		printNormalizeUsage(os.Stderr)
		return fmt.Errorf("unknown normalize subcommand: %s", args[0])
	}
}

func runNormalizePreview(args []string) error {
	fs := flag.NewFlagSet("normalize preview", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	space := fs.String("space", "default", "ConfigHub space label")
	ref := fs.String("ref", "HEAD", "Git ref label to include in output")
	whereResource := fs.String("where-resource", "", "Additional resource filter expression")
	out := fs.String("out", "-", "Output file path, or '-' for stdout")
	jsonOut := fs.Bool("json", true, "Output JSON")
	patchOut := fs.Bool("patch", false, "Output a unified diff for proposed normalize sidecar files")
	pretty := fs.Bool("pretty", true, "Pretty-print JSON output")
	overrideFlags := addHelmCLIOverrideFlags(fs)
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if *patchOut {
		*jsonOut = false
	}
	if !*jsonOut && !*patchOut {
		return errors.New("cub-gen normalize preview requires --json or --patch")
	}
	helmCLIOverrides, err := overrideFlags.parse()
	if err != nil {
		return err
	}
	targetSlug, renderTargetSlug, err := resolveTargetPairArgs(fs, "usage: cub-gen normalize preview [flags] <target-path> [<render-target-path>]")
	if err != nil {
		return err
	}

	plan, err := buildNormalizePlan(targetSlug, renderTargetSlug, normalizeOptions{
		Space:            *space,
		Ref:              *ref,
		WhereResource:    *whereResource,
		HelmCLIOverrides: helmCLIOverrides,
	})
	if err != nil {
		return err
	}
	if *patchOut {
		patch, err := normalizeflow.RenderPatch(plan)
		if err != nil {
			return err
		}
		return writeTextOutput(*out, patch)
	}
	return writeJSONOutput(*out, plan, *pretty)
}

type normalizeOptions struct {
	Space            string
	Ref              string
	WhereResource    string
	HelmCLIOverrides []model.HelmCLIOverride
}

func buildNormalizePlan(targetSlug, renderTargetSlug string, opts normalizeOptions) (normalizeflow.Plan, error) {
	imported, err := gitopsflow.ImportWithOptions(targetSlug, renderTargetSlug, opts.Ref, opts.Space, opts.WhereResource, importer.ImportOptions{
		HelmCLIOverrides: opts.HelmCLIOverrides,
	})
	if err != nil {
		return normalizeflow.Plan{}, err
	}
	return normalizeflow.BuildPlan(imported)
}

func printNormalizeUsage(out io.Writer) {
	printCommandHelp(
		out,
		"cub-gen normalize: propose governed config rewrites without applying them",
		[]string{
			"Use normalize when a repo renders correctly but the review/governance shape is still too implicit.",
			"Preview is read-only. It creates a reviewable patch set for sidecar proposals, not direct app or platform mutations.",
		},
		helpSection{
			Title: "Usage",
			Lines: []string{
				"  cub-gen normalize preview [--space SPACE] [--ref REF] [--where-resource EXPR] [--set KEY=VALUE] [--set-string KEY=VALUE] [--set-file KEY=PATH] [--json|--patch] [--out FILE|-] [--pretty] <target-path> [<render-target-path>]",
			},
		},
		helpSection{
			Title: "Examples",
			Lines: []string{
				"  cub-gen normalize preview --space platform ./examples/springboot-paas",
				"  cub-gen normalize preview --patch --space platform ./examples/springboot-paas",
			},
		},
		helpSection{
			Title: "Transforms",
			Lines: []string{
				"  - add-route-policy-annotation: turn field-route metadata into ConfigHub Unit route policy annotations",
				"  - lift-generated-patch-to-source: show where rendered edits should become source PRs",
				"  - split-env-values-into-variants: expose environment/profile files as Deployment Variants",
				"  - add-missing-owners: propose owner labels from generator provenance",
				"  - replace-implicit-secret-wiring: propose explicit SecretReference records for literal secret-shaped env values",
			},
		},
	)
}
