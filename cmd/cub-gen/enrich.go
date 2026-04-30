package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	enrichflow "github.com/confighub/cub-gen/internal/enrich"
	gitopsflow "github.com/confighub/cub-gen/internal/gitops"
	"github.com/confighub/cub-gen/internal/importer"
	"github.com/confighub/cub-gen/internal/model"
)

func runEnrich(args []string) error {
	if len(args) == 0 {
		printEnrichUsage(os.Stderr)
		return errors.New("enrich subcommand required")
	}

	switch args[0] {
	case "help", "-h", "--help":
		printEnrichUsage(os.Stdout)
		return nil
	case "preview":
		return runEnrichPreview(args[1:])
	case "write":
		return runEnrichWrite(args[1:])
	default:
		printEnrichUsage(os.Stderr)
		return fmt.Errorf("unknown enrich subcommand: %s", args[0])
	}
}

func runEnrichPreview(args []string) error {
	fs := flag.NewFlagSet("enrich preview", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	space := fs.String("space", "default", "ConfigHub space label")
	ref := fs.String("ref", "HEAD", "Git ref label to include in output")
	whereResource := fs.String("where-resource", "", "Additional resource filter expression")
	out := fs.String("out", "-", "Output file path, or '-' for stdout")
	jsonOut := fs.Bool("json", true, "Output JSON")
	patchOut := fs.Bool("patch", false, "Output a unified diff for proposed sidecar files")
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
		return errors.New("cub-gen enrich preview requires --json or --patch")
	}
	helmCLIOverrides, err := overrideFlags.parse()
	if err != nil {
		return err
	}
	targetSlug, renderTargetSlug, err := resolveTargetPairArgs(fs, "usage: cub-gen enrich preview [flags] <target-path> [<render-target-path>]")
	if err != nil {
		return err
	}

	plan, err := buildEnrichmentPlan(targetSlug, renderTargetSlug, enrichOptions{
		Space:            *space,
		Ref:              *ref,
		WhereResource:    *whereResource,
		HelmCLIOverrides: helmCLIOverrides,
	})
	if err != nil {
		return err
	}

	if *patchOut {
		patch, err := enrichflow.RenderPatch(plan)
		if err != nil {
			return err
		}
		return writeTextOutput(*out, patch)
	}
	return writeJSONOutput(*out, plan, *pretty)
}

func runEnrichWrite(args []string) error {
	fs := flag.NewFlagSet("enrich write", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	space := fs.String("space", "default", "ConfigHub space label")
	ref := fs.String("ref", "HEAD", "Git ref label to include in output")
	whereResource := fs.String("where-resource", "", "Additional resource filter expression")
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
	if err := requireJSONOutput("enrich write", *jsonOut); err != nil {
		return err
	}
	helmCLIOverrides, err := overrideFlags.parse()
	if err != nil {
		return err
	}
	targetSlug, renderTargetSlug, err := resolveTargetPairArgs(fs, "usage: cub-gen enrich write [flags] <target-path> [<render-target-path>]")
	if err != nil {
		return err
	}

	plan, err := buildEnrichmentPlan(targetSlug, renderTargetSlug, enrichOptions{
		Space:            *space,
		Ref:              *ref,
		WhereResource:    *whereResource,
		HelmCLIOverrides: helmCLIOverrides,
	})
	if err != nil {
		return err
	}
	result, err := enrichflow.Write(plan.TargetPath, plan)
	if err != nil {
		return err
	}
	return writeJSONOutput(*out, result, *pretty)
}

type enrichOptions struct {
	Space            string
	Ref              string
	WhereResource    string
	HelmCLIOverrides []model.HelmCLIOverride
}

func buildEnrichmentPlan(targetSlug, renderTargetSlug string, opts enrichOptions) (enrichflow.Plan, error) {
	imported, err := gitopsflow.ImportWithOptions(targetSlug, renderTargetSlug, opts.Ref, opts.Space, opts.WhereResource, importer.ImportOptions{
		HelmCLIOverrides: opts.HelmCLIOverrides,
	})
	if err != nil {
		return enrichflow.Plan{}, err
	}
	plan := enrichflow.BuildPlan(imported)
	if err := enrichflow.MarkExisting(imported.TargetPath, &plan); err != nil {
		return enrichflow.Plan{}, err
	}
	return plan, nil
}

func writeTextOutput(path, text string) error {
	raw := strings.TrimSpace(path)
	if raw == "" || raw == "-" {
		_, err := fmt.Fprint(os.Stdout, text)
		return err
	}
	return os.WriteFile(raw, []byte(text), 0o644)
}

func printEnrichUsage(out io.Writer) {
	printCommandHelp(
		out,
		"cub-gen enrich: propose repo-side proof metadata without mutating app manifests",
		[]string{
			"Use enrich when users need provenance, ownership, route badges, and PR/MR link metadata where they review code.",
			"Preview is read-only. Write creates sidecar files under .cub-gen/enrichment and refuses to overwrite existing artifacts.",
		},
		helpSection{
			Title: "Usage",
			Lines: []string{
				"  cub-gen enrich preview [--space SPACE] [--ref REF] [--where-resource EXPR] [--set KEY=VALUE] [--set-string KEY=VALUE] [--set-file KEY=PATH] [--json|--patch] [--out FILE|-] [--pretty] <target-path> [<render-target-path>]",
				"  cub-gen enrich write [--space SPACE] [--ref REF] [--where-resource EXPR] [--set KEY=VALUE] [--set-string KEY=VALUE] [--set-file KEY=PATH] [--json] [--out FILE|-] [--pretty] <target-path> [<render-target-path>]",
			},
		},
		helpSection{
			Title: "Examples",
			Lines: []string{
				"  cub-gen enrich preview --space platform ./testdata/app-of-apps-standalone",
				"  cub-gen enrich preview --patch --space platform ./testdata/openchoreo-hardgate",
				"  cub-gen enrich write --space platform ./examples/helm-paas",
			},
		},
		helpSection{
			Title: "Output",
			Lines: []string{
				"  - source-link annotations point from sidecar proof to source files",
				"  - ownership-label annotations name app, environment, platform, security, or workflow owners",
				"  - route-badge annotations summarize apply-here, lift-upstream, overlay, or block/escalate guidance",
				"  - pr-mr-link annotations carry change_id for GitHub PR and ConfigHub MR correlation",
			},
		},
	)
}
