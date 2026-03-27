-- name: CreatePersonalAccessToken :one
insert into personal_access_tokens (user_id, personal_access_token_name, personal_access_token_prefix, personal_access_token_token_hash, personal_access_token_scopes, personal_access_token_expires_at)
  values (sqlc.arg (user_id), sqlc.arg (personal_access_token_name), sqlc.arg (personal_access_token_prefix), sqlc.arg (personal_access_token_token_hash), sqlc.arg (personal_access_token_scopes), sqlc.arg (personal_access_token_expires_at))
returning
  personal_access_token_id, user_id, personal_access_token_name, personal_access_token_prefix, personal_access_token_token_hash, personal_access_token_scopes, personal_access_token_created_at, personal_access_token_last_used_at, personal_access_token_expires_at, personal_access_token_revoked_at;

-- name: GetPersonalAccessTokenByPrefix :one
select
  personal_access_token_id,
  u.user_id,
  u.user_uuid,
  personal_access_token_name,
  personal_access_token_prefix,
  personal_access_token_token_hash,
  personal_access_token_scopes,
  personal_access_token_created_at,
  personal_access_token_last_used_at,
  personal_access_token_expires_at,
  personal_access_token_revoked_at
from
  personal_access_tokens pat
  join users u on u.user_id = pat.user_id
where
  personal_access_token_prefix = sqlc.arg (personal_access_token_prefix);

-- name: UpdatePersonalAccessTokenLastUsed :exec
update
  personal_access_tokens
set
  personal_access_token_last_used_at = now()
where
  personal_access_token_id = sqlc.arg (personal_access_token_id);

-- name: RevokePersonalAccessToken :exec
update
  personal_access_tokens
set
  personal_access_token_revoked_at = now()
where
  personal_access_token_id = sqlc.arg (personal_access_token_id)
  and personal_access_token_revoked_at is null;

