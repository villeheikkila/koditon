-- name: InsertPostalPendingTask :one
INSERT INTO public.postal_pending_tasks (
    postal_pending_task_entity_id,
    postal_pending_task_type,
    postal_pending_task_priority,
    postal_pending_task_max_attempts
) VALUES ($1, $2, $3, $4)
ON CONFLICT (postal_pending_task_entity_id, postal_pending_task_type) DO NOTHING
RETURNING postal_pending_task_id, postal_pending_task_entity_id, postal_pending_task_type, postal_pending_task_priority, postal_pending_task_max_attempts, postal_pending_task_created_at;

-- name: GetPostalPendingTask :one
SELECT postal_pending_task_id, postal_pending_task_entity_id, postal_pending_task_type, postal_pending_task_priority, postal_pending_task_max_attempts, postal_pending_task_created_at
FROM public.postal_pending_tasks
WHERE postal_pending_task_id = $1;

-- name: DeletePostalPendingTask :exec
DELETE FROM public.postal_pending_tasks
WHERE postal_pending_task_id = $1;
