package widget

// Turnstile embeds an explicitly rendered challenge tied to its containing HTMX form.
// Only public widget configuration belongs here; secrets stay in the backend.
type Turnstile struct {
	Widget[Turnstile]

	SiteKey string
	Action  string
}
