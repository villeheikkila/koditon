-- name: CreateUser :one
INSERT INTO auth.users DEFAULT VALUES
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM auth.users
WHERE user_id = $1 AND user_deleted_at IS NULL;

-- name: DeleteUser :exec
UPDATE auth.users
SET user_deleted_at = now(), user_updated_at = now()
WHERE user_id = $1;

-- name: GetIdentityByProviderAndExternalID :one
SELECT * FROM auth.identities
WHERE identity_provider = $1 AND identity_external_id = $2;

-- name: CreateIdentity :one
INSERT INTO auth.identities (
    user_id,
    identity_provider,
    identity_external_id,
    identity_email,
    identity_email_verified,
    identity_data
) VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: UpdateIdentity :one
UPDATE auth.identities
SET
    identity_email = COALESCE($2, identity_email),
    identity_email_verified = COALESCE($3, identity_email_verified),
    identity_data = COALESCE($4, identity_data),
    identity_updated_at = now()
WHERE identity_id = $1
RETURNING *;

-- name: GetIdentitiesByUserID :many
SELECT * FROM auth.identities
WHERE user_id = $1
ORDER BY identity_created_at ASC;

-- name: CreateSession :one
INSERT INTO auth.sessions (
    user_id,
    session_device_id,
    session_user_agent,
    session_ip,
    session_provider,
    session_refresh_token_hmac_key,
    session_not_after
) VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetSessionByID :one
SELECT * FROM auth.sessions
WHERE session_id = $1;

-- name: GetActiveSessionByID :one
SELECT * FROM auth.sessions
WHERE session_id = $1
  AND session_revoked_at IS NULL
  AND (session_not_after IS NULL OR session_not_after > now());

-- name: GetSessionsByUserID :many
SELECT * FROM auth.sessions
WHERE user_id = $1
  AND session_revoked_at IS NULL
ORDER BY session_created_at DESC;

-- name: UpdateSessionRefreshed :one
UPDATE auth.sessions
SET
    session_refreshed_at = now(),
    session_updated_at = now(),
    session_refresh_token_counter = session_refresh_token_counter + 1
WHERE session_id = $1
RETURNING *;

-- name: RevokeSession :exec
UPDATE auth.sessions
SET session_revoked_at = now(), session_updated_at = now()
WHERE session_id = $1;

-- name: RevokeAllUserSessions :exec
UPDATE auth.sessions
SET session_revoked_at = now(), session_updated_at = now()
WHERE user_id = $1 AND session_revoked_at IS NULL;

-- name: RevokeAllUserSessionsExcept :exec
UPDATE auth.sessions
SET session_revoked_at = now(), session_updated_at = now()
WHERE user_id = $1 AND session_id != $2 AND session_revoked_at IS NULL;

-- name: CreateRefreshToken :one
INSERT INTO auth.refresh_tokens (
    session_id,
    refresh_token_token_hash,
    refresh_token_counter
) VALUES ($1, $2, $3)
RETURNING *;

-- name: GetRefreshTokenByHash :one
SELECT * FROM auth.refresh_tokens
WHERE refresh_token_token_hash = $1;

-- name: GetRefreshTokenBySessionAndCounter :one
SELECT * FROM auth.refresh_tokens
WHERE session_id = $1 AND refresh_token_counter = $2;

-- name: RevokeRefreshToken :exec
UPDATE auth.refresh_tokens
SET refresh_token_revoked = true, refresh_token_updated_at = now()
WHERE refresh_token_id = $1;

-- name: RevokeAllSessionRefreshTokens :exec
UPDATE auth.refresh_tokens
SET refresh_token_revoked = true, refresh_token_updated_at = now()
WHERE session_id = $1 AND refresh_token_revoked = false;

-- name: CleanupExpiredSessions :exec
UPDATE auth.sessions
SET session_revoked_at = now(), session_updated_at = now()
WHERE session_not_after < now() AND session_revoked_at IS NULL;

-- name: CleanupRevokedRefreshTokens :exec
DELETE FROM auth.refresh_tokens
WHERE refresh_token_revoked = true AND refresh_token_updated_at < now() - interval '30 days';

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

-- name: GetUserRoles :many
SELECT r.* FROM auth.roles r
JOIN auth.user_roles ur ON ur.role_id = r.role_id
WHERE ur.user_id = $1
ORDER BY r.role_name;

-- name: GetActiveFeatureFlags :many
SELECT DISTINCT f.flag_name
FROM auth.feature_flags f
WHERE (
    f.flag_default_enabled = true
    OR EXISTS (
        SELECT 1 FROM auth.role_feature_flags rff
        JOIN auth.user_roles ur ON ur.role_id = rff.role_id
        WHERE rff.flag_id = f.flag_id AND ur.user_id = $1
    )
    OR EXISTS (
        SELECT 1 FROM auth.user_feature_flags uff
        WHERE uff.flag_id = f.flag_id AND uff.user_id = $1 AND uff.user_flag_enabled = true
    )
)
AND NOT EXISTS (
    SELECT 1 FROM auth.user_feature_flags uff
    WHERE uff.flag_id = f.flag_id AND uff.user_id = $1 AND uff.user_flag_enabled = false
);

-- name: HasFeatureFlag :one
SELECT EXISTS (
    SELECT 1 FROM auth.feature_flags f
    WHERE f.flag_name = $2
    AND (
        f.flag_default_enabled = true
        OR EXISTS (
            SELECT 1 FROM auth.role_feature_flags rff
            JOIN auth.user_roles ur ON ur.role_id = rff.role_id
            WHERE rff.flag_id = f.flag_id AND ur.user_id = $1
        )
        OR EXISTS (
            SELECT 1 FROM auth.user_feature_flags uff
            WHERE uff.flag_id = f.flag_id AND uff.user_id = $1 AND uff.user_flag_enabled = true
        )
    )
    AND NOT EXISTS (
        SELECT 1 FROM auth.user_feature_flags uff
        WHERE uff.flag_id = f.flag_id AND uff.user_id = $1 AND uff.user_flag_enabled = false
    )
) AS has_flag;

-- name: HasRole :one
SELECT EXISTS (
    SELECT 1 FROM auth.user_roles ur
    JOIN auth.roles r ON r.role_id = ur.role_id
    WHERE ur.user_id = $1 AND r.role_name = $2
) AS has_role;
