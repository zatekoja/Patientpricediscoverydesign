# Coding Agents Navigation Guide

This guide helps coding agents make safe, consistent changes in this repository.

## Repository Topology

- `Frontend/`: user-facing React app (Vite build from repo root)
- `backend/`: Go backend + TypeScript provider code
- `.github/workflows/`: CI/CD pipelines
- `docs/`: active docs and archives

## Which Workflow Runs What

- Go backend CI: `.github/workflows/go-backend-ci.yml`
  - `go vet ./...`
  - `gofmt -l .` check
  - `go test -v -race ./...`
- Frontend CI: `.github/workflows/frontend-ci.yml`
  - optional type-check/lint (`--if-present`)
  - Vitest in `Frontend/src`
  - `npm run build`
- TypeScript provider CI: `.github/workflows/typescript-frontend-ci.yml`
  - `npm run build` in `backend/`
  - tests/lint if present

## Safe Change Flow for Agents

1. Identify changed area:
   - Frontend UI/search/modal -> `Frontend/src/app/`
   - Backend API/handler/service -> `backend/internal/`
   - Provider ingestion/parsing -> `backend/*.ts`, `backend/api/`, `backend/ingestion/`
2. Make minimal, scoped edits in the relevant layer.
3. Run only targeted checks first, then required full checks.
4. Update docs if behavior or conventions changed.

## Pre-Commit Commands

From repo root:

```bash
npm run build
```

From `backend/`:

```bash
gofmt -w .
go vet ./...
go test -race ./...
```

If `.golangci.yml`/`.golangci.yaml` exists in `backend/`, also run:

```bash
golangci-lint run --timeout=15m
```

## Mocking and Test Boundaries

- Use `mockery` for Go interface mocks; do not add ad-hoc/manual mock types when generated mocks are appropriate.
- Keep integration tests behind build tags (`//go:build integration` etc.) so unit pipelines remain deterministic.
- Keep tests close to changed package unless end-to-end behavior requires broader coverage.

## CI/Pipeline Editing Principles

- Keep pipelines simple and composable.
- Prefer reusable workflows over duplicated job logic.
- Do not add new checks unless they enforce a concrete standard already followed by the codebase.

