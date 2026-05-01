package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	gitopsflow "github.com/confighub/cub-gen/internal/gitops"
	"github.com/confighub/cub-gen/internal/importer"
	"github.com/confighub/cub-gen/internal/model"
	"github.com/confighub/cub-gen/internal/registry"
)

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
	adoptionReport := fs.Bool("adoption-report", false, "Include read-only platform adoption report")
	jsonOut := fs.Bool("json", false, "Output JSON")
	pretty := fs.Bool("pretty", true, "Pretty-print JSON output")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	if fs.NArg() != 1 {
		return errors.New("usage: cub-gen gitops discover [flags] <target-path>")
	}
	targetSlug := fs.Arg(0)

	result, err := gitopsflow.DiscoverWithOptions(targetSlug, *ref, *space, *whereResource, gitopsflow.DiscoverOptions{
		AdoptionReport: *adoptionReport,
	})
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
	printAdoptionReport(result.AdoptionReport)
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
	overrideFlags := addHelmCLIOverrideFlags(fs)
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	_ = wait
	helmCLIOverrides, err := overrideFlags.parse()
	if err != nil {
		return err
	}

	targetSlug, renderTargetSlug, err := resolveTargetPairArgs(fs, "usage: cub-gen gitops import [flags] <target-path> [<render-target-path>]")
	if err != nil {
		return err
	}

	result, err := gitopsflow.ImportWithOptions(targetSlug, renderTargetSlug, *ref, *space, *whereResource, importer.ImportOptions{
		HelmCLIOverrides: helmCLIOverrides,
	})
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

	printImportTable(result)
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
		return errors.New("usage: cub-gen gitops cleanup [flags] <target-path>")
	}
	targetSlug := fs.Arg(0)

	cleanupResult, err := gitopsflow.Cleanup(targetSlug, *space)
	if err != nil {
		return err
	}

	if *jsonOut {
		return writeJSON(os.Stdout, cleanupResult, *pretty)
	}

	if cleanupResult.Deleted {
		fmt.Printf("Deleted discover unit state file: %s\n", cleanupResult.DiscoverFile)
	} else {
		fmt.Printf("No discover unit state file found: %s\n", cleanupResult.DiscoverFile)
	}
	return nil
}

func printDiscoverTable(result gitopsflow.DiscoverResult) {
	printGeneratorChainSummaries(result.ChainSummaries)

	kindCapabilities := map[string]string{}
	for _, kind := range registry.Kinds() {
		spec, ok := registry.Spec(kind)
		if !ok {
			continue
		}
		kindCapabilities[string(kind)] = strings.Join(spec.Capabilities, ",")
	}

	confidenceByGeneratorID := map[string]float64{}
	for _, d := range result.Detections {
		confidenceByGeneratorID[d.ID] = d.Confidence
	}

	rows := make([][6]string, 0, len(result.Resources))
	for _, r := range result.Resources {
		confidence := "-"
		if v, ok := confidenceByGeneratorID[r.GeneratorID]; ok {
			confidence = fmt.Sprintf("%.2f", v)
		}
		capabilities := kindCapabilities[r.GeneratorKind]
		if strings.TrimSpace(capabilities) == "" {
			capabilities = "-"
		}
		rows = append(rows, [6]string{
			r.ResourceType,
			r.ResourceName,
			r.GeneratorKind,
			r.GeneratorProfile,
			capabilities,
			confidence,
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i][0] != rows[j][0] {
			return rows[i][0] < rows[j][0]
		}
		return rows[i][1] < rows[j][1]
	})

	fmt.Println("Resource Type\tResource Name\tGenerator Kind\tProfile\tCapabilities\tConfidence")
	for _, row := range rows {
		fmt.Printf("%s\t%s\t%s\t%s\t%s\t%s\n", row[0], row[1], row[2], row[3], row[4], row[5])
	}
}

func printAdoptionReport(report *gitopsflow.AdoptionReport) {
	if report == nil {
		return
	}
	fmt.Println()
	fmt.Println("Adoption report")
	fmt.Println("Generators\tSource Artifacts\tRendered Targets\tOwners")
	fmt.Printf("%d\t%d\t%d\t%s\n",
		report.Summary.GeneratorCount,
		report.Summary.SourceArtifactCount,
		report.Summary.RenderedTargetCount,
		strings.Join(report.Summary.Owners, ","),
	)
	for _, generator := range report.Generators {
		fmt.Printf("%s\t%s\t%s\t%d routes\t%d gaps\n",
			generator.Kind,
			generator.Profile,
			generator.Name,
			len(generator.ChangeRoutes),
			len(generator.UnsupportedGaps),
		)
	}
}

func printImportTable(result gitopsflow.ImportFlowResult) {
	printGeneratorChainSummaries(result.ChainSummaries)

	fmt.Printf("Discovered %d GitOps resources, creating renderer units...\n", len(result.Discovered))
	printImportDiscoveredTable(result)
	fmt.Printf("Created renderer units: %d\n", len(result.DryUnits))
	fmt.Println("Rendering discovered resources...")
	fmt.Printf("Created wet units: %d\n", len(result.WetUnits))
	fmt.Printf("Created links: %d\n", len(result.Links))
	fmt.Printf("Generated contracts: %d\n", len(result.Contracts))
	fmt.Printf("Generated provenance records: %d\n", len(result.Provenance))
	fmt.Printf("Generated inverse transform plans: %d\n", len(result.InversePlans))
	printImportTripleSummary(result)
	fmt.Println("GitOps import complete")
}

func printGeneratorChainSummaries(summaries []model.GeneratorChainSummary) {
	if len(summaries) == 0 {
		return
	}
	fmt.Println("Generator chains")
	fmt.Println("ID\tDisplay\tStages\tMappings")
	for _, summary := range summaries {
		fmt.Printf("%s\t%s\t%d\t%d\n", summary.ID, summary.Display, summary.StageCount, summary.MappingCount)
	}
	fmt.Println()
}

func printImportDiscoveredTable(result gitopsflow.ImportFlowResult) {
	kindCapabilities := map[string]string{}
	for _, kind := range registry.Kinds() {
		spec, ok := registry.Spec(kind)
		if !ok {
			continue
		}
		kindCapabilities[string(kind)] = strings.Join(spec.Capabilities, ",")
	}

	rows := make([][5]string, 0, len(result.Discovered))
	for _, r := range result.Discovered {
		capabilities := kindCapabilities[r.GeneratorKind]
		if strings.TrimSpace(capabilities) == "" {
			capabilities = "-"
		}
		rows = append(rows, [5]string{
			r.ResourceType,
			r.ResourceName,
			r.GeneratorKind,
			r.GeneratorProfile,
			capabilities,
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i][0] != rows[j][0] {
			return rows[i][0] < rows[j][0]
		}
		return rows[i][1] < rows[j][1]
	})

	fmt.Println("Resource Type\tResource Name\tGenerator Kind\tProfile\tCapabilities")
	for _, row := range rows {
		fmt.Printf("%s\t%s\t%s\t%s\t%s\n", row[0], row[1], row[2], row[3], row[4])
	}
}

func printImportTripleSummary(result gitopsflow.ImportFlowResult) {
	if len(result.Contracts) == 0 {
		return
	}

	dryInputsByGeneratorID := map[string]int{}
	for _, dry := range result.DryInputs {
		dryInputsByGeneratorID[dry.GeneratorID]++
	}

	wetTargetsByGeneratorID := map[string]int{}
	for _, wet := range result.WetManifestTargets {
		wetTargetsByGeneratorID[wet.GeneratorID]++
	}

	inversePatchesBySourceRef := map[string]int{}
	reviewRequiredBySourceRef := map[string]int{}
	editableByTotals := map[string]int{}
	for _, plan := range result.InversePlans {
		for _, patch := range plan.Patches {
			inversePatchesBySourceRef[plan.SourceRef]++
			if patch.RequiresReview {
				reviewRequiredBySourceRef[plan.SourceRef]++
			}
			if owner := strings.TrimSpace(patch.EditableBy); owner != "" {
				editableByTotals[owner]++
			}
		}
	}

	totalReviewRequired := 0
	for _, v := range reviewRequiredBySourceRef {
		totalReviewRequired += v
	}

	contracts := append([]model.GeneratorContract(nil), result.Contracts...)
	sort.Slice(contracts, func(i, j int) bool {
		if contracts[i].Kind != contracts[j].Kind {
			return contracts[i].Kind < contracts[j].Kind
		}
		return contracts[i].GeneratorID < contracts[j].GeneratorID
	})

	fmt.Println("Generator Kind\tProfile\tCapabilities\tDry Inputs\tWet Targets\tInverse Patches\tReview Required")
	for _, c := range contracts {
		fmt.Printf("%s\t%s\t%s\t%d\t%d\t%d\t%d\n",
			c.Kind,
			c.Profile,
			strings.Join(c.Capabilities, ","),
			dryInputsByGeneratorID[c.GeneratorID],
			wetTargetsByGeneratorID[c.GeneratorID],
			inversePatchesBySourceRef[c.SourcePath],
			reviewRequiredBySourceRef[c.SourcePath],
		)
	}

	if len(editableByTotals) == 0 {
		printSwampWorkflowSummary(result.Provenance)
		return
	}
	owners := make([]string, 0, len(editableByTotals))
	for owner := range editableByTotals {
		owners = append(owners, owner)
	}
	sort.Strings(owners)

	fmt.Printf("Review required patches: %d\n", totalReviewRequired)
	fmt.Println("Patch owner\tCount")
	for _, owner := range owners {
		fmt.Printf("%s\t%d\n", owner, editableByTotals[owner])
	}

	printSwampWorkflowSummary(result.Provenance)
	printOpsWorkflowSummary(result.Provenance)
}

func printSwampWorkflowSummary(provenance []model.ProvenanceRecord) {
	type swampSummaryRow struct {
		generatorName              string
		profile                    string
		workflowCount              int
		stepCount                  int
		modelRefCount              int
		missingRequiredCount       int
		unapprovedModelCount       int
		unapprovedModelMethodCount int
	}

	rows := make([]swampSummaryRow, 0, len(provenance))
	for _, prov := range provenance {
		if prov.SwampWorkflow == nil {
			continue
		}
		analysis := prov.SwampWorkflow
		rows = append(rows, swampSummaryRow{
			generatorName:              prov.GeneratorName,
			profile:                    prov.GeneratorProfile,
			workflowCount:              len(analysis.WorkflowPaths),
			stepCount:                  len(analysis.StepNames),
			modelRefCount:              len(analysis.ModelRefs),
			missingRequiredCount:       len(analysis.MissingRequiredSteps),
			unapprovedModelCount:       len(analysis.UnapprovedModels),
			unapprovedModelMethodCount: len(analysis.UnapprovedModelMethods),
		})
	}
	if len(rows) == 0 {
		return
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].profile != rows[j].profile {
			return rows[i].profile < rows[j].profile
		}
		return rows[i].generatorName < rows[j].generatorName
	})

	fmt.Println("Swamp workflow analysis")
	fmt.Println("Generator\tProfile\tWorkflows\tSteps\tModel refs\tMissing required\tUnapproved models\tUnapproved model methods")
	for _, row := range rows {
		fmt.Printf(
			"%s\t%s\t%d\t%d\t%d\t%d\t%d\t%d\n",
			row.generatorName,
			row.profile,
			row.workflowCount,
			row.stepCount,
			row.modelRefCount,
			row.missingRequiredCount,
			row.unapprovedModelCount,
			row.unapprovedModelMethodCount,
		)
	}
}

func printOpsWorkflowSummary(provenance []model.ProvenanceRecord) {
	type opsSummaryRow struct {
		generatorName         string
		profile               string
		workflowCount         int
		actionCount           int
		scheduleOverrideCount int
		approvalGateCount     int
		blockedUsedCount      int
		unapprovedActionCount int
	}

	rows := make([]opsSummaryRow, 0, len(provenance))
	for _, prov := range provenance {
		if prov.OpsWorkflow == nil {
			continue
		}
		analysis := prov.OpsWorkflow
		rows = append(rows, opsSummaryRow{
			generatorName:         prov.GeneratorName,
			profile:               prov.GeneratorProfile,
			workflowCount:         len(analysis.WorkflowPaths),
			actionCount:           len(analysis.ActionNames),
			scheduleOverrideCount: len(analysis.ScheduleOverrides),
			approvalGateCount:     len(analysis.ApprovalGates),
			blockedUsedCount:      len(analysis.BlockedActionsUsed),
			unapprovedActionCount: len(analysis.UnapprovedActions),
		})
	}
	if len(rows) == 0 {
		return
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].profile != rows[j].profile {
			return rows[i].profile < rows[j].profile
		}
		return rows[i].generatorName < rows[j].generatorName
	})

	fmt.Println("Ops workflow analysis")
	fmt.Println("Generator\tProfile\tWorkflows\tActions\tSchedule overrides\tApproval gates\tBlocked used\tUnapproved actions")
	for _, row := range rows {
		fmt.Printf(
			"%s\t%s\t%d\t%d\t%d\t%d\t%d\t%d\n",
			row.generatorName,
			row.profile,
			row.workflowCount,
			row.actionCount,
			row.scheduleOverrideCount,
			row.approvalGateCount,
			row.blockedUsedCount,
			row.unapprovedActionCount,
		)
	}
}
