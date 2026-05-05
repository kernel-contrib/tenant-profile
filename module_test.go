package tenantprofile_test

import (
	"context"
	"testing"

	"github.com/edgescaleDev/kernel/sdk"
	"github.com/google/uuid"
	"github.com/kernel-contrib/tenant-profile/internal"
	"github.com/kernel-contrib/tenant-profile/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// ── test DB setup ─────────────────────────────────────────────────────────────

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "open in-memory sqlite")

	ddl := []string{
		`CREATE TABLE tenants_profile (
			tenant_id TEXT PRIMARY KEY,
			address TEXT NOT NULL DEFAULT '{}',
			industry TEXT,
			size TEXT,
			metadata TEXT NOT NULL DEFAULT '{}',
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
	}

	for _, stmt := range ddl {
		require.NoError(t, db.Exec(stmt).Error, "DDL")
	}
	return db
}

// ── test harness ──────────────────────────────────────────────────────────────

type testHarness struct {
	db   *gorm.DB
	ctx  *sdk.Context
	repo *internal.Repository
	svc  *internal.Service
}

func newTestHarness(t *testing.T) *testHarness {
	t.Helper()
	db := newTestDB(t)
	tctx := sdk.NewTestContext("tenant_profile")
	tctx.DB = db

	repo := internal.NewRepository(db)
	svc := internal.NewService(repo, tctx.Bus, tctx.Logger)

	return &testHarness{
		db:   db,
		ctx:  tctx,
		repo: repo,
		svc:  svc,
	}
}

func (h *testHarness) bus() *sdk.TestBus {
	return h.ctx.Bus.(*sdk.TestBus)
}

// ── error helpers ─────────────────────────────────────────────────────────────

func isNotFound(err error) bool {
	se, ok := sdk.IsServiceError(err)
	return ok && se.HTTPStatus == 404
}

func isConflict(err error) bool {
	se, ok := sdk.IsServiceError(err)
	return ok && se.HTTPStatus == 409
}

// ═══════════════════════════════════════════════════════════════════════════════
// Profile CRUD Tests
// ═══════════════════════════════════════════════════════════════════════════════

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
	assert.Equal(t, "50-200", *profile.Size)

	// Event published.
	events := h.bus().Events()
	require.Len(t, events, 1)
	assert.Equal(t, "tenant_profile.profile.created", events[0].Subject)
}

func TestProfileCreate_Duplicate(t *testing.T) {
	h := newTestHarness(t)
	ctx := context.Background()
	tenantID := uuid.New()

	_, err := h.svc.Create(ctx, internal.CreateProfileInput{TenantID: tenantID})
	require.NoError(t, err)

	_, err = h.svc.Create(ctx, internal.CreateProfileInput{TenantID: tenantID})
	assert.True(t, isConflict(err), "expected 409 conflict, got: %v", err)
}

func TestProfileCreate_MissingTenantID(t *testing.T) {
	h := newTestHarness(t)
	ctx := context.Background()

	_, err := h.svc.Create(ctx, internal.CreateProfileInput{})
	require.Error(t, err)
}

func TestProfileGetByTenantID(t *testing.T) {
	h := newTestHarness(t)
	ctx := context.Background()
	tenantID := uuid.New()

	_, err := h.svc.Create(ctx, internal.CreateProfileInput{TenantID: tenantID})
	require.NoError(t, err)

	found, err := h.svc.GetByTenantID(ctx, tenantID)
	require.NoError(t, err)
	assert.Equal(t, tenantID, found.TenantID)
}

func TestProfileGetByTenantID_NotFound(t *testing.T) {
	h := newTestHarness(t)
	_, err := h.svc.GetByTenantID(context.Background(), uuid.New())
	assert.True(t, isNotFound(err))
}

func TestProfileUpdate(t *testing.T) {
	h := newTestHarness(t)
	ctx := context.Background()
	tenantID := uuid.New()

	_, err := h.svc.Create(ctx, internal.CreateProfileInput{TenantID: tenantID})
	require.NoError(t, err)

	newIndustry := "healthcare"
	newSize := "200-500"
	updated, err := h.svc.Update(ctx, tenantID, internal.UpdateProfileInput{
		Industry: &newIndustry,
		Size:     &newSize,
	})
	require.NoError(t, err)
	assert.Equal(t, "healthcare", *updated.Industry)
	assert.Equal(t, "200-500", *updated.Size)

	// Should have 2 events: created + updated.
	events := h.bus().Events()
	require.Len(t, events, 2)
	assert.Equal(t, "tenant_profile.profile.updated", events[1].Subject)
}

func TestProfileUpdate_NotFound(t *testing.T) {
	h := newTestHarness(t)
	industry := "tech"
	_, err := h.svc.Update(context.Background(), uuid.New(), internal.UpdateProfileInput{
		Industry: &industry,
	})
	assert.True(t, isNotFound(err))
}

func TestProfileUpdate_NoChanges(t *testing.T) {
	h := newTestHarness(t)
	ctx := context.Background()
	tenantID := uuid.New()

	_, err := h.svc.Create(ctx, internal.CreateProfileInput{TenantID: tenantID})
	require.NoError(t, err)

	// Update with no fields should return existing profile.
	profile, err := h.svc.Update(ctx, tenantID, internal.UpdateProfileInput{})
	require.NoError(t, err)
	assert.Equal(t, tenantID, profile.TenantID)

	// Only 1 event (the create), no update event.
	events := h.bus().Events()
	assert.Len(t, events, 1)
}

func TestReaderGetProfile(t *testing.T) {
	h := newTestHarness(t)
	ctx := context.Background()
	tenantID := uuid.New()

	industry := "retail"
	_, err := h.svc.Create(ctx, internal.CreateProfileInput{
		TenantID: tenantID,
		Industry: &industry,
	})
	require.NoError(t, err)

	// Test via the repo (same path the reader uses).
	profile, err := h.repo.FindByTenantID(ctx, tenantID)
	require.NoError(t, err)
	assert.Equal(t, tenantID, profile.TenantID)
	assert.Equal(t, "retail", *profile.Industry)

	// Verify the type satisfies the model contract.
	var _ *types.TenantProfile = profile
}
