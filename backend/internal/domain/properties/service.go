package properties

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"koditon/internal/db"
	"koditon/internal/domain/ads"
)

var ErrNotFound = errors.New("property not found")

type Service struct {
	db      db.DBTX
	queries *db.Queries
}

func NewService(dbtx db.DBTX) *Service {
	return &Service{db: dbtx, queries: db.New(dbtx)}
}

func (s *Service) SearchSaleListings(ctx context.Context, params SearchParams) (Page[SaleListingSummary], error) {
	normalized := normalizeParams(params)
	count, err := s.countListings(ctx, normalized, "sale")
	if err != nil {
		return Page[SaleListingSummary]{}, err
	}
	rows, err := s.searchListings(ctx, normalized, "sale")
	if err != nil {
		return Page[SaleListingSummary]{}, err
	}
	out := make([]SaleListingSummary, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.toSaleSummary())
	}
	return Page[SaleListingSummary]{Rows: out, Total: count, Page: normalized.Page, PageSize: normalized.PageSize}, nil
}

func (s *Service) SearchRentals(ctx context.Context, params SearchParams) (Page[RentalSummary], error) {
	normalized := normalizeParams(params)
	count, err := s.countListings(ctx, normalized, "rental")
	if err != nil {
		return Page[RentalSummary]{}, err
	}
	rows, err := s.searchListings(ctx, normalized, "rental")
	if err != nil {
		return Page[RentalSummary]{}, err
	}
	out := make([]RentalSummary, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.toRentalSummary())
	}
	return Page[RentalSummary]{Rows: out, Total: count, Page: normalized.Page, PageSize: normalized.PageSize}, nil
}

func (s *Service) SaleListingByID(ctx context.Context, input string, shortcutBase string, frontdoorBase string) (SaleListing, error) {
	canonicalID, err := s.resolveListingInput(ctx, input, "sale", shortcutBase, frontdoorBase)
	if err != nil {
		return SaleListing{}, err
	}
	source, kind, nativeID, err := ads.ParseCanonicalID(canonicalID)
	if err != nil {
		return SaleListing{}, err
	}
	switch source + ":" + kind {
	case "shortcut:ad":
		adID, err := strconv.ParseInt(nativeID, 10, 64)
		if err != nil {
			return SaleListing{}, fmt.Errorf("parse shortcut ad id: %w", err)
		}
		row, err := s.queries.GetShortcutAdUnifiedDetail(ctx, adID)
		if err != nil {
			return SaleListing{}, mapNotFound(err)
		}
		if row.ShortcutAdType != "listing" {
			return SaleListing{}, fmt.Errorf("%w: not a sale listing", ErrNotFound)
		}
		return saleFromShortcutAd(canonicalID, nativeID, row), nil
	case "frontdoor:ad":
		row, err := s.queries.GetFrontdoorAdUnifiedDetail(ctx, nativeID)
		if err != nil {
			return SaleListing{}, mapNotFound(err)
		}
		return saleFromFrontdoorAd(canonicalID, nativeID, row), nil
	case "frontdoor:announcement":
		announcementID, err := uuid.Parse(nativeID)
		if err != nil {
			return SaleListing{}, fmt.Errorf("parse frontdoor announcement id: %w", err)
		}
		row, err := s.queries.GetFrontdoorAnnouncementUnifiedDetail(ctx, announcementID)
		if err != nil {
			return SaleListing{}, mapNotFound(err)
		}
		if row.FrontdoorBuildingAnnouncementRentPeriod != nil || row.FrontdoorBuildingAnnouncementRentalUniqueNo != nil {
			return SaleListing{}, fmt.Errorf("%w: not a sale listing", ErrNotFound)
		}
		return saleFromFrontdoorAnnouncement(canonicalID, nativeID, row), nil
	default:
		return SaleListing{}, fmt.Errorf("%w: unsupported sale listing id", ErrNotFound)
	}
}

func (s *Service) RentalByID(ctx context.Context, input string, shortcutBase string, frontdoorBase string) (Rental, error) {
	canonicalID, err := s.resolveListingInput(ctx, input, "rental", shortcutBase, frontdoorBase)
	if err != nil {
		return Rental{}, err
	}
	source, kind, nativeID, err := ads.ParseCanonicalID(canonicalID)
	if err != nil {
		return Rental{}, err
	}
	switch source + ":" + kind {
	case "shortcut:ad":
		adID, err := strconv.ParseInt(nativeID, 10, 64)
		if err != nil {
			return Rental{}, fmt.Errorf("parse shortcut ad id: %w", err)
		}
		row, err := s.queries.GetShortcutAdUnifiedDetail(ctx, adID)
		if err != nil {
			return Rental{}, mapNotFound(err)
		}
		if row.ShortcutAdType != "rental" {
			return Rental{}, fmt.Errorf("%w: not a rental", ErrNotFound)
		}
		return rentalFromShortcutAd(canonicalID, nativeID, row), nil
	case "frontdoor:announcement":
		announcementID, err := uuid.Parse(nativeID)
		if err != nil {
			return Rental{}, fmt.Errorf("parse frontdoor announcement id: %w", err)
		}
		row, err := s.queries.GetFrontdoorAnnouncementUnifiedDetail(ctx, announcementID)
		if err != nil {
			return Rental{}, mapNotFound(err)
		}
		if row.FrontdoorBuildingAnnouncementRentPeriod == nil && row.FrontdoorBuildingAnnouncementRentalUniqueNo == nil {
			return Rental{}, fmt.Errorf("%w: not a rental", ErrNotFound)
		}
		return rentalFromFrontdoorAnnouncement(canonicalID, nativeID, row), nil
	default:
		return Rental{}, fmt.Errorf("%w: unsupported rental id", ErrNotFound)
	}
}

func (s *Service) BuildingByID(ctx context.Context, input string, shortcutBase string, frontdoorBase string) (Building, error) {
	canonicalID, err := s.resolveBuildingInput(ctx, input, shortcutBase, frontdoorBase)
	if err != nil {
		return Building{}, err
	}
	source, kind, nativeID, err := ads.ParseCanonicalID(canonicalID)
	if err != nil {
		return Building{}, err
	}
	switch source + ":" + kind {
	case "shortcut:building":
		buildingID, err := uuid.Parse(nativeID)
		if err != nil {
			return Building{}, fmt.Errorf("parse shortcut building id: %w", err)
		}
		row, err := s.queries.GetShortcutBuildingUnifiedDetail(ctx, buildingID)
		if err != nil {
			return Building{}, mapNotFound(err)
		}
		return buildingFromShortcut(canonicalID, nativeID, row), nil
	case "frontdoor:building":
		buildingID, err := uuid.Parse(nativeID)
		if err != nil {
			return Building{}, fmt.Errorf("parse frontdoor building id: %w", err)
		}
		row, err := s.queries.GetFrontdoorBuildingUnifiedDetail(ctx, buildingID)
		if err != nil {
			return Building{}, mapNotFound(err)
		}
		return buildingFromFrontdoor(canonicalID, nativeID, row), nil
	default:
		return Building{}, fmt.Errorf("%w: unsupported building id", ErrNotFound)
	}
}

func (s *Service) resolveListingInput(ctx context.Context, input string, listingType string, shortcutBase string, frontdoorBase string) (string, error) {
	if canonicalID, err := ads.ResolveInput(input, shortcutBase, frontdoorBase); err == nil {
		return canonicalID, nil
	}
	query := resolveSaleListingPublicIDSQL
	if listingType == "rental" {
		query = resolveRentalPublicIDSQL
	}
	var canonicalID string
	if err := s.db.QueryRow(ctx, query, strings.TrimSpace(input)).Scan(&canonicalID); err != nil {
		return "", mapNotFound(err)
	}
	return canonicalID, nil
}

func (s *Service) resolveBuildingInput(ctx context.Context, input string, shortcutBase string, frontdoorBase string) (string, error) {
	if canonicalID, err := ads.ResolveInput(input, shortcutBase, frontdoorBase); err == nil {
		return canonicalID, nil
	}
	var canonicalID string
	if err := s.db.QueryRow(ctx, resolveBuildingPublicIDSQL, strings.TrimSpace(input)).Scan(&canonicalID); err != nil {
		return "", mapNotFound(err)
	}
	return canonicalID, nil
}

func mapNotFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

type listingSearchRow struct {
	Source             string
	Kind               string
	NativeID           string
	CanonicalID        string
	PublicID           string
	URL                string
	Headline           string
	Address            string
	City               string
	Postal             string
	Price              *int64
	Area               *float64
	RoomLayout         string
	PricePerM2         *float64
	DebtFreePrice      *int64
	DebtShareAmount    *int64
	RoomsCount         *int32
	FloorLevel         *int32
	TotalFloors        *int32
	BuildYear          *int32
	Condition          *string
	EnergyClass        *string
	LastSeenAt         *string
	PublishedAt        *string
	BuildingKeyAddress string
}

func (r listingSearchRow) toSaleSummary() SaleListingSummary {
	source := ListingSource{Provider: r.Source, Kind: r.Kind, CanonicalID: r.CanonicalID, NativeID: r.NativeID, URL: r.URL, OriginalURL: r.URL}
	location := Location{StreetAddress: r.Address, City: r.City, Postal: r.Postal}
	identity := computedBuildingIdentity(r.Source, r.Kind, r.NativeID, location, "", "", "")
	return SaleListingSummary{ID: r.PublicID, Source: source, Headline: r.Headline, Unit: UnitDetails{Location: location, RoomLayout: r.RoomLayout, RoomsCount: r.RoomsCount, AreaM2: r.Area, FloorLevel: r.FloorLevel, Condition: valueOrEmpty(r.Condition)}, Building: BuildingDetails{Identity: identity, Location: location, BuildYear: r.BuildYear, FloorCount: r.TotalFloors, EnergyClass: valueOrEmpty(r.EnergyClass)}, Commercial: CommercialDetails{AskingPrice: r.Price, DebtFreePrice: r.DebtFreePrice, DebtShareAmount: r.DebtShareAmount, PricePerSquareMeter: r.PricePerM2, LastSeenAt: parseTimeString(r.LastSeenAt), PublishedAt: parseTimeString(r.PublishedAt)}}
}

func (r listingSearchRow) toRentalSummary() RentalSummary {
	source := ListingSource{Provider: r.Source, Kind: r.Kind, CanonicalID: r.CanonicalID, NativeID: r.NativeID, URL: r.URL, OriginalURL: r.URL}
	location := Location{StreetAddress: r.Address, City: r.City, Postal: r.Postal}
	identity := computedBuildingIdentity(r.Source, r.Kind, r.NativeID, location, "", "", "")
	return RentalSummary{ID: r.PublicID, Source: source, Headline: r.Headline, Unit: UnitDetails{Location: location, RoomLayout: r.RoomLayout, AreaM2: r.Area}, Building: BuildingDetails{Identity: identity, Location: location}, Commercial: CommercialDetails{Rent: r.Price, RentPeriod: "month", LastSeenAt: parseTimeString(r.LastSeenAt), PublishedAt: parseTimeString(r.PublishedAt)}}
}

func parseTimeString(value *string) *time.Time {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(*value))
	if err != nil {
		return nil
	}
	return &t
}
