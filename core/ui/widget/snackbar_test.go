package widget

import "testing"

func TestSnackbarDisableAutoDismiss(t *testing.T) {
	snackbar := NewSnackbarf("Persistent warning").DisableAutoDismiss()
	if got := snackbar.GetAutoDismissTimeout(); got != 0 {
		t.Fatalf("expected auto-dismiss timeout 0, got %d", got)
	}
}
