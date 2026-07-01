DROP VIEW IF EXISTS public.source_housing_companies;
CREATE TABLE public.source_housing_companies (
    source_housing_company_id uuid PRIMARY KEY,
    provider text NOT NULL,
    source_kind text NOT NULL,
    native_id text,
    raw_table text NOT NULL,
    raw_id text NOT NULL,
    url text,
    first_seen_at timestamptz,
    last_seen_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX source_housing_companies_source_key
ON public.source_housing_companies (provider, source_kind, raw_table, raw_id);
CREATE INDEX idx_source_housing_companies_native
ON public.source_housing_companies (provider, source_kind, native_id) WHERE native_id IS NOT NULL;
CREATE OR REPLACE FUNCTION public.fnc__sync_source_housing_company_from_legacy()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        DELETE FROM public.source_housing_companies WHERE source_housing_company_id = OLD.housing_company_source_id;
        RETURN OLD;
    END IF;
    INSERT INTO public.source_housing_companies (
        source_housing_company_id,
        provider,
        source_kind,
        native_id,
        raw_table,
        raw_id,
        url,
        first_seen_at,
        last_seen_at,
        created_at,
        updated_at
    )
    VALUES (
        NEW.housing_company_source_id,
        NEW.housing_company_source_provider,
        NEW.housing_company_source_kind,
        NULLIF(NEW.housing_company_source_external_id, ''),
        NEW.housing_company_source_table,
        NEW.housing_company_source_id_value,
        NEW.housing_company_source_url,
        NEW.housing_company_source_first_seen_at,
        NEW.housing_company_source_last_seen_at,
        NEW.housing_company_source_created_at,
        NEW.housing_company_source_updated_at
    )
    ON CONFLICT (source_housing_company_id) DO UPDATE SET
        provider = EXCLUDED.provider,
        source_kind = EXCLUDED.source_kind,
        native_id = EXCLUDED.native_id,
        raw_table = EXCLUDED.raw_table,
        raw_id = EXCLUDED.raw_id,
        url = EXCLUDED.url,
        first_seen_at = EXCLUDED.first_seen_at,
        last_seen_at = EXCLUDED.last_seen_at,
        updated_at = EXCLUDED.updated_at;
    RETURN NEW;
END;
$$;
INSERT INTO public.source_housing_companies (
    source_housing_company_id,
    provider,
    source_kind,
    native_id,
    raw_table,
    raw_id,
    url,
    first_seen_at,
    last_seen_at,
    created_at,
    updated_at
)
SELECT
    housing_company_source_id,
    housing_company_source_provider,
    housing_company_source_kind,
    NULLIF(housing_company_source_external_id, ''),
    housing_company_source_table,
    housing_company_source_id_value,
    housing_company_source_url,
    housing_company_source_first_seen_at,
    housing_company_source_last_seen_at,
    housing_company_source_created_at,
    housing_company_source_updated_at
FROM public.housing_company_sources
ON CONFLICT (source_housing_company_id) DO UPDATE SET
    provider = EXCLUDED.provider,
    source_kind = EXCLUDED.source_kind,
    native_id = EXCLUDED.native_id,
    raw_table = EXCLUDED.raw_table,
    raw_id = EXCLUDED.raw_id,
    url = EXCLUDED.url,
    first_seen_at = EXCLUDED.first_seen_at,
    last_seen_at = EXCLUDED.last_seen_at,
    updated_at = EXCLUDED.updated_at;
DROP TRIGGER IF EXISTS trg__sync_source_housing_company_from_legacy ON public.housing_company_sources;
CREATE TRIGGER trg__sync_source_housing_company_from_legacy
AFTER INSERT OR UPDATE OR DELETE ON public.housing_company_sources
FOR EACH ROW
EXECUTE FUNCTION public.fnc__sync_source_housing_company_from_legacy();
