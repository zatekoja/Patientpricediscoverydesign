import { FacilitySearchResult } from '../types/api';
import { calculateDistance } from './utils';

export type UIFacility = {
  id: string;
  name: string;
  type: string;
  distanceKm: number;
  priceMin?: number | null;
  priceMax?: number | null;
  currency?: string | null;
  rating: number;
  reviews: number;
  nextAvailableAt?: string | null;
  address: string;
  insurance: string[];
  services: string[];
  servicePrices: {
    procedureId?: string;
    name: string;
    displayName?: string;
    price: number;
    currency: string;
    description?: string;
    category?: string;
    code?: string;
    estimatedDuration?: number;
    normalizedTags?: string[];
    isAvailable?: boolean;
  }[];
  lat?: number;
  lon?: number;
  updatedAt?: string;
  phoneNumber?: string | null;
  whatsAppNumber?: string | null;
  email?: string | null;
  website?: string | null;
  capacityStatus?: string | null;
  avgWaitMinutes?: number | null;
  urgentCareAvailable?: boolean | null;
  wardStatuses?: Record<string, {
    status: string;
    count: number;
    trend: string;
    lastUpdated: string | Date;
  }>;
  wards?: {
    wardName: string;
    wardType?: string;
    capacityStatus?: string;
    avgWaitMinutes?: number;
    urgentCareAvailable?: boolean;
    lastUpdated: string;
  }[];
};

const buildAddress = (facility: FacilitySearchResult): string => {
  const parts = [
    facility.address?.street,
    facility.address?.city,
    facility.address?.state,
  ].filter(Boolean);
  return parts.length > 0 ? parts.join(', ') : 'Address not available';
};

export const mapFacilitySearchResultToUI = (
  facility: FacilitySearchResult,
  center?: { lat: number; lon: number }
): UIFacility => {
  const distanceKm =
    typeof facility.distance_km === 'number'
      ? facility.distance_km
      : center
        ? calculateDistance(
            center.lat,
            center.lon,
            facility.location.latitude,
            facility.location.longitude
          )
        : 0;

  return {
    id: facility.id,
    name: facility.name,
    type: facility.facility_type || 'Health Facility',
    distanceKm,
    priceMin: facility.price?.min ?? null,
    priceMax: facility.price?.max ?? null,
    currency: facility.price?.currency ?? null,
    rating: facility.rating ?? 0,
    reviews: facility.review_count ?? 0,
    nextAvailableAt: facility.next_available_at ?? null,
    address: buildAddress(facility),
    insurance: facility.accepted_insurance ?? [],
    services: facility.services ?? [],
    servicePrices: (facility.service_prices ?? []).map((item) => ({
      procedureId: item.procedure_id,
      name: item.name,
      displayName: item.display_name,
      price: item.price,
      currency: item.currency,
      description: item.description,
      category: item.category,
      code: item.code,
      estimatedDuration: item.estimated_duration,
      normalizedTags: item.normalized_tags,
      isAvailable: item.is_available ?? true,
    })),
    lat: facility.location?.latitude,
    lon: facility.location?.longitude,
    updatedAt: facility.updated_at,
    phoneNumber: facility.phone_number ?? null,
    whatsAppNumber: facility.whatsapp_number ?? null,
    email: facility.email ?? null,
    website: facility.website ?? null,
    capacityStatus: facility.capacity_status ?? null,
    avgWaitMinutes: facility.avg_wait_minutes ?? null,
    urgentCareAvailable: facility.urgent_care_available ?? null,
    wardStatuses: facility.ward_statuses as any ?? undefined,
    wards: facility.wards?.map((ward) => ({
      wardName: ward.ward_name,
      wardType: ward.ward_type,
      capacityStatus: ward.capacity_status,
      avgWaitMinutes: ward.avg_wait_minutes,
      urgentCareAvailable: ward.urgent_care_available,
      lastUpdated: ward.last_updated,
    })),
  };
};
