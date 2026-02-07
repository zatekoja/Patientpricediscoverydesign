import {
  Facility,
  FacilityResponse,
  FacilitySearchResponse,
  FacilitySuggestionResponse,
  FeedbackRequest,
  FeedbackResponse,
  SearchParams,
} from '../types/api';

type ApiEnv = {
  VITE_API_BASE_URL?: string;
};

export const resolveApiBaseUrl = (env?: ApiEnv): string => {
  const envBaseUrl = env?.VITE_API_BASE_URL?.trim();
  if (envBaseUrl) return envBaseUrl;
  return '/api';
};

const metaEnv = import.meta.env;
export const API_BASE_URL = resolveApiBaseUrl(metaEnv);

export interface GeocodeResponse {
  address: string;
  lat: number;
  lon: number;
}

export interface Coordinates {
  Latitude: number;
  Longitude: number;
}

export interface GeocodedAddress {
  FormattedAddress: string;
  Street: string;
  City: string;
  State: string;
  ZipCode: string;
  Country: string;
  Coordinates: Coordinates;
}

class ApiClient {
  private baseUrl: string;

  constructor(baseUrl: string) {
    this.baseUrl = baseUrl;
  }

  private async request<T>(endpoint: string, options?: RequestInit): Promise<T> {
    const response = await fetch(`${this.baseUrl}${endpoint}`, options);

    if (!response.ok) {
      throw new Error(`API request failed: ${response.statusText}`);
    }

    return response.json();
  }

  async getFacilities(type?: string): Promise<FacilityResponse> {
    const query = type ? `?type=${encodeURIComponent(type)}` : '';
    return this.request<FacilityResponse>(`/facilities${query}`);
  }

  async searchFacilities(params: SearchParams): Promise<FacilitySearchResponse> {
    const queryParams = new URLSearchParams({
      lat: params.lat.toString(),
      lon: params.lon.toString(),
    });

    if (params.query) queryParams.append('query', params.query);
    if (params.radius) queryParams.append('radius', params.radius.toString());
    if (params.limit) queryParams.append('limit', params.limit.toString());
    if (params.offset) queryParams.append('offset', params.offset.toString());
    if (params.insurance_provider) queryParams.append('insurance_provider', params.insurance_provider);
    if (params.min_price != null) queryParams.append('min_price', params.min_price.toString());
    if (params.max_price != null) queryParams.append('max_price', params.max_price.toString());

    return this.request<FacilitySearchResponse>(`/facilities/search?${queryParams.toString()}`);
  }

  async getFacility(id: string): Promise<Facility> {
    return this.request<Facility>(`/facilities/${id}`);
  }

  async suggestFacilities(
    params: { query: string; lat: number; lon: number; limit?: number },
    signal?: AbortSignal
  ): Promise<FacilitySuggestionResponse> {
    const queryParams = new URLSearchParams({
      query: params.query,
      lat: params.lat.toString(),
      lon: params.lon.toString(),
    });

    if (params.limit) queryParams.append('limit', params.limit.toString());

    return this.request<FacilitySuggestionResponse>(
      `/facilities/suggest?${queryParams.toString()}`,
      { signal }
    );
  }

  async getProcedures(category?: string): Promise<{ procedures: any[]; count: number }> {
    const query = category ? `?category=${encodeURIComponent(category)}` : '';
    return this.request(`/procedures${query}`);
  }

  async getInsuranceProviders(): Promise<{ providers: any[]; count: number }> {
    return this.request('/insurance-providers');
  }

  async geocode(address: string): Promise<GeocodeResponse> {
    const query = `?address=${encodeURIComponent(address)}`;
    return this.request(`/geocode${query}`);
  }

  async reverseGeocode(lat: number, lon: number): Promise<GeocodedAddress> {
    const query = `?lat=${lat}&lon=${lon}`;
    return this.request(`/reverse-geocode${query}`);
  }

  async getAvailability(facilityId: string, from: Date, to: Date): Promise<{ slots: any[] }> {
    const query = new URLSearchParams({
      from: from.toISOString(),
      to: to.toISOString(),
    });
    return this.request(`/facilities/${facilityId}/availability?${query.toString()}`);
  }

  async bookAppointment(data: any): Promise<any> {
    return this.request('/appointments', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(data),
    });
  }

  async submitFeedback(payload: FeedbackRequest): Promise<FeedbackResponse> {
    return this.request('/feedback', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(payload),
    });
  }
}

export const api = new ApiClient(API_BASE_URL);
