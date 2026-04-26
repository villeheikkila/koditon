-- name: GetRuntimeKV :one
SELECT kv_value
FROM runtime.kv_store
WHERE kv_key = $1
  AND expires_at > now();

-- name: UpsertRuntimeKV :exec
INSERT INTO runtime.kv_store (
  kv_key,
  kv_value,
  expires_at
) VALUES (
  $1,
  $2,
  $3
)
ON CONFLICT (kv_key) DO UPDATE SET
  kv_value = EXCLUDED.kv_value,
  expires_at = EXCLUDED.expires_at,
  updated_at = now();

-- name: DeleteExpiredRuntimeKV :exec
DELETE FROM runtime.kv_store
WHERE expires_at <= now();

-- name: GetSchemaVersion :one
SELECT version
FROM public.schema_migrations
ORDER BY version DESC
LIMIT 1;
