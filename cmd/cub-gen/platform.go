package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	platformflow "github.com/confighub/cub-gen/internal/platform"
)

func runPlatform(args []string) error {
	if len(args) == 0 {
		printPlatformUsage(os.Stderr)
		return errors.New("platform subcommand required")
	}

	switch args[0] {
	case "help", "-h", "--help":
		printPlatformUsage(os.Stdout)
		return nil
	case "import":
		return runPlatformImport(args[1:])
	case "fanout":
		return runPlatformFanout(args[1:])
	case "adapt":
		return runPlatformAdapt(args[1:])
	default:
		printPlatformUsage(os.Stderr)
		return fmt.Errorf("unknown platform subcommand: %s", args[0])
	}
}

func runPlatformImport(args []string) error {
	fs := flag.NewFlagSet("platform import", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	jsonOut := fs.Bool("json", true, "Output JSON")
	pretty := fs.Bool("pretty", true, "Pretty-print JSON output")
	out := fs.String("out", "-", "Output file path, or '-' for stdout")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if err := requireJSONOutput("platform import", *jsonOut); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("usage: cub-gen platform import [flags] <manifest>")
	}
	graph, err := platformflow.ImportManifest(fs.Arg(0), platformflow.ImportOptions{})
	if err != nil {
		return err
	}
	return writeJSONOutput(*out, graph, *pretty)
}

func runPlatformFanout(args []string) error {
	fs := flag.NewFlagSet("platform fanout", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	jsonOut := fs.Bool("json", true, "Output JSON")
	pretty := fs.Bool("pretty", true, "Pretty-print JSON output")
	out := fs.String("out", "-", "Output file path, or '-' for stdout")
	variant := fs.String("variant", "", "Variant filter: variant name, component/variant, or variant id")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if err := requireJSONOutput("platform fanout", *jsonOut); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("usage: cub-gen platform fanout [flags] <manifest>")
	}
	result, err := platformflow.FanoutManifest(fs.Arg(0), platformflow.FanoutOptions{
		VariantFilter: *variant,
	})
	if err != nil {
		return err
	}
	return writeJSONOutput(*out, result, *pretty)
}

func runPlatformAdapt(args []string) error {
	fs := flag.NewFlagSet("platform adapt", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	jsonOut := fs.Bool("json", true, "Output JSON")
	pretty := fs.Bool("pretty", true, "Pretty-print JSON output")
	out := fs.String("out", "-", "Output file path, or '-' for stdout")
	variant := fs.String("variant", "", "Variant filter: variant name, component/variant, or adaptation id")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if err := requireJSONOutput("platform adapt", *jsonOut); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("usage: cub-gen platform adapt [flags] <manifest>")
	}
	result, err := platformflow.AdaptManifest(fs.Arg(0), platformflow.AdaptationOptions{
		VariantFilter: *variant,
	})
	if err != nil {
		return err
	}
	return writeJSONOutput(*out, result, *pretty)
}

func printPlatformUsage(out io.Writer) {
	printCommandHelp(
		out,
		"cub-gen platform: import estates and emit variant bundles without deploying",
		[]string{
			"Use platform import when app source config, platform contracts, environment bindings, and rendered output live in separate repos.",
			"The command reads a manifest, imports each local repo, and emits Components, Variants, Deployment Variants, Targets, generator inputs, WET targets, connections, and diagnostics.",
			"Use platform fanout when the same Component has dev/stage/prod or tenant variants and each needs its own governed ConfigHub bundle.",
			"Use platform adapt when a cloned Deployment Variant has a Target but still has placeholders that block apply.",
		},
		helpSection{
			Title: "Usage",
			Lines: []string{
				"  cub-gen platform import [--json] [--out FILE|-] [--pretty] <manifest>",
				"  cub-gen platform fanout [--json] [--out FILE|-] [--pretty] [--variant NAME] <manifest>",
				"  cub-gen platform adapt [--json] [--out FILE|-] [--pretty] [--variant NAME] <manifest>",
			},
		},
		helpSection{
			Title: "Examples",
			Lines: []string{
				"  cub-gen platform import --json ./testdata/platform-estate/platform.yaml",
				"  cub-gen platform fanout --json ./testdata/variant-fanout/platform.yaml",
				"  cub-gen platform fanout --variant dev --json ./testdata/variant-fanout/platform.yaml",
				"  cub-gen platform adapt --json ./testdata/deployment-adaptation/platform.yaml",
			},
		},
		helpSection{
			Title: "Boundaries",
			Lines: []string{
				"  - read-only: no repo rewrites, no deploys, no control-plane side effects",
				"  - missing repos, missing owners, and unsupported generator paths are diagnostics, not guesses",
				"  - fanout requires explicit variant sources when repo metadata would be ambiguous",
				"  - adapt requires explicit placeholder tokens and context; it proposes sidecars, not hidden writes",
				"  - use gitops import when you only need one repo; use platform import when you need the estate graph",
			},
		},
	)
}
