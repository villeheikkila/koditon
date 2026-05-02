CREATE TABLE public.sync_jobs (
    sync_job_id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    sync_job_provider text NOT NULL,
    sync_job_kind text NOT NULL,
    sync_job_entity_id text NOT NULL,
    sync_job_dedup_key text NOT NULL UNIQUE,
    sync_job_status text NOT NULL DEFAULT 'pending',
    sync_job_priority integer NOT NULL DEFAULT 0,
    sync_job_attempt_count integer NOT NULL DEFAULT 0,
    sync_job_max_attempts integer NOT NULL DEFAULT 3,
    sync_job_run_after timestamptz NOT NULL DEFAULT now(),
    sync_job_capacity_class text NOT NULL DEFAULT 'default',
    sync_job_payload jsonb NOT NULL DEFAULT '{}'::jsonb,
    sync_job_checkpoint jsonb,
    sync_job_result jsonb,
    sync_job_last_error text,
    sync_job_last_error_code text,
    sync_job_last_http_status integer,
    sync_job_last_pgmq_message_id bigint,
    sync_job_claim_token uuid,
    sync_job_created_at timestamptz NOT NULL DEFAULT now(),
    sync_job_updated_at timestamptz NOT NULL DEFAULT now(),
    sync_job_last_enqueued_at timestamptz,
    sync_job_last_started_at timestamptz,
    sync_job_last_finished_at timestamptz,
    CONSTRAINT sync_jobs_status_check CHECK (sync_job_status IN ('pending', 'in_progress', 'succeeded', 'failed', 'not_found', 'noop', 'skipped_lock')),
    CONSTRAINT sync_jobs_attempt_count_check CHECK (sync_job_attempt_count >= 0),
    CONSTRAINT sync_jobs_max_attempts_check CHECK (sync_job_max_attempts >= 1),
    CONSTRAINT sync_jobs_payload_object_check CHECK (jsonb_typeof(sync_job_payload) = 'object'),
    CONSTRAINT sync_jobs_checkpoint_object_check CHECK (sync_job_checkpoint IS NULL OR jsonb_typeof(sync_job_checkpoint) = 'object'),
    CONSTRAINT sync_jobs_result_object_check CHECK (sync_job_result IS NULL OR jsonb_typeof(sync_job_result) = 'object')
);
CREATE TABLE public.sync_job_attempts (
    sync_job_attempt_id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    sync_job_id uuid NOT NULL REFERENCES public.sync_jobs(sync_job_id) ON DELETE CASCADE,
    sync_job_attempt_queue_name text NOT NULL,
    sync_job_attempt_msg_id bigint,
    sync_job_attempt_no integer NOT NULL,
    sync_job_attempt_status text NOT NULL,
    sync_job_attempt_error_code text,
    sync_job_attempt_error_detail text,
    sync_job_attempt_payload_snapshot jsonb,
    sync_job_attempt_created_at timestamptz NOT NULL DEFAULT now(),
    sync_job_attempt_finished_at timestamptz,
    CONSTRAINT sync_job_attempts_status_check CHECK (sync_job_attempt_status IN ('running', 'succeeded', 'failed', 'retry', 'not_found', 'noop', 'skipped_lock')),
    CONSTRAINT sync_job_attempts_payload_snapshot_object_check CHECK (sync_job_attempt_payload_snapshot IS NULL OR jsonb_typeof(sync_job_attempt_payload_snapshot) = 'object')
);
CREATE INDEX idx_sync_jobs_provider_status_run_after ON public.sync_jobs(sync_job_provider, sync_job_status, sync_job_run_after);
CREATE INDEX idx_sync_jobs_kind_status_run_after ON public.sync_jobs(sync_job_kind, sync_job_status, sync_job_run_after);
CREATE INDEX idx_sync_jobs_capacity_status ON public.sync_jobs(sync_job_capacity_class, sync_job_status);
CREATE INDEX idx_sync_jobs_entity ON public.sync_jobs(sync_job_provider, sync_job_entity_id, sync_job_kind);
CREATE INDEX idx_sync_job_attempts_job_created_at_desc ON public.sync_job_attempts(sync_job_id, sync_job_attempt_created_at DESC);

CREATE OR REPLACE FUNCTION public.fnc__enqueue_sync_job(
    p_provider text,
    p_kind text,
    p_entity_id text,
    p_max_attempts integer DEFAULT 3,
    p_delay_seconds integer DEFAULT 0
) RETURNS uuid AS $$
DECLARE
    v_queue_name text;
    v_job public.sync_jobs%ROWTYPE;
    v_msg_id bigint;
    v_run_after timestamptz;
BEGIN
    v_queue_name := CASE p_provider
        WHEN 'frontdoor' THEN 'frontdoor'
        WHEN 'shortcut' THEN 'shortcut'
        WHEN 'prices' THEN 'prices'
        WHEN 'canonical' THEN 'prices'
        WHEN 'postal' THEN 'postal'
        ELSE NULL
    END;
    IF v_queue_name IS NULL THEN
        RAISE EXCEPTION 'unknown sync provider: %', p_provider;
    END IF;
    v_run_after := now() + make_interval(secs => greatest(coalesce(p_delay_seconds, 0), 0));
    INSERT INTO public.sync_jobs (
        sync_job_provider,
        sync_job_kind,
        sync_job_entity_id,
        sync_job_dedup_key,
        sync_job_status,
        sync_job_attempt_count,
        sync_job_max_attempts,
        sync_job_run_after,
        sync_job_capacity_class,
        sync_job_payload
    ) VALUES (
        p_provider,
        p_kind,
        p_entity_id,
        concat_ws(':', p_provider, p_kind, p_entity_id),
        'pending',
        0,
        greatest(coalesce(p_max_attempts, 3), 1),
        v_run_after,
        CASE p_provider
            WHEN 'frontdoor' THEN 'provider_frontdoor'
            WHEN 'shortcut' THEN CASE WHEN p_kind = 'shortcut_scraper_sync' THEN 'provider_shortcut_scraper' ELSE 'provider_shortcut_api' END
            WHEN 'prices' THEN 'provider_prices'
            WHEN 'canonical' THEN 'internal_db'
            WHEN 'postal' THEN 'provider_postal'
            ELSE 'default'
        END,
        '{}'::jsonb
    )
    ON CONFLICT (sync_job_dedup_key) DO UPDATE
    SET sync_job_provider = EXCLUDED.sync_job_provider,
        sync_job_kind = EXCLUDED.sync_job_kind,
        sync_job_entity_id = EXCLUDED.sync_job_entity_id,
        sync_job_status = 'pending',
        sync_job_attempt_count = 0,
        sync_job_max_attempts = EXCLUDED.sync_job_max_attempts,
        sync_job_run_after = EXCLUDED.sync_job_run_after,
        sync_job_capacity_class = EXCLUDED.sync_job_capacity_class,
        sync_job_payload = EXCLUDED.sync_job_payload,
        sync_job_checkpoint = NULL,
        sync_job_result = NULL,
        sync_job_last_error = NULL,
        sync_job_last_error_code = NULL,
        sync_job_last_http_status = NULL,
        sync_job_claim_token = NULL,
        sync_job_last_finished_at = NULL,
        sync_job_updated_at = now()
    WHERE sync_jobs.sync_job_status IN ('succeeded', 'failed', 'not_found', 'noop', 'skipped_lock')
       OR (sync_jobs.sync_job_status = 'pending' AND sync_jobs.sync_job_run_after > EXCLUDED.sync_job_run_after)
    RETURNING * INTO v_job;
    IF v_job.sync_job_id IS NULL THEN
        SELECT *
        INTO v_job
        FROM public.sync_jobs
        WHERE sync_job_dedup_key = concat_ws(':', p_provider, p_kind, p_entity_id);
        RETURN v_job.sync_job_id;
    END IF;
    v_msg_id := pgmq.send(
        v_queue_name,
        jsonb_build_object(
            'sync_job_id', v_job.sync_job_id,
            'entity_id', v_job.sync_job_entity_id,
            'task_type', v_job.sync_job_kind
        ),
        greatest(coalesce(p_delay_seconds, 0), 0)
    );
    UPDATE public.sync_jobs
    SET sync_job_last_pgmq_message_id = v_msg_id,
        sync_job_last_enqueued_at = now(),
        sync_job_updated_at = now()
    WHERE sync_job_id = v_job.sync_job_id;
    RETURN v_job.sync_job_id;
END;
$$ LANGUAGE plpgsql;

SELECT cron.schedule(
    'trigger-frontdoor-sitemap-sync',
    '0 1 * * *',
    $$SELECT public.fnc__enqueue_sync_job('frontdoor', 'frontdoor_sitemap_sync', 'frontdoor:sitemap', 3, 0)$$
) WHERE NOT EXISTS (
    SELECT 1 FROM cron.job WHERE jobname = 'trigger-frontdoor-sitemap-sync'
);

SELECT cron.schedule(
    'trigger-shortcut-sitemap-sync',
    '30 1 * * *',
    $$SELECT public.fnc__enqueue_sync_job('shortcut', 'shortcut_sitemap_sync', 'shortcut:sitemap', 3, 0)$$
) WHERE NOT EXISTS (
    SELECT 1 FROM cron.job WHERE jobname = 'trigger-shortcut-sitemap-sync'
);

SELECT cron.schedule(
    'trigger-prices-cities-init',
    '0 2 * * *',
    $$SELECT public.fnc__enqueue_sync_job('prices', 'prices_cities_init', 'prices:cities', 3, 0)$$
) WHERE NOT EXISTS (
    SELECT 1 FROM cron.job WHERE jobname = 'trigger-prices-cities-init'
);

SELECT cron.schedule(
    'trigger-prices-neighborhood-postal-code-sync',
    '0 3 * * 0',
    $$SELECT public.fnc__enqueue_sync_job('prices', 'prices_neighborhood_postal_code_sync', 'prices:neighborhood_postal_codes', 3, 0)$$
) WHERE NOT EXISTS (
    SELECT 1 FROM cron.job WHERE jobname = 'trigger-prices-neighborhood-postal-code-sync'
);
