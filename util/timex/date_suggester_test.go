package timex

import (
	"testing"
	"time"
)

func TestDateSuggesterSuggestFromText(t *testing.T) {
	suggester := NewDateSuggester()
	suggester.timeNow = func() time.Time {
		return time.Date(2026, time.January, 26, 0, 0, 0, 0, time.Local)
	}

	content := "Invoice 2026-01-05 and 05.01.25 plus 05.01.2026 and 05.01.25 again."
	results := suggester.SuggestFromText(content)

	if len(results) != 2 {
		t.Fatalf("expected 2 suggestions, got %d", len(results))
	}

	if results[0].Format("2006-01-02") != "2025-01-05" {
		t.Fatalf("expected first suggestion 2025-01-05, got %s", results[0].Format("2006-01-02"))
	}
	if results[1].Format("2006-01-02") != "2026-01-05" {
		t.Fatalf("expected second suggestion 2026-01-05, got %s", results[1].Format("2006-01-02"))
	}
}

func TestDateSuggesterOnlySupportsTwoDigitWithDots(t *testing.T) {
	suggester := NewDateSuggester()
	suggester.timeNow = func() time.Time {
		return time.Date(2026, time.January, 26, 0, 0, 0, 0, time.Local)
	}

	results := suggester.SuggestFromText("01/02/25")
	if len(results) != 0 {
		t.Fatalf("expected no suggestions for slash two-digit year, got %d", len(results))
	}
}
