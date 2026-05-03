CREATE OR REPLACE FUNCTION public.fnc__sync_canonical_property_for_sale_listing(listing_id uuid, link_method text DEFAULT 'sync_auto')
RETURNS uuid
LANGUAGE plpgsql
AS $$
DECLARE
    v_housing_company_id uuid;
    unit_id uuid;
    offering_id uuid;
    source_link_id uuid;
BEGIN
    WITH source_values AS (
        SELECT
            sl.sale_listing_id,
            sl.sale_listing_source_provider,
            sl.sale_listing_source_kind,
            sl.sale_listing_native_id,
            sl.sale_listing_headline,
            sl.sale_listing_postal_norm,
            sl.sale_listing_city_norm,
            sl.sale_listing_address_norm,
            sl.sale_listing_building_match_key,
            sl.sale_listing_unit_match_key,
            sl.sale_listing_floor_level,
            sl.sale_listing_area_value,
            sl.sale_listing_rooms_count,
            sl.sale_listing_room_layout,
            sl.sale_listing_asking_price,
            sl.sale_listing_debt_free_price,
            sl.sale_listing_price_per_m2,
            sl.sale_listing_first_seen_at,
            sl.sale_listing_last_seen_at,
            sl.sale_listing_build_year,
            sl.sale_listing_total_floors,
            sl.sale_listing_elevator,
            sl.sale_listing_energy_efficiency_label,
            sl.sale_listing_url,
            sl.shortcut_ad_id,
            sl.frontdoor_ad_id,
            sl.frontdoor_building_announcement_id,
            sa.shortcut_building_id,
            fba.frontdoor_building_id,
            COALESCE(
                sa.shortcut_ad_data #>> '{adData,housingCompanyBusinessId}',
                fa.frontdoor_ad_data #>> '{property,housingCompany,businessId}',
                fb.frontdoor_building_business_id
            ) AS business_id,
            COALESCE(
                sa.shortcut_ad_data #>> '{adData,housingCompanyName}',
                sb.shortcut_building_housing_company,
                fa.frontdoor_ad_data #>> '{property,housingCompany,name}',
                fb.frontdoor_building_company_name
            ) AS housing_company_name,
            COALESCE(
                sa.shortcut_ad_data #>> '{buildingId}',
                sa.shortcut_ad_data #>> '{adData,buildingId}',
                sb.shortcut_building_external_id::text,
                sb.shortcut_building_building_id,
                fa.frontdoor_ad_data #>> '{property,housingCompany,id}',
                fb.frontdoor_building_housing_company_id::text,
                fba.frontdoor_building_id::text
            ) AS provider_building_id
        FROM public.sale_listings sl
        LEFT JOIN public.shortcut_ads sa ON sa.shortcut_ad_id = sl.shortcut_ad_id
        LEFT JOIN public.shortcut_buildings sb ON sb.shortcut_building_id = sa.shortcut_building_id
        LEFT JOIN public.frontdoor_ads fa ON fa.frontdoor_ad_id = sl.frontdoor_ad_id
        LEFT JOIN public.frontdoor_building_announcements fba ON fba.frontdoor_building_announcement_id = sl.frontdoor_building_announcement_id
        LEFT JOIN public.frontdoor_buildings fb ON fb.frontdoor_building_id = fba.frontdoor_building_id
        WHERE sl.sale_listing_id = listing_id
    ),
    identity_values AS (
        SELECT
            *,
            COALESCE(
                'business:' || public.fnc__canonical_identity_part(business_id),
                'provider_housing_company:' || sale_listing_source_provider || ':' || public.fnc__canonical_identity_part(provider_building_id),
                'address:' || public.fnc__canonical_identity_part(concat_ws('|', sale_listing_postal_norm, sale_listing_city_norm, sale_listing_building_match_key, housing_company_name)),
                'source:' || sale_listing_source_provider || ':' || sale_listing_source_kind || ':' || sale_listing_native_id
            ) AS housing_company_key
        FROM source_values
    ),
    inserted AS (
        INSERT INTO public.housing_companies (
            housing_company_identity_key,
            housing_company_postal_norm,
            housing_company_city_norm,
            housing_company_address_norm,
            housing_company_name,
            housing_company_business_id,
            housing_company_build_year,
            housing_company_floor_count,
            housing_company_elevator,
            housing_company_energy_efficiency_label,
            housing_company_match_reasons,
            housing_company_updated_at
        )
        SELECT
            housing_company_key,
            sale_listing_postal_norm,
            sale_listing_city_norm,
            sale_listing_address_norm,
            housing_company_name,
            business_id,
            sale_listing_build_year,
            sale_listing_total_floors,
            sale_listing_elevator,
            sale_listing_energy_efficiency_label,
            jsonb_build_object('source', sale_listing_source_provider, 'provider_building_id', provider_building_id),
            now()
        FROM identity_values
        ON CONFLICT (housing_company_identity_key) DO UPDATE SET
            housing_company_postal_norm = COALESCE(housing_companies.housing_company_postal_norm, EXCLUDED.housing_company_postal_norm),
            housing_company_city_norm = COALESCE(housing_companies.housing_company_city_norm, EXCLUDED.housing_company_city_norm),
            housing_company_address_norm = COALESCE(housing_companies.housing_company_address_norm, EXCLUDED.housing_company_address_norm),
            housing_company_name = COALESCE(housing_companies.housing_company_name, EXCLUDED.housing_company_name),
            housing_company_business_id = COALESCE(housing_companies.housing_company_business_id, EXCLUDED.housing_company_business_id),
            housing_company_build_year = COALESCE(housing_companies.housing_company_build_year, EXCLUDED.housing_company_build_year),
            housing_company_floor_count = COALESCE(housing_companies.housing_company_floor_count, EXCLUDED.housing_company_floor_count),
            housing_company_elevator = COALESCE(housing_companies.housing_company_elevator, EXCLUDED.housing_company_elevator),
            housing_company_energy_efficiency_label = COALESCE(housing_companies.housing_company_energy_efficiency_label, EXCLUDED.housing_company_energy_efficiency_label),
            housing_company_updated_at = now()
        RETURNING housing_company_id
    )
    SELECT inserted.housing_company_id INTO v_housing_company_id FROM inserted;
    WITH source_values AS (
        SELECT sl.*, hc.housing_company_id, hc.housing_company_identity_key
        FROM public.sale_listings sl
        JOIN public.housing_companies hc ON hc.housing_company_id = v_housing_company_id
        WHERE sl.sale_listing_id = listing_id
    ),
    identity_values AS (
        SELECT
            *,
            housing_company_identity_key || ':unit:' || COALESCE(
                public.fnc__canonical_identity_part(sale_listing_unit_match_key),
                public.fnc__canonical_identity_part(concat_ws('|', sale_listing_floor_level::text, sale_listing_area_value::text, public.fnc__layout_match_key(sale_listing_room_layout))),
                sale_listing_id::text
            ) AS unit_key
        FROM source_values
    ),
    inserted AS (
        INSERT INTO public.property_units (
            housing_company_id,
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
            housing_company_id,
            unit_key,
            sale_listing_address_norm,
            sale_listing_floor_level,
            sale_listing_area_value,
            sale_listing_rooms_count,
            sale_listing_room_layout,
            public.fnc__layout_match_key(sale_listing_room_layout),
            jsonb_build_object('source_listing_id', sale_listing_id),
            now()
        FROM identity_values
        ON CONFLICT (property_unit_identity_key) DO UPDATE SET
            housing_company_id = EXCLUDED.housing_company_id,
            property_unit_address_norm = COALESCE(property_units.property_unit_address_norm, EXCLUDED.property_unit_address_norm),
            property_unit_floor_level = COALESCE(property_units.property_unit_floor_level, EXCLUDED.property_unit_floor_level),
            property_unit_area_value = COALESCE(property_units.property_unit_area_value, EXCLUDED.property_unit_area_value),
            property_unit_rooms_count = COALESCE(property_units.property_unit_rooms_count, EXCLUDED.property_unit_rooms_count),
            property_unit_room_layout = COALESCE(property_units.property_unit_room_layout, EXCLUDED.property_unit_room_layout),
            property_unit_layout_match_key = COALESCE(property_units.property_unit_layout_match_key, EXCLUDED.property_unit_layout_match_key),
            property_unit_updated_at = now()
        RETURNING property_unit_id
    )
    SELECT property_unit_id INTO unit_id FROM inserted;
    WITH source_values AS (
        SELECT sl.*, pu.property_unit_id, pu.property_unit_identity_key
        FROM public.sale_listings sl
        JOIN public.property_units pu ON pu.property_unit_id = unit_id
        WHERE sl.sale_listing_id = listing_id
    ),
    identity_values AS (
        SELECT *, property_unit_identity_key || ':sale:' || COALESCE(sale_listing_debt_free_price::text, sale_listing_asking_price::text, 'unknown') AS offering_key
        FROM source_values
    ),
    inserted AS (
        INSERT INTO public.property_offerings (
            property_unit_id,
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
            property_unit_id,
            offering_key,
            'sale',
            sale_listing_headline,
            sale_listing_asking_price,
            sale_listing_debt_free_price,
            sale_listing_price_per_m2,
            sale_listing_first_seen_at,
            sale_listing_last_seen_at,
            sale_listing_prices_match_status,
            sale_listing_id,
            jsonb_build_object('source_listing_id', sale_listing_id, 'identity_key', offering_key),
            now()
        FROM identity_values
        ON CONFLICT (property_offering_identity_key) DO UPDATE SET
            property_offering_headline = CASE WHEN EXCLUDED.property_offering_last_seen_at >= COALESCE(property_offerings.property_offering_last_seen_at, '-infinity'::timestamptz) THEN EXCLUDED.property_offering_headline ELSE property_offerings.property_offering_headline END,
            property_offering_asking_price = COALESCE(EXCLUDED.property_offering_asking_price, property_offerings.property_offering_asking_price),
            property_offering_debt_free_price = COALESCE(EXCLUDED.property_offering_debt_free_price, property_offerings.property_offering_debt_free_price),
            property_offering_price_per_m2 = COALESCE(EXCLUDED.property_offering_price_per_m2, property_offerings.property_offering_price_per_m2),
            property_offering_first_seen_at = LEAST(COALESCE(property_offerings.property_offering_first_seen_at, EXCLUDED.property_offering_first_seen_at), COALESCE(EXCLUDED.property_offering_first_seen_at, property_offerings.property_offering_first_seen_at)),
            property_offering_last_seen_at = GREATEST(COALESCE(property_offerings.property_offering_last_seen_at, EXCLUDED.property_offering_last_seen_at), COALESCE(EXCLUDED.property_offering_last_seen_at, property_offerings.property_offering_last_seen_at)),
            primary_sale_listing_id = CASE WHEN EXCLUDED.property_offering_last_seen_at >= COALESCE(property_offerings.property_offering_last_seen_at, '-infinity'::timestamptz) THEN EXCLUDED.primary_sale_listing_id ELSE property_offerings.primary_sale_listing_id END,
            property_offering_updated_at = now()
        RETURNING property_offering_id
    )
    SELECT property_offering_id INTO offering_id FROM inserted;
    INSERT INTO public.property_offering_sources (
        property_offering_id,
        sale_listing_id,
        property_offering_source_link_status,
        property_offering_source_link_method,
        property_offering_source_link_score,
        property_offering_source_link_reasons,
        property_offering_source_updated_at
    )
    VALUES (
        offering_id,
        listing_id,
        'confirmed',
        link_method,
        120,
        jsonb_build_object('matched_by', 'canonical_identity_key'),
        now()
    )
    ON CONFLICT (sale_listing_id) WHERE property_offering_source_link_status <> 'rejected' DO UPDATE SET
        property_offering_id = CASE WHEN property_offering_sources.property_offering_source_link_method = ANY (ARRAY['manual'::text, 'source_match_auto'::text]) THEN property_offering_sources.property_offering_id ELSE EXCLUDED.property_offering_id END,
        property_offering_source_link_status = CASE WHEN property_offering_sources.property_offering_source_link_method = ANY (ARRAY['manual'::text, 'source_match_auto'::text]) THEN property_offering_sources.property_offering_source_link_status ELSE EXCLUDED.property_offering_source_link_status END,
        property_offering_source_link_method = CASE WHEN property_offering_sources.property_offering_source_link_method = ANY (ARRAY['manual'::text, 'source_match_auto'::text]) THEN property_offering_sources.property_offering_source_link_method ELSE EXCLUDED.property_offering_source_link_method END,
        property_offering_source_link_score = CASE WHEN property_offering_sources.property_offering_source_link_method = ANY (ARRAY['manual'::text, 'source_match_auto'::text]) THEN property_offering_sources.property_offering_source_link_score ELSE EXCLUDED.property_offering_source_link_score END,
        property_offering_source_link_reasons = CASE WHEN property_offering_sources.property_offering_source_link_method = ANY (ARRAY['manual'::text, 'source_match_auto'::text]) THEN property_offering_sources.property_offering_source_link_reasons ELSE EXCLUDED.property_offering_source_link_reasons END,
        property_offering_source_updated_at = now();
    SELECT public.fnc__housing_company_source_upsert(
        v_housing_company_id,
        sl.sale_listing_source_provider,
        sl.sale_listing_source_kind,
        'sale_listings',
        sl.sale_listing_id::text,
        sl.sale_listing_native_id,
        sl.sale_listing_url,
        link_method,
        120,
        jsonb_build_object('matched_by', 'sale_listing_sync'),
        sl.sale_listing_first_seen_at,
        sl.sale_listing_last_seen_at
    ) INTO source_link_id
    FROM public.sale_listings sl
    WHERE sl.sale_listing_id = listing_id;
    INSERT INTO public.property_offering_transactions (
        property_offering_id,
        prices_transaction_id,
        property_offering_transaction_link_status,
        property_offering_transaction_link_method,
        property_offering_transaction_link_score,
        property_offering_transaction_link_reasons,
        property_offering_transaction_updated_at
    )
    SELECT
        COALESCE(pos.property_offering_id, offering_id),
        sl.prices_transaction_id,
        COALESCE(sl.sale_listing_prices_match_status, 'confirmed'),
        'sync_auto',
        COALESCE(c.sale_listing_prices_transaction_match_score, 120),
        COALESCE(c.sale_listing_prices_transaction_match_reasons, '{}'::jsonb),
        now()
    FROM public.sale_listings sl
    LEFT JOIN public.property_offering_sources pos ON pos.sale_listing_id = sl.sale_listing_id
        AND pos.property_offering_source_link_status <> 'rejected'
    LEFT JOIN LATERAL (
        SELECT c.sale_listing_prices_transaction_match_score, c.sale_listing_prices_transaction_match_reasons
        FROM public.sale_listing_prices_transaction_match_candidates c
        WHERE c.sale_listing_id = sl.sale_listing_id
            AND c.prices_transaction_id = sl.prices_transaction_id
        ORDER BY c.sale_listing_prices_transaction_match_created_at DESC
        LIMIT 1
    ) c ON true
    WHERE sl.sale_listing_id = listing_id
        AND sl.prices_transaction_id IS NOT NULL
    ON CONFLICT (property_offering_id, prices_transaction_id) DO UPDATE SET
        property_offering_transaction_link_status = EXCLUDED.property_offering_transaction_link_status,
        property_offering_transaction_link_method = EXCLUDED.property_offering_transaction_link_method,
        property_offering_transaction_link_score = EXCLUDED.property_offering_transaction_link_score,
        property_offering_transaction_link_reasons = EXCLUDED.property_offering_transaction_link_reasons,
        property_offering_transaction_updated_at = now();
    RETURN offering_id;
END;
$$;
