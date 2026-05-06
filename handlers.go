package tenantprofile

import (
	"encoding/json"

	"github.com/edgescaleDev/kernel/sdk"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kernel-contrib/tenant-profile/internal"
)

// ── Request types ─────────────────────────────────────────────────────────────

type updateProfileRequest struct {
	Address  *sdk.JSONB      `json:"address"`
	Industry *string         `json:"industry"`
	Size     *string         `json:"size"`
	Metadata json.RawMessage `json:"metadata"`
}

// ── Handlers ──────────────────────────────────────────────────────────────────

// handleGetProfile returns the business profile for the current tenant.
func (m *Module) handleGetProfile(c *gin.Context) {
	tid := c.MustGet("tenant_id").(uuid.UUID)

	profile, err := m.svc.GetByTenantID(c.Request.Context(), tid)
	if err != nil {
		sdk.FromError(c, err)
		return
	}

	sdk.OK(c, profile)
}

// handleUpdateProfile patches the business profile for the current tenant.
func (m *Module) handleUpdateProfile(c *gin.Context) {
	tid := c.MustGet("tenant_id").(uuid.UUID)

	var req updateProfileRequest
	if !sdk.BindAndValidate(c, &req) {
		return
	}

	in := internal.UpdateProfileInput{
		Address:  req.Address,
		Industry: req.Industry,
		Size:     req.Size,
	}

	// Metadata requires a pointer to sdk.JSONB; convert if present.
	if req.Metadata != nil {
		meta := sdk.JSONB(req.Metadata)
		in.Metadata = &meta
	}

	profile, err := m.svc.Update(c.Request.Context(), tid, in)
	if err != nil {
		sdk.FromError(c, err)
		return
	}

	m.ctx.Audit.Log(c.Request.Context(), sdk.AuditEntry{
		Action:     sdk.AuditUpdate,
		Resource:   "tenant_profile",
		ResourceID: tid.String(),
	})

	sdk.OK(c, profile)
}
