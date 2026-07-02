DO $$
DECLARE
    fn record;
BEGIN
    FOR fn IN
        SELECT n.nspname AS schema_name, p.proname AS function_name, pg_get_function_identity_arguments(p.oid) AS identity_arguments
        FROM pg_proc p
        JOIN pg_namespace n ON n.oid = p.pronamespace
        WHERE n.nspname = 'public'
            AND p.proname IN (
                'fnc__normalize_address_token',
                'fnc__normalize_postal',
                'fnc__prices_transaction_energy_match_code',
                'fnc__sale_listing_property_type_code'
            )
    LOOP
        EXECUTE format('DROP FUNCTION IF EXISTS %I.%I(%s)', fn.schema_name, fn.function_name, fn.identity_arguments);
    END LOOP;
END $$;
