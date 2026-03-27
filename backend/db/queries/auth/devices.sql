-- name: UpsertDevice :one
insert into user_devices (user_device_uuid, user_id, user_device_name, user_device_os, user_device_app_version)
values
  (sqlc.arg (user_device_id),
    (
      select
        user_id
      from
        users u
      where
        u.user_uuid = sqlc.arg (user_uuid)),
      sqlc.arg (user_device_name),
      sqlc.arg (user_device_os),
      sqlc.arg (user_device_app_version))
on conflict (user_device_uuid)
  do update set
    user_device_last_seen_at = now(),
    user_device_updated_at = now()
  returning
    user_device_uuid,
    (
      select
        user_uuid
      from
        users u
      where
        u.user_id = user_devices.user_id) as user_uuid,
    user_device_name,
    user_device_os,
    user_device_app_version,
    user_device_push_token,
    user_device_push_token_type,
    user_device_push_token_updated_at,
    user_device_created_at,
    user_device_updated_at,
    user_device_last_seen_at;

-- name: UpdateDeviceMetadata :exec
update user_devices
set
  user_device_name = coalesce(nullif(sqlc.narg(user_device_name), ''), user_device_name),
  user_device_os = coalesce(nullif(sqlc.narg(user_device_os), ''), user_device_os),
  user_device_model = coalesce(nullif(sqlc.narg(user_device_model), ''), user_device_model),
  user_device_locale = coalesce(nullif(sqlc.narg(user_device_locale), ''), user_device_locale),
  user_device_time_zone = coalesce(nullif(sqlc.narg(user_device_time_zone), ''), user_device_time_zone),
  user_device_app_version = coalesce(nullif(sqlc.narg(user_device_app_version), ''), user_device_app_version),
  user_device_updated_at = now(),
  user_device_last_seen_at = now()
where
  user_device_uuid = sqlc.arg(user_device_uuid);
