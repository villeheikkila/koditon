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
