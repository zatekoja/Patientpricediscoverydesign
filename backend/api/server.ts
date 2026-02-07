import express, { Request, Response, NextFunction } from 'express';
import { IExternalDataProvider, DataProviderOptions } from '../interfaces/IExternalDataProvider';
import { recordApiRequest } from '../observability/metrics';
import { FacilityProfileService } from '../ingestion/facilityProfileService';
import { CapacityRequestService } from '../ingestion/capacityRequestService';

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

function buildCapacityForm(token: string): string {
  return `
    <html>
      <head>
        <title>Update Capacity</title>
        <meta name="viewport" content="width=device-width, initial-scale=1" />
      </head>
      <body style="font-family: Arial, sans-serif; padding: 24px;">
        <h2>Facility Capacity Update</h2>
        <form method="POST" action="/api/v1/capacity/submit">
          <input type="hidden" name="token" value="${token}" />
          <div style="margin-bottom: 12px;">
            <label>Capacity Status</label><br />
            <select name="capacityStatus">
              <option value="available">Available</option>
              <option value="busy">Busy</option>
              <option value="full">Full</option>
              <option value="closed">Closed</option>
            </select>
          </div>
          <div style="margin-bottom: 12px;">
            <label>Average Wait (minutes)</label><br />
            <input type="number" name="avgWaitMinutes" min="0" />
          </div>
          <div style="margin-bottom: 12px;">
            <label>
              <input type="checkbox" name="urgentCareAvailable" value="true" />
              Urgent care available
            </label>
          </div>
          <button type="submit">Submit Update</button>
        </form>
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

  constructor(options?: DataProviderAPIOptions) {
    this.app = express();
    this.facilityProfileService = options?.facilityProfileService;
    this.capacityRequestService = options?.capacityRequestService;
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
   * Setup Express middleware
   */
  private setupMiddleware(): void {
    this.app.use(express.json());
    this.app.use(express.urlencoded({ extended: true }));

    // API request metrics
    this.app.use((req, res, next) => {
      const start = process.hrtime.bigint();
      res.on('finish', () => {
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

      const options: DataProviderOptions = {
        limit: limit ? parseInt(limit as string, 10) : 100,
        offset: offset ? parseInt(offset as string, 10) : 0,
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

      const options: DataProviderOptions = {
        limit: limit ? parseInt(limit as string, 10) : 100,
        offset: offset ? parseInt(offset as string, 10) : 0,
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

      const parsedLimit = limit ? parseInt(limit as string, 10) : 1000;
      const parsedOffset = offset ? parseInt(offset as string, 10) : 0;
      if (Number.isNaN(parsedLimit) || Number.isNaN(parsedOffset)) {
        res.status(400).json({
          error: 'ValidationError',
          message: 'limit and offset must be valid integers.',
        });
        return;
      }

      const options: DataProviderOptions = {
        timeWindow: timeWindow as string,
        startDate: parsedStartDate,
        endDate: parsedEndDate,
        limit: parsedLimit,
        offset: parsedOffset,
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
      const limit = req.query.limit ? parseInt(req.query.limit as string, 10) : 100;
      const offset = req.query.offset ? parseInt(req.query.offset as string, 10) : 0;
      const data = await this.facilityProfileService.listProfiles(limit, offset);
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

      const updated = await this.facilityProfileService.updateStatus(id, {
        capacityStatus,
        avgWaitMinutes,
        urgentCareAvailable,
      });
      res.json(updated);
    } catch (error) {
      next(error);
    }
  }

  /**
   * GET /api/v1/capacity/form/:token
   */
  private async handleCapacityForm(req: Request, res: Response): Promise<void> {
    if (!this.capacityRequestService) {
      res.status(503).send('Capacity service not configured');
      return;
    }
    const token = req.params.token;
    res.setHeader('Content-Type', 'text/html');
    res.send(buildCapacityForm(token));
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

      const record = await this.capacityRequestService.consumeToken(token);
      const capacityStatus = req.body?.capacityStatus ? String(req.body.capacityStatus) : undefined;
      const avgWaitMinutes = req.body?.avgWaitMinutes
        ? parseInt(req.body.avgWaitMinutes, 10)
        : undefined;
      const urgentCareAvailable =
        typeof req.body?.urgentCareAvailable === 'string'
          ? req.body.urgentCareAvailable === 'true'
          : req.body?.urgentCareAvailable === true;

      await this.facilityProfileService.updateStatus(record.facilityId, {
        capacityStatus,
        avgWaitMinutes: Number.isFinite(avgWaitMinutes) ? avgWaitMinutes : undefined,
        urgentCareAvailable,
      });

      res.setHeader('Content-Type', 'text/html');
      res.send('<p>Thank you. Your capacity update has been recorded.</p>');
    } catch (error) {
      next(error);
    }
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
