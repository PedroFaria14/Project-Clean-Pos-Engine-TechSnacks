CREATE TABLE users (
    id text PRIMARY KEY,
    username varchar(64) COLLATE "C" NOT NULL UNIQUE,
    password_hash text NOT NULL,
    role text NOT NULL CHECK (role IN ('operator', 'admin')),
    active boolean NOT NULL DEFAULT true
);
CREATE TABLE user_sessions (
    token_hash char(64) PRIMARY KEY,
    user_id text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at timestamptz NOT NULL
);
CREATE INDEX user_sessions_expiry_idx ON user_sessions(expires_at);
CREATE TABLE auth_login_limits (
    bucket_key char(64) PRIMARY KEY,
    attempts integer NOT NULL,
    window_start timestamptz NOT NULL
);
CREATE INDEX auth_login_limits_window_idx ON auth_login_limits(window_start);
ALTER TABLE order_models ADD COLUMN created_by text REFERENCES users(id);
CREATE INDEX order_models_creator_idx ON order_models(created_by);
-- Internal namespacing "user:<user ID>:<raw key>"; old keys stay unchanged.
ALTER TABLE checkout_idempotency_records ALTER COLUMN key TYPE varchar(256) COLLATE "C";
