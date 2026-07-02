-- name: CreateOAuthAuthorizationHandoff :one
insert into auth.oauth_authorization_handoffs (
  oauth_authorization_handoff_token_hash,
  oauth_authorization_handoff_user_code,
  oauth_client_id,
  oauth_authorization_handoff_redirect_uri,
  oauth_authorization_handoff_scopes,
  oauth_authorization_handoff_audience,
  oauth_authorization_handoff_state,
  oauth_authorization_handoff_code_challenge,
  oauth_authorization_handoff_code_challenge_method,
  oauth_authorization_handoff_expires_at
) values (
  sqlc.arg(oauth_authorization_handoff_token_hash),
  sqlc.arg(oauth_authorization_handoff_user_code),
  sqlc.arg(oauth_client_id),
  sqlc.arg(oauth_authorization_handoff_redirect_uri),
  sqlc.arg(oauth_authorization_handoff_scopes),
  sqlc.arg(oauth_authorization_handoff_audience),
  sqlc.arg(oauth_authorization_handoff_state),
  sqlc.arg(oauth_authorization_handoff_code_challenge),
  sqlc.arg(oauth_authorization_handoff_code_challenge_method),
  sqlc.arg(oauth_authorization_handoff_expires_at)
)
returning
  oauth_authorization_handoff_id,
  oauth_authorization_handoff_token_hash,
  oauth_authorization_handoff_user_code,
  oauth_client_id,
  oauth_authorization_handoff_redirect_uri,
  oauth_authorization_handoff_scopes,
  oauth_authorization_handoff_audience,
  oauth_authorization_handoff_state,
  oauth_authorization_handoff_code_challenge,
  oauth_authorization_handoff_code_challenge_method,
  user_uuid,
  oauth_authorization_handoff_authorization_code,
  oauth_authorization_handoff_redirect_url,
  oauth_authorization_handoff_denied_at,
  oauth_authorization_handoff_completed_at,
  oauth_authorization_handoff_expires_at,
  oauth_authorization_handoff_created_at,
  oauth_authorization_handoff_updated_at;

-- name: GetOAuthAuthorizationHandoffByID :one
select
  oauth_authorization_handoff_id,
  oauth_authorization_handoff_token_hash,
  oauth_authorization_handoff_user_code,
  oauth_client_id,
  oauth_authorization_handoff_redirect_uri,
  oauth_authorization_handoff_scopes,
  oauth_authorization_handoff_audience,
  oauth_authorization_handoff_state,
  oauth_authorization_handoff_code_challenge,
  oauth_authorization_handoff_code_challenge_method,
  user_uuid,
  oauth_authorization_handoff_authorization_code,
  oauth_authorization_handoff_redirect_url,
  oauth_authorization_handoff_denied_at,
  oauth_authorization_handoff_completed_at,
  oauth_authorization_handoff_expires_at,
  oauth_authorization_handoff_created_at,
  oauth_authorization_handoff_updated_at
from auth.oauth_authorization_handoffs
where oauth_authorization_handoff_id = sqlc.arg(oauth_authorization_handoff_id);

-- name: GetOAuthAuthorizationHandoffByTokenHash :one
select
  oauth_authorization_handoff_id,
  oauth_authorization_handoff_token_hash,
  oauth_authorization_handoff_user_code,
  oauth_client_id,
  oauth_authorization_handoff_redirect_uri,
  oauth_authorization_handoff_scopes,
  oauth_authorization_handoff_audience,
  oauth_authorization_handoff_state,
  oauth_authorization_handoff_code_challenge,
  oauth_authorization_handoff_code_challenge_method,
  user_uuid,
  oauth_authorization_handoff_authorization_code,
  oauth_authorization_handoff_redirect_url,
  oauth_authorization_handoff_denied_at,
  oauth_authorization_handoff_completed_at,
  oauth_authorization_handoff_expires_at,
  oauth_authorization_handoff_created_at,
  oauth_authorization_handoff_updated_at
from auth.oauth_authorization_handoffs
where oauth_authorization_handoff_token_hash = sqlc.arg(oauth_authorization_handoff_token_hash);

-- name: GetOAuthAuthorizationHandoffByUserCode :one
select
  oauth_authorization_handoff_id,
  oauth_authorization_handoff_token_hash,
  oauth_authorization_handoff_user_code,
  oauth_client_id,
  oauth_authorization_handoff_redirect_uri,
  oauth_authorization_handoff_scopes,
  oauth_authorization_handoff_audience,
  oauth_authorization_handoff_state,
  oauth_authorization_handoff_code_challenge,
  oauth_authorization_handoff_code_challenge_method,
  user_uuid,
  oauth_authorization_handoff_authorization_code,
  oauth_authorization_handoff_redirect_url,
  oauth_authorization_handoff_denied_at,
  oauth_authorization_handoff_completed_at,
  oauth_authorization_handoff_expires_at,
  oauth_authorization_handoff_created_at,
  oauth_authorization_handoff_updated_at
from auth.oauth_authorization_handoffs
where oauth_authorization_handoff_user_code = sqlc.arg(oauth_authorization_handoff_user_code);

-- name: ApproveOAuthAuthorizationHandoffByID :one
update auth.oauth_authorization_handoffs
set
  user_uuid = sqlc.arg(user_uuid),
  oauth_authorization_handoff_authorization_code = sqlc.arg(oauth_authorization_handoff_authorization_code),
  oauth_authorization_handoff_redirect_url = sqlc.arg(oauth_authorization_handoff_redirect_url),
  oauth_authorization_handoff_updated_at = now()
where
  oauth_authorization_handoff_id = sqlc.arg(oauth_authorization_handoff_id)
  and oauth_authorization_handoff_denied_at is null
  and oauth_authorization_handoff_completed_at is null
  and oauth_authorization_handoff_expires_at > now()
returning
  oauth_authorization_handoff_id,
  oauth_authorization_handoff_token_hash,
  oauth_authorization_handoff_user_code,
  oauth_client_id,
  oauth_authorization_handoff_redirect_uri,
  oauth_authorization_handoff_scopes,
  oauth_authorization_handoff_audience,
  oauth_authorization_handoff_state,
  oauth_authorization_handoff_code_challenge,
  oauth_authorization_handoff_code_challenge_method,
  user_uuid,
  oauth_authorization_handoff_authorization_code,
  oauth_authorization_handoff_redirect_url,
  oauth_authorization_handoff_denied_at,
  oauth_authorization_handoff_completed_at,
  oauth_authorization_handoff_expires_at,
  oauth_authorization_handoff_created_at,
  oauth_authorization_handoff_updated_at;

-- name: DenyOAuthAuthorizationHandoffByID :one
update auth.oauth_authorization_handoffs
set
  oauth_authorization_handoff_denied_at = now(),
  oauth_authorization_handoff_updated_at = now()
where
  oauth_authorization_handoff_id = sqlc.arg(oauth_authorization_handoff_id)
  and oauth_authorization_handoff_denied_at is null
  and oauth_authorization_handoff_completed_at is null
  and oauth_authorization_handoff_expires_at > now()
returning
  oauth_authorization_handoff_id,
  oauth_authorization_handoff_token_hash,
  oauth_authorization_handoff_user_code,
  oauth_client_id,
  oauth_authorization_handoff_redirect_uri,
  oauth_authorization_handoff_scopes,
  oauth_authorization_handoff_audience,
  oauth_authorization_handoff_state,
  oauth_authorization_handoff_code_challenge,
  oauth_authorization_handoff_code_challenge_method,
  user_uuid,
  oauth_authorization_handoff_authorization_code,
  oauth_authorization_handoff_redirect_url,
  oauth_authorization_handoff_denied_at,
  oauth_authorization_handoff_completed_at,
  oauth_authorization_handoff_expires_at,
  oauth_authorization_handoff_created_at,
  oauth_authorization_handoff_updated_at;

-- name: MarkOAuthAuthorizationHandoffCompletedByID :one
update auth.oauth_authorization_handoffs
set
  oauth_authorization_handoff_completed_at = now(),
  oauth_authorization_handoff_updated_at = now()
where
  oauth_authorization_handoff_id = sqlc.arg(oauth_authorization_handoff_id)
  and oauth_authorization_handoff_completed_at is null
returning
  oauth_authorization_handoff_id,
  oauth_authorization_handoff_token_hash,
  oauth_authorization_handoff_user_code,
  oauth_client_id,
  oauth_authorization_handoff_redirect_uri,
  oauth_authorization_handoff_scopes,
  oauth_authorization_handoff_audience,
  oauth_authorization_handoff_state,
  oauth_authorization_handoff_code_challenge,
  oauth_authorization_handoff_code_challenge_method,
  user_uuid,
  oauth_authorization_handoff_authorization_code,
  oauth_authorization_handoff_redirect_url,
  oauth_authorization_handoff_denied_at,
  oauth_authorization_handoff_completed_at,
  oauth_authorization_handoff_expires_at,
  oauth_authorization_handoff_created_at,
  oauth_authorization_handoff_updated_at;
