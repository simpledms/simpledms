package widget

type formConfig struct {
	IgnoreHiddenAttrTypeTag bool
}

// currently not necessary, but could be useful later if non zero default
// values are required
func newFormConfig() *formConfig {
	return &formConfig{}
}
