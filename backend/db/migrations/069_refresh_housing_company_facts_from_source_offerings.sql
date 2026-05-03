CREATE OR REPLACE FUNCTION public.fnc__refresh_housing_company_facts_for_property_source_offering(target_sale_listing_id uuid)
RETURNS void
LANGUAGE sql
AS $$
WITH source_row AS (
    SELECT
        sl.*,
        pu.housing_company_id,
        hcs.housing_company_source_id
    FROM public.property_source_offerings sl
    JOIN public.property_offering_sources pos ON pos.sale_listing_id = sl.sale_listing_id
        AND pos.property_offering_source_link_status <> 'rejected'
    JOIN public.property_offerings po ON po.property_offering_id = pos.property_offering_id
    JOIN public.property_units pu ON pu.property_unit_id = po.property_unit_id
    LEFT JOIN public.housing_company_sources hcs ON hcs.housing_company_source_id_value = sl.sale_listing_id::text
    WHERE sl.sale_listing_id = target_sale_listing_id
    LIMIT 1
),
deleted AS (
    DELETE FROM public.housing_company_facts hcf
    USING source_row
    WHERE hcf.housing_company_source_id = source_row.housing_company_source_id
),
base_facts AS (
    SELECT
        source_row.housing_company_id,
        source_row.housing_company_source_id,
        fact.fact_key,
        fact.value_text,
        fact.value_number,
        fact.value_bool,
        NULL::jsonb AS value_json,
        COALESCE(fact.value_text, fact.value_number::text, fact.value_bool::text) AS raw_value,
        source_row.sale_listing_first_seen_at,
        source_row.sale_listing_last_seen_at
    FROM source_row
    CROSS JOIN LATERAL (
        SELECT *
        FROM (
            VALUES
                ('housing_company_name', source_row.sale_listing_housing_company_name, NULL::double precision, NULL::boolean),
                ('housing_company_business_id', source_row.sale_listing_housing_company_business_id, NULL::double precision, NULL::boolean),
                ('build_year', NULL::text, source_row.sale_listing_build_year::double precision, NULL::boolean),
                ('floor_count', NULL::text, source_row.sale_listing_total_floors::double precision, NULL::boolean),
                ('apartment_count', NULL::text, source_row.sale_listing_apartment_count::double precision, NULL::boolean),
                ('elevator', NULL::text, NULL::double precision, source_row.sale_listing_elevator),
                ('energy_label', source_row.sale_listing_energy_efficiency_label, NULL::double precision, NULL::boolean),
                ('building_material', source_row.sale_listing_building_material, NULL::double precision, NULL::boolean),
                ('heating_system', source_row.sale_listing_heating_system, NULL::double precision, NULL::boolean),
                ('roof_type', source_row.sale_listing_roof_type, NULL::double precision, NULL::boolean),
                ('roof_material', source_row.sale_listing_roof_material, NULL::double precision, NULL::boolean),
                ('car_storage', source_row.sale_listing_car_storage_text, NULL::double precision, NULL::boolean),
                ('description', source_row.sale_listing_building_description_text, NULL::double precision, NULL::boolean),
                ('other_info', source_row.sale_listing_building_other_info_text, NULL::double precision, NULL::boolean)
        ) AS v(fact_key, value_text, value_number, value_bool)
    ) AS fact
    WHERE COALESCE(fact.value_text, fact.value_number::text, fact.value_bool::text) IS NOT NULL
        AND COALESCE(fact.value_text, fact.value_number::text, fact.value_bool::text) <> ''
),
renovation_facts AS (
    SELECT
        source_row.housing_company_id,
        source_row.housing_company_source_id,
        'renovation.' || r.property_source_offering_renovation_category AS fact_key,
        r.property_source_offering_renovation_text AS value_text,
        r.property_source_offering_renovation_year::double precision AS value_number,
        NULL::boolean AS value_bool,
        jsonb_build_object(
            'category', r.property_source_offering_renovation_category,
            'status', r.property_source_offering_renovation_status,
            'year', r.property_source_offering_renovation_year,
            'text', r.property_source_offering_renovation_text
        ) AS value_json,
        COALESCE(r.property_source_offering_renovation_text, r.property_source_offering_renovation_year::text, r.property_source_offering_renovation_category) AS raw_value,
        source_row.sale_listing_first_seen_at,
        source_row.sale_listing_last_seen_at
    FROM source_row
    JOIN public.property_source_offering_renovations r ON r.sale_listing_id = source_row.sale_listing_id
),
facts AS (
    SELECT * FROM base_facts
    UNION ALL
    SELECT * FROM renovation_facts
)
INSERT INTO public.housing_company_facts (
    housing_company_id,
    housing_company_source_id,
    housing_company_fact_key,
    housing_company_fact_value_text,
    housing_company_fact_value_number,
    housing_company_fact_value_bool,
    housing_company_fact_value_json,
    housing_company_fact_raw_value,
    housing_company_fact_confidence,
    housing_company_fact_first_seen_at,
    housing_company_fact_last_seen_at,
    housing_company_fact_updated_at
)
SELECT
    housing_company_id,
    housing_company_source_id,
    fact_key,
    value_text,
    value_number,
    value_bool,
    value_json,
    raw_value,
    100,
    sale_listing_first_seen_at,
    sale_listing_last_seen_at,
    now()
FROM facts
ON CONFLICT DO NOTHING
$$;
