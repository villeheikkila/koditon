-- name: GetUserRoles :many
select
  r.role_uuid,
  r.role_name,
  r.role_description,
  r.role_created_at
from auth.roles r
  join auth.user_roles ur on ur.role_id = r.role_id
where
  ur.user_id = (
    select
      user_id
    from auth.users u
    where
      u.user_uuid = $1)
order by
  r.role_name;

-- name: GetActiveFeatureFlags :many
with u as (
  select
    user_id
  from auth.users
  where
    user_uuid = $1
),
enabled as (
  select
    f.flag_id
  from auth.feature_flags f
  where
    f.flag_default_enabled = true
  union
  select
    rff.flag_id
  from auth.role_feature_flags rff
    join auth.user_roles ur on ur.role_id = rff.role_id
    join u on u.user_id = ur.user_id
union
select
  uff.flag_id
from auth.user_feature_flags uff
  join u on u.user_id = uff.user_id
  where
    uff.user_flag_enabled = true
),
disabled as (
  select
    uff.flag_id
  from auth.user_feature_flags uff
    join u on u.user_id = uff.user_id
  where
    uff.user_flag_enabled = false
)
select
  f.flag_name
from
  enabled e
  join auth.feature_flags f on f.flag_id = e.flag_id
  left join disabled d on d.flag_id = e.flag_id
where
  d.flag_id is null
order by
  f.flag_name;

-- name: HasRole :one
select
  exists (
    select
      1
    from auth.user_roles ur
      join auth.roles r on r.role_id = ur.role_id
    where
      ur.user_id = (
        select
          user_id
        from auth.users u
        where
          u.user_uuid = $1)
        and r.role_name = $2) as has_role;

