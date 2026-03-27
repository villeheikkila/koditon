-- name: GetIdentityByProviderAndExternalID :one
select
  user_identity_uuid,
  user_id as user_id_bigint,
  (
    select
      user_uuid
    from
      users u
    where
      u.user_id = user_identities.user_id) as user_uuid,
  user_identity_provider,
  user_identity_external_id,
  user_identity_email,
  user_identity_email_verified,
  user_identity_data,
  user_identity_created_at,
  user_identity_updated_at
from
  user_identities
where
  user_identity_provider = sqlc.arg (user_identity_provider)
  and user_identity_external_id = sqlc.arg (user_identity_external_id);

-- name: CreateIdentity :one
insert into user_identities (user_id, user_identity_provider, user_identity_external_id, user_identity_email, user_identity_email_verified, user_identity_data)
  values ((
      select
        user_id
      from
        users u
      where
        u.user_uuid = sqlc.arg (user_uuid)),
      sqlc.arg (user_identity_provider),
      sqlc.arg (user_identity_external_id),
      sqlc.arg (user_identity_email),
      sqlc.arg (user_identity_email_verified),
      sqlc.arg (user_identity_data))
returning
  user_identity_uuid,
  (
    select
      user_uuid
    from
      users u
    where
      u.user_id = user_identities.user_id) as user_uuid,
  user_identity_provider,
  user_identity_external_id,
  user_identity_email,
  user_identity_email_verified,
  user_identity_data,
  user_identity_created_at,
  user_identity_updated_at;

-- name: UpdateIdentity :one
update
  user_identities
set
  user_identity_email = COALESCE(sqlc.arg (user_identity_email), user_identity_email),
  user_identity_email_verified = COALESCE(sqlc.arg (user_identity_email_verified), user_identity_email_verified),
  user_identity_data = COALESCE(sqlc.arg (user_identity_data), user_identity_data),
  user_identity_updated_at = now()
where
  user_identity_uuid = sqlc.arg (user_identity_uuid)
returning
  user_identity_uuid,
  (
    select
      user_uuid
    from
      users u
    where
      u.user_id = user_identities.user_id) as user_uuid,
  user_identity_provider,
  user_identity_external_id,
  user_identity_email,
  user_identity_email_verified,
  user_identity_data,
  user_identity_created_at,
  user_identity_updated_at;
