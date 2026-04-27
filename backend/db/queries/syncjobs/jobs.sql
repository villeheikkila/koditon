-- name: UpsertSyncJobForEnqueue :one
INSERT INTO public.sync_jobs (
    sync_job_provider,
    sync_job_kind,
    sync_job_entity_id,
    sync_job_dedup_key,
    sync_job_status,
    sync_job_priority,
    sync_job_attempt_count,
    sync_job_max_attempts,
    sync_job_run_after,
    sync_job_capacity_class,
    sync_job_payload,
    sync_job_checkpoint,
    sync_job_result,
    sync_job_last_error,
    sync_job_last_error_code,
    sync_job_last_http_status,
    sync_job_claim_token,
    sync_job_last_finished_at,
    sync_job_created_at,
    sync_job_updated_at
) VALUES (
    sqlc.arg(sync_job_provider),
    sqlc.arg(sync_job_kind),
    sqlc.arg(sync_job_entity_id),
    sqlc.arg(sync_job_dedup_key),
    'pending',
    sqlc.arg(sync_job_priority),
    0,
    sqlc.arg(sync_job_max_attempts),
    sqlc.arg(sync_job_run_after),
    sqlc.arg(sync_job_capacity_class),
    sqlc.arg(sync_job_payload),
    NULL,
    NULL,
    NULL,
    NULL,
    NULL,
    NULL,
    NULL,
    now(),
    now()
)
ON CONFLICT (sync_job_dedup_key) DO UPDATE
SET sync_job_provider = EXCLUDED.sync_job_provider,
    sync_job_kind = EXCLUDED.sync_job_kind,
    sync_job_entity_id = EXCLUDED.sync_job_entity_id,
    sync_job_status = 'pending',
    sync_job_priority = EXCLUDED.sync_job_priority,
    sync_job_attempt_count = 0,
    sync_job_max_attempts = EXCLUDED.sync_job_max_attempts,
    sync_job_run_after = EXCLUDED.sync_job_run_after,
    sync_job_capacity_class = EXCLUDED.sync_job_capacity_class,
    sync_job_payload = EXCLUDED.sync_job_payload,
    sync_job_checkpoint = NULL,
    sync_job_result = NULL,
    sync_job_last_error = NULL,
    sync_job_last_error_code = NULL,
    sync_job_last_http_status = NULL,
    sync_job_claim_token = NULL,
    sync_job_last_finished_at = NULL,
    sync_job_updated_at = now()
WHERE sync_jobs.sync_job_status IN ('succeeded', 'failed', 'not_found', 'noop', 'skipped_lock')
   OR (sync_jobs.sync_job_status = 'pending' AND sync_jobs.sync_job_run_after > EXCLUDED.sync_job_run_after)
RETURNING sync_job_id, sync_job_provider, sync_job_kind, sync_job_entity_id, sync_job_dedup_key, sync_job_status, sync_job_priority, sync_job_attempt_count, sync_job_max_attempts, sync_job_run_after, sync_job_capacity_class, sync_job_payload, sync_job_checkpoint, sync_job_result, sync_job_last_error, sync_job_last_error_code, sync_job_last_http_status, sync_job_last_pgmq_message_id, sync_job_claim_token, sync_job_created_at, sync_job_updated_at, sync_job_last_enqueued_at, sync_job_last_started_at, sync_job_last_finished_at;

-- name: UpdateSyncJobEnqueueMetadata :exec
UPDATE public.sync_jobs
SET sync_job_last_pgmq_message_id = sqlc.arg(sync_job_last_pgmq_message_id),
    sync_job_last_enqueued_at = now(),
    sync_job_updated_at = now()
WHERE sync_job_id = sqlc.arg(sync_job_id);

-- name: GetSyncJobByID :one
SELECT sync_job_id, sync_job_provider, sync_job_kind, sync_job_entity_id, sync_job_dedup_key, sync_job_status, sync_job_priority, sync_job_attempt_count, sync_job_max_attempts, sync_job_run_after, sync_job_capacity_class, sync_job_payload, sync_job_checkpoint, sync_job_result, sync_job_last_error, sync_job_last_error_code, sync_job_last_http_status, sync_job_last_pgmq_message_id, sync_job_claim_token, sync_job_created_at, sync_job_updated_at, sync_job_last_enqueued_at, sync_job_last_started_at, sync_job_last_finished_at
FROM public.sync_jobs
WHERE sync_job_id = sqlc.arg(sync_job_id);

-- name: GetSyncJobByDedupKey :one
SELECT sync_job_id, sync_job_provider, sync_job_kind, sync_job_entity_id, sync_job_dedup_key, sync_job_status, sync_job_priority, sync_job_attempt_count, sync_job_max_attempts, sync_job_run_after, sync_job_capacity_class, sync_job_payload, sync_job_checkpoint, sync_job_result, sync_job_last_error, sync_job_last_error_code, sync_job_last_http_status, sync_job_last_pgmq_message_id, sync_job_claim_token, sync_job_created_at, sync_job_updated_at, sync_job_last_enqueued_at, sync_job_last_started_at, sync_job_last_finished_at
FROM public.sync_jobs
WHERE sync_job_dedup_key = sqlc.arg(sync_job_dedup_key);

-- name: ListSyncJobs :many
SELECT sync_job_id, sync_job_provider, sync_job_kind, sync_job_entity_id, sync_job_dedup_key, sync_job_status, sync_job_priority, sync_job_attempt_count, sync_job_max_attempts, sync_job_run_after, sync_job_capacity_class, sync_job_payload, sync_job_checkpoint, sync_job_result, sync_job_last_error, sync_job_last_error_code, sync_job_last_http_status, sync_job_last_pgmq_message_id, sync_job_claim_token, sync_job_created_at, sync_job_updated_at, sync_job_last_enqueued_at, sync_job_last_started_at, sync_job_last_finished_at
FROM public.sync_jobs
WHERE (sqlc.narg(sync_job_status)::text IS NULL OR sync_job_status = sqlc.narg(sync_job_status))
  AND (sqlc.narg(sync_job_provider)::text IS NULL OR sync_job_provider = sqlc.narg(sync_job_provider))
  AND (sqlc.narg(sync_job_kind)::text IS NULL OR sync_job_kind = sqlc.narg(sync_job_kind))
ORDER BY sync_job_updated_at DESC, sync_job_created_at DESC
LIMIT sqlc.arg(limit_count);

-- name: ListDuePendingSyncJobsForReconcile :many
SELECT sync_job_id, sync_job_provider, sync_job_kind, sync_job_entity_id, sync_job_dedup_key, sync_job_status, sync_job_priority, sync_job_attempt_count, sync_job_max_attempts, sync_job_run_after, sync_job_capacity_class, sync_job_payload, sync_job_checkpoint, sync_job_result, sync_job_last_error, sync_job_last_error_code, sync_job_last_http_status, sync_job_last_pgmq_message_id, sync_job_claim_token, sync_job_created_at, sync_job_updated_at, sync_job_last_enqueued_at, sync_job_last_started_at, sync_job_last_finished_at
FROM public.sync_jobs
WHERE sync_job_status = 'pending'
  AND sync_job_run_after <= sqlc.arg(now_at)
ORDER BY sync_job_priority DESC, sync_job_run_after ASC, sync_job_created_at ASC
LIMIT sqlc.arg(limit_count)
FOR UPDATE SKIP LOCKED;

-- name: ClaimSyncJob :one
UPDATE public.sync_jobs
SET sync_job_status = 'in_progress',
    sync_job_attempt_count = sync_job_attempt_count + 1,
    sync_job_last_started_at = now(),
    sync_job_last_error = NULL,
    sync_job_last_error_code = NULL,
    sync_job_last_http_status = NULL,
    sync_job_claim_token = sqlc.arg(sync_job_claim_token),
    sync_job_updated_at = now()
WHERE sync_job_id = sqlc.arg(sync_job_id)
  AND sync_job_status = 'pending'
  AND sync_job_run_after <= now()
RETURNING sync_job_id, sync_job_provider, sync_job_kind, sync_job_entity_id, sync_job_dedup_key, sync_job_status, sync_job_priority, sync_job_attempt_count, sync_job_max_attempts, sync_job_run_after, sync_job_capacity_class, sync_job_payload, sync_job_checkpoint, sync_job_result, sync_job_last_error, sync_job_last_error_code, sync_job_last_http_status, sync_job_last_pgmq_message_id, sync_job_claim_token, sync_job_created_at, sync_job_updated_at, sync_job_last_enqueued_at, sync_job_last_started_at, sync_job_last_finished_at;

-- name: CountSyncJobsInProgress :one
SELECT count(*)::bigint
FROM public.sync_jobs
WHERE sync_job_status = 'in_progress';

-- name: CountSyncJobsInProgressByCapacityClass :one
SELECT count(*)::bigint
FROM public.sync_jobs
WHERE sync_job_status = 'in_progress'
  AND sync_job_capacity_class = sqlc.arg(sync_job_capacity_class);

-- name: CountSyncJobsInProgressByKind :one
SELECT count(*)::bigint
FROM public.sync_jobs
WHERE sync_job_status = 'in_progress'
  AND sync_job_kind = sqlc.arg(sync_job_kind);

-- name: AcquireSyncDispatchAdmissionLock :exec
SELECT pg_advisory_xact_lock(sqlc.arg(class_id), sqlc.arg(object_id));

-- name: DeferPendingSyncJobForCapacity :execrows
UPDATE public.sync_jobs
SET sync_job_run_after = sqlc.arg(sync_job_run_after),
    sync_job_last_error_code = sqlc.arg(sync_job_last_error_code),
    sync_job_last_error = sqlc.arg(sync_job_last_error),
    sync_job_updated_at = now()
WHERE sync_job_id = sqlc.arg(sync_job_id)
  AND sync_job_status = 'pending';

-- name: UpdateSyncJobCheckpoint :execrows
UPDATE public.sync_jobs
SET sync_job_checkpoint = sqlc.arg(sync_job_checkpoint),
    sync_job_updated_at = now()
WHERE sync_job_id = sqlc.arg(sync_job_id)
  AND sync_job_status = 'in_progress'
  AND sync_job_claim_token = sqlc.arg(expected_sync_job_claim_token);

-- name: MarkSyncJobPendingRetry :execrows
UPDATE public.sync_jobs
SET sync_job_status = 'pending',
    sync_job_run_after = sqlc.arg(sync_job_run_after),
    sync_job_checkpoint = COALESCE(sqlc.narg(sync_job_checkpoint), sync_job_checkpoint),
    sync_job_last_error_code = sqlc.narg(sync_job_last_error_code),
    sync_job_last_error = sqlc.narg(sync_job_last_error),
    sync_job_last_http_status = sqlc.narg(sync_job_last_http_status),
    sync_job_claim_token = NULL,
    sync_job_last_finished_at = now(),
    sync_job_updated_at = now()
WHERE sync_job_id = sqlc.arg(sync_job_id)
  AND sync_job_status = 'in_progress'
  AND sync_job_claim_token = sqlc.arg(expected_sync_job_claim_token);

-- name: MarkSyncJobFinal :execrows
UPDATE public.sync_jobs
SET sync_job_status = sqlc.arg(sync_job_status),
    sync_job_result = sqlc.narg(sync_job_result),
    sync_job_checkpoint = COALESCE(sqlc.narg(sync_job_checkpoint), sync_job_checkpoint),
    sync_job_last_error_code = sqlc.narg(sync_job_last_error_code),
    sync_job_last_error = sqlc.narg(sync_job_last_error),
    sync_job_last_http_status = sqlc.narg(sync_job_last_http_status),
    sync_job_claim_token = NULL,
    sync_job_last_finished_at = now(),
    sync_job_updated_at = now()
WHERE sync_job_id = sqlc.arg(sync_job_id)
  AND sync_job_status = 'in_progress'
  AND sync_job_claim_token = sqlc.arg(expected_sync_job_claim_token);

-- name: ReapStaleSyncJobs :many
WITH stale_jobs AS (
    SELECT sj.sync_job_id
    FROM public.sync_jobs sj
    WHERE sj.sync_job_status = 'in_progress'
      AND sj.sync_job_last_started_at <= sqlc.arg(stale_before)
    ORDER BY sj.sync_job_last_started_at ASC, sj.sync_job_id ASC
    LIMIT sqlc.arg(limit_count)
    FOR UPDATE SKIP LOCKED
)
UPDATE public.sync_jobs j
SET sync_job_status = 'pending',
    sync_job_run_after = now(),
    sync_job_last_error_code = 'stale_claim_recovered',
    sync_job_last_error = 'stale in_progress sync job claim was reset',
    sync_job_last_http_status = NULL,
    sync_job_claim_token = NULL,
    sync_job_last_finished_at = now(),
    sync_job_updated_at = now()
FROM stale_jobs s
WHERE j.sync_job_id = s.sync_job_id
RETURNING j.sync_job_id, j.sync_job_provider, j.sync_job_kind, j.sync_job_entity_id, j.sync_job_dedup_key, j.sync_job_status, j.sync_job_priority, j.sync_job_attempt_count, j.sync_job_max_attempts, j.sync_job_run_after, j.sync_job_capacity_class, j.sync_job_payload, j.sync_job_checkpoint, j.sync_job_result, j.sync_job_last_error, j.sync_job_last_error_code, j.sync_job_last_http_status, j.sync_job_last_pgmq_message_id, j.sync_job_claim_token, j.sync_job_created_at, j.sync_job_updated_at, j.sync_job_last_enqueued_at, j.sync_job_last_started_at, j.sync_job_last_finished_at;
