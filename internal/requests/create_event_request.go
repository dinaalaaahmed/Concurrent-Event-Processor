package requests

type CreateEventRequest struct {
	UserID    string `json:"user_id" validate:"required"`
	EventType string `json:"event_type" validate:"required"`
	Value     int    `json:"value" validate:"required,gt=0"`
	Timestamp string `json:"timestamp" validate:"required"`
}
