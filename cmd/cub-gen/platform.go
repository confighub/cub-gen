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

func printPlatformUsage(out io.Writer) {
	printCommandHelp(
		out,
		"cub-gen platform: import a multi-repo platform estate as a read-only graph",
		[]string{
			"Use platform import when app intent, platform contracts, environment bindings, and rendered output live in separate repos.",
			"The command reads a manifest, imports each local repo, and emits Components, Deployable Variants, Targets, generator inputs, WET targets, connections, and diagnostics.",
		},
		helpSection{
			Title: "Usage",
			Lines: []string{
				"  cub-gen platform import [--json] [--out FILE|-] [--pretty] <manifest>",
			},
		},
		helpSection{
			Title: "Examples",
			Lines: []string{
				"  cub-gen platform import --json ./testdata/platform-estate/platform.yaml",
			},
		},
		helpSection{
			Title: "Boundaries",
			Lines: []string{
				"  - read-only: no repo rewrites, no deploys, no control-plane side effects",
				"  - missing repos, missing owners, and unsupported generator paths are diagnostics, not guesses",
				"  - use gitops import when you only need one repo; use platform import when you need the estate graph",
			},
		},
	)
}
