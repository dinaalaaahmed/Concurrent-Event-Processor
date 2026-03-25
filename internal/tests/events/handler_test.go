package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"my-app/internal/events"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListAggregatedEvents_Handler_Success(t *testing.T) {
	svc := &mockService{
		listAggregatedEventsFn: func(ctx context.Context, userID string) (map[string]int64, error) {
			return map[string]int64{"click": 5, "view": 3}, nil
		},
	}

	worker := events.NewCreateEventWorker(svc, 1)
	h := events.NewHandler(svc, worker)

	req := httptest.NewRequest(http.MethodGet, "/events?user_id=user-1", nil)
	w := httptest.NewRecorder()

	h.ListAggregatedEvents(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var result map[string]int64
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result["click"] != 5 {
		t.Errorf("expected click=5, got %d", result["click"])
	}
	if result["view"] != 3 {
		t.Errorf("expected view=3, got %d", result["view"])
	}
}

func TestListAggregatedEvents_Handler_ServiceError(t *testing.T) {
	svc := &mockService{
		listAggregatedEventsFn: func(ctx context.Context, userID string) (map[string]int64, error) {
			return nil, errors.New("db error")
		},
	}

	worker := events.NewCreateEventWorker(svc, 1)
	h := events.NewHandler(svc, worker)

	req := httptest.NewRequest(http.MethodGet, "/events?user_id=user-1", nil)
	w := httptest.NewRecorder()

	h.ListAggregatedEvents(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestCreateEvent_Handler_Success(t *testing.T) {
	svc := &mockService{}
	worker := events.NewCreateEventWorker(svc, 1)
	h := events.NewHandler(svc, worker)

	body := bytes.NewBufferString(`{
		"user_id": "user-1",
		"event_type": "click",
		"value": 1,
		"timestamp": "2024-01-01T00:00:00Z"
	}`)

	req := httptest.NewRequest(http.MethodPost, "/events", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.CreateEvent(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("expected 202, got %d", w.Code)
	}

	var result map[string]string
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result["status"] != "queued" {
		t.Errorf("expected status=queued, got %s", result["status"])
	}
	if result["id"] == "" {
		t.Error("expected id to be set")
	}
}

func TestCreateEvent_Handler_InvalidJSON(t *testing.T) {
	svc := &mockService{}
	worker := events.NewCreateEventWorker(svc, 1)
	h := events.NewHandler(svc, worker)

	req := httptest.NewRequest(http.MethodPost, "/events", bytes.NewBufferString(`invalid json`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.CreateEvent(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreateEvent_Handler_MissingFields(t *testing.T) {
	svc := &mockService{}
	worker := events.NewCreateEventWorker(svc, 1)
	h := events.NewHandler(svc, worker)

	body := bytes.NewBufferString(`{
		"user_id": "user-1"
	}`)

	req := httptest.NewRequest(http.MethodPost, "/events", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.CreateEvent(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreateEvent_Handler_InvalidTimestamp(t *testing.T) {
	svc := &mockService{}
	worker := events.NewCreateEventWorker(svc, 1)
	h := events.NewHandler(svc, worker)

	body := bytes.NewBufferString(`{
		"user_id": "user-1",
		"event_type": "click",
		"value": 1,
		"timestamp": "not-a-timestamp"
	}`)

	req := httptest.NewRequest(http.MethodPost, "/events", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.CreateEvent(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreateEvent_Handler_WorkerBusy(t *testing.T) {
	svc := &mockService{}
	worker := events.NewCreateEventWorker(svc, 0) // zero buffer to force busy state
	h := events.NewHandler(svc, worker)

	body := bytes.NewBufferString(`{
		"user_id": "user-1",
		"event_type": "click",
		"value": 1,
		"timestamp": "2024-01-01T00:00:00Z"
	}`)

	req := httptest.NewRequest(http.MethodPost, "/events", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.CreateEvent(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}
