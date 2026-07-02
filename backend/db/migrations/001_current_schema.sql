-- Baseline schema squashed from local database state at migration 132.

--
-- PostgreSQL database dump
--


-- Dumped from database version 18.1 (Debian 18.1-1.pgdg12+2)
-- Dumped by pg_dump version 18.4

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: absurd; Type: SCHEMA; Schema: -; Owner: -
--

CREATE SCHEMA absurd;


--
-- Name: auth; Type: SCHEMA; Schema: -; Owner: -
--

CREATE SCHEMA auth;


--
-- Name: pg_cron; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS pg_cron WITH SCHEMA pg_catalog;


--
-- Name: postgis; Type: SCHEMA; Schema: -; Owner: -
--

CREATE SCHEMA postgis;


--
-- Name: runtime; Type: SCHEMA; Schema: -; Owner: -
--

CREATE SCHEMA runtime;


--
-- Name: pg_trgm; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS pg_trgm WITH SCHEMA public;


--
-- Name: postgis; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS postgis WITH SCHEMA postgis;


--
-- Name: uuid-ossp; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS "uuid-ossp" WITH SCHEMA public;


--
-- Name: auth_provider; Type: TYPE; Schema: auth; Owner: -
--

CREATE TYPE auth.auth_provider AS ENUM (
    'apple',
    'anonymous'
);


--
-- Name: push_token_type; Type: TYPE; Schema: auth; Owner: -
--

CREATE TYPE auth.push_token_type AS ENUM (
    'apns'
);


--
-- Name: enum__name_display; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.enum__name_display AS ENUM (
    'full_name',
    'username'
);


--
-- Name: await_event(text, uuid, uuid, text, text, integer); Type: FUNCTION; Schema: absurd; Owner: -
--

CREATE FUNCTION absurd.await_event(p_queue_name text, p_task_id uuid, p_run_id uuid, p_step_name text, p_event_name text, p_timeout integer DEFAULT NULL::integer) RETURNS TABLE(should_suspend boolean, payload jsonb)
    LANGUAGE plpgsql
    AS $_$
declare
  v_run_state text;
  v_existing_payload jsonb;
  v_event_payload jsonb;
  v_checkpoint_payload jsonb;
  v_resolved_payload jsonb;
  v_timeout_at timestamptz;
  v_available_at timestamptz;
  v_now timestamptz := absurd.current_time();
  v_task_state text;
  v_wake_event text;
begin
  if p_event_name is null or length(trim(p_event_name)) = 0 then
    raise exception 'event_name must be provided';
  end if;

  if p_timeout is not null then
    if p_timeout < 0 then
      raise exception 'timeout must be non-negative';
    end if;
    v_timeout_at := v_now + (p_timeout::double precision * interval '1 second');
  end if;

  v_available_at := coalesce(v_timeout_at, 'infinity'::timestamptz);

  execute format(
    'select state
       from absurd.%I
      where task_id = $1
        and checkpoint_name = $2',
    'c_' || p_queue_name
  )
  into v_checkpoint_payload
  using p_task_id, p_step_name;

  if v_checkpoint_payload is not null then
    return query select false, v_checkpoint_payload;
    return;
  end if;

  -- Ensure a row exists for this event so we can take a row-level lock.
  --
  -- We use payload IS NULL as the sentinel for "not emitted yet".  emit_event
  -- always writes a non-NULL payload (at minimum JSON null).
  --
  -- Lock ordering is important to avoid deadlocks: await_event locks the event
  -- row first (FOR SHARE) and then the run row (FOR UPDATE).  emit_event
  -- naturally locks the event row via its UPSERT before touching waits/runs.
  execute format(
    'insert into absurd.%I (event_name, payload, emitted_at)
     values ($1, null, ''epoch''::timestamptz)
     on conflict (event_name) do nothing',
    'e_' || p_queue_name
  ) using p_event_name;

  execute format(
    'select 1
       from absurd.%I
      where event_name = $1
      for share',
    'e_' || p_queue_name
  ) using p_event_name;

  execute format(
    'select r.state, r.event_payload, r.wake_event, t.state
       from absurd.%I r
       join absurd.%I t on t.task_id = r.task_id
      where r.run_id = $1
      for update',
    'r_' || p_queue_name,
    't_' || p_queue_name
  )
  into v_run_state, v_existing_payload, v_wake_event, v_task_state
  using p_run_id;

  if v_run_state is null then
    raise exception 'Run "%" not found while awaiting event', p_run_id;
  end if;

  if v_task_state = 'cancelled' then
    raise exception sqlstate 'AB001' using message = 'Task has been cancelled';
  end if;

  execute format(
    'select payload
       from absurd.%I
      where event_name = $1',
    'e_' || p_queue_name
  )
  into v_event_payload
  using p_event_name;

  if v_existing_payload is not null then
    execute format(
      'update absurd.%I
          set event_payload = null
        where run_id = $1',
      'r_' || p_queue_name
    ) using p_run_id;

    if v_event_payload is not null and v_event_payload = v_existing_payload then
      v_resolved_payload := v_existing_payload;
    end if;
  end if;

  if v_run_state <> 'running' then
    raise exception 'Run "%" must be running to await events', p_run_id;
  end if;

  if v_resolved_payload is null and v_event_payload is not null then
    v_resolved_payload := v_event_payload;
  end if;

  if v_resolved_payload is not null then
    execute format(
      'insert into absurd.%I (task_id, checkpoint_name, state, status, owner_run_id, updated_at)
       values ($1, $2, $3, ''committed'', $4, $5)
       on conflict (task_id, checkpoint_name)
       do update set state = excluded.state,
                     status = excluded.status,
                     owner_run_id = excluded.owner_run_id,
                     updated_at = excluded.updated_at',
      'c_' || p_queue_name
    ) using p_task_id, p_step_name, v_resolved_payload, p_run_id, v_now;
    return query select false, v_resolved_payload;
    return;
  end if;

  -- Detect if we resumed due to timeout: wake_event matches and payload is null
  if v_resolved_payload is null and v_wake_event = p_event_name and v_existing_payload is null then
    -- Resumed due to timeout; don't re-sleep and don't create a new wait
    execute format(
      'update absurd.%I set wake_event = null where run_id = $1',
      'r_' || p_queue_name
    ) using p_run_id;
    return query select false, null::jsonb;
    return;
  end if;

  execute format(
    'insert into absurd.%I (task_id, run_id, step_name, event_name, timeout_at, created_at)
     values ($1, $2, $3, $4, $5, $6)
     on conflict (run_id, step_name)
     do update set event_name = excluded.event_name,
                   timeout_at = excluded.timeout_at,
                   created_at = excluded.created_at',
    'w_' || p_queue_name
  ) using p_task_id, p_run_id, p_step_name, p_event_name, v_timeout_at, v_now;

  execute format(
    'update absurd.%I
        set state = ''sleeping'',
            claimed_by = null,
            claim_expires_at = null,
            available_at = $3,
            wake_event = $2,
            event_payload = null
      where run_id = $1',
    'r_' || p_queue_name
  ) using p_run_id, p_event_name, v_available_at;

  execute format(
    'update absurd.%I
        set state = ''sleeping''
      where task_id = $1',
    't_' || p_queue_name
  ) using p_task_id;

  return query select true, null::jsonb;
  return;
end;
$_$;


--
-- Name: cancel_task(text, uuid); Type: FUNCTION; Schema: absurd; Owner: -
--

CREATE FUNCTION absurd.cancel_task(p_queue_name text, p_task_id uuid) RETURNS void
    LANGUAGE plpgsql
    AS $_$
declare
  v_now timestamptz := absurd.current_time();
  v_task_state text;
begin
  -- Lock active runs before the task row so cancel_task() uses the same
  -- lock acquisition order as complete_run()/fail_run().
  execute format(
    'select run_id
       from absurd.%I
      where task_id = $1
        and state not in (''completed'', ''failed'', ''cancelled'')
      order by run_id
      for update',
    'r_' || p_queue_name
  ) using p_task_id;

  execute format(
    'select state
       from absurd.%I
      where task_id = $1
      for update',
    't_' || p_queue_name
  )
  into v_task_state
  using p_task_id;

  if v_task_state is null then
    raise exception 'Task "%" not found in queue "%"', p_task_id, p_queue_name;
  end if;

  if v_task_state in ('completed', 'failed', 'cancelled') then
    return;
  end if;

  execute format(
    'update absurd.%I
        set state = ''cancelled'',
            cancelled_at = coalesce(cancelled_at, $2)
      where task_id = $1',
    't_' || p_queue_name
  ) using p_task_id, v_now;

  execute format(
    'update absurd.%I
        set state = ''cancelled'',
            claimed_by = null,
            claim_expires_at = null
      where task_id = $1
        and state not in (''completed'', ''failed'', ''cancelled'')',
    'r_' || p_queue_name
  ) using p_task_id;

  execute format(
    'delete from absurd.%I where task_id = $1',
    'w_' || p_queue_name
  ) using p_task_id;
end;
$_$;


--
-- Name: claim_task(text, text, integer, integer); Type: FUNCTION; Schema: absurd; Owner: -
--

CREATE FUNCTION absurd.claim_task(p_queue_name text, p_worker_id text, p_claim_timeout integer DEFAULT 30, p_qty integer DEFAULT 1) RETURNS TABLE(run_id uuid, task_id uuid, attempt integer, task_name text, params jsonb, retry_strategy jsonb, max_attempts integer, headers jsonb, wake_event text, event_payload jsonb)
    LANGUAGE plpgsql
    AS $_$
declare
  v_now timestamptz := absurd.current_time();
  v_claim_timeout integer := greatest(coalesce(p_claim_timeout, 30), 0);
  v_worker_id text := coalesce(nullif(p_worker_id, ''), 'worker');
  v_qty integer := greatest(coalesce(p_qty, 1), 1);
  v_claim_until timestamptz := null;
  v_sql text;
  v_expired_run record;
  v_cancel_candidate record;
  v_expired_sweep_limit integer;
begin
  if v_claim_timeout > 0 then
    v_claim_until := v_now + make_interval(secs => v_claim_timeout);
  end if;

  -- Keep claim polling work bounded: process at most v_qty expired leases
  -- per claim call.
  v_expired_sweep_limit := greatest(v_qty, 1);

  -- Apply cancellation rules before claiming.
  --
  -- Use cancel_task() so lock order stays consistent (runs first, task second)
  -- with complete_run()/fail_run().
  for v_cancel_candidate in
    execute format(
      'select task_id
         from absurd.%I
        where state in (''pending'', ''sleeping'', ''running'')
          and (
            (
              (cancellation->>''max_delay'')::bigint is not null
              and first_started_at is null
              and extract(epoch from ($1 - enqueue_at)) >= (cancellation->>''max_delay'')::bigint
            )
            or
            (
              (cancellation->>''max_duration'')::bigint is not null
              and first_started_at is not null
              and extract(epoch from ($1 - first_started_at)) >= (cancellation->>''max_duration'')::bigint
            )
          )
        order by task_id',
      't_' || p_queue_name
    )
  using v_now
  loop
    perform absurd.cancel_task(p_queue_name, v_cancel_candidate.task_id);
  end loop;

  for v_expired_run in
    execute format(
      'select run_id,
              claimed_by,
              claim_expires_at,
              attempt
         from absurd.%I
        where state = ''running''
          and claim_expires_at is not null
          and claim_expires_at <= $1
        order by claim_expires_at, run_id
        limit $2
        for update skip locked',
      'r_' || p_queue_name
    )
  using v_now, v_expired_sweep_limit
  loop
    perform absurd.fail_run(
      p_queue_name,
      v_expired_run.run_id,
      jsonb_strip_nulls(jsonb_build_object(
        'name', '$ClaimTimeout',
        'message', 'worker did not finish task within claim interval',
        'workerId', v_expired_run.claimed_by,
        'claimExpiredAt', v_expired_run.claim_expires_at,
        'attempt', v_expired_run.attempt
      )),
      null
    );
  end loop;

  v_sql := format(
    'with candidate as (
        select r.run_id
          from absurd.%1$I r
          join absurd.%2$I t on t.task_id = r.task_id
         where r.state in (''pending'', ''sleeping'')
           and t.state in (''pending'', ''sleeping'', ''running'')
           and r.available_at <= $1
         order by r.available_at, r.run_id
         limit $2
         for update skip locked
     ),
     updated as (
        update absurd.%1$I r
           set state = ''running'',
               claimed_by = $3,
               claim_expires_at = $4,
               started_at = $1,
               available_at = $1
         where run_id in (select run_id from candidate)
         returning r.run_id, r.task_id, r.attempt
     ),
     task_upd as (
        update absurd.%2$I t
           set state = ''running'',
               attempts = greatest(t.attempts, u.attempt),
               first_started_at = coalesce(t.first_started_at, $1),
               last_attempt_run = u.run_id
          from updated u
         where t.task_id = u.task_id
         returning t.task_id
     ),
     wait_cleanup as (
        delete from absurd.%3$I w
         using updated u
        where w.run_id = u.run_id
          and w.timeout_at is not null
          and w.timeout_at <= $1
        returning w.run_id
     )
     select
       u.run_id,
       u.task_id,
       u.attempt,
       t.task_name,
       t.params,
       t.retry_strategy,
       t.max_attempts,
      t.headers,
      r.wake_event,
      r.event_payload
     from updated u
     join absurd.%1$I r on r.run_id = u.run_id
     join absurd.%2$I t on t.task_id = u.task_id
     order by r.available_at, u.run_id',
    'r_' || p_queue_name,
    't_' || p_queue_name,
    'w_' || p_queue_name
  );

  return query execute v_sql using v_now, v_qty, v_worker_id, v_claim_until;
end;
$_$;


--
-- Name: cleanup_all_queues(text); Type: FUNCTION; Schema: absurd; Owner: -
--

CREATE FUNCTION absurd.cleanup_all_queues(p_queue_name text DEFAULT NULL::text) RETURNS TABLE(queue_name text, tasks_deleted integer, events_deleted integer)
    LANGUAGE plpgsql
    AS $$
declare
  v_queue record;
  v_cleanup_ttl_seconds integer;
begin
  if p_queue_name is not null then
    p_queue_name := absurd.validate_queue_name(p_queue_name);

    if not exists (
      select 1
      from absurd.queues q
      where q.queue_name = p_queue_name
    ) then
      raise exception 'Queue "%" does not exist', p_queue_name;
    end if;
  end if;

  for v_queue in
    select
      q.queue_name,
      q.cleanup_ttl,
      q.cleanup_limit
    from absurd.queues q
    where p_queue_name is null or q.queue_name = p_queue_name
    order by q.queue_name
  loop
    v_cleanup_ttl_seconds := greatest(
      floor(extract(epoch from v_queue.cleanup_ttl))::integer,
      0
    );

    queue_name := v_queue.queue_name;
    tasks_deleted := absurd.cleanup_tasks(
      v_queue.queue_name,
      v_cleanup_ttl_seconds,
      v_queue.cleanup_limit
    );
    events_deleted := absurd.cleanup_events(
      v_queue.queue_name,
      v_cleanup_ttl_seconds,
      v_queue.cleanup_limit
    );
    return next;
  end loop;
end;
$$;


--
-- Name: cleanup_events(text, integer, integer); Type: FUNCTION; Schema: absurd; Owner: -
--

CREATE FUNCTION absurd.cleanup_events(p_queue_name text, p_ttl_seconds integer, p_limit integer DEFAULT 1000) RETURNS integer
    LANGUAGE plpgsql
    AS $_$
declare
  v_now timestamptz := absurd.current_time();
  v_cutoff timestamptz;
  v_deleted_count integer;
begin
  if p_ttl_seconds is null or p_ttl_seconds < 0 then
    raise exception 'TTL must be a non-negative number of seconds';
  end if;

  v_cutoff := v_now - (p_ttl_seconds * interval '1 second');

  execute format(
    'with to_delete as (
        select event_name
          from absurd.%I
         where emitted_at < $1
         order by emitted_at
         limit $2
     ),
     del_events as (
        delete from absurd.%I e
         where e.event_name in (select event_name from to_delete)
         returning 1
     )
     select count(*) from del_events',
    'e_' || p_queue_name,
    'e_' || p_queue_name
  )
  into v_deleted_count
  using v_cutoff, p_limit;

  return v_deleted_count;
end;
$_$;


--
-- Name: cleanup_tasks(text, integer, integer); Type: FUNCTION; Schema: absurd; Owner: -
--

CREATE FUNCTION absurd.cleanup_tasks(p_queue_name text, p_ttl_seconds integer, p_limit integer DEFAULT 1000) RETURNS integer
    LANGUAGE plpgsql
    AS $_$
declare
  v_now timestamptz := absurd.current_time();
  v_cutoff timestamptz;
  v_deleted_count integer;
  v_storage_mode text := 'unpartitioned';
begin
  if p_ttl_seconds is null or p_ttl_seconds < 0 then
    raise exception 'TTL must be a non-negative number of seconds';
  end if;

  v_cutoff := v_now - (p_ttl_seconds * interval '1 second');

  select storage_mode into v_storage_mode
  from absurd.queues
  where queue_name = p_queue_name;

  v_storage_mode := coalesce(v_storage_mode, 'unpartitioned');

  if v_storage_mode = 'partitioned' then
    -- Delete in order: wait registrations, checkpoints, runs, idempotency keys,
    -- then tasks.
    execute format(
      'with eligible_tasks as (
          select t.task_id,
                 case
                   when t.state = ''completed'' then r.completed_at
                   when t.state = ''failed'' then r.failed_at
                   when t.state = ''cancelled'' then t.cancelled_at
                   else null
                 end as terminal_at
            from absurd.%1$I t
            left join absurd.%2$I r on r.run_id = t.last_attempt_run
           where t.state in (''completed'', ''failed'', ''cancelled'')
       ),
       to_delete as (
          select task_id
            from eligible_tasks
           where terminal_at is not null
             and terminal_at < $1
           order by terminal_at
           limit $2
       ),
       del_waits as (
          delete from absurd.%3$I w
           where w.task_id in (select task_id from to_delete)
       ),
       del_checkpoints as (
          delete from absurd.%4$I c
           where c.task_id in (select task_id from to_delete)
       ),
       del_runs as (
          delete from absurd.%2$I r
           where r.task_id in (select task_id from to_delete)
       ),
       del_idempotency as (
          delete from absurd.%5$I i
           where i.task_id in (select task_id from to_delete)
       ),
       del_tasks as (
          delete from absurd.%1$I t
           where t.task_id in (select task_id from to_delete)
           returning 1
       )
       select count(*) from del_tasks',
      't_' || p_queue_name,
      'r_' || p_queue_name,
      'w_' || p_queue_name,
      'c_' || p_queue_name,
      'i_' || p_queue_name
    )
    into v_deleted_count
    using v_cutoff, p_limit;
  else
    -- Unpartitioned queues keep idempotency key ownership on the task row,
    -- so no side-table cleanup is needed.
    execute format(
      'with eligible_tasks as (
          select t.task_id,
                 case
                   when t.state = ''completed'' then r.completed_at
                   when t.state = ''failed'' then r.failed_at
                   when t.state = ''cancelled'' then t.cancelled_at
                   else null
                 end as terminal_at
            from absurd.%1$I t
            left join absurd.%2$I r on r.run_id = t.last_attempt_run
           where t.state in (''completed'', ''failed'', ''cancelled'')
       ),
       to_delete as (
          select task_id
            from eligible_tasks
           where terminal_at is not null
             and terminal_at < $1
           order by terminal_at
           limit $2
       ),
       del_waits as (
          delete from absurd.%3$I w
           where w.task_id in (select task_id from to_delete)
       ),
       del_checkpoints as (
          delete from absurd.%4$I c
           where c.task_id in (select task_id from to_delete)
       ),
       del_runs as (
          delete from absurd.%2$I r
           where r.task_id in (select task_id from to_delete)
       ),
       del_tasks as (
          delete from absurd.%1$I t
           where t.task_id in (select task_id from to_delete)
           returning 1
       )
       select count(*) from del_tasks',
      't_' || p_queue_name,
      'r_' || p_queue_name,
      'w_' || p_queue_name,
      'c_' || p_queue_name
    )
    into v_deleted_count
    using v_cutoff, p_limit;
  end if;

  return v_deleted_count;
end;
$_$;


--
-- Name: complete_run(text, uuid, jsonb); Type: FUNCTION; Schema: absurd; Owner: -
--

CREATE FUNCTION absurd.complete_run(p_queue_name text, p_run_id uuid, p_state jsonb DEFAULT NULL::jsonb) RETURNS void
    LANGUAGE plpgsql
    AS $_$
declare
  v_task_id uuid;
  v_state text;
  v_now timestamptz := absurd.current_time();
begin
  execute format(
    'select task_id, state
       from absurd.%I
      where run_id = $1
      for update',
    'r_' || p_queue_name
  )
  into v_task_id, v_state
  using p_run_id;

  if v_task_id is null then
    raise exception 'Run "%" not found in queue "%"', p_run_id, p_queue_name;
  end if;

  if v_state <> 'running' then
    if v_state = 'cancelled' then
      raise exception sqlstate 'AB001' using message = 'Task has been cancelled';
    end if;
    if v_state = 'failed' then
      raise exception sqlstate 'AB002' using message = format('Run "%s" has already failed in queue "%s"', p_run_id, p_queue_name);
    end if;
    raise exception 'Run "%" is not currently running in queue "%"', p_run_id, p_queue_name;
  end if;

  execute format(
    'update absurd.%I
        set state = ''completed'',
            completed_at = $2,
            result = $3
      where run_id = $1',
    'r_' || p_queue_name
  ) using p_run_id, v_now, p_state;

  execute format(
    'update absurd.%I
        set state = ''completed'',
            completed_payload = $2,
            last_attempt_run = $3
      where task_id = $1',
    't_' || p_queue_name
  ) using v_task_id, p_state, p_run_id;

  execute format(
    'delete from absurd.%I where run_id = $1',
    'w_' || p_queue_name
  ) using p_run_id;
end;
$_$;


--
-- Name: create_queue(text); Type: FUNCTION; Schema: absurd; Owner: -
--

CREATE FUNCTION absurd.create_queue(p_queue_name text) RETURNS void
    LANGUAGE plpgsql
    AS $$
begin
  perform absurd.create_queue(p_queue_name, 'unpartitioned');
end;
$$;


--
-- Name: create_queue(text, text); Type: FUNCTION; Schema: absurd; Owner: -
--

CREATE FUNCTION absurd.create_queue(p_queue_name text, p_storage_mode text) RETURNS void
    LANGUAGE plpgsql
    AS $$
declare
  v_storage_mode text;
  v_existing_mode text;
begin
  p_queue_name := absurd.validate_queue_name(p_queue_name);

  v_storage_mode := lower(trim(coalesce(p_storage_mode, '')));
  if v_storage_mode not in ('unpartitioned', 'partitioned') then
    raise exception 'Unsupported queue storage mode "%"', p_storage_mode;
  end if;

  insert into absurd.queues (queue_name, storage_mode)
  values (p_queue_name, v_storage_mode)
  on conflict (queue_name) do nothing;

  select storage_mode into v_existing_mode
  from absurd.queues
  where queue_name = p_queue_name;

  if v_existing_mode is null then
    raise exception 'Queue "%" was not found after create attempt', p_queue_name;
  end if;

  if v_existing_mode <> v_storage_mode then
    raise exception 'Queue "%" already exists with storage mode "%"', p_queue_name, v_existing_mode;
  end if;

  perform absurd.ensure_queue_tables(p_queue_name);
end;
$$;


--
-- Name: current_time(); Type: FUNCTION; Schema: absurd; Owner: -
--

CREATE FUNCTION absurd."current_time"() RETURNS timestamp with time zone
    LANGUAGE plpgsql
    AS $$
declare
  v_fake text;
begin
  v_fake := current_setting('absurd.fake_now', true);
  if v_fake is not null and length(trim(v_fake)) > 0 then
    return v_fake::timestamptz;
  end if;

  return clock_timestamp();
end;
$$;


--
-- Name: disable_cron(text); Type: FUNCTION; Schema: absurd; Owner: -
--

CREATE FUNCTION absurd.disable_cron(p_queue_name text DEFAULT NULL::text) RETURNS TABLE(job_name text, job_id bigint)
    LANGUAGE plpgsql
    AS $_$
declare
  v_job_suffix text;
  v_partition_job_name text;
  v_cleanup_job_name text;
  v_detach_plan_job_name text;
  v_detach_run_pattern text;
  v_drop_run_pattern text;
  v_existing_job record;
begin
  if p_queue_name is not null then
    p_queue_name := absurd.validate_queue_name(p_queue_name);
  end if;

  if to_regclass('cron.job') is null then
    raise exception 'pg_cron is not available (missing cron.job)';
  end if;

  if not exists (
    select 1
    from pg_proc p
    join pg_namespace n on n.oid = p.pronamespace
    where n.nspname = 'cron'
      and p.proname = 'unschedule'
  ) then
    raise exception 'pg_cron is not available (missing cron.unschedule)';
  end if;

  v_job_suffix := case
    when p_queue_name is null then 'all'
    else substr(md5(p_queue_name), 1, 12)
  end;

  v_partition_job_name := 'absurd_partitions_' || v_job_suffix;
  v_cleanup_job_name := 'absurd_cleanup_' || v_job_suffix;
  v_detach_plan_job_name := 'absurd_detach_plan_' || v_job_suffix;
  v_detach_run_pattern := 'absurd_detach_run_' || v_job_suffix || '_%';
  v_drop_run_pattern := 'absurd_drop_run_' || v_job_suffix || '_%';

  for v_existing_job in
    execute 'select jobid, jobname
               from cron.job
              where jobname = $1
                 or jobname = $2
                 or jobname = $3
                 or jobname like $4
                 or jobname like $5
              order by jobname, jobid'
    using v_partition_job_name,
          v_cleanup_job_name,
          v_detach_plan_job_name,
          v_detach_run_pattern,
          v_drop_run_pattern
  loop
    execute 'select cron.unschedule($1)' using v_existing_job.jobid;

    job_name := v_existing_job.jobname;
    job_id := v_existing_job.jobid;
    return next;
  end loop;
end;
$_$;


--
-- Name: drop_detached_partition(text, text); Type: FUNCTION; Schema: absurd; Owner: -
--

CREATE FUNCTION absurd.drop_detached_partition(p_partition_table text, p_unschedule_job_name text DEFAULT NULL::text) RETURNS boolean
    LANGUAGE plpgsql
    AS $$
declare
  v_partition_table text := nullif(trim(coalesce(p_partition_table, '')), '');
  v_partition_oid oid;
  v_is_attached boolean := false;
  v_detach_job_name text;
begin
  if p_unschedule_job_name like 'absurd_drop_run_%' then
    v_detach_job_name :=
      'absurd_detach_run_' || substr(p_unschedule_job_name, length('absurd_drop_run_') + 1);
  end if;

  if v_partition_table is null then
    raise exception 'partition table must be provided';
  end if;

  select c.oid
    into v_partition_oid
    from pg_class c
    join pg_namespace n on n.oid = c.relnamespace
   where n.nspname = 'absurd'
     and c.relname = v_partition_table;

  if v_partition_oid is null then
    if p_unschedule_job_name is not null and to_regclass('cron.job') is not null then
      perform cron.unschedule(jobid)
        from cron.job
       where jobname in (p_unschedule_job_name, coalesce(v_detach_job_name, ''));
    end if;
    return false;
  end if;

  select exists (
    select 1
    from pg_inherits
    where inhrelid = v_partition_oid
  )
  into v_is_attached;

  if v_is_attached then
    return false;
  end if;

  -- Once detached, stop retrying detach runs immediately. Keep drop
  -- scheduled until the table is actually dropped.
  if v_detach_job_name is not null and to_regclass('cron.job') is not null then
    perform cron.unschedule(jobid)
      from cron.job
     where jobname = v_detach_job_name;
  end if;

  execute format('drop table if exists absurd.%I', v_partition_table);

  if p_unschedule_job_name is not null and to_regclass('cron.job') is not null then
    perform cron.unschedule(jobid)
      from cron.job
     where jobname = p_unschedule_job_name;
  end if;

  return true;
end;
$$;


--
-- Name: drop_queue(text); Type: FUNCTION; Schema: absurd; Owner: -
--

CREATE FUNCTION absurd.drop_queue(p_queue_name text) RETURNS void
    LANGUAGE plpgsql
    AS $$
declare
  v_existing_queue text;
begin
  select queue_name into v_existing_queue
  from absurd.queues
  where queue_name = p_queue_name;

  if v_existing_queue is null then
    return;
  end if;

  -- Remove queue-scoped maintenance jobs only when pg_cron is available.
  if to_regclass('cron.job') is not null and exists (
    select 1
    from pg_proc p
    join pg_namespace n on n.oid = p.pronamespace
    where n.nspname = 'cron'
      and p.proname = 'unschedule'
  ) then
    perform absurd.disable_cron(p_queue_name);
  end if;

  execute format('drop table if exists absurd.%I cascade', 'i_' || p_queue_name);
  execute format('drop table if exists absurd.%I cascade', 'w_' || p_queue_name);
  execute format('drop table if exists absurd.%I cascade', 'e_' || p_queue_name);
  execute format('drop table if exists absurd.%I cascade', 'c_' || p_queue_name);
  execute format('drop table if exists absurd.%I cascade', 'r_' || p_queue_name);
  execute format('drop table if exists absurd.%I cascade', 't_' || p_queue_name);

  delete from absurd.queues where queue_name = p_queue_name;
end;
$$;


--
-- Name: emit_event(text, text, jsonb); Type: FUNCTION; Schema: absurd; Owner: -
--

CREATE FUNCTION absurd.emit_event(p_queue_name text, p_event_name text, p_payload jsonb DEFAULT NULL::jsonb) RETURNS void
    LANGUAGE plpgsql
    AS $_$
declare
  v_now timestamptz := absurd.current_time();
  v_payload jsonb := coalesce(p_payload, 'null'::jsonb);
  v_emit_applied integer;
begin
  if p_event_name is null or length(trim(p_event_name)) = 0 then
    raise exception 'event_name must be provided';
  end if;

  -- Events are immutable once emitted: first write wins.
  --
  -- await_event() may pre-create a row with payload=NULL as a "not emitted"
  -- sentinel. We allow exactly one transition NULL -> JSON payload.
  execute format(
    'insert into absurd.%1$I as e (event_name, payload, emitted_at)
     values ($1, $2, $3)
     on conflict (event_name)
     do update set payload = excluded.payload,
                   emitted_at = excluded.emitted_at
      where e.payload is null',
    'e_' || p_queue_name
  ) using p_event_name, v_payload, v_now;

  get diagnostics v_emit_applied = row_count;

  -- Event was already emitted earlier; do not overwrite cached payload or
  -- re-run wakeup side effects.
  if v_emit_applied = 0 then
    return;
  end if;

  execute format(
    'with expired_waits as (
        delete from absurd.%1$I w
         where w.event_name = $1
           and w.timeout_at is not null
           and w.timeout_at <= $2
         returning w.run_id
     ),
     affected as (
        select run_id, task_id, step_name
          from absurd.%1$I
         where event_name = $1
           and (timeout_at is null or timeout_at > $2)
     ),
     updated_runs as (
        update absurd.%2$I r
           set state = ''pending'',
               available_at = $2,
               wake_event = null,
               event_payload = $3,
               claimed_by = null,
               claim_expires_at = null
         where r.run_id in (select run_id from affected)
           and r.state = ''sleeping''
         returning r.run_id, r.task_id
     ),
     checkpoint_upd as (
        insert into absurd.%3$I (task_id, checkpoint_name, state, status, owner_run_id, updated_at)
        select a.task_id, a.step_name, $3, ''committed'', a.run_id, $2
          from affected a
          join updated_runs ur on ur.run_id = a.run_id
        on conflict (task_id, checkpoint_name)
        do update set state = excluded.state,
                      status = excluded.status,
                      owner_run_id = excluded.owner_run_id,
                      updated_at = excluded.updated_at
     ),
     updated_tasks as (
        update absurd.%4$I t
           set state = ''pending''
         where t.task_id in (select task_id from updated_runs)
         returning task_id
     )
     delete from absurd.%5$I w
      where w.event_name = $1
        and w.run_id in (select run_id from updated_runs)',
    'w_' || p_queue_name,
    'r_' || p_queue_name,
    'c_' || p_queue_name,
    't_' || p_queue_name,
    'w_' || p_queue_name
  ) using p_event_name, v_now, v_payload;
end;
$_$;


--
-- Name: enable_cron(text, text, text, text); Type: FUNCTION; Schema: absurd; Owner: -
--

CREATE FUNCTION absurd.enable_cron(p_queue_name text DEFAULT NULL::text, p_partition_schedule text DEFAULT '5 * * * *'::text, p_cleanup_schedule text DEFAULT '17 * * * *'::text, p_detach_schedule text DEFAULT '29 * * * *'::text) RETURNS TABLE(job_name text, job_id bigint)
    LANGUAGE plpgsql
    AS $_$
declare
  v_queue_exists boolean := false;
  v_queue_literal text;
  v_partition_job_name text;
  v_cleanup_job_name text;
  v_detach_plan_job_name text;
  v_partition_command text;
  v_cleanup_command text;
  v_detach_plan_command text;
  v_partitions_job_id bigint;
  v_cleanup_job_id bigint;
  v_detach_plan_job_id bigint;
  v_existing_job_id bigint;
  v_job_suffix text;
begin
  if p_queue_name is not null then
    p_queue_name := absurd.validate_queue_name(p_queue_name);

    select exists (
      select 1
      from absurd.queues
      where queue_name = p_queue_name
    )
    into v_queue_exists;

    if not v_queue_exists then
      raise exception 'Queue "%" does not exist', p_queue_name;
    end if;
  end if;

  if p_partition_schedule is null or length(trim(p_partition_schedule)) = 0 then
    raise exception 'Partition schedule must be provided';
  end if;

  if p_cleanup_schedule is null or length(trim(p_cleanup_schedule)) = 0 then
    raise exception 'Cleanup schedule must be provided';
  end if;

  if p_detach_schedule is null or length(trim(p_detach_schedule)) = 0 then
    raise exception 'Detach schedule must be provided';
  end if;

  if to_regclass('cron.job') is null then
    raise exception 'pg_cron is not available (missing cron.job)';
  end if;

  if not exists (
    select 1
    from pg_proc p
    join pg_namespace n on n.oid = p.pronamespace
    where n.nspname = 'cron'
      and p.proname = 'schedule'
  ) then
    raise exception 'pg_cron is not available (missing cron.schedule)';
  end if;

  if not exists (
    select 1
    from pg_proc p
    join pg_namespace n on n.oid = p.pronamespace
    where n.nspname = 'cron'
      and p.proname = 'unschedule'
  ) then
    raise exception 'pg_cron is not available (missing cron.unschedule)';
  end if;

  v_queue_literal := case
    when p_queue_name is null then 'null::text'
    else quote_literal(p_queue_name)
  end;

  v_partition_command := format(
    'select absurd.ensure_partitions(%s);',
    v_queue_literal
  );

  v_cleanup_command := format(
    'select * from absurd.cleanup_all_queues(%s);',
    v_queue_literal
  );

  v_job_suffix := case
    when p_queue_name is null then 'all'
    else substr(md5(p_queue_name), 1, 12)
  end;

  v_partition_job_name := 'absurd_partitions_' || v_job_suffix;
  v_cleanup_job_name := 'absurd_cleanup_' || v_job_suffix;
  v_detach_plan_job_name := 'absurd_detach_plan_' || v_job_suffix;

  v_detach_plan_command := format(
    'select * from absurd.schedule_detach_jobs(%s);',
    v_queue_literal
  );

  for v_existing_job_id in
    execute 'select jobid from cron.job where jobname = $1'
    using v_partition_job_name
  loop
    execute 'select cron.unschedule($1)' using v_existing_job_id;
  end loop;

  for v_existing_job_id in
    execute 'select jobid from cron.job where jobname = $1'
    using v_cleanup_job_name
  loop
    execute 'select cron.unschedule($1)' using v_existing_job_id;
  end loop;

  for v_existing_job_id in
    execute 'select jobid from cron.job where jobname = $1'
    using v_detach_plan_job_name
  loop
    execute 'select cron.unschedule($1)' using v_existing_job_id;
  end loop;

  execute 'select cron.schedule($1, $2, $3)'
    into v_partitions_job_id
    using v_partition_job_name, p_partition_schedule, v_partition_command;

  execute 'select cron.schedule($1, $2, $3)'
    into v_cleanup_job_id
    using v_cleanup_job_name, p_cleanup_schedule, v_cleanup_command;

  execute 'select cron.schedule($1, $2, $3)'
    into v_detach_plan_job_id
    using v_detach_plan_job_name, p_detach_schedule, v_detach_plan_command;

  job_name := v_partition_job_name;
  job_id := v_partitions_job_id;
  return next;

  job_name := v_cleanup_job_name;
  job_id := v_cleanup_job_id;
  return next;

  job_name := v_detach_plan_job_name;
  job_id := v_detach_plan_job_id;
  return next;
end;
$_$;


--
-- Name: ensure_partitions(text); Type: FUNCTION; Schema: absurd; Owner: -
--

CREATE FUNCTION absurd.ensure_partitions(p_queue_name text DEFAULT NULL::text) RETURNS void
    LANGUAGE plpgsql
    AS $$
declare
  v_now timestamptz := absurd.current_time();
  v_window_start timestamptz;
  v_window_end timestamptz;
  v_week_start timestamptz;
  v_week_end timestamptz;
  v_partition_tag text;
  v_uuid_from uuid;
  v_uuid_to uuid;
  v_queue record;
begin
  if p_queue_name is not null then
    p_queue_name := absurd.validate_queue_name(p_queue_name);

    if not exists (
      select 1
      from absurd.queues q
      where q.queue_name = p_queue_name
    ) then
      raise exception 'Queue "%" does not exist', p_queue_name;
    end if;
  end if;

  for v_queue in
    select
      queue_name,
      default_partition,
      partition_lookahead,
      partition_lookback
    from absurd.queues
    where storage_mode = 'partitioned'
      and (p_queue_name is null or queue_name = p_queue_name)
    order by queue_name
  loop
    v_window_start := absurd.week_bucket_utc(v_now - v_queue.partition_lookback);
    v_window_end := absurd.week_bucket_utc(v_now + v_queue.partition_lookahead);

    if v_queue.default_partition = 'enabled' then
      execute format(
        'create table if not exists absurd.%I partition of absurd.%I default',
        't_' || v_queue.queue_name || '_d',
        't_' || v_queue.queue_name
      );
      execute format(
        'create table if not exists absurd.%I partition of absurd.%I default',
        'r_' || v_queue.queue_name || '_d',
        'r_' || v_queue.queue_name
      );
      execute format(
        'create table if not exists absurd.%I partition of absurd.%I default',
        'c_' || v_queue.queue_name || '_d',
        'c_' || v_queue.queue_name
      );
      execute format(
        'create table if not exists absurd.%I partition of absurd.%I default',
        'w_' || v_queue.queue_name || '_d',
        'w_' || v_queue.queue_name
      );
    end if;

    v_week_start := v_window_start;
    while v_week_start <= v_window_end loop
      v_week_end := v_week_start + interval '7 days';
      v_partition_tag := absurd.partition_week_tag(v_week_start);
      v_uuid_from := absurd.uuidv7_floor(v_week_start);
      v_uuid_to := absurd.uuidv7_floor(v_week_end);

      execute format(
        'create table if not exists absurd.%I partition of absurd.%I
         for values from (%L::uuid) to (%L::uuid)',
        't_' || v_queue.queue_name || '_' || v_partition_tag,
        't_' || v_queue.queue_name,
        v_uuid_from,
        v_uuid_to
      );
      execute format(
        'create table if not exists absurd.%I partition of absurd.%I
         for values from (%L::uuid) to (%L::uuid)',
        'r_' || v_queue.queue_name || '_' || v_partition_tag,
        'r_' || v_queue.queue_name,
        v_uuid_from,
        v_uuid_to
      );
      execute format(
        'create table if not exists absurd.%I partition of absurd.%I
         for values from (%L::uuid) to (%L::uuid)',
        'c_' || v_queue.queue_name || '_' || v_partition_tag,
        'c_' || v_queue.queue_name,
        v_uuid_from,
        v_uuid_to
      );
      execute format(
        'create table if not exists absurd.%I partition of absurd.%I
         for values from (%L::uuid) to (%L::uuid)',
        'w_' || v_queue.queue_name || '_' || v_partition_tag,
        'w_' || v_queue.queue_name,
        v_uuid_from,
        v_uuid_to
      );

      v_week_start := v_week_end;
    end loop;
  end loop;
end;
$$;


--
-- Name: ensure_queue_tables(text); Type: FUNCTION; Schema: absurd; Owner: -
--

CREATE FUNCTION absurd.ensure_queue_tables(p_queue_name text) RETURNS void
    LANGUAGE plpgsql
    AS $$
declare
  v_storage_mode text := 'unpartitioned';
  v_t_suffix text;
  v_r_suffix text;
  v_c_suffix text;
  v_w_suffix text;
  v_t_idempotency_def text;
begin
  perform absurd.validate_queue_name(p_queue_name);

  select storage_mode into v_storage_mode
  from absurd.queues
  where queue_name = p_queue_name;

  v_storage_mode := coalesce(v_storage_mode, 'unpartitioned');

  if v_storage_mode not in ('unpartitioned', 'partitioned') then
    raise exception 'Unsupported queue storage mode "%"', v_storage_mode;
  end if;

  if v_storage_mode = 'partitioned' then
    v_t_suffix := 'partition by range (task_id)';
    v_r_suffix := 'partition by range (run_id)';
    v_c_suffix := 'partition by range (task_id)';
    v_w_suffix := 'partition by range (run_id)';
    v_t_idempotency_def := 'idempotency_key text';
  else
    v_t_suffix := 'with (fillfactor=70)';
    v_r_suffix := 'with (fillfactor=70)';
    v_c_suffix := 'with (fillfactor=70)';
    v_w_suffix := '';
    v_t_idempotency_def := 'idempotency_key text unique';
  end if;

  execute format(
    'create table if not exists absurd.%I (
        task_id uuid primary key,
        task_name text not null,
        params jsonb not null,
        headers jsonb,
        retry_strategy jsonb,
        max_attempts integer,
        cancellation jsonb,
        enqueue_at timestamptz not null default absurd.current_time(),
        first_started_at timestamptz,
        state text not null check (state in (''pending'', ''running'', ''sleeping'', ''completed'', ''failed'', ''cancelled'')),
        attempts integer not null default 0,
        last_attempt_run uuid,
        completed_payload jsonb,
        cancelled_at timestamptz,
        %s
     ) %s',
    't_' || p_queue_name,
    v_t_idempotency_def,
    v_t_suffix
  );

  execute format(
    'create table if not exists absurd.%I (
        run_id uuid primary key,
        task_id uuid not null,
        attempt integer not null,
        state text not null check (state in (''pending'', ''running'', ''sleeping'', ''completed'', ''failed'', ''cancelled'')),
        claimed_by text,
        claim_expires_at timestamptz,
        available_at timestamptz not null,
        wake_event text,
        event_payload jsonb,
        started_at timestamptz,
        completed_at timestamptz,
        failed_at timestamptz,
        result jsonb,
        failure_reason jsonb,
        created_at timestamptz not null default absurd.current_time()
     ) %s',
    'r_' || p_queue_name,
    v_r_suffix
  );

  execute format(
    'create table if not exists absurd.%I (
        task_id uuid not null,
        checkpoint_name text not null,
        state jsonb,
        status text not null default ''committed'',
        owner_run_id uuid,
        updated_at timestamptz not null default absurd.current_time(),
        primary key (task_id, checkpoint_name)
     ) %s',
    'c_' || p_queue_name,
    v_c_suffix
  );

  execute format(
    'create table if not exists absurd.%I (
        event_name text primary key,
        payload jsonb,
        emitted_at timestamptz not null default absurd.current_time()
     )',
    'e_' || p_queue_name
  );

  execute format(
    'create table if not exists absurd.%I (
        task_id uuid not null,
        run_id uuid not null,
        step_name text not null,
        event_name text not null,
        timeout_at timestamptz,
        created_at timestamptz not null default absurd.current_time(),
        primary key (run_id, step_name)
     ) %s',
    'w_' || p_queue_name,
    v_w_suffix
  );

  if v_storage_mode = 'partitioned' then
    execute format(
      'create table if not exists absurd.%I (
          idempotency_key text primary key,
          task_id uuid not null
       )',
      'i_' || p_queue_name
    );
  end if;

  execute format(
    'create index if not exists %I on absurd.%I (state, available_at)',
    ('r_' || p_queue_name) || '_sai',
    'r_' || p_queue_name
  );

  execute format(
    'create index if not exists %I on absurd.%I (task_id)',
    ('r_' || p_queue_name) || '_ti',
    'r_' || p_queue_name
  );

  execute format(
    'create index if not exists %I on absurd.%I (claim_expires_at)
      where state = ''running''
        and claim_expires_at is not null',
    ('r_' || p_queue_name) || '_cei',
    'r_' || p_queue_name
  );

  execute format(
    'create index if not exists %I on absurd.%I (event_name)',
    ('w_' || p_queue_name) || '_eni',
    'w_' || p_queue_name
  );

  execute format(
    'create index if not exists %I on absurd.%I (task_id)',
    ('w_' || p_queue_name) || '_ti',
    'w_' || p_queue_name
  );

  execute format(
    'create index if not exists %I on absurd.%I (emitted_at)',
    ('e_' || p_queue_name) || '_eai',
    'e_' || p_queue_name
  );

  if v_storage_mode = 'partitioned' then
    execute format(
      'create index if not exists %I on absurd.%I (task_id)',
      ('i_' || p_queue_name) || '_ti',
      'i_' || p_queue_name
    );

    perform absurd.ensure_partitions(p_queue_name);
  end if;
end;
$$;


--
-- Name: extend_claim(text, uuid, integer); Type: FUNCTION; Schema: absurd; Owner: -
--

CREATE FUNCTION absurd.extend_claim(p_queue_name text, p_run_id uuid, p_extend_by integer) RETURNS void
    LANGUAGE plpgsql
    AS $_$
declare
  v_now timestamptz := absurd.current_time();
  v_task_state text;
  v_run_state text;
  v_claim_expires_at timestamptz;
begin
  if p_extend_by is null or p_extend_by <= 0 then
    raise exception 'extend_by must be > 0';
  end if;

  execute format(
    'select r.state,
            r.claim_expires_at,
            t.state
       from absurd.%I r
       join absurd.%I t on t.task_id = r.task_id
      where r.run_id = $1
      for update',
    'r_' || p_queue_name,
    't_' || p_queue_name
  )
  into v_run_state, v_claim_expires_at, v_task_state
  using p_run_id;

  if v_run_state is null then
    raise exception 'Run "%" not found in queue "%"', p_run_id, p_queue_name;
  end if;

  if v_task_state = 'cancelled' then
    raise exception sqlstate 'AB001' using message = 'Task has been cancelled';
  end if;

  if v_run_state <> 'running' then
    if v_run_state = 'failed' then
      raise exception sqlstate 'AB002' using message = format('Run "%s" has already failed in queue "%s"', p_run_id, p_queue_name);
    end if;
    raise exception 'Run "%" is not currently running in queue "%"', p_run_id, p_queue_name;
  end if;

  if v_claim_expires_at is null then
    raise exception 'Run "%" does not have an active claim in queue "%"', p_run_id, p_queue_name;
  end if;

  execute format(
    'update absurd.%I
        set claim_expires_at = $2 + make_interval(secs => $3)
      where run_id = $1',
    'r_' || p_queue_name
  )
  using p_run_id, v_now, p_extend_by;
end;
$_$;


--
-- Name: fail_run(text, uuid, jsonb, timestamp with time zone); Type: FUNCTION; Schema: absurd; Owner: -
--

CREATE FUNCTION absurd.fail_run(p_queue_name text, p_run_id uuid, p_reason jsonb, p_retry_at timestamp with time zone DEFAULT NULL::timestamp with time zone) RETURNS void
    LANGUAGE plpgsql
    AS $_$
declare
  v_task_id uuid;
  v_attempt integer;
  v_run_state text;
  v_retry_strategy jsonb;
  v_max_attempts integer;
  v_now timestamptz := absurd.current_time();
  v_next_attempt integer;
  v_delay_seconds double precision := 0;
  v_next_available timestamptz;
  v_retry_kind text;
  v_base double precision;
  v_factor double precision;
  v_max_seconds double precision;
  v_first_started timestamptz;
  v_cancellation jsonb;
  v_max_duration bigint;
  v_task_cancel boolean := false;
  v_new_run_id uuid;
  v_task_state_after text;
  v_recorded_attempt integer;
  v_last_attempt_run uuid := p_run_id;
  v_cancelled_at timestamptz := null;
begin
  execute format(
    'select r.task_id, r.attempt, r.state
       from absurd.%I r
      where r.run_id = $1
      for update',
    'r_' || p_queue_name
  )
  into v_task_id, v_attempt, v_run_state
  using p_run_id;

  if v_task_id is null then
    raise exception 'Run "%" cannot be failed in queue "%"', p_run_id, p_queue_name;
  end if;

  if v_run_state = 'cancelled' then
    raise exception sqlstate 'AB001' using message = 'Task has been cancelled';
  end if;

  if v_run_state = 'failed' then
    raise exception sqlstate 'AB002' using message = format('Run "%s" has already failed in queue "%s"', p_run_id, p_queue_name);
  end if;

  if v_run_state not in ('running', 'sleeping') then
    raise exception 'Run "%" cannot be failed in queue "%"', p_run_id, p_queue_name;
  end if;

  execute format(
    'select retry_strategy, max_attempts, first_started_at, cancellation
       from absurd.%I
      where task_id = $1
      for update',
    't_' || p_queue_name
  )
  into v_retry_strategy, v_max_attempts, v_first_started, v_cancellation
  using v_task_id;

  execute format(
    'update absurd.%I
        set state = ''failed'',
            wake_event = null,
            failed_at = $2,
            failure_reason = $3
      where run_id = $1',
    'r_' || p_queue_name
  ) using p_run_id, v_now, p_reason;

  v_next_attempt := v_attempt + 1;
  v_task_state_after := 'failed';
  v_recorded_attempt := v_attempt;

  if v_max_attempts is null or v_next_attempt <= v_max_attempts then
    if p_retry_at is not null then
      v_next_available := p_retry_at;
    else
      v_retry_kind := coalesce(v_retry_strategy->>'kind', 'none');
      if v_retry_kind = 'fixed' then
        v_base := coalesce((v_retry_strategy->>'base_seconds')::double precision, 60);
        v_delay_seconds := v_base;
      elsif v_retry_kind = 'exponential' then
        v_base := coalesce((v_retry_strategy->>'base_seconds')::double precision, 30);
        v_factor := coalesce((v_retry_strategy->>'factor')::double precision, 2);
        v_delay_seconds := v_base * power(v_factor, greatest(v_attempt - 1, 0));
        v_max_seconds := (v_retry_strategy->>'max_seconds')::double precision;
        if v_max_seconds is not null then
          v_delay_seconds := least(v_delay_seconds, v_max_seconds);
        end if;
      else
        v_delay_seconds := 0;
      end if;
      v_next_available := v_now + (v_delay_seconds * interval '1 second');
    end if;

    if v_next_available < v_now then
      v_next_available := v_now;
    end if;

    if v_cancellation is not null then
      v_max_duration := (v_cancellation->>'max_duration')::bigint;
      if v_max_duration is not null and v_first_started is not null then
        if extract(epoch from (v_next_available - v_first_started)) >= v_max_duration then
          v_task_cancel := true;
        end if;
      end if;
    end if;

    if not v_task_cancel then
      v_task_state_after := case when v_next_available > v_now then 'sleeping' else 'pending' end;
      v_new_run_id := absurd.portable_uuidv7();
      v_recorded_attempt := v_next_attempt;
      v_last_attempt_run := v_new_run_id;
      execute format(
        'insert into absurd.%I (run_id, task_id, attempt, state, available_at, wake_event, event_payload, result, failure_reason)
         values ($1, $2, $3, $4, $5, null, null, null, null)',
        'r_' || p_queue_name
      )
      using v_new_run_id, v_task_id, v_next_attempt, v_task_state_after, v_next_available;
    end if;
  end if;

  if v_task_cancel then
    v_task_state_after := 'cancelled';
    v_cancelled_at := v_now;
    v_recorded_attempt := greatest(v_recorded_attempt, v_attempt);
    v_last_attempt_run := p_run_id;
  end if;

  execute format(
    'update absurd.%I
        set state = $2,
            attempts = greatest(attempts, $3),
            last_attempt_run = $4,
            cancelled_at = coalesce(cancelled_at, $5)
      where task_id = $1',
    't_' || p_queue_name
  ) using v_task_id, v_task_state_after, v_recorded_attempt, v_last_attempt_run, v_cancelled_at;

  execute format(
    'delete from absurd.%I where run_id = $1',
    'w_' || p_queue_name
  ) using p_run_id;
end;
$_$;


--
-- Name: get_queue_policy(text); Type: FUNCTION; Schema: absurd; Owner: -
--

CREATE FUNCTION absurd.get_queue_policy(p_queue_name text) RETURNS TABLE(queue_name text, storage_mode text, default_partition text, partition_lookahead interval, partition_lookback interval, cleanup_ttl interval, cleanup_limit integer, detach_mode text, detach_min_age interval)
    LANGUAGE sql
    AS $$
  select
    q.queue_name,
    q.storage_mode,
    q.default_partition,
    q.partition_lookahead,
    q.partition_lookback,
    q.cleanup_ttl,
    q.cleanup_limit,
    q.detach_mode,
    q.detach_min_age
  from absurd.queues q
  where q.queue_name = p_queue_name;
$$;


--
-- Name: get_schema_version(); Type: FUNCTION; Schema: absurd; Owner: -
--

CREATE FUNCTION absurd.get_schema_version() RETURNS text
    LANGUAGE sql
    AS $$
  select 'main'::text;
$$;


--
-- Name: get_task_checkpoint_state(text, uuid, text, boolean); Type: FUNCTION; Schema: absurd; Owner: -
--

CREATE FUNCTION absurd.get_task_checkpoint_state(p_queue_name text, p_task_id uuid, p_step_name text, p_include_pending boolean DEFAULT false) RETURNS TABLE(checkpoint_name text, state jsonb, status text, owner_run_id uuid, updated_at timestamp with time zone)
    LANGUAGE plpgsql
    AS $_$
begin
  return query execute format(
    'select checkpoint_name, state, status, owner_run_id, updated_at
       from absurd.%I
      where task_id = $1
        and checkpoint_name = $2
        and ($3 or status = ''committed'')',
    'c_' || p_queue_name
  ) using p_task_id, p_step_name, coalesce(p_include_pending, false);
end;
$_$;


--
-- Name: get_task_checkpoint_states(text, uuid, uuid); Type: FUNCTION; Schema: absurd; Owner: -
--

CREATE FUNCTION absurd.get_task_checkpoint_states(p_queue_name text, p_task_id uuid, p_run_id uuid) RETURNS TABLE(checkpoint_name text, state jsonb, status text, owner_run_id uuid, updated_at timestamp with time zone)
    LANGUAGE plpgsql
    AS $_$
declare
  v_run_task_id uuid;
  v_run_attempt integer;
begin
  execute format(
    'select task_id, attempt
       from absurd.%I
      where run_id = $1',
    'r_' || p_queue_name
  )
  into v_run_task_id, v_run_attempt
  using p_run_id;

  if v_run_task_id is null then
    raise exception 'Run "%" not found in queue "%"', p_run_id, p_queue_name;
  end if;

  if v_run_task_id <> p_task_id then
    raise exception 'Run "%" does not belong to task "%" in queue "%"', p_run_id, p_task_id, p_queue_name;
  end if;

  return query execute format(
    'select c.checkpoint_name,
            c.state,
            c.status,
            c.owner_run_id,
            c.updated_at
       from absurd.%1$I c
       left join absurd.%2$I owner_run on owner_run.run_id = c.owner_run_id
      where c.task_id = $1
        and c.status = ''committed''
        and (owner_run.attempt is null or owner_run.attempt <= $2)
      order by c.updated_at asc',
    'c_' || p_queue_name,
    'r_' || p_queue_name
  ) using p_task_id, v_run_attempt;
end;
$_$;


--
-- Name: get_task_result(text, uuid); Type: FUNCTION; Schema: absurd; Owner: -
--

CREATE FUNCTION absurd.get_task_result(p_queue_name text, p_task_id uuid) RETURNS TABLE(task_id uuid, state text, result jsonb, failure_reason jsonb)
    LANGUAGE plpgsql
    AS $_$
begin
  p_queue_name := absurd.validate_queue_name(p_queue_name);

  return query execute format(
    'select t.task_id,
            t.state,
            case when t.state = ''completed'' then t.completed_payload else null end as result,
            case when t.state = ''failed'' then r.failure_reason else null end as failure_reason
       from absurd.%I t
       left join absurd.%I r on r.run_id = t.last_attempt_run
      where t.task_id = $1',
    't_' || p_queue_name,
    'r_' || p_queue_name
  ) using p_task_id;
end;
$_$;


--
-- Name: list_detach_candidates(text); Type: FUNCTION; Schema: absurd; Owner: -
--

CREATE FUNCTION absurd.list_detach_candidates(p_queue_name text DEFAULT NULL::text) RETURNS TABLE(queue_name text, parent_table text, partition_table text)
    LANGUAGE plpgsql
    AS $$
declare
  v_now timestamptz := absurd.current_time();
  v_queue record;
  v_parent_prefix text;
  v_parent_table text;
  v_parent_oid oid;
  v_part record;
  v_upper_uuid uuid;
  v_upper_ts timestamptz;
  v_has_rows boolean;
begin
  if p_queue_name is not null then
    p_queue_name := absurd.validate_queue_name(p_queue_name);

    if not exists (
      select 1
      from absurd.queues q
      where q.queue_name = p_queue_name
    ) then
      raise exception 'Queue "%" does not exist', p_queue_name;
    end if;
  end if;

  for v_queue in
    select
      q.queue_name,
      q.detach_mode,
      q.detach_min_age
    from absurd.queues q
    where q.storage_mode = 'partitioned'
      and q.detach_mode = 'empty'
      and (p_queue_name is null or q.queue_name = p_queue_name)
    order by q.queue_name
  loop
    foreach v_parent_prefix in array array['t', 'r', 'c', 'w'] loop
      v_parent_table := v_parent_prefix || '_' || v_queue.queue_name;

      select c.oid
        into v_parent_oid
        from pg_class c
        join pg_namespace n on n.oid = c.relnamespace
       where n.nspname = 'absurd'
         and c.relname = v_parent_table;

      if v_parent_oid is null then
        continue;
      end if;

      for v_part in
        select
          child.relname as partition_name,
          pg_get_expr(child.relpartbound, child.oid) as part_bound
        from pg_inherits inh
        join pg_class child on child.oid = inh.inhrelid
        where inh.inhparent = v_parent_oid
      loop
        if v_part.part_bound = 'DEFAULT' then
          continue;
        end if;

        select
          (regexp_match(v_part.part_bound, 'TO \(''([^'']+)''(::uuid)?\)'))[1]::uuid
          into v_upper_uuid;

        if v_upper_uuid is null then
          continue;
        end if;

        v_upper_ts := absurd.uuidv7_timestamp(v_upper_uuid);

        if v_upper_ts is null then
          continue;
        end if;

        if v_upper_ts >= (v_now - v_queue.detach_min_age) then
          continue;
        end if;

        execute format(
          'select exists (select 1 from absurd.%I limit 1)',
          v_part.partition_name
        )
        into v_has_rows;

        if coalesce(v_has_rows, false) then
          continue;
        end if;

        queue_name := v_queue.queue_name;
        parent_table := v_parent_table;
        partition_table := v_part.partition_name;
        return next;
      end loop;
    end loop;
  end loop;
end;
$$;


--
-- Name: list_queues(); Type: FUNCTION; Schema: absurd; Owner: -
--

CREATE FUNCTION absurd.list_queues() RETURNS TABLE(queue_name text)
    LANGUAGE sql
    AS $$
  select queue_name from absurd.queues order by queue_name;
$$;


--
-- Name: partition_week_tag(timestamp with time zone); Type: FUNCTION; Schema: absurd; Owner: -
--

CREATE FUNCTION absurd.partition_week_tag(p_ts timestamp with time zone) RETURNS text
    LANGUAGE sql IMMUTABLE STRICT
    AS $$
  with bucket as (
    select absurd.week_bucket_utc(p_ts) at time zone 'UTC' as ts
  )
  select
    ((extract(isoyear from ts)::int % 10)::text) ||
    lpad((extract(week from ts)::int)::text, 2, '0')
  from bucket;
$$;


--
-- Name: portable_uuidv7(); Type: FUNCTION; Schema: absurd; Owner: -
--

CREATE FUNCTION absurd.portable_uuidv7() RETURNS uuid
    LANGUAGE plpgsql
    AS $$
declare
  v_server_num integer := current_setting('server_version_num')::int;
  ts_ms bigint;
  b bytea;
  rnd bytea;
  i int;
begin
  if v_server_num >= 180000 then
    return uuidv7 ();
  end if;
  ts_ms := floor(extract(epoch from absurd.current_time()) * 1000)::bigint;
  rnd := uuid_send(uuid_generate_v4 ());
  b := repeat(E'\\000', 16)::bytea;
  for i in 0..5 loop
    b := set_byte(b, i, ((ts_ms >> ((5 - i) * 8)) & 255)::int);
  end loop;
  for i in 6..15 loop
    b := set_byte(b, i, get_byte(rnd, i));
  end loop;
  b := set_byte(b, 6, ((get_byte(b, 6) & 15) | (7 << 4)));
  b := set_byte(b, 8, ((get_byte(b, 8) & 63) | 128));
  return encode(b, 'hex')::uuid;
end;
$$;


--
-- Name: retry_task(text, uuid, jsonb); Type: FUNCTION; Schema: absurd; Owner: -
--

CREATE FUNCTION absurd.retry_task(p_queue_name text, p_task_id uuid, p_options jsonb DEFAULT '{}'::jsonb) RETURNS TABLE(task_id uuid, run_id uuid, attempt integer, created boolean)
    LANGUAGE plpgsql
    AS $_$
declare
  v_now timestamptz := absurd.current_time();
  v_spawn_new boolean := false;
  v_requested_max_attempts integer;

  v_task_name text;
  v_params jsonb;
  v_headers jsonb;
  v_retry_strategy jsonb;
  v_task_max_attempts integer;
  v_cancellation jsonb;
  v_task_attempts integer;
  v_task_state text;

  v_new_run_id uuid;
  v_new_attempt integer;
  v_spawn_options jsonb;
begin
  if p_options is not null then
    if p_options ? 'spawn_new' then
      v_spawn_new := coalesce((p_options->>'spawn_new')::boolean, false);
    end if;
    if p_options ? 'max_attempts' then
      v_requested_max_attempts := (p_options->>'max_attempts')::int;
      if v_requested_max_attempts is not null and v_requested_max_attempts < 1 then
        raise exception 'max_attempts must be >= 1';
      end if;
    end if;
  end if;

  execute format(
    'select task_name,
            params,
            headers,
            retry_strategy,
            max_attempts,
            cancellation,
            attempts,
            state
       from absurd.%I
      where task_id = $1
      for update',
    't_' || p_queue_name
  )
  into v_task_name,
       v_params,
       v_headers,
       v_retry_strategy,
       v_task_max_attempts,
       v_cancellation,
       v_task_attempts,
       v_task_state
  using p_task_id;

  if v_task_state is null then
    raise exception 'Task "%" not found in queue "%"', p_task_id, p_queue_name;
  end if;

  if v_task_state <> 'failed' then
    raise exception 'Task "%" is not currently failed in queue "%"', p_task_id, p_queue_name;
  end if;

  if v_spawn_new then
    v_spawn_options := jsonb_strip_nulls(jsonb_build_object(
      'headers', v_headers,
      'retry_strategy', v_retry_strategy,
      'max_attempts', coalesce(v_requested_max_attempts, v_task_max_attempts),
      'cancellation', v_cancellation
    ));

    return query
      select s.task_id, s.run_id, s.attempt, s.created
        from absurd.spawn_task(p_queue_name, v_task_name, v_params, v_spawn_options) s;
    return;
  end if;

  if v_requested_max_attempts is null then
    v_requested_max_attempts := coalesce(v_task_max_attempts, v_task_attempts) + 1;
  end if;

  if v_requested_max_attempts <= v_task_attempts then
    raise exception 'max_attempts (%) must be greater than current attempts (%)',
      v_requested_max_attempts,
      v_task_attempts;
  end if;

  v_new_run_id := absurd.portable_uuidv7();
  v_new_attempt := v_task_attempts + 1;

  execute format(
    'insert into absurd.%I (run_id, task_id, attempt, state, available_at, wake_event, event_payload, result, failure_reason)
     values ($1, $2, $3, ''pending'', $4, null, null, null, null)',
    'r_' || p_queue_name
  )
  using v_new_run_id, p_task_id, v_new_attempt, v_now;

  execute format(
    'update absurd.%I
        set state = ''pending'',
            attempts = greatest(attempts, $2),
            max_attempts = $3,
            last_attempt_run = $4,
            cancelled_at = null
      where task_id = $1',
    't_' || p_queue_name
  )
  using p_task_id, v_new_attempt, v_requested_max_attempts, v_new_run_id;

  return query select p_task_id, v_new_run_id, v_new_attempt, false;
end;
$_$;


--
-- Name: schedule_detach_jobs(text); Type: FUNCTION; Schema: absurd; Owner: -
--

CREATE FUNCTION absurd.schedule_detach_jobs(p_queue_name text DEFAULT NULL::text) RETURNS TABLE(job_name text, job_id bigint, queue_name text, partition_table text, job_kind text)
    LANGUAGE plpgsql
    AS $_$
declare
  v_scope text;
  v_candidate record;
  v_parent_key text;
  v_candidate_key text;
  v_detach_job_name text;
  v_drop_job_name text;
  v_detach_command text;
  v_drop_command text;
  v_parent_has_default_partition boolean;
  v_job_id bigint;
begin
  if p_queue_name is not null then
    p_queue_name := absurd.validate_queue_name(p_queue_name);
  end if;

  if to_regclass('cron.job') is null then
    raise exception 'pg_cron is not available (missing cron.job)';
  end if;

  if not exists (
    select 1
    from pg_proc p
    join pg_namespace n on n.oid = p.pronamespace
    where n.nspname = 'cron'
      and p.proname = 'schedule'
  ) then
    raise exception 'pg_cron is not available (missing cron.schedule)';
  end if;

  if not exists (
    select 1
    from pg_proc p
    join pg_namespace n on n.oid = p.pronamespace
    where n.nspname = 'cron'
      and p.proname = 'unschedule'
  ) then
    raise exception 'pg_cron is not available (missing cron.unschedule)';
  end if;

  v_scope := case
    when p_queue_name is null then 'all'
    else substr(md5(p_queue_name), 1, 12)
  end;

  for v_candidate in
    with candidates as (
      select
        c.*,
        absurd.uuidv7_timestamp(
          (regexp_match(
            pg_get_expr(child.relpartbound, child.oid),
            'TO \(''([^'']+)''(::uuid)?\)'
          ))[1]::uuid
        ) as upper_ts
      from absurd.list_detach_candidates(p_queue_name) c
      join pg_class child on child.relname = c.partition_table
      join pg_namespace n on n.oid = child.relnamespace
      where n.nspname = 'absurd'
    ),
    ranked as (
      select
        candidates.*,
        row_number() over (
          partition by candidates.parent_table
          order by candidates.upper_ts asc nulls last, candidates.partition_table asc
        ) as rn
      from candidates
    )
    select
      ranked.queue_name,
      ranked.parent_table,
      ranked.partition_table
    from ranked
    where ranked.rn = 1
    order by ranked.queue_name, ranked.parent_table, ranked.partition_table
  loop
    v_parent_key := substr(md5(v_candidate.parent_table), 1, 8);

    -- Only one active detach pipeline per parent table.
    if exists (
      select 1
      from cron.job
      where jobname like ('absurd_detach_run_%_' || v_parent_key || '_%')
         or jobname like ('absurd_drop_run_%_' || v_parent_key || '_%')
    ) then
      continue;
    end if;

    v_candidate_key := substr(
      md5(v_candidate.parent_table || ':' || v_candidate.partition_table),
      1,
      12
    );

    v_detach_job_name := format(
      'absurd_detach_run_%s_%s_%s',
      v_scope,
      v_parent_key,
      v_candidate_key
    );
    v_drop_job_name := format(
      'absurd_drop_run_%s_%s_%s',
      v_scope,
      v_parent_key,
      v_candidate_key
    );

    if not exists (
      select 1
      from cron.job
      where jobname = v_detach_job_name
         or jobname like ('absurd_detach_run_%_' || v_candidate_key)
    ) then
      select exists (
        select 1
        from pg_class parent
        join pg_namespace pn on pn.oid = parent.relnamespace
        join pg_inherits inh on inh.inhparent = parent.oid
        join pg_class child on child.oid = inh.inhrelid
        where pn.nspname = 'absurd'
          and parent.relname = v_candidate.parent_table
          and pg_get_expr(child.relpartbound, child.oid) = 'DEFAULT'
      )
      into v_parent_has_default_partition;

      v_detach_command := format(
        'alter table absurd.%I detach partition absurd.%I',
        v_candidate.parent_table,
        v_candidate.partition_table
      );

      if not coalesce(v_parent_has_default_partition, false) then
        v_detach_command := v_detach_command || ' concurrently';
      end if;

      execute 'select cron.schedule($1, $2, $3)'
        into v_job_id
        using v_detach_job_name, '* * * * *', v_detach_command;

      job_name := v_detach_job_name;
      job_id := v_job_id;
      queue_name := v_candidate.queue_name;
      partition_table := v_candidate.partition_table;
      job_kind := 'detach';
      return next;
    end if;

    if not exists (
      select 1
      from cron.job
      where jobname = v_drop_job_name
         or jobname like ('absurd_drop_run_%_' || v_candidate_key)
    ) then
      v_drop_command := format(
        'select absurd.drop_detached_partition(%L, %L);',
        v_candidate.partition_table,
        v_drop_job_name
      );

      execute 'select cron.schedule($1, $2, $3)'
        into v_job_id
        using v_drop_job_name, '* * * * *', v_drop_command;

      job_name := v_drop_job_name;
      job_id := v_job_id;
      queue_name := v_candidate.queue_name;
      partition_table := v_candidate.partition_table;
      job_kind := 'drop';
      return next;
    end if;
  end loop;
end;
$_$;


--
-- Name: schedule_run(text, uuid, timestamp with time zone); Type: FUNCTION; Schema: absurd; Owner: -
--

CREATE FUNCTION absurd.schedule_run(p_queue_name text, p_run_id uuid, p_wake_at timestamp with time zone) RETURNS void
    LANGUAGE plpgsql
    AS $_$
declare
  v_task_id uuid;
begin
  execute format(
    'select task_id
       from absurd.%I
      where run_id = $1
        and state = ''running''
      for update',
    'r_' || p_queue_name
  )
  into v_task_id
  using p_run_id;

  if v_task_id is null then
    raise exception 'Run "%" is not currently running in queue "%"', p_run_id, p_queue_name;
  end if;

  execute format(
    'update absurd.%I
        set state = ''sleeping'',
            claimed_by = null,
            claim_expires_at = null,
            available_at = $2,
            wake_event = null
      where run_id = $1',
    'r_' || p_queue_name
  ) using p_run_id, p_wake_at;

  execute format(
    'update absurd.%I
        set state = ''sleeping''
      where task_id = $1',
    't_' || p_queue_name
  ) using v_task_id;
end;
$_$;


--
-- Name: set_queue_policy(text, jsonb); Type: FUNCTION; Schema: absurd; Owner: -
--

CREATE FUNCTION absurd.set_queue_policy(p_queue_name text, p_policy jsonb) RETURNS void
    LANGUAGE plpgsql
    AS $$
declare
  v_policy jsonb := coalesce(p_policy, '{}'::jsonb);
  v_unknown_key text;
  v_exists boolean := false;
  v_storage_mode text;
  v_default_partition text;
  v_previous_default_partition text;
  v_parent_prefix text;
  v_parent_table text;
  v_default_table text;
  v_default_attached boolean;
  v_default_has_rows boolean;

  v_partition_lookahead interval;
  v_partition_lookback interval;
  v_cleanup_ttl interval;
  v_cleanup_limit integer;
  v_detach_mode text;
  v_detach_min_age interval;
begin
  p_queue_name := absurd.validate_queue_name(p_queue_name);

  if jsonb_typeof(v_policy) <> 'object' then
    raise exception 'Queue policy must be a JSON object';
  end if;

  select k.key
    into v_unknown_key
    from jsonb_object_keys(v_policy) as k(key)
   where k.key not in (
      'partition_lookahead',
      'partition_lookback',
      'cleanup_ttl',
      'cleanup_limit',
      'detach_mode',
      'detach_min_age',
      'default_partition'
   )
   limit 1;

  if v_unknown_key is not null then
    raise exception 'Unsupported queue policy key "%"', v_unknown_key;
  end if;

  select exists (
    select 1
    from absurd.queues
    where queue_name = p_queue_name
  )
  into v_exists;

  if not v_exists then
    raise exception 'Queue "%" does not exist', p_queue_name;
  end if;

  select
    storage_mode,
    default_partition,
    partition_lookahead,
    partition_lookback,
    cleanup_ttl,
    cleanup_limit,
    detach_mode,
    detach_min_age
  into
    v_storage_mode,
    v_default_partition,
    v_partition_lookahead,
    v_partition_lookback,
    v_cleanup_ttl,
    v_cleanup_limit,
    v_detach_mode,
    v_detach_min_age
  from absurd.queues
  where queue_name = p_queue_name
  for update;

  if v_policy ? 'partition_lookahead' then
    v_partition_lookahead := (v_policy->>'partition_lookahead')::interval;
  end if;

  if v_policy ? 'partition_lookback' then
    v_partition_lookback := (v_policy->>'partition_lookback')::interval;
  end if;

  if v_policy ? 'cleanup_ttl' then
    v_cleanup_ttl := (v_policy->>'cleanup_ttl')::interval;
  end if;

  if v_policy ? 'cleanup_limit' then
    v_cleanup_limit := (v_policy->>'cleanup_limit')::integer;
  end if;

  if v_policy ? 'detach_mode' then
    v_detach_mode := lower(trim(coalesce(v_policy->>'detach_mode', '')));
  end if;

  if v_policy ? 'detach_min_age' then
    v_detach_min_age := (v_policy->>'detach_min_age')::interval;
  end if;

  v_previous_default_partition := v_default_partition;

  if v_policy ? 'default_partition' then
    v_default_partition := lower(trim(coalesce(v_policy->>'default_partition', '')));
  end if;

  if v_partition_lookahead < interval '0 seconds' then
    raise exception 'partition_lookahead must be non-negative';
  end if;

  if v_partition_lookback < interval '0 seconds' then
    raise exception 'partition_lookback must be non-negative';
  end if;

  if v_cleanup_ttl < interval '0 seconds' then
    raise exception 'cleanup_ttl must be non-negative';
  end if;

  if v_cleanup_limit < 1 then
    raise exception 'cleanup_limit must be at least 1';
  end if;

  if v_detach_mode not in ('none', 'empty') then
    raise exception 'Unsupported detach mode "%"', v_detach_mode;
  end if;

  if v_detach_min_age < interval '0 seconds' then
    raise exception 'detach_min_age must be non-negative';
  end if;

  if v_default_partition not in ('enabled', 'disabled') then
    raise exception 'Unsupported default_partition mode "%"', v_default_partition;
  end if;

  if v_storage_mode <> 'partitioned' and v_policy ? 'default_partition' then
    raise exception 'default_partition policy is only supported for partitioned queues';
  end if;

  update absurd.queues
     set default_partition = v_default_partition,
         partition_lookahead = v_partition_lookahead,
         partition_lookback = v_partition_lookback,
         cleanup_ttl = v_cleanup_ttl,
         cleanup_limit = v_cleanup_limit,
         detach_mode = v_detach_mode,
         detach_min_age = v_detach_min_age
   where queue_name = p_queue_name;

  if v_storage_mode = 'partitioned'
     and v_previous_default_partition <> v_default_partition then
    if v_default_partition = 'enabled' then
      perform absurd.ensure_partitions(p_queue_name);
    else
      foreach v_parent_prefix in array array['t', 'r', 'c', 'w'] loop
        v_parent_table := v_parent_prefix || '_' || p_queue_name;
        v_default_table := v_parent_table || '_d';

        select exists (
          select 1
          from pg_inherits inh
          join pg_class parent on parent.oid = inh.inhparent
          join pg_class child on child.oid = inh.inhrelid
          join pg_namespace n on n.oid = parent.relnamespace
          where n.nspname = 'absurd'
            and parent.relname = v_parent_table
            and child.relname = v_default_table
        )
        into v_default_attached;

        if not coalesce(v_default_attached, false) then
          continue;
        end if;

        -- Block out-of-window writes into the default partition while we
        -- validate emptiness and detach/drop it.
        execute format(
          'lock table absurd.%I in access exclusive mode',
          v_default_table
        );

        execute format(
          'select exists (select 1 from absurd.%I limit 1)',
          v_default_table
        )
        into v_default_has_rows;

        if coalesce(v_default_has_rows, false) then
          raise exception
            'Cannot disable default_partition for queue "%": default partition "%" is not empty',
            p_queue_name,
            v_default_table;
        end if;

        execute format(
          'alter table absurd.%I detach partition absurd.%I',
          v_parent_table,
          v_default_table
        );
        execute format('drop table if exists absurd.%I', v_default_table);
      end loop;
    end if;
  end if;
end;
$$;


--
-- Name: set_task_checkpoint_state(text, uuid, text, jsonb, uuid, integer); Type: FUNCTION; Schema: absurd; Owner: -
--

CREATE FUNCTION absurd.set_task_checkpoint_state(p_queue_name text, p_task_id uuid, p_step_name text, p_state jsonb, p_owner_run uuid, p_extend_claim_by integer DEFAULT NULL::integer) RETURNS void
    LANGUAGE plpgsql
    AS $_$
declare
  v_now timestamptz := absurd.current_time();
  v_new_attempt integer;
  v_existing_attempt integer;
  v_existing_owner uuid;
  v_task_state text;
  v_run_state text;
begin
  if p_step_name is null or length(trim(p_step_name)) = 0 then
    raise exception 'step_name must be provided';
  end if;

  execute format(
    'select r.attempt, r.state, t.state
       from absurd.%I r
       join absurd.%I t on t.task_id = r.task_id
      where r.run_id = $1',
    'r_' || p_queue_name,
    't_' || p_queue_name
  )
  into v_new_attempt, v_run_state, v_task_state
  using p_owner_run;

  if v_new_attempt is null then
    raise exception 'Run "%" not found for checkpoint', p_owner_run;
  end if;

  if v_task_state = 'cancelled' then
    raise exception sqlstate 'AB001' using message = 'Task has been cancelled';
  end if;

  if v_run_state = 'failed' then
    raise exception sqlstate 'AB002' using message = format('Run "%s" has already failed in queue "%s"', p_owner_run, p_queue_name);
  end if;

  -- Extend the claim if requested
  if p_extend_claim_by is not null and p_extend_claim_by > 0 then
    execute format(
      'update absurd.%I
          set claim_expires_at = $2 + make_interval(secs => $3)
        where run_id = $1
          and state = ''running''
          and claim_expires_at is not null',
      'r_' || p_queue_name
    )
    using p_owner_run, v_now, p_extend_claim_by;
  end if;

  execute format(
    'select c.owner_run_id,
            r.attempt
       from absurd.%I c
       left join absurd.%I r on r.run_id = c.owner_run_id
      where c.task_id = $1
        and c.checkpoint_name = $2',
    'c_' || p_queue_name,
    'r_' || p_queue_name
  )
  into v_existing_owner, v_existing_attempt
  using p_task_id, p_step_name;

  if v_existing_owner is null or v_existing_attempt is null or v_new_attempt >= v_existing_attempt then
    execute format(
      'insert into absurd.%I (task_id, checkpoint_name, state, status, owner_run_id, updated_at)
       values ($1, $2, $3, ''committed'', $4, $5)
       on conflict (task_id, checkpoint_name)
       do update set state = excluded.state,
                     status = excluded.status,
                     owner_run_id = excluded.owner_run_id,
                     updated_at = excluded.updated_at',
      'c_' || p_queue_name
    ) using p_task_id, p_step_name, p_state, p_owner_run, v_now;
  end if;
end;
$_$;


--
-- Name: spawn_task(text, text, jsonb, jsonb); Type: FUNCTION; Schema: absurd; Owner: -
--

CREATE FUNCTION absurd.spawn_task(p_queue_name text, p_task_name text, p_params jsonb, p_options jsonb DEFAULT '{}'::jsonb) RETURNS TABLE(task_id uuid, run_id uuid, attempt integer, created boolean)
    LANGUAGE plpgsql
    AS $_$
declare
  v_task_id uuid := absurd.portable_uuidv7();
  v_run_id uuid := absurd.portable_uuidv7();
  v_attempt integer := 1;
  v_headers jsonb;
  v_retry_strategy jsonb;
  v_max_attempts integer;
  v_cancellation jsonb;
  v_idempotency_key text;
  v_existing_task_id uuid;
  v_existing_run_id uuid;
  v_existing_attempt integer;
  v_row_count integer;
  v_storage_mode text := 'unpartitioned';
  v_task_inserted boolean := false;
  v_now timestamptz := absurd.current_time();
  v_params jsonb := coalesce(p_params, 'null'::jsonb);
begin
  if p_task_name is null or length(trim(p_task_name)) = 0 then
    raise exception 'task_name must be provided';
  end if;

  if p_options is not null then
    v_headers := p_options->'headers';
    v_retry_strategy := p_options->'retry_strategy';
    if p_options ? 'max_attempts' then
      v_max_attempts := (p_options->>'max_attempts')::int;
      if v_max_attempts is not null and v_max_attempts < 1 then
        raise exception 'max_attempts must be >= 1';
      end if;
    end if;
    v_cancellation := p_options->'cancellation';
    v_idempotency_key := p_options->>'idempotency_key';
  end if;

  if v_idempotency_key is not null then
    select storage_mode into v_storage_mode
    from absurd.queues
    where queue_name = p_queue_name;

    v_storage_mode := coalesce(v_storage_mode, 'unpartitioned');
    if v_storage_mode not in ('unpartitioned', 'partitioned') then
      raise exception 'Unsupported queue storage mode "%"', v_storage_mode;
    end if;

    if v_storage_mode = 'partitioned' then
      -- Reserve idempotency key via dedicated side table.
      execute format(
        'insert into absurd.%I (idempotency_key, task_id)
         values ($1, $2)
         on conflict (idempotency_key) do nothing',
        'i_' || p_queue_name
      )
      using v_idempotency_key, v_task_id;

      get diagnostics v_row_count = row_count;

      if v_row_count = 0 then
        execute format(
          'select i.task_id, t.last_attempt_run, t.attempts
             from absurd.%I i
             join absurd.%I t on t.task_id = i.task_id
            where i.idempotency_key = $1
              for key share of i',
          'i_' || p_queue_name,
          't_' || p_queue_name
        )
        into v_existing_task_id, v_existing_run_id, v_existing_attempt
        using v_idempotency_key;

        if v_existing_task_id is null then
          raise exception 'Idempotency key "%" in queue "%" was concurrently cleaned up', v_idempotency_key, p_queue_name
            using errcode = '40001',
                  hint = 'Retry spawn_task with the same idempotency key.';
        end if;

        if v_existing_run_id is null then
          raise exception 'Idempotency key "%" in queue "%" resolved to task "%" without a run', v_idempotency_key, p_queue_name, v_existing_task_id;
        end if;

        return query select v_existing_task_id, v_existing_run_id, v_existing_attempt, false;
        return;
      end if;
    else
      -- Unpartitioned queues keep the original unique(idempotency_key)
      -- behavior directly on t_<queue>.
      execute format(
        'insert into absurd.%I (task_id, task_name, params, headers, retry_strategy, max_attempts, cancellation, enqueue_at, first_started_at, state, attempts, last_attempt_run, completed_payload, cancelled_at, idempotency_key)
         values ($1, $2, $3, $4, $5, $6, $7, $8, null, ''pending'', $9, $10, null, null, $11)
         on conflict (idempotency_key) do nothing',
        't_' || p_queue_name
      )
      using v_task_id, p_task_name, v_params, v_headers, v_retry_strategy, v_max_attempts, v_cancellation, v_now, v_attempt, v_run_id, v_idempotency_key;

      get diagnostics v_row_count = row_count;

      if v_row_count = 0 then
        execute format(
          'select task_id, last_attempt_run, attempts
             from absurd.%I
            where idempotency_key = $1',
          't_' || p_queue_name
        )
        into v_existing_task_id, v_existing_run_id, v_existing_attempt
        using v_idempotency_key;

        return query select v_existing_task_id, v_existing_run_id, v_existing_attempt, false;
        return;
      end if;

      v_task_inserted := true;
    end if;
  end if;

  if not v_task_inserted then
    execute format(
      'insert into absurd.%I (task_id, task_name, params, headers, retry_strategy, max_attempts, cancellation, enqueue_at, first_started_at, state, attempts, last_attempt_run, completed_payload, cancelled_at, idempotency_key)
       values ($1, $2, $3, $4, $5, $6, $7, $8, null, ''pending'', $9, $10, null, null, $11)',
      't_' || p_queue_name
    )
    using v_task_id, p_task_name, v_params, v_headers, v_retry_strategy, v_max_attempts, v_cancellation, v_now, v_attempt, v_run_id, v_idempotency_key;
  end if;

  execute format(
    'insert into absurd.%I (run_id, task_id, attempt, state, available_at, wake_event, event_payload, result, failure_reason)
     values ($1, $2, $3, ''pending'', $4, null, null, null, null)',
    'r_' || p_queue_name
  )
  using v_run_id, v_task_id, v_attempt, v_now;

  return query select v_task_id, v_run_id, v_attempt, true;
end;
$_$;


--
-- Name: uuidv7_floor(timestamp with time zone); Type: FUNCTION; Schema: absurd; Owner: -
--

CREATE FUNCTION absurd.uuidv7_floor(p_ts timestamp with time zone) RETURNS uuid
    LANGUAGE plpgsql IMMUTABLE STRICT
    AS $$
declare
  ts_ms bigint := floor(extract(epoch from p_ts) * 1000)::bigint;
  b bytea;
  i int;
begin
  if ts_ms < 0 or ts_ms > 281474976710655 then
    raise exception 'Timestamp "%" is outside UUIDv7 supported range', p_ts;
  end if;

  b := repeat(E'\\000', 16)::bytea;
  for i in 0..5 loop
    b := set_byte(b, i, ((ts_ms >> ((5 - i) * 8)) & 255)::int);
  end loop;

  -- Set UUIDv7 version and RFC4122 variant; keep all randomness bits at 0.
  b := set_byte(b, 6, (7 << 4));
  b := set_byte(b, 8, 128);

  return encode(b, 'hex')::uuid;
end;
$$;


--
-- Name: uuidv7_timestamp(uuid); Type: FUNCTION; Schema: absurd; Owner: -
--

CREATE FUNCTION absurd.uuidv7_timestamp(p_id uuid) RETURNS timestamp with time zone
    LANGUAGE sql IMMUTABLE STRICT
    AS $$
  with bytes as (
    select uuid_send(p_id) as b
  ),
  decoded as (
    select
      (get_byte(b, 6) >> 4) as version,
      ((get_byte(b, 0)::bigint << 40) |
       (get_byte(b, 1)::bigint << 32) |
       (get_byte(b, 2)::bigint << 24) |
       (get_byte(b, 3)::bigint << 16) |
       (get_byte(b, 4)::bigint << 8)  |
        get_byte(b, 5)::bigint) as ts_ms
    from bytes
  )
  select case
           when version = 7 then 'epoch'::timestamptz + (ts_ms * interval '1 millisecond')
           else null
         end
  from decoded;
$$;


--
-- Name: validate_queue_name(text); Type: FUNCTION; Schema: absurd; Owner: -
--

CREATE FUNCTION absurd.validate_queue_name(p_queue_name text) RETURNS text
    LANGUAGE plpgsql
    AS $$
begin
  if p_queue_name is null or p_queue_name = '' then
    raise exception 'Queue name must be provided';
  end if;

  if octet_length(p_queue_name) > 57 then
    raise exception 'Queue name "%" is too long (max 57 bytes).', p_queue_name;
  end if;

  return p_queue_name;
end;
$$;


--
-- Name: week_bucket_utc(timestamp with time zone); Type: FUNCTION; Schema: absurd; Owner: -
--

CREATE FUNCTION absurd.week_bucket_utc(p_ts timestamp with time zone) RETURNS timestamp with time zone
    LANGUAGE sql IMMUTABLE STRICT
    AS $$
  select date_trunc('week', p_ts at time zone 'UTC') at time zone 'UTC';
$$;




--
-- Name: fnc__condition_match_code(text); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fnc__condition_match_code(value text) RETURNS text
    LANGUAGE sql IMMUTABLE
    AS $$
    SELECT CASE public.fnc__match_alias_key(value)
        WHEN 'good' THEN 'good'
        WHEN 'hyvä' THEN 'good'
        WHEN 'hyva' THEN 'good'
        WHEN 'satisfactory' THEN 'satisfactory'
        WHEN 'tyyd' THEN 'satisfactory'
        WHEN 'tyydyttävä' THEN 'satisfactory'
        WHEN 'tyydyttava' THEN 'satisfactory'
        WHEN 'tolerable' THEN 'poor'
        WHEN 'poor' THEN 'poor'
        WHEN 'bad' THEN 'poor'
        WHEN 'huono' THEN 'poor'
        WHEN 'välttävä' THEN 'poor'
        WHEN 'valttava' THEN 'poor'
        WHEN 'unclassified' THEN 'unknown'
        WHEN 'not_known' THEN 'unknown'
        WHEN 'not_shown' THEN 'unknown'
        ELSE NULL
    END
$$;


--
-- Name: fnc__energy_efficiency_match_code(text); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fnc__energy_efficiency_match_code(value text) RETURNS text
    LANGUAGE sql STABLE
    AS $$
    SELECT energy_efficiency_match_code FROM public.fnc__energy_efficiency_normalized(value)
$$;



--
-- Name: fnc__energy_efficiency_normalized(text); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fnc__energy_efficiency_normalized(value text) RETURNS TABLE(energy_efficiency_class_code text, energy_efficiency_standard_year integer, energy_efficiency_status text, energy_efficiency_match_code text)
    LANGUAGE sql STABLE
    AS $_$
    WITH normalized_input AS (
        SELECT public.fnc__match_alias_key(value) AS alias_key
    ),
    mapped AS (
        SELECT
            a.energy_efficiency_class_code,
            a.energy_efficiency_standard_year,
            a.energy_efficiency_status,
            a.energy_efficiency_match_code
        FROM public.energy_efficiency_aliases a
        JOIN normalized_input i ON i.alias_key = a.energy_efficiency_alias
    ),
    provider_code AS (
        SELECT regexp_match(alias_key, '^e([0-9]{2})_([a-h])$') AS parts
        FROM normalized_input
    ),
    label_code AS (
        SELECT regexp_match(alias_key, '(^|_)([a-h])_?((?:19|20|21)[0-9]{2})($|_)') AS parts
        FROM normalized_input
    ),
    energy_label_year AS (
        SELECT regexp_match(alias_key, '(^|_)([a-h])_energialuokkaan_((?:19|20|21)[0-9]{2})($|_)') AS parts
        FROM normalized_input
    ),
    energy_label_class AS (
        SELECT regexp_match(alias_key, '(^|_)energialuokka_([a-h])($|_)') AS parts
        FROM normalized_input
    ),
    leading_certificate_class AS (
        SELECT regexp_match(alias_key, '^([a-h])_energiatodistus') AS parts
        FROM normalized_input
    ),
    class_only AS (
        SELECT regexp_match(alias_key, '^([a-h])$') AS parts
        FROM normalized_input
    ),
    derived AS (
        SELECT
            CASE
                WHEN provider_code.parts IS NOT NULL THEN upper(provider_code.parts[2])
                WHEN label_code.parts IS NOT NULL THEN upper(label_code.parts[2])
                WHEN energy_label_year.parts IS NOT NULL THEN upper(energy_label_year.parts[2])
                WHEN energy_label_class.parts IS NOT NULL THEN upper(energy_label_class.parts[2])
                WHEN leading_certificate_class.parts IS NOT NULL THEN upper(leading_certificate_class.parts[1])
                WHEN class_only.parts IS NOT NULL THEN upper(class_only.parts[1])
                ELSE NULL
            END AS class_code,
            CASE
                WHEN provider_code.parts IS NULL THEN NULL
                WHEN provider_code.parts[1]::integer < 50 THEN 2000 + provider_code.parts[1]::integer
                ELSE 1900 + provider_code.parts[1]::integer
            END AS provider_year,
            CASE WHEN label_code.parts IS NOT NULL THEN label_code.parts[3]::integer ELSE NULL END AS label_year,
            CASE WHEN energy_label_year.parts IS NOT NULL THEN energy_label_year.parts[3]::integer ELSE NULL END AS energy_label_year,
            (provider_code.parts IS NOT NULL OR label_code.parts IS NOT NULL OR energy_label_year.parts IS NOT NULL OR energy_label_class.parts IS NOT NULL OR leading_certificate_class.parts IS NOT NULL OR class_only.parts IS NOT NULL) AS is_known
        FROM provider_code, label_code, energy_label_year, energy_label_class, leading_certificate_class, class_only
    )
    SELECT
        COALESCE(mapped.energy_efficiency_class_code, derived.class_code),
        COALESCE(mapped.energy_efficiency_standard_year, derived.provider_year, derived.label_year, derived.energy_label_year),
        COALESCE(mapped.energy_efficiency_status, CASE WHEN normalized_input.alias_key IS NULL THEN 'unknown' WHEN derived.is_known THEN 'known' ELSE 'unknown' END),
        COALESCE(mapped.energy_efficiency_match_code, CASE WHEN derived.class_code IS NULL THEN NULL WHEN COALESCE(derived.provider_year, derived.label_year, derived.energy_label_year) IS NULL THEN derived.class_code ELSE derived.class_code || COALESCE(derived.provider_year, derived.label_year, derived.energy_label_year)::text END)
    FROM normalized_input
    LEFT JOIN mapped ON true
    JOIN derived ON true
$_$;




--
-- Name: fnc__frontdoor_published_at(jsonb); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fnc__frontdoor_published_at(data jsonb) RETURNS timestamp with time zone
    LANGUAGE sql STABLE
    AS $$
    SELECT CASE
        WHEN public.fnc__try_parse_float8(data #>> '{publishingTime}') IS NULL THEN NULL
        ELSE to_timestamp(public.fnc__try_parse_float8(data #>> '{publishingTime}') / 1000.0)
    END
$$;


--
-- Name: fnc__jsonb_periodic_charge_price(jsonb, text); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fnc__jsonb_periodic_charge_price(payload jsonb, charge_key text) RETURNS double precision
    LANGUAGE sql IMMUTABLE
    AS $_$
SELECT public.fnc__try_parse_float8(
    jsonb_path_query_first(
        COALESCE(payload, '{}'::jsonb),
        '$.property.periodicCharges[*] ? (@.periodicCharge == $charge).price',
        jsonb_build_object('charge', to_jsonb(charge_key))
    ) #>> '{}'
);
$_$;


--
-- Name: fnc__layout_exact_match_key(text); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fnc__layout_exact_match_key(value text) RETURNS text
    LANGUAGE sql IMMUTABLE
    AS $$
    SELECT NULLIF(regexp_replace(lower(trim(COALESCE(value, ''))), '\s+', ' ', 'g'), '')
$$;




-- Name: fnc__match_alias_key(text); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fnc__match_alias_key(value text) RETURNS text
    LANGUAGE sql IMMUTABLE
    AS $$
    SELECT NULLIF(trim(BOTH '_' FROM regexp_replace(lower(trim(COALESCE(value, ''))), '[^[:alnum:]åäö]+', '_', 'g')), '')
$$;




--
-- Name: fnc__normalize_address_token(text); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fnc__normalize_address_token(value text) RETURNS text
    LANGUAGE sql IMMUTABLE
    AS $$
    SELECT NULLIF(trim(regexp_replace(lower(regexp_replace(trim(value), '[^[:alnum:]åäöÅÄÖ]+', ' ', 'g')), '\s+', ' ', 'g')), '')
$$;



--
-- Name: fnc__normalize_postal(text); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fnc__normalize_postal(value text) RETURNS text
    LANGUAGE sql IMMUTABLE
    AS $$
SELECT NULLIF(regexp_replace(trim(COALESCE(value, '')), '[^0-9]+', '', 'g'), '');
$$;




--
-- Name: fnc__prices_transaction_energy_match_code(text); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fnc__prices_transaction_energy_match_code(value text) RETURNS text
    LANGUAGE sql STABLE
    AS $$
    SELECT public.fnc__energy_efficiency_match_code(value)
$$;


--
-- Name: fnc__prices_transaction_floor_level(text); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fnc__prices_transaction_floor_level(value text) RETURNS integer
    LANGUAGE sql IMMUTABLE
    AS $_$
    SELECT NULLIF((regexp_match(COALESCE(value, ''), '^\s*(-?[0-9]+)(?:\s*/\s*-?[0-9]+)?\s*$'))[1], '')::integer
$_$;


--
-- Name: fnc__layout_match_key(text); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fnc__layout_match_key(value text) RETURNS text
    LANGUAGE sql IMMUTABLE
    AS $$
    SELECT NULLIF(regexp_replace(lower(trim(COALESCE(value, ''))), '[^[:alnum:]åäö]+', '', 'g'), '')
$$;



--
-- Name: fnc__prices_transaction_layout_is_truncated(text); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fnc__prices_transaction_layout_is_truncated(value text) RETURNS boolean
    LANGUAGE sql IMMUTABLE
    AS $$
    SELECT COALESCE(value, '') ~ '(\.\.\.|…)'
$$;


--
-- Name: fnc__prices_transaction_layout_match_key(text); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fnc__prices_transaction_layout_match_key(value text) RETURNS text
    LANGUAGE sql IMMUTABLE
    AS $_$
    SELECT public.fnc__layout_match_key(regexp_replace(COALESCE(value, ''), '(\.\.\.|…).*$', '', 'g'))
$_$;





--
-- Name: fnc__prices_transaction_total_floors(text); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fnc__prices_transaction_total_floors(value text) RETURNS integer
    LANGUAGE sql IMMUTABLE
    AS $_$
    SELECT NULLIF((regexp_match(COALESCE(value, ''), '^\s*-?[0-9]+\s*/\s*([0-9]+)\s*$'))[1], '')::integer
$_$;


--
-- Name: fnc__plot_owned(text); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fnc__plot_owned(value text) RETURNS boolean
    LANGUAGE sql STABLE
    AS $$
    SELECT CASE public.fnc__match_alias_key(value)
        WHEN '1' THEN true
        WHEN 'oma' THEN true
        WHEN 'own' THEN true
        WHEN 'owned' THEN true
        WHEN 'omistus' THEN true
        WHEN 'omistettu' THEN true
        WHEN '2' THEN false
        WHEN '3' THEN false
        WHEN 'vuokra' THEN false
        WHEN 'rent' THEN false
        WHEN 'rented' THEN false
        WHEN 'rental' THEN false
        WHEN 'lease' THEN false
        WHEN 'leased' THEN false
        WHEN 'vuokralla' THEN false
        WHEN 'vuokratontti' THEN false
        WHEN 'optional_rental' THEN false
        WHEN 'valinnainen_vuokratontti' THEN false
        ELSE NULL
    END
$$;






--
-- Name: fnc__refresh_housing_company_geom(uuid); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fnc__refresh_housing_company_geom(target_housing_company_id uuid DEFAULT NULL::uuid) RETURNS integer
    LANGUAGE plpgsql
    AS $$
DECLARE
    updated_count integer;
BEGIN
    WITH chosen AS (
        SELECT DISTINCT ON (hc.housing_company_id)
            hc.housing_company_id,
            COALESCE(
                sb.shortcut_building_geom,
                fb.frontdoor_building_geom,
                CASE
                    WHEN fa.frontdoor_ad_data #>> '{property,geoCode,longitude}' IS NOT NULL
                        AND fa.frontdoor_ad_data #>> '{property,geoCode,latitude}' IS NOT NULL
                    THEN postgis.ST_SetSRID(postgis.ST_MakePoint((fa.frontdoor_ad_data #>> '{property,geoCode,longitude}')::double precision, (fa.frontdoor_ad_data #>> '{property,geoCode,latitude}')::double precision), 4326)
                    ELSE NULL
                END
            ) AS geom
        FROM public.housing_companies hc
        JOIN public.property_units pu ON pu.housing_company_id = hc.housing_company_id
        JOIN public.property_offerings po ON po.property_unit_id = pu.property_unit_id
        JOIN public.property_offering_sources pos ON pos.property_offering_id = po.property_offering_id
            AND pos.property_offering_source_link_status <> 'rejected'
        JOIN public.property_source_offerings sl ON sl.sale_listing_id = pos.sale_listing_id
        LEFT JOIN public.shortcut_ads sa ON sa.shortcut_ad_id = sl.shortcut_ad_id
        LEFT JOIN public.shortcut_buildings sb ON sb.shortcut_building_id = sa.shortcut_building_id
        LEFT JOIN public.frontdoor_ads fa ON fa.frontdoor_ad_id = sl.frontdoor_ad_id
        LEFT JOIN public.frontdoor_building_announcements fba ON fba.frontdoor_building_announcement_id = sl.frontdoor_building_announcement_id
        LEFT JOIN public.frontdoor_buildings fb ON fb.frontdoor_building_id = fba.frontdoor_building_id
        WHERE (target_housing_company_id IS NULL OR hc.housing_company_id = target_housing_company_id)
            AND COALESCE(
                sb.shortcut_building_geom,
                fb.frontdoor_building_geom,
                CASE
                    WHEN fa.frontdoor_ad_data #>> '{property,geoCode,longitude}' IS NOT NULL
                        AND fa.frontdoor_ad_data #>> '{property,geoCode,latitude}' IS NOT NULL
                    THEN postgis.ST_SetSRID(postgis.ST_MakePoint((fa.frontdoor_ad_data #>> '{property,geoCode,longitude}')::double precision, (fa.frontdoor_ad_data #>> '{property,geoCode,latitude}')::double precision), 4326)
                    ELSE NULL
                END
            ) IS NOT NULL
        ORDER BY
            hc.housing_company_id,
            CASE WHEN sb.shortcut_building_geom IS NOT NULL THEN 0 WHEN fb.frontdoor_building_geom IS NOT NULL THEN 1 ELSE 2 END,
            sl.sale_listing_last_seen_at DESC NULLS LAST
    ),
    updated AS (
        UPDATE public.housing_companies hc
        SET housing_company_geom = chosen.geom,
            housing_company_updated_at = now()
        FROM chosen
        WHERE hc.housing_company_id = chosen.housing_company_id
            AND (hc.housing_company_geom IS NULL OR NOT postgis.ST_Equals(hc.housing_company_geom, chosen.geom))
        RETURNING hc.housing_company_id
    )
    SELECT count(*)::integer INTO updated_count FROM updated;
    RETURN updated_count;
END;
$$;



--
-- Name: fnc__sale_listing_floor_text(integer, integer); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fnc__sale_listing_floor_text(floor_level integer, total_floors integer) RETURNS text
    LANGUAGE sql IMMUTABLE
    AS $$
    SELECT CASE
        WHEN floor_level IS NULL THEN NULL
        WHEN total_floors IS NULL THEN floor_level::text
        ELSE floor_level::text || '/' || total_floors::text
    END
$$;


--
-- Name: fnc__sale_listing_plot_type_code(text); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fnc__sale_listing_plot_type_code(value text) RETURNS text
    LANGUAGE sql STABLE
    AS $$
    SELECT CASE
        WHEN public.fnc__plot_owned(value) IS TRUE THEN 'own'
        WHEN public.fnc__plot_owned(value) IS FALSE THEN 'rent'
        ELSE NULL
    END
$$;


--
-- Name: fnc__sale_listing_property_type_code(text); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fnc__sale_listing_property_type_code(value text) RETURNS text
    LANGUAGE sql STABLE
    AS $$
    SELECT a.sale_listing_property_type_code
    FROM public.sale_listing_property_type_aliases a
    WHERE a.sale_listing_property_type_alias = public.fnc__match_alias_key(value)
    LIMIT 1
$$;


--
-- Name: fnc__sale_listing_room_category_code(text); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fnc__sale_listing_room_category_code(value text) RETURNS text
    LANGUAGE sql STABLE
    AS $$
    SELECT a.sale_listing_room_category_code
    FROM public.sale_listing_room_category_aliases a
    WHERE a.sale_listing_room_category_alias = public.fnc__match_alias_key(value)
    LIMIT 1
$$;


--
-- Name: fnc__sale_listing_room_category_code(integer, text); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fnc__sale_listing_room_category_code(rooms integer, room_layout text) RETURNS text
    LANGUAGE sql STABLE
    AS $$
    SELECT CASE
        WHEN rooms = 1 THEN 'one_room'
        WHEN rooms = 2 THEN 'two_rooms'
        WHEN rooms = 3 THEN 'three_rooms'
        WHEN rooms >= 4 THEN 'four_plus_rooms'
        WHEN lower(COALESCE(room_layout, '')) ~ '(^|[^0-9])1\s*h' THEN 'one_room'
        WHEN lower(COALESCE(room_layout, '')) ~ '(^|[^0-9])2\s*h' THEN 'two_rooms'
        WHEN lower(COALESCE(room_layout, '')) ~ '(^|[^0-9])3\s*h' THEN 'three_rooms'
        WHEN lower(COALESCE(room_layout, '')) ~ '(^|[^0-9])[4-9]\s*h' THEN 'four_plus_rooms'
        ELSE NULL
    END
$$;


--
-- Name: fnc__shortcut_ad_street_address(jsonb); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fnc__shortcut_ad_street_address(data jsonb) RETURNS text
    LANGUAGE plpgsql IMMUTABLE
    AS $$
DECLARE
    street_name text;
    street_number text;
    building_letter text;
    formatted text;
BEGIN
    street_name := NULLIF(trim(COALESCE(data #>> '{address,street,name}', data #>> '{address,street}')), '');
    street_number := NULLIF(trim(data #>> '{address,streetNumber}'), '');
    building_letter := NULLIF(trim(data #>> '{address,buildingLetter}'), '');
    formatted := NULLIF(trim(data #>> '{address,formattedAddress}'), '');
    IF street_name IS NOT NULL AND street_number IS NOT NULL THEN
        RETURN concat_ws(' ', street_name, street_number, building_letter);
    END IF;
    RETURN COALESCE(formatted, street_name);
END;
$$;



--
-- Name: fnc__try_parse_bigint(text); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fnc__try_parse_bigint(value text) RETURNS bigint
    LANGUAGE sql IMMUTABLE
    AS $$
WITH cleaned AS (
    SELECT NULLIF(regexp_replace(replace(COALESCE(value, ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value
)
SELECT CASE
    WHEN value IS NULL THEN NULL
    WHEN length(value) - length(replace(value, '.', '')) > 1 THEN NULL
    ELSE (value::numeric)::int8
END
FROM cleaned;
$$;


--
-- Name: fnc__try_parse_bool(text); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fnc__try_parse_bool(value text) RETURNS boolean
    LANGUAGE sql IMMUTABLE
    AS $$
SELECT CASE
    WHEN value IS NULL THEN NULL
    WHEN lower(trim(value)) IN ('1', 'true', 'yes', 'on', 'kylla', 'kyllä', 'on') THEN true
    WHEN lower(trim(value)) IN ('0', 'false', 'no', 'off', 'ei') THEN false
    ELSE NULL
END;
$$;


--
-- Name: fnc__try_parse_float8(text); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fnc__try_parse_float8(value text) RETURNS double precision
    LANGUAGE sql IMMUTABLE
    AS $$
WITH cleaned AS (
    SELECT NULLIF(regexp_replace(replace(COALESCE(value, ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value
)
SELECT CASE
    WHEN value IS NULL THEN NULL
    WHEN length(value) - length(replace(value, '.', '')) > 1 THEN NULL
    ELSE value::float8
END
FROM cleaned;
$$;


--
-- Name: fnc__try_parse_int4(text); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fnc__try_parse_int4(value text) RETURNS integer
    LANGUAGE sql IMMUTABLE
    AS $$
SELECT CASE
    WHEN NULLIF(regexp_replace(replace(COALESCE(value, ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') IS NULL THEN NULL
    ELSE (NULLIF(regexp_replace(replace(COALESCE(value, ''), ',', '.'), '[^0-9\.-]', '', 'g'), '')::numeric)::int4
END;
$$;



SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: c_canonical_db; Type: TABLE; Schema: absurd; Owner: -
--

CREATE TABLE absurd.c_canonical_db (
    task_id uuid NOT NULL,
    checkpoint_name text NOT NULL,
    state jsonb,
    status text DEFAULT 'committed'::text NOT NULL,
    owner_run_id uuid,
    updated_at timestamp with time zone DEFAULT absurd."current_time"() NOT NULL
)
WITH (fillfactor='70');


--
-- Name: c_canonical_llm; Type: TABLE; Schema: absurd; Owner: -
--

CREATE TABLE absurd.c_canonical_llm (
    task_id uuid NOT NULL,
    checkpoint_name text NOT NULL,
    state jsonb,
    status text DEFAULT 'committed'::text NOT NULL,
    owner_run_id uuid,
    updated_at timestamp with time zone DEFAULT absurd."current_time"() NOT NULL
)
WITH (fillfactor='70');


--
-- Name: c_frontdoor; Type: TABLE; Schema: absurd; Owner: -
--

CREATE TABLE absurd.c_frontdoor (
    task_id uuid NOT NULL,
    checkpoint_name text NOT NULL,
    state jsonb,
    status text DEFAULT 'committed'::text NOT NULL,
    owner_run_id uuid,
    updated_at timestamp with time zone DEFAULT absurd."current_time"() NOT NULL
)
WITH (fillfactor='70');


--
-- Name: c_postal; Type: TABLE; Schema: absurd; Owner: -
--

CREATE TABLE absurd.c_postal (
    task_id uuid NOT NULL,
    checkpoint_name text NOT NULL,
    state jsonb,
    status text DEFAULT 'committed'::text NOT NULL,
    owner_run_id uuid,
    updated_at timestamp with time zone DEFAULT absurd."current_time"() NOT NULL
)
WITH (fillfactor='70');


--
-- Name: c_prices; Type: TABLE; Schema: absurd; Owner: -
--

CREATE TABLE absurd.c_prices (
    task_id uuid NOT NULL,
    checkpoint_name text NOT NULL,
    state jsonb,
    status text DEFAULT 'committed'::text NOT NULL,
    owner_run_id uuid,
    updated_at timestamp with time zone DEFAULT absurd."current_time"() NOT NULL
)
WITH (fillfactor='70');


--
-- Name: c_shortcut_api; Type: TABLE; Schema: absurd; Owner: -
--

CREATE TABLE absurd.c_shortcut_api (
    task_id uuid NOT NULL,
    checkpoint_name text NOT NULL,
    state jsonb,
    status text DEFAULT 'committed'::text NOT NULL,
    owner_run_id uuid,
    updated_at timestamp with time zone DEFAULT absurd."current_time"() NOT NULL
)
WITH (fillfactor='70');


--
-- Name: c_shortcut_scraper; Type: TABLE; Schema: absurd; Owner: -
--

CREATE TABLE absurd.c_shortcut_scraper (
    task_id uuid NOT NULL,
    checkpoint_name text NOT NULL,
    state jsonb,
    status text DEFAULT 'committed'::text NOT NULL,
    owner_run_id uuid,
    updated_at timestamp with time zone DEFAULT absurd."current_time"() NOT NULL
)
WITH (fillfactor='70');


--
-- Name: e_canonical_db; Type: TABLE; Schema: absurd; Owner: -
--

CREATE TABLE absurd.e_canonical_db (
    event_name text NOT NULL,
    payload jsonb,
    emitted_at timestamp with time zone DEFAULT absurd."current_time"() NOT NULL
);


--
-- Name: e_canonical_llm; Type: TABLE; Schema: absurd; Owner: -
--

CREATE TABLE absurd.e_canonical_llm (
    event_name text NOT NULL,
    payload jsonb,
    emitted_at timestamp with time zone DEFAULT absurd."current_time"() NOT NULL
);


--
-- Name: e_frontdoor; Type: TABLE; Schema: absurd; Owner: -
--

CREATE TABLE absurd.e_frontdoor (
    event_name text NOT NULL,
    payload jsonb,
    emitted_at timestamp with time zone DEFAULT absurd."current_time"() NOT NULL
);


--
-- Name: e_postal; Type: TABLE; Schema: absurd; Owner: -
--

CREATE TABLE absurd.e_postal (
    event_name text NOT NULL,
    payload jsonb,
    emitted_at timestamp with time zone DEFAULT absurd."current_time"() NOT NULL
);


--
-- Name: e_prices; Type: TABLE; Schema: absurd; Owner: -
--

CREATE TABLE absurd.e_prices (
    event_name text NOT NULL,
    payload jsonb,
    emitted_at timestamp with time zone DEFAULT absurd."current_time"() NOT NULL
);


--
-- Name: e_shortcut_api; Type: TABLE; Schema: absurd; Owner: -
--

CREATE TABLE absurd.e_shortcut_api (
    event_name text NOT NULL,
    payload jsonb,
    emitted_at timestamp with time zone DEFAULT absurd."current_time"() NOT NULL
);


--
-- Name: e_shortcut_scraper; Type: TABLE; Schema: absurd; Owner: -
--

CREATE TABLE absurd.e_shortcut_scraper (
    event_name text NOT NULL,
    payload jsonb,
    emitted_at timestamp with time zone DEFAULT absurd."current_time"() NOT NULL
);


--
-- Name: queues; Type: TABLE; Schema: absurd; Owner: -
--

CREATE TABLE absurd.queues (
    queue_name text NOT NULL,
    created_at timestamp with time zone DEFAULT absurd."current_time"() NOT NULL,
    storage_mode text DEFAULT 'unpartitioned'::text NOT NULL,
    default_partition text DEFAULT 'enabled'::text NOT NULL,
    partition_lookahead interval DEFAULT '28 days'::interval NOT NULL,
    partition_lookback interval DEFAULT '1 day'::interval NOT NULL,
    cleanup_ttl interval DEFAULT '30 days'::interval NOT NULL,
    cleanup_limit integer DEFAULT 1000 NOT NULL,
    detach_mode text DEFAULT 'none'::text NOT NULL,
    detach_min_age interval DEFAULT '30 days'::interval NOT NULL,
    CONSTRAINT queues_cleanup_limit_check CHECK ((cleanup_limit >= 1)),
    CONSTRAINT queues_cleanup_ttl_check CHECK ((cleanup_ttl >= '00:00:00'::interval)),
    CONSTRAINT queues_default_partition_check CHECK ((default_partition = ANY (ARRAY['enabled'::text, 'disabled'::text]))),
    CONSTRAINT queues_detach_min_age_check CHECK ((detach_min_age >= '00:00:00'::interval)),
    CONSTRAINT queues_detach_mode_check CHECK ((detach_mode = ANY (ARRAY['none'::text, 'empty'::text]))),
    CONSTRAINT queues_partition_lookahead_check CHECK ((partition_lookahead >= '00:00:00'::interval)),
    CONSTRAINT queues_partition_lookback_check CHECK ((partition_lookback >= '00:00:00'::interval)),
    CONSTRAINT queues_storage_mode_check CHECK ((storage_mode = ANY (ARRAY['unpartitioned'::text, 'partitioned'::text])))
);


--
-- Name: r_canonical_db; Type: TABLE; Schema: absurd; Owner: -
--

CREATE TABLE absurd.r_canonical_db (
    run_id uuid NOT NULL,
    task_id uuid NOT NULL,
    attempt integer NOT NULL,
    state text NOT NULL,
    claimed_by text,
    claim_expires_at timestamp with time zone,
    available_at timestamp with time zone NOT NULL,
    wake_event text,
    event_payload jsonb,
    started_at timestamp with time zone,
    completed_at timestamp with time zone,
    failed_at timestamp with time zone,
    result jsonb,
    failure_reason jsonb,
    created_at timestamp with time zone DEFAULT absurd."current_time"() NOT NULL,
    CONSTRAINT r_canonical_db_state_check CHECK ((state = ANY (ARRAY['pending'::text, 'running'::text, 'sleeping'::text, 'completed'::text, 'failed'::text, 'cancelled'::text])))
)
WITH (fillfactor='70');


--
-- Name: r_canonical_llm; Type: TABLE; Schema: absurd; Owner: -
--

CREATE TABLE absurd.r_canonical_llm (
    run_id uuid NOT NULL,
    task_id uuid NOT NULL,
    attempt integer NOT NULL,
    state text NOT NULL,
    claimed_by text,
    claim_expires_at timestamp with time zone,
    available_at timestamp with time zone NOT NULL,
    wake_event text,
    event_payload jsonb,
    started_at timestamp with time zone,
    completed_at timestamp with time zone,
    failed_at timestamp with time zone,
    result jsonb,
    failure_reason jsonb,
    created_at timestamp with time zone DEFAULT absurd."current_time"() NOT NULL,
    CONSTRAINT r_canonical_llm_state_check CHECK ((state = ANY (ARRAY['pending'::text, 'running'::text, 'sleeping'::text, 'completed'::text, 'failed'::text, 'cancelled'::text])))
)
WITH (fillfactor='70');


--
-- Name: r_frontdoor; Type: TABLE; Schema: absurd; Owner: -
--

CREATE TABLE absurd.r_frontdoor (
    run_id uuid NOT NULL,
    task_id uuid NOT NULL,
    attempt integer NOT NULL,
    state text NOT NULL,
    claimed_by text,
    claim_expires_at timestamp with time zone,
    available_at timestamp with time zone NOT NULL,
    wake_event text,
    event_payload jsonb,
    started_at timestamp with time zone,
    completed_at timestamp with time zone,
    failed_at timestamp with time zone,
    result jsonb,
    failure_reason jsonb,
    created_at timestamp with time zone DEFAULT absurd."current_time"() NOT NULL,
    CONSTRAINT r_frontdoor_state_check CHECK ((state = ANY (ARRAY['pending'::text, 'running'::text, 'sleeping'::text, 'completed'::text, 'failed'::text, 'cancelled'::text])))
)
WITH (fillfactor='70');


--
-- Name: r_postal; Type: TABLE; Schema: absurd; Owner: -
--

CREATE TABLE absurd.r_postal (
    run_id uuid NOT NULL,
    task_id uuid NOT NULL,
    attempt integer NOT NULL,
    state text NOT NULL,
    claimed_by text,
    claim_expires_at timestamp with time zone,
    available_at timestamp with time zone NOT NULL,
    wake_event text,
    event_payload jsonb,
    started_at timestamp with time zone,
    completed_at timestamp with time zone,
    failed_at timestamp with time zone,
    result jsonb,
    failure_reason jsonb,
    created_at timestamp with time zone DEFAULT absurd."current_time"() NOT NULL,
    CONSTRAINT r_postal_state_check CHECK ((state = ANY (ARRAY['pending'::text, 'running'::text, 'sleeping'::text, 'completed'::text, 'failed'::text, 'cancelled'::text])))
)
WITH (fillfactor='70');


--
-- Name: r_prices; Type: TABLE; Schema: absurd; Owner: -
--

CREATE TABLE absurd.r_prices (
    run_id uuid NOT NULL,
    task_id uuid NOT NULL,
    attempt integer NOT NULL,
    state text NOT NULL,
    claimed_by text,
    claim_expires_at timestamp with time zone,
    available_at timestamp with time zone NOT NULL,
    wake_event text,
    event_payload jsonb,
    started_at timestamp with time zone,
    completed_at timestamp with time zone,
    failed_at timestamp with time zone,
    result jsonb,
    failure_reason jsonb,
    created_at timestamp with time zone DEFAULT absurd."current_time"() NOT NULL,
    CONSTRAINT r_prices_state_check CHECK ((state = ANY (ARRAY['pending'::text, 'running'::text, 'sleeping'::text, 'completed'::text, 'failed'::text, 'cancelled'::text])))
)
WITH (fillfactor='70');


--
-- Name: r_shortcut_api; Type: TABLE; Schema: absurd; Owner: -
--

CREATE TABLE absurd.r_shortcut_api (
    run_id uuid NOT NULL,
    task_id uuid NOT NULL,
    attempt integer NOT NULL,
    state text NOT NULL,
    claimed_by text,
    claim_expires_at timestamp with time zone,
    available_at timestamp with time zone NOT NULL,
    wake_event text,
    event_payload jsonb,
    started_at timestamp with time zone,
    completed_at timestamp with time zone,
    failed_at timestamp with time zone,
    result jsonb,
    failure_reason jsonb,
    created_at timestamp with time zone DEFAULT absurd."current_time"() NOT NULL,
    CONSTRAINT r_shortcut_api_state_check CHECK ((state = ANY (ARRAY['pending'::text, 'running'::text, 'sleeping'::text, 'completed'::text, 'failed'::text, 'cancelled'::text])))
)
WITH (fillfactor='70');


--
-- Name: r_shortcut_scraper; Type: TABLE; Schema: absurd; Owner: -
--

CREATE TABLE absurd.r_shortcut_scraper (
    run_id uuid NOT NULL,
    task_id uuid NOT NULL,
    attempt integer NOT NULL,
    state text NOT NULL,
    claimed_by text,
    claim_expires_at timestamp with time zone,
    available_at timestamp with time zone NOT NULL,
    wake_event text,
    event_payload jsonb,
    started_at timestamp with time zone,
    completed_at timestamp with time zone,
    failed_at timestamp with time zone,
    result jsonb,
    failure_reason jsonb,
    created_at timestamp with time zone DEFAULT absurd."current_time"() NOT NULL,
    CONSTRAINT r_shortcut_scraper_state_check CHECK ((state = ANY (ARRAY['pending'::text, 'running'::text, 'sleeping'::text, 'completed'::text, 'failed'::text, 'cancelled'::text])))
)
WITH (fillfactor='70');


--
-- Name: t_canonical_db; Type: TABLE; Schema: absurd; Owner: -
--

CREATE TABLE absurd.t_canonical_db (
    task_id uuid NOT NULL,
    task_name text NOT NULL,
    params jsonb NOT NULL,
    headers jsonb,
    retry_strategy jsonb,
    max_attempts integer,
    cancellation jsonb,
    enqueue_at timestamp with time zone DEFAULT absurd."current_time"() NOT NULL,
    first_started_at timestamp with time zone,
    state text NOT NULL,
    attempts integer DEFAULT 0 NOT NULL,
    last_attempt_run uuid,
    completed_payload jsonb,
    cancelled_at timestamp with time zone,
    idempotency_key text,
    CONSTRAINT t_canonical_db_state_check CHECK ((state = ANY (ARRAY['pending'::text, 'running'::text, 'sleeping'::text, 'completed'::text, 'failed'::text, 'cancelled'::text])))
)
WITH (fillfactor='70');


--
-- Name: t_canonical_llm; Type: TABLE; Schema: absurd; Owner: -
--

CREATE TABLE absurd.t_canonical_llm (
    task_id uuid NOT NULL,
    task_name text NOT NULL,
    params jsonb NOT NULL,
    headers jsonb,
    retry_strategy jsonb,
    max_attempts integer,
    cancellation jsonb,
    enqueue_at timestamp with time zone DEFAULT absurd."current_time"() NOT NULL,
    first_started_at timestamp with time zone,
    state text NOT NULL,
    attempts integer DEFAULT 0 NOT NULL,
    last_attempt_run uuid,
    completed_payload jsonb,
    cancelled_at timestamp with time zone,
    idempotency_key text,
    CONSTRAINT t_canonical_llm_state_check CHECK ((state = ANY (ARRAY['pending'::text, 'running'::text, 'sleeping'::text, 'completed'::text, 'failed'::text, 'cancelled'::text])))
)
WITH (fillfactor='70');


--
-- Name: t_frontdoor; Type: TABLE; Schema: absurd; Owner: -
--

CREATE TABLE absurd.t_frontdoor (
    task_id uuid NOT NULL,
    task_name text NOT NULL,
    params jsonb NOT NULL,
    headers jsonb,
    retry_strategy jsonb,
    max_attempts integer,
    cancellation jsonb,
    enqueue_at timestamp with time zone DEFAULT absurd."current_time"() NOT NULL,
    first_started_at timestamp with time zone,
    state text NOT NULL,
    attempts integer DEFAULT 0 NOT NULL,
    last_attempt_run uuid,
    completed_payload jsonb,
    cancelled_at timestamp with time zone,
    idempotency_key text,
    CONSTRAINT t_frontdoor_state_check CHECK ((state = ANY (ARRAY['pending'::text, 'running'::text, 'sleeping'::text, 'completed'::text, 'failed'::text, 'cancelled'::text])))
)
WITH (fillfactor='70');


--
-- Name: t_postal; Type: TABLE; Schema: absurd; Owner: -
--

CREATE TABLE absurd.t_postal (
    task_id uuid NOT NULL,
    task_name text NOT NULL,
    params jsonb NOT NULL,
    headers jsonb,
    retry_strategy jsonb,
    max_attempts integer,
    cancellation jsonb,
    enqueue_at timestamp with time zone DEFAULT absurd."current_time"() NOT NULL,
    first_started_at timestamp with time zone,
    state text NOT NULL,
    attempts integer DEFAULT 0 NOT NULL,
    last_attempt_run uuid,
    completed_payload jsonb,
    cancelled_at timestamp with time zone,
    idempotency_key text,
    CONSTRAINT t_postal_state_check CHECK ((state = ANY (ARRAY['pending'::text, 'running'::text, 'sleeping'::text, 'completed'::text, 'failed'::text, 'cancelled'::text])))
)
WITH (fillfactor='70');


--
-- Name: t_prices; Type: TABLE; Schema: absurd; Owner: -
--

CREATE TABLE absurd.t_prices (
    task_id uuid NOT NULL,
    task_name text NOT NULL,
    params jsonb NOT NULL,
    headers jsonb,
    retry_strategy jsonb,
    max_attempts integer,
    cancellation jsonb,
    enqueue_at timestamp with time zone DEFAULT absurd."current_time"() NOT NULL,
    first_started_at timestamp with time zone,
    state text NOT NULL,
    attempts integer DEFAULT 0 NOT NULL,
    last_attempt_run uuid,
    completed_payload jsonb,
    cancelled_at timestamp with time zone,
    idempotency_key text,
    CONSTRAINT t_prices_state_check CHECK ((state = ANY (ARRAY['pending'::text, 'running'::text, 'sleeping'::text, 'completed'::text, 'failed'::text, 'cancelled'::text])))
)
WITH (fillfactor='70');


--
-- Name: t_shortcut_api; Type: TABLE; Schema: absurd; Owner: -
--

CREATE TABLE absurd.t_shortcut_api (
    task_id uuid NOT NULL,
    task_name text NOT NULL,
    params jsonb NOT NULL,
    headers jsonb,
    retry_strategy jsonb,
    max_attempts integer,
    cancellation jsonb,
    enqueue_at timestamp with time zone DEFAULT absurd."current_time"() NOT NULL,
    first_started_at timestamp with time zone,
    state text NOT NULL,
    attempts integer DEFAULT 0 NOT NULL,
    last_attempt_run uuid,
    completed_payload jsonb,
    cancelled_at timestamp with time zone,
    idempotency_key text,
    CONSTRAINT t_shortcut_api_state_check CHECK ((state = ANY (ARRAY['pending'::text, 'running'::text, 'sleeping'::text, 'completed'::text, 'failed'::text, 'cancelled'::text])))
)
WITH (fillfactor='70');


--
-- Name: t_shortcut_scraper; Type: TABLE; Schema: absurd; Owner: -
--

CREATE TABLE absurd.t_shortcut_scraper (
    task_id uuid NOT NULL,
    task_name text NOT NULL,
    params jsonb NOT NULL,
    headers jsonb,
    retry_strategy jsonb,
    max_attempts integer,
    cancellation jsonb,
    enqueue_at timestamp with time zone DEFAULT absurd."current_time"() NOT NULL,
    first_started_at timestamp with time zone,
    state text NOT NULL,
    attempts integer DEFAULT 0 NOT NULL,
    last_attempt_run uuid,
    completed_payload jsonb,
    cancelled_at timestamp with time zone,
    idempotency_key text,
    CONSTRAINT t_shortcut_scraper_state_check CHECK ((state = ANY (ARRAY['pending'::text, 'running'::text, 'sleeping'::text, 'completed'::text, 'failed'::text, 'cancelled'::text])))
)
WITH (fillfactor='70');


--
-- Name: w_canonical_db; Type: TABLE; Schema: absurd; Owner: -
--

CREATE TABLE absurd.w_canonical_db (
    task_id uuid NOT NULL,
    run_id uuid NOT NULL,
    step_name text NOT NULL,
    event_name text NOT NULL,
    timeout_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT absurd."current_time"() NOT NULL
);


--
-- Name: w_canonical_llm; Type: TABLE; Schema: absurd; Owner: -
--

CREATE TABLE absurd.w_canonical_llm (
    task_id uuid NOT NULL,
    run_id uuid NOT NULL,
    step_name text NOT NULL,
    event_name text NOT NULL,
    timeout_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT absurd."current_time"() NOT NULL
);


--
-- Name: w_frontdoor; Type: TABLE; Schema: absurd; Owner: -
--

CREATE TABLE absurd.w_frontdoor (
    task_id uuid NOT NULL,
    run_id uuid NOT NULL,
    step_name text NOT NULL,
    event_name text NOT NULL,
    timeout_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT absurd."current_time"() NOT NULL
);


--
-- Name: w_postal; Type: TABLE; Schema: absurd; Owner: -
--

CREATE TABLE absurd.w_postal (
    task_id uuid NOT NULL,
    run_id uuid NOT NULL,
    step_name text NOT NULL,
    event_name text NOT NULL,
    timeout_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT absurd."current_time"() NOT NULL
);


--
-- Name: w_prices; Type: TABLE; Schema: absurd; Owner: -
--

CREATE TABLE absurd.w_prices (
    task_id uuid NOT NULL,
    run_id uuid NOT NULL,
    step_name text NOT NULL,
    event_name text NOT NULL,
    timeout_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT absurd."current_time"() NOT NULL
);


--
-- Name: w_shortcut_api; Type: TABLE; Schema: absurd; Owner: -
--

CREATE TABLE absurd.w_shortcut_api (
    task_id uuid NOT NULL,
    run_id uuid NOT NULL,
    step_name text NOT NULL,
    event_name text NOT NULL,
    timeout_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT absurd."current_time"() NOT NULL
);


--
-- Name: w_shortcut_scraper; Type: TABLE; Schema: absurd; Owner: -
--

CREATE TABLE absurd.w_shortcut_scraper (
    task_id uuid NOT NULL,
    run_id uuid NOT NULL,
    step_name text NOT NULL,
    event_name text NOT NULL,
    timeout_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT absurd."current_time"() NOT NULL
);


--
-- Name: devices; Type: TABLE; Schema: auth; Owner: -
--

CREATE TABLE auth.devices (
    device_id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    user_id uuid NOT NULL,
    device_name text,
    device_os text,
    device_app_version text,
    device_push_token text,
    device_push_token_type auth.push_token_type,
    device_push_token_updated_at timestamp with time zone,
    device_created_at timestamp with time zone DEFAULT now() NOT NULL,
    device_updated_at timestamp with time zone DEFAULT now() NOT NULL,
    device_last_seen_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: feature_flags; Type: TABLE; Schema: auth; Owner: -
--

CREATE TABLE auth.feature_flags (
    flag_id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    flag_name text NOT NULL,
    flag_description text,
    flag_default_enabled boolean DEFAULT false NOT NULL,
    flag_created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: identities; Type: TABLE; Schema: auth; Owner: -
--

CREATE TABLE auth.identities (
    identity_id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    user_id uuid NOT NULL,
    identity_provider auth.auth_provider NOT NULL,
    identity_external_id text NOT NULL,
    identity_email text,
    identity_email_verified boolean DEFAULT false,
    identity_data jsonb DEFAULT '{}'::jsonb,
    identity_created_at timestamp with time zone DEFAULT now() NOT NULL,
    identity_updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: refresh_tokens; Type: TABLE; Schema: auth; Owner: -
--

CREATE TABLE auth.refresh_tokens (
    refresh_token_id bigint NOT NULL,
    session_id uuid NOT NULL,
    refresh_token_token_hash text NOT NULL,
    refresh_token_counter bigint NOT NULL,
    refresh_token_revoked boolean DEFAULT false NOT NULL,
    refresh_token_created_at timestamp with time zone DEFAULT now() NOT NULL,
    refresh_token_updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: refresh_tokens_refresh_token_id_seq; Type: SEQUENCE; Schema: auth; Owner: -
--

CREATE SEQUENCE auth.refresh_tokens_refresh_token_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: refresh_tokens_refresh_token_id_seq; Type: SEQUENCE OWNED BY; Schema: auth; Owner: -
--

ALTER SEQUENCE auth.refresh_tokens_refresh_token_id_seq OWNED BY auth.refresh_tokens.refresh_token_id;


--
-- Name: role_feature_flags; Type: TABLE; Schema: auth; Owner: -
--

CREATE TABLE auth.role_feature_flags (
    role_id uuid NOT NULL,
    flag_id uuid NOT NULL
);


--
-- Name: roles; Type: TABLE; Schema: auth; Owner: -
--

CREATE TABLE auth.roles (
    role_id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    role_name text NOT NULL,
    role_description text,
    role_created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: sessions; Type: TABLE; Schema: auth; Owner: -
--

CREATE TABLE auth.sessions (
    session_id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    user_id uuid NOT NULL,
    session_device_id uuid,
    session_user_agent text,
    session_ip inet,
    session_provider auth.auth_provider NOT NULL,
    session_refresh_token_hmac_key text NOT NULL,
    session_refresh_token_counter bigint DEFAULT 0 NOT NULL,
    session_created_at timestamp with time zone DEFAULT now() NOT NULL,
    session_updated_at timestamp with time zone DEFAULT now() NOT NULL,
    session_refreshed_at timestamp with time zone,
    session_not_after timestamp with time zone,
    session_revoked_at timestamp with time zone
);


--
-- Name: user_feature_flags; Type: TABLE; Schema: auth; Owner: -
--

CREATE TABLE auth.user_feature_flags (
    user_id uuid NOT NULL,
    flag_id uuid NOT NULL,
    user_flag_enabled boolean NOT NULL,
    user_flag_created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: user_roles; Type: TABLE; Schema: auth; Owner: -
--

CREATE TABLE auth.user_roles (
    user_id uuid NOT NULL,
    role_id uuid NOT NULL,
    user_role_created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: users; Type: TABLE; Schema: auth; Owner: -
--

CREATE TABLE auth.users (
    user_id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    user_created_at timestamp with time zone DEFAULT now() NOT NULL,
    user_updated_at timestamp with time zone DEFAULT now() NOT NULL,
    user_deleted_at timestamp with time zone
);


--
-- Name: auth_signup_email_tokens; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.auth_signup_email_tokens (
    auth_signup_email_token_id bigint NOT NULL,
    auth_signup_email_token_uuid uuid DEFAULT gen_random_uuid() NOT NULL,
    auth_signup_email_target_email text CONSTRAINT auth_signup_email_tokens_auth_signup_email_target_emai_not_null NOT NULL,
    auth_signup_email_token_hash text NOT NULL,
    auth_signup_email_expires_at timestamp with time zone NOT NULL,
    auth_signup_email_consumed_at timestamp with time zone,
    auth_signup_email_created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT auth_signup_email_tokens_target_email_not_blank CHECK ((btrim(auth_signup_email_target_email) <> ''::text))
);


--
-- Name: auth_signup_email_tokens_auth_signup_email_token_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.auth_signup_email_tokens ALTER COLUMN auth_signup_email_token_id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.auth_signup_email_tokens_auth_signup_email_token_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: auth_signup_tickets; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.auth_signup_tickets (
    auth_signup_ticket_id bigint NOT NULL,
    auth_signup_ticket_uuid uuid DEFAULT gen_random_uuid() NOT NULL,
    auth_signup_ticket_target_email text NOT NULL,
    auth_signup_ticket_hash text NOT NULL,
    auth_signup_ticket_expires_at timestamp with time zone NOT NULL,
    auth_signup_ticket_consumed_at timestamp with time zone,
    auth_signup_ticket_created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT auth_signup_tickets_target_email_not_blank CHECK ((btrim(auth_signup_ticket_target_email) <> ''::text))
);


--
-- Name: auth_signup_tickets_auth_signup_ticket_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.auth_signup_tickets ALTER COLUMN auth_signup_ticket_id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.auth_signup_tickets_auth_signup_ticket_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: auth_webauthn_challenges; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.auth_webauthn_challenges (
    auth_webauthn_challenge_id bigint NOT NULL,
    auth_webauthn_challenge_uuid uuid DEFAULT gen_random_uuid() NOT NULL,
    auth_webauthn_challenge_flow text NOT NULL,
    auth_webauthn_challenge_session jsonb CONSTRAINT auth_webauthn_challenges_auth_webauthn_challenge_sessi_not_null NOT NULL,
    auth_webauthn_challenge_expires_at timestamp with time zone CONSTRAINT auth_webauthn_challenges_auth_webauthn_challenge_expir_not_null NOT NULL,
    auth_webauthn_challenge_user_handle bytea,
    auth_webauthn_challenge_user_display_name text,
    auth_webauthn_challenge_device_id uuid,
    auth_webauthn_challenge_consumed_at timestamp with time zone,
    auth_webauthn_challenge_created_at timestamp with time zone DEFAULT now() CONSTRAINT auth_webauthn_challenges_auth_webauthn_challenge_creat_not_null NOT NULL,
    auth_webauthn_challenge_verified_email text,
    user_id bigint,
    CONSTRAINT auth_webauthn_challenge_flow_check CHECK ((auth_webauthn_challenge_flow = ANY (ARRAY['authenticate'::text, 'register'::text, 'signup'::text])))
);


--
-- Name: auth_webauthn_challenges_auth_webauthn_challenge_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.auth_webauthn_challenges ALTER COLUMN auth_webauthn_challenge_id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.auth_webauthn_challenges_auth_webauthn_challenge_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: device_sessions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.device_sessions (
    device_session_uuid uuid DEFAULT gen_random_uuid() NOT NULL,
    device_session_user_agent text,
    device_session_ip inet,
    device_session_created_at timestamp with time zone DEFAULT now() NOT NULL,
    device_session_updated_at timestamp with time zone DEFAULT now() NOT NULL,
    device_session_refreshed_at timestamp with time zone,
    device_session_not_after timestamp with time zone,
    device_session_revoked_at timestamp with time zone,
    device_session_provider text NOT NULL,
    device_session_id bigint NOT NULL,
    device_session_user_device_id bigint NOT NULL,
    user_id bigint NOT NULL,
    device_session_device_name text,
    device_session_device_os text,
    device_session_device_model text,
    device_session_app_version text,
    device_session_locale text,
    device_session_time_zone text,
    device_session_location_city text,
    device_session_location_region text,
    device_session_location_country_code text,
    device_session_location_source text,
    CONSTRAINT device_session_provider_check CHECK ((device_session_provider = ANY (ARRAY['apple'::text, 'anonymous'::text, 'email'::text, 'google'::text, 'passkey'::text])))
);


--
-- Name: device_sessions_device_session_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.device_sessions ALTER COLUMN device_session_id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.device_sessions_device_session_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: dimension_claims; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.dimension_claims (
    property_dimension_claim_id uuid DEFAULT gen_random_uuid() CONSTRAINT property_dimension_claims_property_dimension_claim_id_not_null NOT NULL,
    property_dimension_projection_run_id uuid CONSTRAINT property_dimension_claims_property_dimension_projectio_not_null NOT NULL,
    projection_version text CONSTRAINT property_dimension_claims_projection_version_not_null NOT NULL,
    claim_scope text CONSTRAINT property_dimension_claims_claim_scope_not_null NOT NULL,
    target_type text CONSTRAINT property_dimension_claims_target_type_not_null NOT NULL,
    target_id uuid CONSTRAINT property_dimension_claims_target_id_not_null NOT NULL,
    dimension_key text CONSTRAINT property_dimension_claims_dimension_key_not_null NOT NULL,
    value jsonb CONSTRAINT property_dimension_claims_value_not_null NOT NULL,
    value_kind text CONSTRAINT property_dimension_claims_value_kind_not_null NOT NULL,
    unit text,
    source_table text CONSTRAINT property_dimension_claims_source_table_not_null NOT NULL,
    source_id uuid CONSTRAINT property_dimension_claims_source_id_not_null NOT NULL,
    source_field text,
    source_claim_id uuid,
    source_observed_at timestamp with time zone,
    valid_from date,
    valid_until date,
    confidence double precision DEFAULT 0.5 CONSTRAINT property_dimension_claims_confidence_not_null NOT NULL,
    source_reliability double precision DEFAULT 0.5 CONSTRAINT property_dimension_claims_source_reliability_not_null NOT NULL,
    evidence jsonb DEFAULT '{}'::jsonb CONSTRAINT property_dimension_claims_evidence_not_null NOT NULL,
    extraction_model text,
    extraction_prompt_version text,
    created_at timestamp with time zone DEFAULT now() CONSTRAINT property_dimension_claims_created_at_not_null NOT NULL,
    updated_at timestamp with time zone DEFAULT now() CONSTRAINT property_dimension_claims_updated_at_not_null NOT NULL,
    CONSTRAINT dimension_claims_claim_scope_check CHECK ((claim_scope = ANY (ARRAY['source'::text, 'manual'::text]))),
    CONSTRAINT dimension_claims_confidence_check CHECK (((confidence >= (0)::double precision) AND (confidence <= (1)::double precision))),
    CONSTRAINT dimension_claims_source_reliability_check CHECK (((source_reliability >= (0)::double precision) AND (source_reliability <= (1)::double precision))),
    CONSTRAINT dimension_claims_target_type_check CHECK ((target_type = ANY (ARRAY['listing'::text, 'document'::text, 'offering'::text, 'unit'::text, 'building'::text, 'housing_company'::text, 'house'::text]))),
    CONSTRAINT dimension_claims_value_kind_check CHECK ((value_kind = ANY (ARRAY['string'::text, 'number'::text, 'boolean'::text, 'object'::text, 'array'::text, 'null'::text])))
);


--
-- Name: dimension_profiles; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.dimension_profiles (
    target_type text CONSTRAINT property_dimension_profiles_target_type_not_null NOT NULL,
    target_id uuid CONSTRAINT property_dimension_profiles_target_id_not_null NOT NULL,
    dimensions jsonb DEFAULT '{}'::jsonb CONSTRAINT property_dimension_profiles_dimensions_not_null NOT NULL,
    metadata jsonb DEFAULT '{}'::jsonb CONSTRAINT property_dimension_profiles_metadata_not_null NOT NULL,
    conflicts jsonb DEFAULT '{}'::jsonb CONSTRAINT property_dimension_profiles_conflicts_not_null NOT NULL,
    resolved_at timestamp with time zone DEFAULT now() CONSTRAINT property_dimension_profiles_resolved_at_not_null NOT NULL,
    CONSTRAINT dimension_profiles_target_type_check CHECK ((target_type = ANY (ARRAY['offering'::text, 'unit'::text, 'building'::text, 'housing_company'::text, 'house'::text])))
);


--
-- Name: dimension_values; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.dimension_values (
    target_type text CONSTRAINT property_dimension_values_target_type_not_null NOT NULL,
    target_id uuid CONSTRAINT property_dimension_values_target_id_not_null NOT NULL,
    dimension_key text CONSTRAINT property_dimension_values_dimension_key_not_null NOT NULL,
    value jsonb CONSTRAINT property_dimension_values_value_not_null NOT NULL,
    value_kind text CONSTRAINT property_dimension_values_value_kind_not_null NOT NULL,
    unit text,
    confidence double precision CONSTRAINT property_dimension_values_confidence_not_null NOT NULL,
    selected_claim_id uuid,
    selected_reason text CONSTRAINT property_dimension_values_selected_reason_not_null NOT NULL,
    conflict_status text DEFAULT 'none'::text CONSTRAINT property_dimension_values_conflict_status_not_null NOT NULL,
    supporting_claim_ids uuid[] DEFAULT '{}'::uuid[] CONSTRAINT property_dimension_values_supporting_claim_ids_not_null NOT NULL,
    rejected_claim_ids uuid[] DEFAULT '{}'::uuid[] CONSTRAINT property_dimension_values_rejected_claim_ids_not_null NOT NULL,
    resolved_at timestamp with time zone DEFAULT now() CONSTRAINT property_dimension_values_resolved_at_not_null NOT NULL,
    CONSTRAINT dimension_values_confidence_check CHECK (((confidence >= (0)::double precision) AND (confidence <= (1)::double precision))),
    CONSTRAINT dimension_values_conflict_status_check CHECK ((conflict_status = ANY (ARRAY['none'::text, 'compatible'::text, 'conflicting'::text, 'manual_override'::text]))),
    CONSTRAINT dimension_values_target_type_check CHECK ((target_type = ANY (ARRAY['offering'::text, 'unit'::text, 'building'::text, 'housing_company'::text, 'house'::text]))),
    CONSTRAINT dimension_values_value_kind_check CHECK ((value_kind = ANY (ARRAY['string'::text, 'number'::text, 'boolean'::text, 'object'::text, 'array'::text, 'null'::text])))
);


--
-- Name: energy_efficiency_aliases; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.energy_efficiency_aliases (
    energy_efficiency_alias text NOT NULL,
    energy_efficiency_class_code text,
    energy_efficiency_standard_year integer,
    energy_efficiency_status text NOT NULL,
    energy_efficiency_match_code text,
    energy_efficiency_label text NOT NULL,
    CONSTRAINT energy_efficiency_aliases_status_check CHECK ((energy_efficiency_status = ANY (ARRAY['known'::text, 'not_required'::text, 'not_available'::text, 'unknown'::text])))
);


--
-- Name: feature_flags; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.feature_flags (
    flag_uuid uuid DEFAULT gen_random_uuid() NOT NULL,
    flag_name text NOT NULL,
    flag_description text,
    flag_default_enabled boolean DEFAULT false NOT NULL,
    flag_created_at timestamp with time zone DEFAULT now() NOT NULL,
    flag_id bigint NOT NULL
);


--
-- Name: feature_flags_flag_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.feature_flags ALTER COLUMN flag_id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.feature_flags_flag_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: frontdoor_ads; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.frontdoor_ads (
    frontdoor_ad_id uuid DEFAULT gen_random_uuid() NOT NULL,
    frontdoor_ad_external_id text NOT NULL,
    frontdoor_ad_url text NOT NULL,
    frontdoor_ad_first_seen_at timestamp with time zone DEFAULT now() NOT NULL,
    frontdoor_ad_last_seen_at timestamp with time zone DEFAULT now() NOT NULL,
    frontdoor_ad_updated_at timestamp with time zone DEFAULT now() NOT NULL,
    frontdoor_ad_data jsonb,
    frontdoor_ad_processed_at timestamp with time zone,
    frontdoor_ad_page_not_found boolean DEFAULT false NOT NULL,
    frontdoor_ad_data_hash text,
    frontdoor_ad_data_hash_algorithm text DEFAULT 'sha256'::text NOT NULL,
    frontdoor_ad_data_changed_at timestamp with time zone,
    frontdoor_ad_data_normalized_at timestamp with time zone,
    frontdoor_ad_data_normalized_version integer DEFAULT 0 NOT NULL
);


--
-- Name: frontdoor_building_announcements; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.frontdoor_building_announcements (
    frontdoor_building_announcement_id uuid DEFAULT gen_random_uuid() CONSTRAINT frontdoor_building_announce_frontdoor_building_announc_not_null NOT NULL,
    frontdoor_building_announcement_external_id integer,
    frontdoor_building_announcement_friendly_id text,
    frontdoor_building_announcement_unpublishing_time double precision,
    frontdoor_building_announcement_address_line1 text,
    frontdoor_building_announcement_address_line2 text,
    frontdoor_building_announcement_location text,
    frontdoor_building_announcement_search_price double precision,
    frontdoor_building_announcement_notify_price_changed boolean,
    frontdoor_building_announcement_property_type text,
    frontdoor_building_announcement_property_subtype text,
    frontdoor_building_announcement_construction_finished_year integer,
    frontdoor_building_announcement_main_image_uri text,
    frontdoor_building_announcement_has_open_bidding boolean,
    frontdoor_building_announcement_room_structure text,
    frontdoor_building_announcement_area double precision,
    frontdoor_building_announcement_total_area double precision,
    frontdoor_building_announcement_price_per_square double precision,
    frontdoor_building_announcement_days_on_market integer,
    frontdoor_building_announcement_new_building boolean,
    frontdoor_building_announcement_main_image_hidden boolean,
    frontdoor_building_announcement_is_company_announcement boolean,
    frontdoor_building_announcement_show_bidding_indicators boolean,
    frontdoor_building_announcement_published boolean,
    frontdoor_building_announcement_rent_period text,
    frontdoor_building_announcement_rental_unique_no integer,
    frontdoor_building_id uuid NOT NULL,
    frontdoor_building_announcement_first_seen_at timestamp with time zone DEFAULT now() CONSTRAINT frontdoor_building_announc_frontdoor_building_announc_not_null1 NOT NULL,
    frontdoor_building_announcement_last_seen_at timestamp with time zone DEFAULT now() CONSTRAINT frontdoor_building_announc_frontdoor_building_announc_not_null2 NOT NULL,
    frontdoor_building_announcement_unpublishing_time_date date,
    frontdoor_building_announcement_data_normalized_at timestamp with time zone,
    frontdoor_building_announcement_data_normalized_version integer DEFAULT 0 CONSTRAINT frontdoor_building_announc_frontdoor_building_announc_not_null3 NOT NULL
);


--
-- Name: frontdoor_buildings; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.frontdoor_buildings (
    frontdoor_building_id uuid DEFAULT gen_random_uuid() NOT NULL,
    frontdoor_building_url text,
    frontdoor_building_first_seen_at timestamp with time zone DEFAULT now() NOT NULL,
    frontdoor_building_last_seen_at timestamp with time zone DEFAULT now() NOT NULL,
    frontdoor_building_updated_at timestamp with time zone DEFAULT now() NOT NULL,
    frontdoor_building_company_name text,
    frontdoor_building_business_id text,
    frontdoor_building_apartment_count integer,
    frontdoor_building_floor_count integer,
    frontdoor_building_construction_end_year integer,
    frontdoor_building_build_year integer,
    frontdoor_building_has_elevator boolean,
    frontdoor_building_has_sauna boolean,
    frontdoor_building_energy_certificate_code text,
    frontdoor_building_plot_holding_type text,
    frontdoor_building_outer_roof_material text,
    frontdoor_building_outer_roof_type text,
    frontdoor_building_heating text,
    frontdoor_building_heating_fuel text[],
    frontdoor_building_street_address text,
    frontdoor_building_house_number text,
    frontdoor_building_postcode text,
    frontdoor_building_post_area text,
    frontdoor_building_municipality text,
    frontdoor_building_district text,
    frontdoor_building_latitude double precision,
    frontdoor_building_longitude double precision,
    frontdoor_building_elevator_renovated boolean,
    frontdoor_building_elevator_renovated_year integer,
    frontdoor_building_facade_renovated boolean,
    frontdoor_building_facade_renovated_year integer,
    frontdoor_building_window_renovated boolean,
    frontdoor_building_window_renovated_year integer,
    frontdoor_building_roof_renovated boolean,
    frontdoor_building_roof_renovated_year integer,
    frontdoor_building_pipe_renovated boolean,
    frontdoor_building_pipe_renovated_year integer,
    frontdoor_building_balcony_renovated boolean,
    frontdoor_building_balcony_renovated_year integer,
    frontdoor_building_electricity_renovated boolean,
    frontdoor_building_electricity_renovated_year integer,
    frontdoor_building_contact_phone text,
    frontdoor_building_contact_office_name text,
    frontdoor_building_contact_office_id integer,
    frontdoor_building_description text,
    frontdoor_building_car_storage_description text,
    frontdoor_building_other_info text,
    frontdoor_building_additional_addresses jsonb[],
    frontdoor_building_links jsonb[],
    frontdoor_building_data jsonb,
    frontdoor_building_processed_at timestamp with time zone,
    frontdoor_building_housing_company_id bigint,
    frontdoor_building_housing_company_friendly_id text,
    frontdoor_building_geom postgis.geometry(Point,4326)
);


--
-- Name: hintatiedot_cities_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.hintatiedot_cities_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: houses; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.houses (
    house_id uuid NOT NULL,
    identity_key text NOT NULL,
    address_norm text,
    postal_norm text,
    city_norm text,
    latitude double precision,
    longitude double precision,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: housing_companies; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.housing_companies (
    housing_company_id uuid DEFAULT gen_random_uuid() CONSTRAINT property_buildings_property_building_id_not_null NOT NULL,
    housing_company_identity_key text CONSTRAINT property_buildings_property_building_identity_key_not_null NOT NULL,
    housing_company_postal_norm text,
    housing_company_city_norm text,
    housing_company_address_norm text,
    housing_company_name text,
    housing_company_business_id text,
    housing_company_build_year integer,
    housing_company_floor_count integer,
    housing_company_apartment_count integer,
    housing_company_elevator boolean,
    housing_company_energy_efficiency_label text,
    housing_company_match_reasons jsonb DEFAULT '{}'::jsonb CONSTRAINT property_buildings_property_building_match_reasons_not_null NOT NULL,
    housing_company_created_at timestamp with time zone DEFAULT now() CONSTRAINT property_buildings_property_building_created_at_not_null NOT NULL,
    housing_company_updated_at timestamp with time zone DEFAULT now() CONSTRAINT property_buildings_property_building_updated_at_not_null NOT NULL,
    housing_company_geom postgis.geometry(Point,4326)
);


--
-- Name: housing_company_merge_decisions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.housing_company_merge_decisions (
    housing_company_merge_decision_id uuid DEFAULT gen_random_uuid() CONSTRAINT housing_company_merge_decis_housing_company_merge_deci_not_null NOT NULL,
    source_housing_company_id uuid CONSTRAINT housing_company_merge_decisi_source_housing_company_id_not_null NOT NULL,
    target_housing_company_id uuid CONSTRAINT housing_company_merge_decisi_target_housing_company_id_not_null NOT NULL,
    housing_company_merge_decision_status text DEFAULT 'accepted'::text CONSTRAINT housing_company_merge_deci_housing_company_merge_deci_not_null1 NOT NULL,
    housing_company_merge_decision_method text CONSTRAINT housing_company_merge_deci_housing_company_merge_deci_not_null2 NOT NULL,
    housing_company_merge_decision_score integer,
    housing_company_merge_decision_confidence text,
    housing_company_merge_decision_reasons jsonb DEFAULT '{}'::jsonb CONSTRAINT housing_company_merge_deci_housing_company_merge_deci_not_null3 NOT NULL,
    housing_company_merge_decision_created_at timestamp with time zone DEFAULT now() CONSTRAINT housing_company_merge_deci_housing_company_merge_deci_not_null4 NOT NULL,
    housing_company_merge_decision_decided_at timestamp with time zone DEFAULT now() CONSTRAINT housing_company_merge_deci_housing_company_merge_deci_not_null5 NOT NULL,
    CONSTRAINT housing_company_merge_decision_distinct_check CHECK ((source_housing_company_id <> target_housing_company_id)),
    CONSTRAINT housing_company_merge_decision_method_check CHECK ((housing_company_merge_decision_method = ANY (ARRAY['source_match_auto'::text, 'manual'::text, 'backfill_auto'::text]))),
    CONSTRAINT housing_company_merge_decision_status_check CHECK ((housing_company_merge_decision_status = ANY (ARRAY['proposed'::text, 'accepted'::text, 'rejected'::text, 'superseded'::text])))
);


--
-- Name: listings; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.listings (
    listing_id uuid NOT NULL,
    listing_type text NOT NULL,
    listing_status text,
    primary_source_listing_id uuid,
    unit_id uuid,
    house_id uuid,
    first_seen_at timestamp with time zone,
    last_seen_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT listings_listing_type_check CHECK ((listing_type = ANY (ARRAY['sale'::text])))
);


--
-- Name: oauth_authorization_codes; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.oauth_authorization_codes (
    oauth_authorization_code_id uuid DEFAULT gen_random_uuid() NOT NULL,
    oauth_authorization_code_code_hash text CONSTRAINT oauth_authorization_codes_oauth_authorization_code_cod_not_null NOT NULL,
    oauth_client_id text NOT NULL,
    user_uuid uuid NOT NULL,
    oauth_authorization_code_redirect_uri text CONSTRAINT oauth_authorization_codes_oauth_authorization_code_red_not_null NOT NULL,
    oauth_authorization_code_scopes text[] DEFAULT '{}'::text[] CONSTRAINT oauth_authorization_codes_oauth_authorization_code_sco_not_null NOT NULL,
    oauth_authorization_code_code_challenge text CONSTRAINT oauth_authorization_codes_oauth_authorization_code_co_not_null1 NOT NULL,
    oauth_authorization_code_code_challenge_method text CONSTRAINT oauth_authorization_codes_oauth_authorization_code_co_not_null2 NOT NULL,
    oauth_authorization_code_audience text DEFAULT ''::text CONSTRAINT oauth_authorization_codes_oauth_authorization_code_aud_not_null NOT NULL,
    oauth_authorization_code_expires_at timestamp with time zone CONSTRAINT oauth_authorization_codes_oauth_authorization_code_exp_not_null NOT NULL,
    oauth_authorization_code_consumed_at timestamp with time zone,
    oauth_authorization_code_created_at timestamp with time zone DEFAULT now() CONSTRAINT oauth_authorization_codes_oauth_authorization_code_cre_not_null NOT NULL,
    oauth_authorization_code_updated_at timestamp with time zone DEFAULT now() CONSTRAINT oauth_authorization_codes_oauth_authorization_code_upd_not_null NOT NULL
);


--
-- Name: oauth_authorization_handoffs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.oauth_authorization_handoffs (
    oauth_authorization_handoff_id uuid DEFAULT gen_random_uuid() CONSTRAINT oauth_authorization_handoff_oauth_authorization_handof_not_null NOT NULL,
    oauth_authorization_handoff_token_hash text CONSTRAINT oauth_authorization_handof_oauth_authorization_handof_not_null1 NOT NULL,
    oauth_authorization_handoff_user_code text CONSTRAINT oauth_authorization_handof_oauth_authorization_handof_not_null2 NOT NULL,
    oauth_client_id text NOT NULL,
    oauth_authorization_handoff_redirect_uri text CONSTRAINT oauth_authorization_handof_oauth_authorization_handof_not_null3 NOT NULL,
    oauth_authorization_handoff_scopes text[] DEFAULT '{}'::text[] CONSTRAINT oauth_authorization_handof_oauth_authorization_handof_not_null4 NOT NULL,
    oauth_authorization_handoff_audience text DEFAULT ''::text CONSTRAINT oauth_authorization_handof_oauth_authorization_handof_not_null5 NOT NULL,
    oauth_authorization_handoff_state text DEFAULT ''::text CONSTRAINT oauth_authorization_handof_oauth_authorization_handof_not_null6 NOT NULL,
    oauth_authorization_handoff_code_challenge text CONSTRAINT oauth_authorization_handof_oauth_authorization_handof_not_null7 NOT NULL,
    oauth_authorization_handoff_code_challenge_method text CONSTRAINT oauth_authorization_handof_oauth_authorization_handof_not_null8 NOT NULL,
    user_uuid uuid,
    oauth_authorization_handoff_authorization_code text,
    oauth_authorization_handoff_redirect_url text,
    oauth_authorization_handoff_denied_at timestamp with time zone,
    oauth_authorization_handoff_completed_at timestamp with time zone,
    oauth_authorization_handoff_expires_at timestamp with time zone CONSTRAINT oauth_authorization_handof_oauth_authorization_handof_not_null9 NOT NULL,
    oauth_authorization_handoff_created_at timestamp with time zone DEFAULT now() CONSTRAINT oauth_authorization_handof_oauth_authorization_hando_not_null10 NOT NULL,
    oauth_authorization_handoff_updated_at timestamp with time zone DEFAULT now() CONSTRAINT oauth_authorization_handof_oauth_authorization_hando_not_null11 NOT NULL
);


--
-- Name: oauth_device_authorizations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.oauth_device_authorizations (
    oauth_device_authorization_id uuid DEFAULT gen_random_uuid() CONSTRAINT oauth_device_authorizations_oauth_device_authorization_not_null NOT NULL,
    oauth_device_authorization_device_code_hash text CONSTRAINT oauth_device_authorization_oauth_device_authorization_not_null1 NOT NULL,
    oauth_client_id text NOT NULL,
    oauth_device_authorization_user_code text CONSTRAINT oauth_device_authorization_oauth_device_authorization_not_null2 NOT NULL,
    oauth_device_authorization_scopes text[] DEFAULT '{}'::text[] CONSTRAINT oauth_device_authorization_oauth_device_authorization_not_null3 NOT NULL,
    oauth_device_authorization_audience text DEFAULT ''::text CONSTRAINT oauth_device_authorization_oauth_device_authorization_not_null4 NOT NULL,
    user_uuid uuid,
    oauth_device_authorization_expires_at timestamp with time zone CONSTRAINT oauth_device_authorization_oauth_device_authorization_not_null5 NOT NULL,
    oauth_device_authorization_approved_at timestamp with time zone,
    oauth_device_authorization_denied_at timestamp with time zone,
    oauth_device_authorization_consumed_at timestamp with time zone,
    oauth_device_authorization_created_at timestamp with time zone DEFAULT now() CONSTRAINT oauth_device_authorization_oauth_device_authorization_not_null6 NOT NULL,
    oauth_device_authorization_updated_at timestamp with time zone DEFAULT now() CONSTRAINT oauth_device_authorization_oauth_device_authorization_not_null7 NOT NULL
);


--
-- Name: oauth_dynamic_clients; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.oauth_dynamic_clients (
    oauth_dynamic_client_id text NOT NULL,
    oauth_dynamic_client_type text DEFAULT 'public'::text NOT NULL,
    oauth_dynamic_client_redirect_uris text[] DEFAULT '{}'::text[] CONSTRAINT oauth_dynamic_clients_oauth_dynamic_client_redirect_ur_not_null NOT NULL,
    oauth_dynamic_client_scopes text[] DEFAULT '{}'::text[] NOT NULL,
    oauth_dynamic_client_token_endpoint_auth_method text DEFAULT 'none'::text CONSTRAINT oauth_dynamic_clients_oauth_dynamic_client_token_endpo_not_null NOT NULL,
    oauth_dynamic_client_name text,
    oauth_dynamic_client_metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    oauth_dynamic_client_issued_at timestamp with time zone DEFAULT now() NOT NULL,
    oauth_dynamic_client_disabled_at timestamp with time zone,
    oauth_dynamic_client_created_at timestamp with time zone DEFAULT now() NOT NULL,
    oauth_dynamic_client_updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: oauth_refresh_tokens; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.oauth_refresh_tokens (
    oauth_refresh_token_id uuid DEFAULT gen_random_uuid() NOT NULL,
    oauth_refresh_token_token_hash text NOT NULL,
    oauth_client_id text NOT NULL,
    user_uuid uuid NOT NULL,
    oauth_refresh_token_scopes text[] DEFAULT '{}'::text[] NOT NULL,
    oauth_refresh_token_audience text DEFAULT ''::text NOT NULL,
    oauth_refresh_token_expires_at timestamp with time zone NOT NULL,
    oauth_refresh_token_revoked_at timestamp with time zone,
    oauth_refresh_token_rotated_from uuid,
    oauth_refresh_token_created_at timestamp with time zone DEFAULT now() NOT NULL,
    oauth_refresh_token_updated_at timestamp with time zone DEFAULT now() NOT NULL,
    device_session_uuid uuid
);


--
-- Name: personal_access_tokens; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.personal_access_tokens (
    personal_access_token_id uuid DEFAULT gen_random_uuid() NOT NULL,
    personal_access_token_name text NOT NULL,
    personal_access_token_prefix text NOT NULL,
    personal_access_token_token_hash text CONSTRAINT personal_access_tokens_personal_access_token_token_has_not_null NOT NULL,
    personal_access_token_scopes text[],
    personal_access_token_created_at timestamp with time zone DEFAULT now() CONSTRAINT personal_access_tokens_personal_access_token_created_a_not_null NOT NULL,
    personal_access_token_last_used_at timestamp with time zone,
    personal_access_token_expires_at timestamp with time zone,
    personal_access_token_revoked_at timestamp with time zone,
    user_id bigint NOT NULL
);


--
-- Name: physical_buildings; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.physical_buildings (
    physical_building_id uuid DEFAULT gen_random_uuid() NOT NULL,
    housing_company_id uuid,
    physical_building_identity_key text NOT NULL,
    physical_building_address_norm text,
    physical_building_postal_norm text,
    physical_building_city_norm text,
    physical_building_build_year integer,
    physical_building_floor_count integer,
    physical_building_apartment_count integer,
    physical_building_elevator boolean,
    physical_building_latitude double precision,
    physical_building_longitude double precision,
    physical_building_created_at timestamp with time zone DEFAULT now() NOT NULL,
    physical_building_updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: postal_ad_areas; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.postal_ad_areas (
    postal_ad_area_id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    postal_ad_area_code text NOT NULL,
    postal_ad_area_name_fi text NOT NULL,
    postal_ad_area_name_sv text,
    postal_ad_area_created_at timestamp with time zone DEFAULT now() NOT NULL,
    postal_ad_area_updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: postal_municipalities; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.postal_municipalities (
    postal_municipality_id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    postal_municipality_code text NOT NULL,
    postal_municipality_name_fi text NOT NULL,
    postal_municipality_name_sv text,
    postal_municipality_language_ratio_code text,
    postal_municipality_created_at timestamp with time zone DEFAULT now() NOT NULL,
    postal_municipality_updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: postal_postal_codes; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.postal_postal_codes (
    postal_postal_code_id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    postal_postal_code_date date NOT NULL,
    postal_postal_code_code text NOT NULL,
    postal_postal_code_name_fi text NOT NULL,
    postal_postal_code_name_sv text,
    postal_postal_code_abbr_fi text,
    postal_postal_code_abbr_sv text,
    postal_postal_code_valid_from date,
    postal_postal_code_type_code text,
    postal_ad_area_id uuid,
    postal_municipality_id uuid,
    postal_postal_code_created_at timestamp with time zone DEFAULT now() NOT NULL,
    postal_postal_code_updated_at timestamp with time zone DEFAULT now() NOT NULL,
    postal_postal_code_neighborhood_fi text
);


--
-- Name: price_links; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.price_links (
    price_link_id uuid DEFAULT gen_random_uuid() NOT NULL,
    target_type text NOT NULL,
    target_id uuid NOT NULL,
    prices_transaction_id uuid NOT NULL,
    link_status text NOT NULL,
    link_method text NOT NULL,
    link_score integer NOT NULL,
    link_reasons jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT price_links_link_method_check CHECK ((link_method = ANY (ARRAY['sync_auto'::text, 'source_match_auto'::text, 'document_match_auto'::text, 'manual'::text, 'backfill_auto'::text]))),
    CONSTRAINT price_links_link_status_check CHECK ((link_status = ANY (ARRAY['confirmed'::text, 'candidate'::text, 'rejected'::text, 'superseded'::text]))),
    CONSTRAINT price_links_target_type_check CHECK ((target_type = ANY (ARRAY['listing'::text, 'source_listing'::text, 'source_building_announcement'::text, 'building'::text, 'housing_company'::text])))
);


--
-- Name: prices_cities; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.prices_cities (
    prices_city_id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    prices_city_name text NOT NULL,
    prices_city_created_at timestamp with time zone DEFAULT now() NOT NULL,
    prices_city_updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: prices_neighborhoods; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.prices_neighborhoods (
    prices_neighborhood_id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    prices_neighborhood_name text NOT NULL,
    prices_city_id uuid NOT NULL,
    prices_postal_code_id uuid,
    prices_neighborhood_created_at timestamp with time zone DEFAULT now() NOT NULL,
    prices_neighborhood_updated_at timestamp with time zone DEFAULT now() NOT NULL,
    prices_neighborhood_postal_postal_code_id uuid
);


--
-- Name: prices_postal_codes; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.prices_postal_codes (
    prices_postal_code_id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    prices_postal_code_code text NOT NULL,
    prices_city_id uuid NOT NULL,
    prices_postal_code_created_at timestamp with time zone DEFAULT now() NOT NULL,
    prices_postal_code_updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: prices_transactions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.prices_transactions (
    prices_transaction_id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    prices_transaction_description text NOT NULL,
    prices_transaction_type text NOT NULL,
    prices_transaction_area double precision NOT NULL,
    prices_transaction_price integer NOT NULL,
    prices_transaction_price_per_square_meter integer CONSTRAINT prices_transactions_prices_transaction_price_per_squar_not_null NOT NULL,
    prices_transaction_build_year integer NOT NULL,
    prices_transaction_floor text,
    prices_transaction_elevator boolean NOT NULL,
    prices_transaction_condition text,
    prices_transaction_plot text,
    prices_transaction_energy_class text,
    prices_transaction_period_identifier text CONSTRAINT prices_transactions_prices_transaction_period_identifi_not_null NOT NULL,
    prices_transaction_created_at timestamp with time zone DEFAULT now() NOT NULL,
    prices_transaction_updated_at timestamp with time zone DEFAULT now() NOT NULL,
    prices_transaction_category text NOT NULL,
    prices_neighborhood_id uuid,
    prices_transaction_plot_owned boolean
);


--
-- Name: property_dimension_catalog; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.property_dimension_catalog (
    dimension_key text NOT NULL,
    target_type text NOT NULL,
    value_kind text NOT NULL,
    unit text,
    profile_section text NOT NULL,
    profile_key text NOT NULL,
    promoted_to_valuation boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT property_dimension_catalog_target_type_check CHECK ((target_type = ANY (ARRAY['offering'::text, 'unit'::text, 'building'::text, 'housing_company'::text, 'house'::text]))),
    CONSTRAINT property_dimension_catalog_value_kind_check CHECK ((value_kind = ANY (ARRAY['string'::text, 'number'::text, 'boolean'::text, 'object'::text, 'array'::text, 'null'::text])))
);


--
-- Name: property_dimension_dirty_targets; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.property_dimension_dirty_targets (
    target_type text NOT NULL,
    target_id uuid NOT NULL,
    dirty_reasons text[] DEFAULT '{}'::text[] NOT NULL,
    dirty_at timestamp with time zone DEFAULT now() NOT NULL,
    queued_at timestamp with time zone,
    resolved_at timestamp with time zone,
    CONSTRAINT property_dimension_dirty_targets_target_type_check CHECK ((target_type = ANY (ARRAY['listing'::text, 'document'::text, 'transaction'::text, 'offering'::text, 'unit'::text, 'building'::text, 'housing_company'::text, 'house'::text])))
);


--
-- Name: property_dimension_projection_runs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.property_dimension_projection_runs (
    property_dimension_projection_run_id uuid DEFAULT gen_random_uuid() CONSTRAINT property_dimension_projecti_property_dimension_project_not_null NOT NULL,
    projection_type text NOT NULL,
    projection_version text NOT NULL,
    source_table text NOT NULL,
    source_id uuid NOT NULL,
    status text NOT NULL,
    result jsonb DEFAULT '{}'::jsonb NOT NULL,
    error_text text,
    started_at timestamp with time zone DEFAULT now() NOT NULL,
    finished_at timestamp with time zone,
    CONSTRAINT property_dimension_projection_runs_projection_type_check CHECK ((projection_type = ANY (ARRAY['source_claims'::text, 'renovation_events'::text, 'resolved_values'::text, 'profiles'::text, 'system_profiles'::text]))),
    CONSTRAINT property_dimension_projection_runs_status_check CHECK ((status = ANY (ARRAY['running'::text, 'succeeded'::text, 'failed'::text])))
);


--
-- Name: property_dimension_resolution_policies; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.property_dimension_resolution_policies (
    dimension_key text NOT NULL,
    strategy text NOT NULL,
    freshness_half_life_days integer,
    conflict_tolerance jsonb DEFAULT '{}'::jsonb CONSTRAINT property_dimension_resolution_polic_conflict_tolerance_not_null NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT property_dimension_resolution_policies_strategy_check CHECK ((strategy = ANY (ARRAY['manual_override'::text, 'latest_reliable'::text, 'highest_reliability'::text, 'document_preferred'::text, 'stable_identity'::text, 'numeric_consensus'::text])))
);


--
-- Name: property_dimension_source_priorities; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.property_dimension_source_priorities (
    dimension_key text NOT NULL,
    source_table text NOT NULL,
    source_field text,
    priority integer NOT NULL,
    default_reliability double precision CONSTRAINT property_dimension_source_prioriti_default_reliability_not_null NOT NULL,
    CONSTRAINT property_dimension_source_priorities_default_reliability_check CHECK (((default_reliability >= (0)::double precision) AND (default_reliability <= (1)::double precision)))
);


--
-- Name: property_document_extraction_runs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.property_document_extraction_runs (
    property_document_extraction_run_id uuid DEFAULT gen_random_uuid() CONSTRAINT property_document_extractio_property_document_extracti_not_null NOT NULL,
    property_document_id uuid NOT NULL,
    property_document_extraction_run_model text CONSTRAINT property_document_extracti_property_document_extracti_not_null1 NOT NULL,
    property_document_extraction_run_prompt_version text CONSTRAINT property_document_extracti_property_document_extracti_not_null2 NOT NULL,
    property_document_extraction_run_status text CONSTRAINT property_document_extracti_property_document_extracti_not_null3 NOT NULL,
    property_document_extraction_run_raw_json jsonb,
    property_document_extraction_run_error text,
    property_document_extraction_run_started_at timestamp with time zone DEFAULT now() CONSTRAINT property_document_extracti_property_document_extracti_not_null4 NOT NULL,
    property_document_extraction_run_finished_at timestamp with time zone,
    CONSTRAINT property_document_extraction_runs_status_check CHECK ((property_document_extraction_run_status = ANY (ARRAY['running'::text, 'succeeded'::text, 'failed'::text])))
);


--
-- Name: property_document_extractions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.property_document_extractions (
    property_document_extraction_id uuid DEFAULT gen_random_uuid() CONSTRAINT property_document_extracti_property_document_extracti_not_null5 NOT NULL,
    property_document_id uuid NOT NULL,
    property_document_extraction_kind text CONSTRAINT property_document_extracti_property_document_extracti_not_null6 NOT NULL,
    property_document_extraction_schema_version text CONSTRAINT property_document_extracti_property_document_extracti_not_null7 NOT NULL,
    property_document_extraction_model text CONSTRAINT property_document_extracti_property_document_extracti_not_null8 NOT NULL,
    property_document_extraction_prompt_version text CONSTRAINT property_document_extracti_property_document_extracti_not_null9 NOT NULL,
    property_document_extraction_source_json jsonb CONSTRAINT property_document_extracti_property_document_extract_not_null10 NOT NULL,
    property_document_extraction_status text DEFAULT 'succeeded'::text CONSTRAINT property_document_extracti_property_document_extract_not_null11 NOT NULL,
    property_document_extraction_error text,
    property_document_extraction_created_at timestamp with time zone DEFAULT now() CONSTRAINT property_document_extracti_property_document_extract_not_null12 NOT NULL,
    property_document_extraction_extracted_at timestamp with time zone DEFAULT now() CONSTRAINT property_document_extracti_property_document_extract_not_null13 NOT NULL,
    property_document_extraction_superseded_at timestamp with time zone,
    CONSTRAINT property_document_extractions_kind_check CHECK ((property_document_extraction_kind = ANY (ARRAY['manager_certificate'::text]))),
    CONSTRAINT property_document_extractions_schema_version_check CHECK ((property_document_extraction_schema_version <> ''::text)),
    CONSTRAINT property_document_extractions_status_check CHECK ((property_document_extraction_status = ANY (ARRAY['succeeded'::text, 'failed'::text, 'superseded'::text])))
);


--
-- Name: property_documents; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.property_documents (
    property_document_id uuid DEFAULT gen_random_uuid() NOT NULL,
    property_offering_id uuid,
    property_unit_id uuid,
    physical_building_id uuid,
    housing_company_id uuid,
    property_document_type text NOT NULL,
    property_document_filename text NOT NULL,
    property_document_mime_type text NOT NULL,
    property_document_size_bytes bigint NOT NULL,
    property_document_sha256 text NOT NULL,
    property_document_bytes bytea NOT NULL,
    property_document_extraction_status text DEFAULT 'uploaded'::text NOT NULL,
    property_document_extraction_error text,
    property_document_uploaded_at timestamp with time zone DEFAULT now() NOT NULL,
    property_document_extracted_at timestamp with time zone,
    property_document_created_at timestamp with time zone DEFAULT now() NOT NULL,
    property_document_updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT property_documents_document_type_check CHECK ((property_document_type = ANY (ARRAY['manager_certificate'::text]))),
    CONSTRAINT property_documents_extraction_status_check CHECK ((property_document_extraction_status = ANY (ARRAY['uploaded'::text, 'extracting'::text, 'extracted'::text, 'failed'::text]))),
    CONSTRAINT property_documents_mime_type_check CHECK ((property_document_mime_type = 'application/pdf'::text)),
    CONSTRAINT property_documents_sha256_check CHECK ((property_document_sha256 ~ '^[0-9a-f]{64}$'::text)),
    CONSTRAINT property_documents_size_bytes_check CHECK (((property_document_size_bytes > 0) AND (property_document_size_bytes <= 26214400)))
);


--
-- Name: property_houses; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.property_houses (
    property_house_id uuid DEFAULT gen_random_uuid() NOT NULL,
    property_house_identity_key text NOT NULL,
    property_house_address_norm text,
    property_house_postal_norm text,
    property_house_city_norm text,
    property_house_build_year integer,
    property_house_area_value double precision,
    property_house_plot_area_value double precision,
    property_house_rooms_count integer,
    property_house_latitude double precision,
    property_house_longitude double precision,
    property_house_match_reasons jsonb DEFAULT '{}'::jsonb NOT NULL,
    primary_sale_listing_id uuid,
    property_house_created_at timestamp with time zone DEFAULT now() NOT NULL,
    property_house_updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: property_offering_merge_decisions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.property_offering_merge_decisions (
    property_offering_merge_decision_id uuid DEFAULT gen_random_uuid() CONSTRAINT property_offering_merge_dec_property_offering_merge_de_not_null NOT NULL,
    source_property_offering_id uuid CONSTRAINT property_offering_merge_dec_source_property_offering_i_not_null NOT NULL,
    target_property_offering_id uuid CONSTRAINT property_offering_merge_dec_target_property_offering_i_not_null NOT NULL,
    property_offering_merge_decision_status text DEFAULT 'accepted'::text CONSTRAINT property_offering_merge_de_property_offering_merge_de_not_null1 NOT NULL,
    property_offering_merge_decision_method text CONSTRAINT property_offering_merge_de_property_offering_merge_de_not_null2 NOT NULL,
    property_offering_merge_decision_score integer,
    property_offering_merge_decision_confidence text,
    property_offering_merge_decision_reasons jsonb DEFAULT '{}'::jsonb CONSTRAINT property_offering_merge_de_property_offering_merge_de_not_null3 NOT NULL,
    property_offering_merge_decision_created_at timestamp with time zone DEFAULT now() CONSTRAINT property_offering_merge_de_property_offering_merge_de_not_null4 NOT NULL,
    property_offering_merge_decision_decided_at timestamp with time zone DEFAULT now() CONSTRAINT property_offering_merge_de_property_offering_merge_de_not_null5 NOT NULL,
    CONSTRAINT property_offering_merge_decision_distinct_check CHECK ((source_property_offering_id <> target_property_offering_id)),
    CONSTRAINT property_offering_merge_decision_method_check CHECK ((property_offering_merge_decision_method = ANY (ARRAY['source_match_auto'::text, 'manual'::text, 'backfill_auto'::text]))),
    CONSTRAINT property_offering_merge_decision_status_check CHECK ((property_offering_merge_decision_status = ANY (ARRAY['proposed'::text, 'accepted'::text, 'rejected'::text, 'superseded'::text])))
);


--
-- Name: property_offerings; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.property_offerings (
    property_offering_id uuid DEFAULT gen_random_uuid() NOT NULL,
    property_unit_id uuid,
    property_offering_identity_key text NOT NULL,
    property_offering_type text NOT NULL,
    property_offering_headline text NOT NULL,
    property_offering_asking_price bigint,
    property_offering_debt_free_price bigint,
    property_offering_price_per_m2 double precision,
    property_offering_first_seen_at timestamp with time zone,
    property_offering_last_seen_at timestamp with time zone,
    property_offering_status text,
    primary_sale_listing_id uuid,
    property_offering_match_reasons jsonb DEFAULT '{}'::jsonb NOT NULL,
    property_offering_created_at timestamp with time zone DEFAULT now() NOT NULL,
    property_offering_updated_at timestamp with time zone DEFAULT now() NOT NULL,
    property_house_id uuid,
    CONSTRAINT property_offerings_parent_check CHECK (((((property_unit_id IS NOT NULL))::integer + ((property_house_id IS NOT NULL))::integer) = 1)),
    CONSTRAINT property_offerings_type_check CHECK ((property_offering_type = ANY (ARRAY['sale'::text])))
);


--
-- Name: property_renovation_events; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.property_renovation_events (
    property_renovation_event_id uuid DEFAULT gen_random_uuid() CONSTRAINT property_renovation_events_property_renovation_event_i_not_null NOT NULL,
    property_dimension_projection_run_id uuid CONSTRAINT property_renovation_events_property_dimension_projecti_not_null NOT NULL,
    projection_version text NOT NULL,
    event_scope text NOT NULL,
    target_type text NOT NULL,
    target_id uuid NOT NULL,
    source_table text NOT NULL,
    source_id uuid NOT NULL,
    source_field text,
    source_event_id uuid,
    category text NOT NULL,
    component text,
    status text NOT NULL,
    stage text,
    scope text,
    responsibility text,
    year integer,
    start_year integer,
    end_year integer,
    cost_estimate_eur bigint,
    summary text,
    evidence jsonb DEFAULT '{}'::jsonb NOT NULL,
    confidence double precision DEFAULT 0.5 NOT NULL,
    source_reliability double precision DEFAULT 0.5 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    source_observed_at timestamp with time zone,
    CONSTRAINT property_renovation_events_confidence_check CHECK (((confidence >= (0)::double precision) AND (confidence <= (1)::double precision))),
    CONSTRAINT property_renovation_events_event_scope_check CHECK ((event_scope = ANY (ARRAY['source'::text, 'manual'::text]))),
    CONSTRAINT property_renovation_events_source_reliability_check CHECK (((source_reliability >= (0)::double precision) AND (source_reliability <= (1)::double precision))),
    CONSTRAINT property_renovation_events_target_type_check CHECK ((target_type = ANY (ARRAY['listing'::text, 'document'::text, 'offering'::text, 'unit'::text, 'building'::text, 'housing_company'::text, 'house'::text])))
);


--
-- Name: property_source_offering_renovations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.property_source_offering_renovations (
    property_source_offering_renovation_id uuid DEFAULT gen_random_uuid() CONSTRAINT property_source_offering_re_property_source_offering_r_not_null NOT NULL,
    sale_listing_id uuid NOT NULL,
    property_source_offering_renovation_source_field text CONSTRAINT property_source_offering_r_property_source_offering_r_not_null1 NOT NULL,
    property_source_offering_renovation_category text CONSTRAINT property_source_offering_r_property_source_offering_r_not_null2 NOT NULL,
    property_source_offering_renovation_status text CONSTRAINT property_source_offering_r_property_source_offering_r_not_null3 NOT NULL,
    property_source_offering_renovation_year integer,
    property_source_offering_renovation_text text,
    property_source_offering_renovation_confidence integer DEFAULT 100 CONSTRAINT property_source_offering_r_property_source_offering_r_not_null4 NOT NULL,
    property_source_offering_renovation_created_at timestamp with time zone DEFAULT now() CONSTRAINT property_source_offering_r_property_source_offering_r_not_null5 NOT NULL,
    property_source_offering_renovation_updated_at timestamp with time zone DEFAULT now() CONSTRAINT property_source_offering_r_property_source_offering_r_not_null6 NOT NULL,
    property_source_offering_renovation_component text,
    property_source_offering_renovation_scope text,
    property_source_offering_renovation_stage text,
    property_source_offering_renovation_responsibility text,
    property_source_offering_renovation_cost_estimate_eur bigint,
    CONSTRAINT property_source_offering_renovation_status_check CHECK ((property_source_offering_renovation_status = ANY (ARRAY['done'::text, 'planned'::text, 'unknown'::text])))
);


--
-- Name: property_source_offerings; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.property_source_offerings (
    sale_listing_id uuid DEFAULT gen_random_uuid() CONSTRAINT sale_listings_sale_listing_id_not_null NOT NULL,
    shortcut_ad_id bigint,
    frontdoor_ad_id uuid,
    frontdoor_building_announcement_id uuid,
    prices_transaction_id uuid,
    sale_listing_source_provider text CONSTRAINT sale_listings_sale_listing_source_provider_not_null NOT NULL,
    sale_listing_source_kind text CONSTRAINT sale_listings_sale_listing_source_kind_not_null NOT NULL,
    sale_listing_native_id text CONSTRAINT sale_listings_sale_listing_native_id_not_null NOT NULL,
    sale_listing_canonical_id text CONSTRAINT sale_listings_sale_listing_canonical_id_not_null NOT NULL,
    sale_listing_url text,
    sale_listing_headline text CONSTRAINT sale_listings_sale_listing_headline_not_null NOT NULL,
    sale_listing_street_address text,
    sale_listing_city text,
    sale_listing_postal text,
    sale_listing_asking_price bigint,
    sale_listing_area_value double precision,
    sale_listing_room_layout text,
    sale_listing_last_seen_at timestamp with time zone,
    sale_listing_published_at timestamp with time zone,
    sale_listing_search_text text,
    sale_listing_created_at timestamp with time zone DEFAULT now() CONSTRAINT sale_listings_sale_listing_created_at_not_null NOT NULL,
    sale_listing_updated_at timestamp with time zone DEFAULT now() CONSTRAINT sale_listings_sale_listing_updated_at_not_null NOT NULL,
    sale_listing_street_name text,
    sale_listing_street_number text,
    sale_listing_building_letter text,
    sale_listing_apartment text,
    sale_listing_street_name_norm text,
    sale_listing_street_number_norm text,
    sale_listing_building_letter_norm text,
    sale_listing_city_norm text,
    sale_listing_postal_norm text,
    sale_listing_address_norm text,
    sale_listing_address_components jsonb,
    sale_listing_building_match_key text,
    sale_listing_street_match_key text,
    sale_listing_unit_match_key text,
    sale_listing_price_per_m2 double precision,
    sale_listing_debt_free_price bigint,
    sale_listing_debt_share_amount bigint,
    sale_listing_rooms_count integer,
    sale_listing_floor_level integer,
    sale_listing_total_floors integer,
    sale_listing_build_year integer,
    sale_listing_condition text,
    sale_listing_energy_class text,
    sale_listing_description_text text,
    sale_listing_property_type_raw text,
    sale_listing_property_type_code text,
    sale_listing_room_category_code text,
    sale_listing_floor_text text,
    sale_listing_elevator boolean,
    sale_listing_plot_type_raw text,
    sale_listing_plot_type_code text,
    sale_listing_energy_efficiency_label text,
    sale_listing_energy_efficiency_class_code text,
    sale_listing_energy_efficiency_standard_year integer,
    sale_listing_energy_efficiency_status text,
    sale_listing_energy_efficiency_match_code text,
    sale_listing_first_seen_at timestamp with time zone,
    sale_listing_prices_match_status text,
    sale_listing_prices_match_next_attempt_at timestamp with time zone,
    sale_listing_prices_match_last_attempted_at timestamp with time zone,
    sale_listing_prices_match_attempt_count integer DEFAULT 0 CONSTRAINT sale_listings_sale_listing_prices_match_attempt_count_not_null NOT NULL,
    sale_listing_prices_match_expires_at timestamp with time zone,
    sale_listing_prices_match_run_id uuid,
    sale_listing_plot_owned boolean,
    sale_listing_source_match_status text,
    sale_listing_source_match_next_attempt_at timestamp with time zone,
    sale_listing_source_match_last_attempted_at timestamp with time zone,
    sale_listing_source_match_attempt_count integer DEFAULT 0 CONSTRAINT sale_listings_sale_listing_source_match_attempt_count_not_null NOT NULL,
    sale_listing_availability_text text,
    sale_listing_renovations_done_text text,
    sale_listing_renovations_planned_text text,
    sale_listing_additional_info_text text,
    sale_listing_charges_text text,
    sale_listing_maintenance_charge_monthly double precision,
    sale_listing_total_charge_monthly double precision,
    sale_listing_water_charge double precision,
    sale_listing_housing_company_name text,
    sale_listing_housing_company_business_id text,
    sale_listing_building_material text,
    sale_listing_heating_system text,
    sale_listing_roof_type text,
    sale_listing_roof_material text,
    sale_listing_apartment_count integer,
    sale_listing_car_storage_text text,
    sale_listing_building_description_text text,
    sale_listing_building_other_info_text text,
    sale_listing_latitude double precision,
    sale_listing_longitude double precision,
    sale_listing_living_area_value double precision,
    sale_listing_total_area_value double precision,
    sale_listing_other_area_value double precision,
    sale_listing_bedrooms_count integer,
    sale_listing_sauna boolean,
    sale_listing_balcony boolean,
    sale_listing_parking_text text,
    sale_listing_kitchen_description_text text,
    sale_listing_bathroom_description_text text,
    sale_listing_storage_description_text text,
    sale_listing_floor_materials_description_text text,
    sale_listing_wall_materials_description_text text,
    sale_listing_balcony_description_text text,
    sale_listing_sauna_description_text text,
    sale_listing_views_description_text text,
    sale_listing_appliances text[],
    sale_listing_features text[],
    sale_listing_plot_area_value double precision,
    sale_listing_services_text text,
    sale_listing_transport_text text,
    sale_listing_previous_asking_price bigint,
    sale_listing_previous_debt_free_price bigint,
    sale_listing_new_development boolean,
    CONSTRAINT sale_listings_has_source_check CHECK (((shortcut_ad_id IS NOT NULL) OR (frontdoor_ad_id IS NOT NULL) OR (frontdoor_building_announcement_id IS NOT NULL))),
    CONSTRAINT sale_listings_prices_match_status_check CHECK (((sale_listing_prices_match_status IS NULL) OR (sale_listing_prices_match_status = ANY (ARRAY['pending'::text, 'deferred'::text, 'auto_linked'::text, 'needs_review'::text, 'manual_linked'::text, 'rejected'::text, 'expired'::text, 'noop'::text])))),
    CONSTRAINT sale_listings_source_kind_check CHECK ((sale_listing_source_kind = ANY (ARRAY['ad'::text, 'announcement'::text]))),
    CONSTRAINT sale_listings_source_match_status_check CHECK (((sale_listing_source_match_status IS NULL) OR (sale_listing_source_match_status = ANY (ARRAY['pending'::text, 'deferred'::text, 'auto_linked'::text, 'needs_review'::text, 'manual_linked'::text, 'rejected'::text, 'noop'::text])))),
    CONSTRAINT sale_listings_source_provider_check CHECK ((sale_listing_source_provider = ANY (ARRAY['shortcut'::text, 'frontdoor'::text])))
);


--
-- Name: property_units; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.property_units (
    property_unit_id uuid DEFAULT gen_random_uuid() NOT NULL,
    housing_company_id uuid CONSTRAINT property_units_property_building_id_not_null NOT NULL,
    property_unit_identity_key text NOT NULL,
    property_unit_address_norm text,
    property_unit_floor_level integer,
    property_unit_area_value double precision,
    property_unit_rooms_count integer,
    property_unit_room_layout text,
    property_unit_layout_match_key text,
    property_unit_match_reasons jsonb DEFAULT '{}'::jsonb NOT NULL,
    property_unit_created_at timestamp with time zone DEFAULT now() NOT NULL,
    property_unit_updated_at timestamp with time zone DEFAULT now() NOT NULL,
    physical_building_id uuid
);


--
-- Name: role_feature_flags; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.role_feature_flags (
    flag_id bigint NOT NULL,
    role_id bigint NOT NULL
);


--
-- Name: roles; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.roles (
    role_uuid uuid DEFAULT gen_random_uuid() NOT NULL,
    role_name text NOT NULL,
    role_description text,
    role_created_at timestamp with time zone DEFAULT now() NOT NULL,
    role_id bigint NOT NULL
);


--
-- Name: roles_role_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.roles ALTER COLUMN role_id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.roles_role_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: sale_listing_plot_type_aliases; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.sale_listing_plot_type_aliases (
    sale_listing_plot_type_alias text CONSTRAINT sale_listing_plot_type_alia_sale_listing_plot_type_ali_not_null NOT NULL,
    sale_listing_plot_type_code text CONSTRAINT sale_listing_plot_type_alia_sale_listing_plot_type_cod_not_null NOT NULL,
    sale_listing_plot_type_label text CONSTRAINT sale_listing_plot_type_alia_sale_listing_plot_type_lab_not_null NOT NULL
);


--
-- Name: sale_listing_prices_transaction_match_candidates; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.sale_listing_prices_transaction_match_candidates (
    sale_listing_prices_transaction_match_candidate_id uuid DEFAULT gen_random_uuid() CONSTRAINT sale_listing_prices_transa_sale_listing_prices_transa_not_null8 NOT NULL,
    sale_listing_prices_transaction_match_run_id uuid CONSTRAINT sale_listing_prices_transa_sale_listing_prices_transa_not_null9 NOT NULL,
    sale_listing_id uuid CONSTRAINT sale_listing_prices_transaction_match__sale_listing_id_not_null NOT NULL,
    prices_transaction_id uuid CONSTRAINT sale_listing_prices_transaction__prices_transaction_id_not_null NOT NULL,
    sale_listing_prices_transaction_match_score integer CONSTRAINT sale_listing_prices_transa_sale_listing_prices_trans_not_null10 NOT NULL,
    sale_listing_prices_transaction_match_confidence text CONSTRAINT sale_listing_prices_transa_sale_listing_prices_trans_not_null11 NOT NULL,
    sale_listing_prices_transaction_match_status text DEFAULT 'candidate'::text CONSTRAINT sale_listing_prices_transa_sale_listing_prices_trans_not_null12 NOT NULL,
    sale_listing_prices_transaction_match_reasons jsonb DEFAULT '{}'::jsonb CONSTRAINT sale_listing_prices_transa_sale_listing_prices_trans_not_null13 NOT NULL,
    sale_listing_prices_transaction_match_price_delta_percent double precision,
    sale_listing_prices_transaction_match_created_at timestamp with time zone DEFAULT now() CONSTRAINT sale_listing_prices_transa_sale_listing_prices_trans_not_null14 NOT NULL,
    CONSTRAINT sale_listing_prices_transaction_match_confidence_check CHECK ((sale_listing_prices_transaction_match_confidence = ANY (ARRAY['high'::text, 'medium'::text, 'low'::text]))),
    CONSTRAINT sale_listing_prices_transaction_match_status_check CHECK ((sale_listing_prices_transaction_match_status = ANY (ARRAY['candidate'::text, 'auto_linked'::text, 'ambiguous'::text, 'rejected'::text])))
);


--
-- Name: sale_listing_prices_transaction_match_runs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.sale_listing_prices_transaction_match_runs (
    sale_listing_prices_transaction_match_run_id uuid DEFAULT gen_random_uuid() CONSTRAINT sale_listing_prices_transac_sale_listing_prices_transa_not_null NOT NULL,
    sale_listing_prices_transaction_match_run_mode text CONSTRAINT sale_listing_prices_transa_sale_listing_prices_transa_not_null1 NOT NULL,
    sale_listing_prices_transaction_match_score_threshold integer DEFAULT 90 CONSTRAINT sale_listing_prices_transa_sale_listing_prices_transa_not_null2 NOT NULL,
    sale_listing_prices_transaction_match_competitor_margin integer DEFAULT 15 CONSTRAINT sale_listing_prices_transa_sale_listing_prices_transa_not_null3 NOT NULL,
    sale_listing_prices_transaction_match_candidates_count integer DEFAULT 0 CONSTRAINT sale_listing_prices_transa_sale_listing_prices_transa_not_null4 NOT NULL,
    sale_listing_prices_transaction_match_auto_linked_count integer DEFAULT 0 CONSTRAINT sale_listing_prices_transa_sale_listing_prices_transa_not_null5 NOT NULL,
    sale_listing_prices_transaction_match_ambiguous_count integer DEFAULT 0 CONSTRAINT sale_listing_prices_transa_sale_listing_prices_transa_not_null6 NOT NULL,
    sale_listing_prices_transaction_match_started_at timestamp with time zone DEFAULT now() CONSTRAINT sale_listing_prices_transa_sale_listing_prices_transa_not_null7 NOT NULL,
    sale_listing_prices_transaction_match_finished_at timestamp with time zone,
    CONSTRAINT sale_listing_prices_transaction_match_margin_check CHECK ((sale_listing_prices_transaction_match_competitor_margin >= 0)),
    CONSTRAINT sale_listing_prices_transaction_match_run_mode_check CHECK ((sale_listing_prices_transaction_match_run_mode = ANY (ARRAY['dry_run'::text, 'auto_link_safe'::text]))),
    CONSTRAINT sale_listing_prices_transaction_match_threshold_check CHECK ((sale_listing_prices_transaction_match_score_threshold >= 0))
);


--
-- Name: sale_listing_property_type_aliases; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.sale_listing_property_type_aliases (
    sale_listing_property_type_alias text CONSTRAINT sale_listing_property_type__sale_listing_property_type_not_null NOT NULL,
    sale_listing_property_type_code text CONSTRAINT sale_listing_property_type_sale_listing_property_type_not_null1 NOT NULL,
    sale_listing_property_type_label text CONSTRAINT sale_listing_property_type_sale_listing_property_type_not_null2 NOT NULL
);


--
-- Name: sale_listing_room_category_aliases; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.sale_listing_room_category_aliases (
    sale_listing_room_category_alias text CONSTRAINT sale_listing_room_category__sale_listing_room_category_not_null NOT NULL,
    sale_listing_room_category_code text CONSTRAINT sale_listing_room_category_sale_listing_room_category_not_null1 NOT NULL,
    sale_listing_room_category_label text CONSTRAINT sale_listing_room_category_sale_listing_room_category_not_null2 NOT NULL
);



--
-- Name: shortcut_ads; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.shortcut_ads (
    shortcut_ad_id bigint NOT NULL,
    shortcut_ad_url text NOT NULL,
    shortcut_ad_type text NOT NULL,
    shortcut_ad_first_seen_at timestamp with time zone DEFAULT now() NOT NULL,
    shortcut_ad_last_seen_at timestamp with time zone DEFAULT now() NOT NULL,
    shortcut_ad_data jsonb,
    shortcut_ad_updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    shortcut_building_id uuid,
    shortcut_ad_data_schema_version smallint DEFAULT 1 NOT NULL,
    shortcut_ad_data_hash text,
    shortcut_ad_data_hash_algorithm text DEFAULT 'sha256'::text NOT NULL,
    shortcut_ad_data_changed_at timestamp with time zone,
    shortcut_ad_data_normalized_at timestamp with time zone,
    shortcut_ad_data_normalized_version integer DEFAULT 0 NOT NULL
);


--
-- Name: shortcut_building_listings; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.shortcut_building_listings (
    shortcut_building_listing_id uuid DEFAULT gen_random_uuid() CONSTRAINT shortcut_building_listings_shortcut_building_listing_i_not_null NOT NULL,
    shortcut_building_id uuid NOT NULL,
    shortcut_building_listing_layout text,
    shortcut_building_listing_size double precision,
    shortcut_building_listing_price double precision,
    shortcut_building_listing_price_per_sqm double precision,
    shortcut_building_listing_deleted_at timestamp with time zone,
    shortcut_building_listing_created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP CONSTRAINT shortcut_building_listings_shortcut_building_listing_c_not_null NOT NULL,
    shortcut_building_listing_updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP CONSTRAINT shortcut_building_listings_shortcut_building_listing_u_not_null NOT NULL,
    shortcut_building_listing_marketing_time text,
    shortcut_building_listing_idx integer
);


--
-- Name: shortcut_building_rentals; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.shortcut_building_rentals (
    shortcut_building_rental_id uuid DEFAULT gen_random_uuid() NOT NULL,
    shortcut_building_id uuid NOT NULL,
    shortcut_building_rental_layout text,
    shortcut_building_rental_size double precision,
    shortcut_building_rental_price double precision,
    shortcut_building_rental_deleted_at timestamp with time zone,
    shortcut_building_rental_created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP CONSTRAINT shortcut_building_rentals_shortcut_building_rental_cre_not_null NOT NULL,
    shortcut_building_rental_updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP CONSTRAINT shortcut_building_rentals_shortcut_building_rental_upd_not_null NOT NULL,
    shortcut_building_rental_marketing_time text,
    shortcut_building_rental_idx integer
);


--
-- Name: shortcut_buildings; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.shortcut_buildings (
    shortcut_building_id uuid DEFAULT gen_random_uuid() NOT NULL,
    shortcut_building_external_id bigint NOT NULL,
    shortcut_building_building_id text,
    shortcut_building_building_type text,
    shortcut_building_building_subtype text,
    shortcut_building_construction_year integer,
    shortcut_building_floor_count integer,
    shortcut_building_apartment_count integer,
    shortcut_building_heating_system text,
    shortcut_building_building_material text,
    shortcut_building_plot_type text,
    shortcut_building_wall_structure text,
    shortcut_building_heat_source text,
    shortcut_building_has_elevator text,
    shortcut_building_has_sauna text,
    shortcut_building_latitude double precision,
    shortcut_building_longitude double precision,
    shortcut_building_additional_addresses text,
    shortcut_building_url text NOT NULL,
    shortcut_building_created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    shortcut_building_updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    shortcut_building_address text,
    shortcut_building_processed_at timestamp with time zone,
    shortcut_building_page_not_found boolean DEFAULT false,
    shortcut_building_frame_construction_method text,
    shortcut_building_housing_company text,
    shortcut_building_geom postgis.geometry(Point,4326)
);


--
-- Name: shortcut_tokens; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.shortcut_tokens (
    shortcut_token_id uuid DEFAULT gen_random_uuid() NOT NULL,
    shortcut_token_cuid text NOT NULL,
    shortcut_token_token text NOT NULL,
    shortcut_token_loaded text NOT NULL,
    shortcut_token_created_at timestamp with time zone DEFAULT now() NOT NULL,
    shortcut_token_updated_at timestamp with time zone DEFAULT now() NOT NULL,
    shortcut_token_expires_at timestamp with time zone NOT NULL
);


--
-- Name: source_housing_companies; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.source_housing_companies (
    source_housing_company_id uuid NOT NULL,
    provider text NOT NULL,
    source_kind text NOT NULL,
    native_id text,
    raw_table text NOT NULL,
    raw_id text NOT NULL,
    url text,
    first_seen_at timestamp with time zone,
    last_seen_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: source_listings; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.source_listings (
    source_listing_id uuid NOT NULL,
    provider text NOT NULL,
    source_kind text NOT NULL,
    native_id text NOT NULL,
    canonical_source_id text NOT NULL,
    raw_table text NOT NULL,
    raw_id text NOT NULL,
    url text,
    payload_hash text,
    normalized_version integer DEFAULT 0 NOT NULL,
    normalized_at timestamp with time zone,
    first_seen_at timestamp with time zone,
    last_seen_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT source_listings_provider_check CHECK ((provider = ANY (ARRAY['shortcut'::text, 'frontdoor'::text]))),
    CONSTRAINT source_listings_source_kind_check CHECK ((source_kind = ANY (ARRAY['ad'::text, 'announcement'::text])))
);


--
-- Name: target_observations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.target_observations (
    target_observation_id uuid DEFAULT gen_random_uuid() NOT NULL,
    target_type text NOT NULL,
    target_id uuid NOT NULL,
    observation_key text NOT NULL,
    observation_kind text NOT NULL,
    severity text NOT NULL,
    direction text NOT NULL,
    value jsonb,
    text text,
    confidence double precision NOT NULL,
    source_type text NOT NULL,
    source_id uuid NOT NULL,
    evidence jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    superseded_at timestamp with time zone,
    CONSTRAINT target_observations_confidence_check CHECK (((confidence >= (0)::double precision) AND (confidence <= (1)::double precision))),
    CONSTRAINT target_observations_observation_kind_check CHECK ((observation_kind = ANY (ARRAY['risk'::text, 'opportunity'::text, 'inconsistency'::text, 'summary'::text, 'valuation_note'::text]))),
    CONSTRAINT target_observations_source_type_check CHECK ((source_type = ANY (ARRAY['source_listing'::text, 'source_housing_company'::text, 'document'::text, 'price_transaction'::text, 'dimension_claim'::text, 'manual'::text]))),
    CONSTRAINT target_observations_target_type_check CHECK ((target_type = ANY (ARRAY['listing'::text, 'unit'::text, 'building'::text, 'housing_company'::text, 'house'::text])))
);


--
-- Name: target_sources; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.target_sources (
    target_source_id uuid DEFAULT gen_random_uuid() NOT NULL,
    target_type text NOT NULL,
    target_id uuid NOT NULL,
    source_type text NOT NULL,
    source_id uuid NOT NULL,
    link_status text NOT NULL,
    link_method text NOT NULL,
    link_score integer DEFAULT 0 NOT NULL,
    link_reasons jsonb DEFAULT '{}'::jsonb NOT NULL,
    first_seen_at timestamp with time zone,
    last_seen_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT target_sources_link_method_check CHECK ((link_method = ANY (ARRAY['sync_auto'::text, 'source_match_auto'::text, 'document_match_auto'::text, 'manual'::text, 'backfill_auto'::text]))),
    CONSTRAINT target_sources_link_status_check CHECK ((link_status = ANY (ARRAY['confirmed'::text, 'candidate'::text, 'rejected'::text, 'superseded'::text]))),
    CONSTRAINT target_sources_source_type_check CHECK ((source_type = ANY (ARRAY['source_listing'::text, 'source_housing_company'::text, 'document'::text, 'price_transaction'::text, 'manual'::text]))),
    CONSTRAINT target_sources_target_type_check CHECK ((target_type = ANY (ARRAY['listing'::text, 'unit'::text, 'building'::text, 'housing_company'::text, 'house'::text])))
);


--
-- Name: units; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.units (
    unit_id uuid NOT NULL,
    housing_company_id uuid NOT NULL,
    physical_building_id uuid,
    identity_key text NOT NULL,
    address_norm text,
    apartment text,
    floor_level integer,
    area_m2 double precision,
    room_layout text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: user_devices; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_devices (
    user_device_uuid uuid DEFAULT gen_random_uuid() NOT NULL,
    user_device_name text,
    user_device_os text,
    user_device_app_version text,
    user_device_push_token text,
    user_device_push_token_updated_at timestamp with time zone,
    user_device_created_at timestamp with time zone DEFAULT now() NOT NULL,
    user_device_updated_at timestamp with time zone DEFAULT now() NOT NULL,
    user_device_last_seen_at timestamp with time zone DEFAULT now() NOT NULL,
    user_device_push_token_type text,
    user_device_id bigint NOT NULL,
    user_id bigint NOT NULL,
    user_device_push_is_development boolean DEFAULT false NOT NULL,
    user_device_push_token_invalidated_at timestamp with time zone,
    user_device_push_token_invalidated_reason text,
    user_device_model text,
    user_device_locale text,
    user_device_time_zone text,
    CONSTRAINT user_device_push_token_type_check CHECK (((user_device_push_token_type IS NULL) OR (user_device_push_token_type = 'apns'::text)))
);


--
-- Name: user_devices_user_device_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.user_devices ALTER COLUMN user_device_id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.user_devices_user_device_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: user_email_change_tokens; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_email_change_tokens (
    user_email_change_token_id bigint NOT NULL,
    user_email_change_token_uuid uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id bigint NOT NULL,
    user_email_change_target_email text CONSTRAINT user_email_change_tokens_user_email_change_target_emai_not_null NOT NULL,
    user_email_change_token_hash text NOT NULL,
    user_email_change_expires_at timestamp with time zone NOT NULL,
    user_email_change_consumed_at timestamp with time zone,
    user_email_change_created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT user_email_change_tokens_target_email_not_blank CHECK ((btrim(user_email_change_target_email) <> ''::text))
);


--
-- Name: user_email_change_tokens_user_email_change_token_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.user_email_change_tokens ALTER COLUMN user_email_change_token_id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.user_email_change_tokens_user_email_change_token_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: user_feature_flags; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_feature_flags (
    user_flag_enabled boolean NOT NULL,
    user_flag_created_at timestamp with time zone DEFAULT now() NOT NULL,
    flag_id bigint NOT NULL,
    user_id bigint NOT NULL
);


--
-- Name: user_identities; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_identities (
    user_identity_uuid uuid DEFAULT gen_random_uuid() NOT NULL,
    user_identity_external_id text NOT NULL,
    user_identity_email text,
    user_identity_email_verified boolean DEFAULT false NOT NULL,
    user_identity_data jsonb DEFAULT '{}'::jsonb,
    user_identity_created_at timestamp with time zone DEFAULT now() NOT NULL,
    user_identity_updated_at timestamp with time zone DEFAULT now() NOT NULL,
    user_identity_provider text NOT NULL,
    user_identity_id bigint NOT NULL,
    user_id bigint NOT NULL,
    CONSTRAINT user_identity_provider_check CHECK ((user_identity_provider = ANY (ARRAY['apple'::text, 'anonymous'::text, 'email'::text, 'google'::text, 'passkey'::text])))
);


--
-- Name: user_identities_user_identity_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.user_identities ALTER COLUMN user_identity_id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.user_identities_user_identity_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: user_passkeys; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_passkeys (
    user_passkey_id bigint NOT NULL,
    user_passkey_uuid uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id bigint NOT NULL,
    user_identity_id bigint NOT NULL,
    user_passkey_credential_id bytea NOT NULL,
    user_passkey_credential_id_b64url text NOT NULL,
    user_passkey_public_key bytea NOT NULL,
    user_passkey_attestation_type text NOT NULL,
    user_passkey_transports text[] DEFAULT '{}'::text[] NOT NULL,
    user_passkey_user_handle bytea NOT NULL,
    user_passkey_sign_count bigint DEFAULT 0 NOT NULL,
    user_passkey_flags integer,
    user_passkey_aaguid uuid,
    user_passkey_name text,
    user_passkey_backup_eligible boolean,
    user_passkey_backup_state boolean,
    user_passkey_last_used_at timestamp with time zone,
    user_passkey_created_at timestamp with time zone DEFAULT now() NOT NULL,
    user_passkey_updated_at timestamp with time zone DEFAULT now() NOT NULL,
    user_passkey_revoked_at timestamp with time zone
);


--
-- Name: user_passkeys_user_passkey_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.user_passkeys ALTER COLUMN user_passkey_id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.user_passkeys_user_passkey_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: user_roles; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_roles (
    user_role_created_at timestamp with time zone DEFAULT now() NOT NULL,
    role_id bigint NOT NULL,
    user_id bigint NOT NULL
);


--
-- Name: users; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.users (
    user_uuid uuid DEFAULT gen_random_uuid() NOT NULL,
    user_first_name text,
    user_last_name text,
    user_username text,
    user_name_display public.enum__name_display DEFAULT 'username'::public.enum__name_display,
    user_is_private boolean DEFAULT false NOT NULL,
    user_is_onboarded boolean DEFAULT false NOT NULL,
    user_joined_at timestamp with time zone DEFAULT now() NOT NULL,
    user_search text GENERATED ALWAYS AS (((user_username || COALESCE(user_first_name, ''::text)) || COALESCE(user_last_name, ''::text))) STORED,
    user_preferred_name text GENERATED ALWAYS AS (
CASE
    WHEN ((user_name_display = 'full_name'::public.enum__name_display) AND (user_first_name IS NOT NULL) AND (user_last_name IS NOT NULL)) THEN ((user_first_name || ' '::text) || user_last_name)
    ELSE user_username
END) STORED,
    user_id bigint NOT NULL,
    user_email text,
    user_has_seen_passkey_onboarding boolean DEFAULT false NOT NULL,
    CONSTRAINT users_user_username_length CHECK (((user_username IS NULL) OR ((char_length(user_username) >= 2) AND (char_length(user_username) <= 16))))
);


--
-- Name: users_user_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.users ALTER COLUMN user_id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.users_user_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);



--
-- Name: kv_store; Type: TABLE; Schema: runtime; Owner: -
--

CREATE TABLE runtime.kv_store (
    kv_key text NOT NULL,
    kv_value bytea NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: refresh_tokens refresh_token_id; Type: DEFAULT; Schema: auth; Owner: -
--

ALTER TABLE ONLY auth.refresh_tokens ALTER COLUMN refresh_token_id SET DEFAULT nextval('auth.refresh_tokens_refresh_token_id_seq'::regclass);


--
-- Name: c_canonical_db c_canonical_db_pkey; Type: CONSTRAINT; Schema: absurd; Owner: -
--

ALTER TABLE ONLY absurd.c_canonical_db
    ADD CONSTRAINT c_canonical_db_pkey PRIMARY KEY (task_id, checkpoint_name);


--
-- Name: c_canonical_llm c_canonical_llm_pkey; Type: CONSTRAINT; Schema: absurd; Owner: -
--

ALTER TABLE ONLY absurd.c_canonical_llm
    ADD CONSTRAINT c_canonical_llm_pkey PRIMARY KEY (task_id, checkpoint_name);


--
-- Name: c_frontdoor c_frontdoor_pkey; Type: CONSTRAINT; Schema: absurd; Owner: -
--

ALTER TABLE ONLY absurd.c_frontdoor
    ADD CONSTRAINT c_frontdoor_pkey PRIMARY KEY (task_id, checkpoint_name);


--
-- Name: c_postal c_postal_pkey; Type: CONSTRAINT; Schema: absurd; Owner: -
--

ALTER TABLE ONLY absurd.c_postal
    ADD CONSTRAINT c_postal_pkey PRIMARY KEY (task_id, checkpoint_name);


--
-- Name: c_prices c_prices_pkey; Type: CONSTRAINT; Schema: absurd; Owner: -
--

ALTER TABLE ONLY absurd.c_prices
    ADD CONSTRAINT c_prices_pkey PRIMARY KEY (task_id, checkpoint_name);


--
-- Name: c_shortcut_api c_shortcut_api_pkey; Type: CONSTRAINT; Schema: absurd; Owner: -
--

ALTER TABLE ONLY absurd.c_shortcut_api
    ADD CONSTRAINT c_shortcut_api_pkey PRIMARY KEY (task_id, checkpoint_name);


--
-- Name: c_shortcut_scraper c_shortcut_scraper_pkey; Type: CONSTRAINT; Schema: absurd; Owner: -
--

ALTER TABLE ONLY absurd.c_shortcut_scraper
    ADD CONSTRAINT c_shortcut_scraper_pkey PRIMARY KEY (task_id, checkpoint_name);


--
-- Name: e_canonical_db e_canonical_db_pkey; Type: CONSTRAINT; Schema: absurd; Owner: -
--

ALTER TABLE ONLY absurd.e_canonical_db
    ADD CONSTRAINT e_canonical_db_pkey PRIMARY KEY (event_name);


--
-- Name: e_canonical_llm e_canonical_llm_pkey; Type: CONSTRAINT; Schema: absurd; Owner: -
--

ALTER TABLE ONLY absurd.e_canonical_llm
    ADD CONSTRAINT e_canonical_llm_pkey PRIMARY KEY (event_name);


--
-- Name: e_frontdoor e_frontdoor_pkey; Type: CONSTRAINT; Schema: absurd; Owner: -
--

ALTER TABLE ONLY absurd.e_frontdoor
    ADD CONSTRAINT e_frontdoor_pkey PRIMARY KEY (event_name);


--
-- Name: e_postal e_postal_pkey; Type: CONSTRAINT; Schema: absurd; Owner: -
--

ALTER TABLE ONLY absurd.e_postal
    ADD CONSTRAINT e_postal_pkey PRIMARY KEY (event_name);


--
-- Name: e_prices e_prices_pkey; Type: CONSTRAINT; Schema: absurd; Owner: -
--

ALTER TABLE ONLY absurd.e_prices
    ADD CONSTRAINT e_prices_pkey PRIMARY KEY (event_name);


--
-- Name: e_shortcut_api e_shortcut_api_pkey; Type: CONSTRAINT; Schema: absurd; Owner: -
--

ALTER TABLE ONLY absurd.e_shortcut_api
    ADD CONSTRAINT e_shortcut_api_pkey PRIMARY KEY (event_name);


--
-- Name: e_shortcut_scraper e_shortcut_scraper_pkey; Type: CONSTRAINT; Schema: absurd; Owner: -
--

ALTER TABLE ONLY absurd.e_shortcut_scraper
    ADD CONSTRAINT e_shortcut_scraper_pkey PRIMARY KEY (event_name);


--
-- Name: queues queues_pkey; Type: CONSTRAINT; Schema: absurd; Owner: -
--

ALTER TABLE ONLY absurd.queues
    ADD CONSTRAINT queues_pkey PRIMARY KEY (queue_name);


--
-- Name: r_canonical_db r_canonical_db_pkey; Type: CONSTRAINT; Schema: absurd; Owner: -
--

ALTER TABLE ONLY absurd.r_canonical_db
    ADD CONSTRAINT r_canonical_db_pkey PRIMARY KEY (run_id);


--
-- Name: r_canonical_llm r_canonical_llm_pkey; Type: CONSTRAINT; Schema: absurd; Owner: -
--

ALTER TABLE ONLY absurd.r_canonical_llm
    ADD CONSTRAINT r_canonical_llm_pkey PRIMARY KEY (run_id);


--
-- Name: r_frontdoor r_frontdoor_pkey; Type: CONSTRAINT; Schema: absurd; Owner: -
--

ALTER TABLE ONLY absurd.r_frontdoor
    ADD CONSTRAINT r_frontdoor_pkey PRIMARY KEY (run_id);


--
-- Name: r_postal r_postal_pkey; Type: CONSTRAINT; Schema: absurd; Owner: -
--

ALTER TABLE ONLY absurd.r_postal
    ADD CONSTRAINT r_postal_pkey PRIMARY KEY (run_id);


--
-- Name: r_prices r_prices_pkey; Type: CONSTRAINT; Schema: absurd; Owner: -
--

ALTER TABLE ONLY absurd.r_prices
    ADD CONSTRAINT r_prices_pkey PRIMARY KEY (run_id);


--
-- Name: r_shortcut_api r_shortcut_api_pkey; Type: CONSTRAINT; Schema: absurd; Owner: -
--

ALTER TABLE ONLY absurd.r_shortcut_api
    ADD CONSTRAINT r_shortcut_api_pkey PRIMARY KEY (run_id);


--
-- Name: r_shortcut_scraper r_shortcut_scraper_pkey; Type: CONSTRAINT; Schema: absurd; Owner: -
--

ALTER TABLE ONLY absurd.r_shortcut_scraper
    ADD CONSTRAINT r_shortcut_scraper_pkey PRIMARY KEY (run_id);


--
-- Name: t_canonical_db t_canonical_db_idempotency_key_key; Type: CONSTRAINT; Schema: absurd; Owner: -
--

ALTER TABLE ONLY absurd.t_canonical_db
    ADD CONSTRAINT t_canonical_db_idempotency_key_key UNIQUE (idempotency_key);


--
-- Name: t_canonical_db t_canonical_db_pkey; Type: CONSTRAINT; Schema: absurd; Owner: -
--

ALTER TABLE ONLY absurd.t_canonical_db
    ADD CONSTRAINT t_canonical_db_pkey PRIMARY KEY (task_id);


--
-- Name: t_canonical_llm t_canonical_llm_idempotency_key_key; Type: CONSTRAINT; Schema: absurd; Owner: -
--

ALTER TABLE ONLY absurd.t_canonical_llm
    ADD CONSTRAINT t_canonical_llm_idempotency_key_key UNIQUE (idempotency_key);


--
-- Name: t_canonical_llm t_canonical_llm_pkey; Type: CONSTRAINT; Schema: absurd; Owner: -
--

ALTER TABLE ONLY absurd.t_canonical_llm
    ADD CONSTRAINT t_canonical_llm_pkey PRIMARY KEY (task_id);


--
-- Name: t_frontdoor t_frontdoor_idempotency_key_key; Type: CONSTRAINT; Schema: absurd; Owner: -
--

ALTER TABLE ONLY absurd.t_frontdoor
    ADD CONSTRAINT t_frontdoor_idempotency_key_key UNIQUE (idempotency_key);


--
-- Name: t_frontdoor t_frontdoor_pkey; Type: CONSTRAINT; Schema: absurd; Owner: -
--

ALTER TABLE ONLY absurd.t_frontdoor
    ADD CONSTRAINT t_frontdoor_pkey PRIMARY KEY (task_id);


--
-- Name: t_postal t_postal_idempotency_key_key; Type: CONSTRAINT; Schema: absurd; Owner: -
--

ALTER TABLE ONLY absurd.t_postal
    ADD CONSTRAINT t_postal_idempotency_key_key UNIQUE (idempotency_key);


--
-- Name: t_postal t_postal_pkey; Type: CONSTRAINT; Schema: absurd; Owner: -
--

ALTER TABLE ONLY absurd.t_postal
    ADD CONSTRAINT t_postal_pkey PRIMARY KEY (task_id);


--
-- Name: t_prices t_prices_idempotency_key_key; Type: CONSTRAINT; Schema: absurd; Owner: -
--

ALTER TABLE ONLY absurd.t_prices
    ADD CONSTRAINT t_prices_idempotency_key_key UNIQUE (idempotency_key);


--
-- Name: t_prices t_prices_pkey; Type: CONSTRAINT; Schema: absurd; Owner: -
--

ALTER TABLE ONLY absurd.t_prices
    ADD CONSTRAINT t_prices_pkey PRIMARY KEY (task_id);


--
-- Name: t_shortcut_api t_shortcut_api_idempotency_key_key; Type: CONSTRAINT; Schema: absurd; Owner: -
--

ALTER TABLE ONLY absurd.t_shortcut_api
    ADD CONSTRAINT t_shortcut_api_idempotency_key_key UNIQUE (idempotency_key);


--
-- Name: t_shortcut_api t_shortcut_api_pkey; Type: CONSTRAINT; Schema: absurd; Owner: -
--

ALTER TABLE ONLY absurd.t_shortcut_api
    ADD CONSTRAINT t_shortcut_api_pkey PRIMARY KEY (task_id);


--
-- Name: t_shortcut_scraper t_shortcut_scraper_idempotency_key_key; Type: CONSTRAINT; Schema: absurd; Owner: -
--

ALTER TABLE ONLY absurd.t_shortcut_scraper
    ADD CONSTRAINT t_shortcut_scraper_idempotency_key_key UNIQUE (idempotency_key);


--
-- Name: t_shortcut_scraper t_shortcut_scraper_pkey; Type: CONSTRAINT; Schema: absurd; Owner: -
--

ALTER TABLE ONLY absurd.t_shortcut_scraper
    ADD CONSTRAINT t_shortcut_scraper_pkey PRIMARY KEY (task_id);


--
-- Name: w_canonical_db w_canonical_db_pkey; Type: CONSTRAINT; Schema: absurd; Owner: -
--

ALTER TABLE ONLY absurd.w_canonical_db
    ADD CONSTRAINT w_canonical_db_pkey PRIMARY KEY (run_id, step_name);


--
-- Name: w_canonical_llm w_canonical_llm_pkey; Type: CONSTRAINT; Schema: absurd; Owner: -
--

ALTER TABLE ONLY absurd.w_canonical_llm
    ADD CONSTRAINT w_canonical_llm_pkey PRIMARY KEY (run_id, step_name);


--
-- Name: w_frontdoor w_frontdoor_pkey; Type: CONSTRAINT; Schema: absurd; Owner: -
--

ALTER TABLE ONLY absurd.w_frontdoor
    ADD CONSTRAINT w_frontdoor_pkey PRIMARY KEY (run_id, step_name);


--
-- Name: w_postal w_postal_pkey; Type: CONSTRAINT; Schema: absurd; Owner: -
--

ALTER TABLE ONLY absurd.w_postal
    ADD CONSTRAINT w_postal_pkey PRIMARY KEY (run_id, step_name);


--
-- Name: w_prices w_prices_pkey; Type: CONSTRAINT; Schema: absurd; Owner: -
--

ALTER TABLE ONLY absurd.w_prices
    ADD CONSTRAINT w_prices_pkey PRIMARY KEY (run_id, step_name);


--
-- Name: w_shortcut_api w_shortcut_api_pkey; Type: CONSTRAINT; Schema: absurd; Owner: -
--

ALTER TABLE ONLY absurd.w_shortcut_api
    ADD CONSTRAINT w_shortcut_api_pkey PRIMARY KEY (run_id, step_name);


--
-- Name: w_shortcut_scraper w_shortcut_scraper_pkey; Type: CONSTRAINT; Schema: absurd; Owner: -
--

ALTER TABLE ONLY absurd.w_shortcut_scraper
    ADD CONSTRAINT w_shortcut_scraper_pkey PRIMARY KEY (run_id, step_name);


--
-- Name: devices devices_pkey; Type: CONSTRAINT; Schema: auth; Owner: -
--

ALTER TABLE ONLY auth.devices
    ADD CONSTRAINT devices_pkey PRIMARY KEY (device_id);


--
-- Name: feature_flags feature_flags_flag_name_key; Type: CONSTRAINT; Schema: auth; Owner: -
--

ALTER TABLE ONLY auth.feature_flags
    ADD CONSTRAINT feature_flags_flag_name_key UNIQUE (flag_name);


--
-- Name: feature_flags feature_flags_pkey; Type: CONSTRAINT; Schema: auth; Owner: -
--

ALTER TABLE ONLY auth.feature_flags
    ADD CONSTRAINT feature_flags_pkey PRIMARY KEY (flag_id);


--
-- Name: identities identities_pkey; Type: CONSTRAINT; Schema: auth; Owner: -
--

ALTER TABLE ONLY auth.identities
    ADD CONSTRAINT identities_pkey PRIMARY KEY (identity_id);


--
-- Name: identities identities_provider_external_id_unique; Type: CONSTRAINT; Schema: auth; Owner: -
--

ALTER TABLE ONLY auth.identities
    ADD CONSTRAINT identities_provider_external_id_unique UNIQUE (identity_provider, identity_external_id);


--
-- Name: refresh_tokens refresh_tokens_pkey; Type: CONSTRAINT; Schema: auth; Owner: -
--

ALTER TABLE ONLY auth.refresh_tokens
    ADD CONSTRAINT refresh_tokens_pkey PRIMARY KEY (refresh_token_id);


--
-- Name: role_feature_flags role_feature_flags_pkey; Type: CONSTRAINT; Schema: auth; Owner: -
--

ALTER TABLE ONLY auth.role_feature_flags
    ADD CONSTRAINT role_feature_flags_pkey PRIMARY KEY (role_id, flag_id);


--
-- Name: roles roles_pkey; Type: CONSTRAINT; Schema: auth; Owner: -
--

ALTER TABLE ONLY auth.roles
    ADD CONSTRAINT roles_pkey PRIMARY KEY (role_id);


--
-- Name: roles roles_role_name_key; Type: CONSTRAINT; Schema: auth; Owner: -
--

ALTER TABLE ONLY auth.roles
    ADD CONSTRAINT roles_role_name_key UNIQUE (role_name);


--
-- Name: sessions sessions_pkey; Type: CONSTRAINT; Schema: auth; Owner: -
--

ALTER TABLE ONLY auth.sessions
    ADD CONSTRAINT sessions_pkey PRIMARY KEY (session_id);


--
-- Name: user_feature_flags user_feature_flags_pkey; Type: CONSTRAINT; Schema: auth; Owner: -
--

ALTER TABLE ONLY auth.user_feature_flags
    ADD CONSTRAINT user_feature_flags_pkey PRIMARY KEY (user_id, flag_id);


--
-- Name: user_roles user_roles_pkey; Type: CONSTRAINT; Schema: auth; Owner: -
--

ALTER TABLE ONLY auth.user_roles
    ADD CONSTRAINT user_roles_pkey PRIMARY KEY (user_id, role_id);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: auth; Owner: -
--

ALTER TABLE ONLY auth.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (user_id);


--
-- Name: auth_signup_email_tokens auth_signup_email_tokens_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.auth_signup_email_tokens
    ADD CONSTRAINT auth_signup_email_tokens_pkey PRIMARY KEY (auth_signup_email_token_id);


--
-- Name: auth_signup_email_tokens auth_signup_email_tokens_token_hash_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.auth_signup_email_tokens
    ADD CONSTRAINT auth_signup_email_tokens_token_hash_key UNIQUE (auth_signup_email_token_hash);


--
-- Name: auth_signup_email_tokens auth_signup_email_tokens_uuid_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.auth_signup_email_tokens
    ADD CONSTRAINT auth_signup_email_tokens_uuid_key UNIQUE (auth_signup_email_token_uuid);


--
-- Name: auth_signup_tickets auth_signup_tickets_hash_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.auth_signup_tickets
    ADD CONSTRAINT auth_signup_tickets_hash_key UNIQUE (auth_signup_ticket_hash);


--
-- Name: auth_signup_tickets auth_signup_tickets_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.auth_signup_tickets
    ADD CONSTRAINT auth_signup_tickets_pkey PRIMARY KEY (auth_signup_ticket_id);


--
-- Name: auth_signup_tickets auth_signup_tickets_uuid_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.auth_signup_tickets
    ADD CONSTRAINT auth_signup_tickets_uuid_key UNIQUE (auth_signup_ticket_uuid);


--
-- Name: auth_webauthn_challenges auth_webauthn_challenges_auth_webauthn_challenge_uuid_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.auth_webauthn_challenges
    ADD CONSTRAINT auth_webauthn_challenges_auth_webauthn_challenge_uuid_key UNIQUE (auth_webauthn_challenge_uuid);


--
-- Name: auth_webauthn_challenges auth_webauthn_challenges_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.auth_webauthn_challenges
    ADD CONSTRAINT auth_webauthn_challenges_pkey PRIMARY KEY (auth_webauthn_challenge_id);


--
-- Name: device_sessions device_sessions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_sessions
    ADD CONSTRAINT device_sessions_pkey PRIMARY KEY (device_session_id);


--
-- Name: device_sessions device_sessions_uuid_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_sessions
    ADD CONSTRAINT device_sessions_uuid_key UNIQUE (device_session_uuid);


--
-- Name: dimension_claims dimension_claims_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.dimension_claims
    ADD CONSTRAINT dimension_claims_pkey PRIMARY KEY (property_dimension_claim_id);


--
-- Name: dimension_profiles dimension_profiles_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.dimension_profiles
    ADD CONSTRAINT dimension_profiles_pkey PRIMARY KEY (target_type, target_id);


--
-- Name: dimension_values dimension_values_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.dimension_values
    ADD CONSTRAINT dimension_values_pkey PRIMARY KEY (target_type, target_id, dimension_key);


--
-- Name: energy_efficiency_aliases energy_efficiency_aliases_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.energy_efficiency_aliases
    ADD CONSTRAINT energy_efficiency_aliases_pkey PRIMARY KEY (energy_efficiency_alias);


--
-- Name: feature_flags feature_flags_flag_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.feature_flags
    ADD CONSTRAINT feature_flags_flag_name_key UNIQUE (flag_name);


--
-- Name: feature_flags feature_flags_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.feature_flags
    ADD CONSTRAINT feature_flags_pkey PRIMARY KEY (flag_id);


--
-- Name: feature_flags feature_flags_uuid_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.feature_flags
    ADD CONSTRAINT feature_flags_uuid_key UNIQUE (flag_uuid);


--
-- Name: frontdoor_ads frontdoor_ads_frontdoor_ads_external_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.frontdoor_ads
    ADD CONSTRAINT frontdoor_ads_frontdoor_ads_external_id_key UNIQUE (frontdoor_ad_external_id);


--
-- Name: frontdoor_ads frontdoor_ads_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.frontdoor_ads
    ADD CONSTRAINT frontdoor_ads_pkey PRIMARY KEY (frontdoor_ad_id);


--
-- Name: frontdoor_building_announcements frontdoor_building_announcements_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.frontdoor_building_announcements
    ADD CONSTRAINT frontdoor_building_announcements_pkey PRIMARY KEY (frontdoor_building_announcement_id);


--
-- Name: frontdoor_buildings frontdoor_buildings_frontdoor_buildings_housing_company_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.frontdoor_buildings
    ADD CONSTRAINT frontdoor_buildings_frontdoor_buildings_housing_company_id_key UNIQUE (frontdoor_building_housing_company_id);


--
-- Name: frontdoor_buildings frontdoor_buildings_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.frontdoor_buildings
    ADD CONSTRAINT frontdoor_buildings_pkey PRIMARY KEY (frontdoor_building_id);


--
-- Name: houses houses_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.houses
    ADD CONSTRAINT houses_pkey PRIMARY KEY (house_id);


--
-- Name: housing_company_merge_decisions housing_company_merge_decisions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.housing_company_merge_decisions
    ADD CONSTRAINT housing_company_merge_decisions_pkey PRIMARY KEY (housing_company_merge_decision_id);


--
-- Name: listings listings_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.listings
    ADD CONSTRAINT listings_pkey PRIMARY KEY (listing_id);


--
-- Name: oauth_authorization_codes oauth_authorization_codes_oauth_authorization_code_code_has_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.oauth_authorization_codes
    ADD CONSTRAINT oauth_authorization_codes_oauth_authorization_code_code_has_key UNIQUE (oauth_authorization_code_code_hash);


--
-- Name: oauth_authorization_codes oauth_authorization_codes_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.oauth_authorization_codes
    ADD CONSTRAINT oauth_authorization_codes_pkey PRIMARY KEY (oauth_authorization_code_id);


--
-- Name: oauth_authorization_handoffs oauth_authorization_handoffs_oauth_authorization_handoff_to_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.oauth_authorization_handoffs
    ADD CONSTRAINT oauth_authorization_handoffs_oauth_authorization_handoff_to_key UNIQUE (oauth_authorization_handoff_token_hash);


--
-- Name: oauth_authorization_handoffs oauth_authorization_handoffs_oauth_authorization_handoff_us_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.oauth_authorization_handoffs
    ADD CONSTRAINT oauth_authorization_handoffs_oauth_authorization_handoff_us_key UNIQUE (oauth_authorization_handoff_user_code);


--
-- Name: oauth_authorization_handoffs oauth_authorization_handoffs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.oauth_authorization_handoffs
    ADD CONSTRAINT oauth_authorization_handoffs_pkey PRIMARY KEY (oauth_authorization_handoff_id);


--
-- Name: oauth_device_authorizations oauth_device_authorizations_device_code_hash_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.oauth_device_authorizations
    ADD CONSTRAINT oauth_device_authorizations_device_code_hash_key UNIQUE (oauth_device_authorization_device_code_hash);


--
-- Name: oauth_device_authorizations oauth_device_authorizations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.oauth_device_authorizations
    ADD CONSTRAINT oauth_device_authorizations_pkey PRIMARY KEY (oauth_device_authorization_id);


--
-- Name: oauth_device_authorizations oauth_device_authorizations_user_code_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.oauth_device_authorizations
    ADD CONSTRAINT oauth_device_authorizations_user_code_key UNIQUE (oauth_device_authorization_user_code);


--
-- Name: oauth_dynamic_clients oauth_dynamic_clients_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.oauth_dynamic_clients
    ADD CONSTRAINT oauth_dynamic_clients_pkey PRIMARY KEY (oauth_dynamic_client_id);


--
-- Name: oauth_refresh_tokens oauth_refresh_tokens_oauth_refresh_token_token_hash_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.oauth_refresh_tokens
    ADD CONSTRAINT oauth_refresh_tokens_oauth_refresh_token_token_hash_key UNIQUE (oauth_refresh_token_token_hash);


--
-- Name: oauth_refresh_tokens oauth_refresh_tokens_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.oauth_refresh_tokens
    ADD CONSTRAINT oauth_refresh_tokens_pkey PRIMARY KEY (oauth_refresh_token_id);


--
-- Name: personal_access_tokens personal_access_tokens_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.personal_access_tokens
    ADD CONSTRAINT personal_access_tokens_pkey PRIMARY KEY (personal_access_token_id);


--
-- Name: physical_buildings physical_buildings_physical_building_identity_key_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.physical_buildings
    ADD CONSTRAINT physical_buildings_physical_building_identity_key_key UNIQUE (physical_building_identity_key);


--
-- Name: physical_buildings physical_buildings_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.physical_buildings
    ADD CONSTRAINT physical_buildings_pkey PRIMARY KEY (physical_building_id);


--
-- Name: postal_ad_areas postal_ad_areas_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.postal_ad_areas
    ADD CONSTRAINT postal_ad_areas_pkey PRIMARY KEY (postal_ad_area_id);


--
-- Name: postal_ad_areas postal_ad_areas_postal_ad_areas_code_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.postal_ad_areas
    ADD CONSTRAINT postal_ad_areas_postal_ad_areas_code_key UNIQUE (postal_ad_area_code);


--
-- Name: postal_municipalities postal_municipalities_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.postal_municipalities
    ADD CONSTRAINT postal_municipalities_pkey PRIMARY KEY (postal_municipality_id);


--
-- Name: postal_municipalities postal_municipalities_postal_municipalities_code_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.postal_municipalities
    ADD CONSTRAINT postal_municipalities_postal_municipalities_code_key UNIQUE (postal_municipality_code);


--
-- Name: postal_postal_codes postal_postal_codes_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.postal_postal_codes
    ADD CONSTRAINT postal_postal_codes_pkey PRIMARY KEY (postal_postal_code_id);


--
-- Name: postal_postal_codes postal_postal_codes_postal_postal_codes_code_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.postal_postal_codes
    ADD CONSTRAINT postal_postal_codes_postal_postal_codes_code_key UNIQUE (postal_postal_code_code);


--
-- Name: price_links price_links_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.price_links
    ADD CONSTRAINT price_links_pkey PRIMARY KEY (price_link_id);


--
-- Name: prices_cities prices_cities_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.prices_cities
    ADD CONSTRAINT prices_cities_pkey PRIMARY KEY (prices_city_id);


--
-- Name: prices_cities prices_cities_prices_cities_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.prices_cities
    ADD CONSTRAINT prices_cities_prices_cities_name_key UNIQUE (prices_city_name);


--
-- Name: prices_neighborhoods prices_neighborhoods_name_city_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.prices_neighborhoods
    ADD CONSTRAINT prices_neighborhoods_name_city_unique UNIQUE (prices_neighborhood_name, prices_city_id);


--
-- Name: prices_neighborhoods prices_neighborhoods_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.prices_neighborhoods
    ADD CONSTRAINT prices_neighborhoods_pkey PRIMARY KEY (prices_neighborhood_id);


--
-- Name: prices_postal_codes prices_postal_codes_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.prices_postal_codes
    ADD CONSTRAINT prices_postal_codes_pkey PRIMARY KEY (prices_postal_code_id);


--
-- Name: prices_postal_codes prices_postal_codes_prices_postal_codes_code_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.prices_postal_codes
    ADD CONSTRAINT prices_postal_codes_prices_postal_codes_code_key UNIQUE (prices_postal_code_code);


--
-- Name: prices_transactions prices_transaction_unique_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.prices_transactions
    ADD CONSTRAINT prices_transaction_unique_key UNIQUE NULLS NOT DISTINCT (prices_neighborhood_id, prices_transaction_description, prices_transaction_type, prices_transaction_area, prices_transaction_price, prices_transaction_price_per_square_meter, prices_transaction_build_year, prices_transaction_floor, prices_transaction_elevator, prices_transaction_condition, prices_transaction_plot, prices_transaction_energy_class, prices_transaction_category);


--
-- Name: prices_transactions prices_transactions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.prices_transactions
    ADD CONSTRAINT prices_transactions_pkey PRIMARY KEY (prices_transaction_id);


--
-- Name: housing_companies property_buildings_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.housing_companies
    ADD CONSTRAINT property_buildings_pkey PRIMARY KEY (housing_company_id);


--
-- Name: housing_companies property_buildings_property_building_identity_key_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.housing_companies
    ADD CONSTRAINT property_buildings_property_building_identity_key_key UNIQUE (housing_company_identity_key);


--
-- Name: property_dimension_catalog property_dimension_catalog_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.property_dimension_catalog
    ADD CONSTRAINT property_dimension_catalog_pkey PRIMARY KEY (dimension_key);


--
-- Name: property_dimension_dirty_targets property_dimension_dirty_targets_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.property_dimension_dirty_targets
    ADD CONSTRAINT property_dimension_dirty_targets_pkey PRIMARY KEY (target_type, target_id);


--
-- Name: property_dimension_projection_runs property_dimension_projection_runs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.property_dimension_projection_runs
    ADD CONSTRAINT property_dimension_projection_runs_pkey PRIMARY KEY (property_dimension_projection_run_id);


--
-- Name: property_dimension_resolution_policies property_dimension_resolution_policies_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.property_dimension_resolution_policies
    ADD CONSTRAINT property_dimension_resolution_policies_pkey PRIMARY KEY (dimension_key);


--
-- Name: property_document_extraction_runs property_document_extraction_runs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.property_document_extraction_runs
    ADD CONSTRAINT property_document_extraction_runs_pkey PRIMARY KEY (property_document_extraction_run_id);


--
-- Name: property_document_extractions property_document_extractions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.property_document_extractions
    ADD CONSTRAINT property_document_extractions_pkey PRIMARY KEY (property_document_extraction_id);


--
-- Name: property_documents property_documents_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.property_documents
    ADD CONSTRAINT property_documents_pkey PRIMARY KEY (property_document_id);


--
-- Name: property_houses property_houses_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.property_houses
    ADD CONSTRAINT property_houses_pkey PRIMARY KEY (property_house_id);


--
-- Name: property_houses property_houses_property_house_identity_key_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.property_houses
    ADD CONSTRAINT property_houses_property_house_identity_key_key UNIQUE (property_house_identity_key);


--
-- Name: property_offering_merge_decisions property_offering_merge_decisions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.property_offering_merge_decisions
    ADD CONSTRAINT property_offering_merge_decisions_pkey PRIMARY KEY (property_offering_merge_decision_id);


--
-- Name: property_offerings property_offerings_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.property_offerings
    ADD CONSTRAINT property_offerings_pkey PRIMARY KEY (property_offering_id);


--
-- Name: property_offerings property_offerings_property_offering_identity_key_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.property_offerings
    ADD CONSTRAINT property_offerings_property_offering_identity_key_key UNIQUE (property_offering_identity_key);


--
-- Name: property_renovation_events property_renovation_events_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.property_renovation_events
    ADD CONSTRAINT property_renovation_events_pkey PRIMARY KEY (property_renovation_event_id);


--
-- Name: property_source_offering_renovations property_source_offering_renovations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.property_source_offering_renovations
    ADD CONSTRAINT property_source_offering_renovations_pkey PRIMARY KEY (property_source_offering_renovation_id);


--
-- Name: property_units property_units_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.property_units
    ADD CONSTRAINT property_units_pkey PRIMARY KEY (property_unit_id);


--
-- Name: property_units property_units_property_unit_identity_key_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.property_units
    ADD CONSTRAINT property_units_property_unit_identity_key_key UNIQUE (property_unit_identity_key);


--
-- Name: role_feature_flags role_feature_flags_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.role_feature_flags
    ADD CONSTRAINT role_feature_flags_pkey PRIMARY KEY (role_id, flag_id);


--
-- Name: roles roles_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.roles
    ADD CONSTRAINT roles_pkey PRIMARY KEY (role_id);


--
-- Name: roles roles_role_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.roles
    ADD CONSTRAINT roles_role_name_key UNIQUE (role_name);


--
-- Name: roles roles_uuid_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.roles
    ADD CONSTRAINT roles_uuid_key UNIQUE (role_uuid);


--
-- Name: sale_listing_plot_type_aliases sale_listing_plot_type_aliases_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sale_listing_plot_type_aliases
    ADD CONSTRAINT sale_listing_plot_type_aliases_pkey PRIMARY KEY (sale_listing_plot_type_alias);


--
-- Name: sale_listing_prices_transaction_match_candidates sale_listing_prices_transaction_match_candidate_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sale_listing_prices_transaction_match_candidates
    ADD CONSTRAINT sale_listing_prices_transaction_match_candidate_unique UNIQUE (sale_listing_prices_transaction_match_run_id, sale_listing_id, prices_transaction_id);


--
-- Name: sale_listing_prices_transaction_match_candidates sale_listing_prices_transaction_match_candidates_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sale_listing_prices_transaction_match_candidates
    ADD CONSTRAINT sale_listing_prices_transaction_match_candidates_pkey PRIMARY KEY (sale_listing_prices_transaction_match_candidate_id);


--
-- Name: sale_listing_prices_transaction_match_runs sale_listing_prices_transaction_match_runs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sale_listing_prices_transaction_match_runs
    ADD CONSTRAINT sale_listing_prices_transaction_match_runs_pkey PRIMARY KEY (sale_listing_prices_transaction_match_run_id);


--
-- Name: sale_listing_property_type_aliases sale_listing_property_type_aliases_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sale_listing_property_type_aliases
    ADD CONSTRAINT sale_listing_property_type_aliases_pkey PRIMARY KEY (sale_listing_property_type_alias);


--
-- Name: sale_listing_room_category_aliases sale_listing_room_category_aliases_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sale_listing_room_category_aliases
    ADD CONSTRAINT sale_listing_room_category_aliases_pkey PRIMARY KEY (sale_listing_room_category_alias);


--
-- Name: property_source_offerings sale_listings_canonical_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.property_source_offerings
    ADD CONSTRAINT sale_listings_canonical_id_key UNIQUE (sale_listing_canonical_id);


--
-- Name: property_source_offerings sale_listings_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.property_source_offerings
    ADD CONSTRAINT sale_listings_pkey PRIMARY KEY (sale_listing_id);



--
-- Name: shortcut_ads shortcut_ads_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.shortcut_ads
    ADD CONSTRAINT shortcut_ads_pkey PRIMARY KEY (shortcut_ad_id);


--
-- Name: shortcut_building_listings shortcut_building_listings_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.shortcut_building_listings
    ADD CONSTRAINT shortcut_building_listings_pkey PRIMARY KEY (shortcut_building_listing_id);


--
-- Name: shortcut_building_rentals shortcut_building_rentals_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.shortcut_building_rentals
    ADD CONSTRAINT shortcut_building_rentals_pkey PRIMARY KEY (shortcut_building_rental_id);


--
-- Name: shortcut_buildings shortcut_buildings_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.shortcut_buildings
    ADD CONSTRAINT shortcut_buildings_pkey PRIMARY KEY (shortcut_building_id);


--
-- Name: shortcut_buildings shortcut_buildings_shortcut_buildings_external_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.shortcut_buildings
    ADD CONSTRAINT shortcut_buildings_shortcut_buildings_external_id_key UNIQUE (shortcut_building_external_id);


--
-- Name: shortcut_tokens shortcut_token_cuid_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.shortcut_tokens
    ADD CONSTRAINT shortcut_token_cuid_key UNIQUE (shortcut_token_cuid);


--
-- Name: shortcut_tokens shortcut_tokens_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.shortcut_tokens
    ADD CONSTRAINT shortcut_tokens_pkey PRIMARY KEY (shortcut_token_id);


--
-- Name: source_housing_companies source_housing_companies_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.source_housing_companies
    ADD CONSTRAINT source_housing_companies_pkey PRIMARY KEY (source_housing_company_id);


--
-- Name: source_listings source_listings_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.source_listings
    ADD CONSTRAINT source_listings_pkey PRIMARY KEY (source_listing_id);


--
-- Name: target_observations target_observations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.target_observations
    ADD CONSTRAINT target_observations_pkey PRIMARY KEY (target_observation_id);


--
-- Name: target_sources target_sources_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.target_sources
    ADD CONSTRAINT target_sources_pkey PRIMARY KEY (target_source_id);


--
-- Name: units units_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.units
    ADD CONSTRAINT units_pkey PRIMARY KEY (unit_id);


--
-- Name: user_devices user_devices_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_devices
    ADD CONSTRAINT user_devices_pkey PRIMARY KEY (user_device_id);


--
-- Name: user_devices user_devices_uuid_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_devices
    ADD CONSTRAINT user_devices_uuid_key UNIQUE (user_device_uuid);


--
-- Name: user_email_change_tokens user_email_change_tokens_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_email_change_tokens
    ADD CONSTRAINT user_email_change_tokens_pkey PRIMARY KEY (user_email_change_token_id);


--
-- Name: user_email_change_tokens user_email_change_tokens_token_hash_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_email_change_tokens
    ADD CONSTRAINT user_email_change_tokens_token_hash_key UNIQUE (user_email_change_token_hash);


--
-- Name: user_email_change_tokens user_email_change_tokens_uuid_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_email_change_tokens
    ADD CONSTRAINT user_email_change_tokens_uuid_key UNIQUE (user_email_change_token_uuid);


--
-- Name: user_feature_flags user_feature_flags_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_feature_flags
    ADD CONSTRAINT user_feature_flags_pkey PRIMARY KEY (user_id, flag_id);


--
-- Name: user_identities user_identities_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_identities
    ADD CONSTRAINT user_identities_pkey PRIMARY KEY (user_identity_id);


--
-- Name: user_identities user_identities_uuid_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_identities
    ADD CONSTRAINT user_identities_uuid_key UNIQUE (user_identity_uuid);


--
-- Name: user_passkeys user_passkeys_credential_id_b64url_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_passkeys
    ADD CONSTRAINT user_passkeys_credential_id_b64url_key UNIQUE (user_passkey_credential_id_b64url);


--
-- Name: user_passkeys user_passkeys_credential_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_passkeys
    ADD CONSTRAINT user_passkeys_credential_id_key UNIQUE (user_passkey_credential_id);


--
-- Name: user_passkeys user_passkeys_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_passkeys
    ADD CONSTRAINT user_passkeys_pkey PRIMARY KEY (user_passkey_id);


--
-- Name: user_passkeys user_passkeys_user_passkey_uuid_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_passkeys
    ADD CONSTRAINT user_passkeys_user_passkey_uuid_key UNIQUE (user_passkey_uuid);


--
-- Name: user_roles user_roles_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_roles
    ADD CONSTRAINT user_roles_pkey PRIMARY KEY (user_id, role_id);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (user_id);


--
-- Name: users users_username_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_username_key UNIQUE (user_username);


--
-- Name: users users_uuid_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_uuid_key UNIQUE (user_uuid);


--
-- Name: kv_store kv_store_pkey; Type: CONSTRAINT; Schema: runtime; Owner: -
--

ALTER TABLE ONLY runtime.kv_store
    ADD CONSTRAINT kv_store_pkey PRIMARY KEY (kv_key);


--
-- Name: e_canonical_db_eai; Type: INDEX; Schema: absurd; Owner: -
--

CREATE INDEX e_canonical_db_eai ON absurd.e_canonical_db USING btree (emitted_at);


--
-- Name: e_canonical_llm_eai; Type: INDEX; Schema: absurd; Owner: -
--

CREATE INDEX e_canonical_llm_eai ON absurd.e_canonical_llm USING btree (emitted_at);


--
-- Name: e_frontdoor_eai; Type: INDEX; Schema: absurd; Owner: -
--

CREATE INDEX e_frontdoor_eai ON absurd.e_frontdoor USING btree (emitted_at);


--
-- Name: e_postal_eai; Type: INDEX; Schema: absurd; Owner: -
--

CREATE INDEX e_postal_eai ON absurd.e_postal USING btree (emitted_at);


--
-- Name: e_prices_eai; Type: INDEX; Schema: absurd; Owner: -
--

CREATE INDEX e_prices_eai ON absurd.e_prices USING btree (emitted_at);


--
-- Name: e_shortcut_api_eai; Type: INDEX; Schema: absurd; Owner: -
--

CREATE INDEX e_shortcut_api_eai ON absurd.e_shortcut_api USING btree (emitted_at);


--
-- Name: e_shortcut_scraper_eai; Type: INDEX; Schema: absurd; Owner: -
--

CREATE INDEX e_shortcut_scraper_eai ON absurd.e_shortcut_scraper USING btree (emitted_at);


--
-- Name: r_canonical_db_cei; Type: INDEX; Schema: absurd; Owner: -
--

CREATE INDEX r_canonical_db_cei ON absurd.r_canonical_db USING btree (claim_expires_at) WHERE ((state = 'running'::text) AND (claim_expires_at IS NOT NULL));


--
-- Name: r_canonical_db_sai; Type: INDEX; Schema: absurd; Owner: -
--

CREATE INDEX r_canonical_db_sai ON absurd.r_canonical_db USING btree (state, available_at);


--
-- Name: r_canonical_db_ti; Type: INDEX; Schema: absurd; Owner: -
--

CREATE INDEX r_canonical_db_ti ON absurd.r_canonical_db USING btree (task_id);


--
-- Name: r_canonical_llm_cei; Type: INDEX; Schema: absurd; Owner: -
--

CREATE INDEX r_canonical_llm_cei ON absurd.r_canonical_llm USING btree (claim_expires_at) WHERE ((state = 'running'::text) AND (claim_expires_at IS NOT NULL));


--
-- Name: r_canonical_llm_sai; Type: INDEX; Schema: absurd; Owner: -
--

CREATE INDEX r_canonical_llm_sai ON absurd.r_canonical_llm USING btree (state, available_at);


--
-- Name: r_canonical_llm_ti; Type: INDEX; Schema: absurd; Owner: -
--

CREATE INDEX r_canonical_llm_ti ON absurd.r_canonical_llm USING btree (task_id);


--
-- Name: r_frontdoor_cei; Type: INDEX; Schema: absurd; Owner: -
--

CREATE INDEX r_frontdoor_cei ON absurd.r_frontdoor USING btree (claim_expires_at) WHERE ((state = 'running'::text) AND (claim_expires_at IS NOT NULL));


--
-- Name: r_frontdoor_sai; Type: INDEX; Schema: absurd; Owner: -
--

CREATE INDEX r_frontdoor_sai ON absurd.r_frontdoor USING btree (state, available_at);


--
-- Name: r_frontdoor_ti; Type: INDEX; Schema: absurd; Owner: -
--

CREATE INDEX r_frontdoor_ti ON absurd.r_frontdoor USING btree (task_id);


--
-- Name: r_postal_cei; Type: INDEX; Schema: absurd; Owner: -
--

CREATE INDEX r_postal_cei ON absurd.r_postal USING btree (claim_expires_at) WHERE ((state = 'running'::text) AND (claim_expires_at IS NOT NULL));


--
-- Name: r_postal_sai; Type: INDEX; Schema: absurd; Owner: -
--

CREATE INDEX r_postal_sai ON absurd.r_postal USING btree (state, available_at);


--
-- Name: r_postal_ti; Type: INDEX; Schema: absurd; Owner: -
--

CREATE INDEX r_postal_ti ON absurd.r_postal USING btree (task_id);


--
-- Name: r_prices_cei; Type: INDEX; Schema: absurd; Owner: -
--

CREATE INDEX r_prices_cei ON absurd.r_prices USING btree (claim_expires_at) WHERE ((state = 'running'::text) AND (claim_expires_at IS NOT NULL));


--
-- Name: r_prices_sai; Type: INDEX; Schema: absurd; Owner: -
--

CREATE INDEX r_prices_sai ON absurd.r_prices USING btree (state, available_at);


--
-- Name: r_prices_ti; Type: INDEX; Schema: absurd; Owner: -
--

CREATE INDEX r_prices_ti ON absurd.r_prices USING btree (task_id);


--
-- Name: r_shortcut_api_cei; Type: INDEX; Schema: absurd; Owner: -
--

CREATE INDEX r_shortcut_api_cei ON absurd.r_shortcut_api USING btree (claim_expires_at) WHERE ((state = 'running'::text) AND (claim_expires_at IS NOT NULL));


--
-- Name: r_shortcut_api_sai; Type: INDEX; Schema: absurd; Owner: -
--

CREATE INDEX r_shortcut_api_sai ON absurd.r_shortcut_api USING btree (state, available_at);


--
-- Name: r_shortcut_api_ti; Type: INDEX; Schema: absurd; Owner: -
--

CREATE INDEX r_shortcut_api_ti ON absurd.r_shortcut_api USING btree (task_id);


--
-- Name: r_shortcut_scraper_cei; Type: INDEX; Schema: absurd; Owner: -
--

CREATE INDEX r_shortcut_scraper_cei ON absurd.r_shortcut_scraper USING btree (claim_expires_at) WHERE ((state = 'running'::text) AND (claim_expires_at IS NOT NULL));


--
-- Name: r_shortcut_scraper_sai; Type: INDEX; Schema: absurd; Owner: -
--

CREATE INDEX r_shortcut_scraper_sai ON absurd.r_shortcut_scraper USING btree (state, available_at);


--
-- Name: r_shortcut_scraper_ti; Type: INDEX; Schema: absurd; Owner: -
--

CREATE INDEX r_shortcut_scraper_ti ON absurd.r_shortcut_scraper USING btree (task_id);


--
-- Name: w_canonical_db_eni; Type: INDEX; Schema: absurd; Owner: -
--

CREATE INDEX w_canonical_db_eni ON absurd.w_canonical_db USING btree (event_name);


--
-- Name: w_canonical_db_ti; Type: INDEX; Schema: absurd; Owner: -
--

CREATE INDEX w_canonical_db_ti ON absurd.w_canonical_db USING btree (task_id);


--
-- Name: w_canonical_llm_eni; Type: INDEX; Schema: absurd; Owner: -
--

CREATE INDEX w_canonical_llm_eni ON absurd.w_canonical_llm USING btree (event_name);


--
-- Name: w_canonical_llm_ti; Type: INDEX; Schema: absurd; Owner: -
--

CREATE INDEX w_canonical_llm_ti ON absurd.w_canonical_llm USING btree (task_id);


--
-- Name: w_frontdoor_eni; Type: INDEX; Schema: absurd; Owner: -
--

CREATE INDEX w_frontdoor_eni ON absurd.w_frontdoor USING btree (event_name);


--
-- Name: w_frontdoor_ti; Type: INDEX; Schema: absurd; Owner: -
--

CREATE INDEX w_frontdoor_ti ON absurd.w_frontdoor USING btree (task_id);


--
-- Name: w_postal_eni; Type: INDEX; Schema: absurd; Owner: -
--

CREATE INDEX w_postal_eni ON absurd.w_postal USING btree (event_name);


--
-- Name: w_postal_ti; Type: INDEX; Schema: absurd; Owner: -
--

CREATE INDEX w_postal_ti ON absurd.w_postal USING btree (task_id);


--
-- Name: w_prices_eni; Type: INDEX; Schema: absurd; Owner: -
--

CREATE INDEX w_prices_eni ON absurd.w_prices USING btree (event_name);


--
-- Name: w_prices_ti; Type: INDEX; Schema: absurd; Owner: -
--

CREATE INDEX w_prices_ti ON absurd.w_prices USING btree (task_id);


--
-- Name: w_shortcut_api_eni; Type: INDEX; Schema: absurd; Owner: -
--

CREATE INDEX w_shortcut_api_eni ON absurd.w_shortcut_api USING btree (event_name);


--
-- Name: w_shortcut_api_ti; Type: INDEX; Schema: absurd; Owner: -
--

CREATE INDEX w_shortcut_api_ti ON absurd.w_shortcut_api USING btree (task_id);


--
-- Name: w_shortcut_scraper_eni; Type: INDEX; Schema: absurd; Owner: -
--

CREATE INDEX w_shortcut_scraper_eni ON absurd.w_shortcut_scraper USING btree (event_name);


--
-- Name: w_shortcut_scraper_ti; Type: INDEX; Schema: absurd; Owner: -
--

CREATE INDEX w_shortcut_scraper_ti ON absurd.w_shortcut_scraper USING btree (task_id);


--
-- Name: idx_auth_devices_push_token; Type: INDEX; Schema: auth; Owner: -
--

CREATE INDEX idx_auth_devices_push_token ON auth.devices USING btree (device_push_token) WHERE (device_push_token IS NOT NULL);


--
-- Name: idx_auth_devices_user_id; Type: INDEX; Schema: auth; Owner: -
--

CREATE INDEX idx_auth_devices_user_id ON auth.devices USING btree (user_id);


--
-- Name: idx_auth_identities_external_id; Type: INDEX; Schema: auth; Owner: -
--

CREATE INDEX idx_auth_identities_external_id ON auth.identities USING btree (identity_provider, identity_external_id);


--
-- Name: idx_auth_identities_provider; Type: INDEX; Schema: auth; Owner: -
--

CREATE INDEX idx_auth_identities_provider ON auth.identities USING btree (identity_provider);


--
-- Name: idx_auth_identities_user_id; Type: INDEX; Schema: auth; Owner: -
--

CREATE INDEX idx_auth_identities_user_id ON auth.identities USING btree (user_id);


--
-- Name: idx_auth_refresh_tokens_session_id; Type: INDEX; Schema: auth; Owner: -
--

CREATE INDEX idx_auth_refresh_tokens_session_id ON auth.refresh_tokens USING btree (session_id);


--
-- Name: idx_auth_refresh_tokens_session_revoked; Type: INDEX; Schema: auth; Owner: -
--

CREATE INDEX idx_auth_refresh_tokens_session_revoked ON auth.refresh_tokens USING btree (session_id, refresh_token_revoked);


--
-- Name: idx_auth_refresh_tokens_token_hash; Type: INDEX; Schema: auth; Owner: -
--

CREATE UNIQUE INDEX idx_auth_refresh_tokens_token_hash ON auth.refresh_tokens USING btree (refresh_token_token_hash);


--
-- Name: idx_auth_refresh_tokens_updated_at; Type: INDEX; Schema: auth; Owner: -
--

CREATE INDEX idx_auth_refresh_tokens_updated_at ON auth.refresh_tokens USING btree (refresh_token_updated_at DESC);


--
-- Name: idx_auth_sessions_not_after; Type: INDEX; Schema: auth; Owner: -
--

CREATE INDEX idx_auth_sessions_not_after ON auth.sessions USING btree (session_not_after DESC);


--
-- Name: idx_auth_sessions_user_created; Type: INDEX; Schema: auth; Owner: -
--

CREATE INDEX idx_auth_sessions_user_created ON auth.sessions USING btree (user_id, session_created_at);


--
-- Name: idx_auth_sessions_user_id; Type: INDEX; Schema: auth; Owner: -
--

CREATE INDEX idx_auth_sessions_user_id ON auth.sessions USING btree (user_id);


--
-- Name: idx_auth_users_created_at; Type: INDEX; Schema: auth; Owner: -
--

CREATE INDEX idx_auth_users_created_at ON auth.users USING btree (user_created_at);


--
-- Name: idx_role_feature_flags_role_id; Type: INDEX; Schema: auth; Owner: -
--

CREATE INDEX idx_role_feature_flags_role_id ON auth.role_feature_flags USING btree (role_id);


--
-- Name: idx_user_feature_flags_user_id; Type: INDEX; Schema: auth; Owner: -
--

CREATE INDEX idx_user_feature_flags_user_id ON auth.user_feature_flags USING btree (user_id);


--
-- Name: idx_user_roles_role_id; Type: INDEX; Schema: auth; Owner: -
--

CREATE INDEX idx_user_roles_role_id ON auth.user_roles USING btree (role_id);


--
-- Name: idx_user_roles_user_id; Type: INDEX; Schema: auth; Owner: -
--

CREATE INDEX idx_user_roles_user_id ON auth.user_roles USING btree (user_id);


--
-- Name: frontdoor_building_announcements_ext_id_unpub_time_price_key; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX frontdoor_building_announcements_ext_id_unpub_time_price_key ON public.frontdoor_building_announcements USING btree (frontdoor_building_announcement_external_id, frontdoor_building_announcement_unpublishing_time, frontdoor_building_announcement_search_price);


--
-- Name: frontdoor_buildings_housing_company_friendly_id_unique; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX frontdoor_buildings_housing_company_friendly_id_unique ON public.frontdoor_buildings USING btree (frontdoor_building_housing_company_friendly_id) WHERE (frontdoor_building_housing_company_friendly_id IS NOT NULL);


--
-- Name: frontdoor_buildings_url_unique; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX frontdoor_buildings_url_unique ON public.frontdoor_buildings USING btree (frontdoor_building_url);


--
-- Name: houses_identity_key_key; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX houses_identity_key_key ON public.houses USING btree (identity_key);


--
-- Name: idx_auth_signup_email_tokens_active_expires_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_auth_signup_email_tokens_active_expires_at ON public.auth_signup_email_tokens USING btree (auth_signup_email_expires_at) WHERE (auth_signup_email_consumed_at IS NULL);


--
-- Name: idx_auth_signup_email_tokens_target_email; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_auth_signup_email_tokens_target_email ON public.auth_signup_email_tokens USING btree (lower(btrim(auth_signup_email_target_email)));


--
-- Name: idx_auth_signup_tickets_active_expires_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_auth_signup_tickets_active_expires_at ON public.auth_signup_tickets USING btree (auth_signup_ticket_expires_at) WHERE (auth_signup_ticket_consumed_at IS NULL);


--
-- Name: idx_auth_signup_tickets_target_email; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_auth_signup_tickets_target_email ON public.auth_signup_tickets USING btree (lower(btrim(auth_signup_ticket_target_email)));


--
-- Name: idx_auth_webauthn_challenges_active; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_auth_webauthn_challenges_active ON public.auth_webauthn_challenges USING btree (auth_webauthn_challenge_uuid, auth_webauthn_challenge_flow) WHERE (auth_webauthn_challenge_consumed_at IS NULL);


--
-- Name: idx_auth_webauthn_challenges_expires; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_auth_webauthn_challenges_expires ON public.auth_webauthn_challenges USING btree (auth_webauthn_challenge_expires_at);


--
-- Name: idx_auth_webauthn_challenges_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_auth_webauthn_challenges_uuid ON public.auth_webauthn_challenges USING btree (auth_webauthn_challenge_uuid);


--
-- Name: idx_device_sessions_not_after; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_device_sessions_not_after ON public.device_sessions USING btree (device_session_not_after DESC);


--
-- Name: idx_device_sessions_user_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_device_sessions_user_created ON public.device_sessions USING btree (user_id, device_session_created_at);


--
-- Name: idx_device_sessions_user_device_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_device_sessions_user_device_id ON public.device_sessions USING btree (device_session_user_device_id);


--
-- Name: idx_device_sessions_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_device_sessions_user_id ON public.device_sessions USING btree (user_id);


--
-- Name: idx_dimension_claims_dimension; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_dimension_claims_dimension ON public.dimension_claims USING btree (dimension_key);


--
-- Name: idx_dimension_claims_source; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_dimension_claims_source ON public.dimension_claims USING btree (source_table, source_id, projection_version);


--
-- Name: idx_dimension_claims_source_claim; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_dimension_claims_source_claim ON public.dimension_claims USING btree (source_claim_id);


--
-- Name: idx_dimension_claims_target; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_dimension_claims_target ON public.dimension_claims USING btree (claim_scope, target_type, target_id, dimension_key);


--
-- Name: idx_dimension_claims_unique_source; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_dimension_claims_unique_source ON public.dimension_claims USING btree (claim_scope, target_type, target_id, dimension_key, source_table, source_id, COALESCE(source_field, ''::text), projection_version);


--
-- Name: idx_dimension_claims_value_gin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_dimension_claims_value_gin ON public.dimension_claims USING gin (value jsonb_path_ops);


--
-- Name: idx_dimension_profiles_building_build_year; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_dimension_profiles_building_build_year ON public.dimension_profiles USING btree ((((dimensions #>> '{building,build_year}'::text[]))::integer)) WHERE (target_type = 'building'::text);


--
-- Name: idx_dimension_profiles_dimensions_gin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_dimension_profiles_dimensions_gin ON public.dimension_profiles USING gin (dimensions jsonb_path_ops);


--
-- Name: idx_dimension_profiles_unit_area; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_dimension_profiles_unit_area ON public.dimension_profiles USING btree ((((dimensions #>> '{unit,area_m2}'::text[]))::double precision)) WHERE (target_type = 'unit'::text);


--
-- Name: idx_dimension_profiles_unit_total_charge; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_dimension_profiles_unit_total_charge ON public.dimension_profiles USING btree ((((dimensions #>> '{charges,total_monthly_eur}'::text[]))::double precision)) WHERE (target_type = 'unit'::text);


--
-- Name: idx_dimension_values_dimension; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_dimension_values_dimension ON public.dimension_values USING btree (dimension_key);


--
-- Name: idx_dimension_values_selected_claim; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_dimension_values_selected_claim ON public.dimension_values USING btree (selected_claim_id);


--
-- Name: idx_frontdoor_ad_page_not_found; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_frontdoor_ad_page_not_found ON public.frontdoor_ads USING btree (frontdoor_ad_page_not_found);


--
-- Name: idx_frontdoor_ad_processed_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_frontdoor_ad_processed_at ON public.frontdoor_ads USING btree (frontdoor_ad_processed_at);


--
-- Name: idx_frontdoor_ads_data_hash; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_frontdoor_ads_data_hash ON public.frontdoor_ads USING btree (frontdoor_ad_data_hash);


--
-- Name: idx_frontdoor_ads_data_normalized; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_frontdoor_ads_data_normalized ON public.frontdoor_ads USING btree (frontdoor_ad_data_normalized_at) WHERE (frontdoor_ad_data_hash IS NOT NULL);


--
-- Name: idx_frontdoor_ads_data_normalized_version; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_frontdoor_ads_data_normalized_version ON public.frontdoor_ads USING btree (frontdoor_ad_data_normalized_version) WHERE (frontdoor_ad_data_hash IS NOT NULL);


--
-- Name: idx_frontdoor_building_announcement_building_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_frontdoor_building_announcement_building_id ON public.frontdoor_building_announcements USING btree (frontdoor_building_id);


--
-- Name: idx_frontdoor_building_announcements_normalized; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_frontdoor_building_announcements_normalized ON public.frontdoor_building_announcements USING btree (frontdoor_building_announcement_data_normalized_at, frontdoor_building_announcement_data_normalized_version);


--
-- Name: idx_frontdoor_building_business_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_frontdoor_building_business_id ON public.frontdoor_buildings USING btree (frontdoor_building_business_id);


--
-- Name: idx_frontdoor_building_processed_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_frontdoor_building_processed_at ON public.frontdoor_buildings USING btree (frontdoor_building_processed_at);


--
-- Name: idx_houses_address; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_houses_address ON public.houses USING btree (postal_norm, city_norm, address_norm);


--
-- Name: idx_houses_lat_lng; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_houses_lat_lng ON public.houses USING btree (latitude, longitude) WHERE ((latitude IS NOT NULL) AND (longitude IS NOT NULL));


--
-- Name: idx_housing_companies_address; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_housing_companies_address ON public.housing_companies USING btree (housing_company_postal_norm, housing_company_city_norm, housing_company_address_norm);


--
-- Name: idx_housing_companies_business_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_housing_companies_business_id ON public.housing_companies USING btree (housing_company_business_id) WHERE ((housing_company_business_id IS NOT NULL) AND (housing_company_business_id <> ''::text));


--
-- Name: idx_housing_companies_geom; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_housing_companies_geom ON public.housing_companies USING gist (housing_company_geom);


--
-- Name: idx_housing_company_merge_decisions_active_pair; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_housing_company_merge_decisions_active_pair ON public.housing_company_merge_decisions USING btree (source_housing_company_id, target_housing_company_id) WHERE (housing_company_merge_decision_status <> 'rejected'::text);


--
-- Name: idx_housing_company_merge_decisions_source; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_housing_company_merge_decisions_source ON public.housing_company_merge_decisions USING btree (source_housing_company_id, housing_company_merge_decision_status);


--
-- Name: idx_housing_company_merge_decisions_target; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_housing_company_merge_decisions_target ON public.housing_company_merge_decisions USING btree (target_housing_company_id, housing_company_merge_decision_status);


--
-- Name: idx_listings_house; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_listings_house ON public.listings USING btree (house_id) WHERE (house_id IS NOT NULL);


--
-- Name: idx_listings_last_seen; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_listings_last_seen ON public.listings USING btree (last_seen_at DESC);


--
-- Name: idx_listings_primary_source_listing; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_listings_primary_source_listing ON public.listings USING btree (primary_source_listing_id);


--
-- Name: idx_listings_unit; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_listings_unit ON public.listings USING btree (unit_id) WHERE (unit_id IS NOT NULL);


--
-- Name: idx_oauth_authorization_codes_audience; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_oauth_authorization_codes_audience ON public.oauth_authorization_codes USING btree (oauth_authorization_code_audience);


--
-- Name: idx_oauth_authorization_codes_client_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_oauth_authorization_codes_client_id ON public.oauth_authorization_codes USING btree (oauth_client_id);


--
-- Name: idx_oauth_authorization_codes_expires_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_oauth_authorization_codes_expires_at ON public.oauth_authorization_codes USING btree (oauth_authorization_code_expires_at);


--
-- Name: idx_oauth_authorization_codes_user_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_oauth_authorization_codes_user_uuid ON public.oauth_authorization_codes USING btree (user_uuid);


--
-- Name: idx_oauth_authorization_handoffs_client_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_oauth_authorization_handoffs_client_id ON public.oauth_authorization_handoffs USING btree (oauth_client_id);


--
-- Name: idx_oauth_authorization_handoffs_expires_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_oauth_authorization_handoffs_expires_at ON public.oauth_authorization_handoffs USING btree (oauth_authorization_handoff_expires_at);


--
-- Name: idx_oauth_authorization_handoffs_user_code; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_oauth_authorization_handoffs_user_code ON public.oauth_authorization_handoffs USING btree (oauth_authorization_handoff_user_code);


--
-- Name: idx_oauth_device_authorizations_audience; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_oauth_device_authorizations_audience ON public.oauth_device_authorizations USING btree (oauth_device_authorization_audience);


--
-- Name: idx_oauth_device_authorizations_client_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_oauth_device_authorizations_client_id ON public.oauth_device_authorizations USING btree (oauth_client_id);


--
-- Name: idx_oauth_device_authorizations_expires_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_oauth_device_authorizations_expires_at ON public.oauth_device_authorizations USING btree (oauth_device_authorization_expires_at);


--
-- Name: idx_oauth_device_authorizations_user_code; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_oauth_device_authorizations_user_code ON public.oauth_device_authorizations USING btree (oauth_device_authorization_user_code);


--
-- Name: idx_oauth_dynamic_clients_disabled_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_oauth_dynamic_clients_disabled_at ON public.oauth_dynamic_clients USING btree (oauth_dynamic_client_disabled_at);


--
-- Name: idx_oauth_refresh_tokens_audience; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_oauth_refresh_tokens_audience ON public.oauth_refresh_tokens USING btree (oauth_refresh_token_audience);


--
-- Name: idx_oauth_refresh_tokens_client_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_oauth_refresh_tokens_client_id ON public.oauth_refresh_tokens USING btree (oauth_client_id);


--
-- Name: idx_oauth_refresh_tokens_device_session_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_oauth_refresh_tokens_device_session_uuid ON public.oauth_refresh_tokens USING btree (device_session_uuid);


--
-- Name: idx_oauth_refresh_tokens_expires_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_oauth_refresh_tokens_expires_at ON public.oauth_refresh_tokens USING btree (oauth_refresh_token_expires_at);


--
-- Name: idx_oauth_refresh_tokens_user_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_oauth_refresh_tokens_user_uuid ON public.oauth_refresh_tokens USING btree (user_uuid);


--
-- Name: idx_personal_access_tokens_prefix; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_personal_access_tokens_prefix ON public.personal_access_tokens USING btree (personal_access_token_prefix);


--
-- Name: idx_personal_access_tokens_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_personal_access_tokens_user_id ON public.personal_access_tokens USING btree (user_id);


--
-- Name: idx_physical_buildings_housing_company; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_physical_buildings_housing_company ON public.physical_buildings USING btree (housing_company_id);


--
-- Name: idx_physical_buildings_lat_lng; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_physical_buildings_lat_lng ON public.physical_buildings USING btree (physical_building_latitude, physical_building_longitude) WHERE ((physical_building_latitude IS NOT NULL) AND (physical_building_longitude IS NOT NULL));


--
-- Name: idx_postal_municipality_name_fi; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_postal_municipality_name_fi ON public.postal_municipalities USING btree (postal_municipality_name_fi);


--
-- Name: idx_postal_postal_code_ad_area_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_postal_postal_code_ad_area_id ON public.postal_postal_codes USING btree (postal_ad_area_id);


--
-- Name: idx_postal_postal_code_municipality_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_postal_postal_code_municipality_id ON public.postal_postal_codes USING btree (postal_municipality_id);


--
-- Name: idx_postal_postal_code_name_fi; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_postal_postal_code_name_fi ON public.postal_postal_codes USING btree (postal_postal_code_name_fi);


--
-- Name: idx_postal_postal_code_neighborhood_fi; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_postal_postal_code_neighborhood_fi ON public.postal_postal_codes USING btree (postal_postal_code_neighborhood_fi);


--
-- Name: idx_price_links_target; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_price_links_target ON public.price_links USING btree (target_type, target_id, link_status);


--
-- Name: idx_price_links_transaction; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_price_links_transaction ON public.price_links USING btree (prices_transaction_id, link_status);


--
-- Name: idx_prices_neighborhood_postal_postal_code_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_prices_neighborhood_postal_postal_code_id ON public.prices_neighborhoods USING btree (prices_neighborhood_postal_postal_code_id);


--
-- Name: idx_prices_transaction_period_identifier; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_prices_transaction_period_identifier ON public.prices_transactions USING btree (prices_transaction_period_identifier);


--
-- Name: idx_prices_transactions_plot_owned; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_prices_transactions_plot_owned ON public.prices_transactions USING btree (prices_transaction_plot_owned);


--
-- Name: idx_property_dimension_dirty_targets_queue; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_property_dimension_dirty_targets_queue ON public.property_dimension_dirty_targets USING btree (dirty_at) WHERE ((resolved_at IS NULL) OR (resolved_at < dirty_at));


--
-- Name: idx_property_dimension_projection_runs_source; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_property_dimension_projection_runs_source ON public.property_dimension_projection_runs USING btree (projection_type, source_table, source_id, projection_version, started_at DESC);


--
-- Name: idx_property_dimension_source_priorities_unique; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_property_dimension_source_priorities_unique ON public.property_dimension_source_priorities USING btree (dimension_key, source_table, COALESCE(source_field, ''::text));


--
-- Name: idx_property_document_extraction_runs_document; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_property_document_extraction_runs_document ON public.property_document_extraction_runs USING btree (property_document_id, property_document_extraction_run_started_at DESC);


--
-- Name: idx_property_document_extractions_document; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_property_document_extractions_document ON public.property_document_extractions USING btree (property_document_id, property_document_extraction_created_at DESC);


--
-- Name: idx_property_document_extractions_latest; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_property_document_extractions_latest ON public.property_document_extractions USING btree (property_document_id, property_document_extraction_kind) WHERE (property_document_extraction_superseded_at IS NULL);


--
-- Name: idx_property_documents_detached_type_hash; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_property_documents_detached_type_hash ON public.property_documents USING btree (property_document_type, property_document_sha256) WHERE (property_offering_id IS NULL);


--
-- Name: idx_property_documents_housing_company; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_property_documents_housing_company ON public.property_documents USING btree (housing_company_id, property_document_type) WHERE (housing_company_id IS NOT NULL);


--
-- Name: idx_property_documents_offering; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_property_documents_offering ON public.property_documents USING btree (property_offering_id, property_document_type, property_document_uploaded_at DESC);


--
-- Name: idx_property_documents_offering_type_hash; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_property_documents_offering_type_hash ON public.property_documents USING btree (property_offering_id, property_document_type, property_document_sha256);


--
-- Name: idx_property_houses_address; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_property_houses_address ON public.property_houses USING btree (property_house_postal_norm, property_house_city_norm, property_house_address_norm);


--
-- Name: idx_property_houses_lat_lng; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_property_houses_lat_lng ON public.property_houses USING btree (property_house_latitude, property_house_longitude) WHERE ((property_house_latitude IS NOT NULL) AND (property_house_longitude IS NOT NULL));


--
-- Name: idx_property_offering_merge_decisions_active_pair; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_property_offering_merge_decisions_active_pair ON public.property_offering_merge_decisions USING btree (source_property_offering_id, target_property_offering_id) WHERE (property_offering_merge_decision_status <> 'rejected'::text);


--
-- Name: idx_property_offering_merge_decisions_source; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_property_offering_merge_decisions_source ON public.property_offering_merge_decisions USING btree (source_property_offering_id, property_offering_merge_decision_status);


--
-- Name: idx_property_offering_merge_decisions_target; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_property_offering_merge_decisions_target ON public.property_offering_merge_decisions USING btree (target_property_offering_id, property_offering_merge_decision_status);


--
-- Name: idx_property_offerings_house; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_property_offerings_house ON public.property_offerings USING btree (property_house_id) WHERE (property_house_id IS NOT NULL);


--
-- Name: idx_property_offerings_primary_sale_listing; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_property_offerings_primary_sale_listing ON public.property_offerings USING btree (primary_sale_listing_id);


--
-- Name: idx_property_offerings_unit; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_property_offerings_unit ON public.property_offerings USING btree (property_unit_id);


--
-- Name: idx_property_renovation_events_source; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_property_renovation_events_source ON public.property_renovation_events USING btree (source_table, source_id, projection_version);


--
-- Name: idx_property_renovation_events_source_event; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_property_renovation_events_source_event ON public.property_renovation_events USING btree (source_event_id);


--
-- Name: idx_property_renovation_events_target; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_property_renovation_events_target ON public.property_renovation_events USING btree (event_scope, target_type, target_id, category, status);


--
-- Name: idx_property_renovation_events_target_observed; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_property_renovation_events_target_observed ON public.property_renovation_events USING btree (event_scope, target_type, target_id, category, status, source_observed_at DESC);


--
-- Name: idx_property_renovation_events_unique_source; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_property_renovation_events_unique_source ON public.property_renovation_events USING btree (event_scope, target_type, target_id, source_table, source_id, COALESCE(source_field, ''::text), category, status, COALESCE(stage, ''::text), COALESCE(scope, ''::text), COALESCE(year, '-1'::integer), COALESCE(start_year, '-1'::integer), COALESCE(end_year, '-1'::integer), md5(COALESCE(summary, ''::text)), projection_version);


--
-- Name: idx_property_source_offering_renovations_listing; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_property_source_offering_renovations_listing ON public.property_source_offering_renovations USING btree (sale_listing_id);


--
-- Name: idx_property_source_offering_renovations_unique; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_property_source_offering_renovations_unique ON public.property_source_offering_renovations USING btree (sale_listing_id, property_source_offering_renovation_source_field, property_source_offering_renovation_category, property_source_offering_renovation_status, COALESCE(property_source_offering_renovation_year, 0), COALESCE(property_source_offering_renovation_component, ''::text), COALESCE(property_source_offering_renovation_stage, ''::text));


--
-- Name: idx_property_source_offerings_street_name_number_ascii; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_property_source_offerings_street_name_number_ascii ON public.property_source_offerings USING btree (translate(sale_listing_street_name_norm, 'åäö'::text, 'aao'::text), sale_listing_street_number_norm, sale_listing_last_seen_at DESC);


--
-- Name: idx_property_units_housing_company; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_property_units_housing_company ON public.property_units USING btree (housing_company_id);


--
-- Name: idx_property_units_physical_building; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_property_units_physical_building ON public.property_units USING btree (physical_building_id);


--
-- Name: idx_role_feature_flags_flag_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_role_feature_flags_flag_id ON public.role_feature_flags USING btree (flag_id);


--
-- Name: idx_role_feature_flags_role_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_role_feature_flags_role_id ON public.role_feature_flags USING btree (role_id);


--
-- Name: idx_sale_listing_prices_transaction_match_candidates_listing_sc; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sale_listing_prices_transaction_match_candidates_listing_sc ON public.sale_listing_prices_transaction_match_candidates USING btree (sale_listing_id, sale_listing_prices_transaction_match_score DESC);


--
-- Name: idx_sale_listing_prices_transaction_match_candidates_run_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sale_listing_prices_transaction_match_candidates_run_status ON public.sale_listing_prices_transaction_match_candidates USING btree (sale_listing_prices_transaction_match_run_id, sale_listing_prices_transaction_match_status);


--
-- Name: idx_sale_listing_prices_transaction_match_candidates_transactio; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sale_listing_prices_transaction_match_candidates_transactio ON public.sale_listing_prices_transaction_match_candidates USING btree (prices_transaction_id, sale_listing_prices_transaction_match_score DESC);


--
-- Name: idx_sale_listings_area; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sale_listings_area ON public.property_source_offerings USING btree (sale_listing_area_value);


--
-- Name: idx_sale_listings_build_year; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sale_listings_build_year ON public.property_source_offerings USING btree (sale_listing_build_year);


--
-- Name: idx_sale_listings_building_match_key; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sale_listings_building_match_key ON public.property_source_offerings USING btree (sale_listing_building_match_key);


--
-- Name: idx_sale_listings_city; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sale_listings_city ON public.property_source_offerings USING btree (sale_listing_city);


--
-- Name: idx_sale_listings_elevator; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sale_listings_elevator ON public.property_source_offerings USING btree (sale_listing_elevator);


--
-- Name: idx_sale_listings_energy_efficiency_class_year; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sale_listings_energy_efficiency_class_year ON public.property_source_offerings USING btree (sale_listing_energy_efficiency_class_code, sale_listing_energy_efficiency_standard_year);


--
-- Name: idx_sale_listings_energy_efficiency_match_code; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sale_listings_energy_efficiency_match_code ON public.property_source_offerings USING btree (sale_listing_energy_efficiency_match_code);


--
-- Name: idx_sale_listings_energy_efficiency_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sale_listings_energy_efficiency_status ON public.property_source_offerings USING btree (sale_listing_energy_efficiency_status);


--
-- Name: idx_sale_listings_first_seen; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sale_listings_first_seen ON public.property_source_offerings USING btree (sale_listing_first_seen_at);


--
-- Name: idx_sale_listings_floor_level; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sale_listings_floor_level ON public.property_source_offerings USING btree (sale_listing_floor_level);


--
-- Name: idx_sale_listings_last_seen; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sale_listings_last_seen ON public.property_source_offerings USING btree (sale_listing_last_seen_at DESC);


--
-- Name: idx_sale_listings_plot_owned; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sale_listings_plot_owned ON public.property_source_offerings USING btree (sale_listing_plot_owned);


--
-- Name: idx_sale_listings_plot_type_code; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sale_listings_plot_type_code ON public.property_source_offerings USING btree (sale_listing_plot_type_code);


--
-- Name: idx_sale_listings_postal; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sale_listings_postal ON public.property_source_offerings USING btree (sale_listing_postal);


--
-- Name: idx_sale_listings_price; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sale_listings_price ON public.property_source_offerings USING btree (sale_listing_asking_price);


--
-- Name: idx_sale_listings_price_per_m2; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sale_listings_price_per_m2 ON public.property_source_offerings USING btree (sale_listing_price_per_m2);


--
-- Name: idx_sale_listings_prices_match_last_seen; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sale_listings_prices_match_last_seen ON public.property_source_offerings USING btree (sale_listing_last_seen_at) WHERE ((prices_transaction_id IS NULL) AND (sale_listing_source_kind = 'ad'::text));


--
-- Name: idx_sale_listings_prices_match_queue; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sale_listings_prices_match_queue ON public.property_source_offerings USING btree (sale_listing_prices_match_status, sale_listing_prices_match_next_attempt_at) WHERE (prices_transaction_id IS NULL);


--
-- Name: idx_sale_listings_property_type_code; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sale_listings_property_type_code ON public.property_source_offerings USING btree (sale_listing_property_type_code);


--
-- Name: idx_sale_listings_room_category_code; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sale_listings_room_category_code ON public.property_source_offerings USING btree (sale_listing_room_category_code);


--
-- Name: idx_sale_listings_rooms_count; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sale_listings_rooms_count ON public.property_source_offerings USING btree (sale_listing_rooms_count);


--
-- Name: idx_sale_listings_search_trgm; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sale_listings_search_trgm ON public.property_source_offerings USING gin (lower(sale_listing_search_text) public.gin_trgm_ops);


--
-- Name: idx_sale_listings_source; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sale_listings_source ON public.property_source_offerings USING btree (sale_listing_source_provider, sale_listing_source_kind);


--
-- Name: idx_sale_listings_source_match_queue; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sale_listings_source_match_queue ON public.property_source_offerings USING btree (sale_listing_source_match_status, sale_listing_source_match_next_attempt_at) WHERE (sale_listing_source_kind = 'ad'::text);


--
-- Name: idx_sale_listings_street_match_key; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sale_listings_street_match_key ON public.property_source_offerings USING btree (sale_listing_street_match_key);


--
-- Name: idx_sale_listings_unit_match_key; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sale_listings_unit_match_key ON public.property_source_offerings USING btree (sale_listing_unit_match_key);


--
-- Name: idx_shortcut_ads_data_hash; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_shortcut_ads_data_hash ON public.shortcut_ads USING btree (shortcut_ad_data_hash);


--
-- Name: idx_shortcut_ads_data_normalized; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_shortcut_ads_data_normalized ON public.shortcut_ads USING btree (shortcut_ad_data_normalized_at) WHERE (shortcut_ad_data_hash IS NOT NULL);


--
-- Name: idx_shortcut_ads_data_normalized_version; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_shortcut_ads_data_normalized_version ON public.shortcut_ads USING btree (shortcut_ad_data_normalized_version) WHERE (shortcut_ad_data_hash IS NOT NULL);


--
-- Name: idx_shortcut_token_cuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_shortcut_token_cuid ON public.shortcut_tokens USING btree (shortcut_token_cuid);


--
-- Name: idx_shortcut_token_expires_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_shortcut_token_expires_at ON public.shortcut_tokens USING btree (shortcut_token_expires_at DESC);


--
-- Name: idx_source_housing_companies_native; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_source_housing_companies_native ON public.source_housing_companies USING btree (provider, source_kind, native_id) WHERE (native_id IS NOT NULL);


--
-- Name: idx_source_listings_last_seen; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_source_listings_last_seen ON public.source_listings USING btree (last_seen_at DESC);


--
-- Name: idx_source_listings_raw; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_source_listings_raw ON public.source_listings USING btree (raw_table, raw_id);


--
-- Name: idx_target_observations_source; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_target_observations_source ON public.target_observations USING btree (source_type, source_id) WHERE (superseded_at IS NULL);


--
-- Name: idx_target_observations_target; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_target_observations_target ON public.target_observations USING btree (target_type, target_id, observation_kind, severity) WHERE (superseded_at IS NULL);


--
-- Name: idx_units_housing_company; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_units_housing_company ON public.units USING btree (housing_company_id);


--
-- Name: idx_units_physical_building; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_units_physical_building ON public.units USING btree (physical_building_id) WHERE (physical_building_id IS NOT NULL);


--
-- Name: idx_user_devices_push_token; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_user_devices_push_token ON public.user_devices USING btree (user_device_push_token) WHERE (user_device_push_token IS NOT NULL);


--
-- Name: idx_user_devices_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_user_devices_user_id ON public.user_devices USING btree (user_id);


--
-- Name: idx_user_email_change_tokens_active_expires_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_user_email_change_tokens_active_expires_at ON public.user_email_change_tokens USING btree (user_email_change_expires_at) WHERE (user_email_change_consumed_at IS NULL);


--
-- Name: idx_user_email_change_tokens_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_user_email_change_tokens_user_id ON public.user_email_change_tokens USING btree (user_id);


--
-- Name: idx_user_feature_flags_flag_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_user_feature_flags_flag_id ON public.user_feature_flags USING btree (flag_id);


--
-- Name: idx_user_feature_flags_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_user_feature_flags_user_id ON public.user_feature_flags USING btree (user_id);


--
-- Name: idx_user_identities_provider_external_id_unique; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_user_identities_provider_external_id_unique ON public.user_identities USING btree (user_identity_provider, user_identity_external_id);


--
-- Name: idx_user_identities_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_user_identities_user_id ON public.user_identities USING btree (user_id);


--
-- Name: idx_user_passkeys_active_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_user_passkeys_active_user_id ON public.user_passkeys USING btree (user_id) WHERE (user_passkey_revoked_at IS NULL);


--
-- Name: idx_user_passkeys_user_handle; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_user_passkeys_user_handle ON public.user_passkeys USING btree (user_passkey_user_handle);


--
-- Name: idx_user_passkeys_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_user_passkeys_user_id ON public.user_passkeys USING btree (user_id);


--
-- Name: idx_user_roles_role_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_user_roles_role_id ON public.user_roles USING btree (role_id);


--
-- Name: idx_user_roles_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_user_roles_user_id ON public.user_roles USING btree (user_id);


--
-- Name: idx_users_created_by_private; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_users_created_by_private ON public.users USING btree (user_uuid) WHERE (user_is_private = true);


--
-- Name: idx_users_user_email_normalized_unique; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_users_user_email_normalized_unique ON public.users USING btree (lower(btrim(user_email))) WHERE (user_email IS NOT NULL);


--
-- Name: idx_users_username; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_users_username ON public.users USING btree (user_username);


--
-- Name: price_links_one_confirmed_listing_per_transaction; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX price_links_one_confirmed_listing_per_transaction ON public.price_links USING btree (prices_transaction_id) WHERE ((target_type = 'listing'::text) AND (link_status = 'confirmed'::text));


--
-- Name: price_links_unique_target_transaction; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX price_links_unique_target_transaction ON public.price_links USING btree (target_type, target_id, prices_transaction_id);


--
-- Name: prices_transactions_unique_key; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX prices_transactions_unique_key ON public.prices_transactions USING btree (prices_neighborhood_id, prices_transaction_description, prices_transaction_type, prices_transaction_area, prices_transaction_price, prices_transaction_price_per_square_meter, prices_transaction_build_year, prices_transaction_floor, prices_transaction_elevator, prices_transaction_condition, prices_transaction_plot, prices_transaction_energy_class, prices_transaction_category) NULLS NOT DISTINCT;


--
-- Name: sale_listings_frontdoor_ad_id_key; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX sale_listings_frontdoor_ad_id_key ON public.property_source_offerings USING btree (frontdoor_ad_id) WHERE (frontdoor_ad_id IS NOT NULL);


--
-- Name: sale_listings_frontdoor_building_announcement_id_key; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX sale_listings_frontdoor_building_announcement_id_key ON public.property_source_offerings USING btree (frontdoor_building_announcement_id) WHERE (frontdoor_building_announcement_id IS NOT NULL);


--
-- Name: sale_listings_prices_transaction_id_key; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX sale_listings_prices_transaction_id_key ON public.property_source_offerings USING btree (prices_transaction_id) WHERE (prices_transaction_id IS NOT NULL);


--
-- Name: sale_listings_shortcut_ad_id_key; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX sale_listings_shortcut_ad_id_key ON public.property_source_offerings USING btree (shortcut_ad_id) WHERE (shortcut_ad_id IS NOT NULL);


--
-- Name: shortcut_building_geom_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX shortcut_building_geom_idx ON public.shortcut_buildings USING gist (shortcut_building_geom);


--
-- Name: shortcut_building_listings_unique_constraint; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX shortcut_building_listings_unique_constraint ON public.shortcut_building_listings USING btree (shortcut_building_id, shortcut_building_listing_layout, shortcut_building_listing_size, shortcut_building_listing_price, shortcut_building_listing_price_per_sqm, shortcut_building_listing_deleted_at, shortcut_building_listing_marketing_time, shortcut_building_listing_idx);


--
-- Name: shortcut_building_rentals_unique_constraint; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX shortcut_building_rentals_unique_constraint ON public.shortcut_building_rentals USING btree (shortcut_building_id, shortcut_building_rental_layout, shortcut_building_rental_size, shortcut_building_rental_price, shortcut_building_rental_deleted_at, shortcut_building_rental_marketing_time, shortcut_building_rental_idx);


--
-- Name: source_housing_companies_source_key; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX source_housing_companies_source_key ON public.source_housing_companies USING btree (provider, source_kind, raw_table, raw_id);


--
-- Name: source_listings_canonical_source_id_key; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX source_listings_canonical_source_id_key ON public.source_listings USING btree (canonical_source_id);


--
-- Name: source_listings_provider_kind_native_key; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX source_listings_provider_kind_native_key ON public.source_listings USING btree (provider, source_kind, native_id);


--
-- Name: target_observations_active_unique; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX target_observations_active_unique ON public.target_observations USING btree (target_type, target_id, observation_key, source_type, source_id) WHERE (superseded_at IS NULL);


--
-- Name: target_sources_active_source_listing; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX target_sources_active_source_listing ON public.target_sources USING btree (source_id) WHERE ((target_type = 'listing'::text) AND (source_type = 'source_listing'::text) AND (link_status <> 'rejected'::text));


--
-- Name: target_sources_source; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX target_sources_source ON public.target_sources USING btree (source_type, source_id, link_status);


--
-- Name: target_sources_target; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX target_sources_target ON public.target_sources USING btree (target_type, target_id, link_status);


--
-- Name: target_sources_unique_target_source; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX target_sources_unique_target_source ON public.target_sources USING btree (target_type, target_id, source_type, source_id);


--
-- Name: units_identity_key_key; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX units_identity_key_key ON public.units USING btree (identity_key);


--
-- Name: runtime_kv_store_expires_at_idx; Type: INDEX; Schema: runtime; Owner: -
--

CREATE INDEX runtime_kv_store_expires_at_idx ON runtime.kv_store USING btree (expires_at);


--
-- Name: devices devices_user_id_fkey; Type: FK CONSTRAINT; Schema: auth; Owner: -
--

ALTER TABLE ONLY auth.devices
    ADD CONSTRAINT devices_user_id_fkey FOREIGN KEY (user_id) REFERENCES auth.users(user_id) ON DELETE CASCADE;


--
-- Name: sessions fk_sessions_device_id; Type: FK CONSTRAINT; Schema: auth; Owner: -
--

ALTER TABLE ONLY auth.sessions
    ADD CONSTRAINT fk_sessions_device_id FOREIGN KEY (session_device_id) REFERENCES auth.devices(device_id) ON DELETE SET NULL;


--
-- Name: identities identities_user_id_fkey; Type: FK CONSTRAINT; Schema: auth; Owner: -
--

ALTER TABLE ONLY auth.identities
    ADD CONSTRAINT identities_user_id_fkey FOREIGN KEY (user_id) REFERENCES auth.users(user_id) ON DELETE CASCADE;


--
-- Name: refresh_tokens refresh_tokens_session_id_fkey; Type: FK CONSTRAINT; Schema: auth; Owner: -
--

ALTER TABLE ONLY auth.refresh_tokens
    ADD CONSTRAINT refresh_tokens_session_id_fkey FOREIGN KEY (session_id) REFERENCES auth.sessions(session_id) ON DELETE CASCADE;


--
-- Name: role_feature_flags role_feature_flags_flag_id_fkey; Type: FK CONSTRAINT; Schema: auth; Owner: -
--

ALTER TABLE ONLY auth.role_feature_flags
    ADD CONSTRAINT role_feature_flags_flag_id_fkey FOREIGN KEY (flag_id) REFERENCES auth.feature_flags(flag_id) ON DELETE CASCADE;


--
-- Name: role_feature_flags role_feature_flags_role_id_fkey; Type: FK CONSTRAINT; Schema: auth; Owner: -
--

ALTER TABLE ONLY auth.role_feature_flags
    ADD CONSTRAINT role_feature_flags_role_id_fkey FOREIGN KEY (role_id) REFERENCES auth.roles(role_id) ON DELETE CASCADE;


--
-- Name: sessions sessions_user_id_fkey; Type: FK CONSTRAINT; Schema: auth; Owner: -
--

ALTER TABLE ONLY auth.sessions
    ADD CONSTRAINT sessions_user_id_fkey FOREIGN KEY (user_id) REFERENCES auth.users(user_id) ON DELETE CASCADE;


--
-- Name: user_feature_flags user_feature_flags_flag_id_fkey; Type: FK CONSTRAINT; Schema: auth; Owner: -
--

ALTER TABLE ONLY auth.user_feature_flags
    ADD CONSTRAINT user_feature_flags_flag_id_fkey FOREIGN KEY (flag_id) REFERENCES auth.feature_flags(flag_id) ON DELETE CASCADE;


--
-- Name: user_feature_flags user_feature_flags_user_id_fkey; Type: FK CONSTRAINT; Schema: auth; Owner: -
--

ALTER TABLE ONLY auth.user_feature_flags
    ADD CONSTRAINT user_feature_flags_user_id_fkey FOREIGN KEY (user_id) REFERENCES auth.users(user_id) ON DELETE CASCADE;


--
-- Name: user_roles user_roles_role_id_fkey; Type: FK CONSTRAINT; Schema: auth; Owner: -
--

ALTER TABLE ONLY auth.user_roles
    ADD CONSTRAINT user_roles_role_id_fkey FOREIGN KEY (role_id) REFERENCES auth.roles(role_id) ON DELETE CASCADE;


--
-- Name: user_roles user_roles_user_id_fkey; Type: FK CONSTRAINT; Schema: auth; Owner: -
--

ALTER TABLE ONLY auth.user_roles
    ADD CONSTRAINT user_roles_user_id_fkey FOREIGN KEY (user_id) REFERENCES auth.users(user_id) ON DELETE CASCADE;


--
-- Name: auth_webauthn_challenges auth_webauthn_challenges_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.auth_webauthn_challenges
    ADD CONSTRAINT auth_webauthn_challenges_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(user_id) ON DELETE CASCADE;


--
-- Name: device_sessions device_sessions_device_session_user_device_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_sessions
    ADD CONSTRAINT device_sessions_device_session_user_device_id_fkey FOREIGN KEY (device_session_user_device_id) REFERENCES public.user_devices(user_device_id) ON DELETE CASCADE;


--
-- Name: device_sessions device_sessions_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_sessions
    ADD CONSTRAINT device_sessions_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(user_id) ON DELETE CASCADE;


--
-- Name: frontdoor_building_announcements frontdoor_building_announceme_frontdoor_building_announcem_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.frontdoor_building_announcements
    ADD CONSTRAINT frontdoor_building_announceme_frontdoor_building_announcem_fkey FOREIGN KEY (frontdoor_building_id) REFERENCES public.frontdoor_buildings(frontdoor_building_id) ON DELETE CASCADE;


--
-- Name: housing_company_merge_decisions housing_company_merge_decisions_source_housing_company_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.housing_company_merge_decisions
    ADD CONSTRAINT housing_company_merge_decisions_source_housing_company_id_fkey FOREIGN KEY (source_housing_company_id) REFERENCES public.housing_companies(housing_company_id) ON DELETE CASCADE;


--
-- Name: housing_company_merge_decisions housing_company_merge_decisions_target_housing_company_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.housing_company_merge_decisions
    ADD CONSTRAINT housing_company_merge_decisions_target_housing_company_id_fkey FOREIGN KEY (target_housing_company_id) REFERENCES public.housing_companies(housing_company_id) ON DELETE CASCADE;


--
-- Name: oauth_authorization_codes oauth_authorization_codes_user_uuid_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.oauth_authorization_codes
    ADD CONSTRAINT oauth_authorization_codes_user_uuid_fkey FOREIGN KEY (user_uuid) REFERENCES public.users(user_uuid) ON DELETE CASCADE;


--
-- Name: oauth_authorization_handoffs oauth_authorization_handoffs_user_uuid_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.oauth_authorization_handoffs
    ADD CONSTRAINT oauth_authorization_handoffs_user_uuid_fkey FOREIGN KEY (user_uuid) REFERENCES public.users(user_uuid) ON DELETE SET NULL;


--
-- Name: oauth_device_authorizations oauth_device_authorizations_user_uuid_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.oauth_device_authorizations
    ADD CONSTRAINT oauth_device_authorizations_user_uuid_fkey FOREIGN KEY (user_uuid) REFERENCES public.users(user_uuid) ON DELETE SET NULL;


--
-- Name: oauth_refresh_tokens oauth_refresh_tokens_device_session_uuid_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.oauth_refresh_tokens
    ADD CONSTRAINT oauth_refresh_tokens_device_session_uuid_fkey FOREIGN KEY (device_session_uuid) REFERENCES public.device_sessions(device_session_uuid) ON DELETE SET NULL;


--
-- Name: oauth_refresh_tokens oauth_refresh_tokens_oauth_refresh_token_rotated_from_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.oauth_refresh_tokens
    ADD CONSTRAINT oauth_refresh_tokens_oauth_refresh_token_rotated_from_fkey FOREIGN KEY (oauth_refresh_token_rotated_from) REFERENCES public.oauth_refresh_tokens(oauth_refresh_token_id) ON DELETE SET NULL;


--
-- Name: oauth_refresh_tokens oauth_refresh_tokens_user_uuid_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.oauth_refresh_tokens
    ADD CONSTRAINT oauth_refresh_tokens_user_uuid_fkey FOREIGN KEY (user_uuid) REFERENCES public.users(user_uuid) ON DELETE CASCADE;


--
-- Name: personal_access_tokens personal_access_tokens_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.personal_access_tokens
    ADD CONSTRAINT personal_access_tokens_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(user_id) ON DELETE CASCADE;


--
-- Name: physical_buildings physical_buildings_housing_company_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.physical_buildings
    ADD CONSTRAINT physical_buildings_housing_company_id_fkey FOREIGN KEY (housing_company_id) REFERENCES public.housing_companies(housing_company_id) ON DELETE SET NULL;


--
-- Name: postal_postal_codes postal_postal_codes_postal_postal_codes_ad_area_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.postal_postal_codes
    ADD CONSTRAINT postal_postal_codes_postal_postal_codes_ad_area_id_fkey FOREIGN KEY (postal_ad_area_id) REFERENCES public.postal_ad_areas(postal_ad_area_id);


--
-- Name: postal_postal_codes postal_postal_codes_postal_postal_codes_municipality_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.postal_postal_codes
    ADD CONSTRAINT postal_postal_codes_postal_postal_codes_municipality_id_fkey FOREIGN KEY (postal_municipality_id) REFERENCES public.postal_municipalities(postal_municipality_id);


--
-- Name: price_links price_links_prices_transaction_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.price_links
    ADD CONSTRAINT price_links_prices_transaction_id_fkey FOREIGN KEY (prices_transaction_id) REFERENCES public.prices_transactions(prices_transaction_id) ON DELETE CASCADE;


--
-- Name: prices_neighborhoods prices_neighborhoods_prices_neighborhoods_city_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.prices_neighborhoods
    ADD CONSTRAINT prices_neighborhoods_prices_neighborhoods_city_id_fkey FOREIGN KEY (prices_city_id) REFERENCES public.prices_cities(prices_city_id);


--
-- Name: prices_neighborhoods prices_neighborhoods_prices_neighborhoods_postal_code_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.prices_neighborhoods
    ADD CONSTRAINT prices_neighborhoods_prices_neighborhoods_postal_code_id_fkey FOREIGN KEY (prices_postal_code_id) REFERENCES public.prices_postal_codes(prices_postal_code_id);


--
-- Name: prices_neighborhoods prices_neighborhoods_prices_neighborhoods_posti_postal_cod_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.prices_neighborhoods
    ADD CONSTRAINT prices_neighborhoods_prices_neighborhoods_posti_postal_cod_fkey FOREIGN KEY (prices_neighborhood_postal_postal_code_id) REFERENCES public.postal_postal_codes(postal_postal_code_id);


--
-- Name: prices_postal_codes prices_postal_codes_prices_postal_codes_city_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.prices_postal_codes
    ADD CONSTRAINT prices_postal_codes_prices_postal_codes_city_id_fkey FOREIGN KEY (prices_city_id) REFERENCES public.prices_cities(prices_city_id);


--
-- Name: prices_transactions prices_transactions_prices_neighborhoods_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.prices_transactions
    ADD CONSTRAINT prices_transactions_prices_neighborhoods_id_fkey FOREIGN KEY (prices_neighborhood_id) REFERENCES public.prices_neighborhoods(prices_neighborhood_id);


--
-- Name: property_dimension_resolution_policies property_dimension_resolution_policies_dimension_key_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.property_dimension_resolution_policies
    ADD CONSTRAINT property_dimension_resolution_policies_dimension_key_fkey FOREIGN KEY (dimension_key) REFERENCES public.property_dimension_catalog(dimension_key);


--
-- Name: property_dimension_source_priorities property_dimension_source_priorities_dimension_key_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.property_dimension_source_priorities
    ADD CONSTRAINT property_dimension_source_priorities_dimension_key_fkey FOREIGN KEY (dimension_key) REFERENCES public.property_dimension_catalog(dimension_key);


--
-- Name: property_document_extraction_runs property_document_extraction_runs_property_document_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.property_document_extraction_runs
    ADD CONSTRAINT property_document_extraction_runs_property_document_id_fkey FOREIGN KEY (property_document_id) REFERENCES public.property_documents(property_document_id) ON DELETE CASCADE;


--
-- Name: property_document_extractions property_document_extractions_property_document_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.property_document_extractions
    ADD CONSTRAINT property_document_extractions_property_document_id_fkey FOREIGN KEY (property_document_id) REFERENCES public.property_documents(property_document_id) ON DELETE CASCADE;


--
-- Name: property_documents property_documents_housing_company_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.property_documents
    ADD CONSTRAINT property_documents_housing_company_id_fkey FOREIGN KEY (housing_company_id) REFERENCES public.housing_companies(housing_company_id) ON DELETE SET NULL;


--
-- Name: property_documents property_documents_physical_building_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.property_documents
    ADD CONSTRAINT property_documents_physical_building_id_fkey FOREIGN KEY (physical_building_id) REFERENCES public.physical_buildings(physical_building_id) ON DELETE SET NULL;


--
-- Name: property_documents property_documents_property_offering_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.property_documents
    ADD CONSTRAINT property_documents_property_offering_id_fkey FOREIGN KEY (property_offering_id) REFERENCES public.property_offerings(property_offering_id) ON DELETE CASCADE;


--
-- Name: property_documents property_documents_property_unit_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.property_documents
    ADD CONSTRAINT property_documents_property_unit_id_fkey FOREIGN KEY (property_unit_id) REFERENCES public.property_units(property_unit_id) ON DELETE SET NULL;


--
-- Name: property_houses property_houses_primary_sale_listing_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.property_houses
    ADD CONSTRAINT property_houses_primary_sale_listing_id_fkey FOREIGN KEY (primary_sale_listing_id) REFERENCES public.property_source_offerings(sale_listing_id) ON DELETE SET NULL;


--
-- Name: property_offering_merge_decisions property_offering_merge_decisi_source_property_offering_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.property_offering_merge_decisions
    ADD CONSTRAINT property_offering_merge_decisi_source_property_offering_id_fkey FOREIGN KEY (source_property_offering_id) REFERENCES public.property_offerings(property_offering_id) ON DELETE CASCADE;


--
-- Name: property_offering_merge_decisions property_offering_merge_decisi_target_property_offering_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.property_offering_merge_decisions
    ADD CONSTRAINT property_offering_merge_decisi_target_property_offering_id_fkey FOREIGN KEY (target_property_offering_id) REFERENCES public.property_offerings(property_offering_id) ON DELETE CASCADE;


--
-- Name: property_offerings property_offerings_primary_sale_listing_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.property_offerings
    ADD CONSTRAINT property_offerings_primary_sale_listing_id_fkey FOREIGN KEY (primary_sale_listing_id) REFERENCES public.property_source_offerings(sale_listing_id) ON DELETE SET NULL;


--
-- Name: property_offerings property_offerings_property_house_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.property_offerings
    ADD CONSTRAINT property_offerings_property_house_id_fkey FOREIGN KEY (property_house_id) REFERENCES public.property_houses(property_house_id) ON DELETE CASCADE;


--
-- Name: property_offerings property_offerings_property_unit_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.property_offerings
    ADD CONSTRAINT property_offerings_property_unit_id_fkey FOREIGN KEY (property_unit_id) REFERENCES public.property_units(property_unit_id) ON DELETE CASCADE;


--
-- Name: property_renovation_events property_renovation_events_property_dimension_projection_r_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.property_renovation_events
    ADD CONSTRAINT property_renovation_events_property_dimension_projection_r_fkey FOREIGN KEY (property_dimension_projection_run_id) REFERENCES public.property_dimension_projection_runs(property_dimension_projection_run_id) ON DELETE CASCADE;


--
-- Name: property_renovation_events property_renovation_events_source_event_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.property_renovation_events
    ADD CONSTRAINT property_renovation_events_source_event_id_fkey FOREIGN KEY (source_event_id) REFERENCES public.property_renovation_events(property_renovation_event_id);


--
-- Name: property_source_offering_renovations property_source_offering_renovations_sale_listing_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.property_source_offering_renovations
    ADD CONSTRAINT property_source_offering_renovations_sale_listing_id_fkey FOREIGN KEY (sale_listing_id) REFERENCES public.property_source_offerings(sale_listing_id) ON DELETE CASCADE;


--
-- Name: property_units property_units_physical_building_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.property_units
    ADD CONSTRAINT property_units_physical_building_id_fkey FOREIGN KEY (physical_building_id) REFERENCES public.physical_buildings(physical_building_id) ON DELETE SET NULL;


--
-- Name: property_units property_units_property_building_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.property_units
    ADD CONSTRAINT property_units_property_building_id_fkey FOREIGN KEY (housing_company_id) REFERENCES public.housing_companies(housing_company_id) ON DELETE CASCADE;


--
-- Name: role_feature_flags role_feature_flags_flag_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.role_feature_flags
    ADD CONSTRAINT role_feature_flags_flag_id_fkey FOREIGN KEY (flag_id) REFERENCES public.feature_flags(flag_id) ON DELETE CASCADE;


--
-- Name: role_feature_flags role_feature_flags_role_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.role_feature_flags
    ADD CONSTRAINT role_feature_flags_role_id_fkey FOREIGN KEY (role_id) REFERENCES public.roles(role_id) ON DELETE CASCADE;


--
-- Name: sale_listing_prices_transaction_match_candidates sale_listing_prices_transacti_sale_listing_prices_transact_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sale_listing_prices_transaction_match_candidates
    ADD CONSTRAINT sale_listing_prices_transacti_sale_listing_prices_transact_fkey FOREIGN KEY (sale_listing_prices_transaction_match_run_id) REFERENCES public.sale_listing_prices_transaction_match_runs(sale_listing_prices_transaction_match_run_id) ON DELETE CASCADE;


--
-- Name: sale_listing_prices_transaction_match_candidates sale_listing_prices_transaction_matc_prices_transaction_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sale_listing_prices_transaction_match_candidates
    ADD CONSTRAINT sale_listing_prices_transaction_matc_prices_transaction_id_fkey FOREIGN KEY (prices_transaction_id) REFERENCES public.prices_transactions(prices_transaction_id) ON DELETE CASCADE;


--
-- Name: sale_listing_prices_transaction_match_candidates sale_listing_prices_transaction_match_cand_sale_listing_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sale_listing_prices_transaction_match_candidates
    ADD CONSTRAINT sale_listing_prices_transaction_match_cand_sale_listing_id_fkey FOREIGN KEY (sale_listing_id) REFERENCES public.property_source_offerings(sale_listing_id) ON DELETE CASCADE;


--
-- Name: property_source_offerings sale_listings_frontdoor_ad_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.property_source_offerings
    ADD CONSTRAINT sale_listings_frontdoor_ad_id_fkey FOREIGN KEY (frontdoor_ad_id) REFERENCES public.frontdoor_ads(frontdoor_ad_id) ON DELETE SET NULL;


--
-- Name: property_source_offerings sale_listings_frontdoor_building_announcement_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.property_source_offerings
    ADD CONSTRAINT sale_listings_frontdoor_building_announcement_id_fkey FOREIGN KEY (frontdoor_building_announcement_id) REFERENCES public.frontdoor_building_announcements(frontdoor_building_announcement_id) ON DELETE SET NULL;


--
-- Name: property_source_offerings sale_listings_prices_transaction_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.property_source_offerings
    ADD CONSTRAINT sale_listings_prices_transaction_id_fkey FOREIGN KEY (prices_transaction_id) REFERENCES public.prices_transactions(prices_transaction_id) ON DELETE SET NULL;


--
-- Name: property_source_offerings sale_listings_sale_listing_prices_match_run_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.property_source_offerings
    ADD CONSTRAINT sale_listings_sale_listing_prices_match_run_id_fkey FOREIGN KEY (sale_listing_prices_match_run_id) REFERENCES public.sale_listing_prices_transaction_match_runs(sale_listing_prices_transaction_match_run_id) ON DELETE SET NULL;


--
-- Name: property_source_offerings sale_listings_shortcut_ad_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.property_source_offerings
    ADD CONSTRAINT sale_listings_shortcut_ad_id_fkey FOREIGN KEY (shortcut_ad_id) REFERENCES public.shortcut_ads(shortcut_ad_id) ON DELETE SET NULL;


--
-- Name: shortcut_ads shortcut_ads_shortcut_ads_building_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.shortcut_ads
    ADD CONSTRAINT shortcut_ads_shortcut_ads_building_id_fkey FOREIGN KEY (shortcut_building_id) REFERENCES public.shortcut_buildings(shortcut_building_id) ON DELETE SET NULL;


--
-- Name: shortcut_building_listings shortcut_building_listings_shortcut_building_listings_buil_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.shortcut_building_listings
    ADD CONSTRAINT shortcut_building_listings_shortcut_building_listings_buil_fkey FOREIGN KEY (shortcut_building_id) REFERENCES public.shortcut_buildings(shortcut_building_id) ON DELETE CASCADE;


--
-- Name: shortcut_building_rentals shortcut_building_rentals_shortcut_building_rentals_buildi_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.shortcut_building_rentals
    ADD CONSTRAINT shortcut_building_rentals_shortcut_building_rentals_buildi_fkey FOREIGN KEY (shortcut_building_id) REFERENCES public.shortcut_buildings(shortcut_building_id) ON DELETE CASCADE;


--
-- Name: user_devices user_devices_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_devices
    ADD CONSTRAINT user_devices_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(user_id) ON DELETE CASCADE;


--
-- Name: user_email_change_tokens user_email_change_tokens_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_email_change_tokens
    ADD CONSTRAINT user_email_change_tokens_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(user_id) ON DELETE CASCADE;


--
-- Name: user_feature_flags user_feature_flags_flag_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_feature_flags
    ADD CONSTRAINT user_feature_flags_flag_id_fkey FOREIGN KEY (flag_id) REFERENCES public.feature_flags(flag_id) ON DELETE CASCADE;


--
-- Name: user_feature_flags user_feature_flags_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_feature_flags
    ADD CONSTRAINT user_feature_flags_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(user_id) ON DELETE CASCADE;


--
-- Name: user_identities user_identities_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_identities
    ADD CONSTRAINT user_identities_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(user_id) ON DELETE CASCADE;


--
-- Name: user_passkeys user_passkeys_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_passkeys
    ADD CONSTRAINT user_passkeys_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(user_id) ON DELETE CASCADE;


--
-- Name: user_passkeys user_passkeys_user_identity_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_passkeys
    ADD CONSTRAINT user_passkeys_user_identity_id_fkey FOREIGN KEY (user_identity_id) REFERENCES public.user_identities(user_identity_id) ON DELETE CASCADE;


--
-- Name: user_roles user_roles_role_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_roles
    ADD CONSTRAINT user_roles_role_id_fkey FOREIGN KEY (role_id) REFERENCES public.roles(role_id) ON DELETE CASCADE;


--
-- Name: user_roles user_roles_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_roles
    ADD CONSTRAINT user_roles_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(user_id) ON DELETE CASCADE;


--
-- PostgreSQL database dump complete
--



-- Seed/config data preserved from the squashed migrations.

INSERT INTO absurd.queues (queue_name, created_at, storage_mode, default_partition, partition_lookahead, partition_lookback, cleanup_ttl, cleanup_limit, detach_mode, detach_min_age) VALUES ('frontdoor', '2026-06-28 14:39:02.477313+00', 'unpartitioned', 'enabled', '28 days', '1 day', '30 days', 1000, 'none', '30 days');
INSERT INTO absurd.queues (queue_name, created_at, storage_mode, default_partition, partition_lookahead, partition_lookback, cleanup_ttl, cleanup_limit, detach_mode, detach_min_age) VALUES ('shortcut_api', '2026-06-28 14:39:02.507439+00', 'unpartitioned', 'enabled', '28 days', '1 day', '30 days', 1000, 'none', '30 days');
INSERT INTO absurd.queues (queue_name, created_at, storage_mode, default_partition, partition_lookahead, partition_lookback, cleanup_ttl, cleanup_limit, detach_mode, detach_min_age) VALUES ('shortcut_scraper', '2026-06-28 14:39:02.518255+00', 'unpartitioned', 'enabled', '28 days', '1 day', '30 days', 1000, 'none', '30 days');
INSERT INTO absurd.queues (queue_name, created_at, storage_mode, default_partition, partition_lookahead, partition_lookback, cleanup_ttl, cleanup_limit, detach_mode, detach_min_age) VALUES ('prices', '2026-06-28 14:39:02.529161+00', 'unpartitioned', 'enabled', '28 days', '1 day', '30 days', 1000, 'none', '30 days');
INSERT INTO absurd.queues (queue_name, created_at, storage_mode, default_partition, partition_lookahead, partition_lookback, cleanup_ttl, cleanup_limit, detach_mode, detach_min_age) VALUES ('postal', '2026-06-28 14:39:02.538649+00', 'unpartitioned', 'enabled', '28 days', '1 day', '30 days', 1000, 'none', '30 days');
INSERT INTO absurd.queues (queue_name, created_at, storage_mode, default_partition, partition_lookahead, partition_lookback, cleanup_ttl, cleanup_limit, detach_mode, detach_min_age) VALUES ('canonical_db', '2026-06-28 14:39:02.548091+00', 'unpartitioned', 'enabled', '28 days', '1 day', '30 days', 1000, 'none', '30 days');
INSERT INTO absurd.queues (queue_name, created_at, storage_mode, default_partition, partition_lookahead, partition_lookback, cleanup_ttl, cleanup_limit, detach_mode, detach_min_age) VALUES ('canonical_llm', '2026-06-28 14:39:02.556832+00', 'unpartitioned', 'enabled', '28 days', '1 day', '30 days', 1000, 'none', '30 days');
INSERT INTO auth.feature_flags (flag_id, flag_name, flag_description, flag_default_enabled, flag_created_at) VALUES ('defa29a1-0135-4abf-b28d-2764e59e7de0', 'experiment', 'Experimental feature for testing', false, '2026-01-03 08:57:07.962249+00');
INSERT INTO auth.roles (role_id, role_name, role_description, role_created_at) VALUES ('10c34668-6c59-428f-81bb-aa2374725c1c', 'user', 'Default user role', '2026-01-03 08:57:07.962249+00');
INSERT INTO auth.roles (role_id, role_name, role_description, role_created_at) VALUES ('d6109951-4836-4ad0-89bf-98d77de53990', 'admin', 'Administrator with full access', '2026-01-03 08:57:07.962249+00');
INSERT INTO public.energy_efficiency_aliases (energy_efficiency_alias, energy_efficiency_class_code, energy_efficiency_standard_year, energy_efficiency_status, energy_efficiency_match_code, energy_efficiency_label) VALUES ('not_available', NULL, NULL, 'not_available', NULL, 'Not available');
INSERT INTO public.energy_efficiency_aliases (energy_efficiency_alias, energy_efficiency_class_code, energy_efficiency_standard_year, energy_efficiency_status, energy_efficiency_match_code, energy_efficiency_label) VALUES ('no_certificate', NULL, NULL, 'not_available', NULL, 'No certificate');
INSERT INTO public.energy_efficiency_aliases (energy_efficiency_alias, energy_efficiency_class_code, energy_efficiency_standard_year, energy_efficiency_status, energy_efficiency_match_code, energy_efficiency_label) VALUES ('ei_energiatodistusta', NULL, NULL, 'not_available', NULL, 'Ei energiatodistusta');
INSERT INTO public.energy_efficiency_aliases (energy_efficiency_alias, energy_efficiency_class_code, energy_efficiency_standard_year, energy_efficiency_status, energy_efficiency_match_code, energy_efficiency_label) VALUES ('energiatodistus_on', NULL, NULL, 'known', NULL, 'Energiatodistus on');
INSERT INTO public.energy_efficiency_aliases (energy_efficiency_alias, energy_efficiency_class_code, energy_efficiency_standard_year, energy_efficiency_status, energy_efficiency_match_code, energy_efficiency_label) VALUES ('ei_lain_edellyttämää_energiatodistusta', NULL, NULL, 'not_required', NULL, 'Ei lain edellyttämää energiatodistusta');
INSERT INTO public.energy_efficiency_aliases (energy_efficiency_alias, energy_efficiency_class_code, energy_efficiency_standard_year, energy_efficiency_status, energy_efficiency_match_code, energy_efficiency_label) VALUES ('ei_lain_edellytta_ma_a_energiatodistusta', NULL, NULL, 'not_required', NULL, 'Ei lain edellyttämää energiatodistusta');
INSERT INTO public.energy_efficiency_aliases (energy_efficiency_alias, energy_efficiency_class_code, energy_efficiency_standard_year, energy_efficiency_status, energy_efficiency_match_code, energy_efficiency_label) VALUES ('ei_lain_vaatimaa_energiatodistusta', NULL, NULL, 'not_required', NULL, 'Ei lain vaatimaa energiatodistusta');
INSERT INTO public.energy_efficiency_aliases (energy_efficiency_alias, energy_efficiency_class_code, energy_efficiency_standard_year, energy_efficiency_status, energy_efficiency_match_code, energy_efficiency_label) VALUES ('not_required', NULL, NULL, 'not_required', NULL, 'Not required');
INSERT INTO public.energy_efficiency_aliases (energy_efficiency_alias, energy_efficiency_class_code, energy_efficiency_standard_year, energy_efficiency_status, energy_efficiency_match_code, energy_efficiency_label) VALUES ('laki_ei_edellytä_energiatodistusta', NULL, NULL, 'not_required', NULL, 'Laki ei edellytä energiatodistusta');
INSERT INTO public.energy_efficiency_aliases (energy_efficiency_alias, energy_efficiency_class_code, energy_efficiency_standard_year, energy_efficiency_status, energy_efficiency_match_code, energy_efficiency_label) VALUES ('kohteelle_ei_lain_mukaan_tarvita_energiatodistusta', NULL, NULL, 'not_required', NULL, 'Kohteelle ei lain mukaan tarvita energiatodistusta');
INSERT INTO public.energy_efficiency_aliases (energy_efficiency_alias, energy_efficiency_class_code, energy_efficiency_standard_year, energy_efficiency_status, energy_efficiency_match_code, energy_efficiency_label) VALUES ('energiatodistusta_ei_vaadita_kohteelle_ei_lain_mukaan_tarvitse_hankkia_energiatodistusta', NULL, NULL, 'not_required', NULL, 'Energiatodistusta ei vaadita');
INSERT INTO public.energy_efficiency_aliases (energy_efficiency_alias, energy_efficiency_class_code, energy_efficiency_standard_year, energy_efficiency_status, energy_efficiency_match_code, energy_efficiency_label) VALUES ('kohteella_ei_energiatodistuslain_nojalla_tarvitse_olla_energiatodistusta', NULL, NULL, 'not_required', NULL, 'Kohteella ei tarvitse olla energiatodistusta');
INSERT INTO public.energy_efficiency_aliases (energy_efficiency_alias, energy_efficiency_class_code, energy_efficiency_standard_year, energy_efficiency_status, energy_efficiency_match_code, energy_efficiency_label) VALUES ('kohteella_ei_ole_lain_edellyttämää_energiatodistusta_ja_sen_vuoksi_energialuokka_ei_ole_tiedossa', NULL, NULL, 'not_required', NULL, 'Kohteella ei ole lain edellyttämää energiatodistusta');
INSERT INTO public.energy_efficiency_aliases (energy_efficiency_alias, energy_efficiency_class_code, energy_efficiency_standard_year, energy_efficiency_status, energy_efficiency_match_code, energy_efficiency_label) VALUES ('ei_energiatodistusta_kohteella_ei_ole_lain_edellyttämää_energiatodistusta_ja_sen_vuoksi_energialuokka_ei_ole_tiedossa', NULL, NULL, 'not_required', NULL, 'Ei lain edellyttämää energiatodistusta');
INSERT INTO public.property_dimension_catalog (dimension_key, target_type, value_kind, unit, profile_section, profile_key, promoted_to_valuation, created_at, updated_at) VALUES ('unit.area_m2', 'unit', 'number', 'm2', 'unit', 'area_m2', true, '2026-05-09 06:01:09.177383+00', '2026-05-09 06:01:09.177383+00');
INSERT INTO public.property_dimension_catalog (dimension_key, target_type, value_kind, unit, profile_section, profile_key, promoted_to_valuation, created_at, updated_at) VALUES ('unit.living_area_m2', 'unit', 'number', 'm2', 'unit', 'living_area_m2', true, '2026-05-09 06:01:09.177383+00', '2026-05-09 06:01:09.177383+00');
INSERT INTO public.property_dimension_catalog (dimension_key, target_type, value_kind, unit, profile_section, profile_key, promoted_to_valuation, created_at, updated_at) VALUES ('unit.total_area_m2', 'unit', 'number', 'm2', 'unit', 'total_area_m2', true, '2026-05-09 06:01:09.177383+00', '2026-05-09 06:01:09.177383+00');
INSERT INTO public.property_dimension_catalog (dimension_key, target_type, value_kind, unit, profile_section, profile_key, promoted_to_valuation, created_at, updated_at) VALUES ('unit.other_area_m2', 'unit', 'number', 'm2', 'unit', 'other_area_m2', true, '2026-05-09 06:01:09.177383+00', '2026-05-09 06:01:09.177383+00');
INSERT INTO public.property_dimension_catalog (dimension_key, target_type, value_kind, unit, profile_section, profile_key, promoted_to_valuation, created_at, updated_at) VALUES ('unit.floor_level', 'unit', 'number', NULL, 'unit', 'floor_level', true, '2026-05-09 06:01:09.177383+00', '2026-05-09 06:01:09.177383+00');
INSERT INTO public.property_dimension_catalog (dimension_key, target_type, value_kind, unit, profile_section, profile_key, promoted_to_valuation, created_at, updated_at) VALUES ('unit.total_floors', 'unit', 'number', NULL, 'unit', 'total_floors', true, '2026-05-09 06:01:09.177383+00', '2026-05-09 06:01:09.177383+00');
INSERT INTO public.property_dimension_catalog (dimension_key, target_type, value_kind, unit, profile_section, profile_key, promoted_to_valuation, created_at, updated_at) VALUES ('unit.apartment_number', 'unit', 'string', NULL, 'unit', 'apartment_number', false, '2026-05-09 06:01:09.177383+00', '2026-05-09 06:01:09.177383+00');
INSERT INTO public.property_dimension_catalog (dimension_key, target_type, value_kind, unit, profile_section, profile_key, promoted_to_valuation, created_at, updated_at) VALUES ('unit.shares', 'unit', 'string', NULL, 'unit', 'shares', false, '2026-05-09 06:01:09.177383+00', '2026-05-09 06:01:09.177383+00');
INSERT INTO public.property_dimension_catalog (dimension_key, target_type, value_kind, unit, profile_section, profile_key, promoted_to_valuation, created_at, updated_at) VALUES ('layout.room_layout', 'unit', 'string', NULL, 'layout', 'room_layout', true, '2026-05-09 06:01:09.177383+00', '2026-05-09 06:01:09.177383+00');
INSERT INTO public.property_dimension_catalog (dimension_key, target_type, value_kind, unit, profile_section, profile_key, promoted_to_valuation, created_at, updated_at) VALUES ('layout.room_count', 'unit', 'number', NULL, 'layout', 'room_count', true, '2026-05-09 06:01:09.177383+00', '2026-05-09 06:01:09.177383+00');
INSERT INTO public.property_dimension_catalog (dimension_key, target_type, value_kind, unit, profile_section, profile_key, promoted_to_valuation, created_at, updated_at) VALUES ('layout.bedroom_count', 'unit', 'number', NULL, 'layout', 'bedroom_count', true, '2026-05-09 06:01:09.177383+00', '2026-05-09 06:01:09.177383+00');
INSERT INTO public.property_dimension_catalog (dimension_key, target_type, value_kind, unit, profile_section, profile_key, promoted_to_valuation, created_at, updated_at) VALUES ('layout.kitchen_type', 'unit', 'string', NULL, 'layout', 'kitchen_type', true, '2026-05-09 06:01:09.177383+00', '2026-05-09 06:01:09.177383+00');
INSERT INTO public.property_dimension_catalog (dimension_key, target_type, value_kind, unit, profile_section, profile_key, promoted_to_valuation, created_at, updated_at) VALUES ('layout.separate_wc_count', 'unit', 'number', NULL, 'layout', 'separate_wc_count', true, '2026-05-09 06:01:09.177383+00', '2026-05-09 06:01:09.177383+00');
INSERT INTO public.property_dimension_catalog (dimension_key, target_type, value_kind, unit, profile_section, profile_key, promoted_to_valuation, created_at, updated_at) VALUES ('layout.quality', 'unit', 'string', NULL, 'layout', 'quality', true, '2026-05-09 06:01:09.177383+00', '2026-05-09 06:01:09.177383+00');
INSERT INTO public.property_dimension_catalog (dimension_key, target_type, value_kind, unit, profile_section, profile_key, promoted_to_valuation, created_at, updated_at) VALUES ('layout.awkward', 'unit', 'boolean', NULL, 'layout', 'awkward', true, '2026-05-09 06:01:09.177383+00', '2026-05-09 06:01:09.177383+00');
INSERT INTO public.property_dimension_catalog (dimension_key, target_type, value_kind, unit, profile_section, profile_key, promoted_to_valuation, created_at, updated_at) VALUES ('condition.unit_condition', 'unit', 'string', NULL, 'condition', 'unit_condition', true, '2026-05-09 06:01:09.177383+00', '2026-05-09 06:01:09.177383+00');
INSERT INTO public.property_dimension_catalog (dimension_key, target_type, value_kind, unit, profile_section, profile_key, promoted_to_valuation, created_at, updated_at) VALUES ('condition.kitchen_condition', 'unit', 'string', NULL, 'condition', 'kitchen_condition', true, '2026-05-09 06:01:09.177383+00', '2026-05-09 06:01:09.177383+00');
INSERT INTO public.property_dimension_catalog (dimension_key, target_type, value_kind, unit, profile_section, profile_key, promoted_to_valuation, created_at, updated_at) VALUES ('condition.bathroom_condition', 'unit', 'string', NULL, 'condition', 'bathroom_condition', true, '2026-05-09 06:01:09.177383+00', '2026-05-09 06:01:09.177383+00');
INSERT INTO public.property_dimension_catalog (dimension_key, target_type, value_kind, unit, profile_section, profile_key, promoted_to_valuation, created_at, updated_at) VALUES ('condition.surface_renovation_need', 'unit', 'boolean', NULL, 'condition', 'surface_renovation_need', true, '2026-05-09 06:01:09.177383+00', '2026-05-09 06:01:09.177383+00');
INSERT INTO public.property_dimension_catalog (dimension_key, target_type, value_kind, unit, profile_section, profile_key, promoted_to_valuation, created_at, updated_at) VALUES ('condition.modernization_need', 'unit', 'boolean', NULL, 'condition', 'modernization_need', true, '2026-05-09 06:01:09.177383+00', '2026-05-09 06:01:09.177383+00');
INSERT INTO public.property_dimension_catalog (dimension_key, target_type, value_kind, unit, profile_section, profile_key, promoted_to_valuation, created_at, updated_at) VALUES ('features.sauna', 'unit', 'boolean', NULL, 'features', 'sauna', true, '2026-05-09 06:01:09.177383+00', '2026-05-09 06:01:09.177383+00');
INSERT INTO public.property_dimension_catalog (dimension_key, target_type, value_kind, unit, profile_section, profile_key, promoted_to_valuation, created_at, updated_at) VALUES ('features.private_sauna', 'unit', 'boolean', NULL, 'features', 'private_sauna', true, '2026-05-09 06:01:09.177383+00', '2026-05-09 06:01:09.177383+00');
INSERT INTO public.property_dimension_catalog (dimension_key, target_type, value_kind, unit, profile_section, profile_key, promoted_to_valuation, created_at, updated_at) VALUES ('features.balcony', 'unit', 'boolean', NULL, 'features', 'balcony', true, '2026-05-09 06:01:09.177383+00', '2026-05-09 06:01:09.177383+00');
INSERT INTO public.property_dimension_catalog (dimension_key, target_type, value_kind, unit, profile_section, profile_key, promoted_to_valuation, created_at, updated_at) VALUES ('features.balcony_glazing', 'unit', 'boolean', NULL, 'features', 'balcony_glazing', true, '2026-05-09 06:01:09.177383+00', '2026-05-09 06:01:09.177383+00');
INSERT INTO public.property_dimension_catalog (dimension_key, target_type, value_kind, unit, profile_section, profile_key, promoted_to_valuation, created_at, updated_at) VALUES ('features.parking_type', 'unit', 'string', NULL, 'features', 'parking_type', true, '2026-05-09 06:01:09.177383+00', '2026-05-09 06:01:09.177383+00');
INSERT INTO public.property_dimension_catalog (dimension_key, target_type, value_kind, unit, profile_section, profile_key, promoted_to_valuation, created_at, updated_at) VALUES ('features.storage_quality', 'unit', 'string', NULL, 'features', 'storage_quality', true, '2026-05-09 06:01:09.177383+00', '2026-05-09 06:01:09.177383+00');
INSERT INTO public.property_dimension_catalog (dimension_key, target_type, value_kind, unit, profile_section, profile_key, promoted_to_valuation, created_at, updated_at) VALUES ('features.view_quality', 'unit', 'string', NULL, 'features', 'view_quality', true, '2026-05-09 06:01:09.177383+00', '2026-05-09 06:01:09.177383+00');
INSERT INTO public.property_dimension_catalog (dimension_key, target_type, value_kind, unit, profile_section, profile_key, promoted_to_valuation, created_at, updated_at) VALUES ('features.noise_risk', 'unit', 'boolean', NULL, 'features', 'noise_risk', true, '2026-05-09 06:01:09.177383+00', '2026-05-09 06:01:09.177383+00');
INSERT INTO public.property_dimension_catalog (dimension_key, target_type, value_kind, unit, profile_section, profile_key, promoted_to_valuation, created_at, updated_at) VALUES ('features.accessibility', 'unit', 'string', NULL, 'features', 'accessibility', true, '2026-05-09 06:01:09.177383+00', '2026-05-09 06:01:09.177383+00');
INSERT INTO public.property_dimension_catalog (dimension_key, target_type, value_kind, unit, profile_section, profile_key, promoted_to_valuation, created_at, updated_at) VALUES ('charges.maintenance_monthly_eur', 'unit', 'number', 'eur/month', 'charges', 'maintenance_monthly_eur', true, '2026-05-09 06:01:09.177383+00', '2026-05-09 06:01:09.177383+00');
INSERT INTO public.property_dimension_catalog (dimension_key, target_type, value_kind, unit, profile_section, profile_key, promoted_to_valuation, created_at, updated_at) VALUES ('charges.capital_monthly_eur', 'unit', 'number', 'eur/month', 'charges', 'capital_monthly_eur', true, '2026-05-09 06:01:09.177383+00', '2026-05-09 06:01:09.177383+00');
INSERT INTO public.property_dimension_catalog (dimension_key, target_type, value_kind, unit, profile_section, profile_key, promoted_to_valuation, created_at, updated_at) VALUES ('charges.total_monthly_eur', 'unit', 'number', 'eur/month', 'charges', 'total_monthly_eur', true, '2026-05-09 06:01:09.177383+00', '2026-05-09 06:01:09.177383+00');
INSERT INTO public.property_dimension_catalog (dimension_key, target_type, value_kind, unit, profile_section, profile_key, promoted_to_valuation, created_at, updated_at) VALUES ('charges.water_monthly_eur', 'unit', 'number', 'eur/month', 'charges', 'water_monthly_eur', true, '2026-05-09 06:01:09.177383+00', '2026-05-09 06:01:09.177383+00');
INSERT INTO public.property_dimension_catalog (dimension_key, target_type, value_kind, unit, profile_section, profile_key, promoted_to_valuation, created_at, updated_at) VALUES ('charges.parking_monthly_eur', 'unit', 'number', 'eur/month', 'charges', 'parking_monthly_eur', true, '2026-05-09 06:01:09.177383+00', '2026-05-09 06:01:09.177383+00');
INSERT INTO public.property_dimension_catalog (dimension_key, target_type, value_kind, unit, profile_section, profile_key, promoted_to_valuation, created_at, updated_at) VALUES ('charges.debt_share_eur', 'unit', 'number', 'eur', 'charges', 'debt_share_eur', true, '2026-05-09 06:01:09.177383+00', '2026-05-09 06:01:09.177383+00');
INSERT INTO public.property_dimension_catalog (dimension_key, target_type, value_kind, unit, profile_section, profile_key, promoted_to_valuation, created_at, updated_at) VALUES ('charges.charge_risk', 'unit', 'string', NULL, 'charges', 'charge_risk', true, '2026-05-09 06:01:09.177383+00', '2026-05-09 06:01:09.177383+00');
INSERT INTO public.property_dimension_catalog (dimension_key, target_type, value_kind, unit, profile_section, profile_key, promoted_to_valuation, created_at, updated_at) VALUES ('risk.shareholder_liability', 'unit', 'string', NULL, 'risk', 'shareholder_liability', true, '2026-05-09 06:01:09.177383+00', '2026-05-09 06:01:09.177383+00');
INSERT INTO public.property_dimension_catalog (dimension_key, target_type, value_kind, unit, profile_section, profile_key, promoted_to_valuation, created_at, updated_at) VALUES ('building.build_year', 'building', 'number', NULL, 'building', 'build_year', true, '2026-05-09 06:01:09.177383+00', '2026-05-09 06:01:09.177383+00');
INSERT INTO public.property_dimension_catalog (dimension_key, target_type, value_kind, unit, profile_section, profile_key, promoted_to_valuation, created_at, updated_at) VALUES ('building.floor_count', 'building', 'number', NULL, 'building', 'floor_count', true, '2026-05-09 06:01:09.177383+00', '2026-05-09 06:01:09.177383+00');
INSERT INTO public.property_dimension_catalog (dimension_key, target_type, value_kind, unit, profile_section, profile_key, promoted_to_valuation, created_at, updated_at) VALUES ('building.apartment_count', 'building', 'number', NULL, 'building', 'apartment_count', true, '2026-05-09 06:01:09.177383+00', '2026-05-09 06:01:09.177383+00');
INSERT INTO public.property_dimension_catalog (dimension_key, target_type, value_kind, unit, profile_section, profile_key, promoted_to_valuation, created_at, updated_at) VALUES ('building.elevator', 'building', 'boolean', NULL, 'building', 'elevator', true, '2026-05-09 06:01:09.177383+00', '2026-05-09 06:01:09.177383+00');
INSERT INTO public.property_dimension_catalog (dimension_key, target_type, value_kind, unit, profile_section, profile_key, promoted_to_valuation, created_at, updated_at) VALUES ('building.energy_class', 'building', 'string', NULL, 'building', 'energy_class', true, '2026-05-09 06:01:09.177383+00', '2026-05-09 06:01:09.177383+00');
INSERT INTO public.property_dimension_catalog (dimension_key, target_type, value_kind, unit, profile_section, profile_key, promoted_to_valuation, created_at, updated_at) VALUES ('building.heating_method', 'building', 'string', NULL, 'building', 'heating_method', true, '2026-05-09 06:01:09.177383+00', '2026-05-09 06:01:09.177383+00');
INSERT INTO public.property_dimension_catalog (dimension_key, target_type, value_kind, unit, profile_section, profile_key, promoted_to_valuation, created_at, updated_at) VALUES ('building.material', 'building', 'string', NULL, 'building', 'material', true, '2026-05-09 06:01:09.177383+00', '2026-05-09 06:01:09.177383+00');
INSERT INTO public.property_dimension_catalog (dimension_key, target_type, value_kind, unit, profile_section, profile_key, promoted_to_valuation, created_at, updated_at) VALUES ('building.roof_type', 'building', 'string', NULL, 'building', 'roof_type', true, '2026-05-09 06:01:09.177383+00', '2026-05-09 06:01:09.177383+00');
INSERT INTO public.property_dimension_catalog (dimension_key, target_type, value_kind, unit, profile_section, profile_key, promoted_to_valuation, created_at, updated_at) VALUES ('building.roof_material', 'building', 'string', NULL, 'building', 'roof_material', true, '2026-05-09 06:01:09.177383+00', '2026-05-09 06:01:09.177383+00');
INSERT INTO public.property_dimension_catalog (dimension_key, target_type, value_kind, unit, profile_section, profile_key, promoted_to_valuation, created_at, updated_at) VALUES ('building.common_area_quality', 'building', 'string', NULL, 'building', 'common_area_quality', true, '2026-05-09 06:01:09.177383+00', '2026-05-09 06:01:09.177383+00');
INSERT INTO public.property_dimension_catalog (dimension_key, target_type, value_kind, unit, profile_section, profile_key, promoted_to_valuation, created_at, updated_at) VALUES ('building.accessibility', 'building', 'string', NULL, 'building', 'accessibility', true, '2026-05-09 06:01:09.177383+00', '2026-05-09 06:01:09.177383+00');
INSERT INTO public.property_dimension_catalog (dimension_key, target_type, value_kind, unit, profile_section, profile_key, promoted_to_valuation, created_at, updated_at) VALUES ('housing_company.name', 'housing_company', 'string', NULL, 'housing_company', 'name', false, '2026-05-09 06:01:09.177383+00', '2026-05-09 06:01:09.177383+00');
INSERT INTO public.property_dimension_catalog (dimension_key, target_type, value_kind, unit, profile_section, profile_key, promoted_to_valuation, created_at, updated_at) VALUES ('housing_company.business_id', 'housing_company', 'string', NULL, 'housing_company', 'business_id', false, '2026-05-09 06:01:09.177383+00', '2026-05-09 06:01:09.177383+00');
INSERT INTO public.property_dimension_catalog (dimension_key, target_type, value_kind, unit, profile_section, profile_key, promoted_to_valuation, created_at, updated_at) VALUES ('housing_company.apartment_count', 'housing_company', 'number', NULL, 'housing_company', 'apartment_count', true, '2026-05-09 06:01:09.177383+00', '2026-05-09 06:01:09.177383+00');
INSERT INTO public.property_dimension_catalog (dimension_key, target_type, value_kind, unit, profile_section, profile_key, promoted_to_valuation, created_at, updated_at) VALUES ('housing_company.building_count', 'housing_company', 'number', NULL, 'housing_company', 'building_count', false, '2026-05-09 06:01:09.177383+00', '2026-05-09 06:01:09.177383+00');
INSERT INTO public.property_dimension_catalog (dimension_key, target_type, value_kind, unit, profile_section, profile_key, promoted_to_valuation, created_at, updated_at) VALUES ('site.plot_ownership_type', 'housing_company', 'string', NULL, 'site', 'plot_ownership_type', true, '2026-05-09 06:01:09.177383+00', '2026-05-09 06:01:09.177383+00');
INSERT INTO public.property_dimension_catalog (dimension_key, target_type, value_kind, unit, profile_section, profile_key, promoted_to_valuation, created_at, updated_at) VALUES ('site.plot_lease_end_year', 'housing_company', 'number', NULL, 'site', 'plot_lease_end_year', false, '2026-05-09 06:01:09.177383+00', '2026-05-09 06:01:09.177383+00');
INSERT INTO public.property_dimension_catalog (dimension_key, target_type, value_kind, unit, profile_section, profile_key, promoted_to_valuation, created_at, updated_at) VALUES ('site.plot_redemption_possible', 'housing_company', 'boolean', NULL, 'site', 'plot_redemption_possible', false, '2026-05-09 06:01:09.177383+00', '2026-05-09 06:01:09.177383+00');
INSERT INTO public.property_dimension_catalog (dimension_key, target_type, value_kind, unit, profile_section, profile_key, promoted_to_valuation, created_at, updated_at) VALUES ('risk.financial_risk', 'housing_company', 'string', NULL, 'risk', 'financial_risk', true, '2026-05-09 06:01:09.177383+00', '2026-05-09 06:01:09.177383+00');
INSERT INTO public.property_dimension_catalog (dimension_key, target_type, value_kind, unit, profile_section, profile_key, promoted_to_valuation, created_at, updated_at) VALUES ('risk.maintenance_risk', 'housing_company', 'string', NULL, 'risk', 'maintenance_risk', true, '2026-05-09 06:01:09.177383+00', '2026-05-09 06:01:09.177383+00');
INSERT INTO public.property_dimension_catalog (dimension_key, target_type, value_kind, unit, profile_section, profile_key, promoted_to_valuation, created_at, updated_at) VALUES ('risk.repair_backlog_risk', 'housing_company', 'string', NULL, 'risk', 'repair_backlog_risk', true, '2026-05-09 06:01:09.177383+00', '2026-05-09 06:01:09.177383+00');
INSERT INTO public.property_dimension_catalog (dimension_key, target_type, value_kind, unit, profile_section, profile_key, promoted_to_valuation, created_at, updated_at) VALUES ('risk.administrative_legal_risk', 'housing_company', 'string', NULL, 'risk', 'administrative_legal_risk', false, '2026-05-09 06:01:09.177383+00', '2026-05-09 06:01:09.177383+00');
INSERT INTO public.property_dimension_catalog (dimension_key, target_type, value_kind, unit, profile_section, profile_key, promoted_to_valuation, created_at, updated_at) VALUES ('risk.restrictions', 'housing_company', 'array', NULL, 'risk', 'restrictions', false, '2026-05-09 06:01:09.177383+00', '2026-05-09 06:01:09.177383+00');
INSERT INTO public.property_dimension_resolution_policies (dimension_key, strategy, freshness_half_life_days, conflict_tolerance, created_at, updated_at) VALUES ('unit.area_m2', 'numeric_consensus', NULL, '{}', '2026-05-09 06:01:09.177383+00', '2026-05-10 06:23:29.531872+00');
INSERT INTO public.property_dimension_resolution_policies (dimension_key, strategy, freshness_half_life_days, conflict_tolerance, created_at, updated_at) VALUES ('unit.living_area_m2', 'numeric_consensus', NULL, '{}', '2026-05-09 06:01:09.177383+00', '2026-05-10 06:23:29.531872+00');
INSERT INTO public.property_dimension_resolution_policies (dimension_key, strategy, freshness_half_life_days, conflict_tolerance, created_at, updated_at) VALUES ('unit.total_area_m2', 'numeric_consensus', NULL, '{}', '2026-05-09 06:01:09.177383+00', '2026-05-10 06:23:29.531872+00');
INSERT INTO public.property_dimension_resolution_policies (dimension_key, strategy, freshness_half_life_days, conflict_tolerance, created_at, updated_at) VALUES ('unit.other_area_m2', 'numeric_consensus', NULL, '{}', '2026-05-09 06:01:09.177383+00', '2026-05-10 06:23:29.531872+00');
INSERT INTO public.property_dimension_resolution_policies (dimension_key, strategy, freshness_half_life_days, conflict_tolerance, created_at, updated_at) VALUES ('unit.floor_level', 'numeric_consensus', NULL, '{}', '2026-05-09 06:01:09.177383+00', '2026-05-10 06:23:29.531872+00');
INSERT INTO public.property_dimension_resolution_policies (dimension_key, strategy, freshness_half_life_days, conflict_tolerance, created_at, updated_at) VALUES ('unit.total_floors', 'numeric_consensus', NULL, '{}', '2026-05-09 06:01:09.177383+00', '2026-05-10 06:23:29.531872+00');
INSERT INTO public.property_dimension_resolution_policies (dimension_key, strategy, freshness_half_life_days, conflict_tolerance, created_at, updated_at) VALUES ('unit.apartment_number', 'stable_identity', NULL, '{}', '2026-05-09 06:01:09.177383+00', '2026-05-10 06:23:29.531872+00');
INSERT INTO public.property_dimension_resolution_policies (dimension_key, strategy, freshness_half_life_days, conflict_tolerance, created_at, updated_at) VALUES ('unit.shares', 'stable_identity', NULL, '{}', '2026-05-09 06:01:09.177383+00', '2026-05-10 06:23:29.531872+00');
INSERT INTO public.property_dimension_resolution_policies (dimension_key, strategy, freshness_half_life_days, conflict_tolerance, created_at, updated_at) VALUES ('layout.room_layout', 'latest_reliable', NULL, '{}', '2026-05-09 06:01:09.177383+00', '2026-05-10 06:23:29.531872+00');
INSERT INTO public.property_dimension_resolution_policies (dimension_key, strategy, freshness_half_life_days, conflict_tolerance, created_at, updated_at) VALUES ('layout.room_count', 'numeric_consensus', NULL, '{}', '2026-05-09 06:01:09.177383+00', '2026-05-10 06:23:29.531872+00');
INSERT INTO public.property_dimension_resolution_policies (dimension_key, strategy, freshness_half_life_days, conflict_tolerance, created_at, updated_at) VALUES ('layout.bedroom_count', 'numeric_consensus', NULL, '{}', '2026-05-09 06:01:09.177383+00', '2026-05-10 06:23:29.531872+00');
INSERT INTO public.property_dimension_resolution_policies (dimension_key, strategy, freshness_half_life_days, conflict_tolerance, created_at, updated_at) VALUES ('layout.kitchen_type', 'latest_reliable', NULL, '{}', '2026-05-09 06:01:09.177383+00', '2026-05-10 06:23:29.531872+00');
INSERT INTO public.property_dimension_resolution_policies (dimension_key, strategy, freshness_half_life_days, conflict_tolerance, created_at, updated_at) VALUES ('layout.separate_wc_count', 'numeric_consensus', NULL, '{}', '2026-05-09 06:01:09.177383+00', '2026-05-10 06:23:29.531872+00');
INSERT INTO public.property_dimension_resolution_policies (dimension_key, strategy, freshness_half_life_days, conflict_tolerance, created_at, updated_at) VALUES ('layout.quality', 'latest_reliable', NULL, '{}', '2026-05-09 06:01:09.177383+00', '2026-05-10 06:23:29.531872+00');
INSERT INTO public.property_dimension_resolution_policies (dimension_key, strategy, freshness_half_life_days, conflict_tolerance, created_at, updated_at) VALUES ('layout.awkward', 'latest_reliable', NULL, '{}', '2026-05-09 06:01:09.177383+00', '2026-05-10 06:23:29.531872+00');
INSERT INTO public.property_dimension_resolution_policies (dimension_key, strategy, freshness_half_life_days, conflict_tolerance, created_at, updated_at) VALUES ('condition.unit_condition', 'latest_reliable', 365, '{"reason": "listing condition can reflect work after certificate issue date", "newer_listing_can_override_days": 180}', '2026-05-09 06:01:09.177383+00', '2026-05-10 06:23:29.531872+00');
INSERT INTO public.property_dimension_resolution_policies (dimension_key, strategy, freshness_half_life_days, conflict_tolerance, created_at, updated_at) VALUES ('condition.kitchen_condition', 'latest_reliable', 365, '{"reason": "listing condition can reflect work after certificate issue date", "newer_listing_can_override_days": 180}', '2026-05-09 06:01:09.177383+00', '2026-05-10 06:23:29.531872+00');
INSERT INTO public.property_dimension_resolution_policies (dimension_key, strategy, freshness_half_life_days, conflict_tolerance, created_at, updated_at) VALUES ('condition.bathroom_condition', 'latest_reliable', 365, '{"reason": "listing condition can reflect work after certificate issue date", "newer_listing_can_override_days": 180}', '2026-05-09 06:01:09.177383+00', '2026-05-10 06:23:29.531872+00');
INSERT INTO public.property_dimension_resolution_policies (dimension_key, strategy, freshness_half_life_days, conflict_tolerance, created_at, updated_at) VALUES ('condition.surface_renovation_need', 'latest_reliable', 365, '{"reason": "listing condition can reflect work after certificate issue date", "newer_listing_can_override_days": 180}', '2026-05-09 06:01:09.177383+00', '2026-05-10 06:23:29.531872+00');
INSERT INTO public.property_dimension_resolution_policies (dimension_key, strategy, freshness_half_life_days, conflict_tolerance, created_at, updated_at) VALUES ('condition.modernization_need', 'latest_reliable', 365, '{"reason": "listing condition can reflect work after certificate issue date", "newer_listing_can_override_days": 180}', '2026-05-09 06:01:09.177383+00', '2026-05-10 06:23:29.531872+00');
INSERT INTO public.property_dimension_resolution_policies (dimension_key, strategy, freshness_half_life_days, conflict_tolerance, created_at, updated_at) VALUES ('features.sauna', 'latest_reliable', NULL, '{}', '2026-05-09 06:01:09.177383+00', '2026-05-10 06:23:29.531872+00');
INSERT INTO public.property_dimension_resolution_policies (dimension_key, strategy, freshness_half_life_days, conflict_tolerance, created_at, updated_at) VALUES ('features.private_sauna', 'latest_reliable', NULL, '{}', '2026-05-09 06:01:09.177383+00', '2026-05-10 06:23:29.531872+00');
INSERT INTO public.property_dimension_resolution_policies (dimension_key, strategy, freshness_half_life_days, conflict_tolerance, created_at, updated_at) VALUES ('features.balcony', 'latest_reliable', NULL, '{}', '2026-05-09 06:01:09.177383+00', '2026-05-10 06:23:29.531872+00');
INSERT INTO public.property_dimension_resolution_policies (dimension_key, strategy, freshness_half_life_days, conflict_tolerance, created_at, updated_at) VALUES ('features.balcony_glazing', 'latest_reliable', NULL, '{}', '2026-05-09 06:01:09.177383+00', '2026-05-10 06:23:29.531872+00');
INSERT INTO public.property_dimension_resolution_policies (dimension_key, strategy, freshness_half_life_days, conflict_tolerance, created_at, updated_at) VALUES ('features.parking_type', 'latest_reliable', NULL, '{}', '2026-05-09 06:01:09.177383+00', '2026-05-10 06:23:29.531872+00');
INSERT INTO public.property_dimension_resolution_policies (dimension_key, strategy, freshness_half_life_days, conflict_tolerance, created_at, updated_at) VALUES ('features.storage_quality', 'latest_reliable', NULL, '{}', '2026-05-09 06:01:09.177383+00', '2026-05-10 06:23:29.531872+00');
INSERT INTO public.property_dimension_resolution_policies (dimension_key, strategy, freshness_half_life_days, conflict_tolerance, created_at, updated_at) VALUES ('features.view_quality', 'latest_reliable', NULL, '{}', '2026-05-09 06:01:09.177383+00', '2026-05-10 06:23:29.531872+00');
INSERT INTO public.property_dimension_resolution_policies (dimension_key, strategy, freshness_half_life_days, conflict_tolerance, created_at, updated_at) VALUES ('features.noise_risk', 'latest_reliable', NULL, '{}', '2026-05-09 06:01:09.177383+00', '2026-05-10 06:23:29.531872+00');
INSERT INTO public.property_dimension_resolution_policies (dimension_key, strategy, freshness_half_life_days, conflict_tolerance, created_at, updated_at) VALUES ('features.accessibility', 'latest_reliable', NULL, '{}', '2026-05-09 06:01:09.177383+00', '2026-05-10 06:23:29.531872+00');
INSERT INTO public.property_dimension_resolution_policies (dimension_key, strategy, freshness_half_life_days, conflict_tolerance, created_at, updated_at) VALUES ('charges.maintenance_monthly_eur', 'latest_reliable', 180, '{"reason": "charges can change after certificate issue date", "newer_listing_can_override_days": 90}', '2026-05-09 06:01:09.177383+00', '2026-05-10 06:23:29.531872+00');
INSERT INTO public.property_dimension_resolution_policies (dimension_key, strategy, freshness_half_life_days, conflict_tolerance, created_at, updated_at) VALUES ('charges.capital_monthly_eur', 'latest_reliable', 180, '{"reason": "charges can change after certificate issue date", "newer_listing_can_override_days": 90}', '2026-05-09 06:01:09.177383+00', '2026-05-10 06:23:29.531872+00');
INSERT INTO public.property_dimension_resolution_policies (dimension_key, strategy, freshness_half_life_days, conflict_tolerance, created_at, updated_at) VALUES ('charges.total_monthly_eur', 'latest_reliable', 180, '{"reason": "charges can change after certificate issue date", "newer_listing_can_override_days": 90}', '2026-05-09 06:01:09.177383+00', '2026-05-10 06:23:29.531872+00');
INSERT INTO public.property_dimension_resolution_policies (dimension_key, strategy, freshness_half_life_days, conflict_tolerance, created_at, updated_at) VALUES ('charges.water_monthly_eur', 'latest_reliable', 180, '{"reason": "charges can change after certificate issue date", "newer_listing_can_override_days": 90}', '2026-05-09 06:01:09.177383+00', '2026-05-10 06:23:29.531872+00');
INSERT INTO public.property_dimension_resolution_policies (dimension_key, strategy, freshness_half_life_days, conflict_tolerance, created_at, updated_at) VALUES ('charges.parking_monthly_eur', 'latest_reliable', 180, '{"reason": "charges can change after certificate issue date", "newer_listing_can_override_days": 90}', '2026-05-09 06:01:09.177383+00', '2026-05-10 06:23:29.531872+00');
INSERT INTO public.property_dimension_resolution_policies (dimension_key, strategy, freshness_half_life_days, conflict_tolerance, created_at, updated_at) VALUES ('charges.debt_share_eur', 'latest_reliable', 180, '{"reason": "charges can change after certificate issue date", "newer_listing_can_override_days": 90}', '2026-05-09 06:01:09.177383+00', '2026-05-10 06:23:29.531872+00');
INSERT INTO public.property_dimension_resolution_policies (dimension_key, strategy, freshness_half_life_days, conflict_tolerance, created_at, updated_at) VALUES ('charges.charge_risk', 'latest_reliable', 180, '{"reason": "charges can change after certificate issue date", "newer_listing_can_override_days": 90}', '2026-05-09 06:01:09.177383+00', '2026-05-10 06:23:29.531872+00');
INSERT INTO public.property_dimension_resolution_policies (dimension_key, strategy, freshness_half_life_days, conflict_tolerance, created_at, updated_at) VALUES ('risk.shareholder_liability', 'document_preferred', NULL, '{}', '2026-05-09 06:01:09.177383+00', '2026-05-10 06:23:29.531872+00');
INSERT INTO public.property_dimension_resolution_policies (dimension_key, strategy, freshness_half_life_days, conflict_tolerance, created_at, updated_at) VALUES ('building.build_year', 'stable_identity', NULL, '{}', '2026-05-09 06:01:09.177383+00', '2026-05-10 06:23:29.531872+00');
INSERT INTO public.property_dimension_resolution_policies (dimension_key, strategy, freshness_half_life_days, conflict_tolerance, created_at, updated_at) VALUES ('building.floor_count', 'document_preferred', NULL, '{}', '2026-05-09 06:01:09.177383+00', '2026-05-10 06:23:29.531872+00');
INSERT INTO public.property_dimension_resolution_policies (dimension_key, strategy, freshness_half_life_days, conflict_tolerance, created_at, updated_at) VALUES ('building.apartment_count', 'document_preferred', NULL, '{}', '2026-05-09 06:01:09.177383+00', '2026-05-10 06:23:29.531872+00');
INSERT INTO public.property_dimension_resolution_policies (dimension_key, strategy, freshness_half_life_days, conflict_tolerance, created_at, updated_at) VALUES ('building.elevator', 'document_preferred', NULL, '{}', '2026-05-09 06:01:09.177383+00', '2026-05-10 06:23:29.531872+00');
INSERT INTO public.property_dimension_resolution_policies (dimension_key, strategy, freshness_half_life_days, conflict_tolerance, created_at, updated_at) VALUES ('building.energy_class', 'document_preferred', 730, '{}', '2026-05-09 06:01:09.177383+00', '2026-05-10 06:23:29.531872+00');
INSERT INTO public.property_dimension_resolution_policies (dimension_key, strategy, freshness_half_life_days, conflict_tolerance, created_at, updated_at) VALUES ('building.heating_method', 'document_preferred', NULL, '{}', '2026-05-09 06:01:09.177383+00', '2026-05-10 06:23:29.531872+00');
INSERT INTO public.property_dimension_resolution_policies (dimension_key, strategy, freshness_half_life_days, conflict_tolerance, created_at, updated_at) VALUES ('building.material', 'document_preferred', NULL, '{}', '2026-05-09 06:01:09.177383+00', '2026-05-10 06:23:29.531872+00');
INSERT INTO public.property_dimension_resolution_policies (dimension_key, strategy, freshness_half_life_days, conflict_tolerance, created_at, updated_at) VALUES ('building.roof_type', 'document_preferred', NULL, '{}', '2026-05-09 06:01:09.177383+00', '2026-05-10 06:23:29.531872+00');
INSERT INTO public.property_dimension_resolution_policies (dimension_key, strategy, freshness_half_life_days, conflict_tolerance, created_at, updated_at) VALUES ('building.roof_material', 'document_preferred', NULL, '{}', '2026-05-09 06:01:09.177383+00', '2026-05-10 06:23:29.531872+00');
INSERT INTO public.property_dimension_resolution_policies (dimension_key, strategy, freshness_half_life_days, conflict_tolerance, created_at, updated_at) VALUES ('building.common_area_quality', 'latest_reliable', NULL, '{}', '2026-05-09 06:01:09.177383+00', '2026-05-10 06:23:29.531872+00');
INSERT INTO public.property_dimension_resolution_policies (dimension_key, strategy, freshness_half_life_days, conflict_tolerance, created_at, updated_at) VALUES ('building.accessibility', 'latest_reliable', NULL, '{}', '2026-05-09 06:01:09.177383+00', '2026-05-10 06:23:29.531872+00');
INSERT INTO public.property_dimension_resolution_policies (dimension_key, strategy, freshness_half_life_days, conflict_tolerance, created_at, updated_at) VALUES ('housing_company.name', 'stable_identity', NULL, '{}', '2026-05-09 06:01:09.177383+00', '2026-05-10 06:23:29.531872+00');
INSERT INTO public.property_dimension_resolution_policies (dimension_key, strategy, freshness_half_life_days, conflict_tolerance, created_at, updated_at) VALUES ('housing_company.business_id', 'stable_identity', NULL, '{}', '2026-05-09 06:01:09.177383+00', '2026-05-10 06:23:29.531872+00');
INSERT INTO public.property_dimension_resolution_policies (dimension_key, strategy, freshness_half_life_days, conflict_tolerance, created_at, updated_at) VALUES ('housing_company.apartment_count', 'numeric_consensus', NULL, '{}', '2026-05-09 06:01:09.177383+00', '2026-05-10 06:23:29.531872+00');
INSERT INTO public.property_dimension_resolution_policies (dimension_key, strategy, freshness_half_life_days, conflict_tolerance, created_at, updated_at) VALUES ('housing_company.building_count', 'numeric_consensus', NULL, '{}', '2026-05-09 06:01:09.177383+00', '2026-05-10 06:23:29.531872+00');
INSERT INTO public.property_dimension_resolution_policies (dimension_key, strategy, freshness_half_life_days, conflict_tolerance, created_at, updated_at) VALUES ('site.plot_ownership_type', 'document_preferred', NULL, '{}', '2026-05-09 06:01:09.177383+00', '2026-05-10 06:23:29.531872+00');
INSERT INTO public.property_dimension_resolution_policies (dimension_key, strategy, freshness_half_life_days, conflict_tolerance, created_at, updated_at) VALUES ('site.plot_lease_end_year', 'document_preferred', NULL, '{}', '2026-05-09 06:01:09.177383+00', '2026-05-10 06:23:29.531872+00');
INSERT INTO public.property_dimension_resolution_policies (dimension_key, strategy, freshness_half_life_days, conflict_tolerance, created_at, updated_at) VALUES ('site.plot_redemption_possible', 'document_preferred', NULL, '{}', '2026-05-09 06:01:09.177383+00', '2026-05-10 06:23:29.531872+00');
INSERT INTO public.property_dimension_resolution_policies (dimension_key, strategy, freshness_half_life_days, conflict_tolerance, created_at, updated_at) VALUES ('risk.financial_risk', 'latest_reliable', 365, '{"reason": "latest listing text may include newer future work or decisions", "newer_listing_can_override_days": 180}', '2026-05-09 06:01:09.177383+00', '2026-05-10 06:23:29.531872+00');
INSERT INTO public.property_dimension_resolution_policies (dimension_key, strategy, freshness_half_life_days, conflict_tolerance, created_at, updated_at) VALUES ('risk.maintenance_risk', 'latest_reliable', 365, '{"reason": "latest listing text may include newer future work or decisions", "newer_listing_can_override_days": 180}', '2026-05-09 06:01:09.177383+00', '2026-05-10 06:23:29.531872+00');
INSERT INTO public.property_dimension_resolution_policies (dimension_key, strategy, freshness_half_life_days, conflict_tolerance, created_at, updated_at) VALUES ('risk.repair_backlog_risk', 'latest_reliable', 365, '{"reason": "latest listing text may include newer future work or decisions", "newer_listing_can_override_days": 180}', '2026-05-09 06:01:09.177383+00', '2026-05-10 06:23:29.531872+00');
INSERT INTO public.property_dimension_resolution_policies (dimension_key, strategy, freshness_half_life_days, conflict_tolerance, created_at, updated_at) VALUES ('risk.administrative_legal_risk', 'document_preferred', NULL, '{}', '2026-05-09 06:01:09.177383+00', '2026-05-10 06:23:29.531872+00');
INSERT INTO public.property_dimension_resolution_policies (dimension_key, strategy, freshness_half_life_days, conflict_tolerance, created_at, updated_at) VALUES ('risk.restrictions', 'document_preferred', NULL, '{}', '2026-05-09 06:01:09.177383+00', '2026-05-10 06:23:29.531872+00');
INSERT INTO public.property_dimension_source_priorities (dimension_key, source_table, source_field, priority, default_reliability) VALUES ('unit.area_m2', 'property_source_offerings', 'sale_listing_area_value', 70, 0.75);
INSERT INTO public.property_dimension_source_priorities (dimension_key, source_table, source_field, priority, default_reliability) VALUES ('unit.living_area_m2', 'property_source_offerings', 'sale_listing_living_area_value', 70, 0.75);
INSERT INTO public.property_dimension_source_priorities (dimension_key, source_table, source_field, priority, default_reliability) VALUES ('unit.total_area_m2', 'property_source_offerings', 'sale_listing_total_area_value', 70, 0.75);
INSERT INTO public.property_dimension_source_priorities (dimension_key, source_table, source_field, priority, default_reliability) VALUES ('unit.other_area_m2', 'property_source_offerings', 'sale_listing_other_area_value', 70, 0.75);
INSERT INTO public.property_dimension_source_priorities (dimension_key, source_table, source_field, priority, default_reliability) VALUES ('layout.room_layout', 'property_source_offerings', 'sale_listing_room_layout', 70, 0.7);
INSERT INTO public.property_dimension_source_priorities (dimension_key, source_table, source_field, priority, default_reliability) VALUES ('layout.room_count', 'property_source_offerings', 'sale_listing_rooms_count', 70, 0.75);
INSERT INTO public.property_dimension_source_priorities (dimension_key, source_table, source_field, priority, default_reliability) VALUES ('layout.bedroom_count', 'property_source_offerings', 'sale_listing_bedrooms_count', 70, 0.7);
INSERT INTO public.property_dimension_source_priorities (dimension_key, source_table, source_field, priority, default_reliability) VALUES ('unit.floor_level', 'property_source_offerings', 'sale_listing_floor_level', 70, 0.7);
INSERT INTO public.property_dimension_source_priorities (dimension_key, source_table, source_field, priority, default_reliability) VALUES ('building.floor_count', 'property_source_offerings', 'sale_listing_total_floors', 65, 0.65);
INSERT INTO public.property_dimension_source_priorities (dimension_key, source_table, source_field, priority, default_reliability) VALUES ('condition.unit_condition', 'property_source_offerings', 'sale_listing_condition', 60, 0.6);
INSERT INTO public.property_dimension_source_priorities (dimension_key, source_table, source_field, priority, default_reliability) VALUES ('features.sauna', 'property_source_offerings', 'sale_listing_sauna', 70, 0.75);
INSERT INTO public.property_dimension_source_priorities (dimension_key, source_table, source_field, priority, default_reliability) VALUES ('features.balcony', 'property_source_offerings', 'sale_listing_balcony', 70, 0.75);
INSERT INTO public.property_dimension_source_priorities (dimension_key, source_table, source_field, priority, default_reliability) VALUES ('features.parking_type', 'property_source_offerings', 'sale_listing_parking_text', 55, 0.55);
INSERT INTO public.property_dimension_source_priorities (dimension_key, source_table, source_field, priority, default_reliability) VALUES ('charges.maintenance_monthly_eur', 'property_source_offerings', 'sale_listing_maintenance_charge_monthly', 70, 0.7);
INSERT INTO public.property_dimension_source_priorities (dimension_key, source_table, source_field, priority, default_reliability) VALUES ('charges.total_monthly_eur', 'property_source_offerings', 'sale_listing_total_charge_monthly', 70, 0.7);
INSERT INTO public.property_dimension_source_priorities (dimension_key, source_table, source_field, priority, default_reliability) VALUES ('charges.water_monthly_eur', 'property_source_offerings', 'sale_listing_water_charge', 60, 0.6);
INSERT INTO public.property_dimension_source_priorities (dimension_key, source_table, source_field, priority, default_reliability) VALUES ('charges.debt_share_eur', 'property_source_offerings', 'sale_listing_debt_share_amount', 70, 0.7);
INSERT INTO public.property_dimension_source_priorities (dimension_key, source_table, source_field, priority, default_reliability) VALUES ('building.build_year', 'property_source_offerings', 'sale_listing_build_year', 65, 0.65);
INSERT INTO public.property_dimension_source_priorities (dimension_key, source_table, source_field, priority, default_reliability) VALUES ('building.elevator', 'property_source_offerings', 'sale_listing_elevator', 65, 0.65);
INSERT INTO public.property_dimension_source_priorities (dimension_key, source_table, source_field, priority, default_reliability) VALUES ('building.heating_method', 'property_source_offerings', 'sale_listing_heating_system', 60, 0.6);
INSERT INTO public.property_dimension_source_priorities (dimension_key, source_table, source_field, priority, default_reliability) VALUES ('building.energy_class', 'property_source_offerings', 'sale_listing_energy_efficiency_label', 60, 0.6);
INSERT INTO public.property_dimension_source_priorities (dimension_key, source_table, source_field, priority, default_reliability) VALUES ('building.material', 'property_source_offerings', 'sale_listing_building_material', 60, 0.6);
INSERT INTO public.property_dimension_source_priorities (dimension_key, source_table, source_field, priority, default_reliability) VALUES ('building.roof_type', 'property_source_offerings', 'sale_listing_roof_type', 60, 0.6);
INSERT INTO public.property_dimension_source_priorities (dimension_key, source_table, source_field, priority, default_reliability) VALUES ('building.roof_material', 'property_source_offerings', 'sale_listing_roof_material', 60, 0.6);
INSERT INTO public.property_dimension_source_priorities (dimension_key, source_table, source_field, priority, default_reliability) VALUES ('housing_company.apartment_count', 'property_source_offerings', 'sale_listing_apartment_count', 60, 0.6);
INSERT INTO public.property_dimension_source_priorities (dimension_key, source_table, source_field, priority, default_reliability) VALUES ('housing_company.name', 'property_source_offerings', 'sale_listing_housing_company_name', 60, 0.6);
INSERT INTO public.property_dimension_source_priorities (dimension_key, source_table, source_field, priority, default_reliability) VALUES ('housing_company.business_id', 'property_source_offerings', 'sale_listing_housing_company_business_id', 65, 0.65);
INSERT INTO public.property_dimension_source_priorities (dimension_key, source_table, source_field, priority, default_reliability) VALUES ('site.plot_ownership_type', 'property_source_offerings', 'sale_listing_plot_type_code', 60, 0.6);
INSERT INTO public.property_dimension_source_priorities (dimension_key, source_table, source_field, priority, default_reliability) VALUES ('unit.apartment_number', 'property_documents', 'unit.apartment_number', 95, 0.9);
INSERT INTO public.property_dimension_source_priorities (dimension_key, source_table, source_field, priority, default_reliability) VALUES ('unit.shares', 'property_documents', 'unit.shares', 95, 0.9);
INSERT INTO public.property_dimension_source_priorities (dimension_key, source_table, source_field, priority, default_reliability) VALUES ('unit.area_m2', 'property_documents', 'unit.area_m2', 92, 0.9);
INSERT INTO public.property_dimension_source_priorities (dimension_key, source_table, source_field, priority, default_reliability) VALUES ('layout.room_layout', 'property_documents', 'unit.room_layout', 88, 0.86);
INSERT INTO public.property_dimension_source_priorities (dimension_key, source_table, source_field, priority, default_reliability) VALUES ('unit.floor_level', 'property_documents', 'unit.floor_level', 88, 0.86);
INSERT INTO public.property_dimension_source_priorities (dimension_key, source_table, source_field, priority, default_reliability) VALUES ('charges.maintenance_monthly_eur', 'property_documents', 'unit.maintenance_charge_monthly', 90, 0.9);
INSERT INTO public.property_dimension_source_priorities (dimension_key, source_table, source_field, priority, default_reliability) VALUES ('charges.capital_monthly_eur', 'property_documents', 'unit.capital_charge_monthly', 90, 0.9);
INSERT INTO public.property_dimension_source_priorities (dimension_key, source_table, source_field, priority, default_reliability) VALUES ('charges.total_monthly_eur', 'property_documents', 'unit.total_charge_monthly', 90, 0.9);
INSERT INTO public.property_dimension_source_priorities (dimension_key, source_table, source_field, priority, default_reliability) VALUES ('charges.debt_share_eur', 'property_documents', 'unit.debt_share_eur', 90, 0.9);
INSERT INTO public.property_dimension_source_priorities (dimension_key, source_table, source_field, priority, default_reliability) VALUES ('risk.shareholder_liability', 'property_documents', 'unit.shareholder_liability', 92, 0.9);
INSERT INTO public.property_dimension_source_priorities (dimension_key, source_table, source_field, priority, default_reliability) VALUES ('housing_company.name', 'property_documents', 'housing_company.name', 96, 0.92);
INSERT INTO public.property_dimension_source_priorities (dimension_key, source_table, source_field, priority, default_reliability) VALUES ('housing_company.business_id', 'property_documents', 'housing_company.business_id', 98, 0.95);
INSERT INTO public.property_dimension_source_priorities (dimension_key, source_table, source_field, priority, default_reliability) VALUES ('housing_company.apartment_count', 'property_documents', 'housing_company.apartment_count', 90, 0.9);
INSERT INTO public.property_dimension_source_priorities (dimension_key, source_table, source_field, priority, default_reliability) VALUES ('site.plot_ownership_type', 'property_documents', 'housing_company.plot_ownership_type', 94, 0.9);
INSERT INTO public.property_dimension_source_priorities (dimension_key, source_table, source_field, priority, default_reliability) VALUES ('building.energy_class', 'property_documents', 'housing_company.energy_class', 90, 0.88);
INSERT INTO public.property_dimension_source_priorities (dimension_key, source_table, source_field, priority, default_reliability) VALUES ('building.build_year', 'property_documents', 'building.build_year', 94, 0.9);
INSERT INTO public.property_dimension_source_priorities (dimension_key, source_table, source_field, priority, default_reliability) VALUES ('building.floor_count', 'property_documents', 'building.floor_count', 92, 0.9);
INSERT INTO public.property_dimension_source_priorities (dimension_key, source_table, source_field, priority, default_reliability) VALUES ('building.apartment_count', 'property_documents', 'building.apartment_count', 92, 0.9);
INSERT INTO public.property_dimension_source_priorities (dimension_key, source_table, source_field, priority, default_reliability) VALUES ('building.energy_class', 'property_documents', 'building.energy_class', 90, 0.88);
INSERT INTO public.property_dimension_source_priorities (dimension_key, source_table, source_field, priority, default_reliability) VALUES ('building.heating_method', 'property_documents', 'building.heating_method', 90, 0.88);
INSERT INTO public.property_dimension_source_priorities (dimension_key, source_table, source_field, priority, default_reliability) VALUES ('building.material', 'property_documents', 'building.material', 90, 0.88);
INSERT INTO public.property_dimension_source_priorities (dimension_key, source_table, source_field, priority, default_reliability) VALUES ('building.roof_type', 'property_documents', 'building.roof_type', 90, 0.88);
INSERT INTO public.property_dimension_source_priorities (dimension_key, source_table, source_field, priority, default_reliability) VALUES ('building.roof_material', 'property_documents', 'building.roof_material', 90, 0.88);
INSERT INTO public.property_dimension_source_priorities (dimension_key, source_table, source_field, priority, default_reliability) VALUES ('building.elevator', 'property_documents', 'building.elevator', 90, 0.88);
INSERT INTO public.property_dimension_source_priorities (dimension_key, source_table, source_field, priority, default_reliability) VALUES ('risk.financial_risk', 'property_documents', 'finances.financial_risk', 88, 0.86);
INSERT INTO public.property_dimension_source_priorities (dimension_key, source_table, source_field, priority, default_reliability) VALUES ('risk.maintenance_risk', 'property_documents', 'finances.maintenance_risk', 88, 0.86);
INSERT INTO public.property_dimension_source_priorities (dimension_key, source_table, source_field, priority, default_reliability) VALUES ('risk.repair_backlog_risk', 'property_documents', 'finances.repair_backlog_risk', 88, 0.86);
INSERT INTO public.property_dimension_source_priorities (dimension_key, source_table, source_field, priority, default_reliability) VALUES ('risk.administrative_legal_risk', 'property_documents', 'risks.administrative_legal_risk', 92, 0.9);
INSERT INTO public.property_dimension_source_priorities (dimension_key, source_table, source_field, priority, default_reliability) VALUES ('risk.restrictions', 'property_documents', 'risks.restrictions', 92, 0.9);
INSERT INTO public.sale_listing_plot_type_aliases (sale_listing_plot_type_alias, sale_listing_plot_type_code, sale_listing_plot_type_label) VALUES ('oma', 'own', 'Oma');
INSERT INTO public.sale_listing_plot_type_aliases (sale_listing_plot_type_alias, sale_listing_plot_type_code, sale_listing_plot_type_label) VALUES ('own', 'own', 'Oma');
INSERT INTO public.sale_listing_plot_type_aliases (sale_listing_plot_type_alias, sale_listing_plot_type_code, sale_listing_plot_type_label) VALUES ('owned', 'own', 'Oma');
INSERT INTO public.sale_listing_plot_type_aliases (sale_listing_plot_type_alias, sale_listing_plot_type_code, sale_listing_plot_type_label) VALUES ('vuokra', 'rent', 'Vuokra');
INSERT INTO public.sale_listing_plot_type_aliases (sale_listing_plot_type_alias, sale_listing_plot_type_code, sale_listing_plot_type_label) VALUES ('rent', 'rent', 'Vuokra');
INSERT INTO public.sale_listing_plot_type_aliases (sale_listing_plot_type_alias, sale_listing_plot_type_code, sale_listing_plot_type_label) VALUES ('rental', 'rent', 'Vuokra');
INSERT INTO public.sale_listing_plot_type_aliases (sale_listing_plot_type_alias, sale_listing_plot_type_code, sale_listing_plot_type_label) VALUES ('lease', 'rent', 'Vuokra');
INSERT INTO public.sale_listing_plot_type_aliases (sale_listing_plot_type_alias, sale_listing_plot_type_code, sale_listing_plot_type_label) VALUES ('leased', 'rent', 'Vuokra');
INSERT INTO public.sale_listing_plot_type_aliases (sale_listing_plot_type_alias, sale_listing_plot_type_code, sale_listing_plot_type_label) VALUES ('optional_rental', 'rent', 'Vuokra');
INSERT INTO public.sale_listing_plot_type_aliases (sale_listing_plot_type_alias, sale_listing_plot_type_code, sale_listing_plot_type_label) VALUES ('valinnainen_vuokratontti', 'rent', 'Vuokra');
INSERT INTO public.sale_listing_plot_type_aliases (sale_listing_plot_type_alias, sale_listing_plot_type_code, sale_listing_plot_type_label) VALUES ('vuokralla', 'rent', 'Vuokra');
INSERT INTO public.sale_listing_property_type_aliases (sale_listing_property_type_alias, sale_listing_property_type_code, sale_listing_property_type_label) VALUES ('kt', 'apartment_block', 'Kerrostalo');
INSERT INTO public.sale_listing_property_type_aliases (sale_listing_property_type_alias, sale_listing_property_type_code, sale_listing_property_type_label) VALUES ('kerrostalo', 'apartment_block', 'Kerrostalo');
INSERT INTO public.sale_listing_property_type_aliases (sale_listing_property_type_alias, sale_listing_property_type_code, sale_listing_property_type_label) VALUES ('apartment', 'apartment_block', 'Kerrostalo');
INSERT INTO public.sale_listing_property_type_aliases (sale_listing_property_type_alias, sale_listing_property_type_code, sale_listing_property_type_label) VALUES ('apartment_house', 'apartment_block', 'Kerrostalo');
INSERT INTO public.sale_listing_property_type_aliases (sale_listing_property_type_alias, sale_listing_property_type_code, sale_listing_property_type_label) VALUES ('apartment_block', 'apartment_block', 'Kerrostalo');
INSERT INTO public.sale_listing_property_type_aliases (sale_listing_property_type_alias, sale_listing_property_type_code, sale_listing_property_type_label) VALUES ('balcony_access_block', 'apartment_block', 'Kerrostalo');
INSERT INTO public.sale_listing_property_type_aliases (sale_listing_property_type_alias, sale_listing_property_type_code, sale_listing_property_type_label) VALUES ('block_of_flats', 'apartment_block', 'Kerrostalo');
INSERT INTO public.sale_listing_property_type_aliases (sale_listing_property_type_alias, sale_listing_property_type_code, sale_listing_property_type_label) VALUES ('flat', 'apartment_block', 'Kerrostalo');
INSERT INTO public.sale_listing_property_type_aliases (sale_listing_property_type_alias, sale_listing_property_type_code, sale_listing_property_type_label) VALUES ('wooden_house_apartment', 'apartment_block', 'Kerrostalo');
INSERT INTO public.sale_listing_property_type_aliases (sale_listing_property_type_alias, sale_listing_property_type_code, sale_listing_property_type_label) VALUES ('1', 'apartment_block', 'Kerrostalo');
INSERT INTO public.sale_listing_property_type_aliases (sale_listing_property_type_alias, sale_listing_property_type_code, sale_listing_property_type_label) VALUES ('rt', 'row_house', 'Rivitalo');
INSERT INTO public.sale_listing_property_type_aliases (sale_listing_property_type_alias, sale_listing_property_type_code, sale_listing_property_type_label) VALUES ('rivitalo', 'row_house', 'Rivitalo');
INSERT INTO public.sale_listing_property_type_aliases (sale_listing_property_type_alias, sale_listing_property_type_code, sale_listing_property_type_label) VALUES ('row_house', 'row_house', 'Rivitalo');
INSERT INTO public.sale_listing_property_type_aliases (sale_listing_property_type_alias, sale_listing_property_type_code, sale_listing_property_type_label) VALUES ('semi_detached_house', 'row_house', 'Rivitalo');
INSERT INTO public.sale_listing_property_type_aliases (sale_listing_property_type_alias, sale_listing_property_type_code, sale_listing_property_type_label) VALUES ('terraced_house', 'row_house', 'Rivitalo');
INSERT INTO public.sale_listing_property_type_aliases (sale_listing_property_type_alias, sale_listing_property_type_code, sale_listing_property_type_label) VALUES ('terrace_house', 'row_house', 'Rivitalo');
INSERT INTO public.sale_listing_property_type_aliases (sale_listing_property_type_alias, sale_listing_property_type_code, sale_listing_property_type_label) VALUES ('2', 'row_house', 'Rivitalo');
INSERT INTO public.sale_listing_property_type_aliases (sale_listing_property_type_alias, sale_listing_property_type_code, sale_listing_property_type_label) VALUES ('ok', 'detached_house', 'Omakotitalo');
INSERT INTO public.sale_listing_property_type_aliases (sale_listing_property_type_alias, sale_listing_property_type_code, sale_listing_property_type_label) VALUES ('omakotitalo', 'detached_house', 'Omakotitalo');
INSERT INTO public.sale_listing_property_type_aliases (sale_listing_property_type_alias, sale_listing_property_type_code, sale_listing_property_type_label) VALUES ('detached_house', 'detached_house', 'Omakotitalo');
INSERT INTO public.sale_listing_property_type_aliases (sale_listing_property_type_alias, sale_listing_property_type_code, sale_listing_property_type_label) VALUES ('separate_house', 'detached_house', 'Omakotitalo');
INSERT INTO public.sale_listing_property_type_aliases (sale_listing_property_type_alias, sale_listing_property_type_code, sale_listing_property_type_label) VALUES ('single_family_house', 'detached_house', 'Omakotitalo');
INSERT INTO public.sale_listing_property_type_aliases (sale_listing_property_type_alias, sale_listing_property_type_code, sale_listing_property_type_label) VALUES ('3', 'detached_house', 'Omakotitalo');
INSERT INTO public.sale_listing_room_category_aliases (sale_listing_room_category_alias, sale_listing_room_category_code, sale_listing_room_category_label) VALUES ('yksiöt', 'one_room', 'Yksiöt');
INSERT INTO public.sale_listing_room_category_aliases (sale_listing_room_category_alias, sale_listing_room_category_code, sale_listing_room_category_label) VALUES ('yksiot', 'one_room', 'Yksiöt');
INSERT INTO public.sale_listing_room_category_aliases (sale_listing_room_category_alias, sale_listing_room_category_code, sale_listing_room_category_label) VALUES ('one_room', 'one_room', 'Yksiöt');
INSERT INTO public.sale_listing_room_category_aliases (sale_listing_room_category_alias, sale_listing_room_category_code, sale_listing_room_category_label) VALUES ('kaksiot', 'two_rooms', 'Kaksiot');
INSERT INTO public.sale_listing_room_category_aliases (sale_listing_room_category_alias, sale_listing_room_category_code, sale_listing_room_category_label) VALUES ('kaksiöt', 'two_rooms', 'Kaksiot');
INSERT INTO public.sale_listing_room_category_aliases (sale_listing_room_category_alias, sale_listing_room_category_code, sale_listing_room_category_label) VALUES ('two_rooms', 'two_rooms', 'Kaksiot');
INSERT INTO public.sale_listing_room_category_aliases (sale_listing_room_category_alias, sale_listing_room_category_code, sale_listing_room_category_label) VALUES ('kolmiot', 'three_rooms', 'Kolmiot');
INSERT INTO public.sale_listing_room_category_aliases (sale_listing_room_category_alias, sale_listing_room_category_code, sale_listing_room_category_label) VALUES ('three_rooms', 'three_rooms', 'Kolmiot');
INSERT INTO public.sale_listing_room_category_aliases (sale_listing_room_category_alias, sale_listing_room_category_code, sale_listing_room_category_label) VALUES ('neljä_huonetta_tai_enemmän', 'four_plus_rooms', 'Neljä huonetta tai enemmän');
INSERT INTO public.sale_listing_room_category_aliases (sale_listing_room_category_alias, sale_listing_room_category_code, sale_listing_room_category_label) VALUES ('nelja_huonetta_tai_enemman', 'four_plus_rooms', 'Neljä huonetta tai enemmän');
INSERT INTO public.sale_listing_room_category_aliases (sale_listing_room_category_alias, sale_listing_room_category_code, sale_listing_room_category_label) VALUES ('four_plus_rooms', 'four_plus_rooms', 'Neljä huonetta tai enemmän');
SELECT pg_catalog.setval('public.feature_flags_flag_id_seq', 1, false);
SELECT pg_catalog.setval('public.roles_role_id_seq', 1, false);
