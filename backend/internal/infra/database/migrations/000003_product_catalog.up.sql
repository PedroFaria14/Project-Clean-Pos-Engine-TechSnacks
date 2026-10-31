-- Preserve existing catalog data while replacing repository-side AutoMigrate.
CREATE TABLE IF NOT EXISTS products (
    id text PRIMARY KEY,
    name text,
    price bigint
);
