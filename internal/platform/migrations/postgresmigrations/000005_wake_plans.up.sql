CREATE TABLE IF NOT EXISTS wake_plans (
                                          id                  BIGSERIAL PRIMARY KEY,
                                          user_id             BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    date                DATE NOT NULL,

    wake_at             TIMESTAMPTZ NOT NULL,
    prepare_at          TIMESTAMPTZ NOT NULL,
    final_deadline_at   TIMESTAMPTZ NOT NULL,

    status              TEXT NOT NULL DEFAULT 'scheduled',
    source              TEXT NOT NULL DEFAULT 'manual',

    wake_receipt_job_id BIGINT NULL,
    final_report_job_id BIGINT NULL,
    fallback_job_id     BIGINT NULL,

    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT wake_plans_user_date_uq
    UNIQUE (user_id, date),

    CONSTRAINT wake_plans_status_chk
    CHECK (status IN (
           'scheduled',
           'wake_receipt_ready',
           'wake_receipt_printed',
           'waiting_whoop',
           'waiting_advice',
           'final_report_ready',
           'final_report_printed',
           'fallback_printed',
           'done',
           'cancelled',
           'failed'
                     )),

    CONSTRAINT wake_plans_source_chk
    CHECK (source IN (
           'manual',
           'telegram',
           'default',
           'test'
                     )),

    CONSTRAINT wake_plans_time_order_chk
    CHECK (
              prepare_at <= wake_at
              AND wake_at <= final_deadline_at
          )
    );

CREATE INDEX IF NOT EXISTS idx_wake_plans_user_date_desc
    ON wake_plans (user_id, date DESC);

CREATE INDEX IF NOT EXISTS idx_wake_plans_status
    ON wake_plans (status);

CREATE INDEX IF NOT EXISTS idx_wake_plans_wake_at
    ON wake_plans (wake_at);