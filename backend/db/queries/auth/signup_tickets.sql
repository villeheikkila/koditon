-- name: CreateSignupTicket :one
insert into public.auth_signup_tickets (
  auth_signup_ticket_target_email,
  auth_signup_ticket_hash,
  auth_signup_ticket_expires_at
)
values (
  sqlc.arg(auth_signup_ticket_target_email),
  sqlc.arg(auth_signup_ticket_hash),
  sqlc.arg(auth_signup_ticket_expires_at)
)
returning
  auth_signup_ticket_uuid,
  auth_signup_ticket_expires_at;

-- name: ConsumeActiveSignupTicketByHash :one
update public.auth_signup_tickets
set auth_signup_ticket_consumed_at = now()
where auth_signup_ticket_hash = sqlc.arg(auth_signup_ticket_hash)
  and auth_signup_ticket_consumed_at is null
  and auth_signup_ticket_expires_at > now()
returning
  auth_signup_ticket_target_email;

-- name: GetSignupTicketStatusByHash :one
select
  case
    when auth_signup_ticket_consumed_at is not null then 'consumed'
    when auth_signup_ticket_expires_at <= now() then 'expired'
    else 'active'
  end as token_status
from
  public.auth_signup_tickets
where
  auth_signup_ticket_hash = sqlc.arg(auth_signup_ticket_hash)
order by
  auth_signup_ticket_created_at desc
limit 1;
