import { API_BASE_URL } from './api';

export type ConnectionStatus = 'connecting' | 'connected' | 'disconnected' | 'error';

export interface FacilityUpdate {
  id: string;
  facility_id: string;
  event_type:
    | 'capacity_update'
    | 'wait_time_update'
    | 'urgent_care_update'
    | 'service_health_update'
    | 'service_availability_update';
  timestamp: string;
  location: {
    latitude: number;
    longitude: number;
  };
  changed_fields: Record<string, any>;
}

export interface SSEConnectionInfo {
  facility_id?: string;
  lat?: number;
  lon?: number;
  radius_km?: number;
  timestamp: string;
}

export type EventCallback = (event: FacilityUpdate) => void;
export type StatusCallback = (status: ConnectionStatus) => void;

/**
 * FacilitySSEClient manages Server-Sent Events connections for real-time facility updates
 */
export class FacilitySSEClient {
  private eventSource: EventSource | null = null;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 10;
  private reconnectTimeoutId: number | null = null;
  private status: ConnectionStatus = 'disconnected';
  private eventCallbacks: Set<EventCallback> = new Set();
  private statusCallbacks: Set<StatusCallback> = new Set();

  constructor(
    private readonly endpoint: string,
    private readonly autoReconnect: boolean = true
  ) {}

  /**
   * Connect to the SSE stream
   */
  connect(): void {
    if (this.eventSource) {
      this.disconnect();
    }

    this.setStatus('connecting');

    try {
      const sseBaseUrl = (import.meta.env.VITE_SSE_BASE_URL || API_BASE_URL).trim();
      this.eventSource = new EventSource(`${sseBaseUrl}${this.endpoint}`);

      this.eventSource.addEventListener('open', () => {
        this.reconnectAttempts = 0;
        this.setStatus('connected');
        console.log(`SSE connected: ${this.endpoint}`);
      });

      this.eventSource.addEventListener('error', (error) => {
        console.error(`SSE error: ${this.endpoint}`, error);
        this.setStatus('error');
        this.handleReconnect();
      });

      // Listen for connection confirmation
      this.eventSource.addEventListener('connected', (event) => {
        const data = JSON.parse((event as MessageEvent).data) as SSEConnectionInfo;
        console.log('SSE connection confirmed:', data);
      });

      // Listen for heartbeat
      this.eventSource.addEventListener('heartbeat', () => {
        // Keep connection alive
      });

      // Listen for capacity updates
      this.eventSource.addEventListener('capacity_update', (event) => {
        this.handleEvent((event as MessageEvent).data);
      });

      // Listen for wait time updates
      this.eventSource.addEventListener('wait_time_update', (event) => {
        this.handleEvent((event as MessageEvent).data);
      });

      // Listen for urgent care updates
      this.eventSource.addEventListener('urgent_care_update', (event) => {
        this.handleEvent((event as MessageEvent).data);
      });

      // Listen for service health updates
      this.eventSource.addEventListener('service_health_update', (event) => {
        this.handleEvent((event as MessageEvent).data);
      });

      // Listen for service availability updates
      this.eventSource.addEventListener('service_availability_update', (event) => {
        this.handleEvent((event as MessageEvent).data);
      });

    } catch (error) {
      console.error('Failed to create SSE connection:', error);
      this.setStatus('error');
      this.handleReconnect();
    }
  }

  /**
   * Disconnect from the SSE stream
   */
  disconnect(): void {
    if (this.reconnectTimeoutId !== null) {
      clearTimeout(this.reconnectTimeoutId);
      this.reconnectTimeoutId = null;
    }

    if (this.eventSource) {
      this.eventSource.close();
      this.eventSource = null;
    }

    this.setStatus('disconnected');
    this.reconnectAttempts = 0;
  }

  /**
   * Subscribe to facility update events
   */
  onUpdate(callback: EventCallback): () => void {
    this.eventCallbacks.add(callback);
    return () => this.eventCallbacks.delete(callback);
  }

  /**
   * Subscribe to connection status changes
   */
  onStatusChange(callback: StatusCallback): () => void {
    this.statusCallbacks.add(callback);
    // Immediately call with current status
    callback(this.status);
    return () => this.statusCallbacks.delete(callback);
  }

  /**
   * Get current connection status
   */
  getStatus(): ConnectionStatus {
    return this.status;
  }

  /**
   * Check if currently connected
   */
  isConnected(): boolean {
    return this.status === 'connected';
  }

  private handleEvent(data: string): void {
    try {
      const update: FacilityUpdate = JSON.parse(data);
      this.eventCallbacks.forEach(callback => {
        try {
          callback(update);
        } catch (error) {
          console.error('Error in event callback:', error);
        }
      });
    } catch (error) {
      console.error('Failed to parse SSE event:', error);
    }
  }

  private setStatus(status: ConnectionStatus): void {
    if (this.status !== status) {
      this.status = status;
      this.statusCallbacks.forEach(callback => {
        try {
          callback(status);
        } catch (error) {
          console.error('Error in status callback:', error);
        }
      });
    }
  }

  private handleReconnect(): void {
    if (!this.autoReconnect) {
      return;
    }

    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error(`Max reconnection attempts (${this.maxReconnectAttempts}) reached`);
      this.setStatus('disconnected');
      return;
    }

    // Exponential backoff: 1s, 2s, 4s, 8s, 16s, 30s (max)
    const delay = Math.min(1000 * Math.pow(2, this.reconnectAttempts), 30000);
    this.reconnectAttempts++;

    console.log(`Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts}/${this.maxReconnectAttempts})`);

    this.reconnectTimeoutId = window.setTimeout(() => {
      this.connect();
    }, delay);
  }
}

/**
 * Create a client for facility-specific updates
 */
export function createFacilitySSEClient(facilityId: string): FacilitySSEClient {
  return new FacilitySSEClient(`/stream/facilities/${facilityId}`);
}

/**
 * Create a client for regional facility updates
 */
export function createRegionalSSEClient(lat: number, lon: number, radiusKm: number = 50): FacilitySSEClient {
  return new FacilitySSEClient(`/stream/facilities/region?lat=${lat}&lon=${lon}&radius=${radiusKm}`);
}
