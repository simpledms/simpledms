package pluginx

type RegisterActionsHook interface {
	RegisterActions(reg Registrar) error
}

type Registrar interface {
	RegisterActions(actions any)
}
