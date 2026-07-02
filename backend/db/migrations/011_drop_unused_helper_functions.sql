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
                'fnc__frontdoor_published_at',
                'fnc__jsonb_periodic_charge_price',
                'fnc__layout_exact_match_key',
                'fnc__layout_match_key',
                'fnc__plot_owned',
                'fnc__prices_transaction_floor_level',
                'fnc__prices_transaction_layout_is_truncated',
                'fnc__prices_transaction_layout_match_key',
                'fnc__prices_transaction_total_floors',
                'fnc__refresh_housing_company_geom',
                'fnc__sale_listing_floor_text',
                'fnc__sale_listing_plot_type_code',
                'fnc__sale_listing_room_category_code'
            )
    LOOP
        EXECUTE format('DROP FUNCTION IF EXISTS %I.%I(%s)', fn.schema_name, fn.function_name, fn.identity_arguments);
    END LOOP;
END $$;
