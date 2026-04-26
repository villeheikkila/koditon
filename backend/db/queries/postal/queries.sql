-- Ad Areas
-- name: UpsertPostalAdAreasBulk :many
INSERT INTO public.postal_ad_areas (
    postal_ad_area_code,
    postal_ad_area_name_fi,
    postal_ad_area_name_sv,
    postal_ad_area_created_at,
    postal_ad_area_updated_at
)
SELECT
    codes,
    names_fi,
    NULLIF(names_sv, ''),
    now(),
    now()
FROM ROWS FROM (
    unnest(CAST(sqlc.arg(codes) AS text[])),
    unnest(CAST(sqlc.arg(names_fi) AS text[])),
    unnest(CAST(sqlc.arg(names_sv) AS text[]))
) AS t(codes, names_fi, names_sv)
ON CONFLICT (postal_ad_area_code) DO UPDATE
SET postal_ad_area_name_fi = EXCLUDED.postal_ad_area_name_fi,
    postal_ad_area_name_sv = EXCLUDED.postal_ad_area_name_sv,
    postal_ad_area_updated_at = now()
RETURNING *;

-- Municipalities
-- name: UpsertPostalMunicipalitiesBulk :many
INSERT INTO public.postal_municipalities (
    postal_municipality_code,
    postal_municipality_name_fi,
    postal_municipality_name_sv,
    postal_municipality_language_ratio_code,
    postal_municipality_created_at,
    postal_municipality_updated_at
)
SELECT
    codes,
    names_fi,
    NULLIF(names_sv, ''),
    NULLIF(language_ratio_codes, ''),
    now(),
    now()
FROM ROWS FROM (
    unnest(CAST(sqlc.arg(codes) AS text[])),
    unnest(CAST(sqlc.arg(names_fi) AS text[])),
    unnest(CAST(sqlc.arg(names_sv) AS text[])),
    unnest(CAST(sqlc.arg(language_ratio_codes) AS text[]))
) AS t(codes, names_fi, names_sv, language_ratio_codes)
ON CONFLICT (postal_municipality_code) DO UPDATE
SET postal_municipality_name_fi = EXCLUDED.postal_municipality_name_fi,
    postal_municipality_name_sv = EXCLUDED.postal_municipality_name_sv,
    postal_municipality_language_ratio_code = EXCLUDED.postal_municipality_language_ratio_code,
    postal_municipality_updated_at = now()
RETURNING *;

-- Postal Codes
-- name: UpsertPostalPostalCodesBulk :execrows
INSERT INTO public.postal_postal_codes (
    postal_postal_code_date,
    postal_postal_code_code,
    postal_postal_code_name_fi,
    postal_postal_code_name_sv,
    postal_postal_code_abbr_fi,
    postal_postal_code_abbr_sv,
    postal_postal_code_neighborhood_fi,
    postal_postal_code_valid_from,
    postal_postal_code_type_code,
    postal_ad_area_id,
    postal_municipality_id,
    postal_postal_code_created_at,
    postal_postal_code_updated_at
)
SELECT
    dates,
    codes,
    names_fi,
    NULLIF(names_sv, ''),
    NULLIF(abbrs_fi, ''),
    NULLIF(abbrs_sv, ''),
    NULLIF(neighborhoods_fi, ''),
    valids_from,
    NULLIF(type_codes, ''),
    ad_area_ids,
    municipality_ids,
    now(),
    now()
FROM ROWS FROM (
    unnest(CAST(sqlc.arg(dates) AS date[])),
    unnest(CAST(sqlc.arg(codes) AS text[])),
    unnest(CAST(sqlc.arg(names_fi) AS text[])),
    unnest(CAST(sqlc.arg(names_sv) AS text[])),
    unnest(CAST(sqlc.arg(abbrs_fi) AS text[])),
    unnest(CAST(sqlc.arg(abbrs_sv) AS text[])),
    unnest(CAST(sqlc.arg(neighborhoods_fi) AS text[])),
    unnest(CAST(sqlc.arg(valids_from) AS date[])),
    unnest(CAST(sqlc.arg(type_codes) AS text[])),
    unnest(CAST(sqlc.arg(ad_area_ids) AS uuid[])),
    unnest(CAST(sqlc.arg(municipality_ids) AS uuid[]))
) AS t(
    dates,
    codes,
    names_fi,
    names_sv,
    abbrs_fi,
    abbrs_sv,
    neighborhoods_fi,
    valids_from,
    type_codes,
    ad_area_ids,
    municipality_ids
)
ON CONFLICT (postal_postal_code_code) DO UPDATE
SET postal_postal_code_date = EXCLUDED.postal_postal_code_date,
    postal_postal_code_name_fi = EXCLUDED.postal_postal_code_name_fi,
    postal_postal_code_name_sv = EXCLUDED.postal_postal_code_name_sv,
    postal_postal_code_abbr_fi = EXCLUDED.postal_postal_code_abbr_fi,
    postal_postal_code_abbr_sv = EXCLUDED.postal_postal_code_abbr_sv,
    postal_postal_code_neighborhood_fi = EXCLUDED.postal_postal_code_neighborhood_fi,
    postal_postal_code_valid_from = EXCLUDED.postal_postal_code_valid_from,
    postal_postal_code_type_code = EXCLUDED.postal_postal_code_type_code,
    postal_ad_area_id = EXCLUDED.postal_ad_area_id,
    postal_municipality_id = EXCLUDED.postal_municipality_id,
    postal_postal_code_updated_at = now();

-- name: ListMunicipalitiesWithPostalCodes :many
SELECT
    pm.postal_municipality_id,
    pm.postal_municipality_code,
    pm.postal_municipality_name_fi,
    pm.postal_municipality_name_sv,
    pm.postal_municipality_created_at,
    pm.postal_municipality_updated_at,
    ppc.postal_postal_code_id,
    ppc.postal_postal_code_code,
    ppc.postal_postal_code_name_fi,
    ppc.postal_postal_code_name_sv,
    ppc.postal_postal_code_neighborhood_fi,
    ppc.postal_postal_code_created_at,
    ppc.postal_postal_code_updated_at
FROM public.postal_municipalities AS pm
JOIN public.postal_postal_codes AS ppc
    ON ppc.postal_municipality_id = pm.postal_municipality_id
WHERE ppc.postal_postal_code_type_code = '1'
ORDER BY pm.postal_municipality_name_fi, ppc.postal_postal_code_code;

-- name: ListMunicipalitiesWithPriceData :many
SELECT DISTINCT
    pm.postal_municipality_id,
    pm.postal_municipality_code,
    pm.postal_municipality_name_fi,
    pm.postal_municipality_name_sv
FROM public.postal_municipalities AS pm
JOIN public.postal_postal_codes AS ppc
    ON ppc.postal_municipality_id = pm.postal_municipality_id
JOIN public.prices_neighborhoods AS pn
    ON pn.prices_neighborhood_postal_postal_code_id = ppc.postal_postal_code_id
JOIN public.prices_transactions AS pt
    ON pt.prices_neighborhood_id = pn.prices_neighborhood_id
ORDER BY pm.postal_municipality_name_fi;

-- name: ListPostalCodesWithPriceDataForMunicipality :many
SELECT DISTINCT
    ppc.postal_postal_code_id,
    ppc.postal_postal_code_code,
    ppc.postal_postal_code_name_fi,
    ppc.postal_postal_code_name_sv
FROM public.postal_postal_codes AS ppc
JOIN public.prices_neighborhoods AS pn
    ON pn.prices_neighborhood_postal_postal_code_id = ppc.postal_postal_code_id
JOIN public.prices_transactions AS pt
    ON pt.prices_neighborhood_id = pn.prices_neighborhood_id
WHERE ppc.postal_municipality_id = sqlc.arg(municipality_id)
ORDER BY ppc.postal_postal_code_code;
