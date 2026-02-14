
# Patient Price Discovery Design

Patient Price Discovery is a healthcare price transparency platform for discovering facilities, services, and pricing with search, suggestions, booking flows, and feedback capture.

Design source:
https://www.figma.com/design/MZo7gfRAshN9fTb5W6bh5o/Patient-Price-Discovery-Design

## The Nigerian Healthcare Dilemma

### The Challenge

From a bustling Lagos market to a quiet, distant village in Kaduna, a universal worry haunts us all: access to reliable healthcare. For too long, Nigerians have navigated a health landscape riddled with uncertainty. The search for genuine medication often feels like a gamble, with a fear of counterfeit drugs lurking on pharmacy shelves. Finding a specialist means endless referrals and frustrating trips across town, only to face a waitlist and unexpected bills. When an emergency strikes, the critical question isn't just "Can I get to a hospital?" but "Which hospital can I trust, and can I afford it?"

This is the daily burden carried by millions:

- The mother struggling to find the exact, approved vaccine for her child.
- The elderly parent needing consistent, affordable chronic medication.
- The young professional trying to compare the cost of a necessary surgery without being extorted.

These are not just inconveniences; they are life-and-death decisions made without the benefit of clear, reliable information.

### Why We Built Open Health Initiative (OHI/OH!)

We understand this struggle because we are part of it. That is why we built the Open Health Initiative (OHI/OH!), a digital lifeline designed to bring clarity, trust, and affordability back to Nigerian healthcare.

Imagine a world where the power to verify, compare, and connect to legitimate health services is simply a tap away. A world where you know the price of your medication before you leave home and the quality of your hospital before you arrive.

**OH! is your personal, pocket-sized health navigator** with a transparent, verified network of medications, hospital services, and their true costs. This isn't just an app; it's the trusted, 'surest' path to better health for every Nigerian.

## Release

- Current stable tag: `V1`

## Repository Layout

- `Frontend/`: React + Vite web app
- `backend/`: Go API, migrations, ingestion, and tests
- `docker-compose.yml`: local infrastructure orchestration

Note: `Frontend/` and `frontend/` point to the same directory in this repository.

## Quick Start

### 1. Frontend

From the repository root:

```bash
npm install
npm run dev
```

Build check:

```bash
npm run build
```

### 2. Backend

```bash
cd backend
go mod download
go run cmd/api/main.go
```

Backend tests:

```bash
go test ./...
go vet ./...
```

## Core V1 Capabilities

- Facility and service search
- Service suggestions with "Book Now" flow
- Facility modal with selectable services
- Feedback submission endpoint and UI tab
- Search analytics and zero-result query tracking

## Additional Documentation

- Docs index: `docs/README.md`
- Coding agents guide: `AGENTS.md`
- Backend deep dive: `backend/README.md`
  
