-- name: ListSaleListingValuationClaimTargets :many
WITH linked AS (
    SELECT pu.property_unit_id, pu.physical_building_id, pu.housing_company_id
    FROM public.target_sources source_link
    JOIN public.property_offerings po ON po.property_offering_id = source_link.target_id
    JOIN public.property_units pu ON pu.property_unit_id = po.property_unit_id
    WHERE source_link.target_type = 'listing'
        AND source_link.source_type = 'source_listing'
        AND source_link.source_id = $1
        AND source_link.link_status <> 'rejected'
    ORDER BY source_link.link_score DESC, source_link.updated_at DESC
    LIMIT 1
)
SELECT 'sale_listing'::text AS entity_type, $1::uuid AS entity_id
UNION ALL SELECT 'property_unit'::text, property_unit_id FROM linked WHERE property_unit_id IS NOT NULL
UNION ALL SELECT 'physical_building'::text, physical_building_id FROM linked WHERE physical_building_id IS NOT NULL
UNION ALL SELECT 'housing_company'::text, housing_company_id FROM linked WHERE housing_company_id IS NOT NULL;

-- name: GetSaleListingSourceMediaData :one
SELECT
    sl.sale_listing_source_provider,
    sl.sale_listing_source_kind,
    COALESCE(sa.shortcut_ad_data, '{}'::jsonb) AS shortcut_ad_data,
    COALESCE(fa.frontdoor_ad_data, '{}'::jsonb) AS frontdoor_ad_data,
    COALESCE(fba.frontdoor_building_announcement_main_image_uri, '') AS frontdoor_building_announcement_main_image_uri
FROM public.property_source_offerings sl
LEFT JOIN public.shortcut_ads sa ON sa.shortcut_ad_id = sl.shortcut_ad_id
LEFT JOIN public.frontdoor_ads fa ON fa.frontdoor_ad_id = sl.frontdoor_ad_id
LEFT JOIN public.frontdoor_building_announcements fba ON fba.frontdoor_building_announcement_id = sl.frontdoor_building_announcement_id
WHERE sl.sale_listing_id = $1;

-- name: ListSaleListingFallbackRenovations :many
SELECT
    property_source_offering_renovation_category,
    property_source_offering_renovation_status,
    property_source_offering_renovation_year,
    COALESCE(property_source_offering_renovation_component, '') AS property_source_offering_renovation_component,
    COALESCE(property_source_offering_renovation_scope, '') AS property_source_offering_renovation_scope,
    COALESCE(property_source_offering_renovation_stage, '') AS property_source_offering_renovation_stage,
    COALESCE(property_source_offering_renovation_responsibility, '') AS property_source_offering_renovation_responsibility,
    property_source_offering_renovation_cost_estimate_eur,
    COALESCE(property_source_offering_renovation_text, '') AS property_source_offering_renovation_text,
    property_source_offering_renovation_confidence,
    property_source_offering_renovation_source_field
FROM public.property_source_offering_renovations
WHERE sale_listing_id = $1
ORDER BY property_source_offering_renovation_category, property_source_offering_renovation_year NULLS LAST;

-- name: ListHousingCompanyRenovationEvents :many
SELECT
    event.category,
    COALESCE(event.year, event.start_year)::int4 AS year
FROM public.property_renovation_events event
WHERE event.target_type = 'housing_company'
    AND event.target_id = $1
    AND event.category <> ''
    AND event.status = 'done'
ORDER BY event.category, COALESCE(event.year, event.start_year) NULLS LAST;

-- name: ListBuildingOfferingSourceListingIDs :many
SELECT
    sl.sale_listing_id
FROM public.property_units pu
JOIN public.property_offerings po ON po.property_unit_id = pu.property_unit_id
JOIN public.target_sources source_link ON source_link.target_type = 'listing'
    AND source_link.target_id = po.property_offering_id
    AND source_link.source_type = 'source_listing'
JOIN public.property_source_offerings sl ON sl.sale_listing_id = source_link.source_id
WHERE pu.housing_company_id = $1
    AND source_link.link_status <> 'rejected'
    AND sl.sale_listing_source_kind IN ('ad', 'announcement')
ORDER BY
    CASE WHEN sl.sale_listing_source_kind = 'ad' THEN 0 ELSE 1 END,
    sl.sale_listing_last_seen_at DESC NULLS LAST,
    source_link.link_score DESC,
    sl.sale_listing_created_at DESC
LIMIT 200;

-- name: ResolveRentalPublicID :one
WITH unified AS (
    SELECT ('shortcut:ad:' || sa.shortcut_ad_id::text) AS canonical_id
    FROM public.shortcut_ads sa
    WHERE sa.shortcut_ad_type = 'rental'
    UNION ALL
    SELECT ('frontdoor:announcement:' || fba.frontdoor_building_announcement_id::text) AS canonical_id
    FROM public.frontdoor_building_announcements fba
    WHERE fba.frontdoor_building_announcement_rent_period IS NOT NULL OR fba.frontdoor_building_announcement_rental_unique_no IS NOT NULL
)
SELECT canonical_id
FROM unified
WHERE ('r_' || substr(md5(canonical_id), 1, 16)) = $1
LIMIT 1;

-- name: ResolveBuildingPublicID :one
WITH unified AS (
    SELECT ('shortcut:building:' || sb.shortcut_building_id::text) AS canonical_id
    FROM public.shortcut_buildings sb
    UNION ALL
    SELECT ('frontdoor:building:' || fb.frontdoor_building_id::text) AS canonical_id
    FROM public.frontdoor_buildings fb
)
SELECT canonical_id
FROM unified
WHERE ('b_' || substr(md5(canonical_id), 1, 16)) = $1
LIMIT 1;
