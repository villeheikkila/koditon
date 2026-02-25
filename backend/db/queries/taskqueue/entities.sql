-- name: GetEntity :one
SELECT
    entity_id,
    entity_type,
    status,
    scheduling_strategy,
    metadata,
    created_at,
    updated_at
FROM task_queue.entity_registry
WHERE entity_id = $1;

-- name: ListEntities :many
SELECT
    entity_id,
    entity_type,
    status,
    scheduling_strategy,
    metadata,
    created_at,
    updated_at
FROM task_queue.entity_registry
ORDER BY entity_id;

-- name: ListActiveEntities :many
SELECT
    entity_id,
    entity_type,
    status,
    scheduling_strategy,
    metadata,
    created_at,
    updated_at
FROM task_queue.entity_registry
WHERE status = 'active'
ORDER BY entity_id;

-- name: UpsertEntity :one
INSERT INTO task_queue.entity_registry (
    entity_id,
    entity_type,
    status,
    scheduling_strategy,
    metadata,
    created_at,
    updated_at
) VALUES (
    $1,
    $2,
    COALESCE($3, 'active'),
    COALESCE($4, 'manual'),
    COALESCE($5, '{}'::jsonb),
    NOW(),
    NOW()
)
ON CONFLICT (entity_id) DO UPDATE
SET
    entity_type = EXCLUDED.entity_type,
    status = EXCLUDED.status,
    scheduling_strategy = EXCLUDED.scheduling_strategy,
    metadata = EXCLUDED.metadata,
    updated_at = NOW()
RETURNING *;

-- name: UpdateEntityStatus :exec
UPDATE task_queue.entity_registry
SET
    status = $2,
    updated_at = NOW()
WHERE entity_id = $1;

-- name: DeleteEntity :exec
DELETE FROM task_queue.entity_registry
WHERE entity_id = $1;

-- name: CountEntitiesByStatus :one
SELECT COUNT(*) AS count
FROM task_queue.entity_registry
WHERE status = $1;
