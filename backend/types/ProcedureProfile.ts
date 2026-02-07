export interface ProcedureProfile {
  id: string;
  code: string;
  name: string;
  category?: string;
  description?: string;
  estimatedDurationMinutes?: number;
  tags?: string[];
  lastUpdated: Date;
  source: string;
  metadata?: {
    llm?: {
      model?: string;
      generatedAt?: Date;
    };
  };
}
