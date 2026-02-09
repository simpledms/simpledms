package staticpage

const AboutSlug = "about"

// TODO implement as metadata direct in markdown file, similar to website
var staticPageDefinitions = []StaticPageDefinition{
	{
		Slug:         AboutSlug,
		PageTitle:    "About",
		AppBarTitle:  "About SimpleDMS",
		IconName:     "info",
		MarkdownPath: "content/about.md",
	},
}

func StaticPageDefinitionBySlug(slug string) (StaticPageDefinition, bool) {
	for _, definition := range staticPageDefinitions {
		if definition.Slug == slug {
			return definition, true
		}
	}

	return StaticPageDefinition{}, false
}
