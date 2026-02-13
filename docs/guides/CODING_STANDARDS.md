# Coding Standards

This document defines repository-wide coding standards for humans and coding agents.

## 1. General Standards

- Prefer small, reviewable diffs.
- Keep behavior changes explicit in code and tests.
- Do not introduce dead files, duplicate flows, or parallel implementations.
- Update documentation links when files move.

## 2. Go Standards (`backend/`)

- Use `gofmt` formatting (`gofmt -w`).
- Keep `go vet ./...` clean.
- Keep tests race-safe (`go test -race ./...`).
- Follow package boundaries:
  - API layer: `backend/internal/api/`
  - Application layer: `backend/internal/application/`
  - Domain: `backend/internal/domain/`
  - Adapters/infrastructure: `backend/internal/adapters/`, `backend/internal/infrastructure/`

### Mocks

- Use `mockery` with repo config:
  - `backend/.mockery.yml`
  - `backend/.mockery_v3.yml` (where applicable)
- Avoid manually authored mocks when generated mocks can be used.

### Tests

- Unit tests: default `go test` paths.
- Integration tests must be guarded with build tags (`//go:build integration`, etc.).
- Keep flaky/external-service tests out of default unit paths.

## 3. Frontend Standards (`Frontend/`)

- Maintain TypeScript correctness and buildability (`npm run build` from repo root).
- Keep components focused and colocate UI logic with relevant component files.
- Reuse existing API client (`Frontend/src/lib/api.ts`) and type contracts (`Frontend/src/types/api.ts`).

## 4. TypeScript Provider Standards (`backend/*.ts`)

- Ensure `npm run build` in `backend/` remains green.
- Keep provider-specific behavior isolated under provider/ingestion modules.
- Preserve backward compatibility in API payload shapes when changing parser output.

## 5. CI and Pipeline Standards

- Respect existing workflows in `.github/workflows/`.
- Keep checks minimal and meaningful.
- Reuse workflow_call pipelines instead of duplicating logic.
- Validate any workflow change by confirming expected triggers and changed-path filters.

## 6. Documentation Standards

- Active docs belong in:
  - `README.md`
  - `docs/README.md`
  - scoped docs such as `backend/README.md`
- Historical/status/phase docs belong in `docs/archive/`.
- For new standards or agent guidance, update `AGENTS.md` and `docs/guides/`.

