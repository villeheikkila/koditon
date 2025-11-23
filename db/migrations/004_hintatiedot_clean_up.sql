UPDATE hintatiedot_transactions
SET hintatiedot_transactions_elevator = CASE
    WHEN hintatiedot_transactions_elevator = 'on' THEN 'true'
    WHEN hintatiedot_transactions_elevator = 'ei' THEN 'false'
    ELSE NULL
END;

ALTER TABLE hintatiedot_transactions
ALTER COLUMN hintatiedot_transactions_elevator
TYPE BOOLEAN
USING hintatiedot_transactions_elevator::BOOLEAN;

UPDATE hintatiedot_transactions
SET hintatiedot_transactions_condition = NULL
WHERE hintatiedot_transactions_condition = '';

UPDATE hintatiedot_transactions
SET hintatiedot_transactions_floor = NULL
WHERE hintatiedot_transactions_floor = '';

UPDATE hintatiedot_transactions
SET hintatiedot_transactions_plot = NULL
WHERE hintatiedot_transactions_plot = '';
