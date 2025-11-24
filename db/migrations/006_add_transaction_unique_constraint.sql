ALTER TABLE public.hintatiedot_transactions
ADD CONSTRAINT hintatiedot_transactions_unique_key
UNIQUE (
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
);
