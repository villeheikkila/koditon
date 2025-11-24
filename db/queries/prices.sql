-- name: ListCitiesWithNeighborhoods :many
SELECT
    hc.prices_cities_id,
    hc.prices_cities_name,
    hc.prices_cities_created_at,
    hc.prices_cities_updated_at,
    hn.prices_neighborhoods_id,
    hn.prices_neighborhoods_name,
    hn.prices_neighborhoods_created_at,
    hn.prices_neighborhoods_updated_at,
    hp.prices_postal_codes_id,
    hp.prices_postal_codes_code
FROM public.prices_cities AS hc
LEFT JOIN public.prices_neighborhoods AS hn
    ON hn.prices_neighborhoods_city_id = hc.prices_cities_id
LEFT JOIN public.prices_postal_codes AS hp
    ON hn.prices_neighborhoods_postal_code_id = hp.prices_postal_codes_id
ORDER BY hc.prices_cities_name, hn.prices_neighborhoods_name;

-- name: ListTransactionsByNeighborhoods :many
WITH selected_neighborhoods AS (
    SELECT UNNEST(sqlc.narg('neighborhood_ids')::uuid[]) AS neighborhood_id
)
SELECT
    ht.prices_transactions_id,
    ht.prices_transactions_description,
    ht.prices_transactions_type,
    ht.prices_transactions_area

,
    ht.prices_transactions_price,
    ht.prices_transactions_price_per_square_meter,
    ht.prices_transactions_build_year,
    ht.prices_transactions_floor,
    ht.prices_transactions_elevator,
    ht.prices_transactions_condition,
    ht.prices_transactions_plot,
    ht.prices_transactions_energy_class,
    ht.created_at,
    ht.updated_at,
    ht.prices_transactions_category,
    hn.prices_neighborhoods_id,
    hn.prices_neighborhoods_name,
    hp.prices_postal_codes_code,
    hc.prices_cities_name
FROM public.prices_transactions AS ht
JOIN selected_neighborhoods AS sn
    ON sn.neighborhood_id = ht.prices_neighborhoods_id
LEFT JOIN public.prices_neighborhoods AS hn
    ON ht.prices_neighborhoods_id = hn.prices_neighborhoods_id
LEFT JOIN public.prices_postal_codes AS hp
    ON hn.prices_neighborhoods_postal_code_id = hp.prices_postal_codes_id
LEFT JOIN public.prices_cities AS hc
    ON hn.prices_neighborhoods_city_id = hc.prices_cities_id
ORDER BY ht.created_at DESC;

-- name: UpsertPricesCity :one
INSERT INTO public.prices_cities (
    prices_cities_name
) VALUES ($1)
ON CONFLICT (prices_cities_name) DO UPDATE
SET prices_cities_updated_at = now()
RETURNING *;

-- name: UpsertPricesPostalCode :one
INSERT INTO public.prices_postal_codes (
    prices_postal_codes_code,
    prices_postal_codes_city_id
) VALUES ($1, $2)
ON CONFLICT (prices_postal_codes_code) DO UPDATE
SET prices_postal_codes_city_id = EXCLUDED.prices_postal_codes_city_id,
    prices_postal_codes_updated_at = now()
RETURNING *;

-- name: UpsertPricesPostalCodesBulk :many
INSERT INTO public.prices_postal_codes (
    prices_postal_codes_code,
    prices_postal_codes_city_id
)
SELECT code, sqlc.arg(city_id)
FROM unnest(sqlc.arg(codes)::text[]) AS t(code)
ON CONFLICT (prices_postal_codes_code) DO UPDATE
SET prices_postal_codes_city_id = EXCLUDED.prices_postal_codes_city_id,
    prices_postal_codes_updated_at = now()
RETURNING *;

-- name: UpsertPricesNeighborhood :one
INSERT INTO public.prices_neighborhoods (
    prices_neighborhoods_name,
    prices_neighborhoods_city_id,
    prices_neighborhoods_postal_code_id
) VALUES ($1, $2, $3)
ON CONFLICT (prices_neighborhoods_name, prices_neighborhoods_city_id) DO UPDATE
SET prices_neighborhoods_postal_code_id = EXCLUDED.prices_neighborhoods_postal_code_id,
    prices_neighborhoods_updated_at = now()
RETURNING *;

-- name: UpsertPricesNeighborhoodsBulk :many
INSERT INTO public.prices_neighborhoods (
    prices_neighborhoods_name,
    prices_neighborhoods_city_id,
    prices_neighborhoods_postal_code_id
)
SELECT
    name,
    sqlc.arg(city_id),
    NULL::uuid
FROM unnest(sqlc.arg(names)::text[]) AS t(name)
ON CONFLICT (prices_neighborhoods_name, prices_neighborhoods_city_id) DO UPDATE
SET prices_neighborhoods_postal_code_id = EXCLUDED.prices_neighborhoods_postal_code_id,
    prices_neighborhoods_updated_at = now()
RETURNING *;

-- name: UpsertPricesTransaction :one
INSERT INTO public.prices_transactions (
    prices_transactions_id,
    prices_transactions_description,
    prices_transactions_type,
    prices_transactions_area,
    prices_transactions_price,
    prices_transactions_price_per_square_meter,
    prices_transactions_build_year,
    prices_transactions_floor,
    prices_transactions_elevator,
    prices_transactions_condition,
    prices_transactions_plot,
    prices_transactions_energy_class,
    prices_transactions_category,
    prices_neighborhoods_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
)
ON CONFLICT (prices_transactions_id) DO UPDATE
SET prices_transactions_description = EXCLUDED.prices_transactions_description,
    prices_transactions_type = EXCLUDED.prices_transactions_type,
    prices_transactions_area = EXCLUDED.prices_transactions_area,
    prices_transactions_price = EXCLUDED.prices_transactions_price,
    prices_transactions_price_per_square_meter = EXCLUDED.prices_transactions_price_per_square_meter,
    prices_transactions_build_year = EXCLUDED.prices_transactions_build_year,
    prices_transactions_floor = EXCLUDED.prices_transactions_floor,
    prices_transactions_elevator = EXCLUDED.prices_transactions_elevator,
    prices_transactions_condition = EXCLUDED.prices_transactions_condition,
    prices_transactions_plot = EXCLUDED.prices_transactions_plot,
    prices_transactions_energy_class = EXCLUDED.prices_transactions_energy_class,
    prices_transactions_category = EXCLUDED.prices_transactions_category,
    prices_neighborhoods_id = EXCLUDED.prices_neighborhoods_id
RETURNING *;

-- name: UpsertPricesTransactionsBulk :execrows
INSERT INTO public.prices_transactions (
    prices_transactions_description,
    prices_transactions_type,
    prices_transactions_area,
    prices_transactions_price,
    prices_transactions_price_per_square_meter,
    prices_transactions_build_year,
    prices_transactions_floor,
    prices_transactions_elevator,
    prices_transactions_condition,
    prices_transactions_plot,
    prices_transactions_energy_class,
    prices_transactions_category,
    prices_neighborhoods_id
)
SELECT
    descriptions,
    types,
    areas,
    prices,
    price_per_square_meters,
    build_years,
    floors,
    elevators,
    conditions,
    plots,
    energy_classes,
    categories,
    neighborhood_ids
FROM unnest(
    sqlc.arg(descriptions)::text[],
    sqlc.arg(types)::text[],
    sqlc.arg(areas)::double precision[],
    sqlc.arg(prices)::int[],
    sqlc.arg(price_per_square_meters)::int[],
    sqlc.arg(build_years)::int[],
    sqlc.arg(floors)::text[],
    sqlc.arg(elevators)::boolean[],
    sqlc.arg(conditions)::text[],
    sqlc.arg(plots)::text[],
    sqlc.arg(energy_classes)::text[],
    sqlc.arg(categories)::text[],
    sqlc.arg(neighborhood_ids)::uuid[]
) AS t(
    descriptions,
    types,
    areas,
    prices,
    price_per_square_meters,
    build_years,
    floors,
    elevators,
    conditions,
    plots,
    energy_classes,
    categories,
    neighborhood_ids
)
ON CONFLICT (
    prices_neighborhoods_id,
    prices_transactions_description,
    prices_transactions_type,
    prices_transactions_area,
    prices_transactions_price,
    prices_transactions_price_per_square_meter,
    prices_transactions_build_year,
    prices_transactions_floor,
    prices_transactions_elevator,
    prices_transactions_condition,
    prices_transactions_plot,
    prices_transactions_energy_class,
    prices_transactions_category
) DO UPDATE
SET updated_at = now();
