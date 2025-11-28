ALTER TABLE prices_transactions
DROP COLUMN prices_transactions_city;

ALTER TABLE prices_transactions
DROP COLUMN prices_transactions_neighborhood;

ALTER TABLE prices_transactions
DROP COLUMN prices_neighborhoods_postal_code

ALTER TABLE prices_transactions
RENAME COLUMN prices_transactions_first_seen_at TO created_at;

ALTER TABLE prices_transactions
RENAME COLUMN prices_transactions_last_seen_at TO updated_at;
