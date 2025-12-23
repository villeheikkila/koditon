-- name: GetEntity :one
SELECT
    entity_id,
    entity_type,
    status,
    scheduling_strategy,
    metadata,
    created_at,
    updated_at
FROM task_queue.entity_registry
WHERE entity_id = $1;

-- name: ListEntities :many
SELECT
    entity_id,
    entity_type,
    status,
    scheduling_strategy,
    metadata,
    created_at,
    updated_at
FROM task_queue.entity_registry
ORDER BY entity_id;

-- name: ListActiveEntities :many
SELECT
    entity_id,
    entity_type,
    status,
    scheduling_strategy,
    metadata,
    created_at,
    updated_at
FROM task_queue.entity_registry
WHERE status = 'active'
ORDER BY entity_id;

-- name: UpsertEntity :one
INSERT INTO task_queue.entity_registry (
    entity_id,
    entity_type,
    status,
    scheduling_strategy,
    metadata,
    created_at,
    updated_at
) VALUES (
    $1,
    $2,
    COALESCE($3, 'active'),
    COALESCE($4, 'manual'),
    COALESCE($5, '{}'::jsonb),
    NOW(),
    NOW()
)
ON CONFLICT (entity_id) DO UPDATE
SET
    entity_type = EXCLUDED.entity_type,
    status = EXCLUDED.status,
    scheduling_strategy = EXCLUDED.scheduling_strategy,
    metadata = EXCLUDED.metadata,
    updated_at = NOW()
RETURNING *;

-- name: UpdateEntityStatus :exec
UPDATE task_queue.entity_registry
SET
    status = $2,
    updated_at = NOW()
WHERE entity_id = $1;

-- name: DeleteEntity :exec
DELETE FROM task_queue.entity_registry
WHERE entity_id = $1;

-- name: CountEntitiesByStatus :one
SELECT COUNT(*) AS count
FROM task_queue.entity_registry
WHERE status = $1;

-- name: CreateTask :one
INSERT INTO task_queue.task (
    entity_id,
    task_type,
    status,
    priority,
    attempt,
    max_attempts,
    scheduled_for,
    run_on
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING *;

-- name: CreateTaskWithPriority :one
INSERT INTO task_queue.task (
    entity_id,
    task_type,
    status,
    priority,
    attempt,
    max_attempts,
    scheduled_for,
    run_on
) VALUES (
    $1, $2, 'pending', $3, 0, $4, $5, $6
)
RETURNING *;

-- name: GetTask :one
SELECT
    task_id,
    entity_id,
    task_type,
    status,
    priority,
    attempt,
    max_attempts,
    last_error,
    worker_id,
    scheduled_for,
    started_at,
    completed_at,
    run_on,
    queue_message_id,
    created_at,
    updated_at
FROM task_queue.task
WHERE task_id = $1;

-- name: GetTaskByEntityAndDate :one
SELECT
    task_id,
    entity_id,
    task_type,
    status,
    priority,
    attempt,
    max_attempts,
    last_error,
    worker_id,
    scheduled_for,
    started_at,
    completed_at,
    run_on,
    queue_message_id,
    created_at,
    updated_at
FROM task_queue.task
WHERE entity_id = $1
    AND task_type = $2
    AND run_on = $3;

-- name: ListTasks :many
SELECT
    task_id,
    entity_id,
    task_type,
    status,
    priority,
    attempt,
    max_attempts,
    last_error,
    worker_id,
    scheduled_for,
    started_at,
    completed_at,
    run_on,
    queue_message_id,
    created_at,
    updated_at
FROM task_queue.task
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListTasksByStatus :many
SELECT
    task_id,
    entity_id,
    task_type,
    status,
    priority,
    attempt,
    max_attempts,
    last_error,
    worker_id,
    scheduled_for,
    started_at,
    completed_at,
    run_on,
    queue_message_id,
    created_at,
    updated_at
FROM task_queue.task
WHERE status = $1
ORDER BY priority DESC, created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListTasksByEntity :many
SELECT
    task_id,
    entity_id,
    task_type,
    status,
    priority,
    attempt,
    max_attempts,
    last_error,
    worker_id,
    scheduled_for,
    started_at,
    completed_at,
    run_on,
    queue_message_id,
    created_at,
    updated_at
FROM task_queue.task
WHERE entity_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListTasksByWorker :many
SELECT
    task_id,
    entity_id,
    task_type,
    status,
    priority,
    attempt,
    max_attempts,
    last_error,
    worker_id,
    scheduled_for,
    started_at,
    completed_at,
    run_on,
    queue_message_id,
    created_at,
    updated_at
FROM task_queue.task
WHERE worker_id = $1
    AND status = 'processing'
ORDER BY started_at;

-- name: ListPendingTasks :many
-- Orders by priority DESC (higher first), then scheduled_for ASC (older first)
SELECT
    task_id,
    entity_id,
    task_type,
    status,
    priority,
    attempt,
    max_attempts,
    last_error,
    worker_id,
    scheduled_for,
    started_at,
    completed_at,
    run_on,
    queue_message_id,
    created_at,
    updated_at
FROM task_queue.task
WHERE status = 'pending'
    AND scheduled_for <= NOW()
ORDER BY priority DESC, scheduled_for ASC
LIMIT $1;

-- name: ListScheduledTasks :many
SELECT
    task_id,
    entity_id,
    task_type,
    status,
    priority,
    attempt,
    max_attempts,
    last_error,
    worker_id,
    scheduled_for,
    started_at,
    completed_at,
    run_on,
    queue_message_id,
    created_at,
    updated_at
FROM task_queue.task
WHERE status = 'pending'
    AND scheduled_for > NOW()
ORDER BY priority DESC, scheduled_for ASC
LIMIT $1 OFFSET $2;

-- name: ListStuckTasks :many
SELECT
    task_id,
    entity_id,
    task_type,
    status,
    priority,
    attempt,
    max_attempts,
    last_error,
    worker_id,
    scheduled_for,
    started_at,
    completed_at,
    run_on,
    queue_message_id,
    created_at,
    updated_at,
    EXTRACT(EPOCH FROM (NOW() - started_at))::INT AS stuck_seconds
FROM task_queue.task
WHERE status = 'processing'
    AND updated_at < NOW() - INTERVAL '10 minutes'
ORDER BY started_at;

-- name: UpdateTaskStatus :exec
UPDATE task_queue.task
SET
    status = $2,
    updated_at = NOW()
WHERE task_id = $1;

-- name: UpdateTaskPriority :exec
UPDATE task_queue.task
SET
    priority = $2,
    updated_at = NOW()
WHERE task_id = $1;

-- name: UpdateTaskToProcessing :exec
UPDATE task_queue.task
SET
    status = 'processing',
    attempt = attempt + 1,
    worker_id = $2,
    started_at = CASE WHEN started_at IS NULL THEN NOW() ELSE started_at END,
    updated_at = NOW()
WHERE task_id = $1;

-- name: UpdateTaskToCompleted :exec
UPDATE task_queue.task
SET
    status = 'completed',
    last_error = NULL,
    completed_at = NOW(),
    updated_at = NOW()
WHERE task_id = $1;

-- name: UpdateTaskToFailed :exec
UPDATE task_queue.task
SET
    status = 'failed',
    last_error = $2,
    completed_at = NOW(),
    updated_at = NOW()
WHERE task_id = $1;

-- name: UpdateTaskToPending :exec
UPDATE task_queue.task
SET
    status = 'pending',
    attempt = attempt + 1,
    worker_id = NULL,
    scheduled_for = $2,
    started_at = NULL,
    queue_message_id = NULL,
    updated_at = NOW()
WHERE task_id = $1;

-- name: UpdateTaskToPendingForRetry :exec
-- Used when retrying a failed task - does NOT increment attempt since UpdateTaskToProcessing already did
UPDATE task_queue.task
SET
    status = 'pending',
    worker_id = NULL,
    scheduled_for = $2,
    started_at = NULL,
    queue_message_id = NULL,
    updated_at = NOW()
WHERE task_id = $1;

-- name: UpdateTaskQueueMessageId :exec
UPDATE task_queue.task
SET
    queue_message_id = $2,
    updated_at = NOW()
WHERE task_id = $1;

-- name: DeleteTask :exec
DELETE FROM task_queue.task
WHERE task_id = $1;

-- name: DeleteOldCompletedTasks :execrows
DELETE FROM task_queue.task
WHERE status = 'completed'
    AND completed_at < $1;

-- name: DeleteOldFailedTasks :execrows
DELETE FROM task_queue.task
WHERE status IN ('failed', 'stopped')
    AND completed_at < $1;

-- name: CountTasksByStatus :one
SELECT COUNT(*) AS count
FROM task_queue.task
WHERE status = $1;

-- name: GetTaskStatusSummary :one
SELECT
    COUNT(*) FILTER (WHERE status = 'pending') AS pending,
    COUNT(*) FILTER (WHERE status = 'processing') AS processing,
    COUNT(*) FILTER (WHERE status = 'completed') AS completed,
    COUNT(*) FILTER (WHERE status = 'failed') AS failed,
    COUNT(*) FILTER (WHERE status = 'stopped') AS stopped,
    COUNT(*) AS total,
    COALESCE(ROUND(100.0 * COUNT(*) FILTER (WHERE status = 'completed') / NULLIF(COUNT(*), 0), 2), 0) AS success_rate_pct
FROM task_queue.task;

-- name: GetDailyProgress :one
SELECT
    COUNT(*) FILTER (WHERE status = 'completed' AND completed_at >= CURRENT_DATE) AS completed_today,
    COUNT(*) FILTER (WHERE status = 'processing') AS in_progress,
    COUNT(*) FILTER (WHERE status = 'pending' AND scheduled_for <= NOW()) AS ready_to_process,
    COUNT(*) FILTER (WHERE status = 'pending' AND scheduled_for > NOW()) AS scheduled_later,
    COUNT(*) FILTER (WHERE status IN ('failed', 'stopped') AND completed_at >= CURRENT_DATE) AS failed_today
FROM task_queue.task;

-- name: ListActiveWorkers :many
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

-- name: ListRecentFailures :many
SELECT
    task_id,
    entity_id,
    task_type,
    status,
    priority,
    attempt,
    max_attempts,
    last_error,
    completed_at AS failed_at,
    created_at,
    updated_at
FROM task_queue.task
WHERE status IN ('failed', 'stopped')
ORDER BY completed_at DESC
LIMIT $1;

-- name: GetTasksByRunDate :many
SELECT
    task_id,
    entity_id,
    task_type,
    status,
    priority,
    attempt,
    max_attempts,
    last_error,
    worker_id,
    scheduled_for,
    started_at,
    completed_at,
    run_on,
    queue_message_id,
    created_at,
    updated_at
FROM task_queue.task
WHERE run_on = $1
ORDER BY priority DESC, scheduled_for ASC;

-- name: UpsertTaskForDate :one
INSERT INTO task_queue.task (
    entity_id,
    task_type,
    status,
    priority,
    attempt,
    max_attempts,
    scheduled_for,
    run_on
) VALUES (
    $1,
    $2,
    'pending',
    COALESCE($3, 0),
    0,
    $4,
    $5,
    $6
)
ON CONFLICT (entity_id, task_type, run_on)
WHERE run_on IS NOT NULL
DO UPDATE SET
    scheduled_for = COALESCE(task_queue.task.scheduled_for, EXCLUDED.scheduled_for),
    priority = GREATEST(task_queue.task.priority, EXCLUDED.priority),
    updated_at = NOW()
RETURNING *;

-- name: CallRegisterEntity :exec
SELECT task_queue.fnc__register_entity($1::text, $2::text, $3::text, $4::text, $5::jsonb);

-- name: CallRegisterEntities :one
SELECT task_queue.fnc__register_entities($1::text[], $2::text, $3::text) AS count;

-- name: CallEnqueueTask :one
SELECT task_queue.fnc__enqueue_task($1::bigint) AS message_id;

-- name: CallScheduleDailySyncs :one
SELECT task_queue.fnc__schedule_daily_syncs($1::text) AS count;

-- name: CallRequeueStuckTasks :one
SELECT task_queue.fnc__requeue_stuck_tasks() AS count;

-- ============================================================================
-- Dead Letter Queue (DLQ) Queries
-- ============================================================================

-- name: InsertIntoDLQ :one
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
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
)
RETURNING *;

-- name: GetDLQEntry :one
SELECT
    dlq_id,
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
    last_attempted_at,
    moved_to_dlq_at,
    requeued_at,
    requeue_count
FROM task_queue.dead_letter_queue
WHERE dlq_id = $1;

-- name: ListDLQEntries :many
SELECT
    dlq_id,
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
    last_attempted_at,
    moved_to_dlq_at,
    requeued_at,
    requeue_count
FROM task_queue.dead_letter_queue
ORDER BY moved_to_dlq_at DESC
LIMIT $1 OFFSET $2;

-- name: ListDLQEntriesNotRequeued :many
SELECT
    dlq_id,
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
    last_attempted_at,
    moved_to_dlq_at,
    requeued_at,
    requeue_count
FROM task_queue.dead_letter_queue
WHERE requeued_at IS NULL
ORDER BY moved_to_dlq_at DESC
LIMIT $1 OFFSET $2;

-- name: ListDLQEntriesByTaskType :many
SELECT
    dlq_id,
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
    last_attempted_at,
    moved_to_dlq_at,
    requeued_at,
    requeue_count
FROM task_queue.dead_letter_queue
WHERE task_type = $1
    AND requeued_at IS NULL
ORDER BY moved_to_dlq_at DESC
LIMIT $2 OFFSET $3;

-- name: ListDLQEntriesByEntity :many
SELECT
    dlq_id,
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
    last_attempted_at,
    moved_to_dlq_at,
    requeued_at,
    requeue_count
FROM task_queue.dead_letter_queue
WHERE entity_id = $1
ORDER BY moved_to_dlq_at DESC
LIMIT $2 OFFSET $3;

-- name: MarkDLQEntryRequeued :exec
UPDATE task_queue.dead_letter_queue
SET
    requeued_at = NOW(),
    requeue_count = requeue_count + 1
WHERE dlq_id = $1;

-- name: CountDLQEntries :one
SELECT
    COUNT(*) AS total,
    COUNT(*) FILTER (WHERE requeued_at IS NULL) AS pending,
    COUNT(*) FILTER (WHERE requeued_at IS NOT NULL) AS requeued
FROM task_queue.dead_letter_queue;

-- name: CountDLQEntriesByTaskType :many
SELECT
    task_type,
    COUNT(*) AS count
FROM task_queue.dead_letter_queue
WHERE requeued_at IS NULL
GROUP BY task_type
ORDER BY count DESC;

-- name: DeleteDLQEntry :exec
DELETE FROM task_queue.dead_letter_queue
WHERE dlq_id = $1;

-- name: DeleteOldDLQEntries :execrows
DELETE FROM task_queue.dead_letter_queue
WHERE moved_to_dlq_at < $1
    AND requeued_at IS NOT NULL;

-- name: CallMoveToDLQ :one
SELECT task_queue.fnc__move_to_dlq($1::bigint, $2::jsonb) AS dlq_id;

-- name: CallRequeueFromDLQ :one
SELECT task_queue.fnc__requeue_from_dlq($1::bigint, $2::int, $3::int) AS task_id;
