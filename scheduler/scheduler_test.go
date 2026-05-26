package scheduler

import (
	"testing"
	"time"
)

func TestDueFiresOnWeekdayAtMatchingMinute(t *testing.T) {
	loc, _ := time.LoadLocation("America/New_York")
	mondayMorning := time.Date(2026, 5, 25, 7, 0, 30, 0, loc) // Monday 07:00:30

	which, ok := Due(mondayMorning, "07:00", "17:00", time.Time{})
	if !ok || which != "morning" {
		t.Fatalf("expected morning fire, got %q ok=%v", which, ok)
	}
}

func TestDueDoesNotRefireSameMinute(t *testing.T) {
	loc, _ := time.LoadLocation("America/New_York")
	now := time.Date(2026, 5, 25, 7, 0, 30, 0, loc)
	last := time.Date(2026, 5, 25, 7, 0, 5, 0, loc) // already fired this minute
	if _, ok := Due(now, "07:00", "17:00", last); ok {
		t.Fatal("should not refire within the same minute")
	}
}

func TestDueSkipsWeekend(t *testing.T) {
	loc, _ := time.LoadLocation("America/New_York")
	saturday := time.Date(2026, 5, 23, 7, 0, 0, 0, loc)
	if _, ok := Due(saturday, "07:00", "17:00", time.Time{}); ok {
		t.Fatal("should not fire on weekend")
	}
}

func TestDueEveningMatches(t *testing.T) {
	loc, _ := time.LoadLocation("America/New_York")
	now := time.Date(2026, 5, 25, 17, 0, 10, 0, loc)
	which, ok := Due(now, "07:00", "17:00", time.Time{})
	if !ok || which != "evening" {
		t.Fatalf("expected evening, got %q ok=%v", which, ok)
	}
}
