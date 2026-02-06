import { NodeSDK } from '@opentelemetry/sdk-node';
import { getNodeAutoInstrumentations } from '@opentelemetry/auto-instrumentations-node';
import { Resource } from '@opentelemetry/resources';
import { SemanticResourceAttributes } from '@opentelemetry/semantic-conventions';
import { OTLPTraceExporter } from '@opentelemetry/exporter-trace-otlp-http';
import { OTLPMetricExporter } from '@opentelemetry/exporter-metrics-otlp-http';
import { PeriodicExportingMetricReader } from '@opentelemetry/sdk-metrics';

const globalKey = Symbol.for('ppd.otel.initialized');
const globalState = globalThis as unknown as { [key: symbol]: boolean };

const otelEnabled =
  process.env.OTEL_ENABLED === 'true' || process.env.OTEL_ENABLED === '1';

if (otelEnabled && !globalState[globalKey]) {
  globalState[globalKey] = true;

  const serviceName =
    process.env.OTEL_SERVICE_NAME || 'patient-price-discovery-provider';
  const serviceVersion =
    process.env.OTEL_SERVICE_VERSION || '1.0.0';

  const baseEndpoint =
    process.env.OTEL_EXPORTER_OTLP_ENDPOINT || process.env.OTEL_ENDPOINT || '';
  const sanitizedBase = baseEndpoint.replace(/\/$/, '');
  const tracesEndpoint =
    process.env.OTEL_EXPORTER_OTLP_TRACES_ENDPOINT ||
    (sanitizedBase ? `${sanitizedBase}/v1/traces` : undefined);
  const metricsEndpoint =
    process.env.OTEL_EXPORTER_OTLP_METRICS_ENDPOINT ||
    (sanitizedBase ? `${sanitizedBase}/v1/metrics` : undefined);

  const resource = new Resource({
    [SemanticResourceAttributes.SERVICE_NAME]: serviceName,
    [SemanticResourceAttributes.SERVICE_VERSION]: serviceVersion,
  });

  const traceExporter = tracesEndpoint
    ? new OTLPTraceExporter({ url: tracesEndpoint })
    : undefined;
  const metricExporter = metricsEndpoint
    ? new OTLPMetricExporter({ url: metricsEndpoint })
    : undefined;
  const metricReader = metricExporter
    ? new PeriodicExportingMetricReader({
        exporter: metricExporter,
        exportIntervalMillis: 60000,
      })
    : undefined;

  const sdk = new NodeSDK({
    resource,
    traceExporter,
    metricReader,
    instrumentations: [
      getNodeAutoInstrumentations({
        '@opentelemetry/instrumentation-fs': { enabled: false },
      }),
    ],
  });

  sdk
    .start()
    .then(() => {
      console.log('[otel] tracing/metrics initialized');
    })
    .catch((error) => {
      console.error('[otel] failed to start', error);
    });

  const shutdown = () => {
    sdk
      .shutdown()
      .then(() => console.log('[otel] shutdown complete'))
      .catch((error) => console.error('[otel] shutdown error', error));
  };

  process.on('SIGTERM', shutdown);
  process.on('SIGINT', shutdown);
}
