package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"sort"

	"github.com/confighub/cub-gen/internal/springboot"
)

func runSpringBoot(args []string) error {
	if len(args) == 0 {
		printSpringBootUsage(os.Stderr)
		return errors.New("springboot subcommand required")
	}

	switch args[0] {
	case "help", "-h", "--help":
		printSpringBootUsage(os.Stdout)
		return nil
	case "init":
		return runSpringBootInit(args[1:])
	case "validate-mutation":
		return runSpringBootValidateMutation(args[1:])
	default:
		printSpringBootUsage(os.Stderr)
		return fmt.Errorf("unknown springboot subcommand: %s", args[0])
	}
}

func runSpringBootInit(args []string) error {
	fs := flag.NewFlagSet("springboot init", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: cub-gen springboot init [flags] <source-path>")
		fmt.Fprintln(fs.Output())
		fmt.Fprintln(fs.Output(), "Generate starter cub-gen material for a Spring Boot application.")
		fmt.Fprintln(fs.Output())
		fmt.Fprintln(fs.Output(), "This command detects a Spring Boot app and generates:")
		fmt.Fprintln(fs.Output(), "  - platform/base/runtime-policy.yaml    (platform policy skeleton)")
		fmt.Fprintln(fs.Output(), "  - platform/overlays/prod/slo-policy.yaml (SLO skeleton)")
		fmt.Fprintln(fs.Output(), "  - operational/field-routes.yaml        (field ownership rules)")
		fmt.Fprintln(fs.Output(), "  - confighub/{app}-{dev,stage,prod}.yaml (ConfigHub unit starters)")
		fmt.Fprintln(fs.Output(), "  - .cub-gen/config.yaml                 (generator config)")
		fmt.Fprintln(fs.Output())
		fmt.Fprintln(fs.Output(), "Flags:")
		fs.PrintDefaults()
	}

	appName := fs.String("app", "", "Application name (default: derived from detection)")
	output := fs.String("output", "", "Output directory (default: source directory)")
	namespace := fs.String("namespace", "apps", "Kubernetes namespace for generated manifests")
	dryRun := fs.Bool("dry-run", false, "Show what would be generated without writing files")
	jsonOut := fs.Bool("json", false, "Output JSON")
	pretty := fs.Bool("pretty", true, "Pretty-print JSON output")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	if fs.NArg() != 1 {
		return errors.New("usage: cub-gen springboot init [flags] <source-path>")
	}
	sourcePath := fs.Arg(0)

	result, err := springboot.Init(springboot.InitOptions{
		SourcePath: sourcePath,
		OutputPath: *output,
		AppName:    *appName,
		Namespace:  *namespace,
		DryRun:     *dryRun,
	})
	if err != nil {
		return err
	}

	if *jsonOut {
		return writeJSON(os.Stdout, result, *pretty)
	}

	// Human-readable output
	fmt.Println("Spring Boot App Detected")
	fmt.Println("========================")
	fmt.Printf("App name:    %s\n", result.AppName)
	fmt.Printf("Source:      %s\n", result.SourcePath)
	fmt.Printf("Output:      %s\n", result.OutputPath)
	fmt.Printf("Profile:     %s\n", result.Detection.Profile)
	fmt.Printf("Confidence:  %.0f%%\n", result.Detection.Confidence*100)
	fmt.Println()

	if *dryRun {
		fmt.Println("[DRY RUN] Would create:")
	} else {
		fmt.Println("Files created:")
	}
	sort.Strings(result.FilesCreated)
	for _, f := range result.FilesCreated {
		fmt.Printf("  %s\n", f)
	}
	fmt.Println()

	fmt.Println("Next steps:")
	for i, item := range result.Checklist {
		fmt.Printf("  %d. %s\n", i+1, item)
	}
	fmt.Println()

	if !*dryRun {
		fmt.Println("What this does NOT do (still manual):")
		fmt.Println("  - Parse your actual Spring config for field values")
		fmt.Println("  - Generate production-ready manifests for all app shapes")
		fmt.Println("  - Infer ownership beyond the default patterns")
		fmt.Println("  - Create actual Kubernetes manifests (deployment.yaml, etc.)")
		fmt.Println()
		fmt.Println("Review the generated files and customize for your app.")
	}

	return nil
}

func runSpringBootValidateMutation(args []string) error {
	fs := flag.NewFlagSet("springboot validate-mutation", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: cub-gen springboot validate-mutation [flags] <field-path>")
		fmt.Fprintln(fs.Output())
		fmt.Fprintln(fs.Output(), "Validate whether a mutation to a field path is allowed by field-routes.yaml.")
		fmt.Fprintln(fs.Output())
		fmt.Fprintln(fs.Output(), "This enforces the block/escalate boundary: mutations to platform-owned")
		fmt.Fprintln(fs.Output(), "fields (like spring.datasource.*) will fail with a non-zero exit code.")
		fmt.Fprintln(fs.Output())
		fmt.Fprintln(fs.Output(), "Examples:")
		fmt.Fprintln(fs.Output(), "  # Allowed (app-owned)")
		fmt.Fprintln(fs.Output(), "  cub-gen springboot validate-mutation --routes ./operational/field-routes.yaml feature.inventory.reservationMode")
		fmt.Fprintln(fs.Output())
		fmt.Fprintln(fs.Output(), "  # Blocked (platform-owned)")
		fmt.Fprintln(fs.Output(), "  cub-gen springboot validate-mutation --routes ./operational/field-routes.yaml spring.datasource.url")
		fmt.Fprintln(fs.Output())
		fmt.Fprintln(fs.Output(), "Flags:")
		fs.PrintDefaults()
	}

	routesPath := fs.String("routes", "", "Path to field-routes.yaml (required)")
	jsonOut := fs.Bool("json", false, "Output JSON")
	pretty := fs.Bool("pretty", true, "Pretty-print JSON output")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	if fs.NArg() != 1 {
		return errors.New("usage: cub-gen springboot validate-mutation --routes <path> <field-path>")
	}
	if *routesPath == "" {
		return errors.New("--routes is required")
	}
	fieldPath := fs.Arg(0)

	result, err := springboot.ValidateMutation(springboot.ValidateMutationOptions{
		FieldRoutesPath: *routesPath,
		FieldPath:       fieldPath,
	})
	if err != nil {
		return err
	}

	if *jsonOut {
		return writeJSON(os.Stdout, result, *pretty)
	}

	// Human-readable output
	if result.Allowed {
		fmt.Println("ALLOWED")
		fmt.Printf("  Field:  %s\n", result.FieldPath)
		fmt.Printf("  Owner:  %s\n", result.Owner)
		fmt.Printf("  Action: %s\n", result.Action)
		if result.MatchedRule != "" {
			fmt.Printf("  Rule:   %s\n", result.MatchedRule)
		}
		fmt.Printf("  Reason: %s\n", result.Reason)
		return nil
	}

	// Blocked - print and return error for non-zero exit
	fmt.Println("BLOCKED")
	fmt.Printf("  Field:  %s\n", result.FieldPath)
	fmt.Printf("  Owner:  %s\n", result.Owner)
	fmt.Printf("  Action: %s\n", result.Action)
	fmt.Printf("  Rule:   %s\n", result.MatchedRule)
	fmt.Printf("  Reason: %s\n", result.Reason)
	return fmt.Errorf("mutation to %s is blocked by field routes", result.FieldPath)
}

func printSpringBootUsage(out *os.File) {
	fmt.Fprintln(out, "Usage: cub-gen springboot <subcommand> [flags]")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Spring Boot onboarding and enforcement commands.")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Subcommands:")
	fmt.Fprintln(out, "  init               Generate starter cub-gen material for a Spring Boot app")
	fmt.Fprintln(out, "  validate-mutation  Check if a field mutation is allowed by field-routes.yaml")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Run 'cub-gen springboot <subcommand> --help' for details.")
}
