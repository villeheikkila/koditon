-- name: ListPricesTransactions :many
SELECT
    ht.prices_transactions_id,
    ht.prices_transactions_neighborhood,
    ht.prices_transactions_description,
    ht.prices_transactions_type,
    ht.prices_transactions_area,
    ht.prices_transactions_price,
    ht.prices_transactions_price_per_square_meter,
    ht.prices_transactions_build_year,
    ht.prices_transactions_floor,
    ht.prices_transactions_elevator,
    ht.prices_transactions_condition,
    ht.prices_transactions_plot,
    ht.prices_transactions_energy_class,
    ht.prices_transactions_first_seen_at,
    ht.prices_transactions_last_seen_at,
    ht.prices_transactions_category,
    ht.prices_neighborhoods_postal_code,
    hn.prices_neighborhoods_name
FROM public.prices_transactions AS ht
NATURAL JOIN public.prices_neighborhoods AS hn
ORDER BY ht.prices_transactions_first_seen_at;
