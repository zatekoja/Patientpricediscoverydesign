/**
 * SSE (Server-Sent Events) Client Hook for React
 *
 * Provides real-time facility updates via SSE with automatic reconnection
 * and error handling.
 *
 * @example
 * ```tsx
 * // Facility-specific updates
 * const { data, isConnected, error } = useSSEFacility('fac_001');
 *
 * // Regional updates
 * const { data, isConnected, error } = useSSERegional({
 *   lat: 6.5244,
 *   lon: 3.3792,
 *   radius: 25
 * });
 * ```
 */

import { useState, useEffect, useCallback, useRef } from 'react';
import { API_BASE_URL } from '../api';

// Event types
export type FacilityEventType =
  | 'capacity_update'
  | 'ward_capacity_update'
  | 'wait_time_update'
  | 'urgent_care_update'
  | 'service_health_update'
  | 'service_availability_update'
  | 'heartbeat';

// Event structure
export interface FacilityEvent {
  id: string;
  facility_id: string;
  event_type: FacilityEventType;
  timestamp: string;
  location: {
    latitude: number;
    longitude: number;
  };
  changed_fields: Record<string, any>;
}

// Hook return type
interface UseSSEReturn {
  data: FacilityEvent | null;
  isConnected: boolean;
  error: Error | null;
  reconnect: () => void;
}

// Configuration
const MAX_RECONNECT_ATTEMPTS = 10;
const BASE_RECONNECT_DELAY = 1000; // 1 second
const MAX_RECONNECT_DELAY = 30000; // 30 seconds

const getSseBaseUrl = (): string => {
  const raw = (import.meta.env.VITE_SSE_BASE_URL || API_BASE_URL || '').trim();
  return raw.endsWith('/') ? raw.slice(0, -1) : raw;
};

const buildSseUrl = (path: string): string => {
  const base = getSseBaseUrl();
  if (!base) return path;
  return path.startsWith('/') ? `${base}${path}` : `${base}/${path}`;
};

/**
 * Hook for facility-specific SSE connection
 */
export function useSSEFacility(facilityId: string | null): UseSSEReturn {
  const [data, setData] = useState<FacilityEvent | null>(null);
  const [isConnected, setIsConnected] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const eventSourceRef = useRef<EventSource | null>(null);
  const reconnectAttemptsRef = useRef(0);
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const connect = useCallback(() => {
    if (!facilityId) return;

    try {
      const url = buildSseUrl(`/stream/facilities/${facilityId}`);
      const eventSource = new EventSource(url);

      eventSource.onopen = () => {
        console.log('[SSE] Connected to facility:', facilityId);
        setIsConnected(true);
        setError(null);
        reconnectAttemptsRef.current = 0;
      };

      eventSource.onerror = (e) => {
        console.error('[SSE] Connection error:', e);
        setIsConnected(false);
        eventSource.close();
      };

      // Listen for specific event types
      const eventTypes: FacilityEventType[] = [
        'capacity_update',
        'ward_capacity_update',
        'wait_time_update',
        'urgent_care_update',
        'service_health_update',
        'service_availability_update',
      ];

      eventTypes.forEach((eventType) => {
        eventSource.addEventListener(eventType, (event: MessageEvent) => {
          try {
            const data = JSON.parse(event.data) as FacilityEvent;
            console.log('[SSE] Received event:', eventType, data);
            setData(data);
          } catch (err) {
            console.error('[SSE] Failed to parse event data:', err);
          }
        });
      });

      // Handle heartbeat
      eventSource.addEventListener('heartbeat', () => {
        console.log('[SSE] Heartbeat received');
      });

      // Handle connection event
      eventSource.addEventListener('connected', (event: MessageEvent) => {
        try {
          const data = JSON.parse(event.data);
          console.log('[SSE] Connected:', data);
        } catch (err) {
          console.error('[SSE] Failed to parse connection data:', err);
        }
      });

      eventSourceRef.current = eventSource;
    } catch (err) {
      console.error('[SSE] Failed to create connection:', err);
      setError(err as Error);
    }
  }, [facilityId]);

  const disconnect = useCallback(() => {
    if (eventSourceRef.current) {
      console.log('[SSE] Disconnecting');
      eventSourceRef.current.close();
      eventSourceRef.current = null;
    }
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }
    setIsConnected(false);
  }, []);

  const reconnect = useCallback(() => {
    console.log('[SSE] Manual reconnect triggered');
    disconnect();
    reconnectAttemptsRef.current = 0;
    connect();
  }, [connect, disconnect]);

  useEffect(() => {
    if (facilityId) {
      connect();
    }

    return () => {
      disconnect();
    };
  }, [facilityId, connect, disconnect]);

  return { data, isConnected, error, reconnect };
}

/**
 * Hook for regional SSE connection
 */
export function useSSERegional(params: {
  lat: number;
  lon: number;
  radius?: number;
} | null): UseSSEReturn {
  const [data, setData] = useState<FacilityEvent | null>(null);
  const [isConnected, setIsConnected] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const eventSourceRef = useRef<EventSource | null>(null);
  const reconnectAttemptsRef = useRef(0);
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const connect = useCallback(() => {
    if (!params) return;

    try {
      const { lat, lon, radius = 50 } = params;
      const url = buildSseUrl(`/stream/facilities/region?lat=${lat}&lon=${lon}&radius=${radius}`);
      const eventSource = new EventSource(url);

      eventSource.onopen = () => {
        console.log('[SSE] Connected to region:', { lat, lon, radius });
        setIsConnected(true);
        setError(null);
        reconnectAttemptsRef.current = 0;
      };

      eventSource.onerror = (e) => {
        console.error('[SSE] Connection error:', e);
        setIsConnected(false);
        eventSource.close();

        reconnectAttemptsRef.current += 1;
        if (reconnectAttemptsRef.current <= MAX_RECONNECT_ATTEMPTS) {
          const delay = Math.min(
            BASE_RECONNECT_DELAY * Math.pow(2, reconnectAttemptsRef.current - 1),
            MAX_RECONNECT_DELAY
          );
          console.log(
            `[SSE] Reconnecting in ${delay}ms (attempt ${reconnectAttemptsRef.current})`
          );
          reconnectTimeoutRef.current = setTimeout(() => {
            connect();
          }, delay);
        } else {
          setError(new Error('Max reconnection attempts reached'));
        }
      };

      // Listen for specific event types
      const eventTypes: FacilityEventType[] = [
        'capacity_update',
        'ward_capacity_update',
        'wait_time_update',
        'urgent_care_update',
        'service_health_update',
        'service_availability_update',
      ];

      eventTypes.forEach((eventType) => {
        eventSource.addEventListener(eventType, (event: MessageEvent) => {
          try {
            const data = JSON.parse(event.data) as FacilityEvent;
            console.log('[SSE] Received regional event:', eventType, data);
            setData(data);
          } catch (err) {
            console.error('[SSE] Failed to parse event data:', err);
          }
        });
      });

      // Handle heartbeat
      eventSource.addEventListener('heartbeat', () => {
        console.log('[SSE] Heartbeat received');
      });

      // Handle connection event
      eventSource.addEventListener('connected', (event: MessageEvent) => {
        try {
          const data = JSON.parse(event.data);
          console.log('[SSE] Connected to region:', data);
        } catch (err) {
          console.error('[SSE] Failed to parse connection data:', err);
        }
      });

      eventSourceRef.current = eventSource;
    } catch (err) {
      console.error('[SSE] Failed to create connection:', err);
      setError(err as Error);
    }
  }, [params]);

  const disconnect = useCallback(() => {
    if (eventSourceRef.current) {
      console.log('[SSE] Disconnecting');
      eventSourceRef.current.close();
      eventSourceRef.current = null;
    }
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }
    setIsConnected(false);
  }, []);

  const reconnect = useCallback(() => {
    console.log('[SSE] Manual reconnect triggered');
    disconnect();
    reconnectAttemptsRef.current = 0;
    connect();
  }, [connect, disconnect]);

  useEffect(() => {
    if (params) {
      connect();
    }

    return () => {
      disconnect();
    };
  }, [params?.lat, params?.lon, params?.radius, connect, disconnect]);

  return { data, isConnected, error, reconnect };
}

/**
 * Example React Component using SSE
 */
export function FacilityDetailWithSSE({ facilityId }: { facilityId: string }) {
  const { data, isConnected, error, reconnect } = useSSEFacility(facilityId);

  return (
    <div>
      <div className="sse-status">
        {isConnected ? (
          <span className="text-green-600">● Connected</span>
        ) : (
          <span className="text-red-600">● Disconnected</span>
        )}
        {error && (
          <button onClick={reconnect} className="ml-2 text-blue-600">
            Reconnect
          </button>
        )}
      </div>

      {data && (
        <div className="real-time-update">
          <h3>Latest Update</h3>
          <p>Type: {data.event_type}</p>
          <p>Time: {new Date(data.timestamp).toLocaleString()}</p>
          <pre>{JSON.stringify(data.changed_fields, null, 2)}</pre>
        </div>
      )}

      {error && (
        <div className="error">
          <p>Error: {error.message}</p>
        </div>
      )}
    </div>
  );
}

/**
 * Example React Component for Map with Regional SSE
 */
export function FacilityMap({
  userLocation,
  radius = 25,
}: {
  userLocation: { lat: number; lon: number };
  radius?: number;
}) {
  const { data, isConnected } = useSSERegional({
    lat: userLocation.lat,
    lon: userLocation.lon,
    radius,
  });

  // Update map markers when events are received
  useEffect(() => {
    if (data) {
      console.log('Update map marker for facility:', data.facility_id);
      // Update your map marker here based on data.changed_fields
    }
  }, [data]);

  return (
    <div>
      <div className="connection-status">
        {isConnected ? '● Live Updates' : '● Loading...'}
      </div>
      {/* Your map component here */}
    </div>
  );
}
