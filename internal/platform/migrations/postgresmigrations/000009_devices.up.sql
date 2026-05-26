CREATE TABLE IF NOT EXISTS devices (
                                       id                BIGSERIAL PRIMARY KEY,
                                       user_id           BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    name              TEXT NOT NULL,
    type              TEXT NOT NULL,

    agent_id          TEXT NULL,
    capabilities_json JSONB NOT NULL DEFAULT '{}'::jsonb,

    last_seen_at      TIMESTAMPTZ NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT devices_user_name_uq
    UNIQUE (user_id, name),

    CONSTRAINT devices_type_chk
    CHECK (type IN (
           'macmini',
           'printer',
           'bridge',
           'dev'
                   ))
    );

CREATE INDEX IF NOT EXISTS idx_devices_user_id
    ON devices (user_id);

CREATE INDEX IF NOT EXISTS idx_devices_agent_id
    ON devices (agent_id);

CREATE INDEX IF NOT EXISTS idx_devices_last_seen_at
    ON devices (last_seen_at);