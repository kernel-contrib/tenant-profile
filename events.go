package tenantprofile

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/edgescaleDev/kernel/sdk"
	"github.com/google/uuid"
	"github.com/kernel-contrib/tenant-profile/internal"
)

// RegisterEvents subscribes to async events from other modules.
func (m *Module) RegisterEvents(bus sdk.EventBus) {
	bus.Subscribe("tenant_profile", "iam.tenant.created", m.onTenantCreated)
}

// onTenantCreated seeds an empty profile when a new tenant is created via IAM.
// The profile can be updated later through the PATCH endpoint.
func (m *Module) onTenantCreated(ctx context.Context, env sdk.EventEnvelope) error {
	var payload struct {
		TenantID uuid.UUID `json:"tenant_id"`
	}
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		return fmt.Errorf("tenant_profile: unmarshal iam.tenant.created event: %w", err)
	}

	if payload.TenantID == uuid.Nil {
		return fmt.Errorf("tenant_profile: iam.tenant.created event: missing tenant_id")
	}

	m.ctx.Logger.Info("creating profile for new tenant (via iam.tenant.created)",
		"tenant_id", payload.TenantID,
	)

	_, err := m.svc.Create(ctx, internal.CreateProfileInput{
		TenantID: payload.TenantID,
	})
	if err != nil {
		// Profile already exists (idempotent). This can happen if both the
		// hook and event fire for the same tenant (e.g., CLI provisioning).
		if se, ok := sdk.IsServiceError(err); ok && se.HTTPStatus == 409 {
			m.ctx.Logger.Info("tenant profile already exists, skipping",
				"tenant_id", payload.TenantID,
			)
			return nil
		}
		return fmt.Errorf("tenant_profile: create profile on iam.tenant.created: %w", err)
	}

	return nil
}
