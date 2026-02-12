import { BlnkService } from '../internal/adapters/blnk/BlnkService';
import { CapacityService } from '../services/CapacityService';
import { FacilityProfileService } from './facilityProfileService';
import { recordTransactionIngestion, recordCapacityEvent } from '../observability/metrics';

export interface IngestionEvent {
  wardId: string;
  facilityId: string;
  transactionAmount: number;
  currency: string;
  reference: string;
  description?: string;
  sourceAccount?: string;
  destinationAccount?: string;
  timestamp: Date;
}

export class TransactionIngestionService {
  constructor(
    private capacityService: CapacityService,
    private blnkService: BlnkService | null,
    private facilityProfileService: FacilityProfileService | null,
    private config: { capacityThreshold: number; windowMinutes: number } = { capacityThreshold: 50, windowMinutes: 240 }
  ) {}

  async ingestEvent(event: IngestionEvent): Promise<{ 
    capacityCount: number; 
    isBusy: boolean; 
    status: string;
    trend: string;
    thresholds: { busy: number; full: number };
    blnkTransactionId?: string;
  }> {
    // 1. Record in Capacity Service (Scoped by Facility and Ward)
    const count = await this.capacityService.recordEvent(event.facilityId, event.wardId);
    recordCapacityEvent(event.wardId);
    
    // 2. Advanced Capacity Analysis (Scoped)
    const analysis = await this.capacityService.analyzeCapacity(event.facilityId, event.wardId);

    // 3. Update Facility Status if changed
    if (this.facilityProfileService) {
      try {
        await this.facilityProfileService.updateStatus(event.facilityId, {
          wardUpdate: {
            wardId: event.wardId,
            status: analysis.status,
            count: analysis.count,
            trend: analysis.trend,
            estimatedWaitMinutes: analysis.estimatedWaitMinutes
          } as any
        }, { source: 'transaction_ingestion' });
      } catch (e) {
        console.error(`Failed to update facility status for ${event.facilityId}`, e);
      }
    }

    // 4. Record in Blnk
    let blnkTransactionId: string | undefined;
    if (this.blnkService && event.sourceAccount && event.destinationAccount) {
      try {
        const tx = await this.blnkService.recordTransaction({
          amount: event.transactionAmount,
          currency: event.currency,
          source: event.sourceAccount,
          destination: event.destinationAccount,
          reference: event.reference,
          description: event.description || `Transaction for ${event.wardId}`
        });
        blnkTransactionId = tx.transaction_id;
      } catch (e) {
        console.error('Failed to record transaction in Blnk', e);
        // We do not fail the whole ingestion if Blnk fails? Or should we?
        // For now, log and continue, as capacity update might be more real-time critical.
        // But for financial reconciliation, we might want to retry.
      }
    }

    recordTransactionIngestion({ 
      wardId: event.wardId, 
      facilityId: event.facilityId, 
      success: true 
    });

    return { 
      capacityCount: count, 
      isBusy: analysis.status !== 'available', 
      status: analysis.status,
      trend: analysis.trend,
      thresholds: analysis.thresholds,
      blnkTransactionId 
    };
  }
}
