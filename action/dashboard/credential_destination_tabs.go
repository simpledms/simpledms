package dashboard

import (
	"sort"
	"strconv"

	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
)

// credentialDestinationTabs orders the Space tabs of a credential list: accessible
// destinations sorted by label first, followed by destinations the account can no longer
// access. Only destinations with at least one credential get a tab.
type credentialDestinationTabs struct {
	keys              []string
	labelsByKey       map[string]string
	destinationsByKey map[string]*webDAVCredentialDestination
}

func newCredentialDestinationTabs(
	ctx ctxx.Context,
	destinations []*webDAVCredentialDestination,
	usedKeys map[string]bool,
) *credentialDestinationTabs {
	destinationsByKey := make(map[string]*webDAVCredentialDestination, len(destinations))
	var keys []string
	for _, destination := range destinations {
		key := destination.key()
		destinationsByKey[key] = destination
		if usedKeys[key] {
			keys = append(keys, key)
		}
	}
	var unavailableKeys []string
	for key := range usedKeys {
		if destinationsByKey[key] == nil {
			unavailableKeys = append(unavailableKeys, key)
		}
	}
	sort.Strings(unavailableKeys)
	keys = append(keys, unavailableKeys...)

	labelsByKey := make(map[string]string, len(keys))
	unavailableIndex := 1
	for _, key := range keys {
		if destination := destinationsByKey[key]; destination != nil {
			labelsByKey[key] = destination.label
			continue
		}
		labelsByKey[key] = widget.T("Unavailable destination").String(ctx)
		if len(unavailableKeys) > 1 {
			labelsByKey[key] += " " + strconv.Itoa(unavailableIndex)
		}
		unavailableIndex++
	}

	return &credentialDestinationTabs{
		keys:              keys,
		labelsByKey:       labelsByKey,
		destinationsByKey: destinationsByKey,
	}
}

// ActiveKey falls back to the first tab if the requested destination has no credentials.
func (qq *credentialDestinationTabs) ActiveKey(requestedKey string) string {
	for _, key := range qq.keys {
		if key == requestedKey {
			return key
		}
	}
	return qq.keys[0]
}

// Destination returns false for destinations the account can no longer access.
func (qq *credentialDestinationTabs) Destination(key string) (*webDAVCredentialDestination, bool) {
	destination, ok := qq.destinationsByKey[key]
	return destination, ok
}

func (qq *credentialDestinationTabs) TabBar(
	activeKey string,
	tabAttrs func(key string) widget.HTMXAttrs,
	activeTabContent widget.IWidget,
) *widget.TabBar {
	tabs := make([]*widget.Tab, 0, len(qq.keys))
	for _, key := range qq.keys {
		tabs = append(tabs, &widget.Tab{
			Label:     widget.Tu(qq.labelsByKey[key]),
			HTMXAttrs: tabAttrs(key),
		})
	}
	return &widget.TabBar{
		Tabs:             tabs,
		IsFlowing:        true,
		ActiveTab:        webDAVCredentialTabID(qq.labelsByKey[activeKey]),
		ActiveTabContent: activeTabContent,
	}
}
