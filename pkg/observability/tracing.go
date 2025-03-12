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

func InitTracer(serviceName string) (*trace.TracerProvider, error) {

	ctx := context.Background()

	// TODO: pass if cant connect
	client := otlptracehttp.NewClient(
		otlptracehttp.WithEndpoint("localhost:4318"), // TODO: pass to config
		otlptracehttp.WithInsecure(),                 // TODO: add dev flag to cfg
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
