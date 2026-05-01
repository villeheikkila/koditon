-- name: CreateOAuthAuthorizationCode :one
insert into oauth_authorization_codes (
  oauth_authorization_code_code_hash,
  oauth_client_id,
  user_uuid,
  oauth_authorization_code_redirect_uri,
  oauth_authorization_code_scopes,
  oauth_authorization_code_audience,
  oauth_authorization_code_code_challenge,
  oauth_authorization_code_code_challenge_method,
  oauth_authorization_code_expires_at
) values (
  sqlc.arg(oauth_authorization_code_code_hash),
  sqlc.arg(oauth_client_id),
  sqlc.arg(user_uuid),
  sqlc.arg(oauth_authorization_code_redirect_uri),
  sqlc.arg(oauth_authorization_code_scopes),
  sqlc.arg(oauth_authorization_code_audience),
  sqlc.arg(oauth_authorization_code_code_challenge),
  sqlc.arg(oauth_authorization_code_code_challenge_method),
  sqlc.arg(oauth_authorization_code_expires_at)
)
returning
  oauth_authorization_code_id,
  oauth_authorization_code_code_hash,
  oauth_client_id,
  user_uuid,
  oauth_authorization_code_redirect_uri,
  oauth_authorization_code_scopes,
  oauth_authorization_code_audience,
  oauth_authorization_code_code_challenge,
  oauth_authorization_code_code_challenge_method,
  oauth_authorization_code_expires_at,
  oauth_authorization_code_consumed_at,
  oauth_authorization_code_created_at,
  oauth_authorization_code_updated_at;

-- name: ConsumeOAuthAuthorizationCode :one
update oauth_authorization_codes
set
  oauth_authorization_code_consumed_at = now(),
  oauth_authorization_code_updated_at = now()
where
  oauth_authorization_code_code_hash = sqlc.arg(oauth_authorization_code_code_hash)
  and oauth_client_id = sqlc.arg(oauth_client_id)
  and oauth_authorization_code_redirect_uri = sqlc.arg(oauth_authorization_code_redirect_uri)
  and (
    oauth_authorization_code_audience = sqlc.arg(oauth_authorization_code_audience)
    or oauth_authorization_code_audience = ''
  )
  and oauth_authorization_code_code_challenge = sqlc.arg(oauth_authorization_code_code_challenge)
  and oauth_authorization_code_code_challenge_method = sqlc.arg(oauth_authorization_code_code_challenge_method)
  and oauth_authorization_code_consumed_at is null
  and oauth_authorization_code_expires_at > now()
returning
  oauth_authorization_code_id,
  oauth_authorization_code_code_hash,
  oauth_client_id,
  user_uuid,
  oauth_authorization_code_redirect_uri,
  oauth_authorization_code_scopes,
  oauth_authorization_code_audience,
  oauth_authorization_code_code_challenge,
  oauth_authorization_code_code_challenge_method,
  oauth_authorization_code_expires_at,
  oauth_authorization_code_consumed_at,
  oauth_authorization_code_created_at,
  oauth_authorization_code_updated_at;

-- name: CreateOAuthRefreshToken :one
insert into oauth_refresh_tokens (
  oauth_refresh_token_token_hash,
  oauth_client_id,
  user_uuid,
  device_session_uuid,
  oauth_refresh_token_scopes,
  oauth_refresh_token_audience,
  oauth_refresh_token_expires_at,
  oauth_refresh_token_rotated_from
) values (
  sqlc.arg(oauth_refresh_token_token_hash),
  sqlc.arg(oauth_client_id),
  sqlc.arg(user_uuid),
  sqlc.arg(device_session_uuid),
  sqlc.arg(oauth_refresh_token_scopes),
  sqlc.arg(oauth_refresh_token_audience),
  sqlc.arg(oauth_refresh_token_expires_at),
  sqlc.arg(oauth_refresh_token_rotated_from)
)
returning
  oauth_refresh_token_id,
  oauth_refresh_token_token_hash,
  oauth_client_id,
  user_uuid,
  device_session_uuid,
  oauth_refresh_token_scopes,
  oauth_refresh_token_audience,
  oauth_refresh_token_expires_at,
  oauth_refresh_token_revoked_at,
  oauth_refresh_token_rotated_from,
  oauth_refresh_token_created_at,
  oauth_refresh_token_updated_at;

-- name: GetOAuthRefreshTokenByHashForUpdate :one
select
  oauth_refresh_token_id,
  oauth_refresh_token_token_hash,
  oauth_client_id,
  user_uuid,
  device_session_uuid,
  oauth_refresh_token_scopes,
  oauth_refresh_token_audience,
  oauth_refresh_token_expires_at,
  oauth_refresh_token_revoked_at,
  oauth_refresh_token_rotated_from,
  oauth_refresh_token_created_at,
  oauth_refresh_token_updated_at
from
  oauth_refresh_tokens
where
  oauth_refresh_token_token_hash = sqlc.arg(oauth_refresh_token_token_hash)
for update;

-- name: RevokeOAuthRefreshTokenByHash :one
update oauth_refresh_tokens
set
  oauth_refresh_token_revoked_at = now(),
  oauth_refresh_token_updated_at = now()
where
  oauth_refresh_token_token_hash = sqlc.arg(oauth_refresh_token_token_hash)
  and oauth_refresh_token_revoked_at is null
  and oauth_refresh_token_expires_at > now()
returning
  oauth_refresh_token_id,
  oauth_refresh_token_token_hash,
  oauth_client_id,
  user_uuid,
  device_session_uuid,
  oauth_refresh_token_scopes,
  oauth_refresh_token_audience,
  oauth_refresh_token_expires_at,
  oauth_refresh_token_revoked_at,
  oauth_refresh_token_rotated_from,
  oauth_refresh_token_created_at,
  oauth_refresh_token_updated_at;

-- name: RevokeOAuthRefreshTokenByHashAndClientID :one
update oauth_refresh_tokens
set
  oauth_refresh_token_revoked_at = now(),
  oauth_refresh_token_updated_at = now()
where
  oauth_refresh_token_token_hash = sqlc.arg(oauth_refresh_token_token_hash)
  and oauth_client_id = sqlc.arg(oauth_client_id)
  and oauth_refresh_token_revoked_at is null
  and oauth_refresh_token_expires_at > now()
returning
  oauth_refresh_token_id,
  oauth_refresh_token_token_hash,
  oauth_client_id,
  user_uuid,
  device_session_uuid,
  oauth_refresh_token_scopes,
  oauth_refresh_token_audience,
  oauth_refresh_token_expires_at,
  oauth_refresh_token_revoked_at,
  oauth_refresh_token_rotated_from,
  oauth_refresh_token_created_at,
  oauth_refresh_token_updated_at;

-- name: RevokeAllOAuthRefreshTokensByUserID :exec
update oauth_refresh_tokens
set
  oauth_refresh_token_revoked_at = now(),
  oauth_refresh_token_updated_at = now()
where
  user_uuid = sqlc.arg(user_uuid)
  and oauth_refresh_token_revoked_at is null
  and oauth_refresh_token_expires_at > now();

-- name: ListOAuthAppConnectionsByUserID :many
with active_tokens as (
  select
    ort.oauth_client_id,
    ort.oauth_refresh_token_scopes,
    ort.oauth_refresh_token_created_at,
    ort.oauth_refresh_token_updated_at,
    odc.oauth_dynamic_client_name
  from oauth_refresh_tokens ort
  left join oauth_dynamic_clients odc
    on odc.oauth_dynamic_client_id = ort.oauth_client_id
   and odc.oauth_dynamic_client_disabled_at is null
  where
    ort.user_uuid = sqlc.arg(user_uuid)
    and ort.oauth_refresh_token_revoked_at is null
    and ort.oauth_refresh_token_expires_at > now()
)
select
  active_tokens.oauth_client_id,
  coalesce(max(nullif(btrim(active_tokens.oauth_dynamic_client_name), '')), '')::text as oauth_dynamic_client_name,
  min(active_tokens.oauth_refresh_token_created_at)::timestamptz as connected_at,
  max(active_tokens.oauth_refresh_token_updated_at)::timestamptz as last_used_at,
  coalesce(array(
    select distinct scope
    from active_tokens grouped
    cross join lateral unnest(grouped.oauth_refresh_token_scopes) as scope
    where grouped.oauth_client_id = active_tokens.oauth_client_id
    order by scope
  ), '{}'::text[])::text[] as scopes
from active_tokens
group by active_tokens.oauth_client_id
order by max(active_tokens.oauth_refresh_token_updated_at) desc, active_tokens.oauth_client_id asc;

-- name: RevokeAllOAuthRefreshTokensByUserIDAndClientID :execrows
update oauth_refresh_tokens
set
  oauth_refresh_token_revoked_at = now(),
  oauth_refresh_token_updated_at = now()
where
  user_uuid = sqlc.arg(user_uuid)
  and oauth_client_id = sqlc.arg(oauth_client_id)
  and oauth_refresh_token_revoked_at is null
  and oauth_refresh_token_expires_at > now();
