-- name: UpsertFrontdoorPendingTask :one
-- Inserts a new pending task, or resets a completed/failed task back to pending.
-- If the task is already pending/processing, the WHERE clause prevents the update
-- and no row is returned (pgx.ErrNoRows), signaling "already active".
INSERT INTO public.frontdoor_pending_tasks (
    frontdoor_pending_task_entity_id,
    frontdoor_pending_task_type,
    frontdoor_pending_task_priority,
    frontdoor_pending_task_max_attempts
) VALUES ($1, $2, $3, $4)
ON CONFLICT (frontdoor_pending_task_entity_id, frontdoor_pending_task_type) DO UPDATE
SET frontdoor_pending_task_status = 'pending',
    frontdoor_pending_task_priority = EXCLUDED.frontdoor_pending_task_priority,
    frontdoor_pending_task_max_attempts = EXCLUDED.frontdoor_pending_task_max_attempts,
    frontdoor_pending_task_attempts = 0,
    frontdoor_pending_task_last_error = NULL,
    frontdoor_pending_task_started_at = NULL,
    frontdoor_pending_task_completed_at = NULL
WHERE frontdoor_pending_tasks.frontdoor_pending_task_status IN ('completed', 'failed')
RETURNING frontdoor_pending_task_id, frontdoor_pending_task_entity_id, frontdoor_pending_task_type, frontdoor_pending_task_status, frontdoor_pending_task_priority, frontdoor_pending_task_max_attempts, frontdoor_pending_task_attempts, frontdoor_pending_task_last_error, frontdoor_pending_task_created_at, frontdoor_pending_task_started_at, frontdoor_pending_task_completed_at;

-- name: GetFrontdoorPendingTask :one
SELECT frontdoor_pending_task_id, frontdoor_pending_task_entity_id, frontdoor_pending_task_type, frontdoor_pending_task_status, frontdoor_pending_task_priority, frontdoor_pending_task_max_attempts, frontdoor_pending_task_attempts, frontdoor_pending_task_last_error, frontdoor_pending_task_created_at, frontdoor_pending_task_started_at, frontdoor_pending_task_completed_at
FROM public.frontdoor_pending_tasks
WHERE frontdoor_pending_task_id = $1;

-- name: UpdateFrontdoorPendingTaskToProcessing :exec
UPDATE public.frontdoor_pending_tasks
SET frontdoor_pending_task_status = 'processing',
    frontdoor_pending_task_started_at = NOW(),
    frontdoor_pending_task_attempts = frontdoor_pending_task_attempts + 1
WHERE frontdoor_pending_task_id = $1;

-- name: UpdateFrontdoorPendingTaskToCompleted :exec
UPDATE public.frontdoor_pending_tasks
SET frontdoor_pending_task_status = 'completed',
    frontdoor_pending_task_completed_at = NOW()
WHERE frontdoor_pending_task_id = $1;

-- name: UpdateFrontdoorPendingTaskToFailed :exec
UPDATE public.frontdoor_pending_tasks
SET frontdoor_pending_task_status = 'failed',
    frontdoor_pending_task_completed_at = NOW(),
    frontdoor_pending_task_last_error = $2
WHERE frontdoor_pending_task_id = $1;

-- name: ResetFrontdoorPendingTaskToPending :exec
UPDATE public.frontdoor_pending_tasks
SET frontdoor_pending_task_status = 'pending',
    frontdoor_pending_task_last_error = $2
WHERE frontdoor_pending_task_id = $1;
