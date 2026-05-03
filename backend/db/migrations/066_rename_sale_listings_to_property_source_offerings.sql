CREATE TEMP TABLE tmp__sale_listing_function_defs AS
SELECT pg_get_functiondef(p.oid) AS definition
FROM pg_proc p
JOIN pg_namespace n ON n.oid = p.pronamespace
WHERE n.nspname = 'public'
    AND p.prokind IN ('f', 'p')
    AND pg_get_functiondef(p.oid) ILIKE '%public.sale_listings%';

ALTER TABLE public.sale_listings RENAME TO property_source_offerings;

DO $$
DECLARE
    fn record;
BEGIN
    FOR fn IN SELECT definition FROM tmp__sale_listing_function_defs LOOP
        EXECUTE replace(fn.definition, 'public.sale_listings', 'public.property_source_offerings');
    END LOOP;
END;
$$;

DROP TABLE tmp__sale_listing_function_defs;
