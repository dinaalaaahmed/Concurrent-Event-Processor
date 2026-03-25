package events

import (
	"context"
	"maps"
	repo "my-app/internal/adapters/postgresql/sqlc"
	"sync"
)

type Service interface {
	ListAgrregatedEvents(ctx context.Context, userId string) (map[string]int64, error)
	CreateEvent(ctx context.Context, params repo.CreateEventParams) (repo.Event, error)
}

type svc struct {
	mu    sync.RWMutex
	stats map[string]map[string]int64
	repo  repo.Querier
}

func NewService(repo repo.Querier) Service {
	return &svc{repo: repo, stats: make(map[string]map[string]int64)}

}

func (s *svc) ListAgrregatedEvents(ctx context.Context, userID string) (map[string]int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stats[userID] == nil {
		s.stats[userID] = make(map[string]int64)
		rows, err := s.repo.ListAgrregatedEvents(ctx, userID)
		if err != nil {
			return nil, err
		}

		for _, row := range rows {
			s.stats[userID][row.EventType] = row.EventCount
		}
	}

	result := make(map[string]int64)
	maps.Copy(result, s.stats[userID])

	return result, nil
}

func (s *svc) CreateEvent(ctx context.Context, params repo.CreateEventParams) (repo.Event, error) {

	event, err := s.repo.CreateEvent(ctx, params)
	if err != nil {
		return repo.Event{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stats[params.UserID] == nil {
		s.stats[params.UserID] = make(map[string]int64)
	}
	s.stats[params.UserID][params.EventType] += params.Value

	return event, nil
}
