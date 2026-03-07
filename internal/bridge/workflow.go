package bridge

import (
	"fmt"
	"strings"
	"time"
)

const (
	workflowSchemaVersion  = "cub.confighub.io/pr-mr-promotion-flow/v1"
	reviewLinkStatusOpen   = "OPEN"
	reviewLinkStatusMerged = "MERGED"
)

type PullRequestRef struct {
	Repo      string `json:"repo"`
	Number    int    `json:"number"`
	URL       string `json:"url"`
	CommitSHA string `json:"commit_sha,omitempty"`
}

type MergeRequestRef struct {
	ID     string `json:"id"`
	URL    string `json:"url"`
	Status string `json:"status,omitempty"`
}

// ReviewLink correlates one change_id across Git PR and ConfigHub MR records.
type ReviewLink struct {
	SchemaVersion string          `json:"schema_version"`
	ChangeID      string          `json:"change_id"`
	GitPR         PullRequestRef  `json:"git_pr"`
	ConfigHubMR   MergeRequestRef `json:"confighub_mr"`
	Status        string          `json:"status"`
	UpdatedAt     string          `json:"updated_at"`
}

type PromotionState string

const (
	PromotionStateAwaitingDecision  PromotionState = "AWAITING_DECISION"
	PromotionStateDecisionBlocked   PromotionState = "DECISION_BLOCKED"
	PromotionStateReadyForPromotion PromotionState = "READY_FOR_PROMOTION"
	PromotionStatePromotionPROpen   PromotionState = "PROMOTION_PR_OPEN"
	PromotionStatePromoted          PromotionState = "PROMOTED"
)

// PromotionFlow enforces explicit review gates for app-overlay to platform-DRY promotion.
type PromotionFlow struct {
	SchemaVersion            string         `json:"schema_version"`
	ChangeID                 string         `json:"change_id"`
	ReviewLink               ReviewLink     `json:"review_link"`
	GovernanceDecision       DecisionState  `json:"governance_decision,omitempty"`
	GovernanceDecisionRef    string         `json:"governance_decision_ref,omitempty"`
	DeploymentVerified       bool           `json:"deployment_verified"`
	PromotionPR              PullRequestRef `json:"promotion_pr"`
	PlatformReviewApproved   bool           `json:"platform_review_approved"`
	PlatformReviewApprovedBy string         `json:"platform_review_approved_by,omitempty"`
	PlatformReviewApprovedAt string         `json:"platform_review_approved_at,omitempty"`
	PromotionMerged          bool           `json:"promotion_merged"`
	PromotionMergedBy        string         `json:"promotion_merged_by,omitempty"`
	PromotionMergedAt        string         `json:"promotion_merged_at,omitempty"`
	State                    PromotionState `json:"state"`
	UpdatedAt                string         `json:"updated_at"`
}

func NewReviewLink(changeID string, pr PullRequestRef, mr MergeRequestRef, at time.Time) (ReviewLink, error) {
	rec := ReviewLink{
		SchemaVersion: workflowSchemaVersion,
		ChangeID:      strings.TrimSpace(changeID),
		GitPR:         pr,
		ConfigHubMR:   mr,
		Status:        reviewLinkStatusOpen,
		UpdatedAt:     normalizeTime(at),
	}
	return rec, ValidateReviewLink(rec)
}

func ValidateReviewLink(rec ReviewLink) error {
	if rec.SchemaVersion != workflowSchemaVersion {
		return fmt.Errorf("unsupported schema_version %q", rec.SchemaVersion)
	}
	if strings.TrimSpace(rec.ChangeID) == "" {
		return fmt.Errorf("missing change_id")
	}
	if strings.TrimSpace(rec.GitPR.Repo) == "" {
		return fmt.Errorf("missing git_pr.repo")
	}
	if rec.GitPR.Number <= 0 {
		return fmt.Errorf("git_pr.number must be > 0")
	}
	if strings.TrimSpace(rec.GitPR.URL) == "" {
		return fmt.Errorf("missing git_pr.url")
	}
	if strings.TrimSpace(rec.ConfigHubMR.ID) == "" {
		return fmt.Errorf("missing confighub_mr.id")
	}
	if strings.TrimSpace(rec.ConfigHubMR.URL) == "" {
		return fmt.Errorf("missing confighub_mr.url")
	}
	switch strings.TrimSpace(rec.Status) {
	case reviewLinkStatusOpen, reviewLinkStatusMerged:
	default:
		return fmt.Errorf("unsupported review-link status %q", rec.Status)
	}
	if strings.TrimSpace(rec.UpdatedAt) == "" {
		return fmt.Errorf("missing updated_at")
	}
	return nil
}

func NewPromotionFlow(link ReviewLink, at time.Time) (PromotionFlow, error) {
	if err := ValidateReviewLink(link); err != nil {
		return PromotionFlow{}, err
	}
	rec := PromotionFlow{
		SchemaVersion: workflowSchemaVersion,
		ChangeID:      link.ChangeID,
		ReviewLink:    link,
		State:         PromotionStateAwaitingDecision,
		UpdatedAt:     normalizeTime(at),
	}
	return rec, ValidatePromotionFlow(rec)
}

// ApplyGovernanceDecision records explicit ALLOW|ESCALATE|BLOCK outcome before promotion.
func ApplyGovernanceDecision(flow PromotionFlow, decision DecisionState, decisionRef string, at time.Time) (PromotionFlow, error) {
	if err := ValidatePromotionFlow(flow); err != nil {
		return PromotionFlow{}, err
	}
	switch decision {
	case DecisionStateAllow, DecisionStateEscalate, DecisionStateBlock:
	default:
		return PromotionFlow{}, fmt.Errorf("unsupported governance decision %q", decision)
	}
	flow.GovernanceDecision = decision
	flow.GovernanceDecisionRef = strings.TrimSpace(decisionRef)
	if decision == DecisionStateAllow {
		flow.State = PromotionStateReadyForPromotion
	} else {
		flow.State = PromotionStateDecisionBlocked
	}
	flow.UpdatedAt = normalizeTime(at)
	return flow, ValidatePromotionFlow(flow)
}

// MarkDeploymentVerified records successful rollout before upstream DRY promotion.
func MarkDeploymentVerified(flow PromotionFlow, at time.Time) (PromotionFlow, error) {
	if err := ValidatePromotionFlow(flow); err != nil {
		return PromotionFlow{}, err
	}
	if flow.GovernanceDecision != DecisionStateAllow {
		return PromotionFlow{}, fmt.Errorf("deployment verification requires ALLOW decision, got %q", flow.GovernanceDecision)
	}
	flow.DeploymentVerified = true
	flow.State = PromotionStateReadyForPromotion
	flow.UpdatedAt = normalizeTime(at)
	return flow, ValidatePromotionFlow(flow)
}

// OpenPromotionPR starts a separate upstream platform-DRY review gate.
func OpenPromotionPR(flow PromotionFlow, pr PullRequestRef, at time.Time) (PromotionFlow, error) {
	if err := ValidatePromotionFlow(flow); err != nil {
		return PromotionFlow{}, err
	}
	if flow.GovernanceDecision != DecisionStateAllow {
		return PromotionFlow{}, fmt.Errorf("promotion PR requires ALLOW decision, got %q", flow.GovernanceDecision)
	}
	if !flow.DeploymentVerified {
		return PromotionFlow{}, fmt.Errorf("promotion PR requires deployment_verified=true")
	}
	if strings.TrimSpace(pr.Repo) == "" || pr.Number <= 0 || strings.TrimSpace(pr.URL) == "" {
		return PromotionFlow{}, fmt.Errorf("promotion PR requires repo, number, and url")
	}
	flow.PromotionPR = pr
	flow.State = PromotionStatePromotionPROpen
	flow.UpdatedAt = normalizeTime(at)
	return flow, ValidatePromotionFlow(flow)
}

// ApprovePlatformReview marks required upstream review gate as satisfied.
func ApprovePlatformReview(flow PromotionFlow, approvedBy string, at time.Time) (PromotionFlow, error) {
	if err := ValidatePromotionFlow(flow); err != nil {
		return PromotionFlow{}, err
	}
	if flow.State != PromotionStatePromotionPROpen {
		return PromotionFlow{}, fmt.Errorf("platform review approval requires promotion PR open, got %q", flow.State)
	}
	name := strings.TrimSpace(approvedBy)
	if name == "" {
		return PromotionFlow{}, fmt.Errorf("platform review approval requires non-empty approver")
	}
	flow.PlatformReviewApproved = true
	flow.PlatformReviewApprovedBy = name
	flow.PlatformReviewApprovedAt = normalizeTime(at)
	flow.UpdatedAt = flow.PlatformReviewApprovedAt
	return flow, ValidatePromotionFlow(flow)
}

// MergePromotionPR enforces separate upstream review before merging platform DRY promotion.
func MergePromotionPR(flow PromotionFlow, mergedBy string, at time.Time) (PromotionFlow, error) {
	if err := ValidatePromotionFlow(flow); err != nil {
		return PromotionFlow{}, err
	}
	if flow.State != PromotionStatePromotionPROpen {
		return PromotionFlow{}, fmt.Errorf("promotion merge requires promotion PR open, got %q", flow.State)
	}
	if !flow.PlatformReviewApproved {
		return PromotionFlow{}, fmt.Errorf("guardrail: cannot merge platform DRY promotion without separate platform review approval")
	}
	name := strings.TrimSpace(mergedBy)
	if name == "" {
		return PromotionFlow{}, fmt.Errorf("promotion merge requires non-empty merged_by")
	}
	flow.PromotionMerged = true
	flow.PromotionMergedBy = name
	flow.PromotionMergedAt = normalizeTime(at)
	flow.State = PromotionStatePromoted
	flow.UpdatedAt = flow.PromotionMergedAt
	flow.ReviewLink.Status = reviewLinkStatusMerged
	flow.ReviewLink.UpdatedAt = flow.UpdatedAt
	return flow, ValidatePromotionFlow(flow)
}

func ValidatePromotionFlow(flow PromotionFlow) error {
	if flow.SchemaVersion != workflowSchemaVersion {
		return fmt.Errorf("unsupported schema_version %q", flow.SchemaVersion)
	}
	if strings.TrimSpace(flow.ChangeID) == "" {
		return fmt.Errorf("missing change_id")
	}
	if err := ValidateReviewLink(flow.ReviewLink); err != nil {
		return err
	}
	if flow.ChangeID != flow.ReviewLink.ChangeID {
		return fmt.Errorf("review link change_id mismatch: flow=%s review_link=%s", flow.ChangeID, flow.ReviewLink.ChangeID)
	}
	switch flow.State {
	case PromotionStateAwaitingDecision, PromotionStateDecisionBlocked, PromotionStateReadyForPromotion, PromotionStatePromotionPROpen, PromotionStatePromoted:
	default:
		return fmt.Errorf("unsupported promotion state %q", flow.State)
	}
	if flow.State == PromotionStateReadyForPromotion || flow.State == PromotionStatePromotionPROpen || flow.State == PromotionStatePromoted {
		if flow.GovernanceDecision != DecisionStateAllow {
			return fmt.Errorf("state %q requires ALLOW decision", flow.State)
		}
	}
	if flow.State == PromotionStatePromotionPROpen || flow.State == PromotionStatePromoted {
		if strings.TrimSpace(flow.PromotionPR.Repo) == "" || flow.PromotionPR.Number <= 0 || strings.TrimSpace(flow.PromotionPR.URL) == "" {
			return fmt.Errorf("state %q requires promotion_pr linkage", flow.State)
		}
	}
	if flow.State == PromotionStatePromoted {
		if !flow.PlatformReviewApproved {
			return fmt.Errorf("state %q requires platform_review_approved=true", flow.State)
		}
		if !flow.PromotionMerged {
			return fmt.Errorf("state %q requires promotion_merged=true", flow.State)
		}
	}
	if strings.TrimSpace(flow.UpdatedAt) == "" {
		return fmt.Errorf("missing updated_at")
	}
	return nil
}

func normalizeTime(at time.Time) string {
	if at.IsZero() {
		at = time.Now().UTC()
	}
	return at.UTC().Format(time.RFC3339)
}
