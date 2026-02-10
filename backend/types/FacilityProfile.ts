export interface FacilityProfile {
  id: string;
  name: string;
  facilityType?: string;
  description?: string;
  tags?: string[];
  capacityStatus?: string; // Legacy/High-level status
  wardStatuses?: Record<string, {
    status: string;
    count: number;
    trend: string;
    estimatedWaitMinutes?: number;
    lastUpdated: Date;
  }>;
  avgWaitMinutes?: number;
  urgentCareAvailable?: boolean;
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
  };
}
