-- name: InsertSyncJobAttemptRunning :one
INSERT INTO public.sync_job_attempts (
    sync_job_id,
    sync_job_attempt_queue_name,
    sync_job_attempt_msg_id,
    sync_job_attempt_no,
    sync_job_attempt_status,
    sync_job_attempt_payload_snapshot
) VALUES (
    sqlc.arg(sync_job_id),
    sqlc.arg(sync_job_attempt_queue_name),
    sqlc.narg(sync_job_attempt_msg_id),
    sqlc.arg(sync_job_attempt_no),
    'running',
    sqlc.narg(sync_job_attempt_payload_snapshot)
)
RETURNING sync_job_attempt_id;

-- name: FinalizeSyncJobAttempt :exec
UPDATE public.sync_job_attempts
SET sync_job_attempt_status = sqlc.arg(sync_job_attempt_status),
    sync_job_attempt_error_code = sqlc.narg(sync_job_attempt_error_code),
    sync_job_attempt_error_detail = sqlc.narg(sync_job_attempt_error_detail),
    sync_job_attempt_finished_at = now()
WHERE sync_job_attempt_id = sqlc.arg(sync_job_attempt_id);

-- name: FinalizeRunningSyncJobAttemptsForJobs :execrows
UPDATE public.sync_job_attempts
SET sync_job_attempt_status = 'retry',
    sync_job_attempt_error_code = 'stale_claim_recovered',
    sync_job_attempt_error_detail = 'stale in_progress sync job claim was reset',
    sync_job_attempt_finished_at = now()
WHERE sync_job_id = ANY(sqlc.arg(sync_job_ids)::uuid[])
  AND sync_job_attempt_status = 'running';

-- name: ListSyncJobAttempts :many
SELECT sync_job_attempt_id,
       sync_job_id,
       sync_job_attempt_queue_name,
       sync_job_attempt_msg_id,
       sync_job_attempt_no,
       sync_job_attempt_status,
       sync_job_attempt_error_code,
       sync_job_attempt_error_detail,
       sync_job_attempt_payload_snapshot,
       sync_job_attempt_created_at,
       sync_job_attempt_finished_at
FROM public.sync_job_attempts
WHERE sync_job_id = sqlc.arg(sync_job_id)
ORDER BY sync_job_attempt_created_at DESC, sync_job_attempt_id DESC;
