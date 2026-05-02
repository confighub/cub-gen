package mutationgate

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/confighub/cub-gen/internal/model"
	"github.com/confighub/cub-gen/internal/proof"
	"github.com/confighub/cub-gen/internal/publish"
	"github.com/confighub/cub-gen/internal/springboot"
	"gopkg.in/yaml.v3"
)

const (
	SchemaVersion = "cub.confighub.io/mutation-apply-gate-decision/v1"

	RouteApplyHere     = "apply-here"
	RouteOverlay       = "overlay"
	RouteLiftUpstream  = "lift-upstream"
	RouteBlockEscalate = "block/escalate"
	RouteReview        = "review-required"

	DecisionAllow    = "ALLOW"
	DecisionEscalate = "ESCALATE"
	DecisionBlock    = "BLOCK"
)

type Subject struct {
	Space        string `json:"space,omitempty"`
	Component    string `json:"component,omitempty"`
	Variant      string `json:"variant,omitempty"`
	Target       string `json:"target,omitempty"`
	ResourceType string `json:"resource_type,omitempty"`
	ResourceName string `json:"resource_name,omitempty"`
}

type Mutation struct {
	Origin         string `json:"origin,omitempty"`
	RenderedField  string `json:"rendered_field"`
	AttemptedLayer string `json:"attempted_layer"`
	NewValue       string `json:"new_value,omitempty"`
}

type Rule struct {
	ResourceType string  `json:"resource_type,omitempty" yaml:"resource_type,omitempty"`
	ResourceName string  `json:"resource_name,omitempty" yaml:"resource_name,omitempty"`
	Path         string  `json:"path,omitempty" yaml:"path,omitempty"`
	WetPath      string  `json:"wet_path,omitempty" yaml:"wet_path,omitempty"`
	Route        string  `json:"route" yaml:"route"`
	Owner        string  `json:"owner,omitempty" yaml:"owner,omitempty"`
	SourcePath   string  `json:"source_path,omitempty" yaml:"source_path,omitempty"`
	SourceField  string  `json:"source_field,omitempty" yaml:"source_field,omitempty"`
	Generator    string  `json:"generator,omitempty" yaml:"generator,omitempty"`
	ProposalHint string  `json:"proposal_hint,omitempty" yaml:"proposal_hint,omitempty"`
	Confidence   float64 `json:"confidence,omitempty" yaml:"confidence,omitempty"`
}

type Policy struct {
	SchemaVersion string `json:"schema_version" yaml:"schema_version"`
	Routes        []Rule `json:"routes" yaml:"routes"`
}

type ProofSummary struct {
	Source        string  `json:"source"`
	Generator     string  `json:"generator,omitempty"`
	SourceField   string  `json:"source_field,omitempty"`
	SourceFile    string  `json:"source_file,omitempty"`
	Owner         string  `json:"owner,omitempty"`
	Confidence    float64 `json:"confidence,omitempty"`
	MatchedRule   string  `json:"matched_rule,omitempty"`
	OriginalRoute string  `json:"original_route,omitempty"`
}

type RouteDecision struct {
	Kind                   string `json:"kind"`
	DirectRenderedMutation string `json:"direct_rendered_mutation"`
	Reason                 string `json:"reason"`
}

type GateDecision struct {
	State  string `json:"state"`
	Reason string `json:"reason"`
}

type NextAction struct {
	Kind        string   `json:"kind"`
	Description string   `json:"description"`
	Owner       string   `json:"owner,omitempty"`
	Repo        string   `json:"repo,omitempty"`
	Files       []string `json:"files,omitempty"`
}

type Link struct {
	Mode        string `json:"mode,omitempty"`
	ConfigHubMR string `json:"confighub_mr,omitempty"`
	GitHubPR    string `json:"github_pr,omitempty"`
}

type Gate struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type Decision struct {
	SchemaVersion  string        `json:"schema_version"`
	Kind           string        `json:"kind"`
	ChangeID       string        `json:"change_id,omitempty"`
	TraceID        string        `json:"trace_id"`
	DecisionDigest string        `json:"decision_digest,omitempty"`
	Gate           Gate          `json:"gate"`
	Subject        Subject       `json:"subject"`
	Mutation       Mutation      `json:"mutation"`
	Proof          ProofSummary  `json:"proof"`
	Route          RouteDecision `json:"route"`
	Decision       GateDecision  `json:"decision"`
	NextActions    []NextAction  `json:"next_actions,omitempty"`
	Link           *Link         `json:"link,omitempty"`
	ProofEvents    []proof.Event `json:"proof_events"`
}

type Request struct {
	ChangeID     string
	TraceID      string
	Subject      Subject
	Mutation     Mutation
	Link         *Link
	GitHubPRRepo string
	SourceFiles  []string
	EvaluatedAt  time.Time
	PolicySource string
}

func PolicyFromSpringRoutes(routes springboot.FieldRoutes, sourcePath string) Policy {
	out := Policy{SchemaVersion: "cub.confighub.io/generator-route-policy/v1"}
	for _, route := range routes.Routes {
		if strings.TrimSpace(route.Match) == "" {
			continue
		}
		out.Routes = append(out.Routes, Rule{
			Path:         strings.TrimSpace(route.Match),
			WetPath:      strings.TrimSpace(route.Match),
			Route:        routeForSpringAction(route.DefaultAction),
			Owner:        strings.TrimSpace(route.Owner),
			SourcePath:   strings.TrimSpace(sourcePath),
			SourceField:  strings.TrimSpace(route.Match),
			Generator:    "springboot",
			ProposalHint: strings.TrimSpace(route.Reason),
			Confidence:   0.94,
		})
	}
	sortRules(out.Routes)
	return out
}

func PolicyFromBundle(bundle publish.ChangeBundle) (Policy, error) {
	if err := publish.VerifyBundle(bundle); err != nil {
		return Policy{}, err
	}
	out := Policy{SchemaVersion: "cub.confighub.io/generator-route-policy/v1"}
	for _, prov := range bundle.Provenance {
		origins := originsByWetPath(prov.FieldOriginMap)
		for _, pointer := range prov.InverseEditPointers {
			wetPath := strings.TrimSpace(pointer.WetPath)
			if wetPath == "" {
				continue
			}
			origin := origins[wetPath]
			out.Routes = append(out.Routes, Rule{
				WetPath:      wetPath,
				Path:         wetPath,
				Route:        normalizeRoute(pointer.Route),
				Owner:        strings.TrimSpace(pointer.Owner),
				SourcePath:   strings.TrimSpace(origin.SourcePath),
				SourceField:  firstNonEmpty(pointer.DryPath, origin.DryPath),
				Generator:    firstNonEmpty(prov.GeneratorProfile, prov.GeneratorName, prov.GeneratorID),
				ProposalHint: strings.TrimSpace(pointer.EditHint),
				Confidence:   pointer.Confidence,
			})
		}
	}
	sortRules(out.Routes)
	return out, nil
}

func Evaluate(policy Policy, req Request) (Decision, error) {
	mutation := req.Mutation
	mutation.RenderedField = strings.TrimSpace(mutation.RenderedField)
	if mutation.RenderedField == "" {
		return Decision{}, fmt.Errorf("rendered_field is required")
	}
	if strings.TrimSpace(mutation.AttemptedLayer) == "" {
		mutation.AttemptedLayer = "rendered-config"
	}
	at := req.EvaluatedAt
	if at.IsZero() {
		at = time.Now().UTC()
	}
	traceID := strings.TrimSpace(req.TraceID)
	if traceID == "" {
		traceID = proof.TraceID(req.ChangeID, req.Subject.Space, req.Subject.Target, "", mutation.RenderedField)
	}

	rule, ok := selectRule(policy.Routes, req.Subject, mutation.RenderedField)
	if !ok {
		rule = Rule{
			Route:        RouteReview,
			ProposalHint: "missing or ambiguous route metadata; review before apply",
		}
	}
	routeKind := normalizeRoute(rule.Route)
	if routeKind == "" {
		routeKind = RouteReview
	}

	decisionState, direct, reason := routeDecision(routeKind, rule)
	actions := nextActions(routeKind, rule, req)
	decisionReason := gateReason(routeKind, decisionState, rule)

	out := Decision{
		SchemaVersion: SchemaVersion,
		Kind:          "MutationApplyGateDecision",
		ChangeID:      strings.TrimSpace(req.ChangeID),
		TraceID:       traceID,
		Gate: Gate{
			Name:    "generator-route",
			Version: "v1",
		},
		Subject:  req.Subject,
		Mutation: mutation,
		Proof: ProofSummary{
			Source:        firstNonEmpty(req.PolicySource, "generator-route-policy"),
			Generator:     rule.Generator,
			SourceField:   rule.SourceField,
			SourceFile:    rule.SourcePath,
			Owner:         rule.Owner,
			Confidence:    rule.Confidence,
			MatchedRule:   firstNonEmpty(rule.WetPath, rule.Path),
			OriginalRoute: strings.TrimSpace(rule.Route),
		},
		Route: RouteDecision{
			Kind:                   routeKind,
			DirectRenderedMutation: direct,
			Reason:                 reason,
		},
		Decision: GateDecision{
			State:  decisionState,
			Reason: decisionReason,
		},
		NextActions: actions,
		Link:        normalizedLink(req.Link),
	}
	out.ProofEvents = []proof.Event{proof.NewEvent(proof.Input{
		EventType:     proof.EventTypeMutationGateEvaluated,
		EventTime:     at,
		Source:        "cub-gen",
		TraceID:       out.TraceID,
		ChangeID:      out.ChangeID,
		Space:         out.Subject.Space,
		TargetSlug:    out.Subject.Target,
		ArtifactKind:  proof.ArtifactKindMutationGate,
		SummaryCounts: map[string]int{"mutation_apply_gate_decisions": 1},
		GeneratorProfiles: uniqueStrings([]string{
			out.Proof.Generator,
		}),
		RouteKind:      out.Route.Kind,
		Owner:          out.Proof.Owner,
		DecisionState:  out.Decision.State,
		DecisionReason: out.Decision.Reason,
	})}
	out.DecisionDigest = DecisionDigest(out)
	out.ProofEvents = proof.SetArtifactDigest(out.ProofEvents, proof.ArtifactKindMutationGate, out.DecisionDigest)
	return out, nil
}

func ValidateDecisionRecord(decision Decision) error {
	if decision.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported schema_version %q", decision.SchemaVersion)
	}
	if decision.Kind != "MutationApplyGateDecision" {
		return fmt.Errorf("unsupported kind %q", decision.Kind)
	}
	if strings.TrimSpace(decision.TraceID) == "" {
		return fmt.Errorf("missing trace_id")
	}
	if strings.TrimSpace(decision.Mutation.RenderedField) == "" {
		return fmt.Errorf("missing mutation.rendered_field")
	}
	if strings.TrimSpace(decision.Route.Kind) == "" {
		return fmt.Errorf("missing route.kind")
	}
	if strings.TrimSpace(decision.Decision.State) == "" {
		return fmt.Errorf("missing decision.state")
	}
	expectedDigest := DecisionDigest(decision)
	if strings.TrimSpace(decision.DecisionDigest) != expectedDigest {
		return fmt.Errorf("decision_digest mismatch: expected %s, got %s", expectedDigest, decision.DecisionDigest)
	}
	eventTime := ""
	for _, event := range decision.ProofEvents {
		if event.EventType == proof.EventTypeMutationGateEvaluated {
			eventTime = event.EventTime
			break
		}
	}
	if err := proof.ValidateArtifactEvents(decision.ProofEvents, proof.Expected{
		EventType:      proof.EventTypeMutationGateEvaluated,
		EventTime:      eventTime,
		Source:         "cub-gen",
		TraceID:        decision.TraceID,
		ChangeID:       decision.ChangeID,
		Space:          decision.Subject.Space,
		TargetSlug:     decision.Subject.Target,
		ArtifactKind:   proof.ArtifactKindMutationGate,
		ArtifactDigest: decision.DecisionDigest,
	}); err != nil {
		return err
	}
	for i, event := range decision.ProofEvents {
		if event.EventType != proof.EventTypeMutationGateEvaluated {
			continue
		}
		if event.RouteKind != decision.Route.Kind {
			return fmt.Errorf("proof_events[%d]: route_kind mismatch: expected %q, got %q", i, decision.Route.Kind, event.RouteKind)
		}
		if event.DecisionState != decision.Decision.State {
			return fmt.Errorf("proof_events[%d]: decision_state mismatch: expected %q, got %q", i, decision.Decision.State, event.DecisionState)
		}
	}
	return nil
}

func DecisionDigest(decision Decision) string {
	normalized := decision
	normalized.DecisionDigest = ""
	normalized.ProofEvents = proof.BlankArtifactDigests(normalized.ProofEvents)
	b, err := json.Marshal(normalized)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func LoadPolicyFile(path string) (Policy, string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Policy{}, "", fmt.Errorf("read policy: %w", err)
	}
	policy, err := parsePolicy(raw)
	if err != nil {
		return Policy{}, "", err
	}
	return policy, path, nil
}

func LoadSpringRoutesFile(path string) (Policy, string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Policy{}, "", fmt.Errorf("read field routes: %w", err)
	}
	var routes springboot.FieldRoutes
	if err := yaml.Unmarshal(raw, &routes); err != nil {
		return Policy{}, "", fmt.Errorf("parse field routes: %w", err)
	}
	return PolicyFromSpringRoutes(routes, path), path, nil
}

func LoadBundleFile(path string) (Policy, string, string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Policy{}, "", "", fmt.Errorf("read bundle: %w", err)
	}
	var bundle publish.ChangeBundle
	if err := json.Unmarshal(raw, &bundle); err != nil {
		return Policy{}, "", "", fmt.Errorf("parse bundle: %w", err)
	}
	policy, err := PolicyFromBundle(bundle)
	if err != nil {
		return Policy{}, "", "", err
	}
	return policy, path, bundle.ChangeID, nil
}

func parsePolicy(raw []byte) (Policy, error) {
	var policy Policy
	if err := json.Unmarshal(raw, &policy); err == nil && len(policy.Routes) > 0 {
		return normalizePolicy(policy), nil
	}
	var proposal struct {
		Policy Policy `json:"policy" yaml:"policy"`
	}
	if err := json.Unmarshal(raw, &proposal); err == nil && len(proposal.Policy.Routes) > 0 {
		return normalizePolicy(proposal.Policy), nil
	}
	var unit struct {
		Metadata struct {
			Annotations map[string]string `json:"annotations" yaml:"annotations"`
		} `json:"metadata" yaml:"metadata"`
	}
	if err := yaml.Unmarshal(raw, &unit); err == nil && len(unit.Metadata.Annotations) > 0 {
		if value := strings.TrimSpace(unit.Metadata.Annotations["confighub.io/generator-route-policy"]); value != "" {
			var embedded Policy
			if err := json.Unmarshal([]byte(value), &embedded); err != nil {
				return Policy{}, fmt.Errorf("parse generator-route-policy annotation: %w", err)
			}
			return normalizePolicy(embedded), nil
		}
	}
	if err := yaml.Unmarshal(raw, &policy); err == nil && len(policy.Routes) > 0 {
		return normalizePolicy(policy), nil
	}
	if err := yaml.Unmarshal(raw, &proposal); err == nil && len(proposal.Policy.Routes) > 0 {
		return normalizePolicy(proposal.Policy), nil
	}
	return Policy{}, fmt.Errorf("policy must be generator-route-policy/v1, annotation proposal, or Unit annotations")
}

func normalizePolicy(policy Policy) Policy {
	if policy.SchemaVersion == "" {
		policy.SchemaVersion = "cub.confighub.io/generator-route-policy/v1"
	}
	for i := range policy.Routes {
		policy.Routes[i].Route = normalizeRoute(policy.Routes[i].Route)
	}
	sortRules(policy.Routes)
	return policy
}

func routeForSpringAction(action springboot.RouteAction) string {
	switch action {
	case springboot.ActionMutableInCH:
		return RouteApplyHere
	case springboot.ActionLiftUpstream:
		return RouteLiftUpstream
	case springboot.ActionGeneratorOwned:
		return RouteBlockEscalate
	default:
		return RouteReview
	}
}

func normalizeRoute(route string) string {
	switch strings.ToLower(strings.TrimSpace(route)) {
	case "", "unknown", "explain", "review", "review-required":
		return RouteReview
	case "mutable-in-ch", "mutable", "apply", "apply-here":
		return RouteApplyHere
	case "overlay", "temporary-overlay":
		return RouteOverlay
	case "lift", "lift-upstream":
		return RouteLiftUpstream
	case "generator-owned", "platform-owned", "block", "blocked", "block-escalate", "block/escalate":
		return RouteBlockEscalate
	default:
		return RouteReview
	}
}

func routeDecision(routeKind string, rule Rule) (state, direct, reason string) {
	switch routeKind {
	case RouteApplyHere:
		return DecisionAllow, "allowed", firstNonEmpty(rule.ProposalHint, "route metadata says this field may be changed here")
	case RouteOverlay:
		return DecisionEscalate, "blocked", firstNonEmpty(rule.ProposalHint, "overlay changes need owner review before apply")
	case RouteLiftUpstream:
		return DecisionEscalate, "blocked", firstNonEmpty(rule.ProposalHint, "change belongs upstream in source config or code")
	case RouteBlockEscalate:
		return DecisionBlock, "blocked", firstNonEmpty(rule.ProposalHint, "field crosses a generator/platform ownership boundary")
	default:
		return DecisionEscalate, "review-required", firstNonEmpty(rule.ProposalHint, "missing route proof; review before apply")
	}
}

func gateReason(routeKind, state string, rule Rule) string {
	owner := strings.TrimSpace(rule.Owner)
	if owner == "" {
		owner = "the owning team"
	}
	switch routeKind {
	case RouteApplyHere:
		return "route apply-here allows this mutation here with recorded proof"
	case RouteOverlay:
		return "route overlay requires " + owner + " review before this exception is applied"
	case RouteLiftUpstream:
		return "route lift-upstream blocks direct rendered mutation and asks for a source change"
	case RouteBlockEscalate:
		if state == DecisionBlock {
			return "route block/escalate blocks direct mutation and requires " + owner
		}
	}
	return "route review-required needs owner review because route proof is missing or ambiguous"
}

func nextActions(routeKind string, rule Rule, req Request) []NextAction {
	owner := strings.TrimSpace(rule.Owner)
	switch routeKind {
	case RouteApplyHere:
		return []NextAction{{
			Kind:        "apply-config-mutation",
			Description: "apply this mutation through the ConfigHub Initiative/apply-gate flow",
			Owner:       owner,
		}}
	case RouteOverlay:
		return []NextAction{{
			Kind:        "review-overlay",
			Description: "record this as a deployment-specific overlay with owner review and expiry policy",
			Owner:       owner,
		}}
	case RouteLiftUpstream:
		files := uniqueStrings(append(req.SourceFiles, rule.SourcePath))
		return []NextAction{{
			Kind:        "create-or-link-github-pr",
			Description: firstNonEmpty(rule.ProposalHint, "create or link a source PR before accepting this as a durable change"),
			Owner:       owner,
			Repo:        strings.TrimSpace(req.GitHubPRRepo),
			Files:       files,
		}}
	case RouteBlockEscalate:
		return []NextAction{{
			Kind:        "request-owner-review",
			Description: firstNonEmpty(rule.ProposalHint, "request owner review or reject the direct mutation"),
			Owner:       owner,
		}}
	default:
		return []NextAction{{
			Kind:        "import-or-enrich-provenance",
			Description: "import/enrich Generator proof before allowing this mutation",
			Owner:       owner,
		}}
	}
}

func selectRule(rules []Rule, subject Subject, field string) (Rule, bool) {
	type candidate struct {
		rule  Rule
		score int
	}
	candidates := []candidate{}
	for _, rule := range rules {
		score, ok := matchRule(rule, subject, field)
		if ok {
			candidates = append(candidates, candidate{rule: rule, score: score})
		}
	}
	if len(candidates) == 0 {
		return Rule{}, false
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score > candidates[j].score
		}
		return firstNonEmpty(candidates[i].rule.WetPath, candidates[i].rule.Path) < firstNonEmpty(candidates[j].rule.WetPath, candidates[j].rule.Path)
	})
	return candidates[0].rule, true
}

func matchRule(rule Rule, subject Subject, field string) (int, bool) {
	score := 0
	if !matchOptional(rule.ResourceType, subject.ResourceType) || !matchOptional(rule.ResourceName, subject.ResourceName) {
		return 0, false
	}
	if strings.TrimSpace(rule.ResourceType) != "" {
		score += 5
	}
	if strings.TrimSpace(rule.ResourceName) != "" {
		score += 5
	}
	for _, pattern := range []string{rule.WetPath, rule.Path, rule.SourceField} {
		if strings.TrimSpace(pattern) == "" {
			continue
		}
		if matchFieldPattern(pattern, field) {
			return score + specificity(pattern), true
		}
	}
	return 0, false
}

func matchOptional(pattern, value string) bool {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return true
	}
	return strings.EqualFold(pattern, strings.TrimSpace(value)) || matchFieldPattern(pattern, value)
}

func matchFieldPattern(pattern, value string) bool {
	pattern = strings.TrimSpace(pattern)
	value = strings.TrimSpace(value)
	if pattern == value {
		return true
	}
	if ok, err := path.Match(pattern, value); err == nil && ok {
		return true
	}
	if strings.Contains(value, ":") {
		parts := strings.Split(value, ":")
		if tail := parts[len(parts)-1]; tail != value && matchFieldPattern(pattern, tail) {
			return true
		}
	}
	re := regexp.QuoteMeta(pattern)
	re = strings.ReplaceAll(re, `\*`, ".*")
	re = "^" + re + "$"
	ok, err := regexp.MatchString(re, value)
	return err == nil && ok
}

func specificity(pattern string) int {
	return len(strings.ReplaceAll(pattern, "*", ""))
}

func sortRules(rules []Rule) {
	sort.Slice(rules, func(i, j int) bool {
		if rules[i].Route != rules[j].Route {
			return rules[i].Route < rules[j].Route
		}
		if rules[i].Owner != rules[j].Owner {
			return rules[i].Owner < rules[j].Owner
		}
		if rules[i].ResourceType != rules[j].ResourceType {
			return rules[i].ResourceType < rules[j].ResourceType
		}
		if rules[i].ResourceName != rules[j].ResourceName {
			return rules[i].ResourceName < rules[j].ResourceName
		}
		return firstNonEmpty(rules[i].WetPath, rules[i].Path) < firstNonEmpty(rules[j].WetPath, rules[j].Path)
	})
}

func originsByWetPath(origins []model.FieldOrigin) map[string]model.FieldOrigin {
	out := map[string]model.FieldOrigin{}
	for _, origin := range origins {
		if strings.TrimSpace(origin.WetPath) != "" {
			out[origin.WetPath] = origin
		}
	}
	return out
}

func normalizedLink(link *Link) *Link {
	if link == nil {
		return nil
	}
	out := *link
	if strings.TrimSpace(out.Mode) == "" && (strings.TrimSpace(out.ConfigHubMR) != "" || strings.TrimSpace(out.GitHubPR) != "") {
		out.Mode = "paired"
	}
	if strings.TrimSpace(out.Mode) == "" {
		return nil
	}
	return &out
}

func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func StableChangeID(seed string) string {
	sum := sha256.Sum256([]byte(seed))
	return "chg_" + hex.EncodeToString(sum[:])[:16]
}
