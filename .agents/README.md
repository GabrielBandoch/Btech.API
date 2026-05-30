# BTech Platform — AI Agent System

This directory contains specialized AI agents for the **BTech Fleet Control Platform**. Each agent is a focused guidance document designed to give AI coding assistants (and human developers) deep, project-specific knowledge without reading the full Obsidian documentation.

---

## Why Agents Exist

BTech is a multi-tenant SaaS fleet management platform with strict architectural and security requirements:

- **Clean Architecture** — every layer has a specific responsibility and must not bleed into adjacent layers.
- **Multi-Tenancy** — every data operation is scoped to an `organization_id`. A bug here is a security incident.
- **RBAC + Audit** — all sensitive actions require explicit permission checks and must generate audit log events.
- **API Consistency** — endpoints must behave uniformly across all features (response shape, error codes, pagination).

The agent system encodes these constraints so that the **first implementation is correct**, not the third.

---

## How to Use These Agents

Reference an agent at the beginning of a task, a PR review, or before writing any new code:

> "As the Backend Architect agent, review this new `NotificationUseCase` before I implement it."
> "Apply the Security Guardian checklist to this new endpoint."

Each agent can be used as:
- A **pre-implementation prompt** to shape your approach
- A **review checklist** before opening a PR
- A **debugging guide** when something behaves unexpectedly

---

## Agent Directory

| Agent | File | Primary Concern |
|---|---|---|
| Backend Architect | `backend-architect.md` | Clean Architecture, layer boundaries, DI |
| Security & Multi-Tenant Guardian | `security-multitenant-guardian.md` | Tenant isolation, RBAC, JWT, Audit |
| API Reviewer | `api-reviewer.md` | REST conventions, DTOs, response shape |
| PostgreSQL Guardian | `postgresql-guardian.md` | Migrations, indexes, constraints, soft delete |
| Frontend Architect | `frontend-architect.md` | Page structure, components, state |
| UX Consistency Guardian | `ux-consistency-guardian.md` | Design patterns, forms, tables, modals |
| Analytics Specialist | `analytics-specialist.md` | KPIs, charts, dashboards, aggregations |
| Integration Reviewer | `integration-reviewer.md` | API contracts, React Query, error handling |

---

## Recommended Workflow

### Backend Feature

```
1. Backend Architect      → Validate domain entities, use case contracts, DI wiring
2. Security Guardian      → Verify tenant isolation, permissions, and audit events
3. PostgreSQL Guardian    → Review migration, indexes, and constraints
4. API Reviewer           → Validate route naming, DTOs, and response structure
```

### Frontend Feature

```
1. Frontend Architect     → Validate page structure, folder placement, query architecture
2. UX Consistency Guardian → Verify design pattern compliance
3. Integration Reviewer   → Validate API consumption, cache keys, error handling
4. Analytics Specialist   → Review any metrics, KPIs, or chart logic
```

### Full-Stack Feature

Run both workflows in order. Resolve backend issues before starting frontend integration.

---

## Source of Truth

These agents are **derived from** but do not replace the Obsidian documentation. When an agent conflicts with Obsidian, Obsidian wins. Update these agents whenever architecture decisions change.

---

## Module Reference

```
BTech.API (Go 1.25, chi v5, pgx v5, goose, JWT HS256)

internal/
  domain/         ← Entities, repository interfaces, domain errors, constants
  usecase/        ← Business logic, orchestration, audit calls
  repository/
    postgres/     ← SQL implementations of repository interfaces
  delivery/
    http/
      handler/    ← HTTP handlers (thin, no business logic)
      middleware/ ← Auth, RBAC, billing entitlements, rate limiting
      dto/        ← Request/Response structs and mappers
      response/   ← Standardized JSON envelope helpers
  platform/
    security/     ← JWT generation/validation, bcrypt helpers
    database/     ← DB connection pool setup
    logger/       ← Structured slog logger
  config/         ← Environment config loader

migrations/       ← Goose SQL migrations (numbered, sequential)
cmd/api/          ← main.go — wires all dependencies, starts server
```
