import express, { Request, Response, NextFunction } from 'express';
import { IExternalDataProvider, DataProviderOptions } from '../interfaces/IExternalDataProvider';

/**
 * API Router for External Data Provider
 * Provides REST endpoints for accessing price data
 */

export interface ProviderRegistry {
  [key: string]: IExternalDataProvider;
}

export class DataProviderAPI {
  private app: express.Application;
  private providers: ProviderRegistry = {};
  private defaultProviderId?: string;

  constructor() {
    this.app = express();
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
      throw new Error(`Provider not found: ${id || 'no default provider'}`);
    }
    return this.providers[id];
  }

  /**
   * Setup Express middleware
   */
  private setupMiddleware(): void {
    this.app.use(express.json());
    
    // CORS middleware
    this.app.use((req, res, next) => {
      res.header('Access-Control-Allow-Origin', '*');
      res.header('Access-Control-Allow-Methods', 'GET, POST, OPTIONS');
      res.header('Access-Control-Allow-Headers', 'Content-Type');
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

      const options: DataProviderOptions = {
        timeWindow: timeWindow as string,
        startDate: startDate ? new Date(startDate as string) : undefined,
        endDate: endDate ? new Date(endDate as string) : undefined,
        limit: limit ? parseInt(limit as string, 10) : 1000,
        offset: offset ? parseInt(offset as string, 10) : 0,
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
            type: 'External', // Could be enhanced to return provider type
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
   * Error handling middleware
   */
  private errorHandler(err: Error, req: Request, res: Response, next: NextFunction): void {
    console.error('API Error:', err);

    const statusCode = res.statusCode !== 200 ? res.statusCode : 500;
    
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
      console.log(`OpenAPI documentation available at: http://localhost:${port}/api/v1/docs`);
      console.log(`Registered providers: ${Object.keys(this.providers).join(', ') || 'none'}`);
    });
  }
}

export default DataProviderAPI;
