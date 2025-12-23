CREATE SCHEMA IF NOT EXISTS task_queue;

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
CREATE INDEX idx_task_priority_scheduled ON task_queue.task(priority DESC, scheduled_for ASC) WHERE status = 'pending';
CREATE INDEX idx_task_updated ON task_queue.task(updated_at);
CREATE INDEX idx_task_run_on ON task_queue.task(run_on);

-- Dead Letter Queue for permanently failed tasks
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

-- Function signatures for sqlc
CREATE OR REPLACE FUNCTION task_queue.fnc__register_entity(
    p_entity_id TEXT,
    p_entity_type TEXT,
    p_status TEXT DEFAULT 'active',
    p_scheduling_strategy TEXT DEFAULT 'manual',
    p_metadata JSONB DEFAULT '{}'::jsonb
) RETURNS void AS $$ BEGIN END; $$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION task_queue.fnc__register_entities(
    p_entity_ids TEXT[],
    p_entity_type TEXT,
    p_scheduling_strategy TEXT DEFAULT 'daily'
) RETURNS INT AS $$ BEGIN RETURN 0; END; $$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION task_queue.fnc__enqueue_task(
    p_task_id BIGINT
) RETURNS BIGINT AS $$ BEGIN RETURN 0; END; $$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION task_queue.fnc__schedule_daily_syncs(
    p_task_type TEXT DEFAULT 'frontdoor_sync'
) RETURNS INT AS $$ BEGIN RETURN 0; END; $$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION task_queue.fnc__requeue_stuck_tasks()
RETURNS INT AS $$ BEGIN RETURN 0; END; $$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION task_queue.fnc__move_to_dlq(
    p_task_id BIGINT,
    p_error_history JSONB DEFAULT '[]'::jsonb
) RETURNS BIGINT AS $$ BEGIN RETURN 0; END; $$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION task_queue.fnc__requeue_from_dlq(
    p_dlq_id BIGINT,
    p_priority INT DEFAULT NULL,
    p_max_attempts INT DEFAULT 3
) RETURNS BIGINT AS $$ BEGIN RETURN 0; END; $$ LANGUAGE plpgsql;
