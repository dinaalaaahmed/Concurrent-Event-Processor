package tests

import (
	"context"
	"fmt"
	repo "my-app/internal/adapters/postgresql/sqlc"
	"sync"
	"testing"
	"time"

	"my-app/internal/events"
)

func TestEnqueue_Success(t *testing.T) {
	svc := &mockService{}
	worker := events.NewCreateEventWorker(svc, 10)

	job := events.Job{
		ID:        "job-1",
		Payload:   repo.CreateEventParams{},
		CreatedAt: time.Now(),
	}

	if ok := worker.Enqueue(job); !ok {
		t.Error("expected enqueue to succeed, got false")
	}
}

func TestEnqueue_BufferFull(t *testing.T) {
	svc := &mockService{}
	worker := events.NewCreateEventWorker(svc, 0)

	job := events.Job{
		ID:        "job-1",
		Payload:   repo.CreateEventParams{},
		CreatedAt: time.Now(),
	}

	if ok := worker.Enqueue(job); ok {
		t.Error("expected enqueue to fail when buffer is full, got true")
	}
}

func TestWorker_ProcessesJobs(t *testing.T) {
	processed := 0

	svc := &mockService{
		createEventFn: func(ctx context.Context, params repo.CreateEventParams) (repo.Event, error) {
			processed++
			return repo.Event{}, nil
		},
	}

	worker := events.NewCreateEventWorker(svc, 10)
	worker.Start(1)

	for i := range 5 {
		worker.Enqueue(events.Job{
			ID:        fmt.Sprintf("job-%d", i),
			Payload:   repo.CreateEventParams{UserID: "user-1", EventType: "click", Value: 1},
			CreatedAt: time.Now(),
		})
	}

	worker.Stop()

	if processed != 5 {
		t.Errorf("expected 5 jobs processed, got %d", processed)
	}
}

func TestWorker_ConcurrentEnqueue(t *testing.T) {
	processed := 0

	svc := &mockService{
		createEventFn: func(ctx context.Context, params repo.CreateEventParams) (repo.Event, error) {
			processed++
			return repo.Event{}, nil
		},
	}

	worker := events.NewCreateEventWorker(svc, 100)
	worker.Start(1)

	var wg sync.WaitGroup
	for i := range 50 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			worker.Enqueue(events.Job{
				ID:        fmt.Sprintf("job-%d", i),
				Payload:   repo.CreateEventParams{UserID: "user-1", EventType: "click", Value: 1},
				CreatedAt: time.Now(),
			})
		}(i)
	}
	wg.Wait()

	worker.Stop()

	if processed != 50 {
		t.Errorf("expected 50 jobs processed, got %d", processed)
	}
}
