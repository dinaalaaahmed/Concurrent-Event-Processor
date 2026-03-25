package main

import (
	"context"
	"errors"
	"log"
	repo "my-app/internal/adapters/postgresql/sqlc"
	"my-app/internal/env"
	"my-app/internal/events"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
)

func (app *application) mount() (http.Handler, *events.CreateEventWorker) {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Use(middleware.Timeout(5 * time.Second))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("all good"))
	})

	eventService := events.NewService(
		repo.New(app.db),
	)

	workerSizeStr := env.GetString("WORKER_SIZE", "10")
	workerSize, _ := strconv.Atoi(workerSizeStr)

	bufferSizeStr := env.GetString("BUFFER_SIZE", "100")
	bufferSize, _ := strconv.Atoi(bufferSizeStr)

	worker := events.NewCreateEventWorker(eventService, bufferSize)
	worker.Start(workerSize)

	eventHandler := events.NewHandler(eventService, worker)
	r.Get("/stats", eventHandler.ListAggregatedEvents)
	r.Post("/events", eventHandler.CreateEvent)

	return r, worker
}

func (app *application) run(h http.Handler, worker *events.CreateEventWorker) error {
	srv := &http.Server{
		Addr:         app.config.addr,
		Handler:      h,
		WriteTimeout: time.Second * 30,
		ReadTimeout:  time.Second * 10,
		IdleTimeout:  time.Minute,
	}

	shutdown := make(chan error)

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			shutdown <- err
			return
		}

		worker.Stop()
		shutdown <- nil
	}()

	log.Printf("server has started at addr %s", app.config.addr)

	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return <-shutdown
}

type application struct {
	config config
	db     *pgxpool.Pool
}

type config struct {
	addr string
	db   dbConfig
}

type dbConfig struct {
	dsn string
}
