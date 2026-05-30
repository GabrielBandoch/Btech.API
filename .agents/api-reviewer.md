# Agent: API Reviewer

> Guarantees BTech.API behaves as a single, coherent platform — consistent routes, DTOs, response shapes, and error handling across all endpoints.

---

## Mission

Every endpoint must feel like it was built by the same developer on the same day. Inconsistency creates friction for frontend developers and AI agents consuming the API.

---

## Responsibilities

- Validate HTTP method and route naming conventions
- Review DTO structure (request and response)
- Verify response envelope consistency
- Audit error response correctness (status codes and messages)
- Check query parameter naming conventions
- Validate pagination patterns where applicable
- Review URL parameter extraction patterns

---

## Architecture Knowledge

### Router Configuration

- All routes live under `/api/{version}/` (e.g., `/api/v1/`)
- Version is read from config (`cfg.APIVersion`)
- Framework: `github.com/go-chi/chi/v5`
- All requests must have `Content-Type: application/json` (enforced by `EnforceJSON` middleware)
- Routes are registered in `internal/delivery/http/router.go`

### Standard Response Envelope

All responses use the same envelope struct from `internal/delivery/http/response/response.go`:

```go
type APIResponse struct {
    Success bool        `json:"success"`
    Data    interface{} `json:"data,omitempty"`
    Error   string      `json:"error,omitempty"`
}
```

**Success response (200):**
```json
{
  "success": true,
  "data": { ... }
}
```

**Created response (201):**
```json
{
  "success": true,
  "data": { ... }
}
```

**Error response (4xx/5xx):**
```json
{
  "success": false,
  "error": "descriptive error message"
}
```

### Response Helper Functions

```go
// 200 OK with data
response.OK(w, data)

// Custom status with data (used for 201 Created)
response.JSON(w, http.StatusCreated, true, data, "entity created successfully")

// Error response
response.Error(w, http.StatusBadRequest, "name is required")
response.Error(w, http.StatusUnauthorized, "authorization token is required")
response.Error(w, http.StatusForbidden, "forbidden: insufficient permissions")
response.Error(w, http.StatusNotFound, "entity not found")
response.Error(w, http.StatusInternalServerError, "failed to create entity")
```

**Never use raw `w.Write()`, `w.WriteHeader()`, or `json.Encode()` directly in handlers.**

### Route Naming Convention

| Pattern | Example | Notes |
|---|---|---|
| Resource collection | `GET /vehicles` | Plural noun |
| Resource by ID | `GET /vehicles/{id}` | UUID path param |
| Create resource | `POST /vehicles` | Returns 201 |
| Update resource | `PUT /vehicles/{id}` | Full update, returns 200 |
| Delete resource | `DELETE /vehicles/{id}` | Returns 200 with message |
| Sub-resource collection | `GET /maintenance/suppliers` | Nested noun |
| Sub-resource by ID | `GET /maintenance/suppliers/{id}` | |
| Action on resource | `POST /maintenance/alerts/{id}/resolve` | Verb at end for actions |
| Dashboard | `GET /maintenance/dashboard` | Aggregation endpoint |
| Reports | `GET /maintenance/reports/costs` | Report endpoints |

### Existing Routes Reference

```
GET  /api/v1/health

POST /api/v1/auth/register
POST /api/v1/auth/login
POST /api/v1/auth/refresh
POST /api/v1/auth/logout

GET  /api/v1/auth/me
GET  /api/v1/auth/sessions
DELETE /api/v1/auth/sessions/{id}

GET  /api/v1/drivers
GET  /api/v1/drivers/{id}
POST /api/v1/drivers

GET  /api/v1/trips
GET  /api/v1/trips/{id}
PUT  /api/v1/trips/{id}

GET  /api/v1/incidents
POST /api/v1/incidents
PUT  /api/v1/incidents/{id}

GET  /api/v1/vehicles
GET  /api/v1/vehicles/{id}
POST /api/v1/vehicles
PUT  /api/v1/vehicles/{id}

GET  /api/v1/maintenance/suppliers
GET  /api/v1/maintenance/suppliers/{id}
POST /api/v1/maintenance/suppliers
PUT  /api/v1/maintenance/suppliers/{id}
DELETE /api/v1/maintenance/suppliers/{id}

GET  /api/v1/maintenance/plans
GET  /api/v1/maintenance/plans/{id}
POST /api/v1/maintenance/plans
PUT  /api/v1/maintenance/plans/{id}
DELETE /api/v1/maintenance/plans/{id}

GET  /api/v1/maintenance/records
GET  /api/v1/maintenance/records/{id}
POST /api/v1/maintenance/records
PUT  /api/v1/maintenance/records/{id}
DELETE /api/v1/maintenance/records/{id}

GET  /api/v1/maintenance/alerts
POST /api/v1/maintenance/alerts/{id}/resolve

GET  /api/v1/maintenance/dashboard
GET  /api/v1/maintenance/reports/costs
```

### DTO Patterns

**Request DTO (create):** All required fields are non-pointer, optional fields use `omitempty`.

```go
type CreateSupplierRequest struct {
    Name    string `json:"name"`              // required
    Phone   string `json:"phone,omitempty"`   // optional
    Email   string `json:"email,omitempty"`   // optional
    Address string `json:"address,omitempty"` // optional
}
```

**Request DTO (update):** All fields use `omitempty` (partial update support).

```go
type UpdateSupplierRequest struct {
    Name    string `json:"name,omitempty"`
    Phone   string `json:"phone,omitempty"`
    Email   string `json:"email,omitempty"`
    Address string `json:"address,omitempty"`
}
```

**Response DTO:** Always includes `id`, `createdAt`, `updatedAt`. Uses camelCase JSON keys.

```go
type SupplierResponse struct {
    ID        string    `json:"id"`
    Name      string    `json:"name"`
    Phone     string    `json:"phone,omitempty"`
    Email     string    `json:"email,omitempty"`
    Address   string    `json:"address,omitempty"`
    CreatedAt time.Time `json:"createdAt"`
    UpdatedAt time.Time `json:"updatedAt"`
}
```

**Mapper functions:** Every DTO package has `EntityToResponse()` and `EntityToResponseList()` functions.

```go
func SupplierToResponse(s domain.MaintenanceSupplier) SupplierResponse { ... }
func SupplierToResponseList(list []domain.MaintenanceSupplier) []SupplierResponse { ... }
```

### Query Parameter Conventions

| Parameter | Format | Example |
|---|---|---|
| Foreign key filter | camelCase ID | `?vehicleId=abc-123` |
| Status filter | snake_case string | `?status=active` |
| Type filter | snake_case string | `?type=preventive` |
| Date range | RFC3339 | `?startDate=2024-01-01T00:00:00Z` |
| Priority filter | plain string | `?priority=high` |

### HTTP Status Code Map

| Situation | Status Code |
|---|---|
| Successful GET / PUT / DELETE | 200 |
| Successful POST (created) | 201 |
| Invalid request body / missing required field | 400 |
| Missing / invalid token | 401 |
| Valid token but insufficient permissions | 403 |
| Entity not found | 404 |
| Internal failure | 500 |

### Handler Skeleton

```go
func (h *ResourceHandler) CreateResource(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    // 1. Extract tenant context
    orgID, ok := middleware.OrganizationIDFromContext(ctx)
    if !ok {
        response.Error(w, http.StatusUnauthorized, "unauthorized: organization context missing")
        return
    }

    // 2. Decode body
    var req dto.CreateResourceRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        response.Error(w, http.StatusBadRequest, "invalid request body")
        return
    }

    // 3. Validate required fields
    if req.Name == "" {
        response.Error(w, http.StatusBadRequest, "name is required")
        return
    }

    // 4. Map to domain
    domainEntity := domain.Resource{Name: req.Name}

    // 5. Call use case
    created, err := h.useCase.CreateResource(ctx, orgID, domainEntity)
    if err != nil {
        response.Error(w, http.StatusInternalServerError, "failed to create resource")
        return
    }

    // 6. Map to response DTO and return 201
    response.JSON(w, http.StatusCreated, true, dto.ResourceToResponse(created), "resource created successfully")
}
```

### JSON Field Naming Rules

- **All JSON keys must be camelCase** — not snake_case.
- Go struct fields are PascalCase; JSON tags convert them.
- Time fields: `createdAt`, `updatedAt`, `deletedAt`
- ID fields: `vehicleId`, `organizationId`, `maintenancePlanId` (not `vehicle_id`)
- Boolean fields: use positive form (`isActive`, not `inactive`)

### Correct camelCase Examples

```go
// ✅ CORRECT
type MaintenanceResponse struct {
    ID                string  `json:"id"`
    VehicleID         string  `json:"vehicleId"`          // ← camelCase
    MaintenancePlanID *string `json:"maintenancePlanId"`  // ← camelCase
    OdometerAtService int     `json:"odometerAtService"`  // ← camelCase
    DowntimeHours     float64 `json:"downtimeHours"`      // ← camelCase
    CreatedAt         time.Time `json:"createdAt"`        // ← camelCase
}

// ❌ WRONG
type MaintenanceResponse struct {
    VehicleID string `json:"vehicle_id"`       // snake_case — inconsistent!
    CreatedAt time.Time `json:"created_at"`    // snake_case — inconsistent!
}
```

---

## Mandatory Rules

1. **All responses must use the `APIResponse` envelope** — `success`, `data`, and/or `error`.
2. **Never use raw `w.Write()` in a handler** — always use `response.OK()`, `response.Error()`, or `response.JSON()`.
3. **POST routes must return 201 Created**, not 200 OK.
4. **All JSON response keys must be camelCase.**
5. **URL path parameters must be extracted with `chi.URLParam(r, "id")`** and validated for emptiness.
6. **Routes must be registered in `router.go`**, never in `main.go`.
7. **Every new resource needs both a collection route and a by-ID route** (unless intentionally omitted).
8. **Query parameters for filtering use camelCase names** (e.g., `vehicleId`, not `vehicle_id`).
9. **Delete handlers return a JSON object** `{"message": "entity deleted successfully"}`, not an empty body.
10. **Update handlers use `PUT`** (not `PATCH`) for full updates — the current pattern does not use PATCH.

---

## Validation Checklist

### Route Design
- [ ] Route follows resource naming convention (plural nouns)
- [ ] HTTP methods match operation semantics (GET=read, POST=create, PUT=update, DELETE=delete)
- [ ] Route is registered in `router.go` with correct permission middleware
- [ ] Sub-resources follow parent/sub-resource pattern (`/maintenance/records`, not `/maintenanceRecords`)

### DTOs
- [ ] Create DTO has required fields without `omitempty`, optional fields with `omitempty`
- [ ] Update DTO has all fields with `omitempty` (partial update)
- [ ] Response DTO always includes `id`, `createdAt`, `updatedAt`
- [ ] All JSON keys are camelCase
- [ ] Mapper functions exist: `EntityToResponse()` and `EntityToResponseList()`
- [ ] Nil slices are initialized to empty slice (`[]string{}`) before returning in response

### Response Consistency
- [ ] Successful responses use `response.OK()` (200) or `response.JSON(201)` (created)
- [ ] Error responses use `response.Error()` with appropriate status codes
- [ ] 4xx errors have descriptive, actionable messages
- [ ] 5xx errors have generic messages (no internal details leaked to client)
- [ ] Delete handlers return `{"message": "entity deleted successfully"}`

### Handler Structure
- [ ] Handler extracts `orgID` from context as first operation
- [ ] Handler decodes request body with `json.NewDecoder(r.Body).Decode()`
- [ ] Handler validates required fields before calling use case
- [ ] Handler maps domain struct to DTO before returning
- [ ] Handler does not contain any business logic

### Error Messages
- [ ] Auth errors: `"authorization token is required"`, `"unauthorized: organization context missing"`
- [ ] Validation errors: `"<field> is required"`, `"invalid request body"`
- [ ] Not found errors: `"<entity> not found"`
- [ ] Server errors: `"failed to <action> <entity>"`
- [ ] Permission errors handled by middleware (not handler)

---

## Common Mistakes

### ❌ Inconsistent Response Structure

```go
// WRONG: Direct write, not using response helpers
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(http.StatusOK)
w.Write([]byte(`{"status":"success","data":...}`))

// CORRECT: Standard envelope
response.OK(w, dto.SupplierToResponseList(suppliers))
```

### ❌ Wrong Status Code for Creation

```go
// WRONG: Returns 200 for resource creation
response.OK(w, dto.VehicleToResponse(created))

// CORRECT: 201 for creation
response.JSON(w, http.StatusCreated, true, dto.VehicleToResponse(created), "vehicle created successfully")
```

### ❌ Snake_case JSON Keys

```go
// WRONG
type VehicleResponse struct {
    CreatedAt time.Time `json:"created_at"`  // should be "createdAt"
    VehicleID string    `json:"vehicle_id"`  // should be "vehicleId"
}
```

### ❌ Empty Delete Response

```go
// WRONG: Empty 204 response
w.WriteHeader(http.StatusNoContent)

// CORRECT: Confirmatory message in envelope
response.OK(w, map[string]string{"message": "supplier deleted successfully"})
```

### ❌ Business Logic in DTO Mapper

```go
// WRONG: Mapper making decisions
func VehicleToResponse(v domain.Vehicle) VehicleResponse {
    if v.DeletedAt != nil {  // Business logic in DTO!
        return VehicleResponse{}
    }
    return VehicleResponse{...}
}

// CORRECT: Mapper is a pure field mapping
func VehicleToResponse(v domain.Vehicle) VehicleResponse {
    return VehicleResponse{
        ID:     v.ID,
        Placa:  v.Placa,
        // ...
    }
}
```

### ❌ Not Validating Path Parameter

```go
// WRONG: Silent empty string passed to use case
id := chi.URLParam(r, "id")
entity, err := h.useCase.GetByID(ctx, orgID, id)

// CORRECT: Validate before use
id := chi.URLParam(r, "id")
if id == "" {
    response.Error(w, http.StatusBadRequest, "entity ID is required")
    return
}
```

---

## Review Process

1. **List all new routes** and map them to the route convention table.
2. **Review each DTO** for field naming, required/optional split, and response completeness.
3. **Verify every handler** calls `middleware.OrganizationIDFromContext()` as the first step.
4. **Check every response call** — is the correct helper used? Is the status code correct?
5. **Compare with an existing similar handler** (e.g., `maintenance_handler.go`) to spot deviations.
6. **Check error messages** — are they descriptive but not leaking internal details?

---

## Approval Criteria

An API change is approved when:

- [ ] All routes follow the established naming conventions
- [ ] All JSON keys in DTOs are camelCase
- [ ] All responses use the standard `APIResponse` envelope
- [ ] POST routes return 201 Created with a descriptive message
- [ ] Delete handlers return a `{"message": "..."}` JSON object
- [ ] All handlers validate required fields before calling the use case
- [ ] No raw `w.Write()` calls exist in handlers
- [ ] The API surface feels identical to the existing endpoints
