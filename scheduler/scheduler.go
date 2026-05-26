// Package scheduler decides when to fire weekday morning/evening nudges and
// runs that decision on a one-minute ticker.
package scheduler

import (
	"context"
	"log"
	"time"

	"github.com/keshaldee/commute/model"
)

// Due reports whether a nudge should fire at now. It returns "morning" or
// "evening" when the current weekday minute matches a configured time and a
// nudge has not already fired during this same minute (per last).
func Due(now time.Time, morning, evening string, last time.Time) (string, bool) {
	if now.Weekday() == time.Saturday || now.Weekday() == time.Sunday {
		return "", false
	}
	if !last.IsZero() && sameMinute(now, last) {
		return "", false
	}
	cur := now.Format("15:04")
	switch cur {
	case morning:
		return "morning", true
	case evening:
		return "evening", true
	}
	return "", false
}

func sameMinute(a, b time.Time) bool {
	return a.Truncate(time.Minute).Equal(b.Truncate(time.Minute))
}

// Run ticks every minute and calls send(which) when a nudge is due. snapshot
// returns the current snapshot to summarize. It blocks until ctx is cancelled.
func Run(ctx context.Context, morning, evening string, loc *time.Location,
	snapshot func() model.Snapshot, send func(which string, snap model.Snapshot)) {

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	var last time.Time
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := time.Now().In(loc)
			if which, ok := Due(now, morning, evening, last); ok {
				last = now
				log.Printf("scheduler: firing %s nudge", which)
				send(which, snapshot())
			}
		}
	}
}
