DO $$
DECLARE
    ddl text;
BEGIN
    FOR ddl IN
        SELECT pg_get_functiondef(p.oid)
        FROM pg_proc p
        JOIN pg_namespace n ON n.oid = p.pronamespace
        WHERE n.nspname = 'public'
            AND p.prokind = 'f'
            AND (
                pg_get_functiondef(p.oid) LIKE '%public.property_dimension_claims%'
                OR pg_get_functiondef(p.oid) LIKE '%public.property_dimension_values%'
                OR pg_get_functiondef(p.oid) LIKE '%public.property_dimension_profiles%'
            )
    LOOP
        ddl := replace(ddl, 'public.property_dimension_claims', 'public.dimension_claims');
        ddl := replace(ddl, 'public.property_dimension_values', 'public.dimension_values');
        ddl := replace(ddl, 'public.property_dimension_profiles', 'public.dimension_profiles');
        ddl := replace(ddl, '''property_dimension_claims''', '''dimension_claims''');
        EXECUTE ddl;
    END LOOP;
END;
$$;
DROP TRIGGER IF EXISTS trg__sync_dimension_claim_from_legacy ON public.property_dimension_claims;
DROP TRIGGER IF EXISTS trg__sync_dimension_value_from_legacy ON public.property_dimension_values;
DROP TRIGGER IF EXISTS trg__sync_dimension_profile_from_legacy ON public.property_dimension_profiles;
DROP FUNCTION IF EXISTS public.fnc__sync_dimension_claim_from_legacy();
DROP FUNCTION IF EXISTS public.fnc__sync_dimension_value_from_legacy();
DROP FUNCTION IF EXISTS public.fnc__sync_dimension_profile_from_legacy();
