UPDATE prices_transactions
SET prices_transactions_elevator = CASE
    WHEN prices_transactions_elevator = 'on' THEN 'true'
    WHEN prices_transactions_elevator = 'ei' THEN 'false'
    ELSE NULL
END;

ALTER TABLE prices_transactions
ALTER COLUMN prices_transactions_elevator
TYPE BOOLEAN
USING prices_transactions_elevator::BOOLEAN;

UPDATE prices_transactions
SET prices_transactions_condition = NULL
WHERE prices_transactions_condition = '';

UPDATE prices_transactions
SET prices_transactions_floor = NULL
WHERE prices_transactions_floor = '';

UPDATE prices_transactions
SET prices_transactions_plot = NULL
WHERE prices_transactions_plot = '';
