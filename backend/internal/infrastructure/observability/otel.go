package observability

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

// Metrics holds all application metrics
type Metrics struct {
	RequestCount    metric.Int64Counter
	RequestDuration metric.Float64Histogram
	DBQueryDuration metric.Float64Histogram
	CacheHitCount   metric.Int64Counter
	CacheMissCount  metric.Int64Counter
}

// Setup initializes OpenTelemetry
func Setup(ctx context.Context, serviceName, serviceVersion, endpoint string) (func(context.Context) error, error) {
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
		return nil, err
	}

	// Set up trace provider
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Shutdown function
	shutdown := func(ctx context.Context) error {
		return tracerProvider.Shutdown(ctx)
	}

	return shutdown, nil
}

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

	return &Metrics{
		RequestCount:    requestCount,
		RequestDuration: requestDuration,
		DBQueryDuration: dbQueryDuration,
		CacheHitCount:   cacheHitCount,
		CacheMissCount:  cacheMissCount,
	}, nil
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
