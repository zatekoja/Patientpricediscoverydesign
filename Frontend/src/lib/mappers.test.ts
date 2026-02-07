import { describe, expect, it } from 'vitest';
import { mapFacilitySearchResultToUI } from './mappers';
import type { FacilitySearchResult } from '../types/api';

describe('mapFacilitySearchResultToUI', () => {
  it('maps backend search results without mock data', () => {
    const input: FacilitySearchResult = {
      id: 'fac-1',
      name: 'Lagos General Hospital',
      facility_type: 'Hospital',
      address: {
        street: '1 Marina',
        city: 'Lagos',
        state: 'Lagos',
        zip_code: '',
        country: 'Nigeria',
      },
      location: { latitude: 6.5244, longitude: 3.3792 },
      phone_number: '0800-000-0000',
      website: 'https://lagos.example.com',
      rating: 4.7,
      review_count: 120,
      distance_km: 2.4,
      price: { min: 15000, max: 45000, currency: 'NGN' },
      services: ['MRI', 'CT Scan'],
      service_prices: [
        { procedure_id: 'proc-1', name: 'MRI', price: 15000, currency: 'NGN' },
        { procedure_id: 'proc-2', name: 'CT Scan', price: 20000, currency: 'NGN' },
      ],
      accepted_insurance: ['NHIS', 'AXA'],
      next_available_at: '2026-02-07T14:30:00Z',
      updated_at: '2026-02-07T12:30:00Z',
      capacity_status: 'Available',
      avg_wait_minutes: 15,
      urgent_care_available: true,
    };

    const result = mapFacilitySearchResultToUI(input);

    expect(result.id).toBe('fac-1');
    expect(result.name).toBe('Lagos General Hospital');
    expect(result.type).toBe('Hospital');
    expect(result.distanceKm).toBe(2.4);
    expect(result.priceMin).toBe(15000);
    expect(result.priceMax).toBe(45000);
    expect(result.currency).toBe('NGN');
    expect(result.services).toEqual(['MRI', 'CT Scan']);
    expect(result.servicePrices).toEqual([
      { name: 'MRI', price: 15000, currency: 'NGN' },
      { name: 'CT Scan', price: 20000, currency: 'NGN' },
    ]);
    expect(result.insurance).toEqual(['NHIS', 'AXA']);
    expect(result.nextAvailableAt).toBe('2026-02-07T14:30:00Z');
    expect(result.capacityStatus).toBe('Available');
    expect(result.avgWaitMinutes).toBe(15);
    expect(result.urgentCareAvailable).toBe(true);
  });
});
