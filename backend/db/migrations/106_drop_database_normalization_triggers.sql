DO $$
BEGIN
    IF to_regclass('public.frontdoor_ads') IS NOT NULL THEN
        DROP TRIGGER IF EXISTS tg__frontdoor_ads_set_normalized_fields ON public.frontdoor_ads;
        DROP TRIGGER IF EXISTS trg__sync_sale_listing_from_frontdoor_ad ON public.frontdoor_ads;
        DROP TRIGGER IF EXISTS trg__delete_sale_listing_from_frontdoor_ad ON public.frontdoor_ads;
        DROP TRIGGER IF EXISTS tg__frontdoor_ads_link_postal_code ON public.frontdoor_ads;
    END IF;
    IF to_regclass('public.shortcut_ads') IS NOT NULL THEN
        DROP TRIGGER IF EXISTS tg__shortcut_ads_set_normalized_fields ON public.shortcut_ads;
        DROP TRIGGER IF EXISTS trg__sync_sale_listing_from_shortcut_ad ON public.shortcut_ads;
        DROP TRIGGER IF EXISTS trg__delete_sale_listing_from_shortcut_ad ON public.shortcut_ads;
    END IF;
    IF to_regclass('public.frontdoor_building_announcements') IS NOT NULL THEN
        DROP TRIGGER IF EXISTS trg__sync_sale_listing_from_frontdoor_announcement ON public.frontdoor_building_announcements;
        DROP TRIGGER IF EXISTS trg__delete_sale_listing_from_frontdoor_announcement ON public.frontdoor_building_announcements;
    END IF;
    IF to_regclass('public.frontdoor_buildings') IS NOT NULL THEN
        DROP TRIGGER IF EXISTS trg__refresh_sale_listings_from_frontdoor_building ON public.frontdoor_buildings;
    END IF;
    IF to_regclass('public.shortcut_buildings') IS NOT NULL THEN
        DROP TRIGGER IF EXISTS trg__refresh_sale_listings_from_shortcut_building ON public.shortcut_buildings;
    END IF;
    IF to_regclass('public.sale_listings') IS NOT NULL THEN
        DROP TRIGGER IF EXISTS trg__sale_listings_set_address_fields ON public.sale_listings;
        DROP TRIGGER IF EXISTS trg__sale_listings_set_transaction_match_fields ON public.sale_listings;
        DROP TRIGGER IF EXISTS trg__sync_canonical_property_for_sale_listing ON public.sale_listings;
    END IF;
    IF to_regclass('public.prices_transactions') IS NOT NULL THEN
        DROP TRIGGER IF EXISTS trg__prices_transactions_set_plot_owned ON public.prices_transactions;
        DROP TRIGGER IF EXISTS trg__sync_sale_listing_from_prices_transaction ON public.prices_transactions;
    END IF;
    IF to_regclass('public.prices_neighborhoods') IS NOT NULL THEN
        DROP TRIGGER IF EXISTS trg__refresh_sale_listings_from_prices_neighborhood ON public.prices_neighborhoods;
    END IF;
    IF to_regclass('public.property_offering_sources') IS NOT NULL THEN
        DROP TRIGGER IF EXISTS trg_refresh_property_building_geom_for_source ON public.property_offering_sources;
        DROP TRIGGER IF EXISTS trg_refresh_housing_company_geom_for_source ON public.property_offering_sources;
        DROP TRIGGER IF EXISTS trg_property_offering_source_merge ON public.property_offering_sources;
    END IF;
    IF to_regclass('public.property_source_offerings') IS NOT NULL THEN
        DROP TRIGGER IF EXISTS trg__sale_listings_set_address_fields ON public.property_source_offerings;
        DROP TRIGGER IF EXISTS trg__sale_listings_set_transaction_match_fields ON public.property_source_offerings;
        DROP TRIGGER IF EXISTS trg__sync_canonical_property_for_sale_listing ON public.property_source_offerings;
        DROP TRIGGER IF EXISTS trg__sync_property_house_for_sale_listing ON public.property_source_offerings;
    END IF;
END $$;

DROP FUNCTION IF EXISTS public.fnc__frontdoor_ads_set_normalized_fields();
DROP FUNCTION IF EXISTS public.fnc__shortcut_ads_set_normalized_fields();
DROP FUNCTION IF EXISTS public.fnc__sync_sale_listing_from_frontdoor_ad();
DROP FUNCTION IF EXISTS public.fnc__sync_sale_listing_from_shortcut_ad();
DROP FUNCTION IF EXISTS public.fnc__sync_sale_listing_from_frontdoor_announcement();
DROP FUNCTION IF EXISTS public.fnc__refresh_sale_listings_from_frontdoor_building();
DROP FUNCTION IF EXISTS public.fnc__refresh_sale_listings_from_shortcut_building();
DROP FUNCTION IF EXISTS public.fnc__sale_listings_set_address_fields();
DROP FUNCTION IF EXISTS public.fnc__sale_listings_set_transaction_match_fields();
DROP FUNCTION IF EXISTS public.fnc__sync_canonical_property_for_sale_listing_trigger();
DROP FUNCTION IF EXISTS public.fnc__prices_transactions_set_plot_owned();
DROP FUNCTION IF EXISTS public.fnc__refresh_property_building_geom_for_source_trigger();
DROP FUNCTION IF EXISTS public.fnc__refresh_housing_company_geom_for_source_trigger();
DROP FUNCTION IF EXISTS public.fnc__property_offering_source_merge_trigger();
DROP FUNCTION IF EXISTS public.fnc__sync_property_house_for_sale_listing_trigger();
DROP FUNCTION IF EXISTS public.fnc__sync_sale_listing_from_prices_transaction();
DROP FUNCTION IF EXISTS public.fnc__refresh_sale_listings_from_prices_neighborhood();
DROP FUNCTION IF EXISTS public.fnc__link_frontdoor_ads_postal_code();
