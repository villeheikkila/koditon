CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE public.prices_neighborhoods (
    prices_neighborhoods_postal_code    text          PRIMARY KEY,
    prices_neighborhoods_name           text            NOT NULL
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
    prices_transactions_floor                       text             NOT NULL,
    prices_transactions_elevator                    text             NOT NULL,
    prices_transactions_condition                   text             NOT NULL,
    prices_transactions_plot                        text             NOT NULL,
    prices_transactions_energy_class                text,
    prices_transactions_first_seen_at               timestamptz      NOT NULL,
    prices_transactions_last_seen_at                timestamptz      NOT NULL,
    prices_transactions_category                    text             NOT NULL,
    prices_neighborhoods_postal_code                text             NOT NULL
);
