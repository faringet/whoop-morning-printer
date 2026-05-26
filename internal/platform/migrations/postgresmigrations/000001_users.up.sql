CREATE TABLE IF NOT EXISTS users (
                                     id               BIGSERIAL PRIMARY KEY,
                                     telegram_user_id BIGINT NULL,
                                     timezone         TEXT NOT NULL DEFAULT 'Europe/Moscow',
                                     created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT users_telegram_user_id_uq UNIQUE (telegram_user_id)
    );