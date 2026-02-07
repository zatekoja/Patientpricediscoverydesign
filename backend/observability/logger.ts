import pino from 'pino';
import { trace } from '@opentelemetry/api';

const isDevelopment = process.env.NODE_ENV === 'development';

export const logger = pino({
  level: process.env.LOG_LEVEL || 'info',
  formatters: {
    level: (label) => {
      return { level: label };
    },
  },
  ...(isDevelopment
    ? {
        transport: {
          target: 'pino-pretty',
          options: {
            colorize: true,
            translateTime: 'HH:MM:ss',
            ignore: 'pid,hostname',
          },
        },
      }
    : {}),
});

/**
 * Get a logger with trace context
 */
export function getLoggerWithTrace() {
  const span = trace.getActiveSpan();
  if (span) {
    const spanContext = span.spanContext();
    return logger.child({
      trace_id: spanContext.traceId,
      span_id: spanContext.spanId,
    });
  }
  return logger;
}

export default logger;

