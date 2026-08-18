package browse

import "testing"

func TestCanShowPreviewTabs(t *testing.T) {
	for _, test := range []struct {
		name        string
		configured  bool
		sourceFinal bool
		ready       bool
		want        bool
	}{
		{name: "configured final source", configured: true, sourceFinal: true, want: true},
		{name: "Gotenberg unavailable", sourceFinal: true, want: false},
		{name: "source not final", configured: true, want: false},
		{name: "existing preview remains available", sourceFinal: true, ready: true, want: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := canShowPreviewTabs(test.configured, test.sourceFinal, test.ready); got != test.want {
				t.Fatalf("canShowPreviewTabs() = %t, want %t", got, test.want)
			}
		})
	}
}
