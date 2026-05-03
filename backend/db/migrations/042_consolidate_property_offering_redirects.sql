CREATE OR REPLACE FUNCTION public.fnc__consolidate_property_offering_redirect_batch(batch_limit integer DEFAULT 1000)
RETURNS integer
LANGUAGE plpgsql
AS $$
DECLARE
    moved_sources integer := 0;
    moved_transactions integer := 0;
BEGIN
    WITH source_moves AS (
        SELECT
            pos.property_offering_source_id,
            r.target_property_offering_id,
            r.property_offering_redirect_reason
        FROM public.property_offering_sources pos
        JOIN public.property_offering_redirects r ON r.source_property_offering_id = pos.property_offering_id
        WHERE pos.property_offering_source_link_status <> 'rejected'
        ORDER BY pos.property_offering_source_updated_at NULLS FIRST, pos.property_offering_source_id
        LIMIT GREATEST(batch_limit, 1)
    ),
    moved AS (
        UPDATE public.property_offering_sources pos
        SET
            property_offering_id = source_moves.target_property_offering_id,
            property_offering_source_link_status = 'confirmed',
            property_offering_source_link_method = CASE
                WHEN pos.property_offering_source_link_method = 'manual' THEN 'manual'
                ELSE source_moves.property_offering_redirect_reason
            END,
            property_offering_source_updated_at = now()
        FROM source_moves
        WHERE pos.property_offering_source_id = source_moves.property_offering_source_id
        RETURNING 1
    )
    SELECT count(*)::integer INTO moved_sources FROM moved;
    WITH transaction_moves AS (
        SELECT
            pot.property_offering_transaction_id,
            r.target_property_offering_id,
            r.source_property_offering_id,
            r.property_offering_merge_decision_id,
            pot.prices_transaction_id,
            pot.property_offering_transaction_link_status,
            pot.property_offering_transaction_link_method,
            pot.property_offering_transaction_link_score,
            pot.property_offering_transaction_link_reasons
        FROM public.property_offering_transactions pot
        JOIN public.property_offering_redirects r ON r.source_property_offering_id = pot.property_offering_id
        ORDER BY pot.property_offering_transaction_updated_at NULLS FIRST, pot.property_offering_transaction_id
        LIMIT GREATEST(batch_limit, 1)
    ),
    inserted AS (
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
            transaction_moves.target_property_offering_id,
            transaction_moves.prices_transaction_id,
            transaction_moves.property_offering_transaction_link_status,
            transaction_moves.property_offering_transaction_link_method,
            transaction_moves.property_offering_transaction_link_score,
            transaction_moves.property_offering_transaction_link_reasons || jsonb_build_object('redirect_consolidated_from_property_offering_id', transaction_moves.source_property_offering_id, 'merge_decision_id', transaction_moves.property_offering_merge_decision_id),
            now()
        FROM transaction_moves
        ON CONFLICT (property_offering_id, prices_transaction_id) DO UPDATE SET
            property_offering_transaction_link_score = GREATEST(public.property_offering_transactions.property_offering_transaction_link_score, EXCLUDED.property_offering_transaction_link_score),
            property_offering_transaction_link_reasons = public.property_offering_transactions.property_offering_transaction_link_reasons || EXCLUDED.property_offering_transaction_link_reasons,
            property_offering_transaction_updated_at = now()
        RETURNING 1
    ),
    deleted AS (
        DELETE FROM public.property_offering_transactions pot
        USING transaction_moves
        WHERE pot.property_offering_transaction_id = transaction_moves.property_offering_transaction_id
        RETURNING 1
    )
    SELECT count(*)::integer INTO moved_transactions FROM deleted;
    RETURN moved_sources + moved_transactions;
END;
$$;
