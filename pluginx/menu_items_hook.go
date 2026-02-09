package pluginx

type MenuItemsHook interface {
	MenuItems(ctx MenuContext) []MenuItem
}
