package widget

import "testing"

func TestToolbarVariants(t *testing.T) {
	t.Parallel()

	docked := NewToolbar("Document actions")
	if docked.IsFloating() || docked.IsVertical() || docked.IsVibrant() {
		t.Fatal("new toolbar must default to standard horizontal docked layout")
	}

	vertical := NewToolbar("Formatting actions").SetVertical().SetVibrant()
	if !vertical.IsFloating() || !vertical.IsVertical() || !vertical.IsVibrant() {
		t.Fatal("vertical vibrant toolbar variant was not applied")
	}
}
