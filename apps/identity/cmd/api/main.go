package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	"github.com/chouaib-skitou/event-driven-ecommerce-platform/apps/identity/internal/platform/telemetry"
)

func main() {
	if err := run(); err != nil {
		slog.Error("identity failed to start", "error", err)
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
	shutdownTelemetry, err := telemetry.Setup(context.Background(), cfg.OTelEndpoint, cfg.Environment, cfg.OTelInsecure)
	if err != nil {
		return err
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		_ = shutdownTelemetry(shutdownCtx)
	}()

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
	if cfg.MigrationsOnly {
		logger.Info("identity migrations completed")
		return nil
	}
	cache, err := identityredis.Open(cfg.RedisURL)
	if err != nil {
		return err
	}
	defer cache.Close()
	var signer *identitycrypto.Signer
	if cfg.SigningKeyPath != "" {
		signer, err = identitycrypto.LoadSignerWithHistory(cfg.SigningKeyPath, cfg.SigningKeyID, cfg.Issuer, cfg.SigningKeyHistoryDir)
	} else {
		signer, err = identitycrypto.GenerateSigner(cfg.SigningKeyID, cfg.Issuer)
	}
	if err != nil {
		return err
	}
	store := postgres.NewStore(database)
	limiter := identityredis.NewRateLimiterWithSources(cache, cfg.RateLoginFailures, cfg.RateRefreshMinute, cfg.RateResetHour, cfg.RateServiceMinute, cfg.RateLoginSource, cfg.RateRefreshSource, cfg.RateResetSource, cfg.RateServiceSource)
	var mailer identityapp.Mailer
	var mailbox identityapp.Mailbox
	if cfg.MailerMode == "smtp" {
		mailer = runtime.NewConfiguredSMTPMailer(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUsername, cfg.SMTPPassword, cfg.SMTPFrom, cfg.PublicAppURL)
	} else {
		localMailer := runtime.NewSimulatedMailer(logger)
		mailer = localMailer
		if cfg.TestMailerEnabled {
			mailbox = localMailer
		}
	}
	identityService, err := identityapp.NewService(identityapp.Options{Store: store, Passwords: identitycrypto.NewPasswordHasher(), Tokens: identitycrypto.NewTokenServiceForSigner(signer), Clock: runtime.Clock{}, Limiter: limiter, Revocations: cache, Mailer: mailer, HumanAudience: cfg.HumanAudience, MachineAudience: cfg.MachineAudience})
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
	server := httpadapter.NewApplicationServer(cfg.HTTPAddr, cfg.MetricsPath, cfg.ReadinessTimeout, healthService, logger, identityService, signer, cache, cfg.HumanAudience, cfg.DemoRegistration, cfg.TestMailerEnabled, mailbox)
	server.SetTrustProxyHeaders(cfg.TrustProxyHeaders)
	server.SetEmergencyRevocation(cfg.EmergencyRevocation)
	server.SetDBStatsProvider(database.DB().Stats)
	relay.SetPublishObserver(server.ObserveOutboxPublish)
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			countCtx, cancel := context.WithTimeout(ctx, cfg.ReadinessTimeout)
			count, countErr := store.PendingOutboxCount(countCtx)
			cancel()
			if countErr == nil {
				server.SetOutboxBacklog(count)
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
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
