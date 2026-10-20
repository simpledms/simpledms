package route

func PropertiesRoute() string {
	return "GET /org/{tenant_id}/space/{space_id}/fields/"
}

func PropertyActionsRoute() string {
	return "/fields/"
}
