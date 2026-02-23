CREATE EXTENSION IF NOT EXISTS pg_trgm;

ALTER TABLE public.shortcut_ads
ADD COLUMN shortcut_ad_street_address text,
ADD COLUMN shortcut_ad_city text,
ADD COLUMN shortcut_ad_postal text,
ADD COLUMN shortcut_ad_price int8,
ADD COLUMN shortcut_ad_area_value float8,
ADD COLUMN shortcut_ad_address_key text,
ADD COLUMN shortcut_ad_search_text text;

ALTER TABLE public.frontdoor_ads
ADD COLUMN frontdoor_ad_street_address text,
ADD COLUMN frontdoor_ad_city text,
ADD COLUMN frontdoor_ad_postal text,
ADD COLUMN frontdoor_ad_price int8,
ADD COLUMN frontdoor_ad_area_value float8,
ADD COLUMN frontdoor_ad_address_key text,
ADD COLUMN frontdoor_ad_search_text text;

CREATE OR REPLACE FUNCTION public.fnc__normalize_match_text(value text)
RETURNS text
LANGUAGE sql
IMMUTABLE
AS $$
SELECT NULLIF(regexp_replace(lower(trim(COALESCE(value, ''))), '[^[:alnum:]]+', '', 'g'), '');
$$;

CREATE OR REPLACE FUNCTION public.fnc__normalize_postal(value text)
RETURNS text
LANGUAGE sql
IMMUTABLE
AS $$
SELECT NULLIF(regexp_replace(trim(COALESCE(value, '')), '[^0-9]+', '', 'g'), '');
$$;

CREATE OR REPLACE FUNCTION public.fnc__try_parse_bigint(value text)
RETURNS int8
LANGUAGE sql
IMMUTABLE
AS $$
SELECT CASE
    WHEN NULLIF(regexp_replace(replace(COALESCE(value, ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') IS NULL THEN NULL
    ELSE (NULLIF(regexp_replace(replace(COALESCE(value, ''), ',', '.'), '[^0-9\.-]', '', 'g'), '')::numeric)::int8
END;
$$;

CREATE OR REPLACE FUNCTION public.fnc__try_parse_float8(value text)
RETURNS float8
LANGUAGE sql
IMMUTABLE
AS $$
SELECT CASE
    WHEN NULLIF(regexp_replace(replace(COALESCE(value, ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') IS NULL THEN NULL
    ELSE NULLIF(regexp_replace(replace(COALESCE(value, ''), ',', '.'), '[^0-9\.-]', '', 'g'), '')::float8
END;
$$;

CREATE OR REPLACE FUNCTION public.fnc__shortcut_ads_set_normalized_fields()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    street text;
    city text;
    postal text;
    price int8;
    area float8;
BEGIN
    street := NULLIF(trim(COALESCE(
        NEW.shortcut_ad_data #>> '{address,street,name}',
        NEW.shortcut_ad_data #>> '{address,street}',
        NEW.shortcut_ad_data #>> '{address,formattedAddress}'
    )), '');
    city := NULLIF(trim(COALESCE(
        NEW.shortcut_ad_data #>> '{address,city,name}',
        NEW.shortcut_ad_data #>> '{address,city}'
    )), '');
    postal := NULLIF(trim(COALESCE(
        NEW.shortcut_ad_data #>> '{address,zipCode,value}',
        NEW.shortcut_ad_data #>> '{address,zipCode,name}',
        NEW.shortcut_ad_data #>> '{address,zipCode}'
    )), '');
    price := COALESCE(
        public.fnc__try_parse_bigint(NEW.shortcut_ad_data #>> '{priceData,priceSell}'),
        public.fnc__try_parse_bigint(NEW.shortcut_ad_data #>> '{priceData,price}')
    );
    area := public.fnc__try_parse_float8(NEW.shortcut_ad_data #>> '{adData,size}');
    NEW.shortcut_ad_street_address := street;
    NEW.shortcut_ad_city := city;
    NEW.shortcut_ad_postal := postal;
    NEW.shortcut_ad_price := price;
    NEW.shortcut_ad_area_value := area;
    NEW.shortcut_ad_address_key := concat_ws(
        '|',
        public.fnc__normalize_match_text(street),
        public.fnc__normalize_postal(postal),
        public.fnc__normalize_match_text(city)
    );
    NEW.shortcut_ad_search_text := trim(concat_ws(
        ' ',
        NEW.shortcut_ad_id::text,
        NEW.shortcut_ad_url,
        street,
        city,
        postal,
        NEW.shortcut_ad_data #>> '{adData,roomConfiguration}'
    ));
    RETURN NEW;
END;
$$;

CREATE OR REPLACE FUNCTION public.fnc__frontdoor_ads_set_normalized_fields()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    street text;
    city text;
    postal text;
    price int8;
    area float8;
BEGIN
    street := NULLIF(trim(COALESCE(
        NEW.frontdoor_ad_data #>> '{property,streetAddressFreeForm}',
        NEW.frontdoor_ad_data #>> '{property,address}',
        NEW.frontdoor_ad_data #>> '{property,streetNameFreeForm}'
    )), '');
    city := NULLIF(trim(COALESCE(
        NEW.frontdoor_ad_data #>> '{property,municipalityNameFreeForm}',
        NEW.frontdoor_ad_data #>> '{property,municipality}',
        NEW.frontdoor_ad_data #>> '{property,city}',
        NEW.frontdoor_ad_data #>> '{property,postCode,postArea}'
    )), '');
    postal := NULLIF(trim(COALESCE(
        NEW.frontdoor_ad_data #>> '{property,postalCode}',
        NEW.frontdoor_ad_data #>> '{property,addressPostalCode}',
        NEW.frontdoor_ad_data #>> '{property,postCode,postCode}'
    )), '');
    price := COALESCE(
        public.fnc__try_parse_bigint(NEW.frontdoor_ad_data #>> '{debfFreePrice}'),
        public.fnc__try_parse_bigint(NEW.frontdoor_ad_data #>> '{preparsed,price}')
    );
    area := COALESCE(
        public.fnc__try_parse_float8(NEW.frontdoor_ad_data #>> '{preparsed,area}'),
        public.fnc__try_parse_float8(NEW.frontdoor_ad_data #>> '{property,livingArea}')
    );
    NEW.frontdoor_ad_street_address := street;
    NEW.frontdoor_ad_city := city;
    NEW.frontdoor_ad_postal := postal;
    NEW.frontdoor_ad_price := price;
    NEW.frontdoor_ad_area_value := area;
    NEW.frontdoor_ad_address_key := concat_ws(
        '|',
        public.fnc__normalize_match_text(street),
        public.fnc__normalize_postal(postal),
        public.fnc__normalize_match_text(city)
    );
    NEW.frontdoor_ad_search_text := trim(concat_ws(
        ' ',
        NEW.frontdoor_ad_external_id,
        NEW.frontdoor_ad_url,
        street,
        city,
        postal,
        NEW.frontdoor_ad_data #>> '{residenceDetailsDTO,roomStructure}'
    ));
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS tg__shortcut_ads_set_normalized_fields ON public.shortcut_ads;
CREATE TRIGGER tg__shortcut_ads_set_normalized_fields
BEFORE INSERT OR UPDATE OF shortcut_ad_data, shortcut_ad_url
ON public.shortcut_ads
FOR EACH ROW
EXECUTE FUNCTION public.fnc__shortcut_ads_set_normalized_fields();

DROP TRIGGER IF EXISTS tg__frontdoor_ads_set_normalized_fields ON public.frontdoor_ads;
CREATE TRIGGER tg__frontdoor_ads_set_normalized_fields
BEFORE INSERT OR UPDATE OF frontdoor_ad_data, frontdoor_ad_url, frontdoor_ad_external_id
ON public.frontdoor_ads
FOR EACH ROW
EXECUTE FUNCTION public.fnc__frontdoor_ads_set_normalized_fields();

UPDATE public.shortcut_ads sa
SET
    shortcut_ad_street_address = src.street,
    shortcut_ad_city = src.city,
    shortcut_ad_postal = src.postal,
    shortcut_ad_price = src.price,
    shortcut_ad_area_value = src.area,
    shortcut_ad_address_key = concat_ws(
        '|',
        public.fnc__normalize_match_text(src.street),
        public.fnc__normalize_postal(src.postal),
        public.fnc__normalize_match_text(src.city)
    ),
    shortcut_ad_search_text = trim(concat_ws(
        ' ',
        sa.shortcut_ad_id::text,
        sa.shortcut_ad_url,
        src.street,
        src.city,
        src.postal,
        sa.shortcut_ad_data #>> '{adData,roomConfiguration}'
    ))
FROM (
    SELECT
        shortcut_ad_id,
        NULLIF(trim(COALESCE(shortcut_ad_data #>> '{address,street,name}', shortcut_ad_data #>> '{address,street}', shortcut_ad_data #>> '{address,formattedAddress}')), '') AS street,
        NULLIF(trim(COALESCE(shortcut_ad_data #>> '{address,city,name}', shortcut_ad_data #>> '{address,city}')), '') AS city,
        NULLIF(trim(COALESCE(shortcut_ad_data #>> '{address,zipCode,value}', shortcut_ad_data #>> '{address,zipCode,name}', shortcut_ad_data #>> '{address,zipCode}')), '') AS postal,
        COALESCE(public.fnc__try_parse_bigint(shortcut_ad_data #>> '{priceData,priceSell}'), public.fnc__try_parse_bigint(shortcut_ad_data #>> '{priceData,price}')) AS price,
        public.fnc__try_parse_float8(shortcut_ad_data #>> '{adData,size}') AS area
    FROM public.shortcut_ads
) src
WHERE sa.shortcut_ad_id = src.shortcut_ad_id;

UPDATE public.frontdoor_ads fa
SET
    frontdoor_ad_street_address = src.street,
    frontdoor_ad_city = src.city,
    frontdoor_ad_postal = src.postal,
    frontdoor_ad_price = src.price,
    frontdoor_ad_area_value = src.area,
    frontdoor_ad_address_key = concat_ws(
        '|',
        public.fnc__normalize_match_text(src.street),
        public.fnc__normalize_postal(src.postal),
        public.fnc__normalize_match_text(src.city)
    ),
    frontdoor_ad_search_text = trim(concat_ws(
        ' ',
        fa.frontdoor_ad_external_id,
        fa.frontdoor_ad_url,
        src.street,
        src.city,
        src.postal,
        fa.frontdoor_ad_data #>> '{residenceDetailsDTO,roomStructure}'
    ))
FROM (
    SELECT
        frontdoor_ad_id,
        NULLIF(trim(COALESCE(frontdoor_ad_data #>> '{property,streetAddressFreeForm}', frontdoor_ad_data #>> '{property,address}', frontdoor_ad_data #>> '{property,streetNameFreeForm}')), '') AS street,
        NULLIF(trim(COALESCE(frontdoor_ad_data #>> '{property,municipalityNameFreeForm}', frontdoor_ad_data #>> '{property,municipality}', frontdoor_ad_data #>> '{property,city}', frontdoor_ad_data #>> '{property,postCode,postArea}')), '') AS city,
        NULLIF(trim(COALESCE(frontdoor_ad_data #>> '{property,postalCode}', frontdoor_ad_data #>> '{property,addressPostalCode}', frontdoor_ad_data #>> '{property,postCode,postCode}')), '') AS postal,
        COALESCE(public.fnc__try_parse_bigint(frontdoor_ad_data #>> '{debfFreePrice}'), public.fnc__try_parse_bigint(frontdoor_ad_data #>> '{preparsed,price}')) AS price,
        COALESCE(public.fnc__try_parse_float8(frontdoor_ad_data #>> '{preparsed,area}'), public.fnc__try_parse_float8(frontdoor_ad_data #>> '{property,livingArea}')) AS area
    FROM public.frontdoor_ads
) src
WHERE fa.frontdoor_ad_id = src.frontdoor_ad_id;

CREATE INDEX idx_shortcut_ads_address_key ON public.shortcut_ads(shortcut_ad_address_key);
CREATE INDEX idx_shortcut_ads_postal ON public.shortcut_ads(shortcut_ad_postal);
CREATE INDEX idx_shortcut_ads_price ON public.shortcut_ads(shortcut_ad_price);
CREATE INDEX idx_shortcut_ads_area_value ON public.shortcut_ads(shortcut_ad_area_value);
CREATE INDEX idx_shortcut_ads_street_trgm ON public.shortcut_ads USING gin (lower(shortcut_ad_street_address) gin_trgm_ops);
CREATE INDEX idx_shortcut_ads_search_trgm ON public.shortcut_ads USING gin (lower(shortcut_ad_search_text) gin_trgm_ops);

CREATE INDEX idx_frontdoor_ads_address_key ON public.frontdoor_ads(frontdoor_ad_address_key);
CREATE INDEX idx_frontdoor_ads_postal ON public.frontdoor_ads(frontdoor_ad_postal);
CREATE INDEX idx_frontdoor_ads_price ON public.frontdoor_ads(frontdoor_ad_price);
CREATE INDEX idx_frontdoor_ads_area_value ON public.frontdoor_ads(frontdoor_ad_area_value);
CREATE INDEX idx_frontdoor_ads_street_trgm ON public.frontdoor_ads USING gin (lower(frontdoor_ad_street_address) gin_trgm_ops);
CREATE INDEX idx_frontdoor_ads_search_trgm ON public.frontdoor_ads USING gin (lower(frontdoor_ad_search_text) gin_trgm_ops);
