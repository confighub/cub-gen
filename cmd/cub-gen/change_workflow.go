package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	bridgeflow "github.com/confighub/cub-gen/internal/bridge"
	changeflow "github.com/confighub/cub-gen/internal/change"
	gitopsflow "github.com/confighub/cub-gen/internal/gitops"
	"github.com/confighub/cub-gen/internal/model"
	"github.com/confighub/cub-gen/internal/publish"
)

type changePreviewInput = changeflow.PreviewInput
type changePreviewSummary = changeflow.PreviewSummary
type changePreviewCounts = changeflow.PreviewCounts
type changePreviewVerification = changeflow.PreviewVerification
type changePreviewResult = changeflow.PreviewResult

type changeRunDecision struct {
	State     string `json:"state"`
	Authority string `json:"authority"`
	Source    string `json:"source"`
}

type changeRunResult struct {
	Mode           string              `json:"mode"`
	Preview        changePreviewResult `json:"preview"`
	Decision       changeRunDecision   `json:"decision"`
	PromotionReady bool                `json:"promotion_ready"`
}

type changeDiffInput = changeflow.DiffInput
type changeDiffQuery = changeflow.DiffQuery
type changeDiffManifestRef = changeflow.ManifestRef
type changeDiffFieldState = changeflow.DiffFieldState
type changeDiffEntry = changeflow.DiffEntry
type changeDiffSnapshot = changeflow.DiffSnapshot
type changeDiffResult = changeflow.DiffResult
type changeRevisionDiffInput = changeflow.RevisionDiffInput
type changeRevisionDiffQuery = changeflow.RevisionDiffQuery
type changeRevisionDiffCause = changeflow.RevisionDiffCause
type changeRevisionDiffEntry = changeflow.RevisionDiffEntry
type changeRevisionDiffResult = changeflow.RevisionDiffResult

type changeRunOptions struct {
	Space            string
	Ref              string
	WhereResource    string
	HelmCLIOverrides []model.HelmCLIOverride
	Mode             string
	BaseURL          string
	Token            string
	IngestEndpoint   string
	DecisionEndpoint string
	Verifier         string
}

func runChange(args []string) error {
	if len(args) == 0 {
		printChangeUsage(os.Stderr)
		return errors.New("change subcommand required")
	}

	switch args[0] {
	case "help", "-h", "--help":
		printChangeUsage(os.Stdout)
		return nil
	case "preview":
		return runChangePreview(args[1:])
	case "run":
		return runChangeRun(args[1:])
	case "diff":
		return runChangeDiff(args[1:])
	case "revision-diff":
		return runChangeRevisionDiff(args[1:])
	case "impact":
		return runChangeImpact(args[1:])
	case "explain":
		return runChangeExplain(args[1:])
	case "api":
		return runChangeAPI(args[1:])
	default:
		printChangeUsage(os.Stderr)
		return fmt.Errorf("unknown change subcommand: %s", args[0])
	}
}

func runChangePreview(args []string) error {
	fs := flag.NewFlagSet("change preview", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	space := fs.String("space", "default", "ConfigHub space label")
	ref := fs.String("ref", "HEAD", "Git ref label to include in output")
	whereResource := fs.String("where-resource", "", "Additional resource filter expression")
	out := fs.String("out", "-", "Output file path, or '-' for stdout")
	verifier := fs.String("verifier", "cub-gen", "Verifier identity label")
	jsonOut := fs.Bool("json", true, "Output JSON")
	pretty := fs.Bool("pretty", true, "Pretty-print JSON output")
	overrideFlags := addHelmCLIOverrideFlags(fs)
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if err := requireJSONOutput("change preview", *jsonOut); err != nil {
		return err
	}
	helmCLIOverrides, err := overrideFlags.parse()
	if err != nil {
		return err
	}

	targetSlug, renderTargetSlug, err := resolveTargetPairArgs(fs, "usage: cub-gen change preview [flags] <target-path> [<render-target-path>]")
	if err != nil {
		return err
	}

	result, _, _, err := buildChangePreviewResult(
		targetSlug,
		renderTargetSlug,
		*space,
		*ref,
		*whereResource,
		*verifier,
		helmCLIOverrides,
	)
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

func runChangeRun(args []string) error {
	fs := flag.NewFlagSet("change run", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	space := fs.String("space", "default", "ConfigHub space label")
	ref := fs.String("ref", "HEAD", "Git ref label to include in output")
	whereResource := fs.String("where-resource", "", "Additional resource filter expression")
	mode := fs.String("mode", "local", "Execution mode: local or connected")
	baseURL := fs.String("base-url", "", "ConfigHub base URL (connected mode)")
	token := fs.String("token", "", "ConfigHub token (connected mode)")
	ingestEndpoint := fs.String("ingest-endpoint", "", "Override bridge ingest endpoint path (connected mode)")
	decisionEndpoint := fs.String("decision-endpoint", "", "Override bridge decision query endpoint path (connected mode)")
	out := fs.String("out", "-", "Output file path, or '-' for stdout")
	verifier := fs.String("verifier", "cub-gen", "Verifier identity label")
	jsonOut := fs.Bool("json", true, "Output JSON")
	pretty := fs.Bool("pretty", true, "Pretty-print JSON output")
	overrideFlags := addHelmCLIOverrideFlags(fs)
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if err := requireJSONOutput("change run", *jsonOut); err != nil {
		return err
	}
	helmCLIOverrides, err := overrideFlags.parse()
	if err != nil {
		return err
	}

	targetSlug, renderTargetSlug, err := resolveTargetPairArgs(fs, "usage: cub-gen change run [flags] <target-path> [<render-target-path>]")
	if err != nil {
		return err
	}
	runMode := strings.ToLower(strings.TrimSpace(*mode))
	if runMode != "local" && runMode != "connected" {
		return errors.New("change run --mode must be local|connected")
	}

	result, _, err := executeChangeRun(targetSlug, renderTargetSlug, changeRunOptions{
		Space:            *space,
		Ref:              *ref,
		WhereResource:    *whereResource,
		Mode:             runMode,
		BaseURL:          *baseURL,
		Token:            *token,
		IngestEndpoint:   *ingestEndpoint,
		DecisionEndpoint: *decisionEndpoint,
		Verifier:         *verifier,
		HelmCLIOverrides: helmCLIOverrides,
	})
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

func runChangeDiff(args []string) error {
	fs := flag.NewFlagSet("change diff", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	space := fs.String("space", "default", "ConfigHub space label")
	beforeRef := fs.String("before-ref", "", "Git ref to render as the before side")
	afterRef := fs.String("after-ref", "", "Git ref to render as the after side")
	whereResource := fs.String("where-resource", "", "Additional resource filter expression")
	dryFilter := fs.String("dry-path", "", "Filter by DRY path")
	wetFilter := fs.String("wet-path", "", "Filter by WET path")
	ownerFilter := fs.String("owner", "", "Filter by owner")
	out := fs.String("out", "-", "Output file path, or '-' for stdout")
	pretty := fs.Bool("pretty", true, "Pretty-print JSON output")
	overrideFlags := addHelmCLIOverrideFlags(fs)
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	helmCLIOverrides, err := overrideFlags.parse()
	if err != nil {
		return err
	}
	if strings.TrimSpace(*beforeRef) == "" || strings.TrimSpace(*afterRef) == "" {
		return errors.New("change diff requires --before-ref and --after-ref")
	}

	targetSlug, renderTargetSlug, err := resolveTargetPairArgs(fs, "usage: cub-gen change diff --before-ref REF --after-ref REF [flags] <target-path> [<render-target-path>]")
	if err != nil {
		return err
	}

	result, err := buildChangeDiffResult(
		targetSlug,
		renderTargetSlug,
		*space,
		*beforeRef,
		*afterRef,
		*whereResource,
		helmCLIOverrides,
		*dryFilter,
		*wetFilter,
		*ownerFilter,
	)
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

func runChangeRevisionDiff(args []string) error {
	fs := flag.NewFlagSet("change revision-diff", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	space := fs.String("space", "default", "ConfigHub space label")
	fromRef := fs.String("from", "", "Git ref to treat as the before/source revision")
	toRef := fs.String("to", "", "Git ref to treat as the after/target revision")
	whereResource := fs.String("where-resource", "", "Additional resource filter expression")
	dryFilter := fs.String("dry-path", "", "Filter by DRY path")
	wetFilter := fs.String("wet-path", "", "Filter by WET path")
	ownerFilter := fs.String("owner", "", "Filter by owner")
	out := fs.String("out", "-", "Output file path, or '-' for stdout")
	pretty := fs.Bool("pretty", true, "Pretty-print JSON output")
	overrideFlags := addHelmCLIOverrideFlags(fs)
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	helmCLIOverrides, err := overrideFlags.parse()
	if err != nil {
		return err
	}
	if strings.TrimSpace(*fromRef) == "" || strings.TrimSpace(*toRef) == "" {
		return errors.New("change revision-diff requires --from and --to")
	}

	targetSlug, renderTargetSlug, err := resolveTargetPairArgs(fs, "usage: cub-gen change revision-diff --from REF --to REF [flags] <target-path> [<render-target-path>]")
	if err != nil {
		return err
	}

	result, err := buildChangeRevisionDiffResult(
		targetSlug,
		renderTargetSlug,
		*space,
		*fromRef,
		*toRef,
		*whereResource,
		helmCLIOverrides,
		*dryFilter,
		*wetFilter,
		*ownerFilter,
	)
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

func executeChangeRun(targetSlug, renderTargetSlug string, opts changeRunOptions) (changeRunResult, []model.ProvenanceRecord, error) {
	runMode := strings.ToLower(strings.TrimSpace(opts.Mode))
	if runMode != "local" && runMode != "connected" {
		return changeRunResult{}, nil, errors.New("change run --mode must be local|connected")
	}
	verifier := strings.TrimSpace(opts.Verifier)
	if verifier == "" {
		verifier = "cub-gen"
	}

	preview, bundle, imported, err := buildChangePreviewResult(
		targetSlug,
		renderTargetSlug,
		opts.Space,
		opts.Ref,
		opts.WhereResource,
		verifier,
		opts.HelmCLIOverrides,
	)
	if err != nil {
		return changeRunResult{}, nil, err
	}

	decision := changeRunDecision{
		State:     "ALLOW",
		Authority: verifier,
		Source:    "local-preview",
	}
	promotionReady := true

	if runMode == "connected" {
		resolvedBaseURL := strings.TrimSpace(opts.BaseURL)
		if resolvedBaseURL == "" {
			resolvedBaseURL = strings.TrimSpace(os.Getenv("CONFIGHUB_BASE_URL"))
		}
		if resolvedBaseURL == "" {
			return changeRunResult{}, nil, errors.New("change run --mode connected requires --base-url or CONFIGHUB_BASE_URL")
		}

		resolvedToken := strings.TrimSpace(opts.Token)
		if resolvedToken == "" {
			resolvedToken = strings.TrimSpace(os.Getenv("CONFIGHUB_TOKEN"))
		}

		ingestRes, err := bridgeflow.IngestBundle(context.Background(), bridgeflow.Client{
			BaseURL:      resolvedBaseURL,
			BearerToken:  resolvedToken,
			EndpointPath: strings.TrimSpace(opts.IngestEndpoint),
		}, bundle)
		if err != nil {
			return changeRunResult{}, nil, fmt.Errorf("connected ingest: %w", err)
		}

		decisionRec, err := bridgeflow.QueryDecisionByChangeID(context.Background(), bridgeflow.DecisionClient{
			BaseURL:      resolvedBaseURL,
			BearerToken:  resolvedToken,
			EndpointPath: strings.TrimSpace(opts.DecisionEndpoint),
		}, preview.Change.ChangeID)
		if err != nil {
			return changeRunResult{}, nil, fmt.Errorf("connected decision query: %w", err)
		}

		authority := strings.TrimSpace(decisionRec.ApprovedBy)
		if authority == "" {
			authority = strings.TrimSpace(decisionRec.PolicyDecisionRef)
		}
		if authority == "" {
			authority = "confighub-policy"
		}

		decision = changeRunDecision{
			State:     string(decisionRec.State),
			Authority: authority,
			Source:    "confighub-backend",
		}
		if decision.State != "ALLOW" {
			promotionReady = false
		}
		if strings.TrimSpace(ingestRes.ChangeID) == "" {
			promotionReady = false
		}
	}

	result := changeRunResult{
		Mode:           runMode,
		Preview:        preview,
		Decision:       decision,
		PromotionReady: promotionReady,
	}
	return result, imported.Provenance, nil
}

func buildChangeDiffResult(
	targetSlug, renderTargetSlug, space, beforeRef, afterRef, whereResource string,
	helmCLIOverrides []model.HelmCLIOverride,
	dryFilter, wetFilter, ownerFilter string,
) (changeDiffResult, error) {
	return changeflow.BuildDiffResult(targetSlug, renderTargetSlug, changeflow.DiffOptions{
		Space:            space,
		BeforeRef:        beforeRef,
		AfterRef:         afterRef,
		WhereResource:    whereResource,
		HelmCLIOverrides: helmCLIOverrides,
		DryPathFilter:    dryFilter,
		WetPathFilter:    wetFilter,
		OwnerFilter:      ownerFilter,
	})
}

func buildChangeRevisionDiffResult(
	targetSlug, renderTargetSlug, space, fromRef, toRef, whereResource string,
	helmCLIOverrides []model.HelmCLIOverride,
	dryFilter, wetFilter, ownerFilter string,
) (changeRevisionDiffResult, error) {
	return changeflow.BuildRevisionDiffResult(targetSlug, renderTargetSlug, changeflow.RevisionDiffOptions{
		Space:            space,
		FromRef:          fromRef,
		ToRef:            toRef,
		WhereResource:    whereResource,
		HelmCLIOverrides: helmCLIOverrides,
		DryPathFilter:    dryFilter,
		WetPathFilter:    wetFilter,
		OwnerFilter:      ownerFilter,
	})
}

func buildChangePreviewResult(
	targetSlug, renderTargetSlug, space, ref, whereResource, verifier string,
	helmCLIOverrides []model.HelmCLIOverride,
) (changePreviewResult, publish.ChangeBundle, gitopsflow.ImportFlowResult, error) {
	return changeflow.BuildPreviewResult(targetSlug, renderTargetSlug, changeflow.PreviewOptions{
		Space:            space,
		Ref:              ref,
		WhereResource:    whereResource,
		Verifier:         verifier,
		HelmCLIOverrides: helmCLIOverrides,
	})
}

func requireJSONOutput(command string, enabled bool) error {
	if enabled {
		return nil
	}
	return fmt.Errorf("cub-gen %s only supports JSON output; omit --json or pass --json=true", command)
}

func resolveTargetPairArgs(fs *flag.FlagSet, usage string) (string, string, error) {
	switch fs.NArg() {
	case 1:
		target := fs.Arg(0)
		return target, target, nil
	case 2:
		return fs.Arg(0), fs.Arg(1), nil
	default:
		return "", "", errors.New(usage)
	}
}
