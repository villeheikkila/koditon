CREATE OR REPLACE FUNCTION public.fnc__legacy_property_dimension_target_type(p_target_type text)
RETURNS text
LANGUAGE sql
IMMUTABLE
AS $$
SELECT CASE p_target_type
    WHEN 'sale_listing' THEN 'listing'
    WHEN 'property_unit' THEN 'unit'
    WHEN 'physical_building' THEN 'building'
    ELSE p_target_type
END
$$;
CREATE OR REPLACE FUNCTION public.fnc__legacy_property_dimension_claim_scope(p_target_type text)
RETURNS text
LANGUAGE sql
IMMUTABLE
AS $$
SELECT CASE
    WHEN p_target_type IN ('sale_listing','document','transaction') THEN 'source'
    ELSE 'canonical'
END
$$;
CREATE OR REPLACE FUNCTION public.fnc__legacy_property_dimension_key(p_section text, p_key text)
RETURNS text
LANGUAGE sql
IMMUTABLE
AS $$
SELECT CASE p_section || '.' || p_key
    WHEN 'unit.balcony' THEN 'features.balcony'
    WHEN 'balcony.has_balcony' THEN 'features.balcony'
    WHEN 'balcony.glazing' THEN 'features.balcony_glazing'
    WHEN 'unit.sauna' THEN 'features.sauna'
    WHEN 'sauna.has_sauna' THEN 'features.sauna'
    WHEN 'sauna.private_sauna' THEN 'features.private_sauna'
    WHEN 'parking.parking_text' THEN 'features.parking_type'
    WHEN 'storage.storage_quality' THEN 'features.storage_quality'
    WHEN 'views.view_quality' THEN 'features.view_quality'
    WHEN 'views.noise_risk' THEN 'features.noise_risk'
    WHEN 'condition.condition' THEN 'condition.unit_condition'
    WHEN 'layout.layout_quality' THEN 'layout.quality'
    WHEN 'layout.awkward_layout' THEN 'layout.awkward'
    WHEN 'heating.heating_method' THEN 'building.heating_method'
    WHEN 'charges.maintenance_charge_monthly' THEN 'charges.maintenance_monthly_eur'
    WHEN 'charges.capital_charge_monthly' THEN 'charges.capital_monthly_eur'
    WHEN 'charges.total_charge_monthly' THEN 'charges.total_monthly_eur'
    ELSE p_section || '.' || p_key
END
$$;
CREATE OR REPLACE FUNCTION public.fnc__legacy_property_dimension_value_kind(p_value_kind text, p_value_json jsonb)
RETURNS text
LANGUAGE sql
IMMUTABLE
AS $$
SELECT CASE p_value_kind
    WHEN 'text' THEN 'string'
    WHEN 'bool' THEN 'boolean'
    WHEN 'json' THEN COALESCE(jsonb_typeof(p_value_json), 'object')
    ELSE p_value_kind
END
$$;
CREATE OR REPLACE FUNCTION public.fnc__legacy_property_dimension_value(p_value_kind text, p_value_text text, p_value_number double precision, p_value_bool boolean, p_value_json jsonb)
RETURNS jsonb
LANGUAGE sql
IMMUTABLE
AS $$
SELECT CASE p_value_kind
    WHEN 'text' THEN to_jsonb(p_value_text)
    WHEN 'number' THEN to_jsonb(p_value_number)
    WHEN 'bool' THEN to_jsonb(p_value_bool)
    WHEN 'json' THEN p_value_json
    ELSE NULL::jsonb
END
$$;
DROP VIEW IF EXISTS public.property_quality_scores CASCADE;
DROP VIEW IF EXISTS public.housing_company_systems CASCADE;
DROP VIEW IF EXISTS public.housing_company_renovations CASCADE;
DROP VIEW IF EXISTS public.housing_company_profiles CASCADE;
DROP VIEW IF EXISTS public.building_profiles CASCADE;
DROP VIEW IF EXISTS public.apartment_profiles CASCADE;
DROP VIEW IF EXISTS public.property_claims CASCADE;
DROP TABLE IF EXISTS public.property_quality_scores CASCADE;
DROP TABLE IF EXISTS public.housing_company_systems CASCADE;
DROP TABLE IF EXISTS public.housing_company_renovations CASCADE;
DROP TABLE IF EXISTS public.housing_company_profiles CASCADE;
DROP TABLE IF EXISTS public.building_profiles CASCADE;
DROP TABLE IF EXISTS public.apartment_profiles CASCADE;
DROP TABLE IF EXISTS public.property_claims CASCADE;
