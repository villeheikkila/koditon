-- name: GetValidShortcutToken :one
SELECT * FROM public.shortcut_tokens
ORDER BY shortcut_token_created_at DESC
LIMIT 1;

-- name: GetAllValidShortcutTokens :many
SELECT * FROM public.shortcut_tokens
ORDER BY shortcut_token_created_at DESC;

-- name: InsertShortcutToken :one
INSERT INTO public.shortcut_tokens (
    shortcut_token_cuid,
    shortcut_token_token,
    shortcut_token_loaded,
    shortcut_token_expires_at
) VALUES (
    $1, $2, $3, $4
)
ON CONFLICT (shortcut_token_cuid) DO UPDATE SET
    shortcut_token_token = EXCLUDED.shortcut_token_token,
    shortcut_token_loaded = EXCLUDED.shortcut_token_loaded,
    shortcut_token_expires_at = EXCLUDED.shortcut_token_expires_at,
    shortcut_token_updated_at = NOW()
RETURNING *;

-- name: DeleteShortcutToken :exec
DELETE FROM public.shortcut_tokens
WHERE shortcut_token_cuid = $1;
