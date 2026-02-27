-- name: InsertPricesPendingTask :one
INSERT INTO public.prices_pending_tasks (
    prices_pending_task_entity_id,
    prices_pending_task_type,
    prices_pending_task_priority,
    prices_pending_task_max_attempts
) VALUES ($1, $2, $3, $4)
ON CONFLICT (prices_pending_task_entity_id, prices_pending_task_type) DO NOTHING
RETURNING prices_pending_task_id, prices_pending_task_entity_id, prices_pending_task_type, prices_pending_task_priority, prices_pending_task_max_attempts, prices_pending_task_created_at;

-- name: GetPricesPendingTask :one
SELECT prices_pending_task_id, prices_pending_task_entity_id, prices_pending_task_type, prices_pending_task_priority, prices_pending_task_max_attempts, prices_pending_task_created_at
FROM public.prices_pending_tasks
WHERE prices_pending_task_id = $1;

-- name: DeletePricesPendingTask :exec
DELETE FROM public.prices_pending_tasks
WHERE prices_pending_task_id = $1;
