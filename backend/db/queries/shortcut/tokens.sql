-- name: GetValidShortcutToken :one
SELECT shortcut_token_id, shortcut_token_cuid, shortcut_token_token, shortcut_token_loaded, shortcut_token_created_at, shortcut_token_updated_at, shortcut_token_expires_at FROM origin.shortcut_tokens
ORDER BY shortcut_token_created_at DESC
LIMIT 1;

-- name: GetAllValidShortcutTokens :many
SELECT shortcut_token_id, shortcut_token_cuid, shortcut_token_token, shortcut_token_loaded, shortcut_token_created_at, shortcut_token_updated_at, shortcut_token_expires_at FROM origin.shortcut_tokens
ORDER BY shortcut_token_created_at DESC;

-- name: InsertShortcutToken :one
INSERT INTO origin.shortcut_tokens (
    shortcut_token_cuid,
    shortcut_token_token,
    shortcut_token_loaded,
    shortcut_token_expires_at
) VALUES (
    @shortcut_token_cuid,
    @shortcut_token_token,
    @shortcut_token_loaded,
    @shortcut_token_expires_at
)
ON CONFLICT (shortcut_token_cuid) DO UPDATE SET
    shortcut_token_token = EXCLUDED.shortcut_token_token,
    shortcut_token_loaded = EXCLUDED.shortcut_token_loaded,
    shortcut_token_expires_at = EXCLUDED.shortcut_token_expires_at,
    shortcut_token_updated_at = NOW()
RETURNING shortcut_token_id, shortcut_token_cuid, shortcut_token_token, shortcut_token_loaded, shortcut_token_created_at, shortcut_token_updated_at, shortcut_token_expires_at;

-- name: DeleteShortcutToken :exec
DELETE FROM origin.shortcut_tokens
WHERE shortcut_token_cuid = $1;
