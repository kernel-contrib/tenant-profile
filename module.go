package tenantprofile

import (
	"io/fs"

	"github.com/edgescaleDev/kernel/sdk"
	"github.com/kernel-contrib/tenant-profile/internal"
	"github.com/kernel-contrib/tenant-profile/migrations"
)

// Module is the main entry point for the tenant_profile kernel module.
// It manages per-tenant business profile data (address, industry, size)
// as a headless module with no HTTP endpoints. Other modules access
// profile data via the TenantProfileReader interface.
type Module struct {
	ctx  sdk.Context
	repo *internal.Repository
	svc  *internal.Service
}

// New constructs the module.
func New() *Module {
	return &Module{}
}

// Manifest returns immutable metadata for this module.
func (m *Module) Manifest() sdk.Manifest {
	return sdk.Manifest{
		ID:          "tenant_profile",
		Type:        sdk.TypeCore,
		Schema:      "module_tenant_profile",
		Name:        "Tenant Profile",
		Description: "Stores extended business profile data for tenants (address, industry, company size)",
		Version:     "0.2.0",

		Permissions: []sdk.Permission{
			{Key: "tenant_profile.profiles.read", Label: sdk.T("View tenant profile", "ar", "عرض ملف تعريف المنشأة")},
			{Key: "tenant_profile.profiles.manage", Label: sdk.T("Update tenant profile", "ar", "تحديث ملف تعريف المنشأة")},
		},

		PublicEvents: []sdk.EventDef{
			{Subject: "tenant_profile.profile.created", Description: sdk.T("A tenant profile was created", "ar", "تم إنشاء ملف تعريف")},
			{Subject: "tenant_profile.profile.updated", Description: sdk.T("A tenant profile was updated", "ar", "تم تحديث ملف تعريف")},
		},
	}
}

// Migrations returns the embedded SQL migration files.
func (m *Module) Migrations() fs.FS {
	return migrations.FS
}

// Init wires the module's internal services. Called once at startup by the kernel.
func (m *Module) Init(ctx sdk.Context) error {
	m.ctx = ctx
	m.repo = internal.NewRepository(ctx.DB)
	m.svc = internal.NewService(m.repo, ctx.Bus, ctx.Logger)

	// Register a reader so other modules can consume tenant profile data.
	// Other modules resolve it via: sdk.Reader[tenantprofile.TenantProfileReader](&ctx, "tenant_profile")
	ctx.RegisterReader(&moduleReader{
		repo: m.repo,
	})

	ctx.Logger.Info("tenant_profile module initialized")
	return nil
}

// Shutdown performs cleanup before the kernel stops.
func (m *Module) Shutdown() error {
	return nil
}
