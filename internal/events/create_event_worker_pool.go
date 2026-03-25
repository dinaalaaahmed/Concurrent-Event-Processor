package events

import (
	"context"
	"log"
	repo "my-app/internal/adapters/postgresql/sqlc"
	"time"
)

type CreateEventWorker struct {
	jobQueue  chan Job
	processor Service
}

type Job struct {
	ID        string
	Payload   repo.CreateEventParams
	CreatedAt time.Time
}

func NewCreateEventWorker(proc Service, bufferSize int) *CreateEventWorker {
	return &CreateEventWorker{
		jobQueue:  make(chan Job, bufferSize),
		processor: proc,
	}
}

func (d *CreateEventWorker) Start(workerCount int) {
	for i := range workerCount {
		go func(workerID int) {
			for jobData := range d.jobQueue {

				_, err := d.processor.CreateEvent(context.Background(), jobData.Payload)
				if err != nil {
					log.Printf("Worker %d: %v", workerID, err)
				}
			}
		}(i)
	}
}

func (d *CreateEventWorker) Enqueue(data Job) bool {
	select {
	case d.jobQueue <- data:
		return true
	default:
		return false
	}
}
