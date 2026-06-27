CREATE TABLE events (
    id         BIGSERIAL PRIMARY KEY,
    event_name VARCHAR(255) NOT NULL,
    user_id    INT NOT NULL,
    payload    JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE metrics (
    id         BIGSERIAL PRIMARY KEY,
    metric     VARCHAR(100) NOT NULL,
    value      DOUBLE PRECISION NOT NULL,
    recorded_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);
