DO $$
BEGIN
    IF to_regclass('cron.job') IS NOT NULL THEN
        PERFORM cron.unschedule(jobid)
        FROM cron.job
        WHERE jobname IN (
            'trigger-frontdoor-sitemap-sync',
            'trigger-frontdoor-daily-syncs',
            'trigger-shortcut-sitemap-sync',
            'trigger-shortcut-daily-scraper-syncs',
            'trigger-shortcut-daily-api-syncs',
            'trigger-prices-cities-init',
            'trigger-prices-daily-syncs',
            'trigger-prices-sync-all',
            'trigger-prices-neighborhood-postal-code-sync',
            'trigger-prices-sale-listing-match-fanout',
            'trigger-canonical-source-match-fanout'
        );
    END IF;
END;
$$;

DROP FUNCTION IF EXISTS public.fnc__enqueue_sync_job(text, text, text, integer, integer);

UPDATE public.sync_job_attempts
SET sync_job_attempt_status = 'failed',
    sync_job_attempt_error_code = 'absurd_migration_drained',
    sync_job_attempt_error_detail = 'legacy sync_jobs worker stack was replaced by Absurd workflows; this running attempt was intentionally drained',
    sync_job_attempt_finished_at = now()
WHERE sync_job_attempt_status = 'running';

UPDATE public.sync_jobs
SET sync_job_status = 'failed',
    sync_job_last_error_code = 'absurd_migration_drained',
    sync_job_last_error = 'legacy sync_jobs worker stack was replaced by Absurd workflows; enqueue the corresponding Absurd task if this work is still needed',
    sync_job_claim_token = NULL,
    sync_job_last_finished_at = now(),
    sync_job_updated_at = now()
WHERE sync_job_status IN ('pending', 'in_progress');

ALTER TABLE public.sync_jobs
    DROP COLUMN IF EXISTS sync_job_last_pgmq_message_id;

ALTER TABLE public.sync_job_attempts
    DROP COLUMN IF EXISTS sync_job_attempt_msg_id;
