package tests

import (
	"context"
	repo "my-app/internal/adapters/postgresql/sqlc"
	"sync"
)

type mockService struct {
	listAggregatedEventsFn func(ctx context.Context, userID string) (map[string]int64, error)
	createEventFn          func(ctx context.Context, params repo.CreateEventParams) (repo.Event, error)
}

// Mock implementation of repo.Querier
type mockQuerier struct {
	mu                     sync.Mutex
	createEventFn          func(ctx context.Context, params repo.CreateEventParams) (repo.Event, error)
	listAggregatedEventsFn func(ctx context.Context, userID string) ([]repo.ListAgrregatedEventsRow, error)
}

func (m *mockService) ListAgrregatedEvents(ctx context.Context, userID string) (map[string]int64, error) {
	if m.listAggregatedEventsFn != nil {
		return m.listAggregatedEventsFn(ctx, userID)
	}
	return nil, nil
}

func (m *mockService) CreateEvent(ctx context.Context, params repo.CreateEventParams) (repo.Event, error) {
	if m.createEventFn != nil {
		return m.createEventFn(ctx, params)
	}
	return repo.Event{}, nil
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
