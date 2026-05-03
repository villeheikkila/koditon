CREATE TABLE IF NOT EXISTS public.housing_company_merge_decisions (
    housing_company_merge_decision_id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    source_housing_company_id uuid NOT NULL REFERENCES public.housing_companies(housing_company_id) ON DELETE CASCADE,
    target_housing_company_id uuid NOT NULL REFERENCES public.housing_companies(housing_company_id) ON DELETE CASCADE,
    housing_company_merge_decision_status text DEFAULT 'accepted'::text NOT NULL,
    housing_company_merge_decision_method text NOT NULL,
    housing_company_merge_decision_score integer,
    housing_company_merge_decision_confidence text,
    housing_company_merge_decision_reasons jsonb DEFAULT '{}'::jsonb NOT NULL,
    housing_company_merge_decision_created_at timestamptz DEFAULT now() NOT NULL,
    housing_company_merge_decision_decided_at timestamptz DEFAULT now() NOT NULL,
    CONSTRAINT housing_company_merge_decision_distinct_check CHECK (source_housing_company_id <> target_housing_company_id),
    CONSTRAINT housing_company_merge_decision_status_check CHECK (housing_company_merge_decision_status = ANY (ARRAY['proposed'::text, 'accepted'::text, 'rejected'::text, 'superseded'::text])),
    CONSTRAINT housing_company_merge_decision_method_check CHECK (housing_company_merge_decision_method = ANY (ARRAY['source_match_auto'::text, 'manual'::text, 'backfill_auto'::text]))
);
CREATE INDEX IF NOT EXISTS idx_housing_company_merge_decisions_source ON public.housing_company_merge_decisions (source_housing_company_id, housing_company_merge_decision_status);
CREATE INDEX IF NOT EXISTS idx_housing_company_merge_decisions_target ON public.housing_company_merge_decisions (target_housing_company_id, housing_company_merge_decision_status);
CREATE UNIQUE INDEX IF NOT EXISTS idx_housing_company_merge_decisions_active_pair
ON public.housing_company_merge_decisions (source_housing_company_id, target_housing_company_id)
WHERE housing_company_merge_decision_status <> 'rejected'::text;
CREATE TABLE IF NOT EXISTS public.housing_company_redirects (
    source_housing_company_id uuid NOT NULL PRIMARY KEY REFERENCES public.housing_companies(housing_company_id) ON DELETE CASCADE,
    target_housing_company_id uuid NOT NULL REFERENCES public.housing_companies(housing_company_id) ON DELETE CASCADE,
    housing_company_merge_decision_id uuid REFERENCES public.housing_company_merge_decisions(housing_company_merge_decision_id) ON DELETE SET NULL,
    housing_company_redirect_reason text NOT NULL,
    housing_company_redirect_created_at timestamptz DEFAULT now() NOT NULL,
    CONSTRAINT housing_company_redirect_distinct_check CHECK (source_housing_company_id <> target_housing_company_id)
);
CREATE INDEX IF NOT EXISTS idx_housing_company_redirects_target ON public.housing_company_redirects (target_housing_company_id);
CREATE OR REPLACE FUNCTION public.fnc__resolve_housing_company_id(input_housing_company_id uuid)
RETURNS uuid
LANGUAGE plpgsql
AS $$
DECLARE
    current_id uuid := input_housing_company_id;
    next_id uuid;
    depth integer := 0;
BEGIN
    LOOP
        SELECT r.target_housing_company_id
        INTO next_id
        FROM public.housing_company_redirects r
        WHERE r.source_housing_company_id = current_id;
        IF next_id IS NULL OR next_id = current_id THEN
            RETURN current_id;
        END IF;
        current_id := next_id;
        depth := depth + 1;
        IF depth > 8 THEN
            RAISE EXCEPTION 'housing company redirect chain too deep for %', input_housing_company_id;
        END IF;
    END LOOP;
END;
$$;
CREATE OR REPLACE FUNCTION public.fnc__merge_housing_companies(
    source_housing_company_id uuid,
    target_housing_company_id uuid,
    merge_method text DEFAULT 'manual',
    merge_score integer DEFAULT NULL,
    merge_confidence text DEFAULT NULL,
    merge_reasons jsonb DEFAULT '{}'::jsonb
)
RETURNS uuid
LANGUAGE plpgsql
AS $$
DECLARE
    resolved_source uuid;
    resolved_target uuid;
    decision_id uuid;
BEGIN
    IF source_housing_company_id IS NULL OR target_housing_company_id IS NULL THEN
        RETURN NULL;
    END IF;
    resolved_source := public.fnc__resolve_housing_company_id(source_housing_company_id);
    resolved_target := public.fnc__resolve_housing_company_id(target_housing_company_id);
    IF resolved_source = resolved_target THEN
        RETURN NULL;
    END IF;
    INSERT INTO public.housing_company_merge_decisions (
        source_housing_company_id,
        target_housing_company_id,
        housing_company_merge_decision_status,
        housing_company_merge_decision_method,
        housing_company_merge_decision_score,
        housing_company_merge_decision_confidence,
        housing_company_merge_decision_reasons
    )
    VALUES (
        resolved_source,
        resolved_target,
        'accepted',
        CASE WHEN merge_method = ANY (ARRAY['source_match_auto'::text, 'backfill_auto'::text]) THEN merge_method ELSE 'manual'::text END,
        merge_score,
        merge_confidence,
        COALESCE(merge_reasons, '{}'::jsonb)
    )
    ON CONFLICT DO NOTHING
    RETURNING housing_company_merge_decision_id INTO decision_id;
    IF decision_id IS NULL THEN
        SELECT d.housing_company_merge_decision_id
        INTO decision_id
        FROM public.housing_company_merge_decisions d
        WHERE d.source_housing_company_id = resolved_source
            AND d.target_housing_company_id = resolved_target
            AND d.housing_company_merge_decision_status <> 'rejected'
        ORDER BY d.housing_company_merge_decision_created_at DESC
        LIMIT 1;
    END IF;
    UPDATE public.housing_companies target
    SET
        housing_company_name = COALESCE(NULLIF(target.housing_company_name, ''), NULLIF(source.housing_company_name, '')),
        housing_company_business_id = COALESCE(NULLIF(target.housing_company_business_id, ''), NULLIF(source.housing_company_business_id, '')),
        housing_company_postal_norm = COALESCE(NULLIF(target.housing_company_postal_norm, ''), NULLIF(source.housing_company_postal_norm, '')),
        housing_company_city_norm = COALESCE(NULLIF(target.housing_company_city_norm, ''), NULLIF(source.housing_company_city_norm, '')),
        housing_company_address_norm = COALESCE(NULLIF(target.housing_company_address_norm, ''), NULLIF(source.housing_company_address_norm, '')),
        housing_company_build_year = COALESCE(target.housing_company_build_year, source.housing_company_build_year),
        housing_company_floor_count = COALESCE(target.housing_company_floor_count, source.housing_company_floor_count),
        housing_company_apartment_count = COALESCE(target.housing_company_apartment_count, source.housing_company_apartment_count),
        housing_company_elevator = COALESCE(target.housing_company_elevator, source.housing_company_elevator),
        housing_company_energy_efficiency_label = COALESCE(NULLIF(target.housing_company_energy_efficiency_label, ''), NULLIF(source.housing_company_energy_efficiency_label, '')),
        housing_company_geom = COALESCE(target.housing_company_geom, source.housing_company_geom),
        housing_company_match_reasons = target.housing_company_match_reasons || source.housing_company_match_reasons || COALESCE(merge_reasons, '{}'::jsonb) || jsonb_build_object('merged_from_housing_company_id', resolved_source, 'merge_decision_id', decision_id),
        housing_company_updated_at = now()
    FROM public.housing_companies source
    WHERE target.housing_company_id = resolved_target
        AND source.housing_company_id = resolved_source;
    INSERT INTO public.housing_company_redirects (
        source_housing_company_id,
        target_housing_company_id,
        housing_company_merge_decision_id,
        housing_company_redirect_reason
    )
    VALUES (resolved_source, resolved_target, decision_id, merge_method)
    ON CONFLICT (source_housing_company_id) DO UPDATE SET
        target_housing_company_id = EXCLUDED.target_housing_company_id,
        housing_company_merge_decision_id = EXCLUDED.housing_company_merge_decision_id,
        housing_company_redirect_reason = EXCLUDED.housing_company_redirect_reason;
    UPDATE public.housing_company_redirects r
    SET target_housing_company_id = resolved_target
    WHERE r.target_housing_company_id = resolved_source;
    UPDATE public.property_units pu
    SET
        housing_company_id = resolved_target,
        property_unit_updated_at = now()
    WHERE pu.housing_company_id = resolved_source;
    UPDATE public.housing_company_sources hcs
    SET
        housing_company_id = resolved_target,
        housing_company_source_link_status = 'confirmed',
        housing_company_source_link_method = CASE
            WHEN hcs.housing_company_source_link_method = 'manual' THEN 'manual'
            ELSE CASE WHEN merge_method = ANY (ARRAY['source_match_auto'::text, 'backfill_auto'::text]) THEN merge_method ELSE 'manual'::text END
        END,
        housing_company_source_link_score = GREATEST(hcs.housing_company_source_link_score, COALESCE(merge_score, hcs.housing_company_source_link_score)),
        housing_company_source_link_reasons = hcs.housing_company_source_link_reasons || COALESCE(merge_reasons, '{}'::jsonb) || jsonb_build_object('merge_decision_id', decision_id, 'merged_from_housing_company_id', resolved_source),
        housing_company_source_updated_at = now()
    WHERE hcs.housing_company_id = resolved_source
        AND hcs.housing_company_source_link_status <> 'rejected';
    UPDATE public.housing_company_facts hcf
    SET
        housing_company_id = resolved_target,
        housing_company_fact_updated_at = now()
    WHERE hcf.housing_company_id = resolved_source;
    PERFORM public.fnc__refresh_housing_company_geom(resolved_target);
    RETURN decision_id;
END;
$$;
