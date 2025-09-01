package ikgo

import (
	"context"

	"github.com/Jdemon/ktel/telemetry"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.opentelemetry.io/otel"
)

// Producer wraps a kgo.Client to provide a simple interface for producing messages.
type Producer struct {
	client *kgo.Client
}

// NewProducer creates a new Producer.
func NewProducer(client *kgo.Client) *Producer {
	return &Producer{client: client}
}

// Produce produces a single record to the given topic and returns.
func (p *Producer) Produce(ctx context.Context, rec *kgo.Record, promise func(*kgo.Record, error)) {
	p.injectTracingHeaders(ctx, rec)
	p.client.Produce(ctx, rec, promise)
}

func (p *Producer) ProduceSync(ctx context.Context, rec *kgo.Record) {
	p.injectTracingHeaders(ctx, rec)
	p.client.ProduceSync(ctx, rec)
}

func (p *Producer) injectTracingHeaders(ctx context.Context, rec *kgo.Record) {
	propagator := otel.GetTextMapPropagator()
	propagator.Inject(ctx, telemetry.HeaderCarrier(rec.Headers))
}
