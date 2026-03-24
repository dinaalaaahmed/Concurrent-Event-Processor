-- name: ListAgrregatedEvents :many
SELECT
  SUM(value)::BIGINT as event_count,
  event_type
FROM
  events
WHERE user_id = $1
GROUP BY event_type;


-- name: CreateEvent :one
INSERT INTO events (user_id, event_type, value, timestamp)
VALUES ($1, $2, $3, $4) RETURNING *;