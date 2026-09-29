package dashboard

import (
	"testing"

	"github.com/simpledms/simpledms/core/ui/widget"
)

func TestCredentialFilterButtonIndicatesNonDefaultFilter(t *testing.T) {
	for _, testCase := range []struct {
		name         string
		statusValues []string
		isSelected   bool
	}{
		{"no selection", nil, false},
		{"explicit active", []string{credentialStatusActive}, false},
		{"revoked", []string{credentialStatusRevoked}, true},
		{"active and revoked", []string{credentialStatusActive, credentialStatusRevoked}, true},
		{"invalid", []string{"unknown"}, false},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			button := newCredentialFilterButton(
				"filterButton",
				widget.T("Filter"),
				"/filter",
				testCase.statusValues,
				true,
			)
			iconButton := button.Child.(*widget.IconButton)
			if iconButton.IsSelected != testCase.isSelected {
				t.Fatalf("IsSelected = %v, want %v", iconButton.IsSelected, testCase.isSelected)
			}
			if button.HxSwapOOB != "outerHTML" || button.GetID() != "filterButton" {
				t.Fatalf("expected out-of-band swappable container, got %#v", button)
			}
		})
	}
}
