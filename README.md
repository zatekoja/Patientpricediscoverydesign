# Patient Price Discovery

A comprehensive healthcare price transparency platform that helps patients discover and compare healthcare service prices across facilities in Nigeria.

## Overview

This application provides:
- **Frontend**: React/Vite application with interactive maps and search
- **Backend**: Go-based microservices (REST API, GraphQL, SSE)
- **Database**: PostgreSQL with Redis caching
- **Search**: Typesense for fast, typo-tolerant search
- **Infrastructure**: Production-ready deployment on Google Cloud Platform

Original design: https://www.figma.com/design/MZo7gfRAshN9fTb5W6bh5o/Patient-Price-Discovery-Design

## Quick Start - Local Development

### Prerequisites
- Node.js 20+
- Go 1.25+
- Docker and Docker Compose
- PostgreSQL 15
- Redis 7

### Start Development Environment

```bash
# Install dependencies
npm install

# Start all services with Docker Compose
docker-compose up

# Access the application
# Frontend: http://localhost:3000
# API: http://localhost:8080
# GraphQL: http://localhost:8081
# SSE: http://localhost:8082
```

## Cloud Deployment (GCP)

### Production Infrastructure

This project includes complete Infrastructure as Code (IaC) for deploying to Google Cloud Platform.

**Quick Deploy**: See [INFRASTRUCTURE_SETUP.md](INFRASTRUCTURE_SETUP.md) for step-by-step guide.

### What's Included

✅ **Terraform Configuration**
- Cloud Run services (Frontend, API, GraphQL, SSE)
- Cloud SQL PostgreSQL (High Availability)
- Memorystore Redis (High Availability)
- Cloud DNS with custom domain
- Global HTTPS Load Balancer with SSL
- VPC networking and security

✅ **Deployment Automation**
- Build and push scripts for Docker images
- GitHub Actions CI/CD workflow
- One-command infrastructure deployment

✅ **Production Domain**
- Frontend: https://dev.ohealth-ng.com
- API: https://dev.api.ohealth-ng.com

### Deploy to GCP

```bash
# 1. Authenticate with GCP
gcloud auth login
gcloud config set project open-health-index-dev

# 2. Deploy infrastructure
./scripts/deploy.sh dev

# 3. Build and deploy applications
./scripts/build-and-push.sh
```

See full documentation:
- [Infrastructure Setup Guide](INFRASTRUCTURE_SETUP.md) - Quick start
- [GCP Deployment Guide](docs/GCP_DEPLOYMENT.md) - Comprehensive guide
- [Terraform Documentation](terraform/README.md) - Infrastructure details

## Project Structure

```
.
├── Frontend/                 # React frontend application
├── backend/                  # Go backend services
│   ├── cmd/                 # Service entry points
│   │   ├── api/            # REST API service
│   │   ├── graphql/        # GraphQL service
│   │   ├── sse/            # Server-Sent Events service
│   │   └── indexer/        # Search indexer
│   ├── internal/           # Internal packages
│   └── pkg/                # Public packages
├── terraform/              # Infrastructure as Code
│   ├── modules/           # Reusable Terraform modules
│   └── environments/      # Environment configurations
├── scripts/               # Deployment and build scripts
├── docs/                  # Documentation
└── docker-compose.yml    # Local development setup
```

## Technology Stack

### Frontend
- React 18
- Vite
- Material UI
- Google Maps API
- TypeScript

### Backend
- Go 1.25
- PostgreSQL 15
- Redis 7
- Typesense
- GraphQL
- Server-Sent Events

### Infrastructure
- Google Cloud Run
- Cloud SQL (PostgreSQL)
- Memorystore (Redis)
- Cloud DNS
- Cloud Load Balancer
- Container Registry

## Features

- 🔍 **Advanced Search**: Fast, typo-tolerant search powered by Typesense
- 🗺️ **Interactive Maps**: Google Maps integration for facility locations
- 📊 **Price Comparison**: Compare prices across multiple facilities
- 🔄 **Real-time Updates**: Server-Sent Events for live data
- 📱 **Responsive Design**: Mobile-first approach
- 🔐 **Secure**: VPC networking, SSL/TLS, Secret Manager
- 📈 **Scalable**: Auto-scaling Cloud Run services
- 🎯 **High Availability**: Regional databases with failover

## Documentation

- [Infrastructure Setup](INFRASTRUCTURE_SETUP.md) - Quick deployment guide
- [GCP Deployment](docs/GCP_DEPLOYMENT.md) - Comprehensive cloud deployment
- [Terraform Docs](terraform/README.md) - Infrastructure as Code details
- [API Documentation](backend/README.md) - Backend API reference
- [Architecture](backend/ARCHITECTURE.md) - System design and architecture

## Development

### Run Tests
```bash
# Frontend tests
npm test

# Backend tests
cd backend
go test ./...
```

### Build for Production
```bash
# Frontend
npm run build

# Backend
cd backend
go build ./cmd/api
go build ./cmd/graphql
go build ./cmd/sse
```

## Cost Estimation

Development environment: **$450-750/month**
- Cloud Run: $50-150
- Cloud SQL: $200-300
- Redis: $150-200
- Load Balancer: $20-50
- Networking: $30-50
- DNS: $1-5

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests
5. Submit a pull request

## License

[License information]

## Support

For issues or questions:
- Open a GitHub issue
- Check documentation in `docs/`
- Review Terraform logs for infrastructure issues