-- +goose Up
CREATE INDEX idx_on_user_id_and_event_type ON events (user_id, event_type);

-- +goose Down
DROP INDEX idx_on_user_id_and_event_type;;
