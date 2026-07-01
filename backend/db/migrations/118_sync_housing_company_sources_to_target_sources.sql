CREATE OR REPLACE FUNCTION public.fnc__sync_target_source_from_housing_company_source()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        DELETE FROM public.target_sources
        WHERE target_type = 'housing_company'
            AND source_type = 'source_housing_company'
            AND source_id = OLD.housing_company_source_id;
        RETURN OLD;
    END IF;
    INSERT INTO public.target_sources (
        target_type,
        target_id,
        source_type,
        source_id,
        link_status,
        link_method,
        link_score,
        link_reasons,
        first_seen_at,
        last_seen_at,
        created_at,
        updated_at
    )
    VALUES (
        'housing_company',
        NEW.housing_company_id,
        'source_housing_company',
        NEW.housing_company_source_id,
        NEW.housing_company_source_link_status,
        CASE WHEN NEW.housing_company_source_link_method = ANY (ARRAY['sync_auto','source_match_auto','document_match_auto','manual','backfill_auto']) THEN NEW.housing_company_source_link_method ELSE 'backfill_auto' END,
        NEW.housing_company_source_link_score,
        NEW.housing_company_source_link_reasons,
        NEW.housing_company_source_first_seen_at,
        NEW.housing_company_source_last_seen_at,
        NEW.housing_company_source_created_at,
        NEW.housing_company_source_updated_at
    )
    ON CONFLICT (target_type, target_id, source_type, source_id) DO UPDATE SET
        link_status = EXCLUDED.link_status,
        link_method = EXCLUDED.link_method,
        link_score = EXCLUDED.link_score,
        link_reasons = public.target_sources.link_reasons || EXCLUDED.link_reasons,
        first_seen_at = LEAST(COALESCE(public.target_sources.first_seen_at, EXCLUDED.first_seen_at), COALESCE(EXCLUDED.first_seen_at, public.target_sources.first_seen_at)),
        last_seen_at = GREATEST(COALESCE(public.target_sources.last_seen_at, EXCLUDED.last_seen_at), COALESCE(EXCLUDED.last_seen_at, public.target_sources.last_seen_at)),
        updated_at = now();
    RETURN NEW;
END;
$$;
INSERT INTO public.target_sources (
    target_type,
    target_id,
    source_type,
    source_id,
    link_status,
    link_method,
    link_score,
    link_reasons,
    first_seen_at,
    last_seen_at,
    created_at,
    updated_at
)
SELECT
    'housing_company',
    housing_company_id,
    'source_housing_company',
    housing_company_source_id,
    housing_company_source_link_status,
    CASE WHEN housing_company_source_link_method = ANY (ARRAY['sync_auto','source_match_auto','document_match_auto','manual','backfill_auto']) THEN housing_company_source_link_method ELSE 'backfill_auto' END,
    housing_company_source_link_score,
    housing_company_source_link_reasons,
    housing_company_source_first_seen_at,
    housing_company_source_last_seen_at,
    housing_company_source_created_at,
    housing_company_source_updated_at
FROM public.housing_company_sources
ON CONFLICT (target_type, target_id, source_type, source_id) DO UPDATE SET
    link_status = EXCLUDED.link_status,
    link_method = EXCLUDED.link_method,
    link_score = EXCLUDED.link_score,
    link_reasons = public.target_sources.link_reasons || EXCLUDED.link_reasons,
    first_seen_at = LEAST(COALESCE(public.target_sources.first_seen_at, EXCLUDED.first_seen_at), COALESCE(EXCLUDED.first_seen_at, public.target_sources.first_seen_at)),
    last_seen_at = GREATEST(COALESCE(public.target_sources.last_seen_at, EXCLUDED.last_seen_at), COALESCE(EXCLUDED.last_seen_at, public.target_sources.last_seen_at)),
    updated_at = now();
DROP TRIGGER IF EXISTS trg__sync_target_source_from_housing_company_source ON public.housing_company_sources;
CREATE TRIGGER trg__sync_target_source_from_housing_company_source
AFTER INSERT OR UPDATE OR DELETE ON public.housing_company_sources
FOR EACH ROW
EXECUTE FUNCTION public.fnc__sync_target_source_from_housing_company_source();
