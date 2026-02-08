import express, { Request, Response, NextFunction } from 'express';
import { IExternalDataProvider, DataProviderOptions } from '../interfaces/IExternalDataProvider';
import { recordApiRequest, incrementActiveRequests, decrementActiveRequests } from '../observability/metrics';
import { FacilityProfileService } from '../ingestion/facilityProfileService';
import { CapacityRequestService } from '../ingestion/capacityRequestService';
import { recordCapacityWebhook } from '../observability/metrics';

/**
 * API Router for External Data Provider
 * Provides REST endpoints for accessing price data
 */

export interface ProviderRegistry {
  [key: string]: IExternalDataProvider;
}

class ProviderNotFoundError extends Error {
  statusCode = 404;
  constructor(message: string) {
    super(message);
    this.name = 'ProviderNotFoundError';
  }
}

// Predefined ward types for common hospital departments
const PREDEFINED_WARD_TYPES = [
  { value: 'maternity', label: 'Maternity' },
  { value: 'pharmacy', label: 'Pharmacy' },
  { value: 'inpatient', label: 'Inpatient' },
  { value: 'outpatient', label: 'Outpatient' },
  { value: 'emergency', label: 'Emergency' },
  { value: 'surgery', label: 'Surgery' },
  { value: 'icu', label: 'ICU' },
  { value: 'pediatrics', label: 'Pediatrics' },
  { value: 'radiology', label: 'Radiology' },
  { value: 'laboratory', label: 'Laboratory' },
  { value: 'other', label: 'Other (Custom)' },
];

function buildCapacityForm(
  token: string,
  facilityName: string,
  availableWards?: string[],
  preselectedWard?: string,
  errorMessage?: string
): string {
  // Get available wards from facility metadata or use predefined list
  const wardOptions = availableWards && availableWards.length > 0
    ? availableWards.map(ward => {
        const predefined = PREDEFINED_WARD_TYPES.find(w => w.value === ward.toLowerCase());
        return {
          value: ward,
          label: predefined ? predefined.label : ward.charAt(0).toUpperCase() + ward.slice(1)
        };
      })
    : PREDEFINED_WARD_TYPES;

  const wardSelectOptions = wardOptions
    .map(w => `<option value="${w.value}" ${w.value === preselectedWard ? 'selected' : ''}>${w.label}</option>`)
    .join('');

  return `
    <!DOCTYPE html>
    <html lang="en">
      <head>
        <meta charset="UTF-8">
        <meta name="viewport" content="width=device-width, initial-scale=1.0">
        <title>Update Capacity - ${facilityName}</title>
        <style>
          * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
          }
          body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            background: #f3f3f5;
            color: #030213;
            line-height: 1.5;
            padding: 24px;
          }
          .container {
            max-width: 600px;
            margin: 0 auto;
            background: #ffffff;
            border-radius: 0.625rem;
            box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
            padding: 32px;
          }
          h1 {
            font-size: 1.5rem;
            font-weight: 500;
            margin-bottom: 8px;
            color: #030213;
          }
          .subtitle {
            color: #717182;
            font-size: 0.875rem;
            margin-bottom: 24px;
          }
          .error-message {
            background: #fee;
            color: #c33;
            padding: 12px;
            border-radius: 0.5rem;
            margin-bottom: 20px;
            font-size: 0.875rem;
          }
          .form-group {
            margin-bottom: 20px;
          }
          label {
            display: block;
            font-size: 0.875rem;
            font-weight: 500;
            margin-bottom: 8px;
            color: #030213;
          }
          select, input[type="number"], input[type="text"] {
            width: 100%;
            padding: 8px 12px;
            font-size: 0.875rem;
            border: 1px solid rgba(0, 0, 0, 0.1);
            border-radius: 0.625rem;
            background: #f3f3f5;
            color: #030213;
            transition: all 0.2s;
          }
          select:focus, input[type="number"]:focus, input[type="text"]:focus {
            outline: none;
            border-color: #030213;
            box-shadow: 0 0 0 3px rgba(3, 2, 19, 0.1);
          }
          .checkbox-group {
            display: flex;
            align-items: center;
            gap: 8px;
          }
          input[type="checkbox"] {
            width: 18px;
            height: 18px;
            cursor: pointer;
          }
          .checkbox-label {
            font-weight: 400;
            margin: 0;
            cursor: pointer;
          }
          button[type="submit"] {
            width: 100%;
            padding: 10px 16px;
            font-size: 0.875rem;
            font-weight: 500;
            background: #030213;
            color: #ffffff;
            border: none;
            border-radius: 0.625rem;
            cursor: pointer;
            transition: background 0.2s;
          }
          button[type="submit"]:hover {
            background: rgba(3, 2, 19, 0.9);
          }
          button[type="submit"]:active {
            transform: scale(0.98);
          }
          .help-text {
            font-size: 0.75rem;
            color: #717182;
            margin-top: 4px;
          }
          #customWardInput {
            margin-top: 8px;
            display: none;
          }
          #customWardInput.show {
            display: block;
          }
          @media (max-width: 640px) {
            .container {
              padding: 24px;
            }
            h1 {
              font-size: 1.25rem;
            }
          }
        </style>
      </head>
      <body>
        <div class="container">
          <h1>Update Capacity - ${facilityName}</h1>
          <p class="subtitle">Please update your current capacity and wait time information</p>
          ${errorMessage ? `<div class="error-message">${errorMessage}</div>` : ''}
          <form method="POST" action="/api/v1/capacity/submit" id="capacityForm">
            <input type="hidden" name="token" value="${token}" />
            <div class="form-group">
              <label for="wardName">Ward/Department</label>
              <select name="wardName" id="wardName">
                <option value="">Facility-wide (all departments)</option>
                ${wardSelectOptions}
              </select>
              <p class="help-text">Select a specific ward/department or leave blank for facility-wide update</p>
              <input type="text" name="customWardName" id="customWardInput" placeholder="Enter custom ward name" />
            </div>
            <div class="form-group">
              <label for="capacityStatus">Capacity Status *</label>
              <select name="capacityStatus" id="capacityStatus" required>
                <option value="">Select status...</option>
                <option value="available">Available</option>
                <option value="busy">Busy</option>
                <option value="full">Full</option>
                <option value="closed">Closed</option>
              </select>
            </div>
            <div class="form-group">
              <label for="avgWaitMinutes">Average Wait Time (minutes)</label>
              <input type="number" name="avgWaitMinutes" id="avgWaitMinutes" min="0" placeholder="e.g., 30" />
              <p class="help-text">Optional: Estimated average wait time in minutes</p>
            </div>
            <div class="form-group">
              <div class="checkbox-group">
                <input type="checkbox" name="urgentCareAvailable" id="urgentCareAvailable" value="true" />
                <label for="urgentCareAvailable" class="checkbox-label">Urgent care available</label>
              </div>
            </div>
            <button type="submit">Submit Update</button>
          </form>
        </div>
        <script>
          // Show custom ward input when "Other" is selected
          const wardSelect = document.getElementById('wardName');
          const customWardInput = document.getElementById('customWardInput');
          const customWardNameInput = document.getElementById('customWardInput');
          
          wardSelect.addEventListener('change', function() {
            if (this.value === 'other') {
              customWardInput.classList.add('show');
              customWardNameInput.setAttribute('required', 'required');
            } else {
              customWardInput.classList.remove('show');
              customWardNameInput.removeAttribute('required');
              customWardNameInput.value = '';
            }
          });

          // Handle form submission - use custom ward name if "other" selected
          document.getElementById('capacityForm').addEventListener('submit', function(e) {
            if (wardSelect.value === 'other' && customWardNameInput.value.trim()) {
              // Create a hidden input with the custom ward name
              const hiddenInput = document.createElement('input');
              hiddenInput.type = 'hidden';
              hiddenInput.name = 'wardName';
              hiddenInput.value = customWardNameInput.value.trim();
              this.appendChild(hiddenInput);
              // Clear the original wardName select
              wardSelect.removeAttribute('name');
            }
          });
        </script>
      </body>
    </html>
  `;
}

export interface DataProviderAPIOptions {
  facilityProfileService?: FacilityProfileService;
  capacityRequestService?: CapacityRequestService;
}

export class DataProviderAPI {
  private app: express.Application;
  private providers: ProviderRegistry = {};
  private defaultProviderId?: string;
  private facilityProfileService?: FacilityProfileService;
  private capacityRequestService?: CapacityRequestService;
  private adminToken?: string;

  constructor(options?: DataProviderAPIOptions) {
    this.app = express();
    this.facilityProfileService = options?.facilityProfileService;
    this.capacityRequestService = options?.capacityRequestService;
    this.adminToken = process.env.PROVIDER_ADMIN_TOKEN;
    this.setupMiddleware();
    this.setupRoutes();
  }

  /**
   * Register a data provider
   */
  registerProvider(id: string, provider: IExternalDataProvider, isDefault: boolean = false): void {
    this.providers[id] = provider;
    if (isDefault || !this.defaultProviderId) {
      this.defaultProviderId = id;
    }
  }

  /**
   * Get provider by ID or default
   */
  private getProvider(providerId?: string): IExternalDataProvider {
    const id = providerId || this.defaultProviderId;
    if (!id || !this.providers[id]) {
      throw new ProviderNotFoundError(`Provider not found: ${id || 'no default provider'}`);
    }
    return this.providers[id];
  }

  /**
   * Validate pagination parameters
   */
  private validatePagination(
    limit?: string,
    offset?: string,
    options?: {
      defaultLimit?: number;
      defaultOffset?: number;
      maxLimit?: number;
    }
  ): { valid: true; limit: number; offset: number } | { valid: false; error: string } {
    const MAX_LIMIT = options?.maxLimit ?? 5000;
    const DEFAULT_LIMIT = options?.defaultLimit ?? 100;
    const DEFAULT_OFFSET = options?.defaultOffset ?? 0;

    let parsedLimit = DEFAULT_LIMIT;
    let parsedOffset = DEFAULT_OFFSET;

    // Validate limit
    if (limit !== undefined) {
      parsedLimit = parseInt(limit, 10);
      if (Number.isNaN(parsedLimit)) {
        return { valid: false, error: 'limit must be a valid integer' };
      }
      if (parsedLimit < 0) {
        return { valid: false, error: 'limit must be non-negative' };
      }
      if (parsedLimit > MAX_LIMIT) {
        return { valid: false, error: `limit must not exceed ${MAX_LIMIT}` };
      }
    }

    // Validate offset
    if (offset !== undefined) {
      parsedOffset = parseInt(offset, 10);
      if (Number.isNaN(parsedOffset)) {
        return { valid: false, error: 'offset must be a valid integer' };
      }
      if (parsedOffset < 0) {
        return { valid: false, error: 'offset must be non-negative' };
      }
    }

    return { valid: true, limit: parsedLimit, offset: parsedOffset };
  }

  /**
   * Setup Express middleware
   */
  private setupMiddleware(): void {
    this.app.use(express.json());
    this.app.use(express.urlencoded({ extended: true }));

    // API request metrics
    this.app.use((req, res, next) => {
      const start = process.hrtime.bigint();
      incrementActiveRequests();
      res.on('finish', () => {
        decrementActiveRequests();
        const durationMs = Number(process.hrtime.bigint() - start) / 1_000_000;
        const routePath = `${req.baseUrl || ''}${req.route?.path || req.path}`;
        recordApiRequest({
          method: req.method,
          path: routePath,
          status: res.statusCode,
          durationMs,
        });
      });
      next();
    });
    
    // CORS middleware - configure allowed origins for production
    this.app.use((req, res, next) => {
      const allowedOriginsEnv = process.env.ALLOWED_ORIGINS;
      const allowedOrigins = allowedOriginsEnv
        ? allowedOriginsEnv.split(',').map(o => o.trim()).filter(o => o.length > 0)
        : [];
      const origin = req.headers.origin || '';
      
      if (allowedOrigins.length > 0) {
        if (allowedOrigins.includes('*') || allowedOrigins.includes(origin)) {
          res.header('Access-Control-Allow-Origin', allowedOrigins.includes('*') ? '*' : origin);
        }
      }
      
      res.header('Access-Control-Allow-Methods', 'GET, POST, OPTIONS');
      res.header('Access-Control-Allow-Headers', 'Content-Type, Authorization');

      // Short-circuit preflight OPTIONS requests for CORS
      if (req.method === 'OPTIONS') {
        return res.sendStatus(204);
      }
      
      next();
    });

    // Request logging
    this.app.use((req, res, next) => {
      console.log(`${new Date().toISOString()} - ${req.method} ${req.path}`);
      next();
    });
  }

  /**
   * Setup API routes
   */
  private setupRoutes(): void {
    const router = express.Router();

    // Data endpoints
    router.get('/data/current', this.handleGetCurrentData.bind(this));
    router.get('/data/previous', this.handleGetPreviousData.bind(this));
    router.get('/data/historical', this.handleGetHistoricalData.bind(this));

    // Provider endpoints
    router.get('/provider/health', this.handleGetProviderHealth.bind(this));
    router.get('/provider/list', this.handleListProviders.bind(this));

    // Facility profile endpoints
    router.get('/facilities', this.handleListFacilities.bind(this));
    router.get('/facilities/:id', this.handleGetFacility.bind(this));
    router.patch('/facilities/:id/status', this.handleUpdateFacilityStatus.bind(this));
    router.get('/capacity/form/:token', this.handleCapacityForm.bind(this));
    router.post('/capacity/submit', this.handleCapacitySubmit.bind(this));
    router.post('/capacity/request', this.requireAdmin.bind(this), this.handleCapacityRequest.bind(this));

    // Sync endpoints
    router.post('/sync/trigger', this.handleTriggerSync.bind(this));
    router.get('/sync/status', this.handleGetSyncStatus.bind(this));

    // Health check endpoint
    router.get('/health', (req, res) => {
      res.json({ status: 'ok', timestamp: new Date().toISOString() });
    });

    this.app.use('/api/v1', router);

    // Error handling middleware
    this.app.use(this.errorHandler.bind(this));
  }

  /**
   * GET /api/v1/data/current
   */
  private async handleGetCurrentData(req: Request, res: Response, next: NextFunction): Promise<void> {
    try {
      const { limit, offset, providerId } = req.query;
      const provider = this.getProvider(providerId as string);

      const validatedPagination = this.validatePagination(
        limit as string | undefined,
        offset as string | undefined
      );

      if (!validatedPagination.valid) {
        res.status(400).json({
          error: 'ValidationError',
          message: validatedPagination.error,
        });
        return;
      }

      const options: DataProviderOptions = {
        limit: validatedPagination.limit,
        offset: validatedPagination.offset,
      };

      const data = await provider.getCurrentData(options);
      res.json(data);
    } catch (error) {
      next(error);
    }
  }

  /**
   * GET /api/v1/data/previous
   */
  private async handleGetPreviousData(req: Request, res: Response, next: NextFunction): Promise<void> {
    try {
      const { limit, offset, providerId } = req.query;
      const provider = this.getProvider(providerId as string);

      const validatedPagination = this.validatePagination(
        limit as string | undefined,
        offset as string | undefined
      );

      if (!validatedPagination.valid) {
        res.status(400).json({
          error: 'ValidationError',
          message: validatedPagination.error,
        });
        return;
      }

      const options: DataProviderOptions = {
        limit: validatedPagination.limit,
        offset: validatedPagination.offset,
      };

      const data = await provider.getPreviousData(options);
      res.json(data);
    } catch (error) {
      next(error);
    }
  }

  /**
   * GET /api/v1/data/historical
   */
  private async handleGetHistoricalData(req: Request, res: Response, next: NextFunction): Promise<void> {
    try {
      const { timeWindow, startDate, endDate, limit, offset, providerId } = req.query;
      const provider = this.getProvider(providerId as string);

      // Validate that either timeWindow or startDate/endDate is provided
      if (!timeWindow && (!startDate || !endDate)) {
        res.status(400).json({
          error: 'ValidationError',
          message: 'Either timeWindow or both startDate and endDate must be provided',
        });
        return;
      }

      if (timeWindow && !/^\d+[dmy]$/.test(timeWindow as string)) {
        res.status(400).json({
          error: 'ValidationError',
          message: 'Invalid timeWindow format. Use <number><unit> like "30d", "6m", "1y".',
        });
        return;
      }

      const parsedStartDate = startDate ? new Date(startDate as string) : undefined;
      const parsedEndDate = endDate ? new Date(endDate as string) : undefined;

      if (startDate && isNaN(parsedStartDate!.getTime())) {
        res.status(400).json({
          error: 'ValidationError',
          message: 'Invalid startDate format.',
        });
        return;
      }

      if (endDate && isNaN(parsedEndDate!.getTime())) {
        res.status(400).json({
          error: 'ValidationError',
          message: 'Invalid endDate format.',
        });
        return;
      }

      if (parsedStartDate && parsedEndDate && parsedEndDate < parsedStartDate) {
        res.status(400).json({
          error: 'ValidationError',
          message: 'endDate must be greater than or equal to startDate.',
        });
        return;
      }

      const validatedPagination = this.validatePagination(
        limit as string | undefined,
        offset as string | undefined,
        { defaultLimit: 1000 }
      );

      if (!validatedPagination.valid) {
        res.status(400).json({
          error: 'ValidationError',
          message: validatedPagination.error,
        });
        return;
      }

      const options: DataProviderOptions = {
        timeWindow: timeWindow as string,
        startDate: parsedStartDate,
        endDate: parsedEndDate,
        limit: validatedPagination.limit,
        offset: validatedPagination.offset,
      };

      const data = await provider.getHistoricalData(options);
      res.json(data);
    } catch (error) {
      next(error);
    }
  }

  /**
   * GET /api/v1/provider/health
   */
  private async handleGetProviderHealth(req: Request, res: Response, next: NextFunction): Promise<void> {
    try {
      const { providerId } = req.query;
      const provider = this.getProvider(providerId as string);

      const health = await provider.getHealthStatus();
      res.json(health);
    } catch (error) {
      next(error);
    }
  }

  /**
   * GET /api/v1/provider/list
   */
  private async handleListProviders(req: Request, res: Response, next: NextFunction): Promise<void> {
    try {
      const providerList = await Promise.all(
        Object.keys(this.providers).map(async (id) => {
          const provider = this.providers[id];
          const health = await provider.getHealthStatus();
          return {
            id,
            name: provider.getName(),
            type: id, // Use provider ID as type identifier
            healthy: health.healthy,
            lastSync: health.lastSync,
          };
        })
      );

      res.json({ providers: providerList });
    } catch (error) {
      next(error);
    }
  }

  /**
   * POST /api/v1/sync/trigger
   */
  private async handleTriggerSync(req: Request, res: Response, next: NextFunction): Promise<void> {
    try {
      const { providerId } = req.query;
      const provider = this.getProvider(providerId as string);
      
      const result = await provider.syncData();
      if (result.success) {
        await this.enrichFacilities(provider, providerId as string | undefined);
      }
      res.json(result);
    } catch (error) {
      next(error);
    }
  }

  /**
   * GET /api/v1/sync/status
   */
  private async handleGetSyncStatus(req: Request, res: Response, next: NextFunction): Promise<void> {
    try {
      const { providerId } = req.query;
      const provider = this.getProvider(providerId as string);

      // For now, we'll return the health status which includes last sync info
      // In a production system, you might want to track sync history separately
      const health = await provider.getHealthStatus();
      
      res.json({
        success: health.healthy,
        recordsProcessed: 0, // Would need to track this separately
        timestamp: health.lastSync || new Date(),
        error: health.healthy ? null : health.message,
      });
    } catch (error) {
      next(error);
    }
  }

  /**
   * GET /api/v1/facilities
   */
  private async handleListFacilities(req: Request, res: Response, next: NextFunction): Promise<void> {
    try {
      if (!this.facilityProfileService) {
        res.status(503).json({ error: 'Facility profiles not configured' });
        return;
      }
      const validatedPagination = this.validatePagination(
        req.query.limit as string | undefined,
        req.query.offset as string | undefined
      );
      if (!validatedPagination.valid) {
        res.status(400).json({
          error: 'ValidationError',
          message: validatedPagination.error,
        });
        return;
      }
      const data = await this.facilityProfileService.listProfiles(
        validatedPagination.limit,
        validatedPagination.offset
      );
      res.json({ data, count: data.length });
    } catch (error) {
      next(error);
    }
  }

  /**
   * GET /api/v1/facilities/:id
   */
  private async handleGetFacility(req: Request, res: Response, next: NextFunction): Promise<void> {
    try {
      if (!this.facilityProfileService) {
        res.status(503).json({ error: 'Facility profiles not configured' });
        return;
      }
      const id = req.params.id;
      const profile = await this.facilityProfileService.getProfile(id);
      if (!profile) {
        res.status(404).json({ error: 'Facility not found' });
        return;
      }
      res.json(profile);
    } catch (error) {
      next(error);
    }
  }

  /**
   * PATCH /api/v1/facilities/:id/status
   */
  private async handleUpdateFacilityStatus(req: Request, res: Response, next: NextFunction): Promise<void> {
    try {
      if (!this.facilityProfileService) {
        res.status(503).json({ error: 'Facility profiles not configured' });
        return;
      }
      const id = req.params.id;
      if (!id) {
        res.status(400).json({ error: 'Facility id is required' });
        return;
      }

      const payload = req.body || {};
      const capacityStatus =
        typeof payload.capacityStatus === 'string' ? payload.capacityStatus : undefined;
      const avgWaitMinutes =
        typeof payload.avgWaitMinutes === 'number' ? payload.avgWaitMinutes : undefined;
      const urgentCareAvailable =
        typeof payload.urgentCareAvailable === 'boolean' ? payload.urgentCareAvailable : undefined;

      const updated = await this.facilityProfileService.updateStatus(
        id,
        {
          capacityStatus,
          avgWaitMinutes,
          urgentCareAvailable,
        },
        { source: 'admin_patch' }
      );
      res.json(updated);
    } catch (error) {
      next(error);
    }
  }

  /**
   * GET /api/v1/capacity/form/:token
   */
  private async handleCapacityForm(req: Request, res: Response): Promise<void> {
    if (!this.capacityRequestService || !this.facilityProfileService) {
      res.status(503).send('Capacity service not configured');
      return;
    }
    const token = req.params.token;
    
    try {
      // Validate token and get facility info
      const tokenHash = require('crypto').createHash('sha256').update(token).digest('hex');
      const tokenStore = this.capacityRequestService['options'].tokenStore;
      const record = await tokenStore.get(tokenHash);
      
      if (!record) {
        res.status(400).send('Invalid token');
        return;
      }
      
      if (record.usedAt) {
        res.status(400).send('Token already used');
        return;
      }
      
      if (new Date(record.expiresAt) < new Date()) {
        res.status(400).send('Token expired');
        return;
      }

      // Get facility profile to show name and available wards
      const facility = await this.facilityProfileService.getProfile(record.facilityId);
      const facilityName = facility?.name || 'Facility';
      const availableWards = facility?.metadata?.availableWards;
      const preselectedWard = record.wardName;

      res.setHeader('Content-Type', 'text/html');
      res.send(buildCapacityForm(token, facilityName, availableWards, preselectedWard));
    } catch (error: any) {
      res.status(500).send(`Error loading form: ${error.message}`);
    }
  }

  /**
   * POST /api/v1/capacity/submit
   */
  private async handleCapacitySubmit(req: Request, res: Response, next: NextFunction): Promise<void> {
    try {
      if (!this.capacityRequestService || !this.facilityProfileService) {
        res.status(503).send('Capacity service not configured');
        return;
      }
      const token = (req.body?.token || '').toString().trim();
      if (!token) {
        res.status(400).send('Token is required');
        return;
      }

      let record;
      try {
        record = await this.capacityRequestService.consumeToken(token);
      } catch (error: any) {
        // Handle token errors with proper status codes
        if (error.message === 'Invalid token' || error.message === 'Token expired' || error.message === 'Token already used') {
          res.status(400).send(`
            <html>
              <body style="font-family: Arial, sans-serif; padding: 24px;">
                <h2>Token Error</h2>
                <p>${error.message}</p>
                <p>Please request a new capacity update link.</p>
              </body>
            </html>
          `);
          return;
        }
        throw error; // Re-throw other errors
      }
      
      // Validate capacity status
      const capacityStatusRaw = req.body?.capacityStatus ? String(req.body.capacityStatus).toLowerCase().trim() : undefined;
      const validCapacityStatuses = ['available', 'busy', 'full', 'closed'];
      if (capacityStatusRaw && !validCapacityStatuses.includes(capacityStatusRaw)) {
        res.status(400).send(`
          <html>
            <body style="font-family: Arial, sans-serif; padding: 24px;">
              <h2>Invalid Capacity Status</h2>
              <p>Capacity status must be one of: ${validCapacityStatuses.join(', ')}</p>
              <p><a href="/api/v1/capacity/form/${encodeURIComponent(token)}">Go back to form</a></p>
            </body>
          </html>
        `);
        return;
      }
      const capacityStatus = capacityStatusRaw;
      
      const avgWaitMinutes = req.body?.avgWaitMinutes
        ? parseInt(req.body.avgWaitMinutes, 10)
        : undefined;
      const urgentCareAvailable =
        typeof req.body?.urgentCareAvailable === 'string'
          ? req.body.urgentCareAvailable === 'true'
          : req.body?.urgentCareAvailable === true;

      // Get ward name from form (or from token if specified)
      const wardName = req.body?.wardName 
        ? String(req.body.wardName).trim() 
        : record.wardName || undefined;

      await this.facilityProfileService.updateStatus(
        record.facilityId,
        {
          wardName, // Pass ward name for ward-specific update
          capacityStatus,
          avgWaitMinutes: Number.isFinite(avgWaitMinutes) ? avgWaitMinutes : undefined,
          urgentCareAvailable,
        },
        { source: 'form' }
      );

      await this.triggerIngestionWebhook(record.facilityId, record.id);

      res.setHeader('Content-Type', 'text/html');
      res.send(`
        <!DOCTYPE html>
        <html lang="en">
          <head>
            <meta charset="UTF-8">
            <meta name="viewport" content="width=device-width, initial-scale=1.0">
            <title>Update Submitted - Patient Price Discovery</title>
            <style>
              * {
                margin: 0;
                padding: 0;
                box-sizing: border-box;
              }
              body {
                font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
                background: #f3f3f5;
                color: #030213;
                line-height: 1.5;
                padding: 24px;
                display: flex;
                align-items: center;
                justify-content: center;
                min-height: 100vh;
              }
              .container {
                max-width: 500px;
                background: #ffffff;
                border-radius: 0.625rem;
                box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
                padding: 32px;
                text-align: center;
              }
              h2 {
                font-size: 1.5rem;
                font-weight: 500;
                margin-bottom: 16px;
                color: #030213;
              }
              p {
                color: #717182;
                font-size: 0.875rem;
              }
              .success-icon {
                font-size: 3rem;
                margin-bottom: 16px;
              }
            </style>
          </head>
          <body>
            <div class="container">
              <div class="success-icon">✓</div>
              <h2>Thank You</h2>
              <p>Your capacity update has been recorded successfully.</p>
            </div>
          </body>
        </html>
      `);
    } catch (error) {
      next(error);
    }
  }

  private async handleCapacityRequest(req: Request, res: Response, next: NextFunction): Promise<void> {
    try {
      if (!this.capacityRequestService) {
        res.status(503).json({ error: 'Capacity service not configured' });
        return;
      }
      const facilityId = (req.body?.facilityId || req.query?.facilityId || '').toString().trim();
      if (!facilityId) {
        res.status(400).json({ error: 'facilityId is required' });
        return;
      }
      const channelRaw = (req.body?.channel || req.query?.channel || '').toString().trim().toLowerCase();
      const channel = channelRaw === 'email' || channelRaw === 'whatsapp' ? channelRaw : undefined;
      const wardName = req.body?.wardName ? String(req.body.wardName).trim() : undefined;
      await this.capacityRequestService.sendSingleRequest(facilityId, channel, wardName);
      res.json({ success: true });
    } catch (error) {
      next(error);
    }
  }

  private requireAdmin(req: Request, res: Response, next: NextFunction): void {
    if (!this.adminToken) {
      res.status(503).json({ error: 'Admin token not configured' });
      return;
    }
    const header = req.headers.authorization || '';
    const token = header.startsWith('Bearer ') ? header.slice(7) : header;
    if (token !== this.adminToken) {
      res.status(401).json({ error: 'Unauthorized' });
      return;
    }
    next();
  }

  private async triggerIngestionWebhook(facilityId: string, eventId?: string): Promise<void> {
    const webhookUrl = process.env.PROVIDER_INGESTION_WEBHOOK_URL;
    if (!webhookUrl) {
      return;
    }

    const maxRetries = parseInt(process.env.PROVIDER_WEBHOOK_MAX_RETRIES || '3', 10);
    const retryDelayMs = parseInt(process.env.PROVIDER_WEBHOOK_RETRY_DELAY_MS || '1000', 10);
    const useExponentialBackoff = process.env.PROVIDER_WEBHOOK_EXPONENTIAL_BACKOFF !== 'false';

    let lastError: Error | null = null;
    let lastResponse: globalThis.Response | null = null;

    for (let attempt = 0; attempt <= maxRetries; attempt++) {
      try {
        const headers: Record<string, string> = { 'Content-Type': 'application/json' };
        if (eventId) {
          headers['Idempotency-Key'] = eventId;
        }
        
        const response = await fetch(webhookUrl, {
          method: 'POST',
          headers,
          body: JSON.stringify({
            eventId,
            facilityId,
            source: 'capacity_update',
            timestamp: new Date().toISOString(),
          }),
        });

        if (response.ok) {
          recordCapacityWebhook({ success: true });
          return; // Success, exit retry loop
        }

        lastResponse = response;
        const text = await response.text();
        lastError = new Error(`Webhook failed with status ${response.status}: ${text}`);

        // Don't retry on client errors (4xx) except 429 (rate limit)
        if (response.status >= 400 && response.status < 500 && response.status !== 429) {
          console.error(`Ingestion webhook failed (${response.status}): ${text}`);
          recordCapacityWebhook({ success: false });
          return; // Don't retry client errors
        }

        // If this was the last attempt, break
        if (attempt === maxRetries) {
          break;
        }

        // Calculate delay with exponential backoff
        const delay = useExponentialBackoff
          ? retryDelayMs * Math.pow(2, attempt)
          : retryDelayMs;

        console.warn(
          `Webhook attempt ${attempt + 1}/${maxRetries + 1} failed, retrying in ${delay}ms...`
        );
        await new Promise((resolve) => setTimeout(resolve, delay));

      } catch (error) {
        lastError = error instanceof Error ? error : new Error(String(error));
        
        // If this was the last attempt, break
        if (attempt === maxRetries) {
          break;
        }

        // Calculate delay with exponential backoff
        const delay = useExponentialBackoff
          ? retryDelayMs * Math.pow(2, attempt)
          : retryDelayMs;

        console.warn(
          `Webhook attempt ${attempt + 1}/${maxRetries + 1} error, retrying in ${delay}ms...`,
          error
        );
        await new Promise((resolve) => setTimeout(resolve, delay));
      }
    }

    // All retries exhausted
    const errorMsg = lastError?.message || 'Unknown error';
    const statusMsg = lastResponse ? ` (status: ${lastResponse.status})` : '';
    console.error(`Ingestion webhook failed after ${maxRetries + 1} attempts: ${errorMsg}${statusMsg}`);
    recordCapacityWebhook({ success: false });
  }

  private async enrichFacilities(provider: IExternalDataProvider, providerId?: string): Promise<void> {
    if (!this.facilityProfileService) {
      return;
    }
    await this.facilityProfileService.ensureProfilesFromProvider(provider, {
      providerId: providerId || provider.getName(),
      pageSize: 500,
    });
  }

  /**
   * Error handling middleware
   */
  private errorHandler(err: any, req: Request, res: Response, next: NextFunction): void {
    console.error('API Error:', err);

    const statusCode = err.statusCode || (res.statusCode !== 200 ? res.statusCode : 500);
    
    res.status(statusCode).json({
      error: err.name || 'InternalServerError',
      message: err.message || 'An unexpected error occurred',
      details: process.env.NODE_ENV === 'development' ? { stack: err.stack } : undefined,
    });
  }

  /**
   * Get Express app instance
   */
  getApp(): express.Application {
    return this.app;
  }

  /**
   * Start the API server
   */
  listen(port: number = 3000): void {
    this.app.listen(port, () => {
      console.log(`Data Provider API server listening on port ${port}`);
      console.log(`API available at: http://localhost:${port}/api/v1`);
      console.log(`Registered providers: ${Object.keys(this.providers).join(', ') || 'none'}`);
    });
  }
}

export default DataProviderAPI;
