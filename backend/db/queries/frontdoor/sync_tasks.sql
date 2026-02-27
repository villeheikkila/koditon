-- name: UpsertFrontdoorSyncTask :one
INSERT INTO public.frontdoor_sync_tasks (
    frontdoor_sync_task_entity_id,
    frontdoor_sync_task_type,
    frontdoor_sync_task_priority,
    frontdoor_sync_task_max_attempts
) VALUES ($1, $2, $3, $4)
ON CONFLICT (frontdoor_sync_task_entity_id, frontdoor_sync_task_type) DO UPDATE
SET frontdoor_sync_task_status = 'pending',
    frontdoor_sync_task_priority = EXCLUDED.frontdoor_sync_task_priority,
    frontdoor_sync_task_max_attempts = EXCLUDED.frontdoor_sync_task_max_attempts,
    frontdoor_sync_task_attempts = 0,
    frontdoor_sync_task_last_error = NULL,
    frontdoor_sync_task_started_at = NULL,
    frontdoor_sync_task_completed_at = NULL
WHERE frontdoor_sync_tasks.frontdoor_sync_task_status IN ('completed', 'failed')
RETURNING frontdoor_sync_task_id, frontdoor_sync_task_entity_id, frontdoor_sync_task_type, frontdoor_sync_task_status, frontdoor_sync_task_priority, frontdoor_sync_task_max_attempts, frontdoor_sync_task_attempts, frontdoor_sync_task_last_error, frontdoor_sync_task_created_at, frontdoor_sync_task_started_at, frontdoor_sync_task_completed_at;

-- name: GetFrontdoorSyncTask :one
SELECT frontdoor_sync_task_id, frontdoor_sync_task_entity_id, frontdoor_sync_task_type, frontdoor_sync_task_status, frontdoor_sync_task_priority, frontdoor_sync_task_max_attempts, frontdoor_sync_task_attempts, frontdoor_sync_task_last_error, frontdoor_sync_task_created_at, frontdoor_sync_task_started_at, frontdoor_sync_task_completed_at
FROM public.frontdoor_sync_tasks
WHERE frontdoor_sync_task_id = $1;

-- name: UpdateFrontdoorSyncTaskToProcessing :exec
UPDATE public.frontdoor_sync_tasks
SET frontdoor_sync_task_status = 'processing',
    frontdoor_sync_task_started_at = NOW(),
    frontdoor_sync_task_attempts = frontdoor_sync_task_attempts + 1
WHERE frontdoor_sync_task_id = $1;

-- name: UpdateFrontdoorSyncTaskToCompleted :exec
UPDATE public.frontdoor_sync_tasks
SET frontdoor_sync_task_status = 'completed',
    frontdoor_sync_task_completed_at = NOW()
WHERE frontdoor_sync_task_id = $1;

-- name: UpdateFrontdoorSyncTaskToFailed :exec
UPDATE public.frontdoor_sync_tasks
SET frontdoor_sync_task_status = 'failed',
    frontdoor_sync_task_completed_at = NOW(),
    frontdoor_sync_task_last_error = $2
WHERE frontdoor_sync_task_id = $1;

-- name: ResetFrontdoorSyncTaskToPending :exec
UPDATE public.frontdoor_sync_tasks
SET frontdoor_sync_task_status = 'pending',
    frontdoor_sync_task_last_error = $2
WHERE frontdoor_sync_task_id = $1;
