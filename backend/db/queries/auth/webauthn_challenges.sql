-- name: CreateWebauthnChallenge :one
insert into auth.auth_webauthn_challenges (
  auth_webauthn_challenge_flow,
  auth_webauthn_challenge_session,
  auth_webauthn_challenge_expires_at,
  auth_webauthn_challenge_user_handle,
  auth_webauthn_challenge_user_display_name,
  auth_webauthn_challenge_verified_email,
  auth_webauthn_challenge_device_id,
  user_id
)
values (
  sqlc.arg(auth_webauthn_challenge_flow),
  sqlc.arg(auth_webauthn_challenge_session),
  sqlc.arg(auth_webauthn_challenge_expires_at),
  sqlc.arg(auth_webauthn_challenge_user_handle),
  sqlc.arg(auth_webauthn_challenge_user_display_name),
  sqlc.arg(auth_webauthn_challenge_verified_email),
  sqlc.arg(auth_webauthn_challenge_device_id),
  (
    select user_id
    from auth.users u
    where u.user_uuid = sqlc.narg(user_uuid)
  )
)
returning
  auth_webauthn_challenge_uuid,
  auth_webauthn_challenge_flow,
  auth_webauthn_challenge_session,
  auth_webauthn_challenge_expires_at,
  auth_webauthn_challenge_user_handle,
  auth_webauthn_challenge_user_display_name,
  auth_webauthn_challenge_verified_email,
  auth_webauthn_challenge_device_id,
  auth_webauthn_challenge_consumed_at,
  auth_webauthn_challenge_created_at,
  (
    select user_uuid
    from auth.users u
    where u.user_id = auth_webauthn_challenges.user_id
  ) as user_uuid;

-- name: ConsumeWebauthnChallenge :one
update auth.auth_webauthn_challenges
set auth_webauthn_challenge_consumed_at = now()
where auth_webauthn_challenge_uuid = sqlc.arg(auth_webauthn_challenge_uuid)
  and auth_webauthn_challenge_flow = sqlc.arg(auth_webauthn_challenge_flow)
  and auth_webauthn_challenge_consumed_at is null
  and auth_webauthn_challenge_expires_at > now()
returning
  auth_webauthn_challenge_uuid,
  auth_webauthn_challenge_flow,
  auth_webauthn_challenge_session,
  auth_webauthn_challenge_expires_at,
  auth_webauthn_challenge_user_handle,
  auth_webauthn_challenge_user_display_name,
  auth_webauthn_challenge_verified_email,
  auth_webauthn_challenge_device_id,
  auth_webauthn_challenge_consumed_at,
  auth_webauthn_challenge_created_at,
  (
    select user_uuid
    from auth.users u
    where u.user_id = auth_webauthn_challenges.user_id
  ) as user_uuid;
