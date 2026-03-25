package tests

import (
	"context"
	"errors"
	repo "my-app/internal/adapters/postgresql/sqlc"
	"my-app/internal/events"
	"sync"
	"testing"
)

// Mock implementation of repo.Querier
type mockQuerier struct {
	mu                     sync.Mutex
	createEventFn          func(ctx context.Context, params repo.CreateEventParams) (repo.Event, error)
	listAggregatedEventsFn func(ctx context.Context, userID string) ([]repo.ListAgrregatedEventsRow, error)
}

func (m *mockQuerier) CreateEvent(ctx context.Context, params repo.CreateEventParams) (repo.Event, error) {
	if m.createEventFn != nil {
		return m.createEventFn(ctx, params)
	}
	return repo.Event{}, nil
}

func (m *mockQuerier) ListAgrregatedEvents(ctx context.Context, userID string) ([]repo.ListAgrregatedEventsRow, error) {
	if m.listAggregatedEventsFn != nil {
		return m.listAggregatedEventsFn(ctx, userID)
	}
	return nil, nil
}

func TestCreateEvent_Success(t *testing.T) {
	mock := &mockQuerier{
		createEventFn: func(ctx context.Context, params repo.CreateEventParams) (repo.Event, error) {
			return repo.Event{
				UserID:    params.UserID,
				EventType: params.EventType,
				Value:     params.Value,
			}, nil
		},
	}

	svc := events.NewService(mock)
	ctx := context.Background()

	event, err := svc.CreateEvent(ctx, repo.CreateEventParams{
		UserID:    "user-1",
		EventType: "click",
		Value:     1,
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if event.UserID != "user-1" {
		t.Errorf("expected user-1, got %s", event.UserID)
	}
	if event.EventType != "click" {
		t.Errorf("expected click, got %s", event.EventType)
	}
}

func TestCreateEvent_RepoError(t *testing.T) {
	mock := &mockQuerier{
		createEventFn: func(ctx context.Context, params repo.CreateEventParams) (repo.Event, error) {
			return repo.Event{}, errors.New("db error")
		},
	}

	svc := events.NewService(mock)
	_, err := svc.CreateEvent(context.Background(), repo.CreateEventParams{
		UserID:    "user-1",
		EventType: "click",
		Value:     1,
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestListAggregatedEvents_CachesResult(t *testing.T) {
	callCount := 0
	mock := &mockQuerier{
		listAggregatedEventsFn: func(ctx context.Context, userID string) ([]repo.ListAgrregatedEventsRow, error) {
			callCount++
			return []repo.ListAgrregatedEventsRow{
				{EventType: "click", EventCount: 5},
			}, nil
		},
	}

	svc := events.NewService(mock)
	ctx := context.Background()

	// First call — should hit the repo
	result, err := svc.ListAgrregatedEvents(ctx, "user-1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result["click"] != 5 {
		t.Errorf("expected 5, got %d", result["click"])
	}

	// Second call — should use cache, not hit repo again
	svc.ListAgrregatedEvents(ctx, "user-1")
	if callCount != 1 {
		t.Errorf("expected repo to be called once, got %d", callCount)
	}
}

func TestListAggregatedEvents_RepoError(t *testing.T) {
	mock := &mockQuerier{
		listAggregatedEventsFn: func(ctx context.Context, userID string) ([]repo.ListAgrregatedEventsRow, error) {
			return nil, errors.New("db error")
		},
	}

	svc := events.NewService(mock)
	_, err := svc.ListAgrregatedEvents(context.Background(), "user-1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCreateEvent_UpdatesCache(t *testing.T) {
	mock := &mockQuerier{
		listAggregatedEventsFn: func(ctx context.Context, userID string) ([]repo.ListAgrregatedEventsRow, error) {
			return []repo.ListAgrregatedEventsRow{
				{EventType: "click", EventCount: 5},
			}, nil
		},
		createEventFn: func(ctx context.Context, params repo.CreateEventParams) (repo.Event, error) {
			return repo.Event{
				UserID:    params.UserID,
				EventType: params.EventType,
				Value:     params.Value,
			}, nil
		},
	}

	svc := events.NewService(mock)
	ctx := context.Background()

	// Populate cache
	svc.ListAgrregatedEvents(ctx, "user-1")

	// Create event — should update cache
	svc.CreateEvent(ctx, repo.CreateEventParams{
		UserID:    "user-1",
		EventType: "click",
		Value:     3,
	})

	result, err := svc.ListAgrregatedEvents(ctx, "user-1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result["click"] != 8 {
		t.Errorf("expected 8 (5+3), got %d", result["click"])
	}
}

func TestRaceCondition(t *testing.T) {
	mock := &mockQuerier{
		createEventFn: func(ctx context.Context, params repo.CreateEventParams) (repo.Event, error) {
			return repo.Event{
				UserID:    params.UserID,
				EventType: params.EventType,
				Value:     params.Value,
			}, nil
		},
		listAggregatedEventsFn: func(ctx context.Context, userID string) ([]repo.ListAgrregatedEventsRow, error) {
			return []repo.ListAgrregatedEventsRow{}, nil
		},
	}

	svc := events.NewService(mock)
	ctx := context.Background()

	var wg sync.WaitGroup
	for range 100 {
		wg.Add(2)
		go func() {
			defer wg.Done()
			svc.CreateEvent(ctx, repo.CreateEventParams{
				UserID:    "user-1",
				EventType: "click",
				Value:     1,
			})
		}()
		go func() {
			defer wg.Done()
			svc.ListAgrregatedEvents(ctx, "user-1")
		}()
	}
	wg.Wait()
}
