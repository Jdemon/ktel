package ktel

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Jdemon/ktel/config"
	"github.com/Jdemon/ktel/consumer"
	"github.com/Jdemon/ktel/health"
	internalkgo "github.com/Jdemon/ktel/kgo"
	"github.com/Jdemon/ktel/logger"
	"github.com/Jdemon/ktel/otel"
	"github.com/Jdemon/ktel/processor"
	"github.com/Jdemon/ktel/telemetry"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap"
)

type app struct {
	cfg            *config.Config
	Logger         *zap.SugaredLogger
	KafkaClient    *kgo.Client
	HealthChecker  *health.Checker
	tracerProvider *sdktrace.TracerProvider
	meterProvider  *metric.MeterProvider
}

func New() (*app, error) {
	// Load configuration
	cfg, err := config.New()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Initialize logger
	if err = logger.New(cfg.AppName); err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	tp, mp, err := otel.InitOtelProviders(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize OpenTelemetry providers: %w", err)
	}

	healthChecker := health.NewChecker()

	kgoOptions := internalkgo.BuildKgoOptions(cfg, tp, healthChecker)
	kafkaClient, err := kgo.NewClient(kgoOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka client: %w", err)
	}

	return &app{
		cfg:            cfg,
		Logger:         zap.S(),
		KafkaClient:    kafkaClient,
		HealthChecker:  healthChecker,
		tracerProvider: tp,
		meterProvider:  mp,
	}, nil
}

func (a *app) Start(proc processor.Processor, cleanupFns ...func()) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var wg sync.WaitGroup

	// Start health check server
	httpServer := a.startHealthCheckServer(ctx, &wg)

	// Start Kafka consumer
	if err := a.startConsumer(ctx, &wg, proc); err != nil {
		return err
	}

	// Wait for termination signal
	<-ctx.Done()
	a.Logger.Info("Termination signal received, initiating graceful shutdown...")

	// Shutdown HTTP server
	a.shutdownHTTPServer(httpServer)

	// Shutdown OpenTelemetry providers
	a.shutdownOtelProviders()

	// Close Kafka client
	a.KafkaClient.Close()

	for _, fn := range cleanupFns {
		fn()
	}
	a.Logger.Info("Cleanup tasks completed.")

	// Wait for all goroutines to finish
	wg.Wait()

	a.Logger.Debug("All services shut down gracefully.")
	return nil
}

func (a *app) startHealthCheckServer(_ context.Context, wg *sync.WaitGroup) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/live", a.HealthChecker.LivenessProbe)
	mux.HandleFunc("/ready", a.HealthChecker.ReadinessProbe)

	server := &http.Server{
		Addr:    ":" + a.cfg.Server.Port,
		Handler: mux,
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		a.Logger.Debugf("Health check server starting on port %s", a.cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			a.Logger.Fatalf("Health check server failed: %v", err)
		}
		a.Logger.Debug("Health check server stopped.")
	}()

	return server
}

func (a *app) startConsumer(ctx context.Context, wg *sync.WaitGroup, proc processor.Processor) error {
	instrumentor, err := telemetry.NewInstrumentor()
	if err != nil {
		return fmt.Errorf("failed to create telemetry instrumentor: %w", err)
	}

	clientAdapter := &consumer.KgoClientAdapter{Client: a.KafkaClient}
	instrumentedProc := processor.NewInstrumentingProcessor(proc, instrumentor, a.tracerProvider.Tracer(a.cfg.AppName))
	appConsumer := consumer.New(clientAdapter, instrumentedProc, a.Logger)

	a.Logger.Debug("Kafka consumer started...")

	wg.Add(1)
	go func() {
		defer wg.Done()
		appConsumer.Run(ctx)
	}()

	return nil
}

func (a *app) shutdownHTTPServer(server *http.Server) {
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()

	if err := server.Shutdown(shutdownCtx); err != nil {
		a.Logger.Warnw("HTTP server shutdown error", "error", err)
	} else {
		a.Logger.Debug("HTTP server shutdown complete.")
	}
}

func (a *app) shutdownOtelProviders() {
	if a.tracerProvider != nil {
		if err := a.tracerProvider.Shutdown(context.Background()); err != nil {
			a.Logger.Errorf("Error shutting down tracer provider: %v", err)
		}
	}
	if a.meterProvider != nil {
		if err := a.meterProvider.Shutdown(context.Background()); err != nil {
			a.Logger.Errorf("Error shutting down meter provider: %v", err)
		}
	}
}
