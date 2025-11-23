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
    ht.hintatiedot_transactions_area

,
    ht.hintatiedot_transactions_price,
    ht.hintatiedot_transactions_price_per_square_meter,
    ht.hintatiedot_transactions_build_year,
    ht.hintatiedot_transactions_floor,
    ht.hintatiedot_transactions_elevator,
    ht.hintatiedot_transactions_condition,
    ht.hintatiedot_transactions_plot,
    ht.hintatiedot_transactions_energy_class,
    ht.created_at,
    ht.updated_at,
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
ORDER BY ht.created_at DESC;
