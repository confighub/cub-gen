package bridge

import (
	"testing"
	"time"
)

func TestReviewLinkValidation(t *testing.T) {
	link, err := NewReviewLink(
		"chg_1",
		PullRequestRef{
			Repo:      "github.com/confighub/app-repo",
			Number:    42,
			URL:       "https://github.com/confighub/app-repo/pull/42",
			CommitSHA: "abc123",
		},
		MergeRequestRef{
			ID:     "mr_123",
			URL:    "https://confighub.local/spaces/platform/merge-requests/mr_123",
			Status: "OPEN",
		},
		time.Date(2026, 3, 6, 11, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("NewReviewLink returned error: %v", err)
	}
	if link.SchemaVersion != workflowSchemaVersion {
		t.Fatalf("expected schema %q, got %q", workflowSchemaVersion, link.SchemaVersion)
	}
	if link.ChangeID != "chg_1" {
		t.Fatalf("expected change_id chg_1, got %q", link.ChangeID)
	}
	if link.Status != reviewLinkStatusOpen {
		t.Fatalf("expected review-link status %q, got %q", reviewLinkStatusOpen, link.Status)
	}
}

func TestPromotionFlowHappyPath(t *testing.T) {
	link, err := NewReviewLink(
		"chg_1",
		PullRequestRef{
			Repo:   "github.com/confighub/app-repo",
			Number: 42,
			URL:    "https://github.com/confighub/app-repo/pull/42",
		},
		MergeRequestRef{
			ID:  "mr_123",
			URL: "https://confighub.local/spaces/platform/merge-requests/mr_123",
		},
		time.Date(2026, 3, 6, 11, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("NewReviewLink returned error: %v", err)
	}
	flow, err := NewPromotionFlow(link, time.Date(2026, 3, 6, 11, 1, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewPromotionFlow returned error: %v", err)
	}

	flow, err = ApplyGovernanceDecision(flow, DecisionStateAllow, "decision_123", time.Date(2026, 3, 6, 11, 2, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("ApplyGovernanceDecision returned error: %v", err)
	}
	flow, err = MarkDeploymentVerified(flow, time.Date(2026, 3, 6, 11, 3, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("MarkDeploymentVerified returned error: %v", err)
	}
	flow, err = OpenPromotionPR(flow, PullRequestRef{
		Repo:   "github.com/confighub/platform-dry",
		Number: 7,
		URL:    "https://github.com/confighub/platform-dry/pull/7",
	}, time.Date(2026, 3, 6, 11, 4, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("OpenPromotionPR returned error: %v", err)
	}
	flow, err = ApprovePlatformReview(flow, "platform-owner", time.Date(2026, 3, 6, 11, 5, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("ApprovePlatformReview returned error: %v", err)
	}
	flow, err = MergePromotionPR(flow, "platform-owner", time.Date(2026, 3, 6, 11, 6, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("MergePromotionPR returned error: %v", err)
	}

	if flow.State != PromotionStatePromoted {
		t.Fatalf("expected state %q, got %q", PromotionStatePromoted, flow.State)
	}
	if !flow.PlatformReviewApproved || !flow.PromotionMerged {
		t.Fatalf("expected review + merge gates true, got flow=%+v", flow)
	}
	if flow.ReviewLink.Status != reviewLinkStatusMerged {
		t.Fatalf("expected review-link status %q, got %q", reviewLinkStatusMerged, flow.ReviewLink.Status)
	}
}

func TestPromotionFlowRequiresAllowBeforePromotion(t *testing.T) {
	link, err := NewReviewLink(
		"chg_1",
		PullRequestRef{
			Repo:   "github.com/confighub/app-repo",
			Number: 42,
			URL:    "https://github.com/confighub/app-repo/pull/42",
		},
		MergeRequestRef{
			ID:  "mr_123",
			URL: "https://confighub.local/spaces/platform/merge-requests/mr_123",
		},
		time.Date(2026, 3, 6, 11, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("NewReviewLink returned error: %v", err)
	}
	flow, err := NewPromotionFlow(link, time.Date(2026, 3, 6, 11, 1, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewPromotionFlow returned error: %v", err)
	}
	flow, err = ApplyGovernanceDecision(flow, DecisionStateEscalate, "decision_123", time.Date(2026, 3, 6, 11, 2, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("ApplyGovernanceDecision returned error: %v", err)
	}

	_, err = MarkDeploymentVerified(flow, time.Date(2026, 3, 6, 11, 3, 0, 0, time.UTC))
	if err == nil {
		t.Fatal("expected ALLOW gate error, got nil")
	}
	expected := `deployment verification requires ALLOW decision, got "ESCALATE"`
	if err.Error() != expected {
		t.Fatalf("expected %q, got %q", expected, err.Error())
	}
}

func TestPromotionFlowGuardrailBlocksMergeWithoutPlatformReview(t *testing.T) {
	link, err := NewReviewLink(
		"chg_1",
		PullRequestRef{
			Repo:   "github.com/confighub/app-repo",
			Number: 42,
			URL:    "https://github.com/confighub/app-repo/pull/42",
		},
		MergeRequestRef{
			ID:  "mr_123",
			URL: "https://confighub.local/spaces/platform/merge-requests/mr_123",
		},
		time.Date(2026, 3, 6, 11, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("NewReviewLink returned error: %v", err)
	}
	flow, err := NewPromotionFlow(link, time.Date(2026, 3, 6, 11, 1, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewPromotionFlow returned error: %v", err)
	}
	flow, err = ApplyGovernanceDecision(flow, DecisionStateAllow, "decision_123", time.Date(2026, 3, 6, 11, 2, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("ApplyGovernanceDecision returned error: %v", err)
	}
	flow, err = MarkDeploymentVerified(flow, time.Date(2026, 3, 6, 11, 3, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("MarkDeploymentVerified returned error: %v", err)
	}
	flow, err = OpenPromotionPR(flow, PullRequestRef{
		Repo:   "github.com/confighub/platform-dry",
		Number: 7,
		URL:    "https://github.com/confighub/platform-dry/pull/7",
	}, time.Date(2026, 3, 6, 11, 4, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("OpenPromotionPR returned error: %v", err)
	}

	_, err = MergePromotionPR(flow, "platform-owner", time.Date(2026, 3, 6, 11, 5, 0, 0, time.UTC))
	if err == nil {
		t.Fatal("expected separate review guardrail error, got nil")
	}
	expected := "guardrail: cannot merge platform DRY promotion without separate platform review approval"
	if err.Error() != expected {
		t.Fatalf("expected %q, got %q", expected, err.Error())
	}
}
