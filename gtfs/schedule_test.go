package gtfs

import (
	"archive/zip"
	"bytes"
	"testing"
	"time"
)

func buildZip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func sampleFeed(t *testing.T) []byte {
	return buildZip(t, map[string]string{
		"stops.txt": "stop_id,stop_name\nFD,Home\nWork,Work\n",
		"calendar.txt": "service_id,monday,tuesday,wednesday,thursday,friday,saturday,sunday,start_date,end_date\n" +
			"WK,1,1,1,1,1,0,0,20260101,20261231\n",
		"calendar_dates.txt": "service_id,date,exception_type\n",
		"trips.txt": "route_id,service_id,trip_id\nNH,WK,T1\nNH,WK,T2\n",
		"stop_times.txt": "trip_id,arrival_time,departure_time,stop_id,stop_sequence\n" +
			"T1,07:10:00,07:10:00,FD,1\nT1,08:20:00,08:20:00,Work,2\n" +
			"T2,07:40:00,07:40:00,FD,1\nT2,08:50:00,08:50:00,Work,2\n",
	})
}

func TestNextDeparturesReturnsUpcomingInOrder(t *testing.T) {
	sched, err := Load(sampleFeed(t))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	loc, _ := time.LoadLocation("America/New_York")
	// Monday 2026-05-25 07:00 local: both T1 (07:10) and T2 (07:40) are upcoming.
	now := time.Date(2026, 5, 25, 7, 0, 0, 0, loc)
	trains := sched.NextDepartures("FD", "Work", now, 5)
	if len(trains) != 2 {
		t.Fatalf("got %d trains, want 2", len(trains))
	}
	if !trains[0].Departure.Before(trains[1].Departure) {
		t.Fatalf("not sorted: %v", trains)
	}
	if trains[0].TripID != "T1" {
		t.Fatalf("first trip = %s, want T1", trains[0].TripID)
	}
}

func TestNextDeparturesSkipsPastAndWrongDirection(t *testing.T) {
	sched, err := Load(sampleFeed(t))
	if err != nil {
		t.Fatal(err)
	}
	loc, _ := time.LoadLocation("America/New_York")
	now := time.Date(2026, 5, 25, 7, 30, 0, 0, loc) // T1 already departed
	trains := sched.NextDepartures("FD", "Work", now, 5)
	if len(trains) != 1 || trains[0].TripID != "T2" {
		t.Fatalf("got %v, want only T2", trains)
	}
	// Reverse direction has no trips serving Work before FD.
	if rev := sched.NextDepartures("Work", "FD", now, 5); len(rev) != 0 {
		t.Fatalf("reverse should be empty, got %v", rev)
	}
}

func TestNextDeparturesLimitsToN(t *testing.T) {
	sched, _ := Load(sampleFeed(t))
	loc, _ := time.LoadLocation("America/New_York")
	now := time.Date(2026, 5, 25, 6, 0, 0, 0, loc)
	if trains := sched.NextDepartures("FD", "Work", now, 1); len(trains) != 1 {
		t.Fatalf("limit not applied: %d", len(trains))
	}
}
