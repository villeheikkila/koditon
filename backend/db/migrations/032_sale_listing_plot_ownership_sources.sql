CREATE OR REPLACE FUNCTION public.fnc__plot_owned(value text)
RETURNS boolean
LANGUAGE sql
STABLE
AS $$
    SELECT CASE public.fnc__match_alias_key(value)
        WHEN '1' THEN true
        WHEN 'oma' THEN true
        WHEN 'own' THEN true
        WHEN 'owned' THEN true
        WHEN 'omistus' THEN true
        WHEN 'omistettu' THEN true
        WHEN '2' THEN false
        WHEN '3' THEN false
        WHEN 'vuokra' THEN false
        WHEN 'rent' THEN false
        WHEN 'rented' THEN false
        WHEN 'rental' THEN false
        WHEN 'lease' THEN false
        WHEN 'leased' THEN false
        WHEN 'vuokralla' THEN false
        WHEN 'vuokratontti' THEN false
        WHEN 'optional_rental' THEN false
        WHEN 'valinnainen_vuokratontti' THEN false
        ELSE NULL
    END
$$;
CREATE OR REPLACE FUNCTION public.fnc__sale_listings_set_transaction_match_fields()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    property_raw text;
    plot_raw text;
    plot_owned boolean;
    elevator_value boolean;
    energy_label text;
    energy_match_label text;
    energy_normalized record;
BEGIN
    IF NEW.shortcut_ad_id IS NOT NULL THEN
        SELECT
            NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{adData,habitationType}', sa.shortcut_ad_data #>> '{adData,buildingType}', sa.shortcut_ad_data #>> '{buildingData,buildingType}')), ''),
            NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{adData,plotType}', sa.shortcut_ad_data #>> '{property,plotType}', sa.shortcut_ad_data #>> '{adData,buildingOverrideLotOwnership}', sb.shortcut_building_plot_type)), ''),
            COALESCE(sa.shortcut_ad_elevator, public.fnc__try_parse_bool(sb.shortcut_building_has_elevator)),
            public.fnc__energy_efficiency_label(sa.shortcut_ad_data #>> '{adData,buildingOverrideEnergyClass}', sa.shortcut_ad_data #>> '{adData,energyClass}', sa.shortcut_ad_data #>> '{property,energyClass}', sa.shortcut_ad_energy_class),
            public.fnc__energy_efficiency_match_label(sa.shortcut_ad_data #>> '{adData,buildingOverrideEnergyClass}', sa.shortcut_ad_data #>> '{adData,energyClass}', sa.shortcut_ad_data #>> '{property,energyClass}', sa.shortcut_ad_energy_class)
        INTO property_raw, plot_raw, elevator_value, energy_label, energy_match_label
        FROM public.shortcut_ads sa
        LEFT JOIN public.shortcut_buildings sb ON sb.shortcut_building_id = sa.shortcut_building_id
        WHERE sa.shortcut_ad_id = NEW.shortcut_ad_id;
    ELSIF NEW.frontdoor_ad_id IS NOT NULL THEN
        SELECT
            NULLIF(trim(COALESCE(fa.frontdoor_ad_data #>> '{property,residentialPropertyType}', fa.frontdoor_ad_data #>> '{property,specificType}', fa.frontdoor_ad_data #>> '{property,propertyType}')), ''),
            NULLIF(trim(COALESCE(fa.frontdoor_ad_data #>> '{property,plot,holdingType}', fa.frontdoor_ad_data #>> '{property,housingCompany,plot,holdingType}', fa.frontdoor_ad_data #>> '{property,plotOwnershipType}', fa.frontdoor_ad_plot_type)), ''),
            fa.frontdoor_ad_elevator,
            public.fnc__energy_efficiency_label(fa.frontdoor_ad_data #>> '{property,housingCompany,energyCertificate,energyCertificateDescription}', fa.frontdoor_ad_data #>> '{property,energyCertificate,energyCertificateDescription}', fa.frontdoor_ad_data #>> '{property,housingCompany,energyCertificate,energyCertificateType}', fa.frontdoor_ad_data #>> '{property,energyCertificate,energyCertificateType}', fa.frontdoor_ad_energy_class),
            public.fnc__energy_efficiency_match_label(fa.frontdoor_ad_data #>> '{property,housingCompany,energyCertificate,energyCertificateDescription}', fa.frontdoor_ad_data #>> '{property,energyCertificate,energyCertificateDescription}', fa.frontdoor_ad_data #>> '{property,housingCompany,energyCertificate,energyCertificateType}', fa.frontdoor_ad_data #>> '{property,energyCertificate,energyCertificateType}', fa.frontdoor_ad_energy_class)
        INTO property_raw, plot_raw, elevator_value, energy_label, energy_match_label
        FROM public.frontdoor_ads fa
        WHERE fa.frontdoor_ad_id = NEW.frontdoor_ad_id;
    ELSIF NEW.frontdoor_building_announcement_id IS NOT NULL THEN
        SELECT
            NULLIF(trim(COALESCE(fba.frontdoor_building_announcement_property_subtype, fba.frontdoor_building_announcement_property_type)), ''),
            NULL::text,
            fb.frontdoor_building_has_elevator,
            public.fnc__energy_efficiency_label(fb.frontdoor_building_energy_certificate_code),
            public.fnc__energy_efficiency_match_label(fb.frontdoor_building_energy_certificate_code)
        INTO property_raw, plot_raw, elevator_value, energy_label, energy_match_label
        FROM public.frontdoor_building_announcements fba
        JOIN public.frontdoor_buildings fb ON fb.frontdoor_building_id = fba.frontdoor_building_id
        WHERE fba.frontdoor_building_announcement_id = NEW.frontdoor_building_announcement_id;
    END IF;
    SELECT * INTO energy_normalized FROM public.fnc__energy_efficiency_normalized(energy_match_label);
    plot_owned := public.fnc__plot_owned(plot_raw);
    NEW.sale_listing_property_type_raw := property_raw;
    NEW.sale_listing_property_type_code := public.fnc__sale_listing_property_type_code(property_raw);
    NEW.sale_listing_room_category_code := public.fnc__sale_listing_room_category_code(NEW.sale_listing_rooms_count, NEW.sale_listing_room_layout);
    NEW.sale_listing_floor_text := public.fnc__sale_listing_floor_text(NEW.sale_listing_floor_level, NEW.sale_listing_total_floors);
    NEW.sale_listing_elevator := elevator_value;
    NEW.sale_listing_plot_type_raw := plot_raw;
    NEW.sale_listing_plot_type_code := CASE WHEN plot_owned IS TRUE THEN 'own' WHEN plot_owned IS FALSE THEN 'rent' ELSE NULL END;
    NEW.sale_listing_plot_owned := plot_owned;
    NEW.sale_listing_energy_efficiency_label := energy_label;
    NEW.sale_listing_energy_efficiency_class_code := energy_normalized.energy_efficiency_class_code;
    NEW.sale_listing_energy_efficiency_standard_year := energy_normalized.energy_efficiency_standard_year;
    NEW.sale_listing_energy_efficiency_status := energy_normalized.energy_efficiency_status;
    NEW.sale_listing_energy_efficiency_match_code := energy_normalized.energy_efficiency_match_code;
    RETURN NEW;
END;
$$;
WITH source_values AS (
    SELECT
        sl.sale_listing_id,
        NULLIF(trim(COALESCE(
            sa.shortcut_ad_data #>> '{adData,plotType}',
            sa.shortcut_ad_data #>> '{property,plotType}',
            sa.shortcut_ad_data #>> '{adData,buildingOverrideLotOwnership}',
            sb.shortcut_building_plot_type,
            fa.frontdoor_ad_data #>> '{property,plot,holdingType}',
            fa.frontdoor_ad_data #>> '{property,housingCompany,plot,holdingType}',
            fa.frontdoor_ad_data #>> '{property,plotOwnershipType}',
            fa.frontdoor_ad_plot_type,
            sl.sale_listing_plot_type_raw
        )), '') AS plot_raw
    FROM public.sale_listings sl
    LEFT JOIN public.shortcut_ads sa ON sa.shortcut_ad_id = sl.shortcut_ad_id
    LEFT JOIN public.shortcut_buildings sb ON sb.shortcut_building_id = sa.shortcut_building_id
    LEFT JOIN public.frontdoor_ads fa ON fa.frontdoor_ad_id = sl.frontdoor_ad_id
    WHERE sl.sale_listing_plot_owned IS NULL
        OR NULLIF(sl.sale_listing_plot_type_raw, '') IS NULL
        OR sl.sale_listing_plot_type_code IS NULL
),
normalized AS (
    SELECT
        sale_listing_id,
        plot_raw,
        public.fnc__plot_owned(plot_raw) AS plot_owned
    FROM source_values
)
UPDATE public.sale_listings sl
SET
    sale_listing_plot_type_raw = COALESCE(NULLIF(sl.sale_listing_plot_type_raw, ''), normalized.plot_raw),
    sale_listing_plot_type_code = COALESCE(sl.sale_listing_plot_type_code, CASE WHEN normalized.plot_owned IS TRUE THEN 'own' WHEN normalized.plot_owned IS FALSE THEN 'rent' ELSE NULL END),
    sale_listing_plot_owned = normalized.plot_owned
FROM normalized
WHERE sl.sale_listing_id = normalized.sale_listing_id
    AND normalized.plot_owned IS NOT NULL
    AND (
        sl.sale_listing_plot_owned IS DISTINCT FROM normalized.plot_owned
        OR NULLIF(sl.sale_listing_plot_type_raw, '') IS NULL
        OR sl.sale_listing_plot_type_code IS NULL
    );
