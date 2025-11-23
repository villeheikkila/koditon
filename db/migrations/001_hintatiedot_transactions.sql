CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE public.hintatiedot_neighborhoods (
    hintatiedot_neighborhoods_postal_code    text          PRIMARY KEY,
    hintatiedot_neighborhoods_name           text            NOT NULL
);

CREATE TABLE public.hintatiedot_transactions (
    hintatiedot_transactions_id                          uuid             PRIMARY KEY DEFAULT uuid_generate_v4(),
    hintatiedot_transactions_neighborhood                text             NOT NULL,
    hintatiedot_transactions_description                 text             NOT NULL,
    hintatiedot_transactions_type                        text             NOT NULL,
    hintatiedot_transactions_area                        double precision NOT NULL,
    hintatiedot_transactions_price                       integer          NOT NULL,
    hintatiedot_transactions_price_per_square_meter      integer          NOT NULL,
    hintatiedot_transactions_build_year                  integer          NOT NULL,
    hintatiedot_transactions_floor                       text             NOT NULL,
    hintatiedot_transactions_elevator                    text             NOT NULL,
    hintatiedot_transactions_condition                   text             NOT NULL,
    hintatiedot_transactions_plot                        text             NOT NULL,
    hintatiedot_transactions_energy_class                text,
    hintatiedot_transactions_first_seen_at               timestamptz      NOT NULL,
    hintatiedot_transactions_last_seen_at                timestamptz      NOT NULL,
    hintatiedot_transactions_category                    text             NOT NULL,
    hintatiedot_neighborhoods_postal_code                text             NOT NULL
);
