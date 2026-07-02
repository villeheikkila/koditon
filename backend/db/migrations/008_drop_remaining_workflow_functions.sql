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
                'fnc__canonical_identity_part',
                'fnc__mark_property_dimension_target_dirty',
                'fnc__mark_property_offering_dimension_targets_dirty',
                'fnc__mark_property_unit_dimension_targets_dirty',
                'fnc__relink_physical_building_housing_company',
                'fnc__relink_property_document_offering',
                'fnc__relink_property_unit_building',
                'fnc__sync_property_house_for_sale_listing'
            )
    LOOP
        EXECUTE format('DROP FUNCTION IF EXISTS %I.%I(%s)', fn.schema_name, fn.function_name, fn.identity_arguments);
    END LOOP;
END $$;
