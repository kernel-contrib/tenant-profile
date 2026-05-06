package tenantprofile

import (
	"context"

	"github.com/edgescaleDev/kernel/sdk"
	"github.com/google/uuid"
	"github.com/kernel-contrib/tenant-profile/internal"
	"github.com/kernel-contrib/tenant-profile/types"
)

// ── Writer interface ──────────────────────────────────────────────────────────

// TenantProfileWriter is the cross-module write interface.
// Other modules consume this via:
//
//	writer, err := sdk.Reader[tenantprofile.TenantProfileWriter](&m.ctx, "tenant_profile")
//
// Rules:
//   - Resolve lazily in handlers, NEVER in Init().
type TenantProfileWriter interface {
	CreateProfile(ctx context.Context, in CreateProfileInput) (*types.TenantProfile, error)
	UpdateProfile(ctx context.Context, tenantID uuid.UUID, in UpdateProfileInput) (*types.TenantProfile, error)
}

// Re-export input types so consumers don't need to import internal.
type CreateProfileInput = internal.CreateProfileInput
type UpdateProfileInput = internal.UpdateProfileInput

// ── Implementation ────────────────────────────────────────────────────────────

// moduleWriter wraps the internal service for cross-module write access.
// It is embedded in the moduleReader struct so a single RegisterReader call
// exposes both TenantProfileReader and TenantProfileWriter.
type moduleWriter struct {
	svc *internal.Service
}

func (w *moduleWriter) CreateProfile(ctx context.Context, in CreateProfileInput) (*types.TenantProfile, error) {
	return w.svc.Create(ctx, in)
}

func (w *moduleWriter) UpdateProfile(ctx context.Context, tenantID uuid.UUID, in UpdateProfileInput) (*types.TenantProfile, error) {
	// Ensure a profile exists before updating. If the event-based creation
	// hasn't fired yet, seed an empty one first.
	if _, err := w.svc.GetByTenantID(ctx, tenantID); err != nil {
		if se, ok := sdk.IsServiceError(err); ok && se.HTTPStatus == 404 {
			if _, createErr := w.svc.Create(ctx, CreateProfileInput{TenantID: tenantID}); createErr != nil {
				// Ignore conflict (another goroutine created it).
				if se2, ok2 := sdk.IsServiceError(createErr); !ok2 || se2.HTTPStatus != 409 {
					return nil, createErr
				}
			}
		} else {
			return nil, err
		}
	}
	return w.svc.Update(ctx, tenantID, in)
}
