// Package types defines the shared domain types for the tenant_profile module.
// It lives in its own sub-package so that reader consumers and other
// modules can import types without creating a cycle back to the parent package.
package types

import (
	"time"

	"github.com/google/uuid"
	"go.edgescale.dev/kernel/sdk"
	"gorm.io/gorm"
)

// TenantProfile is a 1:1 extension record for a tenant.
// The TenantID serves as the primary key (each tenant has exactly one profile).
type TenantProfile struct {
	TenantID  uuid.UUID      `json:"tenant_id"  gorm:"type:uuid;primaryKey"`
	Address   sdk.JSONB      `json:"address"    gorm:"type:jsonb;not null;default:'{}'"`
	Industry  *string        `json:"industry,omitempty"`
	Size      *string        `json:"size,omitempty"`
	Metadata  sdk.JSONB      `json:"metadata,omitempty" gorm:"type:jsonb;not null;default:'{}'"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}

// TableName overrides the GORM table name.
func (TenantProfile) TableName() string {
	return "tenants_profile"
}
