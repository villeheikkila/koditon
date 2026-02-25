-- name: CallRegisterEntity :exec
SELECT task_queue.fnc__register_entity($1::text, $2::text, $3::text, $4::text, $5::jsonb);

-- name: CallRegisterEntities :one
SELECT task_queue.fnc__register_entities($1::text[], $2::text, $3::text) AS count;

-- name: CallEnqueueTask :one
SELECT task_queue.fnc__enqueue_task($1::bigint) AS message_id;

-- name: CallScheduleDailySyncs :one
SELECT task_queue.fnc__schedule_daily_syncs($1::text) AS count;

-- name: CallRequeueStuckTasks :one
SELECT task_queue.fnc__requeue_stuck_tasks() AS count;
