package main

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/Jdemon/ktel"
	"github.com/Jdemon/ktel/processor"
	"github.com/goccy/go-json"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("Application failed: %v", err)
	}
}

func run() error {
	app, err := ktel.New()
	if err != nil {
		return fmt.Errorf("failed to create application: %w", err)
	}

	return app.Start(NewExampleProcessor(app.Logger, app.KafkaClient))
}

// ResultMessage defines the structure of the incoming Kafka message
type ResultMessage struct {
	TransactionRef string `json:"transactionRef"`
	Code           string `json:"code"`
}

// ExampleProcessor processes
type ExampleProcessor struct {
	logger      *zap.SugaredLogger
	trxPool     *sync.Pool
	KafkaClient *kgo.Client
}

// NewExampleProcessor creates a new NewExampleProcessor.
func NewExampleProcessor(logger *zap.SugaredLogger, kafkaClient *kgo.Client) processor.Processor {
	return &ExampleProcessor{
		logger: logger,
		trxPool: &sync.Pool{
			New: func() interface{} {
				return new(ResultMessage)
			},
		},
		KafkaClient: kafkaClient,
	}
}

// ProcessRecord processes a single Kafka record for e-withholding.
func (p *ExampleProcessor) ProcessRecord(ctx context.Context, record *kgo.Record) (err error) {
	span := trace.SpanFromContext(ctx)

	msg := p.trxPool.Get().(*ResultMessage)
	defer func() {
		*msg = ResultMessage{} // Reset struct for reuse
		p.trxPool.Put(msg)
	}()

	if err = p.unmarshalMessage(record, msg); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to unmarshal message")
		return err
	}

	span.SetAttributes(
		attribute.String("transaction.ref", msg.TransactionRef),
		attribute.String("ddp.result.code", msg.Code),
	)

	p.logMessage(msg)

	resultRecord := &kgo.Record{
		Topic: "result.topic",
		Value: []byte("value"),
	}
	if err = p.KafkaClient.ProduceSync(ctx, resultRecord).FirstErr(); err != nil {
		return err
	}

	span.SetStatus(codes.Ok, "message processed successfully")
	return nil
}

func (p *ExampleProcessor) unmarshalMessage(record *kgo.Record, msg *ResultMessage) error {
	if err := json.Unmarshal(record.Value, msg); err != nil {
		p.logger.Errorw("Failed to unmarshal message", "error", err, "message", string(record.Value))
		return err
	}
	return nil
}

func (p *ExampleProcessor) logMessage(msg *ResultMessage) {
	p.logger.Debugw("Consumed message successfully", "transactionRef", msg.TransactionRef, "status", msg.Code)
}
