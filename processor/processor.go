package processor

import (
	"context"
	"fmt"
	"time"

	"github.com/Jdemon/ktel/telemetry"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.opentelemetry.io/otel/trace"
)

// Processor defines the interface for processing Kafka records.
type Processor interface {
	ProcessRecord(ctx context.Context, record *kgo.Record) error
}

// InstrumentingProcessor is a decorator that adds instrumentation to a Processor.
type InstrumentingProcessor struct {
	processor    Processor
	instrumentor *telemetry.Instrumentor
	tracer       trace.Tracer
}

// NewInstrumentingProcessor creates a new InstrumentingProcessor.
func NewInstrumentingProcessor(processor Processor, instrumentor *telemetry.Instrumentor, tracer trace.Tracer) *InstrumentingProcessor {
	return &InstrumentingProcessor{
		processor:    processor,
		instrumentor: instrumentor,
		tracer:       tracer,
	}
}

// ProcessRecord processes a Kafka record and instruments the operation.
func (p *InstrumentingProcessor) ProcessRecord(ctx context.Context, record *kgo.Record) (err error) {
	spanName := fmt.Sprintf("%s process", record.Topic)
	ctx, span := p.tracer.Start(ctx, spanName)
	defer span.End()

	startTime := time.Now()
	defer func() {
		p.instrumentor.InstrumentMessage(ctx, record, err == nil, startTime)
	}()

	return p.processor.ProcessRecord(ctx, record)
}
