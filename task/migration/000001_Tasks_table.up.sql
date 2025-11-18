-- Create tasks table
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS tasks (
    id               uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    message          text        NOT NULL,
    status           text        NOT NULL DEFAULT 'IN_PROGRESS' CHECK (status IN ('IN_PROGRESS', 'COMPLETED', 'NEEDS_HELP')),
    worker_id        uuid        NOT NULL,
    created_by       uuid        NOT NULL,
    reason           text,                    -- Причина для статуса NEEDS_HELP
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now()
    -- Примечание: FOREIGN KEY убраны, т.к. таблица users находится в другой БД (taskdb)
    -- Валидация существования пользователей должна выполняться на уровне приложения
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_tasks_worker_id ON tasks (worker_id);
CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks (status);
CREATE INDEX IF NOT EXISTS idx_tasks_created_by ON tasks (created_by);
CREATE INDEX IF NOT EXISTS idx_tasks_created_at ON tasks (created_at DESC);

