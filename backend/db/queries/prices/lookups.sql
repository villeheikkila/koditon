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
FROM origin.prices_cities AS hc
LEFT JOIN origin.prices_neighborhoods AS hn
    ON hn.prices_city_id = hc.prices_city_id
LEFT JOIN origin.prices_postal_codes AS hp
    ON hn.prices_postal_code_id = hp.prices_postal_code_id
ORDER BY hc.prices_city_name, hn.prices_neighborhood_name;

-- name: ListPricesPostalCodesByCity :many
SELECT
    prices_postal_code_id,
    prices_postal_code_code,
    prices_city_id,
    prices_postal_code_created_at,
    prices_postal_code_updated_at
FROM origin.prices_postal_codes
WHERE prices_city_id = sqlc.arg(city_id)
ORDER BY prices_postal_code_code;

-- name: ListPricesCities :many
SELECT
    prices_city_id,
    prices_city_name,
    prices_city_created_at,
    prices_city_updated_at
FROM origin.prices_cities
ORDER BY prices_city_name;

-- name: ListAvailableMunicipalities :many
SELECT DISTINCT
    pm.postal_municipality_id,
    pm.postal_municipality_code,
    pm.postal_municipality_name_fi,
    pm.postal_municipality_name_sv
FROM origin.prices_transactions AS pt
JOIN origin.prices_neighborhoods AS pn
    ON pn.prices_neighborhood_id = pt.prices_neighborhood_id
JOIN origin.postal_postal_codes AS ppc
    ON ppc.postal_postal_code_id = pn.prices_neighborhood_postal_postal_code_id
JOIN origin.postal_municipalities AS pm
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
FROM origin.prices_transactions AS pt
JOIN origin.prices_neighborhoods AS pn
    ON pn.prices_neighborhood_id = pt.prices_neighborhood_id
JOIN origin.postal_postal_codes AS ppc
    ON ppc.postal_postal_code_id = pn.prices_neighborhood_postal_postal_code_id
JOIN origin.postal_municipalities AS pm
    ON pm.postal_municipality_id = ppc.postal_municipality_id
WHERE pn.prices_neighborhood_postal_postal_code_id IS NOT NULL
ORDER BY ppc.postal_postal_code_code;

-- name: ListDistinctCategories :many
SELECT DISTINCT prices_transaction_category AS category
FROM origin.prices_transactions
ORDER BY prices_transaction_category;

-- name: ListDistinctTypes :many
SELECT DISTINCT prices_transaction_type AS type
FROM origin.prices_transactions
ORDER BY prices_transaction_type;

-- name: ListDistinctPlots :many
SELECT DISTINCT prices_transaction_plot AS plot
FROM origin.prices_transactions
WHERE prices_transaction_plot IS NOT NULL AND prices_transaction_plot != ''
ORDER BY prices_transaction_plot;
