package webdavresource

import "testing"

func TestStateStringsMatchPersistedValues(t *testing.T) {
	tests := map[State]string{
		Uploading:      "Uploading",
		Active:         "Active",
		CleanupPending: "CleanupPending",
	}

	for state, want := range tests {
		if got := state.String(); got != want {
			t.Fatalf("String() = %q, want %q", got, want)
		}
	}
}
