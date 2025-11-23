-- Create users table to match internal/auth/orm.go Users model
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS users (
    id               uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    name             text        NOT NULL,
    email            text        NOT NULL UNIQUE,
    password_hash    text        NOT NULL,
    role             text        NOT NULL DEFAULT 'employee' CHECK (role IN ('admin', 'employee')),
    manager_id       uuid        NULL,
    completed_tasks  bigint      NOT NULL DEFAULT 0,
    created_at       timestamptz NOT NULL DEFAULT now()
);

-- Useful index (UNIQUE on email already created by constraint above)
CREATE INDEX IF NOT EXISTS idx_users_created_at ON users (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_users_manager_id ON users (manager_id) WHERE manager_id IS NOT NULL;