CREATE SCHEMA IF NOT EXISTS origin;
DO $$
DECLARE
  table_name text;
  table_names text[] := ARRAY[
    'energy_efficiency_aliases',
    'frontdoor_ads',
    'frontdoor_building_announcements',
    'frontdoor_buildings',
    'postal_ad_areas',
    'postal_municipalities',
    'postal_postal_codes',
    'prices_cities',
    'prices_neighborhoods',
    'prices_postal_codes',
    'prices_transactions',
    'sale_listing_plot_type_aliases',
    'sale_listing_property_type_aliases',
    'sale_listing_room_category_aliases',
    'shortcut_ads',
    'shortcut_building_listings',
    'shortcut_building_rentals',
    'shortcut_buildings',
    'shortcut_tokens',
    'source_housing_companies',
    'source_listings'
  ];
BEGIN
  FOREACH table_name IN ARRAY table_names LOOP
    IF to_regclass(format('public.%I', table_name)) IS NOT NULL
       AND to_regclass(format('origin.%I', table_name)) IS NULL THEN
      EXECUTE format('ALTER TABLE public.%I SET SCHEMA origin', table_name);
    END IF;
  END LOOP;
END
$$;
DO $$
DECLARE
  fn record;
  definition text;
BEGIN
  FOR fn IN
    SELECT p.oid
    FROM pg_proc p
    JOIN pg_namespace n ON n.oid = p.pronamespace
    WHERE n.nspname = 'public'
  LOOP
    definition := pg_get_functiondef(fn.oid);
    definition := replace(definition, 'public.energy_efficiency_aliases', 'origin.energy_efficiency_aliases');
    definition := replace(definition, 'public.frontdoor_ads', 'origin.frontdoor_ads');
    definition := replace(definition, 'public.frontdoor_building_announcements', 'origin.frontdoor_building_announcements');
    definition := replace(definition, 'public.frontdoor_buildings', 'origin.frontdoor_buildings');
    definition := replace(definition, 'public.postal_ad_areas', 'origin.postal_ad_areas');
    definition := replace(definition, 'public.postal_municipalities', 'origin.postal_municipalities');
    definition := replace(definition, 'public.postal_postal_codes', 'origin.postal_postal_codes');
    definition := replace(definition, 'public.prices_cities', 'origin.prices_cities');
    definition := replace(definition, 'public.prices_neighborhoods', 'origin.prices_neighborhoods');
    definition := replace(definition, 'public.prices_postal_codes', 'origin.prices_postal_codes');
    definition := replace(definition, 'public.prices_transactions', 'origin.prices_transactions');
    definition := replace(definition, 'public.sale_listing_plot_type_aliases', 'origin.sale_listing_plot_type_aliases');
    definition := replace(definition, 'public.sale_listing_property_type_aliases', 'origin.sale_listing_property_type_aliases');
    definition := replace(definition, 'public.sale_listing_room_category_aliases', 'origin.sale_listing_room_category_aliases');
    definition := replace(definition, 'public.shortcut_ads', 'origin.shortcut_ads');
    definition := replace(definition, 'public.shortcut_building_listings', 'origin.shortcut_building_listings');
    definition := replace(definition, 'public.shortcut_building_rentals', 'origin.shortcut_building_rentals');
    definition := replace(definition, 'public.shortcut_buildings', 'origin.shortcut_buildings');
    definition := replace(definition, 'public.shortcut_tokens', 'origin.shortcut_tokens');
    definition := replace(definition, 'public.source_housing_companies', 'origin.source_housing_companies');
    definition := replace(definition, 'public.source_listings', 'origin.source_listings');
    IF definition IS DISTINCT FROM pg_get_functiondef(fn.oid) THEN
      EXECUTE definition;
    END IF;
  END LOOP;
END
$$;
DROP VIEW IF EXISTS public.view__prices_transactions;
