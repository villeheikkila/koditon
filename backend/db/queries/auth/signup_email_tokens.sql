-- name: InvalidateActiveSignupEmailTokensForEmail :exec
update
  auth.auth_signup_email_tokens
set
  auth_signup_email_consumed_at = now()
where
  lower(btrim(auth_signup_email_target_email)) = lower(btrim(sqlc.arg(auth_signup_email_target_email)))
  and auth_signup_email_consumed_at is null;

-- name: CreateSignupEmailToken :one
insert into auth.auth_signup_email_tokens (
  auth_signup_email_target_email,
  auth_signup_email_token_hash,
  auth_signup_email_expires_at
)
values (
  sqlc.arg(auth_signup_email_target_email),
  sqlc.arg(auth_signup_email_token_hash),
  sqlc.arg(auth_signup_email_expires_at)
)
returning
  auth_signup_email_token_uuid,
  auth_signup_email_expires_at;

-- name: ConsumeActiveSignupEmailTokenByHash :one
update auth.auth_signup_email_tokens
set auth_signup_email_consumed_at = now()
where auth_signup_email_token_hash = sqlc.arg(auth_signup_email_token_hash)
  and auth_signup_email_consumed_at is null
  and auth_signup_email_expires_at > now()
returning
  auth_signup_email_target_email;

-- name: GetSignupEmailTokenStatusByHash :one
select
  case
    when auth_signup_email_consumed_at is not null then 'consumed'
    when auth_signup_email_expires_at <= now() then 'expired'
    else 'active'
  end as token_status
from
  auth.auth_signup_email_tokens
where
  auth_signup_email_token_hash = sqlc.arg(auth_signup_email_token_hash)
order by
  auth_signup_email_created_at desc
limit 1;
