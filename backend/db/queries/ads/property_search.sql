-- name: SearchSaleListings :many
SELECT
    doc.source,
    doc.kind,
    doc.native_id,
    doc.canonical_id,
    doc.listing_id::text AS public_id,
    doc.url,
    doc.headline,
    doc.address,
    doc.city,
    doc.postal,
    doc.asking_price AS price,
    doc.area_m2 AS area,
    doc.room_layout,
    doc.price_per_m2,
    doc.debt_free_price,
    doc.debt_share_amount,
    doc.rooms_count,
    doc.floor_level,
    doc.total_floors,
    doc.build_year,
    doc.condition,
    doc.energy_class,
    doc.energy_efficiency_label,
    doc.last_seen_at::text AS last_seen_at,
    doc.published_at::text AS published_at,
    doc.address AS building_key_address,
    doc.source_providers
FROM public.listing_search_documents doc
WHERE doc.listing_status = 'active'
  AND (sqlc.arg('source') = 'all' OR doc.source_providers @> ARRAY[sqlc.arg('source')::text])
  AND (sqlc.arg('kind') = 'all' OR doc.source_kinds @> ARRAY[sqlc.arg('kind')::text])
  AND (sqlc.narg('query_text')::text IS NULL OR trim(sqlc.narg('query_text')::text) = '' OR lower(doc.search_text) LIKE ('%' || lower(trim(sqlc.narg('query_text')::text)) || '%'))
  AND (sqlc.narg('city')::text IS NULL OR trim(sqlc.narg('city')::text) = '' OR lower(COALESCE(doc.city, '')) LIKE ('%' || lower(trim(sqlc.narg('city')::text)) || '%'))
  AND (sqlc.narg('postal')::text IS NULL OR trim(sqlc.narg('postal')::text) = '' OR lower(COALESCE(doc.postal, '')) LIKE ('%' || lower(trim(sqlc.narg('postal')::text)) || '%'))
  AND (sqlc.narg('min_price')::bigint IS NULL OR doc.asking_price >= sqlc.narg('min_price')::bigint)
  AND (sqlc.narg('max_price')::bigint IS NULL OR doc.asking_price <= sqlc.narg('max_price')::bigint)
  AND (sqlc.narg('min_area')::float8 IS NULL OR doc.area_m2 >= sqlc.narg('min_area')::float8)
  AND (sqlc.narg('max_area')::float8 IS NULL OR doc.area_m2 <= sqlc.narg('max_area')::float8)
  AND (sqlc.narg('published_after')::timestamptz IS NULL OR doc.published_at >= sqlc.narg('published_after')::timestamptz)
  AND (sqlc.narg('published_before')::timestamptz IS NULL OR doc.published_at <= sqlc.narg('published_before')::timestamptz)
  AND (sqlc.narg('min_price_per_m2')::float8 IS NULL OR doc.price_per_m2 >= sqlc.narg('min_price_per_m2')::float8)
  AND (sqlc.narg('max_price_per_m2')::float8 IS NULL OR doc.price_per_m2 <= sqlc.narg('max_price_per_m2')::float8)
  AND (sqlc.narg('rooms')::int4 IS NULL OR doc.rooms_count = sqlc.narg('rooms')::int4)
  AND (sqlc.narg('floor')::int4 IS NULL OR doc.floor_level = sqlc.narg('floor')::int4)
  AND (sqlc.narg('min_build_year')::int4 IS NULL OR doc.build_year >= sqlc.narg('min_build_year')::int4)
  AND (sqlc.narg('max_build_year')::int4 IS NULL OR doc.build_year <= sqlc.narg('max_build_year')::int4)
  AND (sqlc.narg('condition')::text IS NULL OR trim(sqlc.narg('condition')::text) = '' OR lower(COALESCE(doc.condition, '')) LIKE ('%' || lower(trim(sqlc.narg('condition')::text)) || '%'))
  AND (sqlc.narg('energy_class')::text IS NULL OR trim(sqlc.narg('energy_class')::text) = '' OR lower(COALESCE(doc.energy_class, '')) LIKE ('%' || lower(trim(sqlc.narg('energy_class')::text)) || '%'))
ORDER BY
    CASE WHEN sqlc.arg('sort_mode') = 'price_asc' THEN doc.asking_price END ASC NULLS LAST,
    CASE WHEN sqlc.arg('sort_mode') = 'price_desc' THEN doc.asking_price END DESC NULLS LAST,
    CASE WHEN sqlc.arg('sort_mode') = 'area_asc' THEN doc.area_m2 END ASC NULLS LAST,
    CASE WHEN sqlc.arg('sort_mode') = 'area_desc' THEN doc.area_m2 END DESC NULLS LAST,
    CASE WHEN sqlc.arg('sort_mode') = 'price_m2_asc' THEN doc.price_per_m2 END ASC NULLS LAST,
    CASE WHEN sqlc.arg('sort_mode') = 'price_m2_desc' THEN doc.price_per_m2 END DESC NULLS LAST,
    CASE WHEN sqlc.arg('sort_mode') = 'build_year_desc' THEN doc.build_year END DESC NULLS LAST,
    CASE WHEN sqlc.arg('sort_mode') = 'seen_desc' THEN doc.last_seen_at END DESC NULLS LAST,
    doc.last_seen_at DESC
LIMIT sqlc.arg('limit_count')::int OFFSET sqlc.arg('offset_count')::int;

-- name: SearchRentalListings :many
SELECT
    ''::text AS source,
    ''::text AS kind,
    ''::text AS native_id,
    ''::text AS canonical_id,
    ''::text AS public_id,
    NULL::text AS url,
    NULL::text AS headline,
    NULL::text AS address,
    NULL::text AS city,
    NULL::text AS postal,
    NULL::bigint AS price,
    NULL::float8 AS area,
    NULL::text AS room_layout,
    NULL::float8 AS price_per_m2,
    NULL::bigint AS debt_free_price,
    NULL::bigint AS debt_share_amount,
    NULL::int4 AS rooms_count,
    NULL::int4 AS floor_level,
    NULL::int4 AS total_floors,
    NULL::int4 AS build_year,
    NULL::text AS condition,
    NULL::text AS energy_class,
    NULL::text AS energy_efficiency_label,
    NULL::text AS last_seen_at,
    NULL::text AS published_at,
    NULL::text AS building_key_address,
    ARRAY[]::text[] AS source_providers
WHERE false
  AND (sqlc.arg('source') = 'all' OR sqlc.arg('source') <> '')
  AND (sqlc.arg('sort_mode') <> '')
  AND (sqlc.arg('limit_count')::int >= 0)
  AND (sqlc.arg('offset_count')::int >= 0);

-- name: CountSaleListings :one
SELECT count(*)::bigint
FROM public.listing_search_documents doc
WHERE doc.listing_status = 'active'
  AND (sqlc.arg('source') = 'all' OR doc.source_providers @> ARRAY[sqlc.arg('source')::text])
  AND (sqlc.arg('kind') = 'all' OR doc.source_kinds @> ARRAY[sqlc.arg('kind')::text])
  AND (sqlc.narg('query_text')::text IS NULL OR trim(sqlc.narg('query_text')::text) = '' OR lower(doc.search_text) LIKE ('%' || lower(trim(sqlc.narg('query_text')::text)) || '%'))
  AND (sqlc.narg('city')::text IS NULL OR trim(sqlc.narg('city')::text) = '' OR lower(COALESCE(doc.city, '')) LIKE ('%' || lower(trim(sqlc.narg('city')::text)) || '%'))
  AND (sqlc.narg('postal')::text IS NULL OR trim(sqlc.narg('postal')::text) = '' OR lower(COALESCE(doc.postal, '')) LIKE ('%' || lower(trim(sqlc.narg('postal')::text)) || '%'))
  AND (sqlc.narg('min_price')::bigint IS NULL OR doc.asking_price >= sqlc.narg('min_price')::bigint)
  AND (sqlc.narg('max_price')::bigint IS NULL OR doc.asking_price <= sqlc.narg('max_price')::bigint)
  AND (sqlc.narg('min_area')::float8 IS NULL OR doc.area_m2 >= sqlc.narg('min_area')::float8)
  AND (sqlc.narg('max_area')::float8 IS NULL OR doc.area_m2 <= sqlc.narg('max_area')::float8)
  AND (sqlc.narg('published_after')::timestamptz IS NULL OR doc.published_at >= sqlc.narg('published_after')::timestamptz)
  AND (sqlc.narg('published_before')::timestamptz IS NULL OR doc.published_at <= sqlc.narg('published_before')::timestamptz)
  AND (sqlc.narg('min_price_per_m2')::float8 IS NULL OR doc.price_per_m2 >= sqlc.narg('min_price_per_m2')::float8)
  AND (sqlc.narg('max_price_per_m2')::float8 IS NULL OR doc.price_per_m2 <= sqlc.narg('max_price_per_m2')::float8)
  AND (sqlc.narg('rooms')::int4 IS NULL OR doc.rooms_count = sqlc.narg('rooms')::int4)
  AND (sqlc.narg('floor')::int4 IS NULL OR doc.floor_level = sqlc.narg('floor')::int4)
  AND (sqlc.narg('min_build_year')::int4 IS NULL OR doc.build_year >= sqlc.narg('min_build_year')::int4)
  AND (sqlc.narg('max_build_year')::int4 IS NULL OR doc.build_year <= sqlc.narg('max_build_year')::int4)
  AND (sqlc.narg('condition')::text IS NULL OR trim(sqlc.narg('condition')::text) = '' OR lower(COALESCE(doc.condition, '')) LIKE ('%' || lower(trim(sqlc.narg('condition')::text)) || '%'))
  AND (sqlc.narg('energy_class')::text IS NULL OR trim(sqlc.narg('energy_class')::text) = '' OR lower(COALESCE(doc.energy_class, '')) LIKE ('%' || lower(trim(sqlc.narg('energy_class')::text)) || '%'));

-- name: CountRentalListings :one
SELECT count(*)::bigint
WHERE false
  AND (sqlc.arg('source') = 'all' OR sqlc.arg('source') <> '');

-- name: ListRentalCanonicalIDs :many
SELECT ''::text AS canonical_id
WHERE false;

-- name: ListBuildingCanonicalIDs :many
SELECT physical_building_id::text AS canonical_id
FROM public.physical_buildings;
