-- name: CreateDevice :one
INSERT INTO auth.devices (
    user_id,
    device_name,
    device_os,
    device_app_version,
    device_push_token,
    device_push_token_type,
    device_push_token_updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: UpsertDevice :one
INSERT INTO auth.devices (
    device_id,
    user_id,
    device_name,
    device_os,
    device_app_version
) VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (device_id) DO UPDATE SET
    device_last_seen_at = now(),
    device_updated_at = now()
RETURNING *;

-- name: GetDeviceByID :one
SELECT * FROM auth.devices
WHERE device_id = $1;

-- name: GetDevicesByUserID :many
SELECT * FROM auth.devices
WHERE user_id = $1
ORDER BY device_last_seen_at DESC;

-- name: UpdateDevice :one
UPDATE auth.devices
SET
    device_name = COALESCE($2, device_name),
    device_os = COALESCE($3, device_os),
    device_app_version = COALESCE($4, device_app_version),
    device_updated_at = now()
WHERE device_id = $1
RETURNING *;

-- name: UpdateDevicePushToken :one
UPDATE auth.devices
SET
    device_push_token = $2,
    device_push_token_type = $3,
    device_push_token_updated_at = now(),
    device_updated_at = now()
WHERE device_id = $1
RETURNING *;

-- name: UpdateDeviceLastSeen :exec
UPDATE auth.devices
SET device_last_seen_at = now()
WHERE device_id = $1;

-- name: DeleteDevice :exec
DELETE FROM auth.devices
WHERE device_id = $1;

-- name: DeleteDevicesByUserID :exec
DELETE FROM auth.devices
WHERE user_id = $1;
