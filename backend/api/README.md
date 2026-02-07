# External Data Provider REST API

HTTP REST API for accessing healthcare price data from external providers.

## Quick Start

### 1. Install Dependencies

```bash
cd backend
npm install
```

### 2. Configure Environment

Create a `.env` file:

```bash
# Google Sheets Configuration
GOOGLE_CLIENT_EMAIL=your-service-account@project.iam.gserviceaccount.com
GOOGLE_PRIVATE_KEY="-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----"
GOOGLE_PROJECT_ID=your-project-id
SPREADSHEET_IDS=spreadsheet-id-1,spreadsheet-id-2

# Server Configuration
PORT=3000
NODE_ENV=development

# Provider Storage (MongoDB)
PROVIDER_MONGO_URI=mongodb://localhost:27017
PROVIDER_MONGO_DB=provider_data
PROVIDER_MONGO_COLLECTION=price_records
PROVIDER_STATE_COLLECTION=provider_state
PROVIDER_MONGO_TTL_DAYS=30
PROVIDER_RUN_INITIAL_SYNC=true

# File Provider (for the current hospital price lists)
PRICE_LIST_FILES=./fixtures/price_lists/MEGALEK NEW PRICE LIST 2026.csv,./fixtures/price_lists/NEW LASUTH PRICE LIST (SERVICES).csv,./fixtures/price_lists/PRICE LIST FOR RANDLE GENERAL HOSPITAL JANUARY 2026.csv,./fixtures/price_lists/PRICE_LIST_FOR_OFFICE_USE[1].docx
```

### 3. Start the Server

```bash
npm start
```

The API will be available at `http://localhost:3000/api/v1`

## API Endpoints

### Base URL

- **Development**: `http://localhost:3000/api/v1`
- **Production**: `https://api.ateruhealth.com/v1`

### Data Endpoints

#### Get Current Data

```http
GET /api/v1/data/current?limit=100&offset=0
```

**Query Parameters:**
- `limit` (optional): Maximum records to return (default: 100, max: 1000)
- `offset` (optional): Pagination offset (default: 0)
- `providerId` (optional): Specific provider ID

**Response:**

```json
{
  "data": [
    {
      "id": "megalek_ateru_helper_12345",
      "facilityName": "General Hospital",
      "procedureCode": "70450",
      "procedureDescription": "CT Scan - Head without contrast",
      "price": 1250.00,
      "currency": "USD",
      "effectiveDate": "2024-01-01T00:00:00Z",
      "lastUpdated": "2024-01-15T10:30:00Z",
      "source": "megalek_ateru_helper"
    }
  ],
  "timestamp": "2024-01-15T10:30:00Z",
  "metadata": {
    "source": "megalek_ateru_helper",
    "count": 100,
    "hasMore": true
  }
}
```

#### Get Previous Data

```http
GET /api/v1/data/previous?limit=100
```

Returns the previous batch of data (last sync before current).

#### Get Historical Data

```http
GET /api/v1/data/historical?timeWindow=30d&limit=1000
```

**Query Parameters:**
- `timeWindow` (optional): Time window (e.g., "30d", "6m", "1y")
- `startDate` (optional): Start date (ISO 8601 format)
- `endDate` (optional): End date (ISO 8601 format)
- `limit` (optional): Maximum records (default: 1000, max: 5000)
- `offset` (optional): Pagination offset

**Note**: Either `timeWindow` or both `startDate` and `endDate` must be provided.

**Examples:**

```bash
# Last 30 days
GET /api/v1/data/historical?timeWindow=30d

# Specific date range
GET /api/v1/data/historical?startDate=2024-01-01T00:00:00Z&endDate=2024-12-31T23:59:59Z
```

### Provider Endpoints

#### Get Provider Health

```http
GET /api/v1/provider/health
```

**Response:**

```json
{
  "healthy": true,
  "lastSync": "2024-01-15T10:30:00Z",
  "message": "Provider is operational"
}
```

#### List Providers

```http
GET /api/v1/provider/list
```

**Response:**

```json
{
  "providers": [
    {
      "id": "megalek_ateru_helper",
      "name": "MegalekAteruHelper",
      "type": "GoogleSheets",
      "healthy": true,
      "lastSync": "2024-01-15T10:30:00Z"
    }
  ]
}
```

### Sync Endpoints

#### Trigger Manual Sync

```http
POST /api/v1/sync/trigger
```

**Response:**

```json
{
  "success": true,
  "recordsProcessed": 150,
  "timestamp": "2024-01-15T10:30:00Z",
  "error": null
}
```

#### Get Sync Status

```http
GET /api/v1/sync/status
```

Returns the status of the last sync operation.

### Health Check

```http
GET /api/v1/health
```

**Response:**

```json
{
  "status": "ok",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

## Error Responses

All errors follow this format:

```json
{
  "error": "ValidationError",
  "message": "Invalid request parameters",
  "details": {
    "field": "timeWindow",
    "issue": "Invalid format"
  }
}
```

**HTTP Status Codes:**

- `200` - Success
- `400` - Bad Request (validation error)
- `404` - Not Found
- `500` - Internal Server Error

## Usage Examples

### cURL

```bash
# Get current data
curl http://localhost:3000/api/v1/data/current?limit=50

# Get historical data
curl http://localhost:3000/api/v1/data/historical?timeWindow=30d

# Trigger sync
curl -X POST http://localhost:3000/api/v1/sync/trigger

# Check health
curl http://localhost:3000/api/v1/provider/health
```

### JavaScript/Fetch

```javascript
// Get current data
const response = await fetch('http://localhost:3000/api/v1/data/current?limit=100');
const data = await response.json();
console.log(data);

// Get historical data
const historical = await fetch(
  'http://localhost:3000/api/v1/data/historical?timeWindow=30d'
);
const historicalData = await historical.json();

// Trigger sync
const syncResponse = await fetch(
  'http://localhost:3000/api/v1/sync/trigger',
  { method: 'POST' }
);
const syncResult = await syncResponse.json();
```

### Python

```python
import requests

# Get current data
response = requests.get('http://localhost:3000/api/v1/data/current', 
                       params={'limit': 100})
data = response.json()

# Get historical data
historical = requests.get('http://localhost:3000/api/v1/data/historical',
                         params={'timeWindow': '30d'})
historical_data = historical.json()

# Trigger sync
sync_response = requests.post('http://localhost:3000/api/v1/sync/trigger')
sync_result = sync_response.json()
```

## OpenAPI Specification

The complete OpenAPI 3.0 specification is available at:

```
backend/api/openapi.yaml
```

You can use this with tools like:
- **Swagger UI** - Interactive API documentation
- **Postman** - Import for testing
- **Code Generators** - Generate client SDKs

### View with Swagger UI

```bash
# Install swagger-ui-express
npm install swagger-ui-express

# Add to your server
const swaggerUi = require('swagger-ui-express');
const YAML = require('yamljs');
const swaggerDocument = YAML.load('./api/openapi.yaml');

app.use('/api/v1/docs', swaggerUi.serve, swaggerUi.setup(swaggerDocument));
```

Then visit: `http://localhost:3000/api/v1/docs`

## Authentication

Currently, the API does not require authentication. For production deployments, consider:

1. **API Keys** - Require API key in headers
2. **OAuth 2.0** - Use OAuth for user authentication
3. **JWT Tokens** - Issue JWT tokens for session management
4. **IP Whitelisting** - Restrict access by IP address

Example with API Key:

```typescript
app.use((req, res, next) => {
  const apiKey = req.headers['x-api-key'];
  if (apiKey !== process.env.API_KEY) {
    return res.status(401).json({ error: 'Unauthorized' });
  }
  next();
});
```

## Rate Limiting

For production, implement rate limiting:

```bash
npm install express-rate-limit
```

```typescript
import rateLimit from 'express-rate-limit';

const limiter = rateLimit({
  windowMs: 15 * 60 * 1000, // 15 minutes
  max: 100 // limit each IP to 100 requests per windowMs
});

app.use('/api/v1', limiter);
```

## CORS Configuration

The API includes CORS headers by default. For production, configure specific origins:

```typescript
app.use((req, res, next) => {
  res.header('Access-Control-Allow-Origin', 'https://yourdomain.com');
  res.header('Access-Control-Allow-Methods', 'GET, POST, OPTIONS');
  res.header('Access-Control-Allow-Headers', 'Content-Type, Authorization');
  next();
});
```

## Deployment

### Docker

Create a `Dockerfile`:

```dockerfile
FROM node:18-alpine
WORKDIR /app
COPY package*.json ./
RUN npm install
COPY . .
EXPOSE 3000
CMD ["npm", "start"]
```

Build and run:

```bash
docker build -t price-discovery-api .
docker run -p 3000:3000 --env-file .env price-discovery-api
```

### AWS Lambda

For serverless deployment, use AWS Lambda with API Gateway:

```bash
npm install serverless-http
```

```typescript
import serverless from 'serverless-http';
import { DataProviderAPI } from './api/server';

const api = new DataProviderAPI();
// ... register providers ...

export.handler = serverless(api.getApp());
```

### Environment Variables

Required environment variables:

- `GOOGLE_CLIENT_EMAIL` - Service account email
- `GOOGLE_PRIVATE_KEY` - Service account private key
- `GOOGLE_PROJECT_ID` - GCP project ID
- `SPREADSHEET_IDS` - Comma-separated spreadsheet IDs
- `PORT` - Server port (default: 3000)
- `NODE_ENV` - Environment (development/production)

## Monitoring

### Logging

The API logs all requests:

```
2024-01-15T10:30:00.000Z - GET /api/v1/data/current
```

For production, use a logging service like:
- **Winston** - Structured logging
- **Bunyan** - JSON logging
- **CloudWatch** - AWS logging

### Metrics

Track important metrics:
- Request count
- Response time
- Error rate
- Data freshness

Use tools like:
- **Prometheus** - Metrics collection
- **Grafana** - Visualization
- **DataDog** - APM

## Testing

### Manual Testing

Use the provided examples or tools like:
- **Postman** - Import OpenAPI spec
- **Insomnia** - REST client
- **HTTPie** - CLI tool

### Integration Tests

```typescript
import request from 'supertest';
import { DataProviderAPI } from './api/server';

describe('API Tests', () => {
  let api: DataProviderAPI;

  beforeAll(() => {
    api = new DataProviderAPI();
    // Register test provider
  });

  test('GET /api/v1/data/current', async () => {
    const response = await request(api.getApp())
      .get('/api/v1/data/current')
      .expect(200);
    
    expect(response.body.data).toBeDefined();
  });
});
```

## Support

For issues or questions:
1. Check the OpenAPI specification
2. Review the example server code
3. Consult the main backend README

## Next Steps

1. Configure Google Sheets credentials
2. Choose production document store
3. Add authentication/authorization
4. Implement rate limiting
5. Set up monitoring and logging
6. Deploy to production environment
