package route

func StaticPageRoute() string {
	return "GET /pages/{slug}/"
}

func AboutPage() string {
	return "/pages/about/"
}
