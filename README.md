# ktel

`ktel` is a Kafka consumer framework for Go that simplifies the development of observable, production-ready services. It provides a robust set of features to handle common tasks such as configuration, logging, health checks, and OpenTelemetry integration, allowing you to focus on your business logic.

## Features

*   **Configuration Loading**: Easily load and manage your application's configuration.
*   **Structured Logging**: High-performance, structured logging with `zap`.
*   **OpenTelemetry Integration**: Built-in support for distributed tracing and metrics with OpenTelemetry.
*   **Health Checks**: Expose liveness and readiness probes for Kubernetes and other orchestration systems.
*   **Graceful Shutdown**: Handle termination signals to ensure your application shuts down cleanly.
*   **Kafka Consumer**: A managed Kafka consumer that automatically instruments your message processing with traces and metrics.

## Getting Started

### Prerequisites

*   Go 1.24 or later
*   A running Kafka cluster
*   An OpenTelemetry collector (e.g., Jaeger or Prometheus)

### Installation

```bash
go get github.com/Jdemon/ktel
```

### Usage

1.  **Define your message processor and application**:

    Create a `main.go` file. Implement the `processor.Processor` interface with your business logic. Then, use the `ktel` library to create and start your application.

    ```go
    package main

    import (
    	"context"
    	"fmt"
    	"log"

    	"github.com/Jdemon/ktel"
    	"github.com/Jdemon/ktel/processor"
    	"github.com/goccy/go-json"
    	"github.com/twmb/franz-go/pkg/kgo"
    	"go.opentelemetry.io/otel/attribute"
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

    	return app.Start(NewExampleProcessor(app.Logger))
    }

    // Message defines the structure of the incoming Kafka message
    type Message struct {
    	TransactionRef string `json:"transactionRef"`
    	Code           string `json:"code"`
    }

    // ExampleProcessor processes Kafka messages.
    type ExampleProcessor struct {
    	logger *zap.SugaredLogger
    }

    // NewExampleProcessor creates a new ExampleProcessor.
    func NewExampleProcessor(logger *zap.SugaredLogger) processor.Processor {
    	return &ExampleProcessor{
    		logger: logger,
    	}
    }

    // ProcessRecord processes a single Kafka record.
    func (p *ExampleProcessor) ProcessRecord(ctx context.Context, record *kgo.Record) error {
    	span := trace.SpanFromContext(ctx)

    	var msg Message
    	if err := json.Unmarshal(record.Value, &msg); err != nil {
    		p.logger.Errorw("Failed to unmarshal message", "error", err)
    		return err
    	}

    	span.SetAttributes(
    		attribute.String("transaction.ref", msg.TransactionRef),
    		attribute.String("result.code", msg.Code),
    	)

    	p.logger.Infow("Processed message successfully", "transactionRef", msg.TransactionRef)

    	return nil
    }
    ```

2.  **Configure your application**:

    `ktel` uses a `ktel-config.yaml` file for configuration. Here's an example:

    ```yaml
    kafka:
      brokers: "localhost:29092" # Can be overridden by KAFKA_BROKERS env var
      topic: "kafka.topic"
      group: "kafka-consumer-group"
      rebalanceStrategy: "roundrobin" # options: roundrobin, range, sticky
      tls:
        enabled: false
        caFile: ""   # Path to CA certificate file (e.g., /etc/ssl/certs/ca.pem)
        certFile: "" # Path to client certificate file
        keyFile: ""  # Path to client key file
      sasl:
        enabled: false
        mechanism: "SCRAM-SHA-512" # options: PLAIN, SCRAM-SHA-256, SCRAM-SHA-512
        username: ""
        password: ""
    server:
      port: "1323"
    otel:
      enabled: true # Set to true to enable OpenTelemetry
      exporter:
        grpc:
          endpoint: "localhost:4317" # Default gRPC port
    appName: "kafka-consumer" # The application name to include in every log message
    ```

## Contributing

Contributions are welcome! Please feel free to submit a pull request or open an issue.
