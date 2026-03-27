-- name: CreateUser :one
insert into users (user_uuid, user_email)
  values (gen_random_uuid (), sqlc.narg ('user_email'))
returning
  user_uuid, user_id as user_id_bigint;

-- name: GetUserByID :one
select
  user_uuid,
  user_id as user_id_bigint
from
  users
where
  user_uuid = sqlc.arg (user_uuid);

-- name: GetUserByEmail :one
select
  user_uuid,
  user_id as user_id_bigint
from
  users
where
  lower(btrim(user_email)) = lower(btrim(sqlc.arg ('user_email')));

-- name: GetUserEmailByIDBigint :one
select
  user_email
from
  users
where
  user_id = sqlc.arg (user_id);

-- name: GetUserEmailByUUID :one
select
  user_email
from
  users
where
  user_uuid = sqlc.arg (user_uuid);

-- name: UpdateUserEmailIfEmptyByIDBigint :one
update
  users
set
  user_email = COALESCE(sqlc.narg ('user_email')::text, user_email)
where
  user_id = sqlc.arg (user_id)
  and user_email is null
returning
  user_email;

-- name: UpdateUserEmailByIDBigint :one
update
  users
set
  user_email = sqlc.arg ('user_email')
where
  user_id = sqlc.arg (user_id)
returning
  user_email;

-- name: EmailExistsForAnotherUser :one
select
  exists (
    select
      1
    from
      users
    where
      user_id != sqlc.arg (user_id)
      and lower(btrim(user_email)) = lower(btrim(sqlc.arg ('user_email')))) as email_exists;

-- name: EmailExists :one
select
  exists (
    select
      1
    from
      users
    where
      lower(btrim(user_email)) = lower(btrim(sqlc.arg ('user_email')))) as email_exists;
