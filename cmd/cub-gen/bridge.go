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
	case "link":
		return runBridgeLink(args[1:])
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
	resolvedBaseURL := strings.TrimSpace(*baseURL)
	if resolvedBaseURL == "" {
		resolvedBaseURL = defaultConfigHubBaseURL()
	}
	if resolvedBaseURL == "" {
		return errors.New("bridge ingest requires --base-url")
	}
	resolvedToken := strings.TrimSpace(*token)
	if resolvedToken == "" {
		resolvedToken = defaultConfigHubToken()
	}

	var bundle publish.ChangeBundle
	if err := readJSONInput(*in, &bundle); err != nil {
		return fmt.Errorf("read bundle json: %w", err)
	}

	res, err := bridgeflow.IngestBundle(context.Background(), bridgeflow.Client{
		BaseURL:      resolvedBaseURL,
		BearerToken:  resolvedToken,
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

func runBridgeLink(args []string) error {
	fs := flag.NewFlagSet("bridge link", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	bundlePath := fs.String("bundle", "", "Publish bundle JSON path, or '-' for stdin, used to derive change_id")
	changeID := fs.String("change-id", "", "Change ID override when --bundle is not supplied")
	baseURL := fs.String("base-url", "", "Optional ConfigHub base URL; when omitted, writes the local review-link record")
	token := fs.String("token", "", "Bearer token; defaults to CONFIGHUB_TOKEN when --base-url is set")
	endpoint := fs.String("endpoint", "", "Optional review-link endpoint path override")
	githubPRRepo := fs.String("github-pr-repo", "", "GitHub PR repository")
	githubPRNumber := fs.Int("github-pr-number", 0, "GitHub PR number")
	githubPRURL := fs.String("github-pr-url", "", "GitHub PR URL")
	githubPRSHA := fs.String("github-pr-sha", "", "GitHub PR commit SHA")
	appPRRepo := fs.String("app-pr-repo", "", "Alias for --github-pr-repo")
	appPRNumber := fs.Int("app-pr-number", 0, "Alias for --github-pr-number")
	appPRURL := fs.String("app-pr-url", "", "Alias for --github-pr-url")
	appPRSHA := fs.String("app-pr-sha", "", "Alias for --github-pr-sha")
	confighubMRID := fs.String("confighub-mr-id", "", "ConfigHub merge request ID")
	confighubMRURL := fs.String("confighub-mr-url", "", "ConfigHub merge request URL")
	confighubMRStatus := fs.String("confighub-mr-status", "", "Optional ConfigHub merge request status")
	mrID := fs.String("mr-id", "", "Alias for --confighub-mr-id")
	mrURL := fs.String("mr-url", "", "Alias for --confighub-mr-url")
	mrStatus := fs.String("mr-status", "", "Alias for --confighub-mr-status")
	out := fs.String("out", "-", "Review-link JSON output path, or '-' for stdout")
	atRaw := fs.String("at", "", "Optional RFC3339 timestamp override")
	pretty := fs.Bool("pretty", true, "Pretty-print JSON output")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("usage: cub-gen bridge link [flags]")
	}

	resolvedChangeID, err := resolveBridgeLinkChangeID(strings.TrimSpace(*changeID), strings.TrimSpace(*bundlePath))
	if err != nil {
		return err
	}
	repo := firstNonEmpty(*githubPRRepo, *appPRRepo)
	number := firstPositive(*githubPRNumber, *appPRNumber)
	prURL := firstNonEmpty(*githubPRURL, *appPRURL)
	prSHA := firstNonEmpty(*githubPRSHA, *appPRSHA)
	if strings.TrimSpace(repo) == "" || number <= 0 || strings.TrimSpace(prURL) == "" {
		return errors.New("bridge link requires --github-pr-repo, --github-pr-number, and --github-pr-url (or --app-pr-* aliases)")
	}

	linkMRID := firstNonEmpty(*confighubMRID, *mrID)
	linkMRURL := firstNonEmpty(*confighubMRURL, *mrURL)
	linkMRStatus := firstNonEmpty(*confighubMRStatus, *mrStatus)
	if strings.TrimSpace(linkMRID) == "" || strings.TrimSpace(linkMRURL) == "" {
		return errors.New("bridge link requires --confighub-mr-id and --confighub-mr-url (or --mr-* aliases)")
	}

	at, err := parseAt(*atRaw)
	if err != nil {
		return err
	}
	link, err := bridgeflow.NewReviewLink(resolvedChangeID, bridgeflow.PullRequestRef{
		Repo:      strings.TrimSpace(repo),
		Number:    number,
		URL:       strings.TrimSpace(prURL),
		CommitSHA: strings.TrimSpace(prSHA),
	}, bridgeflow.MergeRequestRef{
		ID:     strings.TrimSpace(linkMRID),
		URL:    strings.TrimSpace(linkMRURL),
		Status: strings.TrimSpace(linkMRStatus),
	}, at)
	if err != nil {
		return err
	}

	if strings.TrimSpace(*baseURL) == "" {
		return writeJSONOutput(*out, link, *pretty)
	}

	resolvedToken := strings.TrimSpace(*token)
	if resolvedToken == "" {
		resolvedToken = defaultConfigHubToken()
	}
	if resolvedToken == "" {
		return errors.New("bridge link requires --token, CONFIGHUB_TOKEN, or CUB_TOKEN when --base-url is set")
	}

	result, err := bridgeflow.SubmitReviewLink(context.Background(), bridgeflow.LinkClient{
		BaseURL:      strings.TrimSpace(*baseURL),
		BearerToken:  resolvedToken,
		EndpointPath: strings.TrimSpace(*endpoint),
	}, link)
	if err != nil {
		return err
	}
	return writeJSONOutput(*out, result, *pretty)
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
	resolvedBaseURL := strings.TrimSpace(*baseURL)
	if resolvedBaseURL == "" {
		resolvedBaseURL = defaultConfigHubBaseURL()
	}
	if resolvedBaseURL == "" {
		return errors.New("bridge decision query requires --base-url")
	}
	resolvedToken := strings.TrimSpace(*token)
	if resolvedToken == "" {
		resolvedToken = defaultConfigHubToken()
	}
	if strings.TrimSpace(*changeID) == "" {
		return errors.New("bridge decision query requires --change-id")
	}

	rec, err := bridgeflow.QueryDecisionByChangeID(context.Background(), bridgeflow.DecisionClient{
		BaseURL:      resolvedBaseURL,
		BearerToken:  resolvedToken,
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

func resolveBridgeLinkChangeID(changeID, bundlePath string) (string, error) {
	if strings.TrimSpace(bundlePath) == "" {
		if strings.TrimSpace(changeID) == "" {
			return "", errors.New("bridge link requires --bundle or --change-id")
		}
		return strings.TrimSpace(changeID), nil
	}

	var bundle publish.ChangeBundle
	if err := readJSONInput(bundlePath, &bundle); err != nil {
		return "", fmt.Errorf("read bundle json: %w", err)
	}
	if err := publish.VerifyBundle(bundle); err != nil {
		return "", fmt.Errorf("verify bundle before link: %w", err)
	}
	bundleChangeID := strings.TrimSpace(bundle.ChangeID)
	if bundleChangeID == "" {
		return "", errors.New("bridge link bundle is missing change_id")
	}
	if strings.TrimSpace(changeID) != "" && strings.TrimSpace(changeID) != bundleChangeID {
		return "", fmt.Errorf("bridge link change_id mismatch: --change-id=%s bundle=%s", strings.TrimSpace(changeID), bundleChangeID)
	}
	return bundleChangeID, nil
}

func firstPositive(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
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
	printCommandHelp(
		out,
		"cub-gen bridge: advanced ConfigHub API workflows",
		[]string{
			"Use bridge after local repo proof or connected smoke is already clear.",
			"Most users should start with demo-connected.sh or run-connected-smoke.sh first.",
		},
		helpSection{
			Title: "Usage",
			Lines: []string{
				"  cub-gen bridge ingest [--in FILE|-] --base-url URL [--token TOKEN] [--endpoint PATH] [--json] [--pretty]",
				"  cub-gen bridge link --bundle FILE|- --github-pr-repo REPO --github-pr-number N --github-pr-url URL --confighub-mr-id ID --confighub-mr-url URL [--base-url URL --token TOKEN]",
				"  cub-gen bridge decision <create|attach|apply|query> [flags]",
				"  cub-gen bridge promote <init|govern|verify|open|approve|merge> [flags]",
			},
		},
		helpSection{
			Title: "What it's for",
			Lines: []string{
				"  ingest      Submit a verified bundle to ConfigHub",
				"  link        Create/update GitHub PR <-> ConfigHub MR review links",
				"  decision    Query backend decision state or simulate it offline",
				"  promote     Track PR<->MR and upstream DRY promotion flows",
			},
		},
		helpSection{
			Title: "Examples",
			Lines: []string{
				"  cub-gen bridge ingest --in bundle.json --base-url https://confighub.example",
				"  cub-gen bridge decision create --ingest ingest-result.json",
				"  cub-gen bridge decision attach --decision decision.json --attestation attestation.json",
				"  cub-gen bridge decision apply --decision decision.json --state ALLOW --approved-by platform-admin --reason \"policy checks passed\"",
				"  cub-gen bridge decision query --base-url https://confighub.example --change-id chg_123",
				"  cub-gen bridge link --bundle bundle.json --github-pr-repo github.com/confighub/apps --github-pr-number 42 --github-pr-url https://github.com/confighub/apps/pull/42 --confighub-mr-id mr_123 --confighub-mr-url https://confighub.example/mr/123",
				"  cub-gen bridge promote init --change-id chg_123 --app-pr-repo github.com/confighub/apps --app-pr-number 42 --app-pr-url https://github.com/confighub/apps/pull/42 --mr-id mr_123 --mr-url https://confighub.example/mr/123",
			},
		},
		helpSection{
			Title: "Tips",
			Lines: []string{
				"  - bridge link is the single-command PR/MR correlation path",
				"  - bridge decision query is the authoritative backend lookup path",
				"  - local decision create|attach|apply commands are for offline contract simulation",
			},
		},
	)
}

func printBridgeDecisionUsage(out io.Writer) {
	printCommandHelp(
		out,
		"cub-gen bridge decision: governed decision-state commands",
		nil,
		helpSection{
			Title: "Usage",
			Lines: []string{
				"  cub-gen bridge decision create --ingest FILE|- [--out FILE|-] [--at RFC3339] [--pretty]",
				"  cub-gen bridge decision attach --decision FILE|- --attestation FILE [--out FILE|-] [--at RFC3339] [--pretty]",
				"  cub-gen bridge decision apply --decision FILE|- --state ALLOW|ESCALATE|BLOCK --reason TEXT [--approved-by NAME|--policy-ref REF] [--out FILE|-] [--at RFC3339] [--pretty]",
				"  cub-gen bridge decision query --base-url URL --change-id ID [--token TOKEN] [--endpoint PATH] [--pretty]",
			},
		},
	)
}

func printBridgePromoteUsage(out io.Writer) {
	printCommandHelp(
		out,
		"cub-gen bridge promote: PR<->MR and upstream DRY promotion flow commands",
		nil,
		helpSection{
			Title: "Usage",
			Lines: []string{
				"  cub-gen bridge promote init --change-id ID --app-pr-repo REPO --app-pr-number N --app-pr-url URL --mr-id ID --mr-url URL [--app-pr-sha SHA] [--mr-status STATUS] [--out FILE|-] [--at RFC3339] [--pretty]",
				"  cub-gen bridge promote govern --flow FILE|- --state ALLOW|ESCALATE|BLOCK [--decision-ref REF] [--out FILE|-] [--at RFC3339] [--pretty]",
				"  cub-gen bridge promote verify --flow FILE|- [--out FILE|-] [--at RFC3339] [--pretty]",
				"  cub-gen bridge promote open --flow FILE|- --repo REPO --number N --url URL [--sha SHA] [--out FILE|-] [--at RFC3339] [--pretty]",
				"  cub-gen bridge promote approve --flow FILE|- --by NAME [--out FILE|-] [--at RFC3339] [--pretty]",
				"  cub-gen bridge promote merge --flow FILE|- --by NAME [--out FILE|-] [--at RFC3339] [--pretty]",
			},
		},
		helpSection{
			Title: "Examples",
			Lines: []string{
				"  cub-gen bridge promote govern --flow flow.json --state ALLOW --decision-ref decision_123",
				"  cub-gen bridge promote verify --flow flow.json",
				"  cub-gen bridge promote open --flow flow.json --repo github.com/confighub/platform-dry --number 7 --url https://github.com/confighub/platform-dry/pull/7",
				"  cub-gen bridge promote approve --flow flow.json --by platform-owner",
				"  cub-gen bridge promote merge --flow flow.json --by platform-owner",
			},
		},
	)
}
