CREATE OR REPLACE FUNCTION public.fnc__try_parse_bigint(value text)
RETURNS int8
LANGUAGE sql
IMMUTABLE
AS $$
WITH cleaned AS (
    SELECT NULLIF(regexp_replace(replace(COALESCE(value, ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value
)
SELECT CASE
    WHEN value IS NULL THEN NULL
    WHEN length(value) - length(replace(value, '.', '')) > 1 THEN NULL
    ELSE (value::numeric)::int8
END
FROM cleaned;
$$;
CREATE OR REPLACE FUNCTION public.fnc__try_parse_float8(value text)
RETURNS float8
LANGUAGE sql
IMMUTABLE
AS $$
WITH cleaned AS (
    SELECT NULLIF(regexp_replace(replace(COALESCE(value, ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value
)
SELECT CASE
    WHEN value IS NULL THEN NULL
    WHEN length(value) - length(replace(value, '.', '')) > 1 THEN NULL
    ELSE value::float8
END
FROM cleaned;
$$;
