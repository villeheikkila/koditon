-- name: ListCitiesWithNeighborhoods :many
SELECT
    hc.prices_city_id,
    hc.prices_city_name,
    hc.prices_city_created_at,
    hc.prices_city_updated_at,
    hn.prices_neighborhood_id,
    hn.prices_neighborhood_name,
    hn.prices_neighborhood_created_at,
    hn.prices_neighborhood_updated_at,
    hp.prices_postal_code_id,
    hp.prices_postal_code_code
FROM public.prices_cities AS hc
LEFT JOIN public.prices_neighborhoods AS hn
    ON hn.prices_city_id = hc.prices_city_id
LEFT JOIN public.prices_postal_codes AS hp
    ON hn.prices_postal_code_id = hp.prices_postal_code_id
ORDER BY hc.prices_city_name, hn.prices_neighborhood_name;

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
    sqlc.arg(period_identifiers)::text[],
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

-- name: ListPricesPostalCodesByCity :many
SELECT
    prices_postal_code_id,
    prices_postal_code_code,
    prices_city_id,
    prices_postal_code_created_at,
    prices_postal_code_updated_at
FROM public.prices_postal_codes
WHERE prices_city_id = sqlc.arg(city_id)
ORDER BY prices_postal_code_code;

-- name: UpdateNeighborhoodPostalCode :exec
UPDATE public.prices_neighborhoods
SET prices_postal_code_id = sqlc.arg(postal_code_id),
    prices_neighborhood_updated_at = now()
WHERE prices_neighborhood_name = sqlc.arg(name)
  AND prices_city_id = sqlc.arg(city_id);

-- name: ListPricesCities :many
SELECT
    prices_city_id,
    prices_city_name,
    prices_city_created_at,
    prices_city_updated_at
FROM public.prices_cities
ORDER BY prices_city_name;

-- name: ListUnmatchedNeighborhoodsBatch :many
SELECT
    pn.prices_neighborhood_id,
    pn.prices_neighborhood_name,
    pn.prices_city_id,
    pc.prices_city_name,
    COUNT(*) OVER (PARTITION BY pn.prices_city_id) as unmatched_in_city
FROM public.prices_neighborhoods AS pn
LEFT JOIN public.prices_cities AS pc
    ON pn.prices_city_id = pc.prices_city_id
WHERE pn.prices_neighborhood_postal_postal_code_id IS NULL
ORDER BY pc.prices_city_name, pn.prices_neighborhood_name
LIMIT 50 OFFSET sqlc.arg(batch_offset);

-- name: GetAvailablePostalCodesForMunicipality :many
SELECT
    ppc.postal_postal_code_id,
    ppc.postal_postal_code_code,
    ppc.postal_postal_code_name_fi,
    pm.postal_municipality_name_fi
FROM public.postal_postal_codes AS ppc
JOIN public.postal_municipalities AS pm
    ON ppc.postal_municipality_id = pm.postal_municipality_id
WHERE pm.postal_municipality_name_fi = sqlc.arg(municipality_name)
ORDER BY ppc.postal_postal_code_name_fi;

-- name: UpdateNeighborhoodPostiPostalCode :exec
UPDATE public.prices_neighborhoods
SET prices_neighborhood_postal_postal_code_id = sqlc.arg(postal_code_id),
    prices_neighborhood_updated_at = now()
WHERE prices_neighborhood_id = sqlc.arg(neighborhood_id);

-- name: CountUnmatchedNeighborhoods :one
SELECT COUNT(*) as count
FROM public.prices_neighborhoods
WHERE prices_neighborhood_postal_postal_code_id IS NULL;

-- name: ListAvailableMunicipalities :many
SELECT DISTINCT
    pm.postal_municipality_id,
    pm.postal_municipality_code,
    pm.postal_municipality_name_fi,
    pm.postal_municipality_name_sv
FROM public.prices_transactions AS pt
JOIN public.prices_neighborhoods AS pn
    ON pn.prices_neighborhood_id = pt.prices_neighborhood_id
JOIN public.postal_postal_codes AS ppc
    ON ppc.postal_postal_code_id = pn.prices_neighborhood_postal_postal_code_id
JOIN public.postal_municipalities AS pm
    ON pm.postal_municipality_id = ppc.postal_municipality_id
WHERE pn.prices_neighborhood_postal_postal_code_id IS NOT NULL
ORDER BY pm.postal_municipality_name_fi;

-- name: ListAvailablePostalCodes :many
SELECT DISTINCT
    ppc.postal_postal_code_id,
    ppc.postal_postal_code_code,
    ppc.postal_postal_code_name_fi,
    ppc.postal_postal_code_name_sv,
    pm.postal_municipality_id,
    pm.postal_municipality_name_fi
FROM public.prices_transactions AS pt
JOIN public.prices_neighborhoods AS pn
    ON pn.prices_neighborhood_id = pt.prices_neighborhood_id
JOIN public.postal_postal_codes AS ppc
    ON ppc.postal_postal_code_id = pn.prices_neighborhood_postal_postal_code_id
JOIN public.postal_municipalities AS pm
    ON pm.postal_municipality_id = ppc.postal_municipality_id
WHERE pn.prices_neighborhood_postal_postal_code_id IS NOT NULL
ORDER BY ppc.postal_postal_code_code;

-- name: ListDistinctCategories :many
SELECT DISTINCT prices_transaction_category AS category
FROM public.prices_transactions
ORDER BY prices_transaction_category;

-- name: ListDistinctTypes :many
SELECT DISTINCT prices_transaction_type AS type
FROM public.prices_transactions
ORDER BY prices_transaction_type;

-- name: ListDistinctPlots :many
SELECT DISTINCT prices_transaction_plot AS plot
FROM public.prices_transactions
WHERE prices_transaction_plot IS NOT NULL AND prices_transaction_plot != ''
ORDER BY prices_transaction_plot;

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
