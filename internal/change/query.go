package change

import (
	"fmt"
	"strings"

	"github.com/confighub/cub-gen/internal/model"
	"github.com/confighub/cub-gen/internal/publish"
)

type QueryContext struct {
	Input      PreviewInput
	Change     PreviewSummary
	Provenance []model.ProvenanceRecord
}

type ImpactQuery struct {
	VariantFilter string `json:"variant_filter,omitempty"`
	DryPathFilter string `json:"dry_path_filter,omitempty"`
	WetPathFilter string `json:"wet_path_filter,omitempty"`
	OwnerFilter   string `json:"owner_filter,omitempty"`
	MatchCount    int    `json:"match_count"`
}

type ImpactResult struct {
	Input   PreviewInput   `json:"input"`
	Change  PreviewSummary `json:"change"`
	Query   ImpactQuery    `json:"query"`
	Impacts []ImpactEntry  `json:"impacts"`
}

type ExplainQuery struct {
	VariantFilter string `json:"variant_filter,omitempty"`
	WetPathFilter string `json:"wet_path_filter,omitempty"`
	DryPathFilter string `json:"dry_path_filter,omitempty"`
	OwnerFilter   string `json:"owner_filter,omitempty"`
	MatchCount    int    `json:"match_count"`
}

type ExplainResult struct {
	Input       PreviewInput      `json:"input"`
	Change      PreviewSummary    `json:"change"`
	Query       ExplainQuery      `json:"query"`
	Explanation ExplainSuggestion `json:"explanation"`
}

func NewQueryContext(input PreviewInput, change PreviewSummary, provenance []model.ProvenanceRecord) QueryContext {
	return QueryContext{
		Input:      input,
		Change:     change,
		Provenance: append([]model.ProvenanceRecord(nil), provenance...),
	}
}

func QueryContextFromBundle(bundle publish.ChangeBundle) QueryContext {
	return QueryContext{
		Input: PreviewInput{
			TargetSlug:       bundle.TargetSlug,
			TargetPath:       bundle.TargetPath,
			RenderTargetSlug: bundle.RenderTargetSlug,
			RenderTargetPath: bundle.RenderTargetPath,
			Space:            bundle.Space,
			Ref:              bundle.Ref,
		},
		Change: PreviewSummary{
			ChangeID:          bundle.ChangeID,
			TraceID:           bundle.TraceID,
			BundleDigest:      bundle.BundleDigest,
			AttestationDigest: "",
		},
		Provenance: append([]model.ProvenanceRecord(nil), bundle.Provenance...),
	}
}

func BuildExplainResult(ctx QueryContext, wetFilter, dryFilter, ownerFilter string) (ExplainResult, error) {
	wetFilter = strings.TrimSpace(wetFilter)
	dryFilter = strings.TrimSpace(dryFilter)
	ownerFilter = strings.TrimSpace(ownerFilter)

	suggestion, matchCount, ok := PickInverseSuggestion(ctx.Provenance, wetFilter, dryFilter, ownerFilter)
	if !ok {
		return ExplainResult{}, fmt.Errorf("no inverse edit explanation matched filters (wet_path=%q dry_path=%q owner=%q)", wetFilter, dryFilter, ownerFilter)
	}

	return ExplainResult{
		Input:  ctx.Input,
		Change: ctx.Change,
		Query: ExplainQuery{
			WetPathFilter: wetFilter,
			DryPathFilter: dryFilter,
			OwnerFilter:   ownerFilter,
			MatchCount:    matchCount,
		},
		Explanation: suggestion,
	}, nil
}

func BuildImpactResult(ctx QueryContext, dryFilter, wetFilter, ownerFilter string) (ImpactResult, error) {
	dryFilter = strings.TrimSpace(dryFilter)
	wetFilter = strings.TrimSpace(wetFilter)
	ownerFilter = strings.TrimSpace(ownerFilter)

	impacts, matchCount, ok := CollectImpactSuggestions(ctx.Provenance, dryFilter, wetFilter, ownerFilter)
	if !ok {
		return ImpactResult{}, fmt.Errorf("no change impact matched filters (dry_path=%q wet_path=%q owner=%q)", dryFilter, wetFilter, ownerFilter)
	}

	return ImpactResult{
		Input:  ctx.Input,
		Change: ctx.Change,
		Query: ImpactQuery{
			DryPathFilter: dryFilter,
			WetPathFilter: wetFilter,
			OwnerFilter:   ownerFilter,
			MatchCount:    matchCount,
		},
		Impacts: impacts,
	}, nil
}
