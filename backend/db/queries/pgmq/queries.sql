-- ============================================
-- PGMQ METRICS QUERIES
-- ============================================

-- name: GetQueueMetrics :one
SELECT
    queue_name::text AS queue_name,
    queue_length::bigint AS queue_length,
    newest_msg_age_sec::int AS newest_msg_age_sec,
    oldest_msg_age_sec::int AS oldest_msg_age_sec,
    total_messages::bigint AS total_messages,
    scrape_time::timestamptz AS scrape_time
FROM pgmq.metrics(sqlc.arg(queue_name));

-- name: GetAllQueueMetrics :many
SELECT
    queue_name::text AS queue_name,
    queue_length::bigint AS queue_length,
    newest_msg_age_sec::int AS newest_msg_age_sec,
    oldest_msg_age_sec::int AS oldest_msg_age_sec,
    total_messages::bigint AS total_messages,
    scrape_time::timestamptz AS scrape_time
FROM pgmq.metrics_all();

-- ============================================
-- PGMQ QUEUE MANAGEMENT QUERIES
-- ============================================

-- name: ListQueues :many
SELECT
    queue_name::text AS queue_name,
    is_partitioned::bool AS is_partitioned,
    is_unlogged::bool AS is_unlogged,
    created_at::timestamptz AS created_at
FROM pgmq.list_queues();

-- name: GetQueueInfo :one
SELECT
    queue_name::text AS queue_name,
    is_partitioned::bool AS is_partitioned,
    is_unlogged::bool AS is_unlogged,
    created_at::timestamptz AS created_at
FROM pgmq.list_queues()
WHERE queue_name = sqlc.arg(queue_name);

-- name: CreateQueue :exec
SELECT pgmq.create(sqlc.arg(queue_name));

-- name: CreateUnloggedQueue :exec
SELECT pgmq.create_unlogged(sqlc.arg(queue_name));

-- name: CreatePartitionedQueue :exec
SELECT pgmq.create_partitioned(
    sqlc.arg(queue_name),
    sqlc.arg(partition_interval),
    sqlc.arg(retention_interval)
);

-- name: DropQueue :one
SELECT pgmq.drop_queue(sqlc.arg(queue_name))::bool AS dropped;

-- name: PurgeQueue :one
SELECT pgmq.purge_queue(sqlc.arg(queue_name))::bigint AS count;

-- ============================================
-- PGMQ MESSAGE OPERATIONS
-- ============================================

-- name: Send :one
SELECT pgmq.send(
    sqlc.arg(queue_name),
    sqlc.arg(message),
    sqlc.arg(delay_seconds)::int
)::bigint AS msg_id;

-- name: SendBatch :many
SELECT msg_id::bigint AS msg_id
FROM pgmq.send_batch(
    sqlc.arg(queue_name),
    sqlc.arg(messages)::jsonb[],
    sqlc.arg(delay_seconds)::int
) AS msg_id;

-- name: Read :many
SELECT
    msg_id::bigint AS msg_id,
    read_ct::int AS read_ct,
    enqueued_at::timestamptz AS enqueued_at,
    vt::timestamptz AS vt,
    message::jsonb AS message,
    headers::jsonb AS headers
FROM pgmq.read(
    sqlc.arg(queue_name),
    sqlc.arg(vt_seconds),
    sqlc.arg(num_messages)
);

-- name: Pop :many
SELECT
    msg_id::bigint AS msg_id,
    read_ct::int AS read_ct,
    enqueued_at::timestamptz AS enqueued_at,
    vt::timestamptz AS vt,
    message::jsonb AS message,
    headers::jsonb AS headers
FROM pgmq.pop(sqlc.arg(queue_name));

-- name: Archive :one
SELECT pgmq.archive(
    sqlc.arg(queue_name),
    sqlc.arg(msg_id)::bigint
)::bool AS archived;

-- name: ArchiveBatch :many
SELECT msg_id::bigint AS msg_id
FROM pgmq.archive(
    sqlc.arg(queue_name),
    sqlc.arg(msg_ids)::bigint[]
) AS msg_id;

-- name: Delete :one
SELECT pgmq.delete(
    sqlc.arg(queue_name),
    sqlc.arg(msg_id)::bigint
)::bool AS deleted;

-- name: DeleteBatch :many
SELECT msg_id::bigint AS msg_id
FROM pgmq.delete(
    sqlc.arg(queue_name),
    sqlc.arg(msg_ids)::bigint[]
) AS msg_id;

-- name: SetVT :many
SELECT
    msg_id::bigint AS msg_id,
    read_ct::int AS read_ct,
    enqueued_at::timestamptz AS enqueued_at,
    vt::timestamptz AS vt,
    message::jsonb AS message,
    headers::jsonb AS headers
FROM pgmq.set_vt(
    sqlc.arg(queue_name),
    sqlc.arg(msg_id)::bigint,
    sqlc.arg(vt_seconds)::int
);
