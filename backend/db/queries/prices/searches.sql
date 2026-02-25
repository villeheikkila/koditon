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
FROM public.prices_transactions AS ht
JOIN selected_neighborhoods AS sn
    ON sn.neighborhood_id = ht.prices_neighborhood_id
LEFT JOIN public.prices_neighborhoods AS hn
    ON ht.prices_neighborhood_id = hn.prices_neighborhood_id
LEFT JOIN public.prices_postal_codes AS hp
    ON hn.prices_postal_code_id = hp.prices_postal_code_id
LEFT JOIN public.prices_cities AS hc
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
    pn.prices_neighborhood_id,
    pn.prices_neighborhood_name,
    ppc.postal_postal_code_id,
    ppc.postal_postal_code_code,
    ppc.postal_postal_code_name_fi,
    pm.postal_municipality_id,
    pm.postal_municipality_name_fi
FROM public.prices_transactions AS ht
JOIN public.prices_neighborhoods AS pn
    ON pn.prices_neighborhood_id = ht.prices_neighborhood_id
JOIN public.postal_postal_codes AS ppc
    ON ppc.postal_postal_code_id = pn.prices_neighborhood_postal_postal_code_id
JOIN public.postal_municipalities AS pm
    ON pm.postal_municipality_id = ppc.postal_municipality_id
WHERE pn.prices_neighborhood_postal_postal_code_id IS NOT NULL
  AND pm.postal_municipality_id = sqlc.arg(municipality_id)
  AND ppc.postal_postal_code_id = sqlc.arg(postal_code_id)
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
    COALESCE(ppc.postal_postal_code_name_fi, '') AS postal_area_name_fi,
    COALESCE(pm.postal_municipality_name_fi, '') AS municipality_name_fi,
    pc.prices_city_name
FROM public.prices_transactions AS ht
JOIN public.prices_neighborhoods AS pn
    ON pn.prices_neighborhood_id = ht.prices_neighborhood_id
JOIN public.prices_cities AS pc
    ON pc.prices_city_id = pn.prices_city_id
LEFT JOIN public.prices_postal_codes AS ppc_prices
    ON ppc_prices.prices_postal_code_id = pn.prices_postal_code_id
LEFT JOIN public.postal_postal_codes AS ppc
    ON ppc.postal_postal_code_id = pn.prices_neighborhood_postal_postal_code_id
LEFT JOIN public.postal_municipalities AS pm
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
    pn.prices_neighborhood_id,
    pn.prices_neighborhood_name,
    ppc.postal_postal_code_id,
    ppc.postal_postal_code_code,
    ppc.postal_postal_code_name_fi,
    pm.postal_municipality_id,
    pm.postal_municipality_name_fi
FROM public.prices_transactions AS ht
JOIN public.prices_neighborhoods AS pn
    ON pn.prices_neighborhood_id = ht.prices_neighborhood_id
JOIN public.postal_postal_codes AS ppc
    ON ppc.postal_postal_code_id = pn.prices_neighborhood_postal_postal_code_id
JOIN public.postal_municipalities AS pm
    ON pm.postal_municipality_id = ppc.postal_municipality_id
WHERE pn.prices_neighborhood_postal_postal_code_id IS NOT NULL
  AND (sqlc.narg('municipality_ids')::uuid[] IS NULL OR pm.postal_municipality_id = ANY(sqlc.narg('municipality_ids')::uuid[]))
  AND (sqlc.narg('postal_code_ids')::uuid[] IS NULL OR ppc.postal_postal_code_id = ANY(sqlc.narg('postal_code_ids')::uuid[]))
  AND (sqlc.narg('categories')::text[] IS NULL OR ht.prices_transaction_category = ANY(sqlc.narg('categories')::text[]))
  AND (sqlc.narg('types')::text[] IS NULL OR ht.prices_transaction_type = ANY(sqlc.narg('types')::text[]))
  AND (sqlc.narg('min_area')::double precision IS NULL OR ht.prices_transaction_area >= sqlc.narg('min_area')::double precision)
  AND (sqlc.narg('max_area')::double precision IS NULL OR ht.prices_transaction_area <= sqlc.narg('max_area')::double precision)
ORDER BY ht.prices_transaction_created_at DESC
LIMIT COALESCE(sqlc.narg('limit_count')::int, 100);
