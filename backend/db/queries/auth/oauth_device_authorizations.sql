-- name: CreateOAuthDeviceAuthorization :one
insert into oauth_device_authorizations (
  oauth_device_authorization_device_code_hash,
  oauth_client_id,
  oauth_device_authorization_user_code,
  oauth_device_authorization_scopes,
  oauth_device_authorization_audience,
  oauth_device_authorization_expires_at
) values (
  sqlc.arg(oauth_device_authorization_device_code_hash),
  sqlc.arg(oauth_client_id),
  sqlc.arg(oauth_device_authorization_user_code),
  sqlc.arg(oauth_device_authorization_scopes),
  sqlc.arg(oauth_device_authorization_audience),
  sqlc.arg(oauth_device_authorization_expires_at)
)
returning
  oauth_device_authorization_id,
  oauth_device_authorization_device_code_hash,
  oauth_client_id,
  oauth_device_authorization_user_code,
  oauth_device_authorization_scopes,
  oauth_device_authorization_audience,
  user_uuid,
  oauth_device_authorization_expires_at,
  oauth_device_authorization_approved_at,
  oauth_device_authorization_denied_at,
  oauth_device_authorization_consumed_at,
  oauth_device_authorization_created_at,
  oauth_device_authorization_updated_at;

-- name: GetOAuthDeviceAuthorizationByUserCode :one
select
  oauth_device_authorization_id,
  oauth_device_authorization_device_code_hash,
  oauth_client_id,
  oauth_device_authorization_user_code,
  oauth_device_authorization_scopes,
  oauth_device_authorization_audience,
  user_uuid,
  oauth_device_authorization_expires_at,
  oauth_device_authorization_approved_at,
  oauth_device_authorization_denied_at,
  oauth_device_authorization_consumed_at,
  oauth_device_authorization_created_at,
  oauth_device_authorization_updated_at
from oauth_device_authorizations
where oauth_device_authorization_user_code = sqlc.arg(oauth_device_authorization_user_code);

-- name: ApproveOAuthDeviceAuthorizationByUserCode :one
update oauth_device_authorizations
set
  user_uuid = sqlc.arg(user_uuid),
  oauth_device_authorization_approved_at = now(),
  oauth_device_authorization_updated_at = now()
where
  oauth_device_authorization_user_code = sqlc.arg(oauth_device_authorization_user_code)
  and oauth_device_authorization_approved_at is null
  and oauth_device_authorization_denied_at is null
  and oauth_device_authorization_consumed_at is null
  and oauth_device_authorization_expires_at > now()
returning
  oauth_device_authorization_id,
  oauth_device_authorization_device_code_hash,
  oauth_client_id,
  oauth_device_authorization_user_code,
  oauth_device_authorization_scopes,
  oauth_device_authorization_audience,
  user_uuid,
  oauth_device_authorization_expires_at,
  oauth_device_authorization_approved_at,
  oauth_device_authorization_denied_at,
  oauth_device_authorization_consumed_at,
  oauth_device_authorization_created_at,
  oauth_device_authorization_updated_at;

-- name: DenyOAuthDeviceAuthorizationByUserCode :one
update oauth_device_authorizations
set
  oauth_device_authorization_denied_at = now(),
  oauth_device_authorization_updated_at = now()
where
  oauth_device_authorization_user_code = sqlc.arg(oauth_device_authorization_user_code)
  and oauth_device_authorization_denied_at is null
  and oauth_device_authorization_consumed_at is null
  and oauth_device_authorization_expires_at > now()
returning
  oauth_device_authorization_id,
  oauth_device_authorization_device_code_hash,
  oauth_client_id,
  oauth_device_authorization_user_code,
  oauth_device_authorization_scopes,
  oauth_device_authorization_audience,
  user_uuid,
  oauth_device_authorization_expires_at,
  oauth_device_authorization_approved_at,
  oauth_device_authorization_denied_at,
  oauth_device_authorization_consumed_at,
  oauth_device_authorization_created_at,
  oauth_device_authorization_updated_at;

-- name: GetOAuthDeviceAuthorizationByDeviceCodeHash :one
select
  oauth_device_authorization_id,
  oauth_device_authorization_device_code_hash,
  oauth_client_id,
  oauth_device_authorization_user_code,
  oauth_device_authorization_scopes,
  oauth_device_authorization_audience,
  user_uuid,
  oauth_device_authorization_expires_at,
  oauth_device_authorization_approved_at,
  oauth_device_authorization_denied_at,
  oauth_device_authorization_consumed_at,
  oauth_device_authorization_created_at,
  oauth_device_authorization_updated_at
from oauth_device_authorizations
where oauth_device_authorization_device_code_hash = sqlc.arg(oauth_device_authorization_device_code_hash);

-- name: ConsumeOAuthDeviceAuthorizationByID :one
update oauth_device_authorizations
set
  oauth_device_authorization_consumed_at = now(),
  oauth_device_authorization_updated_at = now()
where
  oauth_device_authorization_id = sqlc.arg(oauth_device_authorization_id)
  and oauth_device_authorization_consumed_at is null
returning
  oauth_device_authorization_id,
  oauth_device_authorization_device_code_hash,
  oauth_client_id,
  oauth_device_authorization_user_code,
  oauth_device_authorization_scopes,
  user_uuid,
  oauth_device_authorization_expires_at,
  oauth_device_authorization_approved_at,
  oauth_device_authorization_denied_at,
  oauth_device_authorization_consumed_at,
  oauth_device_authorization_created_at,
  oauth_device_authorization_updated_at;
