CREATE TABLE IF NOT EXISTS public.property_target_sources (
    property_target_source_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    target_type text NOT NULL,
    target_id uuid NOT NULL,
    source_provider text NOT NULL,
    source_kind text NOT NULL,
    source_table text NOT NULL,
    source_id uuid,
    source_id_value text NOT NULL,
    source_external_id text,
    source_url text,
    link_status text NOT NULL DEFAULT 'confirmed',
    link_method text NOT NULL,
    link_score integer NOT NULL DEFAULT 100,
    link_reasons jsonb NOT NULL DEFAULT '{}'::jsonb,
    first_seen_at timestamptz,
    last_seen_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (target_type = ANY (ARRAY['offering','unit','building','housing_company','document','transaction']::text[])),
    CHECK (link_status = ANY (ARRAY['confirmed','candidate','rejected']::text[]))
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_property_target_sources_unique_source
ON public.property_target_sources (
    target_type,
    target_id,
    source_provider,
    source_kind,
    source_table,
    source_id_value
);
CREATE INDEX IF NOT EXISTS idx_property_target_sources_target
ON public.property_target_sources (target_type, target_id, link_status);
CREATE INDEX IF NOT EXISTS idx_property_target_sources_source
ON public.property_target_sources (source_table, source_id_value, link_status);
CREATE INDEX IF NOT EXISTS idx_physical_buildings_lat_lng
ON public.physical_buildings (physical_building_latitude, physical_building_longitude)
WHERE physical_building_latitude IS NOT NULL
  AND physical_building_longitude IS NOT NULL;
INSERT INTO public.property_target_sources (
    target_type,
    target_id,
    source_provider,
    source_kind,
    source_table,
    source_id,
    source_id_value,
    source_external_id,
    source_url,
    link_status,
    link_method,
    link_score,
    link_reasons,
    first_seen_at,
    last_seen_at,
    created_at,
    updated_at
)
SELECT DISTINCT ON (target_type, target_id, source_provider, source_kind, source_table, source_id_value) *
FROM (
    SELECT
        'offering'::text AS target_type,
        pos.property_offering_id AS target_id,
        sl.sale_listing_source_provider AS source_provider,
        sl.sale_listing_source_kind AS source_kind,
        'property_source_offerings'::text AS source_table,
        sl.sale_listing_id AS source_id,
        sl.sale_listing_id::text AS source_id_value,
        sl.sale_listing_native_id AS source_external_id,
        sl.sale_listing_url AS source_url,
        pos.property_offering_source_link_status AS link_status,
        pos.property_offering_source_link_method AS link_method,
        pos.property_offering_source_link_score AS link_score,
        pos.property_offering_source_link_reasons AS link_reasons,
        sl.sale_listing_first_seen_at AS first_seen_at,
        sl.sale_listing_last_seen_at AS last_seen_at,
        pos.property_offering_source_created_at AS created_at,
        pos.property_offering_source_updated_at AS updated_at
    FROM public.property_offering_sources pos
    JOIN public.property_source_offerings sl ON sl.sale_listing_id = pos.sale_listing_id
) rows
ORDER BY target_type, target_id, source_provider, source_kind, source_table, source_id_value, link_status, last_seen_at DESC NULLS LAST
ON CONFLICT (target_type, target_id, source_provider, source_kind, source_table, source_id_value) DO UPDATE SET
    source_id = COALESCE(EXCLUDED.source_id, property_target_sources.source_id),
    source_external_id = COALESCE(EXCLUDED.source_external_id, property_target_sources.source_external_id),
    source_url = COALESCE(EXCLUDED.source_url, property_target_sources.source_url),
    link_status = EXCLUDED.link_status,
    link_method = EXCLUDED.link_method,
    link_score = EXCLUDED.link_score,
    link_reasons = property_target_sources.link_reasons || EXCLUDED.link_reasons,
    first_seen_at = LEAST(COALESCE(property_target_sources.first_seen_at, EXCLUDED.first_seen_at), COALESCE(EXCLUDED.first_seen_at, property_target_sources.first_seen_at)),
    last_seen_at = GREATEST(COALESCE(property_target_sources.last_seen_at, EXCLUDED.last_seen_at), COALESCE(EXCLUDED.last_seen_at, property_target_sources.last_seen_at)),
    updated_at = now();
INSERT INTO public.property_target_sources (
    target_type,
    target_id,
    source_provider,
    source_kind,
    source_table,
    source_id,
    source_id_value,
    source_external_id,
    source_url,
    link_status,
    link_method,
    link_score,
    link_reasons,
    first_seen_at,
    last_seen_at,
    created_at,
    updated_at
)
SELECT DISTINCT ON (target_type, target_id, source_provider, source_kind, source_table, source_id_value) *
FROM (
    SELECT
        'housing_company'::text AS target_type,
        housing_company_id AS target_id,
        housing_company_source_provider AS source_provider,
        housing_company_source_kind AS source_kind,
        housing_company_source_table AS source_table,
        housing_company_source_id AS source_id,
        housing_company_source_id_value AS source_id_value,
        housing_company_source_external_id AS source_external_id,
        housing_company_source_url AS source_url,
        housing_company_source_link_status AS link_status,
        housing_company_source_link_method AS link_method,
        housing_company_source_link_score AS link_score,
        housing_company_source_link_reasons AS link_reasons,
        housing_company_source_first_seen_at AS first_seen_at,
        housing_company_source_last_seen_at AS last_seen_at,
        housing_company_source_created_at AS created_at,
        housing_company_source_updated_at AS updated_at
    FROM public.housing_company_sources
) rows
ORDER BY target_type, target_id, source_provider, source_kind, source_table, source_id_value, link_status, last_seen_at DESC NULLS LAST
ON CONFLICT (target_type, target_id, source_provider, source_kind, source_table, source_id_value) DO UPDATE SET
    source_id = COALESCE(EXCLUDED.source_id, property_target_sources.source_id),
    source_external_id = COALESCE(EXCLUDED.source_external_id, property_target_sources.source_external_id),
    source_url = COALESCE(EXCLUDED.source_url, property_target_sources.source_url),
    link_status = EXCLUDED.link_status,
    link_method = EXCLUDED.link_method,
    link_score = EXCLUDED.link_score,
    link_reasons = property_target_sources.link_reasons || EXCLUDED.link_reasons,
    first_seen_at = LEAST(COALESCE(property_target_sources.first_seen_at, EXCLUDED.first_seen_at), COALESCE(EXCLUDED.first_seen_at, property_target_sources.first_seen_at)),
    last_seen_at = GREATEST(COALESCE(property_target_sources.last_seen_at, EXCLUDED.last_seen_at), COALESCE(EXCLUDED.last_seen_at, property_target_sources.last_seen_at)),
    updated_at = now();
INSERT INTO public.property_target_sources (
    target_type,
    target_id,
    source_provider,
    source_kind,
    source_table,
    source_id,
    source_id_value,
    source_external_id,
    source_url,
    link_status,
    link_method,
    link_score,
    link_reasons,
    first_seen_at,
    last_seen_at
)
SELECT DISTINCT ON (target_type, target_id, source_provider, source_kind, source_table, source_id_value) *
FROM (
    SELECT
        CASE WHEN pu.physical_building_id IS NULL THEN 'housing_company' ELSE 'building' END AS target_type,
        COALESCE(pu.physical_building_id, pu.housing_company_id) AS target_id,
        'shortcut'::text AS source_provider,
        'building'::text AS source_kind,
        'shortcut_buildings'::text AS source_table,
        sb.shortcut_building_id AS source_id,
        sb.shortcut_building_id::text AS source_id_value,
        sb.shortcut_building_external_id::text AS source_external_id,
        sb.shortcut_building_url AS source_url,
        'confirmed'::text AS link_status,
        'derived_from_listing'::text AS link_method,
        90 AS link_score,
        jsonb_build_object('source', 'linked_listing') AS link_reasons,
        sb.shortcut_building_created_at AS first_seen_at,
        COALESCE(sb.shortcut_building_updated_at, sb.shortcut_building_processed_at) AS last_seen_at
    FROM public.property_offering_sources pos
    JOIN public.property_offerings po ON po.property_offering_id = pos.property_offering_id
    JOIN public.property_units pu ON pu.property_unit_id = po.property_unit_id
    JOIN public.property_source_offerings sl ON sl.sale_listing_id = pos.sale_listing_id
    JOIN public.shortcut_ads sa ON sa.shortcut_ad_id = sl.shortcut_ad_id
    JOIN public.shortcut_buildings sb ON sb.shortcut_building_id = sa.shortcut_building_id
    WHERE pos.property_offering_source_link_status <> 'rejected'
) rows
ORDER BY target_type, target_id, source_provider, source_kind, source_table, source_id_value, last_seen_at DESC NULLS LAST
ON CONFLICT (target_type, target_id, source_provider, source_kind, source_table, source_id_value) DO UPDATE SET
    source_external_id = COALESCE(EXCLUDED.source_external_id, property_target_sources.source_external_id),
    source_url = COALESCE(EXCLUDED.source_url, property_target_sources.source_url),
    link_status = EXCLUDED.link_status,
    link_score = GREATEST(property_target_sources.link_score, EXCLUDED.link_score),
    link_reasons = property_target_sources.link_reasons || EXCLUDED.link_reasons,
    last_seen_at = GREATEST(COALESCE(property_target_sources.last_seen_at, EXCLUDED.last_seen_at), COALESCE(EXCLUDED.last_seen_at, property_target_sources.last_seen_at)),
    updated_at = now();
INSERT INTO public.property_target_sources (
    target_type,
    target_id,
    source_provider,
    source_kind,
    source_table,
    source_id,
    source_id_value,
    source_external_id,
    source_url,
    link_status,
    link_method,
    link_score,
    link_reasons,
    first_seen_at,
    last_seen_at
)
SELECT DISTINCT ON (target_type, target_id, source_provider, source_kind, source_table, source_id_value) *
FROM (
    SELECT
        CASE WHEN pu.physical_building_id IS NULL THEN 'housing_company' ELSE 'building' END AS target_type,
        COALESCE(pu.physical_building_id, pu.housing_company_id) AS target_id,
        'frontdoor'::text AS source_provider,
        'building'::text AS source_kind,
        'frontdoor_buildings'::text AS source_table,
        fb.frontdoor_building_id AS source_id,
        fb.frontdoor_building_id::text AS source_id_value,
        COALESCE(fb.frontdoor_building_housing_company_friendly_id, fb.frontdoor_building_housing_company_id::text) AS source_external_id,
        fb.frontdoor_building_url AS source_url,
        'confirmed'::text AS link_status,
        'derived_from_listing'::text AS link_method,
        90 AS link_score,
        jsonb_build_object('source', 'linked_listing') AS link_reasons,
        fb.frontdoor_building_first_seen_at AS first_seen_at,
        fb.frontdoor_building_last_seen_at AS last_seen_at
    FROM public.property_offering_sources pos
    JOIN public.property_offerings po ON po.property_offering_id = pos.property_offering_id
    JOIN public.property_units pu ON pu.property_unit_id = po.property_unit_id
    JOIN public.property_source_offerings sl ON sl.sale_listing_id = pos.sale_listing_id
    JOIN public.frontdoor_building_announcements fba ON fba.frontdoor_building_announcement_id = sl.frontdoor_building_announcement_id
    JOIN public.frontdoor_buildings fb ON fb.frontdoor_building_id = fba.frontdoor_building_id
    WHERE pos.property_offering_source_link_status <> 'rejected'
) rows
ORDER BY target_type, target_id, source_provider, source_kind, source_table, source_id_value, last_seen_at DESC NULLS LAST
ON CONFLICT (target_type, target_id, source_provider, source_kind, source_table, source_id_value) DO UPDATE SET
    source_external_id = COALESCE(EXCLUDED.source_external_id, property_target_sources.source_external_id),
    source_url = COALESCE(EXCLUDED.source_url, property_target_sources.source_url),
    link_status = EXCLUDED.link_status,
    link_score = GREATEST(property_target_sources.link_score, EXCLUDED.link_score),
    link_reasons = property_target_sources.link_reasons || EXCLUDED.link_reasons,
    last_seen_at = GREATEST(COALESCE(property_target_sources.last_seen_at, EXCLUDED.last_seen_at), COALESCE(EXCLUDED.last_seen_at, property_target_sources.last_seen_at)),
    updated_at = now();
UPDATE public.physical_buildings pb
SET physical_building_latitude = coordinates.lat,
    physical_building_longitude = coordinates.lng,
    physical_building_updated_at = now()
FROM (
    SELECT DISTINCT ON (pu.physical_building_id)
        pu.physical_building_id,
        COALESCE(fb.frontdoor_building_latitude, sb.shortcut_building_latitude, sl.sale_listing_latitude, postgis.ST_Y(hc.housing_company_geom)::double precision) AS lat,
        COALESCE(fb.frontdoor_building_longitude, sb.shortcut_building_longitude, sl.sale_listing_longitude, postgis.ST_X(hc.housing_company_geom)::double precision) AS lng
    FROM public.property_units pu
    LEFT JOIN public.housing_companies hc ON hc.housing_company_id = pu.housing_company_id
    LEFT JOIN public.property_offerings po ON po.property_unit_id = pu.property_unit_id
    LEFT JOIN public.property_offering_sources pos ON pos.property_offering_id = po.property_offering_id
        AND pos.property_offering_source_link_status <> 'rejected'
    LEFT JOIN public.property_source_offerings sl ON sl.sale_listing_id = pos.sale_listing_id
    LEFT JOIN public.shortcut_ads sa ON sa.shortcut_ad_id = sl.shortcut_ad_id
    LEFT JOIN public.shortcut_buildings sb ON sb.shortcut_building_id = sa.shortcut_building_id
    LEFT JOIN public.frontdoor_building_announcements fba ON fba.frontdoor_building_announcement_id = sl.frontdoor_building_announcement_id
    LEFT JOIN public.frontdoor_buildings fb ON fb.frontdoor_building_id = fba.frontdoor_building_id
    WHERE pu.physical_building_id IS NOT NULL
    ORDER BY pu.physical_building_id,
        (fb.frontdoor_building_latitude IS NOT NULL AND fb.frontdoor_building_longitude IS NOT NULL) DESC,
        (sb.shortcut_building_latitude IS NOT NULL AND sb.shortcut_building_longitude IS NOT NULL) DESC,
        (sl.sale_listing_latitude IS NOT NULL AND sl.sale_listing_longitude IS NOT NULL) DESC,
        sl.sale_listing_last_seen_at DESC NULLS LAST
) coordinates
WHERE pb.physical_building_id = coordinates.physical_building_id
  AND coordinates.lat IS NOT NULL
  AND coordinates.lng IS NOT NULL
  AND (pb.physical_building_latitude IS NULL OR pb.physical_building_longitude IS NULL);
INSERT INTO public.property_dimension_dirty_targets (target_type, target_id, dirty_reasons, dirty_at)
SELECT DISTINCT target_type, target_id, ARRAY['property_target_sources_backfill'], now()
FROM public.property_target_sources
WHERE link_status <> 'rejected'
ON CONFLICT (target_type, target_id) DO UPDATE SET
    dirty_reasons = ARRAY(SELECT DISTINCT unnest(property_dimension_dirty_targets.dirty_reasons || EXCLUDED.dirty_reasons)),
    dirty_at = GREATEST(property_dimension_dirty_targets.dirty_at, EXCLUDED.dirty_at);
