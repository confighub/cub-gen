package bridge

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSubmitReviewLinkCreated(t *testing.T) {
	link := sampleReviewLink(t)
	var gotPayload ReviewLink
	var gotKey string
	var gotAuth string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != defaultReviewLinkPath {
			t.Fatalf("expected path %s, got %s", defaultReviewLinkPath, r.URL.Path)
		}
		gotKey = r.Header.Get("Idempotency-Key")
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotPayload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"link_id":"link_123","status":"linked"}`))
	}))
	defer srv.Close()

	res, err := SubmitReviewLink(context.Background(), LinkClient{
		BaseURL:     srv.URL,
		BearerToken: "token-123",
	}, link)
	if err != nil {
		t.Fatalf("SubmitReviewLink returned error: %v", err)
	}

	expectedKey := ReviewLinkIdempotencyKey(link)
	if gotKey != expectedKey {
		t.Fatalf("expected idempotency key %q, got %q", expectedKey, gotKey)
	}
	if gotAuth != "Bearer token-123" {
		t.Fatalf("expected bearer auth header, got %q", gotAuth)
	}
	if gotPayload.ChangeID != link.ChangeID {
		t.Fatalf("expected payload change_id %q, got %q", link.ChangeID, gotPayload.ChangeID)
	}
	if gotPayload.GitPR.Number != link.GitPR.Number {
		t.Fatalf("expected PR number %d, got %d", link.GitPR.Number, gotPayload.GitPR.Number)
	}
	if res.StatusCode != http.StatusCreated || res.LinkID != "link_123" || res.Status != "linked" || res.Idempotent {
		t.Fatalf("unexpected link result: %+v", res)
	}
	if res.ChangeID != link.ChangeID || res.IdempotencyKey != expectedKey {
		t.Fatalf("unexpected link identity: %+v", res)
	}
}

func TestSubmitReviewLinkConflictIsIdempotent(t *testing.T) {
	link := sampleReviewLink(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"link_id":"link_123","status":"exists"}`))
	}))
	defer srv.Close()

	res, err := SubmitReviewLink(context.Background(), LinkClient{BaseURL: srv.URL}, link)
	if err != nil {
		t.Fatalf("SubmitReviewLink returned error: %v", err)
	}
	if !res.Idempotent {
		t.Fatalf("expected idempotent conflict result, got %+v", res)
	}
	if res.Status != "exists" {
		t.Fatalf("expected exists status, got %+v", res)
	}
}

func TestSubmitReviewLinkNotFoundIsActionable(t *testing.T) {
	link := sampleReviewLink(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"not found"}`))
	}))
	defer srv.Close()

	_, err := SubmitReviewLink(context.Background(), LinkClient{BaseURL: srv.URL}, link)
	if err == nil {
		t.Fatal("expected 404 to fail")
	}
	if !strings.Contains(err.Error(), "bridge link endpoint is not available") {
		t.Fatalf("expected actionable 404 error, got %q", err.Error())
	}
}

func sampleReviewLink(t *testing.T) ReviewLink {
	t.Helper()
	link, err := NewReviewLink(
		"chg_123",
		PullRequestRef{
			Repo:      "github.com/confighub/app",
			Number:    42,
			URL:       "https://github.com/confighub/app/pull/42",
			CommitSHA: "abc123",
		},
		MergeRequestRef{
			ID:     "mr_123",
			URL:    "https://hub.confighub.com/mr/123",
			Status: "OPEN",
		},
		time.Date(2026, 5, 1, 7, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("NewReviewLink returned error: %v", err)
	}
	return link
}
