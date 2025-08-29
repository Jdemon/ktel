package consumer

import (
	"context"
	"sync"

	"github.com/Jdemon/ktel/processor"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.uber.org/zap"
)

// KafkaClient defines the interface for the Kafka client operations we need.
type KafkaClient interface {
	PollFetches(context.Context) Fetches
	Close()
}

// Fetches defines the interface for the result of a poll operation.
type Fetches interface {
	Errors() []kgo.FetchError
	EachRecord(func(*kgo.Record))
}

// KgoClientAdapter adapts the concrete *kgo.Client to our KafkaClient interface.
type KgoClientAdapter struct {
	Client *kgo.Client
}

func (a *KgoClientAdapter) PollFetches(ctx context.Context) Fetches {
	return a.Client.PollFetches(ctx)
}

func (a *KgoClientAdapter) Close() {
	a.Client.Close()
}

// Consumer handles the message processing logic.
type Consumer struct {
	client    KafkaClient
	processor processor.Processor
	logger    *zap.SugaredLogger
}

func New(client KafkaClient, processor processor.Processor, logger *zap.SugaredLogger) *Consumer {
	return &Consumer{
		client:    client,
		processor: processor,
		logger:    logger,
	}
}

func (c *Consumer) Run(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			c.logger.Info("Context cancelled, stopping consumer poll loop.")
			return
		}

		fetches := c.client.PollFetches(ctx)
		if errs := fetches.Errors(); len(errs) > 0 {
			for _, e := range errs {
				c.logger.Errorw("Kafka fetch error", "topic", e.Topic, "partition", e.Partition, "error", e.Err)
			}
			continue
		}

		var wg sync.WaitGroup
		fetches.EachRecord(func(record *kgo.Record) {
			wg.Add(1)
			go func(rec *kgo.Record) {
				defer wg.Done()
				if err := c.processor.ProcessRecord(rec.Context, rec); err != nil {
					c.logger.Errorw("Failed to process record", "error", err, "topic", rec.Topic, "partition", rec.Partition, "offset", rec.Offset)
				}
			}(record)
		})
		wg.Wait()
	}
}
