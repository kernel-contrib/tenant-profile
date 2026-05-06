package tenantprofile

import (
	"context"

	"github.com/google/uuid"
	"github.com/kernel-contrib/tenant-profile/internal"
	"github.com/kernel-contrib/tenant-profile/types"
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
// The embedded moduleWriter provides write operations so a single
// RegisterReader call satisfies both TenantProfileReader and
// TenantProfileWriter via Go's implicit composition.
type moduleReader struct {
	*moduleWriter
	repo *internal.Repository
}

func (r *moduleReader) GetProfile(ctx context.Context, tenantID uuid.UUID) (*types.TenantProfile, error) {
	return r.repo.FindByTenantID(ctx, tenantID)
}
