CREATE OR REPLACE FUNCTION public.fnc__frontdoor_ads_set_normalized_fields()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.frontdoor_ad_sauna := CASE
        WHEN jsonb_path_exists(COALESCE(NEW.frontdoor_ad_data, '{}'::jsonb), '$.residenceDetailsDTO.generalDwellingFeatures[*] ? (@ == "HAS_NO_SAUNA")') THEN false
        WHEN jsonb_path_exists(COALESCE(NEW.frontdoor_ad_data, '{}'::jsonb), '$.residenceDetailsDTO.generalDwellingFeatures[*] ? (@ == "HAS_SAUNA")') THEN true
        ELSE NULL
    END;
    RETURN NEW;
END;
$$;

ALTER TABLE public.frontdoor_ads
DROP COLUMN IF EXISTS frontdoor_ad_street_address,
DROP COLUMN IF EXISTS frontdoor_ad_city,
DROP COLUMN IF EXISTS frontdoor_ad_postal,
DROP COLUMN IF EXISTS frontdoor_ad_price,
DROP COLUMN IF EXISTS frontdoor_ad_area_value,
DROP COLUMN IF EXISTS frontdoor_ad_address_key,
DROP COLUMN IF EXISTS frontdoor_ad_search_text,
DROP COLUMN IF EXISTS frontdoor_ad_description_text,
DROP COLUMN IF EXISTS frontdoor_ad_availability_text,
DROP COLUMN IF EXISTS frontdoor_ad_renovations_done_text,
DROP COLUMN IF EXISTS frontdoor_ad_renovations_planned_text,
DROP COLUMN IF EXISTS frontdoor_ad_additional_info_text,
DROP COLUMN IF EXISTS frontdoor_ad_charges_text,
DROP COLUMN IF EXISTS frontdoor_ad_maintenance_charge_monthly,
DROP COLUMN IF EXISTS frontdoor_ad_total_charge_monthly,
DROP COLUMN IF EXISTS frontdoor_ad_water_charge,
DROP COLUMN IF EXISTS frontdoor_ad_debt_free_price,
DROP COLUMN IF EXISTS frontdoor_ad_debt_share_amount,
DROP COLUMN IF EXISTS frontdoor_ad_price_per_m2,
DROP COLUMN IF EXISTS frontdoor_ad_floor_level,
DROP COLUMN IF EXISTS frontdoor_ad_total_floors,
DROP COLUMN IF EXISTS frontdoor_ad_build_year,
DROP COLUMN IF EXISTS frontdoor_ad_condition,
DROP COLUMN IF EXISTS frontdoor_ad_energy_class,
DROP COLUMN IF EXISTS frontdoor_ad_plot_type,
DROP COLUMN IF EXISTS frontdoor_ad_elevator,
DROP COLUMN IF EXISTS frontdoor_ad_rooms_count;
