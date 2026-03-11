package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"

	bridgeflow "github.com/confighub/cub-gen/internal/bridge"
	"github.com/confighub/cub-gen/internal/model"
)

type changeRunOptions struct {
	Space            string
	Ref              string
	WhereResource    string
	Mode             string
	BaseURL          string
	Token            string
	IngestEndpoint   string
	DecisionEndpoint string
	Verifier         string
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

type changeAPIInput struct {
	TargetSlug       string `json:"target_slug"`
	RenderTargetSlug string `json:"render_target_slug"`
	Space            string `json:"space,omitempty"`
	Ref              string `json:"ref,omitempty"`
	WhereResource    string `json:"where_resource,omitempty"`
}

type changeAPIConnected struct {
	BaseURL          string `json:"base_url,omitempty"`
	Token            string `json:"token,omitempty"`
	IngestEndpoint   string `json:"ingest_endpoint,omitempty"`
	DecisionEndpoint string `json:"decision_endpoint,omitempty"`
}

type changeAPIRequest struct {
	Action    string             `json:"action"`
	Mode      string             `json:"mode,omitempty"`
	Input     changeAPIInput     `json:"input"`
	Connected changeAPIConnected `json:"connected,omitempty"`
}

type changeAPIResponse struct {
	Change             changePreviewSummary      `json:"change"`
	Decision           *changeRunDecision        `json:"decision,omitempty"`
	PromotionReady     *bool                     `json:"promotion_ready,omitempty"`
	Verification       changePreviewVerification `json:"verification"`
	EditRecommendation model.InverseEditPointer  `json:"edit_recommendation"`
	Artifacts          map[string]string         `json:"artifacts"`
}

type apiErrorBody struct {
	Error struct {
		Code    string         `json:"code"`
		Message string         `json:"message"`
		Details map[string]any `json:"details,omitempty"`
	} `json:"error"`
}

type changeRecord struct {
	Input              changePreviewInput
	Change             changePreviewSummary
	Verification       changePreviewVerification
	EditRecommendation model.InverseEditPointer
	Decision           *changeRunDecision
	PromotionReady     *bool
	Artifacts          map[string]string
	Provenance         []model.ProvenanceRecord
}

type changeAPIServer struct {
	defaultSpace    string
	defaultRef      string
	defaultVerifier string

	mu      sync.RWMutex
	records map[string]changeRecord
}

func runChangeAPI(args []string) error {
	if len(args) == 0 {
		printChangeAPIUsage(os.Stderr)
		return errors.New("change api subcommand required")
	}

	switch args[0] {
	case "help", "-h", "--help":
		printChangeAPIUsage(os.Stdout)
		return nil
	case "serve":
		return runChangeAPIServe(args[1:])
	default:
		printChangeAPIUsage(os.Stderr)
		return fmt.Errorf("unknown change api subcommand: %s", args[0])
	}
}

func runChangeAPIServe(args []string) error {
	fs := flag.NewFlagSet("change api serve", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	listen := fs.String("listen", "127.0.0.1:8787", "Listen address for compatibility API")
	space := fs.String("space", "default", "Default ConfigHub space label")
	ref := fs.String("ref", "HEAD", "Default git ref label")
	verifier := fs.String("verifier", "cub-gen", "Default verifier identity label")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("usage: cub-gen change api serve [--listen ADDR] [--space SPACE] [--ref REF] [--verifier NAME]")
	}

	handler := newChangeAPIHandler(strings.TrimSpace(*space), strings.TrimSpace(*ref), strings.TrimSpace(*verifier))
	fmt.Fprintf(os.Stderr, "change api listening on http://%s\n", strings.TrimSpace(*listen))
	return http.ListenAndServe(strings.TrimSpace(*listen), handler)
}

func newChangeAPIHandler(defaultSpace, defaultRef, defaultVerifier string) http.Handler {
	space := strings.TrimSpace(defaultSpace)
	if space == "" {
		space = "default"
	}
	ref := strings.TrimSpace(defaultRef)
	if ref == "" {
		ref = "HEAD"
	}
	verifier := strings.TrimSpace(defaultVerifier)
	if verifier == "" {
		verifier = "cub-gen"
	}

	return &changeAPIServer{
		defaultSpace:    space,
		defaultRef:      ref,
		defaultVerifier: verifier,
		records:         map[string]changeRecord{},
	}
}

func (s *changeAPIServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/v1/changes" && r.Method == http.MethodPost:
		s.handlePostChanges(w, r)
		return
	case strings.HasPrefix(r.URL.Path, "/v1/changes/"):
		s.handleChangeByID(w, r)
		return
	case r.URL.Path == "/healthz" && r.Method == http.MethodGet:
		writeJSONResponse(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	default:
		writeAPIError(w, http.StatusNotFound, "NOT_FOUND", "endpoint not found", nil)
		return
	}
}

func (s *changeAPIServer) handlePostChanges(w http.ResponseWriter, r *http.Request) {
	var req changeAPIRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body", map[string]any{"error": err.Error()})
		return
	}
	if dec.More() {
		writeAPIError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body", map[string]any{"error": "multiple JSON values"})
		return
	}

	action := strings.ToLower(strings.TrimSpace(req.Action))
	if action == "" {
		writeAPIError(w, http.StatusBadRequest, "INVALID_REQUEST", "action is required", map[string]any{"field": "action"})
		return
	}
	if action != "preview" && action != "run" {
		writeAPIError(w, http.StatusBadRequest, "INVALID_REQUEST", "action must be preview|run", map[string]any{"field": "action"})
		return
	}

	targetSlug := strings.TrimSpace(req.Input.TargetSlug)
	renderTargetSlug := strings.TrimSpace(req.Input.RenderTargetSlug)
	if targetSlug == "" || renderTargetSlug == "" {
		writeAPIError(w, http.StatusBadRequest, "INVALID_REQUEST", "input.target_slug and input.render_target_slug are required", map[string]any{"field": "input"})
		return
	}

	space := strings.TrimSpace(req.Input.Space)
	if space == "" {
		space = s.defaultSpace
	}
	ref := strings.TrimSpace(req.Input.Ref)
	if ref == "" {
		ref = s.defaultRef
	}
	whereResource := strings.TrimSpace(req.Input.WhereResource)

	if action == "preview" {
		preview, _, imported, err := buildChangePreviewResult(targetSlug, renderTargetSlug, space, ref, whereResource, s.defaultVerifier)
		if err != nil {
			writeAPIError(w, http.StatusUnprocessableEntity, "CHANGE_FAILED", err.Error(), nil)
			return
		}
		record := changeRecord{
			Input:              preview.Input,
			Change:             preview.Change,
			Verification:       preview.Verification,
			EditRecommendation: preview.EditRecommendation,
			Artifacts:          defaultArtifactRefs(preview.Change.ChangeID),
			Provenance:         imported.Provenance,
		}
		s.upsertRecord(record)
		writeJSONResponse(w, http.StatusOK, changeAPIResponse{
			Change:             record.Change,
			Verification:       record.Verification,
			EditRecommendation: record.EditRecommendation,
			Artifacts:          cloneStringMap(record.Artifacts),
		})
		return
	}

	runMode := strings.ToLower(strings.TrimSpace(req.Mode))
	if runMode == "" {
		writeAPIError(w, http.StatusBadRequest, "INVALID_REQUEST", "mode is required for action=run", map[string]any{"field": "mode"})
		return
	}

	runResult, provenance, err := executeChangeRun(targetSlug, renderTargetSlug, changeRunOptions{
		Space:            space,
		Ref:              ref,
		WhereResource:    whereResource,
		Mode:             runMode,
		BaseURL:          strings.TrimSpace(req.Connected.BaseURL),
		Token:            strings.TrimSpace(req.Connected.Token),
		IngestEndpoint:   strings.TrimSpace(req.Connected.IngestEndpoint),
		DecisionEndpoint: strings.TrimSpace(req.Connected.DecisionEndpoint),
		Verifier:         s.defaultVerifier,
	})
	if err != nil {
		status := http.StatusUnprocessableEntity
		if strings.Contains(err.Error(), "requires --base-url") || strings.Contains(err.Error(), "mode must be") {
			status = http.StatusBadRequest
		}
		writeAPIError(w, status, "CHANGE_FAILED", err.Error(), nil)
		return
	}

	promotion := runResult.PromotionReady
	record := changeRecord{
		Input:              runResult.Preview.Input,
		Change:             runResult.Preview.Change,
		Verification:       runResult.Preview.Verification,
		EditRecommendation: runResult.Preview.EditRecommendation,
		Decision: &changeRunDecision{
			State:     runResult.Decision.State,
			Authority: runResult.Decision.Authority,
			Source:    runResult.Decision.Source,
		},
		PromotionReady: &promotion,
		Artifacts:      defaultArtifactRefs(runResult.Preview.Change.ChangeID),
		Provenance:     provenance,
	}
	s.upsertRecord(record)

	writeJSONResponse(w, http.StatusOK, changeAPIResponse{
		Change:             record.Change,
		Decision:           record.Decision,
		PromotionReady:     record.PromotionReady,
		Verification:       record.Verification,
		EditRecommendation: record.EditRecommendation,
		Artifacts:          cloneStringMap(record.Artifacts),
	})
}

func (s *changeAPIServer) handleChangeByID(w http.ResponseWriter, r *http.Request) {
	trimmed := strings.TrimPrefix(r.URL.Path, "/v1/changes/")
	trimmed = strings.Trim(trimmed, "/")
	if trimmed == "" {
		writeAPIError(w, http.StatusNotFound, "NOT_FOUND", "endpoint not found", nil)
		return
	}
	parts := strings.Split(trimmed, "/")
	changeID, err := url.PathUnescape(parts[0])
	if err != nil || strings.TrimSpace(changeID) == "" {
		writeAPIError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid change_id path segment", nil)
		return
	}

	record, ok := s.getRecord(changeID)
	if !ok {
		writeAPIError(w, http.StatusNotFound, "NOT_FOUND", "change_id not found", map[string]any{"change_id": changeID})
		return
	}

	if len(parts) == 1 {
		if r.Method != http.MethodGet {
			writeAPIError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed", nil)
			return
		}
		writeJSONResponse(w, http.StatusOK, changeAPIResponse{
			Change:             record.Change,
			Decision:           record.Decision,
			PromotionReady:     record.PromotionReady,
			Verification:       record.Verification,
			EditRecommendation: record.EditRecommendation,
			Artifacts:          cloneStringMap(record.Artifacts),
		})
		return
	}

	if len(parts) == 2 && parts[1] == "explanations" {
		if r.Method != http.MethodGet {
			writeAPIError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed", nil)
			return
		}
		wetFilter := strings.TrimSpace(r.URL.Query().Get("wet_path"))
		dryFilter := strings.TrimSpace(r.URL.Query().Get("dry_path"))
		ownerFilter := strings.TrimSpace(r.URL.Query().Get("owner"))
		suggestion, matchCount, ok := pickInverseSuggestion(record.Provenance, wetFilter, dryFilter, ownerFilter)
		if !ok {
			writeAPIError(w, http.StatusNotFound, "NOT_FOUND", "no matching explanations", map[string]any{"change_id": changeID})
			return
		}
		resp := changeExplainResult{
			Input:  record.Input,
			Change: record.Change,
			Query: changeExplainQuery{
				WetPathFilter: wetFilter,
				DryPathFilter: dryFilter,
				OwnerFilter:   ownerFilter,
				MatchCount:    matchCount,
			},
			Explanation: suggestion,
		}
		writeJSONResponse(w, http.StatusOK, resp)
		return
	}

	writeAPIError(w, http.StatusNotFound, "NOT_FOUND", "endpoint not found", nil)
}

func (s *changeAPIServer) upsertRecord(record changeRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records[record.Change.ChangeID] = record
}

func (s *changeAPIServer) getRecord(changeID string) (changeRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	record, ok := s.records[changeID]
	if !ok {
		return changeRecord{}, false
	}
	return record, true
}

func defaultArtifactRefs(changeID string) map[string]string {
	items := map[string]string{
		"bundle":      fmt.Sprintf("change://%s/bundle", changeID),
		"attestation": fmt.Sprintf("change://%s/attestation", changeID),
		"provenance":  fmt.Sprintf("change://%s/provenance", changeID),
	}
	keys := make([]string, 0, len(items))
	for k := range items {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	ordered := make(map[string]string, len(items))
	for _, k := range keys {
		ordered[k] = items[k]
	}
	return ordered
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func writeAPIError(w http.ResponseWriter, status int, code, message string, details map[string]any) {
	resp := apiErrorBody{}
	resp.Error.Code = code
	resp.Error.Message = message
	if len(details) > 0 {
		resp.Error.Details = details
	}
	writeJSONResponse(w, status, resp)
}

func writeJSONResponse(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(payload)
}

func printChangeAPIUsage(out *os.File) {
	fmt.Fprintln(out, "Usage:")
	fmt.Fprintln(out, "  cub-gen change api serve [--listen ADDR] [--space SPACE] [--ref REF] [--verifier NAME]")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Compatibility HTTP endpoints:")
	fmt.Fprintln(out, "  POST /v1/changes")
	fmt.Fprintln(out, "  GET  /v1/changes/{change_id}")
	fmt.Fprintln(out, "  GET  /v1/changes/{change_id}/explanations")
}
