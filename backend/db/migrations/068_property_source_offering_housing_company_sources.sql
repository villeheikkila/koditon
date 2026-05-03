UPDATE public.housing_company_sources
SET housing_company_source_table = 'property_source_offerings',
    housing_company_source_updated_at = now()
WHERE housing_company_source_table = 'sale_listings';

CREATE TEMP TABLE tmp__property_source_offering_function_defs AS
SELECT pg_get_functiondef(p.oid) AS definition
FROM pg_proc p
JOIN pg_namespace n ON n.oid = p.pronamespace
WHERE n.nspname = 'public'
    AND p.prokind IN ('f', 'p')
    AND pg_get_functiondef(p.oid) ILIKE '%''sale_listings''%';

DO $$
DECLARE
    fn record;
BEGIN
    FOR fn IN SELECT definition FROM tmp__property_source_offering_function_defs LOOP
        EXECUTE replace(fn.definition, '''sale_listings''', '''property_source_offerings''');
    END LOOP;
END;
$$;

DROP TABLE tmp__property_source_offering_function_defs;
