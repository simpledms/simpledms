package widget

type MainLayout struct {
	Widget[MainLayout]

	Navigation *NavigationRail
	Content    IWidget

	SideSheet *Dialog
	// not in use as of 13 April 2025
	Sheet IWidget
}

func (qq *MainLayout) IsTenantInMaintenanceMode() bool {
	ctx := qq.GetContext()
	return ctx.IsTenantCtx() && ctx.TenantCtx().Tenant.MaintenanceModeEnabledAt != nil
}

func (qq *MainLayout) TenantMaintenanceWarning() *Snackbar {
	snackbar := NewSnackbarf(
		"This organization is in maintenance mode. Some features may not work. " +
			"Please contact your administrator.",
	).DisableAutoDismiss()
	snackbar.ID = "tenantMaintenanceWarning"
	return snackbar
}
