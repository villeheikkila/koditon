\set legacy_schema legacy

BEGIN;

INSERT INTO origin.postal_municipalities (
  postal_municipality_code,
  postal_municipality_name_fi,
  postal_municipality_created_at,
  postal_municipality_updated_at
)
SELECT DISTINCT ON (NULLIF(trim(municipality_code), ''))
  NULLIF(trim(municipality_code), ''),
  COALESCE(NULLIF(trim(city), ''), NULLIF(trim(area), ''), NULLIF(trim(municipality_code), '')),
  now(),
  now()
FROM :"legacy_schema".postal_codes
WHERE NULLIF(trim(municipality_code), '') IS NOT NULL
ORDER BY NULLIF(trim(municipality_code), ''), NULLIF(trim(city), '') NULLS LAST, NULLIF(trim(area), '') NULLS LAST
ON CONFLICT (postal_municipality_code) DO UPDATE SET
  postal_municipality_name_fi = EXCLUDED.postal_municipality_name_fi,
  postal_municipality_updated_at = now();

INSERT INTO origin.postal_postal_codes (
  postal_postal_code_date,
  postal_postal_code_code,
  postal_postal_code_name_fi,
  postal_municipality_id,
  postal_postal_code_neighborhood_fi,
  postal_postal_code_created_at,
  postal_postal_code_updated_at
)
SELECT
  current_date,
  pc.postal_code,
  COALESCE(NULLIF(trim(pc.city), ''), pc.postal_code),
  pm.postal_municipality_id,
  NULLIF(trim(pc.area), ''),
  now(),
  now()
FROM :"legacy_schema".postal_codes pc
LEFT JOIN origin.postal_municipalities pm ON pm.postal_municipality_code = NULLIF(trim(pc.municipality_code), '')
WHERE NULLIF(trim(pc.postal_code), '') IS NOT NULL
ON CONFLICT (postal_postal_code_code) DO UPDATE SET
  postal_postal_code_name_fi = EXCLUDED.postal_postal_code_name_fi,
  postal_municipality_id = COALESCE(EXCLUDED.postal_municipality_id, origin.postal_postal_codes.postal_municipality_id),
  postal_postal_code_neighborhood_fi = COALESCE(EXCLUDED.postal_postal_code_neighborhood_fi, origin.postal_postal_codes.postal_postal_code_neighborhood_fi),
  postal_postal_code_updated_at = now();

INSERT INTO origin.prices_cities (
  prices_city_name,
  prices_city_created_at,
  prices_city_updated_at
)
SELECT DISTINCT name, now(), now()
FROM :"legacy_schema".hintatiedot_cities
WHERE NULLIF(trim(name), '') IS NOT NULL
UNION
SELECT DISTINCT city, now(), now()
FROM :"legacy_schema".hintatiedot_transactions
WHERE NULLIF(trim(city), '') IS NOT NULL
ON CONFLICT (prices_city_name) DO UPDATE SET
  prices_city_updated_at = now();

INSERT INTO origin.prices_postal_codes (
  prices_postal_code_code,
  prices_city_id,
  prices_postal_code_created_at,
  prices_postal_code_updated_at
)
SELECT DISTINCT
  pc.postal_code,
  c.prices_city_id,
  now(),
  now()
FROM :"legacy_schema".postal_codes pc
JOIN origin.prices_cities c ON c.prices_city_name = COALESCE(NULLIF(trim(pc.city), ''), pc.city)
WHERE NULLIF(trim(pc.postal_code), '') IS NOT NULL
ON CONFLICT (prices_postal_code_code) DO UPDATE SET
  prices_city_id = EXCLUDED.prices_city_id,
  prices_postal_code_updated_at = now();

INSERT INTO origin.prices_neighborhoods (
  prices_neighborhood_name,
  prices_city_id,
  prices_postal_code_id,
  prices_neighborhood_postal_postal_code_id,
  prices_neighborhood_created_at,
  prices_neighborhood_updated_at
)
SELECT DISTINCT ON (hnpc.hintatieto_neighborhood, c.prices_city_id)
  hnpc.hintatieto_neighborhood,
  c.prices_city_id,
  ppc.prices_postal_code_id,
  ppc2.postal_postal_code_id,
  now(),
  now()
FROM :"legacy_schema".hintatiedot_neighbourhood_postal_codes hnpc
JOIN :"legacy_schema".hintatiedot_transactions ht ON ht.neighborhood = hnpc.hintatieto_neighborhood
JOIN origin.prices_cities c ON c.prices_city_name = ht.city
LEFT JOIN origin.prices_postal_codes ppc ON ppc.prices_postal_code_code = hnpc.postal_code
LEFT JOIN origin.postal_postal_codes ppc2 ON ppc2.postal_postal_code_code = hnpc.postal_code
WHERE NULLIF(trim(hnpc.hintatieto_neighborhood), '') IS NOT NULL
ORDER BY hnpc.hintatieto_neighborhood, c.prices_city_id, ht.last_seen_at DESC
ON CONFLICT (prices_neighborhood_name, prices_city_id) DO UPDATE SET
  prices_postal_code_id = COALESCE(EXCLUDED.prices_postal_code_id, origin.prices_neighborhoods.prices_postal_code_id),
  prices_neighborhood_postal_postal_code_id = COALESCE(EXCLUDED.prices_neighborhood_postal_postal_code_id, origin.prices_neighborhoods.prices_neighborhood_postal_postal_code_id),
  prices_neighborhood_updated_at = now();

INSERT INTO origin.prices_neighborhoods (
  prices_neighborhood_name,
  prices_city_id,
  prices_neighborhood_created_at,
  prices_neighborhood_updated_at
)
SELECT DISTINCT
  ht.neighborhood,
  c.prices_city_id,
  now(),
  now()
FROM :"legacy_schema".hintatiedot_transactions ht
JOIN origin.prices_cities c ON c.prices_city_name = ht.city
WHERE NULLIF(trim(ht.neighborhood), '') IS NOT NULL
ON CONFLICT (prices_neighborhood_name, prices_city_id) DO UPDATE SET
  prices_neighborhood_updated_at = now();

INSERT INTO origin.prices_transactions (
  prices_transaction_id,
  prices_transaction_description,
  prices_transaction_type,
  prices_transaction_area,
  prices_transaction_price,
  prices_transaction_price_per_square_meter,
  prices_transaction_build_year,
  prices_transaction_floor,
  prices_transaction_elevator,
  prices_transaction_condition,
  prices_transaction_plot,
  prices_transaction_energy_class,
  prices_transaction_period_identifier,
  prices_transaction_created_at,
  prices_transaction_updated_at,
  prices_transaction_category,
  prices_neighborhood_id
)
SELECT
  ht.id,
  ht.description,
  ht.type,
  ht.area,
  ht.price,
  ht.price_per_square_meter,
  ht.build_year,
  NULLIF(ht.floor, ''),
  lower(ht.elevator) = ANY (ARRAY['true','t','yes','y','1','kylla','kyllä']),
  NULLIF(ht.condition, ''),
  NULLIF(ht.plot, ''),
  NULLIF(ht.energy_class, ''),
  to_char(ht.first_seen_at, 'YYYY-MM'),
  ht.first_seen_at,
  ht.last_seen_at,
  ht.category,
  pn.prices_neighborhood_id
FROM :"legacy_schema".hintatiedot_transactions ht
JOIN origin.prices_cities c ON c.prices_city_name = ht.city
LEFT JOIN origin.prices_neighborhoods pn ON pn.prices_neighborhood_name = ht.neighborhood
  AND pn.prices_city_id = c.prices_city_id
ON CONFLICT (prices_transaction_id) DO UPDATE SET
  prices_transaction_description = EXCLUDED.prices_transaction_description,
  prices_transaction_type = EXCLUDED.prices_transaction_type,
  prices_transaction_area = EXCLUDED.prices_transaction_area,
  prices_transaction_price = EXCLUDED.prices_transaction_price,
  prices_transaction_price_per_square_meter = EXCLUDED.prices_transaction_price_per_square_meter,
  prices_transaction_build_year = EXCLUDED.prices_transaction_build_year,
  prices_transaction_floor = EXCLUDED.prices_transaction_floor,
  prices_transaction_elevator = EXCLUDED.prices_transaction_elevator,
  prices_transaction_condition = EXCLUDED.prices_transaction_condition,
  prices_transaction_plot = EXCLUDED.prices_transaction_plot,
  prices_transaction_energy_class = EXCLUDED.prices_transaction_energy_class,
  prices_transaction_period_identifier = EXCLUDED.prices_transaction_period_identifier,
  prices_transaction_updated_at = EXCLUDED.prices_transaction_updated_at,
  prices_transaction_category = EXCLUDED.prices_transaction_category,
  prices_neighborhood_id = COALESCE(EXCLUDED.prices_neighborhood_id, origin.prices_transactions.prices_neighborhood_id);

INSERT INTO origin.shortcut_buildings (
  shortcut_building_id,
  shortcut_building_external_id,
  shortcut_building_building_id,
  shortcut_building_building_type,
  shortcut_building_building_subtype,
  shortcut_building_construction_year,
  shortcut_building_floor_count,
  shortcut_building_apartment_count,
  shortcut_building_heating_system,
  shortcut_building_building_material,
  shortcut_building_plot_type,
  shortcut_building_wall_structure,
  shortcut_building_heat_source,
  shortcut_building_has_elevator,
  shortcut_building_has_sauna,
  shortcut_building_latitude,
  shortcut_building_longitude,
  shortcut_building_additional_addresses,
  shortcut_building_url,
  shortcut_building_created_at,
  shortcut_building_updated_at,
  shortcut_building_address,
  shortcut_building_processed_at,
  shortcut_building_page_not_found,
  shortcut_building_frame_construction_method,
  shortcut_building_housing_company,
  shortcut_building_geom
)
SELECT
  id,
  oikotie_building_id,
  building_id,
  building_type,
  building_subtype,
  construction_year,
  floor_count,
  apartment_count,
  heating_system,
  building_material,
  plot_type,
  wall_structure,
  heat_source,
  has_elevator,
  has_sauna,
  latitude,
  longitude,
  additional_addresses,
  url,
  created_at,
  updated_at,
  address,
  processed_at,
  page_not_found,
  frame_construction_method,
  housing_company,
  geom
FROM :"legacy_schema".oikotie_buildings
ON CONFLICT (shortcut_building_id) DO UPDATE SET
  shortcut_building_external_id = EXCLUDED.shortcut_building_external_id,
  shortcut_building_building_id = EXCLUDED.shortcut_building_building_id,
  shortcut_building_building_type = EXCLUDED.shortcut_building_building_type,
  shortcut_building_building_subtype = EXCLUDED.shortcut_building_building_subtype,
  shortcut_building_construction_year = EXCLUDED.shortcut_building_construction_year,
  shortcut_building_floor_count = EXCLUDED.shortcut_building_floor_count,
  shortcut_building_apartment_count = EXCLUDED.shortcut_building_apartment_count,
  shortcut_building_heating_system = EXCLUDED.shortcut_building_heating_system,
  shortcut_building_building_material = EXCLUDED.shortcut_building_building_material,
  shortcut_building_plot_type = EXCLUDED.shortcut_building_plot_type,
  shortcut_building_wall_structure = EXCLUDED.shortcut_building_wall_structure,
  shortcut_building_heat_source = EXCLUDED.shortcut_building_heat_source,
  shortcut_building_has_elevator = EXCLUDED.shortcut_building_has_elevator,
  shortcut_building_has_sauna = EXCLUDED.shortcut_building_has_sauna,
  shortcut_building_latitude = EXCLUDED.shortcut_building_latitude,
  shortcut_building_longitude = EXCLUDED.shortcut_building_longitude,
  shortcut_building_additional_addresses = EXCLUDED.shortcut_building_additional_addresses,
  shortcut_building_url = EXCLUDED.shortcut_building_url,
  shortcut_building_updated_at = EXCLUDED.shortcut_building_updated_at,
  shortcut_building_address = EXCLUDED.shortcut_building_address,
  shortcut_building_processed_at = EXCLUDED.shortcut_building_processed_at,
  shortcut_building_page_not_found = EXCLUDED.shortcut_building_page_not_found,
  shortcut_building_frame_construction_method = EXCLUDED.shortcut_building_frame_construction_method,
  shortcut_building_housing_company = EXCLUDED.shortcut_building_housing_company,
  shortcut_building_geom = EXCLUDED.shortcut_building_geom;

INSERT INTO origin.shortcut_building_listings (
  shortcut_building_listing_id,
  shortcut_building_id,
  shortcut_building_listing_layout,
  shortcut_building_listing_size,
  shortcut_building_listing_price,
  shortcut_building_listing_price_per_sqm,
  shortcut_building_listing_deleted_at,
  shortcut_building_listing_created_at,
  shortcut_building_listing_updated_at,
  shortcut_building_listing_marketing_time,
  shortcut_building_listing_idx
)
SELECT id, building_id, layout, size, price, price_per_sqm, deleted_at, created_at, updated_at, marketing_time, idx
FROM :"legacy_schema".oikotie_building_listings
ON CONFLICT (shortcut_building_listing_id) DO UPDATE SET
  shortcut_building_id = EXCLUDED.shortcut_building_id,
  shortcut_building_listing_layout = EXCLUDED.shortcut_building_listing_layout,
  shortcut_building_listing_size = EXCLUDED.shortcut_building_listing_size,
  shortcut_building_listing_price = EXCLUDED.shortcut_building_listing_price,
  shortcut_building_listing_price_per_sqm = EXCLUDED.shortcut_building_listing_price_per_sqm,
  shortcut_building_listing_deleted_at = EXCLUDED.shortcut_building_listing_deleted_at,
  shortcut_building_listing_updated_at = EXCLUDED.shortcut_building_listing_updated_at,
  shortcut_building_listing_marketing_time = EXCLUDED.shortcut_building_listing_marketing_time,
  shortcut_building_listing_idx = EXCLUDED.shortcut_building_listing_idx;

INSERT INTO origin.shortcut_building_rentals (
  shortcut_building_rental_id,
  shortcut_building_id,
  shortcut_building_rental_layout,
  shortcut_building_rental_size,
  shortcut_building_rental_price,
  shortcut_building_rental_deleted_at,
  shortcut_building_rental_created_at,
  shortcut_building_rental_updated_at,
  shortcut_building_rental_marketing_time,
  shortcut_building_rental_idx
)
SELECT id, building_id, layout, size, price, deleted_at, created_at, updated_at, marketing_time, idx
FROM :"legacy_schema".oikotie_building_rentals
ON CONFLICT (shortcut_building_rental_id) DO UPDATE SET
  shortcut_building_id = EXCLUDED.shortcut_building_id,
  shortcut_building_rental_layout = EXCLUDED.shortcut_building_rental_layout,
  shortcut_building_rental_size = EXCLUDED.shortcut_building_rental_size,
  shortcut_building_rental_price = EXCLUDED.shortcut_building_rental_price,
  shortcut_building_rental_deleted_at = EXCLUDED.shortcut_building_rental_deleted_at,
  shortcut_building_rental_updated_at = EXCLUDED.shortcut_building_rental_updated_at,
  shortcut_building_rental_marketing_time = EXCLUDED.shortcut_building_rental_marketing_time,
  shortcut_building_rental_idx = EXCLUDED.shortcut_building_rental_idx;

INSERT INTO origin.shortcut_ads (
  shortcut_ad_id,
  shortcut_ad_url,
  shortcut_ad_type,
  shortcut_ad_first_seen_at,
  shortcut_ad_last_seen_at,
  shortcut_ad_data,
  shortcut_ad_updated_at,
  shortcut_building_id
)
SELECT
  id,
  url,
  type,
  first_seen_at,
  last_seen_at,
  data,
  updated_at,
  oikotie_building_id
FROM :"legacy_schema".oikotie_ads
ON CONFLICT (shortcut_ad_id) DO UPDATE SET
  shortcut_ad_url = EXCLUDED.shortcut_ad_url,
  shortcut_ad_type = EXCLUDED.shortcut_ad_type,
  shortcut_ad_first_seen_at = LEAST(origin.shortcut_ads.shortcut_ad_first_seen_at, EXCLUDED.shortcut_ad_first_seen_at),
  shortcut_ad_last_seen_at = GREATEST(origin.shortcut_ads.shortcut_ad_last_seen_at, EXCLUDED.shortcut_ad_last_seen_at),
  shortcut_ad_data = EXCLUDED.shortcut_ad_data,
  shortcut_ad_updated_at = EXCLUDED.shortcut_ad_updated_at,
  shortcut_building_id = EXCLUDED.shortcut_building_id;

INSERT INTO origin.frontdoor_ads (
  frontdoor_ad_id,
  frontdoor_ad_external_id,
  frontdoor_ad_url,
  frontdoor_ad_first_seen_at,
  frontdoor_ad_last_seen_at,
  frontdoor_ad_updated_at,
  frontdoor_ad_data,
  frontdoor_ad_processed_at,
  frontdoor_ad_page_not_found
)
SELECT
  id,
  external_id,
  url,
  first_seen_at,
  last_seen_at,
  updated_at,
  data,
  processed_at,
  page_not_found
FROM :"legacy_schema".etuovi_ads
ON CONFLICT (frontdoor_ad_id) DO UPDATE SET
  frontdoor_ad_external_id = EXCLUDED.frontdoor_ad_external_id,
  frontdoor_ad_url = EXCLUDED.frontdoor_ad_url,
  frontdoor_ad_first_seen_at = LEAST(origin.frontdoor_ads.frontdoor_ad_first_seen_at, EXCLUDED.frontdoor_ad_first_seen_at),
  frontdoor_ad_last_seen_at = GREATEST(origin.frontdoor_ads.frontdoor_ad_last_seen_at, EXCLUDED.frontdoor_ad_last_seen_at),
  frontdoor_ad_updated_at = EXCLUDED.frontdoor_ad_updated_at,
  frontdoor_ad_data = EXCLUDED.frontdoor_ad_data,
  frontdoor_ad_processed_at = EXCLUDED.frontdoor_ad_processed_at,
  frontdoor_ad_page_not_found = EXCLUDED.frontdoor_ad_page_not_found;

INSERT INTO origin.frontdoor_buildings (
  frontdoor_building_id,
  frontdoor_building_url,
  frontdoor_building_first_seen_at,
  frontdoor_building_last_seen_at,
  frontdoor_building_updated_at,
  frontdoor_building_company_name,
  frontdoor_building_business_id,
  frontdoor_building_apartment_count,
  frontdoor_building_floor_count,
  frontdoor_building_construction_end_year,
  frontdoor_building_build_year,
  frontdoor_building_has_elevator,
  frontdoor_building_has_sauna,
  frontdoor_building_energy_certificate_code,
  frontdoor_building_plot_holding_type,
  frontdoor_building_outer_roof_material,
  frontdoor_building_outer_roof_type,
  frontdoor_building_heating,
  frontdoor_building_heating_fuel,
  frontdoor_building_street_address,
  frontdoor_building_house_number,
  frontdoor_building_postcode,
  frontdoor_building_post_area,
  frontdoor_building_municipality,
  frontdoor_building_district,
  frontdoor_building_latitude,
  frontdoor_building_longitude,
  frontdoor_building_elevator_renovated,
  frontdoor_building_elevator_renovated_year,
  frontdoor_building_facade_renovated,
  frontdoor_building_facade_renovated_year,
  frontdoor_building_window_renovated,
  frontdoor_building_window_renovated_year,
  frontdoor_building_roof_renovated,
  frontdoor_building_roof_renovated_year,
  frontdoor_building_pipe_renovated,
  frontdoor_building_pipe_renovated_year,
  frontdoor_building_balcony_renovated,
  frontdoor_building_balcony_renovated_year,
  frontdoor_building_electricity_renovated,
  frontdoor_building_electricity_renovated_year,
  frontdoor_building_contact_phone,
  frontdoor_building_contact_office_name,
  frontdoor_building_contact_office_id,
  frontdoor_building_description,
  frontdoor_building_car_storage_description,
  frontdoor_building_other_info,
  frontdoor_building_additional_addresses,
  frontdoor_building_links,
  frontdoor_building_data,
  frontdoor_building_processed_at,
  frontdoor_building_geom,
  frontdoor_building_housing_company_id,
  frontdoor_building_housing_company_friendly_id
)
SELECT
  id,
  url,
  first_seen_at,
  last_seen_at,
  updated_at,
  company_name,
  business_id,
  apartment_count,
  floor_count,
  construction_end_year,
  build_year,
  has_elevator,
  has_sauna,
  energy_certificate_code,
  plot_holding_type,
  outer_roof_material,
  outer_roof_type,
  heating,
  heating_fuel,
  street_address,
  house_number,
  postcode,
  post_area,
  municipality,
  district,
  latitude,
  longitude,
  elevator_renovated,
  elevator_renovated_year,
  facade_renovated,
  facade_renovated_year,
  window_renovated,
  window_renovated_year,
  roof_renovated,
  roof_renovated_year,
  pipe_renovated,
  pipe_renovated_year,
  balcony_renovated,
  balcony_renovated_year,
  electricity_renovated,
  electricity_renovated_year,
  contact_phone,
  contact_office_name,
  contact_office_id,
  description,
  car_storage_description,
  other_info,
  additional_addresses,
  links,
  data,
  processed_at,
  geom,
  housing_company_id,
  housing_company_friendly_id
FROM :"legacy_schema".etuovi_buildings
ON CONFLICT (frontdoor_building_id) DO UPDATE SET
  frontdoor_building_url = EXCLUDED.frontdoor_building_url,
  frontdoor_building_first_seen_at = LEAST(origin.frontdoor_buildings.frontdoor_building_first_seen_at, EXCLUDED.frontdoor_building_first_seen_at),
  frontdoor_building_last_seen_at = GREATEST(origin.frontdoor_buildings.frontdoor_building_last_seen_at, EXCLUDED.frontdoor_building_last_seen_at),
  frontdoor_building_updated_at = EXCLUDED.frontdoor_building_updated_at,
  frontdoor_building_company_name = EXCLUDED.frontdoor_building_company_name,
  frontdoor_building_business_id = EXCLUDED.frontdoor_building_business_id,
  frontdoor_building_apartment_count = EXCLUDED.frontdoor_building_apartment_count,
  frontdoor_building_floor_count = EXCLUDED.frontdoor_building_floor_count,
  frontdoor_building_construction_end_year = EXCLUDED.frontdoor_building_construction_end_year,
  frontdoor_building_build_year = EXCLUDED.frontdoor_building_build_year,
  frontdoor_building_has_elevator = EXCLUDED.frontdoor_building_has_elevator,
  frontdoor_building_has_sauna = EXCLUDED.frontdoor_building_has_sauna,
  frontdoor_building_energy_certificate_code = EXCLUDED.frontdoor_building_energy_certificate_code,
  frontdoor_building_plot_holding_type = EXCLUDED.frontdoor_building_plot_holding_type,
  frontdoor_building_outer_roof_material = EXCLUDED.frontdoor_building_outer_roof_material,
  frontdoor_building_outer_roof_type = EXCLUDED.frontdoor_building_outer_roof_type,
  frontdoor_building_heating = EXCLUDED.frontdoor_building_heating,
  frontdoor_building_heating_fuel = EXCLUDED.frontdoor_building_heating_fuel,
  frontdoor_building_street_address = EXCLUDED.frontdoor_building_street_address,
  frontdoor_building_house_number = EXCLUDED.frontdoor_building_house_number,
  frontdoor_building_postcode = EXCLUDED.frontdoor_building_postcode,
  frontdoor_building_post_area = EXCLUDED.frontdoor_building_post_area,
  frontdoor_building_municipality = EXCLUDED.frontdoor_building_municipality,
  frontdoor_building_district = EXCLUDED.frontdoor_building_district,
  frontdoor_building_latitude = EXCLUDED.frontdoor_building_latitude,
  frontdoor_building_longitude = EXCLUDED.frontdoor_building_longitude,
  frontdoor_building_data = EXCLUDED.frontdoor_building_data,
  frontdoor_building_processed_at = EXCLUDED.frontdoor_building_processed_at,
  frontdoor_building_geom = EXCLUDED.frontdoor_building_geom,
  frontdoor_building_housing_company_id = EXCLUDED.frontdoor_building_housing_company_id,
  frontdoor_building_housing_company_friendly_id = EXCLUDED.frontdoor_building_housing_company_friendly_id;

INSERT INTO origin.frontdoor_building_announcements (
  frontdoor_building_announcement_id,
  frontdoor_building_announcement_external_id,
  frontdoor_building_announcement_friendly_id,
  frontdoor_building_announcement_unpublishing_time,
  frontdoor_building_announcement_address_line1,
  frontdoor_building_announcement_address_line2,
  frontdoor_building_announcement_location,
  frontdoor_building_announcement_search_price,
  frontdoor_building_announcement_notify_price_changed,
  frontdoor_building_announcement_property_type,
  frontdoor_building_announcement_property_subtype,
  frontdoor_building_announcement_construction_finished_year,
  frontdoor_building_announcement_main_image_uri,
  frontdoor_building_announcement_has_open_bidding,
  frontdoor_building_announcement_room_structure,
  frontdoor_building_announcement_area,
  frontdoor_building_announcement_total_area,
  frontdoor_building_announcement_price_per_square,
  frontdoor_building_announcement_days_on_market,
  frontdoor_building_announcement_new_building,
  frontdoor_building_announcement_main_image_hidden,
  frontdoor_building_announcement_is_company_announcement,
  frontdoor_building_announcement_show_bidding_indicators,
  frontdoor_building_announcement_published,
  frontdoor_building_announcement_rent_period,
  frontdoor_building_announcement_rental_unique_no,
  frontdoor_building_id,
  frontdoor_building_announcement_first_seen_at,
  frontdoor_building_announcement_last_seen_at,
  frontdoor_building_announcement_unpublishing_time_date
)
SELECT
  id,
  external_id,
  friendly_id,
  unpublishing_time,
  address_line1,
  address_line2,
  location,
  search_price,
  notify_price_changed,
  property_type,
  property_subtype,
  construction_finished_year,
  main_image_uri,
  has_open_bidding,
  room_structure,
  area,
  total_area,
  price_per_square,
  days_on_market,
  new_building,
  main_image_hidden,
  is_company_announcement,
  show_bidding_indicators,
  published,
  rent_period,
  rental_unique_no,
  etuovi_building_id,
  first_seen_at,
  last_seen_at,
  unpublishing_time_date
FROM :"legacy_schema".etuovi_building_announcements
ON CONFLICT (frontdoor_building_announcement_id) DO UPDATE SET
  frontdoor_building_announcement_external_id = EXCLUDED.frontdoor_building_announcement_external_id,
  frontdoor_building_announcement_friendly_id = EXCLUDED.frontdoor_building_announcement_friendly_id,
  frontdoor_building_announcement_unpublishing_time = EXCLUDED.frontdoor_building_announcement_unpublishing_time,
  frontdoor_building_announcement_address_line1 = EXCLUDED.frontdoor_building_announcement_address_line1,
  frontdoor_building_announcement_address_line2 = EXCLUDED.frontdoor_building_announcement_address_line2,
  frontdoor_building_announcement_location = EXCLUDED.frontdoor_building_announcement_location,
  frontdoor_building_announcement_search_price = EXCLUDED.frontdoor_building_announcement_search_price,
  frontdoor_building_announcement_room_structure = EXCLUDED.frontdoor_building_announcement_room_structure,
  frontdoor_building_announcement_area = EXCLUDED.frontdoor_building_announcement_area,
  frontdoor_building_announcement_total_area = EXCLUDED.frontdoor_building_announcement_total_area,
  frontdoor_building_announcement_price_per_square = EXCLUDED.frontdoor_building_announcement_price_per_square,
  frontdoor_building_announcement_days_on_market = EXCLUDED.frontdoor_building_announcement_days_on_market,
  frontdoor_building_announcement_published = EXCLUDED.frontdoor_building_announcement_published,
  frontdoor_building_id = EXCLUDED.frontdoor_building_id,
  frontdoor_building_announcement_last_seen_at = GREATEST(origin.frontdoor_building_announcements.frontdoor_building_announcement_last_seen_at, EXCLUDED.frontdoor_building_announcement_last_seen_at),
  frontdoor_building_announcement_unpublishing_time_date = EXCLUDED.frontdoor_building_announcement_unpublishing_time_date;

INSERT INTO origin.source_housing_companies (
  source_housing_company_id,
  provider,
  source_kind,
  native_id,
  raw_table,
  raw_id,
  url,
  first_seen_at,
  last_seen_at,
  created_at,
  updated_at
)
SELECT
  shortcut_building_id,
  'shortcut',
  'building',
  shortcut_building_external_id::text,
  'shortcut_buildings',
  shortcut_building_id::text,
  shortcut_building_url,
  shortcut_building_created_at,
  shortcut_building_updated_at,
  shortcut_building_created_at,
  shortcut_building_updated_at
FROM origin.shortcut_buildings
ON CONFLICT (source_housing_company_id) DO UPDATE SET
  native_id = EXCLUDED.native_id,
  url = EXCLUDED.url,
  first_seen_at = EXCLUDED.first_seen_at,
  last_seen_at = EXCLUDED.last_seen_at,
  updated_at = EXCLUDED.updated_at;

INSERT INTO origin.source_housing_companies (
  source_housing_company_id,
  provider,
  source_kind,
  native_id,
  raw_table,
  raw_id,
  url,
  first_seen_at,
  last_seen_at,
  created_at,
  updated_at
)
SELECT
  frontdoor_building_id,
  'frontdoor',
  'building',
  COALESCE(frontdoor_building_housing_company_friendly_id, frontdoor_building_housing_company_id::text),
  'frontdoor_buildings',
  frontdoor_building_id::text,
  frontdoor_building_url,
  frontdoor_building_first_seen_at,
  frontdoor_building_last_seen_at,
  frontdoor_building_first_seen_at,
  frontdoor_building_updated_at
FROM origin.frontdoor_buildings
ON CONFLICT (source_housing_company_id) DO UPDATE SET
  native_id = EXCLUDED.native_id,
  url = EXCLUDED.url,
  first_seen_at = EXCLUDED.first_seen_at,
  last_seen_at = EXCLUDED.last_seen_at,
  updated_at = EXCLUDED.updated_at;

INSERT INTO origin.source_listings (
  source_listing_id,
  provider,
  source_kind,
  native_id,
  canonical_source_id,
  raw_table,
  raw_id,
  url,
  normalized_version,
  first_seen_at,
  last_seen_at,
  created_at,
  updated_at
)
SELECT
  extensions.uuid_generate_v5('30ef3653-2eba-4f08-bf3d-6882bc483d54'::uuid, 'shortcut_ads:' || shortcut_ad_id::text),
  'shortcut',
  'ad',
  shortcut_ad_id::text,
  'shortcut:ad:' || shortcut_ad_id::text,
  'shortcut_ads',
  shortcut_ad_id::text,
  shortcut_ad_url,
  shortcut_ad_data_normalized_version,
  shortcut_ad_first_seen_at,
  shortcut_ad_last_seen_at,
  shortcut_ad_first_seen_at,
  shortcut_ad_updated_at
FROM origin.shortcut_ads
ON CONFLICT (provider, source_kind, native_id) DO UPDATE SET
  raw_id = EXCLUDED.raw_id,
  url = EXCLUDED.url,
  normalized_version = EXCLUDED.normalized_version,
  first_seen_at = LEAST(origin.source_listings.first_seen_at, EXCLUDED.first_seen_at),
  last_seen_at = GREATEST(origin.source_listings.last_seen_at, EXCLUDED.last_seen_at),
  updated_at = EXCLUDED.updated_at;

INSERT INTO origin.source_listings (
  source_listing_id,
  provider,
  source_kind,
  native_id,
  canonical_source_id,
  raw_table,
  raw_id,
  url,
  normalized_version,
  first_seen_at,
  last_seen_at,
  created_at,
  updated_at
)
SELECT
  frontdoor_ad_id,
  'frontdoor',
  'ad',
  frontdoor_ad_external_id,
  'frontdoor:ad:' || frontdoor_ad_external_id,
  'frontdoor_ads',
  frontdoor_ad_id::text,
  frontdoor_ad_url,
  frontdoor_ad_data_normalized_version,
  frontdoor_ad_first_seen_at,
  frontdoor_ad_last_seen_at,
  frontdoor_ad_first_seen_at,
  frontdoor_ad_updated_at
FROM origin.frontdoor_ads
ON CONFLICT (provider, source_kind, native_id) DO UPDATE SET
  raw_id = EXCLUDED.raw_id,
  url = EXCLUDED.url,
  normalized_version = EXCLUDED.normalized_version,
  first_seen_at = LEAST(origin.source_listings.first_seen_at, EXCLUDED.first_seen_at),
  last_seen_at = GREATEST(origin.source_listings.last_seen_at, EXCLUDED.last_seen_at),
  updated_at = EXCLUDED.updated_at;

INSERT INTO origin.source_listings (
  source_listing_id,
  provider,
  source_kind,
  native_id,
  canonical_source_id,
  raw_table,
  raw_id,
  url,
  normalized_version,
  first_seen_at,
  last_seen_at,
  created_at,
  updated_at
)
SELECT
  frontdoor_building_announcement_id,
  'frontdoor',
  'announcement',
  frontdoor_building_announcement_id::text,
  'frontdoor:announcement:' || frontdoor_building_announcement_id::text,
  'frontdoor_building_announcements',
  frontdoor_building_announcement_id::text,
  NULL,
  frontdoor_building_announcement_data_normalized_version,
  frontdoor_building_announcement_first_seen_at,
  frontdoor_building_announcement_last_seen_at,
  frontdoor_building_announcement_first_seen_at,
  frontdoor_building_announcement_last_seen_at
FROM origin.frontdoor_building_announcements
ON CONFLICT (provider, source_kind, native_id) DO UPDATE SET
  raw_id = EXCLUDED.raw_id,
  normalized_version = EXCLUDED.normalized_version,
  first_seen_at = LEAST(origin.source_listings.first_seen_at, EXCLUDED.first_seen_at),
  last_seen_at = GREATEST(origin.source_listings.last_seen_at, EXCLUDED.last_seen_at),
  updated_at = EXCLUDED.updated_at;

COMMIT;
