package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/confighub/cub-gen/internal/detect"
	gitopsflow "github.com/confighub/cub-gen/internal/gitops"
	"github.com/confighub/cub-gen/internal/importer"
	"github.com/confighub/cub-gen/internal/publish"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		printUsage(os.Stderr)
		return errors.New("command required")
	}

	switch args[0] {
	case "help", "-h", "--help":
		printUsage(os.Stdout)
		return nil
	case "detect":
		return runDetect(args[1:])
	case "import":
		return runLegacyImport(args[1:])
	case "publish":
		return runPublish(args[1:])
	case "verify":
		return runVerify(args[1:])
	case "gitops":
		return runGitOps(args[1:])
	default:
		printUsage(os.Stderr)
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func runDetect(args []string) error {
	fs := flag.NewFlagSet("detect", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	repo := fs.String("repo", ".", "Path to local repository")
	ref := fs.String("ref", "HEAD", "Git ref label to include in output")
	pretty := fs.Bool("pretty", true, "Pretty-print JSON output")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	result, err := detect.ScanRepo(*repo, *ref)
	if err != nil {
		return err
	}
	return writeJSON(os.Stdout, result, *pretty)
}

// runLegacyImport retains the original prototype import command.
func runLegacyImport(args []string) error {
	fs := flag.NewFlagSet("import", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	repo := fs.String("repo", ".", "Path to local repository")
	ref := fs.String("ref", "HEAD", "Git ref label to include in output")
	space := fs.String("space", "default", "Target ConfigHub space")
	out := fs.String("out", "-", "Output file path, or '-' for stdout")
	pretty := fs.Bool("pretty", true, "Pretty-print JSON output")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	result, err := importer.ImportRepo(*repo, *ref, *space)
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

func runPublish(args []string) error {
	fs := flag.NewFlagSet("publish", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	in := fs.String("in", "-", "ImportFlow JSON input path, or '-' for stdin")
	out := fs.String("out", "-", "Bundle JSON output path, or '-' for stdout")
	space := fs.String("space", "default", "ConfigHub space label (direct mode)")
	ref := fs.String("ref", "HEAD", "Git ref label to include in output (direct mode)")
	whereResource := fs.String("where-resource", "", "Additional resource filter expression (direct mode)")
	pretty := fs.Bool("pretty", true, "Pretty-print JSON output")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	var imported gitopsflow.ImportFlowResult
	switch fs.NArg() {
	case 0:
		var inputBytes []byte
		var err error
		if *in == "-" {
			inputBytes, err = io.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("read stdin: %w", err)
			}
		} else {
			inputBytes, err = os.ReadFile(*in)
			if err != nil {
				return fmt.Errorf("read input file: %w", err)
			}
		}
		if err := json.Unmarshal(inputBytes, &imported); err != nil {
			return fmt.Errorf("parse import flow json: %w", err)
		}
	case 2:
		if *in != "-" {
			return errors.New("cannot combine --in with direct target mode")
		}
		targetSlug := fs.Arg(0)
		renderTargetSlug := fs.Arg(1)
		var err error
		imported, err = gitopsflow.Import(targetSlug, renderTargetSlug, *ref, *space, *whereResource)
		if err != nil {
			return err
		}
	default:
		return errors.New("usage: cub-gen publish [flags] [<target-slug> <render-target-slug>]")
	}

	bundle := publish.BuildBundle(imported)
	if *out == "-" {
		return writeJSON(os.Stdout, bundle, *pretty)
	}

	f, err := os.Create(*out)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer func() {
		_ = f.Close()
	}()
	return writeJSON(f, bundle, *pretty)
}

func runVerify(args []string) error {
	fs := flag.NewFlagSet("verify", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	in := fs.String("in", "-", "Bundle JSON input path, or '-' for stdin")
	jsonOut := fs.Bool("json", false, "Output JSON")
	pretty := fs.Bool("pretty", true, "Pretty-print JSON output")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("usage: cub-gen verify [flags]")
	}

	var inputBytes []byte
	var err error
	if *in == "-" {
		inputBytes, err = io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("read stdin: %w", err)
		}
	} else {
		inputBytes, err = os.ReadFile(*in)
		if err != nil {
			return fmt.Errorf("read input file: %w", err)
		}
	}

	var bundle publish.ChangeBundle
	if err := json.Unmarshal(inputBytes, &bundle); err != nil {
		return fmt.Errorf("parse bundle json: %w", err)
	}
	if err := publish.VerifyBundle(bundle); err != nil {
		return err
	}

	if *jsonOut {
		return writeJSON(os.Stdout, map[string]any{
			"valid":            true,
			"digest_algorithm": bundle.DigestAlgorithm,
			"bundle_digest":    bundle.BundleDigest,
			"change_id":        bundle.ChangeID,
		}, *pretty)
	}

	fmt.Printf("Bundle verification OK: %s\n", bundle.BundleDigest)
	return nil
}

func runGitOps(args []string) error {
	if len(args) == 0 {
		printGitOpsUsage(os.Stderr)
		return errors.New("gitops subcommand required")
	}

	switch args[0] {
	case "help", "-h", "--help":
		printGitOpsUsage(os.Stdout)
		return nil
	case "discover":
		return runGitOpsDiscover(args[1:])
	case "import":
		return runGitOpsImport(args[1:])
	case "cleanup":
		return runGitOpsCleanup(args[1:])
	default:
		printGitOpsUsage(os.Stderr)
		return fmt.Errorf("unknown gitops subcommand: %s", args[0])
	}
}

func runGitOpsDiscover(args []string) error {
	fs := flag.NewFlagSet("gitops discover", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	space := fs.String("space", "default", "ConfigHub space label")
	ref := fs.String("ref", "HEAD", "Git ref label to include in output")
	whereResource := fs.String("where-resource", "", "Additional resource filter expression")
	jsonOut := fs.Bool("json", false, "Output JSON")
	pretty := fs.Bool("pretty", true, "Pretty-print JSON output")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	if fs.NArg() != 1 {
		return errors.New("usage: cub-gen gitops discover [flags] <target-slug>")
	}
	targetSlug := fs.Arg(0)

	result, err := gitopsflow.Discover(targetSlug, *ref, *space, *whereResource)
	if err != nil {
		return err
	}

	if *jsonOut {
		return writeJSON(os.Stdout, result, *pretty)
	}

	if len(result.Resources) == 0 {
		fmt.Println("No GitOps resources were discovered for the specified target")
		return nil
	}

	printDiscoverTable(result)
	return nil
}

func runGitOpsImport(args []string) error {
	fs := flag.NewFlagSet("gitops import", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	space := fs.String("space", "default", "ConfigHub space label")
	ref := fs.String("ref", "HEAD", "Git ref label to include in output")
	whereResource := fs.String("where-resource", "", "Additional resource filter expression")
	wait := fs.Bool("wait", false, "Accepted for parity with cub gitops import")
	jsonOut := fs.Bool("json", false, "Output JSON")
	pretty := fs.Bool("pretty", true, "Pretty-print JSON output")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	_ = wait

	if fs.NArg() != 2 {
		return errors.New("usage: cub-gen gitops import [flags] <target-slug> <render-target-slug>")
	}
	targetSlug := fs.Arg(0)
	renderTargetSlug := fs.Arg(1)

	result, err := gitopsflow.Import(targetSlug, renderTargetSlug, *ref, *space, *whereResource)
	if err != nil {
		return err
	}

	if *jsonOut {
		return writeJSON(os.Stdout, result, *pretty)
	}

	if len(result.Discovered) == 0 {
		fmt.Println("No GitOps resources were discovered for the specified target")
		return nil
	}

	fmt.Printf("Discovered %d GitOps resources, creating renderer units...\n", len(result.Discovered))
	fmt.Printf("Created renderer units: %d\n", len(result.DryUnits))
	fmt.Println("Rendering discovered resources...")
	fmt.Printf("Created wet units: %d\n", len(result.WetUnits))
	fmt.Printf("Created links: %d\n", len(result.Links))
	fmt.Println("GitOps import complete")
	return nil
}

func runGitOpsCleanup(args []string) error {
	fs := flag.NewFlagSet("gitops cleanup", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	space := fs.String("space", "default", "ConfigHub space label")
	jsonOut := fs.Bool("json", false, "Output JSON")
	pretty := fs.Bool("pretty", true, "Pretty-print JSON output")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	if fs.NArg() != 1 {
		return errors.New("usage: cub-gen gitops cleanup [flags] <target-slug>")
	}
	targetSlug := fs.Arg(0)

	deleted, filePath, err := gitopsflow.Cleanup(targetSlug, *space)
	if err != nil {
		return err
	}

	result := map[string]any{
		"space":         *space,
		"target_slug":   targetSlug,
		"discover_file": filePath,
		"deleted":       deleted,
	}

	if *jsonOut {
		return writeJSON(os.Stdout, result, *pretty)
	}

	if deleted {
		fmt.Printf("Deleted discover unit state file: %s\n", filePath)
	} else {
		fmt.Printf("No discover unit state file found: %s\n", filePath)
	}
	return nil
}

func printDiscoverTable(result gitopsflow.DiscoverResult) {
	rows := make([][2]string, 0, len(result.Resources))
	for _, r := range result.Resources {
		rows = append(rows, [2]string{r.ResourceType, r.ResourceName})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i][0] != rows[j][0] {
			return rows[i][0] < rows[j][0]
		}
		return rows[i][1] < rows[j][1]
	})

	fmt.Println("Resource Type\tResource Name")
	for _, row := range rows {
		fmt.Printf("%s\t%s\n", row[0], row[1])
	}
}

func writeJSON(out io.Writer, v any, pretty bool) error {
	enc := json.NewEncoder(out)
	if pretty {
		enc.SetIndent("", "  ")
	}
	return enc.Encode(v)
}

func printUsage(out io.Writer) {
	fmt.Fprintln(out, "cub-gen: prototype generator importer for agentic GitOps")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Usage:")
	fmt.Fprintln(out, "  cub-gen detect [--repo PATH] [--ref REF] [--pretty]")
	fmt.Fprintln(out, "  cub-gen import [--repo PATH] [--ref REF] [--space SPACE] [--out FILE|-] [--pretty]")
	fmt.Fprintln(out, "  cub-gen publish [--in FILE|-] [--out FILE|-] [--pretty]")
	fmt.Fprintln(out, "  cub-gen publish [--space SPACE] [--ref REF] [--where-resource EXPR] [--out FILE|-] [--pretty] <target-slug> <render-target-slug>")
	fmt.Fprintln(out, "  cub-gen verify [--in FILE|-] [--json] [--pretty]")
	fmt.Fprintln(out, "  cub-gen gitops <discover|import|cleanup> [flags]")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "GitOps parity examples:")
	fmt.Fprintln(out, "  cub-gen gitops discover --space my-space ./examples/helm-paas")
	fmt.Fprintln(out, "  cub-gen gitops import --space my-space ./examples/helm-paas local-renderer")
	fmt.Fprintln(out, "  cub-gen gitops cleanup --space my-space ./examples/helm-paas")
	fmt.Fprintln(out, "  cub-gen gitops import --space my-space --json ./examples/helm-paas local-renderer | cub-gen publish --in -")
	fmt.Fprintln(out, "  cub-gen publish --space my-space ./examples/helm-paas ./examples/helm-paas")
	fmt.Fprintln(out, "  cub-gen publish --space my-space ./examples/helm-paas ./examples/helm-paas | cub-gen verify --in -")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Note: gitops commands are local-only prototypes that mirror cub gitops stages.")
}

func printGitOpsUsage(out io.Writer) {
	fmt.Fprintln(out, "cub-gen gitops: local parity commands for cub gitops pattern")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Usage:")
	fmt.Fprintln(out, "  cub-gen gitops discover [--space SPACE] [--ref REF] [--where-resource EXPR] [--json] <target-slug>")
	fmt.Fprintln(out, "  cub-gen gitops import [--space SPACE] [--ref REF] [--where-resource EXPR] [--wait] [--json] <target-slug> <render-target-slug>")
	fmt.Fprintln(out, "  cub-gen gitops cleanup [--space SPACE] [--json] <target-slug>")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Supported where-resource clauses:")
	fmt.Fprintln(out, "  kind = 'HelmRelease' | kind = 'Application' | kind = 'Kustomization'")
	fmt.Fprintln(out, "  kind IN ('HelmRelease','Application')")
	fmt.Fprintln(out, "  name = 'checkout-api' | resource_name LIKE '<contains-api>' | root LIKE '<contains-prod>'")
	fmt.Fprintln(out, "  combine clauses with AND")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Examples:")
	fmt.Fprintln(out, "  cub-gen gitops discover --space my-space ./examples/scoredev-paas")
	fmt.Fprintln(out, "  cub-gen gitops discover --where-resource \"kind IN ('HelmRelease') AND resource_name LIKE '<contains-payments>'\" ./examples/helm-paas")
	fmt.Fprintln(out, "  cub-gen gitops import --space my-space ./examples/springboot-paas render-target")
	fmt.Fprintln(out, "  cub-gen gitops cleanup --space my-space ./examples/springboot-paas")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Tip: <target-slug> is a local repo path in this prototype.")
}
