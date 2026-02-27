-- name: UpsertPricesSyncTask :one
INSERT INTO public.prices_sync_tasks (
    prices_sync_task_entity_id,
    prices_sync_task_type,
    prices_sync_task_priority,
    prices_sync_task_max_attempts
) VALUES ($1, $2, $3, $4)
ON CONFLICT (prices_sync_task_entity_id, prices_sync_task_type) DO UPDATE
SET prices_sync_task_status = 'pending',
    prices_sync_task_priority = EXCLUDED.prices_sync_task_priority,
    prices_sync_task_max_attempts = EXCLUDED.prices_sync_task_max_attempts,
    prices_sync_task_attempts = 0,
    prices_sync_task_last_error = NULL,
    prices_sync_task_started_at = NULL,
    prices_sync_task_completed_at = NULL
WHERE prices_sync_tasks.prices_sync_task_status IN ('completed', 'failed')
RETURNING prices_sync_task_id, prices_sync_task_entity_id, prices_sync_task_type, prices_sync_task_status, prices_sync_task_priority, prices_sync_task_max_attempts, prices_sync_task_attempts, prices_sync_task_last_error, prices_sync_task_created_at, prices_sync_task_started_at, prices_sync_task_completed_at;

-- name: GetPricesSyncTask :one
SELECT prices_sync_task_id, prices_sync_task_entity_id, prices_sync_task_type, prices_sync_task_status, prices_sync_task_priority, prices_sync_task_max_attempts, prices_sync_task_attempts, prices_sync_task_last_error, prices_sync_task_created_at, prices_sync_task_started_at, prices_sync_task_completed_at
FROM public.prices_sync_tasks
WHERE prices_sync_task_id = $1;

-- name: UpdatePricesSyncTaskToProcessing :exec
UPDATE public.prices_sync_tasks
SET prices_sync_task_status = 'processing',
    prices_sync_task_started_at = NOW(),
    prices_sync_task_attempts = prices_sync_task_attempts + 1
WHERE prices_sync_task_id = $1;

-- name: UpdatePricesSyncTaskToCompleted :exec
UPDATE public.prices_sync_tasks
SET prices_sync_task_status = 'completed',
    prices_sync_task_completed_at = NOW()
WHERE prices_sync_task_id = $1;

-- name: UpdatePricesSyncTaskToFailed :exec
UPDATE public.prices_sync_tasks
SET prices_sync_task_status = 'failed',
    prices_sync_task_completed_at = NOW(),
    prices_sync_task_last_error = $2
WHERE prices_sync_task_id = $1;

-- name: ResetPricesSyncTaskToPending :exec
UPDATE public.prices_sync_tasks
SET prices_sync_task_status = 'pending',
    prices_sync_task_last_error = $2
WHERE prices_sync_task_id = $1;
