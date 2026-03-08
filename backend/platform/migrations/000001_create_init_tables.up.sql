CREATE TABLE analytics_sessions (
    id BIGSERIAL PRIMARY KEY,
    user_id TEXT NOT NULL,
    category_id TEXT NOT NULL,
    duration_seconds BIGINT NOT NULL CHECK (duration_seconds > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_analytics_sessions_user_id_created_at
    ON analytics_sessions (user_id, id DESC);

CREATE INDEX idx_analytics_sessions_user_id_category_id
    ON analytics_sessions (user_id, category_id);
