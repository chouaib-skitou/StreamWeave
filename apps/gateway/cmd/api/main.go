package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpadapter "github.com/chouaib-skitou/streamweave/apps/gateway/internal/adapters/inbound/http"
	upstreamhttp "github.com/chouaib-skitou/streamweave/apps/gateway/internal/adapters/outbound/http"
	identityclient "github.com/chouaib-skitou/streamweave/apps/gateway/internal/adapters/outbound/identity"
	"github.com/chouaib-skitou/streamweave/apps/gateway/internal/adapters/outbound/jwks"
	redisadapter "github.com/chouaib-skitou/streamweave/apps/gateway/internal/adapters/outbound/redis"
	app "github.com/chouaib-skitou/streamweave/apps/gateway/internal/application/gateway"
	"github.com/chouaib-skitou/streamweave/apps/gateway/internal/platform/config"
	"github.com/chouaib-skitou/streamweave/apps/gateway/internal/platform/logging"
	"github.com/chouaib-skitou/streamweave/apps/gateway/internal/platform/telemetry"
)

func main() {
	if err := run(); err != nil {
		slog.Error("gateway failed to start", "error", err)
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
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	redis, err := redisadapter.Open(cfg.RedisURL)
	if err != nil {
		return err
	}
	defer redis.Close()
	identityHTTP := &http.Client{Timeout: 5 * time.Second}
	verifier, err := jwks.NewVerifier(jwks.HTTPFetcher{URL: cfg.JWKSURL, Client: identityHTTP}, cfg.Issuer, cfg.HumanAudience)
	if err != nil {
		return err
	}
	startupCtx, cancel := context.WithTimeout(ctx, cfg.ReadinessTimeout)
	defer cancel()
	if err := verifier.Check(startupCtx); err != nil {
		return err
	}
	if err := redis.Check(startupCtx); err != nil {
		return err
	}
	var serviceTokens app.ServiceTokenProvider
	if cfg.StaticServiceToken != "" {
		serviceTokens = identityclient.StaticToken(cfg.StaticServiceToken)
	} else {
		serviceTokens, err = identityclient.NewClient(cfg.IdentityURL, cfg.ServiceClientID, cfg.ServiceClientSecret, cfg.ServiceScopes, identityHTTP)
		if err != nil {
			return err
		}
	}
	upstream, err := upstreamhttp.NewClient(map[string]string{"identity": cfg.IdentityURL, "orders": cfg.OrdersURL}, &http.Client{Timeout: 12 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}, cfg.MaxResponseBytes)
	if err != nil {
		return err
	}
	service, err := app.NewService(verifier, redis, serviceTokens, upstream)
	if err != nil {
		return err
	}
	server, err := httpadapter.NewServer(cfg.HTTPAddr, cfg.MetricsPath, cfg.ReadinessTimeout, service, verifier, redis, logger, cfg.OpenAPIPath, cfg.Environment == "development", cfg.TrustProxyHeaders, cfg.CORSOrigins)
	if err != nil {
		return err
	}
	logger.Info("gateway starting", "environment", cfg.Environment, "http_addr", cfg.HTTPAddr, "identity_url", cfg.IdentityURL, "orders_url", cfg.OrdersURL)
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
		logger.Info("gateway stopped")
		return nil
	}
}
