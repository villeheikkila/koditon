CREATE OR REPLACE FUNCTION public.fnc__merge_property_offerings(
    source_offering_id uuid,
    target_offering_id uuid,
    link_method text DEFAULT 'source_match_auto',
    link_score integer DEFAULT NULL,
    link_confidence text DEFAULT NULL,
    link_reasons jsonb DEFAULT '{}'::jsonb,
    match_candidate_id uuid DEFAULT NULL
)
RETURNS uuid
LANGUAGE plpgsql
AS $$
DECLARE
    decision_id uuid;
BEGIN
    IF source_offering_id IS NULL OR target_offering_id IS NULL OR source_offering_id = target_offering_id THEN
        RETURN NULL;
    END IF;
    INSERT INTO public.property_offering_merge_decisions (
        source_property_offering_id,
        target_property_offering_id,
        property_offering_source_match_candidate_id,
        property_offering_merge_decision_status,
        property_offering_merge_decision_method,
        property_offering_merge_decision_score,
        property_offering_merge_decision_confidence,
        property_offering_merge_decision_reasons
    )
    VALUES (
        source_offering_id,
        target_offering_id,
        match_candidate_id,
        'accepted',
        CASE WHEN link_method = ANY (ARRAY['manual'::text, 'backfill_auto'::text]) THEN link_method ELSE 'source_match_auto'::text END,
        link_score,
        link_confidence,
        COALESCE(link_reasons, '{}'::jsonb)
    )
    ON CONFLICT DO NOTHING
    RETURNING property_offering_merge_decision_id INTO decision_id;
    IF decision_id IS NULL THEN
        SELECT property_offering_merge_decision_id INTO decision_id
        FROM public.property_offering_merge_decisions
        WHERE source_property_offering_id = source_offering_id
            AND target_property_offering_id = target_offering_id
            AND property_offering_merge_decision_status <> 'rejected'
        ORDER BY property_offering_merge_decision_created_at DESC
        LIMIT 1;
    END IF;
    UPDATE public.property_offering_sources pos
    SET
        property_offering_id = target_offering_id,
        property_offering_source_link_method = CASE
            WHEN pos.property_offering_source_link_method = 'manual' THEN 'manual'
            ELSE CASE WHEN link_method = ANY (ARRAY['manual'::text, 'backfill_auto'::text]) THEN link_method ELSE 'source_match_auto'::text END
        END,
        property_offering_source_link_status = 'confirmed',
        property_offering_source_link_score = GREATEST(pos.property_offering_source_link_score, COALESCE(link_score, pos.property_offering_source_link_score)),
        property_offering_source_link_reasons = pos.property_offering_source_link_reasons || COALESCE(link_reasons, '{}'::jsonb) || jsonb_build_object('merge_decision_id', decision_id, 'merged_from_property_offering_id', source_offering_id),
        property_offering_source_updated_at = now()
    WHERE pos.property_offering_id = source_offering_id
        AND pos.property_offering_source_link_status <> 'rejected';
    INSERT INTO public.property_offering_transactions (
        property_offering_id,
        prices_transaction_id,
        property_offering_transaction_link_status,
        property_offering_transaction_link_method,
        property_offering_transaction_link_score,
        property_offering_transaction_link_reasons,
        property_offering_transaction_updated_at
    )
    SELECT
        target_offering_id,
        pot.prices_transaction_id,
        pot.property_offering_transaction_link_status,
        pot.property_offering_transaction_link_method,
        pot.property_offering_transaction_link_score,
        pot.property_offering_transaction_link_reasons || jsonb_build_object('merge_decision_id', decision_id, 'merged_from_property_offering_id', source_offering_id),
        now()
    FROM public.property_offering_transactions pot
    WHERE pot.property_offering_id = source_offering_id
    ON CONFLICT (property_offering_id, prices_transaction_id) DO UPDATE SET
        property_offering_transaction_link_score = GREATEST(property_offering_transactions.property_offering_transaction_link_score, EXCLUDED.property_offering_transaction_link_score),
        property_offering_transaction_link_reasons = property_offering_transactions.property_offering_transaction_link_reasons || EXCLUDED.property_offering_transaction_link_reasons,
        property_offering_transaction_updated_at = now();
    DELETE FROM public.property_offering_transactions pot
    WHERE pot.property_offering_id = source_offering_id;
    UPDATE public.property_offering_source_match_candidates c
    SET property_offering_source_match_status = 'auto_linked'
    WHERE c.property_offering_source_match_candidate_id = match_candidate_id
        AND c.property_offering_source_match_status <> 'rejected';
    UPDATE public.property_offerings po
    SET property_offering_updated_at = now()
    WHERE po.property_offering_id = ANY (ARRAY[source_offering_id, target_offering_id]);
    RETURN decision_id;
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
    decision_id uuid;
    merge_source_id uuid := source_housing_company_id;
    merge_target_id uuid := target_housing_company_id;
BEGIN
    IF merge_source_id IS NULL OR merge_target_id IS NULL OR merge_source_id = merge_target_id THEN
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
        merge_source_id,
        merge_target_id,
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
        WHERE d.source_housing_company_id = merge_source_id
            AND d.target_housing_company_id = merge_target_id
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
        housing_company_match_reasons = target.housing_company_match_reasons || source.housing_company_match_reasons || COALESCE(merge_reasons, '{}'::jsonb) || jsonb_build_object('merged_from_housing_company_id', merge_source_id, 'merge_decision_id', decision_id),
        housing_company_updated_at = now()
    FROM public.housing_companies source
    WHERE target.housing_company_id = merge_target_id
        AND source.housing_company_id = merge_source_id;
    UPDATE public.property_units pu
    SET
        housing_company_id = merge_target_id,
        property_unit_updated_at = now()
    WHERE pu.housing_company_id = merge_source_id;
    UPDATE public.housing_company_sources hcs
    SET
        housing_company_id = merge_target_id,
        housing_company_source_link_status = 'confirmed',
        housing_company_source_link_method = CASE
            WHEN hcs.housing_company_source_link_method = 'manual' THEN 'manual'
            ELSE CASE WHEN merge_method = ANY (ARRAY['source_match_auto'::text, 'backfill_auto'::text]) THEN merge_method ELSE 'manual'::text END
        END,
        housing_company_source_link_score = GREATEST(hcs.housing_company_source_link_score, COALESCE(merge_score, hcs.housing_company_source_link_score)),
        housing_company_source_link_reasons = hcs.housing_company_source_link_reasons || COALESCE(merge_reasons, '{}'::jsonb) || jsonb_build_object('merge_decision_id', decision_id, 'merged_from_housing_company_id', merge_source_id),
        housing_company_source_updated_at = now()
    WHERE hcs.housing_company_id = merge_source_id
        AND hcs.housing_company_source_link_status <> 'rejected';
    UPDATE public.housing_company_facts hcf
    SET
        housing_company_id = merge_target_id,
        housing_company_fact_updated_at = now()
    WHERE hcf.housing_company_id = merge_source_id;
    PERFORM public.fnc__refresh_housing_company_geom(merge_target_id);
    RETURN decision_id;
END;
$$;
DROP FUNCTION IF EXISTS public.fnc__resolve_property_offering_id(uuid);
DROP FUNCTION IF EXISTS public.fnc__resolve_housing_company_id(uuid);
DROP FUNCTION IF EXISTS public.fnc__consolidate_property_offering_redirect_batch(integer);
DROP TABLE IF EXISTS public.property_offering_redirects;
DROP TABLE IF EXISTS public.housing_company_redirects;
