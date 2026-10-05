-- +goose Up
CREATE TABLE IF NOT EXISTS subscriptions (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    service_name VARCHAR(255) NOT NULL,
    price        INTEGER NOT NULL CHECK (price >= 0),
    user_id      UUID NOT NULL,
    start_date   VARCHAR(7) NOT NULL,  -- "MM-YYYY"
    end_date     VARCHAR(7),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_subscriptions_user_id ON subscriptions(user_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_start_date ON subscriptions(start_date);

-- +goose Down
DROP TABLE IF EXISTS subscriptions;
