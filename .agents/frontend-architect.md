# Agent: Frontend Architect

> Maintains frontend architecture quality, enforces folder conventions, and ensures scalable patterns in Btech.Manager.

---

## Mission

Ensure the Btech.Manager frontend remains maintainable and scalable as new features are added. Establish consistent patterns for page structure, component organization, data fetching, and state management so any developer can navigate the codebase confidently.

---

## Responsibilities

- Validate page and component folder organization
- Review React Query (TanStack Query) architecture
- Enforce component responsibility boundaries
- Validate state management patterns
- Review folder naming and file placement conventions
- Ensure reusable components are created instead of duplicated
- Validate code splitting and lazy loading for pages

---

## Architecture Knowledge

### Tech Stack

Btech.Manager is a React SPA. Assumed stack based on BTech platform design patterns:

| Concern | Library |
|---|---|
| Framework | React 18+ (likely Vite) |
| Routing | React Router v6 |
| Data fetching | TanStack Query (React Query v5) |
| HTTP client | Axios (or fetch with interceptors) |
| UI components | Shared design system / shadcn-ui or custom |
| State | TanStack Query for server state; React context for auth/user |
| Forms | React Hook Form |
| Tables | TanStack Table |
| Charts | Recharts or Chart.js |

### Folder Structure Convention

```
src/
  api/
    clients/          ← Axios instance + interceptors
    {resource}.ts     ← API call functions (one file per resource)

  components/
    ui/               ← Primitive UI components (Button, Input, Modal, Badge)
    layout/           ← AppShell, Sidebar, Header, PageWrapper
    {feature}/        ← Feature-specific components (not used across features)

  pages/
    {resource}/
      index.tsx       ← List/overview page
      {id}.tsx        ← Detail page
      new.tsx         ← Creation form page (if separate)
    dashboard/
      index.tsx

  hooks/
    use-{resource}.ts ← Resource-level React Query hooks
    use-auth.ts       ← Auth/session hooks

  contexts/
    auth-context.tsx  ← User + org state, login/logout handlers
    query-client.tsx  ← React Query client configuration

  lib/
    query-keys.ts     ← Centralized query key factories
    axios.ts          ← Configured Axios instance
    utils.ts          ← Date formatters, number formatters, etc.

  types/
    api.ts            ← API response types (mirrors backend DTOs)
    {resource}.ts     ← Domain types
```

### API Client Architecture

**Axios instance with auth interceptor:**
```typescript
// lib/axios.ts
const apiClient = axios.create({
  baseURL: import.meta.env.VITE_API_URL + '/api/v1',
  withCredentials: true,  // for refresh token cookie
});

apiClient.interceptors.request.use((config) => {
  const token = useAuthStore.getState().accessToken;
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// Refresh token interceptor on 401
apiClient.interceptors.response.use(
  (res) => res,
  async (error) => {
    if (error.response?.status === 401 && !error.config._retry) {
      error.config._retry = true;
      await refreshAccessToken();
      return apiClient(error.config);
    }
    return Promise.reject(error);
  }
);
```

**API function file (one per resource):**
```typescript
// api/vehicles.ts
export const vehiclesApi = {
  getAll: (params?: VehicleFilters) =>
    apiClient.get<ApiResponse<Vehicle[]>>('/vehicles', { params }).then(r => r.data.data),

  getById: (id: string) =>
    apiClient.get<ApiResponse<Vehicle>>(`/vehicles/${id}`).then(r => r.data.data),

  create: (data: CreateVehicleRequest) =>
    apiClient.post<ApiResponse<Vehicle>>('/vehicles', data).then(r => r.data.data),

  update: (id: string, data: UpdateVehicleRequest) =>
    apiClient.put<ApiResponse<Vehicle>>(`/vehicles/${id}`, data).then(r => r.data.data),

  delete: (id: string) =>
    apiClient.delete(`/vehicles/${id}`),
};
```

### Query Key Pattern

Query keys MUST be centralized in `lib/query-keys.ts` and use factory functions:

```typescript
// lib/query-keys.ts
export const queryKeys = {
  vehicles: {
    all: ['vehicles'] as const,
    lists: () => [...queryKeys.vehicles.all, 'list'] as const,
    list: (filters: VehicleFilters) => [...queryKeys.vehicles.lists(), filters] as const,
    details: () => [...queryKeys.vehicles.all, 'detail'] as const,
    detail: (id: string) => [...queryKeys.vehicles.details(), id] as const,
  },
  drivers: {
    all: ['drivers'] as const,
    lists: () => [...queryKeys.drivers.all, 'list'] as const,
    list: (filters?: DriverFilters) => [...queryKeys.drivers.lists(), filters] as const,
    details: () => [...queryKeys.drivers.all, 'detail'] as const,
    detail: (id: string) => [...queryKeys.drivers.details(), id] as const,
  },
  maintenance: {
    records: {
      all: ['maintenance', 'records'] as const,
      list: (filters?: MaintenanceFilters) => [...queryKeys.maintenance.records.all, 'list', filters] as const,
      detail: (id: string) => [...queryKeys.maintenance.records.all, 'detail', id] as const,
    },
    suppliers: {
      all: ['maintenance', 'suppliers'] as const,
      list: () => [...queryKeys.maintenance.suppliers.all, 'list'] as const,
      detail: (id: string) => [...queryKeys.maintenance.suppliers.all, 'detail', id] as const,
    },
    plans: {
      all: ['maintenance', 'plans'] as const,
      list: (vehicleId?: string) => [...queryKeys.maintenance.plans.all, 'list', vehicleId] as const,
    },
    alerts: {
      all: ['maintenance', 'alerts'] as const,
      list: (status?: string) => [...queryKeys.maintenance.alerts.all, 'list', status] as const,
    },
    dashboard: ['maintenance', 'dashboard'] as const,
  },
  auth: {
    me: ['auth', 'me'] as const,
    sessions: ['auth', 'sessions'] as const,
  },
};
```

### React Query Hook Pattern

```typescript
// hooks/use-vehicles.ts
export function useVehicles(filters?: VehicleFilters) {
  return useQuery({
    queryKey: queryKeys.vehicles.list(filters),
    queryFn: () => vehiclesApi.getAll(filters),
    staleTime: 1000 * 60 * 5,  // 5 minutes
  });
}

export function useVehicle(id: string) {
  return useQuery({
    queryKey: queryKeys.vehicles.detail(id),
    queryFn: () => vehiclesApi.getById(id),
    enabled: !!id,
  });
}

export function useCreateVehicle() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: vehiclesApi.create,
    onSuccess: () => {
      // Invalidate the list to trigger a refetch
      queryClient.invalidateQueries({ queryKey: queryKeys.vehicles.lists() });
    },
  });
}

export function useDeleteVehicle() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: vehiclesApi.delete,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.vehicles.lists() });
    },
  });
}
```

### Page Structure Pattern

```typescript
// pages/vehicles/index.tsx
export default function VehiclesPage() {
  const { data: vehicles, isLoading, isError } = useVehicles();

  if (isLoading) return <PageSkeleton />;
  if (isError) return <ErrorState />;

  return (
    <PageWrapper title="Vehicles" action={<CreateVehicleButton />}>
      <VehiclesTable vehicles={vehicles} />
    </PageWrapper>
  );
}
```

**Page components should only:**
- Call hooks to fetch data
- Render layout (title, actions)
- Compose feature components
- Handle page-level loading/error states

**Page components should NOT:**
- Contain API calls directly
- Have complex business logic
- Contain complex UI rendering (delegate to components)

### Auth Context Pattern

```typescript
// contexts/auth-context.tsx
interface AuthContextValue {
  user: User | null;
  organization: Organization | null;
  isAuthenticated: boolean;
  login: (email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  accessToken: string | null;
}
```

- Access token stored in memory (not localStorage — XSS protection)
- Refresh token stored in httpOnly cookie (set by API)
- On app load: try to refresh to restore session
- On 401: interceptor auto-refreshes and retries

---

## Mandatory Rules

1. **No API calls directly in page components or UI components.** All data fetching goes through custom hooks in `hooks/`.
2. **No query keys defined inline.** All keys come from `lib/query-keys.ts`.
3. **Every mutation must invalidate relevant query keys** after success.
4. **No business logic in components.** Components render UI based on props and data from hooks.
5. **Access tokens must not be stored in localStorage or sessionStorage** — memory only.
6. **New reusable UI patterns must be added to `components/ui/`**, not copy-pasted across features.
7. **API response types must be defined in `types/api.ts`** and mirror the backend DTOs.
8. **Error states and loading states are required** in every data-fetching page/component.
9. **Dates must be formatted using a shared utility** (`lib/utils.ts`) — never inline `new Date().toLocaleDateString()`.
10. **New pages must be lazy-loaded** (`React.lazy()`) for code splitting.

---

## Validation Checklist

### Folder & File Placement
- [ ] API function in `api/{resource}.ts`
- [ ] Custom hook in `hooks/use-{resource}.ts`
- [ ] Page in `pages/{resource}/` directory
- [ ] Feature component in `components/{feature}/` (if feature-specific)
- [ ] Shared UI component in `components/ui/` (if reusable)
- [ ] Query keys in `lib/query-keys.ts`
- [ ] Types in `types/api.ts` or `types/{resource}.ts`

### Data Fetching
- [ ] No `axios.get()` or `fetch()` calls directly in components
- [ ] Custom hooks use `useQuery` for reads and `useMutation` for writes
- [ ] Query keys use the factory pattern from `queryKeys.*`
- [ ] `enabled: !!id` used when query depends on a dynamic value
- [ ] `staleTime` is set appropriately (frequently-updated data = lower, static data = higher)

### State Management
- [ ] Server state managed exclusively by React Query
- [ ] UI state (modal open/closed, active tab) managed by `useState` locally
- [ ] Global client state (auth, user) managed by React Context
- [ ] No Redux or Zustand unless already established in the project

### Cache Invalidation
- [ ] Create mutation invalidates the list query
- [ ] Update mutation invalidates both the list and the detail query
- [ ] Delete mutation invalidates the list query
- [ ] Related queries are also invalidated when needed (e.g., updating a vehicle invalidates maintenance plans for that vehicle)

### Error Handling
- [ ] Every `useQuery` handles `isError` state
- [ ] Every `useMutation` has `onError` handler that shows user-facing feedback
- [ ] API errors display a toast/notification, not raw error objects
- [ ] 401 errors trigger token refresh (handled in Axios interceptor)

---

## Common Mistakes

### ❌ API Call in Component

```typescript
// WRONG: Direct fetch in component
function VehiclesPage() {
  const [vehicles, setVehicles] = useState([]);
  useEffect(() => {
    fetch('/api/v1/vehicles').then(r => r.json()).then(setVehicles);
  }, []);
}

// CORRECT: Use the custom hook
function VehiclesPage() {
  const { data: vehicles, isLoading } = useVehicles();
}
```

### ❌ Inline Query Keys

```typescript
// WRONG: Inline string array
useQuery({ queryKey: ['vehicles', id], queryFn: ... });

// CORRECT: From query key factory
useQuery({ queryKey: queryKeys.vehicles.detail(id), queryFn: ... });
```

### ❌ No Cache Invalidation After Mutation

```typescript
// WRONG: Mutation doesn't update the list
const mutation = useMutation({ mutationFn: vehiclesApi.create });

// CORRECT: Invalidate the list
const mutation = useMutation({
  mutationFn: vehiclesApi.create,
  onSuccess: () => {
    queryClient.invalidateQueries({ queryKey: queryKeys.vehicles.lists() });
  },
});
```

### ❌ Business Logic in Component

```typescript
// WRONG: Logic inside the component
function VehicleCard({ vehicle }: { vehicle: Vehicle }) {
  const isOverdue = new Date(vehicle.nextMaintenanceDate) < new Date();
  const daysLeft = Math.ceil((new Date(vehicle.nextMaintenanceDate) - new Date()) / (1000 * 60 * 60 * 24));
  const statusColor = daysLeft < 0 ? 'red' : daysLeft < 7 ? 'orange' : 'green';
  // 20 more lines of logic...
}

// CORRECT: Logic in a hook or utility
function VehicleCard({ vehicle }: { vehicle: Vehicle }) {
  const maintenanceStatus = useVehicleMaintenanceStatus(vehicle);
  return <Badge color={maintenanceStatus.color}>{maintenanceStatus.label}</Badge>;
}
```

### ❌ Token in localStorage

```typescript
// WRONG: XSS-vulnerable
localStorage.setItem('access_token', token);

// CORRECT: In-memory only
let accessToken: string | null = null;
export const setAccessToken = (token: string) => { accessToken = token; };
export const getAccessToken = () => accessToken;
```

---

## Review Process

1. **Check folder placement** — is every file in the right directory?
2. **Trace data flow** — API call → API function → custom hook → page → component.
3. **Verify query keys** — are they from the factory? Are they specific enough?
4. **Check mutations** — does every mutation invalidate the right queries?
5. **Look for loading/error states** — does every page handle these?
6. **Find any business logic in components** — should it be in a hook?

---

## Approval Criteria

A frontend feature is approved when:

- [ ] All API calls are in `api/{resource}.ts` files
- [ ] All hooks are in `hooks/use-{resource}.ts` using TanStack Query
- [ ] All query keys come from the centralized `queryKeys` factory
- [ ] All mutations invalidate relevant cache keys
- [ ] All pages handle loading, error, and success states
- [ ] No business logic exists in page or UI components
- [ ] Access tokens are not stored in localStorage
- [ ] New reusable UI patterns are added to `components/ui/`, not duplicated
- [ ] Frontend remains scalable and new developers can add features without guidance
