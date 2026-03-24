-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS events (
  id BIGSERIAL PRIMARY KEY,
  user_id TEXT NOT NULL,
  event_type VARCHAR(50) NOT NULL,
  value BIGINT NOT NULL,
  timestamp TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS events;
-- +goose StatementEnd
