-- name: InsertShortcutPendingTask :one
INSERT INTO public.shortcut_pending_tasks (
    shortcut_pending_task_entity_id,
    shortcut_pending_task_type,
    shortcut_pending_task_priority,
    shortcut_pending_task_max_attempts
) VALUES ($1, $2, $3, $4)
ON CONFLICT (shortcut_pending_task_entity_id, shortcut_pending_task_type) DO NOTHING
RETURNING shortcut_pending_task_id, shortcut_pending_task_entity_id, shortcut_pending_task_type, shortcut_pending_task_priority, shortcut_pending_task_max_attempts, shortcut_pending_task_created_at;

-- name: GetShortcutPendingTask :one
SELECT shortcut_pending_task_id, shortcut_pending_task_entity_id, shortcut_pending_task_type, shortcut_pending_task_priority, shortcut_pending_task_max_attempts, shortcut_pending_task_created_at
FROM public.shortcut_pending_tasks
WHERE shortcut_pending_task_id = $1;

-- name: DeleteShortcutPendingTask :exec
DELETE FROM public.shortcut_pending_tasks
WHERE shortcut_pending_task_id = $1;
