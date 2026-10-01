package dashboard

import (
	"html/template"
	"net/http"

	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/util/e"
)

// Shared by the WebDAV and MCP credential filters; both store the selection in the
// `credential_status` query param of their own page.
const (
	credentialStatusActive  = "active"
	credentialStatusRevoked = "revoked"
)

// parseCredentialStatusFilter returns which credential states are visible. Without a
// selection, only active credentials are shown.
func parseCredentialStatusFilter(statusValues []string) (bool, bool, error) {
	if len(statusValues) == 0 {
		return true, false, nil
	}

	var showActive bool
	var showRevoked bool
	for _, status := range statusValues {
		switch status {
		case credentialStatusActive:
			showActive = true
		case credentialStatusRevoked:
			showRevoked = true
		default:
			return false, false, e.NewHTTPErrorf(
				http.StatusBadRequest,
				"Form validation failed.",
			)
		}
	}
	return showActive, showRevoked, nil
}

// isDefaultCredentialStatusFilter reports whether the selection shows only active
// credentials, which is also the case for an explicit `active`-only selection.
func isDefaultCredentialStatusFilter(statusValues []string) bool {
	showActive, showRevoked, err := parseCredentialStatusFilter(statusValues)
	return err != nil || (showActive && !showRevoked)
}

// newCredentialFilterButton marks the button as selected while a non-default filter is
// active. It is wrapped in a container so list refreshes can swap it out-of-band.
func newCredentialFilterButton(
	id string,
	tooltip *widget.Text,
	dialogEndpoint string,
	statusValues []string,
	isOOB bool,
) *widget.Container {
	swapOOB := ""
	if isOOB {
		swapOOB = "outerHTML"
	}
	return &widget.Container{
		Widget: widget.Widget[widget.Container]{
			ID: id,
		},
		HTMXAttrs: widget.HTMXAttrs{
			HxSwapOOB: swapOOB,
		},
		Child: &widget.IconButton{
			Icon:       "filter_alt",
			Tooltip:    tooltip,
			Label:      tooltip,
			IsSelected: !isDefaultCredentialStatusFilter(statusValues),
			HTMXAttrs: widget.HTMXAttrs{
				Role:          "button",
				HxPost:        dialogEndpoint,
				LoadInPopover: true,
			},
		},
	}
}

func newCredentialStatusFilterChip(
	label *widget.Text,
	status string,
	isChecked bool,
	changedEvent event.Event,
) *widget.FilterChip {
	return &widget.FilterChip{
		Type:      widget.FilterChipTypeCheckbox,
		Label:     label,
		Name:      "CredentialStatusValues",
		Value:     status,
		IsChecked: isChecked,
		HTMXAttrs: widget.HTMXAttrs{
			HxOn: &widget.HxOn{
				Event: "change",
				Handler: template.JS(
					"if (event.target.checked) { " +
						"if (!new URLSearchParams(window.location.search).has('credential_status')) { " +
						"_appendQueryParamSliceValue('credential_status', '" +
						credentialStatusActive + "'); } " +
						"_appendQueryParamSliceValue('credential_status', '" + status + "'); " +
						"} else { _deleteQueryParamSliceValue('credential_status', '" + status + "'); } " +
						"this.dispatchEvent(new CustomEvent('" +
						changedEvent.String() + "', { bubbles: true }))",
				),
			},
		},
	}
}
