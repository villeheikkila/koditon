DROP TABLE prices_neighborhoods;

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

ALTER TABLE public.prices_transactions
    ADD COLUMN prices_neighborhoods_id uuid REFERENCES public.prices_neighborhoods(prices_neighborhoods_id);
