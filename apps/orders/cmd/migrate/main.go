package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	postgres "github.com/chouaib-skitou/streamweave/apps/orders/internal/adapters/outbound/postgres"
)

func main() {
	if err := run(); err != nil {
		slog.Error("orders migration failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	databaseURL := os.Getenv("ORDERS_DATABASE_URL")
	if databaseURL == "" {
		return errors.New("ORDERS_DATABASE_URL is required")
	}
	migrationsPath := os.Getenv("ORDERS_MIGRATIONS_PATH")
	if migrationsPath == "" {
		migrationsPath = "database/orders/migrations"
	}
	database, err := postgres.Open(databaseURL)
	if err != nil {
		return err
	}
	defer database.Close()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	deadline, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return database.Migrate(deadline, migrationsPath)
}
