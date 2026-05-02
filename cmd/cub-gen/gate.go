package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/confighub/cub-gen/internal/mutationgate"
)

func runGate(args []string) error {
	if len(args) == 0 {
		printGateUsage(os.Stderr)
		return errors.New("gate subcommand required")
	}
	switch args[0] {
	case "help", "-h", "--help":
		printGateUsage(os.Stdout)
		return nil
	case "mutation":
		return runGateMutation(args[1:])
	default:
		printGateUsage(os.Stderr)
		return fmt.Errorf("unknown gate subcommand: %s", args[0])
	}
}

func runGateMutation(args []string) error {
	fs := flag.NewFlagSet("gate mutation", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), pluginHelpLine("Usage: cub-gen gate mutation [flags] <rendered-field>"))
		fmt.Fprintln(fs.Output())
		fmt.Fprintln(fs.Output(), "Evaluate a proposed mutation with Generator route proof.")
		fmt.Fprintln(fs.Output())
		fmt.Fprintln(fs.Output(), "Exactly one proof source is required:")
		fmt.Fprintln(fs.Output(), "  --policy FILE   generator-route-policy JSON, annotation proposal, or Unit YAML")
		fmt.Fprintln(fs.Output(), "  --routes FILE   Spring field-routes.yaml")
		fmt.Fprintln(fs.Output(), "  --bundle FILE   cub-gen change bundle with inverse edit pointers")
		fmt.Fprintln(fs.Output())
		fmt.Fprintln(fs.Output(), "Flags:")
		fs.PrintDefaults()
	}

	policyPath := fs.String("policy", "", "Route policy JSON/YAML path")
	routesPath := fs.String("routes", "", "Spring field-routes.yaml path")
	bundlePath := fs.String("bundle", "", "cub-gen change bundle path")
	field := fs.String("field", "", "Rendered field path to mutate")
	resourceType := fs.String("resource-type", "", "Rendered resource type, such as Deployment or ConfigMap")
	resourceName := fs.String("resource-name", "", "Rendered resource name")
	space := fs.String("space", "", "ConfigHub space")
	component := fs.String("component", "", "Component id/name")
	variant := fs.String("variant", "", "Variant id/name")
	target := fs.String("target", "", "Target id/name")
	changeID := fs.String("change-id", "", "Change id")
	origin := fs.String("origin", "confighub-initiative", "Mutation origin")
	attemptedLayer := fs.String("attempted-layer", "rendered-config", "Attempted mutation layer")
	newValue := fs.String("new-value", "", "Optional proposed value")
	githubPRRepo := fs.String("github-pr-repo", "", "Optional GitHub repo for lift-upstream next action")
	githubPR := fs.String("github-pr", "", "Optional GitHub PR URL/id to link")
	confighubMR := fs.String("confighub-mr", "", "Optional ConfigHub MR URL/id to link")
	sourceFiles := fs.String("source-files", "", "Comma-separated source files for lift-upstream next action")
	atValue := fs.String("at", "", "RFC3339 evaluation time")
	jsonOut := fs.Bool("json", false, "Output JSON")
	pretty := fs.Bool("pretty", true, "Pretty-print JSON output")
	enforce := fs.Bool("enforce", false, "Return non-zero when decision.state is not ALLOW")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() > 1 {
		return errors.New("usage: cub-gen gate mutation [flags] <rendered-field>")
	}
	renderedField := strings.TrimSpace(*field)
	if renderedField == "" && fs.NArg() == 1 {
		renderedField = fs.Arg(0)
	}
	if renderedField == "" {
		return errors.New("gate mutation requires --field or <rendered-field>")
	}

	policy, policySource, bundleChangeID, err := loadGatePolicy(*policyPath, *routesPath, *bundlePath)
	if err != nil {
		return err
	}
	if strings.TrimSpace(*changeID) == "" {
		*changeID = bundleChangeID
	}
	if strings.TrimSpace(*changeID) == "" {
		*changeID = mutationgate.StableChangeID(strings.Join([]string{policySource, renderedField, *resourceType, *resourceName}, "|"))
	}
	at, err := parseGateTime(*atValue)
	if err != nil {
		return err
	}

	var link *mutationgate.Link
	if strings.TrimSpace(*githubPR) != "" || strings.TrimSpace(*confighubMR) != "" {
		link = &mutationgate.Link{
			Mode:        "paired",
			GitHubPR:    strings.TrimSpace(*githubPR),
			ConfigHubMR: strings.TrimSpace(*confighubMR),
		}
	}
	decision, err := mutationgate.Evaluate(policy, mutationgate.Request{
		ChangeID: strings.TrimSpace(*changeID),
		Subject: mutationgate.Subject{
			Space:        strings.TrimSpace(*space),
			Component:    strings.TrimSpace(*component),
			Variant:      strings.TrimSpace(*variant),
			Target:       strings.TrimSpace(*target),
			ResourceType: strings.TrimSpace(*resourceType),
			ResourceName: strings.TrimSpace(*resourceName),
		},
		Mutation: mutationgate.Mutation{
			Origin:         strings.TrimSpace(*origin),
			RenderedField:  renderedField,
			AttemptedLayer: strings.TrimSpace(*attemptedLayer),
			NewValue:       strings.TrimSpace(*newValue),
		},
		Link:         link,
		GitHubPRRepo: strings.TrimSpace(*githubPRRepo),
		SourceFiles:  splitCSV(*sourceFiles),
		EvaluatedAt:  at,
		PolicySource: policySource,
	})
	if err != nil {
		return err
	}

	if *jsonOut {
		if err := writeJSON(os.Stdout, decision, *pretty); err != nil {
			return err
		}
	} else {
		printGateDecision(os.Stdout, decision)
	}
	if *enforce && decision.Decision.State != mutationgate.DecisionAllow {
		return fmt.Errorf("mutation apply gate decision %s: %s", decision.Decision.State, decision.Decision.Reason)
	}
	return nil
}

func loadGatePolicy(policyPath, routesPath, bundlePath string) (mutationgate.Policy, string, string, error) {
	count := 0
	for _, value := range []string{policyPath, routesPath, bundlePath} {
		if strings.TrimSpace(value) != "" {
			count++
		}
	}
	if count != 1 {
		return mutationgate.Policy{}, "", "", errors.New("gate mutation requires exactly one of --policy, --routes, or --bundle")
	}
	if strings.TrimSpace(policyPath) != "" {
		policy, source, err := mutationgate.LoadPolicyFile(policyPath)
		return policy, source, "", err
	}
	if strings.TrimSpace(routesPath) != "" {
		policy, source, err := mutationgate.LoadSpringRoutesFile(routesPath)
		return policy, source, "", err
	}
	policy, source, changeID, err := mutationgate.LoadBundleFile(bundlePath)
	return policy, source, changeID, err
}

func parseGateTime(value string) (time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return time.Time{}, nil
	}
	at, err := time.Parse(time.RFC3339, strings.TrimSpace(value))
	if err != nil {
		return time.Time{}, fmt.Errorf("parse --at: %w", err)
	}
	return at, nil
}

func splitCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if strings.TrimSpace(part) != "" {
			out = append(out, strings.TrimSpace(part))
		}
	}
	return out
}

func printGateDecision(out io.Writer, decision mutationgate.Decision) {
	fmt.Fprintf(out, "Decision: %s\n", decision.Decision.State)
	fmt.Fprintf(out, "Route:    %s\n", decision.Route.Kind)
	fmt.Fprintf(out, "Field:    %s\n", decision.Mutation.RenderedField)
	if decision.Proof.Owner != "" {
		fmt.Fprintf(out, "Owner:    %s\n", decision.Proof.Owner)
	}
	fmt.Fprintf(out, "Reason:   %s\n", decision.Decision.Reason)
	for _, action := range decision.NextActions {
		fmt.Fprintf(out, "Next:     %s", action.Kind)
		if action.Description != "" {
			fmt.Fprintf(out, " - %s", action.Description)
		}
		fmt.Fprintln(out)
	}
}

func printGateUsage(out io.Writer) {
	printCommandHelp(
		out,
		"cub-gen gate: decide whether a proposed mutation may apply",
		[]string{
			"Mutation apply gates answer: if this deployed field is wrong, where do I fix it?",
		},
		helpSection{
			Title: "COMMANDS",
			Lines: []string{
				"  mutation    Return route.kind, decision.state, next actions, and proof_events",
			},
		},
		helpSection{
			Title: "EXAMPLES",
			Lines: []string{
				"  cub-gen gate mutation --routes ./examples/springboot-paas/operational/field-routes.yaml feature.inventory.reservationMode",
				"  cub-gen gate mutation --bundle bundle.json Deployment/spec/template/spec/containers[0]/image",
			},
		},
	)
}
