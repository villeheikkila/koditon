-- name: ListCitiesWithNeighborhoods :many
SELECT
    hc.hintatiedot_cities_id,
    hc.hintatiedot_cities_name,
    hc.hintatiedot_cities_created_at,
    hc.hintatiedot_cities_updated_at,
    hn.hintatiedot_neighborhoods_id,
    hn.hintatiedot_neighborhoods_name,
    hn.hintatiedot_neighborhoods_created_at,
    hn.hintatiedot_neighborhoods_updated_at,
    hp.hintatiedot_postal_codes_id,
    hp.hintatiedot_postal_codes_code
FROM public.hintatiedot_cities AS hc
LEFT JOIN public.hintatiedot_neighborhoods AS hn
    ON hn.hintatiedot_neighborhoods_city_id = hc.hintatiedot_cities_id
LEFT JOIN public.hintatiedot_postal_codes AS hp
    ON hn.hintatiedot_neighborhoods_postal_code_id = hp.hintatiedot_postal_codes_id
ORDER BY hc.hintatiedot_cities_name, hn.hintatiedot_neighborhoods_name;

-- name: ListTransactionsByNeighborhoods :many
WITH selected_neighborhoods AS (
    SELECT UNNEST(sqlc.narg('neighborhood_ids')::uuid[]) AS neighborhood_id
)
SELECT
    ht.hintatiedot_transactions_id,
    ht.hintatiedot_transactions_description,
    ht.hintatiedot_transactions_type,
    ht.hintatiedot_transactions_area,
    ht.hintatiedot_transactions_price,
    ht.hintatiedot_transactions_price_per_square_meter,
    ht.hintatiedot_transactions_build_year,
    ht.hintatiedot_transactions_floor,
    ht.hintatiedot_transactions_elevator,
    ht.hintatiedot_transactions_condition,
    ht.hintatiedot_transactions_plot,
    ht.hintatiedot_transactions_energy_class,
    ht.hintatiedot_transactions_created_at,
    ht.hintatiedot_transactions_updated_at,
    ht.hintatiedot_transactions_category,
    hn.hintatiedot_neighborhoods_id,
    hn.hintatiedot_neighborhoods_name,
    hp.hintatiedot_postal_codes_code,
    hc.hintatiedot_cities_name
FROM public.hintatiedot_transactions AS ht
JOIN selected_neighborhoods AS sn
    ON sn.neighborhood_id = ht.hintatiedot_neighborhoods_id
LEFT JOIN public.hintatiedot_neighborhoods AS hn
    ON ht.hintatiedot_neighborhoods_id = hn.hintatiedot_neighborhoods_id
LEFT JOIN public.hintatiedot_postal_codes AS hp
    ON hn.hintatiedot_neighborhoods_postal_code_id = hp.hintatiedot_postal_codes_id
LEFT JOIN public.hintatiedot_cities AS hc
    ON hn.hintatiedot_neighborhoods_city_id = hc.hintatiedot_cities_id
ORDER BY ht.hintatiedot_transactions_created_at DESC;

-- name: UpsertHintatiedotCity :one
INSERT INTO public.hintatiedot_cities (
    hintatiedot_cities_name,
    hintatiedot_cities_created_at,
    hintatiedot_cities_updated_at
) VALUES ($1, now(), now())
ON CONFLICT (hintatiedot_cities_name) DO UPDATE
SET hintatiedot_cities_updated_at = now()
RETURNING *;

-- name: UpsertHintatiedotPostalCode :one
INSERT INTO public.hintatiedot_postal_codes (
    hintatiedot_postal_codes_code,
    hintatiedot_postal_codes_city_id,
    hintatiedot_postal_codes_created_at,
    hintatiedot_postal_codes_updated_at
) VALUES ($1, $2, now(), now())
ON CONFLICT (hintatiedot_postal_codes_code) DO UPDATE
SET hintatiedot_postal_codes_city_id = EXCLUDED.hintatiedot_postal_codes_city_id,
    hintatiedot_postal_codes_updated_at = now()
RETURNING *;

-- name: UpsertHintatiedotPostalCodesBulk :many
INSERT INTO public.hintatiedot_postal_codes (
    hintatiedot_postal_codes_code,
    hintatiedot_postal_codes_city_id,
    hintatiedot_postal_codes_created_at,
    hintatiedot_postal_codes_updated_at
)
SELECT code, sqlc.arg(city_id), now(), now()
FROM unnest(sqlc.arg(codes)::text[]) AS t(code)
ON CONFLICT (hintatiedot_postal_codes_code) DO UPDATE
SET hintatiedot_postal_codes_city_id = EXCLUDED.hintatiedot_postal_codes_city_id,
    hintatiedot_postal_codes_updated_at = now()
RETURNING *;

-- name: UpsertHintatiedotNeighborhood :one
INSERT INTO public.hintatiedot_neighborhoods (
    hintatiedot_neighborhoods_name,
    hintatiedot_neighborhoods_city_id,
    hintatiedot_neighborhoods_postal_code_id,
    hintatiedot_neighborhoods_created_at,
    hintatiedot_neighborhoods_updated_at
) VALUES ($1, $2, $3, now(), now())
ON CONFLICT (hintatiedot_neighborhoods_name, hintatiedot_neighborhoods_city_id) DO UPDATE
SET hintatiedot_neighborhoods_postal_code_id = EXCLUDED.hintatiedot_neighborhoods_postal_code_id,
    hintatiedot_neighborhoods_updated_at = now()
RETURNING *;

-- name: UpsertHintatiedotNeighborhoodsBulk :many
INSERT INTO public.hintatiedot_neighborhoods (
    hintatiedot_neighborhoods_name,
    hintatiedot_neighborhoods_city_id,
    hintatiedot_neighborhoods_postal_code_id,
    hintatiedot_neighborhoods_created_at,
    hintatiedot_neighborhoods_updated_at
)
SELECT
    name,
    sqlc.arg(city_id),
    NULL::uuid,
    now(),
    now()
FROM unnest(sqlc.arg(names)::text[]) AS t(name)
ON CONFLICT (hintatiedot_neighborhoods_name, hintatiedot_neighborhoods_city_id) DO UPDATE
SET hintatiedot_neighborhoods_postal_code_id = EXCLUDED.hintatiedot_neighborhoods_postal_code_id,
    hintatiedot_neighborhoods_updated_at = now()
RETURNING *;

-- name: UpsertHintatiedotTransaction :one
INSERT INTO public.hintatiedot_transactions (
    hintatiedot_transactions_id,
    hintatiedot_transactions_description,
    hintatiedot_transactions_type,
    hintatiedot_transactions_area,
    hintatiedot_transactions_price,
    hintatiedot_transactions_price_per_square_meter,
    hintatiedot_transactions_build_year,
    hintatiedot_transactions_floor,
    hintatiedot_transactions_elevator,
    hintatiedot_transactions_condition,
    hintatiedot_transactions_plot,
    hintatiedot_transactions_energy_class,
    hintatiedot_transactions_category,
    hintatiedot_neighborhoods_id,
    hintatiedot_transactions_created_at,
    hintatiedot_transactions_updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, now(), now()
)
ON CONFLICT (hintatiedot_transactions_id) DO UPDATE
SET hintatiedot_transactions_description = EXCLUDED.hintatiedot_transactions_description,
    hintatiedot_transactions_type = EXCLUDED.hintatiedot_transactions_type,
    hintatiedot_transactions_area = EXCLUDED.hintatiedot_transactions_area,
    hintatiedot_transactions_price = EXCLUDED.hintatiedot_transactions_price,
    hintatiedot_transactions_price_per_square_meter = EXCLUDED.hintatiedot_transactions_price_per_square_meter,
    hintatiedot_transactions_build_year = EXCLUDED.hintatiedot_transactions_build_year,
    hintatiedot_transactions_floor = EXCLUDED.hintatiedot_transactions_floor,
    hintatiedot_transactions_elevator = EXCLUDED.hintatiedot_transactions_elevator,
    hintatiedot_transactions_condition = EXCLUDED.hintatiedot_transactions_condition,
    hintatiedot_transactions_plot = EXCLUDED.hintatiedot_transactions_plot,
    hintatiedot_transactions_energy_class = EXCLUDED.hintatiedot_transactions_energy_class,
    hintatiedot_transactions_category = EXCLUDED.hintatiedot_transactions_category,
    hintatiedot_neighborhoods_id = EXCLUDED.hintatiedot_neighborhoods_id,
    hintatiedot_transactions_updated_at = now()
RETURNING *;

-- name: UpsertHintatiedotTransactionsBulk :execrows
INSERT INTO public.hintatiedot_transactions (
    hintatiedot_transactions_description,
    hintatiedot_transactions_type,
    hintatiedot_transactions_area,
    hintatiedot_transactions_price,
    hintatiedot_transactions_price_per_square_meter,
    hintatiedot_transactions_build_year,
    hintatiedot_transactions_floor,
    hintatiedot_transactions_elevator,
    hintatiedot_transactions_condition,
    hintatiedot_transactions_plot,
    hintatiedot_transactions_energy_class,
    hintatiedot_transactions_category,
    hintatiedot_neighborhoods_id,
    hintatiedot_transactions_created_at,
    hintatiedot_transactions_updated_at
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
    neighborhood_ids,
    now(),
    now()
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
    hintatiedot_neighborhoods_id,
    hintatiedot_transactions_description,
    hintatiedot_transactions_type,
    hintatiedot_transactions_area,
    hintatiedot_transactions_price,
    hintatiedot_transactions_price_per_square_meter,
    hintatiedot_transactions_build_year,
    hintatiedot_transactions_floor,
    hintatiedot_transactions_elevator,
    hintatiedot_transactions_condition,
    hintatiedot_transactions_plot,
    hintatiedot_transactions_energy_class,
    hintatiedot_transactions_category
) DO UPDATE
SET hintatiedot_transactions_updated_at = now();
