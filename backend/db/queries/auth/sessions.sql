-- name: CreateSession :one
insert into auth.device_sessions (user_id, device_session_user_device_id, device_session_user_agent, device_session_ip, device_session_provider)
values
  ((
      select
        user_id
      from auth.users u
      where
        u.user_uuid = sqlc.arg (user_uuid)),
      (
        select
          user_device_id
        from auth.user_devices d
        where
          d.user_device_uuid = sqlc.arg (device_session_user_device_uuid)),
        sqlc.arg (device_session_user_agent),
        sqlc.arg (device_session_ip),
        sqlc.arg (device_session_provider))
  returning
    device_session_uuid,
    (
      select
        user_uuid
      from auth.users u
      where
        u.user_id = device_sessions.user_id) as user_uuid,
    (
      select
        user_device_uuid
      from auth.user_devices d
      where
        d.user_device_id = device_sessions.device_session_user_device_id) as device_session_user_device_uuid,
    device_session_user_agent,
    device_session_ip,
    device_session_provider,
    device_session_created_at,
    device_session_updated_at,
    device_session_refreshed_at,
    device_session_not_after,
    device_session_revoked_at;

-- name: GetSessionByID :one
select
  device_session_uuid,
  (
    select
      user_uuid
    from auth.users u
    where
      u.user_id = device_sessions.user_id) as user_uuid,
  (
    select
      user_device_uuid
    from auth.user_devices d
    where
      d.user_device_id = device_sessions.device_session_user_device_id) as device_session_user_device_uuid,
  device_session_user_agent,
  device_session_ip,
  device_session_provider,
  device_session_created_at,
  device_session_updated_at,
  device_session_refreshed_at,
  device_session_not_after,
  device_session_revoked_at
from auth.device_sessions
where
  device_session_uuid = sqlc.arg (device_session_uuid);

-- name: GetActiveSessionByID :one
select
  device_session_uuid,
  (
    select
      user_uuid
    from auth.users u
    where
      u.user_id = device_sessions.user_id) as user_uuid,
  (
    select
      user_device_uuid
    from auth.user_devices d
    where
      d.user_device_id = device_sessions.device_session_user_device_id) as device_session_user_device_uuid,
  device_session_user_agent,
  device_session_ip,
  device_session_provider,
  device_session_created_at,
  device_session_updated_at,
  device_session_refreshed_at,
  device_session_not_after,
  device_session_revoked_at
from auth.device_sessions
where
  device_session_uuid = sqlc.arg (device_session_uuid)
  and device_session_revoked_at is null
  and (device_session_not_after is null
    or device_session_not_after > now());

-- name: GetSessionsByUserID :many
select
  ds.device_session_uuid,
  u.user_uuid,
  d.user_device_uuid as device_session_user_device_uuid,
  ds.device_session_user_agent,
  ds.device_session_ip,
  ds.device_session_provider,
  ds.device_session_created_at,
  ds.device_session_updated_at,
  ds.device_session_refreshed_at,
  ds.device_session_not_after,
  ds.device_session_revoked_at,
  coalesce(ds.device_session_device_name, d.user_device_name) as device_name,
  coalesce(ds.device_session_device_os, d.user_device_os) as device_os,
  coalesce(ds.device_session_device_model, d.user_device_model) as device_model,
  coalesce(ds.device_session_locale, d.user_device_locale) as device_locale,
  coalesce(ds.device_session_time_zone, d.user_device_time_zone) as device_time_zone,
  coalesce(ds.device_session_app_version, d.user_device_app_version) as device_app_version,
  ds.device_session_location_city,
  ds.device_session_location_region,
  ds.device_session_location_country_code,
  ds.device_session_location_source,
  d.user_device_last_seen_at
from auth.device_sessions ds
  join auth.users u on u.user_id = ds.user_id
  join auth.user_devices d on d.user_device_id = ds.device_session_user_device_id
where
  ds.user_id = (
    select
      user_id
    from auth.users u
    where
      u.user_uuid = sqlc.arg (user_uuid))
  and ds.device_session_revoked_at is null
order by
  coalesce(ds.device_session_refreshed_at, ds.device_session_created_at) desc,
  ds.device_session_created_at desc;

-- name: UpdateSessionMetadata :exec
update auth.device_sessions
set
  device_session_device_name = nullif(sqlc.narg(device_session_device_name), ''),
  device_session_device_os = nullif(sqlc.narg(device_session_device_os), ''),
  device_session_device_model = nullif(sqlc.narg(device_session_device_model), ''),
  device_session_app_version = nullif(sqlc.narg(device_session_app_version), ''),
  device_session_locale = nullif(sqlc.narg(device_session_locale), ''),
  device_session_time_zone = nullif(sqlc.narg(device_session_time_zone), ''),
  device_session_location_city = nullif(sqlc.narg(device_session_location_city), ''),
  device_session_location_region = nullif(sqlc.narg(device_session_location_region), ''),
  device_session_location_country_code = nullif(sqlc.narg(device_session_location_country_code), ''),
  device_session_location_source = nullif(sqlc.narg(device_session_location_source), ''),
  device_session_updated_at = now()
where
  device_session_uuid = sqlc.arg(device_session_uuid);

-- name: UpdateSessionRefreshed :exec
update auth.device_sessions
set
  device_session_refreshed_at = now(),
  device_session_updated_at = now()
where
  device_session_uuid = sqlc.arg (device_session_uuid);

-- name: RevokeSession :exec
update auth.device_sessions
set
  device_session_revoked_at = now(),
  device_session_updated_at = now()
where
  device_session_uuid = sqlc.arg (device_session_uuid);

-- name: RevokeAllUserSessions :exec
update auth.device_sessions
set
  device_session_revoked_at = now(),
  device_session_updated_at = now()
where
  user_id = (
    select
      user_id
    from auth.users u
    where
      u.user_uuid = sqlc.arg (user_uuid))
  and device_session_revoked_at is null;
