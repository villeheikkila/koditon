CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE public.prices_cities (
    prices_city_id           uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    prices_city_name         text NOT NULL UNIQUE,
    prices_city_created_at   timestamptz NOT NULL DEFAULT now(),
    prices_city_updated_at   timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE public.prices_postal_codes (
    prices_postal_code_id         uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    prices_postal_code_code       text NOT NULL UNIQUE,
    prices_city_id                uuid NOT NULL REFERENCES public.prices_cities(prices_city_id),
    prices_postal_code_created_at timestamptz NOT NULL DEFAULT now(),
    prices_postal_code_updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE public.prices_neighborhoods (
    prices_neighborhood_id           uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    prices_neighborhood_name         text NOT NULL,
    prices_city_id                   uuid NOT NULL REFERENCES public.prices_cities(prices_city_id),
    prices_postal_code_id            uuid REFERENCES public.prices_postal_codes(prices_postal_code_id),
    prices_neighborhood_created_at   timestamptz NOT NULL DEFAULT now(),
    prices_neighborhood_updated_at   timestamptz NOT NULL DEFAULT now(),
    prices_neighborhood_postal_postal_code_id uuid,
    CONSTRAINT prices_neighborhoods_name_city_unique UNIQUE (prices_neighborhood_name, prices_city_id)
);

CREATE TABLE public.prices_transactions (
    prices_transaction_id                          uuid             PRIMARY KEY DEFAULT uuid_generate_v4(),
    prices_transaction_description                 text             NOT NULL,
    prices_transaction_type                        text             NOT NULL,
    prices_transaction_area                        double precision NOT NULL,
    prices_transaction_price                       integer          NOT NULL,
    prices_transaction_price_per_square_meter      integer          NOT NULL,
    prices_transaction_build_year                  integer          NOT NULL,
    prices_transaction_floor                       text,
    prices_transaction_elevator                    boolean          NOT NULL,
    prices_transaction_condition                   text,
    prices_transaction_plot                        text,
    prices_transaction_energy_class                text,
    prices_transaction_period_identifier           text             NOT NULL,
    prices_transaction_created_at                  timestamptz      NOT NULL DEFAULT now(),
    prices_transaction_updated_at                  timestamptz      NOT NULL DEFAULT now(),
    prices_transaction_category                    text             NOT NULL,
    prices_neighborhood_id                         uuid             REFERENCES public.prices_neighborhoods(prices_neighborhood_id),
    CONSTRAINT prices_transaction_unique_key UNIQUE NULLS NOT DISTINCT (
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
    )
);

CREATE INDEX idx_prices_transaction_period_identifier
    ON public.prices_transactions(prices_transaction_period_identifier);
