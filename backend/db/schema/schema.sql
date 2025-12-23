-- Enable extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS pgmq;
CREATE EXTENSION IF NOT EXISTS pg_cron;

-- ============================================
-- Public Schema Tables
-- ============================================

CREATE TABLE public.shortcut_tokens (
    shortcut_tokens_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shortcut_tokens_cuid TEXT NOT NULL,
    shortcut_tokens_token TEXT NOT NULL,
    shortcut_tokens_loaded TEXT NOT NULL,
    shortcut_tokens_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    shortcut_tokens_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    shortcut_tokens_expires_at TIMESTAMPTZ NOT NULL,
    UNIQUE(shortcut_tokens_cuid)
);

CREATE INDEX idx_shortcut_tokens_expires_at ON public.shortcut_tokens(shortcut_tokens_expires_at DESC);
CREATE INDEX idx_shortcut_tokens_cuid ON public.shortcut_tokens(shortcut_tokens_cuid);

CREATE TABLE public.prices_cities (
    prices_cities_id           uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    prices_cities_name         text NOT NULL UNIQUE,
    prices_cities_created_at   timestamptz NOT NULL DEFAULT now(),
    prices_cities_updated_at   timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE public.prices_postal_codes (
    prices_postal_codes_id         uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    prices_postal_codes_code       text NOT NULL UNIQUE,
    prices_postal_codes_city_id    uuid NOT NULL REFERENCES public.prices_cities(prices_cities_id),
    prices_postal_codes_created_at timestamptz NOT NULL DEFAULT now(),
    prices_postal_codes_updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE public.prices_neighborhoods (
    prices_neighborhoods_id           uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    prices_neighborhoods_name         text NOT NULL,
    prices_neighborhoods_city_id      uuid NOT NULL REFERENCES public.prices_cities(prices_cities_id),
    prices_neighborhoods_postal_code_id uuid REFERENCES public.prices_postal_codes(prices_postal_codes_id),
    prices_neighborhoods_created_at   timestamptz NOT NULL DEFAULT now(),
    prices_neighborhoods_updated_at   timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT prices_neighborhoods_name_city_unique UNIQUE (prices_neighborhoods_name, prices_neighborhoods_city_id)
);

CREATE TABLE public.prices_transactions (
    prices_transactions_id                          uuid             PRIMARY KEY DEFAULT uuid_generate_v4(),
    prices_transactions_neighborhood                text             NOT NULL,
    prices_transactions_description                 text             NOT NULL,
    prices_transactions_type                        text             NOT NULL,
    prices_transactions_area                        double precision NOT NULL,
    prices_transactions_price                       integer          NOT NULL,
    prices_transactions_price_per_square_meter      integer          NOT NULL,
    prices_transactions_build_year                  integer          NOT NULL,
    prices_transactions_floor                       text,
    prices_transactions_elevator                    boolean          NOT NULL,
    prices_transactions_condition                   text,
    prices_transactions_plot                        text,
    prices_transactions_energy_class                text,
    prices_transactions_period_identifier           text             NOT NULL,
    prices_transactions_created_at                  timestamptz      NOT NULL DEFAULT now(),
    prices_transactions_updated_at                  timestamptz      NOT NULL DEFAULT now(),
    prices_transactions_category                    text             NOT NULL,
    prices_neighborhoods_id                         uuid             REFERENCES public.prices_neighborhoods(prices_neighborhoods_id),
    CONSTRAINT prices_transactions_unique_key UNIQUE (
        prices_neighborhoods_id,
        prices_transactions_description,
        prices_transactions_type,
        prices_transactions_area,
        prices_transactions_price,
        prices_transactions_price_per_square_meter,
        prices_transactions_build_year,
        prices_transactions_floor,
        prices_transactions_elevator,
        prices_transactions_condition,
        prices_transactions_plot,
        prices_transactions_energy_class,
        prices_transactions_category,
        prices_transactions_period_identifier
    )
);

CREATE INDEX idx_prices_transactions_period_identifier
    ON public.prices_transactions(prices_transactions_period_identifier);

-- ============================================
-- Task Queue Schema
-- ============================================

CREATE SCHEMA IF NOT EXISTS task_queue;

-- ============================================
-- PGMQ Queues
-- ============================================

SELECT pgmq.create('tasks');
SELECT pgmq.create('tasks_dlq');

SELECT pgmq.enable_notify_insert('tasks');

-- ============================================
-- ENTITY REGISTRY
-- ============================================

CREATE TABLE task_queue.entity_registry (
    entity_id TEXT NOT NULL PRIMARY KEY,
    entity_type TEXT NOT NULL DEFAULT 'unknown',
    status TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'stopped')),
    scheduling_strategy TEXT NOT NULL DEFAULT 'manual'
        CHECK (scheduling_strategy IN ('daily', 'manual', 'on_demand', 'cron')),
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_entity_registry_status ON task_queue.entity_registry(status);
CREATE INDEX idx_entity_registry_entity_type ON task_queue.entity_registry(entity_type);
CREATE INDEX idx_entity_registry_scheduling_strategy ON task_queue.entity_registry(scheduling_strategy);
CREATE INDEX idx_entity_registry_schedulable ON task_queue.entity_registry(scheduling_strategy, status)
    WHERE status = 'active' AND scheduling_strategy IN ('daily', 'cron');

-- ============================================
-- TASK TYPE TO ENTITY TYPE MAPPING
-- ============================================

CREATE TABLE task_queue.task_type_entity_type_mapping (
    task_type TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    PRIMARY KEY (task_type, entity_type)
);

INSERT INTO task_queue.task_type_entity_type_mapping (task_type, entity_type) VALUES
    ('frontdoor_sync', 'frontdoor_ad'),
    ('frontdoor_sync', 'frontdoor_building'),
    ('frontdoor_sitemap_sync', 'frontdoor_sitemap'),
    ('shortcut_scraper_sync', 'shortcut_building'),
    ('shortcut_api_sync', 'shortcut_ad'),
    ('shortcut_sitemap_sync', 'shortcut_sitemap')
ON CONFLICT (task_type, entity_type) DO NOTHING;

-- ============================================
-- TASK TABLE (Source of Truth)
-- ============================================

CREATE TABLE task_queue.task (
    task_id BIGSERIAL PRIMARY KEY,
    entity_id TEXT NOT NULL
        REFERENCES task_queue.entity_registry(entity_id) ON DELETE CASCADE,
    task_type TEXT NOT NULL DEFAULT 'frontdoor_sync',
    status TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'processing', 'completed', 'failed', 'stopped')),
    attempt INT NOT NULL DEFAULT 0,
    max_attempts INT NOT NULL DEFAULT 3,
    last_error TEXT,
    worker_id TEXT,
    scheduled_for TIMESTAMPTZ NOT NULL,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    run_on DATE,
    queue_message_id BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for common queries
CREATE UNIQUE INDEX uniq_task_daily
    ON task_queue.task(entity_id, task_type, run_on)
    WHERE run_on IS NOT NULL;

CREATE INDEX idx_task_entity ON task_queue.task(entity_id);
CREATE INDEX idx_task_status ON task_queue.task(status);
CREATE INDEX idx_task_worker ON task_queue.task(worker_id) WHERE status = 'processing';
CREATE INDEX idx_task_scheduled ON task_queue.task(scheduled_for) WHERE status = 'pending';
CREATE INDEX idx_task_updated ON task_queue.task(updated_at);
CREATE INDEX idx_task_run_on ON task_queue.task(run_on);

-- ============================================
-- FUNCTIONS
-- ============================================

-- ============================================
-- REGISTER ENTITY FOR SYNCING
-- ============================================
CREATE OR REPLACE FUNCTION task_queue.fnc__register_entity(
    p_entity_id TEXT,
    p_entity_type TEXT,
    p_status TEXT DEFAULT 'active',
    p_scheduling_strategy TEXT DEFAULT 'manual',
    p_metadata JSONB DEFAULT '{}'::jsonb
) RETURNS void AS $$
BEGIN
    INSERT INTO task_queue.entity_registry (entity_id, entity_type, status, scheduling_strategy, metadata)
    VALUES (p_entity_id, p_entity_type, COALESCE(p_status, 'active'), COALESCE(p_scheduling_strategy, 'manual'), COALESCE(p_metadata, '{}'::jsonb))
    ON CONFLICT (entity_id) DO UPDATE
    SET status = EXCLUDED.status,
        entity_type = EXCLUDED.entity_type,
        scheduling_strategy = EXCLUDED.scheduling_strategy,
        metadata = EXCLUDED.metadata,
        updated_at = NOW();
END;
$$ LANGUAGE plpgsql;

-- ============================================
-- BULK REGISTER ENTITIES
-- ============================================
CREATE OR REPLACE FUNCTION task_queue.fnc__register_entities(
    p_entity_ids TEXT[],
    p_entity_type TEXT,
    p_scheduling_strategy TEXT DEFAULT 'daily'
) RETURNS INT AS $$
DECLARE
    v_count INT;
BEGIN
    INSERT INTO task_queue.entity_registry (entity_id, entity_type, status, scheduling_strategy)
    SELECT unnest(p_entity_ids), p_entity_type, 'active', p_scheduling_strategy
    ON CONFLICT (entity_id) DO UPDATE
    SET status = 'active',
        entity_type = EXCLUDED.entity_type,
        scheduling_strategy = EXCLUDED.scheduling_strategy,
        updated_at = NOW();

    GET DIAGNOSTICS v_count = ROW_COUNT;
    RETURN v_count;
END;
$$ LANGUAGE plpgsql;

-- ============================================
-- ENQUEUE TASK (idempotent per task_id)
-- ============================================
CREATE OR REPLACE FUNCTION task_queue.fnc__enqueue_task(
    p_task_id BIGINT
) RETURNS BIGINT AS $$
DECLARE
    v_task RECORD;
    v_msg_id BIGINT;
BEGIN
    SELECT
        task_id,
        entity_id,
        attempt,
        scheduled_for,
        status,
        queue_message_id
    INTO v_task
    FROM task_queue.task
    WHERE task_id = p_task_id
    FOR UPDATE;

    IF NOT FOUND THEN
        RAISE EXCEPTION 'task_id % not found', p_task_id;
    END IF;

    IF v_task.status <> 'pending' THEN
        RETURN NULL;
    END IF;

    IF v_task.queue_message_id IS NOT NULL THEN
        RETURN v_task.queue_message_id;
    END IF;

    v_msg_id := pgmq.send(
        'tasks',
        jsonb_build_object(
            'task_id', v_task.task_id,
            'entity_id', v_task.entity_id,
            'attempt', v_task.attempt
        ),
        v_task.scheduled_for
    );

    UPDATE task_queue.task
    SET queue_message_id = v_msg_id,
        updated_at = NOW()
    WHERE task_id = v_task.task_id;

    RETURN v_msg_id;
END;
$$ LANGUAGE plpgsql;

-- ============================================
-- UPDATE TASK STATE
-- Called by Go to update observability table
-- ============================================
CREATE OR REPLACE FUNCTION task_queue.fnc__update_task_state(
    p_task_id BIGINT,
    p_status TEXT,
    p_attempt INT DEFAULT NULL,
    p_error TEXT DEFAULT NULL,
    p_worker_id TEXT DEFAULT NULL,
    p_scheduled_for TIMESTAMPTZ DEFAULT NULL
) RETURNS void AS $$
DECLARE
    v_attempt INT;
BEGIN
    SELECT attempt
    INTO v_attempt
    FROM task_queue.task
    WHERE task_id = p_task_id
    FOR UPDATE;

    IF NOT FOUND THEN
        RAISE EXCEPTION 'task_id % not found', p_task_id;
    END IF;

    IF p_attempt IS NULL THEN
        IF p_status = 'processing' THEN
            v_attempt := v_attempt + 1;
        END IF;
    ELSE
        v_attempt := p_attempt;
    END IF;

    UPDATE task_queue.task
    SET status = p_status,
        attempt = v_attempt,
        last_error = CASE
            WHEN p_error IS NOT NULL THEN p_error
            WHEN p_status = 'completed' THEN NULL
            ELSE last_error
        END,
        worker_id = CASE
            WHEN p_status = 'processing' THEN p_worker_id
            WHEN p_status = 'pending' THEN NULL
            ELSE worker_id
        END,
        scheduled_for = COALESCE(p_scheduled_for, scheduled_for),
        started_at = CASE
            WHEN p_status = 'processing' AND started_at IS NULL THEN NOW()
            WHEN p_status = 'pending' THEN NULL
            ELSE started_at
        END,
        completed_at = CASE
            WHEN p_status = 'pending' THEN NULL
            WHEN p_status IN ('completed', 'failed', 'stopped')
                AND completed_at IS NULL THEN NOW()
            ELSE completed_at
        END,
        queue_message_id = CASE
            WHEN p_status = 'pending' THEN NULL
            ELSE queue_message_id
        END,
        updated_at = NOW()
    WHERE task_id = p_task_id;
END;
$$ LANGUAGE plpgsql;

-- ============================================
-- SCHEDULE DAILY SYNC WITH JITTER
-- Called by pg_cron at midnight
-- ============================================
CREATE OR REPLACE FUNCTION task_queue.fnc__schedule_daily_syncs(
    p_task_type TEXT DEFAULT 'frontdoor_sync'
) RETURNS INT AS $$
DECLARE
    v_total_new INT;
    v_interval_seconds FLOAT;
    v_entity RECORD;
    v_index INT := 0;
    v_scheduled_time TIMESTAMPTZ;
    v_base_time TIMESTAMPTZ;
    v_jitter_seconds FLOAT;
    v_scheduled_seconds FLOAT;
    v_run_on DATE;
    v_task_id BIGINT;
    v_task RECORD;
BEGIN
    v_base_time := DATE_TRUNC('day', NOW() + INTERVAL '1 day');
    v_run_on := v_base_time::DATE;

    -- Count entities with 'daily' scheduling strategy that match the task type
    SELECT COUNT(*) INTO v_total_new
    FROM task_queue.entity_registry e
    WHERE e.status = 'active'
      AND e.scheduling_strategy = 'daily'
      AND EXISTS (
          SELECT 1
          FROM task_queue.task_type_entity_type_mapping m
          WHERE m.task_type = p_task_type
            AND m.entity_type = e.entity_type
      )
      AND NOT EXISTS (
          SELECT 1
          FROM task_queue.task t
          WHERE t.entity_id = e.entity_id
            AND t.task_type = p_task_type
            AND t.run_on = v_run_on
      );

    IF v_total_new > 0 THEN
        v_interval_seconds := 86400.0 / v_total_new;

        FOR v_entity IN
            SELECT e.entity_id
            FROM task_queue.entity_registry e
            WHERE e.status = 'active'
              AND e.scheduling_strategy = 'daily'
              AND EXISTS (
                  SELECT 1
                  FROM task_queue.task_type_entity_type_mapping m
                  WHERE m.task_type = p_task_type
                    AND m.entity_type = e.entity_type
              )
              AND NOT EXISTS (
                  SELECT 1
                  FROM task_queue.task t
                  WHERE t.entity_id = e.entity_id
                    AND t.task_type = p_task_type
                    AND t.run_on = v_run_on
              )
            ORDER BY e.entity_id
        LOOP
            v_jitter_seconds := (RANDOM() - 0.5) * 0.6 * v_interval_seconds;
            v_scheduled_seconds := GREATEST(0, v_index * v_interval_seconds + v_jitter_seconds);
            v_scheduled_time := v_base_time + make_interval(secs => v_scheduled_seconds);

            INSERT INTO task_queue.task (
                entity_id,
                task_type,
                status,
                attempt,
                scheduled_for,
                run_on
            )
            VALUES (
                v_entity.entity_id,
                p_task_type,
                'pending',
                0,
                v_scheduled_time,
                v_run_on
            )
            ON CONFLICT (entity_id, task_type, run_on) WHERE run_on IS NOT NULL DO UPDATE
            SET scheduled_for = COALESCE(task_queue.task.scheduled_for, EXCLUDED.scheduled_for),
                updated_at = NOW()
            RETURNING task_id INTO v_task_id;

            PERFORM task_queue.fnc__enqueue_task(v_task_id);

            v_index := v_index + 1;
        END LOOP;
    END IF;

    -- Enqueue any pending tasks that don't have a queue message ID
    FOR v_task IN
        SELECT task_id
        FROM task_queue.task
        WHERE task_type = p_task_type
          AND run_on = v_run_on
          AND status = 'pending'
          AND queue_message_id IS NULL
        ORDER BY task_id
    LOOP
        PERFORM task_queue.fnc__enqueue_task(v_task.task_id);
    END LOOP;

    RETURN v_total_new;
END;
$$ LANGUAGE plpgsql;

-- ============================================
-- REQUEUE STUCK TASKS (processing > 15 minutes)
-- ============================================
CREATE OR REPLACE FUNCTION task_queue.fnc__requeue_stuck_tasks() RETURNS INT AS $$
DECLARE
    v_task RECORD;
    v_requeued INT := 0;
BEGIN
    FOR v_task IN
        SELECT task_id, attempt, max_attempts
        FROM task_queue.task
        WHERE status = 'processing'
          AND updated_at < NOW() - INTERVAL '15 minutes'
        FOR UPDATE SKIP LOCKED
    LOOP
        IF v_task.attempt + 1 >= v_task.max_attempts THEN
            UPDATE task_queue.task
            SET status = 'failed',
                attempt = LEAST(attempt + 1, max_attempts),
                worker_id = NULL,
                completed_at = NOW(),
                updated_at = NOW(),
                last_error = COALESCE(last_error, 'auto-failed after stuck timeout')
            WHERE task_id = v_task.task_id;
        ELSE
            UPDATE task_queue.task
            SET status = 'pending',
                attempt = attempt + 1,
                worker_id = NULL,
                scheduled_for = NOW(),
                started_at = NULL,
                updated_at = NOW(),
                queue_message_id = NULL
            WHERE task_id = v_task.task_id;

            PERFORM task_queue.fnc__enqueue_task(v_task.task_id);
            v_requeued := v_requeued + 1;
        END IF;
    END LOOP;

    RETURN v_requeued;
END;
$$ LANGUAGE plpgsql;

-- ============================================
-- VIEWS (Observability)
-- ============================================

-- ============================================
-- SYNC STATUS SUMMARY
-- ============================================
CREATE OR REPLACE VIEW task_queue.fnc__status_summary AS
SELECT
    COUNT(*) FILTER (WHERE status = 'pending') AS pending,
    COUNT(*) FILTER (WHERE status = 'processing') AS processing,
    COUNT(*) FILTER (WHERE status = 'completed') AS completed,
    COUNT(*) FILTER (WHERE status = 'failed') AS failed,
    COUNT(*) FILTER (WHERE status = 'stopped') AS stopped,
    COUNT(*) AS total,
    ROUND(100.0 * COUNT(*) FILTER (WHERE status = 'completed') / NULLIF(COUNT(*), 0), 2) AS success_rate_pct
FROM task_queue.task;

-- ============================================
-- ACTIVE WORKERS
-- ============================================
CREATE OR REPLACE VIEW task_queue.fnc__active_workers AS
SELECT
    worker_id,
    COUNT(*) AS active_tasks,
    MIN(started_at) AS oldest_task_started,
    EXTRACT(EPOCH FROM (NOW() - MIN(started_at)))::INT AS oldest_task_age_seconds
FROM task_queue.task
WHERE status = 'processing'
  AND worker_id IS NOT NULL
GROUP BY worker_id
ORDER BY oldest_task_started;

-- ============================================
-- RECENT FAILURES
-- ============================================
CREATE OR REPLACE VIEW task_queue.fnc__recent_failures AS
SELECT
    task_id,
    entity_id,
    status,
    attempt,
    max_attempts,
    last_error,
    completed_at AS failed_at
FROM task_queue.task
WHERE status IN ('failed', 'stopped')
ORDER BY completed_at DESC
LIMIT 100;

-- ============================================
-- STUCK TASKS (processing > 10 minutes)
-- ============================================
CREATE OR REPLACE VIEW task_queue.fnc__stuck_tasks AS
SELECT
    task_id,
    entity_id,
    worker_id,
    started_at,
    EXTRACT(EPOCH FROM (NOW() - started_at))::INT AS stuck_seconds,
    updated_at
FROM task_queue.task
WHERE status = 'processing'
  AND updated_at < NOW() - INTERVAL '10 minutes'
ORDER BY started_at;

-- ============================================
-- DAILY PROGRESS
-- ============================================
CREATE OR REPLACE VIEW task_queue.fnc__daily_progress AS
SELECT
    COUNT(*) FILTER (WHERE status = 'completed' AND completed_at >= CURRENT_DATE) AS completed_today,
    COUNT(*) FILTER (WHERE status = 'processing') AS in_progress,
    COUNT(*) FILTER
 (WHERE status = 'pending' AND scheduled_for <= NOW()) AS ready_to_process,
    COUNT(*) FILTER (WHERE status = 'pending' AND scheduled_for > NOW()) AS scheduled_later,
    COUNT(*) FILTER (WHERE status IN ('failed', 'stopped') AND completed_at >= CURRENT_DATE) AS failed_today
FROM task_queue.task;

-- ============================================
-- CRON JOBS
-- ============================================

-- Schedule daily sync distribution at midnight
SELECT cron.schedule(
    'schedule-daily-frontdoor-syncs',
    '0 0 * * *',
    $$SELECT task_queue.fnc__schedule_daily_syncs('frontdoor_sync')$$
)
WHERE NOT EXISTS (
    SELECT 1 FROM cron.job WHERE jobname = 'schedule-daily-frontdoor-syncs'
);

-- Cleanup old completed tasks (keep 7 days)
SELECT cron.schedule(
    'cleanup-old-completed',
    '0 3 * * *',
    $$DELETE FROM task_queue.task
      WHERE (status = 'completed'
             AND completed_at < NOW() - INTERVAL '7 days')
         OR (status IN ('failed', 'stopped')
             AND completed_at < NOW() - INTERVAL '30 days')$$
)
WHERE NOT EXISTS (
    SELECT 1 FROM cron.job WHERE jobname = 'cleanup-old-completed'
);

-- Reset stuck tasks every 5 minutes
SELECT cron.schedule(
    'reset-stuck-tasks',
    '*/5 * * * *',
    $$SELECT task_queue.fnc__requeue_stuck_tasks()$$
)
WHERE NOT EXISTS (
    SELECT 1 FROM cron.job WHERE jobname = 'reset-stuck-tasks'
);
