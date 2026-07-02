-- name: CreatePasskey :one
insert into auth.user_passkeys (
  user_id,
  user_identity_id,
  user_passkey_credential_id,
  user_passkey_credential_id_b64url,
  user_passkey_public_key,
  user_passkey_attestation_type,
  user_passkey_transports,
  user_passkey_user_handle,
  user_passkey_sign_count,
  user_passkey_flags,
  user_passkey_aaguid,
  user_passkey_name,
  user_passkey_backup_eligible,
  user_passkey_backup_state,
  user_passkey_last_used_at
)
values (
  (
    select user_id
    from auth.users u
    where u.user_uuid = sqlc.arg(user_uuid)
  ),
  (
    select user_identity_id
    from auth.user_identities ui
    where ui.user_identity_uuid = sqlc.arg(user_identity_uuid)
  ),
  sqlc.arg(user_passkey_credential_id),
  sqlc.arg(user_passkey_credential_id_b64url),
  sqlc.arg(user_passkey_public_key),
  sqlc.arg(user_passkey_attestation_type),
  sqlc.arg(user_passkey_transports),
  sqlc.arg(user_passkey_user_handle),
  sqlc.arg(user_passkey_sign_count),
  sqlc.arg(user_passkey_flags),
  sqlc.arg(user_passkey_aaguid),
  sqlc.arg(user_passkey_name),
  sqlc.arg(user_passkey_backup_eligible),
  sqlc.arg(user_passkey_backup_state),
  sqlc.arg(user_passkey_last_used_at)
)
returning
  user_passkey_uuid,
  (
    select user_uuid
    from auth.users u
    where u.user_id = user_passkeys.user_id
  ) as user_uuid,
  (
    select user_identity_uuid
    from auth.user_identities ui
    where ui.user_identity_id = user_passkeys.user_identity_id
  ) as user_identity_uuid,
  user_passkey_credential_id,
  user_passkey_credential_id_b64url,
  user_passkey_public_key,
  user_passkey_attestation_type,
  user_passkey_transports,
  user_passkey_user_handle,
  user_passkey_sign_count,
  user_passkey_flags,
  user_passkey_aaguid,
  user_passkey_name,
  user_passkey_backup_eligible,
  user_passkey_backup_state,
  user_passkey_last_used_at,
  user_passkey_created_at,
  user_passkey_updated_at,
  user_passkey_revoked_at;

-- name: GetPasskeyByCredentialID :one
select
  user_passkey_uuid,
  (
    select user_uuid
    from auth.users u
    where u.user_id = user_passkeys.user_id
  ) as user_uuid,
  (
    select user_identity_uuid
    from auth.user_identities ui
    where ui.user_identity_id = user_passkeys.user_identity_id
  ) as user_identity_uuid,
  user_passkey_credential_id,
  user_passkey_credential_id_b64url,
  user_passkey_public_key,
  user_passkey_attestation_type,
  user_passkey_transports,
  user_passkey_user_handle,
  user_passkey_sign_count,
  user_passkey_flags,
  user_passkey_aaguid,
  user_passkey_name,
  user_passkey_backup_eligible,
  user_passkey_backup_state,
  user_passkey_last_used_at,
  user_passkey_created_at,
  user_passkey_updated_at,
  user_passkey_revoked_at
from auth.user_passkeys
where user_passkey_credential_id = sqlc.arg(user_passkey_credential_id)
  and user_passkey_revoked_at is null;

-- name: GetPasskeyByUserHandleAndCredentialID :one
select
  user_passkey_uuid,
  (
    select user_uuid
    from auth.users u
    where u.user_id = user_passkeys.user_id
  ) as user_uuid,
  (
    select user_identity_uuid
    from auth.user_identities ui
    where ui.user_identity_id = user_passkeys.user_identity_id
  ) as user_identity_uuid,
  user_passkey_credential_id,
  user_passkey_credential_id_b64url,
  user_passkey_public_key,
  user_passkey_attestation_type,
  user_passkey_transports,
  user_passkey_user_handle,
  user_passkey_sign_count,
  user_passkey_flags,
  user_passkey_aaguid,
  user_passkey_name,
  user_passkey_backup_eligible,
  user_passkey_backup_state,
  user_passkey_last_used_at,
  user_passkey_created_at,
  user_passkey_updated_at,
  user_passkey_revoked_at
from auth.user_passkeys
where user_passkey_user_handle = sqlc.arg(user_passkey_user_handle)
  and user_passkey_credential_id = sqlc.arg(user_passkey_credential_id)
  and user_passkey_revoked_at is null;

-- name: ListPasskeysByUserID :many
select
  user_passkey_uuid,
  (
    select user_uuid
    from auth.users u
    where u.user_id = user_passkeys.user_id
  ) as user_uuid,
  (
    select user_identity_uuid
    from auth.user_identities ui
    where ui.user_identity_id = user_passkeys.user_identity_id
  ) as user_identity_uuid,
  user_passkey_credential_id,
  user_passkey_credential_id_b64url,
  user_passkey_public_key,
  user_passkey_attestation_type,
  user_passkey_transports,
  user_passkey_user_handle,
  user_passkey_sign_count,
  user_passkey_flags,
  user_passkey_aaguid,
  user_passkey_name,
  user_passkey_backup_eligible,
  user_passkey_backup_state,
  user_passkey_last_used_at,
  user_passkey_created_at,
  user_passkey_updated_at,
  user_passkey_revoked_at
from auth.user_passkeys
where user_id = (
  select user_id
  from auth.users u
  where u.user_uuid = sqlc.arg(user_uuid)
)
  and user_passkey_revoked_at is null
order by user_passkey_created_at desc;

-- name: CountActivePasskeysByUserID :one
select
  count(*)::bigint as active_count
from auth.user_passkeys
where user_id = (
  select user_id
  from auth.users u
  where u.user_uuid = sqlc.arg(user_uuid)
)
  and user_passkey_revoked_at is null;

-- name: UpdatePasskeyUsage :exec
update auth.user_passkeys
set
  user_passkey_sign_count = sqlc.arg(user_passkey_sign_count),
  user_passkey_last_used_at = now(),
  user_passkey_updated_at = now(),
  user_passkey_backup_state = coalesce(sqlc.arg(user_passkey_backup_state), user_passkey_backup_state)
where user_passkey_credential_id = sqlc.arg(user_passkey_credential_id)
  and user_passkey_revoked_at is null;

-- name: RevokePasskeyByCredentialB64ForUser :one
with target_user as (
  select user_id
  from auth.users u
  where u.user_uuid = sqlc.arg(user_uuid)
),
locked as (
  select
    up.user_passkey_id
  from auth.user_passkeys up
  where up.user_id = (select user_id from target_user)
    and up.user_passkey_revoked_at is null
  for update
),
active_count as (
  select count(*)::bigint as count
  from locked
),
target as (
  select up.user_passkey_id
  from auth.user_passkeys up
  where up.user_id = (select user_id from target_user)
    and up.user_passkey_revoked_at is null
    and up.user_passkey_credential_id_b64url = sqlc.arg(target_credential_id_b64url)
  for update
),
updated as (
  update auth.user_passkeys
  set
    user_passkey_revoked_at = now(),
    user_passkey_updated_at = now()
  where user_passkey_id = (select user_passkey_id from target)
    and (select count from active_count) >= 2
  returning 1
)
select
  case
    when exists(select 1 from updated) then 'deleted'
    when exists(select 1 from target) then 'last_passkey'
    else 'not_found'
  end as outcome;
