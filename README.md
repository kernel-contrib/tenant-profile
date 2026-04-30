# Tenant Profile

A headless [EdgeScale Kernel](https://go.edgescale.dev/kernel) module that stores extended business profile data for tenants — address, industry, company size, and arbitrary metadata. It exposes no HTTP endpoints of its own; consuming modules access profile data through the **`TenantProfileReader`** cross-module interface.

## Features

- **1:1 tenant extension** — Each tenant gets exactly one profile record, keyed by `tenant_id`.
- **Auto-provisioning** — An empty profile is seeded automatically when a tenant is provisioned via the `after.kernel.tenant.provisioned` hook.
- **Cross-module reader** — Other modules query profiles through a type-safe `TenantProfileReader` interface, no direct DB access required.
- **Event-driven** — Publishes `tenant_profile.profile.created` and `tenant_profile.profile.updated` events on mutation.
- **JSONB-native** — `address` and `metadata` fields use PostgreSQL `JSONB` for flexible, schema-less storage.

## Installation

```bash
go get github.com/kernel-contrib/tenant-profile
```

### Register with the kernel

```go
package main

import (
    tenantprofile "github.com/kernel-contrib/tenant-profile"
    "go.edgescale.dev/kernel"
)

func main() {
    k := kernel.New()
    k.Register(tenantprofile.New())
    // … register other modules …
    k.Start()
}
```

## Architecture

```
┌──────────────────────────────────────────────────────┐
│                    Kernel (SDK)                      │
│  ┌──────────┐  ┌──────────┐  ┌────────┐             │
│  │ EventBus │  │  Hooks   │  │   DB   │             │
│  └────┬─────┘  └────┬─────┘  └────┬───┘             │
│       │              │             │                 │
├───────┼──────────────┼─────────────┼─────────────────┤
│       ▼              ▼             ▼                 │
│  ┌─────────────────────────────────────────────────┐ │
│  │            tenant_profile (headless)            │ │
│  │                                                 │ │
│  │  hooks.go ──► internal/service ──► internal/repo│ │
│  │                   │                    │        │ │
│  │                   ├─ events ───────────┘        │ │
│  │                   ├─ validation                 │ │
│  │                   └─ business rules             │ │
│  │                                                 │ │
│  │  reader.go  ← cross-module queries              │ │
│  │  types/types.go ← shared domain model           │ │
│  └─────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────┘
```

> **No HTTP routes.** This module is headless. If you need REST endpoints for tenant profiles, build a thin handler module that delegates to the `TenantProfileReader` or calls the service directly.

## Database Schema

```sql
CREATE TABLE tenants_profile (
    tenant_id  UUID PRIMARY KEY,
    address    JSONB        NOT NULL DEFAULT '{}',
    industry   TEXT,
    size       TEXT,
    metadata   JSONB        NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_tenants_profile_industry
    ON tenants_profile(industry)
    WHERE deleted_at IS NULL;
```

## Domain Model

```go
type TenantProfile struct {
    TenantID  uuid.UUID      `json:"tenant_id"`
    Address   sdk.JSONB      `json:"address"`
    Industry  *string        `json:"industry,omitempty"`
    Size      *string        `json:"size,omitempty"`
    Metadata  sdk.JSONB      `json:"metadata,omitempty"`
    CreatedAt time.Time      `json:"created_at"`
    UpdatedAt time.Time      `json:"updated_at"`
    DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty"`
}
```

### Field Reference

| Field | Type | Description |
| --- | --- | --- |
| `tenant_id` | `UUID` | Primary key — matches the kernel tenant ID. |
| `address` | `JSONB` | Free-form address object (street, city, country, etc.). |
| `industry` | `TEXT` | Industry vertical (e.g. `"technology"`, `"healthcare"`). |
| `size` | `TEXT` | Company size bucket (e.g. `"1-10"`, `"50-200"`, `"500+"`). |
| `metadata` | `JSONB` | Arbitrary key-value metadata for extensibility. |
| `created_at` | `TIMESTAMPTZ` | Row creation timestamp. |
| `updated_at` | `TIMESTAMPTZ` | Last update timestamp. |
| `deleted_at` | `TIMESTAMPTZ` | Soft-delete marker. |

## Events

| Subject | Trigger | Payload |
| --- | --- | --- |
| `tenant_profile.profile.created` | A profile is inserted | `{"tenant_id": "<uuid>"}` |
| `tenant_profile.profile.updated` | A profile is patched | `{"tenant_id": "<uuid>"}` |

### Subscribing to events

```go
func (m *MyModule) Init(ctx sdk.Context) error {
    ctx.Bus.Subscribe("tenant_profile.profile.updated", func(ctx context.Context, payload any) {
        // React to profile changes (e.g. refresh a cache, send a notification).
    })
    return nil
}
```

## Lifecycle Hook

The module registers an `after.kernel.tenant.provisioned` hook that automatically creates an empty profile when a new tenant is provisioned. If the provisioning payload includes `address`, `industry`, `size`, or `metadata` fields, they will be populated on the seed record.

**Hook payload structure:**

```json
{
  "tenant_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "address": {
    "street": "123 Main St",
    "city": "San Francisco",
    "country": "US"
  },
  "industry": "technology",
  "size": "50-200",
  "metadata": {
    "plan": "enterprise"
  }
}
```

Only `tenant_id` is required. All other fields default to empty.

## Cross-Module Reader

Other modules consume tenant profiles through the `TenantProfileReader` interface:

```go
// TenantProfileReader is the cross-module reader interface.
type TenantProfileReader interface {
    GetProfile(ctx context.Context, tenantID uuid.UUID) (*types.TenantProfile, error)
}
```

### Usage in another module

```go
import (
    tenantprofile "github.com/kernel-contrib/tenant-profile"
    "github.com/kernel-contrib/tenant-profile/types"
)

func (h *handler) getCompanyInfo(c *gin.Context) {
    tenantID := sdk.TenantID(c) // extract from auth context

    reader, err := sdk.Reader[tenantprofile.TenantProfileReader](&h.ctx, "tenant_profile")
    if err != nil {
        sdk.FromError(c, err)
        return
    }

    profile, err := reader.GetProfile(c.Request.Context(), tenantID)
    if err != nil {
        sdk.FromError(c, err)
        return
    }

    sdk.OK(c, profile)
}
```

> **Important:** Always resolve readers lazily inside handlers, never in `Init()`.

## Request Examples

Since `tenant_profile` is a headless module, HTTP access requires a consuming module that exposes endpoints. The examples below assume a typical thin handler module that wraps the service layer — a common pattern in kernel applications.

### Get tenant profile

```bash
curl -s \
  -H "Authorization: Bearer <TOKEN>" \
  https://api.example.com/v1/tenant/profile | jq
```

**Response `200 OK`:**

```json
{
  "tenant_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "address": {
    "street": "123 Main St",
    "city": "San Francisco",
    "state": "CA",
    "zip": "94105",
    "country": "US"
  },
  "industry": "technology",
  "size": "50-200",
  "metadata": {
    "plan": "enterprise",
    "tax_id": "US-12345678"
  },
  "created_at": "2026-01-15T10:30:00Z",
  "updated_at": "2026-03-20T14:22:00Z"
}
```

**Response `404 Not Found` (no profile exists):**

```json
{
  "error": "tenant_profile not found",
  "code": 404
}
```

### Create tenant profile

```bash
curl -s -X POST \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "address": {
      "street": "456 Business Ave",
      "city": "Austin",
      "state": "TX",
      "zip": "73301",
      "country": "US"
    },
    "industry": "healthcare",
    "size": "200-500",
    "metadata": {
      "plan": "pro",
      "npi_number": "1234567890"
    }
  }' \
  https://api.example.com/v1/tenant/profile | jq
```

**Response `201 Created`:**

```json
{
  "tenant_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "address": {
    "street": "456 Business Ave",
    "city": "Austin",
    "state": "TX",
    "zip": "73301",
    "country": "US"
  },
  "industry": "healthcare",
  "size": "200-500",
  "metadata": {
    "plan": "pro",
    "npi_number": "1234567890"
  },
  "created_at": "2026-04-30T08:00:00Z",
  "updated_at": "2026-04-30T08:00:00Z"
}
```

**Response `409 Conflict` (profile already exists):**

```json
{
  "error": "tenant profile already exists",
  "code": 409
}
```

### Update tenant profile (partial patch)

Only the fields you include in the request body are updated — omitted fields are left untouched.

```bash
curl -s -X PATCH \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "industry": "fintech",
    "size": "500+",
    "metadata": {
      "plan": "enterprise",
      "tax_id": "US-87654321"
    }
  }' \
  https://api.example.com/v1/tenant/profile | jq
```

**Response `200 OK`:**

```json
{
  "tenant_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "address": {
    "street": "456 Business Ave",
    "city": "Austin",
    "state": "TX",
    "zip": "73301",
    "country": "US"
  },
  "industry": "fintech",
  "size": "500+",
  "metadata": {
    "plan": "enterprise",
    "tax_id": "US-87654321"
  },
  "created_at": "2026-04-30T08:00:00Z",
  "updated_at": "2026-04-30T09:15:00Z"
}
```

### Update only the address

```bash
curl -s -X PATCH \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "address": {
      "street": "789 New HQ Blvd",
      "city": "Denver",
      "state": "CO",
      "zip": "80201",
      "country": "US"
    }
  }' \
  https://api.example.com/v1/tenant/profile | jq
```

**Response `200 OK`:**

```json
{
  "tenant_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "address": {
    "street": "789 New HQ Blvd",
    "city": "Denver",
    "state": "CO",
    "zip": "80201",
    "country": "US"
  },
  "industry": "fintech",
  "size": "500+",
  "metadata": {
    "plan": "enterprise",
    "tax_id": "US-87654321"
  },
  "created_at": "2026-04-30T08:00:00Z",
  "updated_at": "2026-04-30T10:45:00Z"
}
```

## Project Structure

| File | Purpose |
| --- | --- |
| `module.go` | Module lifecycle — `Manifest()`, `Init()`, `Shutdown()` |
| `reader.go` | `TenantProfileReader` interface + implementation |
| `hooks.go` | `after.kernel.tenant.provisioned` hook handler |
| `types/types.go` | `TenantProfile` GORM model (shared, importable) |
| `internal/service.go` | Business logic, validation, event publishing |
| `internal/repository.go` | GORM data-access layer |
| `internal/helpers.go` | Shared utilities (`IsNotFoundErr`) |
| `migrations/` | PostgreSQL migration files |
| `module_test.go` | Unit tests with in-memory SQLite |

## Testing

```bash
go test -v ./...
```

Tests use an in-memory SQLite database and the SDK test harness:

```go
func TestProfileCreate(t *testing.T) {
    h := newTestHarness(t)
    ctx := context.Background()
    tenantID := uuid.New()

    industry := "technology"
    size := "50-200"

    profile, err := h.svc.Create(ctx, internal.CreateProfileInput{
        TenantID: tenantID,
        Industry: &industry,
        Size:     &size,
    })
    require.NoError(t, err)
    assert.Equal(t, tenantID, profile.TenantID)
    assert.Equal(t, "technology", *profile.Industry)

    // Verify event was published.
    events := h.bus().Events()
    require.Len(t, events, 1)
    assert.Equal(t, "tenant_profile.profile.created", events[0].Subject)
}
```

## Requirements

- Go 1.26+
- EdgeScale Kernel SDK v0.2.0+
