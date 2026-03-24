package main

import (
	"context"
	"log"
	"log/slog"
	"my-app/internal/env"
	"os"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	ctx := context.Background()

	cfg := config{
		addr: ":3000",
		db: dbConfig{
			dsn: env.GetString("DB_DSN", "host=localhost user=postgres password=mysecretpassword dbname=events-processing sslmode=disable"),
		},
	}

	// Logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// Database
	// Parse Config from Environment
	dsn := env.GetString("DB_DSN", cfg.db.dsn)
	poolSizeStr := env.GetString("CONNECTION_POOL", "10")
	poolSize, _ := strconv.Atoi(poolSizeStr)

	// Initialize pgxpool
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		log.Fatalf("Unable to parse DSN: %v", err)
	}
	config.MaxConns = int32(poolSize)

	dbPool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		log.Fatalf("Unable to create connection pool: %v", err)
	}
	defer dbPool.Close()

	logger.Info("connected to database", "dsn", cfg.db.dsn)

	api := application{
		config: cfg,
		db:     dbPool,
	}

	if err := api.run(api.mount()); err != nil {
		slog.Error("server failed to start", "error", err)
		os.Exit(1)
	}
}
