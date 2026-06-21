CREATE TYPE flowdo_status AS ENUM ('pending', 'in_progress', 'done');

CREATE TABLE IF NOT EXISTS flowdos (
    id          UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title       VARCHAR(255) NOT NULL,
    description TEXT         NOT NULL DEFAULT '',
    status      flowdo_status  NOT NULL DEFAULT 'pending',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_flowdos_user_id    ON flowdos(user_id);
CREATE INDEX IF NOT EXISTS idx_flowdos_status     ON flowdos(status);
CREATE INDEX IF NOT EXISTS idx_flowdos_deleted_at ON flowdos(deleted_at) WHERE deleted_at IS NULL;
