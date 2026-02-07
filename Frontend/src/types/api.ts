export interface Address {
  street: string;
  city: string;
  state: string;
  zip_code: string;
  country: string;
}

export interface Location {
  latitude: number;
  longitude: number;
}

export interface Facility {
  id: string;
  name: string;
  address: Address;
  location: Location;
  phone_number: string;
  email: string;
  website: string;
  description: string;
  facility_type: string;
  accepted_insurance: string[];
  rating: number;
  review_count: number;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface FacilityResponse {
  facilities: Facility[];
  count: number;
}

export interface SearchParams {
  query?: string;
  lat: number;
  lon: number;
  radius?: number;
  limit?: number;
  offset?: number;
}
