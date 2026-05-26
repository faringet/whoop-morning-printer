CREATE TABLE IF NOT EXISTS morning_advice (
                                              id              BIGSERIAL PRIMARY KEY,
                                              user_id         BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    snapshot_id     BIGINT NOT NULL REFERENCES daily_health_snapshots(id) ON DELETE CASCADE,

    date            DATE NOT NULL,

    status          TEXT NOT NULL DEFAULT 'pending',

    model           TEXT NULL,
    prompt_version  TEXT NOT NULL DEFAULT 'rules_v1',

    main_signal     TEXT NULL,
    day_type        TEXT NULL,
    advice_json     JSONB NULL,
    rendered_text   TEXT NULL,
    motto           TEXT NULL,

    error_message   TEXT NULL,

    processing_by    TEXT NULL,
    processing_until TIMESTAMPTZ NULL,

    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT morning_advice_user_date_prompt_uq
    UNIQUE (user_id, date, prompt_version),

    CONSTRAINT morning_advice_status_chk
    CHECK (status IN (
           'pending',
           'processing',
           'ready',
           'failed'
                     ))
    );

CREATE INDEX IF NOT EXISTS idx_morning_advice_user_date_desc
    ON morning_advice (user_id, date DESC);

CREATE INDEX IF NOT EXISTS idx_morning_advice_status
    ON morning_advice (status);

CREATE INDEX IF NOT EXISTS idx_morning_advice_processing_until
    ON morning_advice (processing_until);

CREATE INDEX IF NOT EXISTS idx_morning_advice_ready_lookup
    ON morning_advice (user_id, date)
    WHERE status = 'ready';