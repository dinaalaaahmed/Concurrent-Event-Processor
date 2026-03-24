package events

import (
	"context"
	repo "my-app/internal/adapters/postgresql/sqlc"
)

type Service interface {
	ListAgrregatedEvents(ctx context.Context, userId string) (map[string]int64, error)
	CreateEvent(ctx context.Context, params repo.CreateEventParams) (repo.Event, error)
}

type svc struct {
	repo repo.Querier
}

func NewService(repo repo.Querier) Service {
	return &svc{repo: repo}
}

func (s *svc) ListAgrregatedEvents(ctx context.Context, userID string) (map[string]int64, error) {
	rows, err := s.repo.ListAgrregatedEvents(ctx, userID)
	if err != nil {
		return nil, err
	}

	result := make(map[string]int64)
	for _, row := range rows {
		result[row.EventType] = row.EventCount
	}

	return result, nil
}

func (s *svc) CreateEvent(ctx context.Context, params repo.CreateEventParams) (repo.Event, error) {
	event, err := s.repo.CreateEvent(ctx, params)
	if err != nil {
		return repo.Event{}, err
	}

	return event, nil
}
