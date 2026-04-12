package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/confighub/cub-gen/internal/score"
)

func runScore(args []string) error {
	if len(args) == 0 {
		printScoreUsage(os.Stderr)
		return errors.New("score subcommand required")
	}

	switch args[0] {
	case "help", "-h", "--help":
		printScoreUsage(os.Stdout)
		return nil
	case "validate-workload":
		return runScoreValidateWorkload(args[1:])
	default:
		printScoreUsage(os.Stderr)
		return fmt.Errorf("unknown score subcommand: %s", args[0])
	}
}

func runScoreValidateWorkload(args []string) error {
	fs := flag.NewFlagSet("score validate-workload", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: cub-gen score validate-workload [flags]")
		fmt.Fprintln(fs.Output())
		fmt.Fprintln(fs.Output(), "Validate whether a Score workload stays within the approved workload-class contract.")
		fmt.Fprintln(fs.Output())
		fmt.Fprintln(fs.Output(), "This is a client-side contract check for Score examples and CI paths.")
		fmt.Fprintln(fs.Output(), "Approved resource types return ALLOW; unapproved resource types return ESCALATE.")
		fmt.Fprintln(fs.Output())
		fmt.Fprintln(fs.Output(), "Examples:")
		fmt.Fprintln(fs.Output(), "  cub-gen score validate-workload --score ./examples/scoredev-paas/score.yaml --contract ./examples/scoredev-paas/platform/contracts/workload-class.yaml")
		fmt.Fprintln(fs.Output())
		fmt.Fprintln(fs.Output(), "Flags:")
		fs.PrintDefaults()
	}

	scorePath := fs.String("score", "", "Path to score.yaml (required)")
	contractPath := fs.String("contract", "", "Path to workload-class.yaml (required)")
	jsonOut := fs.Bool("json", false, "Output JSON")
	pretty := fs.Bool("pretty", true, "Pretty-print JSON output")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("usage: cub-gen score validate-workload --score <path> --contract <path>")
	}
	if strings.TrimSpace(*scorePath) == "" {
		return errors.New("--score is required")
	}
	if strings.TrimSpace(*contractPath) == "" {
		return errors.New("--contract is required")
	}

	result, err := score.ValidateWorkload(score.ValidateWorkloadOptions{
		ScorePath:    *scorePath,
		ContractPath: *contractPath,
	})
	if err != nil {
		return err
	}

	if *jsonOut {
		if err := writeJSON(os.Stdout, result, *pretty); err != nil {
			return err
		}
		if !result.Allowed {
			return fmt.Errorf("workload requires platform review for resource types: %s", strings.Join(result.UnapprovedResourceTypes, ", "))
		}
		return nil
	}

	sort.Strings(result.ResourceTypes)
	sort.Strings(result.ApprovedResourceTypes)
	sort.Strings(result.UnapprovedResourceTypes)

	fmt.Println(result.State)
	fmt.Printf("  Score:          %s\n", result.ScorePath)
	fmt.Printf("  Contract:       %s\n", result.ContractPath)
	if result.WorkloadClass != "" {
		fmt.Printf("  Workload class: %s\n", result.WorkloadClass)
	}
	fmt.Printf("  Resource types: %s\n", strings.Join(result.ResourceTypes, ", "))
	fmt.Printf("  Approved:       %s\n", strings.Join(result.ApprovedResourceTypes, ", "))
	fmt.Printf("  Reason:         %s\n", result.Reason)

	if !result.Allowed {
		fmt.Printf("  Escalated:      %s\n", strings.Join(result.UnapprovedResourceTypes, ", "))
		return fmt.Errorf("workload requires platform review for resource types: %s", strings.Join(result.UnapprovedResourceTypes, ", "))
	}
	return nil
}

func printScoreUsage(out io.Writer) {
	fmt.Fprintln(out, "Usage: cub-gen score <subcommand> [flags]")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Score.dev onboarding and workload-contract checks.")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Subcommands:")
	fmt.Fprintln(out, "  validate-workload  Check if score.yaml resource types stay within the approved workload class")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Run 'cub-gen score <subcommand> --help' for details.")
}
