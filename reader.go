package tenantprofile

import (
	"context"

	"github.com/google/uuid"
	"go.edgescale.dev/kernel-contrib/tenant-profile/internal"
	"go.edgescale.dev/kernel-contrib/tenant-profile/types"
)

// ── Reader interface ──────────────────────────────────────────────────────────

// TenantProfileReader is the cross-module reader interface.
// Other modules consume this via:
//
//	reader, err := sdk.Reader[tenantprofile.TenantProfileReader](&m.ctx, "tenant_profile")
//
// Rules:
//   - All methods MUST be read-only (no writes, no events).
//   - Resolve readers lazily in handlers, NEVER in Init().
type TenantProfileReader interface {
	GetProfile(ctx context.Context, tenantID uuid.UUID) (*types.TenantProfile, error)
}

// ── Implementation ────────────────────────────────────────────────────────────

// moduleReader is the unexported implementation registered with the kernel.
type moduleReader struct {
	repo *internal.Repository
}

func (r *moduleReader) GetProfile(ctx context.Context, tenantID uuid.UUID) (*types.TenantProfile, error) {
	return r.repo.FindByTenantID(ctx, tenantID)
}
