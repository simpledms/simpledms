package route

import "fmt"

func StaticPageRoute() string {
	return "GET /pages/{slug}/"
}

func StaticPage(slug string) string {
	return fmt.Sprintf("/pages/%s/", slug)
}

func AboutPage() string {
	return "/pages/about/"
}

func StaticPageActionsRoute() string {
	return "/pages/"
}
