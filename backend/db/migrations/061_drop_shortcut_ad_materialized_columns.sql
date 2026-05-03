CREATE OR REPLACE FUNCTION public.fnc__shortcut_ads_set_normalized_fields()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    RETURN NEW;
END;
$$;

DROP INDEX IF EXISTS public.idx_shortcut_ads_address_key;
DROP INDEX IF EXISTS public.idx_shortcut_ads_area_value;
DROP INDEX IF EXISTS public.idx_shortcut_ads_build_year;
DROP INDEX IF EXISTS public.idx_shortcut_ads_floor_level;
DROP INDEX IF EXISTS public.idx_shortcut_ads_maintenance_charge;
DROP INDEX IF EXISTS public.idx_shortcut_ads_postal;
DROP INDEX IF EXISTS public.idx_shortcut_ads_price;
DROP INDEX IF EXISTS public.idx_shortcut_ads_search_trgm;
DROP INDEX IF EXISTS public.idx_shortcut_ads_street_trgm;

ALTER TABLE public.shortcut_ads
DROP COLUMN IF EXISTS shortcut_ad_street_address,
DROP COLUMN IF EXISTS shortcut_ad_city,
DROP COLUMN IF EXISTS shortcut_ad_postal,
DROP COLUMN IF EXISTS shortcut_ad_price,
DROP COLUMN IF EXISTS shortcut_ad_area_value,
DROP COLUMN IF EXISTS shortcut_ad_address_key,
DROP COLUMN IF EXISTS shortcut_ad_search_text,
DROP COLUMN IF EXISTS shortcut_ad_description_text,
DROP COLUMN IF EXISTS shortcut_ad_availability_text,
DROP COLUMN IF EXISTS shortcut_ad_renovations_done_text,
DROP COLUMN IF EXISTS shortcut_ad_renovations_planned_text,
DROP COLUMN IF EXISTS shortcut_ad_additional_info_text,
DROP COLUMN IF EXISTS shortcut_ad_charges_text,
DROP COLUMN IF EXISTS shortcut_ad_maintenance_charge_monthly,
DROP COLUMN IF EXISTS shortcut_ad_total_charge_monthly,
DROP COLUMN IF EXISTS shortcut_ad_water_charge,
DROP COLUMN IF EXISTS shortcut_ad_debt_free_price,
DROP COLUMN IF EXISTS shortcut_ad_debt_share_amount,
DROP COLUMN IF EXISTS shortcut_ad_price_per_m2,
DROP COLUMN IF EXISTS shortcut_ad_floor_level,
DROP COLUMN IF EXISTS shortcut_ad_total_floors,
DROP COLUMN IF EXISTS shortcut_ad_build_year,
DROP COLUMN IF EXISTS shortcut_ad_condition,
DROP COLUMN IF EXISTS shortcut_ad_energy_class,
DROP COLUMN IF EXISTS shortcut_ad_plot_type,
DROP COLUMN IF EXISTS shortcut_ad_elevator,
DROP COLUMN IF EXISTS shortcut_ad_sauna,
DROP COLUMN IF EXISTS shortcut_ad_rooms_count;
