-- name: ListHintatiedotTransactions :many
SELECT
    ht.hintatiedot_transactions_id,
    ht.hintatiedot_transactions_neighborhood,
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
    ht.hintatiedot_transactions_first_seen_at,
    ht.hintatiedot_transactions_last_seen_at,
    ht.hintatiedot_transactions_category,
    ht.hintatiedot_neighborhoods_postal_code,
    hn.hintatiedot_neighborhoods_name
FROM public.hintatiedot_transactions AS ht
NATURAL JOIN public.hintatiedot_neighborhoods AS hn
ORDER BY ht.hintatiedot_transactions_first_seen_at;
