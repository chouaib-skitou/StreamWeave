package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	httpadapter "github.com/chouaib-skitou/streamweave/apps/orders/internal/adapters/inbound/http"
	apphealth "github.com/chouaib-skitou/streamweave/apps/orders/internal/application/health"
	"github.com/chouaib-skitou/streamweave/apps/orders/internal/platform/config"
	"github.com/chouaib-skitou/streamweave/apps/orders/internal/platform/logging"
	"github.com/chouaib-skitou/streamweave/apps/orders/internal/platform/telemetry"
)

func main() {
	if err := run(); err != nil {
		slog.Error("orders failed to start", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load(config.OSEnv{})
	if err != nil {
		return err
	}
	logger := logging.New(cfg.LogLevel)
	shutdownTelemetry, err := telemetry.Setup(context.Background(), cfg.OTelEndpoint, cfg.Environment, cfg.OTelInsecure)
	if err != nil {
		return err
	}
	defer func() { _ = shutdownTelemetry(context.Background()) }()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	server, err := httpadapter.NewServer(cfg.HTTPAddr, cfg.MetricsPath, cfg.ReadinessTimeout, apphealth.NewService(), logger, cfg.OpenAPIPath)
	if err != nil {
		return err
	}
	logger.Info("orders starting", "environment", cfg.Environment, "http_addr", cfg.HTTPAddr, "config", cfg.SafeSummary())
	serverErrors := make(chan error, 1)
	go func() { serverErrors <- server.Start() }()
	select {
	case err := <-serverErrors:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil && !errors.Is(err, context.Canceled) {
			return err
		}
		logger.Info("orders stopped")
		return nil
	}
}
