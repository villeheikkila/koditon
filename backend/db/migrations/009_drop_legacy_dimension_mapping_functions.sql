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
                'fnc__legacy_property_dimension_claim_scope',
                'fnc__legacy_property_dimension_key',
                'fnc__legacy_property_dimension_target_type',
                'fnc__legacy_property_dimension_value',
                'fnc__legacy_property_dimension_value_kind'
            )
    LOOP
        EXECUTE format('DROP FUNCTION IF EXISTS %I.%I(%s)', fn.schema_name, fn.function_name, fn.identity_arguments);
    END LOOP;
END $$;
