CREATE TABLE IF NOT EXISTS print_jobs (
                                          id                BIGSERIAL PRIMARY KEY,
                                          user_id           BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    wake_plan_id      BIGINT NULL REFERENCES wake_plans(id) ON DELETE CASCADE,

    type              TEXT NOT NULL,
    status            TEXT NOT NULL DEFAULT 'pending',

    not_before        TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    payload_type      TEXT NOT NULL DEFAULT 'text/plain',
    payload_text      TEXT NULL,

    claimed_by        TEXT NULL,
    processing_until  TIMESTAMPTZ NULL,

    printed_at        TIMESTAMPTZ NULL,
    failed_at         TIMESTAMPTZ NULL,
    error_message     TEXT NULL,

    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT print_jobs_type_chk
    CHECK (type IN (
           'wake_receipt',
           'final_report',
           'fallback_report',
           'test'
                   )),

    CONSTRAINT print_jobs_status_chk
    CHECK (status IN (
           'pending',
           'processing',
           'ready',
           'printed',
           'failed',
           'cancelled'
                     )),

    CONSTRAINT print_jobs_payload_type_chk
    CHECK (payload_type IN (
           'text/plain',
           'escpos',
           'star',
           'cups'
                           ))
    );

CREATE INDEX IF NOT EXISTS idx_print_jobs_pending_lookup
    ON print_jobs (not_before, created_at)
    WHERE status IN ('pending', 'ready');

CREATE INDEX IF NOT EXISTS idx_print_jobs_processing_until
    ON print_jobs (processing_until);

CREATE INDEX IF NOT EXISTS idx_print_jobs_user_created_at_desc
    ON print_jobs (user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_print_jobs_wake_plan_id
    ON print_jobs (wake_plan_id);