-- name: UpsertShortcutSyncTask :one
INSERT INTO public.shortcut_sync_tasks (
    shortcut_sync_task_entity_id,
    shortcut_sync_task_type,
    shortcut_sync_task_priority,
    shortcut_sync_task_max_attempts
) VALUES ($1, $2, $3, $4)
ON CONFLICT (shortcut_sync_task_entity_id, shortcut_sync_task_type) DO UPDATE
SET shortcut_sync_task_status = 'pending',
    shortcut_sync_task_priority = EXCLUDED.shortcut_sync_task_priority,
    shortcut_sync_task_max_attempts = EXCLUDED.shortcut_sync_task_max_attempts,
    shortcut_sync_task_attempts = 0,
    shortcut_sync_task_last_error = NULL,
    shortcut_sync_task_started_at = NULL,
    shortcut_sync_task_completed_at = NULL
WHERE shortcut_sync_tasks.shortcut_sync_task_status IN ('completed', 'failed')
RETURNING shortcut_sync_task_id, shortcut_sync_task_entity_id, shortcut_sync_task_type, shortcut_sync_task_status, shortcut_sync_task_priority, shortcut_sync_task_max_attempts, shortcut_sync_task_attempts, shortcut_sync_task_last_error, shortcut_sync_task_created_at, shortcut_sync_task_started_at, shortcut_sync_task_completed_at;

-- name: GetShortcutSyncTask :one
SELECT shortcut_sync_task_id, shortcut_sync_task_entity_id, shortcut_sync_task_type, shortcut_sync_task_status, shortcut_sync_task_priority, shortcut_sync_task_max_attempts, shortcut_sync_task_attempts, shortcut_sync_task_last_error, shortcut_sync_task_created_at, shortcut_sync_task_started_at, shortcut_sync_task_completed_at
FROM public.shortcut_sync_tasks
WHERE shortcut_sync_task_id = $1;

-- name: UpdateShortcutSyncTaskToProcessing :exec
UPDATE public.shortcut_sync_tasks
SET shortcut_sync_task_status = 'processing',
    shortcut_sync_task_started_at = NOW(),
    shortcut_sync_task_attempts = shortcut_sync_task_attempts + 1
WHERE shortcut_sync_task_id = $1;

-- name: UpdateShortcutSyncTaskToCompleted :exec
UPDATE public.shortcut_sync_tasks
SET shortcut_sync_task_status = 'completed',
    shortcut_sync_task_completed_at = NOW()
WHERE shortcut_sync_task_id = $1;

-- name: UpdateShortcutSyncTaskToFailed :exec
UPDATE public.shortcut_sync_tasks
SET shortcut_sync_task_status = 'failed',
    shortcut_sync_task_completed_at = NOW(),
    shortcut_sync_task_last_error = $2
WHERE shortcut_sync_task_id = $1;

-- name: ResetShortcutSyncTaskToPending :exec
UPDATE public.shortcut_sync_tasks
SET shortcut_sync_task_status = 'pending',
    shortcut_sync_task_last_error = $2
WHERE shortcut_sync_task_id = $1;
