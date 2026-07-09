DELETE FROM public.source_listing_match_candidates
WHERE match_status = 'proposed'
    AND (
        COALESCE((match_reasons ->> 'source_pair_count')::integer, (match_reasons ->> 'source_compatible_pair_count')::integer, 999999) > 5
        OR COALESCE((match_reasons ->> 'matched_pair_count')::integer, (match_reasons ->> 'matched_compatible_pair_count')::integer, 999999) > 5
    );
