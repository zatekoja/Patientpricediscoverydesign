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
  display_name?: string;
  price: number;
  currency: string;
  description?: string;
  category?: string;
  code?: string;
  estimated_duration?: number;
  normalized_tags?: string[];
}

// TDD-driven service search types
export interface ServiceSearchParams {
  limit: number;
  offset: number;
  search?: string;
  category?: string;
  minPrice?: number;
  maxPrice?: number;
  available?: boolean;
  sort?: 'price' | 'name' | 'category' | 'updated_at';
  order?: 'asc' | 'desc';
}

export interface FacilityService {
  id: string;
  facility_id: string;
  procedure_id: string;
  name: string;
  display_name?: string;
  category: string;
  price: number;
  currency: string;
  description?: string;
  code?: string;
  estimated_duration: number;
  normalized_tags?: string[];
  is_available: boolean;
  created_at: string;
  updated_at: string;
}

export interface ServiceSearchResponse {
  services: FacilityService[];
  total_count: number;
  current_page: number;
  total_pages: number;
  page_size: number;
  has_next: boolean;
  has_prev: boolean;
  filters_applied: {
    search?: string;
    category?: string;
    min_price?: number;
    max_price?: number;
    available?: boolean;
    sort_by?: string;
    sort_order?: string;
  };
}

export interface FacilitySearchResult {
  id: string;
  name: string;
  facility_type: string;
  address: Address;
  location: Location;
  phone_number?: string;
  whatsapp_number?: string;
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
  ward_statuses?: Record<string, {
    status: string;
    count: number;
    trend: string;
    last_updated: string;
  }>;
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
  tags?: string[];
  matched_tag?: string;
}

export interface FacilitySuggestionResponse {
  suggestions: FacilitySuggestion[];
  count: number;
}

export interface ProcedureEnrichment {
  id: string;
  procedure_id: string;
  description?: string;
  prep_steps?: string[];
  risks?: string[];
  recovery?: string[];
  provider?: string;
  model?: string;
  created_at?: string;
  updated_at?: string;
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

export interface ProviderHealthResponse {
  healthy: boolean;
  lastSync?: string;
  message?: string;
}

export interface ProviderInfo {
  id: string;
  name: string;
  type: string;
  healthy: boolean;
  lastSync?: string;
}

export interface ProviderListResponse {
  providers: ProviderInfo[];
}

export interface ServiceFeeItem {
  id: string;
  name: string;
  price: number;
  currency: string;
  code: string;
}

export interface ServiceFeeSummary {
  fees: ServiceFeeItem[];
  total: number;
  currency: string;
}

export interface FeeWaiverInfo {
  has_waiver: boolean;
  sponsor_name?: string;
  waiver_type?: string;
  waiver_amount?: number;
}
