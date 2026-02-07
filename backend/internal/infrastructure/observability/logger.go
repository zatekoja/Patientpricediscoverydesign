package observability

import (
	"context"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/trace"
)

// InitLogger initializes the global zerolog logger
func InitLogger(serviceName, env string) {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	if env == "development" {
		log.Logger = log.Output(zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}).With().
			Str("service", serviceName).
			Logger()
	} else {
		log.Logger = zerolog.New(os.Stdout).
			With().
			Timestamp().
			Caller().
			Str("service", serviceName).
			Logger()
	}
}

// LoggerFromContext returns a logger with trace context
func LoggerFromContext(ctx context.Context) *zerolog.Logger {
	logger := log.With().Logger()

	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		logger = logger.With().
			Str("trace_id", span.SpanContext().TraceID().String()).
			Str("span_id", span.SpanContext().SpanID().String()).
			Logger()
	}

	return &logger
}

// GetLogger returns the global logger
func GetLogger() *zerolog.Logger {
	return &log.Logger
}
