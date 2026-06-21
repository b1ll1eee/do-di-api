ALTER TABLE flowdos ADD COLUMN IF NOT EXISTS position INTEGER NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_flowdos_user_position ON flowdos(user_id, position) WHERE deleted_at IS NULL;
