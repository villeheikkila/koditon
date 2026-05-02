CREATE OR REPLACE FUNCTION public.fnc__plot_owned(value text)
RETURNS boolean
LANGUAGE sql
STABLE
AS $$
    SELECT CASE public.fnc__match_alias_key(value)
        WHEN 'oma' THEN true
        WHEN 'own' THEN true
        WHEN 'owned' THEN true
        WHEN 'omistus' THEN true
        WHEN 'omistettu' THEN true
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
CREATE OR REPLACE FUNCTION public.fnc__plot_owned_label(value boolean)
RETURNS text
LANGUAGE sql
IMMUTABLE
AS $$
    SELECT CASE
        WHEN value IS TRUE THEN 'Owned'
        WHEN value IS FALSE THEN 'Rented'
        ELSE NULL
    END
$$;
CREATE OR REPLACE FUNCTION public.fnc__sale_listing_plot_type_code(value text)
RETURNS text
LANGUAGE sql
STABLE
AS $$
    SELECT CASE
        WHEN public.fnc__plot_owned(value) IS TRUE THEN 'own'
        WHEN public.fnc__plot_owned(value) IS FALSE THEN 'rent'
        ELSE NULL
    END
$$;
CREATE OR REPLACE FUNCTION public.fnc__prices_transaction_plot_type_code(value text)
RETURNS text
LANGUAGE sql
STABLE
AS $$
    SELECT public.fnc__sale_listing_plot_type_code(value)
$$;
ALTER TABLE public.sale_listings
ADD COLUMN IF NOT EXISTS sale_listing_plot_owned boolean;
ALTER TABLE public.prices_transactions
ADD COLUMN IF NOT EXISTS prices_transaction_plot_owned boolean;
CREATE OR REPLACE FUNCTION public.fnc__prices_transactions_set_plot_owned()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.prices_transaction_plot_owned := public.fnc__plot_owned(NEW.prices_transaction_plot);
    RETURN NEW;
END;
$$;
DROP TRIGGER IF EXISTS trg__prices_transactions_set_plot_owned ON public.prices_transactions;
CREATE TRIGGER trg__prices_transactions_set_plot_owned
BEFORE INSERT OR UPDATE OF prices_transaction_plot ON public.prices_transactions
FOR EACH ROW
EXECUTE FUNCTION public.fnc__prices_transactions_set_plot_owned();
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
            NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{adData,plotType}', sa.shortcut_ad_data #>> '{property,plotType}', sb.shortcut_building_plot_type)), ''),
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
            NULLIF(trim(COALESCE(fa.frontdoor_ad_data #>> '{property,plot,holdingType}', fa.frontdoor_ad_data #>> '{property,plotOwnershipType}', fa.frontdoor_ad_plot_type)), ''),
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
ALTER TABLE public.sale_listings DISABLE TRIGGER trg__sale_listings_set_transaction_match_fields;
WITH source_values AS (
    SELECT
        sl.sale_listing_id,
        public.fnc__plot_owned(NULLIF(trim(COALESCE(
            sa.shortcut_ad_data #>> '{adData,plotType}',
            sa.shortcut_ad_data #>> '{property,plotType}',
            sb.shortcut_building_plot_type,
            fa.frontdoor_ad_data #>> '{property,plot,holdingType}',
            fa.frontdoor_ad_data #>> '{property,plotOwnershipType}',
            fa.frontdoor_ad_plot_type,
            sl.sale_listing_plot_type_raw
        )), '')) AS plot_owned
    FROM public.sale_listings sl
    LEFT JOIN public.shortcut_ads sa ON sa.shortcut_ad_id = sl.shortcut_ad_id
    LEFT JOIN public.shortcut_buildings sb ON sb.shortcut_building_id = sa.shortcut_building_id
    LEFT JOIN public.frontdoor_ads fa ON fa.frontdoor_ad_id = sl.frontdoor_ad_id
)
UPDATE public.sale_listings sl
SET sale_listing_plot_owned = source_values.plot_owned
FROM source_values
WHERE sl.sale_listing_id = source_values.sale_listing_id
    AND sl.sale_listing_plot_owned IS DISTINCT FROM source_values.plot_owned;
ALTER TABLE public.sale_listings ENABLE TRIGGER trg__sale_listings_set_transaction_match_fields;
UPDATE public.prices_transactions
SET prices_transaction_plot_owned = public.fnc__plot_owned(prices_transaction_plot)
WHERE prices_transaction_plot_owned IS DISTINCT FROM public.fnc__plot_owned(prices_transaction_plot);
CREATE INDEX IF NOT EXISTS idx_sale_listings_plot_owned ON public.sale_listings (sale_listing_plot_owned);
CREATE INDEX IF NOT EXISTS idx_prices_transactions_plot_owned ON public.prices_transactions (prices_transaction_plot_owned);
