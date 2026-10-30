package widget

import "testing"

func TestTuKeepsPercentSignsInUserValues(t *testing.T) {
	for _, value := range []string{"100% done.pdf", "%d %s %v", "%"} {
		if got := Tu(value).String(nil); got != value {
			t.Fatalf("Tu(%q).String() = %q", value, got)
		}
	}
}
