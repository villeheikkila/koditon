CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE public.prices_cities (
    prices_cities_id           uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    prices_cities_name         text NOT NULL UNIQUE,
    prices_cities_created_at   timestamptz NOT NULL DEFAULT now(),
    prices_cities_updated_at   timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE public.prices_postal_codes (
    prices_postal_codes_id         uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    prices_postal_codes_code       text NOT NULL UNIQUE,
    prices_postal_codes_city_id    uuid NOT NULL REFERENCES public.prices_cities(prices_cities_id),
    prices_postal_codes_created_at timestamptz NOT NULL DEFAULT now(),
    prices_postal_codes_updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE public.prices_neighborhoods (
    prices_neighborhoods_id           uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    prices_neighborhoods_name         text NOT NULL,
    prices_neighborhoods_city_id      uuid NOT NULL REFERENCES public.prices_cities(prices_cities_id),
    prices_neighborhoods_postal_code_id uuid REFERENCES public.prices_postal_codes(prices_postal_codes_id),
    prices_neighborhoods_created_at   timestamptz NOT NULL DEFAULT now(),
    prices_neighborhoods_updated_at   timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT prices_neighborhoods_name_city_unique UNIQUE (prices_neighborhoods_name, prices_neighborhoods_city_id)
);

CREATE TABLE public.prices_transactions (
    prices_transactions_id                          uuid             PRIMARY KEY DEFAULT uuid_generate_v4(),
    prices_transactions_neighborhood                text             NOT NULL,
    prices_transactions_description                 text             NOT NULL,
    prices_transactions_type                        text             NOT NULL,
    prices_transactions_area                        double precision NOT NULL,
    prices_transactions_price                       integer          NOT NULL,
    prices_transactions_price_per_square_meter      integer          NOT NULL,
    prices_transactions_build_year                  integer          NOT NULL,
    prices_transactions_floor                       text,
    prices_transactions_elevator                    boolean          NOT NULL,
    prices_transactions_condition                   text,
    prices_transactions_plot                        text,
    prices_transactions_energy_class                text,
    prices_transactions_period_identifier           text             NOT NULL,
    prices_transactions_created_at                  timestamptz      NOT NULL DEFAULT now(),
    prices_transactions_updated_at                  timestamptz      NOT NULL DEFAULT now(),
    prices_transactions_category                    text             NOT NULL,
    prices_neighborhoods_id                         uuid             REFERENCES public.prices_neighborhoods(prices_neighborhoods_id),
    CONSTRAINT prices_transactions_unique_key UNIQUE (
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
        prices_transactions_category,
        prices_transactions_period_identifier
    )
);

CREATE INDEX idx_prices_transactions_period_identifier
    ON public.prices_transactions(prices_transactions_period_identifier);
