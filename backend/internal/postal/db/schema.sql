CREATE TABLE public.postal_ad_areas (
    postal_ad_area_id         uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    postal_ad_area_code       text NOT NULL UNIQUE,
    postal_ad_area_name_fi    text NOT NULL,
    postal_ad_area_name_sv    text,
    postal_ad_area_created_at timestamptz NOT NULL DEFAULT now(),
    postal_ad_area_updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE public.postal_municipalities (
    postal_municipality_id                  uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    postal_municipality_code                text NOT NULL UNIQUE,
    postal_municipality_name_fi             text NOT NULL,
    postal_municipality_name_sv             text,
    postal_municipality_language_ratio_code text,
    postal_municipality_created_at          timestamptz NOT NULL DEFAULT now(),
    postal_municipality_updated_at          timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE public.postal_postal_codes (
    postal_postal_code_id              uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    postal_postal_code_date            date NOT NULL,
    postal_postal_code_code            text NOT NULL UNIQUE,
    postal_postal_code_name_fi         text NOT NULL,
    postal_postal_code_name_sv         text,
    postal_postal_code_abbr_fi         text,
    postal_postal_code_abbr_sv         text,
    postal_postal_code_neighborhood_fi text,
    postal_postal_code_valid_from      date,
    postal_postal_code_type_code       text,
    postal_ad_area_id                  uuid REFERENCES public.postal_ad_areas(postal_ad_area_id),
    postal_municipality_id             uuid REFERENCES public.postal_municipalities(postal_municipality_id),
    postal_postal_code_created_at      timestamptz NOT NULL DEFAULT now(),
    postal_postal_code_updated_at      timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_postal_postal_code_name_fi ON public.postal_postal_codes(postal_postal_code_name_fi);
CREATE INDEX idx_postal_postal_code_ad_area_id ON public.postal_postal_codes(postal_ad_area_id);
CREATE INDEX idx_postal_postal_code_municipality_id ON public.postal_postal_codes(postal_municipality_id);
CREATE INDEX idx_postal_municipality_name_fi ON public.postal_municipalities(postal_municipality_name_fi);
