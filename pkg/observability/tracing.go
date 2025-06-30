package observability

import (
	"context"
	"log"

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

	// TODO: add default endpoint if not provided
	if endpoint != "" {
		clientOptions = append(clientOptions, otlptracehttp.WithEndpoint(endpoint))
	}

	if insecure {
		clientOptions = append(clientOptions, otlptracehttp.WithInsecure())
	}

	client := otlptracehttp.NewClient(
		clientOptions...,
	)

	exporter, err := otlptrace.New(ctx, client)
	if err != nil {
		log.Fatalf("failed to create OpenTelemetry exporter: %v", err)
		return nil, err
	}

	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(serviceName),
		)),
	)

	otel.SetTracerProvider(tp)
	return tp, nil
}
