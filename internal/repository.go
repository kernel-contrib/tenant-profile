package internal

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.edgescale.dev/kernel-contrib/tenant-profile/types"
	"gorm.io/gorm"
)

// Repository is the data-access layer for the tenant_profile module.
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a Repository backed by the provided *gorm.DB.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// FindByTenantID looks up a tenant profile by tenant UUID.
func (r *Repository) FindByTenantID(ctx context.Context, tenantID uuid.UUID) (*types.TenantProfile, error) {
	var profile types.TenantProfile
	if err := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID).First(&profile).Error; err != nil {
		return nil, fmt.Errorf("tenant_profile: find by tenant_id: %w", err)
	}
	return &profile, nil
}

// Upsert creates or updates a tenant profile.
// Uses ON CONFLICT to handle the 1:1 relationship.
func (r *Repository) Upsert(ctx context.Context, profile *types.TenantProfile) error {
	result := r.db.WithContext(ctx).Save(profile)
	if result.Error != nil {
		return fmt.Errorf("tenant_profile: upsert: %w", result.Error)
	}
	return nil
}

// Update patches a tenant profile by tenant ID with the provided field updates.
func (r *Repository) Update(ctx context.Context, tenantID uuid.UUID, updates map[string]any) (*types.TenantProfile, error) {
	if err := r.db.WithContext(ctx).
		Model(&types.TenantProfile{}).
		Where("tenant_id = ?", tenantID).
		Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("tenant_profile: update: %w", err)
	}
	return r.FindByTenantID(ctx, tenantID)
}
