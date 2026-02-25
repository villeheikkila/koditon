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
