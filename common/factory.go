package common

// Deprecated: Factory has no behavior. It is only kept because NewInfra accepts it and
// simpledms-saas constructs Infra in its tests.
type Factory struct{}

func NewFactory() *Factory {
	return &Factory{}
}
