INSERT INTO task_queue.entity_registry (entity_id, entity_type, status, scheduling_strategy)
VALUES ('prices:neighborhood_postal_codes', 'prices_neighborhood_postal_codes', 'active', 'cron')
ON CONFLICT (entity_id) DO NOTHING;

INSERT INTO task_queue.task_type_entity_type_mapping (task_type, entity_type)
VALUES ('prices_neighborhood_postal_code_sync', 'prices_neighborhood_postal_codes')
ON CONFLICT (task_type, entity_type) DO NOTHING;

CREATE OR REPLACE FUNCTION task_queue.fnc__schedule_prices_neighborhood_postal_code_sync() RETURNS BIGINT AS $$
DECLARE
    v_task_id BIGINT;
    v_msg_id BIGINT;
    v_existing_task RECORD;
BEGIN
    SELECT task_id, status INTO v_existing_task
    FROM task_queue.task
    WHERE entity_id = 'prices:neighborhood_postal_codes'
      AND task_type = 'prices_neighborhood_postal_code_sync'
      AND run_on = CURRENT_DATE
    LIMIT 1;
    IF FOUND THEN
        IF v_existing_task.status IN ('pending', 'processing') THEN
            RAISE NOTICE 'Prices neighborhood postal code sync already scheduled for today (task_id: %)', v_existing_task.task_id;
            RETURN v_existing_task.task_id;
        END IF;
    END IF;
    INSERT INTO task_queue.task (
        entity_id,
        task_type,
        status,
        attempt,
        max_attempts,
        scheduled_for,
        run_on
    )
    VALUES (
        'prices:neighborhood_postal_codes',
        'prices_neighborhood_postal_code_sync',
        'pending',
        0,
        3,
        NOW(),
        CURRENT_DATE
    )
    RETURNING task_id INTO v_task_id;
    v_msg_id := pgmq.send(
        'tasks',
        jsonb_build_object(
            'task_id', v_task_id,
            'entity_id', 'prices:neighborhood_postal_codes',
            'attempt', 0
        )
    );
    UPDATE task_queue.task
    SET queue_message_id = v_msg_id,
        updated_at = NOW()
    WHERE task_id = v_task_id;
    RAISE NOTICE 'Prices neighborhood postal code sync scheduled (task_id: %, msg_id: %)', v_task_id, v_msg_id;
    RETURN v_task_id;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION task_queue.fnc__schedule_prices_neighborhood_postal_code_sync() IS
'Creates a prices_neighborhood_postal_code_sync task that workers will process to map neighborhoods to postal codes by iterating through all postal codes and their transactions.';

SELECT cron.schedule(
    'trigger-prices-neighborhood-postal-code-sync',
    '0 5 * * 0',
    $$SELECT task_queue.fnc__schedule_prices_neighborhood_postal_code_sync()$$
)
WHERE NOT EXISTS (
    SELECT 1 FROM cron.job WHERE jobname = 'trigger-prices-neighborhood-postal-code-sync'
);
