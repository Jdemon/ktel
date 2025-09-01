# Release Notes

## Version 0.1.1

This release introduces two new features that improve the observability and configurability of the `ktel` library.

### Features

*   **Automatic `traceparent` header injection**: The Kafka producer now automatically injects the `traceparent` header into every message, ensuring that trace context is propagated to downstream services.
*   **Configurable OpenTelemetry semantic convention version**: The OpenTelemetry semantic convention (semconv) version can now be configured in the `ktel-config.yaml` file, allowing you to change the version without recompiling the application.

## Version 0.1.0

This is the initial release of the `ktel` library, a Kafka consumer framework for Go that simplifies the development of observable, production-ready services.

### Features

*   **Kafka Consumer**: A robust and resilient Kafka consumer built on top of `franz-go`.
*   **Kafka Producer**: A producer to publish messages to other Kafka topics.
*   **Configuration**: Simplified configuration loading from a YAML file.
*   **Observability**: Deep integration with OpenTelemetry for distributed tracing and metrics.
*   **Health Checks**: A configurable health check endpoint to monitor the health of the service.
*   **Structured Logging**: Structured logging for better readability and filterability of logs.
*   **Graceful Shutdown**: Coordinated and graceful shutdown of all components.
*   **Example**: A working example to demonstrate the library's usage.