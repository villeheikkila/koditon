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
