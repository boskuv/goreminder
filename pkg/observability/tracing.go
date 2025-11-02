package observability

import (
	"context"
	"fmt"
	"log"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

func InitTracer(serviceName string, endpoint string, insecure bool) (*trace.TracerProvider, error) {
	ctx := context.Background()

	clientOptions := []otlptracehttp.Option{}

	// Normalize endpoint - WithEndpoint expects host:port format (without protocol)
	// It will add /v1/traces path automatically
	if endpoint != "" {
		endpoint = strings.TrimSpace(endpoint)
		// Remove http:// or https:// prefix if present
		endpoint = strings.TrimPrefix(endpoint, "http://")
		endpoint = strings.TrimPrefix(endpoint, "https://")
		// Remove trailing slash if present
		endpoint = strings.TrimSuffix(endpoint, "/")
		// Remove /v1/traces path if user included it (will be added automatically)
		endpoint = strings.TrimSuffix(endpoint, "/v1/traces")
		endpoint = strings.TrimSuffix(endpoint, "/v1")

		clientOptions = append(clientOptions, otlptracehttp.WithEndpoint(endpoint))
		log.Printf("Initializing OpenTelemetry tracer with endpoint: %s", endpoint)
	} else {
		// Default endpoint (host:port format)
		endpoint = "localhost:4318"
		clientOptions = append(clientOptions, otlptracehttp.WithEndpoint(endpoint))
		log.Printf("Using default OpenTelemetry endpoint: %s", endpoint)
	}

	if insecure {
		clientOptions = append(clientOptions, otlptracehttp.WithInsecure())
	}

	client := otlptracehttp.NewClient(
		clientOptions...,
	)

	exporter, err := otlptrace.New(ctx, client)
	if err != nil {
		log.Printf("WARNING: failed to create OpenTelemetry exporter: %v. Tracing will be disabled.", err)
		return nil, fmt.Errorf("failed to create OpenTelemetry exporter: %w", err)
	}

	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(serviceName),
		)),
	)

	otel.SetTracerProvider(tp)
	log.Printf("OpenTelemetry tracer initialized successfully for service: %s", serviceName)

	// Note: Export errors are handled silently by the batcher by default.
	// If you need to see export errors, you can use a custom error handler
	// or check Jaeger UI to see if traces are being exported.

	return tp, nil
}
