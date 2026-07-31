-- name: UpsertFrontdoorBuildingAnnouncementSourceListing :one
WITH source AS (
    SELECT
        gen_random_uuid() AS source_listing_id,
        'frontdoor'::text AS provider,
        'announcement'::text AS source_kind,
        fba.frontdoor_building_announcement_id::text AS native_id,
        'frontdoor:announcement:' || fba.frontdoor_building_announcement_id::text AS canonical_source_id,
        NULL::bigint AS shortcut_ad_id,
        NULL::uuid AS frontdoor_ad_id,
        fba.frontdoor_building_announcement_id,
        fb.frontdoor_building_url AS url,
        NULL::text AS payload_hash,
        fba.frontdoor_building_announcement_data_normalized_version AS normalized_version,
        now() AS normalized_at,
        fba.frontdoor_building_announcement_first_seen_at AS first_seen_at,
        fba.frontdoor_building_announcement_last_seen_at AS last_seen_at,
        COALESCE(fba.frontdoor_building_announcement_first_seen_at, now()) AS created_at,
        now() AS updated_at
    FROM origin.frontdoor_building_announcements fba
    JOIN origin.frontdoor_buildings fb ON fb.frontdoor_building_id = fba.frontdoor_building_id
    WHERE fba.frontdoor_building_announcement_id = @frontdoor_building_announcement_id
        AND fba.frontdoor_building_announcement_identity_key NOT LIKE 'legacy:%'
        AND fba.frontdoor_building_announcement_rent_period IS NULL
        AND fba.frontdoor_building_announcement_rental_unique_no IS NULL
),
updated AS (
    UPDATE origin.source_listings target
    SET
        canonical_source_id = source.canonical_source_id,
        provider = source.provider,
        source_kind = source.source_kind,
        native_id = source.native_id,
        shortcut_ad_id = source.shortcut_ad_id,
        frontdoor_ad_id = source.frontdoor_ad_id,
        frontdoor_building_announcement_id = source.frontdoor_building_announcement_id,
        url = source.url,
        payload_hash = source.payload_hash,
        normalized_version = source.normalized_version,
        normalized_at = source.normalized_at,
        first_seen_at = source.first_seen_at,
        last_seen_at = source.last_seen_at,
        updated_at = source.updated_at
    FROM source
    WHERE target.canonical_source_id = source.canonical_source_id
        OR (target.provider = source.provider AND target.source_kind = source.source_kind AND target.native_id = source.native_id)
    RETURNING target.source_listing_id
),
inserted AS (
    INSERT INTO origin.source_listings (
        source_listing_id,
        provider,
        source_kind,
        native_id,
        canonical_source_id,
        shortcut_ad_id,
        frontdoor_ad_id,
        frontdoor_building_announcement_id,
        url,
        payload_hash,
        normalized_version,
        normalized_at,
        first_seen_at,
        last_seen_at,
        created_at,
        updated_at
    )
    SELECT
        source.source_listing_id,
        source.provider,
        source.source_kind,
        source.native_id,
        source.canonical_source_id,
        source.shortcut_ad_id,
        source.frontdoor_ad_id,
        source.frontdoor_building_announcement_id,
        source.url,
        source.payload_hash,
        source.normalized_version,
        source.normalized_at,
        source.first_seen_at,
        source.last_seen_at,
        source.created_at,
        source.updated_at
    FROM source
    WHERE NOT EXISTS (SELECT 1 FROM updated)
    RETURNING source_listing_id
)
SELECT source_listing_id FROM updated
UNION ALL
SELECT source_listing_id FROM inserted
LIMIT 1;

-- name: UpsertFrontdoorAdSourceListing :one
WITH source AS (
    SELECT
        gen_random_uuid() AS source_listing_id,
        'frontdoor'::text AS provider,
        'ad'::text AS source_kind,
        fa.frontdoor_ad_external_id AS native_id,
        'frontdoor:ad:' || fa.frontdoor_ad_external_id AS canonical_source_id,
        NULL::bigint AS shortcut_ad_id,
        fa.frontdoor_ad_id,
        NULL::uuid AS frontdoor_building_announcement_id,
        fa.frontdoor_ad_url AS url,
        fa.frontdoor_ad_data_hash AS payload_hash,
        fa.frontdoor_ad_data_normalized_version AS normalized_version,
        now() AS normalized_at,
        fa.frontdoor_ad_first_seen_at AS first_seen_at,
        fa.frontdoor_ad_last_seen_at AS last_seen_at,
        COALESCE(fa.frontdoor_ad_first_seen_at, now()) AS created_at,
        now() AS updated_at
    FROM origin.frontdoor_ads fa
    WHERE fa.frontdoor_ad_id = @frontdoor_ad_id
        AND fa.frontdoor_ad_data IS NOT NULL
),
updated AS (
    UPDATE origin.source_listings target
    SET
        canonical_source_id = source.canonical_source_id,
        provider = source.provider,
        source_kind = source.source_kind,
        native_id = source.native_id,
        shortcut_ad_id = source.shortcut_ad_id,
        frontdoor_ad_id = source.frontdoor_ad_id,
        frontdoor_building_announcement_id = source.frontdoor_building_announcement_id,
        url = source.url,
        payload_hash = source.payload_hash,
        normalized_version = source.normalized_version,
        normalized_at = source.normalized_at,
        first_seen_at = source.first_seen_at,
        last_seen_at = source.last_seen_at,
        updated_at = source.updated_at
    FROM source
    WHERE target.canonical_source_id = source.canonical_source_id
        OR (target.provider = source.provider AND target.source_kind = source.source_kind AND target.native_id = source.native_id)
    RETURNING target.source_listing_id
),
inserted AS (
    INSERT INTO origin.source_listings (
        source_listing_id,
        provider,
        source_kind,
        native_id,
        canonical_source_id,
        shortcut_ad_id,
        frontdoor_ad_id,
        frontdoor_building_announcement_id,
        url,
        payload_hash,
        normalized_version,
        normalized_at,
        first_seen_at,
        last_seen_at,
        created_at,
        updated_at
    )
    SELECT
        source.source_listing_id,
        source.provider,
        source.source_kind,
        source.native_id,
        source.canonical_source_id,
        source.shortcut_ad_id,
        source.frontdoor_ad_id,
        source.frontdoor_building_announcement_id,
        source.url,
        source.payload_hash,
        source.normalized_version,
        source.normalized_at,
        source.first_seen_at,
        source.last_seen_at,
        source.created_at,
        source.updated_at
    FROM source
    WHERE NOT EXISTS (SELECT 1 FROM updated)
    RETURNING source_listing_id
)
SELECT source_listing_id FROM updated
UNION ALL
SELECT source_listing_id FROM inserted
LIMIT 1;

-- name: UpsertShortcutAdSourceListing :one
WITH source AS (
    SELECT
        gen_random_uuid() AS source_listing_id,
        'shortcut'::text AS provider,
        'ad'::text AS source_kind,
        sa.shortcut_ad_id::text AS native_id,
        'shortcut:ad:' || sa.shortcut_ad_id::text AS canonical_source_id,
        sa.shortcut_ad_id,
        NULL::uuid AS frontdoor_ad_id,
        NULL::uuid AS frontdoor_building_announcement_id,
        sa.shortcut_ad_url AS url,
        sa.shortcut_ad_data_hash AS payload_hash,
        sa.shortcut_ad_data_normalized_version AS normalized_version,
        now() AS normalized_at,
        sa.shortcut_ad_first_seen_at AS first_seen_at,
        sa.shortcut_ad_last_seen_at AS last_seen_at,
        COALESCE(sa.shortcut_ad_first_seen_at, now()) AS created_at,
        now() AS updated_at
    FROM origin.shortcut_ads sa
    WHERE sa.shortcut_ad_id = @shortcut_ad_id
        AND sa.shortcut_ad_type = 'listing'
        AND sa.shortcut_ad_data IS NOT NULL
),
updated AS (
    UPDATE origin.source_listings target
    SET
        canonical_source_id = source.canonical_source_id,
        provider = source.provider,
        source_kind = source.source_kind,
        native_id = source.native_id,
        shortcut_ad_id = source.shortcut_ad_id,
        frontdoor_ad_id = source.frontdoor_ad_id,
        frontdoor_building_announcement_id = source.frontdoor_building_announcement_id,
        url = source.url,
        payload_hash = source.payload_hash,
        normalized_version = source.normalized_version,
        normalized_at = source.normalized_at,
        first_seen_at = source.first_seen_at,
        last_seen_at = source.last_seen_at,
        updated_at = source.updated_at
    FROM source
    WHERE target.canonical_source_id = source.canonical_source_id
        OR (target.provider = source.provider AND target.source_kind = source.source_kind AND target.native_id = source.native_id)
    RETURNING target.source_listing_id
),
inserted AS (
    INSERT INTO origin.source_listings (
        source_listing_id,
        provider,
        source_kind,
        native_id,
        canonical_source_id,
        shortcut_ad_id,
        frontdoor_ad_id,
        frontdoor_building_announcement_id,
        url,
        payload_hash,
        normalized_version,
        normalized_at,
        first_seen_at,
        last_seen_at,
        created_at,
        updated_at
    )
    SELECT
        source.source_listing_id,
        source.provider,
        source.source_kind,
        source.native_id,
        source.canonical_source_id,
        source.shortcut_ad_id,
        source.frontdoor_ad_id,
        source.frontdoor_building_announcement_id,
        source.url,
        source.payload_hash,
        source.normalized_version,
        source.normalized_at,
        source.first_seen_at,
        source.last_seen_at,
        source.created_at,
        source.updated_at
    FROM source
    WHERE NOT EXISTS (SELECT 1 FROM updated)
    RETURNING source_listing_id
)
SELECT source_listing_id FROM updated
UNION ALL
SELECT source_listing_id FROM inserted
LIMIT 1;

-- name: DeleteFrontdoorBuildingAnnouncementSourceListing :one
WITH target AS (
    SELECT source_listing.source_listing_id, target_source.target_id
    FROM origin.source_listings source_listing
    LEFT JOIN public.target_sources target_source ON target_source.target_type = 'listing'
        AND target_source.source_type = 'source_listing'
        AND target_source.source_id = source_listing.source_listing_id
        AND target_source.link_status = 'confirmed'
    WHERE source_listing.provider = 'frontdoor'
        AND source_listing.source_kind = 'announcement'
        AND source_listing.frontdoor_building_announcement_id = sqlc.arg(frontdoor_building_announcement_id)
), deleted_source AS (
    DELETE FROM origin.source_listings source_listing
    USING target
    WHERE source_listing.source_listing_id = target.source_listing_id
    RETURNING source_listing.source_listing_id
), deleted_target_source AS (
    DELETE FROM public.target_sources target_source
    USING target
    WHERE target_source.target_type = 'listing'
        AND target_source.source_type = 'source_listing'
        AND target_source.source_id = target.source_listing_id
    RETURNING target_source.target_source_id
), replacement AS (
    SELECT replacement_source.source_id AS source_listing_id
    FROM target
    JOIN public.target_sources replacement_source ON replacement_source.target_type = 'listing'
        AND replacement_source.target_id = target.target_id
        AND replacement_source.source_type = 'source_listing'
        AND replacement_source.source_id <> target.source_listing_id
        AND replacement_source.link_status = 'confirmed'
    JOIN origin.source_listings source_listing ON source_listing.source_listing_id = replacement_source.source_id
    ORDER BY
        CASE WHEN source_listing.source_kind = 'ad' THEN 0 ELSE 1 END,
        source_listing.last_seen_at DESC NULLS LAST,
        source_listing.canonical_source_id
    LIMIT 1
), deleted_listing AS (
    DELETE FROM public.listings listing
    USING target
    WHERE listing.listing_id = target.target_id
        AND NOT EXISTS (SELECT 1 FROM replacement)
    RETURNING listing.listing_id
)
SELECT (SELECT source_listing_id FROM replacement LIMIT 1)::uuid AS replacement_source_listing_id
FROM deleted_source;

-- name: DeleteShortcutAdSourceListing :one
WITH target AS (
    SELECT source_listing.source_listing_id, target_source.target_id
    FROM origin.source_listings source_listing
    LEFT JOIN public.target_sources target_source ON target_source.target_type = 'listing'
        AND target_source.source_type = 'source_listing'
        AND target_source.source_id = source_listing.source_listing_id
        AND target_source.link_status = 'confirmed'
    WHERE source_listing.provider = 'shortcut'
        AND source_listing.source_kind = 'ad'
        AND source_listing.shortcut_ad_id = sqlc.arg(shortcut_ad_id)
), deleted_source AS (
    DELETE FROM origin.source_listings source_listing
    USING target
    WHERE source_listing.source_listing_id = target.source_listing_id
    RETURNING source_listing.source_listing_id
), deleted_target_source AS (
    DELETE FROM public.target_sources target_source
    USING target
    WHERE target_source.target_type = 'listing'
        AND target_source.source_type = 'source_listing'
        AND target_source.source_id = target.source_listing_id
    RETURNING target_source.target_source_id
), replacement AS (
    SELECT replacement_source.source_id AS source_listing_id
    FROM target
    JOIN public.target_sources replacement_source ON replacement_source.target_type = 'listing'
        AND replacement_source.target_id = target.target_id
        AND replacement_source.source_type = 'source_listing'
        AND replacement_source.source_id <> target.source_listing_id
        AND replacement_source.link_status = 'confirmed'
    JOIN origin.source_listings source_listing ON source_listing.source_listing_id = replacement_source.source_id
    ORDER BY
        CASE WHEN source_listing.source_kind = 'ad' THEN 0 ELSE 1 END,
        source_listing.last_seen_at DESC NULLS LAST,
        source_listing.canonical_source_id
    LIMIT 1
), deleted_listing AS (
    DELETE FROM public.listings listing
    USING target
    WHERE listing.listing_id = target.target_id
        AND NOT EXISTS (SELECT 1 FROM replacement)
    RETURNING listing.listing_id
)
SELECT (SELECT source_listing_id FROM replacement LIMIT 1)::uuid AS replacement_source_listing_id
FROM deleted_source;

-- name: ReconcileSourceListingModel :one
WITH announcement_source AS (
    SELECT
        NULL::uuid AS frontdoor_ad_id,
        fba.frontdoor_building_announcement_id,
        lower(trim(concat_ws(' ', fba.frontdoor_building_announcement_address_line1, fba.frontdoor_building_announcement_address_line2))) AS sale_listing_address_norm,
        fb.frontdoor_building_apartment_count AS sale_listing_apartment_count,
        fba.frontdoor_building_announcement_area AS sale_listing_area_value,
        CASE WHEN current_price.source_listing_id IS NULL THEN fba.frontdoor_building_announcement_search_price::bigint ELSE current_price.asking_price END AS sale_listing_asking_price,
        COALESCE(fba.frontdoor_building_announcement_construction_finished_year, fb.frontdoor_building_build_year, fb.frontdoor_building_construction_end_year) AS sale_listing_build_year,
        lower(trim(concat_ws(' ', fba.frontdoor_building_announcement_address_line1, fba.frontdoor_building_announcement_address_line2, fb.frontdoor_building_postcode, COALESCE(fba.frontdoor_building_announcement_location, fb.frontdoor_building_municipality, fb.frontdoor_building_post_area)))) AS sale_listing_building_match_key,
        sl.canonical_source_id AS sale_listing_canonical_id,
        COALESCE(fba.frontdoor_building_announcement_location, fb.frontdoor_building_municipality, fb.frontdoor_building_post_area) AS sale_listing_city,
        lower(trim(COALESCE(fba.frontdoor_building_announcement_location, fb.frontdoor_building_municipality, fb.frontdoor_building_post_area))) AS sale_listing_city_norm,
        NULL::text AS sale_listing_condition,
        current_price.debt_free_price AS sale_listing_debt_free_price,
        current_price.debt_share_amount AS sale_listing_debt_share_amount,
        concat_ws(' ', fb.frontdoor_building_description, fb.frontdoor_building_other_info) AS sale_listing_description_text,
        fb.frontdoor_building_has_elevator AS sale_listing_elevator,
        fb.frontdoor_building_energy_certificate_code AS sale_listing_energy_class,
        fb.frontdoor_building_energy_certificate_code AS sale_listing_energy_efficiency_label,
        fba.frontdoor_building_announcement_first_seen_at AS sale_listing_first_seen_at,
        NULL::integer AS sale_listing_floor_level,
        COALESCE(fba.frontdoor_building_announcement_address_line1, fba.frontdoor_building_announcement_friendly_id, fba.frontdoor_building_announcement_external_id::text, fba.frontdoor_building_announcement_id::text) AS sale_listing_headline,
        fb.frontdoor_building_business_id AS sale_listing_housing_company_business_id,
        fb.frontdoor_building_company_name AS sale_listing_housing_company_name,
        sl.source_listing_id AS sale_listing_id,
        fba.frontdoor_building_announcement_last_seen_at AS sale_listing_last_seen_at,
        fb.frontdoor_building_latitude AS sale_listing_latitude,
        NULL::double precision AS sale_listing_living_area_value,
        fb.frontdoor_building_longitude AS sale_listing_longitude,
        sl.native_id AS sale_listing_native_id,
        NULL::double precision AS sale_listing_plot_area_value,
        NULL::boolean AS sale_listing_plot_owned,
        fb.frontdoor_building_postcode AS sale_listing_postal,
        fb.frontdoor_building_postcode AS sale_listing_postal_norm,
        CASE WHEN current_price.source_listing_id IS NULL THEN fba.frontdoor_building_announcement_price_per_square ELSE current_price.price_per_m2 END AS sale_listing_price_per_m2,
        property_type.sale_listing_property_type_code AS sale_listing_property_type_code,
        NULL::timestamptz AS sale_listing_published_at,
        NULL::text AS sale_listing_room_category_code,
        fba.frontdoor_building_announcement_room_structure AS sale_listing_room_layout,
        NULL::integer AS sale_listing_rooms_count,
        concat_ws(' ', fba.frontdoor_building_announcement_id::text, fba.frontdoor_building_announcement_external_id::text, fba.frontdoor_building_announcement_friendly_id, fba.frontdoor_building_announcement_address_line1, fba.frontdoor_building_announcement_address_line2, fba.frontdoor_building_announcement_location, fb.frontdoor_building_postcode, fb.frontdoor_building_municipality, fb.frontdoor_building_post_area, fb.frontdoor_building_url, fba.frontdoor_building_announcement_room_structure, fb.frontdoor_building_company_name, fb.frontdoor_building_business_id) AS sale_listing_search_text,
        NULL::boolean AS sale_listing_sauna,
        sl.source_kind AS sale_listing_source_kind,
        sl.provider AS sale_listing_source_provider,
        concat_ws(' ', fba.frontdoor_building_announcement_address_line1, fba.frontdoor_building_announcement_address_line2) AS sale_listing_street_address,
        fb.frontdoor_building_floor_count AS sale_listing_total_floors,
        CASE
            WHEN NULLIF(fb.frontdoor_building_postcode, '') IS NOT NULL
                AND NULLIF(lower(trim(concat_ws(' ', fba.frontdoor_building_announcement_address_line1, fba.frontdoor_building_announcement_address_line2))), '') IS NOT NULL
                AND fba.frontdoor_building_announcement_area IS NOT NULL
                THEN concat_ws('|', 'unit_v1', regexp_replace(trim(fb.frontdoor_building_postcode), '[^0-9]+', '', 'g'), lower(trim(concat_ws(' ', fba.frontdoor_building_announcement_address_line1, fba.frontdoor_building_announcement_address_line2))), round(fba.frontdoor_building_announcement_area::numeric * 10)::text)
            ELSE sl.canonical_source_id
        END AS sale_listing_unit_match_key,
        sl.url AS sale_listing_url,
        fba.frontdoor_building_announcement_new_building AS sale_listing_new_development,
        NULL::boolean AS sale_listing_balcony,
        CASE WHEN fba.frontdoor_building_announcement_published IS FALSE THEN 'removed'::text ELSE 'active'::text END AS sale_listing_status,
        NULL::bigint AS shortcut_ad_id
    FROM origin.source_listings sl
    JOIN origin.frontdoor_building_announcements fba ON fba.frontdoor_building_announcement_id = sl.frontdoor_building_announcement_id
    JOIN origin.frontdoor_buildings fb ON fb.frontdoor_building_id = fba.frontdoor_building_id
    LEFT JOIN origin.source_listing_price_periods current_price ON current_price.source_listing_id = sl.source_listing_id AND current_price.superseded_at IS NULL
    LEFT JOIN origin.sale_listing_property_type_aliases property_type
        ON property_type.sale_listing_property_type_alias = lower(trim(COALESCE(fba.frontdoor_building_announcement_property_subtype, fba.frontdoor_building_announcement_property_type)))
    WHERE sl.source_listing_id = $1::uuid
        AND sl.provider = 'frontdoor'
        AND sl.source_kind = 'announcement'
        AND fba.frontdoor_building_announcement_identity_key NOT LIKE 'legacy:%'
),
shortcut_source AS (
    SELECT
        NULL::uuid AS frontdoor_ad_id,
        NULL::uuid AS frontdoor_building_announcement_id,
        lower(trim(COALESCE(raw.street_address, sb.shortcut_building_address))) AS sale_listing_address_norm,
        NULL::integer AS sale_listing_apartment_count,
        raw.area AS sale_listing_area_value,
        CASE WHEN current_price.source_listing_id IS NULL THEN raw.price ELSE current_price.asking_price END AS sale_listing_asking_price,
        COALESCE(raw.build_year, sb.shortcut_building_construction_year) AS sale_listing_build_year,
        lower(trim(concat_ws(' ', COALESCE(raw.street_address, sb.shortcut_building_address), raw.postal, raw.city))) AS sale_listing_building_match_key,
        sl.canonical_source_id AS sale_listing_canonical_id,
        raw.city AS sale_listing_city,
        lower(trim(raw.city)) AS sale_listing_city_norm,
        raw.condition AS sale_listing_condition,
        CASE WHEN current_price.source_listing_id IS NULL THEN raw.debt_free_price ELSE current_price.debt_free_price END AS sale_listing_debt_free_price,
        CASE WHEN current_price.source_listing_id IS NULL THEN raw.debt_share_amount ELSE current_price.debt_share_amount END AS sale_listing_debt_share_amount,
        raw.description_text AS sale_listing_description_text,
        NULL::boolean AS sale_listing_elevator,
        raw.energy_class AS sale_listing_energy_class,
        raw.energy_class AS sale_listing_energy_efficiency_label,
        sl.first_seen_at AS sale_listing_first_seen_at,
        raw.floor_level AS sale_listing_floor_level,
        COALESCE(raw.street_address, sb.shortcut_building_address, sa.shortcut_ad_id::text) AS sale_listing_headline,
        NULL::text AS sale_listing_housing_company_business_id,
        sb.shortcut_building_housing_company AS sale_listing_housing_company_name,
        sl.source_listing_id AS sale_listing_id,
        sa.shortcut_ad_last_seen_at AS sale_listing_last_seen_at,
        NULL::double precision AS sale_listing_latitude,
        raw.living_area AS sale_listing_living_area_value,
        NULL::double precision AS sale_listing_longitude,
        sl.native_id AS sale_listing_native_id,
        raw.plot_area AS sale_listing_plot_area_value,
        NULL::boolean AS sale_listing_plot_owned,
        raw.postal AS sale_listing_postal,
        raw.postal AS sale_listing_postal_norm,
        CASE WHEN current_price.source_listing_id IS NULL THEN COALESCE(raw.price_per_m2, CASE WHEN raw.price IS NOT NULL AND raw.area IS NOT NULL AND raw.area > 0 THEN raw.price::double precision / raw.area ELSE NULL END) ELSE COALESCE(current_price.price_per_m2, CASE WHEN current_price.asking_price IS NOT NULL AND raw.area IS NOT NULL AND raw.area > 0 THEN current_price.asking_price::double precision / raw.area ELSE NULL END) END AS sale_listing_price_per_m2,
        NULL::text AS sale_listing_property_type_code,
        (sa.shortcut_ad_data #>> '{adData,published}')::timestamptz AS sale_listing_published_at,
        NULL::text AS sale_listing_room_category_code,
        sa.shortcut_ad_data #>> '{adData,roomConfiguration}' AS sale_listing_room_layout,
        raw.rooms_count AS sale_listing_rooms_count,
        trim(concat_ws(' ', sa.shortcut_ad_id::text, sa.shortcut_ad_url, raw.street_address, raw.city, raw.postal, sa.shortcut_ad_data #>> '{adData,roomConfiguration}', sb.shortcut_building_address, sb.shortcut_building_housing_company)) AS sale_listing_search_text,
        raw.sauna AS sale_listing_sauna,
        sl.source_kind AS sale_listing_source_kind,
        sl.provider AS sale_listing_source_provider,
        COALESCE(raw.street_address, sb.shortcut_building_address) AS sale_listing_street_address,
        raw.total_floors AS sale_listing_total_floors,
        CASE
            WHEN NULLIF(raw.postal, '') IS NOT NULL
                AND NULLIF(lower(trim(COALESCE(raw.street_address, sb.shortcut_building_address))), '') IS NOT NULL
                AND COALESCE(raw.living_area, raw.area) IS NOT NULL
                THEN concat_ws('|', 'unit_v1', regexp_replace(trim(raw.postal), '[^0-9]+', '', 'g'), lower(trim(COALESCE(raw.street_address, sb.shortcut_building_address))), round(COALESCE(raw.living_area, raw.area)::numeric * 10)::text)
            ELSE sl.canonical_source_id
        END AS sale_listing_unit_match_key,
        sl.url AS sale_listing_url,
        raw.new_development AS sale_listing_new_development,
        raw.balcony AS sale_listing_balcony,
        CASE
            WHEN lower(trim(COALESCE(sa.shortcut_ad_data #>> '{status}', sa.shortcut_ad_data #>> '{adData,status}', sa.shortcut_ad_data #>> '{listingStatus}', ''))) IN ('sold', 'unpublished', 'removed', 'inactive', 'expired', 'deleted', 'archived') THEN 'removed'::text
            ELSE 'active'::text
        END AS sale_listing_status,
        sa.shortcut_ad_id AS shortcut_ad_id
    FROM origin.source_listings sl
    JOIN origin.shortcut_ads sa ON sa.shortcut_ad_id = sl.shortcut_ad_id
    LEFT JOIN origin.shortcut_buildings sb ON sb.shortcut_building_id = sa.shortcut_building_id
    LEFT JOIN origin.source_listing_price_periods current_price ON current_price.source_listing_id = sl.source_listing_id AND current_price.superseded_at IS NULL
    CROSS JOIN LATERAL (
        SELECT
            COALESCE(CASE WHEN NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{address,street,name}', sa.shortcut_ad_data #>> '{address,street}')), '') IS NOT NULL AND NULLIF(trim(sa.shortcut_ad_data #>> '{address,streetNumber}'), '') IS NOT NULL THEN concat_ws(' ', NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{address,street,name}', sa.shortcut_ad_data #>> '{address,street}')), ''), NULLIF(trim(sa.shortcut_ad_data #>> '{address,streetNumber}'), ''), NULLIF(trim(sa.shortcut_ad_data #>> '{address,buildingLetter}'), '')) ELSE NULL END, NULLIF(trim(sa.shortcut_ad_data #>> '{address,formattedAddress}'), ''), NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{address,street,name}', sa.shortcut_ad_data #>> '{address,street}')), '')) AS street_address,
            NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{address,city,name}', sa.shortcut_ad_data #>> '{address,city}')), '') AS city,
            NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{address,zipCode,value}', sa.shortcut_ad_data #>> '{address,zipCode,name}', sa.shortcut_ad_data #>> '{address,zipCode}')), '') AS postal,
            COALESCE((SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{priceData,priceSell}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{priceData,price}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value)) AS price,
            COALESCE((SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{adData,size}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{adData,sizeTotal}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{adData,sizeLiving}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value)) AS area,
            (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{adData,sizeLiving}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value) AS living_area,
            COALESCE((SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{priceData,priceDebtFree}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{priceData,priceSell}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value)) AS debt_free_price,
            (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{priceData,debtShare}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value) AS debt_share_amount,
            COALESCE((SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{priceData,pricePerSqm}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{priceData,pricePerSquareMeter}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value)) AS price_per_m2,
            COALESCE((SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int4 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{adData,rooms}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int4 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{rooms}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value)) AS rooms_count,
            COALESCE((SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int4 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{adData,floor}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int4 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{floor}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value)) AS floor_level,
            COALESCE((SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int4 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{adData,totalFloors}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int4 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{buildingData,floors}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value)) AS total_floors,
            COALESCE((SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int4 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{buildingData,year}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int4 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{adData,constructionYear}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value)) AS build_year,
            NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{adData,condition}', sa.shortcut_ad_data #>> '{property,condition}')), '') AS condition,
            NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{adData,energyClass}', sa.shortcut_ad_data #>> '{property,energyClass}')), '') AS energy_class,
            NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{adData,description}', sa.shortcut_ad_data #>> '{description}', sa.shortcut_ad_data #>> '{text}')), '') AS description_text,
            COALESCE(CASE WHEN sa.shortcut_ad_data #>> '{adData,sauna}' IS NULL THEN NULL WHEN lower(trim(sa.shortcut_ad_data #>> '{adData,sauna}')) IN ('1', 'true', 'yes', 'on', 'kylla', 'kyllä', 'on') THEN true WHEN lower(trim(sa.shortcut_ad_data #>> '{adData,sauna}')) IN ('0', 'false', 'no', 'off', 'ei') THEN false ELSE NULL END, CASE WHEN sa.shortcut_ad_data #>> '{adData,hasSauna}' IS NULL THEN NULL WHEN lower(trim(sa.shortcut_ad_data #>> '{adData,hasSauna}')) IN ('1', 'true', 'yes', 'on', 'kylla', 'kyllä', 'on') THEN true WHEN lower(trim(sa.shortcut_ad_data #>> '{adData,hasSauna}')) IN ('0', 'false', 'no', 'off', 'ei') THEN false ELSE NULL END) AS sauna,
            CASE WHEN sa.shortcut_ad_data #>> '{adData,balcony}' IS NULL THEN NULL WHEN lower(trim(sa.shortcut_ad_data #>> '{adData,balcony}')) IN ('1', 'true', 'yes', 'on', 'kylla', 'kyllä', 'on') THEN true WHEN lower(trim(sa.shortcut_ad_data #>> '{adData,balcony}')) IN ('0', 'false', 'no', 'off', 'ei') THEN false ELSE NULL END AS balcony,
            COALESCE((SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{adData,plotArea}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{buildingData,plotArea}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value)) AS plot_area,
            CASE WHEN sa.shortcut_ad_data #>> '{adData,newDevelopment}' IS NULL THEN NULL WHEN lower(trim(sa.shortcut_ad_data #>> '{adData,newDevelopment}')) IN ('1', 'true', 'yes', 'on', 'kylla', 'kyllä', 'on') THEN true WHEN lower(trim(sa.shortcut_ad_data #>> '{adData,newDevelopment}')) IN ('0', 'false', 'no', 'off', 'ei') THEN false ELSE NULL END AS new_development
    ) raw
    WHERE sl.source_listing_id = $1::uuid
        AND sl.provider = 'shortcut'
        AND sl.source_kind = 'ad'
        AND sa.shortcut_ad_type = 'listing'
        AND sa.shortcut_ad_data IS NOT NULL
),
frontdoor_ad_source AS (
    SELECT
        fa.frontdoor_ad_id,
        NULL::uuid AS frontdoor_building_announcement_id,
        lower(trim(raw.street_address)) AS sale_listing_address_norm,
        NULL::integer AS sale_listing_apartment_count,
        raw.area AS sale_listing_area_value,
        CASE WHEN current_price.source_listing_id IS NULL THEN raw.price ELSE current_price.asking_price END AS sale_listing_asking_price,
        raw.build_year AS sale_listing_build_year,
        lower(trim(concat_ws(' ', raw.street_address, raw.postal, raw.city))) AS sale_listing_building_match_key,
        sl.canonical_source_id AS sale_listing_canonical_id,
        raw.city AS sale_listing_city,
        lower(trim(raw.city)) AS sale_listing_city_norm,
        raw.condition AS sale_listing_condition,
        CASE WHEN current_price.source_listing_id IS NULL THEN raw.debt_free_price ELSE current_price.debt_free_price END AS sale_listing_debt_free_price,
        CASE WHEN current_price.source_listing_id IS NULL THEN raw.debt_share_amount ELSE current_price.debt_share_amount END AS sale_listing_debt_share_amount,
        raw.description_text AS sale_listing_description_text,
        NULL::boolean AS sale_listing_elevator,
        raw.energy_class AS sale_listing_energy_class,
        raw.energy_class AS sale_listing_energy_efficiency_label,
        sl.first_seen_at AS sale_listing_first_seen_at,
        raw.floor_level AS sale_listing_floor_level,
        COALESCE(raw.street_address, fa.frontdoor_ad_external_id) AS sale_listing_headline,
        NULL::text AS sale_listing_housing_company_business_id,
        NULL::text AS sale_listing_housing_company_name,
        sl.source_listing_id AS sale_listing_id,
        fa.frontdoor_ad_last_seen_at AS sale_listing_last_seen_at,
        NULL::double precision AS sale_listing_latitude,
        raw.living_area AS sale_listing_living_area_value,
        NULL::double precision AS sale_listing_longitude,
        sl.native_id AS sale_listing_native_id,
        raw.plot_area AS sale_listing_plot_area_value,
        NULL::boolean AS sale_listing_plot_owned,
        raw.postal AS sale_listing_postal,
        raw.postal AS sale_listing_postal_norm,
        CASE WHEN current_price.source_listing_id IS NULL THEN COALESCE(raw.price_per_m2, CASE WHEN raw.price IS NOT NULL AND raw.area IS NOT NULL AND raw.area > 0 THEN raw.price::double precision / raw.area ELSE NULL END) ELSE COALESCE(current_price.price_per_m2, CASE WHEN current_price.asking_price IS NOT NULL AND raw.area IS NOT NULL AND raw.area > 0 THEN current_price.asking_price::double precision / raw.area ELSE NULL END) END AS sale_listing_price_per_m2,
        NULL::text AS sale_listing_property_type_code,
        raw.published_at AS sale_listing_published_at,
        NULL::text AS sale_listing_room_category_code,
        fa.frontdoor_ad_data #>> '{residenceDetailsDTO,roomStructure}' AS sale_listing_room_layout,
        raw.rooms_count AS sale_listing_rooms_count,
        trim(concat_ws(' ', fa.frontdoor_ad_external_id, fa.frontdoor_ad_url, raw.street_address, raw.city, raw.postal, fa.frontdoor_ad_data #>> '{residenceDetailsDTO,roomStructure}')) AS sale_listing_search_text,
        raw.sauna AS sale_listing_sauna,
        sl.source_kind AS sale_listing_source_kind,
        sl.provider AS sale_listing_source_provider,
        raw.street_address AS sale_listing_street_address,
        raw.total_floors AS sale_listing_total_floors,
        CASE
            WHEN NULLIF(raw.postal, '') IS NOT NULL
                AND NULLIF(lower(trim(raw.street_address)), '') IS NOT NULL
                AND COALESCE(raw.living_area, raw.area) IS NOT NULL
                THEN concat_ws('|', 'unit_v1', regexp_replace(trim(raw.postal), '[^0-9]+', '', 'g'), lower(trim(raw.street_address)), round(COALESCE(raw.living_area, raw.area)::numeric * 10)::text)
            ELSE sl.canonical_source_id
        END AS sale_listing_unit_match_key,
        sl.url AS sale_listing_url,
        raw.new_development AS sale_listing_new_development,
        raw.balcony AS sale_listing_balcony,
        CASE
            WHEN fa.frontdoor_ad_page_not_found OR lower(trim(COALESCE(fa.frontdoor_ad_data #>> '{status}', ''))) IN ('sold', 'unpublished', 'removed', 'inactive', 'expired', 'deleted', 'archived') THEN 'removed'::text
            ELSE 'active'::text
        END AS sale_listing_status,
        NULL::bigint AS shortcut_ad_id
    FROM origin.source_listings sl
    JOIN origin.frontdoor_ads fa ON fa.frontdoor_ad_id = sl.frontdoor_ad_id
    LEFT JOIN origin.source_listing_price_periods current_price ON current_price.source_listing_id = sl.source_listing_id AND current_price.superseded_at IS NULL
    CROSS JOIN LATERAL (
        SELECT
            NULLIF(trim(COALESCE(fa.frontdoor_ad_data #>> '{property,streetAddressFreeForm}', fa.frontdoor_ad_data #>> '{property,address}', fa.frontdoor_ad_data #>> '{property,streetNameFreeForm}')), '') AS street_address,
            NULLIF(trim(COALESCE(fa.frontdoor_ad_data #>> '{property,municipalityNameFreeForm}', fa.frontdoor_ad_data #>> '{property,municipality}', fa.frontdoor_ad_data #>> '{property,city}', fa.frontdoor_ad_data #>> '{property,postCode,postArea}')), '') AS city,
            NULLIF(trim(COALESCE(fa.frontdoor_ad_data #>> '{property,postalCode}', fa.frontdoor_ad_data #>> '{property,addressPostalCode}', fa.frontdoor_ad_data #>> '{property,postCode,postCode}')), '') AS postal,
            COALESCE((SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(fa.frontdoor_ad_data #>> '{sellingPrice}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(fa.frontdoor_ad_data #>> '{preparsed,price}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value)) AS price,
            COALESCE((SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(fa.frontdoor_ad_data #>> '{preparsed,area}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(fa.frontdoor_ad_data #>> '{property,livingArea}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value)) AS area,
            (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(fa.frontdoor_ad_data #>> '{debfFreePrice}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value) AS debt_free_price,
            (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(fa.frontdoor_ad_data #>> '{debtShareAmount}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value) AS debt_share_amount,
            COALESCE((SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(fa.frontdoor_ad_data #>> '{pricePerSquareMeter}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(fa.frontdoor_ad_data #>> '{preparsed,pricePerSquareMeter}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value)) AS price_per_m2,
            (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int4 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(fa.frontdoor_ad_data #>> '{residenceDetailsDTO,totalRoomCount}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value) AS rooms_count,
            COALESCE((SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int4 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(fa.frontdoor_ad_data #>> '{residenceDetailsDTO,housingCompanyApartmentInformationDTO,floorLevel}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int4 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(fa.frontdoor_ad_data #>> '{property,floorLevel}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value)) AS floor_level,
            COALESCE((SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int4 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(fa.frontdoor_ad_data #>> '{property,housingCompany,floorCount}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int4 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(fa.frontdoor_ad_data #>> '{residenceDetailsDTO,floorCount}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value)) AS total_floors,
            COALESCE((SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int4 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(fa.frontdoor_ad_data #>> '{residenceDetailsDTO,constructionFinishedYear}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int4 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(fa.frontdoor_ad_data #>> '{property,housingCompany,usageStartYear}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value)) AS build_year,
            NULLIF(trim(COALESCE(fa.frontdoor_ad_data #>> '{residenceDetailsDTO,inspection,overallCondition}', fa.frontdoor_ad_data #>> '{property,condition}')), '') AS condition,
            NULLIF(trim(COALESCE(fa.frontdoor_ad_data #>> '{property,housingCompany,energyCertificate,energyCertificateType}', fa.frontdoor_ad_data #>> '{property,energyCertificate,energyCertificateType}')), '') AS energy_class,
            NULLIF(trim(COALESCE(fa.frontdoor_ad_data #>> '{text}', fa.frontdoor_ad_data #>> '{property,description}')), '') AS description_text,
            (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(fa.frontdoor_ad_data #>> '{residenceDetailsDTO,livingArea}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value) AS living_area,
            CASE
                WHEN jsonb_path_exists(COALESCE(fa.frontdoor_ad_data, '{}'::jsonb), '$.residenceDetailsDTO.generalDwellingFeatures[*] ? (@ == "HAS_NO_SAUNA")') THEN false
                WHEN jsonb_path_exists(COALESCE(fa.frontdoor_ad_data, '{}'::jsonb), '$.residenceDetailsDTO.generalDwellingFeatures[*] ? (@ == "HAS_SAUNA")') THEN true
                ELSE CASE WHEN fa.frontdoor_ad_data #>> '{property,housingCompany,hasSauna}' IS NULL THEN NULL WHEN lower(trim(fa.frontdoor_ad_data #>> '{property,housingCompany,hasSauna}')) IN ('1', 'true', 'yes', 'on', 'kylla', 'kyllä', 'on') THEN true WHEN lower(trim(fa.frontdoor_ad_data #>> '{property,housingCompany,hasSauna}')) IN ('0', 'false', 'no', 'off', 'ei') THEN false ELSE NULL END
            END AS sauna,
            COALESCE(CASE WHEN fa.frontdoor_ad_data #>> '{property,hasBalcony}' IS NULL THEN NULL WHEN lower(trim(fa.frontdoor_ad_data #>> '{property,hasBalcony}')) IN ('1', 'true', 'yes', 'on', 'kylla', 'kyllä', 'on') THEN true WHEN lower(trim(fa.frontdoor_ad_data #>> '{property,hasBalcony}')) IN ('0', 'false', 'no', 'off', 'ei') THEN false ELSE NULL END, CASE WHEN NULLIF(trim(COALESCE(fa.frontdoor_ad_data #>> '{residenceDetailsDTO,balconyDescription}', fa.frontdoor_ad_data #>> '{property,balconyDescription}')), '') IS NOT NULL THEN true ELSE NULL::boolean END) AS balcony,
            (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(fa.frontdoor_ad_data #>> '{property,plot,area}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value) AS plot_area,
            CASE WHEN fa.frontdoor_ad_data #>> '{newProperty}' IS NULL THEN NULL WHEN lower(trim(fa.frontdoor_ad_data #>> '{newProperty}')) IN ('1', 'true', 'yes', 'on', 'kylla', 'kyllä', 'on') THEN true WHEN lower(trim(fa.frontdoor_ad_data #>> '{newProperty}')) IN ('0', 'false', 'no', 'off', 'ei') THEN false ELSE NULL END AS new_development,
            CASE WHEN (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(fa.frontdoor_ad_data #>> '{publishingTime}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value) IS NULL THEN NULL ELSE to_timestamp((SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(fa.frontdoor_ad_data #>> '{publishingTime}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value) / 1000.0) END AS published_at
    ) raw
    WHERE sl.source_listing_id = $1::uuid
        AND sl.provider = 'frontdoor'
        AND sl.source_kind = 'ad'
        AND fa.frontdoor_ad_data IS NOT NULL
),
source AS (
    SELECT
        frontdoor_ad_id,
        frontdoor_building_announcement_id,
        sale_listing_address_norm,
        sale_listing_apartment_count,
        sale_listing_area_value,
        sale_listing_asking_price,
        sale_listing_build_year,
        sale_listing_building_match_key,
        sale_listing_canonical_id,
        sale_listing_city,
        sale_listing_city_norm,
        sale_listing_condition,
        sale_listing_debt_free_price,
        sale_listing_debt_share_amount,
        sale_listing_description_text,
        sale_listing_elevator,
        sale_listing_energy_class,
        sale_listing_energy_efficiency_label,
        sale_listing_first_seen_at,
        sale_listing_floor_level,
        sale_listing_headline,
        sale_listing_housing_company_business_id,
        sale_listing_housing_company_name,
        sale_listing_id,
        sale_listing_last_seen_at,
        sale_listing_latitude,
        sale_listing_living_area_value,
        sale_listing_longitude,
        sale_listing_native_id,
        sale_listing_plot_area_value,
        sale_listing_plot_owned,
        sale_listing_postal,
        sale_listing_postal_norm,
        sale_listing_price_per_m2,
        sale_listing_property_type_code,
        sale_listing_published_at,
        sale_listing_room_category_code,
        sale_listing_room_layout,
        sale_listing_rooms_count,
        sale_listing_search_text,
        sale_listing_sauna,
        sale_listing_source_kind,
        sale_listing_source_provider,
        sale_listing_street_address,
        sale_listing_total_floors,
        sale_listing_unit_match_key,
        sale_listing_url,
        sale_listing_new_development,
        sale_listing_balcony,
        sale_listing_status,
        shortcut_ad_id
    FROM announcement_source
    UNION ALL
    SELECT
        frontdoor_ad_id,
        frontdoor_building_announcement_id,
        sale_listing_address_norm,
        sale_listing_apartment_count,
        sale_listing_area_value,
        sale_listing_asking_price,
        sale_listing_build_year,
        sale_listing_building_match_key,
        sale_listing_canonical_id,
        sale_listing_city,
        sale_listing_city_norm,
        sale_listing_condition,
        sale_listing_debt_free_price,
        sale_listing_debt_share_amount,
        sale_listing_description_text,
        sale_listing_elevator,
        sale_listing_energy_class,
        sale_listing_energy_efficiency_label,
        sale_listing_first_seen_at,
        sale_listing_floor_level,
        sale_listing_headline,
        sale_listing_housing_company_business_id,
        sale_listing_housing_company_name,
        sale_listing_id,
        sale_listing_last_seen_at,
        sale_listing_latitude,
        sale_listing_living_area_value,
        sale_listing_longitude,
        sale_listing_native_id,
        sale_listing_plot_area_value,
        sale_listing_plot_owned,
        sale_listing_postal,
        sale_listing_postal_norm,
        sale_listing_price_per_m2,
        sale_listing_property_type_code,
        sale_listing_published_at,
        sale_listing_room_category_code,
        sale_listing_room_layout,
        sale_listing_rooms_count,
        sale_listing_search_text,
        sale_listing_sauna,
        sale_listing_source_kind,
        sale_listing_source_provider,
        sale_listing_street_address,
        sale_listing_total_floors,
        sale_listing_unit_match_key,
        sale_listing_url,
        sale_listing_new_development,
        sale_listing_balcony,
        sale_listing_status,
        shortcut_ad_id
    FROM shortcut_source
    UNION ALL
    SELECT
        frontdoor_ad_id,
        frontdoor_building_announcement_id,
        sale_listing_address_norm,
        sale_listing_apartment_count,
        sale_listing_area_value,
        sale_listing_asking_price,
        sale_listing_build_year,
        sale_listing_building_match_key,
        sale_listing_canonical_id,
        sale_listing_city,
        sale_listing_city_norm,
        sale_listing_condition,
        sale_listing_debt_free_price,
        sale_listing_debt_share_amount,
        sale_listing_description_text,
        sale_listing_elevator,
        sale_listing_energy_class,
        sale_listing_energy_efficiency_label,
        sale_listing_first_seen_at,
        sale_listing_floor_level,
        sale_listing_headline,
        sale_listing_housing_company_business_id,
        sale_listing_housing_company_name,
        sale_listing_id,
        sale_listing_last_seen_at,
        sale_listing_latitude,
        sale_listing_living_area_value,
        sale_listing_longitude,
        sale_listing_native_id,
        sale_listing_plot_area_value,
        sale_listing_plot_owned,
        sale_listing_postal,
        sale_listing_postal_norm,
        sale_listing_price_per_m2,
        sale_listing_property_type_code,
        sale_listing_published_at,
        sale_listing_room_category_code,
        sale_listing_room_layout,
        sale_listing_rooms_count,
        sale_listing_search_text,
        sale_listing_sauna,
        sale_listing_source_kind,
        sale_listing_source_provider,
        sale_listing_street_address,
        sale_listing_total_floors,
        sale_listing_unit_match_key,
        sale_listing_url,
        sale_listing_new_development,
        sale_listing_balcony,
        sale_listing_status,
        shortcut_ad_id
    FROM frontdoor_ad_source
),
frontdoor_evidence AS (
    INSERT INTO public.evidence_sources (
        source_kind,
        provider,
        external_id,
        url,
        payload_hash,
        observed_at,
        frontdoor_ad_id
    )
    SELECT
        'frontdoor_ad',
        'frontdoor',
        source.sale_listing_native_id,
        source.sale_listing_url,
        fa.frontdoor_ad_data_hash,
        source.sale_listing_last_seen_at,
        source.frontdoor_ad_id
    FROM source
    JOIN origin.frontdoor_ads fa ON fa.frontdoor_ad_id = source.frontdoor_ad_id
    WHERE source.frontdoor_ad_id IS NOT NULL
    ON CONFLICT (frontdoor_ad_id) WHERE frontdoor_ad_id IS NOT NULL DO UPDATE SET
        external_id = EXCLUDED.external_id,
        url = EXCLUDED.url,
        payload_hash = EXCLUDED.payload_hash,
        observed_at = EXCLUDED.observed_at,
        updated_at = now()
    RETURNING evidence_source_id
),
shortcut_evidence AS (
    INSERT INTO public.evidence_sources (
        source_kind,
        provider,
        external_id,
        url,
        payload_hash,
        observed_at,
        shortcut_ad_id
    )
    SELECT
        'shortcut_ad',
        'shortcut',
        source.sale_listing_native_id,
        source.sale_listing_url,
        sa.shortcut_ad_data_hash,
        source.sale_listing_last_seen_at,
        source.shortcut_ad_id
    FROM source
    JOIN origin.shortcut_ads sa ON sa.shortcut_ad_id = source.shortcut_ad_id
    WHERE source.shortcut_ad_id IS NOT NULL
    ON CONFLICT (shortcut_ad_id) WHERE shortcut_ad_id IS NOT NULL DO UPDATE SET
        external_id = EXCLUDED.external_id,
        url = EXCLUDED.url,
        payload_hash = EXCLUDED.payload_hash,
        observed_at = EXCLUDED.observed_at,
        updated_at = now()
    RETURNING evidence_source_id
),
announcement_evidence AS (
    INSERT INTO public.evidence_sources (
        source_kind,
        provider,
        external_id,
        url,
        payload_hash,
        observed_at,
        frontdoor_building_announcement_id
    )
    SELECT
        'frontdoor_building_announcement',
        'frontdoor',
        source.sale_listing_native_id,
        source.sale_listing_url,
        NULL::text,
        source.sale_listing_last_seen_at,
        source.frontdoor_building_announcement_id
    FROM source
    WHERE source.frontdoor_building_announcement_id IS NOT NULL
    ON CONFLICT (frontdoor_building_announcement_id) WHERE frontdoor_building_announcement_id IS NOT NULL DO UPDATE SET
        external_id = EXCLUDED.external_id,
        url = EXCLUDED.url,
        observed_at = EXCLUDED.observed_at,
        updated_at = now()
    RETURNING evidence_source_id
),
evidence AS (
    SELECT evidence_source_id FROM frontdoor_evidence
    UNION ALL
    SELECT evidence_source_id FROM shortcut_evidence
    UNION ALL
    SELECT evidence_source_id FROM announcement_evidence
),
source_listing AS (
    INSERT INTO origin.source_listings (
        source_listing_id,
        provider,
        source_kind,
        native_id,
        canonical_source_id,
        shortcut_ad_id,
        frontdoor_ad_id,
        frontdoor_building_announcement_id,
        url,
        payload_hash,
        normalized_version,
        normalized_at,
        first_seen_at,
        last_seen_at,
        created_at,
        updated_at
    )
    SELECT
        source.sale_listing_id,
        source.sale_listing_source_provider,
        source.sale_listing_source_kind,
        source.sale_listing_native_id,
        source.sale_listing_canonical_id,
        source.shortcut_ad_id,
        source.frontdoor_ad_id,
        source.frontdoor_building_announcement_id,
        source.sale_listing_url,
        COALESCE(sa.shortcut_ad_data_hash, fa.frontdoor_ad_data_hash),
        GREATEST(COALESCE(sa.shortcut_ad_data_normalized_version, 0), COALESCE(fa.frontdoor_ad_data_normalized_version, 0), COALESCE(fba.frontdoor_building_announcement_data_normalized_version, 0)),
        now(),
        source.sale_listing_first_seen_at,
        source.sale_listing_last_seen_at,
        COALESCE(source.sale_listing_first_seen_at, now()),
        now()
    FROM source
    LEFT JOIN origin.shortcut_ads sa ON sa.shortcut_ad_id = source.shortcut_ad_id
    LEFT JOIN origin.frontdoor_ads fa ON fa.frontdoor_ad_id = source.frontdoor_ad_id
    LEFT JOIN origin.frontdoor_building_announcements fba ON fba.frontdoor_building_announcement_id = source.frontdoor_building_announcement_id
    ON CONFLICT (canonical_source_id) DO UPDATE SET
        source_listing_id = EXCLUDED.source_listing_id,
        provider = EXCLUDED.provider,
        source_kind = EXCLUDED.source_kind,
        native_id = EXCLUDED.native_id,
        shortcut_ad_id = EXCLUDED.shortcut_ad_id,
        frontdoor_ad_id = EXCLUDED.frontdoor_ad_id,
        frontdoor_building_announcement_id = EXCLUDED.frontdoor_building_announcement_id,
        url = EXCLUDED.url,
        payload_hash = EXCLUDED.payload_hash,
        normalized_version = EXCLUDED.normalized_version,
        normalized_at = EXCLUDED.normalized_at,
        first_seen_at = EXCLUDED.first_seen_at,
        last_seen_at = EXCLUDED.last_seen_at,
        updated_at = EXCLUDED.updated_at
    RETURNING source_listing_id
),
source_match_base AS (
    SELECT
        source_listing.source_listing_id,
        source.sale_listing_source_provider AS provider,
        source.sale_listing_source_kind AS source_kind,
        NULLIF(regexp_replace(trim(COALESCE(source.sale_listing_postal_norm, source.sale_listing_postal, '')), '[^0-9]+', '', 'g'), '') AS postal_norm,
        NULLIF(lower(trim(COALESCE(source.sale_listing_city_norm, source.sale_listing_city, ''))), '') AS city_norm,
        NULLIF(lower(trim(regexp_replace(COALESCE(source.sale_listing_address_norm, source.sale_listing_street_address, ''), '[[:space:]]+', ' ', 'g'))), '') AS address_norm,
        COALESCE(source.sale_listing_living_area_value, source.sale_listing_area_value) AS area_m2,
        source.sale_listing_rooms_count AS rooms_count,
        NULLIF(lower(trim(source.sale_listing_room_layout)), '') AS room_layout_norm,
        source.sale_listing_floor_level AS floor_level,
        source.sale_listing_asking_price AS asking_price,
        source.sale_listing_debt_free_price AS debt_free_price,
        source.sale_listing_build_year AS build_year,
        NULLIF(lower(trim(source.sale_listing_housing_company_business_id)), '') AS housing_company_business_id,
        NULLIF(lower(trim(source.sale_listing_housing_company_name)), '') AS housing_company_name_norm,
        source.sale_listing_first_seen_at AS first_seen_at,
        source.sale_listing_last_seen_at AS last_seen_at
    FROM source
    JOIN source_listing ON true
),
source_match_parsed AS (
    SELECT
        source_match_base.source_listing_id,
        source_match_base.provider,
        source_match_base.source_kind,
        source_match_base.postal_norm,
        source_match_base.city_norm,
        source_match_base.address_norm,
        source_match_base.area_m2,
        source_match_base.rooms_count,
        source_match_base.room_layout_norm,
        source_match_base.floor_level,
        source_match_base.asking_price,
        source_match_base.debt_free_price,
        source_match_base.build_year,
        source_match_base.housing_company_business_id,
        source_match_base.housing_company_name_norm,
        source_match_base.first_seen_at,
        source_match_base.last_seen_at,
        NULLIF(regexp_replace(source_match_base.address_norm, '[[:space:]]+[0-9]+.*$', ''), '') AS street_norm,
        NULLIF(substring(source_match_base.address_norm from '([0-9]+([-–][0-9]+)?)[[:alpha:]]?($|[[:space:]])'), '') AS house_norm,
        NULLIF(substring(source_match_base.address_norm from '[0-9]+[-–]?[0-9]*[[:space:]]*([[:alpha:]])([[:space:]]*[0-9]+)?[[:space:]]*$'), '') AS stair_norm,
        NULLIF(substring(source_match_base.address_norm from '[0-9]+[-–]?[0-9]*[[:space:]]*[[:alpha:]]?[[:space:]]+([0-9]{1,4})[[:space:]]*$'), '') AS apartment_norm,
        CASE WHEN source_match_base.area_m2 IS NULL THEN NULL ELSE round(source_match_base.area_m2::numeric * 10)::int4 END AS area_tenths
    FROM source_match_base
),
match_facts AS (
    INSERT INTO public.source_listing_match_facts (
        source_listing_id,
        provider,
        source_kind,
        postal_norm,
        city_norm,
        address_norm,
        street_norm,
        house_norm,
        stair_norm,
        apartment_norm,
        area_m2,
        area_tenths,
        rooms_count,
        room_layout_norm,
        floor_level,
        asking_price,
        debt_free_price,
        build_year,
        housing_company_business_id,
        housing_company_name_norm,
        first_seen_at,
        last_seen_at,
        refreshed_at
    )
    SELECT
        source_listing_id,
        provider,
        source_kind,
        postal_norm,
        city_norm,
        address_norm,
        street_norm,
        house_norm,
        stair_norm,
        apartment_norm,
        area_m2,
        area_tenths,
        rooms_count,
        room_layout_norm,
        floor_level,
        asking_price,
        debt_free_price,
        build_year,
        housing_company_business_id,
        housing_company_name_norm,
        first_seen_at,
        last_seen_at,
        now()
    FROM source_match_parsed
    ON CONFLICT (source_listing_id) DO UPDATE SET
        provider = EXCLUDED.provider,
        source_kind = EXCLUDED.source_kind,
        postal_norm = EXCLUDED.postal_norm,
        city_norm = EXCLUDED.city_norm,
        address_norm = EXCLUDED.address_norm,
        street_norm = EXCLUDED.street_norm,
        house_norm = EXCLUDED.house_norm,
        stair_norm = EXCLUDED.stair_norm,
        apartment_norm = EXCLUDED.apartment_norm,
        area_m2 = EXCLUDED.area_m2,
        area_tenths = EXCLUDED.area_tenths,
        rooms_count = EXCLUDED.rooms_count,
        room_layout_norm = EXCLUDED.room_layout_norm,
        floor_level = EXCLUDED.floor_level,
        asking_price = EXCLUDED.asking_price,
        debt_free_price = EXCLUDED.debt_free_price,
        build_year = EXCLUDED.build_year,
        housing_company_business_id = EXCLUDED.housing_company_business_id,
        housing_company_name_norm = EXCLUDED.housing_company_name_norm,
        first_seen_at = EXCLUDED.first_seen_at,
        last_seen_at = EXCLUDED.last_seen_at,
        refreshed_at = now()
    RETURNING
        source_listing_id,
        provider,
        source_kind,
        postal_norm,
        city_norm,
        address_norm,
        street_norm,
        house_norm,
        stair_norm,
        apartment_norm,
        area_m2,
        area_tenths,
        rooms_count,
        room_layout_norm,
        floor_level,
        asking_price,
        debt_free_price,
        build_year,
        housing_company_business_id,
        housing_company_name_norm,
        first_seen_at,
        last_seen_at
),
candidate_match_facts AS MATERIALIZED (
    SELECT
        existing.source_listing_id,
        existing.provider,
        existing.source_kind,
        existing.postal_norm,
        existing.address_norm,
        existing.street_norm,
        existing.house_norm,
        existing.stair_norm,
        existing.apartment_norm,
        existing.area_tenths,
        existing.asking_price,
        existing.debt_free_price
    FROM public.source_listing_match_facts existing
    WHERE NOT EXISTS (
        SELECT 1
        FROM match_facts current
        WHERE current.source_listing_id = existing.source_listing_id
    )
    UNION ALL
    SELECT
        current.source_listing_id,
        current.provider,
        current.source_kind,
        current.postal_norm,
        current.address_norm,
        current.street_norm,
        current.house_norm,
        current.stair_norm,
        current.apartment_norm,
        current.area_tenths,
        current.asking_price,
        current.debt_free_price
    FROM match_facts current
),
match_lock AS MATERIALIZED (
    SELECT pg_advisory_xact_lock(hashtextextended(concat_ws('|', postal_norm, street_norm, house_norm, area_tenths::text), 0))
    FROM match_facts
    WHERE postal_norm IS NOT NULL
        AND street_norm IS NOT NULL
        AND house_norm IS NOT NULL
        AND area_tenths IS NOT NULL
),
match_blocks AS (
    SELECT DISTINCT match_facts.postal_norm, match_facts.street_norm, match_facts.house_norm, match_facts.area_tenths
    FROM match_facts
    JOIN match_lock ON true
    WHERE match_facts.postal_norm IS NOT NULL
        AND match_facts.street_norm IS NOT NULL
        AND match_facts.house_norm IS NOT NULL
        AND match_facts.area_tenths IS NOT NULL
),
compatible_match_pairs AS (
    SELECT
        a.source_listing_id AS source_listing_id_a,
        b.source_listing_id AS source_listing_id_b,
        a.provider AS source_provider,
        b.provider AS matched_provider,
        a.address_norm AS source_address_norm,
        b.address_norm AS matched_address_norm,
        a.postal_norm,
        a.street_norm,
        a.house_norm,
        a.area_tenths,
        a.stair_norm AS source_stair_norm,
        b.stair_norm AS matched_stair_norm,
        a.apartment_norm AS source_apartment_norm,
        b.apartment_norm AS matched_apartment_norm,
        a.asking_price AS source_asking_price,
        b.asking_price AS matched_asking_price,
        a.debt_free_price AS source_debt_free_price,
        b.debt_free_price AS matched_debt_free_price
    FROM match_blocks
    JOIN candidate_match_facts a ON a.postal_norm = match_blocks.postal_norm
        AND a.street_norm = match_blocks.street_norm
        AND a.house_norm = match_blocks.house_norm
        AND a.area_tenths = match_blocks.area_tenths
    JOIN candidate_match_facts b ON b.source_listing_id > a.source_listing_id
        AND b.postal_norm = a.postal_norm
        AND b.street_norm = a.street_norm
        AND b.house_norm = a.house_norm
        AND b.area_tenths = a.area_tenths
    WHERE (a.stair_norm IS NULL OR b.stair_norm IS NULL OR a.stair_norm = b.stair_norm)
        AND (a.apartment_norm IS NULL OR b.apartment_norm IS NULL OR a.apartment_norm = b.apartment_norm)
),
pair_counts AS (
    SELECT source_listing_id, count(*)::int4 AS compatible_pair_count
    FROM (
        SELECT source_listing_id_a AS source_listing_id FROM compatible_match_pairs
        UNION ALL
        SELECT source_listing_id_b AS source_listing_id FROM compatible_match_pairs
    ) pair_members
    GROUP BY source_listing_id
),
classified_match_pairs AS (
    SELECT
        compatible_match_pairs.source_listing_id_a,
        compatible_match_pairs.source_listing_id_b,
        source_counts.compatible_pair_count AS source_compatible_pair_count,
        matched_counts.compatible_pair_count AS matched_compatible_pair_count,
        CASE
            WHEN source_address_norm = matched_address_norm THEN 'exact_provider_neutral_unit_v1'
            ELSE 'address_missing_stair_one_to_one_v1'
        END AS match_method,
        CASE WHEN source_address_norm = matched_address_norm THEN 100 ELSE 95 END AS match_score,
        CASE WHEN source_address_norm = matched_address_norm THEN 'high'::text ELSE 'medium'::text END AS match_confidence,
        'proposed'::text AS match_status,
        jsonb_strip_nulls(jsonb_build_object(
            'postal_norm', postal_norm,
            'street_norm', street_norm,
            'house_norm', house_norm,
            'area_tenths', area_tenths,
            'source_provider', source_provider,
            'matched_provider', matched_provider,
            'source_address_norm', source_address_norm,
            'matched_address_norm', matched_address_norm,
            'source_stair_norm', source_stair_norm,
            'matched_stair_norm', matched_stair_norm,
            'source_apartment_norm', source_apartment_norm,
            'matched_apartment_norm', matched_apartment_norm,
            'source_asking_price', source_asking_price,
            'matched_asking_price', matched_asking_price,
            'source_debt_free_price', source_debt_free_price,
            'matched_debt_free_price', matched_debt_free_price,
            'source_compatible_pair_count', source_counts.compatible_pair_count,
            'matched_compatible_pair_count', matched_counts.compatible_pair_count
        )) AS match_reasons
    FROM compatible_match_pairs
    JOIN pair_counts source_counts ON source_counts.source_listing_id = compatible_match_pairs.source_listing_id_a
    JOIN pair_counts matched_counts ON matched_counts.source_listing_id = compatible_match_pairs.source_listing_id_b
),
superseded_match_candidates AS (
    UPDATE public.source_listing_match_candidates candidate
    SET match_status = 'superseded',
        updated_at = now(),
        decided_at = now()
    FROM match_facts
    WHERE candidate.match_status IN ('proposed', 'accepted')
        AND (candidate.source_listing_id_a = match_facts.source_listing_id OR candidate.source_listing_id_b = match_facts.source_listing_id)
        AND NOT EXISTS (
            SELECT 1
            FROM classified_match_pairs current_candidate
            WHERE current_candidate.source_listing_id_a = candidate.source_listing_id_a
                AND current_candidate.source_listing_id_b = candidate.source_listing_id_b
                AND current_candidate.match_method = candidate.match_method
        )
    RETURNING candidate.source_listing_match_candidate_id
),
source_match_candidates AS (
    INSERT INTO public.source_listing_match_candidates (
        source_listing_id_a,
        source_listing_id_b,
        match_method,
        match_score,
        match_confidence,
        match_status,
        match_reasons,
        evaluation_version,
        updated_at,
        decided_at
    )
    SELECT
        source_listing_id_a,
        source_listing_id_b,
        match_method,
        match_score,
        match_confidence,
        match_status,
        match_reasons,
        'source_listing_match_v2',
        now(),
        NULL::timestamptz
    FROM classified_match_pairs
    WHERE source_compatible_pair_count <= 2
        AND matched_compatible_pair_count <= 2
        AND NOT EXISTS (
            SELECT 1
            FROM public.source_listing_match_candidates rejected
            WHERE rejected.source_listing_id_a = classified_match_pairs.source_listing_id_a
                AND rejected.source_listing_id_b = classified_match_pairs.source_listing_id_b
                AND rejected.match_method = classified_match_pairs.match_method
                AND rejected.match_status = 'rejected'
        )
    ON CONFLICT (source_listing_id_a, source_listing_id_b, match_method) WHERE (match_status IN ('proposed', 'accepted')) DO UPDATE SET
        match_score = EXCLUDED.match_score,
        match_confidence = EXCLUDED.match_confidence,
        match_reasons = EXCLUDED.match_reasons,
        evaluation_version = EXCLUDED.evaluation_version,
        updated_at = now()
    RETURNING source_listing_id_a, source_listing_id_b, match_method, match_score, match_status
),
assigned_listing AS (
    SELECT
        target_source.target_id AS listing_id,
        po.property_offering_identity_key,
        doc.primary_source_listing_id,
        CASE
            WHEN doc.listing_id IS NULL OR doc.primary_source_listing_id IS NULL OR doc.primary_source_listing_id = match_facts.source_listing_id THEN true
            WHEN ROW(
                CASE WHEN match_facts.source_kind = 'ad' THEN 0 ELSE 1 END,
                CASE WHEN match_facts.asking_price IS NOT NULL OR match_facts.debt_free_price IS NOT NULL THEN 0 ELSE 1 END,
                -extract(epoch FROM COALESCE(match_facts.last_seen_at, '-infinity'::timestamptz)),
                match_facts.source_listing_id::text
            ) < ROW(
                CASE WHEN doc.kind = 'ad' THEN 0 ELSE 1 END,
                CASE WHEN doc.asking_price IS NOT NULL OR doc.debt_free_price IS NOT NULL THEN 0 ELSE 1 END,
                -extract(epoch FROM COALESCE(doc.last_seen_at, '-infinity'::timestamptz)),
                COALESCE(doc.primary_source_listing_id::text, '')
            ) THEN true
            ELSE false
        END AS replace_projection
    FROM public.target_sources target_source
    JOIN match_facts ON true
    JOIN public.property_offerings po ON po.property_offering_id = target_source.target_id
    LEFT JOIN public.listing_search_documents doc ON doc.listing_id = target_source.target_id
    WHERE target_source.target_type = 'listing'
        AND target_source.source_type = 'source_listing'
        AND target_source.source_id = match_facts.source_listing_id
        AND target_source.link_status = 'confirmed'
    ORDER BY (target_source.link_method = 'manual') DESC, target_source.created_at
    LIMIT 1
),
projection_policy AS (
    SELECT COALESCE((SELECT replace_projection FROM assigned_listing LIMIT 1), true) AS replace_projection
),
housing_company AS (
    INSERT INTO public.housing_companies (
        housing_company_identity_key,
        housing_company_postal_norm,
        housing_company_city_norm,
        housing_company_address_norm,
        housing_company_name,
        housing_company_business_id,
        housing_company_build_year,
        housing_company_floor_count,
        housing_company_apartment_count,
        housing_company_elevator,
        housing_company_energy_efficiency_label,
        housing_company_match_reasons,
        housing_company_updated_at,
        housing_company_geom
    )
    SELECT
        'company:' || COALESCE(NULLIF(source.sale_listing_housing_company_business_id, ''), NULLIF(source.sale_listing_housing_company_name, ''), NULLIF(concat_ws('|', source.sale_listing_postal_norm, source.sale_listing_city_norm, source.sale_listing_building_match_key), ''), source.sale_listing_canonical_id),
        COALESCE(source.sale_listing_postal_norm, source.sale_listing_postal),
        COALESCE(source.sale_listing_city_norm, source.sale_listing_city),
        COALESCE(source.sale_listing_address_norm, source.sale_listing_street_address),
        source.sale_listing_housing_company_name,
        source.sale_listing_housing_company_business_id,
        source.sale_listing_build_year,
        source.sale_listing_total_floors,
        source.sale_listing_apartment_count,
        source.sale_listing_elevator,
        COALESCE(source.sale_listing_energy_efficiency_label, source.sale_listing_energy_class),
        jsonb_build_object('evidence_source_id', (SELECT evidence_source_id FROM evidence LIMIT 1), 'method', 'listingmodel_reconcile'),
        now(),
        CASE
            WHEN source.sale_listing_latitude IS NULL OR source.sale_listing_longitude IS NULL THEN NULL
            ELSE postgis.ST_SetSRID(postgis.ST_MakePoint(source.sale_listing_longitude, source.sale_listing_latitude), 4326)
        END
    FROM source
    WHERE COALESCE(source.sale_listing_property_type_code, '') <> 'detached_house'
    ON CONFLICT (housing_company_identity_key) DO UPDATE SET
        housing_company_postal_norm = COALESCE(public.housing_companies.housing_company_postal_norm, EXCLUDED.housing_company_postal_norm),
        housing_company_city_norm = COALESCE(public.housing_companies.housing_company_city_norm, EXCLUDED.housing_company_city_norm),
        housing_company_address_norm = COALESCE(public.housing_companies.housing_company_address_norm, EXCLUDED.housing_company_address_norm),
        housing_company_name = COALESCE(public.housing_companies.housing_company_name, EXCLUDED.housing_company_name),
        housing_company_business_id = COALESCE(public.housing_companies.housing_company_business_id, EXCLUDED.housing_company_business_id),
        housing_company_build_year = COALESCE(public.housing_companies.housing_company_build_year, EXCLUDED.housing_company_build_year),
        housing_company_floor_count = COALESCE(public.housing_companies.housing_company_floor_count, EXCLUDED.housing_company_floor_count),
        housing_company_apartment_count = COALESCE(public.housing_companies.housing_company_apartment_count, EXCLUDED.housing_company_apartment_count),
        housing_company_elevator = COALESCE(public.housing_companies.housing_company_elevator, EXCLUDED.housing_company_elevator),
        housing_company_energy_efficiency_label = COALESCE(public.housing_companies.housing_company_energy_efficiency_label, EXCLUDED.housing_company_energy_efficiency_label),
        housing_company_match_reasons = public.housing_companies.housing_company_match_reasons || EXCLUDED.housing_company_match_reasons,
        housing_company_geom = COALESCE(public.housing_companies.housing_company_geom, EXCLUDED.housing_company_geom),
        housing_company_updated_at = now()
    RETURNING housing_company_id
),
physical_building AS (
    INSERT INTO public.physical_buildings (
        housing_company_id,
        physical_building_identity_key,
        physical_building_address_norm,
        physical_building_postal_norm,
        physical_building_city_norm,
        physical_building_build_year,
        physical_building_floor_count,
        physical_building_apartment_count,
        physical_building_elevator,
        physical_building_latitude,
        physical_building_longitude,
        physical_building_updated_at
    )
    SELECT
        housing_company.housing_company_id,
        'building:' || COALESCE(NULLIF(source.sale_listing_building_match_key, ''), NULLIF(concat_ws('|', source.sale_listing_postal_norm, source.sale_listing_city_norm, source.sale_listing_address_norm), ''), source.sale_listing_canonical_id),
        COALESCE(source.sale_listing_address_norm, source.sale_listing_street_address),
        COALESCE(source.sale_listing_postal_norm, source.sale_listing_postal),
        COALESCE(source.sale_listing_city_norm, source.sale_listing_city),
        source.sale_listing_build_year,
        source.sale_listing_total_floors,
        source.sale_listing_apartment_count,
        source.sale_listing_elevator,
        source.sale_listing_latitude,
        source.sale_listing_longitude,
        now()
    FROM source
    JOIN housing_company ON true
    ON CONFLICT (physical_building_identity_key) DO UPDATE SET
        housing_company_id = COALESCE(public.physical_buildings.housing_company_id, EXCLUDED.housing_company_id),
        physical_building_address_norm = COALESCE(public.physical_buildings.physical_building_address_norm, EXCLUDED.physical_building_address_norm),
        physical_building_postal_norm = COALESCE(public.physical_buildings.physical_building_postal_norm, EXCLUDED.physical_building_postal_norm),
        physical_building_city_norm = COALESCE(public.physical_buildings.physical_building_city_norm, EXCLUDED.physical_building_city_norm),
        physical_building_build_year = COALESCE(public.physical_buildings.physical_building_build_year, EXCLUDED.physical_building_build_year),
        physical_building_floor_count = COALESCE(public.physical_buildings.physical_building_floor_count, EXCLUDED.physical_building_floor_count),
        physical_building_apartment_count = COALESCE(public.physical_buildings.physical_building_apartment_count, EXCLUDED.physical_building_apartment_count),
        physical_building_elevator = COALESCE(public.physical_buildings.physical_building_elevator, EXCLUDED.physical_building_elevator),
        physical_building_latitude = COALESCE(public.physical_buildings.physical_building_latitude, EXCLUDED.physical_building_latitude),
        physical_building_longitude = COALESCE(public.physical_buildings.physical_building_longitude, EXCLUDED.physical_building_longitude),
        physical_building_updated_at = now()
    RETURNING physical_building_id, housing_company_id
),
property_unit AS (
    INSERT INTO public.property_units (
        housing_company_id,
        physical_building_id,
        property_unit_identity_key,
        property_unit_address_norm,
        property_unit_floor_level,
        property_unit_area_value,
        property_unit_rooms_count,
        property_unit_room_layout,
        property_unit_layout_match_key,
        property_unit_match_reasons,
        property_unit_updated_at
    )
    SELECT
        physical_building.housing_company_id,
        physical_building.physical_building_id,
        'unit:' || COALESCE(NULLIF(source.sale_listing_unit_match_key, ''), source.sale_listing_canonical_id),
        COALESCE(source.sale_listing_address_norm, source.sale_listing_street_address),
        source.sale_listing_floor_level,
        COALESCE(source.sale_listing_living_area_value, source.sale_listing_area_value),
        source.sale_listing_rooms_count,
        source.sale_listing_room_layout,
        source.sale_listing_room_category_code,
        jsonb_build_object('evidence_source_id', (SELECT evidence_source_id FROM evidence LIMIT 1), 'method', 'listingmodel_reconcile'),
        now()
    FROM source
    JOIN physical_building ON true
    ON CONFLICT (property_unit_identity_key) DO UPDATE SET
        housing_company_id = COALESCE(public.property_units.housing_company_id, EXCLUDED.housing_company_id),
        physical_building_id = COALESCE(public.property_units.physical_building_id, EXCLUDED.physical_building_id),
        property_unit_address_norm = COALESCE(public.property_units.property_unit_address_norm, EXCLUDED.property_unit_address_norm),
        property_unit_floor_level = COALESCE(public.property_units.property_unit_floor_level, EXCLUDED.property_unit_floor_level),
        property_unit_area_value = COALESCE(public.property_units.property_unit_area_value, EXCLUDED.property_unit_area_value),
        property_unit_rooms_count = COALESCE(public.property_units.property_unit_rooms_count, EXCLUDED.property_unit_rooms_count),
        property_unit_room_layout = COALESCE(public.property_units.property_unit_room_layout, EXCLUDED.property_unit_room_layout),
        property_unit_layout_match_key = COALESCE(public.property_units.property_unit_layout_match_key, EXCLUDED.property_unit_layout_match_key),
        property_unit_match_reasons = public.property_units.property_unit_match_reasons || EXCLUDED.property_unit_match_reasons,
        property_unit_updated_at = now()
    RETURNING property_unit_id, housing_company_id, physical_building_id
),
property_house AS (
    INSERT INTO public.property_houses (
        property_house_identity_key,
        property_house_address_norm,
        property_house_postal_norm,
        property_house_city_norm,
        property_house_build_year,
        property_house_area_value,
        property_house_plot_area_value,
        property_house_rooms_count,
        property_house_latitude,
        property_house_longitude,
        property_house_match_reasons,
        primary_sale_listing_id,
        property_house_updated_at
    )
    SELECT
        'house:' || COALESCE(NULLIF(source.sale_listing_building_match_key, ''), NULLIF(concat_ws('|', source.sale_listing_postal_norm, source.sale_listing_city_norm, source.sale_listing_address_norm), ''), source.sale_listing_canonical_id),
        COALESCE(source.sale_listing_address_norm, source.sale_listing_street_address),
        COALESCE(source.sale_listing_postal_norm, source.sale_listing_postal),
        COALESCE(source.sale_listing_city_norm, source.sale_listing_city),
        source.sale_listing_build_year,
        COALESCE(source.sale_listing_living_area_value, source.sale_listing_area_value),
        source.sale_listing_plot_area_value,
        source.sale_listing_rooms_count,
        source.sale_listing_latitude,
        source.sale_listing_longitude,
        jsonb_build_object('evidence_source_id', (SELECT evidence_source_id FROM evidence LIMIT 1), 'method', 'listingmodel_reconcile'),
        source.sale_listing_id,
        now()
    FROM source
    WHERE source.sale_listing_property_type_code = 'detached_house'
    ON CONFLICT (property_house_identity_key) DO UPDATE SET
        property_house_address_norm = COALESCE(public.property_houses.property_house_address_norm, EXCLUDED.property_house_address_norm),
        property_house_postal_norm = COALESCE(public.property_houses.property_house_postal_norm, EXCLUDED.property_house_postal_norm),
        property_house_city_norm = COALESCE(public.property_houses.property_house_city_norm, EXCLUDED.property_house_city_norm),
        property_house_build_year = COALESCE(public.property_houses.property_house_build_year, EXCLUDED.property_house_build_year),
        property_house_area_value = COALESCE(public.property_houses.property_house_area_value, EXCLUDED.property_house_area_value),
        property_house_plot_area_value = COALESCE(public.property_houses.property_house_plot_area_value, EXCLUDED.property_house_plot_area_value),
        property_house_rooms_count = COALESCE(public.property_houses.property_house_rooms_count, EXCLUDED.property_house_rooms_count),
        property_house_latitude = COALESCE(public.property_houses.property_house_latitude, EXCLUDED.property_house_latitude),
        property_house_longitude = COALESCE(public.property_houses.property_house_longitude, EXCLUDED.property_house_longitude),
        property_house_match_reasons = public.property_houses.property_house_match_reasons || EXCLUDED.property_house_match_reasons,
        primary_sale_listing_id = COALESCE(public.property_houses.primary_sale_listing_id, EXCLUDED.primary_sale_listing_id),
        property_house_updated_at = now()
    RETURNING property_house_id, property_house_latitude, property_house_longitude
),
offering AS (
    INSERT INTO public.property_offerings (
        property_unit_id,
        property_house_id,
        property_offering_identity_key,
        property_offering_type,
        property_offering_headline,
        property_offering_asking_price,
        property_offering_debt_free_price,
        property_offering_price_per_m2,
        property_offering_first_seen_at,
        property_offering_last_seen_at,
        property_offering_status,
        primary_sale_listing_id,
        property_offering_match_reasons,
        property_offering_updated_at
    )
    SELECT
        (SELECT property_unit_id FROM property_unit LIMIT 1),
        (SELECT property_house_id FROM property_house LIMIT 1),
        COALESCE((SELECT property_offering_identity_key FROM assigned_listing LIMIT 1), 'offering:source:' || source.sale_listing_id::text),
        'sale',
        COALESCE(source.sale_listing_headline, source.sale_listing_street_address, source.sale_listing_native_id),
        source.sale_listing_asking_price,
        source.sale_listing_debt_free_price,
        source.sale_listing_price_per_m2,
        source.sale_listing_first_seen_at,
        source.sale_listing_last_seen_at,
        source.sale_listing_status,
        source.sale_listing_id,
        jsonb_build_object('evidence_source_id', (SELECT evidence_source_id FROM evidence LIMIT 1), 'method', 'listingmodel_reconcile'),
        now()
    FROM source
    ON CONFLICT (property_offering_identity_key) DO UPDATE SET
        property_unit_id = COALESCE(public.property_offerings.property_unit_id, EXCLUDED.property_unit_id),
        property_house_id = COALESCE(public.property_offerings.property_house_id, EXCLUDED.property_house_id),
        property_offering_headline = CASE WHEN (SELECT replace_projection FROM projection_policy) THEN EXCLUDED.property_offering_headline ELSE public.property_offerings.property_offering_headline END,
        property_offering_asking_price = CASE WHEN (SELECT replace_projection FROM projection_policy) THEN EXCLUDED.property_offering_asking_price ELSE public.property_offerings.property_offering_asking_price END,
        property_offering_debt_free_price = CASE WHEN (SELECT replace_projection FROM projection_policy) THEN EXCLUDED.property_offering_debt_free_price ELSE public.property_offerings.property_offering_debt_free_price END,
        property_offering_price_per_m2 = CASE WHEN (SELECT replace_projection FROM projection_policy) THEN EXCLUDED.property_offering_price_per_m2 ELSE public.property_offerings.property_offering_price_per_m2 END,
        property_offering_first_seen_at = LEAST(COALESCE(public.property_offerings.property_offering_first_seen_at, EXCLUDED.property_offering_first_seen_at), COALESCE(EXCLUDED.property_offering_first_seen_at, public.property_offerings.property_offering_first_seen_at)),
        property_offering_last_seen_at = GREATEST(COALESCE(public.property_offerings.property_offering_last_seen_at, EXCLUDED.property_offering_last_seen_at), COALESCE(EXCLUDED.property_offering_last_seen_at, public.property_offerings.property_offering_last_seen_at)),
        property_offering_status = CASE WHEN (SELECT replace_projection FROM projection_policy) THEN EXCLUDED.property_offering_status ELSE public.property_offerings.property_offering_status END,
        primary_sale_listing_id = CASE WHEN (SELECT replace_projection FROM projection_policy) THEN EXCLUDED.primary_sale_listing_id ELSE public.property_offerings.primary_sale_listing_id END,
        property_offering_match_reasons = public.property_offerings.property_offering_match_reasons || EXCLUDED.property_offering_match_reasons,
        property_offering_updated_at = now()
    RETURNING property_offering_id, property_unit_id, property_house_id
),
listing AS (
    INSERT INTO public.listings (
        listing_id,
        listing_type,
        listing_status,
        primary_source_listing_id,
        unit_id,
        house_id,
        first_seen_at,
        last_seen_at,
        updated_at
    )
    SELECT
        offering.property_offering_id,
        'sale',
        source.sale_listing_status,
        source_listing.source_listing_id,
        offering.property_unit_id,
        offering.property_house_id,
        source.sale_listing_first_seen_at,
        source.sale_listing_last_seen_at,
        now()
    FROM source
    JOIN source_listing ON true
    JOIN offering ON true
    ON CONFLICT (listing_id) DO UPDATE SET
        listing_type = EXCLUDED.listing_type,
        listing_status = CASE WHEN (SELECT replace_projection FROM projection_policy) THEN EXCLUDED.listing_status ELSE public.listings.listing_status END,
        primary_source_listing_id = CASE WHEN (SELECT replace_projection FROM projection_policy) THEN EXCLUDED.primary_source_listing_id ELSE public.listings.primary_source_listing_id END,
        unit_id = EXCLUDED.unit_id,
        house_id = EXCLUDED.house_id,
        first_seen_at = LEAST(COALESCE(public.listings.first_seen_at, EXCLUDED.first_seen_at), COALESCE(EXCLUDED.first_seen_at, public.listings.first_seen_at)),
        last_seen_at = GREATEST(COALESCE(public.listings.last_seen_at, EXCLUDED.last_seen_at), COALESCE(EXCLUDED.last_seen_at, public.listings.last_seen_at)),
        updated_at = now()
    RETURNING listing_id
),
target_source AS (
    INSERT INTO public.target_sources (
        target_type,
        target_id,
        source_type,
        source_id,
        link_status,
        link_method,
        link_score,
        link_reasons,
        first_seen_at,
        last_seen_at,
        updated_at
    )
    SELECT
        'listing',
        listing.listing_id,
        'source_listing',
        source_listing.source_listing_id,
        'confirmed',
        'sync_auto',
        100,
        jsonb_build_object('method', 'listingmodel_reconcile'),
        source.sale_listing_first_seen_at,
        source.sale_listing_last_seen_at,
        now()
    FROM source
    JOIN source_listing ON true
    JOIN listing ON true
    ON CONFLICT (target_type, target_id, source_type, source_id) DO UPDATE SET
        first_seen_at = LEAST(COALESCE(public.target_sources.first_seen_at, EXCLUDED.first_seen_at), COALESCE(EXCLUDED.first_seen_at, public.target_sources.first_seen_at)),
        last_seen_at = GREATEST(COALESCE(public.target_sources.last_seen_at, EXCLUDED.last_seen_at), COALESCE(EXCLUDED.last_seen_at, public.target_sources.last_seen_at)),
        updated_at = now()
    RETURNING target_id, source_id
),
listing_evidence AS (
    INSERT INTO public.entity_evidence (
        evidence_source_id,
        listing_id,
        link_status,
        link_method,
        confidence,
        reasons
    )
    SELECT evidence.evidence_source_id, listing.listing_id, 'confirmed', 'sync_auto', 1, jsonb_build_object('method', 'listingmodel_reconcile')
    FROM evidence
    JOIN listing ON true
    ON CONFLICT DO NOTHING
    RETURNING entity_evidence_id
),
offering_evidence AS (
    INSERT INTO public.entity_evidence (
        evidence_source_id,
        property_offering_id,
        link_status,
        link_method,
        confidence,
        reasons
    )
    SELECT evidence.evidence_source_id, offering.property_offering_id, 'confirmed', 'sync_auto', 1, jsonb_build_object('method', 'listingmodel_reconcile')
    FROM evidence
    JOIN offering ON true
    ON CONFLICT DO NOTHING
    RETURNING entity_evidence_id
),
unit_evidence AS (
    INSERT INTO public.entity_evidence (
        evidence_source_id,
        property_unit_id,
        link_status,
        link_method,
        confidence,
        reasons
    )
    SELECT evidence.evidence_source_id, property_unit.property_unit_id, 'confirmed', 'sync_auto', 1, jsonb_build_object('method', 'listingmodel_reconcile')
    FROM evidence
    JOIN property_unit ON true
    ON CONFLICT DO NOTHING
    RETURNING entity_evidence_id
),
house_evidence AS (
    INSERT INTO public.entity_evidence (
        evidence_source_id,
        property_house_id,
        link_status,
        link_method,
        confidence,
        reasons
    )
    SELECT evidence.evidence_source_id, property_house.property_house_id, 'confirmed', 'sync_auto', 1, jsonb_build_object('method', 'listingmodel_reconcile')
    FROM evidence
    JOIN property_house ON true
    ON CONFLICT DO NOTHING
    RETURNING entity_evidence_id
),
linked_listing_sources AS (
    SELECT target_source.target_id, target_source.source_id
    FROM public.target_sources target_source
    JOIN offering ON offering.property_offering_id = target_source.target_id
    WHERE target_source.target_type = 'listing'
        AND target_source.source_type = 'source_listing'
        AND target_source.link_status = 'confirmed'
    UNION
    SELECT target_source.target_id, target_source.source_id
    FROM target_source
),
source_summary AS (
    SELECT
        offering.property_offering_id,
        array_agg(DISTINCT linked_source.provider ORDER BY linked_source.provider) AS source_providers,
        array_agg(DISTINCT linked_source.source_kind ORDER BY linked_source.source_kind) AS source_kinds
    FROM offering
    JOIN linked_listing_sources source_link ON source_link.target_id = offering.property_offering_id
    JOIN origin.source_listings linked_source ON linked_source.source_listing_id = source_link.source_id
    GROUP BY offering.property_offering_id
),
search_document AS (
    INSERT INTO public.listing_search_documents (
        listing_id,
        property_offering_id,
        primary_evidence_source_id,
        primary_source_listing_id,
        source,
        kind,
        native_id,
        canonical_id,
        url,
        headline,
        address,
        city,
        postal,
        latitude,
        longitude,
        asking_price,
        debt_free_price,
        debt_share_amount,
        area_m2,
        room_layout,
        price_per_m2,
        rooms_count,
        floor_level,
        total_floors,
        build_year,
        property_type_code,
        condition,
        energy_class,
        energy_efficiency_label,
        elevator,
        sauna,
        balcony,
        plot_owned,
        new_development,
        listing_type,
        listing_status,
        published_at,
        first_seen_at,
        last_seen_at,
        search_text,
        source_providers,
        source_kinds,
        refreshed_at
    )
    SELECT
        listing.listing_id,
        offering.property_offering_id,
        evidence.evidence_source_id,
        source_listing.source_listing_id,
        source.sale_listing_source_provider,
        source.sale_listing_source_kind,
        source.sale_listing_native_id,
        source.sale_listing_canonical_id,
        source.sale_listing_url,
        COALESCE(source.sale_listing_headline, source.sale_listing_street_address, source.sale_listing_native_id),
        source.sale_listing_street_address,
        COALESCE(source.sale_listing_city, source.sale_listing_city_norm),
        COALESCE(source.sale_listing_postal, source.sale_listing_postal_norm),
        COALESCE(source.sale_listing_latitude, property_house.property_house_latitude),
        COALESCE(source.sale_listing_longitude, property_house.property_house_longitude),
        source.sale_listing_asking_price,
        source.sale_listing_debt_free_price,
        source.sale_listing_debt_share_amount,
        COALESCE(source.sale_listing_living_area_value, source.sale_listing_area_value),
        source.sale_listing_room_layout,
        source.sale_listing_price_per_m2,
        source.sale_listing_rooms_count,
        source.sale_listing_floor_level,
        source.sale_listing_total_floors,
        source.sale_listing_build_year,
        source.sale_listing_property_type_code,
        source.sale_listing_condition,
        source.sale_listing_energy_class,
        source.sale_listing_energy_efficiency_label,
        source.sale_listing_elevator,
        source.sale_listing_sauna,
        source.sale_listing_balcony,
        source.sale_listing_plot_owned,
        source.sale_listing_new_development,
        'sale',
        source.sale_listing_status,
        source.sale_listing_published_at,
        source.sale_listing_first_seen_at,
        source.sale_listing_last_seen_at,
        trim(concat_ws(' ', source.sale_listing_canonical_id, source.sale_listing_native_id, source.sale_listing_search_text, source.sale_listing_description_text, source.sale_listing_street_address, source.sale_listing_city, source.sale_listing_postal, source.sale_listing_housing_company_name, source.sale_listing_housing_company_business_id)),
        COALESCE(source_summary.source_providers, ARRAY[source.sale_listing_source_provider]::text[]),
        COALESCE(source_summary.source_kinds, ARRAY[source.sale_listing_source_kind]::text[]),
        now()
    FROM source
    JOIN source_listing ON true
    JOIN evidence ON true
    JOIN offering ON true
    JOIN listing ON true
    LEFT JOIN property_house ON true
    LEFT JOIN source_summary ON source_summary.property_offering_id = offering.property_offering_id
    ON CONFLICT (listing_id) DO UPDATE SET
        property_offering_id = EXCLUDED.property_offering_id,
        primary_evidence_source_id = EXCLUDED.primary_evidence_source_id,
        primary_source_listing_id = EXCLUDED.primary_source_listing_id,
        source = EXCLUDED.source,
        kind = EXCLUDED.kind,
        native_id = EXCLUDED.native_id,
        canonical_id = EXCLUDED.canonical_id,
        url = EXCLUDED.url,
        headline = EXCLUDED.headline,
        address = EXCLUDED.address,
        city = EXCLUDED.city,
        postal = EXCLUDED.postal,
        latitude = EXCLUDED.latitude,
        longitude = EXCLUDED.longitude,
        asking_price = EXCLUDED.asking_price,
        debt_free_price = EXCLUDED.debt_free_price,
        debt_share_amount = EXCLUDED.debt_share_amount,
        area_m2 = EXCLUDED.area_m2,
        room_layout = EXCLUDED.room_layout,
        price_per_m2 = EXCLUDED.price_per_m2,
        rooms_count = EXCLUDED.rooms_count,
        floor_level = EXCLUDED.floor_level,
        total_floors = EXCLUDED.total_floors,
        build_year = EXCLUDED.build_year,
        property_type_code = EXCLUDED.property_type_code,
        condition = EXCLUDED.condition,
        energy_class = EXCLUDED.energy_class,
        energy_efficiency_label = EXCLUDED.energy_efficiency_label,
        elevator = EXCLUDED.elevator,
        sauna = EXCLUDED.sauna,
        balcony = EXCLUDED.balcony,
        plot_owned = EXCLUDED.plot_owned,
        new_development = EXCLUDED.new_development,
        listing_type = EXCLUDED.listing_type,
        listing_status = EXCLUDED.listing_status,
        published_at = EXCLUDED.published_at,
        first_seen_at = EXCLUDED.first_seen_at,
        last_seen_at = EXCLUDED.last_seen_at,
        search_text = EXCLUDED.search_text,
        source_providers = EXCLUDED.source_providers,
        source_kinds = EXCLUDED.source_kinds,
        refreshed_at = now()
    WHERE (SELECT replace_projection FROM projection_policy)
    RETURNING listing_id
),
refreshed_source_summary AS (
    UPDATE public.listing_search_documents document
    SET source_providers = source_summary.source_providers,
        source_kinds = source_summary.source_kinds,
        refreshed_at = now()
    FROM source_summary
    JOIN offering ON offering.property_offering_id = source_summary.property_offering_id
    WHERE NOT (SELECT replace_projection FROM projection_policy)
        AND document.listing_id = offering.property_offering_id
    RETURNING document.listing_id
)
SELECT
    evidence.evidence_source_id,
    offering.property_offering_id,
    listing.listing_id,
    ((SELECT count(*) FROM search_document) + (SELECT count(*) FROM refreshed_source_summary))::int4 AS search_documents
FROM evidence
JOIN offering ON true
JOIN listing ON true;
