CREATE TABLE IF NOT EXISTS daily_health_snapshots (
                                                      id                         BIGSERIAL PRIMARY KEY,
                                                      user_id                    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    date                       DATE NOT NULL,
    data_state                 TEXT NOT NULL DEFAULT 'pending',

    sleep_whoop_id             TEXT NULL,
    recovery_whoop_id          TEXT NULL,
    cycle_whoop_id             TEXT NULL,

    sleep_score                INTEGER NULL,
    recovery_score             INTEGER NULL,
    day_strain                 NUMERIC(5, 2) NULL,

    sleep_minutes              INTEGER NULL,
    sleep_needed_minutes       INTEGER NULL,
    sleep_vs_need_pct          INTEGER NULL,

    awake_minutes              INTEGER NULL,
    light_sleep_minutes        INTEGER NULL,
    deep_sleep_minutes         INTEGER NULL,
    rem_sleep_minutes          INTEGER NULL,
    restorative_sleep_minutes  INTEGER NULL,

    sleep_efficiency_pct       NUMERIC(5, 2) NULL,
    sleep_consistency_pct      NUMERIC(5, 2) NULL,
    respiratory_rate           NUMERIC(5, 2) NULL,

    hrv_rmssd_ms               NUMERIC(8, 2) NULL,
    resting_heart_rate_bpm     INTEGER NULL,
    spo2_pct                   NUMERIC(5, 2) NULL,
    skin_temp_celsius          NUMERIC(5, 2) NULL,

    source_updated_at          TIMESTAMPTZ NULL,
    created_at                 TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                 TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT daily_health_snapshots_user_date_uq
    UNIQUE (user_id, date),

    CONSTRAINT daily_health_snapshots_data_state_chk
    CHECK (data_state IN ('pending', 'partial', 'ready', 'failed')),

    CONSTRAINT daily_health_snapshots_sleep_score_chk
    CHECK (sleep_score IS NULL OR (sleep_score >= 0 AND sleep_score <= 100)),

    CONSTRAINT daily_health_snapshots_recovery_score_chk
    CHECK (recovery_score IS NULL OR (recovery_score >= 0 AND recovery_score <= 100)),

    CONSTRAINT daily_health_snapshots_sleep_vs_need_pct_chk
    CHECK (sleep_vs_need_pct IS NULL OR sleep_vs_need_pct >= 0)
    );

CREATE INDEX IF NOT EXISTS idx_daily_health_snapshots_user_date_desc
    ON daily_health_snapshots (user_id, date DESC);

CREATE INDEX IF NOT EXISTS idx_daily_health_snapshots_data_state
    ON daily_health_snapshots (data_state);