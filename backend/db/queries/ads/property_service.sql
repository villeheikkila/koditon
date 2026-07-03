-- name: ListSaleListingValuationClaimTargets :many
WITH linked AS (
    SELECT pu.property_unit_id, pu.physical_building_id, pu.housing_company_id
    FROM public.listing_search_documents doc
    JOIN public.property_offerings po ON po.property_offering_id = doc.property_offering_id
    JOIN public.property_units pu ON pu.property_unit_id = po.property_unit_id
    WHERE doc.primary_source_listing_id = $1
        AND doc.listing_status = 'active'
    ORDER BY doc.last_seen_at DESC NULLS LAST, doc.refreshed_at DESC
    LIMIT 1
)
SELECT 'sale_listing'::text AS entity_type, $1::uuid AS entity_id
UNION ALL SELECT 'property_unit'::text, property_unit_id FROM linked WHERE property_unit_id IS NOT NULL
UNION ALL SELECT 'physical_building'::text, physical_building_id FROM linked WHERE physical_building_id IS NOT NULL
UNION ALL SELECT 'housing_company'::text, housing_company_id FROM linked WHERE housing_company_id IS NOT NULL;

-- name: GetSaleListingSourceMediaData :one
SELECT
    evidence.provider AS sale_listing_source_provider,
    CASE
        WHEN evidence.source_kind = 'frontdoor_building_announcement' THEN 'announcement'
        WHEN evidence.source_kind = 'frontdoor_ad' THEN 'ad'
        WHEN evidence.source_kind = 'shortcut_ad' THEN 'ad'
        ELSE evidence.source_kind
    END AS sale_listing_source_kind,
    COALESCE(sa.shortcut_ad_data, '{}'::jsonb) AS shortcut_ad_data,
    COALESCE(fa.frontdoor_ad_data, '{}'::jsonb) AS frontdoor_ad_data,
    fba.frontdoor_building_announcement_main_image_uri AS frontdoor_building_announcement_main_image_uri
FROM public.listing_search_documents doc
JOIN public.evidence_sources evidence ON evidence.evidence_source_id = doc.primary_evidence_source_id
LEFT JOIN origin.shortcut_ads sa ON sa.shortcut_ad_id = evidence.shortcut_ad_id
LEFT JOIN origin.frontdoor_ads fa ON fa.frontdoor_ad_id = evidence.frontdoor_ad_id
LEFT JOIN origin.frontdoor_building_announcements fba ON fba.frontdoor_building_announcement_id = evidence.frontdoor_building_announcement_id
WHERE doc.primary_source_listing_id = $1
    AND doc.listing_status = 'active'
ORDER BY doc.last_seen_at DESC NULLS LAST, doc.refreshed_at DESC
LIMIT 1;

-- name: ListSaleListingFallbackRenovations :many
SELECT
    source_listing_renovation_category,
    source_listing_renovation_status,
    source_listing_renovation_year,
    source_listing_renovation_component AS source_listing_renovation_component,
    source_listing_renovation_scope AS source_listing_renovation_scope,
    source_listing_renovation_stage AS source_listing_renovation_stage,
    source_listing_renovation_responsibility AS source_listing_renovation_responsibility,
    source_listing_renovation_cost_estimate_eur,
    source_listing_renovation_text AS source_listing_renovation_text,
    source_listing_renovation_confidence,
    source_listing_renovation_source_field
FROM public.source_listing_renovations
WHERE source_listing_id = $1
ORDER BY source_listing_renovation_category, source_listing_renovation_year NULLS LAST;

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
    doc.primary_source_listing_id AS sale_listing_id
FROM public.property_units pu
JOIN public.property_offerings po ON po.property_unit_id = pu.property_unit_id
JOIN public.listing_search_documents doc ON doc.property_offering_id = po.property_offering_id
WHERE pu.housing_company_id = $1
    AND doc.listing_status = 'active'
    AND doc.kind IN ('ad', 'announcement')
    AND doc.primary_source_listing_id IS NOT NULL
ORDER BY
    CASE WHEN doc.kind = 'ad' THEN 0 ELSE 1 END,
    doc.last_seen_at DESC NULLS LAST,
    doc.refreshed_at DESC
LIMIT 200;

-- name: ResolveRentalPublicID :one
SELECT doc.canonical_id
FROM public.listing_search_documents doc
WHERE doc.listing_type = 'rental'
    AND doc.listing_status = 'active'
    AND ('r_' || substr(md5(doc.canonical_id), 1, 16)) = $1
ORDER BY doc.last_seen_at DESC NULLS LAST
LIMIT 1;

-- name: ResolveBuildingPublicID :one
SELECT physical_building_id::text AS canonical_id
FROM public.physical_buildings
WHERE ('b_' || substr(md5(physical_building_id::text), 1, 16)) = $1
LIMIT 1;
