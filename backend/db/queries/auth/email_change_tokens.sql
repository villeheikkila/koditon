-- name: InvalidateActiveEmailChangeTokensForUser :exec
update
  auth.user_email_change_tokens
set
  user_email_change_consumed_at = now()
where
  user_id = sqlc.arg (user_id)
  and user_email_change_consumed_at is null;

-- name: CreateEmailChangeToken :one
insert into auth.user_email_change_tokens (user_id, user_email_change_target_email, user_email_change_token_hash, user_email_change_expires_at)
  values (sqlc.arg (user_id), sqlc.arg (user_email_change_target_email), sqlc.arg (user_email_change_token_hash), sqlc.arg (user_email_change_expires_at))
returning
  user_email_change_token_uuid,
  user_email_change_expires_at;

-- name: ConsumeActiveEmailChangeTokenByHash :one
with consumed as (
  update
    auth.user_email_change_tokens
  set
    user_email_change_consumed_at = now()
  where
    user_email_change_token_hash = sqlc.arg (user_email_change_token_hash)
    and user_email_change_consumed_at is null
    and user_email_change_expires_at > now()
  returning
    user_id,
    user_email_change_target_email
),
updated_user as (
  update
    auth.users u
  set
    user_email = lower(btrim(c.user_email_change_target_email))
  from
    consumed c
  where
    u.user_id = c.user_id
  returning
    u.user_id,
    u.user_email
),
updated_identity as (
  update
    auth.user_identities ui
  set
    user_identity_external_id = lower(btrim(c.user_email_change_target_email)),
    user_identity_email = lower(btrim(c.user_email_change_target_email)),
    user_identity_email_verified = true,
    user_identity_updated_at = now()
  from
    consumed c
  where
    ui.user_id = c.user_id
    and ui.user_identity_provider = 'email'
  returning
    ui.user_identity_id
),
inserted_identity as (
  insert into auth.user_identities (
    user_id,
    user_identity_provider,
    user_identity_external_id,
    user_identity_email,
    user_identity_email_verified,
    user_identity_data
  )
  select
    c.user_id,
    'email',
    lower(btrim(c.user_email_change_target_email)),
    lower(btrim(c.user_email_change_target_email)),
    true,
    jsonb_build_object('email', lower(btrim(c.user_email_change_target_email)))
  from
    consumed c
  where
    not exists(select 1 from updated_identity)
)
select
  uu.user_id,
  uu.user_email
from
  updated_user uu;

-- name: GetEmailChangeTokenStatusByHash :one
select
  case
  when user_email_change_consumed_at is not null then
    'consumed'
  when user_email_change_expires_at <= now() then
    'expired'
  else
    'active'
  end as token_status
from
  auth.user_email_change_tokens
where
  user_email_change_token_hash = sqlc.arg (user_email_change_token_hash)
order by
  user_email_change_created_at desc
limit 1;
