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
