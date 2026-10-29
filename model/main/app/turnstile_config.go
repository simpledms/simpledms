package app

// Public test keys are used only at runtime in explicit development mode.
const (
	TurnstileTestSiteKey = "1x00000000000000000000AA"
	TurnstileTestSecret  = "1x0000000000000000000000000000000AA"
)

// TurnstileConfig contains the stored signup bot-verification configuration.
type TurnstileConfig struct {
	TurnstileSiteKey string
	TurnstileSecret  string `form_attr_type:"password"`
}
