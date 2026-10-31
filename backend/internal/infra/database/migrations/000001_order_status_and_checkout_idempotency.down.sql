-- Manual rollback only. Removes deduplication history and status metadata.
-- Order rows and item rows are deliberately preserved.
DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM techsnacks_schema_migrations
               WHERE version > '000001_order_status_and_checkout_idempotency.up.sql') THEN
        RAISE EXCEPTION 'Cannot roll back an order migration with dependent migrations applied';
    END IF;
END $$;
DROP TABLE checkout_idempotency_records;
ALTER TABLE order_models DROP CONSTRAINT order_models_status_check;
ALTER TABLE order_models DROP COLUMN status;
DELETE FROM techsnacks_schema_migrations
WHERE version = '000001_order_status_and_checkout_idempotency.up.sql';
