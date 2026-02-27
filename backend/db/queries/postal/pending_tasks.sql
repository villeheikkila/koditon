-- name: UpsertPostalPendingTask :one
INSERT INTO public.postal_pending_tasks (
    postal_pending_task_entity_id,
    postal_pending_task_type,
    postal_pending_task_priority,
    postal_pending_task_max_attempts
) VALUES ($1, $2, $3, $4)
ON CONFLICT (postal_pending_task_entity_id, postal_pending_task_type) DO UPDATE
SET postal_pending_task_status = 'pending',
    postal_pending_task_priority = EXCLUDED.postal_pending_task_priority,
    postal_pending_task_max_attempts = EXCLUDED.postal_pending_task_max_attempts,
    postal_pending_task_attempts = 0,
    postal_pending_task_last_error = NULL,
    postal_pending_task_started_at = NULL,
    postal_pending_task_completed_at = NULL
WHERE postal_pending_tasks.postal_pending_task_status IN ('completed', 'failed')
RETURNING postal_pending_task_id, postal_pending_task_entity_id, postal_pending_task_type, postal_pending_task_status, postal_pending_task_priority, postal_pending_task_max_attempts, postal_pending_task_attempts, postal_pending_task_last_error, postal_pending_task_created_at, postal_pending_task_started_at, postal_pending_task_completed_at;

-- name: GetPostalPendingTask :one
SELECT postal_pending_task_id, postal_pending_task_entity_id, postal_pending_task_type, postal_pending_task_status, postal_pending_task_priority, postal_pending_task_max_attempts, postal_pending_task_attempts, postal_pending_task_last_error, postal_pending_task_created_at, postal_pending_task_started_at, postal_pending_task_completed_at
FROM public.postal_pending_tasks
WHERE postal_pending_task_id = $1;

-- name: UpdatePostalPendingTaskToProcessing :exec
UPDATE public.postal_pending_tasks
SET postal_pending_task_status = 'processing',
    postal_pending_task_started_at = NOW(),
    postal_pending_task_attempts = postal_pending_task_attempts + 1
WHERE postal_pending_task_id = $1;

-- name: UpdatePostalPendingTaskToCompleted :exec
UPDATE public.postal_pending_tasks
SET postal_pending_task_status = 'completed',
    postal_pending_task_completed_at = NOW()
WHERE postal_pending_task_id = $1;

-- name: UpdatePostalPendingTaskToFailed :exec
UPDATE public.postal_pending_tasks
SET postal_pending_task_status = 'failed',
    postal_pending_task_completed_at = NOW(),
    postal_pending_task_last_error = $2
WHERE postal_pending_task_id = $1;

-- name: ResetPostalPendingTaskToPending :exec
UPDATE public.postal_pending_tasks
SET postal_pending_task_status = 'pending',
    postal_pending_task_last_error = $2
WHERE postal_pending_task_id = $1;
