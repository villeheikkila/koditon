ALTER TABLE public.prices_transactions
ADD CONSTRAINT prices_transactions_unique_key
UNIQUE (
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
);
