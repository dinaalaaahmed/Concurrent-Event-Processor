package events

import (
	"fmt"
	repo "my-app/internal/adapters/postgresql/sqlc"
	"my-app/internal/json"
	"my-app/internal/requests"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type handler struct {
	service           Service
	createEventWorker *CreateEventWorker
}

func NewHandler(service Service, createEventWorker CreateEventWorker) *handler {
	return &handler{
		service:           service,
		createEventWorker: &createEventWorker,
	}
}

func (h *handler) ListAggregatedEvents(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")

	// Get the map from the service
	aggregatedData, err := h.service.ListAgrregatedEvents(r.Context(), userID)
	if err != nil {
		http.Error(w, "Failed to aggregate", http.StatusInternalServerError)
		return
	}

	json.Write(w, http.StatusOK, aggregatedData)
}

func (h *handler) CreateEvent(w http.ResponseWriter, r *http.Request) {
	// 2. Decode into the struct that HAS the rules
	var req requests.CreateEventRequest
	if err := json.Read(r, &req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// 3. Check the rules
	validate := validator.New()
	if err := validate.Struct(req); err != nil {
		// This will now catch "required" and return an error!
		http.Error(w, fmt.Sprintf("Validation Error: %s", err.Error()), http.StatusBadRequest)
		return
	}

	parsedTime, err := time.Parse(time.RFC3339, req.Timestamp)
	if err != nil {
		http.Error(w, "Invalid timestamp format. Use RFC3339 (ISO8601)", http.StatusBadRequest)
		return
	}

	eventParams := repo.CreateEventParams{
		UserID:    req.UserID,
		EventType: req.EventType,
		Value:     int64(req.Value),
		Timestamp: pgtype.Timestamptz{
			Time:  parsedTime,
			Valid: true,
		},
	}

	job := Job{
		ID:        uuid.NewString(),
		Payload:   eventParams,
		CreatedAt: time.Now(),
	}

	// Send to worker pool
	if ok := h.createEventWorker.Enqueue(job); !ok {
		// This happens if the buffer (e.g. 100) is completely full
		http.Error(w, "Server busy, try again later", http.StatusServiceUnavailable)
		return
	}

	// Success! The user doesn't wait for the DB.
	json.Write(w, http.StatusAccepted, map[string]string{"id": job.ID, "status": "queued"})
}
