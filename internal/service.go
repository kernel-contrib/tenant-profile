package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/edgescaleDev/kernel/sdk"
	"github.com/google/uuid"
	"github.com/kernel-contrib/tenant-profile/types"
)

// Service provides business logic for tenant profile operations.
type Service struct {
	repo *Repository
	bus  sdk.EventBus
	log  *slog.Logger
}

// NewService constructs a Service.
func NewService(repo *Repository, bus sdk.EventBus, log *slog.Logger) *Service {
	return &Service{repo: repo, bus: bus, log: log}
}

// ── Query ─────────────────────────────────────────────────────────────────────

// GetByTenantID returns a tenant profile by tenant UUID.
func (s *Service) GetByTenantID(ctx context.Context, tenantID uuid.UUID) (*types.TenantProfile, error) {
	profile, err := s.repo.FindByTenantID(ctx, tenantID)
	if IsNotFoundErr(err) {
		return nil, sdk.NotFound("tenant_profile", tenantID)
	}
	return profile, err
}

// ── Mutations ─────────────────────────────────────────────────────────────────

// CreateProfileInput contains the fields for creating a new tenant profile.
type CreateProfileInput struct {
	TenantID uuid.UUID
	Address  sdk.JSONB
	Industry *string
	Size     *string
	Metadata sdk.JSONB
}

// Create inserts a new tenant profile. If one already exists, it returns a conflict error.
func (s *Service) Create(ctx context.Context, in CreateProfileInput) (*types.TenantProfile, error) {
	if in.TenantID == uuid.Nil {
		return nil, sdk.BadRequest("tenant_id is required")
	}

	// Check if a profile already exists.
	if existing, _ := s.repo.FindByTenantID(ctx, in.TenantID); existing != nil {
		return nil, sdk.Conflict("tenant profile already exists")
	}

	profile := &types.TenantProfile{
		TenantID: in.TenantID,
		Address:  in.Address,
		Industry: in.Industry,
		Size:     in.Size,
		Metadata: in.Metadata,
	}

	if err := s.repo.Upsert(ctx, profile); err != nil {
		return nil, fmt.Errorf("tenant_profile: create: %w", err)
	}

	s.publish(ctx, "tenant_profile.profile.created", map[string]any{
		"tenant_id": in.TenantID,
	})

	return profile, nil
}

// UpdateProfileInput is a partial update for tenant profile fields.
type UpdateProfileInput struct {
	Address  *sdk.JSONB
	Industry *string
	Size     *string
	Metadata *sdk.JSONB
}

// Update patches tenant profile fields and publishes tenant_profile.profile.updated.
func (s *Service) Update(ctx context.Context, tenantID uuid.UUID, in UpdateProfileInput) (*types.TenantProfile, error) {
	// Verify the profile exists and fetch it for metadata merging.
	existing, err := s.GetByTenantID(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	updates := make(map[string]any)
	if in.Address != nil {
		updates["address"] = *in.Address
	}
	if in.Industry != nil {
		updates["industry"] = *in.Industry
	}
	if in.Size != nil {
		updates["size"] = *in.Size
	}
	if in.Metadata != nil {
		// Merge incoming metadata keys into existing metadata so that
		// callers can update individual fields without wiping others.
		merged := mergeJSONB(existing.Metadata, *in.Metadata)
		updates["metadata"] = merged
	}
	if len(updates) == 0 {
		return s.repo.FindByTenantID(ctx, tenantID)
	}

	profile, err := s.repo.Update(ctx, tenantID, updates)
	if err != nil {
		return nil, err
	}

	s.publish(ctx, "tenant_profile.profile.updated", map[string]any{
		"tenant_id": tenantID,
	})

	return profile, nil
}

// ── internal ──────────────────────────────────────────────────────────────────

func (s *Service) publish(ctx context.Context, subject string, payload map[string]any) {
	if s.bus == nil {
		return
	}
	s.bus.Publish(ctx, subject, payload)
}

// mergeJSONB does a shallow merge of two JSONB values. Keys in patch
// overwrite keys in base; keys absent from patch are preserved.
// A null value in patch explicitly removes that key.
func mergeJSONB(base, patch sdk.JSONB) sdk.JSONB {
	var baseMap map[string]any
	if err := json.Unmarshal(base, &baseMap); err != nil || baseMap == nil {
		baseMap = make(map[string]any)
	}

	var patchMap map[string]any
	if err := json.Unmarshal(patch, &patchMap); err != nil {
		// If the patch isn't a valid JSON object, just replace entirely.
		return patch
	}

	for k, v := range patchMap {
		if v == nil {
			delete(baseMap, k)
		} else {
			baseMap[k] = v
		}
	}

	merged, _ := json.Marshal(baseMap)
	return sdk.JSONB(merged)
}
