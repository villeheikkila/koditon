-- name: InsertFrontdoorPendingTask :one
INSERT INTO public.frontdoor_pending_tasks (
    frontdoor_pending_task_entity_id,
    frontdoor_pending_task_type,
    frontdoor_pending_task_priority,
    frontdoor_pending_task_max_attempts
) VALUES ($1, $2, $3, $4)
ON CONFLICT (frontdoor_pending_task_entity_id, frontdoor_pending_task_type) DO NOTHING
RETURNING frontdoor_pending_task_id, frontdoor_pending_task_entity_id, frontdoor_pending_task_type, frontdoor_pending_task_priority, frontdoor_pending_task_max_attempts, frontdoor_pending_task_created_at;

-- name: GetFrontdoorPendingTask :one
SELECT frontdoor_pending_task_id, frontdoor_pending_task_entity_id, frontdoor_pending_task_type, frontdoor_pending_task_priority, frontdoor_pending_task_max_attempts, frontdoor_pending_task_created_at
FROM public.frontdoor_pending_tasks
WHERE frontdoor_pending_task_id = $1;

-- name: DeleteFrontdoorPendingTask :exec
DELETE FROM public.frontdoor_pending_tasks
WHERE frontdoor_pending_task_id = $1;
