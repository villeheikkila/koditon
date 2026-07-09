CREATE TABLE IF NOT EXISTS public.source_listing_match_facts (
    source_listing_id uuid NOT NULL,
    provider text NOT NULL,
    source_kind text NOT NULL,
    postal_norm text,
    city_norm text,
    address_norm text,
    street_norm text,
    house_norm text,
    stair_norm text,
    apartment_norm text,
    area_m2 double precision,
    area_tenths integer,
    rooms_count integer,
    room_layout_norm text,
    floor_level integer,
    asking_price bigint,
    debt_free_price bigint,
    build_year integer,
    housing_company_business_id text,
    housing_company_name_norm text,
    first_seen_at timestamp with time zone,
    last_seen_at timestamp with time zone,
    refreshed_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE TABLE IF NOT EXISTS public.source_listing_match_candidates (
    source_listing_match_candidate_id uuid DEFAULT gen_random_uuid() NOT NULL,
    source_listing_id_a uuid NOT NULL,
    source_listing_id_b uuid NOT NULL,
    match_method text NOT NULL,
    match_score integer NOT NULL,
    match_confidence text NOT NULL,
    match_status text DEFAULT 'proposed'::text NOT NULL,
    match_reasons jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    decided_at timestamp with time zone
);

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE connamespace = 'public'::regnamespace AND conrelid = 'public.source_listing_match_facts'::regclass AND conname = 'source_listing_match_facts_pkey') THEN
        ALTER TABLE public.source_listing_match_facts ADD CONSTRAINT source_listing_match_facts_pkey PRIMARY KEY (source_listing_id);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE connamespace = 'public'::regnamespace AND conrelid = 'public.source_listing_match_candidates'::regclass AND conname = 'source_listing_match_candidates_pkey') THEN
        ALTER TABLE public.source_listing_match_candidates ADD CONSTRAINT source_listing_match_candidates_pkey PRIMARY KEY (source_listing_match_candidate_id);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE connamespace = 'public'::regnamespace AND conrelid = 'public.source_listing_match_candidates'::regclass AND conname = 'source_listing_match_candidates_distinct_check') THEN
        ALTER TABLE public.source_listing_match_candidates ADD CONSTRAINT source_listing_match_candidates_distinct_check CHECK ((source_listing_id_a <> source_listing_id_b));
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE connamespace = 'public'::regnamespace AND conrelid = 'public.source_listing_match_candidates'::regclass AND conname = 'source_listing_match_candidates_order_check') THEN
        ALTER TABLE public.source_listing_match_candidates ADD CONSTRAINT source_listing_match_candidates_order_check CHECK ((source_listing_id_a < source_listing_id_b));
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE connamespace = 'public'::regnamespace AND conrelid = 'public.source_listing_match_candidates'::regclass AND conname = 'source_listing_match_candidates_confidence_check') THEN
        ALTER TABLE public.source_listing_match_candidates ADD CONSTRAINT source_listing_match_candidates_confidence_check CHECK ((match_confidence = ANY (ARRAY['high'::text, 'medium'::text, 'low'::text])));
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE connamespace = 'public'::regnamespace AND conrelid = 'public.source_listing_match_candidates'::regclass AND conname = 'source_listing_match_candidates_method_check') THEN
        ALTER TABLE public.source_listing_match_candidates ADD CONSTRAINT source_listing_match_candidates_method_check CHECK ((match_method = ANY (ARRAY['exact_provider_neutral_unit_v1'::text, 'address_missing_stair_one_to_one_v1'::text])));
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE connamespace = 'public'::regnamespace AND conrelid = 'public.source_listing_match_candidates'::regclass AND conname = 'source_listing_match_candidates_status_check') THEN
        ALTER TABLE public.source_listing_match_candidates ADD CONSTRAINT source_listing_match_candidates_status_check CHECK ((match_status = ANY (ARRAY['proposed'::text, 'accepted'::text, 'rejected'::text, 'superseded'::text])));
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE connamespace = 'public'::regnamespace AND conrelid = 'public.source_listing_match_facts'::regclass AND conname = 'source_listing_match_facts_source_listing_id_fkey') THEN
        ALTER TABLE public.source_listing_match_facts ADD CONSTRAINT source_listing_match_facts_source_listing_id_fkey FOREIGN KEY (source_listing_id) REFERENCES origin.source_listings(source_listing_id) ON DELETE CASCADE;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE connamespace = 'public'::regnamespace AND conrelid = 'public.source_listing_match_candidates'::regclass AND conname = 'source_listing_match_candidates_source_listing_id_a_fkey') THEN
        ALTER TABLE public.source_listing_match_candidates ADD CONSTRAINT source_listing_match_candidates_source_listing_id_a_fkey FOREIGN KEY (source_listing_id_a) REFERENCES origin.source_listings(source_listing_id) ON DELETE CASCADE;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE connamespace = 'public'::regnamespace AND conrelid = 'public.source_listing_match_candidates'::regclass AND conname = 'source_listing_match_candidates_source_listing_id_b_fkey') THEN
        ALTER TABLE public.source_listing_match_candidates ADD CONSTRAINT source_listing_match_candidates_source_listing_id_b_fkey FOREIGN KEY (source_listing_id_b) REFERENCES origin.source_listings(source_listing_id) ON DELETE CASCADE;
    END IF;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS idx_source_listing_match_candidates_active_pair_method ON public.source_listing_match_candidates USING btree (source_listing_id_a, source_listing_id_b, match_method) WHERE (match_status <> 'rejected'::text);
CREATE INDEX IF NOT EXISTS idx_source_listing_match_candidates_a ON public.source_listing_match_candidates USING btree (source_listing_id_a, match_status);
CREATE INDEX IF NOT EXISTS idx_source_listing_match_candidates_b ON public.source_listing_match_candidates USING btree (source_listing_id_b, match_status);
CREATE INDEX IF NOT EXISTS idx_source_listing_match_facts_block ON public.source_listing_match_facts USING btree (postal_norm, street_norm, house_norm, area_tenths) WHERE ((postal_norm IS NOT NULL) AND (street_norm IS NOT NULL) AND (house_norm IS NOT NULL) AND (area_tenths IS NOT NULL));
CREATE INDEX IF NOT EXISTS idx_source_listing_match_facts_provider ON public.source_listing_match_facts USING btree (provider, source_kind);

WITH base AS (
    SELECT
        doc.primary_source_listing_id AS source_listing_id,
        sl.provider,
        sl.source_kind,
        NULLIF(regexp_replace(trim(COALESCE(doc.postal, '')), '[^0-9]+', '', 'g'), '') AS postal_norm,
        NULLIF(lower(trim(COALESCE(doc.city, ''))), '') AS city_norm,
        NULLIF(lower(trim(regexp_replace(COALESCE(doc.address, ''), '[[:space:]]+', ' ', 'g'))), '') AS address_norm,
        doc.area_m2,
        doc.rooms_count,
        NULLIF(lower(trim(doc.room_layout)), '') AS room_layout_norm,
        doc.floor_level,
        doc.asking_price,
        doc.debt_free_price,
        doc.build_year,
        doc.first_seen_at,
        doc.last_seen_at
    FROM public.listing_search_documents doc
    JOIN origin.source_listings sl ON sl.source_listing_id = doc.primary_source_listing_id
    WHERE doc.primary_source_listing_id IS NOT NULL
),
parsed AS (
    SELECT
        base.*,
        NULLIF(regexp_replace(base.address_norm, '[[:space:]]+[0-9]+.*$', ''), '') AS street_norm,
        NULLIF(substring(base.address_norm from '([0-9]+([-–][0-9]+)?)[[:alpha:]]?($|[[:space:]])'), '') AS house_norm,
        NULLIF(substring(base.address_norm from '[0-9]+[-–]?[0-9]*[[:space:]]*([[:alpha:]])([[:space:]]*[0-9]+)?[[:space:]]*$'), '') AS stair_norm,
        NULLIF(substring(base.address_norm from '[0-9]+[-–]?[0-9]*[[:space:]]*[[:alpha:]]?[[:space:]]+([0-9]{1,4})[[:space:]]*$'), '') AS apartment_norm,
        CASE WHEN base.area_m2 IS NULL THEN NULL ELSE round(base.area_m2::numeric * 10)::int4 END AS area_tenths
    FROM base
)
INSERT INTO public.source_listing_match_facts (
    source_listing_id,
    provider,
    source_kind,
    postal_norm,
    city_norm,
    address_norm,
    street_norm,
    house_norm,
    stair_norm,
    apartment_norm,
    area_m2,
    area_tenths,
    rooms_count,
    room_layout_norm,
    floor_level,
    asking_price,
    debt_free_price,
    build_year,
    first_seen_at,
    last_seen_at,
    refreshed_at
)
SELECT
    source_listing_id,
    provider,
    source_kind,
    postal_norm,
    city_norm,
    address_norm,
    street_norm,
    house_norm,
    stair_norm,
    apartment_norm,
    area_m2,
    area_tenths,
    rooms_count,
    room_layout_norm,
    floor_level,
    asking_price,
    debt_free_price,
    build_year,
    first_seen_at,
    last_seen_at,
    now()
FROM parsed
ON CONFLICT (source_listing_id) DO UPDATE SET
    provider = EXCLUDED.provider,
    source_kind = EXCLUDED.source_kind,
    postal_norm = EXCLUDED.postal_norm,
    city_norm = EXCLUDED.city_norm,
    address_norm = EXCLUDED.address_norm,
    street_norm = EXCLUDED.street_norm,
    house_norm = EXCLUDED.house_norm,
    stair_norm = EXCLUDED.stair_norm,
    apartment_norm = EXCLUDED.apartment_norm,
    area_m2 = EXCLUDED.area_m2,
    area_tenths = EXCLUDED.area_tenths,
    rooms_count = EXCLUDED.rooms_count,
    room_layout_norm = EXCLUDED.room_layout_norm,
    floor_level = EXCLUDED.floor_level,
    asking_price = EXCLUDED.asking_price,
    debt_free_price = EXCLUDED.debt_free_price,
    build_year = EXCLUDED.build_year,
    first_seen_at = EXCLUDED.first_seen_at,
    last_seen_at = EXCLUDED.last_seen_at,
    refreshed_at = now();

WITH compatible_pairs AS (
    SELECT
        a.source_listing_id AS source_listing_id_a,
        b.source_listing_id AS source_listing_id_b,
        a.provider AS source_provider,
        b.provider AS matched_provider,
        a.address_norm AS source_address_norm,
        b.address_norm AS matched_address_norm,
        a.postal_norm,
        a.street_norm,
        a.house_norm,
        a.area_tenths,
        a.stair_norm AS source_stair_norm,
        b.stair_norm AS matched_stair_norm,
        a.apartment_norm AS source_apartment_norm,
        b.apartment_norm AS matched_apartment_norm,
        a.asking_price AS source_asking_price,
        b.asking_price AS matched_asking_price,
        a.debt_free_price AS source_debt_free_price,
        b.debt_free_price AS matched_debt_free_price
    FROM public.source_listing_match_facts a
    JOIN public.source_listing_match_facts b ON b.source_listing_id > a.source_listing_id
        AND b.postal_norm = a.postal_norm
        AND b.street_norm = a.street_norm
        AND b.house_norm = a.house_norm
        AND b.area_tenths = a.area_tenths
    WHERE a.postal_norm IS NOT NULL
        AND a.street_norm IS NOT NULL
        AND a.house_norm IS NOT NULL
        AND a.area_tenths IS NOT NULL
        AND (a.stair_norm IS NULL OR b.stair_norm IS NULL OR a.stair_norm = b.stair_norm)
        AND (a.apartment_norm IS NULL OR b.apartment_norm IS NULL OR a.apartment_norm = b.apartment_norm)
),
pair_counts AS (
    SELECT source_listing_id, count(*)::int4 AS compatible_pair_count
    FROM (
        SELECT source_listing_id_a AS source_listing_id FROM compatible_pairs
        UNION ALL
        SELECT source_listing_id_b AS source_listing_id FROM compatible_pairs
    ) pair_members
    GROUP BY source_listing_id
),
classified_pairs AS (
    SELECT
        compatible_pairs.*,
        source_counts.compatible_pair_count AS source_pair_count,
        matched_counts.compatible_pair_count AS matched_pair_count
    FROM compatible_pairs
    JOIN pair_counts source_counts ON source_counts.source_listing_id = compatible_pairs.source_listing_id_a
    JOIN pair_counts matched_counts ON matched_counts.source_listing_id = compatible_pairs.source_listing_id_b
),
candidates AS (
    SELECT
        source_listing_id_a,
        source_listing_id_b,
        CASE
            WHEN source_address_norm = matched_address_norm THEN 'exact_provider_neutral_unit_v1'
            ELSE 'address_missing_stair_one_to_one_v1'
        END AS match_method,
        CASE WHEN source_address_norm = matched_address_norm THEN 100 ELSE 95 END AS match_score,
        'high'::text AS match_confidence,
        CASE WHEN source_pair_count = 1 AND matched_pair_count = 1 THEN 'accepted' ELSE 'proposed' END AS match_status,
        jsonb_strip_nulls(jsonb_build_object(
            'postal_norm', postal_norm,
            'street_norm', street_norm,
            'house_norm', house_norm,
            'area_tenths', area_tenths,
            'source_provider', source_provider,
            'matched_provider', matched_provider,
            'source_address_norm', source_address_norm,
            'matched_address_norm', matched_address_norm,
            'source_stair_norm', source_stair_norm,
            'matched_stair_norm', matched_stair_norm,
            'source_apartment_norm', source_apartment_norm,
            'matched_apartment_norm', matched_apartment_norm,
            'source_asking_price', source_asking_price,
            'matched_asking_price', matched_asking_price,
            'source_debt_free_price', source_debt_free_price,
            'matched_debt_free_price', matched_debt_free_price,
            'source_pair_count', source_pair_count,
            'matched_pair_count', matched_pair_count
        )) AS match_reasons
    FROM classified_pairs
)
INSERT INTO public.source_listing_match_candidates (
    source_listing_id_a,
    source_listing_id_b,
    match_method,
    match_score,
    match_confidence,
    match_status,
    match_reasons,
    updated_at,
    decided_at
)
SELECT
    source_listing_id_a,
    source_listing_id_b,
    match_method,
    match_score,
    match_confidence,
    match_status,
    match_reasons,
    now(),
    CASE WHEN match_status = 'accepted' THEN now() ELSE NULL::timestamptz END
FROM candidates
WHERE match_status = 'accepted'
    OR ((match_reasons ->> 'source_pair_count')::integer <= 2 AND (match_reasons ->> 'matched_pair_count')::integer <= 2)
ON CONFLICT (source_listing_id_a, source_listing_id_b, match_method) WHERE (match_status <> 'rejected') DO UPDATE SET
    match_score = EXCLUDED.match_score,
    match_confidence = EXCLUDED.match_confidence,
    match_status = EXCLUDED.match_status,
    match_reasons = EXCLUDED.match_reasons,
    updated_at = now(),
    decided_at = CASE WHEN EXCLUDED.match_status = 'accepted' THEN COALESCE(public.source_listing_match_candidates.decided_at, now()) ELSE public.source_listing_match_candidates.decided_at END;
