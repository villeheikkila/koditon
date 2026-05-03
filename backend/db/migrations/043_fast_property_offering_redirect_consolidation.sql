CREATE OR REPLACE FUNCTION public.fnc__consolidate_property_offering_redirect_batch(batch_limit integer DEFAULT 1000)
RETURNS integer
LANGUAGE plpgsql
AS $$
DECLARE
    moved_sources integer := 0;
    moved_transactions integer := 0;
BEGIN
    CREATE TEMP TABLE tmp_property_offering_source_redirect_moves ON COMMIT DROP AS
    SELECT
        pos.property_offering_source_id,
        r.target_property_offering_id,
        r.property_offering_redirect_reason
    FROM public.property_offering_sources pos
    JOIN public.property_offering_redirects r ON r.source_property_offering_id = pos.property_offering_id
    WHERE pos.property_offering_source_link_status <> 'rejected'
    LIMIT GREATEST(batch_limit, 1);
    ALTER TABLE public.property_offering_sources DISABLE TRIGGER trg_property_offering_source_merge;
    UPDATE public.property_offering_sources pos
    SET
        property_offering_id = moves.target_property_offering_id,
        property_offering_source_link_status = 'confirmed',
        property_offering_source_link_method = CASE
            WHEN pos.property_offering_source_link_method = 'manual' THEN 'manual'
            ELSE moves.property_offering_redirect_reason
        END,
        property_offering_source_updated_at = now()
    FROM tmp_property_offering_source_redirect_moves moves
    WHERE pos.property_offering_source_id = moves.property_offering_source_id;
    GET DIAGNOSTICS moved_sources = ROW_COUNT;
    ALTER TABLE public.property_offering_sources ENABLE TRIGGER trg_property_offering_source_merge;
    CREATE TEMP TABLE tmp_property_offering_transaction_redirect_moves ON COMMIT DROP AS
    SELECT
        pot.property_offering_transaction_id,
        r.target_property_offering_id,
        pot.prices_transaction_id,
        pot.property_offering_transaction_link_status,
        pot.property_offering_transaction_link_method,
        pot.property_offering_transaction_link_score,
        pot.property_offering_transaction_link_reasons
    FROM public.property_offering_transactions pot
    JOIN public.property_offering_redirects r ON r.source_property_offering_id = pot.property_offering_id
    LIMIT GREATEST(batch_limit, 1);
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
        moves.target_property_offering_id,
        moves.prices_transaction_id,
        moves.property_offering_transaction_link_status,
        moves.property_offering_transaction_link_method,
        moves.property_offering_transaction_link_score,
        moves.property_offering_transaction_link_reasons,
        now()
    FROM tmp_property_offering_transaction_redirect_moves moves
    ON CONFLICT (property_offering_id, prices_transaction_id) DO UPDATE SET
        property_offering_transaction_link_score = GREATEST(public.property_offering_transactions.property_offering_transaction_link_score, EXCLUDED.property_offering_transaction_link_score),
        property_offering_transaction_link_reasons = public.property_offering_transactions.property_offering_transaction_link_reasons || EXCLUDED.property_offering_transaction_link_reasons,
        property_offering_transaction_updated_at = now();
    DELETE FROM public.property_offering_transactions pot
    USING tmp_property_offering_transaction_redirect_moves moves
    WHERE pot.property_offering_transaction_id = moves.property_offering_transaction_id;
    GET DIAGNOSTICS moved_transactions = ROW_COUNT;
    RETURN moved_sources + moved_transactions;
END;
$$;
