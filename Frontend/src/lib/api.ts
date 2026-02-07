import { Facility, FacilityResponse, SearchParams } from '../types/api';

export const API_BASE_URL = 'http://localhost:8080/api';

export interface GeocodeResponse {
  address: string;
  lat: number;
  lon: number;
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

  async searchFacilities(params: SearchParams): Promise<FacilityResponse> {
    const queryParams = new URLSearchParams({
      lat: params.lat.toString(),
      lon: params.lon.toString(),
    });

    if (params.query) queryParams.append('query', params.query);
    if (params.radius) queryParams.append('radius', params.radius.toString());
    if (params.limit) queryParams.append('limit', params.limit.toString());
    if (params.offset) queryParams.append('offset', params.offset.toString());

    return this.request<FacilityResponse>(`/facilities/search?${queryParams.toString()}`);
  }

  async getFacility(id: string): Promise<Facility> {
    return this.request<Facility>(`/facilities/${id}`);
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
}

export const api = new ApiClient(API_BASE_URL);
