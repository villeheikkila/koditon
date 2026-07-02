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
                'fnc__try_parse_bigint',
                'fnc__try_parse_bool',
                'fnc__try_parse_float8',
                'fnc__try_parse_int4'
            )
    LOOP
        EXECUTE format('DROP FUNCTION IF EXISTS %I.%I(%s)', fn.schema_name, fn.function_name, fn.identity_arguments);
    END LOOP;
END $$;
