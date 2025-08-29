package kgo

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strings"

	"github.com/Jdemon/ktel/config"
	"github.com/Jdemon/ktel/health"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/sasl/plain"
	"github.com/twmb/franz-go/pkg/sasl/scram"
	"github.com/twmb/franz-go/plugin/kotel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap"
)

// BuildKgoOptions builds the options for the franz-go Kafka client.
func BuildKgoOptions(cfg *config.Config, tp *sdktrace.TracerProvider, checker *health.Checker) []kgo.Opt {
	opts := []kgo.Opt{
		kgo.RequiredAcks(kgo.AllISRAcks()),
		kgo.SeedBrokers(strings.Split(cfg.Kafka.Brokers, ",")...),
		kgo.ConsumerGroup(cfg.Kafka.GroupID),
		kgo.ConsumeTopics(cfg.Kafka.Topic),
		kgo.OnPartitionsAssigned(func(_ context.Context, c *kgo.Client, assigned map[string][]int32) {
			zap.S().Infow("Partitions assigned", "partitions", assigned)
			checker.SetReady(true)
		}),
		kgo.OnPartitionsRevoked(func(_ context.Context, c *kgo.Client, revoked map[string][]int32) {
			zap.S().Infow("Partitions revoked", "partitions", revoked)
			checker.SetReady(false)
		}),
		kgo.OnPartitionsLost(func(_ context.Context, c *kgo.Client, lost map[string][]int32) {
			zap.S().Warnw("Partitions lost", "partitions", lost)
			checker.SetReady(false)
		}),
		// Performance tuning options
		kgo.FetchMaxBytes(1024 * 1024 * 5), // 5MB
	}

	if cfg.Otel.Enabled {
		tracerOpts := []kotel.TracerOpt{
			kotel.TracerProvider(tp),
			kotel.TracerPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{})),
		}
		tracer := kotel.NewTracer(tracerOpts...)
		kotelOps := []kotel.Opt{
			kotel.WithTracer(tracer),
		}
		kotelService := kotel.NewKotel(kotelOps...)
		opts = append(opts, kgo.WithHooks(kotelService.Hooks()...))
	}

	switch strings.ToLower(cfg.Kafka.RebalanceStrategy) {
	case "roundrobin":
		opts = append(opts, kgo.Balancers(kgo.RoundRobinBalancer()))
	case "range":
		opts = append(opts, kgo.Balancers(kgo.RangeBalancer()))
	case "sticky":
		opts = append(opts, kgo.Balancers(kgo.StickyBalancer()))
	default:
		zap.S().Warnf("Unknown or empty rebalance strategy '%s', using default.", cfg.Kafka.RebalanceStrategy)
	}

	if cfg.Kafka.TLS.Enabled {
		tlsConfig, err := createTLSConfig(cfg.Kafka.TLS.CertFile, cfg.Kafka.TLS.KeyFile, cfg.Kafka.TLS.CAFile)
		if err != nil {
			zap.S().Fatalf("Failed to create TLS config: %v", err)
		}
		opts = append(opts, kgo.DialTLSConfig(tlsConfig))
	}

	if cfg.Kafka.SASL.Enabled {
		switch strings.ToUpper(cfg.Kafka.SASL.Mechanism) {
		case "PLAIN":
			opts = append(opts, kgo.SASL(plain.Auth{User: cfg.Kafka.SASL.Username, Pass: cfg.Kafka.SASL.Password}.AsMechanism()))
		case "SCRAM-SHA-256":
			opts = append(opts, kgo.SASL(scram.Auth{User: cfg.Kafka.SASL.Username, Pass: cfg.Kafka.SASL.Password}.AsSha256Mechanism()))
		case "SCRAM-SHA-512":
			opts = append(opts, kgo.SASL(scram.Auth{User: cfg.Kafka.SASL.Username, Pass: cfg.Kafka.SASL.Password}.AsSha512Mechanism()))
		default:
			zap.S().Fatalf("Unsupported SASL mechanism: %s", cfg.Kafka.SASL.Mechanism)
		}
	}

	return opts
}

func createTLSConfig(certFile, keyFile, caFile string) (*tls.Config, error) {
	tlsConfig := &tls.Config{}
	if certFile != "" && keyFile != "" {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return nil, fmt.Errorf("could not load client key pair: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	if caFile != "" {
		caCert, err := os.ReadFile(caFile)
		if err != nil {
			return nil, fmt.Errorf("could not read CA certificate: %w", err)
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
		tlsConfig.RootCAs = caCertPool
	}

	return tlsConfig, nil
}
