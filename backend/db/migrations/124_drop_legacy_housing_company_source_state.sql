DROP FUNCTION IF EXISTS public.fnc__refresh_housing_company_facts_for_property_source_offering(uuid);
DROP FUNCTION IF EXISTS public.fnc__housing_company_source_upsert(uuid, text, text, text, text, text, text, text, integer, jsonb, timestamptz, timestamptz);
DROP TABLE IF EXISTS public.housing_company_facts;
DROP TABLE IF EXISTS public.housing_company_sources;
