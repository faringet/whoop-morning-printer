CREATE TABLE IF NOT EXISTS raw_whoop_objects (
                                                 id           BIGSERIAL PRIMARY KEY,
                                                 user_id      BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    object_type  TEXT NOT NULL,
    whoop_id     TEXT NOT NULL,

    start_at     TIMESTAMPTZ NULL,
    end_at       TIMESTAMPTZ NULL,
    score_state  TEXT NULL,

    payload_json JSONB NOT NULL,

    fetched_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT raw_whoop_objects_user_type_whoop_id_uq
    UNIQUE (user_id, object_type, whoop_id)
    );

CREATE INDEX IF NOT EXISTS idx_raw_whoop_objects_user_type_start_at
    ON raw_whoop_objects (user_id, object_type, start_at DESC);

CREATE INDEX IF NOT EXISTS idx_raw_whoop_objects_score_state
    ON raw_whoop_objects (score_state);

CREATE INDEX IF NOT EXISTS idx_raw_whoop_objects_payload_json_gin
    ON raw_whoop_objects
    USING gin (payload_json);