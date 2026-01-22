package route

func AuthRoute() string {
	return "GET /auth/"
}

// TODO call login or auth
func AuthActionsRoute() string {
	return "/auth/"
}

// necessary to prevent circular dependency from main menu
// not sure if a good idea to have command routes here...
// should just get defined in one place
func SignOutCmd() string {
	return "/-/cmd/auth/sign-out-cmd"
}
