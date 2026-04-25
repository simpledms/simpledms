package server

import "github.com/marcobeierer/go-core/pluginx"

func newDefaultPluginRegistry() *pluginx.Registry {
	registry := pluginx.NewRegistry()
	registry.SetPlugins(NewSimpleDMSMainMenuPlugin())
	return registry
}
