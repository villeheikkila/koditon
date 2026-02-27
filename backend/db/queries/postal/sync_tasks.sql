-- name: UpsertPostalSyncTask :one
INSERT INTO public.postal_sync_tasks (
    postal_sync_task_entity_id,
    postal_sync_task_type,
    postal_sync_task_priority,
    postal_sync_task_max_attempts
) VALUES ($1, $2, $3, $4)
ON CONFLICT (postal_sync_task_entity_id, postal_sync_task_type) DO UPDATE
SET postal_sync_task_status = 'pending',
    postal_sync_task_priority = EXCLUDED.postal_sync_task_priority,
    postal_sync_task_max_attempts = EXCLUDED.postal_sync_task_max_attempts,
    postal_sync_task_attempts = 0,
    postal_sync_task_last_error = NULL,
    postal_sync_task_started_at = NULL,
    postal_sync_task_completed_at = NULL
WHERE postal_sync_tasks.postal_sync_task_status IN ('completed', 'failed')
RETURNING postal_sync_task_id, postal_sync_task_entity_id, postal_sync_task_type, postal_sync_task_status, postal_sync_task_priority, postal_sync_task_max_attempts, postal_sync_task_attempts, postal_sync_task_last_error, postal_sync_task_created_at, postal_sync_task_started_at, postal_sync_task_completed_at;

-- name: GetPostalSyncTask :one
SELECT postal_sync_task_id, postal_sync_task_entity_id, postal_sync_task_type, postal_sync_task_status, postal_sync_task_priority, postal_sync_task_max_attempts, postal_sync_task_attempts, postal_sync_task_last_error, postal_sync_task_created_at, postal_sync_task_started_at, postal_sync_task_completed_at
FROM public.postal_sync_tasks
WHERE postal_sync_task_id = $1;

-- name: UpdatePostalSyncTaskToProcessing :exec
UPDATE public.postal_sync_tasks
SET postal_sync_task_status = 'processing',
    postal_sync_task_started_at = NOW(),
    postal_sync_task_attempts = postal_sync_task_attempts + 1
WHERE postal_sync_task_id = $1;

-- name: UpdatePostalSyncTaskToCompleted :exec
UPDATE public.postal_sync_tasks
SET postal_sync_task_status = 'completed',
    postal_sync_task_completed_at = NOW()
WHERE postal_sync_task_id = $1;

-- name: UpdatePostalSyncTaskToFailed :exec
UPDATE public.postal_sync_tasks
SET postal_sync_task_status = 'failed',
    postal_sync_task_completed_at = NOW(),
    postal_sync_task_last_error = $2
WHERE postal_sync_task_id = $1;

-- name: ResetPostalSyncTaskToPending :exec
UPDATE public.postal_sync_tasks
SET postal_sync_task_status = 'pending',
    postal_sync_task_last_error = $2
WHERE postal_sync_task_id = $1;
