package bridge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const defaultReviewLinkPath = "/api/v1/pr-mr-links"

// LinkClient holds bridge API connection settings for PR/MR review links.
type LinkClient struct {
	BaseURL      string
	BearerToken  string
	EndpointPath string
	HTTPClient   *http.Client
}

// LinkResult captures ConfigHub review-link create/update response metadata.
type LinkResult struct {
	StatusCode     int        `json:"status_code"`
	LinkID         string     `json:"link_id,omitempty"`
	Status         string     `json:"status,omitempty"`
	Idempotent     bool       `json:"idempotent"`
	ChangeID       string     `json:"change_id"`
	IdempotencyKey string     `json:"idempotency_key"`
	ReviewLink     ReviewLink `json:"review_link"`
}

type linkAPIResponse struct {
	LinkID  string `json:"link_id,omitempty"`
	Status  string `json:"status,omitempty"`
	Message string `json:"message,omitempty"`
}

func ReviewLinkIdempotencyKey(link ReviewLink) string {
	return strings.Join([]string{
		strings.TrimSpace(link.ChangeID),
		strings.TrimSpace(link.GitPR.Repo),
		strconv.Itoa(link.GitPR.Number),
		strings.TrimSpace(link.ConfigHubMR.ID),
	}, ":")
}

// SubmitReviewLink creates or updates the ConfigHub-side PR/MR linkage record.
// HTTP 409 is treated as idempotent success.
func SubmitReviewLink(ctx context.Context, client LinkClient, link ReviewLink) (LinkResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ValidateReviewLink(link); err != nil {
		return LinkResult{}, err
	}

	endpoint, err := resolveReviewLinkEndpoint(client)
	if err != nil {
		return LinkResult{}, err
	}

	body, err := json.Marshal(link)
	if err != nil {
		return LinkResult{}, fmt.Errorf("marshal review link: %w", err)
	}

	idempotencyKey := ReviewLinkIdempotencyKey(link)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return LinkResult{}, fmt.Errorf("build review-link request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", idempotencyKey)
	if token := strings.TrimSpace(client.BearerToken); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	httpClient := client.HTTPClient
	if httpClient == nil {
		httpClient = defaultHTTPClient
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return LinkResult{}, fmt.Errorf("send review-link request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return LinkResult{}, fmt.Errorf("read review-link response: %w", err)
	}

	var decoded linkAPIResponse
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusAccepted || resp.StatusCode == http.StatusConflict {
		if len(bytes.TrimSpace(respBody)) > 0 {
			if err := json.Unmarshal(respBody, &decoded); err != nil {
				return LinkResult{}, fmt.Errorf("decode review-link response: %w", err)
			}
		}
	}

	result := LinkResult{
		StatusCode:     resp.StatusCode,
		LinkID:         strings.TrimSpace(decoded.LinkID),
		Status:         strings.TrimSpace(decoded.Status),
		Idempotent:     resp.StatusCode == http.StatusConflict,
		ChangeID:       link.ChangeID,
		IdempotencyKey: idempotencyKey,
		ReviewLink:     link,
	}

	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated, http.StatusAccepted:
		if result.Status == "" {
			result.Status = "linked"
		}
		return result, nil
	case http.StatusConflict:
		if result.Status == "" {
			result.Status = "exists"
		}
		return result, nil
	case http.StatusNotFound:
		return LinkResult{}, fmt.Errorf("bridge link endpoint is not available: status=404; set --endpoint if your ConfigHub backend uses a different path")
	default:
		msg := strings.TrimSpace(string(respBody))
		if msg == "" {
			msg = "<empty>"
		}
		return LinkResult{}, fmt.Errorf("bridge link failed: status=%d body=%s", resp.StatusCode, msg)
	}
}

func resolveReviewLinkEndpoint(client LinkClient) (string, error) {
	base := strings.TrimSpace(client.BaseURL)
	if base == "" {
		return "", fmt.Errorf("bridge link base URL is required")
	}
	parsed, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("parse bridge link base URL: %w", err)
	}
	path := strings.TrimSpace(client.EndpointPath)
	if path == "" {
		path = defaultReviewLinkPath
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	parsed.Path = strings.TrimSuffix(parsed.Path, "/") + path
	return parsed.String(), nil
}
