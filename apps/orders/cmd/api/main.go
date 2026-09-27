package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	httpadapter "github.com/chouaib-skitou/streamweave/apps/orders/internal/adapters/inbound/http"
	postgres "github.com/chouaib-skitou/streamweave/apps/orders/internal/adapters/outbound/postgres"
	apphealth "github.com/chouaib-skitou/streamweave/apps/orders/internal/application/health"
	apporders "github.com/chouaib-skitou/streamweave/apps/orders/internal/application/orders"
	"github.com/chouaib-skitou/streamweave/apps/orders/internal/platform/config"
	"github.com/chouaib-skitou/streamweave/apps/orders/internal/platform/logging"
	"github.com/chouaib-skitou/streamweave/apps/orders/internal/platform/telemetry"
	"github.com/google/uuid"
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
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		_ = shutdownTelemetry(ctx)
	}()
	database, err := postgres.Open(cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer database.Close()
	store := postgres.NewStoreWithQuoteVerifierAndCursor(database, nil, cfg.CursorSecret)
	service, err := apporders.NewService(store, orderIDs{}, systemClock{})
	if err != nil {
		return err
	}
	authenticator, err := httpadapter.NewGatewayAuthenticator(cfg.GatewayServiceToken, cfg.GatewayJWKSURL, cfg.GatewayIssuer, cfg.GatewayAudience, cfg.GatewaySubject)
	if err != nil {
		return err
	}
	dependencies := []apphealth.Dependency{database}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	server, err := httpadapter.NewApplicationServer(cfg.HTTPAddr, cfg.MetricsPath, cfg.ReadinessTimeout, apphealth.NewService(dependencies...), logger, cfg.OpenAPIPath, service, authenticator)
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

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now().UTC() }

type orderIDs struct{}

func (orderIDs) NewOrderID() string { return "ord_" + strings.ReplaceAll(uuid.NewString(), "-", "") }
