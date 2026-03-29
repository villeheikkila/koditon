package api

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
		// Canonical
		CanonicalID string     `json:"canonical_id"`
		Source      string     `json:"source"`
		Kind        string     `json:"kind"`
		NativeID    string     `json:"native_id"`
		Headline    string     `json:"headline"`
		URL         string     `json:"url,omitempty"`
		LastSeenAt  *time.Time `json:"last_seen_at,omitempty"`

		// Location
		StreetAddress string `json:"street_address,omitempty"`
		City          string `json:"city,omitempty"`
		Postal        string `json:"postal,omitempty"`

		// Pricing
		AskingPrice         *int64   `json:"asking_price,omitempty"`
		DebtFreePrice       *int64   `json:"debt_free_price,omitempty"`
		DebtShareAmount     *int64   `json:"debt_share_amount,omitempty"`
		PricePerSquareMeter *float64 `json:"price_per_m2,omitempty"`

		// Property details
		AreaM2       *float64 `json:"area_m2,omitempty"`
		RoomLayout   string   `json:"room_layout,omitempty"`
		RoomsCount   *int32   `json:"rooms_count,omitempty"`
		FloorLevel   *int32   `json:"floor_level,omitempty"`
		TotalFloors  *int32   `json:"total_floors,omitempty"`
		BuildYear    *int32   `json:"build_year,omitempty"`
		Condition    string   `json:"condition,omitempty"`
		EnergyClass  string   `json:"energy_class,omitempty"`
		PlotType     string   `json:"plot_type,omitempty"`
		Elevator     *bool    `json:"elevator,omitempty"`
		Sauna        *bool    `json:"sauna,omitempty"`

		// Monthly charges
		MaintenanceChargeMonthly *float64 `json:"maintenance_charge_monthly,omitempty"`
		TotalChargeMonthly       *float64 `json:"total_charge_monthly,omitempty"`
		WaterCharge              *float64 `json:"water_charge,omitempty"`

		// Text sections
		DescriptionText        string `json:"description_text,omitempty"`
		AvailabilityText       string `json:"availability_text,omitempty"`
		RenovationsDoneText    string `json:"renovations_done_text,omitempty"`
		RenovationsPlannedText string `json:"renovations_planned_text,omitempty"`
		AdditionalInfoText     string `json:"additional_info_text,omitempty"`
		ChargesText            string `json:"charges_text,omitempty"`

		// Source-specific & related
		CanonicalExtra []detailFieldOutput `json:"canonical_extra,omitempty"`
		SourceSpecific []detailFieldOutput `json:"source_specific,omitempty"`
		Related        []detailFieldOutput `json:"related,omitempty"`
	}
}

func (a *API) entityDetailHandler(ctx context.Context, input *entityDetailInput) (*entityDetailOutput, error) {
	canonicalID, err := ads.ResolveInput(input.ID, a.cfg.Shortcut.SitemapBase, a.cfg.Frontdoor.SitemapBase)
	if err != nil {
		return nil, huma.Error400BadRequest("invalid ID or URL: " + input.ID)
	}

	detail, err := a.adsService.DetailByCanonicalID(ctx, canonicalID)
	if err != nil {
		if errors.Is(err, ads.ErrNotFound) {
			return nil, huma.Error404NotFound("entity not found")
		}
		a.logger.ErrorContext(ctx, "entity detail lookup failed", "canonical_id", canonicalID, "error", err)
		return nil, huma.Error500InternalServerError("failed to fetch entity detail")
	}

	n := detail.Normalized
	out := &entityDetailOutput{}
	b := &out.Body

	b.CanonicalID = detail.Canonical.CanonicalID
	b.Source = detail.Canonical.Source
	b.Kind = detail.Canonical.Kind
	b.NativeID = detail.Canonical.NativeID
	b.Headline = detail.Canonical.Headline
	b.URL = detail.Canonical.URL
	if !detail.Canonical.LastSeenAt.IsZero() {
		t := detail.Canonical.LastSeenAt
		b.LastSeenAt = &t
	}

	b.StreetAddress = n.StreetAddress
	b.City = n.City
	b.Postal = n.Postal

	b.AskingPrice = n.AskingPrice
	b.DebtFreePrice = n.DebtFreePrice
	b.DebtShareAmount = n.DebtShareAmount
	b.PricePerSquareMeter = n.PricePerSquareMeter

	b.AreaM2 = n.AreaM2
	b.RoomLayout = n.RoomLayout
	b.RoomsCount = n.RoomsCount
	b.FloorLevel = n.FloorLevel
	b.TotalFloors = n.TotalFloors
	b.BuildYear = n.BuildYear
	b.Condition = n.Condition
	b.EnergyClass = n.EnergyClass
	b.PlotType = n.PlotType
	b.Elevator = n.Elevator
	b.Sauna = n.Sauna

	b.MaintenanceChargeMonthly = n.MaintenanceChargeMonthly
	b.TotalChargeMonthly = n.TotalChargeMonthly
	b.WaterCharge = n.WaterCharge

	b.DescriptionText = n.DescriptionText
	b.AvailabilityText = n.AvailabilityText
	b.RenovationsDoneText = n.RenovationsDoneText
	b.RenovationsPlannedText = n.RenovationsPlannedText
	b.AdditionalInfoText = n.AdditionalInfoText
	b.ChargesText = n.ChargesText

	b.CanonicalExtra = toDetailFields(detail.CanonicalExtra)
	b.SourceSpecific = toDetailFields(detail.SourceSpecific)
	b.Related = toDetailFields(detail.Related)

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
