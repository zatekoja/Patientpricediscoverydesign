export interface FacilityProfile {
  id: string;
  name: string;
  facilityType?: string;
  description?: string;
  tags?: string[];
  capacityStatus?: string;
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
