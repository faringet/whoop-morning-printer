CREATE TABLE IF NOT EXISTS print_attempts (
                                              id             BIGSERIAL PRIMARY KEY,
                                              print_job_id   BIGINT NOT NULL REFERENCES print_jobs(id) ON DELETE CASCADE,

    device_id      BIGINT NULL,
    agent_id       TEXT NULL,

    status         TEXT NOT NULL,
    output_target  TEXT NULL,
    error_message  TEXT NULL,

    started_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at    TIMESTAMPTZ NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT print_attempts_status_chk
    CHECK (status IN (
           'started',
           'printed',
           'failed'
                     ))
    );

CREATE INDEX IF NOT EXISTS idx_print_attempts_print_job_id
    ON print_attempts (print_job_id);

CREATE INDEX IF NOT EXISTS idx_print_attempts_created_at_desc
    ON print_attempts (created_at DESC);