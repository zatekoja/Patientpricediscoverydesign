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

export interface FacilityPriceRange {
  min: number;
  max: number;
  currency: string;
}

export interface ServicePrice {
  procedure_id: string;
  name: string;
  price: number;
  currency: string;
}

export interface FacilitySearchResult {
  id: string;
  name: string;
  facility_type: string;
  address: Address;
  location: Location;
  phone_number?: string;
  email?: string;
  website?: string;
  rating: number;
  review_count: number;
  distance_km: number;
  price?: FacilityPriceRange;
  services: string[];
  service_prices?: ServicePrice[];
  accepted_insurance: string[];
  next_available_at?: string;
  avg_wait_minutes?: number;
  capacity_status?: string;
  urgent_care_available?: boolean;
  updated_at: string;
}

export interface FacilitySearchResponse {
  facilities: FacilitySearchResult[];
  count: number;
}

export interface FacilitySuggestion {
  id: string;
  name: string;
  facility_type: string;
  address: Address;
  location: Location;
  rating: number;
  price?: FacilityPriceRange;
  service_prices?: ServicePrice[];
  matched_service_price?: ServicePrice;
}

export interface FacilitySuggestionResponse {
  suggestions: FacilitySuggestion[];
  count: number;
}

export interface FeedbackRequest {
  rating: number;
  message?: string;
  email?: string;
  page?: string;
}

export interface FeedbackResponse {
  status: string;
  id: string;
}

export interface SearchParams {
  query?: string;
  lat: number;
  lon: number;
  radius?: number;
  limit?: number;
  offset?: number;
  insurance_provider?: string;
  min_price?: number;
  max_price?: number;
}
