-- name: GetDimensionApartmentProfileForSaleListing :one
WITH linked AS (
    SELECT pu.property_unit_id, pu.housing_company_id
    FROM public.listing_search_documents doc
    JOIN public.property_offerings po ON po.property_offering_id = doc.property_offering_id
    JOIN public.property_units pu ON pu.property_unit_id = po.property_unit_id
    WHERE doc.primary_source_listing_id = $1
        AND doc.listing_status = 'active'
    ORDER BY doc.last_seen_at DESC NULLS LAST, doc.refreshed_at DESC
    LIMIT 1
)
SELECT
    linked.housing_company_id,
    linked.property_unit_id,
    (p.dimensions #>> '{unit,area_m2}')::double precision AS area_m2,
    (p.dimensions #>> '{unit,living_area_m2}')::double precision AS living_area_m2,
    NULLIF(p.dimensions #>> '{layout,room_layout}', '')::text AS room_layout,
    (p.dimensions #>> '{layout,room_count}')::integer AS room_count,
    (p.dimensions #>> '{layout,bedroom_count}')::integer AS bedroom_count,
    (p.dimensions #>> '{unit,floor_level}')::integer AS floor_level,
    (p.dimensions #>> '{unit,total_floors}')::integer AS total_floors,
    NULLIF(p.dimensions #>> '{layout,kitchen_type}', '')::text AS kitchen_type,
    NULLIF(p.dimensions #>> '{layout,quality}', '')::text AS layout_quality,
    (p.dimensions #>> '{layout,awkward}')::boolean AS awkward_layout,
    NULLIF(p.dimensions #>> '{condition,unit_condition}', '')::text AS condition,
    NULLIF(p.dimensions #>> '{condition,kitchen_condition}', '')::text AS kitchen_condition,
    NULLIF(p.dimensions #>> '{condition,bathroom_condition}', '')::text AS bathroom_condition,
    (p.dimensions #>> '{condition,surface_renovation_need}')::boolean AS surface_renovation_need,
    (p.dimensions #>> '{condition,modernization_need}')::boolean AS modernization_need,
    (p.dimensions #>> '{features,sauna}')::boolean AS sauna,
    (p.dimensions #>> '{features,balcony}')::boolean AS balcony,
    (p.dimensions #>> '{features,balcony_glazing}')::boolean AS balcony_glazing,
    NULLIF(p.dimensions #>> '{features,parking_type}', '')::text AS parking_type,
    NULLIF(p.dimensions #>> '{features,storage_quality}', '')::text AS storage_quality,
    NULLIF(p.dimensions #>> '{features,view_quality}', '')::text AS view_quality,
    (p.dimensions #>> '{features,noise_risk}')::boolean AS noise_risk,
    NULLIF(p.dimensions #>> '{features,accessibility}', '')::text AS accessibility,
    (p.dimensions #>> '{charges,maintenance_monthly_eur}')::double precision AS maintenance_charge_monthly,
    (p.dimensions #>> '{charges,capital_monthly_eur}')::double precision AS capital_charge_monthly,
    (p.dimensions #>> '{charges,total_monthly_eur}')::double precision AS total_charge_monthly,
    (p.dimensions #>> '{charges,debt_share_eur}')::bigint AS debt_share_eur,
    NULLIF(p.dimensions #>> '{risk,shareholder_liability}', '')::text AS shareholder_liability,
    'medium'::text AS confidence,
    p.resolved_at
FROM linked
JOIN public.dimension_profiles p ON p.target_type = 'unit'
    AND p.target_id = linked.property_unit_id;

-- name: GetCanonicalBuildingProfileForSaleListing :one
WITH linked AS (
    SELECT pu.physical_building_id, pu.housing_company_id
    FROM public.listing_search_documents doc
    JOIN public.property_offerings po ON po.property_offering_id = doc.property_offering_id
    JOIN public.property_units pu ON pu.property_unit_id = po.property_unit_id
    WHERE doc.primary_source_listing_id = $1
        AND doc.listing_status = 'active'
    ORDER BY doc.last_seen_at DESC NULLS LAST, doc.refreshed_at DESC
    LIMIT 1
)
SELECT
    linked.physical_building_id,
    linked.housing_company_id AS building_housing_company_id,
    (bp.dimensions #>> '{building,build_year}')::integer AS build_year,
    (bp.dimensions #>> '{building,floor_count}')::integer AS floor_count,
    (bp.dimensions #>> '{building,apartment_count}')::integer AS apartment_count,
    NULLIF(bp.dimensions #>> '{building,energy_class}', '')::text AS energy_class,
    NULLIF(bp.dimensions #>> '{building,heating_method}', '')::text AS heating_method,
    NULLIF(bp.dimensions #>> '{building,material}', '')::text AS material,
    NULLIF(bp.dimensions #>> '{building,roof_type}', '')::text AS roof_type,
    NULLIF(bp.dimensions #>> '{building,roof_material}', '')::text AS roof_material,
    (bp.dimensions #>> '{building,elevator}')::boolean AS elevator,
    CASE WHEN bp.target_id IS NULL THEN '' ELSE 'medium' END AS building_confidence,
    bp.resolved_at AS building_resolved_at,
    linked.housing_company_id,
    NULLIF(hcp.dimensions #>> '{housing_company,name}', '')::text AS housing_company_name,
    NULLIF(hcp.dimensions #>> '{housing_company,business_id}', '')::text AS business_id,
    NULL::integer AS housing_company_build_year,
    (hcp.dimensions #>> '{housing_company,apartment_count}')::integer AS housing_company_apartment_count,
    NULLIF(hcp.dimensions #>> '{site,plot_ownership_type}', '')::text AS plot_ownership_type,
    ''::text AS housing_company_energy_class,
    NULLIF(hcp.dimensions #>> '{risk,maintenance_risk}', '')::text AS maintenance_risk,
    NULLIF(hcp.dimensions #>> '{risk,financial_risk}', '')::text AS financial_risk,
    NULLIF(hcp.dimensions #>> '{risk,repair_backlog_risk}', '')::text AS repair_backlog_risk,
    CASE WHEN hcp.target_id IS NULL THEN '' ELSE 'medium' END AS housing_company_confidence,
    hcp.resolved_at AS housing_company_resolved_at
FROM linked
LEFT JOIN public.dimension_profiles bp ON bp.target_type = 'building'
    AND bp.target_id = linked.physical_building_id
LEFT JOIN public.dimension_profiles hcp ON hcp.target_type = 'housing_company'
    AND hcp.target_id = linked.housing_company_id;

-- name: ListSaleListingQualityScores :many
WITH linked AS (
    SELECT po.property_offering_id, pu.property_unit_id, pu.physical_building_id, pu.housing_company_id
    FROM public.listing_search_documents doc
    JOIN public.property_offerings po ON po.property_offering_id = doc.property_offering_id
    JOIN public.property_units pu ON pu.property_unit_id = po.property_unit_id
    WHERE doc.primary_source_listing_id = $1
        AND doc.listing_status = 'active'
    ORDER BY doc.last_seen_at DESC NULLS LAST, doc.refreshed_at DESC
    LIMIT 1
),
targets AS (
    SELECT 'offering'::text AS target_type, property_offering_id AS target_id FROM linked WHERE property_offering_id IS NOT NULL
    UNION ALL SELECT 'unit', property_unit_id FROM linked WHERE property_unit_id IS NOT NULL
    UNION ALL SELECT 'building', physical_building_id FROM linked WHERE physical_building_id IS NOT NULL
    UNION ALL SELECT 'housing_company', housing_company_id FROM linked WHERE housing_company_id IS NOT NULL
)
SELECT
    CASE dv.target_type
        WHEN 'offering' THEN 'property_offering'
        WHEN 'unit' THEN 'property_unit'
        WHEN 'building' THEN 'physical_building'
        ELSE dv.target_type
    END::text AS target_type,
    substring(dv.dimension_key from position('.' in dv.dimension_key) + 1)::text AS dimension,
    round((dv.confidence * 100)::numeric)::integer AS value,
    CASE WHEN dv.confidence >= 0.8 THEN 'high' WHEN dv.confidence >= 0.6 THEN 'medium' ELSE 'low' END AS confidence,
    jsonb_build_array(dv.selected_reason) AS reasons,
    dv.resolved_at
FROM targets
JOIN public.dimension_values dv
    ON dv.target_type = targets.target_type
    AND dv.target_id = targets.target_id
    AND dv.dimension_key LIKE 'score.%'
ORDER BY
    CASE dv.target_type WHEN 'offering' THEN 1 WHEN 'unit' THEN 2 WHEN 'building' THEN 3 WHEN 'housing_company' THEN 4 ELSE 5 END,
    dv.dimension_key;
