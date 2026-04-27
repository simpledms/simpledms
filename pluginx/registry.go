package pluginx

import (
	"sync"

	"github.com/simpledms/simpledms/ctxx"
	wx "github.com/simpledms/simpledms/ui/widget"
)

type Registry struct {
	mu      sync.RWMutex
	plugins []Plugin
}

func NewRegistry() *Registry {
	return &Registry{}
}

func (qq *Registry) SetPlugins(plugins ...Plugin) {
	qq.mu.Lock()
	defer qq.mu.Unlock()
	qq.plugins = append([]Plugin(nil), plugins...)
}

func (qq *Registry) Plugins() []Plugin {
	qq.mu.RLock()
	defer qq.mu.RUnlock()
	return append([]Plugin(nil), qq.plugins...)
}

func (qq *Registry) RegisterActions(reg Registrar) error {
	for _, plugin := range qq.Plugins() {
		hook, ok := plugin.(RegisterActionsHook)
		if !ok {
			continue
		}
		if err := hook.RegisterActions(reg); err != nil {
			return err
		}
	}
	return nil
}

// TODO add Position argument and call multiple times for different positions?
func (qq *Registry) ExtendMenuItems(ctx ctxx.Context, items []*wx.MenuItem) []*wx.MenuItem {
	for _, plugin := range qq.Plugins() {
		hook, ok := plugin.(ExtendMenuItemsHook)
		if !ok {
			continue
		}
		// TODO does this work correctly if multiple plugins are registered?
		items = hook.ExtendMenuItems(ctx, items)
	}
	return items
}

func (qq *Registry) EmitSignUp(ctx ctxx.Context, event SignUpEvent) error {
	for _, plugin := range qq.Plugins() {
		hook, ok := plugin.(OnSignUpHook)
		if !ok {
			continue
		}
		if err := hook.OnSignUp(ctx, event); err != nil {
			return err
		}
	}
	return nil
}
