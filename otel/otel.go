package otel

import (
	"context"
	"fmt"

	"github.com/Jdemon/ktel/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.34.0"
	"go.uber.org/zap"
)

// InitOtelProviders initializes the OpenTelemetry tracer and meter providers.
func InitOtelProviders(cfg *config.Config) (*sdktrace.TracerProvider, *metric.MeterProvider, error) {
	if !cfg.Otel.Enabled {
		zap.S().Info("OpenTelemetry is disabled.")
		return nil, nil, nil
	}

	zap.S().Info("OpenTelemetry is enabled. Initializing providers...")

	ctx := context.Background()

	res, err := resource.Merge(resource.Default(), resource.NewWithAttributes(semconv.SchemaURL, semconv.ServiceName(cfg.AppName)))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create resource: %w", err)
	}

	traceExporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithEndpoint(cfg.Otel.Exporter.Grpc.Endpoint), otlptracegrpc.WithInsecure())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create OTLP trace exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(traceExporter), sdktrace.WithResource(res))
	otel.SetTracerProvider(tp)
	zap.S().Info("OpenTelemetry tracer provider initialized.")

	metricExporter, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithEndpoint(cfg.Otel.Exporter.Grpc.Endpoint), otlpmetricgrpc.WithInsecure())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create OTLP metric exporter: %w", err)
	}

	mp := metric.NewMeterProvider(metric.WithReader(metric.NewPeriodicReader(metricExporter)), metric.WithResource(res))
	otel.SetMeterProvider(mp)
	zap.S().Info("OpenTelemetry meter provider initialized.")

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	return tp, mp, nil
}
