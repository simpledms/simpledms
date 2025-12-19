package browse

import (
	"fmt"
	"slices"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/documenttype"
	"github.com/simpledms/simpledms/model/common/fieldtype"
	"github.com/simpledms/simpledms/renderable"
	"github.com/simpledms/simpledms/ui/uix/route"
	"github.com/simpledms/simpledms/ui/util"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type PropertyFilterValue struct {
	PropertyID int64  `url:"id,omitempty"`
	Operator   string `url:"operator,omitempty"`
	Value      string `url:"value,omitempty"` // TODO typed?
}

type PropertiesFilterState struct {
	// TODO rename to Selected or Active or Enabled or without Prefix?
	// PropertyIDs    []int64                `url:"property_ids,omitempty"`
	PropertyValues []PropertyFilterValue `url:"properties,omitempty"`
}

type ListFilterPropertiesData struct {
	CurrentDirID   string
	DocumentTypeID int64 // can be 0
}

// TODO rename
type ListFilterProperties struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewListFilterProperties(infra *common.Infra, actions *Actions) *ListFilterProperties {
	return &ListFilterProperties{
		infra:   infra,
		actions: actions,
		Config: actionx.NewConfig(
			actions.Route("list-filter-properties"),
			true,
		),
	}
}

func (qq *ListFilterProperties) Data(currentDirID string, documentTypeID int64) *ListFilterPropertiesData {
	return &ListFilterPropertiesData{
		CurrentDirID:   currentDirID,
		DocumentTypeID: documentTypeID,
	}
}

func (qq *ListFilterProperties) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[ListFilterPropertiesData](rw, req, ctx)
	if err != nil {
		return err
	}

	state := autil.StateX[ListDirState](rw, req)

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		qq.Widget(ctx, data, state),
	)
}

func (qq *ListFilterProperties) Widget(
	ctx ctxx.Context,
	data *ListFilterPropertiesData,
	state *ListDirState,
) renderable.Renderable {
	// TODO only available ones?
	var propertiesx []*enttenant.Property
	if data.DocumentTypeID == 0 {
		propertiesx = ctx.SpaceCtx().Space.QueryProperties().AllX(ctx)
	} else {
		// TODO is this okay? or inefficient
		propertiesx = ctx.SpaceCtx().Space.
			QueryDocumentTypes().
			Where(documenttype.ID(data.DocumentTypeID)).
			QueryAttributes().
			QueryProperty().
			AllX(ctx)
	}

	if len(propertiesx) == 0 {
		return &wx.EmptyState{
			Headline: wx.T("No fields available yet."),
			Actions: []wx.IWidget{
				&wx.Button{
					Icon:  wx.NewIcon("tune"),
					Label: wx.T("Manage fields"),
					HTMXAttrs: wx.HTMXAttrs{
						HxGet: route.ManageProperties(
							ctx.SpaceCtx().TenantID, ctx.SpaceCtx().SpaceID,
						),
					},
				},
			},
		}
	}

	var chips []*wx.FilterChip
	var filters []wx.IWidget

	var children []wx.IWidget

	for _, propertyx := range propertiesx {
		isChecked := slices.ContainsFunc(state.PropertyValues, func(value PropertyFilterValue) bool {
			return value.PropertyID == propertyx.ID
		})

		chips = append(chips, &wx.FilterChip{
			Label:     wx.Tu(propertyx.Name),
			IsChecked: isChecked,
			HTMXAttrs: wx.HTMXAttrs{
				HxPost:    qq.actions.TogglePropertyFilter.Endpoint(),
				HxVals:    util.JSON(qq.actions.TogglePropertyFilter.Data(data.CurrentDirID, propertyx.ID)),
				HxTrigger: "click",
				HxHeaders: autil.QueryHeader(
					qq.Endpoint(),
					qq.Data(data.CurrentDirID, data.DocumentTypeID),
				),
				HxTarget: "#" + qq.id(),
				HxSelect: "#" + qq.id(),
			},
		})

		if isChecked {
			filters = append(filters, qq.filter(state, propertyx, data.CurrentDirID))
		}
	}

	children = append(children, &wx.Container{
		Child: chips,
	})
	if len(filters) > 0 {
		children = append(children, filters)
	}

	if len(children) == 0 {
		children = append(children, wx.T("No fields available."))
	}

	return &wx.Container{
		Widget: wx.Widget[wx.Container]{
			ID: qq.id(),
		},
		GapY: true,
		Child: &wx.Column{
			GapYSize: wx.Gap4,
			Children: children,
		},
	}
}

func (qq *ListFilterProperties) filter(state *ListDirState, propertyx *enttenant.Property, currentDirID string) wx.IWidget {
	valueIndex := slices.IndexFunc(state.PropertiesFilterState.PropertyValues, func(value PropertyFilterValue) bool {
		return value.PropertyID == propertyx.ID
	})
	var value PropertyFilterValue
	if valueIndex == -1 {
		value = PropertyFilterValue{
			PropertyID: propertyx.ID,
			// TODO default operator
		}
	} else {
		value = state.PropertiesFilterState.PropertyValues[valueIndex]
	}

	switch propertyx.Type {
	case fieldtype.Text:
		return qq.renderTextFilter(propertyx, value, currentDirID)
	case fieldtype.Number,
		fieldtype.Money,
		fieldtype.Date:
		return qq.renderNumberMoneyDateFilter(propertyx, value, currentDirID)
	case fieldtype.Checkbox:
		return qq.renderCheckboxFilter(propertyx, value, currentDirID)
	default:
		return &wx.View{}
	}
}

type operatorValue string

func (qq operatorValue) Equals(str string) bool {
	return string(qq) == str
}

func (qq operatorValue) String() string {
	return string(qq)
}

var (
	textOperatorValueContains   = operatorValue("contains")
	textOperatorValueStartsWith = operatorValue("starts_with")

	operatorValueEquals      = operatorValue("equals")
	operatorValueGreaterThan = operatorValue("greater_than")
	operatorValueLessThan    = operatorValue("less_than")
	// TODO add between

	checkboxOperatorValueIsChecked    = operatorValue("is_checked")
	checkboxOperatorValueIsNotChecked = operatorValue("is_not_checked")
)

func (qq *ListFilterProperties) renderTextFilter(propertyx *enttenant.Property, value PropertyFilterValue, currentDirID string) wx.IWidget {
	if value.Operator == "" {
		value.Operator = textOperatorValueContains.String()
	}

	containerID := autil.GenerateID(propertyx.Name)

	attrsFn := func(trigger string) wx.HTMXAttrs {
		return wx.HTMXAttrs{
			HxPost:    qq.actions.UpdatePropertyFilter.Endpoint(),
			HxVals:    util.JSON(qq.actions.UpdatePropertyFilter.Data(currentDirID, propertyx.ID)),
			HxSwap:    "none",
			HxInclude: fmt.Sprintf("#%s input", containerID),
			HxTrigger: trigger,
		}
	}

	// TODO as filter chips or select field in same line as text field?
	operators := []*wx.FilterChip{
		{
			Type:      wx.FilterChipTypeRadio,
			Label:     wx.T("Contains"),
			Name:      "Operator",
			Value:     textOperatorValueContains.String(),
			IsChecked: textOperatorValueContains.Equals(value.Operator),
			HTMXAttrs: attrsFn("change"),
		},
		{
			Type:      wx.FilterChipTypeRadio,
			Label:     wx.T("Starts with"),
			Name:      "Operator",
			Value:     textOperatorValueStartsWith.String(),
			IsChecked: textOperatorValueStartsWith.Equals(value.Operator),
			HTMXAttrs: attrsFn("change"),
		},
		{
			Type:      wx.FilterChipTypeRadio,
			Label:     wx.T("Equals"),
			Name:      "Operator",
			Value:     operatorValueEquals.String(),
			IsChecked: operatorValueEquals.Equals(value.Operator),
			HTMXAttrs: attrsFn("change"),
		},
		// TODO `doesn't contain`, etc.
	}

	field := &wx.TextField{
		// `change` more reliable in modal, triggers also on modal close, input doesn't trigger
		// if modal gets closed before delay; input necessary for use as sidebar when change
		// should get applied directly
		HTMXAttrs:    attrsFn("change, input delay:1000ms"),
		Label:        wx.Tu(propertyx.Name),
		Name:         "Value",
		Type:         "text",
		DefaultValue: value.Value,
	}

	return &wx.Container{
		Widget: wx.Widget[wx.Container]{
			ID: containerID,
		},
		Child: []wx.IWidget{
			field,
			operators,
		},
	}
}

func (qq *ListFilterProperties) renderNumberMoneyDateFilter(propertyx *enttenant.Property, value PropertyFilterValue, currentDirID string) wx.IWidget {
	if value.Operator == "" {
		value.Operator = operatorValueEquals.String()
	}

	containerID := autil.GenerateID(propertyx.Name)

	attrsFn := func(trigger string) wx.HTMXAttrs {
		return wx.HTMXAttrs{
			HxPost:    qq.actions.UpdatePropertyFilter.Endpoint(),
			HxVals:    util.JSON(qq.actions.UpdatePropertyFilter.Data(currentDirID, propertyx.ID)),
			HxSwap:    "none",
			HxInclude: fmt.Sprintf("#%s input", containerID),
			HxTrigger: trigger,
		}
	}

	operators := qq.operatorRadioGroup(value, attrsFn)

	fieldType := ""
	fieldStep := ""

	switch propertyx.Type {
	case fieldtype.Number:
		fieldType = "number"
		fieldStep = "1"
	case fieldtype.Money:
		fieldType = "number"
		fieldStep = "0.01"
	case fieldtype.Date:
		fieldType = "date"
	default:
		panic(fmt.Errorf("unsupported property type: %s", propertyx.Type))
	}

	field := &wx.TextField{
		HTMXAttrs:    attrsFn("change, input delay:1000ms"),
		Label:        wx.Tu(propertyx.Name),
		Name:         "Value",
		Type:         fieldType,
		Step:         fieldStep,
		DefaultValue: value.Value,
	}

	return &wx.Container{
		Widget: wx.Widget[wx.Container]{
			ID: containerID,
		},
		Child: []wx.IWidget{
			field,
			operators,
		},
	}
}

func (qq *ListFilterProperties) operatorRadioGroup(value PropertyFilterValue, attrsFn func(string) wx.HTMXAttrs) wx.IWidget {
	return []*wx.FilterChip{
		{
			Type:      wx.FilterChipTypeRadio,
			Label:     wx.T("Equals"),
			Name:      "Operator",
			Value:     operatorValueEquals.String(),
			IsChecked: operatorValueEquals.Equals(value.Operator),
			HTMXAttrs: attrsFn("change"),
		},
		{
			Type:      wx.FilterChipTypeRadio,
			Label:     wx.T("Greater than"),
			Name:      "Operator",
			Value:     operatorValueGreaterThan.String(),
			IsChecked: operatorValueGreaterThan.Equals(value.Operator),
			HTMXAttrs: attrsFn("change"),
		},
		{
			Type:      wx.FilterChipTypeRadio,
			Label:     wx.T("Less than"),
			Name:      "Operator",
			Value:     operatorValueLessThan.String(),
			IsChecked: operatorValueLessThan.Equals(value.Operator),
			HTMXAttrs: attrsFn("change"),
		},
		// TODO add between
	}
}

func (qq *ListFilterProperties) renderCheckboxFilter(propertyx *enttenant.Property, value PropertyFilterValue, currentDirID string) wx.IWidget {
	if value.Operator == "" {
		value.Operator = checkboxOperatorValueIsChecked.String()
	}

	containerID := autil.GenerateID(propertyx.Name)

	attrsFn := func(trigger string) wx.HTMXAttrs {
		return wx.HTMXAttrs{
			HxPost:    qq.actions.UpdatePropertyFilter.Endpoint(),
			HxVals:    util.JSON(qq.actions.UpdatePropertyFilter.Data(currentDirID, propertyx.ID)),
			HxSwap:    "none",
			HxInclude: fmt.Sprintf("#%s input", containerID),
			HxTrigger: trigger,
		}
	}

	// TODO as filter chips or select field in same line as text field?
	operators := []*wx.FilterChip{
		{
			Type:  wx.FilterChipTypeRadio,
			Label: wx.Tf("«%s» is checked", propertyx.Name),
			Name:  "Value",
			// Value:     checkboxOperatorValueIsChecked.String(),
			// IsChecked: checkboxOperatorValueIsChecked.Equals(value.Operator),
			Value:     "true", // TODO or 1?
			IsChecked: value.Value == "true",
			HTMXAttrs: attrsFn("change"),
		},
		{
			Type:  wx.FilterChipTypeRadio,
			Label: wx.Tf("«%s» is not checked", propertyx.Name),
			Name:  "Value",
			// Value:     checkboxOperatorValueIsNotChecked.String(),
			// IsChecked: checkboxOperatorValueIsNotChecked.Equals(value.Operator),
			Value:     "false", // TODO or 0?
			IsChecked: value.Value == "false",
			HTMXAttrs: attrsFn("change"),
		},
	}

	/*
		field := &wx.TextField{
			Name:         "Value",
			Type:         "hidden",
			DefaultValue: value.Value,
		}
	*/

	return &wx.Container{
		Widget: wx.Widget[wx.Container]{
			ID: containerID,
		},
		Child: []wx.IWidget{
			// field,
			operators,
		},
	}
}

func (qq *ListFilterProperties) id() string {
	return "listFilterProperties"
}

/*
func (qq *ListFilterProperties) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[ListFilterPropertiesData](rw, req, ctx)
	if err != nil {
		return err
	}
	state := autil.StateX[ListDirState](rw, req)

	// Ensure PropertyFilters is initialized from FlatPropertyFilters if needed
	EnsurePropertyFilters(&state.PropertiesFilterState)

	// If a property filter is being applied
	if data.PropertyID != 0 && data.FilterType != "" {
		// Handle property selection
		if data.FilterType == "select_property" {
			// Initialize the selected property IDs slice if it's nil
			if state.PropertyIDs == nil {
				state.PropertyIDs = []int64{}
			}

			// Check if the property is already selected
			isSelected := false
			index := -1
			for i, id := range state.PropertyIDs {
				if id == data.PropertyID {
					isSelected = true
					index = i
					break
				}
			}

			// Add or remove the property ID based on the filter value
			if data.FilterValue != "" && !isSelected {
				// Add the property ID to the selected properties
				state.PropertyIDs = append(state.PropertyIDs, data.PropertyID)
			} else if data.FilterValue == "" && isSelected {
				// Remove the property ID from the selected properties
				state.PropertyIDs = append(state.PropertyIDs[:index], state.PropertyIDs[index+1:]...)

				// Also remove any filters for this property
				if state.PropertyFilters != nil {
					delete(state.PropertyFilters, data.PropertyID)
					// Update the flattened representation
					UpdateFlatPropertyFilters(&state.PropertiesFilterState)
				}
			}
		} else {
			// Handle regular property filters
			// Initialize the map if it's nil
			if state.PropertyFilters == nil {
				state.PropertyFilters = make(map[int64]map[string]string)
			}

			// Initialize the property filter map if it's nil
			if state.PropertyFilters[data.PropertyID] == nil {
				state.PropertyFilters[data.PropertyID] = make(map[string]string)
			}

			// Set the filter value
			if data.FilterValue != "" {
				state.PropertyFilters[data.PropertyID][data.FilterType] = data.FilterValue
			} else {
				// Remove the filter if the value is empty
				delete(state.PropertyFilters[data.PropertyID], data.FilterType)
				if len(state.PropertyFilters[data.PropertyID]) == 0 {
					delete(state.PropertyFilters, data.PropertyID)
				}
			}

			// Update the flattened representation
			UpdateFlatPropertyFilters(&state.PropertiesFilterState)
		}

		// Trigger the properties filter changed event
		rw.Header().Set("HX-Trigger", event.PropertyFilterChanged.String())
		return nil
	}

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		autil.WrapWidget(
			"Fields | Filter",
			"",
			qq.Widget(ctx, data.CurrentDirID, state.PropertiesFilterState),
			actionx.ResponseWrapperDialog,
			false,
		),
	)
}

func (qq *ListFilterProperties) Widget(
	ctx ctxx.Context,
	currentDirID string,
	propertiesFilterState PropertiesFilterState,
) renderable.Renderable {
	// Get all properties in the space
	propertiesInSpace := ctx.SpaceCtx().Space.QueryProperties().
		Order(property.ByName()).
		AllX(ctx)

	var children []wx.IWidget

	// Create property selection chips
	var propertySelectionChips []wx.IWidget
	for _, prop := range propertiesInSpace {
		// Check if this property is selected
		isSelected := false
		for _, selectedID := range propertiesFilterState.PropertyIDs {
			if selectedID == prop.ID {
				isSelected = true
				break
			}
		}

		propertySelectionChips = append(propertySelectionChips, &wx.FilterChip{
			Label:     wx.Tu(prop.Name),
			IsChecked: isSelected,
			HTMXAttrs: wx.HTMXAttrs{
				HxPost: qq.actions.TogglePropertyFilter.Endpoint(),
				HxVals: util.JSON(qq.actions.TogglePropertyFilter.Data(currentDirID, prop.ID)),
				// HxSwap:    "none",
				HxTrigger: "click",
				HxHeaders: autil.QueryHeader(
					qq.Endpoint(),
					qq.Data(currentDirID),
				),
				HxTarget: "#" + qq.id(),
				HxSelect: "#" + qq.id(),
			},
		})
	}

	// Add field selection section
	children = append(children,
		&wx.Container{
			Gap:   true,
			Child: propertySelectionChips,
		})

	// Only show filters for selected properties
	if len(propertiesFilterState.PropertyIDs) > 0 {
		// Group properties by type
		propertyGroups := map[propertytype.PropertyType][]wx.IWidget{}

		for _, prop := range propertiesInSpace {
			// Skip if this property is not selected
			isSelected := false
			for _, selectedID := range propertiesFilterState.PropertyIDs {
				if selectedID == prop.ID {
					isSelected = true
					break
				}
			}
			if !isSelected {
				continue
			}

			// Create filter options based on property type
			var filterOptions []wx.IWidget

			// Get current filters for this property
			var currentFilters map[string]string
			if propertiesFilterState.PropertyFilters != nil {
				currentFilters = propertiesFilterState.PropertyFilters[prop.ID]
			}

			switch prop.Type {
			case propertytype.Text:
				filterOptions = append(filterOptions, qq.createTextFilterWithChips(ctx, currentDirID, prop.ID, currentFilters))

			case propertytype.Number, propertytype.Money:
				// Number property filter with filter chips for filter type
				filterOptions = append(filterOptions, qq.createNumberFilterWithChips(ctx, currentDirID, prop.ID, currentFilters))
				// Keep the range filter separate as it requires two input fields
				filterOptions = append(filterOptions, qq.createNumberRangeFilter(ctx, currentDirID, prop.ID, "between", "Between", currentFilters))

			case propertytype.Date:
				// Date property filter with filter chips for filter type
				filterOptions = append(filterOptions, qq.createDateFilterWithChips(ctx, currentDirID, prop.ID, currentFilters))
				// Keep the range filter separate as it requires two input fields
				filterOptions = append(filterOptions, qq.createDateRangeFilter(ctx, currentDirID, prop.ID, "between", "Between", currentFilters))

			case propertytype.Checkbox:
				// Checkbox property filters: is checked
				filterOptions = append(filterOptions, qq.createCheckboxFilter(ctx, currentDirID, prop.ID, "isChecked", "Is checked", currentFilters))
			}

			// Create a container for this property
			propertyContainer := &wx.Container{
				Child: []wx.IWidget{
					wx.H(wx.HeadingTypeTitleMd, wx.Tu(prop.Name)),
					&wx.Container{
						Child: filterOptions,
					},
				},
			}

			// Add to the appropriate group
			propType := prop.Type
			if propType == propertytype.Unknown {
				propType = propertytype.Text // Default to text for unknown types
			}

			group, found := propertyGroups[propType]
			if !found {
				group = []wx.IWidget{}
			}
			group = append(group, propertyContainer)
			propertyGroups[propType] = group
		}

		// Add field type headings and their properties
		propertyTypeNames := map[propertytype.PropertyType]string{
			propertytype.Text:     "Text Properties",
			propertytype.Number:   "Number Properties",
			propertytype.Money:    "Money Properties",
			propertytype.Date:     "Date Properties",
			propertytype.Checkbox: "Checkbox Properties",
		}

		// Sort property types for consistent order
		propertyTypes := []propertytype.PropertyType{
			propertytype.Text,
			propertytype.Number,
			propertytype.Money,
			propertytype.Date,
			propertytype.Checkbox,
		}

		// Add a divider between property selection and filters
		children = append(children, &wx.Container{
			Child: &wx.Divider{},
		})

		// Add selected properties filters
		for _, propType := range propertyTypes {
			group, found := propertyGroups[propType]
			if found && len(group) > 0 {
				children = append(children, wx.H(wx.HeadingTypeTitleLg, wx.Tu(propertyTypeNames[propType])))
				children = append(children, group...)
			}
		}
	} else {
		// If no properties selected, show a message
		children = append(children, &wx.Container{
			Child: wx.T("Select properties above to show filter options."),
		})
	}

	return &wx.Container{
		Widget: wx.Widget[wx.Container]{
			ID: qq.id(),
		},
		GapY:  true,
		Child: children,
	}
}

// Helper methods to create different types of filters

func (qq *ListFilterProperties) createTextFilterWithChips(
	ctx ctxx.Context,
	currentDirID string,
	propertyID int64,
	currentFilters map[string]string,
) wx.IWidget {
	// Define filter types
	filterTypes := []struct {
		Value string
		Label string
	}{
		{"equals", "Equals"},
		{"startsWith", "Starts with"},
		{"contains", "Contains"},
	}

	// Determine the current filter type and value
	currentFilterType := "contains" // Default filter type
	currentFilterValue := ""

	if currentFilters != nil {
		// Check if any filter type is active and get its value
		for _, ft := range filterTypes {
			if value, exists := currentFilters[ft.Value]; exists {
				currentFilterType = ft.Value
				currentFilterValue = value
				break
			}
		}
	}

	// Create filter chips for filter types
	var filterChips []wx.IWidget
	for _, ft := range filterTypes {
		filterChips = append(filterChips, &wx.FilterChip{
			Label:     wx.Tu(ft.Label),
			IsChecked: ft.Value == currentFilterType,
			HTMXAttrs: wx.HTMXAttrs{
				// When clicked, update the hidden input with the filter type
				HxOn: &wx.HxOn{
					Event: "click",
					// TODO XSS? Handler: template.JS("document.querySelector('#filterType_" + fmt.Sprintf("%d", propertyID) + "').value = '" + ft.Value + "'"),
				},
			},
		})
	}

	return &wx.Container{
		Child: []wx.IWidget{
			&wx.Container{
				Gap:   true,
				Child: filterChips,
			},
			&wx.TextField{
				Label:        wx.Tu(ft.Label),
				Name:         "FilterValue",
				Type:         "text",
				DefaultValue: currentFilterValue,
				HTMXAttrs: wx.HTMXAttrs{
					HxPost:    qq.Endpoint(),
					HxVals:    util.JSON(qq.Data(currentDirID)),
					HxTrigger: "keyup changed delay:500ms",
					HxSwap:    "none",
				},
			},
		},
	}
}

func (qq *ListFilterProperties) createNumberFilterWithChips(
	ctx ctxx.Context,
	currentDirID string,
	propertyID int64,
	currentFilters map[string]string,
) wx.IWidget {
	// Define filter types
	filterTypes := []struct {
		Value string
		Label string
	}{
		{"equals", "Equals"},
		{"lessThan", "Less than"},
		{"greaterThan", "Greater than"},
	}

	// Determine the current filter type and value
	currentFilterType := "equals" // Default filter type
	currentFilterValue := ""

	if currentFilters != nil {
		// Check if any filter type is active and get its value
		for _, ft := range filterTypes {
			if value, exists := currentFilters[ft.Value]; exists {
				currentFilterType = ft.Value
				currentFilterValue = value
				break
			}
		}
	}

	// Create filter chips for filter types
	var filterChips []wx.IWidget
	for _, ft := range filterTypes {
		filterChips = append(filterChips, &wx.FilterChip{
			Label:     wx.Tu(ft.Label),
			IsChecked: ft.Value == currentFilterType,
			HTMXAttrs: wx.HTMXAttrs{
				// When clicked, update the hidden input with the filter type
				HxOn: &wx.HxOn{
					Event: "click",
					// TODO XSS?
					// Handler: template.JS("document.querySelector('#filterType_" + fmt.Sprintf("%d", propertyID) + "').value = '" + ft.Value + "'"),
				},
			},
		})
	}

	return &wx.Container{
		Child: []wx.IWidget{
			// Filter type chips
			&wx.Container{
				Child: []wx.IWidget{
					&wx.Label{
						Text: wx.Tu("Filter type"),
						Type: wx.LabelTypeMd,
					},
					&wx.Container{
						Gap:   true,
						Child: filterChips,
					},
				},
			},
			// Hidden input to store the current filter type
			&wx.TextField{
				Widget: wx.Widget[wx.TextField]{
					ID: "filterType_" + fmt.Sprintf("%d", propertyID),
				},
				Type:         "hidden",
				Name:         "FilterType",
				DefaultValue: currentFilterType,
			},
			// Number field for filter value
			&wx.TextField{
				Label:        wx.Tu("Value"),
				Name:         "FilterValue",
				Type:         "number",
				DefaultValue: currentFilterValue,
				HTMXAttrs: wx.HTMXAttrs{
					HxPost: qq.Endpoint(),
					HxVals: util.JSON(map[string]interface{}{
						"CurrentDirID": currentDirID,
						"PropertyID":   propertyID,
						"FilterType":   currentFilterType, // Will be updated by JS
					}),
					HxTrigger: "keyup changed delay:500ms",
					HxSwap:    "none",
				},
			},
		},
	}
}

func (qq *ListFilterProperties) createTextFilter(
	ctx ctxx.Context,
	currentDirID string,
	propertyID int64,
	filterType string,
	label string,
	currentFilters map[string]string,
) wx.IWidget {
	currentValue := ""
	if currentFilters != nil {
		currentValue = currentFilters[filterType]
	}

	return &wx.Container{
		Child: []wx.IWidget{
			&wx.TextField{
				Label:        wx.Tu(label),
				Name:         "FilterValue",
				Type:         "text",
				DefaultValue: currentValue,
				HTMXAttrs: wx.HTMXAttrs{
					HxPost: qq.Endpoint(),
					HxVals: util.JSON(map[string]interface{}{
						"CurrentDirID": currentDirID,
						"PropertyID":   propertyID,
						"FilterType":   filterType,
					}),
					HxTrigger: "keyup changed delay:500ms",
					HxSwap:    "none",
				},
			},
		},
	}
}

func (qq *ListFilterProperties) createNumberFilter(
	ctx ctxx.Context,
	currentDirID string,
	propertyID int64,
	filterType string,
	label string,
	currentFilters map[string]string,
) wx.IWidget {
	currentValue := ""
	if currentFilters != nil {
		currentValue = currentFilters[filterType]
	}

	return &wx.Container{
		Child: []wx.IWidget{
			&wx.TextField{
				Label:        wx.Tu(label),
				Name:         "FilterValue",
				Type:         "number",
				DefaultValue: currentValue,
				HTMXAttrs: wx.HTMXAttrs{
					HxPost: qq.Endpoint(),
					HxVals: util.JSON(map[string]interface{}{
						"CurrentDirID": currentDirID,
						"PropertyID":   propertyID,
						"FilterType":   filterType,
					}),
					HxTrigger: "keyup changed delay:500ms",
					HxSwap:    "none",
				},
			},
		},
	}
}

func (qq *ListFilterProperties) createNumberRangeFilter(
	ctx ctxx.Context,
	currentDirID string,
	propertyID int64,
	filterType string,
	label string,
	currentFilters map[string]string,
) wx.IWidget {
	currentValue := ""
	minValue := ""
	maxValue := ""

	if currentFilters != nil {
		currentValue = currentFilters[filterType]
		if currentValue != "" {
			// Parse the range value (format: "min,max")
			parts := strings.Split(currentValue, ",")
			if len(parts) == 2 {
				minValue = parts[0]
				maxValue = parts[1]
			}
		}
	}

	return &wx.Container{
		Child: []wx.IWidget{
			wx.Tu(label),
			&wx.Row{
				Children: []wx.IWidget{
					&wx.TextField{
						Label:        wx.Tu("Min"),
						Name:         "MinValue",
						Type:         "number",
						DefaultValue: minValue,
						HTMXAttrs: wx.HTMXAttrs{
							HxPost: qq.Endpoint(),
							HxVals: util.JSON(map[string]interface{}{
								"CurrentDirID": currentDirID,
								"PropertyID":   propertyID,
								"FilterType":   filterType,
								"FilterValue":  minValue + "," + maxValue, // Will be updated by JS
							}),
							HxTrigger: "keyup changed delay:500ms",
							HxSwap:    "none",
						},
					},
					&wx.TextField{
						Label:        wx.Tu("Max"),
						Name:         "MaxValue",
						Type:         "number",
						DefaultValue: maxValue,
						HTMXAttrs: wx.HTMXAttrs{
							HxPost: qq.Endpoint(),
							HxVals: util.JSON(map[string]interface{}{
								"CurrentDirID": currentDirID,
								"PropertyID":   propertyID,
								"FilterType":   filterType,
								"FilterValue":  minValue + "," + maxValue, // Will be updated by JS
							}),
							HxTrigger: "keyup changed delay:500ms",
							HxSwap:    "none",
						},
					},
					// Hidden field to store the combined value
					&wx.TextField{
						Type:         "hidden",
						Name:         "FilterValue",
						DefaultValue: minValue + "," + maxValue,
					},
				},
			},
		},
	}
}

func (qq *ListFilterProperties) createDateFilterWithChips(
	ctx ctxx.Context,
	currentDirID string,
	propertyID int64,
	currentFilters map[string]string,
) wx.IWidget {
	// Define filter types
	filterTypes := []struct {
		Value string
		Label string
	}{
		{"equals", "Equals"},
		{"before", "Before"},
		{"after", "After"},
	}

	// Determine the current filter type and value
	currentFilterType := "equals" // Default filter type
	currentFilterValue := ""

	if currentFilters != nil {
		// Check if any filter type is active and get its value
		for _, ft := range filterTypes {
			if value, exists := currentFilters[ft.Value]; exists {
				currentFilterType = ft.Value
				currentFilterValue = value
				break
			}
		}
	}

	// Create filter chips for filter types
	var filterChips []wx.IWidget
	for _, ft := range filterTypes {
		filterChips = append(filterChips, &wx.FilterChip{
			Label:     wx.Tu(ft.Label),
			IsChecked: ft.Value == currentFilterType,
			HTMXAttrs: wx.HTMXAttrs{
				// When clicked, update the hidden input with the filter type
				HxOn: &wx.HxOn{
					Event: "click",
					// TODO XSS?
					// Handler: template.JS("document.querySelector('#filterType_" + fmt.Sprintf("%d", propertyID) + "').value = '" + ft.Value + "'"),
				},
			},
		})
	}

	return &wx.Container{
		Child: []wx.IWidget{
			// Filter type chips
			&wx.Container{
				Child: []wx.IWidget{
					&wx.Label{
						Text: wx.Tu("Filter type"),
						Type: wx.LabelTypeMd,
					},
					&wx.Container{
						Gap:   true,
						Child: filterChips,
					},
				},
			},
			// Hidden input to store the current filter type
			&wx.TextField{
				Widget: wx.Widget[wx.TextField]{
					ID: "filterType_" + fmt.Sprintf("%d", propertyID),
				},
				Type:         "hidden",
				Name:         "FilterType",
				DefaultValue: currentFilterType,
			},
			// Date field for filter value
			&wx.TextField{
				Label:        wx.Tu("Value"),
				Name:         "FilterValue",
				Type:         "date",
				DefaultValue: currentFilterValue,
				HTMXAttrs: wx.HTMXAttrs{
					HxPost: qq.Endpoint(),
					HxVals: util.JSON(map[string]interface{}{
						"CurrentDirID": currentDirID,
						"PropertyID":   propertyID,
						"FilterType":   currentFilterType, // Will be updated by JS
					}),
					HxTrigger: "change",
					HxSwap:    "none",
				},
			},
		},
	}
}

func (qq *ListFilterProperties) createDateFilter(
	ctx ctxx.Context,
	currentDirID string,
	propertyID int64,
	filterType string,
	label string,
	currentFilters map[string]string,
) wx.IWidget {
	currentValue := ""
	if currentFilters != nil {
		currentValue = currentFilters[filterType]
	}

	return &wx.Container{
		Child: []wx.IWidget{
			&wx.TextField{
				Label:        wx.Tu(label),
				Name:         "FilterValue",
				Type:         "date",
				DefaultValue: currentValue,
				HTMXAttrs: wx.HTMXAttrs{
					HxPost: qq.Endpoint(),
					HxVals: util.JSON(map[string]interface{}{
						"CurrentDirID": currentDirID,
						"PropertyID":   propertyID,
						"FilterType":   filterType,
					}),
					HxTrigger: "change",
					HxSwap:    "none",
				},
			},
		},
	}
}

func (qq *ListFilterProperties) createDateRangeFilter(
	ctx ctxx.Context,
	currentDirID string,
	propertyID int64,
	filterType string,
	label string,
	currentFilters map[string]string,
) wx.IWidget {
	currentValue := ""
	startDate := ""
	endDate := ""

	if currentFilters != nil {
		currentValue = currentFilters[filterType]
		if currentValue != "" {
			// Parse the range value (format: "start,end")
			parts := strings.Split(currentValue, ",")
			if len(parts) == 2 {
				startDate = parts[0]
				endDate = parts[1]
			}
		}
	}

	return &wx.Container{
		Child: []wx.IWidget{
			wx.Tu(label),
			&wx.Row{
				Children: []wx.IWidget{
					&wx.TextField{
						Label:        wx.Tu("Start"),
						Name:         "StartDate",
						Type:         "date",
						DefaultValue: startDate,
						HTMXAttrs: wx.HTMXAttrs{
							HxPost: qq.Endpoint(),
							HxVals: util.JSON(map[string]interface{}{
								"CurrentDirID": currentDirID,
								"PropertyID":   propertyID,
								"FilterType":   filterType,
								"FilterValue":  startDate + "," + endDate, // Will be updated by JS
							}),
							HxTrigger: "change",
							HxSwap:    "none",
						},
					},
					&wx.TextField{
						Label:        wx.Tu("End"),
						Name:         "EndDate",
						Type:         "date",
						DefaultValue: endDate,
						HTMXAttrs: wx.HTMXAttrs{
							HxPost: qq.Endpoint(),
							HxVals: util.JSON(map[string]interface{}{
								"CurrentDirID": currentDirID,
								"PropertyID":   propertyID,
								"FilterType":   filterType,
								"FilterValue":  startDate + "," + endDate, // Will be updated by JS
							}),
							HxTrigger: "change",
							HxSwap:    "none",
						},
					},
					// Hidden field to store the combined value
					&wx.TextField{
						Type:         "hidden",
						Name:         "FilterValue",
						DefaultValue: startDate + "," + endDate,
					},
				},
			},
		},
	}
}

func (qq *ListFilterProperties) createCheckboxFilter(
	ctx ctxx.Context,
	currentDirID string,
	propertyID int64,
	filterType string,
	label string,
	currentFilters map[string]string,
) wx.IWidget {
	isChecked := false
	if currentFilters != nil {
		isChecked = currentFilters[filterType] == "true"
	}

	return &wx.Container{
		Child: []wx.IWidget{
			&wx.Checkbox{
				Label:     wx.Tu(label),
				Name:      "FilterValue",
				IsChecked: isChecked,
				HTMXAttrs: wx.HTMXAttrs{
					HxPost: qq.Endpoint(),
					HxVals: util.JSON(map[string]interface{}{
						"CurrentDirID": currentDirID,
						"PropertyID":   propertyID,
						"FilterType":   filterType,
						"FilterValue":  "true", // Will be set when checked, empty when unchecked
					}),
					HxTrigger: "change",
					HxSwap:    "none",
				},
			},
		},
	}
}
*/
