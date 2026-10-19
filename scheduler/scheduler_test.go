package scheduler

import "testing"

func TestRunRecoveringRestartsJobAfterPanicAndStopsWhenJobReturns(t *testing.T) {
	calls := 0
	job := func() {
		calls++
		if calls < 3 {
			panic("job failed")
		}
	}

	(&Scheduler{}).runRecovering("test job", 0, job)

	if calls != 3 {
		t.Fatalf("job calls = %d, want 3", calls)
	}
}
