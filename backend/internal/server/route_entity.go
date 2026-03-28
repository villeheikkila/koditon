package server

import (
	"context"
	"errors"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"koditon-go/internal/ads"
)

type entityDetailInput struct {
	ID string `query:"id" required:"true" doc:"Canonical ID or source URL"`
}

type detailFieldOutput struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type entityDetailOutput struct {
	Body struct {
		CanonicalID    string              `json:"canonical_id"`
		Source         string              `json:"source"`
		Kind           string              `json:"kind"`
		NativeID       string              `json:"native_id"`
		Headline       string              `json:"headline"`
		Address        string              `json:"address,omitempty"`
		City           string              `json:"city,omitempty"`
		Postal         string              `json:"postal,omitempty"`
		Price          *int64              `json:"price,omitempty"`
		Area           *float64            `json:"area,omitempty"`
		RoomLayout     string              `json:"room_layout,omitempty"`
		URL            string              `json:"url,omitempty"`
		LastSeenAt     *time.Time          `json:"last_seen_at,omitempty"`
		CanonicalExtra []detailFieldOutput `json:"canonical_extra,omitempty"`
		SourceSpecific []detailFieldOutput `json:"source_specific,omitempty"`
		Related        []detailFieldOutput `json:"related,omitempty"`
		RawJSON        string              `json:"raw_json,omitempty"`
	}
}

func (s *Server) entityDetailHandler(ctx context.Context, input *entityDetailInput) (*entityDetailOutput, error) {
	canonicalID, err := ads.ResolveInput(input.ID, s.cfg.Shortcut.SitemapBase, s.cfg.Frontdoor.SitemapBase)
	if err != nil {
		return nil, huma.Error400BadRequest("invalid ID or URL: " + input.ID)
	}

	detail, err := s.adsService.DetailByCanonicalID(ctx, canonicalID)
	if err != nil {
		if errors.Is(err, ads.ErrNotFound) {
			return nil, huma.Error404NotFound("entity not found")
		}
		s.logger.ErrorContext(ctx, "entity detail lookup failed", "canonical_id", canonicalID, "error", err)
		return nil, huma.Error500InternalServerError("failed to fetch entity detail")
	}

	out := &entityDetailOutput{}
	out.Body.CanonicalID = detail.Canonical.CanonicalID
	out.Body.Source = detail.Canonical.Source
	out.Body.Kind = detail.Canonical.Kind
	out.Body.NativeID = detail.Canonical.NativeID
	out.Body.Headline = detail.Canonical.Headline
	out.Body.Address = detail.Canonical.Address
	out.Body.City = detail.Canonical.City
	out.Body.Postal = detail.Canonical.Postal
	out.Body.Price = detail.Canonical.Price
	out.Body.Area = detail.Canonical.Area
	out.Body.RoomLayout = detail.Canonical.RoomLayout
	out.Body.URL = detail.Canonical.URL
	if !detail.Canonical.LastSeenAt.IsZero() {
		t := detail.Canonical.LastSeenAt
		out.Body.LastSeenAt = &t
	}
	out.Body.CanonicalExtra = toDetailFields(detail.CanonicalExtra)
	out.Body.SourceSpecific = toDetailFields(detail.SourceSpecific)
	out.Body.Related = toDetailFields(detail.Related)
	out.Body.RawJSON = detail.Raw.Pretty
	return out, nil
}

func toDetailFields(fields []ads.DetailField) []detailFieldOutput {
	if len(fields) == 0 {
		return nil
	}
	out := make([]detailFieldOutput, len(fields))
	for i, f := range fields {
		out[i] = detailFieldOutput{Label: f.Label, Value: f.Value}
	}
	return out
}
