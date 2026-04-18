package tenantprofile

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"go.edgescale.dev/kernel-contrib/tenant-profile/internal"
	"go.edgescale.dev/kernel/sdk"
)

// RegisterHooks subscribes to kernel lifecycle hooks.
func (m *Module) RegisterHooks(hooks *sdk.HookRegistry) {
	hooks.After("after.kernel.tenant.provisioned", m.onTenantProvisioned)
}

// ── Hook handlers ─────────────────────────────────────────────────────────────

// onTenantProvisioned creates an empty profile record when a tenant is provisioned.
func (m *Module) onTenantProvisioned(ctx context.Context, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("tenant_profile: marshal hook payload: %w", err)
	}

	var p struct {
		TenantID uuid.UUID `json:"tenant_id"`
		Address  sdk.JSONB `json:"address"`
		Industry *string   `json:"industry"`
		Size     *string   `json:"size"`
		Metadata sdk.JSONB `json:"metadata"`
	}
	if err := json.Unmarshal(data, &p); err != nil {
		return fmt.Errorf("tenant_profile: unmarshal hook payload: %w", err)
	}

	if p.TenantID == uuid.Nil {
		return fmt.Errorf("tenant_profile: tenant.provisioned hook: missing tenant_id")
	}

	m.ctx.Logger.Info("creating profile for new tenant", "tenant_id", p.TenantID)

	_, err = m.svc.Create(ctx, internal.CreateProfileInput{
		TenantID: p.TenantID,
		Address:  p.Address,
		Industry: p.Industry,
		Size:     p.Size,
		Metadata: p.Metadata,
	})
	if err != nil {
		if se, ok := sdk.IsServiceError(err); ok && se.HTTPStatus == 409 {
			m.ctx.Logger.Info("tenant profile already exists, skipping", "tenant_id", p.TenantID)
			return nil
		}
		return fmt.Errorf("tenant_profile: seed profile on provision: %w", err)
	}

	return nil
}
