-- name: UpsertShortcutPendingTask :one
INSERT INTO public.shortcut_pending_tasks (
    shortcut_pending_task_entity_id,
    shortcut_pending_task_type,
    shortcut_pending_task_priority,
    shortcut_pending_task_max_attempts
) VALUES ($1, $2, $3, $4)
ON CONFLICT (shortcut_pending_task_entity_id, shortcut_pending_task_type) DO UPDATE
SET shortcut_pending_task_status = 'pending',
    shortcut_pending_task_priority = EXCLUDED.shortcut_pending_task_priority,
    shortcut_pending_task_max_attempts = EXCLUDED.shortcut_pending_task_max_attempts,
    shortcut_pending_task_attempts = 0,
    shortcut_pending_task_last_error = NULL,
    shortcut_pending_task_started_at = NULL,
    shortcut_pending_task_completed_at = NULL
WHERE shortcut_pending_tasks.shortcut_pending_task_status IN ('completed', 'failed')
RETURNING shortcut_pending_task_id, shortcut_pending_task_entity_id, shortcut_pending_task_type, shortcut_pending_task_status, shortcut_pending_task_priority, shortcut_pending_task_max_attempts, shortcut_pending_task_attempts, shortcut_pending_task_last_error, shortcut_pending_task_created_at, shortcut_pending_task_started_at, shortcut_pending_task_completed_at;

-- name: GetShortcutPendingTask :one
SELECT shortcut_pending_task_id, shortcut_pending_task_entity_id, shortcut_pending_task_type, shortcut_pending_task_status, shortcut_pending_task_priority, shortcut_pending_task_max_attempts, shortcut_pending_task_attempts, shortcut_pending_task_last_error, shortcut_pending_task_created_at, shortcut_pending_task_started_at, shortcut_pending_task_completed_at
FROM public.shortcut_pending_tasks
WHERE shortcut_pending_task_id = $1;

-- name: UpdateShortcutPendingTaskToProcessing :exec
UPDATE public.shortcut_pending_tasks
SET shortcut_pending_task_status = 'processing',
    shortcut_pending_task_started_at = NOW(),
    shortcut_pending_task_attempts = shortcut_pending_task_attempts + 1
WHERE shortcut_pending_task_id = $1;

-- name: UpdateShortcutPendingTaskToCompleted :exec
UPDATE public.shortcut_pending_tasks
SET shortcut_pending_task_status = 'completed',
    shortcut_pending_task_completed_at = NOW()
WHERE shortcut_pending_task_id = $1;

-- name: UpdateShortcutPendingTaskToFailed :exec
UPDATE public.shortcut_pending_tasks
SET shortcut_pending_task_status = 'failed',
    shortcut_pending_task_completed_at = NOW(),
    shortcut_pending_task_last_error = $2
WHERE shortcut_pending_task_id = $1;

-- name: ResetShortcutPendingTaskToPending :exec
UPDATE public.shortcut_pending_tasks
SET shortcut_pending_task_status = 'pending',
    shortcut_pending_task_last_error = $2
WHERE shortcut_pending_task_id = $1;
