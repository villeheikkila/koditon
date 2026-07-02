DROP VIEW IF EXISTS public.view__prices_transactions;
DO $$
DECLARE
  fn record;
BEGIN
  FOR fn IN
    SELECT p.proname, pg_get_function_identity_arguments(p.oid) AS args
    FROM pg_proc p
    JOIN pg_namespace n ON n.oid = p.pronamespace
    WHERE n.nspname = 'public'
      AND p.proname = ANY (ARRAY[
        'fnc__area_match_key',
        'fnc__energy_efficiency_class_code',
        'fnc__energy_efficiency_label',
        'fnc__energy_efficiency_match_label',
        'fnc__energy_efficiency_standard_year',
        'fnc__energy_efficiency_status',
        'fnc__layout_match_code',
        'fnc__layout_prefix_match',
        'fnc__link_sale_listing_prices_transaction',
        'fnc__mark_listing_dimension_targets_dirty',
        'fnc__merge_housing_companies',
        'fnc__merge_property_offerings',
        'fnc__normalize_match_text',
        'fnc__parse_finnish_address',
        'fnc__plot_owned_label',
        'fnc__prices_transaction_floor_text',
        'fnc__prices_transaction_period_month',
        'fnc__prices_transaction_plot_type_code',
        'fnc__prices_transaction_property_type_code',
        'fnc__prices_transaction_room_category_code',
        'fnc__refresh_property_building_geom',
        'fnc__relink_property_offering_source',
        'fnc__sync_canonical_property_for_sale_listing',
        'fnc__unlink_sale_listing_prices_transaction'
      ])
    ORDER BY p.proname
  LOOP
    EXECUTE format('DROP FUNCTION IF EXISTS public.%I(%s)', fn.proname, fn.args);
  END LOOP;
END
$$;
