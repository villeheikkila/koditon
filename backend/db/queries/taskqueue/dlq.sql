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
