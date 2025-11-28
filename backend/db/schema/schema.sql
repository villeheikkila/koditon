CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE public.hintatiedot_cities (
    hintatiedot_cities_id         uuid        PRIMARY KEY DEFAULT uuid_generate_v4(),
    hintatiedot_cities_name       text        NOT NULL UNIQUE,
    hintatiedot_cities_created_at timestamptz NOT NULL DEFAULT now(),
    hintatiedot_cities_updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE public.hintatiedot_postal_codes (
    hintatiedot_postal_codes_id         uuid        PRIMARY KEY DEFAULT uuid_generate_v4(),
    hintatiedot_postal_codes_code       text        NOT NULL UNIQUE,
    hintatiedot_postal_codes_city_id    uuid        NOT NULL REFERENCES public.hintatiedot_cities(hintatiedot_cities_id),
    hintatiedot_postal_codes_created_at timestamptz NOT NULL DEFAULT now(),
    hintatiedot_postal_codes_updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE public.hintatiedot_neighborhoods (
    hintatiedot_neighborhoods_id              uuid        PRIMARY KEY DEFAULT uuid_generate_v4(),
    hintatiedot_neighborhoods_name            text        NOT NULL,
    hintatiedot_neighborhoods_city_id         uuid        NOT NULL REFERENCES public.hintatiedot_cities(hintatiedot_cities_id),
    hintatiedot_neighborhoods_postal_code_id  uuid        REFERENCES public.hintatiedot_postal_codes(hintatiedot_postal_codes_id),
    hintatiedot_neighborhoods_created_at      timestamptz NOT NULL DEFAULT now(),
    hintatiedot_neighborhoods_updated_at      timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT hintatiedot_neighborhoods_name_city_unique UNIQUE (hintatiedot_neighborhoods_name, hintatiedot_neighborhoods_city_id)
);

CREATE TABLE public.hintatiedot_transactions (
    hintatiedot_transactions_id                     uuid             PRIMARY KEY DEFAULT uuid_generate_v4(),
    hintatiedot_transactions_description            text             NOT NULL,
    hintatiedot_transactions_type                   text             NOT NULL,
    hintatiedot_transactions_area                   double precision NOT NULL,
    hintatiedot_transactions_price                  integer          NOT NULL,
    hintatiedot_transactions_price_per_square_meter integer          NOT NULL,
    hintatiedot_transactions_build_year             integer          NOT NULL,
    hintatiedot_transactions_floor                  text,
    hintatiedot_transactions_elevator               boolean          NOT NULL,
    hintatiedot_transactions_condition              text,
    hintatiedot_transactions_plot                   text,
    hintatiedot_transactions_energy_class           text,
    hintatiedot_transactions_created_at             timestamptz      NOT NULL DEFAULT now(),
    hintatiedot_transactions_updated_at             timestamptz      NOT NULL DEFAULT now(),
    hintatiedot_transactions_category               text             NOT NULL,
    hintatiedot_neighborhoods_id                    uuid             REFERENCES public.hintatiedot_neighborhoods(hintatiedot_neighborhoods_id),
    CONSTRAINT hintatiedot_transactions_unique UNIQUE (
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
    )
);
