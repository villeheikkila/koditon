-- name: ListTransactionsByNeighborhoods :many
WITH selected_neighborhoods AS (
    SELECT UNNEST(sqlc.narg('neighborhood_ids')::uuid[]) AS neighborhood_id
)
SELECT
    ht.prices_transaction_id,
    ht.prices_transaction_description,
    ht.prices_transaction_type,
    ht.prices_transaction_area,
    ht.prices_transaction_price,
    ht.prices_transaction_price_per_square_meter,
    ht.prices_transaction_build_year,
    ht.prices_transaction_floor,
    ht.prices_transaction_elevator,
    ht.prices_transaction_condition,
    ht.prices_transaction_plot,
    ht.prices_transaction_energy_class,
    ht.prices_transaction_period_identifier,
    ht.prices_transaction_created_at,
    ht.prices_transaction_updated_at,
    ht.prices_transaction_category,
    hn.prices_neighborhood_id,
    hn.prices_neighborhood_name,
    hp.prices_postal_code_code,
    hc.prices_city_name
FROM origin.prices_transactions AS ht
JOIN selected_neighborhoods AS sn
    ON sn.neighborhood_id = ht.prices_neighborhood_id
LEFT JOIN origin.prices_neighborhoods AS hn
    ON ht.prices_neighborhood_id = hn.prices_neighborhood_id
LEFT JOIN origin.prices_postal_codes AS hp
    ON hn.prices_postal_code_id = hp.prices_postal_code_id
LEFT JOIN origin.prices_cities AS hc
    ON hn.prices_city_id = hc.prices_city_id
ORDER BY ht.prices_transaction_created_at DESC;

-- name: ListTransactionsByPostalSelection :many
SELECT
    ht.prices_transaction_id,
    ht.prices_transaction_description,
    ht.prices_transaction_type,
    ht.prices_transaction_area,
    ht.prices_transaction_price,
    ht.prices_transaction_price_per_square_meter,
    ht.prices_transaction_build_year,
    ht.prices_transaction_floor,
    ht.prices_transaction_elevator,
    ht.prices_transaction_condition,
    ht.prices_transaction_plot,
    ht.prices_transaction_energy_class,
    ht.prices_transaction_period_identifier,
    ht.prices_transaction_created_at,
    ht.prices_transaction_updated_at,
    ht.prices_transaction_category,
    EXISTS (
        SELECT 1
        FROM public.price_links AS pl
        WHERE pl.prices_transaction_id = ht.prices_transaction_id
          AND pl.link_status <> 'rejected'
    ) AS is_matched,
    (
        SELECT count(*)::integer
        FROM public.price_links AS pl
        WHERE pl.prices_transaction_id = ht.prices_transaction_id
          AND pl.target_type = 'source_listing'
          AND pl.link_status <> 'rejected'
    ) AS matched_listing_count,
    (
        SELECT count(*)::integer
        FROM public.price_links AS pl
        WHERE pl.prices_transaction_id = ht.prices_transaction_id
          AND pl.target_type = 'listing'
          AND pl.link_status <> 'rejected'
    ) AS matched_offering_count,
    pn.prices_neighborhood_id,
    pn.prices_neighborhood_name,
    COALESCE(ppc_scraped.postal_postal_code_id, ppc.postal_postal_code_id) AS postal_postal_code_id,
    COALESCE(ppc_scraped.postal_postal_code_code, ppc.postal_postal_code_code, ppc_prices.prices_postal_code_code) AS postal_postal_code_code,
    COALESCE(ppc_scraped.postal_postal_code_name_fi, ppc.postal_postal_code_name_fi, '') AS postal_postal_code_name_fi,
    COALESCE(pm_scraped.postal_municipality_id, pm.postal_municipality_id) AS postal_municipality_id,
    COALESCE(pm_scraped.postal_municipality_name_fi, pm.postal_municipality_name_fi, '') AS postal_municipality_name_fi
FROM origin.prices_transactions AS ht
JOIN origin.prices_neighborhoods AS pn
    ON pn.prices_neighborhood_id = ht.prices_neighborhood_id
LEFT JOIN origin.prices_postal_codes AS ppc_prices
    ON ppc_prices.prices_postal_code_id = pn.prices_postal_code_id
LEFT JOIN origin.postal_postal_codes AS ppc_scraped
    ON ppc_scraped.postal_postal_code_code = ppc_prices.prices_postal_code_code
LEFT JOIN origin.postal_municipalities AS pm_scraped
    ON pm_scraped.postal_municipality_id = ppc_scraped.postal_municipality_id
LEFT JOIN origin.postal_postal_codes AS ppc
    ON ppc.postal_postal_code_id = pn.prices_neighborhood_postal_postal_code_id
LEFT JOIN origin.postal_municipalities AS pm
    ON pm.postal_municipality_id = ppc.postal_municipality_id
WHERE COALESCE(pm_scraped.postal_municipality_id, pm.postal_municipality_id) = sqlc.arg(municipality_id)
  AND (ppc_scraped.postal_postal_code_id = sqlc.arg(postal_code_id) OR ppc.postal_postal_code_id = sqlc.arg(postal_code_id))
ORDER BY ht.prices_transaction_created_at DESC;

-- name: SearchTransactionsByCityAndAddress :many
SELECT
    ht.prices_transaction_id,
    ht.prices_transaction_description,
    ht.prices_transaction_type,
    ht.prices_transaction_area,
    ht.prices_transaction_price,
    ht.prices_transaction_price_per_square_meter,
    ht.prices_transaction_build_year,
    ht.prices_transaction_floor,
    ht.prices_transaction_elevator,
    ht.prices_transaction_condition,
    ht.prices_transaction_plot,
    ht.prices_transaction_energy_class,
    ht.prices_transaction_period_identifier,
    ht.prices_transaction_created_at,
    ht.prices_transaction_updated_at,
    ht.prices_transaction_category,
    pn.prices_neighborhood_name,
    COALESCE(ppc.postal_postal_code_code, ppc_prices.prices_postal_code_code) AS postal_code,
    ppc.postal_postal_code_name_fi AS postal_area_name_fi,
    pm.postal_municipality_name_fi AS municipality_name_fi,
    pc.prices_city_name
FROM origin.prices_transactions AS ht
JOIN origin.prices_neighborhoods AS pn
    ON pn.prices_neighborhood_id = ht.prices_neighborhood_id
JOIN origin.prices_cities AS pc
    ON pc.prices_city_id = pn.prices_city_id
LEFT JOIN origin.prices_postal_codes AS ppc_prices
    ON ppc_prices.prices_postal_code_id = pn.prices_postal_code_id
LEFT JOIN origin.postal_postal_codes AS ppc
    ON ppc.postal_postal_code_id = pn.prices_neighborhood_postal_postal_code_id
LEFT JOIN origin.postal_municipalities AS pm
    ON pm.postal_municipality_id = ppc.postal_municipality_id
WHERE lower(trim(pc.prices_city_name)) LIKE ('%' || lower(trim(sqlc.arg(city_name))) || '%')
  AND (
      trim(sqlc.arg(search_term)) = ''
      OR pn.prices_neighborhood_name ILIKE ('%' || sqlc.arg(search_term) || '%')
      OR COALESCE(ppc.postal_postal_code_code, '') ILIKE ('%' || sqlc.arg(search_term) || '%')
      OR COALESCE(ppc_prices.prices_postal_code_code, '') ILIKE ('%' || sqlc.arg(search_term) || '%')
      OR COALESCE(ppc.postal_postal_code_name_fi, '') ILIKE ('%' || sqlc.arg(search_term) || '%')
      OR COALESCE(pm.postal_municipality_name_fi, '') ILIKE ('%' || sqlc.arg(search_term) || '%')
      OR lower(regexp_replace(COALESCE(pn.prices_neighborhood_name, ''), '[^[:alnum:]]+', '', 'g'))
            LIKE ('%' || lower(regexp_replace(sqlc.arg(search_term), '[^[:alnum:]]+', '', 'g')) || '%')
  )
ORDER BY ht.prices_transaction_created_at DESC
LIMIT COALESCE(sqlc.narg('limit_count')::int, 200);

-- name: ListTransactionsFiltered :many
SELECT
    ht.prices_transaction_id,
    ht.prices_transaction_description,
    ht.prices_transaction_type,
    ht.prices_transaction_area,
    ht.prices_transaction_price,
    ht.prices_transaction_price_per_square_meter,
    ht.prices_transaction_build_year,
    ht.prices_transaction_floor,
    ht.prices_transaction_elevator,
    ht.prices_transaction_condition,
    ht.prices_transaction_plot,
    ht.prices_transaction_energy_class,
    ht.prices_transaction_period_identifier,
    ht.prices_transaction_created_at,
    ht.prices_transaction_updated_at,
    ht.prices_transaction_category,
    EXISTS (
        SELECT 1
        FROM public.price_links AS pl
        WHERE pl.prices_transaction_id = ht.prices_transaction_id
          AND pl.link_status <> 'rejected'
    ) AS is_matched,
    (
        SELECT count(*)::integer
        FROM public.price_links AS pl
        WHERE pl.prices_transaction_id = ht.prices_transaction_id
          AND pl.target_type = 'source_listing'
          AND pl.link_status <> 'rejected'
    ) AS matched_listing_count,
    (
        SELECT count(*)::integer
        FROM public.price_links AS pl
        WHERE pl.prices_transaction_id = ht.prices_transaction_id
          AND pl.target_type = 'listing'
          AND pl.link_status <> 'rejected'
    ) AS matched_offering_count,
    pn.prices_neighborhood_id,
    pn.prices_neighborhood_name,
    COALESCE(ppc_scraped.postal_postal_code_id, ppc.postal_postal_code_id) AS postal_postal_code_id,
    COALESCE(ppc_scraped.postal_postal_code_code, ppc.postal_postal_code_code, ppc_prices.prices_postal_code_code) AS postal_postal_code_code,
    COALESCE(ppc_scraped.postal_postal_code_name_fi, ppc.postal_postal_code_name_fi, '') AS postal_postal_code_name_fi,
    COALESCE(pm_scraped.postal_municipality_id, pm.postal_municipality_id) AS postal_municipality_id,
    COALESCE(pm_scraped.postal_municipality_name_fi, pm.postal_municipality_name_fi, '') AS postal_municipality_name_fi
FROM origin.prices_transactions AS ht
JOIN origin.prices_neighborhoods AS pn
    ON pn.prices_neighborhood_id = ht.prices_neighborhood_id
LEFT JOIN origin.prices_postal_codes AS ppc_prices
    ON ppc_prices.prices_postal_code_id = pn.prices_postal_code_id
LEFT JOIN origin.postal_postal_codes AS ppc_scraped
    ON ppc_scraped.postal_postal_code_code = ppc_prices.prices_postal_code_code
LEFT JOIN origin.postal_municipalities AS pm_scraped
    ON pm_scraped.postal_municipality_id = ppc_scraped.postal_municipality_id
LEFT JOIN origin.postal_postal_codes AS ppc
    ON ppc.postal_postal_code_id = pn.prices_neighborhood_postal_postal_code_id
LEFT JOIN origin.postal_municipalities AS pm
    ON pm.postal_municipality_id = ppc.postal_municipality_id
WHERE (sqlc.narg('municipality_ids')::uuid[] IS NULL OR COALESCE(pm_scraped.postal_municipality_id, pm.postal_municipality_id) = ANY(sqlc.narg('municipality_ids')::uuid[]))
  AND (sqlc.narg('postal_code_ids')::uuid[] IS NULL OR ppc_scraped.postal_postal_code_id = ANY(sqlc.narg('postal_code_ids')::uuid[]) OR ppc.postal_postal_code_id = ANY(sqlc.narg('postal_code_ids')::uuid[]))
  AND (sqlc.narg('categories')::text[] IS NULL OR ht.prices_transaction_category = ANY(sqlc.narg('categories')::text[]))
  AND (sqlc.narg('types')::text[] IS NULL OR ht.prices_transaction_type = ANY(sqlc.narg('types')::text[]))
  AND (sqlc.narg('min_area')::double precision IS NULL OR ht.prices_transaction_area >= sqlc.narg('min_area')::double precision)
  AND (sqlc.narg('max_area')::double precision IS NULL OR ht.prices_transaction_area <= sqlc.narg('max_area')::double precision)
ORDER BY ht.prices_transaction_created_at DESC
LIMIT COALESCE(sqlc.narg('limit_count')::int, 2147483647);

-- name: SearchTransactionsAdvanced :many
SELECT
    ht.prices_transaction_id AS transaction_id,
    ht.prices_transaction_description AS description,
    ht.prices_transaction_type AS type,
    ht.prices_transaction_category AS category,
    ht.prices_transaction_area AS area,
    ht.prices_transaction_price AS price,
    ht.prices_transaction_price_per_square_meter AS price_per_square_meter,
    ht.prices_transaction_build_year AS build_year,
    ht.prices_transaction_floor AS floor,
    ht.prices_transaction_elevator AS elevator,
    ht.prices_transaction_condition AS condition,
    ht.prices_transaction_plot AS plot,
    ht.prices_transaction_energy_class AS energy_class,
    ht.prices_transaction_period_identifier AS period_identifier,
    ht.prices_transaction_created_at AS created_at,
    ht.prices_transaction_updated_at AS updated_at,
    pn.prices_neighborhood_id AS neighborhood_id,
    pn.prices_neighborhood_name AS neighborhood,
    COALESCE(ppc_scraped.postal_postal_code_id, ppc.postal_postal_code_id) AS postal_code_id,
    COALESCE(ppc_scraped.postal_postal_code_code, ppc.postal_postal_code_code, ppc_prices.prices_postal_code_code) AS postal_code,
    COALESCE(ppc_scraped.postal_postal_code_name_fi, ppc.postal_postal_code_name_fi, '') AS postal_area,
    COALESCE(pm_scraped.postal_municipality_id, pm.postal_municipality_id) AS municipality_id,
    COALESCE(pm_scraped.postal_municipality_name_fi, pm.postal_municipality_name_fi, '') AS municipality,
    pc.prices_city_name AS city
FROM origin.prices_transactions AS ht
JOIN origin.prices_neighborhoods AS pn
    ON pn.prices_neighborhood_id = ht.prices_neighborhood_id
JOIN origin.prices_cities AS pc
    ON pc.prices_city_id = pn.prices_city_id
LEFT JOIN origin.prices_postal_codes AS ppc_prices
    ON ppc_prices.prices_postal_code_id = pn.prices_postal_code_id
LEFT JOIN origin.postal_postal_codes AS ppc_scraped
    ON ppc_scraped.postal_postal_code_code = ppc_prices.prices_postal_code_code
LEFT JOIN origin.postal_municipalities AS pm_scraped
    ON pm_scraped.postal_municipality_id = ppc_scraped.postal_municipality_id
LEFT JOIN origin.postal_postal_codes AS ppc
    ON ppc.postal_postal_code_id = pn.prices_neighborhood_postal_postal_code_id
LEFT JOIN origin.postal_municipalities AS pm
    ON pm.postal_municipality_id = ppc.postal_municipality_id
WHERE (trim(sqlc.arg(city)::text) = '' OR lower(trim(pc.prices_city_name)) LIKE ('%' || lower(trim(sqlc.arg(city)::text)) || '%'))
  AND (COALESCE(cardinality(sqlc.narg('municipality_ids')::uuid[]), 0) = 0 OR COALESCE(pm_scraped.postal_municipality_id, pm.postal_municipality_id) = ANY(sqlc.narg('municipality_ids')::uuid[]))
  AND (COALESCE(cardinality(sqlc.narg('postal_code_ids')::uuid[]), 0) = 0 OR ppc_scraped.postal_postal_code_id = ANY(sqlc.narg('postal_code_ids')::uuid[]) OR ppc.postal_postal_code_id = ANY(sqlc.narg('postal_code_ids')::uuid[]))
  AND (COALESCE(cardinality(sqlc.narg('postal_codes')::text[]), 0) = 0 OR COALESCE(ppc_scraped.postal_postal_code_code, ppc.postal_postal_code_code, ppc_prices.prices_postal_code_code) = ANY(sqlc.narg('postal_codes')::text[]))
  AND (COALESCE(cardinality(sqlc.narg('categories')::text[]), 0) = 0 OR ht.prices_transaction_category = ANY(sqlc.narg('categories')::text[]))
  AND (COALESCE(cardinality(sqlc.narg('types')::text[]), 0) = 0 OR ht.prices_transaction_type = ANY(sqlc.narg('types')::text[]))
  AND (sqlc.narg('min_price')::int IS NULL OR ht.prices_transaction_price >= sqlc.narg('min_price')::int)
  AND (sqlc.narg('max_price')::int IS NULL OR ht.prices_transaction_price <= sqlc.narg('max_price')::int)
  AND (sqlc.narg('min_area')::double precision IS NULL OR ht.prices_transaction_area >= sqlc.narg('min_area')::double precision)
  AND (sqlc.narg('max_area')::double precision IS NULL OR ht.prices_transaction_area <= sqlc.narg('max_area')::double precision)
  AND (
      trim(sqlc.arg(query)::text) = ''
      OR ht.prices_transaction_description ILIKE ('%' || sqlc.arg(query)::text || '%')
      OR pn.prices_neighborhood_name ILIKE ('%' || sqlc.arg(query)::text || '%')
      OR COALESCE(ppc.postal_postal_code_code, '') ILIKE ('%' || sqlc.arg(query)::text || '%')
      OR COALESCE(ppc_prices.prices_postal_code_code, '') ILIKE ('%' || sqlc.arg(query)::text || '%')
      OR COALESCE(ppc.postal_postal_code_name_fi, '') ILIKE ('%' || sqlc.arg(query)::text || '%')
      OR COALESCE(pm.postal_municipality_name_fi, '') ILIKE ('%' || sqlc.arg(query)::text || '%')
      OR ht.prices_transaction_category ILIKE ('%' || sqlc.arg(query)::text || '%')
      OR ht.prices_transaction_type ILIKE ('%' || sqlc.arg(query)::text || '%')
      OR lower(regexp_replace(COALESCE(ht.prices_transaction_description, ''), '[^[:alnum:]]+', '', 'g')) LIKE ('%' || sqlc.arg(normalized_query)::text || '%')
  )
ORDER BY
    CASE WHEN sqlc.arg(sort_mode)::text = 'price_asc' THEN ht.prices_transaction_price END ASC,
    CASE WHEN sqlc.arg(sort_mode)::text = 'price_desc' THEN ht.prices_transaction_price END DESC,
    CASE WHEN sqlc.arg(sort_mode)::text = 'area_asc' THEN ht.prices_transaction_area END ASC,
    CASE WHEN sqlc.arg(sort_mode)::text = 'area_desc' THEN ht.prices_transaction_area END DESC,
    CASE WHEN sqlc.arg(sort_mode)::text = 'date_asc' THEN ht.prices_transaction_created_at END ASC,
    CASE WHEN sqlc.arg(sort_mode)::text IN ('date_desc', '') THEN ht.prices_transaction_created_at END DESC,
    CASE WHEN sqlc.arg(sort_mode)::text IN ('price_asc', 'price_desc', 'area_asc', 'area_desc') THEN ht.prices_transaction_created_at END DESC,
    CASE WHEN sqlc.arg(sort_mode)::text IN ('date_asc', 'date_desc', '') THEN ht.prices_transaction_price END ASC
LIMIT sqlc.arg(limit_count)::int;

-- name: GetTransactionAdvancedByID :one
SELECT
    ht.prices_transaction_id AS transaction_id,
    ht.prices_transaction_description AS description,
    ht.prices_transaction_type AS type,
    ht.prices_transaction_category AS category,
    ht.prices_transaction_area AS area,
    ht.prices_transaction_price AS price,
    ht.prices_transaction_price_per_square_meter AS price_per_square_meter,
    ht.prices_transaction_build_year AS build_year,
    ht.prices_transaction_floor AS floor,
    ht.prices_transaction_elevator AS elevator,
    ht.prices_transaction_condition AS condition,
    ht.prices_transaction_plot AS plot,
    ht.prices_transaction_energy_class AS energy_class,
    ht.prices_transaction_period_identifier AS period_identifier,
    ht.prices_transaction_created_at AS created_at,
    ht.prices_transaction_updated_at AS updated_at,
    pn.prices_neighborhood_id AS neighborhood_id,
    pn.prices_neighborhood_name AS neighborhood,
    ppc.postal_postal_code_id AS postal_code_id,
    COALESCE(ppc.postal_postal_code_code, ppc_prices.prices_postal_code_code) AS postal_code,
    ppc.postal_postal_code_name_fi AS postal_area,
    pm.postal_municipality_id AS municipality_id,
    pm.postal_municipality_name_fi AS municipality,
    pc.prices_city_name AS city
FROM origin.prices_transactions AS ht
JOIN origin.prices_neighborhoods AS pn
    ON pn.prices_neighborhood_id = ht.prices_neighborhood_id
JOIN origin.prices_cities AS pc
    ON pc.prices_city_id = pn.prices_city_id
LEFT JOIN origin.prices_postal_codes AS ppc_prices
    ON ppc_prices.prices_postal_code_id = pn.prices_postal_code_id
LEFT JOIN origin.postal_postal_codes AS ppc
    ON ppc.postal_postal_code_id = pn.prices_neighborhood_postal_postal_code_id
LEFT JOIN origin.postal_municipalities AS pm
    ON pm.postal_municipality_id = ppc.postal_municipality_id
WHERE ht.prices_transaction_id = sqlc.arg(transaction_id)
LIMIT 1;
