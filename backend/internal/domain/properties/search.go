package properties

import (
	"context"
	"fmt"
)

const searchSaleListingsSQL = `
WITH unified AS (
    SELECT 'shortcut'::text AS source, 'ad'::text AS kind, sa.shortcut_ad_id::text AS native_id, ('shortcut:ad:' || sa.shortcut_ad_id::text) AS canonical_id, sa.shortcut_ad_url AS url, COALESCE(sa.shortcut_ad_street_address, sb.shortcut_building_address, sa.shortcut_ad_id::text) AS headline, COALESCE(sa.shortcut_ad_street_address, sb.shortcut_building_address) AS address, sa.shortcut_ad_city AS city, sa.shortcut_ad_postal AS postal, sa.shortcut_ad_price AS price, sa.shortcut_ad_area_value AS area, sa.shortcut_ad_data #>> '{adData,roomConfiguration}' AS room_layout, sa.shortcut_ad_last_seen_at AS last_seen_at, (sa.shortcut_ad_data #>> '{adData,published}')::timestamptz AS published_at, concat_ws(' ', sa.shortcut_ad_search_text, sb.shortcut_building_address, sb.shortcut_building_housing_company) AS searchable
    FROM public.shortcut_ads sa
    LEFT JOIN public.shortcut_buildings sb ON sb.shortcut_building_id = sa.shortcut_building_id
    WHERE sa.shortcut_ad_type = 'listing'
    UNION ALL
    SELECT 'frontdoor'::text AS source, 'ad'::text AS kind, fa.frontdoor_ad_external_id AS native_id, ('frontdoor:ad:' || fa.frontdoor_ad_external_id) AS canonical_id, fa.frontdoor_ad_url AS url, COALESCE(fa.frontdoor_ad_street_address, fa.frontdoor_ad_external_id) AS headline, fa.frontdoor_ad_street_address AS address, fa.frontdoor_ad_city AS city, fa.frontdoor_ad_postal AS postal, fa.frontdoor_ad_price AS price, fa.frontdoor_ad_area_value AS area, fa.frontdoor_ad_data #>> '{residenceDetailsDTO,roomStructure}' AS room_layout, fa.frontdoor_ad_last_seen_at AS last_seen_at, fa.frontdoor_ad_publishing_time AS published_at, fa.frontdoor_ad_search_text AS searchable
    FROM public.frontdoor_ads fa
    UNION ALL
    SELECT 'frontdoor'::text AS source, 'announcement'::text AS kind, fba.frontdoor_building_announcement_id::text AS native_id, ('frontdoor:announcement:' || fba.frontdoor_building_announcement_id::text) AS canonical_id, fb.frontdoor_building_url AS url, COALESCE(fba.frontdoor_building_announcement_address_line1, fba.frontdoor_building_announcement_friendly_id, fba.frontdoor_building_announcement_external_id::text, fba.frontdoor_building_announcement_id::text) AS headline, concat_ws(' ', fba.frontdoor_building_announcement_address_line1, fba.frontdoor_building_announcement_address_line2) AS address, COALESCE(fba.frontdoor_building_announcement_location, fb.frontdoor_building_municipality, fb.frontdoor_building_post_area) AS city, fb.frontdoor_building_postcode AS postal, CASE WHEN fba.frontdoor_building_announcement_search_price IS NULL THEN NULL ELSE fba.frontdoor_building_announcement_search_price::bigint END AS price, fba.frontdoor_building_announcement_area AS area, fba.frontdoor_building_announcement_room_structure AS room_layout, fba.frontdoor_building_announcement_last_seen_at AS last_seen_at, NULL::timestamptz AS published_at, concat_ws(' ', fba.frontdoor_building_announcement_id::text, fba.frontdoor_building_announcement_external_id::text, fba.frontdoor_building_announcement_friendly_id, fba.frontdoor_building_announcement_address_line1, fba.frontdoor_building_announcement_address_line2, fba.frontdoor_building_announcement_location, fb.frontdoor_building_postcode, fb.frontdoor_building_municipality, fb.frontdoor_building_post_area, fb.frontdoor_building_url, fba.frontdoor_building_announcement_room_structure) AS searchable
    FROM public.frontdoor_building_announcements fba
    JOIN public.frontdoor_buildings fb ON fb.frontdoor_building_id = fba.frontdoor_building_id
    WHERE fba.frontdoor_building_announcement_rent_period IS NULL AND fba.frontdoor_building_announcement_rental_unique_no IS NULL
)
SELECT source, kind, native_id, canonical_id, url, headline, address, city, postal, price, area, room_layout, last_seen_at::text, published_at::text, address
FROM unified u
WHERE ($4 = 'all' OR u.source = $4)
  AND ($5::text IS NULL OR trim($5::text) = '' OR lower(u.searchable) LIKE ('%' || lower(trim($5::text)) || '%'))
  AND ($6::text IS NULL OR trim($6::text) = '' OR lower(COALESCE(u.city, '')) LIKE ('%' || lower(trim($6::text)) || '%'))
  AND ($7::text IS NULL OR trim($7::text) = '' OR lower(COALESCE(u.postal, '')) LIKE ('%' || lower(trim($7::text)) || '%'))
  AND ($8::bigint IS NULL OR u.price >= $8::bigint)
  AND ($9::bigint IS NULL OR u.price <= $9::bigint)
  AND ($10::float8 IS NULL OR u.area >= $10::float8)
  AND ($11::float8 IS NULL OR u.area <= $11::float8)
  AND ($12::timestamptz IS NULL OR u.published_at >= $12::timestamptz)
  AND ($13::timestamptz IS NULL OR u.published_at <= $13::timestamptz)
ORDER BY
    CASE WHEN $1 = 'price_asc' THEN price END ASC NULLS LAST,
    CASE WHEN $1 = 'price_desc' THEN price END DESC NULLS LAST,
    CASE WHEN $1 = 'area_asc' THEN area END ASC NULLS LAST,
    CASE WHEN $1 = 'area_desc' THEN area END DESC NULLS LAST,
    CASE WHEN $1 = 'seen_desc' THEN last_seen_at END DESC NULLS LAST,
    last_seen_at DESC
LIMIT $3::int OFFSET $2::int`

const searchRentalsSQL = `
WITH unified AS (
    SELECT 'shortcut'::text AS source, 'ad'::text AS kind, sa.shortcut_ad_id::text AS native_id, ('shortcut:ad:' || sa.shortcut_ad_id::text) AS canonical_id, sa.shortcut_ad_url AS url, COALESCE(sa.shortcut_ad_street_address, sb.shortcut_building_address, sa.shortcut_ad_id::text) AS headline, COALESCE(sa.shortcut_ad_street_address, sb.shortcut_building_address) AS address, sa.shortcut_ad_city AS city, sa.shortcut_ad_postal AS postal, sa.shortcut_ad_price AS price, sa.shortcut_ad_area_value AS area, sa.shortcut_ad_data #>> '{adData,roomConfiguration}' AS room_layout, sa.shortcut_ad_last_seen_at AS last_seen_at, (sa.shortcut_ad_data #>> '{adData,published}')::timestamptz AS published_at, concat_ws(' ', sa.shortcut_ad_search_text, sb.shortcut_building_address, sb.shortcut_building_housing_company) AS searchable
    FROM public.shortcut_ads sa
    LEFT JOIN public.shortcut_buildings sb ON sb.shortcut_building_id = sa.shortcut_building_id
    WHERE sa.shortcut_ad_type = 'rental'
    UNION ALL
    SELECT 'frontdoor'::text AS source, 'announcement'::text AS kind, fba.frontdoor_building_announcement_id::text AS native_id, ('frontdoor:announcement:' || fba.frontdoor_building_announcement_id::text) AS canonical_id, fb.frontdoor_building_url AS url, COALESCE(fba.frontdoor_building_announcement_address_line1, fba.frontdoor_building_announcement_friendly_id, fba.frontdoor_building_announcement_external_id::text, fba.frontdoor_building_announcement_id::text) AS headline, concat_ws(' ', fba.frontdoor_building_announcement_address_line1, fba.frontdoor_building_announcement_address_line2) AS address, COALESCE(fba.frontdoor_building_announcement_location, fb.frontdoor_building_municipality, fb.frontdoor_building_post_area) AS city, fb.frontdoor_building_postcode AS postal, CASE WHEN fba.frontdoor_building_announcement_search_price IS NULL THEN NULL ELSE fba.frontdoor_building_announcement_search_price::bigint END AS price, fba.frontdoor_building_announcement_area AS area, fba.frontdoor_building_announcement_room_structure AS room_layout, fba.frontdoor_building_announcement_last_seen_at AS last_seen_at, NULL::timestamptz AS published_at, concat_ws(' ', fba.frontdoor_building_announcement_id::text, fba.frontdoor_building_announcement_external_id::text, fba.frontdoor_building_announcement_friendly_id, fba.frontdoor_building_announcement_address_line1, fba.frontdoor_building_announcement_address_line2, fba.frontdoor_building_announcement_location, fb.frontdoor_building_postcode, fb.frontdoor_building_municipality, fb.frontdoor_building_post_area, fb.frontdoor_building_url, fba.frontdoor_building_announcement_room_structure) AS searchable
    FROM public.frontdoor_building_announcements fba
    JOIN public.frontdoor_buildings fb ON fb.frontdoor_building_id = fba.frontdoor_building_id
    WHERE fba.frontdoor_building_announcement_rent_period IS NOT NULL OR fba.frontdoor_building_announcement_rental_unique_no IS NOT NULL
)
SELECT source, kind, native_id, canonical_id, url, headline, address, city, postal, price, area, room_layout, last_seen_at::text, published_at::text, address
FROM unified u
WHERE ($4 = 'all' OR u.source = $4)
  AND ($5::text IS NULL OR trim($5::text) = '' OR lower(u.searchable) LIKE ('%' || lower(trim($5::text)) || '%'))
  AND ($6::text IS NULL OR trim($6::text) = '' OR lower(COALESCE(u.city, '')) LIKE ('%' || lower(trim($6::text)) || '%'))
  AND ($7::text IS NULL OR trim($7::text) = '' OR lower(COALESCE(u.postal, '')) LIKE ('%' || lower(trim($7::text)) || '%'))
  AND ($8::bigint IS NULL OR u.price >= $8::bigint)
  AND ($9::bigint IS NULL OR u.price <= $9::bigint)
  AND ($10::float8 IS NULL OR u.area >= $10::float8)
  AND ($11::float8 IS NULL OR u.area <= $11::float8)
  AND ($12::timestamptz IS NULL OR u.published_at >= $12::timestamptz)
  AND ($13::timestamptz IS NULL OR u.published_at <= $13::timestamptz)
ORDER BY
    CASE WHEN $1 = 'price_asc' THEN price END ASC NULLS LAST,
    CASE WHEN $1 = 'price_desc' THEN price END DESC NULLS LAST,
    CASE WHEN $1 = 'area_asc' THEN area END ASC NULLS LAST,
    CASE WHEN $1 = 'area_desc' THEN area END DESC NULLS LAST,
    CASE WHEN $1 = 'seen_desc' THEN last_seen_at END DESC NULLS LAST,
    last_seen_at DESC
LIMIT $3::int OFFSET $2::int`

const countSaleListingsSQL = `
WITH unified AS (
    SELECT 'shortcut'::text AS source, sa.shortcut_ad_city AS city, sa.shortcut_ad_postal AS postal, sa.shortcut_ad_price AS price, sa.shortcut_ad_area_value AS area, (sa.shortcut_ad_data #>> '{adData,published}')::timestamptz AS published_at, concat_ws(' ', sa.shortcut_ad_search_text, sb.shortcut_building_address, sb.shortcut_building_housing_company) AS searchable
    FROM public.shortcut_ads sa
    LEFT JOIN public.shortcut_buildings sb ON sb.shortcut_building_id = sa.shortcut_building_id
    WHERE sa.shortcut_ad_type = 'listing'
    UNION ALL
    SELECT 'frontdoor'::text AS source, fa.frontdoor_ad_city AS city, fa.frontdoor_ad_postal AS postal, fa.frontdoor_ad_price AS price, fa.frontdoor_ad_area_value AS area, fa.frontdoor_ad_publishing_time AS published_at, fa.frontdoor_ad_search_text AS searchable
    FROM public.frontdoor_ads fa
    UNION ALL
    SELECT 'frontdoor'::text AS source, COALESCE(fba.frontdoor_building_announcement_location, fb.frontdoor_building_municipality, fb.frontdoor_building_post_area) AS city, fb.frontdoor_building_postcode AS postal, CASE WHEN fba.frontdoor_building_announcement_search_price IS NULL THEN NULL ELSE fba.frontdoor_building_announcement_search_price::bigint END AS price, fba.frontdoor_building_announcement_area AS area, NULL::timestamptz AS published_at, concat_ws(' ', fba.frontdoor_building_announcement_id::text, fba.frontdoor_building_announcement_external_id::text, fba.frontdoor_building_announcement_friendly_id, fba.frontdoor_building_announcement_address_line1, fba.frontdoor_building_announcement_address_line2, fba.frontdoor_building_announcement_location, fb.frontdoor_building_postcode, fb.frontdoor_building_municipality, fb.frontdoor_building_post_area, fb.frontdoor_building_url, fba.frontdoor_building_announcement_room_structure) AS searchable
    FROM public.frontdoor_building_announcements fba
    JOIN public.frontdoor_buildings fb ON fb.frontdoor_building_id = fba.frontdoor_building_id
    WHERE fba.frontdoor_building_announcement_rent_period IS NULL AND fba.frontdoor_building_announcement_rental_unique_no IS NULL
)
SELECT count(*)::bigint
FROM unified u
WHERE ($1 = 'all' OR u.source = $1)
  AND ($2::text IS NULL OR trim($2::text) = '' OR lower(u.searchable) LIKE ('%' || lower(trim($2::text)) || '%'))
  AND ($3::text IS NULL OR trim($3::text) = '' OR lower(COALESCE(u.city, '')) LIKE ('%' || lower(trim($3::text)) || '%'))
  AND ($4::text IS NULL OR trim($4::text) = '' OR lower(COALESCE(u.postal, '')) LIKE ('%' || lower(trim($4::text)) || '%'))
  AND ($5::bigint IS NULL OR u.price >= $5::bigint)
  AND ($6::bigint IS NULL OR u.price <= $6::bigint)
  AND ($7::float8 IS NULL OR u.area >= $7::float8)
  AND ($8::float8 IS NULL OR u.area <= $8::float8)
  AND ($9::timestamptz IS NULL OR u.published_at >= $9::timestamptz)
  AND ($10::timestamptz IS NULL OR u.published_at <= $10::timestamptz)`

const countRentalsSQL = `
WITH unified AS (
    SELECT 'shortcut'::text AS source, sa.shortcut_ad_city AS city, sa.shortcut_ad_postal AS postal, sa.shortcut_ad_price AS price, sa.shortcut_ad_area_value AS area, (sa.shortcut_ad_data #>> '{adData,published}')::timestamptz AS published_at, concat_ws(' ', sa.shortcut_ad_search_text, sb.shortcut_building_address, sb.shortcut_building_housing_company) AS searchable
    FROM public.shortcut_ads sa
    LEFT JOIN public.shortcut_buildings sb ON sb.shortcut_building_id = sa.shortcut_building_id
    WHERE sa.shortcut_ad_type = 'rental'
    UNION ALL
    SELECT 'frontdoor'::text AS source, COALESCE(fba.frontdoor_building_announcement_location, fb.frontdoor_building_municipality, fb.frontdoor_building_post_area) AS city, fb.frontdoor_building_postcode AS postal, CASE WHEN fba.frontdoor_building_announcement_search_price IS NULL THEN NULL ELSE fba.frontdoor_building_announcement_search_price::bigint END AS price, fba.frontdoor_building_announcement_area AS area, NULL::timestamptz AS published_at, concat_ws(' ', fba.frontdoor_building_announcement_id::text, fba.frontdoor_building_announcement_external_id::text, fba.frontdoor_building_announcement_friendly_id, fba.frontdoor_building_announcement_address_line1, fba.frontdoor_building_announcement_address_line2, fba.frontdoor_building_announcement_location, fb.frontdoor_building_postcode, fb.frontdoor_building_municipality, fb.frontdoor_building_post_area, fb.frontdoor_building_url, fba.frontdoor_building_announcement_room_structure) AS searchable
    FROM public.frontdoor_building_announcements fba
    JOIN public.frontdoor_buildings fb ON fb.frontdoor_building_id = fba.frontdoor_building_id
    WHERE fba.frontdoor_building_announcement_rent_period IS NOT NULL OR fba.frontdoor_building_announcement_rental_unique_no IS NOT NULL
)
SELECT count(*)::bigint
FROM unified u
WHERE ($1 = 'all' OR u.source = $1)
  AND ($2::text IS NULL OR trim($2::text) = '' OR lower(u.searchable) LIKE ('%' || lower(trim($2::text)) || '%'))
  AND ($3::text IS NULL OR trim($3::text) = '' OR lower(COALESCE(u.city, '')) LIKE ('%' || lower(trim($3::text)) || '%'))
  AND ($4::text IS NULL OR trim($4::text) = '' OR lower(COALESCE(u.postal, '')) LIKE ('%' || lower(trim($4::text)) || '%'))
  AND ($5::bigint IS NULL OR u.price >= $5::bigint)
  AND ($6::bigint IS NULL OR u.price <= $6::bigint)
  AND ($7::float8 IS NULL OR u.area >= $7::float8)
  AND ($8::float8 IS NULL OR u.area <= $8::float8)
  AND ($9::timestamptz IS NULL OR u.published_at >= $9::timestamptz)
  AND ($10::timestamptz IS NULL OR u.published_at <= $10::timestamptz)`

func (s *Service) searchListings(ctx context.Context, params SearchParams, listingType string) ([]listingSearchRow, error) {
	query := searchSaleListingsSQL
	if listingType == "rental" {
		query = searchRentalsSQL
	}
	rows, err := s.db.Query(ctx, query, params.Sort, (params.Page-1)*params.PageSize, params.PageSize, params.Source, emptyToNil(params.Query), emptyToNil(params.City), emptyToNil(params.Postal), params.MinPrice, params.MaxPrice, params.MinArea, params.MaxArea, params.PublishedAfter, params.PublishedBefore)
	if err != nil {
		return nil, fmt.Errorf("search %s listings: %w", listingType, err)
	}
	defer rows.Close()
	out := []listingSearchRow{}
	for rows.Next() {
		var row listingSearchRow
		if err := rows.Scan(&row.Source, &row.Kind, &row.NativeID, &row.CanonicalID, &row.URL, &row.Headline, &row.Address, &row.City, &row.Postal, &row.Price, &row.Area, &row.RoomLayout, &row.LastSeenAt, &row.PublishedAt, &row.BuildingKeyAddress); err != nil {
			return nil, fmt.Errorf("scan %s listing: %w", listingType, err)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate %s listings: %w", listingType, err)
	}
	return out, nil
}

func (s *Service) countListings(ctx context.Context, params SearchParams, listingType string) (int64, error) {
	query := countSaleListingsSQL
	if listingType == "rental" {
		query = countRentalsSQL
	}
	var count int64
	err := s.db.QueryRow(ctx, query, params.Source, emptyToNil(params.Query), emptyToNil(params.City), emptyToNil(params.Postal), params.MinPrice, params.MaxPrice, params.MinArea, params.MaxArea, params.PublishedAfter, params.PublishedBefore).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count %s listings: %w", listingType, err)
	}
	return count, nil
}
