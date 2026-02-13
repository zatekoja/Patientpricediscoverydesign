# Coding Agents Guide

Use this file as the entry point for autonomous coding agents.

## Start Here

1. Read `docs/README.md`
2. Read `docs/guides/CODING_AGENTS_GUIDE.md`
3. Follow `docs/guides/CODING_STANDARDS.md`

## Scope Map

- Frontend app: `Frontend/` (React + Vite)
- Backend API: `backend/` (Go)
- TypeScript provider subsystem: `backend/` (`*.ts` + `backend/api/`)

## Mandatory Quality Gates

- Frontend: `npm run build`
- Backend: `go vet ./...` and `go test -race ./...` from `backend/`
- Formatting:
  - Frontend/TS: keep existing formatter conventions
  - Backend/Go: `gofmt -w` on changed files

## Mocking Rule

Do not hand-write new Go mocks when interface mocks are needed.
Use `mockery` with repo config in `backend/.mockery.yml` (or v3 config if required).

