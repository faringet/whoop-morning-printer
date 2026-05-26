DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'wake_plans_wake_receipt_job_id_fk'
    ) THEN
ALTER TABLE wake_plans
    ADD CONSTRAINT wake_plans_wake_receipt_job_id_fk
        FOREIGN KEY (wake_receipt_job_id)
            REFERENCES print_jobs(id)
            ON DELETE SET NULL;
END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'wake_plans_final_report_job_id_fk'
    ) THEN
ALTER TABLE wake_plans
    ADD CONSTRAINT wake_plans_final_report_job_id_fk
        FOREIGN KEY (final_report_job_id)
            REFERENCES print_jobs(id)
            ON DELETE SET NULL;
END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'wake_plans_fallback_job_id_fk'
    ) THEN
ALTER TABLE wake_plans
    ADD CONSTRAINT wake_plans_fallback_job_id_fk
        FOREIGN KEY (fallback_job_id)
            REFERENCES print_jobs(id)
            ON DELETE SET NULL;
END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'print_attempts_device_id_fk'
    ) THEN
ALTER TABLE print_attempts
    ADD CONSTRAINT print_attempts_device_id_fk
        FOREIGN KEY (device_id)
            REFERENCES devices(id)
            ON DELETE SET NULL;
END IF;
END $$;