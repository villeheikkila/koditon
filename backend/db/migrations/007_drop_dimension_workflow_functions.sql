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
        'fnc__clear_listing_dimension_targets_dirty',
        'fnc__clear_property_dimension_target_dirty',
        'fnc__project_dimension_profile_for_target',
        'fnc__project_listing_provider_dimension_claims',
        'fnc__rebuild_listing_dimension_layer',
        'fnc__resolve_dimension_target',
        'fnc__resolve_dimension_values_for_target'
      ])
    ORDER BY p.proname
  LOOP
    EXECUTE format('DROP FUNCTION IF EXISTS public.%I(%s)', fn.proname, fn.args);
  END LOOP;
END
$$;
