// Package logger has helpers to setup a zerolog.Logger
//
//	https://github.com/rs/zerolog
package logger

import (
	"context"
	"io"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
	"go.opentelemetry.io/otel/trace"
)

// RequestIDContextKey is the context.Value key for the HTTP request ID.
// Must match middleware.RequestIDKey.
const RequestIDContextKey = "request_id"

// New is a convenience function to initialize a zerolog.Logger
// with an initial minimum accepted level and timestamp (if true)
// for a given io.Writer.
func New(w io.Writer, lvl zerolog.Level, withTimestamp bool) zerolog.Logger {
	// Set global time format to RFC3339
	zerolog.TimeFieldFormat = time.RFC3339

	// logger is initialized with the writer and level passed in.
	// All logs will be written at the given level (unless raised
	// using zerolog.SetGlobalLevel)
	lgr := zerolog.New(w).Level(lvl)
	if withTimestamp {
		lgr = lgr.With().Timestamp().Logger()
	}

	return lgr
}

// LogErrorStackViaPkgErrors is a convenience function to set the zerolog
// ErrorStackMarshaler global variable.
// If true, writes error stacks for logs using "github.com/pkg/errors".
// If false, will use the internal errs.Op stack instead of "github.com/pkg/errors".
func LogErrorStackViaPkgErrors(p bool) {
	if !p {
		zerolog.ErrorStackMarshaler = nil
		return
	}
	// set ErrorStackMarshaler to pkgerrors.MarshalStack
	// to enable error stack traces
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
}

// WithTraceContext adds request_id / trace_id / span_id from ctx to the logger
// so logs correlate with access logs and Jaeger traces.
func WithTraceContext(ctx context.Context, log zerolog.Logger) zerolog.Logger {
	if ctx == nil {
		return log
	}

	ctxLog := log.With()

	if requestID, ok := ctx.Value(RequestIDContextKey).(string); ok && requestID != "" {
		ctxLog = ctxLog.Str("request_id", requestID)
	}

	spanContext := trace.SpanContextFromContext(ctx)
	if spanContext.IsValid() {
		ctxLog = ctxLog.
			Str("trace_id", spanContext.TraceID().String()).
			Str("span_id", spanContext.SpanID().String())
	}

	return ctxLog.Logger()
}
