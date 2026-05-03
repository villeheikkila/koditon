CREATE TABLE IF NOT EXISTS public.property_offering_merge_decisions (
    property_offering_merge_decision_id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    source_property_offering_id uuid NOT NULL REFERENCES public.property_offerings(property_offering_id) ON DELETE CASCADE,
    target_property_offering_id uuid NOT NULL REFERENCES public.property_offerings(property_offering_id) ON DELETE CASCADE,
    property_offering_source_match_candidate_id uuid REFERENCES public.property_offering_source_match_candidates(property_offering_source_match_candidate_id) ON DELETE SET NULL,
    property_offering_merge_decision_status text DEFAULT 'accepted'::text NOT NULL,
    property_offering_merge_decision_method text NOT NULL,
    property_offering_merge_decision_score integer,
    property_offering_merge_decision_confidence text,
    property_offering_merge_decision_reasons jsonb DEFAULT '{}'::jsonb NOT NULL,
    property_offering_merge_decision_created_at timestamptz DEFAULT now() NOT NULL,
    property_offering_merge_decision_decided_at timestamptz DEFAULT now() NOT NULL,
    CONSTRAINT property_offering_merge_decision_distinct_check CHECK (source_property_offering_id <> target_property_offering_id),
    CONSTRAINT property_offering_merge_decision_status_check CHECK (property_offering_merge_decision_status = ANY (ARRAY['proposed'::text, 'accepted'::text, 'rejected'::text, 'superseded'::text])),
    CONSTRAINT property_offering_merge_decision_method_check CHECK (property_offering_merge_decision_method = ANY (ARRAY['source_match_auto'::text, 'manual'::text, 'backfill_auto'::text]))
);
CREATE INDEX IF NOT EXISTS idx_property_offering_merge_decisions_source ON public.property_offering_merge_decisions (source_property_offering_id, property_offering_merge_decision_status);
CREATE INDEX IF NOT EXISTS idx_property_offering_merge_decisions_target ON public.property_offering_merge_decisions (target_property_offering_id, property_offering_merge_decision_status);
CREATE UNIQUE INDEX IF NOT EXISTS idx_property_offering_merge_decisions_active_pair
ON public.property_offering_merge_decisions (source_property_offering_id, target_property_offering_id)
WHERE property_offering_merge_decision_status <> 'rejected'::text;
CREATE TABLE IF NOT EXISTS public.property_offering_redirects (
    source_property_offering_id uuid NOT NULL PRIMARY KEY REFERENCES public.property_offerings(property_offering_id) ON DELETE CASCADE,
    target_property_offering_id uuid NOT NULL REFERENCES public.property_offerings(property_offering_id) ON DELETE CASCADE,
    property_offering_merge_decision_id uuid REFERENCES public.property_offering_merge_decisions(property_offering_merge_decision_id) ON DELETE SET NULL,
    property_offering_redirect_reason text NOT NULL,
    property_offering_redirect_created_at timestamptz DEFAULT now() NOT NULL,
    CONSTRAINT property_offering_redirect_distinct_check CHECK (source_property_offering_id <> target_property_offering_id)
);
CREATE INDEX IF NOT EXISTS idx_property_offering_redirects_target ON public.property_offering_redirects (target_property_offering_id);
CREATE OR REPLACE FUNCTION public.fnc__resolve_property_offering_id(offering_id uuid)
RETURNS uuid
LANGUAGE plpgsql
AS $$
DECLARE
    current_id uuid := offering_id;
    next_id uuid;
    depth integer := 0;
BEGIN
    LOOP
        SELECT r.target_property_offering_id
        INTO next_id
        FROM public.property_offering_redirects r
        WHERE r.source_property_offering_id = current_id;
        IF next_id IS NULL OR next_id = current_id THEN
            RETURN current_id;
        END IF;
        current_id := next_id;
        depth := depth + 1;
        IF depth > 16 THEN
            RAISE EXCEPTION 'property offering redirect chain too deep for %', offering_id;
        END IF;
    END LOOP;
END;
$$;
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
    resolved_target uuid;
    decision_id uuid;
BEGIN
    IF source_offering_id IS NULL OR target_offering_id IS NULL OR source_offering_id = target_offering_id THEN
        RETURN NULL;
    END IF;
    resolved_target := public.fnc__resolve_property_offering_id(target_offering_id);
    IF source_offering_id = resolved_target THEN
        RETURN NULL;
    END IF;
    IF EXISTS (
        SELECT 1
        FROM public.property_offering_redirects r
        WHERE r.source_property_offering_id = source_offering_id
            AND r.target_property_offering_id = resolved_target
    ) THEN
        SELECT r.property_offering_merge_decision_id INTO decision_id
        FROM public.property_offering_redirects r
        WHERE r.source_property_offering_id = source_offering_id;
        RETURN decision_id;
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
        resolved_target,
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
            AND target_property_offering_id = resolved_target
            AND property_offering_merge_decision_status <> 'rejected'
        ORDER BY property_offering_merge_decision_created_at DESC
        LIMIT 1;
    END IF;
    INSERT INTO public.property_offering_redirects (
        source_property_offering_id,
        target_property_offering_id,
        property_offering_merge_decision_id,
        property_offering_redirect_reason
    )
    VALUES (source_offering_id, resolved_target, decision_id, link_method)
    ON CONFLICT (source_property_offering_id) DO UPDATE SET
        target_property_offering_id = EXCLUDED.target_property_offering_id,
        property_offering_merge_decision_id = EXCLUDED.property_offering_merge_decision_id,
        property_offering_redirect_reason = EXCLUDED.property_offering_redirect_reason;
    UPDATE public.property_offering_sources pos
    SET
        property_offering_id = resolved_target,
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
        resolved_target,
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
    WHERE po.property_offering_id = ANY (ARRAY[source_offering_id, resolved_target]);
    RETURN decision_id;
END;
$$;
CREATE OR REPLACE FUNCTION public.fnc__property_offering_source_merge_trigger()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    candidate_id uuid;
    confidence text;
BEGIN
    IF pg_trigger_depth() > 1 THEN
        RETURN NEW;
    END IF;
    IF TG_OP <> 'UPDATE' OR OLD.property_offering_id = NEW.property_offering_id THEN
        RETURN NEW;
    END IF;
    IF NEW.property_offering_source_link_status = 'rejected' OR NOT (NEW.property_offering_source_link_method = ANY (ARRAY['source_match_auto'::text, 'manual'::text])) THEN
        RETURN NEW;
    END IF;
    SELECT
        c.property_offering_source_match_candidate_id,
        c.property_offering_source_match_confidence
    INTO candidate_id, confidence
    FROM public.property_offering_source_match_candidates c
    WHERE c.source_sale_listing_id = NEW.sale_listing_id
        AND c.source_property_offering_id = OLD.property_offering_id
        AND c.target_property_offering_id = NEW.property_offering_id
    ORDER BY c.property_offering_source_match_created_at DESC
    LIMIT 1;
    PERFORM public.fnc__merge_property_offerings(
        OLD.property_offering_id,
        NEW.property_offering_id,
        NEW.property_offering_source_link_method,
        NEW.property_offering_source_link_score,
        confidence,
        NEW.property_offering_source_link_reasons || jsonb_build_object('source_listing_id', NEW.sale_listing_id),
        candidate_id
    );
    RETURN NEW;
END;
$$;
DROP TRIGGER IF EXISTS trg_property_offering_source_merge ON public.property_offering_sources;
CREATE TRIGGER trg_property_offering_source_merge
AFTER UPDATE OF property_offering_id, property_offering_source_link_status, property_offering_source_link_method ON public.property_offering_sources
FOR EACH ROW
EXECUTE FUNCTION public.fnc__property_offering_source_merge_trigger();
INSERT INTO public.property_offering_merge_decisions (
    source_property_offering_id,
    target_property_offering_id,
    property_offering_source_match_candidate_id,
    property_offering_merge_decision_status,
    property_offering_merge_decision_method,
    property_offering_merge_decision_score,
    property_offering_merge_decision_confidence,
    property_offering_merge_decision_reasons,
    property_offering_merge_decision_decided_at
)
SELECT DISTINCT ON (c.source_property_offering_id, c.target_property_offering_id)
    c.source_property_offering_id,
    c.target_property_offering_id,
    c.property_offering_source_match_candidate_id,
    'accepted',
    'source_match_auto',
    c.property_offering_source_match_score,
    c.property_offering_source_match_confidence,
    c.property_offering_source_match_reasons || jsonb_build_object('backfilled_from_source_match_candidate', c.property_offering_source_match_candidate_id),
    c.property_offering_source_match_created_at
FROM public.property_offering_source_match_candidates c
WHERE c.property_offering_source_match_status = 'auto_linked'
    AND c.source_property_offering_id <> c.target_property_offering_id
ON CONFLICT DO NOTHING;
INSERT INTO public.property_offering_redirects (
    source_property_offering_id,
    target_property_offering_id,
    property_offering_merge_decision_id,
    property_offering_redirect_reason,
    property_offering_redirect_created_at
)
SELECT
    d.source_property_offering_id,
    public.fnc__resolve_property_offering_id(d.target_property_offering_id),
    d.property_offering_merge_decision_id,
    d.property_offering_merge_decision_method,
    d.property_offering_merge_decision_decided_at
FROM public.property_offering_merge_decisions d
WHERE d.property_offering_merge_decision_status = 'accepted'
    AND d.source_property_offering_id <> public.fnc__resolve_property_offering_id(d.target_property_offering_id)
ON CONFLICT (source_property_offering_id) DO NOTHING;
