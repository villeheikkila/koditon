-- name: UpsertPricesPendingTask :one
INSERT INTO public.prices_pending_tasks (
    prices_pending_task_entity_id,
    prices_pending_task_type,
    prices_pending_task_priority,
    prices_pending_task_max_attempts
) VALUES ($1, $2, $3, $4)
ON CONFLICT (prices_pending_task_entity_id, prices_pending_task_type) DO UPDATE
SET prices_pending_task_status = 'pending',
    prices_pending_task_priority = EXCLUDED.prices_pending_task_priority,
    prices_pending_task_max_attempts = EXCLUDED.prices_pending_task_max_attempts,
    prices_pending_task_attempts = 0,
    prices_pending_task_last_error = NULL,
    prices_pending_task_started_at = NULL,
    prices_pending_task_completed_at = NULL
WHERE prices_pending_tasks.prices_pending_task_status IN ('completed', 'failed')
RETURNING prices_pending_task_id, prices_pending_task_entity_id, prices_pending_task_type, prices_pending_task_status, prices_pending_task_priority, prices_pending_task_max_attempts, prices_pending_task_attempts, prices_pending_task_last_error, prices_pending_task_created_at, prices_pending_task_started_at, prices_pending_task_completed_at;

-- name: GetPricesPendingTask :one
SELECT prices_pending_task_id, prices_pending_task_entity_id, prices_pending_task_type, prices_pending_task_status, prices_pending_task_priority, prices_pending_task_max_attempts, prices_pending_task_attempts, prices_pending_task_last_error, prices_pending_task_created_at, prices_pending_task_started_at, prices_pending_task_completed_at
FROM public.prices_pending_tasks
WHERE prices_pending_task_id = $1;

-- name: UpdatePricesPendingTaskToProcessing :exec
UPDATE public.prices_pending_tasks
SET prices_pending_task_status = 'processing',
    prices_pending_task_started_at = NOW(),
    prices_pending_task_attempts = prices_pending_task_attempts + 1
WHERE prices_pending_task_id = $1;

-- name: UpdatePricesPendingTaskToCompleted :exec
UPDATE public.prices_pending_tasks
SET prices_pending_task_status = 'completed',
    prices_pending_task_completed_at = NOW()
WHERE prices_pending_task_id = $1;

-- name: UpdatePricesPendingTaskToFailed :exec
UPDATE public.prices_pending_tasks
SET prices_pending_task_status = 'failed',
    prices_pending_task_completed_at = NOW(),
    prices_pending_task_last_error = $2
WHERE prices_pending_task_id = $1;

-- name: ResetPricesPendingTaskToPending :exec
UPDATE public.prices_pending_tasks
SET prices_pending_task_status = 'pending',
    prices_pending_task_last_error = $2
WHERE prices_pending_task_id = $1;
