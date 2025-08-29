package telemetry

import (
	"context"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

const (
	instrumentationName = "kafka-consumer"
)

// Instrumentor holds the OpenTelemetry instruments and provides methods for common instrumentation.
type Instrumentor struct {
	MessagesProcessedCounter metric.Int64Counter
	ProcessingTimeHistogram  metric.Float64Histogram
}

// NewInstrumentor creates and initializes the OpenTelemetry instruments.
func NewInstrumentor() (*Instrumentor, error) {
	meter := otel.Meter(instrumentationName)
	messagesProcessedCounter, err := meter.Int64Counter(
		"kafka.messages.processed",
		metric.WithDescription("The number of Kafka messages processed"),
		metric.WithUnit("{message}"),
	)
	if err != nil {
		return nil, err
	}

	processingTimeHistogram, err := meter.Float64Histogram(
		"kafka.message.processing.duration",
		metric.WithDescription("The latency of processing Kafka messages"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, err
	}

	return &Instrumentor{
		MessagesProcessedCounter: messagesProcessedCounter,
		ProcessingTimeHistogram:  processingTimeHistogram,
	}, nil
}

// InstrumentMessage instruments a message processing operation with metrics and trace attributes.
func (i *Instrumentor) InstrumentMessage(ctx context.Context, record *kgo.Record, success bool, startTime time.Time) {
	duration := float64(time.Since(startTime).Microseconds()) / 1000.0
	metricAttrs := attribute.NewSet(
		attribute.String("messaging.kafka.topic", record.Topic),
		attribute.Bool("success", success),
	)
	i.MessagesProcessedCounter.Add(ctx, 1, metric.WithAttributeSet(metricAttrs))
	i.ProcessingTimeHistogram.Record(ctx, duration, metric.WithAttributeSet(metricAttrs))

	span := trace.SpanFromContext(ctx)
	attrs := []attribute.KeyValue{
		attribute.String("messaging.kafka.topic", record.Topic),
		attribute.Int("messaging.kafka.partition", int(record.Partition)),
	}
	span.SetAttributes(attrs...)
}

// Tracer returns a new tracer from the global tracer provider.
func Tracer() trace.Tracer {
	return otel.Tracer(instrumentationName)
}

// HeaderCarrier adapts kafka headers to the TextMapCarrier interface for propagation.
type HeaderCarrier []kgo.RecordHeader

func (hc HeaderCarrier) Get(key string) string {
	for _, h := range hc {
		if h.Key == key {
			return string(h.Value)
		}
	}
	return ""
}
func (hc HeaderCarrier) Set(key, value string) {}
func (hc HeaderCarrier) Keys() []string {
	keys := make([]string, 0, len(hc))
	for _, h := range hc {
		keys = append(keys, h.Key)
	}
	return keys
}
