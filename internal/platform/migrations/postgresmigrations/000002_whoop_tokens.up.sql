CREATE TABLE IF NOT EXISTS whoop_tokens (
                                            id                      BIGSERIAL PRIMARY KEY,
                                            user_id                 BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    access_token_encrypted  TEXT NOT NULL,
    refresh_token_encrypted TEXT NOT NULL,
    token_type              TEXT NOT NULL DEFAULT 'Bearer',
    scopes                  TEXT[] NOT NULL DEFAULT '{}',
    expires_at              TIMESTAMPTZ NOT NULL,

    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT whoop_tokens_user_id_uq UNIQUE (user_id)
    );

CREATE INDEX IF NOT EXISTS idx_whoop_tokens_expires_at
    ON whoop_tokens (expires_at);