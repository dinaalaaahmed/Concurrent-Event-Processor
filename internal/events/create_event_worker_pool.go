package events

import (
	"context"
	"log"
	repo "my-app/internal/adapters/postgresql/sqlc"
	"sync"
	"time"
)

type CreateEventWorker struct {
	jobQueue  chan Job
	processor Service
	wg        sync.WaitGroup
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
		d.wg.Add(1)
		go func(workerID int) {
			defer d.wg.Done()
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

func (d *CreateEventWorker) Stop() {
	close(d.jobQueue)
	d.wg.Wait()
}
