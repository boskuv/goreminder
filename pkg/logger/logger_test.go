package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/trace"
)

func TestWithTraceContext_AddsRequestAndTraceIDs(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	base := zerolog.New(&buf)

	traceID, err := trace.TraceIDFromHex("537c0afe7701df904b5ce1a1d5c0565a")
	if err != nil {
		t.Fatal(err)
	}
	spanID, err := trace.SpanIDFromHex("00592c7c36ccb437")
	if err != nil {
		t.Fatal(err)
	}
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})

	ctx := context.WithValue(context.Background(), RequestIDContextKey, "req-123")
	ctx = trace.ContextWithSpanContext(ctx, sc)

	log := WithTraceContext(ctx, base)
	log.Info().Msg("hello")

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("unmarshal log: %v body=%q", err, buf.String())
	}
	if entry["request_id"] != "req-123" {
		t.Fatalf("request_id=%v", entry["request_id"])
	}
	if entry["trace_id"] != "537c0afe7701df904b5ce1a1d5c0565a" {
		t.Fatalf("trace_id=%v", entry["trace_id"])
	}
	if entry["span_id"] != "00592c7c36ccb437" {
		t.Fatalf("span_id=%v", entry["span_id"])
	}
}
