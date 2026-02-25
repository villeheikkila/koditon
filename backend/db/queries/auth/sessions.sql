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
