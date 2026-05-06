package tenantprofile

import "github.com/edgescaleDev/kernel/sdk"

// RouteHandlers registers all HTTP endpoints for the tenant_profile module.
func (m *Module) RouteHandlers() []sdk.RouteHandler {
	return []sdk.RouteHandler{
		{Type: sdk.RouteClient, Register: m.registerClientRoutes},
	}
}

func (m *Module) registerClientRoutes(r *sdk.Router) {
	t := r.Tenant()
	t.GET("/profile", "tenant_profile.profiles.read", m.handleGetProfile)
	t.PATCH("/profile", "tenant_profile.profiles.manage", m.handleUpdateProfile)
}
