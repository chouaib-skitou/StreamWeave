package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	httpadapter "github.com/chouaib-skitou/event-driven-ecommerce-platform/apps/identity/internal/adapters/inbound/http"
	identitycrypto "github.com/chouaib-skitou/event-driven-ecommerce-platform/apps/identity/internal/adapters/outbound/crypto"
	identitykafka "github.com/chouaib-skitou/event-driven-ecommerce-platform/apps/identity/internal/adapters/outbound/kafka"
	"github.com/chouaib-skitou/event-driven-ecommerce-platform/apps/identity/internal/adapters/outbound/postgres"
	identityredis "github.com/chouaib-skitou/event-driven-ecommerce-platform/apps/identity/internal/adapters/outbound/redis"
	"github.com/chouaib-skitou/event-driven-ecommerce-platform/apps/identity/internal/application/health"
	identityapp "github.com/chouaib-skitou/event-driven-ecommerce-platform/apps/identity/internal/application/identity"
	"github.com/chouaib-skitou/event-driven-ecommerce-platform/apps/identity/internal/platform/config"
	"github.com/chouaib-skitou/event-driven-ecommerce-platform/apps/identity/internal/platform/logging"
	"github.com/chouaib-skitou/event-driven-ecommerce-platform/apps/identity/internal/platform/runtime"
)

func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load(config.OSEnv{})
	if err != nil {
		return err
	}
	logger := logging.New(cfg.LogLevel)
	logger.Info("identity starting", "summary", cfg.SafeSummary())

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	database, err := postgres.Open(cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer database.Close()

	startupCtx, cancelStartup := context.WithTimeout(ctx, cfg.ReadinessTimeout)
	defer cancelStartup()
	if err := database.Check(startupCtx); err != nil {
		return err
	}
	if cfg.RunMigrations {
		if err := database.Migrate(startupCtx, cfg.MigrationDir); err != nil {
			return err
		}
	}
	cache, err := identityredis.Open(cfg.RedisURL)
	if err != nil {
		return err
	}
	defer cache.Close()
	var signer *identitycrypto.Signer
	if cfg.SigningKeyPath != "" {
		signer, err = identitycrypto.LoadSigner(cfg.SigningKeyPath, cfg.SigningKeyID, cfg.Issuer)
	} else {
		signer, err = identitycrypto.GenerateSigner(cfg.SigningKeyID, cfg.Issuer)
	}
	if err != nil {
		return err
	}
	store := postgres.NewStore(database)
	limiter := identityredis.NewRateLimiter(cache, cfg.RateLoginFailures, cfg.RateRefreshMinute, cfg.RateResetHour, cfg.RateServiceMinute)
	identityService, err := identityapp.NewService(identityapp.Options{Store: store, Passwords: identitycrypto.NewPasswordHasher(), Tokens: identitycrypto.NewTokenServiceForSigner(signer), Clock: runtime.Clock{}, Limiter: limiter, Revocations: cache, Mailer: runtime.NewSimulatedMailer(logger), HumanAudience: cfg.HumanAudience, MachineAudience: cfg.MachineAudience})
	if err != nil {
		return err
	}

	healthService := health.NewService(database, cache)
	healthService.MarkStarted()
	publisher := identitykafka.NewPublisher(cfg.KafkaBrokers)
	defer publisher.Close()
	relay := identitykafka.NewRelay(store, publisher)
	go func() {
		if relayErr := relay.Run(ctx); relayErr != nil {
			logger.Error("identity outbox relay stopped", "error", relayErr)
		}
	}()
	server := httpadapter.NewApplicationServer(cfg.HTTPAddr, cfg.MetricsPath, cfg.ReadinessTimeout, healthService, logger, identityService, signer, cache, cfg.HumanAudience, cfg.DemoRegistration)
	serverErrors := make(chan error, 1)
	go func() { serverErrors <- server.Start() }()

	select {
	case err := <-serverErrors:
		if err != nil {
			logger.Error("identity server stopped unexpectedly", "error", err)
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancelShutdown()
		if err := server.Shutdown(shutdownCtx); err != nil && !errors.Is(err, context.Canceled) {
			logger.Error("identity shutdown failed", "error", err)
			return err
		}
		logger.Info("identity stopped", "reason", slog.StringValue("signal"))
		return nil
	}
}
