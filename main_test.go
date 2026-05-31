package main

import (
	"testing"
	"time"
)

func TestPrimaryDirection(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	at := func(h, m int) time.Time {
		return time.Date(2026, 5, 30, h, m, 0, 0, loc)
	}
	cases := []struct {
		name   string
		now    time.Time
		cutoff string
		want   string
	}{
		{"empty cutoff is outbound", at(15, 0), "", "outbound"},
		{"before noon", at(8, 59), "12:00", "outbound"},
		{"at noon", at(12, 0), "12:00", "inbound"},
		{"after noon", at(15, 30), "12:00", "inbound"},
		{"custom 14:30 just before", at(14, 29), "14:30", "outbound"},
		{"custom 14:30 at boundary", at(14, 30), "14:30", "inbound"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := primaryDirection(tc.now, tc.cutoff)
			if got != tc.want {
				t.Fatalf("primaryDirection(%v, %q) = %q, want %q", tc.now, tc.cutoff, got, tc.want)
			}
		})
	}
}
