package observability

import (
	"context"
	"os"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
)

// Metrics holds all application metrics
type Metrics struct {
	RequestCount          metric.Int64Counter
	RequestDuration       metric.Float64Histogram
	DBQueryDuration       metric.Float64Histogram
	CacheHitCount         metric.Int64Counter
	CacheMissCount        metric.Int64Counter
	ActiveRequests        metric.Int64ObservableGauge
	SSEActiveConnections  metric.Int64ObservableGauge
	ZeroResultSearchCount metric.Int64Counter
}

// ObservabilityShutdown holds shutdown functions for all observability components
type ObservabilityShutdown struct {
	TracerProvider *sdktrace.TracerProvider
	MeterProvider  *sdkmetric.MeterProvider
	LoggerProvider *sdklog.LoggerProvider
}

// Setup initializes OpenTelemetry with traces, metrics, logs, and runtime metrics
func Setup(ctx context.Context, serviceName, serviceVersion, endpoint string) (func(context.Context) error, error) {
	// Initialize zerolog for structured logging
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	if os.Getenv("ENV") == "development" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})
	} else {
		log.Logger = zerolog.New(os.Stdout).With().Timestamp().Caller().Logger()
	}

	log.Info().
		Str("service", serviceName).
		Str("version", serviceVersion).
		Str("endpoint", endpoint).
		Msg("Initializing OpenTelemetry")

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(serviceVersion),
		),
	)
	if err != nil {
		return nil, err
	}

	// Set up trace exporter
	traceExporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create trace exporter")
		return nil, err
	}

	// Set up trace provider
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	log.Info().Msg("Trace provider initialized")

	// Set up metric exporter
	metricExporter, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpoint(endpoint),
		otlpmetricgrpc.WithInsecure(),
	)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create metric exporter")
		return nil, err
	}

	// Set up meter provider
	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter,
			sdkmetric.WithInterval(60*time.Second))),
		sdkmetric.WithResource(res),
	)
	otel.SetMeterProvider(meterProvider)
	log.Info().Msg("Meter provider initialized")

	// Set up log exporter
	logExporter, err := otlploggrpc.New(ctx,
		otlploggrpc.WithEndpoint(endpoint),
		otlploggrpc.WithInsecure(),
	)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to create log exporter, continuing without OTLP logs")
	}

	var loggerProvider *sdklog.LoggerProvider
	if logExporter != nil {
		loggerProvider = sdklog.NewLoggerProvider(
			sdklog.WithProcessor(sdklog.NewBatchProcessor(logExporter)),
			sdklog.WithResource(res),
		)
		log.Info().Msg("Logger provider initialized")
	}

	// Start runtime metrics collection
	err = runtime.Start(runtime.WithMinimumReadMemStatsInterval(time.Second))
	if err != nil {
		log.Warn().Err(err).Msg("Failed to start runtime metrics collection")
	} else {
		log.Info().Msg("Runtime metrics collection started")
	}

	// Shutdown function
	shutdown := func(ctx context.Context) error {
		log.Info().Msg("Shutting down OpenTelemetry")
		var errs []error

		if err := tracerProvider.Shutdown(ctx); err != nil {
			log.Error().Err(err).Msg("Error shutting down tracer provider")
			errs = append(errs, err)
		}

		if err := meterProvider.Shutdown(ctx); err != nil {
			log.Error().Err(err).Msg("Error shutting down meter provider")
			errs = append(errs, err)
		}

		if loggerProvider != nil {
			if err := loggerProvider.Shutdown(ctx); err != nil {
				log.Error().Err(err).Msg("Error shutting down logger provider")
				errs = append(errs, err)
			}
		}

		if len(errs) > 0 {
			return errs[0]
		}
		log.Info().Msg("OpenTelemetry shutdown complete")
		return nil
	}

	return shutdown, nil
}

var (
	activeRequests int64
)

// InitMetrics initializes application metrics
func InitMetrics() (*Metrics, error) {
	meter := otel.Meter("github.com/zatekoja/Patientpricediscoverydesign/backend")

	requestCount, err := meter.Int64Counter(
		"http.server.request.count",
		metric.WithDescription("Number of HTTP requests"),
	)
	if err != nil {
		return nil, err
	}

	requestDuration, err := meter.Float64Histogram(
		"http.server.request.duration",
		metric.WithDescription("HTTP request duration in milliseconds"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, err
	}

	activeRequestsGauge, err := meter.Int64ObservableGauge(
		"http.server.active_requests",
		metric.WithDescription("Number of active HTTP requests"),
	)
	if err != nil {
		return nil, err
	}

	_, err = meter.RegisterCallback(func(ctx context.Context, obs metric.Observer) error {
		obs.ObserveInt64(activeRequestsGauge, atomic.LoadInt64(&activeRequests))
		return nil
	}, activeRequestsGauge)
	if err != nil {
		return nil, err
	}

	dbQueryDuration, err := meter.Float64Histogram(
		"db.query.duration",
		metric.WithDescription("Database query duration in milliseconds"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, err
	}

	cacheHitCount, err := meter.Int64Counter(
		"cache.hit.count",
		metric.WithDescription("Number of cache hits"),
	)
	if err != nil {
		return nil, err
	}

	cacheMissCount, err := meter.Int64Counter(
		"cache.miss.count",
		metric.WithDescription("Number of cache misses"),
	)
	if err != nil {
		return nil, err
	}

	sseActiveConnections, err := meter.Int64ObservableGauge(
		"sse.active_connections",
		metric.WithDescription("Number of active SSE connections"),
	)
	if err != nil {
		return nil, err
	}

	zeroResultSearchCount, err := meter.Int64Counter(
		"search.zero_results.count",
		metric.WithDescription("Number of searches that returned zero results"),
	)
	if err != nil {
		return nil, err
	}

	return &Metrics{
		RequestCount:          requestCount,
		RequestDuration:       requestDuration,
		DBQueryDuration:       dbQueryDuration,
		CacheHitCount:         cacheHitCount,
		CacheMissCount:        cacheMissCount,
		ActiveRequests:        activeRequestsGauge,
		SSEActiveConnections:  sseActiveConnections,
		ZeroResultSearchCount: zeroResultSearchCount,
	}, nil
}

// RegisterSSECallback registers a callback for the SSE active connections metric
func (m *Metrics) RegisterSSECallback(callback func() int64) error {
	meter := otel.Meter("github.com/zatekoja/Patientpricediscoverydesign/backend")
	_, err := meter.RegisterCallback(func(ctx context.Context, obs metric.Observer) error {
		obs.ObserveInt64(m.SSEActiveConnections, callback())
		return nil
	}, m.SSEActiveConnections)
	return err
}

// RecordZeroResultSearch records a search that returned no results
func RecordZeroResultSearch(ctx context.Context, metrics *Metrics, intent string) {
	if metrics == nil || metrics.ZeroResultSearchCount == nil {
		return
	}
	attrs := []attribute.KeyValue{
		attribute.String("search.intent", intent),
	}
	metrics.ZeroResultSearchCount.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// IncrementActiveRequests increments the active request counter
func IncrementActiveRequests() {
	atomic.AddInt64(&activeRequests, 1)
}

// DecrementActiveRequests decrements the active request counter
func DecrementActiveRequests() {
	atomic.AddInt64(&activeRequests, -1)
}

// StartSpan starts a new trace span
func StartSpan(ctx context.Context, spanName string) (context.Context, trace.Span) {
	tracer := otel.Tracer("github.com/zatekoja/Patientpricediscoverydesign/backend")
	return tracer.Start(ctx, spanName)
}

// RecordError records an error in the current span
func RecordError(span trace.Span, err error) {
	if err != nil {
		span.RecordError(err)
	}
}

// SetSpanAttributes sets attributes on a span
func SetSpanAttributes(span trace.Span, attrs ...attribute.KeyValue) {
	span.SetAttributes(attrs...)
}

// RecordMetric records a metric with attributes
func RecordRequestMetric(ctx context.Context, metrics *Metrics, method, path string, statusCode int, duration time.Duration) {
	attrs := []attribute.KeyValue{
		attribute.String("http.method", method),
		attribute.String("http.route", path),
		attribute.Int("http.status_code", statusCode),
	}

	metrics.RequestCount.Add(ctx, 1, metric.WithAttributes(attrs...))
	metrics.RequestDuration.Record(ctx, float64(duration.Milliseconds()), metric.WithAttributes(attrs...))
}

// RecordDBMetric records a database operation metric
func RecordDBMetric(ctx context.Context, metrics *Metrics, operation string, duration time.Duration) {
	attrs := []attribute.KeyValue{
		attribute.String("db.operation", operation),
	}
	metrics.DBQueryDuration.Record(ctx, float64(duration.Milliseconds()), metric.WithAttributes(attrs...))
}

// RecordCacheHit records a cache hit
func RecordCacheHit(ctx context.Context, metrics *Metrics, key string) {
	attrs := []attribute.KeyValue{
		attribute.String("cache.key", key),
	}
	metrics.CacheHitCount.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordCacheMiss records a cache miss
func RecordCacheMiss(ctx context.Context, metrics *Metrics, key string) {
	attrs := []attribute.KeyValue{
		attribute.String("cache.key", key),
	}
	metrics.CacheMissCount.Add(ctx, 1, metric.WithAttributes(attrs...))
}
