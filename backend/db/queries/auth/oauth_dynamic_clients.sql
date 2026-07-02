-- name: CreateOAuthDynamicClient :one
insert into auth.oauth_dynamic_clients (
  oauth_dynamic_client_id,
  oauth_dynamic_client_type,
  oauth_dynamic_client_redirect_uris,
  oauth_dynamic_client_scopes,
  oauth_dynamic_client_token_endpoint_auth_method,
  oauth_dynamic_client_name,
  oauth_dynamic_client_metadata,
  oauth_dynamic_client_issued_at
) values (
  sqlc.arg(oauth_dynamic_client_id),
  sqlc.arg(oauth_dynamic_client_type),
  sqlc.arg(oauth_dynamic_client_redirect_uris),
  sqlc.arg(oauth_dynamic_client_scopes),
  sqlc.arg(oauth_dynamic_client_token_endpoint_auth_method),
  sqlc.arg(oauth_dynamic_client_name),
  coalesce(sqlc.arg(oauth_dynamic_client_metadata)::jsonb, '{}'::jsonb),
  coalesce(sqlc.arg(oauth_dynamic_client_issued_at), now())
)
returning
  oauth_dynamic_client_id,
  oauth_dynamic_client_type,
  oauth_dynamic_client_redirect_uris,
  oauth_dynamic_client_scopes,
  oauth_dynamic_client_token_endpoint_auth_method,
  oauth_dynamic_client_name,
  oauth_dynamic_client_metadata,
  oauth_dynamic_client_issued_at,
  oauth_dynamic_client_disabled_at,
  oauth_dynamic_client_created_at,
  oauth_dynamic_client_updated_at;

-- name: GetOAuthDynamicClientByID :one
select
  oauth_dynamic_client_id,
  oauth_dynamic_client_type,
  oauth_dynamic_client_redirect_uris,
  oauth_dynamic_client_scopes,
  oauth_dynamic_client_token_endpoint_auth_method,
  oauth_dynamic_client_name,
  oauth_dynamic_client_metadata,
  oauth_dynamic_client_issued_at,
  oauth_dynamic_client_disabled_at,
  oauth_dynamic_client_created_at,
  oauth_dynamic_client_updated_at
from auth.oauth_dynamic_clients
where
  oauth_dynamic_client_id = sqlc.arg(oauth_dynamic_client_id)
  and oauth_dynamic_client_disabled_at is null;
