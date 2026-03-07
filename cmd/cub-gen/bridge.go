package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/confighub/cub-gen/internal/attest"
	bridgeflow "github.com/confighub/cub-gen/internal/bridge"
	"github.com/confighub/cub-gen/internal/publish"
)

func runBridge(args []string) error {
	if len(args) == 0 {
		printBridgeUsage(os.Stderr)
		return errors.New("bridge subcommand required")
	}

	switch args[0] {
	case "help", "-h", "--help":
		printBridgeUsage(os.Stdout)
		return nil
	case "ingest":
		return runBridgeIngest(args[1:])
	case "decision":
		return runBridgeDecision(args[1:])
	case "promote":
		return runBridgePromote(args[1:])
	default:
		printBridgeUsage(os.Stderr)
		return fmt.Errorf("unknown bridge subcommand: %s", args[0])
	}
}

func runBridgeIngest(args []string) error {
	fs := flag.NewFlagSet("bridge ingest", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	in := fs.String("in", "-", "Bundle JSON input path, or '-' for stdin")
	baseURL := fs.String("base-url", "", "ConfigHub base URL")
	token := fs.String("token", "", "Optional bearer token")
	endpoint := fs.String("endpoint", "", "Optional ingest endpoint path override")
	jsonOut := fs.Bool("json", true, "Output JSON")
	pretty := fs.Bool("pretty", true, "Pretty-print JSON output")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("usage: cub-gen bridge ingest [flags]")
	}
	if strings.TrimSpace(*baseURL) == "" {
		return errors.New("bridge ingest requires --base-url")
	}

	var bundle publish.ChangeBundle
	if err := readJSONInput(*in, &bundle); err != nil {
		return fmt.Errorf("read bundle json: %w", err)
	}

	res, err := bridgeflow.IngestBundle(context.Background(), bridgeflow.Client{
		BaseURL:      strings.TrimSpace(*baseURL),
		BearerToken:  strings.TrimSpace(*token),
		EndpointPath: strings.TrimSpace(*endpoint),
	}, bundle)
	if err != nil {
		return err
	}

	if *jsonOut {
		return writeJSON(os.Stdout, res, *pretty)
	}
	fmt.Printf("Bridge ingest OK: change_id=%s status=%s artifact_id=%s idempotent=%t\n", res.ChangeID, res.Status, res.ArtifactID, res.Idempotent)
	return nil
}

func runBridgeDecision(args []string) error {
	if len(args) == 0 {
		printBridgeDecisionUsage(os.Stderr)
		return errors.New("bridge decision subcommand required")
	}

	switch args[0] {
	case "help", "-h", "--help":
		printBridgeDecisionUsage(os.Stdout)
		return nil
	case "create":
		return runBridgeDecisionCreate(args[1:])
	case "attach":
		return runBridgeDecisionAttach(args[1:])
	case "apply":
		return runBridgeDecisionApply(args[1:])
	case "query":
		return runBridgeDecisionQuery(args[1:])
	default:
		printBridgeDecisionUsage(os.Stderr)
		return fmt.Errorf("unknown bridge decision subcommand: %s", args[0])
	}
}

func runBridgeDecisionCreate(args []string) error {
	fs := flag.NewFlagSet("bridge decision create", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	ingestPath := fs.String("ingest", "-", "Bridge ingest result JSON path, or '-' for stdin")
	out := fs.String("out", "-", "Decision JSON output path, or '-' for stdout")
	atRaw := fs.String("at", "", "Optional RFC3339 timestamp override")
	pretty := fs.Bool("pretty", true, "Pretty-print JSON output")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("usage: cub-gen bridge decision create [flags]")
	}

	var ingest bridgeflow.IngestResult
	if err := readJSONInput(*ingestPath, &ingest); err != nil {
		return fmt.Errorf("read ingest result json: %w", err)
	}
	at, err := parseAt(*atRaw)
	if err != nil {
		return err
	}

	rec, err := bridgeflow.NewDecisionRecord(ingest, at)
	if err != nil {
		return err
	}
	return writeJSONOutput(*out, rec, *pretty)
}

func runBridgeDecisionAttach(args []string) error {
	fs := flag.NewFlagSet("bridge decision attach", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	decisionPath := fs.String("decision", "-", "Decision JSON input path, or '-' for stdin")
	attPath := fs.String("attestation", "", "Attestation JSON input path")
	out := fs.String("out", "-", "Decision JSON output path, or '-' for stdout")
	atRaw := fs.String("at", "", "Optional RFC3339 timestamp override")
	pretty := fs.Bool("pretty", true, "Pretty-print JSON output")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("usage: cub-gen bridge decision attach [flags]")
	}
	if strings.TrimSpace(*attPath) == "" {
		return errors.New("bridge decision attach requires --attestation")
	}

	var rec bridgeflow.DecisionRecord
	if err := readJSONInput(*decisionPath, &rec); err != nil {
		return fmt.Errorf("read decision json: %w", err)
	}
	var attRec attest.Record
	if err := readJSONInput(*attPath, &attRec); err != nil {
		return fmt.Errorf("read attestation json: %w", err)
	}
	at, err := parseAt(*atRaw)
	if err != nil {
		return err
	}

	updated, err := bridgeflow.AttachAttestation(rec, attRec, at)
	if err != nil {
		return err
	}
	return writeJSONOutput(*out, updated, *pretty)
}

func runBridgeDecisionApply(args []string) error {
	fs := flag.NewFlagSet("bridge decision apply", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	decisionPath := fs.String("decision", "-", "Decision JSON input path, or '-' for stdin")
	out := fs.String("out", "-", "Decision JSON output path, or '-' for stdout")
	stateRaw := fs.String("state", "", "Decision state: ALLOW|ESCALATE|BLOCK")
	reason := fs.String("reason", "", "Decision reason")
	approvedBy := fs.String("approved-by", "", "Approver identity")
	policyRef := fs.String("policy-ref", "", "Policy decision reference")
	atRaw := fs.String("at", "", "Optional RFC3339 timestamp override")
	pretty := fs.Bool("pretty", true, "Pretty-print JSON output")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("usage: cub-gen bridge decision apply [flags]")
	}

	state, err := parseDecisionState(*stateRaw)
	if err != nil {
		return err
	}
	if state != bridgeflow.DecisionStateAllow && state != bridgeflow.DecisionStateEscalate && state != bridgeflow.DecisionStateBlock {
		return errors.New("bridge decision apply --state must be ALLOW|ESCALATE|BLOCK")
	}

	var rec bridgeflow.DecisionRecord
	if err := readJSONInput(*decisionPath, &rec); err != nil {
		return fmt.Errorf("read decision json: %w", err)
	}
	at, err := parseAt(*atRaw)
	if err != nil {
		return err
	}

	updated, err := bridgeflow.ApplyDecision(rec, bridgeflow.DecisionRequest{
		State:             state,
		PolicyDecisionRef: strings.TrimSpace(*policyRef),
		ApprovedBy:        strings.TrimSpace(*approvedBy),
		DecisionReason:    strings.TrimSpace(*reason),
	}, at)
	if err != nil {
		return err
	}
	return writeJSONOutput(*out, updated, *pretty)
}

func runBridgeDecisionQuery(args []string) error {
	fs := flag.NewFlagSet("bridge decision query", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	baseURL := fs.String("base-url", "", "ConfigHub base URL")
	changeID := fs.String("change-id", "", "Change ID to query")
	token := fs.String("token", "", "Optional bearer token")
	endpoint := fs.String("endpoint", "", "Optional decision endpoint path override")
	pretty := fs.Bool("pretty", true, "Pretty-print JSON output")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("usage: cub-gen bridge decision query [flags]")
	}
	if strings.TrimSpace(*baseURL) == "" {
		return errors.New("bridge decision query requires --base-url")
	}
	if strings.TrimSpace(*changeID) == "" {
		return errors.New("bridge decision query requires --change-id")
	}

	rec, err := bridgeflow.QueryDecisionByChangeID(context.Background(), bridgeflow.DecisionClient{
		BaseURL:      strings.TrimSpace(*baseURL),
		BearerToken:  strings.TrimSpace(*token),
		EndpointPath: strings.TrimSpace(*endpoint),
	}, strings.TrimSpace(*changeID))
	if err != nil {
		return err
	}
	return writeJSON(os.Stdout, rec, *pretty)
}

func runBridgePromote(args []string) error {
	if len(args) == 0 {
		printBridgePromoteUsage(os.Stderr)
		return errors.New("bridge promote subcommand required")
	}

	switch args[0] {
	case "help", "-h", "--help":
		printBridgePromoteUsage(os.Stdout)
		return nil
	case "init":
		return runBridgePromoteInit(args[1:])
	case "govern":
		return runBridgePromoteGovern(args[1:])
	case "verify":
		return runBridgePromoteVerify(args[1:])
	case "open":
		return runBridgePromoteOpen(args[1:])
	case "approve":
		return runBridgePromoteApprove(args[1:])
	case "merge":
		return runBridgePromoteMerge(args[1:])
	default:
		printBridgePromoteUsage(os.Stderr)
		return fmt.Errorf("unknown bridge promote subcommand: %s", args[0])
	}
}

func runBridgePromoteInit(args []string) error {
	fs := flag.NewFlagSet("bridge promote init", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	changeID := fs.String("change-id", "", "Change ID")
	appPRRepo := fs.String("app-pr-repo", "", "App PR repository")
	appPRNumber := fs.Int("app-pr-number", 0, "App PR number")
	appPRURL := fs.String("app-pr-url", "", "App PR URL")
	appPRSHA := fs.String("app-pr-sha", "", "App PR commit SHA")
	mrID := fs.String("mr-id", "", "ConfigHub merge request ID")
	mrURL := fs.String("mr-url", "", "ConfigHub merge request URL")
	mrStatus := fs.String("mr-status", "", "Optional ConfigHub merge request status")
	out := fs.String("out", "-", "Promotion flow JSON output path, or '-' for stdout")
	atRaw := fs.String("at", "", "Optional RFC3339 timestamp override")
	pretty := fs.Bool("pretty", true, "Pretty-print JSON output")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("usage: cub-gen bridge promote init [flags]")
	}

	at, err := parseAt(*atRaw)
	if err != nil {
		return err
	}
	link, err := bridgeflow.NewReviewLink(strings.TrimSpace(*changeID), bridgeflow.PullRequestRef{
		Repo:      strings.TrimSpace(*appPRRepo),
		Number:    *appPRNumber,
		URL:       strings.TrimSpace(*appPRURL),
		CommitSHA: strings.TrimSpace(*appPRSHA),
	}, bridgeflow.MergeRequestRef{
		ID:     strings.TrimSpace(*mrID),
		URL:    strings.TrimSpace(*mrURL),
		Status: strings.TrimSpace(*mrStatus),
	}, at)
	if err != nil {
		return err
	}
	flow, err := bridgeflow.NewPromotionFlow(link, at)
	if err != nil {
		return err
	}
	return writeJSONOutput(*out, flow, *pretty)
}

func runBridgePromoteGovern(args []string) error {
	fs := flag.NewFlagSet("bridge promote govern", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	flowPath := fs.String("flow", "-", "Promotion flow JSON input path, or '-' for stdin")
	stateRaw := fs.String("state", "", "Governance state: ALLOW|ESCALATE|BLOCK")
	decisionRef := fs.String("decision-ref", "", "Governance decision reference")
	out := fs.String("out", "-", "Promotion flow JSON output path, or '-' for stdout")
	atRaw := fs.String("at", "", "Optional RFC3339 timestamp override")
	pretty := fs.Bool("pretty", true, "Pretty-print JSON output")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("usage: cub-gen bridge promote govern [flags]")
	}

	state, err := parseDecisionState(*stateRaw)
	if err != nil {
		return err
	}
	if state != bridgeflow.DecisionStateAllow && state != bridgeflow.DecisionStateEscalate && state != bridgeflow.DecisionStateBlock {
		return errors.New("bridge promote govern --state must be ALLOW|ESCALATE|BLOCK")
	}

	var flow bridgeflow.PromotionFlow
	if err := readJSONInput(*flowPath, &flow); err != nil {
		return fmt.Errorf("read promotion flow json: %w", err)
	}
	at, err := parseAt(*atRaw)
	if err != nil {
		return err
	}
	updated, err := bridgeflow.ApplyGovernanceDecision(flow, state, strings.TrimSpace(*decisionRef), at)
	if err != nil {
		return err
	}
	return writeJSONOutput(*out, updated, *pretty)
}

func runBridgePromoteVerify(args []string) error {
	fs := flag.NewFlagSet("bridge promote verify", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	flowPath := fs.String("flow", "-", "Promotion flow JSON input path, or '-' for stdin")
	out := fs.String("out", "-", "Promotion flow JSON output path, or '-' for stdout")
	atRaw := fs.String("at", "", "Optional RFC3339 timestamp override")
	pretty := fs.Bool("pretty", true, "Pretty-print JSON output")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("usage: cub-gen bridge promote verify [flags]")
	}

	var flow bridgeflow.PromotionFlow
	if err := readJSONInput(*flowPath, &flow); err != nil {
		return fmt.Errorf("read promotion flow json: %w", err)
	}
	at, err := parseAt(*atRaw)
	if err != nil {
		return err
	}
	updated, err := bridgeflow.MarkDeploymentVerified(flow, at)
	if err != nil {
		return err
	}
	return writeJSONOutput(*out, updated, *pretty)
}

func runBridgePromoteOpen(args []string) error {
	fs := flag.NewFlagSet("bridge promote open", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	flowPath := fs.String("flow", "-", "Promotion flow JSON input path, or '-' for stdin")
	repo := fs.String("repo", "", "Promotion PR repository")
	number := fs.Int("number", 0, "Promotion PR number")
	url := fs.String("url", "", "Promotion PR URL")
	sha := fs.String("sha", "", "Promotion PR commit SHA")
	out := fs.String("out", "-", "Promotion flow JSON output path, or '-' for stdout")
	atRaw := fs.String("at", "", "Optional RFC3339 timestamp override")
	pretty := fs.Bool("pretty", true, "Pretty-print JSON output")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("usage: cub-gen bridge promote open [flags]")
	}

	var flow bridgeflow.PromotionFlow
	if err := readJSONInput(*flowPath, &flow); err != nil {
		return fmt.Errorf("read promotion flow json: %w", err)
	}
	at, err := parseAt(*atRaw)
	if err != nil {
		return err
	}
	updated, err := bridgeflow.OpenPromotionPR(flow, bridgeflow.PullRequestRef{
		Repo:      strings.TrimSpace(*repo),
		Number:    *number,
		URL:       strings.TrimSpace(*url),
		CommitSHA: strings.TrimSpace(*sha),
	}, at)
	if err != nil {
		return err
	}
	return writeJSONOutput(*out, updated, *pretty)
}

func runBridgePromoteApprove(args []string) error {
	fs := flag.NewFlagSet("bridge promote approve", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	flowPath := fs.String("flow", "-", "Promotion flow JSON input path, or '-' for stdin")
	approvedBy := fs.String("by", "", "Platform review approver")
	out := fs.String("out", "-", "Promotion flow JSON output path, or '-' for stdout")
	atRaw := fs.String("at", "", "Optional RFC3339 timestamp override")
	pretty := fs.Bool("pretty", true, "Pretty-print JSON output")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("usage: cub-gen bridge promote approve [flags]")
	}

	var flow bridgeflow.PromotionFlow
	if err := readJSONInput(*flowPath, &flow); err != nil {
		return fmt.Errorf("read promotion flow json: %w", err)
	}
	at, err := parseAt(*atRaw)
	if err != nil {
		return err
	}
	updated, err := bridgeflow.ApprovePlatformReview(flow, strings.TrimSpace(*approvedBy), at)
	if err != nil {
		return err
	}
	return writeJSONOutput(*out, updated, *pretty)
}

func runBridgePromoteMerge(args []string) error {
	fs := flag.NewFlagSet("bridge promote merge", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	flowPath := fs.String("flow", "-", "Promotion flow JSON input path, or '-' for stdin")
	mergedBy := fs.String("by", "", "Promotion merge actor")
	out := fs.String("out", "-", "Promotion flow JSON output path, or '-' for stdout")
	atRaw := fs.String("at", "", "Optional RFC3339 timestamp override")
	pretty := fs.Bool("pretty", true, "Pretty-print JSON output")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("usage: cub-gen bridge promote merge [flags]")
	}

	var flow bridgeflow.PromotionFlow
	if err := readJSONInput(*flowPath, &flow); err != nil {
		return fmt.Errorf("read promotion flow json: %w", err)
	}
	at, err := parseAt(*atRaw)
	if err != nil {
		return err
	}
	updated, err := bridgeflow.MergePromotionPR(flow, strings.TrimSpace(*mergedBy), at)
	if err != nil {
		return err
	}
	return writeJSONOutput(*out, updated, *pretty)
}

func parseDecisionState(raw string) (bridgeflow.DecisionState, error) {
	state := strings.ToUpper(strings.TrimSpace(raw))
	switch state {
	case string(bridgeflow.DecisionStateIngested):
		return bridgeflow.DecisionStateIngested, nil
	case string(bridgeflow.DecisionStateAttested):
		return bridgeflow.DecisionStateAttested, nil
	case string(bridgeflow.DecisionStateAllow):
		return bridgeflow.DecisionStateAllow, nil
	case string(bridgeflow.DecisionStateEscalate):
		return bridgeflow.DecisionStateEscalate, nil
	case string(bridgeflow.DecisionStateBlock):
		return bridgeflow.DecisionStateBlock, nil
	default:
		return "", fmt.Errorf("unsupported decision state %q", raw)
	}
}

func parseAt(raw string) (time.Time, error) {
	if strings.TrimSpace(raw) == "" {
		return time.Time{}, nil
	}
	at, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse --at as RFC3339: %w", err)
	}
	return at, nil
}

func readJSONInput(path string, out any) error {
	raw := strings.TrimSpace(path)
	if raw == "" {
		raw = "-"
	}

	var data []byte
	var err error
	if raw == "-" {
		data, err = io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("read stdin: %w", err)
		}
	} else {
		data, err = os.ReadFile(raw)
		if err != nil {
			return fmt.Errorf("read file: %w", err)
		}
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("parse json: %w", err)
	}
	return nil
}

func writeJSONOutput(path string, v any, pretty bool) error {
	raw := strings.TrimSpace(path)
	if raw == "" || raw == "-" {
		return writeJSON(os.Stdout, v, pretty)
	}

	f, err := os.Create(raw)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer func() {
		_ = f.Close()
	}()
	return writeJSON(f, v, pretty)
}

func printBridgeUsage(out io.Writer) {
	fmt.Fprintln(out, "cub-gen bridge: ConfigHub bridge flow commands (ingest, decision, promotion)")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Usage:")
	fmt.Fprintln(out, "  cub-gen bridge ingest [--in FILE|-] --base-url URL [--token TOKEN] [--endpoint PATH] [--json] [--pretty]")
	fmt.Fprintln(out, "  cub-gen bridge decision <create|attach|apply|query> [flags]")
	fmt.Fprintln(out, "  cub-gen bridge promote <init|govern|verify|open|approve|merge> [flags]")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Examples:")
	fmt.Fprintln(out, "  cub-gen bridge ingest --in bundle.json --base-url https://confighub.example")
	fmt.Fprintln(out, "  cub-gen bridge decision create --ingest ingest-result.json")
	fmt.Fprintln(out, "  cub-gen bridge decision attach --decision decision.json --attestation attestation.json")
	fmt.Fprintln(out, "  cub-gen bridge decision apply --decision decision.json --state ALLOW --approved-by platform-admin --reason \"policy checks passed\"")
	fmt.Fprintln(out, "  cub-gen bridge decision query --base-url https://confighub.example --change-id chg_123")
	fmt.Fprintln(out, "  cub-gen bridge promote init --change-id chg_123 --app-pr-repo github.com/confighub/apps --app-pr-number 42 --app-pr-url https://github.com/confighub/apps/pull/42 --mr-id mr_123 --mr-url https://confighub.example/mr/123")
}

func printBridgeDecisionUsage(out io.Writer) {
	fmt.Fprintln(out, "cub-gen bridge decision: governed decision-state commands")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Usage:")
	fmt.Fprintln(out, "  cub-gen bridge decision create --ingest FILE|- [--out FILE|-] [--at RFC3339] [--pretty]")
	fmt.Fprintln(out, "  cub-gen bridge decision attach --decision FILE|- --attestation FILE [--out FILE|-] [--at RFC3339] [--pretty]")
	fmt.Fprintln(out, "  cub-gen bridge decision apply --decision FILE|- --state ALLOW|ESCALATE|BLOCK --reason TEXT [--approved-by NAME|--policy-ref REF] [--out FILE|-] [--at RFC3339] [--pretty]")
	fmt.Fprintln(out, "  cub-gen bridge decision query --base-url URL --change-id ID [--token TOKEN] [--endpoint PATH] [--pretty]")
}

func printBridgePromoteUsage(out io.Writer) {
	fmt.Fprintln(out, "cub-gen bridge promote: PR<->MR and upstream DRY promotion flow commands")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Usage:")
	fmt.Fprintln(out, "  cub-gen bridge promote init --change-id ID --app-pr-repo REPO --app-pr-number N --app-pr-url URL --mr-id ID --mr-url URL [--app-pr-sha SHA] [--mr-status STATUS] [--out FILE|-] [--at RFC3339] [--pretty]")
	fmt.Fprintln(out, "  cub-gen bridge promote govern --flow FILE|- --state ALLOW|ESCALATE|BLOCK [--decision-ref REF] [--out FILE|-] [--at RFC3339] [--pretty]")
	fmt.Fprintln(out, "  cub-gen bridge promote verify --flow FILE|- [--out FILE|-] [--at RFC3339] [--pretty]")
	fmt.Fprintln(out, "  cub-gen bridge promote open --flow FILE|- --repo REPO --number N --url URL [--sha SHA] [--out FILE|-] [--at RFC3339] [--pretty]")
	fmt.Fprintln(out, "  cub-gen bridge promote approve --flow FILE|- --by NAME [--out FILE|-] [--at RFC3339] [--pretty]")
	fmt.Fprintln(out, "  cub-gen bridge promote merge --flow FILE|- --by NAME [--out FILE|-] [--at RFC3339] [--pretty]")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Examples:")
	fmt.Fprintln(out, "  cub-gen bridge promote govern --flow flow.json --state ALLOW --decision-ref decision_123")
	fmt.Fprintln(out, "  cub-gen bridge promote verify --flow flow.json")
	fmt.Fprintln(out, "  cub-gen bridge promote open --flow flow.json --repo github.com/confighub/platform-dry --number 7 --url https://github.com/confighub/platform-dry/pull/7")
	fmt.Fprintln(out, "  cub-gen bridge promote approve --flow flow.json --by platform-owner")
	fmt.Fprintln(out, "  cub-gen bridge promote merge --flow flow.json --by platform-owner")
}
