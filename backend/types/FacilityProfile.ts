export interface WardCapacity {
  wardName: string;
  wardType?: string; // 'maternity', 'pharmacy', 'inpatient', 'emergency', 'surgery', 'icu', 'pediatrics', 'radiology', 'laboratory', or custom
  capacityStatus?: string;
  avgWaitMinutes?: number;
  urgentCareAvailable?: boolean;
  lastUpdated: Date;
}

export interface FacilityProfile {
  id: string;
  name: string;
  facilityType?: string;
  description?: string;
  tags?: string[];
  // Facility-wide capacity (legacy/fallback)
  capacityStatus?: string;
  avgWaitMinutes?: number;
  urgentCareAvailable?: boolean;
  // Ward-specific capacity (new)
  wards?: WardCapacity[];
  address?: {
    street?: string;
    city?: string;
    state?: string;
    zipCode?: string;
    country?: string;
  };
  location?: {
    latitude?: number;
    longitude?: number;
  };
  phoneNumber?: string;
  email?: string;
  website?: string;
  lastUpdated: Date;
  source: string;
  metadata?: {
    curatedSources?: string[];
    matchedRules?: string[];
    sampleProcedures?: string[];
    llm?: {
      model?: string;
      generatedAt?: Date;
    };
    capacityTokenTTLMinutes?: number; // Per-facility token TTL override
    availableWards?: string[]; // List of available wards for this facility (e.g., ['maternity', 'pharmacy', 'inpatient'])
  };
}
