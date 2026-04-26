-- name: UpsertPricesCity :one
INSERT INTO public.prices_cities (
    prices_city_name,
    prices_city_created_at,
    prices_city_updated_at
) VALUES (sqlc.arg(name), now(), now())
ON CONFLICT (prices_city_name) DO UPDATE
SET prices_city_updated_at = now()
RETURNING *;

-- name: UpsertPricesPostalCode :one
INSERT INTO public.prices_postal_codes (
    prices_postal_code_code,
    prices_city_id,
    prices_postal_code_created_at,
    prices_postal_code_updated_at
) VALUES (sqlc.arg(code), sqlc.arg(city_id), now(), now())
ON CONFLICT (prices_postal_code_code) DO UPDATE
SET prices_city_id = EXCLUDED.prices_city_id,
    prices_postal_code_updated_at = now()
RETURNING *;

-- name: UpsertPricesPostalCodesBulk :many
INSERT INTO public.prices_postal_codes (
    prices_postal_code_code,
    prices_city_id,
    prices_postal_code_created_at,
    prices_postal_code_updated_at
)
SELECT code, sqlc.arg(city_id), now(), now()
FROM unnest(sqlc.arg(codes)::text[]) AS t(code)
ON CONFLICT (prices_postal_code_code) DO UPDATE
SET prices_city_id = EXCLUDED.prices_city_id,
    prices_postal_code_updated_at = now()
RETURNING *;

-- name: UpsertPricesNeighborhood :one
INSERT INTO public.prices_neighborhoods (
    prices_neighborhood_name,
    prices_city_id,
    prices_postal_code_id,
    prices_neighborhood_created_at,
    prices_neighborhood_updated_at
) VALUES (sqlc.arg(name), sqlc.arg(city_id), sqlc.arg(postal_code_id), now(), now())
ON CONFLICT (prices_neighborhood_name, prices_city_id) DO UPDATE
SET prices_postal_code_id = EXCLUDED.prices_postal_code_id,
    prices_neighborhood_updated_at = now()
RETURNING *;

-- name: UpsertPricesNeighborhoodsBulk :many
INSERT INTO public.prices_neighborhoods (
    prices_neighborhood_name,
    prices_city_id,
    prices_postal_code_id,
    prices_neighborhood_created_at,
    prices_neighborhood_updated_at
)
SELECT
    name,
    sqlc.arg(city_id),
    NULL::uuid,
    now(),
    now()
FROM unnest(sqlc.arg(names)::text[]) AS t(name)
ON CONFLICT (prices_neighborhood_name, prices_city_id) DO UPDATE
SET prices_postal_code_id = EXCLUDED.prices_postal_code_id,
    prices_neighborhood_updated_at = now()
RETURNING *;

-- name: UpsertPricesTransaction :one
INSERT INTO public.prices_transactions (
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
    prices_transaction_category,
    prices_transaction_period_identifier,
    prices_neighborhood_id,
    prices_transaction_created_at,
    prices_transaction_updated_at
) VALUES (
    sqlc.arg(description),
    sqlc.arg(type),
    sqlc.arg(area),
    sqlc.arg(price),
    sqlc.arg(price_per_square_meter),
    sqlc.arg(build_year),
    NULLIF(sqlc.arg(floor), ''),
    sqlc.arg(elevator),
    NULLIF(sqlc.arg(condition), ''),
    NULLIF(sqlc.arg(plot), ''),
    NULLIF(sqlc.arg(energy_class), ''),
    sqlc.arg(category),
    sqlc.arg(period_identifier),
    sqlc.arg(neighborhood_id),
    now(),
    now()
)
ON CONFLICT (
    prices_neighborhood_id,
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
    prices_transaction_category
) DO UPDATE
SET prices_transaction_updated_at = now(),
    prices_transaction_period_identifier = EXCLUDED.prices_transaction_period_identifier
WHERE prices_transactions.prices_transaction_updated_at >= now() - interval '12 months'
RETURNING *;

-- name: UpsertPricesTransactionsBulk :execrows
INSERT INTO public.prices_transactions (
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
    prices_transaction_category,
    prices_transaction_period_identifier,
    prices_neighborhood_id,
    prices_transaction_created_at,
    prices_transaction_updated_at
)
SELECT DISTINCT ON (
    neighborhood_ids,
    descriptions,
    types,
    areas,
    prices,
    price_per_square_meters,
    build_years,
    NULLIF(floors, ''),
    elevators,
    NULLIF(conditions, ''),
    NULLIF(plots, ''),
    NULLIF(energy_classes, ''),
    categories,
    period_identifiers
)
    descriptions,
    types,
    areas,
    prices,
    price_per_square_meters,
    build_years,
    NULLIF(floors, ''),
    elevators,
    NULLIF(conditions, ''),
    NULLIF(plots, ''),
    NULLIF(energy_classes, ''),
    categories,
    period_identifiers,
    neighborhood_ids,
    now(),
    now()
FROM ROWS FROM (
    unnest(CAST(sqlc.arg(descriptions) AS text[])),
    unnest(CAST(sqlc.arg(types) AS text[])),
    unnest(CAST(sqlc.arg(areas) AS double precision[])),
    unnest(CAST(sqlc.arg(prices) AS int[])),
    unnest(CAST(sqlc.arg(price_per_square_meters) AS int[])),
    unnest(CAST(sqlc.arg(build_years) AS int[])),
    unnest(CAST(sqlc.arg(floors) AS text[])),
    unnest(CAST(sqlc.arg(elevators) AS boolean[])),
    unnest(CAST(sqlc.arg(conditions) AS text[])),
    unnest(CAST(sqlc.arg(plots) AS text[])),
    unnest(CAST(sqlc.arg(energy_classes) AS text[])),
    unnest(CAST(sqlc.arg(categories) AS text[])),
    unnest(CAST(sqlc.arg(period_identifiers) AS text[])),
    unnest(CAST(sqlc.arg(neighborhood_ids) AS uuid[]))
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
    period_identifiers,
    neighborhood_ids
)
ON CONFLICT (
    prices_neighborhood_id,
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
    prices_transaction_category
) DO UPDATE
SET prices_transaction_updated_at = now(),
    prices_transaction_period_identifier = EXCLUDED.prices_transaction_period_identifier
WHERE prices_transactions.prices_transaction_updated_at >= now() - interval '12 months';

-- name: UpdateNeighborhoodPostalCode :exec
UPDATE public.prices_neighborhoods
SET prices_postal_code_id = sqlc.arg(postal_code_id),
    prices_neighborhood_updated_at = now()
WHERE prices_neighborhood_name = sqlc.arg(name)
  AND prices_city_id = sqlc.arg(city_id);

-- name: UpdateNeighborhoodPostiPostalCode :exec
UPDATE public.prices_neighborhoods
SET prices_neighborhood_postal_postal_code_id = sqlc.arg(postal_code_id),
    prices_neighborhood_updated_at = now()
WHERE prices_neighborhood_id = sqlc.arg(neighborhood_id);
