CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS pgmq;
CREATE EXTENSION IF NOT EXISTS pg_cron;
CREATE SCHEMA IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS postgis SCHEMA postgis;

SET search_path TO public, postgis;

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

CREATE SCHEMA IF NOT EXISTS task_queue;

SELECT pgmq.create('tasks');
SELECT pgmq.create('tasks_dlq');
SELECT pgmq.enable_notify_insert('tasks');

CREATE TABLE task_queue.entity_registry (
    entity_id TEXT NOT NULL PRIMARY KEY,
    entity_type TEXT NOT NULL,
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
    WHERE status = 'active' AND scheduling_strategy = 'daily';

CREATE TABLE task_queue.task_type_entity_type_mapping (
    task_type TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    PRIMARY KEY (task_type, entity_type)
);

COMMENT ON TABLE task_queue.task_type_entity_type_mapping IS
'Maps task types to the entity types they can process. Ensures correct task type scheduling for entity types.';

INSERT INTO task_queue.task_type_entity_type_mapping (task_type, entity_type) VALUES
    ('frontdoor_sync', 'frontdoor_ad'),
    ('frontdoor_sync', 'frontdoor_building'),
    ('frontdoor_sitemap_sync', 'frontdoor_sitemap'),
    ('shortcut_scraper_sync', 'shortcut_building'),
    ('shortcut_api_sync', 'shortcut_ad'),
    ('shortcut_sitemap_sync', 'shortcut_sitemap');

CREATE TABLE task_queue.task (
    task_id BIGSERIAL PRIMARY KEY,
    entity_id TEXT NOT NULL
        REFERENCES task_queue.entity_registry(entity_id) ON DELETE CASCADE,
    task_type TEXT NOT NULL DEFAULT 'frontdoor_sync',
    status TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'processing', 'completed', 'failed', 'stopped')),
    priority INT NOT NULL DEFAULT 0,
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

COMMENT ON COLUMN task_queue.task.priority IS 'Higher values = higher priority. Default 0, use negative for low priority, positive for high priority.';

CREATE UNIQUE INDEX uniq_task_daily
    ON task_queue.task(entity_id, task_type, run_on)
    WHERE run_on IS NOT NULL;

CREATE INDEX idx_task_entity ON task_queue.task(entity_id);
CREATE INDEX idx_task_status ON task_queue.task(status);
CREATE INDEX idx_task_worker ON task_queue.task(worker_id) WHERE status = 'processing';
CREATE INDEX idx_task_scheduled ON task_queue.task(scheduled_for) WHERE status = 'pending';
CREATE INDEX idx_task_updated ON task_queue.task(updated_at);
CREATE INDEX idx_task_run_on ON task_queue.task(run_on);
CREATE INDEX idx_task_priority_scheduled ON task_queue.task(priority DESC, scheduled_for ASC) WHERE status = 'pending';

CREATE TABLE task_queue.dead_letter_queue (
    dlq_id BIGSERIAL PRIMARY KEY,
    original_task_id BIGINT NOT NULL,
    entity_id TEXT NOT NULL,
    task_type TEXT NOT NULL,
    priority INT NOT NULL DEFAULT 0,
    total_attempts INT NOT NULL,
    first_error TEXT,
    last_error TEXT NOT NULL,
    error_history JSONB NOT NULL DEFAULT '[]'::jsonb,
    task_metadata JSONB DEFAULT '{}'::jsonb,
    original_created_at TIMESTAMPTZ NOT NULL,
    first_attempted_at TIMESTAMPTZ,
    last_attempted_at TIMESTAMPTZ NOT NULL,
    moved_to_dlq_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    requeued_at TIMESTAMPTZ,
    requeue_count INT NOT NULL DEFAULT 0
);

COMMENT ON TABLE task_queue.dead_letter_queue IS 'Stores tasks that have exhausted all retry attempts for debugging and manual reprocessing.';
COMMENT ON COLUMN task_queue.dead_letter_queue.error_history IS 'JSON array of {attempt, error, timestamp} objects for each failure.';
COMMENT ON COLUMN task_queue.dead_letter_queue.task_metadata IS 'Snapshot of entity metadata at time of failure.';
COMMENT ON COLUMN task_queue.dead_letter_queue.requeued_at IS 'Set when task is manually requeued for retry.';
COMMENT ON COLUMN task_queue.dead_letter_queue.requeue_count IS 'Number of times this task has been requeued from DLQ.';

CREATE INDEX idx_dlq_entity_id ON task_queue.dead_letter_queue(entity_id);
CREATE INDEX idx_dlq_task_type ON task_queue.dead_letter_queue(task_type);
CREATE INDEX idx_dlq_moved_at ON task_queue.dead_letter_queue(moved_to_dlq_at DESC);
CREATE INDEX idx_dlq_not_requeued ON task_queue.dead_letter_queue(moved_to_dlq_at DESC) WHERE requeued_at IS NULL;

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

CREATE OR REPLACE FUNCTION task_queue.fnc__move_to_dlq(
    p_task_id BIGINT,
    p_error_history JSONB DEFAULT '[]'::jsonb
) RETURNS BIGINT AS $$
DECLARE
    v_task task_queue.task%ROWTYPE;
    v_entity task_queue.entity_registry%ROWTYPE;
    v_dlq_id BIGINT;
BEGIN
    SELECT * INTO v_task FROM task_queue.task WHERE task_id = p_task_id;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'Task % not found', p_task_id;
    END IF;
    SELECT * INTO v_entity FROM task_queue.entity_registry WHERE entity_id = v_task.entity_id;
    INSERT INTO task_queue.dead_letter_queue (
        original_task_id,
        entity_id,
        task_type,
        priority,
        total_attempts,
        first_error,
        last_error,
        error_history,
        task_metadata,
        original_created_at,
        first_attempted_at,
        last_attempted_at
    ) VALUES (
        v_task.task_id,
        v_task.entity_id,
        v_task.task_type,
        v_task.priority,
        v_task.attempt,
        v_task.last_error,
        COALESCE(v_task.last_error, 'unknown error'),
        p_error_history,
        COALESCE(v_entity.metadata, '{}'::jsonb),
        v_task.created_at,
        v_task.started_at,
        NOW()
    )
    RETURNING dlq_id INTO v_dlq_id;
    RETURN v_dlq_id;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION task_queue.fnc__requeue_from_dlq(
    p_dlq_id BIGINT,
    p_priority INT DEFAULT NULL,
    p_max_attempts INT DEFAULT 3
) RETURNS BIGINT AS $$
DECLARE
    v_dlq task_queue.dead_letter_queue%ROWTYPE;
    v_task_id BIGINT;
BEGIN
    SELECT * INTO v_dlq FROM task_queue.dead_letter_queue WHERE dlq_id = p_dlq_id;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'DLQ entry % not found', p_dlq_id;
    END IF;
    INSERT INTO task_queue.task (
        entity_id,
        task_type,
        status,
        priority,
        attempt,
        max_attempts,
        scheduled_for
    ) VALUES (
        v_dlq.entity_id,
        v_dlq.task_type,
        'pending',
        COALESCE(p_priority, v_dlq.priority),
        0,
        p_max_attempts,
        NOW()
    )
    RETURNING task_id INTO v_task_id;
    UPDATE task_queue.dead_letter_queue
    SET
        requeued_at = NOW(),
        requeue_count = requeue_count + 1
    WHERE dlq_id = p_dlq_id;
    RETURN v_task_id;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION task_queue.fnc__schedule_frontdoor_sitemap_sync() RETURNS BIGINT AS $$
DECLARE
    v_task_id BIGINT;
    v_msg_id BIGINT;
    v_existing_task RECORD;
BEGIN
    SELECT task_id, status INTO v_existing_task
    FROM task_queue.task
    WHERE entity_id = 'frontdoor:sitemap'
      AND task_type = 'frontdoor_sitemap_sync'
      AND run
_on = CURRENT_DATE
    LIMIT 1;
    IF FOUND THEN
        IF v_existing_task.status IN ('pending', 'processing') THEN
            RAISE NOTICE 'Frontdoor sitemap sync already scheduled for today (task_id: %)', v_existing_task.task_id;
            RETURN v_existing_task.task_id;
        END IF;
    END IF;
    INSERT INTO task_queue.task (
        entity_id,
        task_type,
        status,
        attempt,
        max_attempts,
        scheduled_for,
        run_on
    )
    VALUES (
        'frontdoor:sitemap',
        'frontdoor_sitemap_sync',
        'pending',
        0,
        3,
        NOW(),
        CURRENT_DATE
    )
    RETURNING task_id INTO v_task_id;
    v_msg_id := pgmq.send(
        'tasks',
        jsonb_build_object(
            'task_id', v_task_id,
            'entity_id', 'frontdoor:sitemap',
            'attempt', 0
        )
    );
    UPDATE task_queue.task
    SET queue_message_id = v_msg_id,
        updated_at = NOW()
    WHERE task_id = v_task_id;
    RAISE NOTICE 'Frontdoor sitemap sync scheduled (task_id: %, msg_id: %)', v_task_id, v_msg_id;
    RETURN v_task_id;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION task_queue.fnc__schedule_shortcut_sitemap_sync()
RETURNS BIGINT AS $$
DECLARE
    v_task_id BIGINT;
    v_msg_id BIGINT;
    v_existing_task RECORD;
BEGIN
    SELECT task_id INTO v_existing_task
    FROM task_queue.task
    WHERE entity_id = 'shortcut:sitemap'
      AND task_type = 'shortcut_sitemap_sync'
      AND run_on = CURRENT_DATE
    LIMIT 1;
    IF v_existing_task.task_id IS NOT NULL THEN
        RETURN NULL;
    END IF;
    INSERT INTO task_queue.task (
        entity_id,
        task_type,
        status,
        attempt,
        scheduled_for,
        run_on
    )
    VALUES (
        'shortcut:sitemap',
        'shortcut_sitemap_sync',
        'pending',
        0,
        NOW(),
        CURRENT_DATE
    )
    RETURNING task_id INTO v_task_id;
    IF v_task_id IS NOT NULL THEN
        v_msg_id := task_queue.fnc__enqueue_task(v_task_id);
    END IF;
    RETURN v_msg_id;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION task_queue.fnc__schedule_shortcut_sitemap_sync() IS
'Creates a shortcut_sitemap_sync task that workers will process to fetch shortcut sitemaps and register entities.';

CREATE OR REPLACE FUNCTION task_queue.fnc__schedule_prices_cities_init() RETURNS BIGINT AS $$
DECLARE
    v_task_id BIGINT;
    v_msg_id BIGINT;
    v_existing_task RECORD;
BEGIN
    SELECT task_id, status INTO v_existing_task
    FROM task_queue.task
    WHERE entity_id = 'prices:cities'
      AND task_type = 'prices_cities_init'
      AND run_on = CURRENT_DATE
    LIMIT 1;
    IF FOUND THEN
        IF v_existing_task.status IN ('pending', 'processing') THEN
            RAISE NOTICE 'Prices cities init already scheduled for today (task_id: %)', v_existing_task.task_id;
            RETURN v_existing_task.task_id;
        END IF;
    END IF;
    INSERT INTO task_queue.task (
        entity_id,
        task_type,
        status,
        attempt,
        max_attempts,
        scheduled_for,
        run_on
    )
    VALUES (
        'prices:cities',
        'prices_cities_init',
        'pending',
        0,
        3,
        NOW(),
        CURRENT_DATE
    )
    RETURNING task_id INTO v_task_id;
    v_msg_id := pgmq.send(
        'tasks',
        jsonb_build_object(
            'task_id', v_task_id,
            'entity_id', 'prices:cities',
            'attempt', 0
        )
    );
    UPDATE task_queue.task
    SET queue_message_id = v_msg_id,
        updated_at = NOW()
    WHERE task_id = v_task_id;
    RAISE NOTICE 'Prices cities init scheduled (task_id: %, msg_id: %)', v_task_id, v_msg_id;
    RETURN v_task_id;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION task_queue.fnc__schedule_prices_cities_init() IS
'Creates a prices_cities_init task that workers will process to fetch all available cities and register them as entities for daily transaction data syncing.';

CREATE OR REPLACE FUNCTION task_queue.fnc__get_sync_statistics() RETURNS TABLE(
    total_entities INT,
    active_entities INT,
    stopped_entities INT,
    total_tasks INT,
    pending_tasks INT,
    processing_tasks INT,
    completed_tasks_today INT,
    failed_tasks_today INT,
    avg_task_duration_seconds NUMERIC,
    sitemap_last_sync TIMESTAMPTZ,
    sitemap_last_status TEXT
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        COUNT(*)::INT AS total_entities,
        COUNT(*) FILTER (WHERE status = 'active')::INT AS active_entities,
        COUNT(*) FILTER (WHERE status = 'stopped')::INT AS stopped_entities,
        (SELECT COUNT(*)::INT FROM task_queue.task) AS total_tasks,
        (SELECT COUNT(*)::INT FROM task_queue.task WHERE status = 'pending') AS pending_tasks,
        (SELECT COUNT(*)::INT FROM task_queue.task WHERE status = 'processing') AS processing_tasks,
        (SELECT COUNT(*)::INT FROM task_queue.task WHERE status = 'completed' AND completed_at >= CURRENT_DATE) AS completed_tasks_today,
        (SELECT COUNT(*)::INT FROM task_queue.task WHERE status IN ('failed', 'stopped') AND completed_at >= CURRENT_DATE) AS failed_tasks_today,
        (SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (completed_at - started_at))), 0)::NUMERIC(10,2)
         FROM task_queue.task
         WHERE status = 'completed'
           AND completed_at >= CURRENT_DATE
           AND started_at IS NOT NULL) AS avg_task_duration_seconds,
        (SELECT MAX(completed_at)
         FROM task_queue.task
         WHERE entity_id = 'frontdoor:sitemap'
           AND task_type = 'frontdoor_sitemap_sync'
           AND status = 'completed') AS sitemap_last_sync,
        (SELECT status
         FROM task_queue.task
         WHERE entity_id = 'frontdoor:sitemap'
           AND task_type = 'frontdoor_sitemap_sync'
         ORDER BY created_at DESC
         LIMIT 1) AS sitemap_last_status
    FROM task_queue.entity_registry
    WHERE scheduling_strategy != 'cron';
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION task_queue.fnc__get_entity_sync_status(p_entity_id TEXT) RETURNS TABLE(
    entity_id TEXT,
    entity_status TEXT,
    last_task_id BIGINT,
    last_task_status TEXT,
    last_task_completed_at TIMESTAMPTZ,
    last_task_error TEXT,
    total_completed_count BIGINT,
    total_failed_count BIGINT,
    success_rate NUMERIC
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        e.entity_id,
        e.status AS entity_status,
        t_last.task_id AS last_task_id,
        t_last.status AS last_task_status,
        t_last.completed_at AS last_task_completed_at,
        t_last.last_error AS last_task_error,
        COUNT(*) FILTER (WHERE t.status = 'completed') AS total_completed_count,
        COUNT(*) FILTER (WHERE t.status = 'failed') AS total_failed_count,
        CASE
            WHEN COUNT(*) > 0 THEN
                ROUND(100.0 * COUNT(*) FILTER (WHERE t.status = 'completed') / COUNT(*), 2)
            ELSE 0
        END AS success_rate
    FROM task_queue.entity_registry e
    LEFT JOIN task_queue.task t ON t.entity_id = e.entity_id
    LEFT JOIN LATERAL (
        SELECT task_id, status, completed_at, last_error
        FROM task_queue.task
        WHERE entity_id = e.entity_id
        ORDER BY created_at DESC
        LIMIT 1
    ) t_last ON true
    WHERE e.entity_id = p_entity_id
    GROUP BY e.entity_id, e.status, t_last.task_id, t_last.status, t_last.completed_at, t_last.last_error;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE VIEW task_queue.vw_entity_sync_health AS
SELECT
    e.entity_id,
    e.status AS entity_status,
    e.entity_type,
    e.scheduling_strategy,
    COUNT(t.task_id) AS total_tasks,
    COUNT(*) FILTER (WHERE t.status = 'completed') AS completed_tasks,
    COUNT(*) FILTER (WHERE t.status = 'failed') AS failed_tasks,
    COUNT(*) FILTER (WHERE t.status = 'pending') AS pending_tasks,
    COUNT(*) FILTER (WHERE t.status = 'processing') AS processing_tasks,
    MAX(t.completed_at) AS last_completed_at,
    MAX(t.last_error) FILTER (WHERE t.status = 'failed') AS last_error,
    CASE
        WHEN COUNT(t.task_id) > 0 THEN
            ROUND(100.0 * COUNT(*) FILTER (WHERE t.status = 'completed') / COUNT(t.task_id), 2)
        ELSE 0
    END AS success_rate_pct
FROM task_queue.entity_registry e
LEFT JOIN task_queue.task t ON t.entity_id = e.entity_id
GROUP BY e.entity_id, e.status, e.entity_type, e.scheduling_strategy
ORDER BY last_completed_at DESC NULLS LAST;

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

CREATE OR REPLACE VIEW task_queue.fnc__daily_progress AS
SELECT
    COUNT(*) FILTER (WHERE status = 'completed' AND completed_at >= CURRENT_DATE) AS completed_today,
    COUNT(*) FILTER (WHERE status = 'processing') AS in_progress,
    COUNT(*) FILTER (WHERE status = 'pending' AND scheduled_for <= NOW()) AS ready_to_process,
    COUNT(*) FILTER (WHERE status = 'pending' AND scheduled_for > NOW()) AS scheduled_later,
    COUNT(*) FILTER (WHERE status IN ('failed', 'stopped') AND completed_at >= CURRENT_DATE) AS failed_today
FROM task_queue.task;

CREATE TABLE public.shortcut_buildings (
    shortcut_buildings_id uuid NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    shortcut_buildings_external_id int8 NOT NULL UNIQUE,
    shortcut_buildings_building_id text,
    shortcut_buildings_building_type text,
    shortcut_buildings_building_subtype text,
    shortcut_buildings_construction_year int4,
    shortcut_buildings_floor_count int4,
    shortcut_buildings_apartment_count int4,
    shortcut_buildings_heating_system text,
    shortcut_buildings_building_material text,
    shortcut_buildings_plot_type text,
    shortcut_buildings_wall_structure text,
    shortcut_buildings_heat_source text,
    shortcut_buildings_has_elevator text,
    shortcut_buildings_has_sauna text,
    shortcut_buildings_latitude float8,
    shortcut_buildings_longitude float8,
    shortcut_buildings_additional_addresses text,
    shortcut_buildings_url text NOT NULL,
    shortcut_buildings_created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    shortcut_buildings_updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    shortcut_buildings_address text,
    shortcut_buildings_processed_at timestamptz,
    shortcut_buildings_page_not_found bool DEFAULT false,
    shortcut_buildings_frame_construction_method text,
    shortcut_buildings_housing_company text,
    shortcut_buildings_geom geometry(Point, 4326)
);

CREATE INDEX shortcut_buildings_geom_idx ON public.shortcut_buildings USING GIST (shortcut_buildings_geom);

CREATE TABLE public.shortcut_building_listings (
    shortcut_building_listings_id uuid NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    shortcut_building_listings_building_id uuid NOT NULL REFERENCES public.shortcut_buildings(shortcut_buildings_id) ON DELETE CASCADE,
    shortcut_building_listings_layout text,
    shortcut_building_listings_size float8,
    shortcut_building_listings_price float8,
    shortcut_building_listings_price_per_sqm float8,
    shortcut_building_listings_deleted_at timestamptz,
    shortcut_building_listings_created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    shortcut_building_listings_updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    shortcut_building_listings_marketing_time text,
    shortcut_building_listings_idx int4
);

CREATE UNIQUE INDEX shortcut_building_listings_unique_constraint ON public.shortcut_building_listings(
    shortcut_building_listings_building_id,
    shortcut_building_listings_layout,
    shortcut_building_listings_size,
    shortcut_building_listings_price,
    shortcut_building_listings_price_per_sqm,
    shortcut_building_listings_deleted_at,
    shortcut_building_listings_marketing_time,
    shortcut_building_listings_idx
);

CREATE TABLE public.shortcut_building_rentals (
    shortcut_building_rentals_id uuid NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    shortcut_building_rentals_building_id uuid NOT NULL REFERENCES public.shortcut_buildings(shortcut_buildings_id) ON DELETE CASCADE,
    shortcut_building_rentals_layout text,
    shortcut_building_rentals_size float8,
    shortcut_building_rentals_price float8,
    shortcut_building_rentals_deleted_at timestamptz,
    shortcut_building_rentals_created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    shortcut_building_rentals_updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    shortcut_building_rentals_marketing_time text,
    shortcut_building_rentals_idx int4
);

CREATE UNIQUE INDEX shortcut_building_rentals_unique_constraint ON public.shortcut_building_rentals(
    shortcut_building_rentals_building_id,
    shortcut_building_rentals_layout,
    shortcut_building_rentals_size,
    shortcut_building_rentals_price,
    shortcut_building_rentals_deleted_at,
    shortcut_building_rentals_marketing_time,
    shortcut_building_rentals_idx
);

CREATE TABLE public.shortcut_ads (
    shortcut_ads_id int8 NOT NULL PRIMARY KEY,
    shortcut_ads_url text NOT NULL,
    shortcut_ads_type text NOT NULL,
    shortcut_ads_first_seen_at timestamptz NOT NULL DEFAULT now(),
    shortcut_ads_last_seen_at timestamptz NOT NULL DEFAULT now(),
    shortcut_ads_data jsonb,
    shortcut_ads_updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    shortcut_ads_building_id uuid REFERENCES public.shortcut_buildings(shortcut_buildings_id) ON DELETE SET NULL
);

CREATE INDEX idx_shortcut_ads_zipcode_name ON public.shortcut_ads(((((shortcut_ads_data -> 'address'::text) -> 'zipCode'::text) ->> 'name'::text)));

CREATE TABLE public.frontdoor_ads (
    frontdoor_ads_id uuid NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    frontdoor_ads_external_id text NOT NULL UNIQUE,
    frontdoor_ads_url text NOT NULL,
    frontdoor_ads_first_seen_at timestamptz NOT NULL DEFAULT now(),
    frontdoor_ads_last_seen_at timestamptz NOT NULL DEFAULT now(),
    frontdoor_ads_updated_at timestamptz NOT NULL DEFAULT now(),
    frontdoor_ads_data jsonb,
    frontdoor_ads_processed_at timestamptz,
    frontdoor_ads_page_not_found bool NOT NULL DEFAULT false,
    frontdoor_ads_publishing_time timestamptz
);

CREATE INDEX idx_frontdoor_ads_processed_at ON public.frontdoor_ads(frontdoor_ads_processed_at);
CREATE INDEX idx_frontdoor_ads_page_not_found ON public.frontdoor_ads(frontdoor_ads_page_not_found);

CREATE TABLE public.frontdoor_buildings (
    frontdoor_buildings_id uuid NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    frontdoor_buildings_url text,
    frontdoor_buildings_first_seen_at timestamptz NOT NULL DEFAULT now(),
    frontdoor_buildings_last_seen_at timestamptz NOT NULL DEFAULT now(),
    frontdoor_buildings_updated_at timestamptz NOT NULL DEFAULT now(),
    frontdoor_buildings_company_name text,
    frontdoor_buildings_business_id text,
    frontdoor_buildings_apartment_count int4,
    frontdoor_buildings_floor_count int4,
    frontdoor_buildings_construction_end_year int4,
    frontdoor_buildings_build_year int4,
    frontdoor_buildings_has_elevator bool,
    frontdoor_buildings_has_sauna bool,
    frontdoor_buildings_energy_certificate_code text,
    frontdoor_buildings_plot_holding_type text,
    frontdoor_buildings_outer_roof_material text,
    frontdoor_buildings_outer_roof_type text,
    frontdoor_buildings_heating text,
    frontdoor_buildings_heating_fuel text[],
    frontdoor_buildings_street_address text,
    frontdoor_buildings_house_number text,
    frontdoor_buildings_postcode text,
    frontdoor_buildings_post_area text,
    frontdoor_buildings_municipality text,
    frontdoor_buildings_district text,
    frontdoor_buildings_latitude float8,
    frontdoor_buildings_longitude float8,
    frontdoor_buildings_elevator_renovated bool,
    frontdoor_buildings_elevator_renovated_year int4,
    frontdoor_buildings_facade_renovated bool,
    frontdoor_buildings_facade_renovated_year int4,
    frontdoor_buildings_window_renovated bool,
    frontdoor_buildings_window_renovated_year int4,
    frontdoor_buildings_roof_renovated bool,
    frontdoor_buildings_roof_renovated_year int4,
    frontdoor_buildings_pipe_renovated bool,
    frontdoor_buildings_pipe_renovated_year int4,
    frontdoor_buildings_balcony_renovated bool,
    frontdoor_buildings_balcony_renovated_year int4,
    frontdoor_buildings_electricity_renovated bool,
    frontdoor_buildings_electricity_renovated_year int4,
    frontdoor_buildings_contact_phone text,
    frontdoor_buildings_contact_office_name text,
    frontdoor_buildings_contact_office_id int4,
    frontdoor_buildings_description text,
    frontdoor_buildings_car_storage_description text,
    frontdoor_buildings_other_info text,
    frontdoor_buildings_additional_addresses jsonb[],
    frontdoor_buildings_links jsonb[],
    frontdoor_buildings_data jsonb,
    frontdoor_buildings_processed_at timestamptz,
    frontdoor_buildings_housing_company_id int8 UNIQUE,
    frontdoor_buildings_housing_company_friendly_id text,
    frontdoor_buildings_geom geometry(Point, 4326)
);

CREATE INDEX idx_frontdoor_buildings_processed_at ON public.frontdoor_buildings(frontdoor_buildings_processed_at);
CREATE INDEX idx_frontdoor_buildings_business_id ON public.frontdoor_buildings(frontdoor_buildings_business_id);

CREATE TABLE public.frontdoor_building_announcements (
    frontdoor_building_announcements_id uuid NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    frontdoor_building_announcements_external_id int4,
    frontdoor_building_announcements_friendly_id text,
    frontdoor_building_announcements_unpublishing_time float8,
    frontdoor_building_announcements_address_line1 text,
    frontdoor_building_announcements_address_line2 text,
    frontdoor_building_announcements_location text,
    frontdoor_building_announcements_search_price float8,
    frontdoor_building_announcements_notify_price_changed bool,
    frontdoor_building_announcements_property_type text,
    frontdoor_building_announcements_property_subtype text,
    frontdoor_building_announcements_construction_finished_year int4,
    frontdoor_building_announcements_main_image_uri text,
    frontdoor_building_announcements_has_open_bidding bool,
    frontdoor_building_announcements_room_structure text,
    frontdoor_building_announcements_area float8,
    frontdoor_building_announcements_total_area float8,
    frontdoor_building_announcements_price_per_square float8,
    frontdoor_building_announcements_days_on_market int4,
    frontdoor_building_announcements_new_building bool,
    frontdoor_building_announcements_main_image_hidden bool,
    frontdoor_building_announcements_is_company_announcement bool,
    frontdoor_building_announcements_show_bidding_indicators bool,
    frontdoor_building_announcements_published bool,
    frontdoor_building_announcements_rent_period text,
    frontdoor_building_announcements_rental_unique_no int4,
    frontdoor_building_announcements_building_id uuid NOT NULL REFERENCES public.frontdoor_buildings(frontdoor_buildings_id) ON DELETE CASCADE,
    frontdoor_building_announcements_first_seen_at timestamptz NOT NULL DEFAULT now(),
    frontdoor_building_announcements_last_seen_at timestamptz NOT NULL DEFAULT now(),
    frontdoor_building_announcements_unpublishing_time_date date
);

CREATE UNIQUE INDEX frontdoor_building_announcements_ext_id_unpub_time_price_key
    ON public.frontdoor_building_announcements(
        frontdoor_building_announcements_external_id,
        frontdoor_building_announcements_unpublishing_time,
        frontdoor_building_announcements_search_price
    );
CREATE INDEX idx_frontdoor_building_announcements_building_id
    ON public.frontdoor_building_announcements(frontdoor_building_announcements_building_id);

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

COMMENT ON TABLE public.shortcut_tokens IS 'Token storage for shortcut.fi users - supports multiple users for data fetching';
COMMENT ON COLUMN public.shortcut_tokens.shortcut_tokens_cuid IS 'Unique user identifier from shortcut.fi';
COMMENT ON COLUMN public.shortcut_tokens.shortcut_tokens_token IS 'Authentication token';
COMMENT ON COLUMN public.shortcut_tokens.shortcut_tokens_loaded IS 'Loaded token value';

CREATE INDEX idx_shortcut_tokens_expires_at ON public.shortcut_tokens(shortcut_tokens_expires_at DESC);
CREATE INDEX idx_shortcut_tokens_cuid ON public.shortcut_tokens(shortcut_tokens_cuid);

INSERT INTO task_queue.entity_registry (entity_id, entity_type, status, scheduling_strategy)
VALUES ('frontdoor:sitemap', 'frontdoor_sitemap', 'active', 'cron')
ON CONFLICT (entity_id) DO NOTHING;

INSERT INTO task_queue.entity_registry (entity_id, entity_type, status, scheduling_strategy)
VALUES ('shortcut:sitemap', 'shortcut_sitemap', 'active', 'cron')
ON CONFLICT (entity_id) DO NOTHING;

INSERT INTO task_queue.entity_registry (entity_id, entity_type, status, scheduling_strategy)
VALUES ('prices:cities', 'prices_cities', 'active', 'cron')
ON CONFLICT (entity_id) DO NOTHING;

SELECT cron.schedule(
    'trigger-frontdoor-sitemap-sync',
    '0 1 * * *',
    $$SELECT task_queue.fnc__schedule_frontdoor_sitemap_sync()$$
)
WHERE NOT EXISTS (
    SELECT 1 FROM cron.job WHERE jobname = 'trigger-frontdoor-sitemap-sync'
);

SELECT cron.schedule(
    'schedule-daily-frontdoor-syncs',
    '0 2 * * *',
    $$SELECT task_queue.fnc__schedule_daily_syncs('frontdoor_sync')$$
)
WHERE NOT EXISTS (
    SELECT 1 FROM cron.job WHERE jobname = 'schedule-daily-frontdoor-syncs'
);

SELECT cron.schedule(
    'cleanup-old-completed',
    '0 3 * * *',
    $$DELETE FROM task_queue.task
      WHERE (status = 'completed' AND completed_at < NOW() - INTERVAL '7 days')
         OR (status IN ('failed', 'stopped') AND completed_at < NOW() - INTERVAL '30 days')$$
)
WHERE NOT EXISTS (
    SELECT 1 FROM cron.job WHERE jobname = 'cleanup-old-completed'
);

SELECT cron.schedule(
    'reset-stuck-tasks',
    '*/5 * * * *',
    $$SELECT task_queue.fnc__requeue_stuck_tasks()$$
)
WHERE NOT EXISTS (
    SELECT 1 FROM cron.job WHERE jobname = 'reset-stuck-tasks'
);

SELECT cron.schedule(
    'trigger-shortcut-sitemap-sync',
    '30 1 * * *',
    $$SELECT task_queue.fnc__schedule_shortcut_sitemap_sync()$$
)
WHERE NOT EXISTS (
    SELECT 1 FROM cron.job WHERE jobname = 'trigger-shortcut-sitemap-sync'
);

SELECT cron.schedule(
    'schedule-daily-shortcut-scraper-syncs',
    '30 2 * * *',
    $$SELECT task_queue.fnc__schedule_daily_syncs('shortcut_scraper_sync')$$
)
WHERE NOT EXISTS (
    SELECT 1 FROM cron.job WHERE jobname = 'schedule-daily-shortcut-scraper-syncs'
);

SELECT cron.schedule(
    'schedule-daily-shortcut-api-syncs',
    '30 3 * * *',
    $$SELECT task_queue.fnc__schedule_daily_syncs('shortcut_api_sync')$$
)
WHERE NOT EXISTS (
    SELECT 1 FROM cron.job WHERE jobname = 'schedule-daily-shortcut-api-syncs'
);

SELECT cron.schedule(
    'trigger-prices-cities-init',
    '0 4 * * 0',
    $$SELECT task_queue.fnc__schedule_prices_cities_init()$$
)
WHERE NOT EXISTS (
    SELECT 1 FROM cron.job WHERE jobname = 'trigger-prices-cities-init'
);

SELECT cron.schedule(
    'schedule-daily-prices-syncs',
    '30 4 * * *',
    $$SELECT task_queue.fnc__schedule_daily_syncs('prices_sync')$$
)
WHERE NOT EXISTS (
    SELECT 1 FROM cron.job WHERE jobname = 'schedule-daily-prices-syncs'
);

COMMENT ON TABLE task_queue.entity_registry IS
'Registry of all entities that can be synced. Uses entity_type and scheduling_strategy for flexible, scalable task management.';

COMMENT ON COLUMN task_queue.entity_registry.entity_type IS
'Type of entity (e.g., frontdoor_ad, frontdoor_building, frontdoor_sitemap). Used for grouping and filtering.';

COMMENT ON COLUMN task_queue.entity_registry.scheduling_strategy IS
'How this entity should be scheduled: daily (auto-scheduled by cron), manual (on-demand only), on_demand (triggered by events), cron (scheduled by specific cron job).';

COMMENT ON COLUMN task_queue.entity_registry.metadata IS
'Additional JSON metadata for entity-specific configuration and tracking.';

COMMENT ON FUNCTION task_queue.fnc__schedule_frontdoor_sitemap_sync() IS
'Creates a frontdoor_sitemap_sync task that workers will process to fetch frontdoor sitemaps and register entities. Called by pg_cron daily at 1 AM.';

COMMENT ON FUNCTION task_queue.fnc__schedule_daily_syncs(TEXT) IS
'Creates sync tasks for all active entities with scheduling_strategy = daily. Scalable design without hardcoded exclusions.';

COMMENT ON FUNCTION task_queue.fnc__get_sync_statistics() IS
'Returns overall sync statistics including entity counts, task counts, average processing time, and last sitemap sync status.';

COMMENT ON FUNCTION task_queue.fnc__get_entity_sync_status(TEXT) IS
'Returns detailed sync status for a specific entity including success rate and last task information.';

COMMENT ON VIEW task_queue.vw_entity_sync_health IS
'Dashboard view showing sync health status for all entities with success rates and error information.';

---- create above / drop below ----

DROP VIEW IF EXISTS task_queue.vw_entity_sync_health;
DROP VIEW IF EXISTS task_queue.fnc__status_summary;
DROP VIEW IF EXISTS task_queue.fnc__active_workers;
DROP VIEW IF EXISTS task_queue.fnc__recent_failures;
DROP VIEW IF EXISTS task_queue.fnc__stuck_tasks;
DROP VIEW IF EXISTS task_queue.fnc__daily_progress;

DROP FUNCTION IF EXISTS task_queue.fnc__get_entity_sync_status(TEXT);
DROP FUNCTION IF EXISTS task_queue.fnc__get_sync_statistics();
DROP FUNCTION IF EXISTS task_queue.fnc__schedule_prices_cities_init();
DROP FUNCTION IF EXISTS task_queue.fnc__schedule_shortcut_sitemap_sync();
DROP FUNCTION IF EXISTS task_queue.fnc__schedule_frontdoor_sitemap_sync();
DROP FUNCTION IF EXISTS task_queue.fnc__requeue_from_dlq(BIGINT, INT, INT);
DROP FUNCTION IF EXISTS task_queue.fnc__move_to_dlq(BIGINT, JSONB);
DROP FUNCTION IF EXISTS task_queue.fnc__requeue_stuck_tasks();
DROP FUNCTION IF EXISTS task_queue.fnc__schedule_daily_syncs(TEXT);
DROP FUNCTION IF EXISTS task_queue.fnc__enqueue_task(BIGINT);
DROP FUNCTION IF EXISTS task_queue.fnc__register_entities(TEXT[], TEXT, TEXT);
DROP FUNCTION IF EXISTS task_queue.fnc__register_entity(TEXT, TEXT, TEXT, TEXT, JSONB);

DROP TABLE IF EXISTS public.shortcut_tokens CASCADE;
DROP TABLE IF EXISTS public.frontdoor_building_announcements CASCADE;
DROP TABLE IF EXISTS public.frontdoor_buildings CASCADE;
DROP TABLE IF EXISTS public.frontdoor_ads CASCADE;
DROP TABLE IF EXISTS public.shortcut_ads CASCADE;
DROP TABLE IF EXISTS public.shortcut_building_rentals CASCADE;
DROP TABLE IF EXISTS public.shortcut_building_listings CASCADE;
DROP TABLE IF EXISTS public.shortcut_buildings CASCADE;

DROP TABLE IF EXISTS task_queue.dead_letter_queue CASCADE;
DROP TABLE IF EXISTS task_queue.task CASCADE;
DROP TABLE IF EXISTS task_queue.task_type_entity_type_mapping CASCADE;
DROP TABLE IF EXISTS task_queue.entity_registry CASCADE;

SELECT pgmq.drop_queue('tasks_dlq');
SELECT pgmq.drop_queue('tasks');

DROP SCHEMA IF EXISTS task_queue CASCADE;

DROP TABLE IF EXISTS public.prices_transactions CASCADE;
DROP TABLE IF EXISTS public.prices_neighborhoods CASCADE;
DROP TABLE IF EXISTS public.prices_postal_codes CASCADE;
DROP TABLE IF EXISTS public.prices_cities CASCADE;

DROP EXTENSION IF EXISTS pg_cron CASCADE;
DROP EXTENSION IF EXISTS pgmq CASCADE;
DROP EXTENSION IF EXISTS postgis CASCADE;
DROP SCHEMA IF EXISTS postgis CASCADE;
DROP EXTENSION IF EXISTS "uuid-ossp" CASCADE;
